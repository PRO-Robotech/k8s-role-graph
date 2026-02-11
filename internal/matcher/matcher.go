package matcher

import (
	"strings"

	rbacv1 "k8s.io/api/rbac/v1"

	api "k8s-role-graph/pkg/apis/rbacgraph/v1alpha1"
)

type MatchResult struct {
	Matched  bool
	RuleRefs []api.RuleRef
}

func MatchRule(rule rbacv1.PolicyRule, selector api.Selector, mode api.MatchMode, sourceUID string, ruleIndex int) MatchResult {
	mode = normalizeMode(mode)
	resourceQuery := len(selector.Resources) > 0 || len(selector.APIGroups) > 0 || len(selector.ResourceNames) > 0
	nonResourceQuery := len(selector.NonResourceURLs) > 0

	resourceRefs, resourceMatched := matchResourceRule(rule, selector, mode, sourceUID, ruleIndex)
	nonResourceRefs, nonResourceMatched := matchNonResourceRule(rule, selector, mode, sourceUID, ruleIndex)

	switch {
	case resourceQuery && nonResourceQuery:
		if resourceMatched && nonResourceMatched {
			return MatchResult{Matched: true, RuleRefs: append(resourceRefs, nonResourceRefs...)}
		}
		return MatchResult{}
	case resourceQuery:
		if resourceMatched {
			return MatchResult{Matched: true, RuleRefs: resourceRefs}
		}
		return MatchResult{}
	case nonResourceQuery:
		if nonResourceMatched {
			return MatchResult{Matched: true, RuleRefs: nonResourceRefs}
		}
		return MatchResult{}
	default:
		if resourceMatched {
			return MatchResult{Matched: true, RuleRefs: resourceRefs}
		}
		if nonResourceMatched {
			return MatchResult{Matched: true, RuleRefs: nonResourceRefs}
		}
		return MatchResult{}
	}
}

func matchResourceRule(rule rbacv1.PolicyRule, selector api.Selector, mode api.MatchMode, sourceUID string, ruleIndex int) ([]api.RuleRef, bool) {
	if len(rule.Resources) == 0 {
		return nil, false
	}

	groupsRequested := ensureWildcard(selector.APIGroups)
	resourcesRequested := ensureWildcard(selector.Resources)
	verbsRequested := ensureWildcard(selector.Verbs)

	groupMatches := matchRequested(groupsRequested, rule.APIGroups, mode, exactOrWildcard)
	if len(groupMatches) == 0 {
		return nil, false
	}
	if len(selector.APIGroups) == 0 {
		groupMatches = append([]string(nil), rule.APIGroups...)
	}

	resourceMatches := matchRequested(resourcesRequested, rule.Resources, mode, resourceWildcardMatch)
	if len(resourceMatches) == 0 {
		return nil, false
	}
	if len(selector.Resources) == 0 {
		resourceMatches = append([]string(nil), rule.Resources...)
	}

	verbMatches := matchRequested(verbsRequested, rule.Verbs, mode, exactOrWildcard)
	if len(verbMatches) == 0 {
		return nil, false
	}
	if len(selector.Verbs) == 0 {
		verbMatches = append([]string(nil), rule.Verbs...)
	}

	if !matchResourceNames(selector.ResourceNames, rule.ResourceNames, mode) {
		return nil, false
	}

	refs := make([]api.RuleRef, 0, len(groupMatches)*len(resourceMatches)*len(verbMatches))
	for _, g := range groupMatches {
		for _, r := range resourceMatches {
			resource, subresource := splitResource(r)
			for _, v := range verbMatches {
				refs = append(refs, api.RuleRef{
					APIGroup:        g,
					Resource:        resource,
					Subresource:     subresource,
					Verb:            v,
					ResourceNames:   append([]string(nil), selector.ResourceNames...),
					SourceObjectUID: sourceUID,
					SourceRuleIndex: ruleIndex,
				})
			}
		}
	}

	return refs, true
}

func matchNonResourceRule(rule rbacv1.PolicyRule, selector api.Selector, mode api.MatchMode, sourceUID string, ruleIndex int) ([]api.RuleRef, bool) {
	if len(rule.NonResourceURLs) == 0 {
		return nil, false
	}

	requestedURLs := ensureWildcard(selector.NonResourceURLs)
	urlMatches := matchRequested(requestedURLs, rule.NonResourceURLs, mode, nonResourceWildcardMatch)
	if len(urlMatches) == 0 {
		return nil, false
	}
	if len(selector.NonResourceURLs) == 0 {
		urlMatches = append([]string(nil), rule.NonResourceURLs...)
	}

	verbsRequested := ensureWildcard(selector.Verbs)
	verbMatches := matchRequested(verbsRequested, rule.Verbs, mode, exactOrWildcard)
	if len(verbMatches) == 0 {
		return nil, false
	}
	if len(selector.Verbs) == 0 {
		verbMatches = append([]string(nil), rule.Verbs...)
	}

	refs := make([]api.RuleRef, 0, len(urlMatches)*len(verbMatches))
	for _, url := range urlMatches {
		for _, v := range verbMatches {
			refs = append(refs, api.RuleRef{
				NonResourceURLs: []string{url},
				Verb:            v,
				SourceObjectUID: sourceUID,
				SourceRuleIndex: ruleIndex,
			})
		}
	}

	return refs, true
}

func matchRequested(requested, allowed []string, mode api.MatchMode, fn func(string, string) bool) []string {
	if len(requested) == 0 {
		requested = []string{"*"}
	}
	if len(allowed) == 0 {
		return nil
	}

	matches := make([]string, 0, len(requested))
	for _, req := range requested {
		if req == "" {
			req = ""
		}
		ok := false
		for _, allow := range allowed {
			if fn(req, allow) {
				ok = true
				break
			}
		}
		if ok {
			matches = append(matches, req)
			continue
		}
		if mode == api.MatchModeAll {
			return nil
		}
	}

	if mode == api.MatchModeAny && len(matches) == 0 {
		return nil
	}
	return matches
}

func exactOrWildcard(requested, allowed string) bool {
	r := strings.TrimSpace(strings.ToLower(requested))
	a := strings.TrimSpace(strings.ToLower(allowed))
	return a == "*" || r == "*" || r == a
}

func resourceWildcardMatch(requested, allowed string) bool {
	r := strings.TrimSpace(strings.ToLower(requested))
	a := strings.TrimSpace(strings.ToLower(allowed))
	if a == "*" || r == "*" {
		return true
	}
	if r == a {
		return true
	}
	if strings.HasSuffix(a, "/*") {
		prefix := strings.TrimSuffix(a, "/*")
		return strings.HasPrefix(r, prefix+"/")
	}
	return false
}

func nonResourceWildcardMatch(requested, allowed string) bool {
	r := strings.TrimSpace(requested)
	a := strings.TrimSpace(allowed)
	if a == "*" || r == "*" {
		return true
	}
	if r == a {
		return true
	}
	if strings.HasSuffix(a, "*") {
		prefix := strings.TrimSuffix(a, "*")
		return strings.HasPrefix(r, prefix)
	}
	return false
}

func matchResourceNames(requested, allowed []string, mode api.MatchMode) bool {
	if len(requested) == 0 {
		return true
	}
	if len(allowed) == 0 {
		return true
	}
	lookup := make(map[string]struct{}, len(allowed))
	for _, name := range allowed {
		lookup[name] = struct{}{}
	}
	if mode == api.MatchModeAll {
		for _, name := range requested {
			if _, ok := lookup[name]; !ok {
				return false
			}
		}
		return true
	}
	for _, name := range requested {
		if _, ok := lookup[name]; ok {
			return true
		}
	}
	return false
}

func splitResource(resource string) (string, string) {
	parts := strings.SplitN(resource, "/", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

func ensureWildcard(values []string) []string {
	if len(values) == 0 {
		return []string{"*"}
	}
	return values
}

func normalizeMode(mode api.MatchMode) api.MatchMode {
	if mode == api.MatchModeAll {
		return api.MatchModeAll
	}
	return api.MatchModeAny
}
