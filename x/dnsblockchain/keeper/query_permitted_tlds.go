package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	// "cosmossdk.io/collections"
	// sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	// "github.com/cosmos/cosmos-sdk/types/query"

	"dnsblockchain/x/dnsblockchain/types"
)

func (k queryServer) ListPermittedTLDs(ctx context.Context, req *types.QueryListPermittedTLDsRequest) (*types.QueryListPermittedTLDsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var tlds []string
	err := k.k.PermittedTLDs.Walk(ctx, nil, func(tld string) (stop bool, err error) {
		tlds = append(tlds, tld)
		return false, nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryListPermittedTLDsResponse{Tlds: tlds}, nil
}
