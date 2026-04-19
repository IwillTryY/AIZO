package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/realityos/aizo/security"
	"github.com/realityos/aizo/storage"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Set up AIZO on this machine",
	Long:  "Creates directories, database, certificates, default policies and rules. Run once on a new machine.",
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")

		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot find home directory: %w", err)
		}

		aizoDir := filepath.Join(home, ".aizo")
		certsDir := filepath.Join(aizoDir, "certs")
		policiesDir := filepath.Join(aizoDir, "policies")
		rulesDir := filepath.Join(aizoDir, "rules")
		dbPath := filepath.Join(aizoDir, "aizo.db")

		fmt.Println("Installing AIZO...\n")

		// 1. Create directory structure
		fmt.Print("  Creating directories... ")
		dirs := []string{aizoDir, certsDir, policiesDir, rulesDir}
		for _, d := range dirs {
			if err := os.MkdirAll(d, 0755); err != nil {
				return fmt.Errorf("mkdir %s: %w", d, err)
			}
		}
		fmt.Println("done")

		// 2. Create database
		fmt.Print("  Creating database... ")
		d, err := storage.Open(dbPath)
		if err != nil {
			return fmt.Errorf("database: %w", err)
		}
		d.Close()
		fmt.Println("done")

		// 3. Generate TLS certificates
		caPath := filepath.Join(certsDir, "ca.crt")
		if force || !fileExists(caPath) {
			fmt.Print("  Generating TLS certificates... ")
			caCert, caKey, err := security.GenerateSelfSignedCA(certsDir)
			if err != nil {
				return fmt.Errorf("generate CA: %w", err)
			}

			hostname, _ := os.Hostname()
			_, _, err = security.GenerateNodeCert(certsDir, hostname, caCert, caKey)
			if err != nil {
				return fmt.Errorf("generate node cert: %w", err)
			}
			fmt.Println("done")
		} else {
			fmt.Println("  TLS certificates already exist (use --force to regenerate)")
		}

		// 4. Write default policy
		policyPath := filepath.Join(policiesDir, "default.yaml")
		if force || !fileExists(policyPath) {
			fmt.Print("  Writing default policy... ")
			if err := os.WriteFile(policyPath, []byte(defaultPolicy), 0644); err != nil {
				return fmt.Errorf("write policy: %w", err)
			}
			fmt.Println("done")
		} else {
			fmt.Println("  Default policy already exists")
		}

		// 5. Write default rules
		rulesPath := filepath.Join(rulesDir, "default.yaml")
		if force || !fileExists(rulesPath) {
			fmt.Print("  Writing default rules... ")
			if err := os.WriteFile(rulesPath, []byte(defaultRules), 0644); err != nil {
				return fmt.Errorf("write rules: %w", err)
			}
			fmt.Println("done")
		} else {
			fmt.Println("  Default rules already exist")
		}

		// 6. Platform-specific setup
		switch runtime.GOOS {
		case "windows":
			fmt.Println("\n  Windows detected. Setting up WSL2...")
			setupWindows()
		case "linux":
			fmt.Println("\n  Linux detected. Setting up container directories...")
			setupLinux(home)
		case "darwin":
			fmt.Println("\n  macOS detected. Setting up container directories...")
			setupDarwin(home)
		}

		// Summary
		fmt.Println("\n  AIZO installed successfully!\n")
		fmt.Println("  Directories:")
		fmt.Printf("    Config:       %s\n", aizoDir)
		fmt.Printf("    Database:     %s\n", dbPath)
		fmt.Printf("    Certificates: %s\n", certsDir)
		fmt.Printf("    Policies:     %s\n", policiesDir)
		fmt.Printf("    Rules:        %s\n", rulesDir)
		fmt.Println("\n  Next steps:")
		fmt.Println("    aizo list containers          # see containers")
		fmt.Println("    aizo create container 1 web   # create a container")
		fmt.Println("    aizo tui                      # launch terminal UI")

		return nil
	},
}

func setupWindows() {
	// Check WSL2
	out, err := exec.Command("wsl", "--status").Output()
	if err != nil {
		fmt.Println("    WSL2 not found. Install it with: wsl --install -d Ubuntu")
		return
	}
	_ = out
	fmt.Println("    WSL2 detected")

	// Check Ubuntu
	check := exec.Command("wsl", "-d", "Ubuntu", "--exec", "echo", "ok")
	if err := check.Run(); err != nil {
		fmt.Println("    Ubuntu not found. Install it with: wsl --install -d Ubuntu")
		return
	}
	fmt.Println("    Ubuntu distro found")

	// Install busybox
	fmt.Print("    Checking busybox... ")
	checkBusybox := exec.Command("wsl", "-d", "Ubuntu", "--exec", "which", "busybox")
	if err := checkBusybox.Run(); err != nil {
		fmt.Println("not found")
		fmt.Println("    Run manually: wsl -d Ubuntu --exec sudo apt-get install -y busybox-static")
	} else {
		fmt.Println("found")
	}

	// Create container directories
	fmt.Print("    Creating container directories... ")
	exec.Command("wsl", "-d", "Ubuntu", "--exec", "mkdir", "-p",
		"~/realityos/containers", "~/realityos/images", "~/realityos/volumes").Run()
	fmt.Println("done")
}

func setupLinux(home string) {
	dirs := []string{
		filepath.Join(home, "realityos", "containers"),
		filepath.Join(home, "realityos", "images"),
		filepath.Join(home, "realityos", "volumes"),
	}
	for _, d := range dirs {
		os.MkdirAll(d, 0755)
	}
	fmt.Println("    Container directories created")

	// Check busybox
	if _, err := exec.LookPath("busybox"); err != nil {
		fmt.Println("    busybox not found. Install it: sudo apt-get install -y busybox-static")
	} else {
		fmt.Println("    busybox found")
	}
}

func setupDarwin(home string) {
	dirs := []string{
		filepath.Join(home, "realityos", "containers"),
		filepath.Join(home, "realityos", "images"),
		filepath.Join(home, "realityos", "volumes"),
	}
	for _, d := range dirs {
		os.MkdirAll(d, 0755)
	}
	fmt.Println("    Container directories created")
	fmt.Println("    Note: Full container isolation requires Linux. macOS uses process-level isolation.")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

const defaultPolicy = `- id: default-allow
  name: Default Allow All
  description: Allows all actions for all actors (starter policy)
  rules:
    - actions: ["*"]
      resources: ["*"]
      actors: ["*"]
      effect: allow
  effect: allow
  priority: 1
  enabled: true
`

const defaultRules = `- id: custom-example
  name: Example Custom Rule
  description: Example rule - edit this file to add your own
  conditions:
    - metric: error_rate
      operator: ">"
      value: 100
  action:
    type: investigate
    risk: low
    reversible: true
    auto_approve: true
    reasoning: Error rate exceeded threshold
  priority: 50
  enabled: false
`

func init() {
	installCmd.Flags().Bool("force", false, "overwrite existing config")
}
