package dao // o el nombre de tu paquete de módulo

import (
	"cosmossdk.io/core/address"
	"cosmossdk.io/core/appmodule"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/depinject"
	"cosmossdk.io/depinject/appconfig"

	"github.com/cosmos/cosmos-sdk/codec"
	// sdk "github.com/cosmos/cosmos-sdk/types" // No parece ser necesario sdk.AccAddress aquí directamente
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types" // Para DefaultGovAuthority

	"dnsblockchain/x/dao/keeper"
	"dnsblockchain/x/dao/types"
	// Asegúrate de que las importaciones para los tipos concretos de keeper (si los necesitas) estén aquí
	// o que las interfaces en types.AccountKeeper, etc., sean suficientes para depinject.
	// Para DnsblockchainKeeper, ya tienes la interfaz en types.
)

// Asegúrate que AppModule implementa IsOnePerModuleType
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
	StoreService storetypes.KVStoreService
	Cdc          codec.Codec
	AddressCodec address.Codec

	AccountKeeper       types.AccountKeeper       // Interfaz definida en dao/types
	BankKeeper          types.BankKeeper          // Interfaz definida en dao/types
	DnsblockchainKeeper types.DnsblockchainKeeper // Interfaz definida en dao/types
}

type ModuleOutputs struct {
	depinject.Out

	DaoKeeper keeper.Keeper
	Module    appmodule.AppModule
}

func ProvideModule(in ModuleInputs) ModuleOutputs {
	// default to governance authority if not provided
	// La dirección del módulo x/gov se usa comúnmente como la autoridad por defecto
	authority := authtypes.NewModuleAddress(govtypes.ModuleName) // Corregido: govtypes.ModuleName
	if in.Config != nil && in.Config.Authority != "" {
		authority = authtypes.NewModuleAddressOrBech32Address(in.Config.Authority)
	}

	k := keeper.NewKeeper(
		in.StoreService,
		in.Cdc,
		in.AddressCodec,
		authority,
		in.AccountKeeper, // Añadido el AccountKeeper que faltaba
		in.BankKeeper,
		in.DnsblockchainKeeper,
	)
	// AppModule también necesita los keepers si los va a usar directamente o para simulación
	m := NewAppModule(in.Cdc, k, in.AccountKeeper, in.BankKeeper, in.DnsblockchainKeeper)

	return ModuleOutputs{DaoKeeper: k, Module: m}
}
