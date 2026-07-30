package keeper

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	query "github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
)

func TestQueryPositionReturnsNotFoundForUnknownID(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	server := NewQueryServerImpl(keeper)

	_, err := server.Position(sdk.WrapSDKContext(ctx), &types.QueryPositionRequest{PositionId: 99})

	require.Error(t, err)
}

func TestQueryPositionsByOwnerIsDeterministicAndPaginated(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	other := testAddress()
	for _, position := range []types.Position{
		{Id: 3, Owner: owner, Shares: math.NewInt(30)},
		{Id: 1, Owner: owner, Shares: math.NewInt(10)},
		{Id: 2, Owner: other, Shares: math.NewInt(20)},
	} {
		keeper.SetPosition(ctx, position)
	}
	server := NewQueryServerImpl(keeper)

	response, err := server.PositionsByOwner(sdk.WrapSDKContext(ctx), &types.QueryPositionsByOwnerRequest{
		Owner: owner, Pagination: &query.PageRequest{Limit: 1},
	})

	require.NoError(t, err)
	require.Len(t, response.Positions, 1)
	require.Equal(t, uint64(1), response.Positions[0].Id)
	require.NotEmpty(t, response.Pagination.NextKey)
}

func TestQueriesDoNotMutateRewardDebt(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	position := types.Position{
		Id: ownerPositionID, Owner: owner, Shares: math.NewInt(10),
		RewardDebt: math.LegacyZeroDec(), Status: types.PositionStatus_POSITION_STATUS_ACTIVE,
	}
	keeper.SetPosition(ctx, position)
	keeper.SetRewardState(ctx, types.RewardState{RewardIndex: math.LegacyOneDec(), TotalShares: math.NewInt(10)})
	server := NewQueryServerImpl(keeper)

	_, err := server.Position(sdk.WrapSDKContext(ctx), &types.QueryPositionRequest{PositionId: ownerPositionID})
	require.NoError(t, err)
	got, _ := keeper.GetPosition(ctx, ownerPositionID)
	require.True(t, got.RewardDebt.IsZero())
}

func TestQueryRewardStateAndParams(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	server := NewQueryServerImpl(keeper)
	keeper.SetRewardState(ctx, types.RewardState{RewardIndex: math.LegacyOneDec(), TotalShares: math.NewInt(7)})

	rewardState, err := server.RewardState(sdk.WrapSDKContext(ctx), &types.QueryRewardStateRequest{})
	require.NoError(t, err)
	require.Equal(t, math.NewInt(7), rewardState.RewardState.TotalShares)

	params, err := server.Params(sdk.WrapSDKContext(ctx), &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, types.DefaultParams(), params.Params)
}

const ownerPositionID uint64 = 42
