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
	if (!genesis.RewardState.RewardIndex.IsNil() && genesis.RewardState.RewardIndex.IsNegative()) ||
		(!genesis.RewardState.TotalShares.IsNil() && genesis.RewardState.TotalShares.IsNegative()) {
		return ErrInvalidRewardState.Wrap("reward index and total shares must be non-negative")
	}

	activeShares := math.ZeroInt()
	seen := make(map[uint64]struct{}, len(genesis.Positions))
	maxID := uint64(0)
	for _, position := range genesis.Positions {
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
		if err := validateCoin(position.Principal); err != nil {
			return ErrInvalidPosition.Wrapf("position %d principal: %s", position.Id, err)
		}
		if err := validateClaimable(position.ClaimableRewards); err != nil {
			return ErrInvalidPosition.Wrapf("position %d claimable rewards: %s", position.Id, err)
		}
		if (!position.Shares.IsNil() && position.Shares.IsNegative()) ||
			(!position.RewardDebt.IsNil() && position.RewardDebt.IsNegative()) {
			return ErrInvalidPosition.Wrapf("position %d has negative accounting state", position.Id)
		}
		switch position.Status {
		case PositionStatus_POSITION_STATUS_ACTIVE:
			if position.Shares.IsNil() || !position.Shares.IsPositive() {
				return ErrInvalidPosition.Wrapf("active position %d must have shares", position.Id)
			}
			activeShares = activeShares.Add(position.Shares)
		case PositionStatus_POSITION_STATUS_UNBONDING:
			if (!position.Shares.IsNil() && position.Shares.IsPositive()) || position.UnbondingEndTime == nil {
				return ErrInvalidPosition.Wrapf("unbonding position %d has invalid shares or maturity", position.Id)
			}
		case PositionStatus_POSITION_STATUS_WITHDRAWN:
			if (!position.Shares.IsNil() && position.Shares.IsPositive()) ||
				(!position.Principal.Amount.IsNil() && !position.Principal.Amount.IsZero()) {
				return ErrInvalidPosition.Wrapf("withdrawn position %d retains principal or shares", position.Id)
			}
		default:
			return ErrInvalidPosition.Wrapf("position %d has unknown status", position.Id)
		}
	}
	totalShares := genesis.RewardState.TotalShares
	if totalShares.IsNil() {
		totalShares = math.ZeroInt()
	}
	if !activeShares.Equal(totalShares) {
		return ErrInvalidRewardState.Wrapf("total shares %s does not equal active shares %s", genesis.RewardState.TotalShares, activeShares)
	}
	if genesis.NextPositionId <= maxID {
		return ErrInvalidGenesis.Wrapf("next position id %d must exceed max position id %d", genesis.NextPositionId, maxID)
	}
	return nil
}

func validateCoin(coin sdk.Coin) error {
	if coin.Denom != BondDenom || coin.Amount.IsNil() || !coin.Amount.IsPositive() {
		return ErrInvalidDenom.Wrapf("expected positive %s coin", BondDenom)
	}
	return nil
}

func validateClaimable(coin sdk.Coin) error {
	if coin.Denom == "" && (coin.Amount.IsNil() || coin.Amount.IsZero()) {
		return nil
	}
	if coin.Denom != BondDenom || (!coin.Amount.IsNil() && coin.Amount.IsNegative()) {
		return ErrInvalidDenom.Wrapf("expected non-negative %s claimable rewards", BondDenom)
	}
	return nil
}
