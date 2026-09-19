package types

import "cosmossdk.io/errors"

var (
	ErrInvalidDenom         = errors.Register(ModuleName, 1, "invalid denomination")
	ErrInvalidLockTier      = errors.Register(ModuleName, 2, "invalid lock tier")
	ErrDuplicateLockTier    = errors.Register(ModuleName, 3, "duplicate lock tier")
	ErrInvalidAuthority     = errors.Register(ModuleName, 4, "invalid authority")
	ErrInvalidRewardState   = errors.Register(ModuleName, 5, "invalid reward state")
	ErrInvalidPosition      = errors.Register(ModuleName, 6, "invalid position")
	ErrInvalidGenesis       = errors.Register(ModuleName, 7, "invalid genesis")
	ErrNoActiveShares       = errors.Register(ModuleName, 8, "no active shares")
	ErrRewardPoolInsolvent  = errors.Register(ModuleName, 9, "reward pool is insolvent")
	ErrPositionNotFound     = errors.Register(ModuleName, 10, "position not found")
	ErrModulePaused         = errors.Register(ModuleName, 11, "module is paused")
	ErrUnauthorized         = errors.Register(ModuleName, 12, "unauthorized")
	ErrPositionOwner        = errors.Register(ModuleName, 13, "position owner mismatch")
	ErrInvalidPositionState = errors.Register(ModuleName, 14, "invalid position state")
	ErrNotMatured           = errors.Register(ModuleName, 15, "position has not matured")
	ErrPositionIDExhausted  = errors.Register(ModuleName, 16, "position ID exhausted")
	ErrInvalidLockSnapshot  = errors.Register(ModuleName, 17, "invalid lock snapshot")
	ErrCorruptOwnerIndex    = errors.Register(ModuleName, 18, "corrupt owner index")
)
