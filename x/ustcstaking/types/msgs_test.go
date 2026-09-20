package types

import (
	"testing"

	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestMsgFundRewardsUsesGovernanceAuthoritySigner(t *testing.T) {
	authority := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	msg := MsgFundRewards{
		Authority: authority.String(),
		Amount:    sdk.NewInt64Coin(BondDenom, 42),
	}

	require.NoError(t, msg.ValidateBasic())
	require.Equal(t, []sdk.AccAddress{authority}, msg.GetSigners())
}
