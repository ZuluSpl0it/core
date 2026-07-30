package types

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	_ sdk.Msg = &MsgStake{}
	_ sdk.Msg = &MsgBeginUnbonding{}
	_ sdk.Msg = &MsgWithdraw{}
	_ sdk.Msg = &MsgClaimRewards{}
	_ sdk.Msg = &MsgFundRewards{}
	_ sdk.Msg = &MsgUpdateParams{}
	_ sdk.Msg = &MsgUpdateFundingAuthority{}
)

func (MsgStake) Route() string                  { return RouterKey }
func (MsgBeginUnbonding) Route() string         { return RouterKey }
func (MsgWithdraw) Route() string               { return RouterKey }
func (MsgClaimRewards) Route() string           { return RouterKey }
func (MsgFundRewards) Route() string            { return RouterKey }
func (MsgUpdateParams) Route() string           { return RouterKey }
func (MsgUpdateFundingAuthority) Route() string { return RouterKey }

func (MsgStake) Type() string                  { return "Stake" }
func (MsgBeginUnbonding) Type() string         { return "BeginUnbonding" }
func (MsgWithdraw) Type() string               { return "Withdraw" }
func (MsgClaimRewards) Type() string           { return "ClaimRewards" }
func (MsgFundRewards) Type() string            { return "FundRewards" }
func (MsgUpdateParams) Type() string           { return "UpdateParams" }
func (MsgUpdateFundingAuthority) Type() string { return "UpdateFundingAuthority" }

func (msg MsgStake) GetSigners() []sdk.AccAddress                  { return signer(msg.Owner) }
func (msg MsgBeginUnbonding) GetSigners() []sdk.AccAddress         { return signer(msg.Owner) }
func (msg MsgWithdraw) GetSigners() []sdk.AccAddress               { return signer(msg.Owner) }
func (msg MsgClaimRewards) GetSigners() []sdk.AccAddress           { return signer(msg.Owner) }
func (msg MsgFundRewards) GetSigners() []sdk.AccAddress            { return signer(msg.Sender) }
func (msg MsgUpdateParams) GetSigners() []sdk.AccAddress           { return signer(msg.Authority) }
func (msg MsgUpdateFundingAuthority) GetSigners() []sdk.AccAddress { return signer(msg.Authority) }

func (msg MsgStake) GetSignBytes() []byte                  { return signBytes(msg) }
func (msg MsgBeginUnbonding) GetSignBytes() []byte         { return signBytes(msg) }
func (msg MsgWithdraw) GetSignBytes() []byte               { return signBytes(msg) }
func (msg MsgClaimRewards) GetSignBytes() []byte           { return signBytes(msg) }
func (msg MsgFundRewards) GetSignBytes() []byte            { return signBytes(msg) }
func (msg MsgUpdateParams) GetSignBytes() []byte           { return signBytes(msg) }
func (msg MsgUpdateFundingAuthority) GetSignBytes() []byte { return signBytes(msg) }

func (msg MsgStake) ValidateBasic() error {
	if err := validateSigner(msg.Owner); err != nil {
		return err
	}
	if msg.LockTierId == 0 {
		return sdkerrors.ErrInvalidRequest.Wrap("lock tier id must be positive")
	}
	return validateAmount(msg.Amount)
}

func (msg MsgBeginUnbonding) ValidateBasic() error {
	return validatePositionRequest(msg.Owner, msg.PositionId)
}
func (msg MsgWithdraw) ValidateBasic() error {
	return validatePositionRequest(msg.Owner, msg.PositionId)
}
func (msg MsgClaimRewards) ValidateBasic() error {
	return validatePositionRequest(msg.Owner, msg.PositionId)
}

func (msg MsgFundRewards) ValidateBasic() error {
	if err := validateSigner(msg.Sender); err != nil {
		return err
	}
	return validateAmount(msg.Amount)
}

func (msg MsgUpdateParams) ValidateBasic() error {
	if err := validateSigner(msg.Authority); err != nil {
		return err
	}
	return msg.Params.Validate()
}

func (msg MsgUpdateFundingAuthority) ValidateBasic() error {
	if err := validateSigner(msg.Authority); err != nil {
		return err
	}
	return validateSigner(msg.FundingAuthority)
}

func signer(value string) []sdk.AccAddress {
	address, err := sdk.AccAddressFromBech32(value)
	if err != nil {
		return []sdk.AccAddress{}
	}
	return []sdk.AccAddress{address}
}

func validateSigner(value string) error {
	if _, err := sdk.AccAddressFromBech32(value); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid signer address: %s", err)
	}
	return nil
}

func validatePositionRequest(owner string, positionID uint64) error {
	if err := validateSigner(owner); err != nil {
		return err
	}
	if positionID == 0 {
		return sdkerrors.ErrInvalidRequest.Wrap("position id must be positive")
	}
	return nil
}

func validateAmount(amount sdk.Coin) error {
	if amount.Denom != BondDenom {
		return ErrInvalidDenom.Wrapf("expected %s, got %s", BondDenom, amount.Denom)
	}
	if !amount.IsPositive() {
		return sdkerrors.ErrInvalidCoins.Wrap("amount must be positive")
	}
	return nil
}

func signBytes(message interface{}) []byte {
	bz, err := json.Marshal(message)
	if err != nil {
		panic(fmt.Errorf("failed to marshal message: %w", err))
	}
	return sdk.MustSortJSON(bz)
}
