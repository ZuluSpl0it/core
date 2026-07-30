package keeper

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestPositionRoundTripPreservesTierSnapshot(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	position := types.Position{
		Id:               7,
		Owner:            "cosmos1owner",
		Principal:        sdk.NewCoin(types.BondDenom, math.ZeroInt()),
		Shares:           math.NewInt(150),
		ShareMultiplier:  math.LegacyNewDecWithPrec(15, 1),
		RewardDebt:       math.LegacyZeroDec(),
		Status:           types.PositionStatus_POSITION_STATUS_ACTIVE,
		ClaimableRewards: sdk.NewCoin(types.BondDenom, math.ZeroInt()),
	}

	keeper.SetPosition(ctx, position)
	got, found := keeper.GetPosition(ctx, position.Id)

	require.True(t, found)
	require.Equal(t, position, got)
}

func TestAccruedRewardsNeverReturnsNegative(t *testing.T) {
	position := types.Position{
		Shares:     math.NewInt(10),
		RewardDebt: math.LegacyNewDec(20),
		Status:     types.PositionStatus_POSITION_STATUS_ACTIVE,
	}
	state := types.RewardState{
		RewardIndex: math.LegacyOneDec(),
		TotalShares: math.NewInt(10),
	}

	require.True(t, keeperAccruedRewards(position, state).IsZero())
}
