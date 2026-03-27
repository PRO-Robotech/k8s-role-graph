package rolepermissionsview

import (
	"context"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fake "k8s.io/client-go/kubernetes/fake"

	"k8s-role-graph/internal/indexer"
	"k8s-role-graph/pkg/apis/rbacgraph"
)

func newTestREST(roles map[indexer.RoleID]*indexer.RoleRecord) *REST {
	idx := indexer.New(fake.NewSimpleClientset(), 0)
	idx.SetSnapshotForTest(&indexer.Snapshot{
		RolesByID: roles,
	})

	return NewREST(idx)
}

func TestCreate_ClusterRole(t *testing.T) {
	r := newTestREST(map[indexer.RoleID]*indexer.RoleRecord{
		"clusterrole:admin": {
			Kind: "ClusterRole",
			Name: "admin",
			Rules: []rbacv1.PolicyRule{
				{APIGroups: []string{""}, Resources: []string{"pods", "services"}, Verbs: []string{"get", "list"}},
				{NonResourceURLs: []string{"/metrics"}, Verbs: []string{"get"}},
			},
		},
	})

	view := &rbacgraph.RolePermissionsView{
		Spec: rbacgraph.RolePermissionsViewSpec{
			Role: rbacgraph.RoleRef{Kind: "clusterRole", Name: "admin"},
		},
	}

	obj, err := r.Create(context.Background(), view, nil, &metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	result := obj.(*rbacgraph.RolePermissionsView)

	if result.Status.Name != "admin" {
		t.Errorf("status.name = %q, want %q", result.Status.Name, "admin")
	}
	if result.Status.Scope != rbacgraph.RoleScopeCluster {
		t.Errorf("status.scope = %q, want %q", result.Status.Scope, rbacgraph.RoleScopeCluster)
	}
	if len(result.Status.APIGroups) != 1 {
		t.Fatalf("expected 1 API group, got %d", len(result.Status.APIGroups))
	}

	ag := result.Status.APIGroups[0]
	if ag.APIGroup != "core" {
		t.Errorf("apiGroup = %q, want %q", ag.APIGroup, "core")
	}
	if ag.ResourcesCount != 2 {
		t.Errorf("resourcesCount = %d, want 2", ag.ResourcesCount)
	}

	// Check pods verbs
	foundPods := false
	for _, res := range ag.Resources {
		if res.Plural == "pods" {
			foundPods = true
			if len(res.Verbs) != 2 {
				t.Errorf("pods verbs count = %d, want 2", len(res.Verbs))
			}
			for _, verb := range []string{"get", "list"} {
				vp, ok := res.Verbs[verb]
				if !ok {
					t.Errorf("pods missing verb %q", verb)
				}
				if !vp.Granted {
					t.Errorf("pods verb %q not granted", verb)
				}
			}
		}
	}
	if !foundPods {
		t.Error("expected pods in resources")
	}

	// Check nonResourceURLs
	if result.Status.NonResourceURLs == nil {
		t.Fatal("expected nonResourceUrls to be non-nil")
	}
	if result.Status.NonResourceURLs.URLsCount != 1 {
		t.Errorf("urlsCount = %d, want 1", result.Status.NonResourceURLs.URLsCount)
	}
	if result.Status.NonResourceURLs.URLs[0].URL != "/metrics" {
		t.Errorf("url = %q, want /metrics", result.Status.NonResourceURLs.URLs[0].URL)
	}
}

func TestCreate_NamespacedRole(t *testing.T) {
	r := newTestREST(map[indexer.RoleID]*indexer.RoleRecord{
		"role:default/reader": {
			Kind:      "Role",
			Namespace: "default",
			Name:      "reader",
			Rules: []rbacv1.PolicyRule{
				{APIGroups: []string{""}, Resources: []string{"configmaps"}, Verbs: []string{"get"}},
			},
		},
	})

	view := &rbacgraph.RolePermissionsView{
		Spec: rbacgraph.RolePermissionsViewSpec{
			Role: rbacgraph.RoleRef{Kind: "role", Name: "reader", Namespace: "default"},
		},
	}

	obj, err := r.Create(context.Background(), view, nil, &metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	result := obj.(*rbacgraph.RolePermissionsView)
	if result.Status.Scope != rbacgraph.RoleScopeNamespace {
		t.Errorf("scope = %q, want namespace", result.Status.Scope)
	}
	if len(result.Status.APIGroups) != 1 || result.Status.APIGroups[0].ResourcesCount != 1 {
		t.Error("expected 1 apiGroup with 1 resource")
	}
}

func TestCreate_NotFound(t *testing.T) {
	r := newTestREST(map[indexer.RoleID]*indexer.RoleRecord{})

	view := &rbacgraph.RolePermissionsView{
		Spec: rbacgraph.RolePermissionsViewSpec{
			Role: rbacgraph.RoleRef{Kind: "clusterRole", Name: "nonexistent"},
		},
	}

	_, err := r.Create(context.Background(), view, nil, &metav1.CreateOptions{})
	if err == nil {
		t.Fatal("expected error for nonexistent role")
	}
}

func TestCreate_SelectorFilter(t *testing.T) {
	r := newTestREST(map[indexer.RoleID]*indexer.RoleRecord{
		"clusterrole:admin": {
			Kind: "ClusterRole",
			Name: "admin",
			Rules: []rbacv1.PolicyRule{
				{APIGroups: []string{""}, Resources: []string{"pods"}, Verbs: []string{"get", "list", "create"}},
				{APIGroups: []string{""}, Resources: []string{"services"}, Verbs: []string{"get"}},
				{APIGroups: []string{"apps"}, Resources: []string{"deployments"}, Verbs: []string{"get"}},
			},
		},
	})

	view := &rbacgraph.RolePermissionsView{
		Spec: rbacgraph.RolePermissionsViewSpec{
			Role: rbacgraph.RoleRef{Kind: "clusterRole", Name: "admin"},
			Selector: rbacgraph.Selector{
				Resources: []string{"pods"},
			},
		},
	}

	obj, err := r.Create(context.Background(), view, nil, &metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	result := obj.(*rbacgraph.RolePermissionsView)
	if len(result.Status.APIGroups) != 1 {
		t.Fatalf("expected 1 API group (filtered), got %d", len(result.Status.APIGroups))
	}
	if result.Status.APIGroups[0].ResourcesCount != 1 {
		t.Errorf("expected 1 resource (pods only), got %d", result.Status.APIGroups[0].ResourcesCount)
	}
	if result.Status.APIGroups[0].Resources[0].Plural != "pods" {
		t.Errorf("expected pods, got %q", result.Status.APIGroups[0].Resources[0].Plural)
	}
}

func TestCreate_ValidationErrors(t *testing.T) {
	r := newTestREST(map[indexer.RoleID]*indexer.RoleRecord{})

	tests := []struct {
		name string
		spec rbacgraph.RolePermissionsViewSpec
	}{
		{
			name: "empty name",
			spec: rbacgraph.RolePermissionsViewSpec{Role: rbacgraph.RoleRef{Kind: "clusterRole"}},
		},
		{
			name: "invalid kind",
			spec: rbacgraph.RolePermissionsViewSpec{Role: rbacgraph.RoleRef{Kind: "invalid", Name: "test"}},
		},
		{
			name: "role without namespace",
			spec: rbacgraph.RolePermissionsViewSpec{Role: rbacgraph.RoleRef{Kind: "role", Name: "test", Namespace: ""}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := &rbacgraph.RolePermissionsView{Spec: tt.spec}
			_, err := r.Create(context.Background(), view, nil, &metav1.CreateOptions{})
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}
