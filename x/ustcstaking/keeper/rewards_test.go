package keeper

import (
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
		require.NoError(t, keeper.FundRewards(ctx, sdk.AccAddress{}, sdk.NewCoin(types.BondDenom, math.OneInt())))
	}

	require.Equal(t, math.OneInt(), keeper.AccruedRewards(position, keeper.GetRewardState(ctx)))
	require.Equal(t, 1000, keeper.bankKeeper.(*recordingBankKeeper).accountToModuleCalls)
}

func TestFundingWithNoActiveSharesFailsWithoutTransfer(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	keeper.SetRewardState(ctx, types.RewardState{RewardIndex: math.LegacyZeroDec(), TotalShares: math.ZeroInt()})

	err := keeper.FundRewards(ctx, sdk.AccAddress{}, sdk.NewCoin(types.BondDenom, math.OneInt()))

	require.ErrorIs(t, err, types.ErrNoActiveShares)
	require.Zero(t, keeper.bankKeeper.(*recordingBankKeeper).accountToModuleCalls)
}
