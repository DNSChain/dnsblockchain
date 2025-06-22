package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy" // Para RegisterAminoMsg
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterLegacyAminoCodec registers the necessary x/dao interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgSubmitProposal{}, "dnsblockchain/MsgSubmitProposal") // Usar el nombre del módulo del SDK
	legacy.RegisterAminoMsg(cdc, &MsgVote{}, "dnsblockchain/MsgVote")
	// No es necesario registrar ProposalContent aquí ya que Amino no maneja 'Any' bien.
}

// RegisterInterfaces registers the x/dao interfaces types with the interface registry
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubmitProposal{},
		&MsgVote{},
	)

	registry.RegisterInterface(
		"dnsblockchain.dao.v1.ProposalContent", // Nombre completo del tipo de interfaz Protobuf
		(*ProposalContent)(nil),                // Puntero a la interfaz Go
		&AddTldProposalContent{},
		&RequestTokensProposalContent{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	// Amino es el codec legacy.
	Amino = codec.NewLegacyAmino()
	// ModuleCdc es el codec Proto del módulo.
	// Se debe asegurar que NewInterfaceRegistry() se llame solo una vez
	// y se pase a todos los lugares donde se necesite, o usar la InterfaceRegistry de la app.
	// Por ahora, para ModuleCdc, es mejor usar el que se configura a nivel de app si es posible,
	// o asegurarse de que este registro se use consistentemente.
	// La práctica común es NO usar un ModuleCdc propio para unmarshalling de txs en la CLI,
	// sino usar el clientCtx.Codec.
	ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
)

func init() {
	RegisterLegacyAminoCodec(Amino)
	// No es necesario llamar a RegisterInterfaces aquí para ModuleCdc si no se usa para unmarshalling
	// de mensajes con 'Any' dentro del propio módulo de forma aislada. El registro principal
	// ocurre en la AppModule.
}
