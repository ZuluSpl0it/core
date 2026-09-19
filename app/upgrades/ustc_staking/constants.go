//nolint:revive
package ustc_staking

import (
	store "cosmossdk.io/store/types"
	"github.com/classic-terra/core/v4/app/upgrades"
	ustcstakingtypes "github.com/classic-terra/core/v4/x/ustcstaking/types"
)

const UpgradeName = "ustc_staking"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUSTCStakingUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{ustcstakingtypes.StoreKey},
	},
}
