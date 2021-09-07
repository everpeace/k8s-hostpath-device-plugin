package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Version of the program.  It will be injected at build time.
	Version string = "undefined"
	// Revision of the program.  It will be injected at build time.
	Revision string = "undefined"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print Version",
	Long:  `Print version.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(VersionString())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func VersionString() string {
	return fmt.Sprintf(`{"Version": "%s", "Revision": "%s"}
`, Version, Revision)
}
