package dnsblockchain

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"dnsblockchain/testutil/sample"
	dnsblockchainsimulation "dnsblockchain/x/dnsblockchain/simulation"
	"dnsblockchain/x/dnsblockchain/types"
)

// GenerateGenesisState creates a randomized GenState of the module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	dnsblockchainGenesis := types.GenesisState{
		Params:     types.DefaultParams(),
		DomainList: []types.Domain{{Id: 0, Creator: sample.AccAddress()}, {Id: 1, Creator: sample.AccAddress()}}, DomainCount: 2,
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&dnsblockchainGenesis)
}

// RegisterStoreDecoder registers a decoder.
func (am AppModule) RegisterStoreDecoder(_ simtypes.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)
	const (
		opWeightMsgCreateDomain          = "op_weight_msg_dnsblockchain"
		defaultWeightMsgCreateDomain int = 100
	)

	var weightMsgCreateDomain int
	simState.AppParams.GetOrGenerate(opWeightMsgCreateDomain, &weightMsgCreateDomain, nil,
		func(_ *rand.Rand) {
			weightMsgCreateDomain = defaultWeightMsgCreateDomain
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgCreateDomain,
		dnsblockchainsimulation.SimulateMsgCreateDomain(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))
	const (
		opWeightMsgUpdateDomain          = "op_weight_msg_dnsblockchain"
		defaultWeightMsgUpdateDomain int = 100
	)

	var weightMsgUpdateDomain int
	simState.AppParams.GetOrGenerate(opWeightMsgUpdateDomain, &weightMsgUpdateDomain, nil,
		func(_ *rand.Rand) {
			weightMsgUpdateDomain = defaultWeightMsgUpdateDomain
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgUpdateDomain,
		dnsblockchainsimulation.SimulateMsgUpdateDomain(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))
	const (
		opWeightMsgDeleteDomain          = "op_weight_msg_dnsblockchain"
		defaultWeightMsgDeleteDomain int = 100
	)

	var weightMsgDeleteDomain int
	simState.AppParams.GetOrGenerate(opWeightMsgDeleteDomain, &weightMsgDeleteDomain, nil,
		func(_ *rand.Rand) {
			weightMsgDeleteDomain = defaultWeightMsgDeleteDomain
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgDeleteDomain,
		dnsblockchainsimulation.SimulateMsgDeleteDomain(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))

	return operations
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{}
}
