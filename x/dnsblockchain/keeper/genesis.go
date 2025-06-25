package keeper

import (
	"context"
	"strings" // Añadido para normalizar nombres de dominio

	"dnsblockchain/x/dnsblockchain/types"

	sdk "github.com/cosmos/cosmos-sdk/types" // Para sdk.UnwrapSDKContext
)

// InitGenesis initializes the module's state from a provided genesis state.
func (k Keeper) InitGenesis(ctx context.Context, genState types.GenesisState) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx) // Para usar con k.Logger si es necesario

	for _, elem := range genState.DomainList {
		if err := k.Domain.Set(ctx, elem.Id, elem); err != nil {
			return err
		}
		// Poblar el índice de nombres
		normalizedName := strings.ToLower(strings.Trim(elem.Name, "."))
		if normalizedName == "" && elem.Name != "" { // Evitar nombres vacíos si el original no lo era
			k.Logger(sdkCtx).Error("Genesis: normalized domain name became empty", "original_name", elem.Name, "id", elem.Id)
			// Podrías decidir si esto es un error fatal o solo un warning
			// return fmt.Errorf("genesis error: normalized domain name for id %d is empty", elem.Id)
			continue // Omitir esta entrada del índice si el nombre normalizado es problemático
		}
		if normalizedName != "" { // Solo indexar si el nombre normalizado no es vacío
			if err := k.DomainName.Set(ctx, normalizedName, elem.Id); err != nil {
				// Esto podría indicar un nombre duplicado en el genesis, lo cual debería ser validado por GenesisState.Validate()
				k.Logger(sdkCtx).Error("Genesis: failed to set domain name index", "name", normalizedName, "id", elem.Id, "error", err)
				return err // Considerar esto un error fatal en génesis
			}
		}
	}

	if err := k.DomainSeq.Set(ctx, genState.DomainCount); err != nil {
		return err
	}

	for _, elem := range genState.PermittedTlds {
		normalizedTLD := strings.ToLower(strings.Trim(elem, "."))
		if normalizedTLD != "" { // Solo añadir TLDs no vacíos
			if err := k.PermittedTLDs.Set(ctx, normalizedTLD); err != nil {
				k.Logger(sdkCtx).Error("Genesis: failed to set permitted TLD", "tld", normalizedTLD, "error", err)
				return err
			}
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

	err = k.PermittedTLDs.Walk(ctx, nil, func(tld string) (bool, error) {
		genesis.PermittedTlds = append(genesis.PermittedTlds, tld)
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	// El índice DomainName no necesita ser exportado explícitamente si se reconstruye
	// durante InitGenesis a partir de DomainList.

	return genesis, nil
}
