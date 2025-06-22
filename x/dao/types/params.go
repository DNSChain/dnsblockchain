package types

import (
	"fmt"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// DefaultGenesis returns the default dao genesis state
// func DefaultGenesis() *GenesisState {
// 	return &GenesisState{
// 		Params: DefaultParams(),
// 	}
// }

// NewParams creates a new Params instance
func NewParams(
	votingPeriodBlocks uint64,
	proposalSubmissionDeposit sdk.Coins,
	minYesThresholdPercent math.LegacyDec,
	votingTokenDenom string, // Este es el que cambiaremos
	votingPowerDecayDurationBlocks uint64,
	validatorRewardVotingTokensAmount sdk.Coin, // Y este
) Params {
	return Params{
		VotingPeriodBlocks:                votingPeriodBlocks,
		ProposalSubmissionDeposit:         proposalSubmissionDeposit,
		MinYesThresholdPercent:            minYesThresholdPercent,
		VotingTokenDenom:                  votingTokenDenom,
		VotingPowerDecayDurationBlocks:    votingPowerDecayDurationBlocks,
		ValidatorRewardVotingTokensAmount: validatorRewardVotingTokensAmount,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	votingPeriod := uint64(500)
	decayDuration := uint64(2592000)

	// Define tu nuevo denom. Es común usar un prefijo 'u' para la unidad base (micro).
	// Por ejemplo, 'udnsc' para micro-dnsc, y luego 'dnsc' sería 1,000,000 udnsc.
	// Por ahora, usaremos "dnsc" directamente, pero considera "udnsc".
	// Si usas "udnsc", ajusta las cantidades en consecuencia.
	// Para el ejemplo, usaré "dnsc" directamente y cantidades más pequeñas.
	// Si quieres que "dnsc" sea la unidad principal con 6 decimales, entonces usa "udnsc"
	// y multiplica las cantidades por 1,000,000.

	// Opción 1: "dnsc" es la unidad base (sin decimales implícitos por el nombre)
	// votingTokenDenom := "dnsc"
	// validatorRewardAmount := sdk.NewInt64Coin(votingTokenDenom, 1) // 1 dnsc de recompensa

	// Opción 2: "udnsc" es la unidad base (micro-dnsc, 6 decimales para "dnsc") - RECOMENDADO
	votingTokenDenom := "udnsc" // micro-dnsc
	// 1 dnsc = 1,000,000 udnsc. Recompensa de, por ejemplo, 1 dnsc por validador/bloque
	validatorRewardAmount := sdk.NewInt64Coin(votingTokenDenom, 1000000)

	return NewParams(
		votingPeriod,
		sdk.NewCoins(sdk.NewInt64Coin("stake", 10000000)), // Depósito de 10 stake (puedes cambiar 'stake' a tu BondDenom real si es diferente)
		math.LegacyMustNewDecFromStr("0.50"),
		votingTokenDenom, // Usar la variable
		decayDuration,
		validatorRewardAmount, // Usar la variable
	)
}

// Validate validates the set of params
func (p Params) Validate() error {
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

	if err := sdk.ValidateDenom(p.VotingTokenDenom); err != nil { // Validará "udnsc" o "dnsc"
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid voting token denom %s: %v", p.VotingTokenDenom, err)
	}
	if p.VotingPowerDecayDurationBlocks == 0 {
		return fmt.Errorf("voting power decay duration blocks cannot be zero")
	}
	if err := p.ValidatorRewardVotingTokensAmount.Validate(); err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidCoins, "invalid validator reward voting tokens amount: %v", err)
	}
	// La validación del denom de la recompensa ahora usará la variable
	if !p.ValidatorRewardVotingTokensAmount.IsZero() && p.ValidatorRewardVotingTokensAmount.Denom != p.VotingTokenDenom {
		return fmt.Errorf("validator reward token denom (%s) must match voting token denom (%s) if amount is non-zero",
			p.ValidatorRewardVotingTokensAmount.Denom, p.VotingTokenDenom)
	}
	return nil
}
