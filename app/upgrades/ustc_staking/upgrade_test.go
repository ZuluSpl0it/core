package ustc_staking

import (
	"testing"

	ustcstakingtypes "github.com/classic-terra/core/v4/x/ustcstaking/types"
	"github.com/stretchr/testify/require"
)

func TestUpgradeAddsOnlyUSTCStakingStore(t *testing.T) {
	require.Equal(t, "ustc_staking", UpgradeName)
	require.Equal(t, []string{ustcstakingtypes.StoreKey}, Upgrade.StoreUpgrades.Added)
	require.Empty(t, Upgrade.StoreUpgrades.Deleted)
	require.Empty(t, Upgrade.StoreUpgrades.Renamed)
}
