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
			Id:        1,
			Owner:     genesis.Params.Authority,
			Principal: uusdCoin(10),
			Shares:    math.NewInt(9),
			Status:    PositionStatus_POSITION_STATUS_ACTIVE,
		},
	}

	require.ErrorIs(t, ValidateGenesis(genesis), ErrInvalidRewardState)
}

func TestGenesisValidateAcceptsWithdrawnPositionWithZeroPrincipal(t *testing.T) {
	genesis := DefaultGenesisState()
	genesis.Positions = []Position{
		{
			Id:               1,
			Owner:            genesis.Params.Authority,
			Principal:        uusdCoin(0),
			Shares:           math.ZeroInt(),
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

func uusdCoin(amount int64) sdk.Coin {
	return sdk.NewCoin(BondDenom, math.NewInt(amount))
}
