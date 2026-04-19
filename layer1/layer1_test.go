package layer1_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/realityos/aizo/layer1"
)

// --- Adapter Tests ---

func TestBaseAdapter_RetryWithBackoff(t *testing.T) {
	config := &layer1.AdapterConfig{
		ID:            "test",
		Type:          layer1.AdapterTypeHTTP,
		Target:        "http://example.com",
		RetryAttempts: 3,
		RetryBackoff:  1 * time.Millisecond,
	}

	base := layer1.NewBaseAdapter(config, nil)

	attempts := 0
	err := base.RetryWithBackoff(context.Background(), func() error {
		attempts++
		if attempts < 3 {
			return fmt.Errorf("temporary error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected success after retries, got: %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestBaseAdapter_RetryExhausted(t *testing.T) {
	config := &layer1.AdapterConfig{
		ID:            "test",
		Type:          layer1.AdapterTypeHTTP,
		Target:        "http://example.com",
		RetryAttempts: 2,
		RetryBackoff:  1 * time.Millisecond,
	}

	base := layer1.NewBaseAdapter(config, nil)

	err := base.RetryWithBackoff(context.Background(), func() error {
		return fmt.Errorf("always fails")
	})

	if err == nil {
		t.Error("expected error after exhausted retries")
	}
}

func TestBaseAdapter_ContextCancellation(t *testing.T) {
	config := &layer1.AdapterConfig{
		ID:            "test",
		Type:          layer1.AdapterTypeHTTP,
		Target:        "http://example.com",
		RetryAttempts: 10,
		RetryBackoff:  100 * time.Millisecond,
	}

	base := layer1.NewBaseAdapter(config, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := base.RetryWithBackoff(ctx, func() error {
		return fmt.Errorf("error")
	})

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

// --- HTTP Adapter Tests ---

func TestHTTPAdapter_Connect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	adapter := layer1.NewHTTPAdapter(&layer1.AdapterConfig{
		ID:            "test-http",
		Target:        server.URL,
		Timeout:       5 * time.Second,
		RetryAttempts: 1,
		RetryBackoff:  1 * time.Millisecond,
	})

	err := adapter.Connect(context.Background())
	if err != nil {
		t.Fatalf("expected successful connect, got: %v", err)
	}
}

func TestHTTPAdapter_ConnectFails(t *testing.T) {
	adapter := layer1.NewHTTPAdapter(&layer1.AdapterConfig{
		ID:            "test-http-fail",
		Target:        "http://localhost:1", // nothing listening here
		Timeout:       1 * time.Second,
		RetryAttempts: 1,
		RetryBackoff:  1 * time.Millisecond,
	})

	err := adapter.Connect(context.Background())
	if err == nil {
		t.Error("expected connection failure")
	}
}

func TestHTTPAdapter_ReadState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","version":"1.0"}`))
	}))
	defer server.Close()

	adapter := layer1.NewHTTPAdapter(&layer1.AdapterConfig{
		ID:            "test-read",
		Target:        server.URL,
		Timeout:       5 * time.Second,
		RetryAttempts: 1,
		RetryBackoff:  1 * time.Millisecond,
	})

	adapter.Connect(context.Background())

	state, err := adapter.ReadState(context.Background())
	if err != nil {
		t.Fatalf("expected successful read, got: %v", err)
	}

	if state == nil {
		t.Fatal("expected state data, got nil")
	}

	if state.Data["status"] != "ok" {
		t.Errorf("expected status=ok, got: %v", state.Data["status"])
	}
}

func TestHTTPAdapter_SendCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"result":"executed"}`))
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	adapter := layer1.NewHTTPAdapter(&layer1.AdapterConfig{
		ID:            "test-cmd",
		Target:        server.URL,
		Timeout:       5 * time.Second,
		RetryAttempts: 1,
		RetryBackoff:  1 * time.Millisecond,
	})

	adapter.Connect(context.Background())

	resp, err := adapter.SendCommand(context.Background(), &layer1.CommandRequest{
		ID:      "cmd-1",
		Command: "restart",
		Args:    map[string]interface{}{"force": true},
	})

	if err != nil {
		t.Fatalf("expected successful command, got: %v", err)
	}
	if !resp.Success {
		t.Error("expected success=true")
	}
}

func TestHTTPAdapter_HealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	adapter := layer1.NewHTTPAdapter(&layer1.AdapterConfig{
		ID:            "test-health",
		Target:        server.URL,
		Timeout:       5 * time.Second,
		RetryAttempts: 1,
		RetryBackoff:  1 * time.Millisecond,
	})

	health, err := adapter.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("expected successful health check, got: %v", err)
	}
	if health.Status != layer1.HealthStatusHealthy {
		t.Errorf("expected healthy, got: %s", health.Status)
	}
}

func TestHTTPAdapter_Auth_Bearer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	adapter := layer1.NewHTTPAdapter(&layer1.AdapterConfig{
		ID:     "test-auth",
		Target: server.URL,
		Credentials: map[string]string{
			"bearer_token": "test-token",
		},
		Timeout:       5 * time.Second,
		RetryAttempts: 1,
		RetryBackoff:  1 * time.Millisecond,
	})

	err := adapter.Connect(context.Background())
	if err != nil {
		t.Fatalf("expected successful connect with bearer auth, got: %v", err)
	}
}

// --- Registry Tests ---

func TestRegistry_RegisterAndGet(t *testing.T) {
	registry := layer1.NewAdapterRegistry()

	adapter := layer1.NewHTTPAdapter(&layer1.AdapterConfig{
		ID:     "reg-test",
		Target: "http://example.com",
	})

	err := registry.Register(adapter)
	if err != nil {
		t.Fatalf("expected successful register, got: %v", err)
	}

	got, err := registry.Get("reg-test")
	if err != nil {
		t.Fatalf("expected to find adapter, got: %v", err)
	}
	if got.GetConfig().ID != "reg-test" {
		t.Errorf("expected ID=reg-test, got: %s", got.GetConfig().ID)
	}
}

func TestRegistry_DuplicateRegister(t *testing.T) {
	registry := layer1.NewAdapterRegistry()

	adapter := layer1.NewHTTPAdapter(&layer1.AdapterConfig{
		ID:     "dup-test",
		Target: "http://example.com",
	})

	registry.Register(adapter)
	err := registry.Register(adapter)
	if err == nil {
		t.Error("expected error on duplicate registration")
	}
}

func TestRegistry_Unregister(t *testing.T) {
	registry := layer1.NewAdapterRegistry()

	adapter := layer1.NewHTTPAdapter(&layer1.AdapterConfig{
		ID:     "unreg-test",
		Target: "http://example.com",
	})

	registry.Register(adapter)
	err := registry.Unregister("unreg-test")
	if err != nil {
		t.Fatalf("expected successful unregister, got: %v", err)
	}

	_, err = registry.Get("unreg-test")
	if err == nil {
		t.Error("expected error after unregister")
	}
}

func TestRegistry_ListByType(t *testing.T) {
	registry := layer1.NewAdapterRegistry()

	registry.Register(layer1.NewHTTPAdapter(&layer1.AdapterConfig{ID: "http-1", Target: "http://a.com"}))
	registry.Register(layer1.NewHTTPAdapter(&layer1.AdapterConfig{ID: "http-2", Target: "http://b.com"}))
	registry.Register(layer1.NewSSHAdapter(&layer1.AdapterConfig{ID: "ssh-1", Target: "10.0.0.1:22"}))

	httpAdapters := registry.ListByType(layer1.AdapterTypeHTTP)
	if len(httpAdapters) != 2 {
		t.Errorf("expected 2 HTTP adapters, got %d", len(httpAdapters))
	}

	sshAdapters := registry.ListByType(layer1.AdapterTypeSSH)
	if len(sshAdapters) != 1 {
		t.Errorf("expected 1 SSH adapter, got %d", len(sshAdapters))
	}
}

func TestRegistry_Count(t *testing.T) {
	registry := layer1.NewAdapterRegistry()

	if registry.Count() != 0 {
		t.Error("expected empty registry")
	}

	registry.Register(layer1.NewHTTPAdapter(&layer1.AdapterConfig{ID: "count-1", Target: "http://a.com"}))
	registry.Register(layer1.NewHTTPAdapter(&layer1.AdapterConfig{ID: "count-2", Target: "http://b.com"}))

	if registry.Count() != 2 {
		t.Errorf("expected count=2, got %d", registry.Count())
	}
}

// --- Manager Tests ---

func TestManager_CreateAdapter(t *testing.T) {
	manager := layer1.NewManager()

	_, err := manager.CreateAdapter(&layer1.AdapterConfig{
		ID:     "mgr-test",
		Type:   layer1.AdapterTypeHTTP,
		Target: "http://example.com",
	})

	if err != nil {
		t.Fatalf("expected successful create, got: %v", err)
	}

	if len(manager.ListAdapters()) != 1 {
		t.Error("expected 1 adapter in manager")
	}
}

func TestManager_GetStats(t *testing.T) {
	manager := layer1.NewManager()

	manager.CreateAdapter(&layer1.AdapterConfig{ID: "s1", Type: layer1.AdapterTypeHTTP, Target: "http://a.com"})
	manager.CreateAdapter(&layer1.AdapterConfig{ID: "s2", Type: layer1.AdapterTypeHTTP, Target: "http://b.com"})
	manager.CreateAdapter(&layer1.AdapterConfig{ID: "s3", Type: layer1.AdapterTypeSSH, Target: "10.0.0.1:22"})

	stats := manager.GetStats()

	if stats.TotalAdapters != 3 {
		t.Errorf("expected 3 total adapters, got %d", stats.TotalAdapters)
	}
	if stats.ByType[layer1.AdapterTypeHTTP] != 2 {
		t.Errorf("expected 2 HTTP adapters, got %d", stats.ByType[layer1.AdapterTypeHTTP])
	}
	if stats.ByType[layer1.AdapterTypeSSH] != 1 {
		t.Errorf("expected 1 SSH adapter, got %d", stats.ByType[layer1.AdapterTypeSSH])
	}
}

func TestManager_Shutdown(t *testing.T) {
	manager := layer1.NewManager()

	manager.CreateAdapter(&layer1.AdapterConfig{ID: "shut-1", Type: layer1.AdapterTypeHTTP, Target: "http://a.com"})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := manager.Shutdown(ctx)
	if err != nil {
		t.Fatalf("expected clean shutdown, got: %v", err)
	}

	if len(manager.ListAdapters()) != 0 {
		t.Error("expected empty registry after shutdown")
	}
}
