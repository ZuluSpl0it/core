package ustcstaking

import (
	"strings"
	"testing"

	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	"github.com/cosmos/cosmos-sdk/types/kv"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/stretchr/testify/require"
)

func TestRegisterStoreDecoderUsesUSTCDecoder(t *testing.T) {
	registry := simtypes.StoreDecoderRegistry{}
	module := AppModule{}

	module.RegisterStoreDecoder(registry)
	output := registry[types.StoreKey](kv.Pair{Key: []byte{0x99}, Value: []byte{1}}, kv.Pair{Key: []byte{0x99}, Value: []byte{1}})

	require.True(t, strings.Contains(output, "unknown ustcstaking key prefix"))
}
