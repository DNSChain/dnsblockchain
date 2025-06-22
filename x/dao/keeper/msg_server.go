package keeper

import (
	"context"
	"errors" // Importar el paquete errors estándar de Go
	"fmt"

	"dnsblockchain/x/dao/types"

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
	logger := k.Logger(ctx) // Usar el logger del keeper
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

	params, err := k.Params.Get(ctx)
	if err != nil {
		logger.Error("Failed to get dao params", "error", err)
		return nil, errorsmod.Wrap(err, "failed to get dao params")
	}
	logger.Info("DAO Params fetched", "deposit", params.ProposalSubmissionDeposit.String(), "voting_period", params.VotingPeriodBlocks)

	if !params.ProposalSubmissionDeposit.IsZero() {
		logger.Info("Attempting to send deposit", "from", proposer.String(), "amount", params.ProposalSubmissionDeposit.String())
		err = k.Keeper.bankKeeper.SendCoinsFromAccountToModule(ctx, proposer, types.ModuleName, params.ProposalSubmissionDeposit)
		if err != nil {
			logger.Error("Failed to send deposit", "error", err)
			return nil, errorsmod.Wrap(err, "failed to send deposit")
		}
		logger.Info("Deposit sent successfully")
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
		Status:                     types.ProposalStatus_PROPOSAL_STATUS_VOTING_PERIOD, // <--- ESTADO INICIAL
		SubmitBlock:                submitBlock,
		VotingStartBlock:           submitBlock, // La votación empieza inmediatamente
		VotingEndBlock:             votingEndBlock,
		YesVotes:                   math.ZeroInt(),
		NoVotes:                    math.ZeroInt(),
		AbstainVotes:               math.ZeroInt(),
		TotalVotingPowerAtSnapshot: math.ZeroInt(), // TODO: Set this correctly
	}
	logger.Info("Proposal object created", "proposalID", proposal.Id, "status", proposal.Status.String())

	if err := k.SetProposal(ctx, proposal); err != nil {
		logger.Error("Failed to set proposal in store", "error", err)
		return nil, errorsmod.Wrap(err, "failed to set proposal")
	}
	logger.Info("Proposal set in store successfully", "proposalID", proposal.Id)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeSubmitProposal,
			sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", proposalID)),
			sdk.NewAttribute(govtypes.AttributeKeyProposalType, content.ProposalType()), // Asegúrate que content.ProposalType() está definido
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
	_ = voter // Para evitar error de no usado si no se usa más adelante explícitamente

	proposal, err := k.GetProposal(ctx, msg.ProposalId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) { // Usar errors.Is del paquete estándar 'errors'
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

	// Lógica para comprobar si ya votó (corregida)
	existingVote, err := k.GetVote(ctx, msg.ProposalId, voter)
	if err == nil && existingVote.Voter != "" { // Si no hay error Y el voto recuperado tiene un votante (es decir, existe y no es un voto vacío por defecto)
		logger.Warn("Voter already voted", "proposalID", msg.ProposalId, "voter", msg.Voter)
		return nil, errorsmod.Wrapf(types.ErrAlreadyVoted, "voter %s already voted on proposal %d", msg.Voter, msg.ProposalId)
	} else if err != nil && !errors.Is(err, collections.ErrNotFound) { // Si hay un error Y NO ES ErrNotFound, es un problema
		logger.Error("Failed to check for existing vote", "proposalID", msg.ProposalId, "voter", msg.Voter, "error", err)
		return nil, errorsmod.Wrap(err, "failed to check existing vote")
	}
	// Si err es collections.ErrNotFound, o si err es nil pero existingVote.Voter es "", entonces el votante no ha votado todavía, lo cual es bueno.

	// TODO: Implementar GetVotingPower(ctx, voter)
	// votingPower := k.GetVotingPower(ctx, voter)
	// if votingPower.IsZero() {
	// 	return nil, errorsmod.Wrapf(types.ErrNoVotingPower, "voter %s has no voting power", msg.Voter)
	// }

	vote := types.Vote{
		ProposalId: msg.ProposalId,
		Voter:      msg.Voter,
		Option:     msg.Option,
		// VotingPowerUsed: votingPower, // Placeholder
	}

	if err := k.SetVote(ctx, vote); err != nil {
		logger.Error("Failed to set vote in store", "error", err)
		return nil, errorsmod.Wrap(err, "failed to set vote")
	}
	logger.Info("Vote set in store successfully", "proposalID", msg.ProposalId, "voter", msg.Voter, "option", msg.Option.String())

	// TODO: Actualizar el tally de la propuesta aquí o en el EndBlocker.
	// Por ahora, el tally se hará en el EndBlocker.

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeProposalVote,
			sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", msg.ProposalId)),
			sdk.NewAttribute(types.AttributeKeyVoter, msg.Voter),
			sdk.NewAttribute(types.AttributeKeyVoteOption, msg.Option.String()),
		),
	})
	logger.Info("Vote successful", "proposalID", msg.ProposalId, "voter", msg.Voter)
	return &types.MsgVoteResponse{}, nil
}
