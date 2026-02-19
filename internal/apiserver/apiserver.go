package apiserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/healthz"
	"k8s.io/klog/v2"

	"k8s-role-graph/internal/authz"
	"k8s-role-graph/internal/engine"
	"k8s-role-graph/internal/indexer"
	reviewstorage "k8s-role-graph/internal/registry/rolegraphreview"
	"k8s-role-graph/pkg/apis/rbacgraph"
	"k8s-role-graph/pkg/apis/rbacgraph/v1alpha1"
)

type Config struct {
	GenericConfig *genericapiserver.RecommendedConfig
	Indexer       *indexer.Indexer
	Engine        *engine.Engine
	AuthzResolver authz.ScopeResolver // nil when --enforce-caller-scope is disabled
}

type completedConfig struct {
	GenericConfig genericapiserver.CompletedConfig
	Indexer       *indexer.Indexer
	Engine        *engine.Engine
	AuthzResolver authz.ScopeResolver
}

type CompletedConfig struct {
	*completedConfig
}

func (cfg *Config) Complete() CompletedConfig {
	c := completedConfig{
		GenericConfig: cfg.GenericConfig.Complete(),
		Indexer:       cfg.Indexer,
		Engine:        cfg.Engine,
		AuthzResolver: cfg.AuthzResolver,
	}

	return CompletedConfig{&c}
}

type RbacGraphServer struct {
	GenericAPIServer *genericapiserver.GenericAPIServer
	Indexer          *indexer.Indexer
}

func (c CompletedConfig) New() (*RbacGraphServer, error) {
	genericServer, err := c.GenericConfig.New("rbacgraph-apiserver", genericapiserver.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}

	s := &RbacGraphServer{
		GenericAPIServer: genericServer,
		Indexer:          c.Indexer,
	}

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(rbacgraph.GroupName, Scheme, ParameterCodec, Codecs)
	v1alpha1storage := map[string]rest.Storage{}
	v1alpha1storage["rolegraphreviews"] = reviewstorage.NewREST(c.Engine, c.Indexer, Scheme, c.AuthzResolver)
	apiGroupInfo.VersionedResourcesStorageMap[v1alpha1.Version] = v1alpha1storage

	if err := s.GenericAPIServer.InstallAPIGroup(&apiGroupInfo); err != nil {
		return nil, fmt.Errorf("install API group: %w", err)
	}

	s.GenericAPIServer.AddPostStartHookOrDie("start-rbacgraph-indexer", func(hookCtx genericapiserver.PostStartHookContext) error {
		go func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go func() {
				<-hookCtx.Done()
				cancel()
			}()
			if err := c.Indexer.Start(ctx); err != nil {
				klog.Errorf("indexer failed: %v", err)
			}
		}()

		return nil
	})

	if err := s.GenericAPIServer.AddReadyzChecks(indexerHealthChecker{indexer: c.Indexer}); err != nil {
		return nil, fmt.Errorf("add readyz check: %w", err)
	}

	return s, nil
}

type indexerHealthChecker struct {
	indexer *indexer.Indexer
}

func (c indexerHealthChecker) Name() string {
	return "rbacgraph-indexer"
}

func (c indexerHealthChecker) Check(_ *http.Request) error {
	if !c.indexer.IsReady() {
		return errors.New("indexer not ready")
	}

	return nil
}

var _ healthz.HealthChecker = indexerHealthChecker{}
