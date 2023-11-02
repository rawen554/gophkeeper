package cmd

import (
	"context"

	"github.com/spf13/cobra"
)

const GroupID = "record"

func init() {
	recordCmd.AddCommand(putRecordCmd)
	recordCmd.AddCommand(getRecordCmd)
	recordCmd.AddCommand(listRecordsCmd)
	rootCmd.AddCommand(recordCmd)
}

var recordCmd = &cobra.Command{
	Use:   "record [sub]",
	Short: "Manage data records",
}

var putRecordCmd = &cobra.Command{
	Use:   "put",
	Short: "Put data record",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		Login(context.Background())
	},
}

var getRecordCmd = &cobra.Command{
	Use:   "get",
	Short: "Get data record",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		Login(context.Background())
	},
}

var listRecordsCmd = &cobra.Command{
	Use:   "list",
	Short: "List data records",
	Run: func(cmd *cobra.Command, args []string) {
		Login(context.Background())
	},
}
