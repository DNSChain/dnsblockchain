package keeper

import (
	"context"
	"errors" // Asegúrate de que esté importado

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/collections"
	// "cosmossdk.io/errors" // Ya no se usa directamente 'errors.Is' de cosmossdk.io/errors
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"dnsblockchain/x/dao/types"
)

// Vote queries a vote by proposal ID and voter address.
func (k queryServer) Vote(goCtx context.Context, req *types.QueryVoteRequest) (*types.QueryVoteResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	// if req.ProposalId == 0 { // La propuesta ID 0 puede ser válida si la secuencia empieza ahí
	// 	 return nil, status.Error(codes.InvalidArgument, "proposal_id cannot be 0")
	// }
	if req.Voter == "" {
		return nil, status.Error(codes.InvalidArgument, "voter address cannot be empty")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	voterAddr, err := k.k.addressCodec.StringToBytes(req.Voter)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid voter address: %v", err)
	}

	vote, err := k.k.GetVote(ctx, req.ProposalId, sdk.AccAddress(voterAddr))
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) { // Usar errors.Is del paquete estándar
			return nil, status.Errorf(codes.NotFound, "vote by %s on proposal %d not found", req.Voter, req.ProposalId)
		}
		return nil, status.Errorf(codes.Internal, "failed to get vote: %v", err)
	}

	return &types.QueryVoteResponse{Vote: vote}, nil
}

// Votes queries all votes by proposal ID.
func (k queryServer) Votes(goCtx context.Context, req *types.QueryVotesRequest) (*types.QueryVotesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	// if req.ProposalId == 0 {
	// 	 return nil, status.Error(codes.InvalidArgument, "proposal_id cannot be 0")
	// }

	ctx := sdk.UnwrapSDKContext(goCtx)

	var votes []types.Vote
	var pageRes *query.PageResponse

	var allVotesForProposal []types.Vote
	prefixRange := collections.NewPrefixedPairRange[uint64, sdk.AccAddress](req.ProposalId)
	err := k.k.Votes.Walk(ctx, prefixRange, func(key collections.Pair[uint64, sdk.AccAddress], vote types.Vote) (stop bool, iterErr error) {
		allVotesForProposal = append(allVotesForProposal, vote)
		return false, nil
	})

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to iterate votes: %v", err)
	}

	page := req.Pagination
	if page == nil {
		page = &query.PageRequest{
			Limit: query.DefaultLimit,
		}
	}
	if page.Limit == 0 {
		page.Limit = query.DefaultLimit
	}
	if page.Offset < 0 {
		page.Offset = 0
	}

	total := uint64(len(allVotesForProposal))
	start := int(page.Offset)
	end := start + int(page.Limit)

	if start < 0 {
		start = 0
	}
	if start > len(allVotesForProposal) {
		start = len(allVotesForProposal)
	}
	if end < 0 {
		end = 0
	}
	if end > len(allVotesForProposal) {
		end = len(allVotesForProposal)
	}

	if start >= end {
		votes = []types.Vote{}
	} else {
		votes = allVotesForProposal[start:end]
	}

	pageRes = &query.PageResponse{
		Total: total,
	}

	return &types.QueryVotesResponse{Votes: votes, Pagination: pageRes}, nil
}
