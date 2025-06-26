package types

import (
	"fmt"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// NewParams crea una nueva instancia de Params.
func NewParams(domainCreationFee sdk.Coins) Params {
	return Params{
		DomainCreationFee: domainCreationFee,
	}
}

// DefaultParams devuelve un conjunto de parámetros por defecto.
func DefaultParams() Params {
	return NewParams(sdk.NewCoins(sdk.NewInt64Coin("udns", 20000000))) // 20 dns = 20,000,000 udns
}

// Validate valida el conjunto de parámetros.
func (p Params) Validate() error {
	if err := validateDomainCreationFee(p.DomainCreationFee); err != nil {
		return err
	}
	return nil
}

func validateDomainCreationFee(i interface{}) error {
	v, ok := i.(sdk.Coins)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if err := v.Validate(); err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidCoins, "invalid domain creation fee: %v", err)
	}
	// Asegurarse de que la tarifa sea no negativa. Cero podría ser válido si no se cobra tarifa.
	// Si debe ser estrictamente positiva y no cero:
	if v.IsAnyNegative() {
		return errors.Wrap(sdkerrors.ErrInvalidCoins, "domain creation fee cannot be negative")
	}
	if v.IsZero() { // Permitir tarifa cero, pero puedes cambiar esto
		// return errors.Wrap(sdkerrors.ErrInvalidCoins, "domain creation fee cannot be zero")
	}
	return nil
}
