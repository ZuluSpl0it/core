package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewGenesisState(params Params, positions []Position, rewardState RewardState, nextPositionID uint64) *GenesisState {
	return &GenesisState{
		Params:         params,
		Positions:      positions,
		RewardState:    rewardState,
		NextPositionId: nextPositionID,
	}
}

func DefaultGenesisState() *GenesisState {
	return NewGenesisState(
		DefaultParams(),
		nil,
		RewardState{RewardIndex: math.LegacyZeroDec(), TotalShares: math.ZeroInt()},
		1,
	)
}

func ValidateGenesis(genesis *GenesisState) error {
	if genesis == nil {
		return ErrInvalidGenesis.Wrap("genesis is nil")
	}
	if err := genesis.Params.Validate(); err != nil {
		return ErrInvalidGenesis.Wrapf("params: %s", err)
	}
	if genesis.RewardState.RewardIndex.IsNil() || genesis.RewardState.RewardIndex.IsNegative() ||
		genesis.RewardState.TotalShares.IsNil() || genesis.RewardState.TotalShares.IsNegative() {
		return ErrInvalidRewardState.Wrap("reward index and total shares must be initialized and non-negative")
	}

	activeShares := math.ZeroInt()
	seen := make(map[uint64]struct{}, len(genesis.Positions))
	maxID := uint64(0)
	for _, position := range genesis.Positions {
		if position.Id == ^uint64(0) {
			return ErrInvalidGenesis.Wrap("maximum position id cannot be incremented")
		}
		if _, exists := seen[position.Id]; exists || position.Id == 0 {
			return ErrInvalidPosition.Wrapf("duplicate or zero position id %d", position.Id)
		}
		seen[position.Id] = struct{}{}
		if position.Id > maxID {
			maxID = position.Id
		}
		if _, err := sdk.AccAddressFromBech32(position.Owner); err != nil {
			return ErrInvalidPosition.Wrapf("position %d owner: %s", position.Id, err)
		}
		if err := validatePrincipal(position.Principal, position.Status); err != nil {
			return ErrInvalidPosition.Wrapf("position %d principal: %s", position.Id, err)
		}
		if err := validateClaimable(position.ClaimableRewards); err != nil {
			return ErrInvalidPosition.Wrapf("position %d claimable rewards: %s", position.Id, err)
		}
		if position.LockTierId == 0 || position.LockDuration == nil || *position.LockDuration <= 0 ||
			position.ShareMultiplier.IsNil() || !position.ShareMultiplier.IsPositive() {
			return ErrInvalidLockSnapshot.Wrapf("position %d has invalid lock snapshot", position.Id)
		}
		if position.Shares.IsNil() || position.RewardDebt.IsNil() || position.Shares.IsNegative() || position.RewardDebt.IsNegative() {
			return ErrInvalidPosition.Wrapf("position %d has nil or negative accounting state", position.Id)
		}
		switch position.Status {
		case PositionStatus_POSITION_STATUS_ACTIVE:
			if position.UnbondingEndTime != nil {
				return ErrInvalidPosition.Wrapf("active position %d has unbonding completion time", position.Id)
			}
			if position.Shares.IsNil() || !position.Shares.IsPositive() {
				return ErrInvalidPosition.Wrapf("active position %d must have shares", position.Id)
			}
			if position.Shares.ToLegacyDec().Mul(genesis.RewardState.RewardIndex).Sub(position.RewardDebt).IsNegative() {
				return ErrInvalidPosition.Wrapf("active position %d reward debt exceeds accrued rewards", position.Id)
			}
			activeShares = activeShares.Add(position.Shares)
		case PositionStatus_POSITION_STATUS_UNBONDING:
			if !position.Shares.IsZero() || position.UnbondingEndTime == nil || position.UnbondingEndTime.IsZero() ||
				!position.RewardDebt.IsZero() {
				return ErrInvalidPosition.Wrapf("unbonding position %d has invalid shares or maturity", position.Id)
			}
		case PositionStatus_POSITION_STATUS_WITHDRAWN:
			if !position.Shares.IsZero() || !position.Principal.Amount.IsZero() || !position.RewardDebt.IsZero() {
				return ErrInvalidPosition.Wrapf("withdrawn position %d retains principal or shares", position.Id)
			}
		default:
			return ErrInvalidPosition.Wrapf("position %d has unknown status", position.Id)
		}
	}
	if !activeShares.Equal(genesis.RewardState.TotalShares) {
		return ErrInvalidRewardState.Wrapf("total shares %s does not equal active shares %s", genesis.RewardState.TotalShares, activeShares)
	}
	if genesis.NextPositionId <= maxID {
		return ErrInvalidGenesis.Wrapf("next position id %d must exceed max position id %d", genesis.NextPositionId, maxID)
	}
	return nil
}

func validatePrincipal(coin sdk.Coin, status PositionStatus) error {
	if coin.Denom != BondDenom || coin.Amount.IsNil() {
		return ErrInvalidDenom.Wrapf("expected %s principal coin", BondDenom)
	}
	if status == PositionStatus_POSITION_STATUS_WITHDRAWN {
		if !coin.Amount.IsZero() {
			return ErrInvalidDenom.Wrapf("expected zero %s principal for withdrawn position", BondDenom)
		}
		return nil
	}
	if !coin.Amount.IsPositive() {
		return ErrInvalidDenom.Wrapf("expected positive %s coin", BondDenom)
	}
	return nil
}

func validateClaimable(coin sdk.Coin) error {
	if coin.Denom != BondDenom || coin.Amount.IsNil() || coin.Amount.IsNegative() {
		return ErrInvalidDenom.Wrapf("expected non-negative %s claimable rewards", BondDenom)
	}
	return nil
}
