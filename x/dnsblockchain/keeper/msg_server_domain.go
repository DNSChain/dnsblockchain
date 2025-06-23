package keeper

import (
	"context"
	"errors"
	"fmt"
	"strings" // Necesario para strings.Split y strings.Trim

	"dnsblockchain/x/dnsblockchain/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) CreateDomain(ctx context.Context, msg *types.MsgCreateDomain) (*types.MsgCreateDomainResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	var err error // Declarar err una vez al principio

	// Validar dirección creator
	if _, err = k.addressCodec.StringToBytes(msg.Creator); err != nil { // Reasignar a err
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, fmt.Sprintf("invalid creator address: %s", err))
	}

	// Validar dirección owner
	if _, err = k.addressCodec.StringToBytes(msg.Owner); err != nil { // Reasignar a err
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, fmt.Sprintf("invalid owner address: %s", err))
	}

	// Validate domain name structure: must be label.TLD
	// msg.Name should not contain subdomains at the blockchain registration level.
	// Subdomains are managed off-chain by the domain owner using their registered NS.
	normalizedName := strings.Trim(msg.Name, ".")
	parts := strings.Split(normalizedName, ".")
	if len(parts) != 2 {
		return nil, errorsmod.Wrapf(types.ErrInvalidDomainName, "domain name '%s' must be in 'label.tld' format; subdomains are not registered directly on-chain", msg.Name)
	}
	// label := parts[0] // El label es la primera parte
	extractedTLDFromMsgName := parts[1] // El TLD es la segunda parte

	// Ahora, usamos el TLD extraído para la verificación
	if extractedTLDFromMsgName == "" { // Aunque la validación de len(parts) != 2 debería cubrir esto
		return nil, errorsmod.Wrapf(types.ErrInvalidDomainName, "TLD could not be extracted from domain name '%s'", msg.Name)
	}

	isPermitted, err := k.IsTLDPermitted(sdkCtx, extractedTLDFromMsgName)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to check if TLD is permitted")
	}
	if !isPermitted {
		return nil, errorsmod.Wrapf(types.ErrTLDNotPermitted, "TLD '%s' from domain name '%s' is not permitted", extractedTLDFromMsgName, msg.Name)
	}

	// Verificar si ya existe un dominio con el mismo nombre (msg.Name ya está normalizado en la práctica por el Trim anterior)
	walkErr := k.Domain.Walk(sdkCtx, nil, func(key uint64, value types.Domain) (bool, error) {
		if value.Name == normalizedName { // Comparar con normalizedName
			return true, types.ErrDuplicateDomainName
		}
		return false, nil
	})

	if walkErr != nil {
		if errors.Is(walkErr, types.ErrDuplicateDomainName) {
			return nil, walkErr
		}
		return nil, errorsmod.Wrap(walkErr, "failed to check for duplicate domain name")
	}

	nextID, err := k.DomainSeq.Next(sdkCtx)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "failed to get next id")
	}

	domain := types.Domain{
		Id:         nextID,
		Creator:    msg.Creator,
		Name:       normalizedName, // Guardar el nombre normalizado
		Owner:      msg.Owner,      // Usar el msg.Owner proporcionado
		Ns:         msg.Ns,
		Expiration: uint64(sdkCtx.BlockTime().AddDate(1, 0, 0).Unix()),
	}

	if err = k.Domain.Set(sdkCtx, nextID, domain); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to set domain")
	}

	return &types.MsgCreateDomainResponse{
		Id: nextID,
	}, nil
}

func (k msgServer) UpdateDomain(ctx context.Context, msg *types.MsgUpdateDomain) (*types.MsgUpdateDomainResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	var err error // Declarar err una vez

	// Validar dirección del firmante (msg.Creator)
	if _, err = k.addressCodec.StringToBytes(msg.Creator); err != nil { // Reasignar
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, fmt.Sprintf("invalid signer address: %s", err))
	}

	// Verificar que el dominio existe
	val, err := k.Domain.Get(sdkCtx, msg.Id) // err es una nueva variable en este scope, está bien
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, errorsmod.Wrap(sdkerrors.ErrKeyNotFound, fmt.Sprintf("key %d doesn't exist", msg.Id))
		}
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to get domain")
	}

	// Determinar los campos a actualizar y verificar permisos
	newOwner := val.Owner // Por defecto, el owner no cambia
	newNs := val.Ns       // Por defecto, los NS no cambian

	isCreator := msg.Creator == val.Creator
	isOwner := msg.Creator == val.Owner

	if !isCreator && !isOwner {
		return nil, errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "signer %s is neither the creator nor the owner of domain %d", msg.Creator, msg.Id)
	}

	// Si el firmante es el creator
	if isCreator {
		// El creator puede cambiar el owner (si se proporciona en el mensaje y es diferente)
		if msg.Owner != "" && msg.Owner != val.Owner {
			if _, err = k.addressCodec.StringToBytes(msg.Owner); err != nil { // Reasignar
				return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, fmt.Sprintf("invalid new owner address in message: %s", err))
			}
			newOwner = msg.Owner
		}
		// El creator puede cambiar los NS (si se proporcionan en el mensaje y son diferentes)
		if msg.Ns != "" && msg.Ns != val.Ns {
			newNs = msg.Ns
		}
	} else if isOwner { // Si el firmante es solo el owner (y no el creator)
		// El owner solo puede cambiar los NS
		if msg.Ns != "" && msg.Ns != val.Ns {
			newNs = msg.Ns
		}
		// Si el owner intenta cambiar el owner, es un error (a menos que también sea el creator, cubierto arriba)
		if msg.Owner != "" && msg.Owner != val.Owner {
			return nil, errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "owner %s cannot change domain ownership; only creator %s can", msg.Creator, val.Creator)
		}
	}

	// Construir el dominio actualizado
	// El Nombre y el Creator original NUNCA cambian en UpdateDomain.
	// La Expiración tampoco cambia aquí.
	domain := types.Domain{
		Id:         msg.Id,
		Creator:    val.Creator, // Creator original no cambia
		Name:       val.Name,    // Nombre no cambia
		Owner:      newOwner,    // Owner actualizado según la lógica de permisos
		Ns:         newNs,       // NS actualizados según la lógica de permisos
		Expiration: val.Expiration,
	}

	// Guardar cambios
	if err = k.Domain.Set(sdkCtx, msg.Id, domain); err != nil { // Reasignar
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to update domain")
	}

	return &types.MsgUpdateDomainResponse{}, nil
}

func (k msgServer) DeleteDomain(ctx context.Context, msg *types.MsgDeleteDomain) (*types.MsgDeleteDomainResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	var err error // Declarar err una vez

	// Validar dirección creator
	if _, err = k.addressCodec.StringToBytes(msg.Creator); err != nil { // Reasignar
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, fmt.Sprintf("invalid address: %s", err))
	}

	// Verificar que el dominio existe
	val, errGet := k.Domain.Get(sdkCtx, msg.Id) // Usar nueva variable errGet para evitar shadowing
	if errGet != nil {
		if errors.Is(errGet, collections.ErrNotFound) {
			return nil, errorsmod.Wrap(sdkerrors.ErrKeyNotFound, fmt.Sprintf("key %d doesn't exist", msg.Id))
		}
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to get domain")
	}

	// Solo el creator original puede borrar el dominio
	if msg.Creator != val.Creator {
		return nil, errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "signer %s is not the creator %s of the domain", msg.Creator, val.Creator)
	}

	// Eliminar dominio
	if err = k.Domain.Remove(sdkCtx, msg.Id); err != nil { // Reasignar
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to delete domain")
	}

	return &types.MsgDeleteDomainResponse{}, nil
}

// HeartbeatDomain extiende la expiración del dominio 1 año más desde el bloque actual
func (k msgServer) HeartbeatDomain(ctx context.Context, msg *types.MsgHeartbeatDomain) (*types.MsgHeartbeatDomainResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	var err error // Declarar err una vez

	// Validar dirección del firmante (msg.Creator)
	if _, err = k.addressCodec.StringToBytes(msg.Creator); err != nil { // Reasignar
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, fmt.Sprintf("invalid address: %s", err))
	}

	// Obtener dominio
	domain, errGet := k.Domain.Get(sdkCtx, msg.Id) // Usar nueva variable errGet
	if errGet != nil {
		if errors.Is(errGet, collections.ErrNotFound) {
			return nil, errorsmod.Wrap(sdkerrors.ErrKeyNotFound, fmt.Sprintf("domain id %d not found", msg.Id))
		}
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to get domain")
	}

	// Verificar que quien firma es el creator original o el owner actual
	if msg.Creator != domain.Creator && msg.Creator != domain.Owner {
		return nil, errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "signer %s is neither the creator (%s) nor the owner (%s) of the domain; only creator or owner can send heartbeat", msg.Creator, domain.Creator, domain.Owner)
	}

	// Extender expiración a 1 año desde ahora
	domain.Expiration = uint64(sdkCtx.BlockTime().AddDate(1, 0, 0).Unix())

	// Guardar dominio actualizado
	if err = k.Domain.Set(sdkCtx, msg.Id, domain); err != nil { // Reasignar
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to update domain expiration")
	}

	return &types.MsgHeartbeatDomainResponse{}, nil
}

func (k msgServer) TransferDomain(goCtx context.Context, msg *types.MsgTransferDomain) (*types.MsgTransferDomainResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	var err error // Declarar err una vez

	// Validar dirección del firmante (msg.Creator)
	if _, err = k.addressCodec.StringToBytes(msg.Creator); err != nil { // Reasignar
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, fmt.Sprintf("invalid creator address: %s", err))
	}
	// Validar nueva dirección de owner
	if _, err = k.addressCodec.StringToBytes(msg.NewOwner); err != nil { // Reasignar
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, fmt.Sprintf("invalid new owner address: %s", err))
	}

	domain, errGet := k.Domain.Get(ctx, msg.Id) // Usar nueva variable errGet
	if errGet != nil {
		// Devuelve un error más específico si no se encuentra
		if errors.Is(errGet, collections.ErrNotFound) {
			return nil, errorsmod.Wrapf(sdkerrors.ErrKeyNotFound, "domain with id %d not found", msg.Id)
		}
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to get domain")
	}

	// Solo el creator original (que también podría ser el owner) puede transferir la "autoría" (el campo creator)
	if domain.Creator != msg.Creator {
		return nil, errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "signer %s is not the creator %s; only the original creator can fully transfer the domain (creator and owner fields)", msg.Creator, domain.Creator)
	}

	// El nuevo propietario también se convierte en el nuevo creador
	domain.Creator = msg.NewOwner
	domain.Owner = msg.NewOwner

	if err = k.Domain.Set(ctx, msg.Id, domain); err != nil { // Reasignar
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "failed to transfer domain ownership and creatorship")
	}

	return &types.MsgTransferDomainResponse{}, nil
}
