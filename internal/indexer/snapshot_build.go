package indexer

import (
	"fmt"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

func newEmptySnapshot() *Snapshot {
	return &Snapshot{
		BuiltAt:               time.Now().UTC(),
		RolesByID:             make(map[RoleID]*RoleRecord),
		BindingsByRoleRef:     make(map[RoleRefKey][]*BindingRecord),
		AggregatedRoleSources: make(map[RoleID][]RoleID),
		PodsByServiceAccount:  make(map[ServiceAccountKey][]*PodRecord),
		WorkloadsByUID:        make(map[types.UID]*WorkloadRecord),
		RoleIDsByVerb:         make(map[string]map[RoleID]struct{}),
		RoleIDsByResource:     make(map[string]map[RoleID]struct{}),
		RoleIDsByAPIGroup:     make(map[string]map[RoleID]struct{}),
		AllRoleIDs:            []RoleID{},
	}
}

func listWithWarning[T any](
	listFn func(labels.Selector) ([]*T, error),
	resourceName string,
	warnings *[]string,
) []*T {
	items, err := listFn(labels.Everything())
	if err != nil {
		*warnings = append(*warnings, fmt.Sprintf("%s list failed: %v", resourceName, err))
	}
	return items
}

func indexRoleTokens(s *Snapshot, roleID RoleID, rules []rbacv1.PolicyRule) {
	for _, rule := range rules {
		for _, group := range normalizedSlice(rule.APIGroups) {
			insertIndex(s.RoleIDsByAPIGroup, group, roleID)
		}
		for _, resource := range normalizedSlice(rule.Resources) {
			insertIndex(s.RoleIDsByResource, resource, roleID)
		}
		for _, verb := range normalizedSlice(rule.Verbs) {
			insertIndex(s.RoleIDsByVerb, verb, roleID)
		}
	}
}

func indexWorkload(snapshot *Snapshot, uid types.UID, apiVersion, kind, namespace, name string, ownerReferences []metav1.OwnerReference) {
	if uid == "" {
		return
	}
	snapshot.WorkloadsByUID[uid] = &WorkloadRecord{
		UID:             uid,
		APIVersion:      apiVersion,
		Kind:            kind,
		Namespace:       namespace,
		Name:            name,
		OwnerReferences: cloneOwnerReferences(ownerReferences),
	}
}

func indexRoles(next *Snapshot, roles []*rbacv1.Role) {
	for _, role := range roles {
		rec := &RoleRecord{
			UID:         role.UID,
			Kind:        KindRole,
			Namespace:   role.Namespace,
			Name:        role.Name,
			Labels:      cloneMap(role.Labels),
			Annotations: cloneMap(role.Annotations),
			Rules:       cloneRules(role.Rules),
			RuleCount:   len(role.Rules),
		}
		id := recID(rec.Kind, rec.Namespace, rec.Name)
		next.RolesByID[id] = rec
		next.AllRoleIDs = append(next.AllRoleIDs, id)
		indexRoleTokens(next, id, rec.Rules)
	}
}

func indexClusterRoles(next *Snapshot, clusterRoles []*rbacv1.ClusterRole) {
	for _, role := range clusterRoles {
		rec := &RoleRecord{
			UID:         role.UID,
			Kind:        KindClusterRole,
			Namespace:   "",
			Name:        role.Name,
			Labels:      cloneMap(role.Labels),
			Annotations: cloneMap(role.Annotations),
			Rules:       cloneRules(role.Rules),
			RuleCount:   len(role.Rules),
		}
		if role.AggregationRule != nil && len(role.Rules) == 0 {
			next.KnownGaps = append(next.KnownGaps, fmt.Sprintf("clusterrole/%s has aggregationRule but resolved rules are empty", role.Name))
		}
		id := recID(rec.Kind, rec.Namespace, rec.Name)
		next.RolesByID[id] = rec
		next.AllRoleIDs = append(next.AllRoleIDs, id)
		indexRoleTokens(next, id, rec.Rules)
	}
}

func indexAggregatedClusterRoles(snapshot *Snapshot, clusterRoles []*rbacv1.ClusterRole) {
	if len(clusterRoles) == 0 {
		return
	}

	for _, target := range clusterRoles {
		if target.AggregationRule == nil || len(target.AggregationRule.ClusterRoleSelectors) == 0 {
			continue
		}
		targetID := recID(KindClusterRole, "", target.Name)
		sourceSet := make(map[RoleID]struct{})

		for _, selector := range target.AggregationRule.ClusterRoleSelectors {
			labelSelector, err := metav1.LabelSelectorAsSelector(&selector)
			if err != nil {
				snapshot.Warnings = append(snapshot.Warnings, fmt.Sprintf("clusterrole/%s has invalid aggregation selector: %v", target.Name, err))
				continue
			}
			for _, candidate := range clusterRoles {
				if candidate.Name == target.Name {
					continue
				}
				if !labelSelector.Matches(labels.Set(candidate.Labels)) {
					continue
				}
				sourceID := recID(KindClusterRole, "", candidate.Name)
				sourceSet[sourceID] = struct{}{}
			}
		}

		if len(sourceSet) == 0 {
			continue
		}
		sources := make([]RoleID, 0, len(sourceSet))
		for sourceID := range sourceSet {
			sources = append(sources, sourceID)
		}
		sort.Slice(sources, func(i, j int) bool {
			return sources[i] < sources[j]
		})
		snapshot.AggregatedRoleSources[targetID] = sources
	}
}

func indexRoleBindings(next *Snapshot, roleBindings []*rbacv1.RoleBinding) {
	for _, binding := range roleBindings {
		key := RoleRefKey{Kind: binding.RoleRef.Kind, Namespace: "", Name: binding.RoleRef.Name}
		if strings.EqualFold(binding.RoleRef.Kind, KindRole) {
			key.Namespace = binding.Namespace
		}
		bindRec := &BindingRecord{
			UID:       binding.UID,
			Kind:      KindRoleBinding,
			Namespace: binding.Namespace,
			Name:      binding.Name,
			RoleRef:   key,
			Subjects:  append([]rbacv1.Subject(nil), binding.Subjects...),
		}
		next.BindingsByRoleRef[key] = append(next.BindingsByRoleRef[key], bindRec)
	}
}

func indexClusterRoleBindings(next *Snapshot, clusterRoleBindings []*rbacv1.ClusterRoleBinding) {
	for _, binding := range clusterRoleBindings {
		key := RoleRefKey{Kind: binding.RoleRef.Kind, Namespace: "", Name: binding.RoleRef.Name}
		bindRec := &BindingRecord{
			UID:       binding.UID,
			Kind:      KindClusterRoleBinding,
			Namespace: "",
			Name:      binding.Name,
			RoleRef:   key,
			Subjects:  append([]rbacv1.Subject(nil), binding.Subjects...),
		}
		next.BindingsByRoleRef[key] = append(next.BindingsByRoleRef[key], bindRec)
	}
}

func indexPods(next *Snapshot, pods []*corev1.Pod) {
	for _, pod := range pods {
		sa := normalizeServiceAccountName(pod.Spec.ServiceAccountName)
		key := serviceAccountKey(pod.Namespace, sa)
		next.PodsByServiceAccount[key] = append(next.PodsByServiceAccount[key], &PodRecord{
			UID:                pod.UID,
			Namespace:          pod.Namespace,
			Name:               pod.Name,
			ServiceAccountName: sa,
			Phase:              pod.Status.Phase,
			OwnerReferences:    cloneOwnerReferences(pod.OwnerReferences),
		})
	}
}

func sortSnapshot(next *Snapshot) {
	for key := range next.PodsByServiceAccount {
		sort.Slice(next.PodsByServiceAccount[key], func(i, j int) bool {
			left := next.PodsByServiceAccount[key][i]
			right := next.PodsByServiceAccount[key][j]
			if left.Namespace != right.Namespace {
				return left.Namespace < right.Namespace
			}
			if left.Name != right.Name {
				return left.Name < right.Name
			}
			return string(left.UID) < string(right.UID)
		})
	}

	sort.Slice(next.AllRoleIDs, func(i, j int) bool {
		return next.AllRoleIDs[i] < next.AllRoleIDs[j]
	})
	for key := range next.RoleIDsByVerb {
		if key == "" {
			delete(next.RoleIDsByVerb, key)
		}
	}
	for key := range next.RoleIDsByResource {
		if key == "" {
			delete(next.RoleIDsByResource, key)
		}
	}
	for key := range next.RoleIDsByAPIGroup {
		if key == "" {
			// empty group is valid core API group
			continue
		}
	}
}
