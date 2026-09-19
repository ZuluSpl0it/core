package simulation

import (
	"strings"
	"testing"

	"cosmossdk.io/math"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/kv"
	"github.com/stretchr/testify/require"
)

func TestDecodeUSTCStoreRecords(t *testing.T) {
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	owner := sdk.AccAddress(make([]byte, 20))
	positionKey := append(append([]byte{}, types.PositionKeyPrefix...), sdk.Uint64ToBigEndian(7)...)
	ownerKey := append(append([]byte{}, types.OwnerPositionKeyPrefix...), types.OwnerPositionKey(owner, 7)...)
	dec := NewDecodeStore(cdc)

	tests := []struct {
		name  string
		pair  kv.Pair
		match string
	}{
		{"position", kv.Pair{Key: positionKey, Value: cdc.MustMarshal(&types.Position{Id: 7, Shares: math.NewInt(3)})}, "Position"},
		{"owner index", kv.Pair{Key: ownerKey, Value: []byte{1}}, "owner-index"},
		{"reward state", kv.Pair{Key: types.RewardStateKey, Value: cdc.MustMarshal(&types.RewardState{RewardIndex: math.LegacyOneDec(), TotalShares: math.NewInt(3)})}, "RewardState"},
		{"params", kv.Pair{Key: types.ParamsKey, Value: cdc.MustMarshal(&types.Params{BondDenom: types.BondDenom})}, "Params"},
		{"next id", kv.Pair{Key: types.NextPositionIDKey, Value: sdk.Uint64ToBigEndian(8)}, "next-position-id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := dec(tt.pair, tt.pair)
			require.Contains(t, output, tt.match)
		})
	}
}

func TestDecodeUSTCStoreUnknownKeyIsDeterministic(t *testing.T) {
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	dec := NewDecodeStore(cdc)
	pair := kv.Pair{Key: []byte{0x99}, Value: []byte{0x01}}

	output := dec(pair, pair)

	require.True(t, strings.Contains(output, "unknown ustcstaking key prefix"))
	require.Equal(t, output, dec(pair, pair))
}
