package keeper

import (
	"context"
	"fmt"
	"strings" // Para normalizar el TLD

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	corestore "cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	// Para errores estándar
	"dnsblockchain/x/dnsblockchain/types"
)

type Keeper struct {
	storeService corestore.KVStoreService
	cdc          codec.Codec
	addressCodec address.Codec
	// Address capable of executing a MsgUpdateParams message.
	// Typically, this should be the x/gov module account.
	authority []byte

	Schema        collections.Schema
	Params        collections.Item[types.Params]
	DomainSeq     collections.Sequence
	Domain        collections.Map[uint64, types.Domain]
	PermittedTLDs collections.KeySet[string]
}

func NewKeeper(
	storeService corestore.KVStoreService,
	cdc codec.Codec,
	addressCodec address.Codec,
	authority []byte,

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

		Params:        collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		Domain:        collections.NewMap(sb, types.DomainKey, "domain", collections.Uint64Key, codec.CollValue[types.Domain](cdc)),
		DomainSeq:     collections.NewSequence(sb, types.DomainCountKey, "domainSequence"),
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

	if normalizedTLD == "" {
		return errorsmod.Wrap(types.ErrInvalidTLD, "TLD cannot be empty")
	}
	// Validación simple de formato (puedes hacerla más robusta)
	if strings.Contains(normalizedTLD, ".") {
		return errorsmod.Wrapf(types.ErrInvalidTLD, "TLD '%s' cannot contain dots", normalizedTLD)
	}
	if len(normalizedTLD) > 63 || len(normalizedTLD) < 2 { // Límites comunes para TLDs
		return errorsmod.Wrapf(types.ErrInvalidTLD, "TLD '%s' length must be between 2 and 63 characters", normalizedTLD)
	}
	// Podrías añadir validación de caracteres aquí (e.g., solo alfanuméricos y guiones no al inicio/fin)

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

// IsTLDPermitted verifica si un TLD está en la lista de permitidos.
func (k Keeper) IsTLDPermitted(ctx context.Context, tld string) (bool, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	normalizedTLD := strings.ToLower(strings.TrimSpace(tld))
	if normalizedTLD == "" { // Un TLD vacío no debería considerarse permitido ni causar error de store
		return false, nil
	}
	return k.PermittedTLDs.Has(sdkCtx, normalizedTLD)
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() []byte {
	return k.authority
}
