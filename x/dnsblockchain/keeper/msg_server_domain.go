package keeper

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"dnsblockchain/x/dnsblockchain/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) CreateDomain(goCtx context.Context, msg *types.MsgCreateDomain) (*types.MsgCreateDomainResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	var err error

	creatorAddr, err := k.addressCodec.StringToBytes(msg.Creator)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, fmt.Sprintf("invalid creator address: %s", err))
	}
	if _, err = k.addressCodec.StringToBytes(msg.Owner); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, fmt.Sprintf("invalid owner address: %s", err))
	}

	// Get module parameters
	params, err := k.Keeper.Params.Get(ctx) // Acceder a Params a través de k.Keeper
	if err != nil {
		k.Keeper.Logger(ctx).Error("Failed to get dnsblockchain module params", "error", err)
		return nil, errorsmod.Wrap(err, "failed to get dnsblockchain module params")
	}

	domainCreationFee := params.DomainCreationFee

	// Charge and burn the domain creation fee
	if !domainCreationFee.IsZero() {
		err = k.Keeper.bankKeeper.SendCoinsFromAccountToModule(ctx, sdk.AccAddress(creatorAddr), types.ModuleName, domainCreationFee)
		if err != nil {
			k.Keeper.Logger(ctx).Error("Failed to send domain creation fee from creator to module", "creator", msg.Creator, "fee", domainCreationFee.String(), "error", err)
			return nil, errorsmod.Wrapf(err, "failed to send domain creation fee from creator %s to module account", msg.Creator)
		}

		err = k.Keeper.bankKeeper.BurnCoins(ctx, types.ModuleName, domainCreationFee)
		if err != nil {
			k.Keeper.Logger(ctx).Error("CRITICAL: Failed to burn domain creation fee from module account", "fee", domainCreationFee.String(), "error", err)
			return nil, errorsmod.Wrapf(err, "failed to burn domain creation fee from module account")
		}
		k.Keeper.Logger(ctx).Info("Domain creation fee collected and burned", "creator", msg.Creator, "amount", domainCreationFee.String())

		// Emit an event for fee payment and burn
		ctx.EventManager().EmitEvents(sdk.Events{
			sdk.NewEvent(
				types.EventTypeDomainFeeCollected,
				sdk.NewAttribute(types.AttributeKeyFeeCollector, msg.Creator),
				sdk.NewAttribute(sdk.AttributeKeyAmount, domainCreationFee.String()),
			),
			sdk.NewEvent(
				types.EventTypeDomainFeeBurned,
				sdk.NewAttribute(types.AttributeKeyBurnerModule, types.ModuleName),
				sdk.NewAttribute(sdk.AttributeKeyAmount, domainCreationFee.String()),
			),
			// EL EVENTO DE TRANSFERDOMAIN SE ELIMINÓ DE AQUÍ, ERA INCORRECTO
		})
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

	isPermitted, err := k.Keeper.IsTLDPermitted(ctx, extractedTLDFromMsgName) // Acceder a IsTLDPermitted a través de k.Keeper
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to check if TLD is permitted")
	}
	if !isPermitted {
		return nil, errorsmod.Wrapf(types.ErrTLDNotPermitted, "TLD '%s' from domain name '%s' is not permitted", extractedTLDFromMsgName, msg.Name)
	}

	_, err = k.Keeper.DomainName.Get(ctx, normalizedName) // Acceder a DomainName a través de k.Keeper
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
			if net.ParseIP(ipStr) == nil {
				return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid IPv4 address '%s' in ns_records entry %d", ipStr, i)
			}
		}
		for _, ipStr := range nsEntry.Ipv6Addresses {
			if net.ParseIP(ipStr) == nil {
				return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid IPv6 address '%s' in ns_records entry %d", ipStr, i)
			}
		}
	}

	nextID, err := k.Keeper.DomainSeq.Next(ctx) // Acceder a DomainSeq a través de k.Keeper
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "failed to get next id")
	}

	domain := types.Domain{
		Id:         nextID,
		Creator:    msg.Creator,
		Name:       normalizedName,
		Owner:      msg.Owner,
		NsRecords:  msg.NsRecords,
		Expiration: uint64(ctx.BlockTime().AddDate(1, 0, 0).Unix()),
	}

	if err = k.Keeper.Domain.Set(ctx, nextID, domain); err != nil { // Acceder a Domain a través de k.Keeper
		if !domainCreationFee.IsZero() {
			k.Keeper.Logger(ctx).Error("CRITICAL: Domain creation fee burned, but failed to set domain in store", "creator", msg.Creator, "fee", domainCreationFee.String(), "error", err)
		}
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to set domain by id")
	}

	if err = k.Keeper.DomainName.Set(ctx, domain.Name, domain.Id); err != nil { // Acceder a DomainName a través de k.Keeper
		_ = k.Keeper.Domain.Remove(ctx, domain.Id) // Acceder a Domain a través de k.Keeper
		if !domainCreationFee.IsZero() {
			k.Keeper.Logger(ctx).Error("CRITICAL: Domain creation fee burned, domain removed, but failed to set domain name index", "creator", msg.Creator, "fee", domainCreationFee.String(), "error", err)
		}
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to set domain name index")
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCreateDomain,
			sdk.NewAttribute(types.AttributeKeyDomainID, fmt.Sprintf("%d", nextID)),
			sdk.NewAttribute(types.AttributeKeyDomainName, normalizedName),
			sdk.NewAttribute(types.AttributeKeyOwner, msg.Owner),
			sdk.NewAttribute(types.AttributeKeyCreator, msg.Creator),
		),
	})

	return &types.MsgCreateDomainResponse{
		Id: nextID,
	}, nil
}

func (k msgServer) UpdateDomain(goCtx context.Context, msg *types.MsgUpdateDomain) (*types.MsgUpdateDomainResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	var err error

	if _, err = k.Keeper.addressCodec.StringToBytes(msg.Creator); err != nil { // Acceder a addressCodec a través de k.Keeper
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, fmt.Sprintf("invalid signer address: %s", err))
	}

	val, errGet := k.Keeper.Domain.Get(ctx, msg.Id) // Acceder a Domain a través de k.Keeper
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
			if _, err = k.Keeper.addressCodec.StringToBytes(msg.Owner); err != nil { // Acceder a addressCodec a través de k.Keeper
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
					if net.ParseIP(ipStr) == nil {
						return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid IPv4 address '%s'", ipStr)
					}
				}
				for _, ipStr := range nsEntry.Ipv6Addresses {
					if net.ParseIP(ipStr) == nil {
						return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid IPv6 address '%s'", ipStr)
					}
				}
			}
			newNsRecords = msg.NsRecords
			changed = true
		}
	} else if isOwner {
		if msg.Owner != "" && msg.Owner != val.Owner {
			return nil, errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "owner %s cannot change domain ownership; only creator %s can", msg.Creator, val.Creator)
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
					if net.ParseIP(ipStr) == nil {
						return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid IPv4 address '%s'", ipStr)
					}
				}
				for _, ipStr := range nsEntry.Ipv6Addresses {
					if net.ParseIP(ipStr) == nil {
						return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid IPv6 address '%s'", ipStr)
					}
				}
			}
			newNsRecords = msg.NsRecords
			changed = true
		}
	}

	if !changed {
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

	if err = k.Keeper.Domain.Set(ctx, msg.Id, domain); err != nil { // Acceder a Domain a través de k.Keeper
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to update domain")
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeUpdateDomain,
			sdk.NewAttribute(types.AttributeKeyDomainID, fmt.Sprintf("%d", msg.Id)),
			sdk.NewAttribute(types.AttributeKeyUpdater, msg.Creator),
		),
	})

	return &types.MsgUpdateDomainResponse{}, nil
}

func (k msgServer) DeleteDomain(goCtx context.Context, msg *types.MsgDeleteDomain) (*types.MsgDeleteDomainResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	var err error

	signerAddr, err := k.Keeper.addressCodec.StringToBytes(msg.Creator)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, fmt.Sprintf("invalid address: %s", err))
	}
	_ = signerAddr // Evita error de no usado, aunque el signer es msg.Creator

	val, errGet := k.Keeper.Domain.Get(ctx, msg.Id)
	if errGet != nil {
		if errors.Is(errGet, collections.ErrNotFound) {
			return nil, errorsmod.Wrapf(sdkerrors.ErrKeyNotFound, "key %d doesn't exist", msg.Id)
		}
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to get domain for deletion")
	}

	// CORRECCIÓN: Solo el PROPIETARIO actual puede eliminar el dominio.
	if msg.Creator != val.Owner {
		return nil, errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "signer %s is not the current owner %s of the domain", msg.Creator, val.Owner)
	}

	domainNameToRemoveFromIndex := val.Name

	if err = k.Keeper.DomainName.Remove(ctx, domainNameToRemoveFromIndex); err != nil {
		k.Keeper.Logger(ctx).Error("failed to remove domain name from index during delete, but proceeding", "name", domainNameToRemoveFromIndex, "id", msg.Id, "error", err)
	}

	if err = k.Keeper.Domain.Remove(ctx, msg.Id); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to delete domain by id")
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeDeleteDomain,
			sdk.NewAttribute(types.AttributeKeyDomainID, fmt.Sprintf("%d", msg.Id)),
			sdk.NewAttribute(types.AttributeKeyDomainName, val.Name),
			sdk.NewAttribute(types.AttributeKeyActor, msg.Creator),
		),
	})

	return &types.MsgDeleteDomainResponse{}, nil
}

func (k msgServer) HeartbeatDomain(goCtx context.Context, msg *types.MsgHeartbeatDomain) (*types.MsgHeartbeatDomainResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	var err error

	if _, err = k.Keeper.addressCodec.StringToBytes(msg.Creator); err != nil { // Acceder a addressCodec a través de k.Keeper
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, fmt.Sprintf("invalid address: %s", err))
	}

	domain, errGet := k.Keeper.Domain.Get(ctx, msg.Id) // Acceder a Domain a través de k.Keeper
	if errGet != nil {
		if errors.Is(errGet, collections.ErrNotFound) {
			return nil, errorsmod.Wrap(sdkerrors.ErrKeyNotFound, fmt.Sprintf("domain id %d not found", msg.Id))
		}
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to get domain for heartbeat")
	}

	if msg.Creator != domain.Creator && msg.Creator != domain.Owner {
		return nil, errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "signer %s is neither the creator (%s) nor the owner (%s)", msg.Creator, domain.Creator, domain.Owner)
	}

	domain.Expiration = uint64(ctx.BlockTime().AddDate(1, 0, 0).Unix())

	if err = k.Keeper.Domain.Set(ctx, msg.Id, domain); err != nil { // Acceder a Domain a través de k.Keeper
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to update domain expiration for heartbeat")
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeHeartbeatDomain,
			sdk.NewAttribute(types.AttributeKeyDomainID, fmt.Sprintf("%d", msg.Id)),
			sdk.NewAttribute(types.AttributeKeyDomainName, domain.Name),
			sdk.NewAttribute(types.AttributeKeyActor, msg.Creator),
			sdk.NewAttribute(types.AttributeKeyNewExpiration, fmt.Sprintf("%d", domain.Expiration)),
		),
	})

	return &types.MsgHeartbeatDomainResponse{}, nil
}

func (k msgServer) TransferDomain(goCtx context.Context, msg *types.MsgTransferDomain) (*types.MsgTransferDomainResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	var err error

	if _, err = k.Keeper.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, fmt.Sprintf("invalid signer address: %s", err))
	}
	if _, err = k.Keeper.addressCodec.StringToBytes(msg.NewOwner); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, fmt.Sprintf("invalid new owner address: %s", err))
	}

	domain, errGet := k.Keeper.Domain.Get(ctx, msg.Id)
	if errGet != nil {
		if errors.Is(errGet, collections.ErrNotFound) {
			return nil, errorsmod.Wrapf(sdkerrors.ErrKeyNotFound, "domain with id %d not found", msg.Id)
		}
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to get domain for transfer")
	}

	// CORRECCIÓN: Solo el CREADOR original puede transferir la "creatorship".
	if domain.Creator != msg.Creator {
		return nil, errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "signer %s is not the creator %s; only the original creator can transfer the domain", msg.Creator, domain.Creator)
	}

	oldCreator := domain.Creator
	oldOwner := domain.Owner // Guardar para el evento

	// En una transferencia completa, tanto el creator como el owner se convierten en el nuevo dueño.
	domain.Creator = msg.NewOwner
	domain.Owner = msg.NewOwner

	if err = k.Keeper.Domain.Set(ctx, msg.Id, domain); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to transfer domain ownership and creatorship")
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeTransferDomain,
			sdk.NewAttribute(types.AttributeKeyDomainID, fmt.Sprintf("%d", msg.Id)),
			sdk.NewAttribute(types.AttributeKeyDomainName, domain.Name),
			sdk.NewAttribute(types.AttributeKeyOldOwner, oldOwner),
			sdk.NewAttribute(types.AttributeKeyNewOwner, msg.NewOwner),
			sdk.NewAttribute("old_creator", oldCreator),   // Evento adicional para claridad
			sdk.NewAttribute("new_creator", msg.NewOwner), // Evento adicional para claridad
			sdk.NewAttribute(types.AttributeKeyActor, msg.Creator),
		),
	})

	return &types.MsgTransferDomainResponse{}, nil
}
