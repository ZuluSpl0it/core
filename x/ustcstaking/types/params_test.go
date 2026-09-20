package types

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"
)

func TestParamsValidateRejectsNonUSTCDenom(t *testing.T) {
	params := DefaultParams()
	params.BondDenom = "uluna"

	require.ErrorIs(t, params.Validate(), ErrInvalidDenom)
}

func TestDefaultParamsUseOnlyGovernanceAuthority(t *testing.T) {
	params := DefaultParams()
	require.Equal(t, types.NewModuleAddress(govtypes.ModuleName).String(), params.Authority)
	require.NoError(t, params.Validate())
}

func TestParamsValidateRejectsDuplicateTierIDs(t *testing.T) {
	params := DefaultParams()
	params.LockTiers = []LockTier{
		{Id: 1, Duration: durationPtr(time.Hour), Multiplier: math.LegacyOneDec()},
		{Id: 1, Duration: durationPtr(2 * time.Hour), Multiplier: math.LegacyOneDec()},
	}

	require.ErrorIs(t, params.Validate(), ErrDuplicateLockTier)
}

func TestParamsValidateRejectsNonPositiveMultipliers(t *testing.T) {
	params := DefaultParams()
	params.LockTiers = []LockTier{
		{Id: 1, Duration: durationPtr(time.Hour), Multiplier: math.LegacyZeroDec()},
	}

	require.ErrorIs(t, params.Validate(), ErrInvalidLockTier)
}

func TestParamsValidateRejectsZeroLockTierID(t *testing.T) {
	params := DefaultParams()
	params.LockTiers = []LockTier{{Id: 0, Duration: durationPtr(time.Hour), Multiplier: math.LegacyOneDec()}}

	require.ErrorIs(t, params.Validate(), ErrInvalidLockTier)
}

func durationPtr(value time.Duration) *time.Duration {
	return &value
}
