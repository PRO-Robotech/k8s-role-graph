package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s-role-graph/internal/indexer"
	api "k8s-role-graph/pkg/apis/rbacgraph/v1alpha1"
)

func TestQuery_BuildsGraph(t *testing.T) {
	snapshot := &indexer.Snapshot{
		BuiltAt:           time.Now(),
		RolesByID:         map[indexer.RoleID]*indexer.RoleRecord{},
		BindingsByRoleRef: map[indexer.RoleRefKey][]*indexer.BindingRecord{},
		RoleIDsByVerb:     map[string]map[indexer.RoleID]struct{}{},
		RoleIDsByResource: map[string]map[indexer.RoleID]struct{}{},
		RoleIDsByAPIGroup: map[string]map[indexer.RoleID]struct{}{},
		AllRoleIDs:        []indexer.RoleID{},
	}

	role := &indexer.RoleRecord{
		UID:       types.UID("r1"),
		Kind:      indexer.KindClusterRole,
		Name:      "exec-role",
		Namespace: "",
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{""},
			Resources: []string{"pods/exec"},
			Verbs:     []string{"create"},
		}},
	}

	roleID := indexer.RoleID("clusterrole:exec-role")
	snapshot.RolesByID[roleID] = role
	snapshot.AllRoleIDs = []indexer.RoleID{roleID}
	snapshot.RoleIDsByAPIGroup[""] = map[indexer.RoleID]struct{}{roleID: {}}
	snapshot.RoleIDsByResource["pods/exec"] = map[indexer.RoleID]struct{}{roleID: {}}
	snapshot.RoleIDsByVerb["create"] = map[indexer.RoleID]struct{}{roleID: {}}

	binding := &indexer.BindingRecord{
		UID:      types.UID("b1"),
		Kind:     indexer.KindClusterRoleBinding,
		Name:     "bind-exec",
		RoleRef:  indexer.RoleRefKey{Kind: indexer.KindClusterRole, Name: "exec-role"},
		Subjects: []rbacv1.Subject{{Kind: indexer.SubjectKindUser, Name: "alice"}},
	}
	snapshot.BindingsByRoleRef[binding.RoleRef] = []*indexer.BindingRecord{binding}

	e := New()
	status := e.Query(snapshot, api.RoleGraphReviewSpec{
		Selector: api.Selector{
			APIGroups: []string{""},
			Resources: []string{"pods/exec"},
			Verbs:     []string{"create"},
		},
		MatchMode:           api.MatchModeAny,
		IncludeRuleMetadata: true,
	})

	if status.MatchedRoles != 1 {
		t.Fatalf("expected 1 matched role, got %d", status.MatchedRoles)
	}
	if status.MatchedBindings != 1 {
		t.Fatalf("expected 1 matched binding, got %d", status.MatchedBindings)
	}
	if status.MatchedSubjects != 1 {
		t.Fatalf("expected 1 matched subject, got %d", status.MatchedSubjects)
	}
	if len(status.Graph.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(status.Graph.Nodes))
	}
	if len(status.Graph.Edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(status.Graph.Edges))
	}
	if len(status.ResourceMap) == 0 {
		t.Fatalf("expected resource map entries")
	}
}

func TestQuery_AnnotatesAggregatedClusterRoles(t *testing.T) {
	snapshot := &indexer.Snapshot{
		BuiltAt:               time.Now(),
		RolesByID:             map[indexer.RoleID]*indexer.RoleRecord{},
		BindingsByRoleRef:     map[indexer.RoleRefKey][]*indexer.BindingRecord{},
		AggregatedRoleSources: map[indexer.RoleID][]indexer.RoleID{},
		RoleIDsByVerb:         map[string]map[indexer.RoleID]struct{}{},
		RoleIDsByResource:     map[string]map[indexer.RoleID]struct{}{},
		RoleIDsByAPIGroup:     map[string]map[indexer.RoleID]struct{}{},
		AllRoleIDs:            []indexer.RoleID{},
	}

	sourceRole := &indexer.RoleRecord{
		UID:  types.UID("src"),
		Kind: indexer.KindClusterRole,
		Name: "aggregate-to-edit-source",
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{""},
			Resources: []string{"pods/exec"},
			Verbs:     []string{"get"},
		}},
	}
	aggregatedRole := &indexer.RoleRecord{
		UID:  types.UID("agg"),
		Kind: indexer.KindClusterRole,
		Name: "edit",
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{""},
			Resources: []string{"pods/exec"},
			Verbs:     []string{"get"},
		}},
	}

	sourceRoleID := indexer.RoleID("clusterrole:aggregate-to-edit-source")
	aggregatedRoleID := indexer.RoleID("clusterrole:edit")
	snapshot.RolesByID[sourceRoleID] = sourceRole
	snapshot.RolesByID[aggregatedRoleID] = aggregatedRole
	snapshot.AllRoleIDs = []indexer.RoleID{aggregatedRoleID}
	snapshot.AggregatedRoleSources[aggregatedRoleID] = []indexer.RoleID{sourceRoleID}
	snapshot.RoleIDsByAPIGroup[""] = map[indexer.RoleID]struct{}{aggregatedRoleID: {}}
	snapshot.RoleIDsByResource["pods/exec"] = map[indexer.RoleID]struct{}{aggregatedRoleID: {}}
	snapshot.RoleIDsByVerb["get"] = map[indexer.RoleID]struct{}{aggregatedRoleID: {}}

	binding := &indexer.BindingRecord{
		UID:      types.UID("b1"),
		Kind:     indexer.KindClusterRoleBinding,
		Name:     "bind-edit",
		RoleRef:  indexer.RoleRefKey{Kind: indexer.KindClusterRole, Name: "edit"},
		Subjects: []rbacv1.Subject{{Kind: indexer.SubjectKindUser, Name: "bob"}},
	}
	snapshot.BindingsByRoleRef[binding.RoleRef] = []*indexer.BindingRecord{binding}

	e := New()
	status := e.Query(snapshot, api.RoleGraphReviewSpec{
		Selector: api.Selector{
			APIGroups: []string{""},
			Resources: []string{"pods/exec"},
			Verbs:     []string{"get"},
		},
		MatchMode:           api.MatchModeAny,
		IncludeRuleMetadata: true,
	})

	var hasAggregatedNode bool
	var hasAggregatesEdge bool
	for _, node := range status.Graph.Nodes {
		if node.Name != "edit" {
			continue
		}
		hasAggregatedNode = node.Aggregated && len(node.AggregationSources) == 1 && node.AggregationSources[0] == string(sourceRoleID)
	}
	for _, edge := range status.Graph.Edges {
		if edge.Type == api.GraphEdgeTypeAggregates && strings.Contains(edge.ID, "aggregate-to-edit-source") && strings.Contains(edge.ID, "edit") {
			hasAggregatesEdge = true
		}
	}

	if !hasAggregatedNode {
		t.Fatalf("expected aggregated node metadata for edit clusterrole")
	}
	if !hasAggregatesEdge {
		t.Fatalf("expected aggregates edge between source and target clusterrole")
	}
}

func TestQuery_RuntimeChainServiceAccountToWorkload(t *testing.T) {
	snapshot := runtimeSnapshotForTests()
	e := New()

	status := e.Query(snapshot, api.RoleGraphReviewSpec{
		Selector: api.Selector{
			Resources: []string{"pods/exec"},
			Verbs:     []string{"get"},
		},
		IncludePods:      true,
		IncludeWorkloads: true,
		PodPhaseMode:     api.PodPhaseModeActive,
	})

	if status.MatchedPods != 1 {
		t.Fatalf("expected 1 matched pod, got %d", status.MatchedPods)
	}
	if status.MatchedWorkloads != 2 {
		t.Fatalf("expected 2 matched workloads, got %d", status.MatchedWorkloads)
	}

	assertHasNodeType(t, status.Graph.Nodes, api.GraphNodeTypePod)
	assertHasNodeType(t, status.Graph.Nodes, api.GraphNodeTypeWorkload)
	assertHasEdgeType(t, status.Graph.Edges, api.GraphEdgeTypeRunsAs)
	assertHasEdgeType(t, status.Graph.Edges, api.GraphEdgeTypeOwnedBy)
}

func TestQuery_RuntimeIncludeWorkloadsAutoEnablesPods(t *testing.T) {
	snapshot := runtimeSnapshotForTests()
	e := New()

	status := e.Query(snapshot, api.RoleGraphReviewSpec{
		Selector: api.Selector{
			Resources: []string{"pods/exec"},
			Verbs:     []string{"get"},
		},
		IncludeWorkloads: true,
	})

	if status.MatchedPods == 0 {
		t.Fatalf("expected pods to be included when includeWorkloads=true")
	}
	if !contains(status.Warnings, "includeWorkloads=true requires includePods=true; includePods was enabled automatically") {
		t.Fatalf("expected warning about auto-enabled includePods, warnings=%v", status.Warnings)
	}
}

func TestQuery_RuntimeLimitsAndPhaseFilter(t *testing.T) {
	snapshot := runtimeSnapshotForTests()
	saKey := indexer.ServiceAccountKey{Namespace: "team", Name: "demo-sa"}
	snapshot.PodsByServiceAccount[saKey] = append(snapshot.PodsByServiceAccount[saKey], &indexer.PodRecord{
		UID:                types.UID("pod-2"),
		Namespace:          "team",
		Name:               "pending-pod",
		ServiceAccountName: "demo-sa",
		Phase:              corev1.PodPending,
		OwnerReferences:    nil,
	})
	snapshot.PodsByServiceAccount[saKey] = append(snapshot.PodsByServiceAccount[saKey], &indexer.PodRecord{
		UID:                types.UID("pod-3"),
		Namespace:          "team",
		Name:               "running-pod-2",
		ServiceAccountName: "demo-sa",
		Phase:              corev1.PodRunning,
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion: "apps/v1",
			Kind:       "ReplicaSet",
			Name:       "demo-rs",
			UID:        types.UID("rs-1"),
		}},
	})

	e := New()
	status := e.Query(snapshot, api.RoleGraphReviewSpec{
		Selector: api.Selector{
			Resources: []string{"pods/exec"},
			Verbs:     []string{"get"},
		},
		IncludePods:         true,
		IncludeWorkloads:    true,
		PodPhaseMode:        api.PodPhaseModeRunning,
		MaxPodsPerSubject:   1,
		MaxWorkloadsPerPod:  1,
		IncludeRuleMetadata: true,
	})

	if status.MatchedPods != 1 {
		t.Fatalf("expected running filter + maxPods limit to keep 1 pod, got %d", status.MatchedPods)
	}
	if status.MatchedWorkloads != 1 {
		t.Fatalf("expected maxWorkloads limit to keep 1 workload, got %d", status.MatchedWorkloads)
	}
	assertHasNodeType(t, status.Graph.Nodes, api.GraphNodeTypePodOverflow)
	assertHasNodeType(t, status.Graph.Nodes, api.GraphNodeTypeWorkloadOverflow)
}

func TestQuery_NamespaceScopeStrictFalseKeepsClusterScopedBindings(t *testing.T) {
	snapshot := runtimeSnapshotForTests()
	e := New()

	status := e.Query(snapshot, api.RoleGraphReviewSpec{
		Selector: api.Selector{
			Resources: []string{"pods/exec"},
			Verbs:     []string{"get"},
		},
		NamespaceScope: api.NamespaceScope{
			Namespaces: []string{"team"},
			Strict:     false,
		},
	})

	if status.MatchedRoles != 1 {
		t.Fatalf("expected cluster-scoped role to remain visible with strict=false, got matchedRoles=%d", status.MatchedRoles)
	}
	if status.MatchedBindings != 1 {
		t.Fatalf("expected cluster-scoped binding to remain visible with strict=false, got matchedBindings=%d", status.MatchedBindings)
	}
}

func TestQuery_NamespaceScopeStrictTrueDropsClusterScopedRoleWithoutNamespacedBindings(t *testing.T) {
	snapshot := runtimeSnapshotForTests()
	e := New()

	status := e.Query(snapshot, api.RoleGraphReviewSpec{
		Selector: api.Selector{
			Resources: []string{"pods/exec"},
			Verbs:     []string{"get"},
		},
		NamespaceScope: api.NamespaceScope{
			Namespaces: []string{"team"},
			Strict:     true,
		},
	})

	if status.MatchedRoles != 0 {
		t.Fatalf("expected strict namespace scope to drop cluster-scoped role without namespaced bindings, got matchedRoles=%d", status.MatchedRoles)
	}
	if status.MatchedBindings != 0 {
		t.Fatalf("expected strict namespace scope to drop cluster-scoped bindings, got matchedBindings=%d", status.MatchedBindings)
	}
}

func TestQuery_NamespaceScopeStrictTrueKeepsClusterRoleWhenNamespacedBindingMatches(t *testing.T) {
	snapshot := runtimeSnapshotForTests()
	roleRef := indexer.RoleRefKey{Kind: indexer.KindClusterRole, Name: "exec-role"}
	snapshot.BindingsByRoleRef[roleRef] = []*indexer.BindingRecord{
		{
			UID:       types.UID("binding-ns-1"),
			Kind:      indexer.KindRoleBinding,
			Name:      "bind-exec-in-team",
			Namespace: "team",
			RoleRef:   roleRef,
			Subjects:  []rbacv1.Subject{{Kind: indexer.SubjectKindServiceAccount, Namespace: "team", Name: "demo-sa"}},
		},
	}

	e := New()
	status := e.Query(snapshot, api.RoleGraphReviewSpec{
		Selector: api.Selector{
			Resources: []string{"pods/exec"},
			Verbs:     []string{"get"},
		},
		NamespaceScope: api.NamespaceScope{
			Namespaces: []string{"team"},
			Strict:     true,
		},
	})

	if status.MatchedRoles != 1 {
		t.Fatalf("expected cluster-scoped role to remain when it reaches namespaced binding, got matchedRoles=%d", status.MatchedRoles)
	}
	if status.MatchedBindings != 1 {
		t.Fatalf("expected namespaced binding to remain in strict mode, got matchedBindings=%d", status.MatchedBindings)
	}
	if status.MatchedSubjects != 1 {
		t.Fatalf("expected namespaced subject to remain in strict mode, got matchedSubjects=%d", status.MatchedSubjects)
	}
}

func runtimeSnapshotForTests() *indexer.Snapshot {
	snapshot := &indexer.Snapshot{
		BuiltAt:               time.Now(),
		RolesByID:             map[indexer.RoleID]*indexer.RoleRecord{},
		BindingsByRoleRef:     map[indexer.RoleRefKey][]*indexer.BindingRecord{},
		AggregatedRoleSources: map[indexer.RoleID][]indexer.RoleID{},
		PodsByServiceAccount:  map[indexer.ServiceAccountKey][]*indexer.PodRecord{},
		WorkloadsByUID:        map[types.UID]*indexer.WorkloadRecord{},
		RoleIDsByVerb:         map[string]map[indexer.RoleID]struct{}{},
		RoleIDsByResource:     map[string]map[indexer.RoleID]struct{}{},
		RoleIDsByAPIGroup:     map[string]map[indexer.RoleID]struct{}{},
		AllRoleIDs:            []indexer.RoleID{},
	}

	role := &indexer.RoleRecord{
		UID:  types.UID("role-1"),
		Kind: indexer.KindClusterRole,
		Name: "exec-role",
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{""},
			Resources: []string{"pods/exec"},
			Verbs:     []string{"get"},
		}},
	}
	roleID := indexer.RoleID("clusterrole:exec-role")
	snapshot.RolesByID[roleID] = role
	snapshot.AllRoleIDs = []indexer.RoleID{roleID}
	snapshot.RoleIDsByAPIGroup[""] = map[indexer.RoleID]struct{}{roleID: {}}
	snapshot.RoleIDsByResource["pods/exec"] = map[indexer.RoleID]struct{}{roleID: {}}
	snapshot.RoleIDsByVerb["get"] = map[indexer.RoleID]struct{}{roleID: {}}

	binding := &indexer.BindingRecord{
		UID:      types.UID("binding-1"),
		Kind:     indexer.KindClusterRoleBinding,
		Name:     "bind-exec",
		RoleRef:  indexer.RoleRefKey{Kind: indexer.KindClusterRole, Name: "exec-role"},
		Subjects: []rbacv1.Subject{{Kind: indexer.SubjectKindServiceAccount, Namespace: "team", Name: "demo-sa"}},
	}
	snapshot.BindingsByRoleRef[binding.RoleRef] = []*indexer.BindingRecord{binding}

	snapshot.PodsByServiceAccount[indexer.ServiceAccountKey{Namespace: "team", Name: "demo-sa"}] = []*indexer.PodRecord{{
		UID:                types.UID("pod-1"),
		Namespace:          "team",
		Name:               "running-pod",
		ServiceAccountName: "demo-sa",
		Phase:              corev1.PodRunning,
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion: "apps/v1",
			Kind:       "ReplicaSet",
			Name:       "demo-rs",
			UID:        types.UID("rs-1"),
			Controller: boolPtr(true),
		}},
	}}

	snapshot.WorkloadsByUID[types.UID("rs-1")] = &indexer.WorkloadRecord{
		UID:        types.UID("rs-1"),
		APIVersion: "apps/v1",
		Kind:       "ReplicaSet",
		Namespace:  "team",
		Name:       "demo-rs",
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "demo-deploy",
			UID:        types.UID("deploy-1"),
			Controller: boolPtr(true),
		}},
	}
	snapshot.WorkloadsByUID[types.UID("deploy-1")] = &indexer.WorkloadRecord{
		UID:             types.UID("deploy-1"),
		APIVersion:      "apps/v1",
		Kind:            "Deployment",
		Namespace:       "team",
		Name:            "demo-deploy",
		OwnerReferences: nil,
	}

	return snapshot
}

func TestMergeRuleRefs_DedupesWithoutLoss(t *testing.T) {
	existing := []api.RuleRef{
		{
			APIGroup:        "",
			Resource:        "pods",
			Subresource:     "exec",
			Verb:            "get",
			SourceObjectUID: "uid-a",
			SourceRuleIndex: 1,
		},
	}
	incoming := []api.RuleRef{
		{
			APIGroup:        "",
			Resource:        "pods",
			Subresource:     "exec",
			Verb:            "get",
			SourceObjectUID: "uid-a",
			SourceRuleIndex: 1,
		},
		{
			APIGroup:        "",
			Resource:        "pods",
			Subresource:     "exec",
			Verb:            "create",
			SourceObjectUID: "uid-a",
			SourceRuleIndex: 2,
		},
		{
			APIGroup:      "",
			Resource:      "pods",
			Subresource:   "log",
			Verb:          "get",
			ResourceNames: []string{"p1"},
		},
	}

	merged := mergeRuleRefs(existing, incoming)
	if len(merged) != 3 {
		t.Fatalf("expected 3 unique refs, got %d (%#v)", len(merged), merged)
	}
}

func TestAccumulateResourceRows_AggregatesByStructuredKey(t *testing.T) {
	rows := make(map[resourceRowKey]*resourceAccumulator)
	roleA := indexer.RoleID("clusterrole:role-a")
	roleB := indexer.RoleID("clusterrole:role-b")

	rbacRef := api.RuleRef{
		APIGroup:    "*",
		Resource:    "pods",
		Subresource: "exec",
		Verb:        "get",
	}
	nonResourceRef := api.RuleRef{
		Verb:            "get",
		NonResourceURLs: []string{"/healthz"},
	}

	accumulateResourceRows(rows, []api.RuleRef{rbacRef}, roleA, "binding:a", "subject:a")
	accumulateResourceRows(rows, []api.RuleRef{rbacRef}, roleA, "binding:a", "subject:a")
	accumulateResourceRows(rows, []api.RuleRef{rbacRef}, roleB, "binding:b", "subject:b")
	accumulateResourceRows(rows, []api.RuleRef{nonResourceRef}, roleA, "", "")

	collapsed := collapseResourceRows(rows)
	if len(collapsed) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(collapsed))
	}

	var rbacRow *api.ResourceMapRow
	var nonResourceRow *api.ResourceMapRow
	for i := range collapsed {
		row := collapsed[i]
		if row.Resource == "pods/exec" && row.Verb == "get" {
			rbacRow = &row
			continue
		}
		if row.Resource == "/healthz" && row.Verb == "get" {
			nonResourceRow = &row
		}
	}

	if rbacRow == nil {
		t.Fatalf("expected pods/exec row in collapsed output: %#v", collapsed)
	}
	if rbacRow.RoleCount != 2 || rbacRow.BindingCount != 2 || rbacRow.SubjectCount != 2 {
		t.Fatalf("unexpected pods/exec counts: %#v", rbacRow)
	}
	if nonResourceRow == nil {
		t.Fatalf("expected non-resource row in collapsed output: %#v", collapsed)
	}
	if nonResourceRow.RoleCount != 1 || nonResourceRow.BindingCount != 0 || nonResourceRow.SubjectCount != 0 {
		t.Fatalf("unexpected non-resource counts: %#v", nonResourceRow)
	}
}

func TestQuery_GoldenJSONContracts(t *testing.T) {
	t.Parallel()
	update := os.Getenv("UPDATE_GOLDEN") == "1"
	engine := New()

	testCases := []struct {
		name     string
		snapshot *indexer.Snapshot
		spec     api.RoleGraphReviewSpec
	}{
		{
			name:     "basic-rbac",
			snapshot: basicSnapshotForGolden(),
			spec: api.RoleGraphReviewSpec{
				Selector: api.Selector{
					APIGroups: []string{""},
					Resources: []string{"pods/exec"},
					Verbs:     []string{"create"},
				},
				MatchMode:           api.MatchModeAny,
				IncludeRuleMetadata: true,
			},
		},
		{
			name:     "runtime-chain",
			snapshot: runtimeSnapshotForTests(),
			spec: api.RoleGraphReviewSpec{
				Selector: api.Selector{
					Resources: []string{"pods/exec"},
					Verbs:     []string{"get"},
				},
				IncludePods:      true,
				IncludeWorkloads: true,
				PodPhaseMode:     api.PodPhaseModeActive,
			},
		},
		{
			name:     "aggregated-rbac",
			snapshot: aggregatedSnapshotForGolden(),
			spec: api.RoleGraphReviewSpec{
				Selector: api.Selector{
					APIGroups: []string{""},
					Resources: []string{"pods/exec"},
					Verbs:     []string{"get"},
				},
				MatchMode:           api.MatchModeAny,
				IncludeRuleMetadata: true,
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			status := engine.Query(tc.snapshot, tc.spec)
			got, err := json.MarshalIndent(status, "", "  ")
			if err != nil {
				t.Fatalf("marshal status: %v", err)
			}
			got = append(got, '\n')

			goldenPath := filepath.Join("testdata", tc.name+".golden.json")
			if update {
				if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
					t.Fatalf("write golden %s: %v", goldenPath, err)
				}
			}

			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read golden %s: %v", goldenPath, err)
			}
			if string(got) != string(want) {
				t.Fatalf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", tc.name, string(got), string(want))
			}
		})
	}
}

func basicSnapshotForGolden() *indexer.Snapshot {
	snapshot := &indexer.Snapshot{
		BuiltAt:           time.Now(),
		RolesByID:         map[indexer.RoleID]*indexer.RoleRecord{},
		BindingsByRoleRef: map[indexer.RoleRefKey][]*indexer.BindingRecord{},
		RoleIDsByVerb:     map[string]map[indexer.RoleID]struct{}{},
		RoleIDsByResource: map[string]map[indexer.RoleID]struct{}{},
		RoleIDsByAPIGroup: map[string]map[indexer.RoleID]struct{}{},
		AllRoleIDs:        []indexer.RoleID{},
	}

	role := &indexer.RoleRecord{
		UID:       types.UID("r1"),
		Kind:      indexer.KindClusterRole,
		Name:      "exec-role",
		Namespace: "",
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{""},
			Resources: []string{"pods/exec"},
			Verbs:     []string{"create"},
		}},
	}
	roleID := indexer.RoleID("clusterrole:exec-role")
	snapshot.RolesByID[roleID] = role
	snapshot.AllRoleIDs = []indexer.RoleID{roleID}
	snapshot.RoleIDsByAPIGroup[""] = map[indexer.RoleID]struct{}{roleID: {}}
	snapshot.RoleIDsByResource["pods/exec"] = map[indexer.RoleID]struct{}{roleID: {}}
	snapshot.RoleIDsByVerb["create"] = map[indexer.RoleID]struct{}{roleID: {}}

	binding := &indexer.BindingRecord{
		UID:      types.UID("b1"),
		Kind:     indexer.KindClusterRoleBinding,
		Name:     "bind-exec",
		RoleRef:  indexer.RoleRefKey{Kind: indexer.KindClusterRole, Name: "exec-role"},
		Subjects: []rbacv1.Subject{{Kind: indexer.SubjectKindUser, Name: "alice"}},
	}
	snapshot.BindingsByRoleRef[binding.RoleRef] = []*indexer.BindingRecord{binding}
	return snapshot
}

func aggregatedSnapshotForGolden() *indexer.Snapshot {
	snapshot := &indexer.Snapshot{
		BuiltAt:               time.Now(),
		RolesByID:             map[indexer.RoleID]*indexer.RoleRecord{},
		BindingsByRoleRef:     map[indexer.RoleRefKey][]*indexer.BindingRecord{},
		AggregatedRoleSources: map[indexer.RoleID][]indexer.RoleID{},
		RoleIDsByVerb:         map[string]map[indexer.RoleID]struct{}{},
		RoleIDsByResource:     map[string]map[indexer.RoleID]struct{}{},
		RoleIDsByAPIGroup:     map[string]map[indexer.RoleID]struct{}{},
		AllRoleIDs:            []indexer.RoleID{},
	}

	sourceRole := &indexer.RoleRecord{
		UID:  types.UID("src"),
		Kind: indexer.KindClusterRole,
		Name: "aggregate-to-edit-source",
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{""},
			Resources: []string{"pods/exec"},
			Verbs:     []string{"get"},
		}},
	}
	aggregatedRole := &indexer.RoleRecord{
		UID:  types.UID("agg"),
		Kind: indexer.KindClusterRole,
		Name: "edit",
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{""},
			Resources: []string{"pods/exec"},
			Verbs:     []string{"get"},
		}},
	}

	sourceRoleID := indexer.RoleID("clusterrole:aggregate-to-edit-source")
	aggregatedRoleID := indexer.RoleID("clusterrole:edit")
	snapshot.RolesByID[sourceRoleID] = sourceRole
	snapshot.RolesByID[aggregatedRoleID] = aggregatedRole
	snapshot.AllRoleIDs = []indexer.RoleID{aggregatedRoleID}
	snapshot.AggregatedRoleSources[aggregatedRoleID] = []indexer.RoleID{sourceRoleID}
	snapshot.RoleIDsByAPIGroup[""] = map[indexer.RoleID]struct{}{aggregatedRoleID: {}}
	snapshot.RoleIDsByResource["pods/exec"] = map[indexer.RoleID]struct{}{aggregatedRoleID: {}}
	snapshot.RoleIDsByVerb["get"] = map[indexer.RoleID]struct{}{aggregatedRoleID: {}}

	binding := &indexer.BindingRecord{
		UID:      types.UID("b1"),
		Kind:     indexer.KindClusterRoleBinding,
		Name:     "bind-edit",
		RoleRef:  indexer.RoleRefKey{Kind: indexer.KindClusterRole, Name: "edit"},
		Subjects: []rbacv1.Subject{{Kind: indexer.SubjectKindUser, Name: "bob"}},
	}
	snapshot.BindingsByRoleRef[binding.RoleRef] = []*indexer.BindingRecord{binding}

	return snapshot
}

func boolPtr(v bool) *bool {
	return &v
}

func assertHasNodeType(t *testing.T, nodes []api.GraphNode, nodeType api.GraphNodeType) {
	t.Helper()
	for _, node := range nodes {
		if node.Type == nodeType {
			return
		}
	}
	t.Fatalf("expected node type %q to be present", nodeType)
}

func assertHasEdgeType(t *testing.T, edges []api.GraphEdge, edgeType api.GraphEdgeType) {
	t.Helper()
	for _, edge := range edges {
		if edge.Type == edgeType {
			return
		}
	}
	t.Fatalf("expected edge type %q to be present", edgeType)
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
