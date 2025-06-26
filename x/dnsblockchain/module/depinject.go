package dnsblockchain

import (
	"cosmossdk.io/core/address"
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/store"
	"cosmossdk.io/depinject"
	"cosmossdk.io/depinject/appconfig"
	"github.com/cosmos/cosmos-sdk/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types" // Asegúrate que este sea el GovModuleName correcto

	"dnsblockchain/x/dnsblockchain/keeper"
	"dnsblockchain/x/dnsblockchain/types"
)

var _ depinject.OnePerModuleType = AppModule{}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (AppModule) IsOnePerModuleType() {}

func init() {
	appconfig.Register(
		&types.Module{},
		appconfig.Provide(ProvideModule),
	)
}

type ModuleInputs struct {
	depinject.In

	Config       *types.Module
	StoreService store.KVStoreService
	Cdc          codec.Codec
	AddressCodec address.Codec

	AuthKeeper types.AuthKeeper // Esto es AccountKeeper
	BankKeeper types.BankKeeper // Este es el que necesitamos pasar
}

type ModuleOutputs struct {
	depinject.Out

	DnsblockchainKeeper keeper.Keeper
	Module              appmodule.AppModule
}

func ProvideModule(in ModuleInputs) ModuleOutputs {
	authority := authtypes.NewModuleAddress(types.GovModuleName)
	if in.Config.Authority != "" {
		authority = authtypes.NewModuleAddressOrBech32Address(in.Config.Authority)
	}
	k := keeper.NewKeeper(
		in.StoreService,
		in.Cdc,
		in.AddressCodec,
		authority,
		in.BankKeeper, // <--- PASAR EL BANKKEEPER
	)
	// El AppModule también necesita BankKeeper si lo va a usar para simulación
	m := NewAppModule(in.Cdc, k, in.AuthKeeper, in.BankKeeper)

	return ModuleOutputs{DnsblockchainKeeper: k, Module: m}
}
