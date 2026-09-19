package ustc_staking_test

import (
	"testing"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	apphelpers "github.com/classic-terra/core/v4/app/testing"
	ustcstakingupgrade "github.com/classic-terra/core/v4/app/upgrades/ustc_staking"
	ustcstaking "github.com/classic-terra/core/v4/x/ustcstaking"
	ustcstakingtypes "github.com/classic-terra/core/v4/x/ustcstaking/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/stretchr/testify/require"
)

func TestUpgradeAddsOnlyUSTCStakingStore(t *testing.T) {
	require.Equal(t, "ustc_staking", ustcstakingupgrade.UpgradeName)
	require.Equal(t, []string{ustcstakingtypes.StoreKey}, ustcstakingupgrade.Upgrade.StoreUpgrades.Added)
	require.Empty(t, ustcstakingupgrade.Upgrade.StoreUpgrades.Deleted)
	require.Empty(t, ustcstakingupgrade.Upgrade.StoreUpgrades.Renamed)
}

func TestUpgradeHandlerRunsWithRealAppKeepers(t *testing.T) {
	var suite apphelpers.KeeperTestHelper
	suite.SetT(t)
	suite.Setup(t, "ustc-staking-upgrade")

	manager := module.NewManager(ustcstaking.NewAppModule(suite.App.AppCodec(), suite.App.UstcStakingKeeper))
	configurator := module.NewConfigurator(suite.App.AppCodec(), suite.App.MsgServiceRouter(), suite.App.GRPCQueryRouter())
	handler := ustcstakingupgrade.CreateUSTCStakingUpgradeHandler(manager, configurator, nil, suite.App.AppKeepers)

	versionMap, err := handler(suite.Ctx, upgradetypes.Plan{Name: ustcstakingupgrade.UpgradeName}, module.VersionMap{})
	require.NoError(t, err)
	require.Equal(t, uint64(1), versionMap[ustcstakingtypes.ModuleName])

	for _, moduleName := range []string{ustcstakingtypes.PrincipalPoolName, ustcstakingtypes.RewardPoolName} {
		account := suite.App.AccountKeeper.GetModuleAccount(suite.Ctx, moduleName)
		require.NotNil(t, account)
		_, permissions := suite.App.AccountKeeper.GetModuleAccountAndPermissions(suite.Ctx, moduleName)
		require.Empty(t, permissions)
	}
}
