package keeper

import (
	"context"

	"dnsblockchain/x/dao/types"

	"cosmossdk.io/collections" // Asegúrate de tener esta importación
)

// InitGenesis initializes the module's state from a provided genesis state.
func (k Keeper) InitGenesis(ctx context.Context, genState types.GenesisState) error {
	// Si genState.Params está vacío o no es válido, usa DefaultParams
	// Aunque DefaultGenesis() ya debería hacer esto, es una doble verificación.
	paramsToSet := genState.Params
	if err := paramsToSet.Validate(); err != nil { // Asumiendo que DefaultParams siempre es válido
		paramsToSet = types.DefaultParams()
	}
	return k.Params.Set(ctx, paramsToSet)
}

// ExportGenesis returns the module's exported genesis.
func (k Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	genesis := types.DefaultGenesis() // Esto ya establece genesis.Params a DefaultParams()

	params, err := k.Params.Get(ctx)
	if err != nil {
		// Si no se encuentran los parámetros (ErrNotFound), NO es un error fatal para ExportGenesis.
		// Simplemente significa que no se han establecido explícitamente en la store,
		// y el genesis usará los DefaultParams() que ya están en la variable `genesis`.
		// Si el error es diferente a ErrNotFound, entonces sí es un problema.
		if err != collections.ErrNotFound {
			return nil, err // Error real al intentar leer de la store
		}
		// Si es ErrNotFound, genesis.Params ya tiene DefaultParams(), así que está bien.
	} else {
		// Si se encontraron parámetros en la store, úsalos.
		genesis.Params = params
	}

	// Aquí continuarías exportando otros estados del módulo dao si los tuvieras
	// Por ejemplo, Proposals, Votes, etc.
	// genesis.ProposalList = ...
	// genesis.ProposalCount = ...

	return genesis, nil
}
