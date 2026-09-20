package keeper

import (
	"errors"
	"testing"

	"cosmossdk.io/math"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestFundThenAccrueUsesFixedPointIndex(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	keeper.SetRewardState(ctx, types.RewardState{
		RewardIndex: math.LegacyZeroDec(),
		TotalShares: math.NewInt(1000),
	})
	position := types.Position{
		Id:         1,
		Shares:     math.OneInt(),
		RewardDebt: math.LegacyZeroDec(),
		Status:     types.PositionStatus_POSITION_STATUS_ACTIVE,
	}

	for i := 0; i < 1000; i++ {
		require.NoError(t, keeper.FundRewards(ctx, sdk.NewCoin(types.BondDenom, math.OneInt())))
	}

	require.Equal(t, math.OneInt(), keeper.AccruedRewards(position, keeper.GetRewardState(ctx)))
	distributor := keeper.communityPoolKeeper.(*recordingCommunityPoolKeeper)
	require.Equal(t, 1000, distributor.calls)
	require.Equal(t, types.RewardPoolName, distributor.recipientModule)
	require.Equal(t, sdk.NewCoins(sdk.NewCoin(types.BondDenom, math.OneInt())), distributor.amount)
}

func TestFundingWithNoActiveSharesFailsWithoutTransfer(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	keeper.SetRewardState(ctx, types.RewardState{RewardIndex: math.LegacyZeroDec(), TotalShares: math.ZeroInt()})

	err := keeper.FundRewards(ctx, sdk.NewCoin(types.BondDenom, math.OneInt()))

	require.ErrorIs(t, err, types.ErrNoActiveShares)
	require.Zero(t, keeper.communityPoolKeeper.(*recordingCommunityPoolKeeper).calls)
}

func TestFundRewardsLeavesIndexUnchangedWhenCommunityPoolFails(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	initial := types.RewardState{RewardIndex: math.LegacyOneDec(), TotalShares: math.NewInt(100)}
	keeper.SetRewardState(ctx, initial)
	keeper.communityPoolKeeper.(*recordingCommunityPoolKeeper).err = errors.New("insufficient community pool")

	err := keeper.FundRewards(ctx, sdk.NewCoin(types.BondDenom, math.NewInt(40)))

	require.Error(t, err)
	require.Equal(t, initial, keeper.GetRewardState(ctx))
}
