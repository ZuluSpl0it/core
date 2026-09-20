package app_test

import (
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	helperstest "github.com/classic-terra/core/v4/app/testing"
	ustcstakingkeeper "github.com/classic-terra/core/v4/x/ustcstaking/keeper"
	ustcstakingtypes "github.com/classic-terra/core/v4/x/ustcstaking/types"
	"github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktestutil "github.com/cosmos/cosmos-sdk/x/bank/testutil"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"
)

func TestUSTCStakingUsesRealAppKeepers(t *testing.T) {
	var suite helperstest.KeeperTestHelper
	suite.SetT(t)
	suite.Setup(t, "ustc-staking-integration")

	owner := suite.TestAccs[0]
	suite.FundAcc(owner, types.NewCoins(types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(140))))

	principalModule := suite.App.AccountKeeper.GetModuleAccount(suite.Ctx, ustcstakingtypes.PrincipalPoolName)
	rewardModule := suite.App.AccountKeeper.GetModuleAccount(suite.Ctx, ustcstakingtypes.RewardPoolName)
	require.NotNil(t, principalModule)
	require.NotNil(t, rewardModule)
	require.True(t, suite.App.BankKeeper.BlockedAddr(rewardModule.GetAddress()))
	_, principalPermissions := suite.App.AccountKeeper.GetModuleAccountAndPermissions(suite.Ctx, ustcstakingtypes.PrincipalPoolName)
	_, rewardPermissions := suite.App.AccountKeeper.GetModuleAccountAndPermissions(suite.Ctx, ustcstakingtypes.RewardPoolName)
	require.Empty(t, principalPermissions)
	require.Empty(t, rewardPermissions)

	params := ustcstakingtypes.DefaultParams()
	params.Authority = authtypes.NewModuleAddress(govtypes.ModuleName).String()
	params.LockTiers = []ustcstakingtypes.LockTier{{
		Id:         1,
		Duration:   durationPtr(time.Hour),
		Multiplier: sdkmath.LegacyOneDec(),
	}}
	suite.App.UstcStakingKeeper.SetParams(suite.Ctx, params)
	require.NoError(t, banktestutil.FundModuleAccount(suite.Ctx, suite.App.BankKeeper, distrtypes.ModuleName,
		types.NewCoins(types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(40)))))
	feePool, err := suite.App.DistrKeeper.FeePool.Get(suite.Ctx)
	require.NoError(t, err)
	feePool.CommunityPool = types.NewDecCoinsFromCoins(types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(40)))
	require.NoError(t, suite.App.DistrKeeper.FeePool.Set(suite.Ctx, feePool))
	distributionModule := suite.App.AccountKeeper.GetModuleAccount(suite.Ctx, distrtypes.ModuleName)
	require.NotNil(t, distributionModule)
	require.Equal(t, sdkmath.NewInt(40), suite.App.BankKeeper.GetBalance(suite.Ctx, distributionModule.GetAddress(), ustcstakingtypes.BondDenom).Amount)

	initialSupply := suite.App.BankKeeper.GetSupply(suite.Ctx, ustcstakingtypes.BondDenom)
	msgServer := ustcstakingkeeper.NewMsgServerImpl(suite.App.UstcStakingKeeper)
	stakeResponse, err := msgServer.Stake(
		suite.Ctx,
		&ustcstakingtypes.MsgStake{Owner: owner.String(), Amount: types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(100)), LockTierId: 1},
	)
	require.NoError(t, err)
	require.Equal(t, uint64(1), stakeResponse.PositionId)

	_, err = msgServer.FundRewards(
		suite.Ctx,
		&ustcstakingtypes.MsgFundRewards{Authority: params.Authority, Amount: types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(40))},
	)
	require.NoError(t, err)
	require.Equal(t, sdkmath.NewInt(40), suite.App.BankKeeper.GetBalance(suite.Ctx, rewardModule.GetAddress(), ustcstakingtypes.BondDenom).Amount)
	feePool, err = suite.App.DistrKeeper.FeePool.Get(suite.Ctx)
	require.NoError(t, err)
	require.True(t, feePool.CommunityPool.IsZero())
	require.True(t, suite.App.BankKeeper.GetBalance(suite.Ctx, distributionModule.GetAddress(), ustcstakingtypes.BondDenom).IsZero())

	claimResponse, err := msgServer.ClaimRewards(
		suite.Ctx,
		&ustcstakingtypes.MsgClaimRewards{Owner: owner.String(), PositionId: stakeResponse.PositionId},
	)
	require.NoError(t, err)
	require.Equal(t, sdkmath.NewInt(40), claimResponse.Amount.Amount)

	require.Equal(t, sdkmath.NewInt(100), suite.App.BankKeeper.GetBalance(suite.Ctx, principalModule.GetAddress(), ustcstakingtypes.BondDenom).Amount)
	require.True(t, suite.App.BankKeeper.GetBalance(suite.Ctx, rewardModule.GetAddress(), ustcstakingtypes.BondDenom).IsZero())
	feePool, err = suite.App.DistrKeeper.FeePool.Get(suite.Ctx)
	require.NoError(t, err)
	require.True(t, feePool.CommunityPool.IsZero())
	require.True(t, suite.App.BankKeeper.GetBalance(suite.Ctx, distributionModule.GetAddress(), ustcstakingtypes.BondDenom).IsZero())
	require.Equal(t, initialSupply, suite.App.BankKeeper.GetSupply(suite.Ctx, ustcstakingtypes.BondDenom))
	require.NoError(t, suite.App.UstcStakingKeeper.ValidateState(suite.Ctx))
}

func TestUSTCStakingRejectsFundingAboveCommunityPool(t *testing.T) {
	var suite helperstest.KeeperTestHelper
	suite.SetT(t)
	suite.Setup(t, "ustc-staking-insufficient-community-pool")

	owner := suite.TestAccs[0]
	suite.FundAcc(owner, types.NewCoins(types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(20))))
	params := ustcstakingtypes.DefaultParams()
	params.LockTiers = []ustcstakingtypes.LockTier{{
		Id: 1, Duration: durationPtr(time.Hour), Multiplier: sdkmath.LegacyOneDec(),
	}}
	suite.App.UstcStakingKeeper.SetParams(suite.Ctx, params)
	distributionModule := suite.App.AccountKeeper.GetModuleAccount(suite.Ctx, distrtypes.ModuleName)
	require.NotNil(t, distributionModule)
	require.NoError(t, banktestutil.FundModuleAccount(suite.Ctx, suite.App.BankKeeper, distrtypes.ModuleName,
		types.NewCoins(types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(10)))))
	feePool, err := suite.App.DistrKeeper.FeePool.Get(suite.Ctx)
	require.NoError(t, err)
	feePool.CommunityPool = types.NewDecCoinsFromCoins(types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(5)))
	require.NoError(t, suite.App.DistrKeeper.FeePool.Set(suite.Ctx, feePool))
	_, err = ustcstakingkeeper.NewMsgServerImpl(suite.App.UstcStakingKeeper).Stake(suite.Ctx, &ustcstakingtypes.MsgStake{
		Owner: owner.String(), Amount: types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(20)), LockTierId: 1,
	})
	require.NoError(t, err)

	_, err = ustcstakingkeeper.NewMsgServerImpl(suite.App.UstcStakingKeeper).FundRewards(suite.Ctx, &ustcstakingtypes.MsgFundRewards{
		Authority: params.Authority, Amount: types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(10)),
	})

	require.ErrorIs(t, err, distrtypes.ErrBadDistribution)
	feePool, err = suite.App.DistrKeeper.FeePool.Get(suite.Ctx)
	require.NoError(t, err)
	require.Equal(t, sdkmath.NewInt(5), feePool.CommunityPool.AmountOf(ustcstakingtypes.BondDenom).TruncateInt())
	require.Equal(t, sdkmath.NewInt(10), suite.App.BankKeeper.GetBalance(suite.Ctx, distributionModule.GetAddress(), ustcstakingtypes.BondDenom).Amount)
	rewardModule := suite.App.AccountKeeper.GetModuleAccount(suite.Ctx, ustcstakingtypes.RewardPoolName)
	require.True(t, suite.App.BankKeeper.GetBalance(suite.Ctx, rewardModule.GetAddress(), ustcstakingtypes.BondDenom).IsZero())
	require.True(t, suite.App.UstcStakingKeeper.GetRewardState(suite.Ctx).RewardIndex.IsZero())
}

func durationPtr(value time.Duration) *time.Duration {
	return &value
}
