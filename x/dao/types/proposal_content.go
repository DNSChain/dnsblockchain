package types

import (
	"regexp"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/gogoproto/proto" // Asegúrate de importar esto
)

// ProposalContent defines the common interface for proposal content types.
type ProposalContent interface {
	proto.Message // Embeber proto.Message para asegurar que los tipos concretos lo implementen
	ProposalRoute() string
	ProposalType() string
	ValidateBasic() error
}

// Implementaciones para AddTldProposalContent
func (m *AddTldProposalContent) ProposalRoute() string { return ModuleName }
func (m *AddTldProposalContent) ProposalType() string  { return "AddTld" }

var tldRegexp = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)

func (m *AddTldProposalContent) ValidateBasic() error {
	if m.Tld == "" {
		// Usar un error del módulo si está definido, o un error genérico del SDK
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "TLD in proposal content cannot be empty")
	}

	if len(m.Tld) < 2 || len(m.Tld) > 63 {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "TLD length must be between 2 and 63 characters, got %d", len(m.Tld))
	}

	if !tldRegexp.MatchString(m.Tld) {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "TLD '%s' contains invalid characters or format. Only alphanumerics and hyphens are allowed. Hyphens cannot be at the start or end.", m.Tld)
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
