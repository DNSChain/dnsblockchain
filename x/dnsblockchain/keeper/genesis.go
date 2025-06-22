package keeper

import (
	"context"

	"dnsblockchain/x/dnsblockchain/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func (k Keeper) InitGenesis(ctx context.Context, genState types.GenesisState) error {
	for _, elem := range genState.DomainList {
		if err := k.Domain.Set(ctx, elem.Id, elem); err != nil {
			return err
		}
	}

	if err := k.DomainSeq.Set(ctx, genState.DomainCount); err != nil {
		return err
	}

	// Set all the permittedTlds
	for _, elem := range genState.PermittedTlds {
		if err := k.PermittedTLDs.Set(ctx, elem); err != nil { // Asumiendo que PermittedTLDs es un KeySet[string]
			return err
		}
	}
	return k.Params.Set(ctx, genState.Params)
}

// ExportGenesis returns the module's exported genesis.
func (k Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	var err error

	genesis := types.DefaultGenesis()
	genesis.Params, err = k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}
	err = k.Domain.Walk(ctx, nil, func(key uint64, elem types.Domain) (bool, error) {
		genesis.DomainList = append(genesis.DomainList, elem)
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	genesis.DomainCount, err = k.DomainSeq.Peek(ctx)
	if err != nil {
		return nil, err
	}

	// Get all permittedTlds
	err = k.PermittedTLDs.Walk(ctx, nil, func(tld string) (bool, error) {
		genesis.PermittedTlds = append(genesis.PermittedTlds, tld)
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	return genesis, nil
}
