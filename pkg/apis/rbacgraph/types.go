package rbacgraph

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RoleGraphReview is the internal (hub) representation of a role graph query.
type RoleGraphReview struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec   RoleGraphReviewSpec
	Status RoleGraphReviewStatus
}

type RoleGraphReviewSpec struct {
	Selector            Selector
	MatchMode           string
	IncludeRuleMetadata bool
	NamespaceScope      NamespaceScope
	IncludePods         bool
	IncludeWorkloads    bool
	PodPhaseMode        string
	MaxPodsPerSubject   int
	MaxWorkloadsPerPod  int
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
	Type               string
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
	Type     string
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
}

type ResourceMapRow struct {
	APIGroup     string
	Resource     string
	Verb         string
	RoleCount    int
	BindingCount int
	SubjectCount int
}
