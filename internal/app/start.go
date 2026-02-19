package app

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/apiserver/pkg/server"
	serveroptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/apiserver/pkg/util/compatibility"
	"k8s.io/client-go/kubernetes"

	internalserver "k8s-role-graph/internal/apiserver"
	"k8s-role-graph/internal/authz"
	"k8s-role-graph/internal/engine"
	"k8s-role-graph/internal/indexer"
	"k8s-role-graph/pkg/apis/rbacgraph/v1alpha1"
	"k8s-role-graph/pkg/kube"
)

type ServerOptions struct {
	RecommendedOptions *serveroptions.RecommendedOptions
	ResyncPeriod       time.Duration
	EnforceCallerScope bool

	StdOut io.Writer
	StdErr io.Writer
}

func NewServerOptions(out, errOut io.Writer) *ServerOptions {
	o := &ServerOptions{
		RecommendedOptions: serveroptions.NewRecommendedOptions(
			"",
			internalserver.Codecs.LegacyCodec(schema.GroupVersion{
				Group:   "rbacgraph.incloud.io",
				Version: "v1alpha1",
			}),
		),
		StdOut: out,
		StdErr: errOut,
	}
	o.RecommendedOptions.Etcd = nil
	o.RecommendedOptions.Admission = nil
	o.RecommendedOptions.Features.EnablePriorityAndFairness = false

	return o
}

func NewCommandStartServer(ctx context.Context, defaults *ServerOptions) *cobra.Command {
	o := defaults
	cmd := &cobra.Command{
		Use:   "rbacgraph-apiserver",
		Short: "Launch the rbacgraph aggregated API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}

			return o.Run(ctx)
		},
	}

	flags := cmd.Flags()
	o.RecommendedOptions.AddFlags(flags)
	flags.DurationVar(&o.ResyncPeriod, "resync-period", 0, "Informer resync period (0 = no periodic resync)")
	flags.BoolVar(&o.EnforceCallerScope, "enforce-caller-scope", false,
		"Restrict query results to RBAC objects the caller has permission to list")

	return cmd
}

// Complete fills in fields derived from other options. Currently a no-op because
// all configuration is self-contained; kept for cobra RunE convention.
func (o *ServerOptions) Complete() error {
	return nil
}

// Validate checks ServerOptions for consistency. Currently a no-op because
// ResyncPeriod=0 is valid (disables resync) and EnforceCallerScope is a boolean.
func (o *ServerOptions) Validate() error {
	return nil
}

func (o *ServerOptions) Run(ctx context.Context) error {
	serverConfig := server.NewRecommendedConfig(internalserver.Codecs)
	serverConfig.EffectiveVersion = compatibility.DefaultBuildEffectiveVersion()

	if err := o.RecommendedOptions.ApplyTo(serverConfig); err != nil {
		return fmt.Errorf("apply recommended options: %w", err)
	}

	namer := openapi.NewDefinitionNamer(internalserver.Scheme)
	serverConfig.OpenAPIConfig = server.DefaultOpenAPIConfig(v1alpha1.GetOpenAPIDefinitionsWithEnums, namer)
	serverConfig.OpenAPIConfig.Info.Title = "RbacGraph"
	serverConfig.OpenAPIConfig.Info.Version = v1alpha1.Version
	serverConfig.OpenAPIV3Config = server.DefaultOpenAPIV3Config(v1alpha1.GetOpenAPIDefinitionsWithEnums, namer)
	serverConfig.OpenAPIV3Config.Info.Title = "RbacGraph"
	serverConfig.OpenAPIV3Config.Info.Version = v1alpha1.Version

	clientset, err := buildClientset(o.RecommendedOptions.CoreAPI.CoreAPIKubeconfigPath)
	if err != nil {
		return fmt.Errorf("build kubernetes clientset: %w", err)
	}

	eng := engine.New()
	idx := indexer.New(clientset, o.ResyncPeriod)

	var resolver authz.ScopeResolver
	if o.EnforceCallerScope {
		resolver = authz.NewLocalResolver(idx.Snapshot)
	}

	config := &internalserver.Config{
		GenericConfig: serverConfig,
		Indexer:       idx,
		Engine:        eng,
		AuthzResolver: resolver,
	}

	completedConfig := config.Complete()
	rbacGraphServer, err := completedConfig.New()
	if err != nil {
		return fmt.Errorf("create apiserver: %w", err)
	}

	return rbacGraphServer.GenericAPIServer.PrepareRun().RunWithContext(ctx)
}

func buildClientset(kubeconfig string) (kubernetes.Interface, error) {
	cfg, err := kube.ClientConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("build client config: %w", err)
	}

	return kubernetes.NewForConfig(cfg)
}
