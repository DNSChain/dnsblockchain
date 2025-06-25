// dnsblockchain/x/dnsblockchain/types/messages_domain.go
package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// ---------- MsgCreateDomain ----------
func NewMsgCreateDomain(creator string, name string, owner string, nsRecords []*NSRecordWithIP) *MsgCreateDomain {
	return &MsgCreateDomain{
		Creator:   creator,
		Name:      name,
		Owner:     owner,
		NsRecords: nsRecords,
	}
}

func (msg *MsgCreateDomain) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid owner address: %s", err)
	}
	if msg.Name == "" {
		return sdkerrors.ErrInvalidRequest.Wrap("name cannot be empty")
	}
	if len(msg.NsRecords) == 0 {
		return sdkerrors.ErrInvalidRequest.Wrap("ns_records cannot be empty")
	}
	for i, nsEntry := range msg.NsRecords {
		if nsEntry == nil {
			return sdkerrors.ErrInvalidRequest.Wrapf("ns_records entry %d cannot be nil", i)
		}
		if nsEntry.Name == "" {
			return sdkerrors.ErrInvalidRequest.Wrapf("name in ns_records entry %d cannot be empty", i)
		}
	}
	return nil
}

func (msg *MsgCreateDomain) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

// ---------- MsgUpdateDomain ----------
func NewMsgUpdateDomain(creator string, id uint64, owner string, nsRecords []*NSRecordWithIP) *MsgUpdateDomain {
	return &MsgUpdateDomain{
		Creator:   creator,
		Id:        id,
		Owner:     owner,
		NsRecords: nsRecords,
	}
}

func (msg *MsgUpdateDomain) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
	}
	if msg.Owner != "" {
		if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
			return sdkerrors.ErrInvalidAddress.Wrapf("invalid owner address: %s", err)
		}
	}
	for i, nsEntry := range msg.NsRecords {
		if nsEntry == nil {
			return sdkerrors.ErrInvalidRequest.Wrapf("ns_records entry %d cannot be nil if provided", i)
		}
		if nsEntry.Name == "" {
			return sdkerrors.ErrInvalidRequest.Wrapf("name in ns_records entry %d cannot be empty if provided", i)
		}
	}
	if msg.Owner == "" && len(msg.NsRecords) == 0 {
		return sdkerrors.ErrInvalidRequest.Wrap("either new owner or new ns_records must be provided for update")
	}
	return nil
}

func (msg *MsgUpdateDomain) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

// ---------- MsgDeleteDomain ----------
func NewMsgDeleteDomain(creator string, id uint64) *MsgDeleteDomain {
	return &MsgDeleteDomain{
		Creator: creator,
		Id:      id,
	}
}

func (msg *MsgDeleteDomain) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
	}
	return nil
}

func (msg *MsgDeleteDomain) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

// ---------- MsgTransferDomain ----------
func NewMsgTransferDomain(creator string, id uint64, newOwner string) *MsgTransferDomain {
	return &MsgTransferDomain{
		Creator:  creator,
		Id:       id,
		NewOwner: newOwner,
	}
}

func (msg *MsgTransferDomain) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.NewOwner); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid new owner address: %s", err)
	}
	return nil
}

func (msg *MsgTransferDomain) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

// ---------- MsgHeartbeatDomain ----------
func NewMsgHeartbeatDomain(creator string, id uint64) *MsgHeartbeatDomain {
	return &MsgHeartbeatDomain{
		Creator: creator,
		Id:      id,
	}
}

func (msg *MsgHeartbeatDomain) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
	}
	return nil
}

func (msg *MsgHeartbeatDomain) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}
