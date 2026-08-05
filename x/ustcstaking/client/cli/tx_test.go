package cli

import (
	"testing"

	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

const testOwner = "terra1l28d80zfpncdjvn4jadx6at3rnj6jplhrkuayx"

func TestParseUint32ArgumentRejectsMalformedInput(t *testing.T) {
	for _, value := range []string{"7junk", "1 2", "-1", "4294967296"} {
		t.Run(value, func(t *testing.T) {
			_, err := parseUint32Argument(value, "lock tier id")
			require.Error(t, err)
		})
	}

	value, err := parseUint32Argument("7", "lock tier id")
	require.NoError(t, err)
	require.Equal(t, uint32(7), value)
}

func TestBuildStakeMessage(t *testing.T) {
	msg, err := buildStakeMessage(testOwner, "1000000uusd", "7")
	require.NoError(t, err)
	require.Equal(t, &types.MsgStake{
		Owner:      testOwner,
		Amount:     sdk.NewInt64Coin(types.BondDenom, 1_000_000),
		LockTierId: 7,
	}, msg)
}

func TestBuildPositionMessage(t *testing.T) {
	msg, err := buildPositionMessage(testOwner, "7", func(owner string, id uint64) sdk.Msg {
		return &types.MsgWithdraw{Owner: owner, PositionId: id}
	})
	require.NoError(t, err)
	require.Equal(t, &types.MsgWithdraw{Owner: testOwner, PositionId: 7}, msg)
}
