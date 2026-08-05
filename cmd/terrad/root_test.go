package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	cmtcli "github.com/cometbft/cometbft/libs/cli"
	"github.com/stretchr/testify/require"
)

func TestModuleGroupHelpSkipsConfigInitialization(t *testing.T) {
	home := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(home, "config"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(home, "config", "config.json"), []byte("/"), 0o600))

	rootCmd, _ := NewRootCmd()
	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	executor := cmtcli.PrepareBaseCmd(rootCmd, "", home)
	executor.Exit = func(int) {}

	for _, tc := range []struct {
		name     string
		args     []string
		expected string
	}{
		{name: "transactions", args: []string{"tx", "ustcstaking", "--help"}, expected: "USTC staking transaction subcommands"},
		{name: "queries", args: []string{"query", "ustcstaking", "--help"}, expected: "Querying commands for the USTC staking module"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output.Reset()
			rootCmd.SetArgs(tc.args)
			err := executor.Execute()
			require.NoError(t, err)
			require.Contains(t, output.String(), tc.expected)
		})
	}
}
