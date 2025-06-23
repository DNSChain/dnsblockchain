package keeper

import (
	"context"
	"errors"

	"dnsblockchain/x/dnsblockchain/types"

	"cosmossdk.io/collections"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q queryServer) ListDomain(ctx context.Context, req *types.QueryAllDomainRequest) (*types.QueryAllDomainResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	domains, pageRes, err := query.CollectionPaginate(
		ctx,
		q.k.Domain,
		req.Pagination,
		func(_ uint64, value types.Domain) (types.Domain, error) {
			return value, nil
		},
	)

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllDomainResponse{Domain: domains, Pagination: pageRes}, nil
}

func (q queryServer) GetDomain(ctx context.Context, req *types.QueryGetDomainRequest) (*types.QueryGetDomainResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	domain, err := q.k.Domain.Get(ctx, req.Id)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, sdkerrors.ErrKeyNotFound
		}

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &types.QueryGetDomainResponse{Domain: domain}, nil
}
