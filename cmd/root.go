package cmd

import (
	"github.com/spf13/cobra"
)

var (
	debug     bool
	logPretty bool
)

var rootCmd = &cobra.Command{
	Use:   "k8s-hostpath-device-plugin",
	Short: "Kubernetes hostPath device plugin",
	Long:  `This is a very thin device plugin which just exposed a host path to a container.`,
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "sets log level to debug")
	rootCmd.PersistentFlags().BoolVar(&logPretty, "log-pretty", true, "set pretty logging(human-friendly & colorized output), json logging if false")
}
