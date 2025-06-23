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
	addTldProposalCost sdk.Coins,
	quorumPercent math.LegacyDec, // <-- NUEVO PARÁMETRO
) DaoParams {
	return DaoParams{
		VotingPeriodBlocks:                votingPeriodBlocks,
		ProposalSubmissionDeposit:         proposalSubmissionDeposit,
		MinYesThresholdPercent:            minYesThresholdPercent,
		VotingTokenDenom:                  votingTokenDenom,
		VotingPowerDecayDurationBlocks:    votingPowerDecayDurationBlocks,
		ValidatorRewardVotingTokensAmount: validatorRewardVotingTokensAmount,
		AddTldProposalCost:                addTldProposalCost,
		QuorumPercent:                     quorumPercent, // <-- ASIGNAR
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() DaoParams {
	votingPeriod := uint64(500)
	decayDuration := uint64(100)
	votingTokenDenom := "udns"
	validatorRewardAmount := sdk.NewInt64Coin(votingTokenDenom, 0)

	return NewParams(
		votingPeriod,
		sdk.NewCoins(sdk.NewInt64Coin(votingTokenDenom, 10000000)),
		math.LegacyMustNewDecFromStr("0.50"),
		votingTokenDenom,
		decayDuration,
		validatorRewardAmount,
		sdk.NewCoins(sdk.NewInt64Coin(votingTokenDenom, 5000000)),
		math.LegacyMustNewDecFromStr("0.334"), // Quórum del 33.4% por defecto
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
	if !p.ProposalSubmissionDeposit.IsAllPositive() && !p.ProposalSubmissionDeposit.IsZero() {
		return errors.Wrap(sdkerrors.ErrInvalidCoins, "proposal submission deposit must be positive if not zero")
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
	// votingPowerDecayDurationBlocks puede ser 0

	if err := p.ValidatorRewardVotingTokensAmount.Validate(); err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidCoins, "invalid validator reward voting tokens amount: %v", err)
	}
	if !p.ValidatorRewardVotingTokensAmount.IsZero() && p.ValidatorRewardVotingTokensAmount.Denom != p.VotingTokenDenom {
		return fmt.Errorf("validator reward token denom (%s) must match voting token denom (%s) if amount is non-zero",
			p.ValidatorRewardVotingTokensAmount.Denom, p.VotingTokenDenom)
	}
	if err := p.AddTldProposalCost.Validate(); err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidCoins, "invalid add TLD proposal cost: %v", err)
	}
	if !p.AddTldProposalCost.IsAllPositive() && !p.AddTldProposalCost.IsZero() {
		return errors.Wrap(sdkerrors.ErrInvalidCoins, "add TLD proposal cost must be positive if not zero")
	}

	// Validar QuorumPercent
	if p.QuorumPercent.IsNil() {
		return fmt.Errorf("quorum percent cannot be nil")
	}
	if p.QuorumPercent.IsNegative() {
		return fmt.Errorf("quorum percent cannot be negative: %s", p.QuorumPercent)
	}
	if p.QuorumPercent.GT(math.LegacyOneDec()) {
		return fmt.Errorf("quorum percent cannot be greater than 1: %s", p.QuorumPercent)
	}

	return nil
}
