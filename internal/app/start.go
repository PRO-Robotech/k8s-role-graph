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
	"k8s-role-graph/internal/engine"
	"k8s-role-graph/internal/indexer"
	"k8s-role-graph/pkg/apis/rbacgraph/v1alpha1"
	"k8s-role-graph/pkg/kube"
)

// ServerOptions contains the state for the apiserver command.
type ServerOptions struct {
	RecommendedOptions *serveroptions.RecommendedOptions
	ResyncPeriod       time.Duration

	StdOut io.Writer
	StdErr io.Writer
}

// NewServerOptions returns default ServerOptions.
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

// NewCommandStartServer creates the cobra command for the apiserver.
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

	return cmd
}

// Complete fills in remaining defaults.
func (o *ServerOptions) Complete() error {
	return nil
}

// Validate checks option values.
func (o *ServerOptions) Validate() error {
	return nil
}

// Run starts the API server.
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

	// Build the kubernetes clientset for the indexer, using the same
	// --kubeconfig flag registered by RecommendedOptions.CoreAPI.
	clientset, err := buildClientset(o.RecommendedOptions.CoreAPI.CoreAPIKubeconfigPath)
	if err != nil {
		return fmt.Errorf("build kubernetes clientset: %w", err)
	}

	eng := engine.New()
	idx := indexer.New(clientset, o.ResyncPeriod)

	config := &internalserver.Config{
		GenericConfig: serverConfig,
		Indexer:       idx,
		Engine:        eng,
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
		return nil, err
	}
	return kubernetes.NewForConfig(cfg)
}
