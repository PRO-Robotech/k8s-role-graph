# k8s-role-graph

Aggregated RBAC Graph API for Kubernetes (`rbacgraph.incloud.io/v1alpha1`) with live informer indexing and web DAG visualization.

## What is implemented

- Aggregated API endpoint:
  - `POST /apis/rbacgraph.incloud.io/v1alpha1/rolegraphreviews`
- REST alias:
  - `POST /role-graph/query`
- Cluster-scoped review resource shape:
  - `RoleGraphReview`
- Live in-memory indexing via informers for:
  - `Role`, `ClusterRole`, `RoleBinding`, `ClusterRoleBinding`
- Query matching:
  - wildcard-aware matching, `matchMode=any|all` (default `any`)
- Graph response shape:
  - `Role/ClusterRole -> RoleBinding/ClusterRoleBinding -> Subject`
- Resource map aggregation:
  - `(apiGroup, resource, verb)` with role/binding/subject counts
- Warnings + known gaps fields in response
- Web viewer (separate pod) that calls aggregated endpoint via kube-apiserver

## Repository layout

- `cmd/rbacgraph-apiserver`: aggregated API server
- `cmd/rbacgraph-web`: web UI and API proxy client
- `pkg/apis/rbacgraph/v1alpha1`: request/response types
- `pkg/indexer`: informer-based RBAC snapshot indexer
- `pkg/matcher`: selector-rule matching
- `pkg/engine`: graph/resource-map query execution
- `deploy/kustomize/base`: all cluster objects
- `deploy/kustomize/overlays/kind`: kind-specific overlay

## Build images

```bash
docker build -f Dockerfile.apiserver -t rbacgraph-apiserver:dev .
docker build -f Dockerfile.web -t rbacgraph-web:dev .
```

## Load images into kind

```bash
kind load docker-image rbacgraph-apiserver:dev --name <kind-cluster-name>
kind load docker-image rbacgraph-web:dev --name <kind-cluster-name>
```

## Deploy to kind

Prerequisite: cert-manager already installed in cluster.

```bash
kubectl apply -k deploy/kustomize/overlays/kind
```

## Smoke checks

```bash
kubectl get apiservice v1alpha1.rbacgraph.incloud.io
kubectl get --raw /apis/rbacgraph.incloud.io/v1alpha1
```

OpenAPI spec:

```bash
kubectl get --raw /openapi/v3
kubectl get --raw /apis/rbacgraph.incloud.io/v1alpha1/openapi
```

Example query:

```bash
cat <<'JSON' | kubectl create --raw /apis/rbacgraph.incloud.io/v1alpha1/rolegraphreviews -f -
{
  "apiVersion": "rbacgraph.incloud.io/v1alpha1",
  "kind": "RoleGraphReview",
  "metadata": {"name": "demo"},
  "spec": {
    "selector": {
      "apiGroups": [""],
      "resources": ["pods/exec"],
      "verbs": ["get", "create"]
    },
    "matchMode": "any",
    "includeRuleMetadata": true
  }
}
JSON
```

## Open web UI

```bash
kubectl -n rbac-graph-system port-forward svc/rbacgraph-web 8080:80
# open http://localhost:8080
```

## OpenAPI / Swagger generation

```bash
make openapi         # generate pkg/openapi/openapi.json
make verify-openapi  # fail if spec is stale
```

The generated OpenAPI file is produced from Go API types and HTTP route definitions in `cmd/rbacgraph-openapi`.

## Notes

- If informer caches are not synced yet, API returns `503 index_not_ready`.
- Backend indexing policy: prefer typed key structs/types for domain indexes; avoid ad-hoc composite string keys for new map indexes.
