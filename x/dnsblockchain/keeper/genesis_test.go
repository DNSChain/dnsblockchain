package keeper_test

import (
	"testing"

	"dnsblockchain/x/dnsblockchain/types"

	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params:      types.DefaultParams(),
		DomainList:  []types.Domain{{Id: 0}, {Id: 1}},
		DomainCount: 2,
	}
	f := initFixture(t)
	err := f.keeper.InitGenesis(f.ctx, genesisState)
	require.NoError(t, err)
	got, err := f.keeper.ExportGenesis(f.ctx)
	require.NoError(t, err)
	require.NotNil(t, got)

	require.EqualExportedValues(t, genesisState.Params, got.Params)
	require.EqualExportedValues(t, genesisState.DomainList, got.DomainList)
	require.Equal(t, genesisState.DomainCount, got.DomainCount)

}
