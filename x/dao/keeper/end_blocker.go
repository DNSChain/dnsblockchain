package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	// govtypes "github.com/cosmos/cosmos-sdk/x/gov/types" // Para tipos de eventos si es necesario

	"dnsblockchain/x/dao/types"
)

// EndBlocker es llamado al final de cada bloque.
// Aquí procesaremos las propuestas cuyo período de votación ha terminado.
func (k Keeper) EndBlocker(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentBlockHeight := uint64(sdkCtx.BlockHeight())
	logger := k.Logger(sdkCtx)

	params, err := k.Params.Get(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "failed to get dao params in EndBlocker")
	}

	var proposalsToProcess []types.Proposal

	err = k.Proposals.Walk(sdkCtx, nil, func(proposalID uint64, proposal types.Proposal) (stop bool, err error) {
		if proposal.Status == types.ProposalStatus_PROPOSAL_STATUS_VOTING_PERIOD && currentBlockHeight > proposal.VotingEndBlock {
			proposalsToProcess = append(proposalsToProcess, proposal)
		}
		return false, nil
	})
	if err != nil {
		logger.Error("Error walking proposals in EndBlocker", "error", err)
		return errorsmod.Wrap(err, "failed to iterate proposals")
	}

	for i := range proposalsToProcess {
		proposal := proposalsToProcess[i]
		logger.Info("Processing proposal in EndBlocker", "proposalID", proposal.Id, "votingEndBlock", proposal.VotingEndBlock, "currentBlock", currentBlockHeight)

		var yesVotes, noVotes, abstainVotes math.Int
		yesVotes = math.ZeroInt()
		noVotes = math.ZeroInt()
		abstainVotes = math.ZeroInt()
		totalVotingPowerVoted := math.ZeroInt()

		prefixRange := collections.NewPrefixedPairRange[uint64, sdk.AccAddress](proposal.Id)
		err = k.Votes.Walk(sdkCtx, prefixRange, func(key collections.Pair[uint64, sdk.AccAddress], vote types.Vote) (stop bool, err error) {
			votingPower := vote.VotingPower

			switch vote.Option {
			case types.VoteOption_VOTE_OPTION_YES:
				yesVotes = yesVotes.Add(votingPower)
			case types.VoteOption_VOTE_OPTION_NO:
				noVotes = noVotes.Add(votingPower)
			case types.VoteOption_VOTE_OPTION_ABSTAIN:
				abstainVotes = abstainVotes.Add(votingPower)
			}
			// totalVotingPowerVoted incluye el poder de los votos de abstención para el quórum
			if vote.Option != types.VoteOption_VOTE_OPTION_UNSPECIFIED {
				totalVotingPowerVoted = totalVotingPowerVoted.Add(votingPower)
			}
			return false, nil
		})
		if err != nil {
			logger.Error("Error tallying votes for proposal", "proposalID", proposal.Id, "error", err)
			proposal.Status = types.ProposalStatus_PROPOSAL_STATUS_FAILED
			if setErr := k.SetProposal(sdkCtx, proposal); setErr != nil {
				logger.Error("Failed to set proposal to FAILED after tally error", "proposalID", proposal.Id, "error", setErr)
			}
			continue
		}

		proposal.YesVotes = yesVotes
		proposal.NoVotes = noVotes
		proposal.AbstainVotes = abstainVotes

		// ---- Verificación de Quórum ----
		hasQuorum := false
		if proposal.TotalVotingPowerAtSnapshot.IsPositive() {
			totalVotedDec, convErrVoted := math.LegacyNewDecFromStr(totalVotingPowerVoted.String())
			totalSnapshotDec, convErrSnap := math.LegacyNewDecFromStr(proposal.TotalVotingPowerAtSnapshot.String())

			if convErrVoted != nil || convErrSnap != nil {
				logger.Error("Error converting voting powers to LegacyDec for quorum check", "proposalID", proposal.Id, "errVoted", convErrVoted, "errSnap", convErrSnap)
				proposal.Status = types.ProposalStatus_PROPOSAL_STATUS_FAILED
			} else {
				// QuorumPercent ya es math.LegacyDec
				if totalVotedDec.Quo(totalSnapshotDec).GTE(params.QuorumPercent) {
					hasQuorum = true
				}
			}
		} else if totalVotingPowerVoted.IsPositive() && params.QuorumPercent.IsZero() {
			// Caso especial: snapshot es 0, pero alguien votó y quórum es 0%, se considera alcanzado.
			hasQuorum = true
		} else if proposal.TotalVotingPowerAtSnapshot.IsZero() && params.QuorumPercent.IsZero() {
			// Caso especial: Si el snapshot es 0 y el quórum requerido es 0, cualquier voto (incluso 0 poder) lo cumple
			// o si no hay votos, también se podría considerar que el quórum se cumple si es 0.
			// Por seguridad, si el snapshot es 0, solo un quórum de 0% puede pasarlo, y solo si alguien votó.
			// La lógica de `totalVotingPowerVoted.IsZero()` más abajo manejará el "no votos".
			// Si no hay votos y el quórum es 0 y snapshot es 0, 'passed' quedará false.
			// Si hubo votos y quórum es 0 y snapshot es 0, hasQuorum = true.
			if params.QuorumPercent.IsZero() { // Solo si el quórum es cero y el snapshot es cero
				hasQuorum = true // Permitir que proceda si el quórum es 0, incluso con snapshot 0
			}
		}

		logger.Info("Quorum check", "proposalID", proposal.Id, "totalVotingPowerVoted", totalVotingPowerVoted.String(), "totalVotingPowerAtSnapshot", proposal.TotalVotingPowerAtSnapshot.String(), "quorumParam", params.QuorumPercent.String(), "hasQuorum", hasQuorum)

		passed := false
		if proposal.Status == types.ProposalStatus_PROPOSAL_STATUS_VOTING_PERIOD && hasQuorum { // Solo si estaba en votación y tiene quórum
			if totalVotingPowerVoted.IsZero() {
				logger.Info("No votes with positive voting power cast for proposal (even if quorum met with 0 snapshot/0 quorum)", "proposalID", proposal.Id)
			} else {
				totalVotesNonAbstain := yesVotes.Add(noVotes)
				if totalVotesNonAbstain.IsPositive() {
					yesDec, convErrYes := math.LegacyNewDecFromStr(yesVotes.String())
					if convErrYes != nil {
						logger.Error("Error converting yesVotes to LegacyDec", "proposalID", proposal.Id, "error", convErrYes)
						proposal.Status = types.ProposalStatus_PROPOSAL_STATUS_FAILED
					} else {
						denominatorDec, convErrDen := math.LegacyNewDecFromStr(totalVotesNonAbstain.String())
						if convErrDen != nil {
							logger.Error("Error converting totalVotesNonAbstain to LegacyDec", "proposalID", proposal.Id, "error", convErrDen)
							proposal.Status = types.ProposalStatus_PROPOSAL_STATUS_FAILED
						} else {
							minThresholdDec := params.MinYesThresholdPercent
							if !denominatorDec.IsZero() {
								percentageYes := yesDec.Quo(denominatorDec)
								if percentageYes.GTE(minThresholdDec) {
									passed = true
								}
								logger.Info("Proposal tally", "proposalID", proposal.Id, "yes", yesVotes.String(), "no", noVotes.String(), "abstain", abstainVotes.String(), "total_voted_power", totalVotingPowerVoted.String(), "percentageYes", percentageYes.String(), "threshold", minThresholdDec.String(), "denominator_used", denominatorDec.String(), "passed", passed)
							} else {
								logger.Info("Denominator for threshold calculation is zero (no YES/NO votes with power)", "proposalID", proposal.Id)
							}
						}
					}
				} else {
					logger.Info("No YES/NO votes with positive power cast for proposal", "proposalID", proposal.Id)
				}
			}
		} else if proposal.Status == types.ProposalStatus_PROPOSAL_STATUS_VOTING_PERIOD && !hasQuorum { // Estaba en votación pero no alcanzó quórum
			logger.Info("Proposal rejected due to not meeting quorum", "proposalID", proposal.Id)
			proposal.Status = types.ProposalStatus_PROPOSAL_STATUS_REJECTED // Marcar como rechazada por falta de quórum
		}

		// ---- Lógica de manejo de depósitos y ejecución ----
		depositToHandle := params.ProposalSubmissionDeposit
		var proposalContent types.ProposalContent
		// isAddTld := false // Ya no se usa explícitamente
		if err := k.cdc.UnpackAny(proposal.Content, &proposalContent); err == nil {
			if _, ok := proposalContent.(*types.AddTldProposalContent); ok {
				depositToHandle = params.AddTldProposalCost
				// isAddTld = true
			}
		} else {
			logger.Error("Failed to unpack proposal content for deposit handling in EndBlocker", "proposalID", proposal.Id, "error", err)
			// Si no se puede desempaquetar el contenido, podría ser una razón para marcarla como FAILED
			if proposal.Status == types.ProposalStatus_PROPOSAL_STATUS_VOTING_PERIOD { // Solo si no fue rechazada por quórum
				proposal.Status = types.ProposalStatus_PROPOSAL_STATUS_FAILED
			}
		}

		if passed { // Solo si pasó la votación Y el quórum
			proposal.Status = types.ProposalStatus_PROPOSAL_STATUS_PASSED
			logger.Info("Proposal passed voting and quorum", "proposalID", proposal.Id)

			execErr := k.executeProposal(sdkCtx, proposal)
			if execErr != nil {
				logger.Error("Failed to execute proposal", "proposalID", proposal.Id, "error", execErr)
				proposal.Status = types.ProposalStatus_PROPOSAL_STATUS_FAILED
				if !depositToHandle.IsZero() {
					if errBurn := k.bankKeeper.BurnCoins(sdkCtx, types.ModuleName, depositToHandle); errBurn != nil {
						logger.Error("Failed to burn deposit after execution failure", "proposalID", proposal.Id, "deposit", depositToHandle.String(), "error", errBurn)
					} else {
						logger.Info("Deposit burned after execution failure", "proposalID", proposal.Id, "amount", depositToHandle.String())
					}
				}
			} else {
				proposal.Status = types.ProposalStatus_PROPOSAL_STATUS_EXECUTED
				logger.Info("Proposal executed successfully", "proposalID", proposal.Id)
				if !depositToHandle.IsZero() {
					proposerAddr, addrErr := k.addressCodec.StringToBytes(proposal.Proposer)
					if addrErr != nil {
						logger.Error("Failed to parse proposer address for refund", "proposalID", proposal.Id, "proposer", proposal.Proposer, "error", addrErr)
					} else {
						if errRefund := k.bankKeeper.SendCoinsFromModuleToAccount(sdkCtx, types.ModuleName, sdk.AccAddress(proposerAddr), depositToHandle); errRefund != nil {
							logger.Error("Failed to refund deposit for executed proposal", "proposalID", proposal.Id, "proposer", proposal.Proposer, "deposit", depositToHandle.String(), "error", errRefund)
						} else {
							logger.Info("Deposit refunded for executed proposal", "proposalID", proposal.Id, "amount", depositToHandle.String())
						}
					}
				}
			}
		} else { // No pasó la votación (ya sea por umbral o por quórum)
			// Si ya se marcó como REJECTED (por quórum) o FAILED (por error de unpack), no cambiar estado.
			if proposal.Status == types.ProposalStatus_PROPOSAL_STATUS_VOTING_PERIOD {
				proposal.Status = types.ProposalStatus_PROPOSAL_STATUS_REJECTED // No pasó el umbral de SÍ
				logger.Info("Proposal rejected (did not pass YES threshold)", "proposalID", proposal.Id)
			}
			// Quemar depósito si la propuesta es rechazada o falló (y el depósito no es cero)
			if (proposal.Status == types.ProposalStatus_PROPOSAL_STATUS_REJECTED || proposal.Status == types.ProposalStatus_PROPOSAL_STATUS_FAILED) && !depositToHandle.IsZero() {
				if errBurn := k.bankKeeper.BurnCoins(sdkCtx, types.ModuleName, depositToHandle); errBurn != nil {
					logger.Error("Failed to burn deposit for rejected/failed proposal", "proposalID", proposal.Id, "deposit", depositToHandle.String(), "error", errBurn)
				} else {
					logger.Info("Deposit burned for rejected/failed proposal", "proposalID", proposal.Id, "amount", depositToHandle.String())
				}
			}
		}

		if err := k.SetProposal(sdkCtx, proposal); err != nil {
			logger.Error("Failed to set proposal status after processing", "proposalID", proposal.Id, "error", err)
			return errorsmod.Wrapf(err, "failed to set proposal %d status after processing", proposal.Id)
		}

		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeProposalFinished,
				sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", proposal.Id)),
				sdk.NewAttribute(types.AttributeKeyProposalStatus, proposal.Status.String()),
			),
		)
	}

	return nil
}

func (k Keeper) executeProposal(ctx sdk.Context, proposal types.Proposal) error {
	var content types.ProposalContent
	if err := k.cdc.UnpackAny(proposal.Content, &content); err != nil {
		return errorsmod.Wrapf(types.ErrInvalidProposalContent, "failed to unpack proposal content for execution: %v", err)
	}

	switch c := content.(type) {
	case *types.AddTldProposalContent:
		return k.executeAddTldProposal(ctx, c)
	case *types.RequestTokensProposalContent:
		return k.executeRequestTokensProposal(ctx, c, proposal)
	default:
		return errorsmod.Wrapf(types.ErrInvalidProposalContent, "unknown proposal content type: %T", c)
	}
}

func (k Keeper) executeAddTldProposal(ctx sdk.Context, content *types.AddTldProposalContent) error {
	k.Logger(ctx).Info("Executing AddTldProposal", "tld", content.Tld)
	return k.dnsblockchainKeeper.AddPermittedTLD(ctx, content.Tld)
}

func (k Keeper) executeRequestTokensProposal(ctx sdk.Context, content *types.RequestTokensProposalContent, proposal types.Proposal) error {
	k.Logger(ctx).Info("Executing RequestTokensProposal", "recipient", content.RecipientAddress, "amount", content.AmountRequested.String())
	params, errParams := k.Params.Get(ctx)
	if errParams != nil {
		k.Logger(ctx).Error("Failed to get DAO params during RequestTokensProposal execution", "error", errParams)
		return errorsmod.Wrap(errParams, "failed to get dao params")
	}

	recipient, err := k.addressCodec.StringToBytes(content.RecipientAddress)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid recipient address in proposal: %s", err)
	}

	err = k.bankKeeper.MintCoins(ctx, types.ModuleName, content.AmountRequested)
	if err != nil {
		return errorsmod.Wrapf(err, "failed to mint coins for proposal execution")
	}

	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, sdk.AccAddress(recipient), content.AmountRequested)
	if err != nil {
		burnErr := k.bankKeeper.BurnCoins(ctx, types.ModuleName, content.AmountRequested)
		if burnErr != nil {
			k.Logger(ctx).Error("CRITICAL: failed to burn coins after failed send in proposal execution", "burn_error", burnErr, "original_send_error", err)
		}
		return errorsmod.Wrapf(err, "failed to send coins to recipient for proposal execution")
	}

	for _, coin := range content.AmountRequested {
		if coin.Denom == params.VotingTokenDenom && coin.Amount.IsPositive() {
			lot := types.VoterVotingPowerLot{
				VoterAddress:        content.RecipientAddress,
				GrantedByProposalId: proposal.Id,
				InitialAmount:       coin.Amount,
				GrantBlockHeight:    uint64(ctx.BlockHeight()),
			}
			recipientAccAddr := sdk.AccAddress(recipient)
			lotKey := collections.Join(recipientAccAddr, proposal.Id)

			if errSetLot := k.VoterPowerLots.Set(ctx, lotKey, lot); errSetLot != nil {
				k.Logger(ctx).Error("CRITICAL: Failed to set voter power lot after token grant", "recipient", content.RecipientAddress, "proposalID", proposal.Id, "amount", coin.String(), "error", errSetLot)
			}
			k.Logger(ctx).Info("Voting power lot granted", "recipient", content.RecipientAddress, "proposalID", proposal.Id, "amount", coin.String())
		}
	}
	return nil
}
