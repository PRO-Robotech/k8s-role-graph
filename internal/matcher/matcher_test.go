package matcher

import (
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"

	api "k8s-role-graph/pkg/apis/rbacgraph"
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

	result := MatchRule(MatchInput{Rule: rule, Selector: sel, Mode: api.MatchModeAny, SourceUID: "uid-1"})
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

	result := MatchRule(MatchInput{Rule: rule, Selector: sel, Mode: api.MatchModeAll, SourceUID: "uid-1"})
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

	result := MatchRule(MatchInput{Rule: rule, Selector: sel, Mode: api.MatchModeAny, SourceUID: "uid-1"})
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

	result := MatchRule(MatchInput{Rule: rule, Selector: sel, Mode: api.MatchModeAny, SourceUID: "uid-1"})
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

	result := MatchRule(MatchInput{Rule: rule, Selector: sel, Mode: api.MatchModeAny, SourceUID: "uid-1"})
	if result.Matched {
		t.Fatalf("resource names should not match")
	}
}

func TestRuleRef_ResourceNames_NoRuleRestriction(t *testing.T) {
	// When the rule has no resourceNames restriction (= all names allowed),
	// the RuleRef should NOT echo the selector's resourceNames.
	rule := rbacv1.PolicyRule{
		APIGroups: []string{""},
		Resources: []string{"pods"},
		Verbs:     []string{"get"},
		// ResourceNames is empty — no restriction.
	}
	sel := api.Selector{
		APIGroups:     []string{""},
		Resources:     []string{"pods"},
		Verbs:         []string{"get"},
		ResourceNames: []string{"my-pod"},
	}

	result := MatchRule(MatchInput{Rule: rule, Selector: sel, Mode: api.MatchModeAny, SourceUID: "uid-1"})
	if !result.Matched {
		t.Fatal("expected match")
	}
	if len(result.RuleRefs) == 0 {
		t.Fatal("expected non-empty rule refs")
	}
	for _, ref := range result.RuleRefs {
		if len(ref.ResourceNames) != 0 {
			t.Fatalf("expected empty ResourceNames in RuleRef when rule has no restriction, got %v", ref.ResourceNames)
		}
	}
}

func TestRuleRef_ResourceNames_WithRuleRestriction(t *testing.T) {
	// When the rule restricts by resourceNames, the RuleRef should include
	// the queried resourceNames that matched.
	rule := rbacv1.PolicyRule{
		APIGroups:     []string{""},
		Resources:     []string{"configmaps"},
		ResourceNames: []string{"my-config"},
		Verbs:         []string{"get"},
	}
	sel := api.Selector{
		APIGroups:     []string{""},
		Resources:     []string{"configmaps"},
		ResourceNames: []string{"my-config"},
		Verbs:         []string{"get"},
	}

	result := MatchRule(MatchInput{Rule: rule, Selector: sel, Mode: api.MatchModeAny, SourceUID: "uid-1"})
	if !result.Matched {
		t.Fatal("expected match")
	}
	if len(result.RuleRefs) == 0 {
		t.Fatal("expected non-empty rule refs")
	}
	for _, ref := range result.RuleRefs {
		if len(ref.ResourceNames) != 1 || ref.ResourceNames[0] != "my-config" {
			t.Fatalf("expected ResourceNames=[my-config] in RuleRef, got %v", ref.ResourceNames)
		}
	}
}

// --- WildcardMode: exact tests ---

func TestMatchRule_ExactMode_WildcardRuleDoesNotMatchConcrete(t *testing.T) {
	// In exact mode, a rule with verbs: ["*"] should NOT match selector verbs: ["get"].
	rule := rbacv1.PolicyRule{
		APIGroups: []string{""},
		Resources: []string{"pods"},
		Verbs:     []string{"*"},
	}
	sel := api.Selector{
		APIGroups: []string{""},
		Resources: []string{"pods"},
		Verbs:     []string{"get"},
	}

	result := MatchRule(MatchInput{Rule: rule, Selector: sel, Mode: api.MatchModeAny, WildcardMode: api.WildcardModeExact, SourceUID: "uid-1"})
	if result.Matched {
		t.Fatalf("exact mode: wildcard verb rule should NOT match concrete selector verb")
	}
}

func TestMatchRule_ExactMode_LiteralStarMatchesStar(t *testing.T) {
	// In exact mode, rule ["*"] + selector ["*"] → match (literal equality).
	rule := rbacv1.PolicyRule{
		APIGroups: []string{"*"},
		Resources: []string{"*"},
		Verbs:     []string{"*"},
	}
	sel := api.Selector{
		APIGroups: []string{"*"},
		Resources: []string{"*"},
		Verbs:     []string{"*"},
	}

	result := MatchRule(MatchInput{Rule: rule, Selector: sel, Mode: api.MatchModeAny, WildcardMode: api.WildcardModeExact, SourceUID: "uid-1"})
	if !result.Matched {
		t.Fatalf("exact mode: literal '*' selector should match literal '*' rule")
	}
}

func TestMatchRule_ExactMode_EmptyFieldIsNoConstraint(t *testing.T) {
	// In exact mode, an empty selector field means "no constraint" —
	// all rule values pass through without filtering.
	rule := rbacv1.PolicyRule{
		APIGroups: []string{""},
		Resources: []string{"pods"},
		Verbs:     []string{"get"},
	}
	sel := api.Selector{
		// All fields empty = no constraint = match everything
	}

	result := MatchRule(MatchInput{Rule: rule, Selector: sel, Mode: api.MatchModeAny, WildcardMode: api.WildcardModeExact, SourceUID: "uid-1"})
	if !result.Matched {
		t.Fatalf("exact mode: empty selector should match any rule (no constraint)")
	}
}

func TestMatchRule_ExactMode_SubresourceNoExpand(t *testing.T) {
	// In exact mode, rule ["pods/*"] should NOT match selector ["pods/exec"]
	// because there is no wildcard expansion.
	rule := rbacv1.PolicyRule{
		APIGroups: []string{""},
		Resources: []string{"pods/*"},
		Verbs:     []string{"create"},
	}
	sel := api.Selector{
		APIGroups: []string{""},
		Resources: []string{"pods/exec"},
		Verbs:     []string{"create"},
	}

	result := MatchRule(MatchInput{Rule: rule, Selector: sel, Mode: api.MatchModeAny, WildcardMode: api.WildcardModeExact, SourceUID: "uid-1"})
	if result.Matched {
		t.Fatalf("exact mode: pods/* rule should NOT match pods/exec selector")
	}
}

func TestMatchRule_ExactMode_NonResourceNoExpand(t *testing.T) {
	// In exact mode, rule ["/metrics*"] should NOT match selector ["/metrics/cadvisor"].
	rule := rbacv1.PolicyRule{
		NonResourceURLs: []string{"/metrics*"},
		Verbs:           []string{"get"},
	}
	sel := api.Selector{
		NonResourceURLs: []string{"/metrics/cadvisor"},
		Verbs:           []string{"get"},
	}

	result := MatchRule(MatchInput{Rule: rule, Selector: sel, Mode: api.MatchModeAny, WildcardMode: api.WildcardModeExact, SourceUID: "uid-1"})
	if result.Matched {
		t.Fatalf("exact mode: /metrics* rule should NOT match /metrics/cadvisor selector")
	}
}

func TestMatchRule_ExpandMode_BackwardsCompatible(t *testing.T) {
	// Confirm that expand mode (default) still works as before.
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

	result := MatchRule(MatchInput{Rule: rule, Selector: sel, Mode: api.MatchModeAny, WildcardMode: api.WildcardModeExpand, SourceUID: "uid-1"})
	if !result.Matched {
		t.Fatalf("expand mode: wildcard rule must match concrete selector (backwards compat)")
	}
}

func TestMatchRule_ExactMode_WildcardResourceDoesNotMatchConcrete(t *testing.T) {
	// In exact mode, rule resources: ["*"] should NOT match selector resources: ["pods"].
	rule := rbacv1.PolicyRule{
		APIGroups: []string{""},
		Resources: []string{"*"},
		Verbs:     []string{"get"},
	}
	sel := api.Selector{
		APIGroups: []string{""},
		Resources: []string{"pods"},
		Verbs:     []string{"get"},
	}

	result := MatchRule(MatchInput{Rule: rule, Selector: sel, Mode: api.MatchModeAny, WildcardMode: api.WildcardModeExact, SourceUID: "uid-1"})
	if result.Matched {
		t.Fatalf("exact mode: wildcard resource rule should NOT match concrete selector resource")
	}
}

func TestMatchRule_ExactMode_WildcardAPIGroupDoesNotMatchConcrete(t *testing.T) {
	// In exact mode, rule apiGroups: ["*"] should NOT match selector apiGroups: ["apps"].
	rule := rbacv1.PolicyRule{
		APIGroups: []string{"*"},
		Resources: []string{"deployments"},
		Verbs:     []string{"get"},
	}
	sel := api.Selector{
		APIGroups: []string{"apps"},
		Resources: []string{"deployments"},
		Verbs:     []string{"get"},
	}

	result := MatchRule(MatchInput{Rule: rule, Selector: sel, Mode: api.MatchModeAny, WildcardMode: api.WildcardModeExact, SourceUID: "uid-1"})
	if result.Matched {
		t.Fatalf("exact mode: wildcard apiGroup rule should NOT match concrete selector apiGroup")
	}
}
