package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	// sdkerrors "github.com/cosmos/cosmos-sdk/types/errors" // Ya no se usa directamente aquí

	"dnsblockchain/x/dao/types"
	// govtypes "github.com/cosmos/cosmos-sdk/x/gov/types" // Para tipos de eventos si es necesario
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
		// originalStatus := proposal.Status // Comentado o eliminado ya que no se usa actualmente

		logger.Info("Processing proposal in EndBlocker", "proposalID", proposal.Id, "votingEndBlock", proposal.VotingEndBlock, "currentBlock", currentBlockHeight)

		var yesVotes, noVotes, abstainVotes math.Int
		yesVotes = math.ZeroInt()
		noVotes = math.ZeroInt()
		abstainVotes = math.ZeroInt()
		totalVotingPowerVoted := math.ZeroInt()

		prefixRange := collections.NewPrefixedPairRange[uint64, sdk.AccAddress](proposal.Id)
		tallyErr := k.Votes.Walk(sdkCtx, prefixRange, func(key collections.Pair[uint64, sdk.AccAddress], vote types.Vote) (stop bool, err error) {
			votingPower := vote.VotingPower

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

		if tallyErr != nil {
			logger.Error("Error tallying votes for proposal", "proposalID", proposal.Id, "error", tallyErr)
			proposal.Status = types.ProposalStatus_PROPOSAL_STATUS_FAILED
			// El depósito se manejará más adelante basado en este estado FAILED
		} else {
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
				} else if totalSnapshotDec.IsZero() && params.QuorumPercent.IsPositive() {
					// Si el snapshot es cero pero se requiere un quórum positivo, no se puede cumplir.
					logger.Info("Quorum check: TotalVotingPowerAtSnapshot is zero, but quorum > 0% is required.", "proposalID", proposal.Id)
					hasQuorum = false
				} else if totalSnapshotDec.IsZero() && params.QuorumPercent.IsZero() {
					// Si snapshot es cero y quórum es cero, se considera cumplido.
					logger.Info("Quorum check: TotalVotingPowerAtSnapshot is zero and quorum is 0%. Quorum met.", "proposalID", proposal.Id)
					hasQuorum = true
				} else {
					// QuorumPercent ya es math.LegacyDec
					if totalVotedDec.Quo(totalSnapshotDec).GTE(params.QuorumPercent) {
						hasQuorum = true
					}
				}
			} else if params.QuorumPercent.IsZero() { // Snapshot es cero, pero el quórum requerido también es cero.
				logger.Info("Quorum check: TotalVotingPowerAtSnapshot is zero and quorum is 0%. Quorum met.", "proposalID", proposal.Id)
				hasQuorum = true
			} else { // Snapshot es cero, y se requiere un quórum > 0%. No se puede cumplir.
				logger.Info("Quorum check: TotalVotingPowerAtSnapshot is zero, quorum > 0% required. Quorum not met.", "proposalID", proposal.Id)
				hasQuorum = false
			}

			logger.Info("Quorum check result", "proposalID", proposal.Id, "totalVotingPowerVoted", totalVotingPowerVoted.String(), "totalVotingPowerAtSnapshot", proposal.TotalVotingPowerAtSnapshot.String(), "quorumParam", params.QuorumPercent.String(), "hasQuorum", hasQuorum)

			if proposal.Status == types.ProposalStatus_PROPOSAL_STATUS_VOTING_PERIOD { // Solo si no falló antes
				if !hasQuorum {
					logger.Info("Proposal rejected due to not meeting quorum", "proposalID", proposal.Id)
					proposal.Status = types.ProposalStatus_PROPOSAL_STATUS_REJECTED
				} else {
					// Quórum cumplido, ahora verificar umbral de SÍ
					passedVotingThreshold := false
					if totalVotingPowerVoted.IsZero() && !params.MinYesThresholdPercent.IsZero() {
						// Nadie votó con poder, y se requiere un umbral > 0% de SÍ. No pasa.
						logger.Info("No votes with positive voting power cast, and MinYesThresholdPercent > 0.", "proposalID", proposal.Id)
					} else if totalVotingPowerVoted.IsZero() && params.MinYesThresholdPercent.IsZero() {
						// Nadie votó, pero el umbral de SÍ es 0%. Podría interpretarse como que pasa.
						// Sin embargo, es más seguro requerir al menos un voto SÍ si el umbral no es estrictamente para el "no rechazo".
						// Por ahora, si no hay votos SÍ/NO, no pasa a menos que MinYesThresholdPercent sea exactamente cero
						// Y totalVotesNonAbstain sea también cero.
						// Vamos a mantener la lógica: si MinYesThresholdPercent > 0, se necesita que totalVotesNonAbstain > 0.
						logger.Info("No votes with positive power, but MinYesThresholdPercent is 0.", "proposalID", proposal.Id)
						// La lógica de 'passedVotingThreshold' quedará false, lo que es correcto.
					} else { // totalVotingPowerVoted > 0
						totalVotesNonAbstain := yesVotes.Add(noVotes)
						if totalVotesNonAbstain.IsPositive() {
							yesDec, convErrYes := math.LegacyNewDecFromStr(yesVotes.String())
							denominatorDec, convErrDen := math.LegacyNewDecFromStr(totalVotesNonAbstain.String())

							if convErrYes != nil || convErrDen != nil {
								logger.Error("Error converting votes to LegacyDec for YES threshold check", "proposalID", proposal.Id, "errYes", convErrYes, "errDen", convErrDen)
								proposal.Status = types.ProposalStatus_PROPOSAL_STATUS_FAILED
							} else {
								minThresholdDec := params.MinYesThresholdPercent
								if !denominatorDec.IsZero() {
									percentageYes := yesDec.Quo(denominatorDec)
									if percentageYes.GTE(minThresholdDec) {
										passedVotingThreshold = true
									}
									logger.Info("Proposal YES threshold tally", "proposalID", proposal.Id, "yes", yesVotes.String(), "no", noVotes.String(), "percentageYes", percentageYes.String(), "threshold", minThresholdDec.String(), "passedThreshold", passedVotingThreshold)
								} else if params.MinYesThresholdPercent.IsZero() { // No hay votos SI/NO, pero el umbral es 0%
									passedVotingThreshold = true
									logger.Info("No YES/NO votes, but MinYesThresholdPercent is 0%. Threshold met.", "proposalID", proposal.Id)
								} else {
									logger.Info("Denominator for YES threshold is zero (no YES/NO votes with power) and threshold > 0%.", "proposalID", proposal.Id)
								}
							}
						} else if params.MinYesThresholdPercent.IsZero() { // No hay votos SI/NO, pero el umbral es 0%
							passedVotingThreshold = true
							logger.Info("No YES/NO votes, but MinYesThresholdPercent is 0%. Threshold met.", "proposalID", proposal.Id)
						} else {
							logger.Info("No YES/NO votes with positive power cast for proposal, and MinYesThresholdPercent > 0%.", "proposalID", proposal.Id)
						}
					}

					if passedVotingThreshold {
						proposal.Status = types.ProposalStatus_PROPOSAL_STATUS_PASSED
					} else if proposal.Status == types.ProposalStatus_PROPOSAL_STATUS_VOTING_PERIOD { // Solo si no falló antes
						proposal.Status = types.ProposalStatus_PROPOSAL_STATUS_REJECTED
						logger.Info("Proposal rejected (did not pass YES threshold)", "proposalID", proposal.Id)
					}
				}
			}
		}

		// ---- Lógica de manejo de depósitos y ejecución ----
		// Esta sección se ejecuta después de que el estado de la propuesta (PASSED, REJECTED, FAILED) se ha determinado.
		depositToHandle := params.ProposalSubmissionDeposit
		var proposalContent types.ProposalContent
		if err := k.cdc.UnpackAny(proposal.Content, &proposalContent); err == nil {
			if _, ok := proposalContent.(*types.AddTldProposalContent); ok {
				depositToHandle = params.AddTldProposalCost
			}
		} else {
			logger.Error("Failed to unpack proposal content for deposit handling in EndBlocker", "proposalID", proposal.Id, "error", err)
			if proposal.Status != types.ProposalStatus_PROPOSAL_STATUS_REJECTED { // Si no fue ya rechazada por quórum/umbral
				proposal.Status = types.ProposalStatus_PROPOSAL_STATUS_FAILED // Marcar como fallida si no se puede desempaquetar
			}
		}

		if proposal.Status == types.ProposalStatus_PROPOSAL_STATUS_PASSED {
			logger.Info("Proposal passed voting and quorum, attempting execution", "proposalID", proposal.Id)
			execErr := k.executeProposal(sdkCtx, proposal)
			if execErr != nil {
				logger.Error("Failed to execute proposal", "proposalID", proposal.Id, "error", execErr)
				proposal.Status = types.ProposalStatus_PROPOSAL_STATUS_FAILED
				// Quemar depósito si la ejecución falla
				if !depositToHandle.IsZero() {
					if errBurn := k.bankKeeper.BurnCoins(sdkCtx, types.ModuleName, depositToHandle); errBurn != nil {
						logger.Error("CRITICAL: Failed to burn deposit after execution failure", "proposalID", proposal.Id, "deposit", depositToHandle.String(), "error", errBurn)
					} else {
						logger.Info("Deposit burned after execution failure", "proposalID", proposal.Id, "amount", depositToHandle.String())
					}
				}
			} else {
				proposal.Status = types.ProposalStatus_PROPOSAL_STATUS_EXECUTED
				logger.Info("Proposal executed successfully, refunding deposit", "proposalID", proposal.Id)
				// Devolver depósito si la ejecución es exitosa
				if !depositToHandle.IsZero() {
					proposerAddr, addrErr := k.addressCodec.StringToBytes(proposal.Proposer)
					if addrErr != nil {
						logger.Error("Failed to parse proposer address for refund", "proposalID", proposal.Id, "proposer", proposal.Proposer, "error", addrErr)
					} else {
						if errRefund := k.bankKeeper.SendCoinsFromModuleToAccount(sdkCtx, types.ModuleName, sdk.AccAddress(proposerAddr), depositToHandle); errRefund != nil {
							logger.Error("CRITICAL: Failed to refund deposit for executed proposal", "proposalID", proposal.Id, "proposer", proposal.Proposer, "deposit", depositToHandle.String(), "error", errRefund)
						} else {
							logger.Info("Deposit refunded for executed proposal", "proposalID", proposal.Id, "amount", depositToHandle.String())
						}
					}
				}
			}
		} else if proposal.Status == types.ProposalStatus_PROPOSAL_STATUS_REJECTED || proposal.Status == types.ProposalStatus_PROPOSAL_STATUS_FAILED {
			// Propuesta fue RECHAZADA (por quórum o umbral de SÍ) o FALLÓ (por error de unpack o ejecución)
			logger.Info("Proposal was rejected or failed, burning deposit", "proposalID", proposal.Id, "final_status", proposal.Status.String())
			if !depositToHandle.IsZero() {
				if errBurn := k.bankKeeper.BurnCoins(sdkCtx, types.ModuleName, depositToHandle); errBurn != nil {
					logger.Error("CRITICAL: Failed to burn deposit for rejected/failed proposal", "proposalID", proposal.Id, "deposit", depositToHandle.String(), "error", errBurn)
				} else {
					logger.Info("Deposit burned for rejected/failed proposal", "proposalID", proposal.Id, "amount", depositToHandle.String())
				}
			}
		}

		if err := k.SetProposal(sdkCtx, proposal); err != nil {
			logger.Error("Failed to set final proposal status after processing", "proposalID", proposal.Id, "error", err)
			return errorsmod.Wrapf(err, "failed to set proposal %d final status", proposal.Id)
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
