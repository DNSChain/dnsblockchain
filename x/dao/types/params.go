package types

import (
	"fmt"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// NewParams creates a new Params instance
func NewParams(
	votingPeriodBlocks uint64,
	proposalSubmissionDeposit sdk.Coins,
	minYesThresholdPercent math.LegacyDec,
	votingTokenDenom string,
	votingPowerDecayDurationBlocks uint64,
	validatorRewardVotingTokensAmount sdk.Coin,
	addTldProposalCost sdk.Coins, // <-- NUEVO PARÁMETRO
) DaoParams {
	return DaoParams{
		VotingPeriodBlocks:                votingPeriodBlocks,
		ProposalSubmissionDeposit:         proposalSubmissionDeposit,
		MinYesThresholdPercent:            minYesThresholdPercent,
		VotingTokenDenom:                  votingTokenDenom,
		VotingPowerDecayDurationBlocks:    votingPowerDecayDurationBlocks,
		ValidatorRewardVotingTokensAmount: validatorRewardVotingTokensAmount,
		AddTldProposalCost:                addTldProposalCost, // <-- ASIGNAR
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() DaoParams {
	votingPeriod := uint64(500) // O el valor que prefieras para pruebas
	decayDuration := uint64(2592000)
	votingTokenDenom := "udnsc"
	validatorRewardAmount := sdk.NewInt64Coin(votingTokenDenom, 1000000)

	return NewParams(
		votingPeriod,
		sdk.NewCoins(sdk.NewInt64Coin("stake", 10000000)), // Depósito general
		math.LegacyMustNewDecFromStr("0.50"),
		votingTokenDenom,
		decayDuration,
		validatorRewardAmount,
		sdk.NewCoins(sdk.NewInt64Coin("stake", 5000000)), // Costo de propuesta TLD: 5 stake
	)
}

// Validate validates the set of params
func (p DaoParams) Validate() error {
	if p.VotingPeriodBlocks == 0 {
		return fmt.Errorf("voting period blocks cannot be zero")
	}
	if err := p.ProposalSubmissionDeposit.Validate(); err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidCoins, "invalid proposal submission deposit: %v", err)
	}

	if p.MinYesThresholdPercent.IsNil() {
		return fmt.Errorf("min yes threshold percent cannot be nil")
	}
	if p.MinYesThresholdPercent.IsNegative() {
		return fmt.Errorf("min yes threshold percent cannot be negative: %s", p.MinYesThresholdPercent)
	}
	if p.MinYesThresholdPercent.GT(math.LegacyOneDec()) {
		return fmt.Errorf("min yes threshold percent cannot be greater than 1: %s", p.MinYesThresholdPercent)
	}

	if err := sdk.ValidateDenom(p.VotingTokenDenom); err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid voting token denom %s: %v", p.VotingTokenDenom, err)
	}
	if p.VotingPowerDecayDurationBlocks == 0 {
		return fmt.Errorf("voting power decay duration blocks cannot be zero")
	}
	if err := p.ValidatorRewardVotingTokensAmount.Validate(); err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidCoins, "invalid validator reward voting tokens amount: %v", err)
	}
	if !p.ValidatorRewardVotingTokensAmount.IsZero() && p.ValidatorRewardVotingTokensAmount.Denom != p.VotingTokenDenom {
		return fmt.Errorf("validator reward token denom (%s) must match voting token denom (%s) if amount is non-zero",
			p.ValidatorRewardVotingTokensAmount.Denom, p.VotingTokenDenom)
	}
	if err := p.AddTldProposalCost.Validate(); err != nil { // <-- VALIDAR NUEVO PARÁMETRO
		return errors.Wrapf(sdkerrors.ErrInvalidCoins, "invalid add TLD proposal cost: %v", err)
	}
	if !p.AddTldProposalCost.IsAllPositive() && !p.AddTldProposalCost.IsZero() { // Permitir cero, pero si no es cero, debe ser positivo
		return errors.Wrap(sdkerrors.ErrInvalidCoins, "add TLD proposal cost must be positive if not zero")
	}

	return nil
}
