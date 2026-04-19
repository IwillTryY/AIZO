# AIZO

## Infrastructure That Heals Itself

---

## The Problem

Every company running infrastructure deals with the same cycle:

Something breaks. An alert fires. An engineer wakes up, logs in, reads dashboards, guesses the cause, tries a fix, waits, tries another fix, and eventually resolves it. The next time the same thing breaks, the same cycle repeats.

This costs real money. Downtime costs revenue. On-call burns out engineers. Manual remediation is slow, error-prone, and doesn't scale.

Monitoring tools tell you something is wrong. Orchestrators restart things blindly. Neither understands what actually caused the problem or whether the fix worked.

---

## What AIZO Does

AIZO is a self-healing infrastructure control plane. It connects to your systems, watches them continuously, detects failures, diagnoses root causes, and fixes problems autonomously.

It works in three stages:

**Detect.** AIZO reads real metrics from every machine — CPU, memory, disk, network, process state. It flags anomalies based on configurable rules with severity levels: low, medium, high, critical.

**Diagnose.** When an anomaly is detected, AIZO doesn't just alert. It scans running processes, identifies which specific process is consuming resources, checks whether the cause is internal or external, and assigns a confidence score to its diagnosis.

**Enforce.** Based on severity and diagnosis, AIZO takes graduated action:
- Low severity: log and monitor
- Medium: soft remediation (free memory, clean temp files)
- High: targeted fix (kill the offending process, restart the service)
- Critical: emergency response (kill all stress, free all resources, clean everything)

Every action is logged to a tamper-evident audit trail. Every rule learns from its outcomes and auto-tunes its thresholds over time.

---

## How It Works

AIZO is built as seven layers, each with a clean interface:

```
Layer 7: CLI + Terminal UI
Layer 6: Self-Learning Rule Engine
Layer 5: Container Runtime (Linux namespaces)
Layer 4: State Management + Drift Detection
Layer 3: Telemetry (Metrics, Logs, Traces)
Layer 2: Entity Discovery + Registration
Layer 1: Universal Adapters (HTTP, SSH, gRPC, MQTT, Mesh)
```

**Layer 1** connects to anything. HTTP APIs, SSH servers, gRPC services, MQTT brokers, or a mesh of machines across Windows, Linux, and macOS. Each adapter handles connection management, health checks, and command execution.

**Layer 2** discovers and catalogs every entity in your infrastructure — servers, containers, APIs, databases, IoT devices. It maps relationships between them and detects capabilities automatically.

**Layer 3** collects metrics, logs, distributed traces, and events from all entities. Everything persists to SQLite. No external database required.

**Layer 4** tracks desired vs actual state for every entity. When state drifts, it detects the difference and can reconcile automatically. Supports point-in-time snapshots and time-travel queries.

**Layer 5** runs isolated containers using real Linux namespaces. Each container has its own filesystem, process tree, and network stack. Containers are stored persistently on disk.

**Layer 6** is the brain. A rule engine evaluates conditions against live metrics and fires proposals. Rules auto-tune: if a rule's success rate drops below 70%, its threshold tightens. If it succeeds consistently, it relaxes. New rules are suggested by mining incident patterns. No external AI APIs. No API costs. Everything runs locally.

**Layer 7** provides a full CLI and terminal UI. One command to install. Natural syntax to operate.

---

## Why AIZO Instead of What Exists

**vs Datadog / New Relic / Grafana**
These are monitoring tools. They tell you something is wrong. AIZO tells you what caused it and fixes it.

**vs Kubernetes / Nomad**
These are orchestrators. They restart containers when they crash. AIZO diagnoses why they crashed, fixes the root cause, and learns to prevent it next time.

**vs PagerDuty / OpsGenie**
These are alert routers. They wake up a human. AIZO is the human — it investigates, decides, and acts.

**vs Ansible / Terraform**
These are configuration tools. They set up infrastructure. AIZO keeps it running after setup.

AIZO is not a replacement for any of these. It sits on top of your existing stack and adds the layer that's missing: autonomous diagnosis and remediation.

---

## The Self-Learning Loop

Every time AIZO takes an action, it records the outcome:
- Did the fix work?
- How long did recovery take?
- Did the same issue recur?

This feedback drives three learning mechanisms:

**Threshold tuning.** If a rule fires at 80% memory and the fix fails (too late), the threshold automatically tightens to 75%. If it consistently succeeds, it relaxes to 82%. No manual tuning required.

**Auto-approve promotion.** New rules require human approval. After 10 consecutive successful executions, the rule is promoted to auto-approve. Trust is earned, not configured.

**Pattern mining.** AIZO scans incident history for recurring event types not covered by existing rules. It surfaces suggested rules with confidence scores. You approve the ones that make sense.

The result: AIZO gets smarter with every incident. The longer it runs, the fewer problems reach a human.

---

## Cross-Platform Mesh

AIZO includes a mesh networking layer that connects Windows, Linux, and macOS machines into a single virtual cluster.

Each machine runs a mesh node. Nodes discover each other automatically via multicast UDP on the local network, or connect via configured peer addresses across networks.

The mesh uses a custom binary protocol over TCP with:
- Protocol versioning (peers reject incompatible versions)
- mTLS encryption (mutual certificate authentication)
- HMAC message signing (tamper detection)
- Circuit breakers (prevent cascading failures)
- Idempotency guarantees (no duplicate command execution)

For clusters over 10 nodes, AIZO automatically switches from full-mesh to hierarchical topology with coordinator election, reducing connection overhead from O(n²) to O(n).

---

## Security

AIZO was designed for environments where security matters:

- **mTLS everywhere.** All mesh connections use mutual TLS 1.3 with auto-generated certificates.
- **HMAC-signed messages.** Every mesh message carries a cryptographic signature. Unsigned or tampered messages are rejected.
- **Role-based access control.** Nodes are assigned roles (admin, operator, reader) that determine what actions they can perform.
- **Tamper-evident audit trail.** Every action is logged with a SHA-256 checksum chained to the previous entry. Any modification breaks the chain and is detectable.
- **Policy engine.** YAML-based policies control who can do what to which resources. Deny by default. Rate limiting per actor.
- **Multi-tenancy.** All data is namespaced per tenant. Teams and environments are fully isolated at the storage layer.

---

## Durability

AIZO stores everything in a single SQLite database file. No external database server required.

An append-only event log records every state change, command, proposal, and mesh message with chained SHA-256 checksums. This provides:

- **Full audit trail** — every action is traceable
- **Event replay** — reconstruct system state at any point in time
- **Integrity verification** — detect any tampering in the log
- **Crash recovery** — WAL mode ensures consistency after unexpected shutdown

---

## Getting Started

```bash
# Install
git clone <repo>
cd aizo
go build -o aizo ./cmd/aizo/

# Set up everything (one command)
./aizo install

# Create containers
./aizo create container 3 web api worker

# Start them
./aizo start container web

# Monitor
./aizo list containers
./aizo list rules
./aizo list proposals

# Terminal UI
./aizo tui
```

---

## What It Costs

AIZO has zero external dependencies at runtime. No AI API calls. No cloud services. No license servers. It runs on a single binary with a single SQLite file.

The rule engine, learning loop, and diagnostic pipeline all run locally. There are no per-query costs, no per-node fees, and no usage-based pricing from third parties.

---

## Where This Goes

AIZO is the control plane for infrastructure that manages itself. The immediate value is fewer pages, faster recovery, and less manual toil. The long-term trajectory is infrastructure that grows, heals, learns, and optimizes without human intervention.

Every company running more than a handful of servers is paying engineers to do what software should do. AIZO is that software.

---

*Built with Go. Runs anywhere. Heals everything.*
