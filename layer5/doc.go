// Package layer5 provides the container runtime for AIZO.
// It manages the full lifecycle of isolated containers using Linux
// namespaces via WSL2 (on Windows) or directly on Linux.
// Each container gets its own filesystem (rootfs), process namespace,
// and optional persistent volumes. Networking between containers
// is handled via veth pairs or socat port forwarding.
package layer5
