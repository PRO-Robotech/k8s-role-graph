package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"k8s-role-graph/pkg/apis/rbacgraph/v1alpha1"
	"k8s-role-graph/pkg/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

//go:embed static/*
var staticFiles embed.FS

type webServer struct {
	logger      *log.Logger
	httpClient  *http.Client
	apiEndpoint string
}

func main() {
	var (
		listenAddr string
		kubeconfig string
	)
	flag.StringVar(&listenAddr, "listen", ":8080", "HTTP listen address")
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig. Empty means in-cluster")
	flag.Parse()

	logger := log.New(os.Stdout, "rbacgraph-web ", log.LstdFlags|log.LUTC)
	cfg, err := kube.ClientConfig(kubeconfig)
	if err != nil {
		logger.Fatalf("build kubernetes config: %v", err)
	}
	transport, err := rest.TransportFor(cfg)
	if err != nil {
		logger.Fatalf("build transport: %v", err)
	}

	ws := &webServer{
		logger: logger,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   20 * time.Second,
		},
		apiEndpoint: strings.TrimRight(cfg.Host, "/") + "/apis/" + v1alpha1.GroupName + "/" + v1alpha1.Version + "/" + v1alpha1.Resource,
	}

	mux := http.NewServeMux()
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		logger.Fatalf("build static fs: %v", err)
	}
	fileServer := http.FileServer(http.FS(staticFS))
	mux.Handle("/static/", http.StripPrefix("/static/", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Cache-Control", "public, max-age=3600")
		fileServer.ServeHTTP(rw, req)
	})))
	mux.HandleFunc("/", ws.handleIndex)
	mux.HandleFunc("/api/query", ws.handleQuery)
	mux.HandleFunc("/api/health", ws.handleHealth)

	srv := &http.Server{
		Addr:              listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	logger.Printf("starting web server on %s", listenAddr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("listen: %v", err)
	}
}

func (w *webServer) handleIndex(rw http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		http.NotFound(rw, req)
		return
	}
	rw.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	rw.Header().Set("Pragma", "no-cache")
	rw.Header().Set("Expires", "0")
	content, err := staticFiles.ReadFile("static/index.html")
	if err != nil {
		http.Error(rw, fmt.Sprintf("failed to read index: %v", err), http.StatusInternalServerError)
		return
	}
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = rw.Write(content)
}

func (w *webServer) handleHealth(rw http.ResponseWriter, _ *http.Request) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write([]byte(`{"status":"ok"}`))
}

func (w *webServer) handleQuery(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(req.Body, 1<<20))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	review, err := decodeClientRequest(body)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	review.EnsureDefaults()

	payload, err := json.Marshal(review)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	httpReq, err := http.NewRequestWithContext(req.Context(), http.MethodPost, w.apiEndpoint, bytes.NewReader(payload))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Propagate impersonation headers from the browser request.
	// K8s requires Impersonate-User when Impersonate-Group is set.
	impUser := req.Header.Get("X-Impersonate-User")
	impGroup := req.Header.Get("X-Impersonate-Group")
	if impUser == "" && impGroup != "" {
		http.Error(rw, "Impersonate-Group requires Impersonate-User to be set", http.StatusBadRequest)
		return
	}
	if impUser != "" {
		httpReq.Header.Set("Impersonate-User", impUser)
	}
	if impGroup != "" {
		httpReq.Header.Set("Impersonate-Group", impGroup)
	}

	resp, err := w.httpClient.Do(httpReq)
	if err != nil {
		http.Error(rw, fmt.Sprintf("query API server failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	apiResponse, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadGateway)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(resp.StatusCode)
	_, _ = rw.Write(apiResponse)
}

func decodeClientRequest(body []byte) (*v1alpha1.RoleGraphReview, error) {
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil, fmt.Errorf("empty request body")
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode request JSON: %w", err)
	}

	review := &v1alpha1.RoleGraphReview{}
	if _, ok := raw["spec"]; ok {
		if err := json.Unmarshal(body, review); err != nil {
			return nil, fmt.Errorf("decode roleGraphReview: %w", err)
		}
		return review, nil
	}

	if _, ok := raw["selector"]; ok {
		if err := json.Unmarshal(body, &review.Spec); err != nil {
			return nil, fmt.Errorf("decode roleGraphReviewSpec: %w", err)
		}
		review.ObjectMeta = metav1.ObjectMeta{Name: "web-query"}
		return review, nil
	}

	if err := json.Unmarshal(body, &review.Spec.Selector); err != nil {
		return nil, fmt.Errorf("decode selector: %w", err)
	}
	review.ObjectMeta = metav1.ObjectMeta{Name: "web-query"}
	return review, nil
}
