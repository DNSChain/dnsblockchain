package dao

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	"dnsblockchain/x/dao/types"
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
				{ // <-- NUEVA ENTRADA PARA QUERY PROPOSAL
					RpcMethod:      "Proposal",
					Use:            "proposal [proposal-id]",
					Short:          "Query a proposal by ID",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "proposal_id"}},
				},
				{ // NUEVO
					RpcMethod: "Proposals",
					Use:       "proposals [--proposal-status <status>] [--voter <address>] [--proposer <address>] [--page-key <key>] [--offset <offset>] [--limit <limit>] [--count-total]",
					Short:     "Query proposals with optional filters and pagination",
					Long:      "Query proposals with optional filters by status, proposer, or voter (Note: voter filter might require specific implementation). Supports pagination.",
					Example:   `dnsblockchaind query dao proposals --proposal-status PROPOSAL_STATUS_VOTING_PERIOD`,
				},
				{ // NUEVO
					RpcMethod: "Vote",
					Use:       "vote [proposal-id] [voter-address]",
					Short:     "Query a specific vote on a proposal by a voter",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "proposal_id"},
						{ProtoField: "voter"},
					},
				},
				{ // NUEVO
					RpcMethod:      "Votes",
					Use:            "votes [proposal-id] [--page-key <key>] [--offset <offset>] [--limit <limit>] [--count-total]",
					Short:          "Query all votes on a proposal with optional pagination",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "proposal_id"}},
				},
				{ // NUEVO
					RpcMethod:      "TallyResult",
					Use:            "tally [proposal-id]",
					Short:          "Query the tally result of a proposal",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "proposal_id"}},
				},
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              types.Msg_serviceDesc.ServiceName,
			EnhanceCustomCommand: true, // Esto se aplica a todos los comandos de este servicio Msg
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "SubmitProposal", // Mantenemos el método para que el servicio se registre en gRPC
					Skip:      true,             // PERO lo saltamos para la generación de CLI, ya que tenemos uno manual
				},
				{
					RpcMethod: "Vote",
					Use:       "vote [proposal-id] [option]",
					Short:     "Vote on an active proposal",
					Example:   `dnsblockchaind tx dao vote 1 VOTE_OPTION_YES --from <key_or_address>`, // Ejemplo podría necesitar minúsculas para option
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "proposal_id"},
						{ProtoField: "option"},
					},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
