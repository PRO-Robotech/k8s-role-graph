package indexer

import (
	"strings"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func recID(kind, namespace, name string) RoleID {
	if namespace == "" {
		return RoleID(strings.ToLower(kind) + ":" + name)
	}
	return RoleID(strings.ToLower(kind) + ":" + namespace + "/" + name)
}

func cloneMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneRules(in []rbacv1.PolicyRule) []rbacv1.PolicyRule {
	if len(in) == 0 {
		return nil
	}
	out := make([]rbacv1.PolicyRule, len(in))
	copy(out, in)
	return out
}

func cloneOwnerReferences(in []metav1.OwnerReference) []metav1.OwnerReference {
	if len(in) == 0 {
		return nil
	}
	out := make([]metav1.OwnerReference, len(in))
	copy(out, in)
	return out
}

func serviceAccountKey(namespace, name string) ServiceAccountKey {
	return ServiceAccountKey{Namespace: namespace, Name: name}
}

func normalizeServiceAccountName(name string) string {
	normalized := strings.TrimSpace(name)
	if normalized == "" {
		return DefaultServiceAccountName
	}
	return normalized
}

func normalizedSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	uniq := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		n := strings.TrimSpace(strings.ToLower(v))
		if _, exists := uniq[n]; exists {
			continue
		}
		uniq[n] = struct{}{}
		out = append(out, n)
	}
	return out
}

func insertIndex(idx map[string]map[RoleID]struct{}, token string, roleID RoleID) {
	bucket, ok := idx[token]
	if !ok {
		bucket = make(map[RoleID]struct{})
		idx[token] = bucket
	}
	bucket[roleID] = struct{}{}
}
