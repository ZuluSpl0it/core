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

func (k Keeper) SetPosition(ctx sdk.Context, position types.Position) error {
	owner, err := sdk.AccAddressFromBech32(position.Owner)
	if err != nil {
		return types.ErrInvalidPosition.Wrapf("position %d owner: %s", position.Id, err)
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.PositionKeyPrefix)
	if existing, found := k.GetPosition(ctx, position.Id); found && existing.Owner != position.Owner {
		return types.ErrPositionOwner
	}
	store.Set(sdk.Uint64ToBigEndian(position.Id), k.cdc.MustMarshal(&position))
	ownerIndex := prefix.NewStore(ctx.KVStore(k.storeKey), types.OwnerPositionKeyPrefix)
	prefix.NewStore(ownerIndex, owner.Bytes()).Set(sdk.Uint64ToBigEndian(position.Id), []byte{1})
	return nil
}

func (k Keeper) DeletePosition(ctx sdk.Context, id uint64) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.PositionKeyPrefix)
	position, found := k.GetPosition(ctx, id)
	if !found {
		return nil
	}
	owner, err := sdk.AccAddressFromBech32(position.Owner)
	if err != nil {
		return types.ErrInvalidPosition.Wrapf("position %d owner: %s", id, err)
	}
	store.Delete(sdk.Uint64ToBigEndian(id))
	ownerIndex := prefix.NewStore(ctx.KVStore(k.storeKey), types.OwnerPositionKeyPrefix)
	prefix.NewStore(ownerIndex, owner.Bytes()).Delete(sdk.Uint64ToBigEndian(id))
	return nil
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
