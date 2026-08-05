package ustcstaking

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestAppModuleBasicExposesCLICommands(t *testing.T) {
	basic := AppModuleBasic{}

	txCmd := basic.GetTxCmd()
	queryCmd := basic.GetQueryCmd()

	require.NotNil(t, txCmd)
	require.NotNil(t, queryCmd)
	require.Equal(t, "ustcstaking", txCmd.Use)
	require.Equal(t, "ustcstaking", queryCmd.Use)
	require.ElementsMatch(t,
		[]string{"stake", "begin-unbonding", "withdraw", "claim-rewards"},
		commandUses(txCmd),
	)
	require.ElementsMatch(t,
		[]string{"position", "positions", "reward-state", "params"},
		commandUses(queryCmd),
	)
}

func commandUses(cmd interface{ Commands() []*cobra.Command }) []string {
	commands := cmd.Commands()
	uses := make([]string, 0, len(commands))
	for _, child := range commands {
		uses = append(uses, strings.Fields(child.Use)[0])
	}
	return uses
}
