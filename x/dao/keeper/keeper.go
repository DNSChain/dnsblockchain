package keeper

import (
	"fmt"
	// Importar context para sdk.Context
	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
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
	ProposalSeq         collections.Sequence
	Proposals           collections.Map[uint64, types.Proposal]
	Votes               collections.Map[collections.Pair[uint64, sdk.AccAddress], types.Vote]
	ActiveProposalQueue collections.KeySet[uint64]
	VoterPowerLots      collections.Map[collections.Pair[sdk.AccAddress, uint64], types.VoterVotingPowerLot]
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
		ProposalSeq:         collections.NewSequence(sb, types.ProposalSeqKey, "proposal_sequence"),
		Proposals:           collections.NewMap(sb, types.ProposalsKeyPrefix, "proposals", collections.Uint64Key, codec.CollValue[types.Proposal](cdc)),
		Votes: collections.NewMap(sb, types.VotesKeyPrefix, "votes",
			collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey),
			codec.CollValue[types.Vote](cdc),
		),
		ActiveProposalQueue: collections.NewKeySet(sb, types.ActiveProposalQueueKeyPrefix, "active_proposal_queue", collections.Uint64Key),
		VoterPowerLots: collections.NewMap(sb, types.VoterPowerLotsKeyPrefix, "voter_power_lots",
			collections.PairKeyCodec(sdk.AccAddressKey, collections.Uint64Key),
			codec.CollValue[types.VoterVotingPowerLot](cdc),
		),
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

func (k Keeper) GetNextProposalID(ctx sdk.Context) (uint64, error) {
	return k.ProposalSeq.Next(ctx)
}

func (k Keeper) SetProposal(ctx sdk.Context, proposal types.Proposal) error {
	return k.Proposals.Set(ctx, proposal.Id, proposal)
}

func (k Keeper) GetProposal(ctx sdk.Context, proposalID uint64) (types.Proposal, error) {
	return k.Proposals.Get(ctx, proposalID)
}

func (k Keeper) IterateProposals(ctx sdk.Context, cb func(proposal types.Proposal) (stop bool)) error {
	return k.Proposals.Walk(ctx, nil, func(key uint64, proposal types.Proposal) (bool, error) {
		return cb(proposal), nil
	})
}

func (k Keeper) SetVote(ctx sdk.Context, vote types.Vote) error {
	voterAddr, err := k.addressCodec.StringToBytes(vote.Voter)
	if err != nil {
		return err
	}
	key := collections.Join(vote.ProposalId, sdk.AccAddress(voterAddr))
	return k.Votes.Set(ctx, key, vote)
}

func (k Keeper) GetVote(ctx sdk.Context, proposalID uint64, voter sdk.AccAddress) (types.Vote, error) {
	key := collections.Join(proposalID, voter)
	return k.Votes.Get(ctx, key)
}

func (k Keeper) IterateVotesByProposal(ctx sdk.Context, proposalID uint64, cb func(vote types.Vote) (stop bool)) error {
	prefixRange := collections.NewPrefixedPairRange[uint64, sdk.AccAddress](proposalID)
	return k.Votes.Walk(ctx, prefixRange, func(key collections.Pair[uint64, sdk.AccAddress], vote types.Vote) (bool, error) {
		return cb(vote), nil
	})
}

func (k Keeper) AddToActiveProposalQueue(ctx sdk.Context, proposalID uint64) error {
	return k.ActiveProposalQueue.Set(ctx, proposalID)
}

func (k Keeper) RemoveFromActiveProposalQueue(ctx sdk.Context, proposalID uint64) error {
	return k.ActiveProposalQueue.Remove(ctx, proposalID)
}

func (k Keeper) GetActiveProposals(ctx sdk.Context) ([]uint64, error) {
	var proposalIDs []uint64
	err := k.ActiveProposalQueue.Walk(ctx, nil, func(proposalID uint64) (bool, error) {
		proposalIDs = append(proposalIDs, proposalID)
		return false, nil
	})
	return proposalIDs, err
}

// IsLotBasedVotingActive verifica si ya existe al menos un lote de poder de voto en el sistema.
func (k Keeper) IsLotBasedVotingActive(ctx sdk.Context) (bool, error) {
	var foundOne bool
	// Iteramos solo hasta encontrar el primero para eficiencia.
	// Usamos un rango nil para iterar sobre todos los VoterPowerLots.
	err := k.VoterPowerLots.Walk(ctx, nil, func(key collections.Pair[sdk.AccAddress, uint64], lot types.VoterVotingPowerLot) (stop bool, err error) {
		foundOne = true
		return true, nil // Detener la iteración
	})
	if err != nil {
		// Este error sería un problema de store, no que no se encontró nada.
		return false, fmt.Errorf("error checking for active power lots: %w", err)
	}
	return foundOne, nil
}

// CalculateVoterPower calcula el poder de voto actual de un votante considerando el decaimiento.
func (k Keeper) CalculateVoterPower(ctx sdk.Context, voterAddr sdk.AccAddress, currentBlockHeight uint64) (math.Int, error) {
	logger := k.Logger(ctx)
	totalPower := math.ZeroInt()
	params, err := k.Params.Get(ctx)

	if err != nil {
		return math.ZeroInt(), fmt.Errorf("failed to get dao params for power calculation: %w", err)
	}

	lotBasedVotingActive, err := k.IsLotBasedVotingActive(ctx)
	if err != nil {
		logger.Error("CalculateVoterPower: Error checking if lot-based voting is active", "error", err)
		return math.ZeroInt(), err
	}

	if !lotBasedVotingActive {
		logger.Info("CalculateVoterPower: Lot-based voting not active. Using general balance.", "voter", voterAddr.String())
		basePowerCoin := k.bankKeeper.GetBalance(ctx, voterAddr, params.VotingTokenDenom)
		return basePowerCoin.Amount, nil
	}

	// Si lotBasedVotingActive es true, procedemos con la lógica de lotes
	decayDurationBlocks := params.VotingPowerDecayDurationBlocks
	if decayDurationBlocks == 0 {
		logger.Info("CalculateVoterPower: VotingPowerDecayDurationBlocks is zero, power from specific lots will not decay.")
	}

	prefixRange := collections.NewPrefixedPairRange[sdk.AccAddress, uint64](voterAddr)
	foundLots := false
	err = k.VoterPowerLots.Walk(ctx, prefixRange, func(key collections.Pair[sdk.AccAddress, uint64], lot types.VoterVotingPowerLot) (stop bool, err error) {
		foundLots = true
		logger.Debug("CalculateVoterPower: Processing lot", "voter", voterAddr.String(), "grantProposalID", lot.GrantedByProposalId, "initialAmount", lot.InitialAmount.String(), "grantBlock", lot.GrantBlockHeight)
		if lot.InitialAmount.IsZero() {
			return false, nil
		}
		if decayDurationBlocks == 0 {
			totalPower = totalPower.Add(lot.InitialAmount)
			logger.Debug("CalculateVoterPower: Lot power added (no decay)", "addedPower", lot.InitialAmount.String(), "newTotalPower", totalPower.String())
			return false, nil
		}

		blocksPassed := currentBlockHeight - lot.GrantBlockHeight
		if blocksPassed >= decayDurationBlocks {
			return false, nil
		}

		remainingRatio := math.LegacyOneDec().Sub(math.LegacyNewDec(int64(blocksPassed)).QuoInt64(int64(decayDurationBlocks)))
		decayedPower := math.LegacyNewDecFromInt(lot.InitialAmount).Mul(remainingRatio).TruncateInt()

		logger.Debug("CalculateVoterPower: Lot decay calculation", "blocksPassed", blocksPassed, "remainingRatio", remainingRatio.String(), "decayedPower", decayedPower.String())

		if decayedPower.IsPositive() {
			totalPower = totalPower.Add(decayedPower)
			logger.Debug("CalculateVoterPower: Lot power added (with decay)", "addedDecayedPower", decayedPower.String(), "newTotalPower", totalPower.String())
		}
		return false, nil
	})

	if err != nil {
		logger.Error("CalculateVoterPower: Error iterating voter power lots", "voter", voterAddr.String(), "error", err)
		return math.ZeroInt(), fmt.Errorf("failed to iterate voter power lots for %s: %w", voterAddr.String(), err)
	}

	// Si el sistema de lotes está activo (lotBasedVotingActive == true):
	// - Si el votante tiene lotes, totalPower es la suma del poder decaído de esos lotes.
	// - Si el votante NO tiene lotes (foundLots == false), totalPower seguirá siendo ZeroInt.
	// - Si el votante tenía lotes pero todos decayeron, totalPower será ZeroInt.
	// En ninguno de estos casos (cuando lotBasedVotingActive es true) recurrimos al balance general.
	if lotBasedVotingActive {
		if !foundLots {
			logger.Info("CalculateVoterPower: Lot-based voting active. Voter has no lots.", "voter", voterAddr.String(), "finalPower", totalPower.String())
		} else if totalPower.IsZero() { // Found lots, but they all decayed or were zero initially
			logger.Info("CalculateVoterPower: Lot-based voting active. All lots for voter decayed or were zero.", "voter", voterAddr.String(), "finalPower", totalPower.String())
		} else { // Found lots and they have some power
			logger.Info("CalculateVoterPower: Using power from active/decaying lots", "voter", voterAddr.String(), "power_from_lots", totalPower.String())
		}
	}
	// El caso de !lotBasedVotingActive ya se manejó al principio de la función.

	return totalPower, nil
}
