package rolegraphreview

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/rest"

	"k8s-role-graph/internal/engine"
	"k8s-role-graph/internal/indexer"
	"k8s-role-graph/pkg/apis/rbacgraph"
	"k8s-role-graph/pkg/apis/rbacgraph/v1alpha1"
)

// REST implements rest.Storage and rest.Creater for RoleGraphReview.
// It is a create-only resource — no persistence.
type REST struct {
	engine  *engine.Engine
	indexer *indexer.Indexer
	scheme  *runtime.Scheme
}

var _ rest.Storage = &REST{}
var _ rest.Creater = &REST{}
var _ rest.SingularNameProvider = &REST{}

func NewREST(eng *engine.Engine, idx *indexer.Indexer, scheme *runtime.Scheme) *REST {
	return &REST{
		engine:  eng,
		indexer: idx,
		scheme:  scheme,
	}
}

// New returns a new internal RoleGraphReview object.
func (r *REST) New() runtime.Object {
	return &rbacgraph.RoleGraphReview{}
}

// Destroy is a no-op.
func (r *REST) Destroy() {}

// NamespaceScoped returns false — RoleGraphReview is cluster-scoped.
func (r *REST) NamespaceScoped() bool {
	return false
}

// GetSingularName returns the singular name of the resource.
func (r *REST) GetSingularName() string {
	return "rolegraphreview"
}

// Create handles POST /apis/rbacgraph.incloud.io/v1alpha1/rolegraphreviews.
// It converts the internal RoleGraphReview spec to v1alpha1, executes the engine query,
// and populates the status.
func (r *REST) Create(ctx context.Context, obj runtime.Object, _ rest.ValidateObjectFunc, _ *metav1.CreateOptions) (runtime.Object, error) {
	review, ok := obj.(*rbacgraph.RoleGraphReview)
	if !ok {
		return nil, fmt.Errorf("unexpected object type: %T", obj)
	}

	// Convert internal spec to v1alpha1 for engine consumption.
	v1Spec := convertSpecToV1alpha1(review.Spec)
	v1Spec.EnsureDefaults()
	if err := v1Spec.Validate(); err != nil {
		return nil, err
	}

	snapshot := r.indexer.Snapshot()
	v1Status := r.engine.Query(snapshot, v1Spec)

	// Convert v1alpha1 status back to internal.
	review.Status = convertStatusToInternal(v1Status)
	review.CreationTimestamp = metav1.Now()
	return review, nil
}

func convertSpecToV1alpha1(in rbacgraph.RoleGraphReviewSpec) v1alpha1.RoleGraphReviewSpec {
	return v1alpha1.RoleGraphReviewSpec{
		Selector: v1alpha1.Selector{
			APIGroups:       in.Selector.APIGroups,
			Resources:       in.Selector.Resources,
			Verbs:           in.Selector.Verbs,
			ResourceNames:   in.Selector.ResourceNames,
			NonResourceURLs: in.Selector.NonResourceURLs,
		},
		MatchMode:           v1alpha1.MatchMode(in.MatchMode),
		IncludeRuleMetadata: in.IncludeRuleMetadata,
		NamespaceScope: v1alpha1.NamespaceScope{
			Namespaces: in.NamespaceScope.Namespaces,
			Strict:     in.NamespaceScope.Strict,
		},
		IncludePods:        in.IncludePods,
		IncludeWorkloads:   in.IncludeWorkloads,
		PodPhaseMode:       v1alpha1.PodPhaseMode(in.PodPhaseMode),
		MaxPodsPerSubject:  in.MaxPodsPerSubject,
		MaxWorkloadsPerPod: in.MaxWorkloadsPerPod,
	}
}

func convertStatusToInternal(in v1alpha1.RoleGraphReviewStatus) rbacgraph.RoleGraphReviewStatus {
	out := rbacgraph.RoleGraphReviewStatus{
		MatchedRoles:     in.MatchedRoles,
		MatchedBindings:  in.MatchedBindings,
		MatchedSubjects:  in.MatchedSubjects,
		MatchedPods:      in.MatchedPods,
		MatchedWorkloads: in.MatchedWorkloads,
		Warnings:         in.Warnings,
		KnownGaps:        in.KnownGaps,
	}
	out.Graph.Nodes = make([]rbacgraph.GraphNode, len(in.Graph.Nodes))
	for i, n := range in.Graph.Nodes {
		out.Graph.Nodes[i] = rbacgraph.GraphNode{
			ID: n.ID, Type: string(n.Type), Name: n.Name, Namespace: n.Namespace,
			Aggregated: n.Aggregated, AggregationSources: n.AggregationSources,
			Labels: n.Labels, Annotations: n.Annotations,
			PodPhase: n.PodPhase, WorkloadKind: n.WorkloadKind,
			Synthetic: n.Synthetic, HiddenCount: n.HiddenCount,
		}
		out.Graph.Nodes[i].MatchedRuleRefs = make([]rbacgraph.RuleRef, len(n.MatchedRuleRefs))
		for j, r := range n.MatchedRuleRefs {
			out.Graph.Nodes[i].MatchedRuleRefs[j] = rbacgraph.RuleRef{
				APIVersion: r.APIVersion, APIGroup: r.APIGroup,
				Resource: r.Resource, Subresource: r.Subresource,
				Verb: r.Verb, ResourceNames: r.ResourceNames,
				NonResourceURLs: r.NonResourceURLs,
				SourceObjectUID: r.SourceObjectUID, SourceRuleIndex: r.SourceRuleIndex,
			}
		}
	}
	out.Graph.Edges = make([]rbacgraph.GraphEdge, len(in.Graph.Edges))
	for i, e := range in.Graph.Edges {
		out.Graph.Edges[i] = rbacgraph.GraphEdge{
			ID: e.ID, From: e.From, To: e.To, Type: string(e.Type), Explain: e.Explain,
		}
		out.Graph.Edges[i].RuleRefs = make([]rbacgraph.RuleRef, len(e.RuleRefs))
		for j, r := range e.RuleRefs {
			out.Graph.Edges[i].RuleRefs[j] = rbacgraph.RuleRef{
				APIVersion: r.APIVersion, APIGroup: r.APIGroup,
				Resource: r.Resource, Subresource: r.Subresource,
				Verb: r.Verb, ResourceNames: r.ResourceNames,
				NonResourceURLs: r.NonResourceURLs,
				SourceObjectUID: r.SourceObjectUID, SourceRuleIndex: r.SourceRuleIndex,
			}
		}
	}
	out.ResourceMap = make([]rbacgraph.ResourceMapRow, len(in.ResourceMap))
	for i, r := range in.ResourceMap {
		out.ResourceMap[i] = rbacgraph.ResourceMapRow{
			APIGroup: r.APIGroup, Resource: r.Resource, Verb: r.Verb,
			RoleCount: r.RoleCount, BindingCount: r.BindingCount, SubjectCount: r.SubjectCount,
		}
	}
	return out
}
