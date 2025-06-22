package dao // o el nombre de tu paquete de módulo

import (
	"encoding/json"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"

	"cosmossdk.io/core/appmodule"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	// authtypes "github.com/cosmos/cosmos-sdk/x/auth/types" // Si AuthKeeper es authtypes.AccountKeeper

	"dnsblockchain/x/dao/keeper"
	"dnsblockchain/x/dao/types"
)

var (
	_ module.AppModuleBasic = (*AppModule)(nil)
	_ module.AppModule      = (*AppModule)(nil) // AppModule del SDK tradicional
	_ module.HasGenesis     = (*AppModule)(nil)

	_ appmodule.AppModule = (*AppModule)(nil) // AppModule de cosmossdk.io/core
	// Si tu módulo DAO necesita BeginBlocker o EndBlocker, descomenta y implementa:
	// _ appmodule.HasBeginBlocker = (*AppModule)(nil)
	// _ appmodule.HasEndBlocker   = (*AppModule)(nil)
)

// AppModule implements the AppModule interface that defines the inter-dependent methods that modules need to implement
type AppModule struct {
	cdc    codec.Codec
	keeper keeper.Keeper // El keeper de nuestro módulo DAO

	// Keepers de otros módulos que AppModule podría necesitar
	// Para el SDK v0.50+, se prefiere usar las interfaces definidas en types.AccountKeeper, etc.
	// El error decía que NewAppModule esperaba types.AuthKeeper. Vamos a usar AccountKeeper
	// y asegurarnos de que AccountKeeper en types/expected_keepers.go sea lo que el SDK x/auth provee.
	// Por convención, a menudo se le llama 'authKeeper' al AccountKeeper.
	authKeeper          types.AccountKeeper // Cambiado de types.AuthKeeper si esa era la intención
	bankKeeper          types.BankKeeper
	dnsblockchainKeeper types.DnsblockchainKeeper // Añadido
}

func NewAppModule(
	cdc codec.Codec,
	keeper keeper.Keeper, // Keeper de este módulo
	// Mantener consistencia con lo que se espera y lo que se pasa.
	// Si types.AuthKeeper es la interfaz de auth que quieres, úsala.
	// Si es types.AccountKeeper, úsala. El error original decía AuthKeeper.
	// Vamos a asumir que types.AuthKeeper es la interfaz correcta para x/auth.
	// Si no, la renombraremos en types.AccountKeeper en expected_keepers.go
	authKeeper types.AccountKeeper, // Cambiado a AccountKeeper para coincidir con lo que se pasa
	bankKeeper types.BankKeeper,
	dnsblockchainKeeper types.DnsblockchainKeeper, // Añadido
) AppModule {
	return AppModule{
		cdc:                 cdc,
		keeper:              keeper,
		authKeeper:          authKeeper,
		bankKeeper:          bankKeeper,
		dnsblockchainKeeper: dnsblockchainKeeper, // Añadido
	}
}

// IsAppModule implements the appmodule.AppModule interface.
func (AppModule) IsAppModule() {}

// Name returns the name of the module as a string.
func (AppModule) Name() string {
	return types.ModuleName
}

// RegisterLegacyAminoCodec registers the amino codec (generalmente vacío para módulos modernos)
func (AppModule) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the module.
func (AppModule) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	if err := types.RegisterQueryHandlerClient(clientCtx.CmdContext, mux, types.NewQueryClient(clientCtx)); err != nil {
		// Considerar un panic más informativo o un log fatal
		panic(fmt.Errorf("failed to register DAO gRPC gateway routes: %w", err))
	}
}

// RegisterInterfaces registers a module's interface types and their concrete implementations as proto.Message.
func (AppModule) RegisterInterfaces(registrar codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(registrar)
}

// RegisterServices registers a gRPC query service to respond to the module-specific gRPC queries
func (am AppModule) RegisterServices(registrar grpc.ServiceRegistrar) error {
	types.RegisterMsgServer(registrar, keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(registrar, keeper.NewQueryServerImpl(am.keeper))
	// Si tuvieras un QueryServer para Params, también iría aquí, pero lo manejaremos por gov.
	// err := keeper.RegisterMigrateHandler(registrar, am.keeper) // Ejemplo para migraciones
	return nil
}

// DefaultGenesis returns a default GenesisState for the module, marshalled to json.RawMessage.
func (am AppModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesis())
}

// ValidateGenesis used to validate the GenesisState, given in its json.RawMessage form.
func (am AppModule) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var genState types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}
	return genState.Validate()
}

// InitGenesis performs the module's genesis initialization.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, gs json.RawMessage) {
	var genState types.GenesisState
	cdc.MustUnmarshalJSON(gs, &genState)
	if err := am.keeper.InitGenesis(ctx, genState); err != nil {
		panic(fmt.Errorf("failed to init %s genesis: %w", types.ModuleName, err))
	}
}

// ExportGenesis returns the module's exported genesis state as raw JSON bytes.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	genState, err := am.keeper.ExportGenesis(ctx)
	if err != nil {
		panic(fmt.Errorf("failed to export %s genesis: %w", types.ModuleName, err))
	}
	return cdc.MustMarshalJSON(genState)
}

// ConsensusVersion is a sequence number for state-breaking change of the module.
func (AppModule) ConsensusVersion() uint64 { return 1 } // Empieza en 1

// BeginBlock y EndBlock (si los necesitas para la lógica de la DAO, como procesar propuestas)
// Por ahora, los dejaremos vacíos pero con la estructura para implementarlos si es necesario
// con appmodule.HasBeginBlocker / appmodule.HasEndBlocker

// BeginBlock executes all ABCI BeginBlock logic respective to the dao module.
// func (am AppModule) BeginBlock(ctx context.Context) error {
// 	// sdkCtx := sdk.UnwrapSDKContext(ctx)
// 	// Lógica de BeginBlock para el módulo DAO
// 	return nil
// }

// EndBlock executes all ABCI EndBlock logic respective to the dao module. It
// returns no validator updates.
// func (am AppModule) EndBlock(ctx context.Context) error {
// 	// sdkCtx := sdk.UnwrapSDKContext(ctx)
// 	// Lógica de EndBlock para el módulo DAO (ej. procesar propuestas que finalizan su votación)
//  // err := am.keeper.TallyAndProcessProposals(sdkCtx)
//  // if err != nil {
//  //  return err
//  // }
// 	return nil
// }
