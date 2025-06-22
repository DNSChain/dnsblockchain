package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/collections"
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	// sdkerrors "github.com/cosmos/cosmos-sdk/types/errors" // Si necesitas errores específicos del SDK

	"dnsblockchain/x/dao/types"
)

// Proposal queries a proposal by ID
func (k queryServer) Proposal(goCtx context.Context, req *types.QueryProposalRequest) (*types.QueryProposalResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.ProposalId == 0 && req.ProposalId != 0 { // Esto es una forma de permitir ID 0 temporalmente, aunque una propuesta con ID 0 real podría no existir si la secuencia empieza en 1.
		// Para ser estrictos, si los IDs de propuesta *nunca* pueden ser 0, entonces:
		// if req.ProposalId == 0 {
		// 	return nil, status.Error(codes.InvalidArgument, "proposal_id cannot be 0")
		// }
		// Por ahora, lo dejaremos como está para permitir consultar el ID 0 si es el primero.
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	proposal, err := k.k.GetProposal(ctx, req.ProposalId) // k.k para acceder al keeper desde queryServer
	if err != nil {
		if err == collections.ErrNotFound {
			return nil, errors.Wrapf(types.ErrProposalNotFound, "proposal with id %d not found", req.ProposalId)
		}
		return nil, status.Errorf(codes.Internal, "failed to get proposal %d: %v", req.ProposalId, err)
	}

	return &types.QueryProposalResponse{Proposal: proposal}, nil
}
