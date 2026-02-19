package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecode_FullReview(t *testing.T) {
	body := `{
		"metadata": {"name": "test"},
		"spec": {
			"selector": {"verbs": ["get"], "resources": ["pods"]}
		}
	}`
	review, err := decodeClientRequest([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(review.Spec.Selector.Verbs) != 1 || review.Spec.Selector.Verbs[0] != "get" {
		t.Errorf("expected verbs=[get], got %v", review.Spec.Selector.Verbs)
	}
	if len(review.Spec.Selector.Resources) != 1 || review.Spec.Selector.Resources[0] != "pods" {
		t.Errorf("expected resources=[pods], got %v", review.Spec.Selector.Resources)
	}
}

func TestDecode_SpecOnly(t *testing.T) {
	body := `{
		"selector": {"verbs": ["list"], "apiGroups": [""]},
		"matchMode": "all"
	}`
	review, err := decodeClientRequest([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if review.Name != "web-query" {
		t.Errorf("expected name=web-query, got %s", review.Name)
	}
	if len(review.Spec.Selector.Verbs) != 1 || review.Spec.Selector.Verbs[0] != "list" {
		t.Errorf("expected verbs=[list], got %v", review.Spec.Selector.Verbs)
	}
}

func TestDecode_SelectorOnly(t *testing.T) {
	body := `{"verbs": ["create"], "resources": ["configmaps"]}`
	review, err := decodeClientRequest([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if review.Name != "web-query" {
		t.Errorf("expected name=web-query, got %s", review.Name)
	}
	if len(review.Spec.Selector.Verbs) != 1 || review.Spec.Selector.Verbs[0] != "create" {
		t.Errorf("expected verbs=[create], got %v", review.Spec.Selector.Verbs)
	}
}

func TestDecode_EmptyBody(t *testing.T) {
	_, err := decodeClientRequest([]byte(""))
	if err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestDecode_InvalidJSON(t *testing.T) {
	_, err := decodeClientRequest([]byte("{not valid json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestHandleQuery_MethodNotAllowed(t *testing.T) {
	ws := &webServer{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/query", http.NoBody)
	ws.handleQuery(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

func TestHandleQuery_ImpersonationGroupWithoutUser(t *testing.T) {
	// Start a dummy backend that returns 200 (should never be reached).
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":{}}`))
	}))
	defer backend.Close()

	ws := &webServer{
		httpClient:  backend.Client(),
		apiEndpoint: backend.URL,
	}
	body := `{"selector": {"verbs": ["get"]}}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/query", strings.NewReader(body))
	req.Header.Set("X-Impersonate-Group", "some-group")
	// No X-Impersonate-User header.
	ws.handleQuery(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandleQuery_ProxiesToBackend(t *testing.T) {
	expectedResponse := `{"metadata":{},"spec":{},"status":{"matchedRoles":1}}`
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(expectedResponse))
	}))
	defer backend.Close()

	ws := &webServer{
		httpClient:  backend.Client(),
		apiEndpoint: backend.URL,
	}
	body := `{"spec": {"selector": {"verbs": ["get"]}}}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/query", strings.NewReader(body))
	ws.handleQuery(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
	respBody, _ := io.ReadAll(rec.Body)
	var parsed map[string]any
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
}

func TestHandleQuery_ImpersonationHeadersForwarded(t *testing.T) {
	var capturedUser, capturedGroup string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUser = r.Header.Get("Impersonate-User")
		capturedGroup = r.Header.Get("Impersonate-Group")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer backend.Close()

	ws := &webServer{
		httpClient:  backend.Client(),
		apiEndpoint: backend.URL,
	}
	body := `{"selector": {"verbs": ["get"]}}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/query", strings.NewReader(body))
	req.Header.Set("X-Impersonate-User", "alice")
	req.Header.Set("X-Impersonate-Group", "developers")
	ws.handleQuery(rec, req)
	if capturedUser != "alice" {
		t.Errorf("expected Impersonate-User=alice, got %s", capturedUser)
	}
	if capturedGroup != "developers" {
		t.Errorf("expected Impersonate-Group=developers, got %s", capturedGroup)
	}
}

func TestSecurityHeaders(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := securityHeaders(inner)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	handler.ServeHTTP(rec, req)

	checks := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
		"Permissions-Policy":     "camera=(), microphone=(), geolocation=()",
	}
	for header, expected := range checks {
		if got := rec.Header().Get(header); got != expected {
			t.Errorf("expected %s=%s, got %s", header, expected, got)
		}
	}
}
