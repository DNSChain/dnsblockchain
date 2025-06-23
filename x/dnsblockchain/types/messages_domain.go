package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// ---------- MsgCreateDomain ----------

func NewMsgCreateDomain(creator string, name string, owner string, expiration uint64, ns string) *MsgCreateDomain {
	return &MsgCreateDomain{
		Creator:    creator,
		Name:       name,
		Owner:      owner,
		Expiration: expiration,
		Ns:         ns,
	}
}

// ValidateBasic for MsgCreateDomain (auto-generated in tx.pb.go, but can be overridden or extended here if needed)
// For now, we rely on the generated one and specific keeper validations.
// func (msg *MsgCreateDomain) ValidateBasic() error {
// 	// ...
// 	return nil
// }

// ---------- MsgUpdateDomain ----------

func NewMsgUpdateDomain(creator string, id uint64, owner string, expiration uint64, ns string) *MsgUpdateDomain {
	return &MsgUpdateDomain{
		Id:         id,
		Creator:    creator,
		Owner:      owner,
		Expiration: expiration,
		Ns:         ns,
	}
}

// ValidateBasic for MsgUpdateDomain (auto-generated in tx.pb.go)
// We are relaxing ID check here if it were to be strict.
// It is generally better to let the keeper handle "not found" for ID 0 if 0 is a valid ID.
// If the generated ValidateBasic is too strict, this is where you'd override.
// For now, assuming generated one is okay or ID 0 being invalid for update is acceptable.
// func (msg *MsgUpdateDomain) ValidateBasic() error {
// 	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
// 		return sdkerrors.ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
// 	}
// 	// If ID 0 needs to be updatable, remove or adjust ID checks.
// 	return nil
// }

// ---------- MsgDeleteDomain ----------

func NewMsgDeleteDomain(creator string, id uint64) *MsgDeleteDomain {
	return &MsgDeleteDomain{
		Id:      id,
		Creator: creator,
	}
}

// ValidateBasic for MsgDeleteDomain (auto-generated in tx.pb.go)
// Similar to Update, if ID 0 needs to be deletable, ensure validation allows it.
// func (msg *MsgDeleteDomain) ValidateBasic() error {
// 	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
// 		return sdkerrors.ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
// 	}
// 	// If ID 0 needs to be deletable, remove or adjust ID checks.
// 	return nil
// }

// ---------- MsgTransferDomain ----------

func NewMsgTransferDomain(creator string, id uint64, newOwner string) *MsgTransferDomain {
	return &MsgTransferDomain{
		Creator:  creator,
		Id:       id,
		NewOwner: newOwner,
	}
}

func (msg *MsgTransferDomain) Route() string {
	return RouterKey
}

func (msg *MsgTransferDomain) Type() string {
	return "TransferDomain"
}

func (msg *MsgTransferDomain) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgTransferDomain) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgTransferDomain) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.NewOwner); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid new owner address: %s", err)
	}
	// ID 0 es un ID válido si la secuencia comienza en 0.
	// La validación original `if msg.Id == 0` era demasiado estricta.
	// Si quisiéramos que los IDs empiecen estrictamente en 1, cambiaríamos la lógica del keeper.
	// Por ahora, permitimos ID 0.
	return nil
}

// ---------- MsgHeartbeatDomain ----------

func NewMsgHeartbeatDomain(creator string, id uint64) *MsgHeartbeatDomain {
	return &MsgHeartbeatDomain{
		Creator: creator,
		Id:      id,
	}
}

func (msg *MsgHeartbeatDomain) Route() string {
	return RouterKey
}

func (msg *MsgHeartbeatDomain) Type() string {
	return "HeartbeatDomain"
}

func (msg *MsgHeartbeatDomain) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgHeartbeatDomain) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgHeartbeatDomain) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
	}
	// ID 0 es un ID válido si la secuencia comienza en 0.
	return nil
}
