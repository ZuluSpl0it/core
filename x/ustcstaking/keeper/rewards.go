package keeper

import (
	"cosmossdk.io/math"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) GetRewardState(ctx sdk.Context) types.RewardState {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.RewardStateKey)
	if bz == nil {
		return types.RewardState{RewardIndex: math.LegacyZeroDec(), TotalShares: math.ZeroInt()}
	}
	var state types.RewardState
	k.cdc.MustUnmarshal(bz, &state)
	return state
}

func (k Keeper) SetRewardState(ctx sdk.Context, state types.RewardState) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.RewardStateKey, k.cdc.MustMarshal(&state))
}

func (k Keeper) AccruedRewards(position types.Position, state types.RewardState) math.Int {
	return keeperAccruedRewards(position, state)
}

func keeperAccruedRewards(position types.Position, state types.RewardState) math.Int {
	if position.Status != types.PositionStatus_POSITION_STATUS_ACTIVE || position.Shares.IsNil() || !position.Shares.IsPositive() {
		if position.ClaimableRewards.Amount.IsNil() {
			return math.ZeroInt()
		}
		return position.ClaimableRewards.Amount
	}

	index := state.RewardIndex
	if index.IsNil() {
		index = math.LegacyZeroDec()
	}
	debt := position.RewardDebt
	if debt.IsNil() {
		debt = math.LegacyZeroDec()
	}
	raw := position.Shares.ToLegacyDec().Mul(index).Sub(debt)
	if raw.IsNegative() {
		return math.ZeroInt()
	}
	return raw.TruncateInt()
}

func (k Keeper) FundRewards(ctx sdk.Context, amount sdk.Coin) error {
	if amount.Denom != types.BondDenom || amount.Amount.IsNil() || !amount.Amount.IsPositive() {
		return types.ErrInvalidDenom.Wrapf("expected positive %s funding amount", types.BondDenom)
	}

	state := k.GetRewardState(ctx)
	if state.TotalShares.IsNil() || !state.TotalShares.IsPositive() {
		return types.ErrNoActiveShares
	}
	if err := k.communityPoolKeeper.DistributeFromCommunityPoolToModule(ctx, sdk.NewCoins(amount), types.RewardPoolName); err != nil {
		return err
	}

	index := state.RewardIndex
	if index.IsNil() {
		index = math.LegacyZeroDec()
	}
	state.RewardIndex = index.Add(amount.Amount.ToLegacyDec().QuoInt(state.TotalShares))
	k.SetRewardState(ctx, state)
	return nil
}
