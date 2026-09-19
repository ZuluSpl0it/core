package keeper

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/log"
	store "cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
)

type recordingBankKeeper struct {
	accountToModuleCalls      int
	moduleToAccountCalls      int
	lastRecipientModule       string
	lastAccountToModuleAmount sdk.Coins
	rewardPoolBalance         sdk.Coin
	principalPoolBalance      sdk.Coin
}

func (b *recordingBankKeeper) SendCoinsFromAccountToModule(_ context.Context, _ sdk.AccAddress, module string, amount sdk.Coins) error {
	b.accountToModuleCalls++
	b.lastRecipientModule = module
	b.lastAccountToModuleAmount = amount
	return nil
}

func (b *recordingBankKeeper) SendCoinsFromModuleToAccount(context.Context, string, sdk.AccAddress, sdk.Coins) error {
	b.moduleToAccountCalls++
	return nil
}

func (b *recordingBankKeeper) GetBalance(context.Context, sdk.AccAddress, string) sdk.Coin {
	return b.rewardPoolBalance
}

func (b *recordingBankKeeper) GetAllBalances(_ context.Context, addr sdk.AccAddress) sdk.Coins {
	if addr.Equals(authtypes.NewModuleAddress(types.PrincipalPoolName)) {
		if b.principalPoolBalance.IsNil() {
			return sdk.NewCoins()
		}
		return sdk.NewCoins(b.principalPoolBalance)
	}
	if b.rewardPoolBalance.IsNil() {
		return sdk.NewCoins()
	}
	return sdk.NewCoins(b.rewardPoolBalance)
}

func newKeeperTest(t *testing.T) (sdk.Context, Keeper) {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())
	ctx := sdk.NewContext(ms, tmproto.Header{Time: time.Now().UTC()}, false, log.NewNopLogger())
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	return ctx, NewKeeper(cdc, key, &recordingBankKeeper{})
}
