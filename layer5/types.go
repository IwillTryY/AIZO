package layer5

import (
	"time"
)

// Container represents an isolated execution environment
type Container struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Image         string                 `json:"image"`
	Status        ContainerStatus        `json:"status"`
	State         *ContainerState        `json:"state"`
	Config        *ContainerConfig       `json:"config"`
	NetworkConfig *NetworkConfig         `json:"network_config"`
	Resources     *ResourceLimits        `json:"resources"`
	Mounts        []Mount                `json:"mounts"`
	Environment   map[string]string      `json:"environment"`
	Labels        map[string]string      `json:"labels"`
	CreatedAt     time.Time              `json:"created_at"`
	StartedAt     time.Time              `json:"started_at,omitempty"`
	FinishedAt    time.Time              `json:"finished_at,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ContainerStatus represents the container lifecycle state
type ContainerStatus string

const (
	StatusCreated    ContainerStatus = "created"
	StatusRunning    ContainerStatus = "running"
	StatusPaused     ContainerStatus = "paused"
	StatusRestarting ContainerStatus = "restarting"
	StatusExited     ContainerStatus = "exited"
	StatusDead       ContainerStatus = "dead"
)

// ContainerState represents the runtime state
type ContainerState struct {
	Status     ContainerStatus `json:"status"`
	Running    bool            `json:"running"`
	Paused     bool            `json:"paused"`
	Restarting bool            `json:"restarting"`
	OOMKilled  bool            `json:"oom_killed"`
	Dead       bool            `json:"dead"`
	Pid        int             `json:"pid"`
	ExitCode   int             `json:"exit_code"`
	Error      string          `json:"error,omitempty"`
	StartedAt  time.Time       `json:"started_at"`
	FinishedAt time.Time       `json:"finished_at,omitempty"`
}

// ContainerConfig holds container configuration
type ContainerConfig struct {
	Hostname     string            `json:"hostname"`
	User         string            `json:"user"`
	WorkingDir   string            `json:"working_dir"`
	Entrypoint   []string          `json:"entrypoint"`
	Cmd          []string          `json:"cmd"`
	Env          []string          `json:"env"`
	Labels       map[string]string `json:"labels"`
	StopSignal   string            `json:"stop_signal"`
	StopTimeout  int               `json:"stop_timeout"`
	AttachStdin  bool              `json:"attach_stdin"`
	AttachStdout bool              `json:"attach_stdout"`
	AttachStderr bool              `json:"attach_stderr"`
	Tty          bool              `json:"tty"`
	OpenStdin    bool              `json:"open_stdin"`
}

// NetworkConfig holds network configuration
type NetworkConfig struct {
	NetworkMode  string            `json:"network_mode"` // bridge, host, none, container:<name>
	Hostname     string            `json:"hostname"`
	IPAddress    string            `json:"ip_address"`
	Gateway      string            `json:"gateway"`
	MacAddress   string            `json:"mac_address"`
	Ports        []PortMapping     `json:"ports"`
	DNS          []string          `json:"dns"`
	DNSSearch    []string          `json:"dns_search"`
	ExtraHosts   []string          `json:"extra_hosts"`
	Links        []string          `json:"links"`
	NetworkID    string            `json:"network_id"`
}

// PortMapping represents a port mapping
type PortMapping struct {
	HostIP        string `json:"host_ip"`
	HostPort      int    `json:"host_port"`
	ContainerPort int    `json:"container_port"`
	Protocol      string `json:"protocol"` // tcp, udp
}

// ResourceLimits defines resource constraints
type ResourceLimits struct {
	CPUShares          int64   `json:"cpu_shares"`
	CPUQuota           int64   `json:"cpu_quota"`
	CPUPeriod          int64   `json:"cpu_period"`
	CPUSetCPUs         string  `json:"cpuset_cpus"`
	CPUSetMems         string  `json:"cpuset_mems"`
	Memory             int64   `json:"memory"`              // bytes
	MemoryReservation  int64   `json:"memory_reservation"`  // bytes
	MemorySwap         int64   `json:"memory_swap"`         // bytes
	MemorySwappiness   int64   `json:"memory_swappiness"`
	KernelMemory       int64   `json:"kernel_memory"`       // bytes
	OomKillDisable     bool    `json:"oom_kill_disable"`
	PidsLimit          int64   `json:"pids_limit"`
	BlkioWeight        uint16  `json:"blkio_weight"`
	BlkioWeightDevice  []WeightDevice `json:"blkio_weight_device"`
	BlkioDeviceReadBps []ThrottleDevice `json:"blkio_device_read_bps"`
	BlkioDeviceWriteBps []ThrottleDevice `json:"blkio_device_write_bps"`
}

// WeightDevice represents a device weight
type WeightDevice struct {
	Path   string `json:"path"`
	Weight uint16 `json:"weight"`
}

// ThrottleDevice represents a device throttle
type ThrottleDevice struct {
	Path string `json:"path"`
	Rate uint64 `json:"rate"`
}

// Mount represents a filesystem mount
type Mount struct {
	Type        MountType `json:"type"`
	Source      string    `json:"source"`
	Target      string    `json:"target"`
	ReadOnly    bool      `json:"read_only"`
	Consistency string    `json:"consistency"` // default, consistent, cached, delegated
	BindOptions *BindOptions `json:"bind_options,omitempty"`
	VolumeOptions *VolumeOptions `json:"volume_options,omitempty"`
}

// MountType represents the type of mount
type MountType string

const (
	MountTypeBind   MountType = "bind"
	MountTypeVolume MountType = "volume"
	MountTypeTmpfs  MountType = "tmpfs"
)

// BindOptions represents bind mount options
type BindOptions struct {
	Propagation string `json:"propagation"` // private, rprivate, shared, rshared, slave, rslave
}

// VolumeOptions represents volume mount options
type VolumeOptions struct {
	NoCopy       bool              `json:"no_copy"`
	Labels       map[string]string `json:"labels"`
	DriverConfig *VolumeDriverConfig `json:"driver_config,omitempty"`
}

// VolumeDriverConfig represents volume driver configuration
type VolumeDriverConfig struct {
	Name    string            `json:"name"`
	Options map[string]string `json:"options"`
}

// Image represents a container image
type Image struct {
	ID          string            `json:"id"`
	RepoTags    []string          `json:"repo_tags"`
	RepoDigests []string          `json:"repo_digests"`
	Parent      string            `json:"parent"`
	Comment     string            `json:"comment"`
	Created     time.Time         `json:"created"`
	Author      string            `json:"author"`
	Architecture string           `json:"architecture"`
	OS          string            `json:"os"`
	Size        int64             `json:"size"`
	VirtualSize int64             `json:"virtual_size"`
	Labels      map[string]string `json:"labels"`
	Config      *ImageConfig      `json:"config"`
	Layers      []string          `json:"layers"`
}

// ImageConfig represents image configuration
type ImageConfig struct {
	User         string            `json:"user"`
	ExposedPorts map[string]struct{} `json:"exposed_ports"`
	Env          []string          `json:"env"`
	Entrypoint   []string          `json:"entrypoint"`
	Cmd          []string          `json:"cmd"`
	Volumes      map[string]struct{} `json:"volumes"`
	WorkingDir   string            `json:"working_dir"`
	Labels       map[string]string `json:"labels"`
	StopSignal   string            `json:"stop_signal"`
}

// Volume represents a data volume
type Volume struct {
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	Mountpoint string            `json:"mountpoint"`
	Labels     map[string]string `json:"labels"`
	Scope      string            `json:"scope"` // local, global
	Options    map[string]string `json:"options"`
	CreatedAt  time.Time         `json:"created_at"`
}

// Network represents a container network
type Network struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Driver     string                 `json:"driver"` // bridge, host, overlay, macvlan, none
	Scope      string                 `json:"scope"`  // local, global, swarm
	Internal   bool                   `json:"internal"`
	Attachable bool                   `json:"attachable"`
	IPAM       *IPAMConfig            `json:"ipam"`
	Containers map[string]*EndpointConfig `json:"containers"`
	Options    map[string]string      `json:"options"`
	Labels     map[string]string      `json:"labels"`
	CreatedAt  time.Time              `json:"created_at"`
}

// IPAMConfig represents IP address management configuration
type IPAMConfig struct {
	Driver  string       `json:"driver"`
	Config  []IPAMPool   `json:"config"`
	Options map[string]string `json:"options"`
}

// IPAMPool represents an IP address pool
type IPAMPool struct {
	Subnet     string `json:"subnet"`
	IPRange    string `json:"ip_range"`
	Gateway    string `json:"gateway"`
	AuxAddress map[string]string `json:"aux_address"`
}

// EndpointConfig represents network endpoint configuration
type EndpointConfig struct {
	EndpointID          string   `json:"endpoint_id"`
	Gateway             string   `json:"gateway"`
	IPAddress           string   `json:"ip_address"`
	IPPrefixLen         int      `json:"ip_prefix_len"`
	IPv6Gateway         string   `json:"ipv6_gateway"`
	GlobalIPv6Address   string   `json:"global_ipv6_address"`
	GlobalIPv6PrefixLen int      `json:"global_ipv6_prefix_len"`
	MacAddress          string   `json:"mac_address"`
	Aliases             []string `json:"aliases"`
}

// ContainerStats represents container resource usage statistics
type ContainerStats struct {
	ContainerID string          `json:"container_id"`
	Timestamp   time.Time       `json:"timestamp"`
	CPU         *CPUStats       `json:"cpu"`
	Memory      *MemoryStats    `json:"memory"`
	Network     *NetworkStats   `json:"network"`
	BlockIO     *BlockIOStats   `json:"block_io"`
	PIDs        *PIDsStats      `json:"pids"`
}

// CPUStats represents CPU usage statistics
type CPUStats struct {
	TotalUsage        uint64 `json:"total_usage"`
	UsageInKernelmode uint64 `json:"usage_in_kernelmode"`
	UsageInUsermode   uint64 `json:"usage_in_usermode"`
	SystemCPUUsage    uint64 `json:"system_cpu_usage"`
	OnlineCPUs        uint32 `json:"online_cpus"`
	ThrottlingData    *ThrottlingData `json:"throttling_data"`
}

// ThrottlingData represents CPU throttling data
type ThrottlingData struct {
	Periods          uint64 `json:"periods"`
	ThrottledPeriods uint64 `json:"throttled_periods"`
	ThrottledTime    uint64 `json:"throttled_time"`
}

// MemoryStats represents memory usage statistics
type MemoryStats struct {
	Usage    uint64 `json:"usage"`
	MaxUsage uint64 `json:"max_usage"`
	Limit    uint64 `json:"limit"`
	Cache    uint64 `json:"cache"`
	RSS      uint64 `json:"rss"`
	Swap     uint64 `json:"swap"`
}

// NetworkStats represents network usage statistics
type NetworkStats struct {
	RxBytes   uint64 `json:"rx_bytes"`
	RxPackets uint64 `json:"rx_packets"`
	RxErrors  uint64 `json:"rx_errors"`
	RxDropped uint64 `json:"rx_dropped"`
	TxBytes   uint64 `json:"tx_bytes"`
	TxPackets uint64 `json:"tx_packets"`
	TxErrors  uint64 `json:"tx_errors"`
	TxDropped uint64 `json:"tx_dropped"`
}

// BlockIOStats represents block I/O statistics
type BlockIOStats struct {
	ReadBytes  uint64 `json:"read_bytes"`
	WriteBytes uint64 `json:"write_bytes"`
	ReadOps    uint64 `json:"read_ops"`
	WriteOps   uint64 `json:"write_ops"`
}

// PIDsStats represents process ID statistics
type PIDsStats struct {
	Current uint64 `json:"current"`
	Limit   uint64 `json:"limit"`
}

// ExecConfig represents configuration for executing commands in a container
type ExecConfig struct {
	ID           string   `json:"id"`
	ContainerID  string   `json:"container_id"`
	User         string   `json:"user"`
	Privileged   bool     `json:"privileged"`
	Tty          bool     `json:"tty"`
	AttachStdin  bool     `json:"attach_stdin"`
	AttachStdout bool     `json:"attach_stdout"`
	AttachStderr bool     `json:"attach_stderr"`
	Detach       bool     `json:"detach"`
	DetachKeys   string   `json:"detach_keys"`
	Env          []string `json:"env"`
	Cmd          []string `json:"cmd"`
	WorkingDir   string   `json:"working_dir"`
}

// ContainerLogs represents container log configuration
type ContainerLogs struct {
	Stdout     bool      `json:"stdout"`
	Stderr     bool      `json:"stderr"`
	Follow     bool      `json:"follow"`
	Since      time.Time `json:"since"`
	Until      time.Time `json:"until"`
	Timestamps bool      `json:"timestamps"`
	Tail       string    `json:"tail"` // "all" or number
}
