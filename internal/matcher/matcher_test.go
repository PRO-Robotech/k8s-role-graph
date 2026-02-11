package matcher

import (
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"

	api "k8s-role-graph/pkg/apis/rbacgraph/v1alpha1"
)

func TestMatchRule_VerbAny(t *testing.T) {
	rule := rbacv1.PolicyRule{
		APIGroups: []string{""},
		Resources: []string{"pods/exec"},
		Verbs:     []string{"create"},
	}
	sel := api.Selector{
		APIGroups: []string{""},
		Resources: []string{"pods/exec"},
		Verbs:     []string{"get", "create"},
	}

	result := MatchRule(rule, sel, api.MatchModeAny, "uid-1", 0)
	if !result.Matched {
		t.Fatalf("expected selector to match rule")
	}
	if len(result.RuleRefs) == 0 {
		t.Fatalf("expected non-empty rule refs")
	}
}

func TestMatchRule_VerbAll(t *testing.T) {
	rule := rbacv1.PolicyRule{
		APIGroups: []string{""},
		Resources: []string{"pods/exec"},
		Verbs:     []string{"create"},
	}
	sel := api.Selector{
		APIGroups: []string{""},
		Resources: []string{"pods/exec"},
		Verbs:     []string{"get", "create"},
	}

	result := MatchRule(rule, sel, api.MatchModeAll, "uid-1", 0)
	if result.Matched {
		t.Fatalf("expected selector to fail in all mode")
	}
}

func TestMatchRule_Wildcards(t *testing.T) {
	rule := rbacv1.PolicyRule{
		APIGroups: []string{"*"},
		Resources: []string{"*"},
		Verbs:     []string{"*"},
	}
	sel := api.Selector{
		APIGroups: []string{"apps"},
		Resources: []string{"deployments"},
		Verbs:     []string{"patch"},
	}

	result := MatchRule(rule, sel, api.MatchModeAny, "uid-1", 0)
	if !result.Matched {
		t.Fatalf("wildcard rule must match")
	}
}

func TestMatchRule_NonResourceURLs(t *testing.T) {
	rule := rbacv1.PolicyRule{
		NonResourceURLs: []string{"/metrics*"},
		Verbs:           []string{"get"},
	}
	sel := api.Selector{
		NonResourceURLs: []string{"/metrics/cadvisor"},
		Verbs:           []string{"get"},
	}

	result := MatchRule(rule, sel, api.MatchModeAny, "uid-1", 0)
	if !result.Matched {
		t.Fatalf("nonResourceURL prefix should match")
	}
}

func TestMatchRule_ResourceNames(t *testing.T) {
	rule := rbacv1.PolicyRule{
		APIGroups:     []string{""},
		Resources:     []string{"configmaps"},
		ResourceNames: []string{"allowed"},
		Verbs:         []string{"get"},
	}

	sel := api.Selector{
		APIGroups:     []string{""},
		Resources:     []string{"configmaps"},
		ResourceNames: []string{"denied"},
		Verbs:         []string{"get"},
	}

	result := MatchRule(rule, sel, api.MatchModeAny, "uid-1", 0)
	if result.Matched {
		t.Fatalf("resource names should not match")
	}
}
