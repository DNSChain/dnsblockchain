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
		proposal := proposalsToProcess[i] // Evitar problemas de referencia de bucle en goroutines (no aplica aquí pero es buena práctica)
		logger.Info("Processing proposal in EndBlocker", "proposalID", proposal.Id, "votingEndBlock", proposal.VotingEndBlock, "currentBlock", currentBlockHeight)

		var yesVotes, noVotes, abstainVotes math.Int
		yesVotes = math.ZeroInt()
		noVotes = math.ZeroInt()
		abstainVotes = math.ZeroInt()
		totalVotingPowerVoted := math.ZeroInt()

		prefixRange := collections.NewPrefixedPairRange[uint64, sdk.AccAddress](proposal.Id)
		err = k.Votes.Walk(sdkCtx, prefixRange, func(key collections.Pair[uint64, sdk.AccAddress], vote types.Vote) (stop bool, err error) {
			// votingPower := math.NewInt(1) // Placeholder - Se reemplaza por el poder de voto almacenado
			votingPower := vote.VotingPower // Usar el poder de voto almacenado

			switch vote.Option {
			case types.VoteOption_VOTE_OPTION_YES:
				yesVotes = yesVotes.Add(votingPower)
			case types.VoteOption_VOTE_OPTION_NO:
				noVotes = noVotes.Add(votingPower)
			case types.VoteOption_VOTE_OPTION_ABSTAIN:
				abstainVotes = abstainVotes.Add(votingPower)
			}
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

		passed := false
		if totalVotingPowerVoted.IsZero() {
			logger.Info("No votes with positive voting power cast for proposal", "proposalID", proposal.Id)
		} else {
			totalVotesNonAbstain := yesVotes.Add(noVotes)
			if totalVotesNonAbstain.IsPositive() {
				yesDec, convErr := math.LegacyNewDecFromStr(yesVotes.String())
				if convErr != nil {
					logger.Error("Error converting yesVotes to LegacyDec", "proposalID", proposal.Id, "error", convErr)
					proposal.Status = types.ProposalStatus_PROPOSAL_STATUS_FAILED
					if setErr := k.SetProposal(sdkCtx, proposal); setErr != nil {
						logger.Error("Failed to set proposal to FAILED after dec conversion error (yesVotes)", "proposalID", proposal.Id, "error", setErr)
					}
					continue
				}

				// Decidir si el umbral se calcula contra el total de votos emitidos (no abstenciones)
				// o contra el TotalVotingPowerAtSnapshot de la propuesta.
				// Por ahora, lo mantenemos contra totalVotesNonAbstain.
				// Si se quisiera contra el snapshot total:
				// denominatorDec, convErr := math.LegacyNewDecFromStr(proposal.TotalVotingPowerAtSnapshot.String())
				denominatorDec, convErr := math.LegacyNewDecFromStr(totalVotesNonAbstain.String())
				if convErr != nil {
					logger.Error("Error converting denominator to LegacyDec", "proposalID", proposal.Id, "error", convErr)
					proposal.Status = types.ProposalStatus_PROPOSAL_STATUS_FAILED
					if setErr := k.SetProposal(sdkCtx, proposal); setErr != nil {
						logger.Error("Failed to set proposal to FAILED after dec conversion error (denominator)", "proposalID", proposal.Id, "error", setErr)
					}
					continue
				}

				minThresholdDec := params.MinYesThresholdPercent

				if !denominatorDec.IsZero() { // Evitar división por cero
					percentageYes := yesDec.Quo(denominatorDec)
					if percentageYes.GTE(minThresholdDec) {
						passed = true
					}
					logger.Info("Proposal tally", "proposalID", proposal.Id, "yes", yesVotes.String(), "no", noVotes.String(), "abstain", abstainVotes.String(), "total_voted_power", totalVotingPowerVoted.String(), "percentageYes", percentageYes.String(), "threshold", minThresholdDec.String(), "denominator_used", denominatorDec.String(), "passed", passed)
				} else {
					logger.Info("No YES/NO votes cast or denominator is zero", "proposalID", proposal.Id)
				}
			} else {
				logger.Info("No YES/NO votes with positive power cast for proposal", "proposalID", proposal.Id)
			}
		}

		depositToHandle := params.ProposalSubmissionDeposit // Depósito general por defecto
		var proposalContent types.ProposalContent
		if err := k.cdc.UnpackAny(proposal.Content, &proposalContent); err == nil {
			if _, ok := proposalContent.(*types.AddTldProposalContent); ok {
				depositToHandle = params.AddTldProposalCost
			}
		} else {
			logger.Error("Failed to unpack proposal content for deposit handling in EndBlocker", "proposalID", proposal.Id, "error", err)
		}

		if passed {
			proposal.Status = types.ProposalStatus_PROPOSAL_STATUS_PASSED
			logger.Info("Proposal passed", "proposalID", proposal.Id)

			execErr := k.executeProposal(sdkCtx, proposal) // Pasamos la 'proposal' actualizada con los conteos
			if execErr != nil {
				logger.Error("Failed to execute proposal", "proposalID", proposal.Id, "error", execErr)
				proposal.Status = types.ProposalStatus_PROPOSAL_STATUS_FAILED
				// Si la ejecución falla, el depósito específico de TLD (o general) se quema
				if !depositToHandle.IsZero() {
					// No necesitamos el proposerAddr para quemar desde el módulo
					if errBurn := k.bankKeeper.BurnCoins(sdkCtx, types.ModuleName, depositToHandle); errBurn != nil {
						logger.Error("Failed to burn deposit after execution failure", "proposalID", proposal.Id, "deposit", depositToHandle.String(), "error", errBurn)
					} else {
						logger.Info("Deposit burned after execution failure", "proposalID", proposal.Id, "amount", depositToHandle.String())
					}
				}
			} else {
				proposal.Status = types.ProposalStatus_PROPOSAL_STATUS_EXECUTED
				logger.Info("Proposal executed successfully", "proposalID", proposal.Id)
				// Si la ejecución es exitosa, el depósito (general o TLD) se devuelve
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
		} else { // Propuesta no pasó la votación
			if proposal.Status == types.ProposalStatus_PROPOSAL_STATUS_VOTING_PERIOD { // Solo cambiar a REJECTED si aún estaba en votación
				proposal.Status = types.ProposalStatus_PROPOSAL_STATUS_REJECTED
			}
			logger.Info("Proposal rejected or failed (did not pass vote)", "proposalID", proposal.Id, "status", proposal.Status.String())
			// Quemar depósito si la propuesta es rechazada
			if proposal.Status == types.ProposalStatus_PROPOSAL_STATUS_REJECTED && !depositToHandle.IsZero() {
				if errBurn := k.bankKeeper.BurnCoins(sdkCtx, types.ModuleName, depositToHandle); errBurn != nil {
					logger.Error("Failed to burn deposit for rejected proposal", "proposalID", proposal.Id, "deposit", depositToHandle.String(), "error", errBurn)
				} else {
					logger.Info("Deposit burned for rejected proposal", "proposalID", proposal.Id, "amount", depositToHandle.String())
				}
			}
		}

		// Actualizar la propuesta en el store con el estado final y los conteos
		if err := k.SetProposal(sdkCtx, proposal); err != nil {
			logger.Error("Failed to set proposal status after processing", "proposalID", proposal.Id, "error", err)
			// Considerar si se debe continuar con otras propuestas o devolver error
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
		return k.executeRequestTokensProposal(ctx, c, proposal) // Pasar toda la propuesta
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
		// Si falla el envío, intentar quemar las monedas minteadas para mantener la consistencia del suministro
		burnErr := k.bankKeeper.BurnCoins(ctx, types.ModuleName, content.AmountRequested)
		if burnErr != nil {
			k.Logger(ctx).Error("CRITICAL: failed to burn coins after failed send in proposal execution", "burn_error", burnErr, "original_send_error", err)
			// Podríamos incluso entrar en pánico aquí si la quema falla, ya que el suministro de tokens estaría inflado.
		}
		return errorsmod.Wrapf(err, "failed to send coins to recipient for proposal execution")
	}

	// Otorgar poder de voto al destinatario por los tokens recibidos, si son del denom de voto
	for _, coin := range content.AmountRequested {
		if coin.Denom == params.VotingTokenDenom && coin.Amount.IsPositive() {
			lot := types.VoterVotingPowerLot{
				VoterAddress:        content.RecipientAddress,
				GrantedByProposalId: proposal.Id, // Usamos el ID de la propuesta que se está ejecutando
				InitialAmount:       coin.Amount,
				GrantBlockHeight:    uint64(ctx.BlockHeight()),
			}
			recipientAccAddr := sdk.AccAddress(recipient)
			// La clave debe ser única para cada lote. Usar (VoterAddr, GrantingProposalID) podría no ser único si una cuenta recibe múltiples veces de la misma propuesta (aunque no es el caso ahora).
			// Una forma de asegurar unicidad si eso fuera posible sería (VoterAddr, GrantingProposalID, GrantBlockHeight) o una secuencia para lotes.
			// Por ahora, (VoterAddr, GrantingProposalID) es suficiente ya que una propuesta RequestTokens se ejecuta una vez.
			lotKey := collections.Join(recipientAccAddr, proposal.Id)

			if errSetLot := k.VoterPowerLots.Set(ctx, lotKey, lot); errSetLot != nil {
				k.Logger(ctx).Error("CRITICAL: Failed to set voter power lot after token grant", "recipient", content.RecipientAddress, "proposalID", proposal.Id, "amount", coin.String(), "error", errSetLot)
				// Podríamos considerar devolver este error para fallar la ejecución de la propuesta,
				// lo que haría que el depósito se queme si la propuesta había pasado.
				// return errorsmod.Wrap(errSetLot, "failed to record voting power lot")
			}
			k.Logger(ctx).Info("Voting power lot granted", "recipient", content.RecipientAddress, "proposalID", proposal.Id, "amount", coin.String())
		}
	}
	return nil
}
