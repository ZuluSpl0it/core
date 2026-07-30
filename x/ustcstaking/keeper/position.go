package keeper

import (
	"cosmossdk.io/store/prefix"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) GetPosition(ctx sdk.Context, id uint64) (types.Position, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.PositionKeyPrefix)
	bz := store.Get(sdk.Uint64ToBigEndian(id))
	if bz == nil {
		return types.Position{}, false
	}
	var position types.Position
	k.cdc.MustUnmarshal(bz, &position)
	return position, true
}

func (k Keeper) SetPosition(ctx sdk.Context, position types.Position) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.PositionKeyPrefix)
	store.Set(sdk.Uint64ToBigEndian(position.Id), k.cdc.MustMarshal(&position))
}

func (k Keeper) DeletePosition(ctx sdk.Context, id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.PositionKeyPrefix)
	store.Delete(sdk.Uint64ToBigEndian(id))
}

func (k Keeper) IteratePositions(ctx sdk.Context, callback func(types.Position) bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.PositionKeyPrefix)
	iterator := store.Iterator(nil, nil)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var position types.Position
		k.cdc.MustUnmarshal(iterator.Value(), &position)
		if callback(position) {
			return
		}
	}
}
