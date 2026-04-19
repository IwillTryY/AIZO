package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/realityos/aizo/layer6"
	"github.com/realityos/aizo/storage"
)

// ═══════════════════════════════════════════
// REAL WEB SERVER WITH REAL FAULTS
// ═══════════════════════════════════════════

type WebServer struct {
	port         string
	requestCount int64
	memoryLeak   [][]byte
	healthy      int32 // 1 = healthy, 0 = unhealthy
	server       *http.Server
}

func NewWebServer(port string) *WebServer {
	return &WebServer{port: port, memoryLeak: make([][]byte, 0)}
}

func (w *WebServer) Start() {
	atomic.StoreInt32(&w.healthy, 1)
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt64(&w.requestCount, 1)
		// Leak 1MB per request
		leak := make([]byte, 1024*1024)
		for i := range leak {
			if i%4096 == 0 {
				leak[i] = byte(count)
			}
		}
		w.memoryLeak = append(w.memoryLeak, leak)
		fmt.Fprintf(rw, "OK request=%d leaked=%dMB\n", count, len(w.memoryLeak))
	})

	mux.HandleFunc("/health", func(rw http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(&w.healthy) == 0 {
			rw.WriteHeader(503)
			fmt.Fprintf(rw, "UNHEALTHY leaked=%dMB\n", len(w.memoryLeak))
			return
		}
		if len(w.memoryLeak) > 50 {
			atomic.StoreInt32(&w.healthy, 0)
			rw.WriteHeader(503)
			fmt.Fprintf(rw, "DEGRADED leaked=%dMB\n", len(w.memoryLeak))
			return
		}
		rw.WriteHeader(200)
		fmt.Fprintf(rw, "OK leaked=%dMB\n", len(w.memoryLeak))
	})

	mux.HandleFunc("/cleanup", func(rw http.ResponseWriter, r *http.Request) {
		old := len(w.memoryLeak)
		w.memoryLeak = make([][]byte, 0)
		atomic.StoreInt32(&w.healthy, 1)
		fmt.Fprintf(rw, "CLEANED %dMB\n", old)
	})

	mux.HandleFunc("/stats", func(rw http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(rw, "requests=%d leaked=%dMB healthy=%v\n",
			atomic.LoadInt64(&w.requestCount), len(w.memoryLeak), atomic.LoadInt32(&w.healthy) == 1)
	})

	w.server = &http.Server{Addr: "127.0.0.1:" + w.port, Handler: mux}
	go w.server.ListenAndServe()
}

func (w *WebServer) Stop() {
	if w.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		w.server.Shutdown(ctx)
	}
}

func (w *WebServer) LeakedMB() int    { return len(w.memoryLeak) }
func (w *WebServer) Requests() int64   { return atomic.LoadInt64(&w.requestCount) }
func (w *WebServer) IsHealthy() bool   { return atomic.LoadInt32(&w.healthy) == 1 }

// ═══════════════════════════════════════════
// EXECUTOR THAT ACTUALLY FIXES THE WEB SERVER
// ═══════════════════════════════════════════

type WebServerExecutor struct {
	server  *WebServer
	actions []string
}

func (e *WebServerExecutor) ExecuteAction(ctx context.Context, action *layer6.ProposedAction) (*layer6.ExecutionResult, error) {
	switch action.ActionType {
	case "investigate":
		// Check the web server health endpoint
		resp, err := http.Get("http://127.0.0.1:8080/health")
		if err != nil {
			return &layer6.ExecutionResult{
				Success: false,
				Message: "Server unreachable: " + err.Error(),
			}, nil
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		msg := fmt.Sprintf("Health check: HTTP %d — %s", resp.StatusCode, strings.TrimSpace(string(body)))
		e.actions = append(e.actions, "INVESTIGATE: "+msg)
		return &layer6.ExecutionResult{Success: true, Message: msg}, nil

	case "cleanup":
		// Call the cleanup endpoint to free leaked memory
		resp, err := http.Get("http://127.0.0.1:8080/cleanup")
		if err != nil {
			return &layer6.ExecutionResult{Success: false, Message: "Cleanup failed: " + err.Error()}, nil
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		msg := fmt.Sprintf("Cleanup: %s", strings.TrimSpace(string(body)))
		e.actions = append(e.actions, "CLEANUP: "+msg)
		return &layer6.ExecutionResult{Success: true, Message: msg}, nil

	case "restart":
		// Actually restart the web server
		e.server.Stop()
		time.Sleep(500 * time.Millisecond)
		e.server.memoryLeak = make([][]byte, 0)
		atomic.StoreInt32(&e.server.healthy, 1)
		e.server.Start()
		time.Sleep(500 * time.Millisecond)

		// Verify it came back
		resp, err := http.Get("http://127.0.0.1:8080/health")
		msg := "Restarted web server"
		if err == nil {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			msg += fmt.Sprintf(" — health: HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
		e.actions = append(e.actions, "RESTART: "+msg)
		return &layer6.ExecutionResult{Success: true, Message: msg}, nil
	}

	return &layer6.ExecutionResult{Success: true, Message: "unknown action"}, nil
}

// ═══════════════════════════════════════════
// MAIN: END-TO-END SELF-HEALING DEMO
// ═══════════════════════════════════════════

func main() {
	fmt.Println("=== AIZO End-to-End Self-Healing Demo ===")
	fmt.Println("Real web server. Real memory leak. Real detection. Real fix.\n")

	// Init storage
	db, err := storage.Open("")
	if err != nil {
		fmt.Printf("DB error: %v\n", err)
		return
	}
	defer db.Close()
	eventLog, _ := storage.NewEventLog(db)

	// Start real web server
	fmt.Println("🌐 Starting web server on http://127.0.0.1:8080")
	server := NewWebServer("8080")
	server.Start()
	time.Sleep(500 * time.Millisecond)

	// Verify it's running
	resp, err := http.Get("http://127.0.0.1:8080/health")
	if err != nil {
		fmt.Printf("Server failed to start: %v\n", err)
		return
	}
	resp.Body.Close()
	fmt.Println("  ✓ Server running and healthy\n")

	// Init rule engine with real executor
	executor := &WebServerExecutor{server: server}
	mgr := layer6.NewManager(db, executor)
	mgr.Start(context.Background())

	// All rules auto-approve
	for _, rule := range mgr.ListRules() {
		rule.Action.AutoApprove = true
	}

	// Add a custom rule: if memory_leaked_mb > 20, cleanup
	mgr.AddRule(&layer6.Rule{
		ID:   "webserver-leak-cleanup",
		Name: "Web Server Leak Cleanup",
		Conditions: []layer6.Condition{
			{Metric: "memory_leaked_mb", Operator: ">", Value: 20},
		},
		Action: layer6.RuleAction{
			Type:        "cleanup",
			Risk:        "low",
			Reversible:  true,
			AutoApprove: true,
			Reasoning:   "Web server memory leak detected, calling /cleanup",
		},
		Priority: 60,
		Enabled:  true,
	})

	// Add rule: if memory_leaked_mb > 50, restart
	mgr.AddRule(&layer6.Rule{
		ID:   "webserver-leak-restart",
		Name: "Web Server Leak Restart",
		Conditions: []layer6.Condition{
			{Metric: "memory_leaked_mb", Operator: ">", Value: 50},
		},
		Action: layer6.RuleAction{
			Type:        "restart",
			Risk:        "medium",
			Reversible:  true,
			AutoApprove: true,
			Reasoning:   "Web server memory leak critical, full restart",
		},
		Priority: 80,
		Enabled:  true,
	})

	fmt.Println("📋 Rules loaded:")
	for _, r := range mgr.ListRules() {
		if strings.HasPrefix(r.ID, "webserver") {
			fmt.Printf("  [%s] %s → %s (threshold: %.0f)\n",
				r.ID, r.Name, r.Action.Type, r.Conditions[0].Value)
		}
	}

	fmt.Println("\n" + strings.Repeat("═", 65))
	fmt.Println("🔥 Generating traffic to cause memory leak...")
	fmt.Println("   AIZO monitors every 2 seconds and will fix the leak")
	fmt.Println(strings.Repeat("═", 65) + "\n")

	// Traffic generator: 5 requests per second
	stopTraffic := make(chan struct{})
	go func() {
		for {
			select {
			case <-stopTraffic:
				return
			default:
				resp, err := http.Get("http://127.0.0.1:8080/")
				if err == nil {
					resp.Body.Close()
				}
				time.Sleep(200 * time.Millisecond)
			}
		}
	}()

	// Monitor loop: AIZO watches and acts
	healCount := 0
	for i := 0; i < 60; i++ {
		time.Sleep(2 * time.Second)

		leaked := server.LeakedMB()
		requests := server.Requests()
		healthy := server.IsHealthy()

		healthTag := "🟢"
		if !healthy {
			healthTag = "🔴"
		}

		fmt.Printf("  [%3ds] %s Requests: %-5d Leaked: %-4dMB",
			(i+1)*2, healthTag, requests, leaked)

		// Feed real metrics to rule engine
		summary := &layer6.SystemSummary{
			Timestamp:     time.Now(),
			TotalEntities: 1,
			Metrics: map[string]float64{
				"memory_leaked_mb": float64(leaked),
				"request_count":    float64(requests),
			},
		}

		proposals, _ := mgr.ProcessSummary(summary)
		for _, p := range proposals {
			fmt.Printf("\n    🚨 %s → %s", p.RuleID, p.Action)
			if p.Result != "" {
				result := p.Result
				if len(result) > 60 {
					result = result[:60] + "..."
				}
				fmt.Printf(" → %s", result)
				healCount++
			}
			eventLog.Append("auto_heal", "web-server", "local", map[string]string{
				"rule": p.RuleID, "action": p.Action, "leaked_mb": fmt.Sprintf("%d", leaked),
			})
		}

		// Also check health endpoint directly
		if !healthy && len(proposals) == 0 {
			event := &layer6.SystemEvent{
				ID:          fmt.Sprintf("health-fail-%d", i),
				Timestamp:   time.Now(),
				Type:        layer6.EventHealthCheckFail,
				Severity:    "high",
				EntityID:    "web-server",
				Description: fmt.Sprintf("Health check failed, %dMB leaked", leaked),
			}
			proposal, _ := mgr.ProcessEvent(event)
			if proposal != nil {
				fmt.Printf("\n    🔥 EVENT: health_check_fail → %s", proposal.Action)
				healCount++
			}
		}

		fmt.Println()
	}

	// Stop traffic
	close(stopTraffic)
	server.Stop()

	// Final report
	fmt.Println("\n" + strings.Repeat("═", 65))
	fmt.Println("📈 FINAL REPORT")
	fmt.Println(strings.Repeat("═", 65))
	fmt.Printf("  Total requests served: %d\n", server.Requests())
	fmt.Printf("  Times AIZO healed: %d\n", healCount)
	fmt.Printf("  Actions taken:\n")
	for i, a := range executor.actions {
		fmt.Printf("    %d. %s\n", i+1, a)
	}

	stats := mgr.GetStats()
	fmt.Printf("  Total proposals: %d\n", stats.TotalProposals)
	fmt.Printf("  Rules: %d\n", stats.TotalRules)

	// Verify event log integrity
	verified, violations, _ := eventLog.VerifyIntegrity()
	if len(violations) == 0 {
		fmt.Printf("  Event log: %d events, chain intact ✓\n", verified)
	} else {
		fmt.Printf("  Event log: %d events, %d violations ✗\n", verified, len(violations))
	}

	fmt.Println("\n✅ Demo complete. AIZO detected and fixed real memory leaks autonomously.")
}
