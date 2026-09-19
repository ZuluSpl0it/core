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
	_, principalPermissions := suite.App.AccountKeeper.GetModuleAccountAndPermissions(suite.Ctx, ustcstakingtypes.PrincipalPoolName)
	_, rewardPermissions := suite.App.AccountKeeper.GetModuleAccountAndPermissions(suite.Ctx, ustcstakingtypes.RewardPoolName)
	require.Empty(t, principalPermissions)
	require.Empty(t, rewardPermissions)

	params := ustcstakingtypes.DefaultParams()
	params.Authority = authtypes.NewModuleAddress(govtypes.ModuleName).String()
	params.FundingAuthority = owner.String()
	params.LockTiers = []ustcstakingtypes.LockTier{{
		Id:         1,
		Duration:   durationPtr(time.Hour),
		Multiplier: sdkmath.LegacyOneDec(),
	}}
	suite.App.UstcStakingKeeper.SetParams(suite.Ctx, params)

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
		&ustcstakingtypes.MsgFundRewards{Sender: owner.String(), Amount: types.NewCoin(ustcstakingtypes.BondDenom, sdkmath.NewInt(40))},
	)
	require.NoError(t, err)

	claimResponse, err := msgServer.ClaimRewards(
		suite.Ctx,
		&ustcstakingtypes.MsgClaimRewards{Owner: owner.String(), PositionId: stakeResponse.PositionId},
	)
	require.NoError(t, err)
	require.Equal(t, sdkmath.NewInt(40), claimResponse.Amount.Amount)

	require.Equal(t, sdkmath.NewInt(100), suite.App.BankKeeper.GetBalance(suite.Ctx, principalModule.GetAddress(), ustcstakingtypes.BondDenom).Amount)
	require.True(t, suite.App.BankKeeper.GetBalance(suite.Ctx, rewardModule.GetAddress(), ustcstakingtypes.BondDenom).IsZero())
	require.Equal(t, initialSupply, suite.App.BankKeeper.GetSupply(suite.Ctx, ustcstakingtypes.BondDenom))
	require.NoError(t, suite.App.UstcStakingKeeper.ValidateState(suite.Ctx))
}

func durationPtr(value time.Duration) *time.Duration {
	return &value
}
