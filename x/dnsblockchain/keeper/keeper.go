package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"dnsblockchain/x/dnsblockchain/types"
)

type Keeper struct {
	storeService corestore.KVStoreService
	cdc          codec.Codec
	addressCodec address.Codec
	// Address capable of executing a MsgUpdateParams message.
	// Typically, this should be the x/gov module account.
	authority []byte

	Schema    collections.Schema
	Params    collections.Item[types.Params]
	DomainSeq collections.Sequence
	Domain    collections.Map[uint64, types.Domain]
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

		Params:    collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		Domain:    collections.NewMap(sb, types.DomainKey, "domain", collections.Uint64Key, codec.CollValue[types.Domain](cdc)),
		DomainSeq: collections.NewSequence(sb, types.DomainCountKey, "domainSequence"),
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

// AddPermittedTLD is a placeholder method to satisfy the DnsblockchainKeeper interface
// expected by the x/dao module.
// TODO: Implement actual logic if/when this is used by DAO proposals.
func (k Keeper) AddPermittedTLD(ctx context.Context, tld string) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	k.Logger(sdkCtx).Info("AddPermittedTLD called (placeholder)", "tld", tld)
	// Example: store the TLD in a new collection or update params
	// For instance, if you had a collection for permitted TLDs:
	// return k.PermittedTLDs.Set(ctx, tld, true)
	return nil
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() []byte {
	return k.authority
}
