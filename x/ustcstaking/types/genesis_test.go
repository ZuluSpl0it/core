package types

import (
	"testing"

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

func uusdCoin(amount int64) sdk.Coin {
	return sdk.NewCoin(BondDenom, math.NewInt(amount))
}
