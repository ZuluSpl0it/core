package cli

import (
	"context"

	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
)

func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the USTC staking module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(positionQueryCmd(), positionsByOwnerQueryCmd(), rewardStateQueryCmd(), paramsQueryCmd())
	return cmd
}

func positionQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "position [position-id]",
		Args:  cobra.ExactArgs(1),
		Short: "Query a USTC staking position",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			id, err := parseUint64Argument(args[0], "position id")
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).Position(context.Background(), &types.QueryPositionRequest{PositionId: id})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func positionsByOwnerQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "positions [owner]",
		Aliases: []string{"positions-by-owner"},
		Args:    cobra.ExactArgs(1),
		Short:   "Query USTC staking positions owned by an address",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			if _, err := sdk.AccAddressFromBech32(args[0]); err != nil {
				return err
			}
			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).PositionsByOwner(context.Background(), &types.QueryPositionsByOwnerRequest{Owner: args[0], Pagination: pageReq})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func rewardStateQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reward-state",
		Args:  cobra.NoArgs,
		Short: "Query USTC staking reward state",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).RewardState(context.Background(), &types.QueryRewardStateRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func paramsQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Args:  cobra.NoArgs,
		Short: "Query USTC staking parameters",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).Params(context.Background(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
