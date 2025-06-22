package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterLegacyAminoCodec registers the necessary x/dao interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgSubmitProposal{}, "dao/MsgSubmitProposal")
	legacy.RegisterAminoMsg(cdc, &MsgVote{}, "dao/MsgVote")
	// Agrega otros mensajes si los tienes
}

// RegisterInterfaces registers the x/dao interfaces types with the interface registry
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubmitProposal{},
		&MsgVote{},
		// Agrega otros mensajes si los tienes
	)

	// Registrar ProposalContent como una interfaz
	// El nombre "Content" aquí es una convención, puedes elegir otro.
	// Debe coincidir con lo que pongas en `option (cosmos_proto.implements_interface) = "Content";` en tus protos de contenido.
	registry.RegisterInterface(
		"dnsblockchain.dao.v1.ProposalContent", // Nombre completo del tipo de interfaz (paquete.NombreInterfaz)
		(*ProposalContent)(nil),                // Puntero a la interfaz Go que definirás
		&AddTldProposalContent{},
		&RequestTokensProposalContent{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

// ModuleCdc es el codec del módulo.
var ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
