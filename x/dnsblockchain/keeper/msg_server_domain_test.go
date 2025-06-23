package keeper_test

import (
	"fmt"
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"dnsblockchain/x/dnsblockchain/keeper"
	"dnsblockchain/x/dnsblockchain/types"
)

func TestDomainMsgServerCreate(t *testing.T) {
	f := initFixture(t)
	srv := keeper.NewMsgServerImpl(f.keeper)

	creator, err := f.addressCodec.BytesToString([]byte("signerAddr__________________"))
	require.NoError(t, err)

	// Create first domain
	resp, err := srv.CreateDomain(f.ctx, &types.MsgCreateDomain{Creator: creator, Name: "test.com", Owner: creator, Ns: "ns1.test.com"})
	require.NoError(t, err)
	require.Equal(t, 0, int(resp.Id))

	// Try to create domain with the same name
	_, err = srv.CreateDomain(f.ctx, &types.MsgCreateDomain{Creator: creator, Name: "test.com", Owner: creator, Ns: "ns2.test.com"})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrDuplicateDomainName)

	// Create a few more unique domains
	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("test%d.com", i)
		resp, err := srv.CreateDomain(f.ctx, &types.MsgCreateDomain{Creator: creator, Name: name, Owner: creator, Ns: "ns1." + name})
		require.NoError(t, err)
		// ID will be i+1 because the first domain had ID 0
		require.Equal(t, i+1, int(resp.Id))
	}
}

func TestDomainMsgServerUpdate(t *testing.T) {
	f := initFixture(t)
	srv := keeper.NewMsgServerImpl(f.keeper)

	creator, err := f.addressCodec.BytesToString([]byte("signerAddr__________________"))
	require.NoError(t, err)

	unauthorizedAddr, err := f.addressCodec.BytesToString([]byte("unauthorizedAddr___________"))
	require.NoError(t, err)

	_, err = srv.CreateDomain(f.ctx, &types.MsgCreateDomain{Creator: creator})
	require.NoError(t, err)

	tests := []struct {
		desc    string
		request *types.MsgUpdateDomain
		err     error
	}{
		{
			desc:    "invalid address",
			request: &types.MsgUpdateDomain{Creator: "invalid"},
			err:     sdkerrors.ErrInvalidAddress,
		},
		{
			desc:    "unauthorized",
			request: &types.MsgUpdateDomain{Creator: unauthorizedAddr},
			err:     sdkerrors.ErrUnauthorized,
		},
		{
			desc:    "key not found",
			request: &types.MsgUpdateDomain{Creator: creator, Id: 10},
			err:     sdkerrors.ErrKeyNotFound,
		},
		{
			desc:    "completed",
			request: &types.MsgUpdateDomain{Creator: creator},
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			_, err = srv.UpdateDomain(f.ctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDomainMsgServerDelete(t *testing.T) {
	f := initFixture(t)
	srv := keeper.NewMsgServerImpl(f.keeper)

	creator, err := f.addressCodec.BytesToString([]byte("signerAddr__________________"))
	require.NoError(t, err)

	unauthorizedAddr, err := f.addressCodec.BytesToString([]byte("unauthorizedAddr___________"))
	require.NoError(t, err)

	_, err = srv.CreateDomain(f.ctx, &types.MsgCreateDomain{Creator: creator})
	require.NoError(t, err)

	tests := []struct {
		desc    string
		request *types.MsgDeleteDomain
		err     error
	}{
		{
			desc:    "invalid address",
			request: &types.MsgDeleteDomain{Creator: "invalid"},
			err:     sdkerrors.ErrInvalidAddress,
		},
		{
			desc:    "unauthorized",
			request: &types.MsgDeleteDomain{Creator: unauthorizedAddr},
			err:     sdkerrors.ErrUnauthorized,
		},
		{
			desc:    "key not found",
			request: &types.MsgDeleteDomain{Creator: creator, Id: 10},
			err:     sdkerrors.ErrKeyNotFound,
		},
		{
			desc:    "completed",
			request: &types.MsgDeleteDomain{Creator: creator},
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			_, err = srv.DeleteDomain(f.ctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
