package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	// "cosmossdk.io/collections" // No es necesario si usamos Walk directamente
	// "cosmossdk.io/store/prefix" // No es necesario
	// "github.com/cosmos/cosmos-sdk/runtime" // No es necesario
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"dnsblockchain/x/dao/types"
)

// Proposals queries all proposals based on given filters.
func (k queryServer) Proposals(goCtx context.Context, req *types.QueryProposalsRequest) (*types.QueryProposalsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	// storeAdapter := runtime.KVStoreAdapter(k.k.storeService.OpenKVStore(ctx)) // No se usa
	// store := prefix.NewStore(storeAdapter, types.ProposalsKeyPrefix.Bytes()) // No se usa

	var proposals []types.Proposal
	var pageRes *query.PageResponse
	var err error

	var filteredProposals []types.Proposal
	err = k.k.Proposals.Walk(ctx, nil, func(key uint64, p types.Proposal) (stop bool, iterErr error) {
		match := true
		if req.Proposer != "" {
			// Validar la dirección del proponente si se proporciona
			_, errAddr := k.k.addressCodec.StringToBytes(req.Proposer)
			if errAddr != nil {
				// Opcional: podrías devolver un error de argumento inválido aquí en lugar de solo no coincidir
				// return true, status.Errorf(codes.InvalidArgument, "invalid proposer address in filter: %v", errAddr)
				match = false // Considerar inválido si la dirección no es parseable
			} else if p.Proposer != req.Proposer {
				match = false
			}
		}
		if match && req.ProposalStatus != types.ProposalStatus_PROPOSAL_STATUS_UNSPECIFIED {
			if p.Status != req.ProposalStatus {
				match = false
			}
		}
		// El filtro por 'voter' se omite por simplicidad como se discutió.

		if match {
			filteredProposals = append(filteredProposals, p)
		}
		return false, nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	page := req.Pagination
	if page == nil {
		page = &query.PageRequest{
			Limit: query.DefaultLimit, // Asegurar un límite si la paginación es nil
		}
	}
	if page.Limit == 0 {
		page.Limit = query.DefaultLimit
	}
	if page.Offset < 0 { // Sanity check para offset
		page.Offset = 0
	}

	total := uint64(len(filteredProposals))

	// query.Paginate espera page 1-based, nuestro offset es 0-based
	// El SDK más reciente podría manejar esto de forma diferente,
	// pero para una paginación en memoria manual:
	start := int(page.Offset)
	end := start + int(page.Limit)

	if start < 0 {
		start = 0
	}
	if start > len(filteredProposals) {
		start = len(filteredProposals)
	}
	if end < 0 { // Aunque start >= 0 lo haría improbable
		end = 0
	}
	if end > len(filteredProposals) {
		end = len(filteredProposals)
	}

	if start >= end { // Si el offset está más allá de los datos, o límite es 0
		proposals = []types.Proposal{}
	} else {
		proposals = filteredProposals[start:end]
	}

	pageRes = &query.PageResponse{
		Total: total,
		// NextKey es difícil de calcular de forma fiable con paginación en memoria y filtrado.
		// Se deja vacío por ahora. Para una paginación robusta con NextKey,
		// la iteración del store y el filtrado deberían integrarse con la lógica de paginación.
	}

	return &types.QueryProposalsResponse{Proposals: proposals, Pagination: pageRes}, nil
}
