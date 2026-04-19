# AIZO

A universal infrastructure control plane. Connect to anything, observe everything, heal automatically.

AIZO manages heterogeneous infrastructure — servers, containers, APIs, databases, IoT devices — through a unified seven-layer architecture. It detects failures, proposes fixes, learns from outcomes, and gets smarter over time. No cloud account required. No vendor lock-in.

---

## Architecture

```
┌─────────────────────────────────────────────────────┐
│  Layer 7: CLI / TUI          aizo <command>          │
├─────────────────────────────────────────────────────┤
│  Layer 6: Intelligence       Self-learning rules     │
├─────────────────────────────────────────────────────┤
│  Layer 5: Control Plane      Container runtime       │
├─────────────────────────────────────────────────────┤
│  Layer 4: State Management   Drift detection         │
├─────────────────────────────────────────────────────┤
│  Layer 3: Telemetry          Metrics, logs, traces   │
├─────────────────────────────────────────────────────┤
│  Layer 2: Discovery          Entity catalog          │
├─────────────────────────────────────────────────────┤
│  Layer 1: Adapters           HTTP, SSH, gRPC, MQTT   │
└─────────────────────────────────────────────────────┘
```

Each layer has a clean interface and can be used independently.

---

## Quick Start

**Requirements:** Go 1.22+, Windows with WSL2 (Ubuntu) or Linux

```bash
git clone https://github.com/yourname/aizo
cd aizo
go mod tidy
go build -o aizo ./cmd/aizo/
```

**Launch the TUI:**
```bash
./aizo tui
```

**Or use the CLI:**
```bash
./aizo --help
```

All data is stored in `~/.aizo/aizo.db` (SQLite). No external services required.

---

## Layers

### Layer 1 — Adapters
Connect to any system via HTTP/REST, SSH, gRPC, or MQTT. Each adapter handles connection, health checks, state reads, and command execution.

```bash
./aizo adapter add --id web-1 --type http --target http://localhost:8080
./aizo adapter health web-1
```

### Layer 2 — Discovery & Registration
Catalog every entity in your infrastructure. Entities can be servers, APIs, databases, containers, pipelines, or devices.

```bash
./aizo entity register --id api-1 --name "Payment API" --type api --endpoint http://pay.internal
./aizo entity list
./aizo entity inspect api-1
```

### Layer 3 — Telemetry
Collect and query metrics, logs, and distributed traces. All data persists to SQLite.

```bash
./aizo metrics query --entity api-1 --last 1h
./aizo logs search "error" --entity api-1
```

### Layer 4 — State Management
Track desired vs actual state. Detect drift. Reconcile automatically.

```bash
./aizo state get api-1
./aizo state drift api-1
```

### Layer 5 — Container Runtime
Real isolated containers using Linux namespaces (via WSL2 on Windows). Each container has its own filesystem stored on disk.

```bash
./aizo container create my-app
./aizo container start my-app
./aizo container list
# In TUI: navigate to Containers tab, press Enter to open a shell
```

Containers are stored in `~/realityos/` inside WSL2.

### Layer 6 — Intelligence
A self-learning rule engine that detects issues and proposes fixes. Rules auto-tune their thresholds based on historical success rates. New rules are suggested by mining incident patterns.

```bash
./aizo rules list                        # see all rules + success rates
./aizo rules tune                        # trigger threshold auto-tuning
./aizo rules suggest                     # see pattern-mined suggestions
./aizo proposals list                    # see pending proposals
./aizo proposals approve <id>            # approve and execute
./aizo incidents list                    # full incident history
./aizo incidents stats                   # success rate per rule
```

**Default rules** (built-in, auto-loaded):
| Rule | Condition | Action | Auto-approve |
|------|-----------|--------|-------------|
| Memory Cleanup | memory > 80% | cleanup | yes |
| Memory Restart | memory > 95% | restart | no |
| CPU Investigate | cpu > 90% | investigate | yes |
| Disk Cleanup | disk > 85% | cleanup | yes |
| Container Crash | crash event | restart | no |
| Health Check Fail | health fail event | restart | no |
| Service Down | service down event | restart | no |
| Failed Containers | failed_containers > 0 | investigate | yes |

**Custom rules** — add YAML files to `~/.aizo/rules/`:
```yaml
- id: my-custom-rule
  name: High Error Rate
  description: Restart when error rate spikes
  conditions:
    - metric: error_rate
      operator: ">"
      value: 50
  action:
    type: restart
    risk: medium
    reversible: true
    auto_approve: false
  priority: 70
  enabled: true
```

---

## Policy Engine

Control who can do what with YAML policy files in `~/.aizo/policies/`:

```yaml
- id: ops-policy
  name: Operators Policy
  rules:
    - actions: ["container.start", "container.stop"]
      resources: ["*"]
      actors: ["operator"]
      effect: allow
  effect: deny
  priority: 100
  enabled: true
```

```bash
./aizo policy list
./aizo policy evaluate --action container.start --actor operator --resource my-app
```

---

## Multi-Tenancy

```bash
./aizo tenant create staging "Staging Environment"
./aizo tenant switch staging
./aizo tenant list
```

All data is namespaced per tenant in the database.

---

## Audit Trail

Every action is logged:

```bash
./aizo audit list --last 24h
./aizo audit list --actor cli --layer layer6
```

---

## TUI

```bash
./aizo tui
```

Navigate with `Ctrl+N` / `Ctrl+P` (or arrow keys). Tabs:
1. **Dashboard** — system overview
2. **Containers** — manage containers (`c` create, `Enter` shell, `s` start, `x` stop, `d` delete)
3. **Entities** — browse registered entities
4. **Metrics** — telemetry data
5. **Logs** — log search
6. **AI Chat** — rule engine interaction
7. **Audit** — audit trail

---

## Examples

Each layer has a standalone demo:

```bash
cd examples
go run layer1_demo.go       # adapter connectivity
go run layer2_demo.go       # entity discovery
go run layer3_demo.go       # telemetry collection
go run layer4_demo.go       # state management
go run layer5_demo.go       # container runtime
go run layer6_demo.go       # rule engine + learning
go run integration_demo.go  # layer 1+2 bridge
go run wsl2_container_demo.go  # WSL2 containers
```

---

## Project Structure

```
aizo/
├── cmd/aizo/          # CLI entry point
│   └── cmd/           # Cobra commands
├── tui/               # Bubble Tea TUI
├── layer1/            # Adapter layer
├── layer2/            # Discovery & registration
├── layer3/            # Telemetry & observability
├── layer4/            # State management
├── layer5/            # Container runtime
├── layer6/            # Intelligence (rule engine)
├── integration/       # Layer bridges
├── storage/           # SQLite persistence
├── policy/            # Policy engine
├── tenant/            # Multi-tenancy
└── examples/          # Per-layer demos
```

---

## Roadmap

- [ ] Multi-node gossip protocol (fleet management)
- [ ] Docker runtime backend (Linux production)
- [ ] Container networking (veth pairs)
- [ ] Image registry support
- [ ] Prometheus metrics endpoint
- [ ] REST API
- [ ] Web dashboard

---

## License

MIT
