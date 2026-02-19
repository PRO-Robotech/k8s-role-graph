package rbacgraph

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RoleGraphReview is the internal (hub) representation of a role graph query.
type RoleGraphReview struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec   RoleGraphReviewSpec
	Status RoleGraphReviewStatus
}

// ---------- typed enums ----------

type MatchMode string

const (
	MatchModeAny MatchMode = "any"
	MatchModeAll MatchMode = "all"
)

type WildcardMode string

const (
	WildcardModeExpand WildcardMode = "expand"
	WildcardModeExact  WildcardMode = "exact"
)

type PodPhaseMode string

const (
	PodPhaseModeActive  PodPhaseMode = "active"
	PodPhaseModeAll     PodPhaseMode = "all"
	PodPhaseModeRunning PodPhaseMode = "running"

	DefaultMaxPodsPerSubject  = 20
	DefaultMaxWorkloadsPerPod = 10
)

type GraphNodeType string

const (
	GraphNodeTypeRole               GraphNodeType = "role"
	GraphNodeTypeClusterRole        GraphNodeType = "clusterRole"
	GraphNodeTypeRoleBinding        GraphNodeType = "roleBinding"
	GraphNodeTypeClusterRoleBinding GraphNodeType = "clusterRoleBinding"
	GraphNodeTypeUser               GraphNodeType = "user"
	GraphNodeTypeGroup              GraphNodeType = "group"
	GraphNodeTypeServiceAccount     GraphNodeType = "serviceAccount"
	GraphNodeTypePod                GraphNodeType = "pod"
	GraphNodeTypeWorkload           GraphNodeType = "workload"
	GraphNodeTypePodOverflow        GraphNodeType = "podOverflow"
	GraphNodeTypeWorkloadOverflow   GraphNodeType = "workloadOverflow"
)

type GraphEdgeType string

const (
	GraphEdgeTypeAggregates GraphEdgeType = "aggregates"
	GraphEdgeTypeGrants     GraphEdgeType = "grants"
	GraphEdgeTypeSubjects   GraphEdgeType = "subjects"
	GraphEdgeTypeRunsAs     GraphEdgeType = "runsAs"
	GraphEdgeTypeOwnedBy    GraphEdgeType = "ownedBy"
)

// ---------- spec / status types ----------

type RoleGraphReviewSpec struct {
	Selector            Selector
	MatchMode           MatchMode
	WildcardMode        WildcardMode
	IncludeRuleMetadata bool
	NamespaceScope      NamespaceScope
	IncludePods         bool
	IncludeWorkloads    bool
	PodPhaseMode        PodPhaseMode
	MaxPodsPerSubject   int
	MaxWorkloadsPerPod  int
	FilterPhantomAPIs   bool
}

type NamespaceScope struct {
	Namespaces []string
	Strict     bool
}

type Selector struct {
	APIGroups       []string
	Resources       []string
	Verbs           []string
	ResourceNames   []string
	NonResourceURLs []string
}

type RoleGraphReviewStatus struct {
	MatchedRoles     int
	MatchedBindings  int
	MatchedSubjects  int
	MatchedPods      int
	MatchedWorkloads int
	Warnings         []string
	KnownGaps        []string
	Graph            Graph
	ResourceMap      []ResourceMapRow
}

type Graph struct {
	Nodes []GraphNode
	Edges []GraphEdge
}

type GraphNode struct {
	ID                 string
	Type               GraphNodeType
	Name               string
	Namespace          string
	Aggregated         bool
	AggregationSources []string
	MatchedRuleRefs    []RuleRef
	Labels             map[string]string
	Annotations        map[string]string
	PodPhase           string
	WorkloadKind       string
	Synthetic          bool
	HiddenCount        int
}

type GraphEdge struct {
	ID       string
	From     string
	To       string
	Type     GraphEdgeType
	RuleRefs []RuleRef
	Explain  string
}

type RuleRef struct {
	APIVersion      string
	APIGroup        string
	Resource        string
	Subresource     string
	Verb            string
	ResourceNames   []string
	NonResourceURLs []string
	SourceObjectUID string
	SourceRuleIndex int
	Phantom         bool
	UnsupportedVerb bool
	ExpandedRefs    []RuleRef
}

type ResourceMapRow struct {
	APIGroup     string
	Resource     string
	Verb         string
	RoleCount    int
	BindingCount int
	SubjectCount int
}

// ---------- spec methods ----------
// SYNC: Keep EnsureDefaults/Validate in sync with pkg/apis/rbacgraph/v1alpha1/types.go

func (s *RoleGraphReviewSpec) EnsureDefaults() {
	if s.MatchMode == "" {
		s.MatchMode = MatchModeAny
	}
	if s.WildcardMode == "" {
		s.WildcardMode = WildcardModeExpand
	}
	if s.PodPhaseMode == "" {
		s.PodPhaseMode = PodPhaseModeActive
	}
	if s.MaxPodsPerSubject <= 0 {
		s.MaxPodsPerSubject = DefaultMaxPodsPerSubject
	}
	if s.MaxWorkloadsPerPod <= 0 {
		s.MaxWorkloadsPerPod = DefaultMaxWorkloadsPerPod
	}
}

func (s RoleGraphReviewSpec) Validate() error {
	if s.MatchMode != MatchModeAny && s.MatchMode != MatchModeAll {
		return fmt.Errorf("invalid matchMode %q", s.MatchMode)
	}
	if s.WildcardMode != WildcardModeExpand && s.WildcardMode != WildcardModeExact {
		return fmt.Errorf("invalid wildcardMode %q", s.WildcardMode)
	}
	podPhaseMode := s.PodPhaseMode
	if podPhaseMode == "" {
		podPhaseMode = PodPhaseModeActive
	}
	if podPhaseMode != PodPhaseModeActive && podPhaseMode != PodPhaseModeAll && podPhaseMode != PodPhaseModeRunning {
		return fmt.Errorf("invalid podPhaseMode %q", s.PodPhaseMode)
	}

	return nil
}

func (s *RoleGraphReviewSpec) NormalizeRuntimeFlags() []string {
	if s.IncludeWorkloads && !s.IncludePods {
		s.IncludePods = true

		return []string{"includeWorkloads=true requires includePods=true; includePods was enabled automatically"}
	}

	return nil
}
