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
	index := prefix.NewStore(ctx.KVStore(q.storeKey), types.OwnerPositionKeyPrefix)
	owner, _ := sdk.AccAddressFromBech32(req.Owner)
	store := prefix.NewStore(index, owner.Bytes())
	positions := make([]types.Position, 0)
	pageRes, err := query.Paginate(store, req.Pagination, func(key []byte, _ []byte) error {
		if len(key) != 8 {
			return types.ErrInvalidPosition.Wrapf("invalid owner index key length %d", len(key))
		}
		positionID := sdk.BigEndianToUint64(key)
		position, found := q.GetPosition(ctx, positionID)
		if !found {
			return types.ErrInvalidPosition.Wrapf("owner index references missing position %d", positionID)
		}
		if position.Owner != req.Owner {
			return types.ErrInvalidPosition.Wrapf("owner index position %d belongs to %s", positionID, position.Owner)
		}
		positions = append(positions, position)
		return nil
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
