package keeper

import (
	"context"
	"errors"
	"fmt"
	"net" // <--- AÑADIR ESTA IMPORTACIÓN
	"strings"

	"dnsblockchain/x/dnsblockchain/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) CreateDomain(ctx context.Context, msg *types.MsgCreateDomain) (*types.MsgCreateDomainResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	var err error

	if _, err = k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, fmt.Sprintf("invalid creator address: %s", err))
	}
	if _, err = k.addressCodec.StringToBytes(msg.Owner); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, fmt.Sprintf("invalid owner address: %s", err))
	}

	normalizedName := strings.ToLower(strings.Trim(msg.Name, "."))
	parts := strings.Split(normalizedName, ".")
	if len(parts) != 2 {
		return nil, errorsmod.Wrapf(types.ErrInvalidDomainName, "domain name '%s' must be in 'label.tld' format", msg.Name)
	}
	extractedTLDFromMsgName := parts[1]

	if extractedTLDFromMsgName == "" {
		return nil, errorsmod.Wrapf(types.ErrInvalidDomainName, "TLD could not be extracted from domain name '%s'", msg.Name)
	}

	isPermitted, err := k.IsTLDPermitted(sdkCtx, extractedTLDFromMsgName)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to check if TLD is permitted")
	}
	if !isPermitted {
		return nil, errorsmod.Wrapf(types.ErrTLDNotPermitted, "TLD '%s' from domain name '%s' is not permitted", extractedTLDFromMsgName, msg.Name)
	}

	_, err = k.DomainName.Get(sdkCtx, normalizedName)
	if err == nil {
		return nil, errorsmod.Wrapf(types.ErrDuplicateDomainName, "domain name '%s' already exists", normalizedName)
	}
	if !errors.Is(err, collections.ErrNotFound) {
		return nil, errorsmod.Wrap(err, "failed to check for duplicate domain name in index")
	}

	if len(msg.NsRecords) == 0 {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "at least one NS record (ns_records entry) must be provided")
	}
	for i, nsEntry := range msg.NsRecords {
		if nsEntry == nil {
			return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "ns_records entry %d cannot be nil", i)
		}
		if strings.TrimSpace(nsEntry.Name) == "" {
			return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "name in ns_records entry %d cannot be empty", i)
		}
		for _, ipStr := range nsEntry.Ipv4Addresses {
			if net.ParseIP(ipStr) == nil { // <--- CAMBIO AQUÍ
				return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid IPv4 address '%s' in ns_records entry %d", ipStr, i)
			}
		}
		for _, ipStr := range nsEntry.Ipv6Addresses {
			if net.ParseIP(ipStr) == nil { // <--- CAMBIO AQUÍ
				return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid IPv6 address '%s' in ns_records entry %d", ipStr, i)
			}
		}
	}

	nextID, err := k.DomainSeq.Next(sdkCtx)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "failed to get next id")
	}

	domain := types.Domain{
		Id:         nextID,
		Creator:    msg.Creator,
		Name:       normalizedName,
		Owner:      msg.Owner,
		NsRecords:  msg.NsRecords,
		Expiration: uint64(sdkCtx.BlockTime().AddDate(1, 0, 0).Unix()),
	}

	if err = k.Domain.Set(sdkCtx, nextID, domain); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to set domain by id")
	}

	if err = k.DomainName.Set(sdkCtx, domain.Name, domain.Id); err != nil {
		_ = k.Domain.Remove(sdkCtx, domain.Id)
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to set domain name index")
	}

	return &types.MsgCreateDomainResponse{
		Id: nextID,
	}, nil
}

func (k msgServer) UpdateDomain(ctx context.Context, msg *types.MsgUpdateDomain) (*types.MsgUpdateDomainResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	var err error

	if _, err = k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, fmt.Sprintf("invalid signer address: %s", err))
	}

	val, errGet := k.Domain.Get(sdkCtx, msg.Id)
	if errGet != nil {
		if errors.Is(errGet, collections.ErrNotFound) {
			return nil, errorsmod.Wrap(sdkerrors.ErrKeyNotFound, fmt.Sprintf("key %d doesn't exist", msg.Id))
		}
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to get domain")
	}

	newOwner := val.Owner
	newNsRecords := val.NsRecords

	isCreator := msg.Creator == val.Creator
	isOwner := msg.Creator == val.Owner

	if !isCreator && !isOwner {
		return nil, errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "signer %s is neither the creator nor the owner of domain %d", msg.Creator, msg.Id)
	}

	changed := false
	if isCreator {
		if msg.Owner != "" && msg.Owner != val.Owner {
			if _, err = k.addressCodec.StringToBytes(msg.Owner); err != nil {
				return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, fmt.Sprintf("invalid new owner address in message: %s", err))
			}
			newOwner = msg.Owner
			changed = true
		}
		if len(msg.NsRecords) > 0 {
			for i, nsEntry := range msg.NsRecords {
				if nsEntry == nil {
					return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "ns_records entry %d cannot be nil", i)
				}
				if strings.TrimSpace(nsEntry.Name) == "" {
					return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "name in ns_records entry %d cannot be empty", i)
				}
				for _, ipStr := range nsEntry.Ipv4Addresses {
					if net.ParseIP(ipStr) == nil { // <--- CAMBIO AQUÍ
						return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid IPv4 address '%s'", ipStr)
					}
				}
				for _, ipStr := range nsEntry.Ipv6Addresses {
					if net.ParseIP(ipStr) == nil { // <--- CAMBIO AQUÍ
						return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid IPv6 address '%s'", ipStr)
					}
				}
			}
			newNsRecords = msg.NsRecords
			changed = true
		}
	} else if isOwner {
		if len(msg.NsRecords) > 0 {
			for i, nsEntry := range msg.NsRecords {
				if nsEntry == nil {
					return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "ns_records entry %d cannot be nil", i)
				}
				if strings.TrimSpace(nsEntry.Name) == "" {
					return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "name in ns_records entry %d cannot be empty", i)
				}
				for _, ipStr := range nsEntry.Ipv4Addresses {
					if net.ParseIP(ipStr) == nil { // <--- CAMBIO AQUÍ
						return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid IPv4 address '%s'", ipStr)
					}
				}
				for _, ipStr := range nsEntry.Ipv6Addresses {
					if net.ParseIP(ipStr) == nil { // <--- CAMBIO AQUÍ
						return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid IPv6 address '%s'", ipStr)
					}
				}
			}
			newNsRecords = msg.NsRecords
			changed = true
		}
		if msg.Owner != "" && msg.Owner != val.Owner {
			return nil, errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "owner %s cannot change domain ownership; only creator %s can", msg.Creator, val.Creator)
		}
	}

	if !changed && msg.Owner == "" && len(msg.NsRecords) == 0 {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "no fields to update were provided or new values are same as existing")
	}

	domain := types.Domain{
		Id:         msg.Id,
		Creator:    val.Creator,
		Name:       val.Name,
		Owner:      newOwner,
		NsRecords:  newNsRecords,
		Expiration: val.Expiration,
	}

	if err = k.Domain.Set(sdkCtx, msg.Id, domain); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to update domain")
	}

	return &types.MsgUpdateDomainResponse{}, nil
}

func (k msgServer) DeleteDomain(ctx context.Context, msg *types.MsgDeleteDomain) (*types.MsgDeleteDomainResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	var err error

	if _, err = k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, fmt.Sprintf("invalid address: %s", err))
	}

	val, errGet := k.Domain.Get(sdkCtx, msg.Id)
	if errGet != nil {
		if errors.Is(errGet, collections.ErrNotFound) {
			return nil, errorsmod.Wrap(sdkerrors.ErrKeyNotFound, fmt.Sprintf("key %d doesn't exist", msg.Id))
		}
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to get domain for deletion")
	}

	if msg.Creator != val.Creator {
		return nil, errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "signer %s is not the creator %s of the domain", msg.Creator, val.Creator)
	}

	domainNameToRemoveFromIndex := val.Name

	if err = k.DomainName.Remove(sdkCtx, domainNameToRemoveFromIndex); err != nil {
		k.Logger(sdkCtx).Error("failed to remove domain name from index during delete, but proceeding", "name", domainNameToRemoveFromIndex, "id", msg.Id, "error", err)
	}

	if err = k.Domain.Remove(sdkCtx, msg.Id); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to delete domain by id")
	}

	return &types.MsgDeleteDomainResponse{}, nil
}

func (k msgServer) HeartbeatDomain(ctx context.Context, msg *types.MsgHeartbeatDomain) (*types.MsgHeartbeatDomainResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	var err error

	if _, err = k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, fmt.Sprintf("invalid address: %s", err))
	}

	domain, errGet := k.Domain.Get(sdkCtx, msg.Id)
	if errGet != nil {
		if errors.Is(errGet, collections.ErrNotFound) {
			return nil, errorsmod.Wrap(sdkerrors.ErrKeyNotFound, fmt.Sprintf("domain id %d not found", msg.Id))
		}
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to get domain for heartbeat")
	}

	if msg.Creator != domain.Creator && msg.Creator != domain.Owner {
		return nil, errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "signer %s is neither the creator (%s) nor the owner (%s)", msg.Creator, domain.Creator, domain.Owner)
	}

	domain.Expiration = uint64(sdkCtx.BlockTime().AddDate(1, 0, 0).Unix())

	if err = k.Domain.Set(sdkCtx, msg.Id, domain); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to update domain expiration for heartbeat")
	}

	return &types.MsgHeartbeatDomainResponse{}, nil
}

func (k msgServer) TransferDomain(goCtx context.Context, msg *types.MsgTransferDomain) (*types.MsgTransferDomainResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	var err error

	if _, err = k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, fmt.Sprintf("invalid creator address: %s", err))
	}
	if _, err = k.addressCodec.StringToBytes(msg.NewOwner); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, fmt.Sprintf("invalid new owner address: %s", err))
	}

	domain, errGet := k.Domain.Get(ctx, msg.Id)
	if errGet != nil {
		if errors.Is(errGet, collections.ErrNotFound) {
			return nil, errorsmod.Wrapf(sdkerrors.ErrKeyNotFound, "domain with id %d not found", msg.Id)
		}
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to get domain for transfer")
	}

	if domain.Creator != msg.Creator {
		return nil, errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "signer %s is not the creator %s; only the original creator can transfer the domain", msg.Creator, domain.Creator)
	}

	domain.Creator = msg.NewOwner
	domain.Owner = msg.NewOwner

	if err = k.Domain.Set(ctx, msg.Id, domain); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to transfer domain ownership and creatorship")
	}

	return &types.MsgTransferDomainResponse{}, nil
}
