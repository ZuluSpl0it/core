package cli

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	"github.com/classic-terra/core/v4/x/ustcstaking/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/spf13/cobra"
)

func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the USTC staking module",
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(positionQueryCmd(), positionsByOwnerQueryCmd(), rewardStateQueryCmd(), paramsQueryCmd(), validateStateQueryCmd())
	return cmd
}

func validateStateQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate-state",
		Args:  cobra.NoArgs,
		Short: "Check USTC staking accounting invariants",
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			if clientCtx.Height <= 0 {
				status, err := clientCtx.Client.Status(cmd.Context())
				if err != nil {
					return fmt.Errorf("get validation snapshot height: %w", err)
				}
				clientCtx = clientCtx.WithHeight(status.SyncInfo.LatestBlockHeight)
			}
			queryClient := types.NewQueryClient(clientCtx)
			pageKey := []byte(nil)
			principalLiability, activeShares, rewardLiability := math.ZeroInt(), math.ZeroInt(), math.ZeroInt()
			var principalPool, rewardPool []sdk.Coin
			var rewardState types.RewardState
			for {
				page, err := queryClient.ValidateState(context.Background(), &types.QueryValidateStateRequest{
					Pagination: &query.PageRequest{Key: pageKey, Limit: 500},
				})
				if err != nil {
					return err
				}
				if !page.Valid {
					return fmt.Errorf("USTC staking validation failed: %s", page.Detail)
				}
				pagePrincipal, ok := math.NewIntFromString(page.PrincipalLiability)
				if !ok {
					return fmt.Errorf("invalid principal liability returned by validation query")
				}
				pageShares, ok := math.NewIntFromString(page.ActiveShares)
				if !ok {
					return fmt.Errorf("invalid active shares returned by validation query")
				}
				pageRewards, ok := math.NewIntFromString(page.RewardLiability)
				if !ok {
					return fmt.Errorf("invalid reward liability returned by validation query")
				}
				principalLiability = principalLiability.Add(pagePrincipal)
				activeShares = activeShares.Add(pageShares)
				rewardLiability = rewardLiability.Add(pageRewards)
				principalPool, rewardPool, rewardState = page.PrincipalPoolBalances, page.RewardPoolBalances, page.RewardState
				pageKey = page.Pagination.NextKey
				if len(pageKey) == 0 {
					break
				}
			}
			principalBalance, err := validationPoolBalance(principalPool)
			if err != nil {
				return fmt.Errorf("principal pool: %w", err)
			}
			rewardBalance, err := validationPoolBalance(rewardPool)
			if err != nil {
				return fmt.Errorf("reward pool: %w", err)
			}
			rewardIndexValid := !rewardState.RewardIndex.IsNil() && !rewardState.RewardIndex.IsNegative()
			totalSharesValid := !rewardState.TotalShares.IsNil() && !rewardState.TotalShares.IsNegative() && rewardState.TotalShares.Equal(activeShares)
			valid := principalBalance.Equal(principalLiability) && rewardBalance.GTE(rewardLiability) && rewardIndexValid && totalSharesValid
			detail := "all USTC staking accounting checks passed at the pinned height"
			if !valid {
				detail = fmt.Sprintf("accounting mismatch: principal liability=%s pool=%s; active shares=%s stored=%s; reward liability=%s pool=%s; reward index valid=%t",
					principalLiability, principalBalance, activeShares, rewardState.TotalShares, rewardLiability, rewardBalance, rewardIndexValid)
			}
			return clientCtx.PrintProto(&types.QueryValidateStateResponse{Valid: valid, Detail: detail})
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func validationPoolBalance(balances []sdk.Coin) (math.Int, error) {
	balance := math.ZeroInt()
	for _, coin := range balances {
		if coin.Denom != types.BondDenom {
			return math.ZeroInt(), fmt.Errorf("unexpected denomination %s", coin.Denom)
		}
		if coin.Amount.IsNil() || coin.Amount.IsNegative() {
			return math.ZeroInt(), fmt.Errorf("invalid balance %s", coin)
		}
		balance = balance.Add(coin.Amount)
	}
	return balance, nil
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
