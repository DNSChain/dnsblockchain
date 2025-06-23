package keeper

import (
	"context"
	"errors" // Importar el paquete errors estándar de Go
	"fmt"
	"strings" // For TLD normalization

	"dnsblockchain/x/dao/types"

	dnstypes "dnsblockchain/x/dnsblockchain/types" // Import for dnsblockchain types

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors" // Usar un alias para el de cosmossdk
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

type msgServer struct {
	Keeper
}

func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = (*msgServer)(nil)

func (k *msgServer) SubmitProposal(goCtx context.Context, msg *types.MsgSubmitProposal) (*types.MsgSubmitProposalResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Logger(ctx)
	logger.Info("MsgSubmitProposal received", "proposer", msg.Proposer, "title", msg.Title)

	proposerAddr, err := k.addressCodec.StringToBytes(msg.Proposer)
	if err != nil {
		logger.Error("Failed to parse proposer address", "error", err)
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "invalid proposer address")
	}
	proposer := sdk.AccAddress(proposerAddr)

	var content types.ProposalContent
	if msg.Content == nil {
		logger.Error("Proposal content is nil")
		return nil, errorsmod.Wrap(types.ErrInvalidProposalContent, "proposal content is nil")
	}
	if err := k.cdc.UnpackAny(msg.Content, &content); err != nil {
		logger.Error("Failed to unpack proposal content", "error", err)
		return nil, errorsmod.Wrap(types.ErrInvalidProposalContent, fmt.Sprintf("failed to unpack proposal content: %v", err))
	}

	if content == nil {
		logger.Error("Unpacked proposal content is nil")
		return nil, errorsmod.Wrap(types.ErrInvalidProposalContent, "unpacked proposal content is nil")
	}

	if err := content.ValidateBasic(); err != nil {
		logger.Error("Invalid proposal content", "error", err)
		return nil, errorsmod.Wrap(err, "invalid proposal content")
	}

	var activeProposalDuplicateFound bool
	err = k.Proposals.Walk(ctx, nil, func(proposalID uint64, existingProposal types.Proposal) (stop bool, iterErr error) {
		if existingProposal.Status == types.ProposalStatus_PROPOSAL_STATUS_VOTING_PERIOD {
			var existingContent types.ProposalContent
			if errUnpack := k.cdc.UnpackAny(existingProposal.Content, &existingContent); errUnpack != nil {
				logger.Error("Failed to unpack existing proposal content during duplicate check", "existingProposalID", existingProposal.Id, "error", errUnpack)
				return false, nil
			}

			if msg.Content.TypeUrl == existingProposal.Content.TypeUrl {
				newContentBytes, _ := k.cdc.MarshalInterface(content)
				existingContentBytes, _ := k.cdc.MarshalInterface(existingContent)

				if string(newContentBytes) == string(existingContentBytes) {
					activeProposalDuplicateFound = true
					return true, nil
				}
			}
		}
		return false, nil
	})
	if err != nil {
		logger.Error("Error walking proposals for duplicate check", "error", err)
		return nil, errorsmod.Wrap(err, "failed to check for duplicate active proposals")
	}
	if activeProposalDuplicateFound {
		logger.Warn("Attempt to submit a proposal with content identical to an active proposal")
		return nil, types.ErrDuplicateActiveProposal
	}

	params, err := k.Params.Get(ctx)
	if err != nil {
		logger.Error("Failed to get dao params", "error", err)
		return nil, errorsmod.Wrap(err, "failed to get dao params")
	}
	logger.Info("DAO Params fetched", "general_deposit", params.ProposalSubmissionDeposit.String(), "add_tld_cost", params.AddTldProposalCost.String(), "voting_period", params.VotingPeriodBlocks)

	var depositToPay sdk.Coins
	isAddTldProposal := false

	if addTldContent, ok := content.(*types.AddTldProposalContent); ok {
		normalizedProposedTLD := strings.ToLower(strings.Trim(addTldContent.Tld, "."))
		isAddTldProposal = true
		isGloballyReserved, errGloballyReserved := k.dnsblockchainKeeper.IsTLDGloballyReserved(ctx, normalizedProposedTLD)
		if errGloballyReserved != nil {
			logger.Error("Failed to check if TLD is globally reserved", "tld", normalizedProposedTLD, "error", errGloballyReserved)
			return nil, errorsmod.Wrap(errGloballyReserved, "failed to check TLD global reservation status")
		}
		if isGloballyReserved {
			logger.Warn("Attempt to submit proposal for a globally reserved TLD", "tld", normalizedProposedTLD)
			return nil, errorsmod.Wrapf(dnstypes.ErrTLDReservedByICANN, "TLD '%s' is reserved by ICANN and cannot be proposed", normalizedProposedTLD)
		}
		isPermitted, errPermitted := k.dnsblockchainKeeper.IsTLDPermitted(ctx, normalizedProposedTLD)
		if errPermitted != nil {
			logger.Error("Failed to check if TLD is already permitted", "tld", addTldContent.Tld, "error", errPermitted)
			return nil, errorsmod.Wrap(errPermitted, "failed to check TLD permission status")
		}
		if isPermitted {
			logger.Warn("Attempt to submit proposal for an already permitted TLD", "tld", addTldContent.Tld)
			return nil, errorsmod.Wrapf(types.ErrInvalidProposalContent, "TLD '%s' is already permitted or registered", addTldContent.Tld)
		}
		depositToPay = params.AddTldProposalCost
		logger.Info("Proposal identified as AddTldProposal", "cost", depositToPay.String())
	} else {
		depositToPay = params.ProposalSubmissionDeposit
		logger.Info("Proposal identified as general proposal", "deposit", depositToPay.String())
	}

	if !depositToPay.IsZero() {
		if err := k.Keeper.bankKeeper.SendCoinsFromAccountToModule(ctx, proposer, types.ModuleName, depositToPay); err != nil {
			logger.Error("Failed to send required deposit/cost from proposer to DAO module", "proposer", proposer.String(), "amount", depositToPay.String(), "error", err)
			return nil, errorsmod.Wrap(err, "failed to send required deposit/cost")
		}
		logger.Info("Required deposit/cost sent successfully", "amount", depositToPay.String())
	} else {
		if !msg.InitialDeposit.IsZero() {
			logger.Error("Parameter deposit/cost is zero, but non-zero initial_deposit provided in message", "initial_deposit_msg", msg.InitialDeposit.String())
			return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "parameter deposit/cost is zero, initial_deposit in message must also be zero")
		}
		logger.Info("No deposit/cost required by params, and no deposit provided in message.")
	}

	proposalID, err := k.GetNextProposalID(ctx)
	if err != nil {
		logger.Error("Failed to get next proposal ID", "error", err)
		return nil, errorsmod.Wrap(err, "failed to get next proposal ID")
	}
	logger.Info("Next proposal ID obtained", "proposalID", proposalID)

	submitBlock := uint64(ctx.BlockHeight())
	votingPeriodBlocks := params.VotingPeriodBlocks
	votingEndBlock := submitBlock + votingPeriodBlocks
	logger.Info("Proposal timing calculated", "submitBlock", submitBlock, "votingPeriodBlocks", votingPeriodBlocks, "votingEndBlock", votingEndBlock)

	proposal := types.Proposal{
		Id:                         proposalID,
		Proposer:                   msg.Proposer,
		Title:                      msg.Title,
		Description:                msg.Description,
		Content:                    msg.Content,
		Status:                     types.ProposalStatus_PROPOSAL_STATUS_VOTING_PERIOD,
		SubmitBlock:                submitBlock,
		VotingStartBlock:           submitBlock,
		VotingEndBlock:             votingEndBlock,
		YesVotes:                   math.ZeroInt(),
		NoVotes:                    math.ZeroInt(),
		AbstainVotes:               math.ZeroInt(),
		TotalVotingPowerAtSnapshot: k.bankKeeper.GetSupply(ctx, params.VotingTokenDenom).Amount,
	}
	logger.Info("Proposal object created", "proposalID", proposal.Id, "status", proposal.Status.String())

	if err := k.SetProposal(ctx, proposal); err != nil {
		logger.Error("Failed to set proposal in store", "error", err)
		return nil, errorsmod.Wrap(err, "failed to set proposal")
	}
	logger.Info("Proposal set in store successfully", "proposalID", proposal.Id)

	proposalType := content.ProposalType()
	if isAddTldProposal {
		proposalType = "AddTld"
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeSubmitProposal,
			sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", proposalID)),
			sdk.NewAttribute(govtypes.AttributeKeyProposalType, proposalType),
			sdk.NewAttribute(types.AttributeKeyProposer, msg.Proposer),
		),
	})
	logger.Info("SubmitProposal successful", "proposalID", proposalID)
	return &types.MsgSubmitProposalResponse{ProposalId: proposalID}, nil
}

func (k *msgServer) Vote(goCtx context.Context, msg *types.MsgVote) (*types.MsgVoteResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Logger(ctx)
	logger.Info("MsgVote received", "proposalID", msg.ProposalId, "voter", msg.Voter, "option", msg.Option.String())

	voterAddrBytes, err := k.addressCodec.StringToBytes(msg.Voter)
	if err != nil {
		logger.Error("Invalid voter address", "address", msg.Voter, "error", err)
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "invalid voter address")
	}
	voter := sdk.AccAddress(voterAddrBytes)
	_ = voter

	proposal, err := k.GetProposal(ctx, msg.ProposalId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			logger.Warn("Proposal not found for voting", "proposalID", msg.ProposalId)
			return nil, errorsmod.Wrapf(types.ErrProposalNotFound, "proposalID %d not found", msg.ProposalId)
		}
		logger.Error("Failed to get proposal for voting", "proposalID", msg.ProposalId, "error", err)
		return nil, errorsmod.Wrapf(types.ErrProposalNotFound, "failed to get proposalID %d: %s", msg.ProposalId, err.Error())
	}
	logger.Info("Proposal found for voting", "proposalID", proposal.Id, "currentStatus", proposal.Status.String(), "votingEndBlock", proposal.VotingEndBlock, "currentBlock", ctx.BlockHeight())

	if proposal.Status != types.ProposalStatus_PROPOSAL_STATUS_VOTING_PERIOD {
		logger.Warn("Attempt to vote on proposal not in voting period", "proposalID", proposal.Id, "status", proposal.Status.String())
		return nil, errorsmod.Wrapf(types.ErrInactiveProposal, "proposalID %d: not in voting period (current status: %s)", msg.ProposalId, proposal.Status.String())
	}

	currentBlock := uint64(ctx.BlockHeight())
	if currentBlock > proposal.VotingEndBlock {
		logger.Warn("Attempt to vote on proposal whose voting period has ended", "proposalID", proposal.Id, "votingEndBlock", proposal.VotingEndBlock, "currentBlock", currentBlock)
		return nil, errorsmod.Wrapf(types.ErrVotingPeriodEnded, "proposalID %d: voting period has ended", msg.ProposalId)
	}

	existingVote, err := k.GetVote(ctx, msg.ProposalId, voter)
	if err == nil && existingVote.Voter != "" {
		logger.Warn("Voter already voted", "proposalID", msg.ProposalId, "voter", msg.Voter)
		return nil, errorsmod.Wrapf(types.ErrAlreadyVoted, "voter %s already voted on proposal %d", msg.Voter, msg.ProposalId)
	} else if err != nil && !errors.Is(err, collections.ErrNotFound) {
		logger.Error("Failed to check for existing vote", "proposalID", msg.ProposalId, "voter", msg.Voter, "error", err)
		return nil, errorsmod.Wrap(err, "failed to check existing vote")
	}

	// CalculateVoterPower obtendrá los params internamente.
	votingPower, err := k.CalculateVoterPower(ctx, voter, uint64(ctx.BlockHeight()))
	if err != nil {
		logger.Error("Failed to calculate voter power", "voter", msg.Voter, "error", err)
		return nil, errorsmod.Wrap(err, "failed to calculate voter power")
	}

	if votingPower.IsZero() {
		logger.Warn("Voter has no voting power for this proposal", "proposalID", msg.ProposalId, "voter", msg.Voter)
		return nil, types.ErrNoVotingPower
	}

	vote := types.Vote{
		ProposalId:  msg.ProposalId,
		Voter:       msg.Voter,
		Option:      msg.Option,
		VotingPower: votingPower,
	}

	if err := k.SetVote(ctx, vote); err != nil {
		logger.Error("Failed to set vote in store", "error", err)
		return nil, errorsmod.Wrap(err, "failed to set vote")
	}
	logger.Info("Vote set in store successfully", "proposalID", msg.ProposalId, "voter", msg.Voter, "option", msg.Option.String())

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeProposalVote,
			sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", msg.ProposalId)),
			sdk.NewAttribute(types.AttributeKeyVoter, msg.Voter),
			sdk.NewAttribute(types.AttributeKeyVoteOption, msg.Option.String()),
			sdk.NewAttribute(types.AttributeKeyVotingPower, votingPower.String()),
		),
	})
	logger.Info("Vote successful", "proposalID", msg.ProposalId, "voter", msg.Voter)
	return &types.MsgVoteResponse{}, nil
}
