package v1alpha1

import (
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"

	"k8s-role-graph/pkg/apis/rbacgraph"
)

func init() {
	localSchemeBuilder.Register(RegisterConversions)
}

var localSchemeBuilder = &SchemeBuilder

func RegisterConversions(s *runtime.Scheme) error {
	if err := s.AddConversionFunc((*RoleGraphReview)(nil), (*rbacgraph.RoleGraphReview)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_RoleGraphReview_To_rbacgraph_RoleGraphReview(a.(*RoleGraphReview), b.(*rbacgraph.RoleGraphReview), scope)
	}); err != nil {
		return err
	}
	if err := s.AddConversionFunc((*rbacgraph.RoleGraphReview)(nil), (*RoleGraphReview)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_rbacgraph_RoleGraphReview_To_v1alpha1_RoleGraphReview(a.(*rbacgraph.RoleGraphReview), b.(*RoleGraphReview), scope)
	}); err != nil {
		return err
	}
	return nil
}

func Convert_v1alpha1_RoleGraphReview_To_rbacgraph_RoleGraphReview(in *RoleGraphReview, out *rbacgraph.RoleGraphReview, _ conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta

	// Spec
	out.Spec.MatchMode = string(in.Spec.MatchMode)
	out.Spec.IncludeRuleMetadata = in.Spec.IncludeRuleMetadata
	out.Spec.IncludePods = in.Spec.IncludePods
	out.Spec.IncludeWorkloads = in.Spec.IncludeWorkloads
	out.Spec.PodPhaseMode = string(in.Spec.PodPhaseMode)
	out.Spec.MaxPodsPerSubject = in.Spec.MaxPodsPerSubject
	out.Spec.MaxWorkloadsPerPod = in.Spec.MaxWorkloadsPerPod

	out.Spec.Selector.APIGroups = in.Spec.Selector.APIGroups
	out.Spec.Selector.Resources = in.Spec.Selector.Resources
	out.Spec.Selector.Verbs = in.Spec.Selector.Verbs
	out.Spec.Selector.ResourceNames = in.Spec.Selector.ResourceNames
	out.Spec.Selector.NonResourceURLs = in.Spec.Selector.NonResourceURLs

	out.Spec.NamespaceScope.Namespaces = in.Spec.NamespaceScope.Namespaces
	out.Spec.NamespaceScope.Strict = in.Spec.NamespaceScope.Strict

	// Status
	convertStatusV1alpha1ToInternal(&in.Status, &out.Status)
	return nil
}

func Convert_rbacgraph_RoleGraphReview_To_v1alpha1_RoleGraphReview(in *rbacgraph.RoleGraphReview, out *RoleGraphReview, _ conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta

	// Spec
	out.Spec.MatchMode = MatchMode(in.Spec.MatchMode)
	out.Spec.IncludeRuleMetadata = in.Spec.IncludeRuleMetadata
	out.Spec.IncludePods = in.Spec.IncludePods
	out.Spec.IncludeWorkloads = in.Spec.IncludeWorkloads
	out.Spec.PodPhaseMode = PodPhaseMode(in.Spec.PodPhaseMode)
	out.Spec.MaxPodsPerSubject = in.Spec.MaxPodsPerSubject
	out.Spec.MaxWorkloadsPerPod = in.Spec.MaxWorkloadsPerPod

	out.Spec.Selector.APIGroups = in.Spec.Selector.APIGroups
	out.Spec.Selector.Resources = in.Spec.Selector.Resources
	out.Spec.Selector.Verbs = in.Spec.Selector.Verbs
	out.Spec.Selector.ResourceNames = in.Spec.Selector.ResourceNames
	out.Spec.Selector.NonResourceURLs = in.Spec.Selector.NonResourceURLs

	out.Spec.NamespaceScope.Namespaces = in.Spec.NamespaceScope.Namespaces
	out.Spec.NamespaceScope.Strict = in.Spec.NamespaceScope.Strict

	// Status
	convertStatusInternalToV1alpha1(&in.Status, &out.Status)
	return nil
}

func convertStatusV1alpha1ToInternal(in *RoleGraphReviewStatus, out *rbacgraph.RoleGraphReviewStatus) {
	out.MatchedRoles = in.MatchedRoles
	out.MatchedBindings = in.MatchedBindings
	out.MatchedSubjects = in.MatchedSubjects
	out.MatchedPods = in.MatchedPods
	out.MatchedWorkloads = in.MatchedWorkloads
	out.Warnings = in.Warnings
	out.KnownGaps = in.KnownGaps

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
			out.Graph.Nodes[i].MatchedRuleRefs[j] = convertRuleRefToInternal(r)
		}
	}
	out.Graph.Edges = make([]rbacgraph.GraphEdge, len(in.Graph.Edges))
	for i, e := range in.Graph.Edges {
		out.Graph.Edges[i] = rbacgraph.GraphEdge{
			ID: e.ID, From: e.From, To: e.To, Type: string(e.Type), Explain: e.Explain,
		}
		out.Graph.Edges[i].RuleRefs = make([]rbacgraph.RuleRef, len(e.RuleRefs))
		for j, r := range e.RuleRefs {
			out.Graph.Edges[i].RuleRefs[j] = convertRuleRefToInternal(r)
		}
	}
	out.ResourceMap = make([]rbacgraph.ResourceMapRow, len(in.ResourceMap))
	for i, r := range in.ResourceMap {
		out.ResourceMap[i] = rbacgraph.ResourceMapRow{
			APIGroup: r.APIGroup, Resource: r.Resource, Verb: r.Verb,
			RoleCount: r.RoleCount, BindingCount: r.BindingCount, SubjectCount: r.SubjectCount,
		}
	}
}

func convertStatusInternalToV1alpha1(in *rbacgraph.RoleGraphReviewStatus, out *RoleGraphReviewStatus) {
	out.MatchedRoles = in.MatchedRoles
	out.MatchedBindings = in.MatchedBindings
	out.MatchedSubjects = in.MatchedSubjects
	out.MatchedPods = in.MatchedPods
	out.MatchedWorkloads = in.MatchedWorkloads
	out.Warnings = in.Warnings
	out.KnownGaps = in.KnownGaps

	out.Graph.Nodes = make([]GraphNode, len(in.Graph.Nodes))
	for i, n := range in.Graph.Nodes {
		out.Graph.Nodes[i] = GraphNode{
			ID: n.ID, Type: GraphNodeType(n.Type), Name: n.Name, Namespace: n.Namespace,
			Aggregated: n.Aggregated, AggregationSources: n.AggregationSources,
			Labels: n.Labels, Annotations: n.Annotations,
			PodPhase: n.PodPhase, WorkloadKind: n.WorkloadKind,
			Synthetic: n.Synthetic, HiddenCount: n.HiddenCount,
		}
		out.Graph.Nodes[i].MatchedRuleRefs = make([]RuleRef, len(n.MatchedRuleRefs))
		for j, r := range n.MatchedRuleRefs {
			out.Graph.Nodes[i].MatchedRuleRefs[j] = convertRuleRefToV1alpha1(r)
		}
	}
	out.Graph.Edges = make([]GraphEdge, len(in.Graph.Edges))
	for i, e := range in.Graph.Edges {
		out.Graph.Edges[i] = GraphEdge{
			ID: e.ID, From: e.From, To: e.To, Type: GraphEdgeType(e.Type), Explain: e.Explain,
		}
		out.Graph.Edges[i].RuleRefs = make([]RuleRef, len(e.RuleRefs))
		for j, r := range e.RuleRefs {
			out.Graph.Edges[i].RuleRefs[j] = convertRuleRefToV1alpha1(r)
		}
	}
	out.ResourceMap = make([]ResourceMapRow, len(in.ResourceMap))
	for i, r := range in.ResourceMap {
		out.ResourceMap[i] = ResourceMapRow{
			APIGroup: r.APIGroup, Resource: r.Resource, Verb: r.Verb,
			RoleCount: r.RoleCount, BindingCount: r.BindingCount, SubjectCount: r.SubjectCount,
		}
	}
}

func convertRuleRefToInternal(in RuleRef) rbacgraph.RuleRef {
	return rbacgraph.RuleRef{
		APIVersion: in.APIVersion, APIGroup: in.APIGroup,
		Resource: in.Resource, Subresource: in.Subresource,
		Verb: in.Verb, ResourceNames: in.ResourceNames,
		NonResourceURLs: in.NonResourceURLs,
		SourceObjectUID: in.SourceObjectUID, SourceRuleIndex: in.SourceRuleIndex,
	}
}

func convertRuleRefToV1alpha1(in rbacgraph.RuleRef) RuleRef {
	return RuleRef{
		APIVersion: in.APIVersion, APIGroup: in.APIGroup,
		Resource: in.Resource, Subresource: in.Subresource,
		Verb: in.Verb, ResourceNames: in.ResourceNames,
		NonResourceURLs: in.NonResourceURLs,
		SourceObjectUID: in.SourceObjectUID, SourceRuleIndex: in.SourceRuleIndex,
	}
}
