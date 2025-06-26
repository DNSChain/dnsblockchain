package keeper

import (
	"context"
	"fmt"
	"strings"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	corestore "cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"dnsblockchain/x/dnsblockchain/types"
)

type Keeper struct {
	storeService corestore.KVStoreService
	cdc          codec.Codec
	addressCodec address.Codec
	authority    []byte

	bankKeeper types.BankKeeper // <--- AÑADIR ESTA LÍNEA

	Schema        collections.Schema
	Params        collections.Item[types.Params]
	DomainSeq     collections.Sequence
	Domain        collections.Map[uint64, types.Domain]
	DomainName    collections.Map[string, uint64]
	PermittedTLDs collections.KeySet[string]
}

func NewKeeper(
	storeService corestore.KVStoreService,
	cdc codec.Codec,
	addressCodec address.Codec,
	authority []byte,
	bk types.BankKeeper, // <--- AÑADIR bk COMO PARÁMETRO
) Keeper {
	if _, err := addressCodec.BytesToString(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address %s: %s", authority, err))
	}

	sb := collections.NewSchemaBuilder(storeService)

	k := Keeper{
		storeService: storeService,
		cdc:          cdc,
		addressCodec: addressCodec,
		authority:    authority,
		bankKeeper:   bk, // <--- ASIGNAR bk

		Params:        collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		Domain:        collections.NewMap(sb, types.DomainKey, "domain_by_id", collections.Uint64Key, codec.CollValue[types.Domain](cdc)),
		DomainName:    collections.NewMap(sb, types.DomainNameKey, "domain_by_name", collections.StringKey, collections.Uint64Value),
		DomainSeq:     collections.NewSequence(sb, types.DomainCountKey, "domain_sequence"),
		PermittedTLDs: collections.NewKeySet(sb, types.PermittedTLDsKey, "permitted_tlds", collections.StringKey),
	}
	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema

	return k
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// AddPermittedTLD añade un TLD a la lista de permitidos.
func (k Keeper) AddPermittedTLD(ctx context.Context, tld string) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	normalizedTLD := strings.ToLower(strings.TrimSpace(tld)) // Normalizar

	// Check against hardcoded ICANN reserved list first
	isGloballyReserved, _ := k.IsTLDGloballyReserved(sdkCtx, normalizedTLD)
	if isGloballyReserved {
		k.Logger(sdkCtx).Error("Attempt to add globally reserved TLD directly to permitted list", "tld", normalizedTLD)
		return errorsmod.Wrapf(types.ErrTLDReservedByICANN, "TLD '%s' is globally reserved and cannot be added to permitted TLDs", normalizedTLD)
	}

	if normalizedTLD == "" {
		return errorsmod.Wrap(types.ErrInvalidTLD, "TLD cannot be empty")
	}
	if strings.Contains(normalizedTLD, ".") {
		return errorsmod.Wrapf(types.ErrInvalidTLD, "TLD '%s' cannot contain dots", normalizedTLD)
	}
	if len(normalizedTLD) > 63 || len(normalizedTLD) < 2 {
		return errorsmod.Wrapf(types.ErrInvalidTLD, "TLD '%s' length must be between 2 and 63 characters", normalizedTLD)
	}

	has, err := k.PermittedTLDs.Has(sdkCtx, normalizedTLD)
	if err != nil {
		return errorsmod.Wrap(err, "failed to check if TLD exists")
	}
	if has {
		return errorsmod.Wrapf(types.ErrInvalidTLD, "TLD '%s' is already permitted", normalizedTLD)
	}

	k.Logger(sdkCtx).Info("Adding permitted TLD to store", "tld", normalizedTLD)
	return k.PermittedTLDs.Set(sdkCtx, normalizedTLD)
}

// IsTLDGloballyReserved checks if a TLD is in the hardcoded ICANN deny list.
func (k Keeper) IsTLDGloballyReserved(ctx context.Context, tld string) (bool, error) {
	return types.IsReservedTLD(tld), nil
}

// IsTLDPermitted verifica si un TLD está en la lista de permitidos.
func (k Keeper) IsTLDPermitted(ctx context.Context, tld string) (bool, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	normalizedTLD := strings.ToLower(strings.TrimSpace(tld))
	if normalizedTLD == "" {
		return false, nil
	}
	return k.PermittedTLDs.Has(sdkCtx, normalizedTLD)
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() []byte {
	return k.authority
}
