package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/realityos/aizo/layer3"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Search and view logs",
}

var logsSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search logs by message content",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		entity, _ := cmd.Flags().GetString("entity")
		last, _ := cmd.Flags().GetString("last")
		limit, _ := cmd.Flags().GetInt("limit")

		duration, err := time.ParseDuration(last)
		if err != nil {
			duration = 1 * time.Hour
		}

		store := layer3.NewSQLiteLogStorage(sqlDB)
		ctx := context.Background()

		logs, err := store.Search(ctx, args[0], entity, time.Now().Add(-duration), time.Now(), limit)
		if err != nil {
			return err
		}

		if len(logs) == 0 {
			fmt.Println("No logs found")
			return nil
		}

		for _, l := range logs {
			fmt.Printf("  %s  [%-7s]  %-15s  %s\n",
				l.Timestamp.Format("15:04:05"), l.Level, l.EntityID, l.Message)
		}
		fmt.Printf("\n%d logs returned\n", len(logs))
		return nil
	},
}

func init() {
	logsSearchCmd.Flags().String("entity", "", "filter by entity ID")
	logsSearchCmd.Flags().String("last", "1h", "time range")
	logsSearchCmd.Flags().Int("limit", 50, "max results")

	logsCmd.AddCommand(logsSearchCmd)
}
