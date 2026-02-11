package indexer

import (
	"sort"
	"strings"

	api "k8s-role-graph/pkg/apis/rbacgraph/v1alpha1"
)

func (s *Snapshot) CandidateRoleIDs(selector api.Selector) []RoleID {
	constraints := make([]map[RoleID]struct{}, 0, 3)

	if len(selector.APIGroups) > 0 {
		constraints = append(constraints, s.collectMatches(s.RoleIDsByAPIGroup, selector.APIGroups))
	}
	if len(selector.Resources) > 0 {
		constraints = append(constraints, s.collectMatches(s.RoleIDsByResource, selector.Resources))
	}
	if len(selector.Verbs) > 0 {
		constraints = append(constraints, s.collectMatches(s.RoleIDsByVerb, selector.Verbs))
	}

	if len(constraints) == 0 {
		out := make([]RoleID, len(s.AllRoleIDs))
		copy(out, s.AllRoleIDs)
		return out
	}

	intersected := constraints[0]
	for i := 1; i < len(constraints); i++ {
		intersected = intersect(intersected, constraints[i])
		if len(intersected) == 0 {
			return nil
		}
	}

	out := make([]RoleID, 0, len(intersected))
	for roleID := range intersected {
		out = append(out, roleID)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i] < out[j]
	})
	return out
}

func (s *Snapshot) collectMatches(index map[string]map[RoleID]struct{}, requested []string) map[RoleID]struct{} {
	matches := make(map[RoleID]struct{})
	for _, token := range requested {
		n := strings.ToLower(strings.TrimSpace(token))
		if n == "" {
			n = ""
		}
		for _, key := range []string{n, "*"} {
			bucket, ok := index[key]
			if !ok {
				continue
			}
			for roleID := range bucket {
				matches[roleID] = struct{}{}
			}
		}
	}
	return matches
}

func intersect(left, right map[RoleID]struct{}) map[RoleID]struct{} {
	if len(left) > len(right) {
		left, right = right, left
	}
	out := make(map[RoleID]struct{})
	for roleID := range left {
		if _, ok := right[roleID]; ok {
			out[roleID] = struct{}{}
		}
	}
	return out
}
