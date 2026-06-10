package cli

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	Version  = "0.1.0-dev"
	BuildDate = "unknown"

	cfgFile   string
	verbose   bool
	quiet     bool
	noColor   bool
	showVersion bool
)

var rootCmd = &cobra.Command{
	Use:   "gdc-sentinel",
	Short: "GDC Sentinel — drift detection and agent orchestration for GDC",
	Long: `GDC Sentinel is a companion tool for GDC (Graph-Driven Codebase).
It consumes GDC CLI JSON outputs to detect drift, create context packets,
and orchestrate external coding agents.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if noColor {
			color.NoColor = true
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			fmt.Printf("gdc-sentinel %s (built %s)\n", Version, BuildDate)
			return nil
		}
		return cmd.Help()
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .gdc-sentinel/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "suppress non-essential output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
	rootCmd.Flags().BoolVar(&showVersion, "version", false, "print version")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(explainCmd)
	rootCmd.AddCommand(packetCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(noteCmd)
}

func Execute() error {
	return rootCmd.Execute()
}

func printSuccess(format string, args ...interface{}) {
	if !quiet {
		color.Green(format, args...)
	}
}

func printInfo(format string, args ...interface{}) {
	if !quiet {
		color.Cyan(format, args...)
	}
}

func printWarning(format string, args ...interface{}) {
	if !quiet {
		color.Yellow(format, args...)
	}
}

func printError(format string, args ...interface{}) {
	color.Red(format, args...)
	fmt.Fprintln(os.Stderr)
}
