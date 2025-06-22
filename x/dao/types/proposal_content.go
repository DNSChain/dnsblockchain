package types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// ProposalContent defines the common interface for proposal content types.
type ProposalContent interface {
	ProposalRoute() string
	ProposalType() string
	ValidateBasic() error
}

// Implementaciones para AddTldProposalContent
func (m *AddTldProposalContent) ProposalRoute() string { return ModuleName }
func (m *AddTldProposalContent) ProposalType() string  { return "AddTld" }
func (m *AddTldProposalContent) ValidateBasic() error {
	if m.Tld == "" {
		// Usar un error del módulo si está definido, o un error genérico del SDK
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "TLD in proposal content cannot be empty")
	}
	// Añadir más validaciones de formato de TLD si es necesario
	return nil
}

// Implementaciones para RequestTokensProposalContent
func (m *RequestTokensProposalContent) ProposalRoute() string { return ModuleName }
func (m *RequestTokensProposalContent) ProposalType() string  { return "RequestTokens" }
func (m *RequestTokensProposalContent) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.RecipientAddress); err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid recipient address in proposal content: %v", err)
	}
	if err := m.AmountRequested.Validate(); err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidCoins, "invalid amount requested in proposal content: %v", err)
	}
	if !m.AmountRequested.IsAllPositive() {
		return errors.Wrap(sdkerrors.ErrInvalidCoins, "requested amount in proposal content must be positive")
	}
	if m.ActivityDescription == "" {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "activity description in proposal content cannot be empty")
	}
	return nil
}
