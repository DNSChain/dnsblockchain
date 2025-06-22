// x/dao/keeper/keeper.go

package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"dnsblockchain/x/dao/types"
)

type Keeper struct {
	storeService storetypes.KVStoreService
	cdc          codec.Codec
	addressCodec address.Codec

	accountKeeper       types.AccountKeeper
	bankKeeper          types.BankKeeper
	dnsblockchainKeeper types.DnsblockchainKeeper

	authority sdk.AccAddress

	Schema              collections.Schema
	Params              collections.Item[types.DaoParams]
	ProposalSeq         collections.Sequence // Esta es la secuencia que usaremos para los IDs de propuesta
	Proposals           collections.Map[uint64, types.Proposal]
	Votes               collections.Map[collections.Pair[uint64, sdk.AccAddress], types.Vote]
	ActiveProposalQueue collections.KeySet[uint64]
}

func NewKeeper(
	storeService storetypes.KVStoreService,
	cdc codec.Codec,
	addressCodec address.Codec,
	authority sdk.AccAddress,
	ak types.AccountKeeper,
	bk types.BankKeeper,
	dk types.DnsblockchainKeeper,
) Keeper {
	if _, err := addressCodec.BytesToString(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		storeService:        storeService,
		cdc:                 cdc,
		addressCodec:        addressCodec,
		authority:           authority,
		accountKeeper:       ak,
		bankKeeper:          bk,
		dnsblockchainKeeper: dk,
		Params:              collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.DaoParams](cdc)),
		ProposalSeq:         collections.NewSequence(sb, types.ProposalSeqKey, "proposal_sequence"), // Usamos esta
		Proposals:           collections.NewMap(sb, types.ProposalsKeyPrefix, "proposals", collections.Uint64Key, codec.CollValue[types.Proposal](cdc)),
		Votes: collections.NewMap(sb, types.VotesKeyPrefix, "votes",
			collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey),
			codec.CollValue[types.Vote](cdc),
		),
		ActiveProposalQueue: collections.NewKeySet(sb, types.ActiveProposalQueueKeyPrefix, "active_proposal_queue", collections.Uint64Key),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema

	return k
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() sdk.AccAddress {
	return k.authority
}

// --- Proposal Helpers ---

// GetNextProposalID retrieves the next proposal ID by calling Next on the ProposalSeq sequence.
func (k Keeper) GetNextProposalID(ctx sdk.Context) (uint64, error) {
	// El método Next() de collections.Sequence ya incrementa y devuelve el nuevo valor.
	// Si quieres el valor actual y LUEGO incrementarlo, necesitarías Peek() y luego Set(),
	// pero Next() es más atómico para este caso de uso.
	// La primera vez que se llama a Next() en una secuencia vacía, devuelve 1 (si está inicializada a 0 o no existe).
	// Si quieres que los IDs empiecen en 0, puedes hacer Peek y luego Set(id+1), devolviendo Peek.
	// Pero empezar en 1 para los IDs suele ser más común.
	return k.ProposalSeq.Next(ctx)
}

// SetProposal sets a proposal in the store.
func (k Keeper) SetProposal(ctx sdk.Context, proposal types.Proposal) error {
	return k.Proposals.Set(ctx, proposal.Id, proposal)
}

// GetProposal retrieves a proposal from the store by its ID.
func (k Keeper) GetProposal(ctx sdk.Context, proposalID uint64) (types.Proposal, error) {
	// El método Get de collections.Map devuelve el valor y un error.
	// El error será collections.ErrNotFound si no se encuentra.
	return k.Proposals.Get(ctx, proposalID)
}

// DeleteProposal removes a proposal from the store.
// (Podrías necesitar esto más adelante, por ejemplo, para limpiar propuestas antiguas o fallidas)
// func (k Keeper) DeleteProposal(ctx sdk.Context, proposalID uint64) error {
// 	return k.Proposals.Remove(ctx, proposalID)
// }

// IterateProposals iterates over all proposals in the store and performs a callback function.
func (k Keeper) IterateProposals(ctx sdk.Context, cb func(proposal types.Proposal) (stop bool)) error {
	return k.Proposals.Walk(ctx, nil, func(key uint64, proposal types.Proposal) (bool, error) {
		return cb(proposal), nil
	})
}

// --- Vote Helpers ---

// SetVote sets a vote in the store.
func (k Keeper) SetVote(ctx sdk.Context, vote types.Vote) error {
	voterAddr, err := k.addressCodec.StringToBytes(vote.Voter)
	if err != nil {
		return err // Debería haber sido validado antes, pero por seguridad
	}
	// La clave para los votos es un par (ProposalID, VoterAddress)
	key := collections.Join(vote.ProposalId, sdk.AccAddress(voterAddr))
	return k.Votes.Set(ctx, key, vote)
}

// GetVote retrieves a vote from the store.
func (k Keeper) GetVote(ctx sdk.Context, proposalID uint64, voter sdk.AccAddress) (types.Vote, error) {
	key := collections.Join(proposalID, voter)
	return k.Votes.Get(ctx, key)
}

// IterateVotesByProposal iterates over all votes for a specific proposal.
func (k Keeper) IterateVotesByProposal(ctx sdk.Context, proposalID uint64, cb func(vote types.Vote) (stop bool)) error {
	// Para iterar votos por ProposalID, necesitarías un índice secundario o un prefijo.
	// El KeySet actual para Votes es (ProposalID, VoterAddress).
	// Podrías usar un iterador con un prefijo si construyes la clave de prefijo correctamente.
	// Ejemplo de iteración (puede necesitar ajuste para el PairKey):
	prefixRange := collections.NewPrefixedPairRange[uint64, sdk.AccAddress](proposalID)
	return k.Votes.Walk(ctx, prefixRange, func(key collections.Pair[uint64, sdk.AccAddress], vote types.Vote) (bool, error) {
		return cb(vote), nil
	})
}

// --- Active Proposal Queue Helpers ---
// (Estos son placeholders y probablemente necesitarán una implementación más robusta
//  usando el end_block para la ordenación y el procesamiento en EndBlocker)

// AddToActiveProposalQueue adds a proposal ID to a simple set for active proposals.
func (k Keeper) AddToActiveProposalQueue(ctx sdk.Context, proposalID uint64) error {
	return k.ActiveProposalQueue.Set(ctx, proposalID)
}

// RemoveFromActiveProposalQueue removes a proposal ID from the active set.
func (k Keeper) RemoveFromActiveProposalQueue(ctx sdk.Context, proposalID uint64) error {
	return k.ActiveProposalQueue.Remove(ctx, proposalID)
}

// GetActiveProposals returns all proposal IDs currently in the active queue.
// En una implementación real, esto iteraría sobre propuestas con estado VOTING_PERIOD.
func (k Keeper) GetActiveProposals(ctx sdk.Context) ([]uint64, error) {
	var proposalIDs []uint64
	err := k.ActiveProposalQueue.Walk(ctx, nil, func(proposalID uint64) (bool, error) {
		proposalIDs = append(proposalIDs, proposalID)
		return false, nil
	})
	return proposalIDs, err
}
