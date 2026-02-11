# Руководство по запросам

## Форматы запросов

Веб-сервер (`/api/query`) принимает три JSON-формата. Через `kubectl create --raw` всегда требуется полный формат.

### Формат 1: Полный RoleGraphReview

Полная Kubernetes-стиль обёртка. Обязательна при использовании `kubectl create --raw`.

```json
{
  "apiVersion": "rbacgraph.incloud.io/v1alpha1",
  "kind": "RoleGraphReview",
  "metadata": {"name": "my-query"},
  "spec": {
    "selector": {
      "resources": ["pods/exec"],
      "verbs": ["create"]
    },
    "matchMode": "all"
  }
}
```

### Формат 2: Только spec

Без обёртки — отправляйте только spec. Веб-сервер автоматически заполняет `apiVersion`, `kind` и `metadata`.

```json
{
  "selector": {
    "resources": ["pods/exec"],
    "verbs": ["create"]
  },
  "matchMode": "all"
}
```

### Формат 3: Сокращённый selector

Отправьте только поля селектора. Сервер обернёт их в spec со всеми значениями по умолчанию.

```json
{
  "resources": ["pods/exec"],
  "verbs": ["create"]
}
```

### Через kubectl

```bash
kubectl create --raw /apis/rbacgraph.incloud.io/v1alpha1/rolegraphreviews -f request.json
```

> **Примечание:** `kubectl create --raw` требует полный формат RoleGraphReview (Формат 1).

### Через веб-сервер

```bash
# Работает любой из трёх форматов:
curl -X POST http://localhost:8080/api/query \
  -H 'Content-Type: application/json' \
  -d '{"resources": ["secrets"], "verbs": ["get"]}'
```

---

## Рецепты запросов

### Кто может exec в поды?

```json
{
  "apiGroups": [""],
  "resources": ["pods/exec"],
  "verbs": ["get", "create"]
}
```

> `pods/exec` требует оба глагола `get` и `create`. Используйте `matchMode: "any"` (по умолчанию), чтобы найти роли, предоставляющие любой из них.

### Кто может создавать Deployments?

```json
{
  "apiGroups": ["apps"],
  "resources": ["deployments"],
  "verbs": ["create"]
}
```

### Кто может читать Secrets?

```json
{
  "apiGroups": [""],
  "resources": ["secrets"],
  "verbs": ["get", "list", "watch"]
}
```

### Все правила для конкретной API-группы

```json
{
  "apiGroups": ["apps"]
}
```

> Пустые `resources` / `verbs` означают «совпасть с любыми» — поэтому вернутся все роли с любыми правилами в группе `apps`.

### Кто имеет доступ к non-resource URL?

```json
{
  "nonResourceURLs": ["/healthz"]
}
```

### Найти все wildcard-правила

```json
{
  "resources": ["*"]
}
```

> Это находит правила, у которых в списке resources есть `"*"` — то есть правила, предоставляющие доступ ко всем ресурсам.

### Аудит эскалации: кто может изменять RBAC?

```json
{
  "apiGroups": ["rbac.authorization.k8s.io"],
  "resources": ["clusterroles", "clusterrolebindings", "roles", "rolebindings"],
  "verbs": ["create", "update", "patch", "delete", "bind", "escalate"]
}
```

### Всё в кластере (пустой селектор)

```json
{}
```

> Пустой селектор совпадает со **всеми** RBAC-правилами. На кластерах с множеством ролей это может породить очень большой граф.

---

## matchMode: `any` vs `all`

### `any` (по умолчанию) — логика ИЛИ

Правило совпадает, если **любое** непустое поле селектора совпадает. Полезно для широких поисковых запросов.

```json
{
  "selector": {
    "resources": ["secrets"],
    "verbs": ["delete"]
  },
  "matchMode": "any"
}
```

Этот запрос находит роли, у которых есть правила, упоминающие `secrets` **ИЛИ** правила, упоминающие `delete`. Роль с `{"resources": ["pods"], "verbs": ["delete"]}` совпадёт (потому что совпал глагол), даже если она не упоминает secrets.

### `all` — логика И

Правило совпадает, только если **каждое** непустое поле селектора совпадает. Полезно для точных запросов.

```json
{
  "selector": {
    "resources": ["secrets"],
    "verbs": ["delete"]
  },
  "matchMode": "all"
}
```

Этот запрос находит роли, где **одно правило** предоставляет `delete` на `secrets`. Роль должна иметь правило, совпадающее с обоими условиями одновременно.

---

## Фильтрация по namespace

### Базовая фильтрация

Показать результаты только из определённых namespace:

```json
{
  "selector": {"resources": ["secrets"]},
  "namespaceScope": {
    "namespaces": ["production", "staging"]
  }
}
```

По умолчанию (`strict: false`) включаются:
- Roles и RoleBindings в `production` и `staging`
- ClusterRoles и ClusterRoleBindings, действующие на уровне всего кластера (потому что они также затрагивают эти namespace)

### Строгая фильтрация по namespace

Исключить кластерные объекты полностью:

```json
{
  "selector": {"resources": ["secrets"]},
  "namespaceScope": {
    "namespaces": ["production"],
    "strict": true
  }
}
```

При `strict: true` включаются только namespace-scoped Roles и RoleBindings в `production`. ClusterRoles, на которые ссылаются RoleBindings в `production`, всё ещё включаются (так как они применяются к этому namespace через привязку).

---

## Цепочка рантайма

Цепочка рантайма расширяет граф за пределы RBAC, показывая какие поды и воркнагрузки реально используют найденные serviceAccounts.

### Включение подов

```json
{
  "selector": {"resources": ["secrets"], "verbs": ["get"]},
  "includePods": true
}
```

Добавляет узлы `pod`, связанные с узлами `serviceAccount` через рёбра `runsAs`. Включаются только поды, соответствующие фильтру `podPhaseMode`.

### Включение воркнагрузок

```json
{
  "selector": {"resources": ["secrets"], "verbs": ["get"]},
  "includeWorkloads": true
}
```

> Установка `includeWorkloads: true` автоматически включает `includePods: true` (в ответ добавляется предупреждение).

Добавляет узлы `workload` (Deployments, StatefulSets, DaemonSets, ReplicaSets, Jobs, CronJobs), связанные с узлами `pod` через рёбра `ownedBy`.

### Фильтрация по фазе подов

| Режим | Включённые фазы |
|---|---|
| `"active"` (по умолчанию) | Pending, Running, Unknown |
| `"running"` | Только Running |
| `"all"` | Все фазы, включая Succeeded и Failed |

```json
{
  "selector": {"resources": ["secrets"]},
  "includePods": true,
  "podPhaseMode": "running"
}
```

---

## Overflow-узлы

Когда у serviceAccount много подов или у пода много воркнагрузок, граф использует overflow-узлы, чтобы визуализация оставалась управляемой.

### Overflow подов

Если у serviceAccount больше подов, чем `maxPodsPerSubject` (по умолчанию: 20), лишние поды схлопываются в один узел `podOverflow`. Поле `hiddenCount` на overflow-узле показывает, сколько подов было скрыто.

```json
{
  "includePods": true,
  "maxPodsPerSubject": 5
}
```

### Overflow воркнагрузок

Аналогично, если у пода больше владельцев-воркнагрузок, чем `maxWorkloadsPerPod` (по умолчанию: 10), лишние воркнагрузки схлопываются в узел `workloadOverflow`.

```json
{
  "includeWorkloads": true,
  "maxWorkloadsPerPod": 3
}
```

---

## Интерпретация ответа

### Счётчики

В `status` содержатся сводные счётчики:

```json
{
  "matchedRoles": 5,
  "matchedBindings": 8,
  "matchedSubjects": 12,
  "matchedPods": 30,
  "matchedWorkloads": 10
}
```

### Структура графа

Граф представляет собой направленный ациклический граф со следующими связями:

```
ClusterRole ←(aggregates)← ClusterRole
     │
  (grants)
     ↓
ClusterRoleBinding / RoleBinding
     │
  (subjects)
     ↓
User / Group / ServiceAccount
     │
  (runsAs)          ← только с includePods
     ↓
Pod
     │
  (ownedBy)         ← только с includeWorkloads
     ↓
Workload (Deployment, StatefulSet и т.д.)
```

Для обхода графа:
1. Начните с **узлов ролей** (`type: "role"` или `type: "clusterRole"`) — это найденные RBAC-правила
2. Следуйте по рёбрам `grants`, чтобы найти **привязки**, ссылающиеся на каждую роль
3. Следуйте по рёбрам `subjects`, чтобы найти **пользователей, группы и serviceAccounts**, получающих разрешения
4. (Опционально) Следуйте по рёбрам `runsAs`, чтобы найти **поды**, запущенные под этими serviceAccounts
5. (Опционально) Следуйте по рёбрам `ownedBy`, чтобы найти **воркнагрузки**, владеющие этими подами

### Карта ресурсов

`resourceMap` предоставляет сводку в плоском виде — одна строка на уникальную комбинацию (apiGroup, resource, verb):

```json
{
  "resourceMap": [
    {"apiGroup": "", "resource": "secrets", "verb": "get", "roleCount": 3, "bindingCount": 5, "subjectCount": 8},
    {"apiGroup": "", "resource": "secrets", "verb": "list", "roleCount": 2, "bindingCount": 4, "subjectCount": 6}
  ]
}
```

### Предупреждения и известные ограничения

- **`warnings`**: Некритичные проблемы, обнаруженные при выполнении запроса (например, ошибки листинга информеров, автоматическое включение `includePods`).
- **`knownGaps`**: Ограничения текущего запроса, которые могут привести к неполным результатам (например, «цепочка рантайма ограничена субъектами serviceAccount»).

Всегда проверяйте эти массивы, чтобы понимать полноту результатов.
