package keeper

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	secp256k1 "github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestStakeTransfersOnlyUSTCToPrincipalPool(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	configureKeeper(t, ctx, keeper, owner)
	server := NewMsgServerImpl(keeper)

	response, err := server.Stake(sdk.WrapSDKContext(ctx), &types.MsgStake{
		Owner: owner, Amount: sdk.NewCoin(types.BondDenom, math.NewInt(100)), LockTierId: 1,
	})

	require.NoError(t, err)
	require.Equal(t, uint64(1), response.PositionId)
	position, found := keeper.GetPosition(ctx, response.PositionId)
	require.True(t, found)
	require.Equal(t, math.NewInt(150), position.Shares)
	require.Equal(t, types.PositionStatus_POSITION_STATUS_ACTIVE, position.Status)

	bank := keeper.bankKeeper.(*recordingBankKeeper)
	require.Equal(t, types.PrincipalPoolName, bank.lastRecipientModule)
	require.Equal(t, sdk.NewCoins(sdk.NewCoin(types.BondDenom, math.NewInt(100))), bank.lastAccountToModuleAmount)
}

func TestStakeRejectsPositionIDExhaustionBeforeTransfer(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	configureKeeper(t, ctx, keeper, owner)
	keeper.SetNextPositionID(ctx, ^uint64(0))
	server := NewMsgServerImpl(keeper)
	bank := keeper.bankKeeper.(*recordingBankKeeper)

	_, err := server.Stake(sdk.WrapSDKContext(ctx), &types.MsgStake{
		Owner: owner, Amount: sdk.NewCoin(types.BondDenom, math.NewInt(100)), LockTierId: 1,
	})

	require.ErrorIs(t, err, types.ErrPositionIDExhausted)
	require.Zero(t, bank.accountToModuleCalls)
}

func TestBeginUnbondingSettlesRewardsAndStopsFutureAccrual(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	configureKeeper(t, ctx, keeper, owner)
	duration := 24 * time.Hour
	keeper.SetRewardState(ctx, types.RewardState{RewardIndex: math.LegacyNewDecWithPrec(5, 1), TotalShares: math.NewInt(100)})
	keeper.SetPosition(ctx, types.Position{
		Id: 1, Owner: owner, Principal: sdk.NewCoin(types.BondDenom, math.NewInt(100)),
		Shares: math.NewInt(100), RewardDebt: math.LegacyZeroDec(), Status: types.PositionStatus_POSITION_STATUS_ACTIVE,
		LockDuration: &duration,
	})
	server := NewMsgServerImpl(keeper)

	_, err := server.BeginUnbonding(sdk.WrapSDKContext(ctx), &types.MsgBeginUnbonding{Owner: owner, PositionId: 1})
	require.NoError(t, err)

	position, _ := keeper.GetPosition(ctx, 1)
	require.True(t, position.Shares.IsZero())
	require.Equal(t, math.NewInt(50), position.ClaimableRewards.Amount)
	require.Equal(t, types.PositionStatus_POSITION_STATUS_UNBONDING, position.Status)
	require.Equal(t, math.ZeroInt(), keeper.GetRewardState(ctx).TotalShares)

	state := keeper.GetRewardState(ctx)
	state.RewardIndex = math.LegacyNewDec(10)
	require.Equal(t, math.NewInt(50), keeper.AccruedRewards(position, state))
}

func TestWithdrawBeforeCompletionFails(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	configureKeeper(t, ctx, keeper, owner)
	end := ctx.BlockTime().Add(time.Hour)
	keeper.SetPosition(ctx, types.Position{
		Id: 1, Owner: owner, Principal: sdk.NewCoin(types.BondDenom, math.NewInt(100)),
		Status: types.PositionStatus_POSITION_STATUS_UNBONDING, UnbondingEndTime: &end,
		ClaimableRewards: sdk.NewCoin(types.BondDenom, math.ZeroInt()),
	})
	server := NewMsgServerImpl(keeper)

	_, err := server.Withdraw(sdk.WrapSDKContext(ctx), &types.MsgWithdraw{Owner: owner, PositionId: 1})
	require.ErrorIs(t, err, types.ErrNotMatured)
}

func TestClaimFailsWhenRewardPoolCannotCoverLiability(t *testing.T) {
	ctx, keeper := newKeeperTest(t)
	owner := testAddress()
	configureKeeper(t, ctx, keeper, owner)
	keeper.SetRewardState(ctx, types.RewardState{RewardIndex: math.LegacyOneDec(), TotalShares: math.NewInt(10)})
	keeper.SetPosition(ctx, types.Position{
		Id: 1, Owner: owner, Shares: math.NewInt(10), RewardDebt: math.LegacyZeroDec(),
		Status: types.PositionStatus_POSITION_STATUS_ACTIVE,
	})
	keeper.bankKeeper.(*recordingBankKeeper).rewardPoolBalance = sdk.NewCoin(types.BondDenom, math.NewInt(5))
	server := NewMsgServerImpl(keeper)

	_, err := server.ClaimRewards(sdk.WrapSDKContext(ctx), &types.MsgClaimRewards{Owner: owner, PositionId: 1})
	require.ErrorIs(t, err, types.ErrRewardPoolInsolvent)
	require.Zero(t, keeper.bankKeeper.(*recordingBankKeeper).moduleToAccountCalls)
}

func configureKeeper(t *testing.T, ctx sdk.Context, keeper Keeper, authority string) {
	t.Helper()
	duration := 24 * time.Hour
	keeper.SetParams(ctx, types.Params{
		BondDenom: types.BondDenom,
		LockTiers: []types.LockTier{{Id: 1, Duration: &duration, Multiplier: math.LegacyNewDecWithPrec(15, 1)}},
		Authority: authority, FundingAuthority: authority,
	})
	keeper.SetRewardState(ctx, types.RewardState{RewardIndex: math.LegacyZeroDec(), TotalShares: math.ZeroInt()})
}

func testAddress() string {
	return sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
}
