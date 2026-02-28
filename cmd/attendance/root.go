package attendancecmd

import "github.com/spf13/cobra"

// Command returns the attendance parent command with all subcommands registered.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attendance",
		Short: "Attendance management commands",
	}

	cmd.AddCommand(increaseQuotaCmd)

	return cmd
}
