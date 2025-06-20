package dnsblockchain

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	"dnsblockchain/x/dnsblockchain/types"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: types.Query_serviceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Shows the parameters of the module",
				},
				{
					RpcMethod: "ListDomain",
					Use:       "list-domain",
					Short:     "List all domain",
				},
				{
					RpcMethod:      "GetDomain",
					Use:            "get-domain [id]",
					Short:          "Gets a domain by id",
					Alias:          []string{"show-domain"},
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
				},
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              types.Msg_serviceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "UpdateParams",
					Skip:      true, // skipped because authority gated
				},
				{
					RpcMethod:      "CreateDomain",
					Use:            "create-domain [owner] [expiration] [ns]",
					Short:          "Create domain",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "owner"}, {ProtoField: "expiration"}, {ProtoField: "ns"}},
				},
				{
					RpcMethod: "UpdateDomain",
					Use:       "update-domain [id] [owner] [ns]", // Expiration ya no se pide si no se puede cambiar
					Short:     "Update domain owner (creator only) or NS (creator or owner)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "id"},
						{ProtoField: "owner"}, // Opcional si el caller no es creator
						{ProtoField: "ns"},
					},
				},
				{
					RpcMethod:      "DeleteDomain",
					Use:            "delete-domain [id]",
					Short:          "Delete domain",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
				},
				{
					RpcMethod:      "HeartbeatDomain",
					Use:            "heartbeat-domain [id]",
					Short:          "Send a heartbeat to a domain to extend its expiration",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
				},
				{
					RpcMethod:      "TransferDomain",
					Use:            "transfer-domain [id] [new-owner]",
					Short:          "Transfer a domain to a new owner",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}, {ProtoField: "new_owner"}},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
