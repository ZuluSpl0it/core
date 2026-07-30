package types

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
