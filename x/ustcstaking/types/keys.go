package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	ModuleName   = "ustcstaking"
	StoreKey     = "x_" + ModuleName
	RouterKey    = ModuleName
	QuerierRoute = ModuleName

	BondDenom = "uusd"

	PrincipalPoolName = ModuleName
	RewardPoolName    = ModuleName + "_reward_pool"
)

var (
	PositionKeyPrefix      = []byte{0x10}
	OwnerPositionKeyPrefix = []byte{0x11}
	UnbondingKeyPrefix     = []byte{0x12}
	NextPositionIDKey      = []byte{0x13}
	RewardStateKey         = []byte{0x20}
	ParamsKey              = []byte{0x30}
)

const accAddressLength = 20

func OwnerPositionKey(owner sdk.AccAddress, id uint64) []byte {
	key := make([]byte, 0, len(owner.Bytes())+8)
	key = append(key, owner.Bytes()...)
	key = append(key, sdk.Uint64ToBigEndian(id)...)
	return key
}

func OwnerFromPositionKey(key []byte) (sdk.AccAddress, uint64, error) {
	if len(key) != accAddressLength+8 {
		return nil, 0, fmt.Errorf("invalid owner position key length %d", len(key))
	}
	return sdk.AccAddress(key[:accAddressLength]), sdk.BigEndianToUint64(key[accAddressLength:]), nil
}
