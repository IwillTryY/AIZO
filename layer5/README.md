# Layer 5: Control Plane - Container Runtime - Version 1

## Overview
Layer 5 is the Control Plane for RealityOS - a Docker-like container runtime that provides isolated execution environments for applications and services. It manages the complete container lifecycle, images, networks, and volumes.

## Architecture

### Core Components

1. **Types** (`types.go`)
   - Container: Isolated execution environment
   - Image: Container image with layers
   - Network: Container networking
   - Volume: Persistent data storage
   - Resource limits and statistics
   - Configuration structures

2. **Container Runtime** (`container_runtime.go`)
   - Container lifecycle management (create, start, stop, remove)
   - Container state tracking
   - Pause/unpause operations
   - Container statistics
   - Command execution in containers
   - Log retrieval

3. **Image Manager** (`image_manager.go`)
   - Image pulling from registries
   - Image building from Dockerfiles
   - Image tagging and versioning
   - Image removal and pruning
   - Layer management

4. **Network Manager** (`network_manager.go`)
   - Network creation and management
   - Default networks (bridge, host, none)
   - Container network connectivity
   - IP address management (IPAM)
   - Port mapping
   - Network isolation

5. **Volume Manager** (`volume_manager.go`)
   - Volume creation and management
   - Persistent data storage
   - Volume mounting
   - Volume pruning

6. **Manager** (`manager.go`)
   - High-level orchestration
   - Unified API for all operations
   - System statistics
   - Resource pruning
   - Multi-container management

## Features Implemented

### ✅ Container Management
- Create, start, stop, remove containers
- Pause and unpause containers
- Restart containers
- Container state tracking
- Resource limits (CPU, memory, PIDs)
- Environment variables and labels

### ✅ Image Management
- Pull images from registries
- Build images from Dockerfiles
- Tag and version images
- List and inspect images
- Remove images
- Layer-based storage

### ✅ Network Management
- Create custom networks
- Default networks (bridge, host, none)
- Connect/disconnect containers
- IP address allocation
- Port mapping
- Network isolation

### ✅ Volume Management
- Create and manage volumes
- Persistent data storage
- Volume mounting
- Volume drivers

### ✅ Resource Management
- CPU limits and shares
- Memory limits
- PID limits
- Block I/O limits
- Resource statistics

### ✅ Monitoring
- Container statistics (CPU, memory, network, I/O)
- Container logs
- System-wide statistics

## Usage Example

```go
// Create Layer 5 manager
config := &layer5.ManagerConfig{
    DataRoot:        "/var/lib/realityos",
    EnableMetrics:   true,
    MetricsInterval: 30 * time.Second,
}

manager := layer5.NewManager(config)
ctx := context.Background()

// Start manager
manager.Start(ctx)

// Pull an image
image, err := manager.PullImage(ctx, "nginx:latest")

// Create a network
network, err := manager.CreateNetwork(ctx, "app-network", "bridge", false, nil)

// Create a volume
volume, err := manager.CreateVolume(ctx, "app-data", "local", nil)

// Create container configuration
config := &layer5.ContainerConfig{
    Hostname:   "web-server",
    WorkingDir: "/app",
    Cmd:        []string{"nginx", "-g", "daemon off;"},
    Env:        []string{"PORT=8080"},
    Labels: map[string]string{
        "app": "nginx",
    },
}

// Create and start container
container, err := manager.RunContainer(ctx, config, "web-server-1")

// Get container stats
stats, err := manager.GetContainerStats(ctx, container.ID)
fmt.Printf("CPU: %.2f%%, Memory: %.2f MB\n",
    float64(stats.CPU.TotalUsage)/float64(stats.CPU.SystemCPUUsage)*100,
    float64(stats.Memory.Usage)/(1024*1024))

// Connect to network
err = manager.ConnectContainerToNetwork(ctx, network.ID, container.ID, nil)

// Get logs
logs, err := manager.GetContainerLogs(ctx, container.ID, &layer5.ContainerLogs{
    Stdout: true,
    Stderr: true,
    Tail:   "100",
})

// Stop container
err = manager.StopContainer(ctx, container.ID, 10)

// Remove container
err = manager.RemoveContainer(ctx, container.ID, false)
```

## Data Models

### Container
```go
Container {
    id: "unique-id"
    name: "container-name"
    image: "image:tag"
    status: created | running | paused | restarting | exited | dead
    state: {
        running: bool
        paused: bool
        pid: int
        exit_code: int
        started_at: timestamp
    }
    config: ContainerConfig
    network_config: NetworkConfig
    resources: ResourceLimits
    mounts: []Mount
    environment: {key: value}
    labels: {key: value}
}
```

### Image
```go
Image {
    id: "unique-id"
    repo_tags: ["name:tag"]
    created: timestamp
    architecture: "amd64"
    os: "linux"
    size: bytes
    config: {
        user: "root"
        env: []string
        cmd: []string
        working_dir: "/app"
    }
    layers: ["layer-id-1", "layer-id-2"]
}
```

### Network
```go
Network {
    id: "unique-id"
    name: "network-name"
    driver: "bridge" | "host" | "overlay" | "none"
    ipam: {
        driver: "default"
        config: [{
            subnet: "172.17.0.0/16"
            gateway: "172.17.0.1"
        }]
    }
    containers: {container_id: endpoint_config}
}
```

### Volume
```go
Volume {
    name: "volume-name"
    driver: "local"
    mountpoint: "/var/lib/realityos/volumes/name"
    scope: "local" | "global"
    labels: {key: value}
}
```

## Running the Demo

```bash
cd examples
go run layer5_demo.go
```

The demo demonstrates:
1. Image management (pull, build, list)
2. Network creation and management
3. Volume creation
4. Container lifecycle (create, start, pause, unpause, stop)
5. Container operations (stats, logs, exec)
6. Multiple container orchestration
7. Network connectivity
8. Resource limits
9. System statistics
10. Cleanup and pruning

## Integration with Previous Layers

### With Layer 1 (Adapters)
- Containers can be managed via adapters
- Container metrics reported through adapters

### With Layer 2 (Discovery & Registration)
- Containers registered as entities
- Container metadata synced

### With Layer 3 (Telemetry)
- Container metrics collected
- Container logs aggregated
- Container events published

### With Layer 4 (State Management)
- Container desired vs actual state
- Container drift detection
- Container reconciliation

Example integration:
```go
// Layer 5: Create container
container, _ := layer5Manager.RunContainer(ctx, config, "app-1")

// Layer 2: Register as entity
entity := &layer2.RegistrationRequest{
    ID:   container.ID,
    Type: layer2.EntityTypeServer,
    Name: container.Name,
}
layer2Manager.RegisterEntity(ctx, entity)

// Layer 3: Collect metrics
stats, _ := layer5Manager.GetContainerStats(ctx, container.ID)
layer3Manager.CollectMetric(&layer3.Metric{
    Name:     "container_cpu_usage",
    EntityID: container.ID,
    Value:    float64(stats.CPU.TotalUsage),
})

// Layer 4: Track state
state := &layer4.EntityState{
    EntityID: container.ID,
    Status:   layer4.StatusOnline,
    ActualState: map[string]interface{}{
        "status": container.Status,
        "pid":    container.State.Pid,
    },
}
layer4Manager.SetEntityState(ctx, state)
```

## What's Next (Future Versions)

### Advanced Features
- Real namespace isolation (PID, NET, IPC, UTS, MNT)
- Cgroups v2 support
- Seccomp profiles
- AppArmor/SELinux integration
- User namespaces
- Rootless containers

### Orchestration
- Container scheduling
- Service discovery
- Load balancing
- Health checks
- Auto-restart policies
- Rolling updates

### Storage
- Overlay filesystem
- Copy-on-write layers
- Storage drivers (overlay2, btrfs, zfs)
- Volume plugins
- Snapshot support

### Networking
- CNI plugin support
- Service mesh integration
- Network policies
- DNS resolution
- Load balancer integration

### Registry
- Private registry support
- Image signing and verification
- Image scanning
- Multi-architecture images

## Design Decisions

1. **Docker-Compatible API**: Familiar interface for users
2. **Modular Design**: Separate managers for different concerns
3. **Thread-Safe**: All operations use proper locking
4. **Context-Aware**: All operations support cancellation
5. **Resource Limits**: Built-in resource management
6. **Simulated Implementation**: V1 simulates actual container operations

## Implementation Notes

This is a **simulated** container runtime for demonstration purposes. A production implementation would require:

1. **Linux Namespaces**: PID, NET, IPC, UTS, MNT, USER
2. **Cgroups**: Resource limits and accounting
3. **Overlay Filesystem**: Layer-based storage
4. **Network Stack**: Virtual ethernet, bridges, iptables
5. **Process Management**: Fork/exec, signal handling
6. **Security**: Seccomp, capabilities, AppArmor/SELinux

## API Reference

### Manager
- `Start(ctx)` - Start manager
- `CreateContainer(ctx, config, name)` - Create container
- `StartContainer(ctx, containerID)` - Start container
- `StopContainer(ctx, containerID, timeout)` - Stop container
- `RemoveContainer(ctx, containerID, force)` - Remove container
- `PauseContainer(ctx, containerID)` - Pause container
- `UnpauseContainer(ctx, containerID)` - Unpause container
- `RestartContainer(ctx, containerID, timeout)` - Restart container
- `ListContainers(ctx, all)` - List containers
- `GetContainer(ctx, containerID)` - Get container
- `RunContainer(ctx, config, name)` - Create and start container
- `GetContainerStats(ctx, containerID)` - Get statistics
- `GetContainerLogs(ctx, containerID, config)` - Get logs
- `ExecInContainer(ctx, config)` - Execute command

### Image Manager
- `PullImage(ctx, imageName)` - Pull image
- `BuildImage(ctx, dockerfile, tags)` - Build image
- `GetImage(ctx, imageID)` - Get image
- `ListImages(ctx)` - List images
- `RemoveImage(ctx, imageID, force)` - Remove image
- `TagImage(ctx, imageID, tag)` - Tag image
- `PushImage(ctx, imageID, registry)` - Push image

### Network Manager
- `CreateNetwork(ctx, name, driver, internal, labels)` - Create network
- `GetNetwork(ctx, networkID)` - Get network
- `ListNetworks(ctx)` - List networks
- `RemoveNetwork(ctx, networkID)` - Remove network
- `ConnectContainer(ctx, networkID, containerID, config)` - Connect container
- `DisconnectContainer(ctx, networkID, containerID, force)` - Disconnect container

### Volume Manager
- `CreateVolume(ctx, name, driver, labels)` - Create volume
- `GetVolume(ctx, name)` - Get volume
- `ListVolumes(ctx)` - List volumes
- `RemoveVolume(ctx, name, force)` - Remove volume

## Limitations (V1)

- Simulated container execution (no real isolation)
- No actual namespace/cgroup implementation
- No real filesystem layers
- No registry integration
- No image building from actual Dockerfiles
- In-memory storage only
- No persistence across restarts
- No security features (seccomp, capabilities)

## Performance Considerations

- In-memory storage is fast but not persistent
- No actual process isolation overhead
- Simulated statistics
- No real network stack overhead
- Production implementation would have significant overhead from isolation

## Security Notes

A production container runtime must implement:
- Namespace isolation
- Cgroup resource limits
- Seccomp system call filtering
- Capability dropping
- AppArmor/SELinux profiles
- User namespaces for rootless
- Image signing and verification
- Network policies
