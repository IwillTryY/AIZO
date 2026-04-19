package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/realityos/aizo/layer6"
	"github.com/realityos/aizo/storage"
)

// ═══════════════════════════════════════════
// METRICS (PowerShell, real Windows metrics)
// ═══════════════════════════════════════════

func ps(script string) string {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	out, _ := cmd.Output()
	return strings.TrimSpace(string(out))
}

func readCPU() float64 {
	v, _ := strconv.ParseFloat(ps(`(Get-CimInstance Win32_Processor).LoadPercentage`), 64)
	return v
}

func readMemory() float64 {
	out := ps(`$os = Get-CimInstance Win32_OperatingSystem; [math]::Round(100 - ($os.FreePhysicalMemory / $os.TotalVisibleMemorySize * 100), 1)`)
	v, _ := strconv.ParseFloat(out, 64)
	return v
}

func readDisk() float64 {
	out := ps(`$d = Get-CimInstance Win32_LogicalDisk -Filter "DeviceID='C:'"; [math]::Round(100 - ($d.FreeSpace / $d.Size * 100), 1)`)
	v, _ := strconv.ParseFloat(out, 64)
	return v
}

type ProcessInfo struct {
	PID     int
	Name    string
	CPUTime float64
	MemMB   float64
}

func findTopProcesses(sortBy string, n int) []ProcessInfo {
	var script string
	if sortBy == "cpu" {
		script = fmt.Sprintf(`Get-Process | Sort-Object CPU -Descending | Select-Object -First %d | ForEach-Object { "$($_.Id)|$($_.ProcessName)|$([math]::Round($_.CPU,1))|$([math]::Round($_.WorkingSet64/1MB,1))" }`, n)
	} else {
		script = fmt.Sprintf(`Get-Process | Sort-Object WorkingSet64 -Descending | Select-Object -First %d | ForEach-Object { "$($_.Id)|$($_.ProcessName)|$([math]::Round($_.CPU,1))|$([math]::Round($_.WorkingSet64/1MB,1))" }`, n)
	}
	out := ps(script)
	procs := make([]ProcessInfo, 0)
	for _, line := range strings.Split(out, "\n") {
		parts := strings.Split(strings.TrimSpace(line), "|")
		if len(parts) < 4 {
			continue
		}
		pid, _ := strconv.Atoi(parts[0])
		cpuTime, _ := strconv.ParseFloat(parts[2], 64)
		memMB, _ := strconv.ParseFloat(parts[3], 64)
		procs = append(procs, ProcessInfo{pid, parts[1], cpuTime, memMB})
	}
	return procs
}

// ═══════════════════════════════════════════
// STRESS INJECTORS
// ═══════════════════════════════════════════

var (
	cpuBurnActive int32
	memHeld       [][]byte
	memMu         sync.Mutex
	diskFile      string
)

func startCPUBurn(stop chan struct{}) {
	atomic.StoreInt32(&cpuBurnActive, 1)
	for i := 0; i < runtime.NumCPU(); i++ {
		go func() {
			for atomic.LoadInt32(&cpuBurnActive) == 1 {
				select {
				case <-stop:
					return
				default:
					x := 1.1
					for j := 0; j < 50000; j++ {
						x = math.Sin(x) * math.Cos(x)
					}
					_ = x
				}
			}
		}()
	}
}

func stopCPUBurn()    { atomic.StoreInt32(&cpuBurnActive, 0); time.Sleep(200 * time.Millisecond) }
func isCPUBurning() bool { return atomic.LoadInt32(&cpuBurnActive) == 1 }

func allocateMemory(mb int) {
	memMu.Lock()
	defer memMu.Unlock()
	for i := 0; i < mb; i++ {
		chunk := make([]byte, 1024*1024)
		for j := 0; j < len(chunk); j += 4096 {
			chunk[j] = byte(i)
		}
		memHeld = append(memHeld, chunk)
	}
}

func freeMemory() int {
	memMu.Lock()
	defer memMu.Unlock()
	n := len(memHeld)
	memHeld = nil
	runtime.GC()
	return n
}

func currentMemHeldMB() int {
	memMu.Lock()
	defer memMu.Unlock()
	return len(memHeld)
}

func fillDisk(mb int) {
	diskFile = os.TempDir() + "\\aizo_stress.bin"
	f, _ := os.Create(diskFile)
	if f == nil {
		return
	}
	chunk := make([]byte, 1024*1024)
	for i := 0; i < mb; i++ {
		f.Write(chunk)
	}
	f.Close()
}

func cleanDisk() {
	if diskFile != "" {
		os.Remove(diskFile)
		diskFile = ""
	}
}

// ═══════════════════════════════════════════
// DETECTION → DIAGNOSIS → ENFORCEMENT PIPELINE
// ═══════════════════════════════════════════

// Severity levels for escalation
type Severity int

const (
	SevNone     Severity = 0
	SevLow      Severity = 1
	SevMedium   Severity = 2
	SevHigh     Severity = 3
	SevCritical Severity = 4
)

func (s Severity) String() string {
	switch s {
	case SevLow:
		return "LOW"
	case SevMedium:
		return "MEDIUM"
	case SevHigh:
		return "HIGH"
	case SevCritical:
		return "CRITICAL"
	}
	return "NONE"
}

// Issue represents a detected problem
type Issue struct {
	Resource  string // "cpu", "memory", "disk"
	Value     float64
	Severity  Severity
	Cause     string
	CausePID  int
	Internal  bool // caused by our own stress injectors
}

// Diagnosis is the result of analyzing an issue
type Diagnosis struct {
	Issue       Issue
	RootCause   string
	Confidence  float64 // 0-1
	Recommended string  // "monitor", "throttle", "kill", "restart", "free", "clean"
}

// Enforcement is the action taken
type Enforcement struct {
	Diagnosis Diagnosis
	Action    string
	Success   bool
	Detail    string
}

// --- STAGE 1: DETECTION ---
// Pure observation. No action. Just reads metrics and flags anomalies.

func detect(cpu, mem, disk float64) []Issue {
	issues := make([]Issue, 0)

	if cpu > 90 {
		sev := SevHigh
		if cpu > 98 {
			sev = SevCritical
		}
		issues = append(issues, Issue{Resource: "cpu", Value: cpu, Severity: sev})
	} else if cpu > 75 {
		issues = append(issues, Issue{Resource: "cpu", Value: cpu, Severity: SevMedium})
	}

	if mem > 90 {
		sev := SevHigh
		if mem > 95 {
			sev = SevCritical
		}
		issues = append(issues, Issue{Resource: "memory", Value: mem, Severity: sev})
	} else if mem > 75 {
		issues = append(issues, Issue{Resource: "memory", Value: mem, Severity: SevMedium})
	}

	if disk > 90 {
		issues = append(issues, Issue{Resource: "disk", Value: disk, Severity: SevHigh})
	} else if disk > 80 {
		issues = append(issues, Issue{Resource: "disk", Value: disk, Severity: SevMedium})
	}

	return issues
}

// --- STAGE 2: DIAGNOSIS ---
// Identifies root cause. Scans processes. Checks internal state.
// Does NOT take action.

func diagnose(issue Issue) Diagnosis {
	d := Diagnosis{Issue: issue, Confidence: 0.5, Recommended: "monitor"}

	switch issue.Resource {
	case "cpu":
		// Check internal burn first
		if isCPUBurning() {
			d.RootCause = "Internal CPU burn goroutines"
			d.Confidence = 1.0
			d.Issue.Internal = true
			d.Issue.CausePID = os.Getpid()
			if issue.Severity >= SevHigh {
				d.Recommended = "kill"
			} else {
				d.Recommended = "throttle"
			}
			return d
		}

		// Scan external processes
		procs := findTopProcesses("cpu", 3)
		myPID := os.Getpid()
		for _, p := range procs {
			if p.PID != myPID && p.CPUTime > 50 {
				d.RootCause = fmt.Sprintf("Process '%s' (PID %d) using %.0fs CPU", p.Name, p.PID, p.CPUTime)
				d.Confidence = 0.8
				d.Issue.CausePID = p.PID
				if issue.Severity >= SevHigh {
					d.Recommended = "kill"
				} else {
					d.Recommended = "monitor"
				}
				return d
			}
		}
		d.RootCause = "Unknown CPU consumer"

	case "memory":
		held := currentMemHeldMB()
		if held > 100 {
			d.RootCause = fmt.Sprintf("Internal stress holding %dMB", held)
			d.Confidence = 1.0
			d.Issue.Internal = true
			d.Issue.CausePID = os.Getpid()
			if issue.Severity >= SevHigh {
				d.Recommended = "free"
			} else {
				d.Recommended = "monitor"
			}
			return d
		}

		procs := findTopProcesses("memory", 3)
		myPID := os.Getpid()
		for _, p := range procs {
			if p.PID != myPID && p.MemMB > 500 {
				d.RootCause = fmt.Sprintf("Process '%s' (PID %d) using %.0fMB", p.Name, p.PID, p.MemMB)
				d.Confidence = 0.8
				d.Issue.CausePID = p.PID
				if issue.Severity >= SevHigh {
					d.Recommended = "restart"
				} else {
					d.Recommended = "monitor"
				}
				return d
			}
		}
		d.RootCause = "Unknown memory consumer"

	case "disk":
		if diskFile != "" {
			fi, _ := os.Stat(diskFile)
			sizeMB := int64(0)
			if fi != nil {
				sizeMB = fi.Size() / 1024 / 1024
			}
			d.RootCause = fmt.Sprintf("Stress file %s (%dMB)", diskFile, sizeMB)
			d.Confidence = 1.0
			d.Issue.Internal = true
			d.Recommended = "clean"
			return d
		}
		d.RootCause = "Unknown disk consumer"
	}

	return d
}

// --- STAGE 3: ENFORCEMENT ---
// Takes action based on diagnosis. Escalates if previous action failed.
// Tracks history to avoid repeating failed actions.

type EnforcementEngine struct {
	history     []Enforcement
	failCounts  map[string]int // resource -> consecutive failures
	mu          sync.Mutex
}

func NewEnforcementEngine() *EnforcementEngine {
	return &EnforcementEngine{
		history:    make([]Enforcement, 0),
		failCounts: make(map[string]int),
	}
}

func (e *EnforcementEngine) Enforce(d Diagnosis) Enforcement {
	e.mu.Lock()
	defer e.mu.Unlock()

	result := Enforcement{Diagnosis: d}

	// Escalation: if we've failed before on this resource, escalate
	fails := e.failCounts[d.Issue.Resource]
	action := d.Recommended

	if fails >= 2 {
		// Escalate: monitor → throttle → kill → restart
		switch action {
		case "monitor":
			action = "throttle"
		case "throttle":
			action = "kill"
		case "kill":
			action = "restart"
		}
	}

	result.Action = action

	switch action {
	case "monitor":
		result.Success = true
		result.Detail = fmt.Sprintf("Monitoring %s (%.0f%%, cause: %s)", d.Issue.Resource, d.Issue.Value, d.RootCause)

	case "throttle":
		// Partial mitigation
		if d.Issue.Resource == "memory" && d.Issue.Internal {
			// Free half the memory
			held := currentMemHeldMB()
			memMu.Lock()
			if len(memHeld) > held/2 {
				memHeld = memHeld[:held/2]
			}
			memMu.Unlock()
			runtime.GC()
			result.Success = true
			result.Detail = fmt.Sprintf("Throttled: freed %dMB (kept %dMB)", held/2, held/2)
		} else if d.Issue.Resource == "cpu" && d.Issue.Internal {
			// Can't partially throttle goroutines — escalate to kill
			stopCPUBurn()
			result.Success = true
			result.Detail = "Throttle not possible for CPU burn — killed instead"
		} else {
			result.Success = false
			result.Detail = "Cannot throttle external process"
		}

	case "kill":
		if d.Issue.Internal {
			switch d.Issue.Resource {
			case "cpu":
				stopCPUBurn()
				result.Success = true
				result.Detail = "Killed internal CPU burn"
			case "memory":
				freed := freeMemory()
				result.Success = true
				result.Detail = fmt.Sprintf("Killed: freed %dMB", freed)
			case "disk":
				cleanDisk()
				result.Success = true
				result.Detail = "Killed: removed stress file"
			}
		} else if d.Issue.CausePID > 0 {
			// Kill external process
			cmd := exec.Command("taskkill", "/F", "/PID", fmt.Sprintf("%d", d.Issue.CausePID))
			if err := cmd.Run(); err != nil {
				result.Success = false
				result.Detail = fmt.Sprintf("Failed to kill PID %d: %v", d.Issue.CausePID, err)
			} else {
				result.Success = true
				result.Detail = fmt.Sprintf("Killed external process PID %d", d.Issue.CausePID)
			}
		} else {
			result.Success = false
			result.Detail = "No target PID to kill"
		}

	case "restart":
		// Nuclear option: kill everything related to this resource
		if d.Issue.Resource == "cpu" {
			stopCPUBurn()
		}
		if d.Issue.Resource == "memory" {
			freeMemory()
		}
		if d.Issue.Resource == "disk" {
			cleanDisk()
		}
		result.Success = true
		result.Detail = fmt.Sprintf("Full restart/cleanup of %s subsystem", d.Issue.Resource)

	case "free":
		freed := freeMemory()
		result.Success = true
		result.Detail = fmt.Sprintf("Freed %dMB", freed)

	case "clean":
		cleanDisk()
		result.Success = true
		result.Detail = "Cleaned disk stress file"
	}

	// Track success/failure for escalation
	if result.Success {
		e.failCounts[d.Issue.Resource] = 0
	} else {
		e.failCounts[d.Issue.Resource]++
	}

	e.history = append(e.history, result)
	return result
}

// --- BRIDGE: connects pipeline to AIZO rule engine ---

type PipelineExecutor struct {
	enforcer *EnforcementEngine
	log      []string
}

func (p *PipelineExecutor) ExecuteAction(ctx context.Context, action *layer6.ProposedAction) (*layer6.ExecutionResult, error) {
	// The rule engine fires — but we run our own pipeline
	cpu := readCPU()
	mem := readMemory()
	disk := readDisk()

	issues := detect(cpu, mem, disk)
	if len(issues) == 0 {
		return &layer6.ExecutionResult{Success: true, Message: "No issues detected"}, nil
	}

	var allResults []string
	for _, issue := range issues {
		d := diagnose(issue)
		e := p.enforcer.Enforce(d)

		sevIcon := "🟡"
		if d.Issue.Severity >= SevHigh {
			sevIcon = "🔴"
		}
		if d.Issue.Severity == SevCritical {
			sevIcon = "💀"
		}

		resultIcon := "✅"
		if !e.Success {
			resultIcon = "❌"
		}

		line := fmt.Sprintf("%s %s %.0f%% → CAUSE: %s (%.0f%% confident) → ACTION: %s → %s %s",
			sevIcon, issue.Resource, issue.Value,
			d.RootCause, d.Confidence*100,
			e.Action, resultIcon, e.Detail)

		allResults = append(allResults, line)
		p.log = append(p.log, line)
	}

	return &layer6.ExecutionResult{
		Success: true,
		Message: strings.Join(allResults, "\n"),
	}, nil
}

// ═══════════════════════════════════════════
// MAIN
// ═══════════════════════════════════════════

func main() {
	fmt.Println("=== AIZO Detection → Diagnosis → Enforcement Pipeline ===")
	fmt.Printf("Machine: %s (%d cores)\n", runtime.GOOS, runtime.NumCPU())
	fmt.Println("Three-stage pipeline: DETECT anomaly → DIAGNOSE root cause → ENFORCE fix")
	fmt.Println("Escalation: monitor → throttle → kill → restart")
	fmt.Println("Open Task Manager to watch!\n")

	db, err := storage.Open("")
	if err != nil {
		fmt.Printf("DB error: %v\n", err)
		return
	}
	defer db.Close()
	eventLog, _ := storage.NewEventLog(db)

	enforcer := NewEnforcementEngine()
	executor := &PipelineExecutor{enforcer: enforcer}
	mgr := layer6.NewManager(db, executor)
	mgr.Start(context.Background())

	for _, rule := range mgr.ListRules() {
		rule.Action.AutoApprove = true
	}

	stop := make(chan struct{})

	fmt.Println("📊 Baseline:")
	fmt.Printf("  CPU: %.0f%% | Memory: %.0f%% | Disk: %.0f%%\n", readCPU(), readMemory(), readDisk())

	// WAVE 1: CPU
	fmt.Println("\n" + strings.Repeat("═", 70))
	fmt.Println("🔥 WAVE 1: CPU burn on all cores")
	fmt.Println(strings.Repeat("═", 70))
	startCPUBurn(stop)

	// WAVE 2: Memory after 15s
	go func() {
		time.Sleep(15 * time.Second)
		fmt.Println("\n🔥 WAVE 2: Allocating 3GB memory")
		allocateMemory(3072)
	}()

	// WAVE 3: Disk after 30s
	go func() {
		time.Sleep(30 * time.Second)
		fmt.Println("\n🔥 WAVE 3: Writing 2GB to disk")
		fillDisk(2048)
	}()

	// RE-INJECT after AIZO fixes things
	go func() {
		time.Sleep(50 * time.Second)
		if !isCPUBurning() {
			fmt.Println("\n🔄 RE-INJECT: Stress is back!")
			startCPUBurn(stop)
			allocateMemory(1024)
		}
	}()

	fmt.Println("\n📊 Pipeline running. DETECT → DIAGNOSE → ENFORCE every 2 seconds.\n")

	for i := 0; i < 50; i++ {
		time.Sleep(2 * time.Second)

		cpu := readCPU()
		mem := readMemory()
		disk := readDisk()

		tags := ""
		if isCPUBurning() {
			tags += " 🔴CPU"
		}
		if currentMemHeldMB() > 0 {
			tags += fmt.Sprintf(" 🔴MEM(%dMB)", currentMemHeldMB())
		}
		if diskFile != "" {
			tags += " 🔴DISK"
		}
		if tags == "" {
			tags = " 🟢HEALTHY"
		}

		fmt.Printf("[%3ds] CPU:%5.0f%% Mem:%5.0f%% Disk:%5.0f%% %s\n", (i+1)*2, cpu, mem, disk, tags)

		// Feed to rule engine — it calls our PipelineExecutor
		summary := &layer6.SystemSummary{
			Timestamp:         time.Now(),
			TotalEntities:     1,
			RunningContainers: 1,
			MemoryUsage:       mem,
			CPUUsage:          cpu,
			DiskUsage:         disk,
			Metrics: map[string]float64{
				"memory_usage": mem,
				"cpu_usage":    cpu,
				"disk_usage":   disk,
			},
		}

		proposals, _ := mgr.ProcessSummary(summary)
		for _, p := range proposals {
			if p.Result != "" {
				for _, line := range strings.Split(p.Result, "\n") {
					fmt.Printf("  → %s\n", line)
				}
			}
			eventLog.Append("pipeline", "system", "local", map[string]string{
				"rule": p.RuleID, "result": p.Result,
			})
		}
	}

	// Cleanup
	close(stop)
	stopCPUBurn()
	freeMemory()
	cleanDisk()

	// Final report
	fmt.Println("\n" + strings.Repeat("═", 70))
	fmt.Println("📈 PIPELINE REPORT")
	fmt.Println(strings.Repeat("═", 70))
	fmt.Printf("Final: CPU=%.0f%% Mem=%.0f%% Disk=%.0f%%\n\n", readCPU(), readMemory(), readDisk())

	fmt.Printf("Total enforcement actions: %d\n", len(enforcer.history))
	successes := 0
	for _, e := range enforcer.history {
		if e.Success {
			successes++
		}
	}
	fmt.Printf("Successful: %d/%d (%.0f%%)\n\n", successes, len(enforcer.history), float64(successes)/float64(max(len(enforcer.history), 1))*100)

	fmt.Println("Action log:")
	for i, line := range executor.log {
		fmt.Printf("  %d. %s\n", i+1, line)
	}

	count, _ := eventLog.Count()
	verified, violations, _ := eventLog.VerifyIntegrity()
	fmt.Printf("\nEvent log: %d events, %d verified", count, verified)
	if len(violations) == 0 {
		fmt.Println(", chain intact ✓")
	} else {
		fmt.Printf(", %d violations ✗\n", len(violations))
	}
}
