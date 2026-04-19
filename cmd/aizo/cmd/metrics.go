package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/realityos/aizo/layer3"
)

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Query and manage metrics",
}

var metricsQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query metrics",
	RunE: func(cmd *cobra.Command, args []string) error {
		entity, _ := cmd.Flags().GetString("entity")
		name, _ := cmd.Flags().GetString("name")
		last, _ := cmd.Flags().GetString("last")

		duration, err := time.ParseDuration(last)
		if err != nil {
			duration = 1 * time.Hour
		}

		store := layer3.NewSQLiteMetricsStorage(sqlDB)
		ctx := context.Background()

		req := &layer3.QueryRequest{
			EntityID:  entity,
			StartTime: time.Now().Add(-duration),
			EndTime:   time.Now(),
			Limit:     50,
			Filters:   make(map[string]interface{}),
		}
		if name != "" {
			req.Filters["name"] = name
		}

		metrics, err := store.Query(ctx, req)
		if err != nil {
			return err
		}

		if len(metrics) == 0 {
			fmt.Println("No metrics found")
			return nil
		}

		for _, m := range metrics {
			fmt.Printf("  %s  %-20s  %-15s  %.2f\n",
				m.Timestamp.Format("15:04:05"), m.Name, m.EntityID, m.Value)
		}
		fmt.Printf("\n%d metrics returned\n", len(metrics))
		return nil
	},
}

func init() {
	metricsQueryCmd.Flags().String("entity", "", "filter by entity ID")
	metricsQueryCmd.Flags().String("name", "", "filter by metric name")
	metricsQueryCmd.Flags().String("last", "1h", "time range (e.g. 1h, 30m, 24h)")

	metricsCmd.AddCommand(metricsQueryCmd)
}
