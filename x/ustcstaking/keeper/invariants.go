package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func (k Keeper) ValidateState(ctx sdk.Context) error {
	for _, invariant := range []sdk.Invariant{
		principalInvariant(k),
		activeSharesInvariant(k),
		rewardSolvencyInvariant(k),
	} {
		message, broken := invariant(ctx)
		if broken {
			return types.ErrInvalidRewardState.Wrap(message)
		}
	}
	return nil
}

func RegisterInvariants(registry sdk.InvariantRegistry, keeper Keeper) {
	registry.RegisterRoute(types.ModuleName, "principal-custody", principalInvariant(keeper))
	registry.RegisterRoute(types.ModuleName, "active-shares", activeSharesInvariant(keeper))
	registry.RegisterRoute(types.ModuleName, "reward-solvency", rewardSolvencyInvariant(keeper))
}

func principalInvariant(keeper Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		expected := math.ZeroInt()
		var offending uint64
		keeper.IteratePositions(ctx, func(position types.Position) bool {
			if position.Status == types.PositionStatus_POSITION_STATUS_WITHDRAWN {
				if !position.Principal.Amount.IsNil() && position.Principal.Amount.IsPositive() {
					offending = position.Id
					return true
				}
				return false
			}
			if !position.Principal.Amount.IsNil() {
				expected = expected.Add(position.Principal.Amount)
			}
			return false
		})
		actual, valid := poolBalance(ctx, keeper, types.PrincipalPoolName)
		if offending != 0 || !valid || !actual.Equal(expected) {
			return fmt.Sprintf("principal custody: expected=%s actual=%s offending_position=%d", expected, actual, offending), true
		}
		return "", false
	}
}

func activeSharesInvariant(keeper Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		expected := math.ZeroInt()
		var offending uint64
		keeper.IteratePositions(ctx, func(position types.Position) bool {
			shares := position.Shares
			if shares.IsNil() {
				shares = math.ZeroInt()
			}
			if position.Status == types.PositionStatus_POSITION_STATUS_ACTIVE {
				expected = expected.Add(shares)
			} else if shares.IsPositive() {
				offending = position.Id
				return true
			}
			return false
		})
		actual := keeper.GetRewardState(ctx).TotalShares
		if actual.IsNil() {
			actual = math.ZeroInt()
		}
		if offending != 0 || actual.IsNegative() || !actual.Equal(expected) {
			return fmt.Sprintf("active shares: expected=%s actual=%s offending_position=%d", expected, actual, offending), true
		}
		return "", false
	}
}

func rewardSolvencyInvariant(keeper Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		state := keeper.GetRewardState(ctx)
		liability := math.ZeroInt()
		var offending uint64
		keeper.IteratePositions(ctx, func(position types.Position) bool {
			amount := keeper.AccruedRewards(position, state)
			if amount.IsNegative() {
				offending = position.Id
				return true
			}
			liability = liability.Add(amount)
			return false
		})
		actual, valid := poolBalance(ctx, keeper, types.RewardPoolName)
		if offending != 0 || !valid || actual.LT(liability) {
			return fmt.Sprintf("reward solvency: expected=%s actual=%s offending_position=%d", liability, actual, offending), true
		}
		return "", false
	}
}

func poolBalance(ctx sdk.Context, keeper Keeper, moduleName string) (math.Int, bool) {
	balances := keeper.bankKeeper.GetAllBalances(ctx, authtypes.NewModuleAddress(moduleName))
	amount := math.ZeroInt()
	valid := true
	for _, coin := range balances {
		if coin.Denom != types.BondDenom {
			valid = false
			continue
		}
		amount = amount.Add(coin.Amount)
	}
	return amount, valid
}
