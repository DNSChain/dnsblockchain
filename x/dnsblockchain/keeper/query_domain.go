package keeper

import (
	"context"
	"errors"
	"strings"

	"dnsblockchain/x/dnsblockchain/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetDomain implementa el RPC para obtener un dominio por ID.
func (q queryServer) GetDomain(ctx context.Context, req *types.QueryGetDomainRequest) (*types.QueryGetDomainResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	domain, err := q.k.Domain.Get(ctx, req.Id)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, errorsmod.Wrapf(sdkerrors.ErrKeyNotFound, "domain with id %d not found", req.Id)
		}
		return nil, status.Errorf(codes.Internal, "internal error getting domain by id %d: %v", req.Id, err)
	}

	return &types.QueryGetDomainResponse{Domain: domain}, nil
}

// ListDomain implementa el RPC para listar dominios.
func (q queryServer) ListDomain(ctx context.Context, req *types.QueryAllDomainRequest) (*types.QueryAllDomainResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	domains, pageRes, err := query.CollectionPaginate(
		ctx,
		q.k.Domain, // Usa el mapa de dominios del keeper
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

// GetDomainByName implementa el RPC para obtener un dominio por su nombre FQDN.
func (q queryServer) GetDomainByName(goCtx context.Context, req *types.QueryGetDomainByNameRequest) (*types.QueryGetDomainByNameResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "domain name cannot be empty")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	normalizedName := strings.ToLower(strings.Trim(req.Name, "."))

	domainID, err := q.k.DomainName.Get(ctx, normalizedName)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			q.k.Logger(ctx).Info("Domain not found by name in index", "name", normalizedName)
			return &types.QueryGetDomainByNameResponse{Found: false, Expired: false}, nil
		}
		q.k.Logger(ctx).Error("Error getting domain ID from name index", "name", normalizedName, "error", err)
		return nil, status.Errorf(codes.Internal, "internal error getting domain ID by name: %v", err)
	}

	domain, err := q.k.Domain.Get(ctx, domainID)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			q.k.Logger(ctx).Error("Data inconsistency: Domain ID found in name index, but domain object not found by ID", "name", normalizedName, "id", domainID)
			// Opcional: Limpiar la entrada del índice si se detecta esta inconsistencia
			// _ = q.k.DomainName.Remove(ctx, normalizedName)
			return &types.QueryGetDomainByNameResponse{Found: false, Expired: false}, nil
		}
		q.k.Logger(ctx).Error("Error getting domain object by ID after lookup from name index", "name", normalizedName, "id", domainID, "error", err)
		return nil, status.Errorf(codes.Internal, "internal error getting domain by ID: %v", err)
	}

	isExpired := uint64(ctx.BlockTime().Unix()) >= domain.Expiration
	if isExpired {
		q.k.Logger(ctx).Info("Domain found by name, but is expired", "name", normalizedName, "expiration", domain.Expiration)
	}

	return &types.QueryGetDomainByNameResponse{Domain: domain, Found: true, Expired: isExpired}, nil
}
