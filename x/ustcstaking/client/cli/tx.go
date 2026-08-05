package cli

import (
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
)

func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "USTC staking transaction subcommands",
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(
		stakeCmd(),
		positionCmd("begin-unbonding", "Begin unbonding a USTC staking position", func(owner string, id uint64) sdk.Msg {
			return &types.MsgBeginUnbonding{Owner: owner, PositionId: id}
		}),
		positionCmd("withdraw", "Withdraw a matured USTC staking position", func(owner string, id uint64) sdk.Msg {
			return &types.MsgWithdraw{Owner: owner, PositionId: id}
		}),
		positionCmd("claim-rewards", "Claim rewards from a USTC staking position", func(owner string, id uint64) sdk.Msg {
			return &types.MsgClaimRewards{Owner: owner, PositionId: id}
		}),
	)
	return cmd
}

func stakeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stake [amount] [lock-tier-id]",
		Args:  cobra.ExactArgs(2),
		Short: "Lock USTC in a staking position",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg, err := buildStakeMessage(clientCtx.GetFromAddress().String(), args[0], args[1])
			if err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func positionCmd(use, short string, build func(string, uint64) sdk.Msg) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use + " [position-id]",
		Args:  cobra.ExactArgs(1),
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg, err := buildPositionMessage(clientCtx.GetFromAddress().String(), args[0], build)
			if err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func buildStakeMessage(owner, amountValue, tierValue string) (sdk.Msg, error) {
	amount, err := sdk.ParseCoinNormalized(amountValue)
	if err != nil {
		return nil, err
	}
	tier, err := parseUint32Argument(tierValue, "lock tier id")
	if err != nil {
		return nil, err
	}
	return &types.MsgStake{Owner: owner, Amount: amount, LockTierId: tier}, nil
}

func buildPositionMessage(owner, idValue string, build func(string, uint64) sdk.Msg) (sdk.Msg, error) {
	id, err := parseUint64Argument(idValue, "position id")
	if err != nil {
		return nil, err
	}
	return build(owner, id), nil
}
