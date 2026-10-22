package main

import (
	"testing"

	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestApplyConfigOverrides_NoUncoveredFunctionsFlag(t *testing.T) {
	cmd := &cobra.Command{}
	initFlags(cmd)

	flag := cmd.Flags().Lookup(NoUncoveredFunctionsFlag)
	require.NotNil(t, flag)
	require.Equal(t, "O", flag.Shorthand)

	cfg := &config.Config{}
	cfg.ApplyDefaults()
	applyConfigOverrides(cfg, cmd, false)
	require.False(t, cfg.NoUncoveredFunctions, "uncovered functions are shown by default")

	require.NoError(t, cmd.Flags().Set(NoUncoveredFunctionsFlag, "true"))
	applyConfigOverrides(cfg, cmd, false)
	require.True(t, cfg.NoUncoveredFunctions)
	require.False(t, cfg.NoUncoveredLines, "the lines setting is independent")
}

func TestApplyConfigOverrides_NoUncoveredLinesLeavesFunctions(t *testing.T) {
	cmd := &cobra.Command{}
	initFlags(cmd)

	cfg := &config.Config{}
	cfg.ApplyDefaults()
	require.NoError(t, cmd.Flags().Set(NoUncoveredLinesFlag, "true"))
	applyConfigOverrides(cfg, cmd, false)
	require.True(t, cfg.NoUncoveredLines)
	require.False(t, cfg.NoUncoveredFunctions)
}

func TestConfig_NoUncoveredFunctionsKey(t *testing.T) {
	cfg := &config.Config{}
	require.NoError(t, yaml.Unmarshal([]byte("noUncoveredFunctions: true\n"), cfg))
	require.True(t, cfg.NoUncoveredFunctions)
	require.False(t, cfg.NoUncoveredLines)
}
