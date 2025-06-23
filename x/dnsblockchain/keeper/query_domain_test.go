package keeper_test

import (
	"context"
	"strconv"
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"dnsblockchain/x/dnsblockchain/keeper"
	"dnsblockchain/x/dnsblockchain/types"
)

func createNDomain(keeper keeper.Keeper, ctx context.Context, n int) []types.Domain {
	items := make([]types.Domain, n)
	for i := range items {
		iu := uint64(i)
		items[i].Id = iu
		items[i].Name = strconv.Itoa(i)
		items[i].Owner = strconv.Itoa(i)
		items[i].Expiration = uint64(i)
		items[i].Ns = strconv.Itoa(i)
		_ = keeper.Domain.Set(ctx, iu, items[i])
		_ = keeper.DomainSeq.Set(ctx, iu)
	}
	return items
}

func TestDomainQuerySingle(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	msgs := createNDomain(f.keeper, f.ctx, 2)
	tests := []struct {
		desc     string
		request  *types.QueryGetDomainRequest
		response *types.QueryGetDomainResponse
		err      error
	}{
		{
			desc:     "First",
			request:  &types.QueryGetDomainRequest{Id: msgs[0].Id},
			response: &types.QueryGetDomainResponse{Domain: msgs[0]},
		},
		{
			desc:     "Second",
			request:  &types.QueryGetDomainRequest{Id: msgs[1].Id},
			response: &types.QueryGetDomainResponse{Domain: msgs[1]},
		},
		{
			desc:    "KeyNotFound",
			request: &types.QueryGetDomainRequest{Id: uint64(len(msgs))},
			err:     sdkerrors.ErrKeyNotFound,
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := qs.GetDomain(f.ctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				require.EqualExportedValues(t, tc.response, response)
			}
		})
	}
}

func TestDomainQueryPaginated(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	msgs := createNDomain(f.keeper, f.ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllDomainRequest {
		return &types.QueryAllDomainRequest{
			Pagination: &query.PageRequest{
				Key:        next,
				Offset:     offset,
				Limit:      limit,
				CountTotal: total,
			},
		}
	}
	t.Run("ByOffset", func(t *testing.T) {
		step := 2
		for i := 0; i < len(msgs); i += step {
			resp, err := qs.ListDomain(f.ctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Domain), step)
			require.Subset(t, msgs, resp.Domain)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := qs.ListDomain(f.ctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Domain), step)
			require.Subset(t, msgs, resp.Domain)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := qs.ListDomain(f.ctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.EqualExportedValues(t, msgs, resp.Domain)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := qs.ListDomain(f.ctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
