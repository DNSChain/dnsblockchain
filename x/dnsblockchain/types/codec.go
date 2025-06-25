// dnsblockchain/x/dnsblockchain/types/codec.go
package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

var (
// ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry()) // Comentado o eliminado si no se usa
)

func RegisterInterfaces(registrar codectypes.InterfaceRegistry) {
	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateDomain{},
		&MsgUpdateDomain{},
		&MsgDeleteDomain{},
		&MsgTransferDomain{},  // Añadido si no estaba
		&MsgHeartbeatDomain{}, // Añadido si no estaba
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
	)
	msgservice.RegisterMsgServiceDesc(registrar, &_Msg_serviceDesc)
}
