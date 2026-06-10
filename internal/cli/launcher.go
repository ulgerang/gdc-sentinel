package cli

import (
	"github.com/spf13/cobra"
	"github.com/ulgerang/gdc-sentinel/internal/daemon"
)

var sentinelDirFlag string

var launcherCmd = &cobra.Command{
	Use:    "_daemon",
	Short:  "Internal: run as the daemon launcher process",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return daemon.RunLauncher(sentinelDirFlag)
	},
}

func init() {
	launcherCmd.Flags().StringVar(&sentinelDirFlag, "sentinel-dir", "", "path to .gdc-sentinel directory")
	rootCmd.AddCommand(launcherCmd)
}
