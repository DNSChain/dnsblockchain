package keeper

import (
	"context"
	"fmt"

	"dnsblockchain/x/dao/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/errors"
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

	proposerAddr, err := k.addressCodec.StringToBytes(msg.Proposer)
	if err != nil {
		return nil, errors.Wrap(sdkerrors.ErrInvalidAddress, "invalid proposer address")
	}
	proposer := sdk.AccAddress(proposerAddr)

	var content types.ProposalContent
	// Asumimos que UnpackInterfaces ya fue llamado por el framework.
	// GetContent() debería devolver el valor cacheado.
	// Si MsgSubmitProposal tiene un método GetContent() que devuelve (ProposalContent, bool)
	// y hace el msg.Content.GetCachedValue().(types.ProposalContent)
	// entonces la siguiente línea es correcta.
	// Si MsgSubmitProposal no tiene GetContent(), o msg.Content es directamente Any, desempaquetar aquí.
	if msg.Content == nil {
		return nil, errors.Wrap(types.ErrInvalidProposalContent, "proposal content is nil")
	}
	if err := k.cdc.UnpackAny(msg.Content, &content); err != nil { // Usar el cdc del keeper
		return nil, errors.Wrap(types.ErrInvalidProposalContent, fmt.Sprintf("failed to unpack proposal content: %v", err))
	}

	if content == nil {
		return nil, errors.Wrap(types.ErrInvalidProposalContent, "unpacked proposal content is nil")
	}

	if err := content.ValidateBasic(); err != nil {
		return nil, errors.Wrap(err, "invalid proposal content")
	}

	params, err := k.Params.Get(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get dao params")
	}
	if !params.ProposalSubmissionDeposit.IsZero() {
		err = k.Keeper.bankKeeper.SendCoinsFromAccountToModule(ctx, proposer, types.ModuleName, params.ProposalSubmissionDeposit)
		if err != nil {
			return nil, errors.Wrap(err, "failed to send deposit")
		}
	}

	proposalID, err := k.GetNextProposalID(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get next proposal ID")
	}

	submitBlock := uint64(ctx.BlockHeight())
	votingPeriodBlocks := params.VotingPeriodBlocks
	votingEndBlock := submitBlock + votingPeriodBlocks

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
		TotalVotingPowerAtSnapshot: math.ZeroInt(),
	}

	if err := k.SetProposal(ctx, proposal); err != nil {
		return nil, errors.Wrap(err, "failed to set proposal")
	}

	// err = k.AddToActiveProposalQueue(ctx, proposalID, votingEndBlock)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "failed to add to active proposal queue")
	// }

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeSubmitProposal,
			sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", proposalID)),
			sdk.NewAttribute(govtypes.AttributeKeyProposalType, content.ProposalType()),
			sdk.NewAttribute(types.AttributeKeyProposer, msg.Proposer),
		),
	})

	return &types.MsgSubmitProposalResponse{ProposalId: proposalID}, nil
}

func (k *msgServer) Vote(goCtx context.Context, msg *types.MsgVote) (*types.MsgVoteResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	voterAddrBytes, err := k.addressCodec.StringToBytes(msg.Voter)
	if err != nil {
		return nil, errors.Wrap(sdkerrors.ErrInvalidAddress, "invalid voter address")
	}
	voter := sdk.AccAddress(voterAddrBytes)
	_ = voter

	proposal, err := k.GetProposal(ctx, msg.ProposalId)
	if err != nil {
		if err == collections.ErrNotFound {
			return nil, errors.Wrapf(types.ErrProposalNotFound, "proposalID %d not found", msg.ProposalId)
		}
		return nil, errors.Wrapf(types.ErrProposalNotFound, "failed to get proposalID %d: %s", msg.ProposalId, err.Error())
	}

	if proposal.Status != types.ProposalStatus_PROPOSAL_STATUS_VOTING_PERIOD {
		return nil, errors.Wrapf(types.ErrInactiveProposal, "proposalID %d: not in voting period", msg.ProposalId)
	}

	currentBlock := uint64(ctx.BlockHeight())
	if currentBlock > proposal.VotingEndBlock {
		return nil, errors.Wrapf(types.ErrVotingPeriodEnded, "proposalID %d: voting period has ended", msg.ProposalId)
	}

	// _, err = k.GetVote(ctx, msg.ProposalId, voter)
	// if err == nil {
	// 	return nil, errors.Wrapf(types.ErrAlreadyVoted, "voter %s already voted on proposal %d", msg.Voter, msg.ProposalId)
	// } else if err != collections.ErrNotFound {
	// 	return nil, errors.Wrap(err, "failed to check existing vote")
	// }

	// votingPower := k.GetVotingPower(ctx, voter)
	// if votingPower.IsZero() {
	// 	return nil, errors.Wrapf(types.ErrNoVotingPower, "voter %s has no voting power", msg.Voter)
	// }

	vote := types.Vote{
		ProposalId: msg.ProposalId,
		Voter:      msg.Voter,
		Option:     msg.Option,
		// VotingPowerUsed: votingPower,
	}

	if err := k.SetVote(ctx, vote); err != nil {
		return nil, errors.Wrap(err, "failed to set vote")
	}

	// switch msg.Option {
	// case types.VoteOption_VOTE_OPTION_YES:
	// 	proposal.YesVotes = proposal.YesVotes.Add(math.ZeroInt())
	// case types.VoteOption_VOTE_OPTION_NO:
	// 	proposal.NoVotes = proposal.NoVotes.Add(math.ZeroInt())
	// case types.VoteOption_VOTE_OPTION_ABSTAIN:
	// 	proposal.AbstainVotes = proposal.AbstainVotes.Add(math.ZeroInt())
	// }
	// if err := k.SetProposal(ctx, proposal); err != nil {
	// 	return nil, errors.Wrap(err, "failed to update proposal tally")
	// }

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeProposalVote,
			sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", msg.ProposalId)),
			sdk.NewAttribute(types.AttributeKeyVoter, msg.Voter),
			sdk.NewAttribute(types.AttributeKeyVoteOption, msg.Option.String()),
		),
	})

	return &types.MsgVoteResponse{}, nil
}
