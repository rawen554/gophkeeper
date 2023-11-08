package cmd

import (
	"context"
	"errors"
	"fmt"
	"syscall"

	"github.com/rawen554/goph-keeper/cmd/client/internal/logic"
	"github.com/spf13/cobra"
)

func init() {
	putRecordCmd.AddCommand()
	recordCmd.AddCommand(putRecordCmd)
	recordCmd.AddCommand(getRecordCmd)
	recordCmd.AddCommand(listRecordsCmd)
	recordCmd.AddCommand(syncRecordsCmd)
	rootCmd.AddCommand(recordCmd)
}

var recordCmd = &cobra.Command{
	Use:   "records [sub]",
	Short: "Manage data records",
}

var putRecordCmd = &cobra.Command{
	Use:   "put [record_type] [path|data] [name]",
	Short: "Put data record",
	Long:  "record_type=PASS|TEXT|BIN|CARD\nFor PASS data type required following pattern %LOGIN%:%PASSWORD%\nName is required.",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		record, err := logic.PutRecord(context.Background(), args)
		if err != nil {
			if errors.Is(err, syscall.ECONNREFUSED) {
				if err := logic.SaveOrUpdateData(record); err != nil {
					fmt.Printf("error saving locally %s: [%w]\n", record.Name, err)
				}
				fmt.Printf("saved local data: %s\n", record.Name)
				return
			}

			fmt.Printf("error: %v", err)
		}

		if err := logic.SaveOrUpdateData(record); err != nil {
			fmt.Printf("error saving locally: %s\n", record.Name)
		}

		fmt.Printf("%+v\n", record)
	},
}

var getRecordCmd = &cobra.Command{
	Use:   "get [name]",
	Short: "Get data record",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		record, err := logic.GetRecord(context.Background(), args[0])
		if err != nil {
			fmt.Printf("error: %v", err)
		}

		fmt.Printf("%+v\n", record)
	},
}

var listRecordsCmd = &cobra.Command{
	Use:   "list",
	Short: "List data records",
	Run: func(cmd *cobra.Command, args []string) {
		records, err := logic.ListRecords(context.Background())
		if err != nil {
			fmt.Printf("error: %v", err)
		}

		for _, r := range records {
			fmt.Printf("%+v\n", r)
		}
	},
}

var syncRecordsCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync data records",
	Run: func(cmd *cobra.Command, args []string) {
		if err := logic.SyncDataRecords(context.Background()); err != nil {
			fmt.Printf("error: %v", err)
		}

		fmt.Println("sync successfull")
	},
}
