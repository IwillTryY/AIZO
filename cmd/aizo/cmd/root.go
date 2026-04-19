package cmd

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/realityos/aizo/policy"
	"github.com/realityos/aizo/storage"
	"github.com/realityos/aizo/tenant"
)

var (
	dbPath   string
	tenantID string
	db       *storage.DB
	sqlDB    *sql.DB
	audit    *storage.AuditStore
	tenantMgr *tenant.Manager
	policyEng *policy.Engine
)

var rootCmd = &cobra.Command{
	Use:   "aizo",
	Short: "AIZO - Universal Operations Layer",
	Long:  "A universal control plane that transforms heterogeneous infrastructure into a unified, observable, controllable, and self-healing network.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip init for help/completion
		if cmd.Name() == "help" || cmd.Name() == "completion" {
			return nil
		}

		var err error
		db, err = storage.Open(dbPath)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		sqlDB = db.SQL()
		audit = storage.NewAuditStore(db)
		tenantMgr = tenant.NewManager(sqlDB)
		if tenantID != "" {
			tenantMgr.Switch(tenantID)
		}

		// Load policies
		policyEng = policy.NewEngine()
		home, _ := os.UserHomeDir()
		policyDir := filepath.Join(home, ".aizo", "policies")
		if policies, err := policy.LoadFromDir(policyDir); err == nil {
			for _, p := range policies {
				policyEng.AddPolicy(p)
			}
		}

		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if db != nil {
			db.Close()
		}
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "database path (default: ~/.aizo/aizo.db)")
	rootCmd.PersistentFlags().StringVar(&tenantID, "tenant", "", "tenant ID (default: default)")

	rootCmd.AddCommand(adapterCmd)
	rootCmd.AddCommand(entityCmd)
	rootCmd.AddCommand(metricsCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(stateCmd)
	rootCmd.AddCommand(containerCmd)
	rootCmd.AddCommand(rulesCmd)
	rootCmd.AddCommand(proposalsCmd)
	rootCmd.AddCommand(incidentsCmd)
	rootCmd.AddCommand(policyCmd)
	rootCmd.AddCommand(tenantCmd)
	rootCmd.AddCommand(auditCmd)

	// Shortcuts
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(execCmd)
	rootCmd.AddCommand(listCmd)
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}
