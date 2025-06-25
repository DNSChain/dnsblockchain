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
				{
					RpcMethod: "ListPermittedTLDs",
					Use:       "list-permitted-tlds",
					Short:     "List all permitted TLDs",
				},
				{
					RpcMethod:      "GetDomainByName",
					Use:            "get-domain-by-name [name]",
					Short:          "Gets a domain by its FQDN (e.g., example.dweb)",
					Alias:          []string{"show-domain-by-name"},
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "name"}},
				},
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              types.Msg_serviceDesc.ServiceName,
			EnhanceCustomCommand: true,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "UpdateParams",
					Skip:      true,
				},
				{
					RpcMethod: "CreateDomain",
					Use:       "create-domain [name] [owner] --ns-records-json <json_string_or_path>", // Cambiado
					Short:     "Create a new domain. NS records must be provided as a JSON string or path to a JSON file via the --ns-records-json flag.",
					Long: `Create a new domain.
Example:
dnsblockchaind tx dnsblockchain create-domain example.web3 cosmos1... --ns-records-json '[{"name":"ns1.example.web3","ipv4_addresses":["1.2.3.4"]},{"name":"ns2.example.web3","ipv4_addresses":["5.6.7.8"]}]' --from mykey

Or using a file:
dnsblockchaind tx dnsblockchain create-domain example.web3 cosmos1... --ns-records-json path/to/ns_records.json --from mykey

ns_records.json content:
[
  {"name":"ns1.example.web3","ipv4_addresses":["1.2.3.4"],"ipv6_addresses":["2001:db8::1"]},
  {"name":"ns2.example.web3","ipv4_addresses":["5.6.7.8"]}
]
`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "name"},
						{ProtoField: "owner"},
						// No hay argumento posicional para ns_records; se usará un flag.
					},
					// Flags para campos que no son posicionales o son complejos
					// Autocli debería generar un flag para --ns-records si el campo existe en el mensaje.
					// Pero como es `repeated`, la entrada directa por flag es compleja.
					// A menudo, para `repeated` mensajes, se usa un flag que toma un JSON string.
					// Si autocli no genera un flag útil para NsRecords, se podría necesitar un comando CLI personalizado.
					// Por ahora, vamos a ver si autocli genera un flag --ns-records y cómo se comporta.
					// Si no, tendríamos que hacer el flag --ns-records-json manualmente o un comando wrapper.
				},
				{
					RpcMethod: "UpdateDomain",
					Use:       "update-domain [id] --owner [new-owner] --ns-records-json <json_string_or_path>", // Cambiado
					Short:     "Update domain owner (creator only) or NS records (creator or owner).",
					Long: `Update an existing domain.
Provide the domain ID. Optionally provide a new owner (if you are the creator)
and/or new NS records as a JSON string or path to a file via --ns-records-json.
`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "id"},
						// Owner y ns_records se manejarán con flags o JSON.
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
