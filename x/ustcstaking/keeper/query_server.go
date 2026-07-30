package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
)

type queryServer struct{ Keeper }

func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

func (q queryServer) Position(goCtx context.Context, req *types.QueryPositionRequest) (*types.QueryPositionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	position, found := q.GetPosition(ctx, req.PositionId)
	if !found {
		return nil, sdkerrors.ErrNotFound.Wrapf("position %d", req.PositionId)
	}
	return &types.QueryPositionResponse{Position: position}, nil
}

func (q queryServer) PositionsByOwner(goCtx context.Context, req *types.QueryPositionsByOwnerRequest) (*types.QueryPositionsByOwnerResponse, error) {
	if _, err := sdk.AccAddressFromBech32(req.Owner); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("owner: %s", err)
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	store := prefix.NewStore(ctx.KVStore(q.storeKey), types.PositionKeyPrefix)
	positions := make([]types.Position, 0)
	pageRes, err := query.FilteredPaginate(store, req.Pagination, func(_ []byte, value []byte, accumulate bool) (bool, error) {
		var position types.Position
		q.cdc.MustUnmarshal(value, &position)
		if position.Owner != req.Owner {
			return false, nil
		}
		if accumulate {
			positions = append(positions, position)
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return &types.QueryPositionsByOwnerResponse{Positions: positions, Pagination: pageRes}, nil
}

func (q queryServer) RewardState(goCtx context.Context, _ *types.QueryRewardStateRequest) (*types.QueryRewardStateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	return &types.QueryRewardStateResponse{RewardState: q.GetRewardState(ctx)}, nil
}

func (q queryServer) Params(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	return &types.QueryParamsResponse{Params: q.GetParams(ctx)}, nil
}
