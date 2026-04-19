package main

import (
	"context"
	"fmt"
	"time"

	"github.com/realityos/aizo/layer5"
)

func main() {
	fmt.Println("=== RealityOS Layer 5 Demo ===")
	fmt.Println("Control Plane - Container Runtime\n")

	// Create Layer 5 manager
	config := &layer5.ManagerConfig{
		DataRoot:        "/var/lib/realityos",
		EnableMetrics:   true,
		MetricsInterval: 30 * time.Second,
		DefaultNetwork:  "bridge",
	}

	manager := layer5.NewManager(config)
	ctx := context.Background()

	// Start manager
	manager.Start(ctx)
	fmt.Println("✓ Layer 5 manager started\n")

	// 1. Image Management
	fmt.Println("1. Image Management")
	fmt.Println("-------------------")

	// Pull an image
	image, err := manager.PullImage(ctx, "realityos/nginx:latest")
	if err != nil {
		fmt.Printf("Error pulling image: %v\n", err)
	} else {
		fmt.Printf("✓ Pulled image: %s\n", image.RepoTags[0])
		fmt.Printf("  ID: %s\n", image.ID[:12])
		fmt.Printf("  Size: %.2f MB\n", float64(image.Size)/(1024*1024))
		fmt.Printf("  Layers: %d\n", len(image.Layers))
	}

	// Build an image
	dockerfile := `
FROM realityos/base:latest
WORKDIR /app
COPY . .
RUN npm install
CMD ["npm", "start"]
`
	builtImage, err := manager.BuildImage(ctx, dockerfile, []string{"myapp:v1.0"})
	if err != nil {
		fmt.Printf("Error building image: %v\n", err)
	} else {
		fmt.Printf("✓ Built image: %s\n", builtImage.RepoTags[0])
		fmt.Printf("  ID: %s\n", builtImage.ID[:12])
	}

	// List images
	images, _ := manager.ListImages(ctx)
	fmt.Printf("\n✓ Total images: %d\n", len(images))

	// 2. Network Management
	fmt.Println("\n2. Network Management")
	fmt.Println("---------------------")

	// Create a custom network
	network, err := manager.CreateNetwork(ctx, "app-network", "bridge", false, map[string]string{
		"purpose": "application",
	})
	if err != nil {
		fmt.Printf("Error creating network: %v\n", err)
	} else {
		fmt.Printf("✓ Created network: %s\n", network.Name)
		fmt.Printf("  ID: %s\n", network.ID[:12])
		fmt.Printf("  Driver: %s\n", network.Driver)
		fmt.Printf("  Subnet: %s\n", network.IPAM.Config[0].Subnet)
	}

	// List networks
	networks, _ := manager.ListNetworks(ctx)
	fmt.Printf("\n✓ Total networks: %d\n", len(networks))
	for _, net := range networks {
		fmt.Printf("  - %s (%s)\n", net.Name, net.Driver)
	}

	// 3. Volume Management
	fmt.Println("\n3. Volume Management")
	fmt.Println("--------------------")

	// Create a volume
	volume, err := manager.CreateVolume(ctx, "app-data", "local", map[string]string{
		"type": "persistent",
	})
	if err != nil {
		fmt.Printf("Error creating volume: %v\n", err)
	} else {
		fmt.Printf("✓ Created volume: %s\n", volume.Name)
		fmt.Printf("  Driver: %s\n", volume.Driver)
		fmt.Printf("  Mountpoint: %s\n", volume.Mountpoint)
	}

	// List volumes
	volumes, _ := manager.ListVolumes(ctx)
	fmt.Printf("\n✓ Total volumes: %d\n", len(volumes))

	// 4. Container Lifecycle
	fmt.Println("\n4. Container Lifecycle")
	fmt.Println("----------------------")

	// Create container configuration
	containerConfig := &layer5.ContainerConfig{
		Hostname:   "web-server",
		User:       "root",
		WorkingDir: "/app",
		Cmd:        []string{"nginx", "-g", "daemon off;"},
		Env:        []string{"PORT=8080", "ENV=production"},
		Labels: map[string]string{
			"app":     "nginx",
			"version": "1.0",
		},
		StopTimeout: 10,
	}

	// Create container
	container, err := manager.CreateContainer(ctx, containerConfig, "web-server-1")
	if err != nil {
		fmt.Printf("Error creating container: %v\n", err)
	} else {
		fmt.Printf("✓ Created container: %s\n", container.Name)
		fmt.Printf("  ID: %s\n", container.ID[:12])
		fmt.Printf("  Status: %s\n", container.Status)
	}

	// Start container
	err = manager.StartContainer(ctx, container.ID)
	if err != nil {
		fmt.Printf("Error starting container: %v\n", err)
	} else {
		fmt.Printf("✓ Started container: %s\n", container.Name)

		// Get updated container
		container, _ = manager.GetContainer(ctx, container.ID)
		fmt.Printf("  Status: %s\n", container.Status)
		fmt.Printf("  PID: %d\n", container.State.Pid)
		fmt.Printf("  Started at: %s\n", container.StartedAt.Format(time.RFC3339))
	}

	// 5. Container Operations
	fmt.Println("\n5. Container Operations")
	fmt.Println("-----------------------")

	// Pause container
	err = manager.PauseContainer(ctx, container.ID)
	if err != nil {
		fmt.Printf("Error pausing container: %v\n", err)
	} else {
		fmt.Println("✓ Paused container")
		container, _ = manager.GetContainer(ctx, container.ID)
		fmt.Printf("  Status: %s\n", container.Status)
	}

	// Unpause container
	time.Sleep(100 * time.Millisecond)
	err = manager.UnpauseContainer(ctx, container.ID)
	if err != nil {
		fmt.Printf("Error unpausing container: %v\n", err)
	} else {
		fmt.Println("✓ Unpaused container")
		container, _ = manager.GetContainer(ctx, container.ID)
		fmt.Printf("  Status: %s\n", container.Status)
	}

	// Get container stats
	stats, err := manager.GetContainerStats(ctx, container.ID)
	if err != nil {
		fmt.Printf("Error getting stats: %v\n", err)
	} else {
		fmt.Printf("\n✓ Container stats:\n")
		fmt.Printf("  CPU usage: %.2f%%\n", float64(stats.CPU.TotalUsage)/float64(stats.CPU.SystemCPUUsage)*100)
		fmt.Printf("  Memory usage: %.2f MB\n", float64(stats.Memory.Usage)/(1024*1024))
		fmt.Printf("  Memory limit: %.2f MB\n", float64(stats.Memory.Limit)/(1024*1024))
		fmt.Printf("  Network RX: %.2f KB\n", float64(stats.Network.RxBytes)/1024)
		fmt.Printf("  Network TX: %.2f KB\n", float64(stats.Network.TxBytes)/1024)
		fmt.Printf("  PIDs: %d/%d\n", stats.PIDs.Current, stats.PIDs.Limit)
	}

	// Get container logs
	logs, err := manager.GetContainerLogs(ctx, container.ID, &layer5.ContainerLogs{
		Stdout:     true,
		Stderr:     true,
		Timestamps: true,
		Tail:       "10",
	})
	if err != nil {
		fmt.Printf("Error getting logs: %v\n", err)
	} else {
		fmt.Printf("\n✓ Container logs:\n")
		for _, log := range logs {
			fmt.Printf("  %s\n", log)
		}
	}

	// 6. Multiple Containers
	fmt.Println("\n6. Running Multiple Containers")
	fmt.Println("------------------------------")

	// Create and start multiple containers
	for i := 1; i <= 3; i++ {
		config := &layer5.ContainerConfig{
			Hostname:   fmt.Sprintf("worker-%d", i),
			WorkingDir: "/app",
			Cmd:        []string{"worker", "process"},
			Labels: map[string]string{
				"role": "worker",
			},
		}

		c, err := manager.RunContainer(ctx, config, fmt.Sprintf("worker-%d", i))
		if err != nil {
			fmt.Printf("Error running container: %v\n", err)
		} else {
			fmt.Printf("✓ Started worker-%d (ID: %s)\n", i, c.ID[:12])
		}
	}

	// List all containers
	allContainers, _ := manager.ListContainers(ctx, true)
	fmt.Printf("\n✓ Total containers: %d\n", len(allContainers))
	fmt.Println("  Running containers:")
	for _, c := range allContainers {
		if c.State.Running {
			fmt.Printf("    - %s (%s) - %s\n", c.Name, c.ID[:12], c.Status)
		}
	}

	// 7. Network Connectivity
	fmt.Println("\n7. Network Connectivity")
	fmt.Println("-----------------------")

	// Connect container to custom network
	err = manager.ConnectContainerToNetwork(ctx, network.ID, container.ID, nil)
	if err != nil {
		fmt.Printf("Error connecting to network: %v\n", err)
	} else {
		fmt.Printf("✓ Connected %s to %s\n", container.Name, network.Name)

		// Get network details
		net, _ := manager.GetNetworkManager().GetNetwork(ctx, network.ID)
		if endpoint, exists := net.Containers[container.ID]; exists {
			fmt.Printf("  IP Address: %s\n", endpoint.IPAddress)
			fmt.Printf("  Gateway: %s\n", endpoint.Gateway)
		}
	}

	// 8. Resource Management
	fmt.Println("\n8. Resource Management")
	fmt.Println("----------------------")

	// Create container with resource limits
	limitedConfig := &layer5.ContainerConfig{
		Hostname:   "limited-container",
		WorkingDir: "/app",
		Cmd:        []string{"app"},
	}

	limitedContainer, err := manager.CreateContainer(ctx, limitedConfig, "limited-app")
	if err != nil {
		fmt.Printf("Error creating container: %v\n", err)
	} else {
		// Set resource limits
		limitedContainer.Resources = &layer5.ResourceLimits{
			Memory:    512 * 1024 * 1024, // 512MB
			CPUShares: 512,
			CPUQuota:  50000,
			CPUPeriod: 100000,
			PidsLimit: 100,
		}

		fmt.Printf("✓ Created container with resource limits:\n")
		fmt.Printf("  Memory: %.0f MB\n", float64(limitedContainer.Resources.Memory)/(1024*1024))
		fmt.Printf("  CPU Shares: %d\n", limitedContainer.Resources.CPUShares)
		fmt.Printf("  PIDs Limit: %d\n", limitedContainer.Resources.PidsLimit)
	}

	// 9. System Statistics
	fmt.Println("\n9. System Statistics")
	fmt.Println("--------------------")

	systemStats, _ := manager.GetStats(ctx)
	fmt.Printf("✓ System overview:\n")
	fmt.Printf("  Total containers: %d\n", systemStats.TotalContainers)
	fmt.Printf("  Running containers: %d\n", systemStats.RunningContainers)
	fmt.Printf("  Total images: %d\n", systemStats.TotalImages)
	fmt.Printf("  Total networks: %d\n", systemStats.TotalNetworks)
	fmt.Printf("  Total volumes: %d\n", systemStats.TotalVolumes)
	fmt.Printf("\n  Containers by status:\n")
	for status, count := range systemStats.ByStatus {
		fmt.Printf("    %s: %d\n", status, count)
	}

	// 10. Cleanup
	fmt.Println("\n10. Cleanup Operations")
	fmt.Println("----------------------")

	// Stop all running containers
	fmt.Println("Stopping containers...")
	for _, c := range allContainers {
		if c.State.Running {
			_ = manager.StopContainer(ctx, c.ID, 5)
			fmt.Printf("  ✓ Stopped %s\n", c.Name)
		}
	}

	// Prune system
	pruneResult, err := manager.Prune(ctx)
	if err != nil {
		fmt.Printf("Error pruning: %v\n", err)
	} else {
		fmt.Printf("\n✓ Prune results:\n")
		fmt.Printf("  Containers removed: %d\n", len(pruneResult.ContainersRemoved))
		fmt.Printf("  Images removed: %d\n", len(pruneResult.ImagesRemoved))
		fmt.Printf("  Networks removed: %d\n", len(pruneResult.NetworksRemoved))
		fmt.Printf("  Volumes removed: %d\n", len(pruneResult.VolumesRemoved))
	}

	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("\nLayer 5 provides:")
	fmt.Println("  • Container lifecycle management")
	fmt.Println("  • Image building and management")
	fmt.Println("  • Network isolation and connectivity")
	fmt.Println("  • Volume management")
	fmt.Println("  • Resource limits and controls")
	fmt.Println("  • Container statistics and monitoring")
	fmt.Println("  • Multi-container orchestration")
}
