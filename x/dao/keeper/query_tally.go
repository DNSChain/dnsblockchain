package keeper

import (
	"context"
	"errors" // Importar el paquete errors estándar

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/collections"
	// "cosmossdk.io/errors" // Ya no se usa directamente 'errors.Is' de cosmossdk.io/errors
	sdk "github.com/cosmos/cosmos-sdk/types"

	"dnsblockchain/x/dao/types"
)

// TallyResult queries the tally result of a proposal.
func (k queryServer) TallyResult(goCtx context.Context, req *types.QueryTallyResultRequest) (*types.QueryTallyResultResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	// if req.ProposalId == 0 {
	// 	 return nil, status.Error(codes.InvalidArgument, "proposal_id cannot be 0")
	// }

	ctx := sdk.UnwrapSDKContext(goCtx)
	proposal, err := k.k.GetProposal(ctx, req.ProposalId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) { // Usar errors.Is del paquete estándar
			return nil, status.Errorf(codes.NotFound, "proposal %d not found", req.ProposalId)
		}
		return nil, status.Errorf(codes.Internal, "failed to get proposal %d: %v", req.ProposalId, err)
	}

	tally := types.TallyResult{
		YesCount:     proposal.YesVotes,
		AbstainCount: proposal.AbstainVotes,
		NoCount:      proposal.NoVotes,
	}

	return &types.QueryTallyResultResponse{Tally: tally}, nil
}
