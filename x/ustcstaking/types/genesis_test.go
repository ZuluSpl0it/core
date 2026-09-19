package types

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestGenesisValidateRejectsNegativeRewardState(t *testing.T) {
	genesis := DefaultGenesisState()
	genesis.RewardState.RewardIndex = math.LegacyOneDec().Neg()

	require.ErrorIs(t, ValidateGenesis(genesis), ErrInvalidRewardState)
}

func TestGenesisValidateRejectsActiveSharesMismatch(t *testing.T) {
	genesis := DefaultGenesisState()
	genesis.RewardState.TotalShares = math.NewInt(10)
	genesis.Positions = []Position{
		{
			Id:               1,
			Owner:            genesis.Params.Authority,
			Principal:        uusdCoin(10),
			Shares:           math.NewInt(9),
			ShareMultiplier:  math.LegacyOneDec(),
			RewardDebt:       math.LegacyZeroDec(),
			LockDuration:     durationPtr(time.Hour),
			Status:           PositionStatus_POSITION_STATUS_ACTIVE,
			ClaimableRewards: uusdCoin(0),
		},
	}

	require.ErrorIs(t, ValidateGenesis(genesis), ErrInvalidRewardState)
}

func TestGenesisValidateAcceptsWithdrawnPositionWithZeroPrincipal(t *testing.T) {
	genesis := DefaultGenesisState()
	duration := time.Hour
	genesis.Positions = []Position{
		{
			Id:               1,
			Owner:            genesis.Params.Authority,
			Principal:        uusdCoin(0),
			Shares:           math.ZeroInt(),
			ShareMultiplier:  math.LegacyOneDec(),
			LockDuration:     &duration,
			Status:           PositionStatus_POSITION_STATUS_WITHDRAWN,
			ClaimableRewards: uusdCoin(0),
		},
	}
	genesis.NextPositionId = 2

	require.NoError(t, ValidateGenesis(genesis))
}

func TestGenesisValidateRejectsInvalidPrincipalForPositionStatus(t *testing.T) {
	maturity := time.Now().UTC()
	tests := []struct {
		name      string
		principal sdk.Coin
		status    PositionStatus
		shares    math.Int
		maturity  *time.Time
	}{
		{name: "withdrawn positive", principal: uusdCoin(1), status: PositionStatus_POSITION_STATUS_WITHDRAWN, shares: math.ZeroInt()},
		{name: "withdrawn negative", principal: sdk.Coin{Denom: BondDenom, Amount: math.NewInt(-1)}, status: PositionStatus_POSITION_STATUS_WITHDRAWN, shares: math.ZeroInt()},
		{name: "withdrawn wrong denom", principal: sdk.NewCoin("uluna", math.ZeroInt()), status: PositionStatus_POSITION_STATUS_WITHDRAWN, shares: math.ZeroInt()},
		{name: "withdrawn nil amount", principal: sdk.Coin{Denom: BondDenom}, status: PositionStatus_POSITION_STATUS_WITHDRAWN, shares: math.ZeroInt()},
		{name: "active zero", principal: uusdCoin(0), status: PositionStatus_POSITION_STATUS_ACTIVE, shares: math.OneInt()},
		{name: "unbonding zero", principal: uusdCoin(0), status: PositionStatus_POSITION_STATUS_UNBONDING, shares: math.ZeroInt(), maturity: &maturity},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			genesis := DefaultGenesisState()
			genesis.Positions = []Position{
				{
					Id:               1,
					Owner:            genesis.Params.Authority,
					Principal:        tt.principal,
					Shares:           tt.shares,
					ShareMultiplier:  math.LegacyOneDec(),
					LockDuration:     durationPtr(time.Hour),
					UnbondingEndTime: tt.maturity,
					Status:           tt.status,
					ClaimableRewards: uusdCoin(0),
				},
			}
			genesis.NextPositionId = 2
			if tt.status == PositionStatus_POSITION_STATUS_ACTIVE {
				genesis.RewardState.TotalShares = tt.shares
			}

			require.ErrorIs(t, ValidateGenesis(genesis), ErrInvalidPosition)
		})
	}
}

func TestGenesisValidateRejectsNilOrZeroLockSnapshot(t *testing.T) {
	genesis := validGenesisPositionState()
	genesis.Positions[0].LockDuration = nil
	require.ErrorIs(t, ValidateGenesis(genesis), ErrInvalidLockSnapshot)

	genesis = validGenesisPositionState()
	genesis.Positions[0].ShareMultiplier = math.LegacyZeroDec()
	require.ErrorIs(t, ValidateGenesis(genesis), ErrInvalidLockSnapshot)
}

func TestGenesisValidateRejectsActiveMaturityTimestamp(t *testing.T) {
	genesis := validGenesisPositionState()
	maturity := time.Now().UTC().Add(time.Hour)
	genesis.Positions[0].UnbondingEndTime = &maturity

	require.ErrorIs(t, ValidateGenesis(genesis), ErrInvalidPosition)
}

func TestGenesisValidateRejectsWrongClaimableDenom(t *testing.T) {
	genesis := validGenesisPositionState()
	genesis.Positions[0].ClaimableRewards = sdk.NewCoin("uluna", math.ZeroInt())

	require.ErrorIs(t, ValidateGenesis(genesis), ErrInvalidPosition)
}

func TestGenesisValidateRejectsMaxPositionID(t *testing.T) {
	genesis := validGenesisPositionState()
	genesis.Positions[0].Id = ^uint64(0)
	genesis.NextPositionId = ^uint64(0)

	require.ErrorIs(t, ValidateGenesis(genesis), ErrInvalidGenesis)
}

func validGenesisPositionState() *GenesisState {
	duration := time.Hour
	genesis := DefaultGenesisState()
	genesis.Positions = []Position{{
		Id:               1,
		Owner:            genesis.Params.Authority,
		Principal:        uusdCoin(10),
		Shares:           math.NewInt(10),
		ShareMultiplier:  math.LegacyOneDec(),
		RewardDebt:       math.LegacyZeroDec(),
		LockDuration:     &duration,
		Status:           PositionStatus_POSITION_STATUS_ACTIVE,
		ClaimableRewards: uusdCoin(0),
	}}
	genesis.RewardState.TotalShares = math.NewInt(10)
	genesis.NextPositionId = 2
	return genesis
}

func uusdCoin(amount int64) sdk.Coin {
	return sdk.NewCoin(BondDenom, math.NewInt(amount))
}
