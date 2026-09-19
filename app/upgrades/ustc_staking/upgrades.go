//nolint:revive
package ustc_staking

import (
	"context"
	"fmt"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/classic-terra/core/v4/app/keepers"
	"github.com/classic-terra/core/v4/app/upgrades"
	ustcstakingtypes "github.com/classic-terra/core/v4/x/ustcstaking/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

func CreateUSTCStakingUpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	_ upgrades.BaseAppParamManager,
	appKeepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		versionMap, err := mm.RunMigrations(ctx, cfg, fromVM)
		if err != nil {
			return nil, err
		}
		for _, moduleName := range []string{ustcstakingtypes.PrincipalPoolName, ustcstakingtypes.RewardPoolName} {
			_, permissions := appKeepers.AccountKeeper.GetModuleAccountAndPermissions(ctx, moduleName)
			if len(permissions) != 0 {
				return nil, fmt.Errorf("%s module account has unexpected permissions %v", moduleName, permissions)
			}
		}
		return versionMap, nil
	}
}
