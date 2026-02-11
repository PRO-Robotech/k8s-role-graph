# Справочник API

## Обзор

| Свойство | Значение |
|---|---|
| API-группа | `rbacgraph.incloud.io` |
| Версия | `v1alpha1` |
| Kind | `RoleGraphReview` |
| Resource | `rolegraphreviews` |
| Область видимости | Кластерная (не привязана к namespace) |
| Глаголы | Только `create` |
| Эндпоинт | `POST /apis/rbacgraph.incloud.io/v1alpha1/rolegraphreviews` |

Это **review-стиль API** (как `SubjectAccessReview`): клиент отправляет `spec`, сервер возвращает вычисленный `status`. Данные не сохраняются.

---

## RoleGraphReview

Верхнеуровневая обёртка запроса/ответа.

```json
{
  "apiVersion": "rbacgraph.incloud.io/v1alpha1",
  "kind": "RoleGraphReview",
  "metadata": {
    "name": "my-query"
  },
  "spec": { ... },
  "status": { ... }
}
```

| Поле | Тип | Обязательное | Описание |
|---|---|---|---|
| `apiVersion` | string | Да | Должно быть `rbacgraph.incloud.io/v1alpha1`. Заполняется автоматически, если пусто. |
| `kind` | string | Да | Должно быть `RoleGraphReview`. Заполняется автоматически, если пусто. |
| `metadata.name` | string | Да | Произвольное имя запроса. |
| `spec` | [RoleGraphReviewSpec](#rolegraphreviewspec) | Да | Параметры запроса. |
| `status` | [RoleGraphReviewStatus](#rolegraphreviewstatus) | — | Ответ (заполняется сервером). |

---

## RoleGraphReviewSpec

Определяет, что запрашивать и как возвращать результаты.

| Поле | Тип | По умолчанию | Описание |
|---|---|---|---|
| `selector` | [Selector](#selector) | `{}` | Какие RBAC-правила искать. |
| `matchMode` | string | `"any"` | Как комбинируются поля селектора: `"any"` или `"all"`. |
| `includeRuleMetadata` | bool | `true` | Включить `sourceObjectUID` и `sourceRuleIndex` в ссылки правил для трассировки. |
| `namespaceScope` | [NamespaceScope](#namespacescope) | `{}` | Фильтрация результатов по namespace. |
| `includePods` | bool | `false` | Включить поды в цепочку рантайма (serviceAccount → pod). |
| `includeWorkloads` | bool | `false` | Включить воркнагрузки в цепочку рантайма. Автоматически включает `includePods`. |
| `podPhaseMode` | string | `"active"` | Какие фазы подов включать: `"active"`, `"running"` или `"all"`. |
| `maxPodsPerSubject` | int | `20` | Максимум подов на один serviceAccount-субъект. Превышение создаёт overflow-узел. |
| `maxWorkloadsPerPod` | int | `10` | Максимум воркнагрузок на один под. Превышение создаёт overflow-узел. |

### matchMode

| Значение | Поведение |
|---|---|
| `"any"` | Правило совпадает, если **любое** непустое поле селектора совпадает (логика ИЛИ). |
| `"all"` | Правило совпадает, только если **все** непустые поля селектора совпадают (логика И). |

### podPhaseMode

| Значение | Включённые фазы |
|---|---|
| `"active"` | Pending, Running, Unknown (исключаются Succeeded и Failed) |
| `"running"` | Только Running |
| `"all"` | Все фазы |

---

## Selector

Определяет, какие RBAC-правила политик должны совпасть. Все поля поддерживают wildcard `"*"`, который совпадает со всем.

| Поле | Тип | Описание |
|---|---|---|
| `apiGroups` | string[] | API-группы для поиска (например, `[""]` для core, `["apps"]`). |
| `resources` | string[] | Ресурсы для поиска (например, `["pods"]`, `["pods/exec"]`, `["deployments"]`). |
| `verbs` | string[] | Глаголы для поиска (например, `["get", "list"]`, `["create"]`, `["*"]`). |
| `resourceNames` | string[] | Конкретные имена ресурсов для поиска. |
| `nonResourceURLs` | string[] | Non-resource URL для поиска (например, `["/healthz"]`, `["/metrics"]`). |

Пустой селектор (`{}`) совпадает со **всеми** RBAC-правилами в кластере.

> **Примечание:** `resources` и `nonResourceURLs` — это независимые измерения запроса. Селектор может содержать оба, и результаты будут включать правила, совпадающие с любым из них (в режиме `"any"`) или с обоими (в режиме `"all"`).

---

## NamespaceScope

Фильтрует результаты по конкретным namespace.

| Поле | Тип | По умолчанию | Описание |
|---|---|---|---|
| `namespaces` | string[] | `[]` | Имена namespace для включения. Пустой список означает все namespace. |
| `strict` | bool | `false` | При `true` — исключить ClusterRole/ClusterRoleBinding, которые действуют на уровне всего кластера. При `false` — включить их вместе с namespace-scoped результатами. |

---

## RoleGraphReviewStatus

Ответ, заполняемый сервером.

| Поле | Тип | Описание |
|---|---|---|
| `matchedRoles` | int | Количество найденных Roles и ClusterRoles. |
| `matchedBindings` | int | Количество найденных RoleBindings и ClusterRoleBindings. |
| `matchedSubjects` | int | Количество найденных субъектов (users, groups, serviceAccounts). |
| `matchedPods` | int | Количество найденных подов (только при `includePods: true`). |
| `matchedWorkloads` | int | Количество найденных воркнагрузок (только при `includeWorkloads: true`). |
| `warnings` | string[] | Некритичные проблемы (например, ошибки листинга информеров, автоматически включённые флаги). |
| `knownGaps` | string[] | Известные ограничения текущего запроса (например, рантайм-цепочка покрывает только serviceAccounts). |
| `graph` | [Graph](#graph) | Граф RBAC-отношений. |
| `resourceMap` | [ResourceMapRow[]](#resourcemaprow) | Сводная таблица найденных API-ресурсов. |

---

## Graph

| Поле | Тип | Описание |
|---|---|---|
| `nodes` | [GraphNode[]](#graphnode) | Все узлы графа. |
| `edges` | [GraphEdge[]](#graphedge) | Все направленные рёбра графа. |

---

## GraphNode

| Поле | Тип | Описание |
|---|---|---|
| `id` | string | Уникальный идентификатор узла (обычно Kubernetes UID или синтетический ID). |
| `type` | string | Тип узла (см. таблицу ниже). |
| `name` | string | Отображаемое имя объекта. |
| `namespace` | string | Kubernetes namespace (пусто для кластерных объектов). |
| `aggregated` | bool | `true`, если этот ClusterRole создан через агрегацию. |
| `aggregationSources` | string[] | UID ClusterRole, агрегированных в эту роль. |
| `matchedRuleRefs` | [RuleRef[]](#ruleref) | Какие конкретные правила этой роли совпали с запросом. |
| `labels` | map[string]string | Kubernetes labels. |
| `annotations` | map[string]string | Kubernetes annotations. |
| `podPhase` | string | Фаза пода (только для узлов типа `pod`). |
| `workloadKind` | string | Тип воркнагрузки, например `"Deployment"` (только для узлов типа `workload`). |
| `synthetic` | bool | `true` для синтетических overflow-узлов. |
| `hiddenCount` | int | Количество скрытых элементов, представленных overflow-узлом. |

### Типы узлов

| Тип | Описание |
|---|---|
| `role` | Namespace-scoped Role |
| `clusterRole` | Кластерный ClusterRole |
| `roleBinding` | Namespace-scoped RoleBinding |
| `clusterRoleBinding` | Кластерный ClusterRoleBinding |
| `user` | Субъект-пользователь |
| `group` | Субъект-группа |
| `serviceAccount` | Субъект ServiceAccount |
| `pod` | Под (цепочка рантайма) |
| `workload` | Контроллер воркнагрузки — Deployment, StatefulSet, DaemonSet, ReplicaSet, Job, CronJob |
| `podOverflow` | Синтетический узел, представляющий скрытые поды при превышении `maxPodsPerSubject` |
| `workloadOverflow` | Синтетический узел, представляющий скрытые воркнагрузки при превышении `maxWorkloadsPerPod` |

---

## GraphEdge

| Поле | Тип | Описание |
|---|---|---|
| `id` | string | Уникальный идентификатор ребра. |
| `from` | string | ID узла-источника. |
| `to` | string | ID узла-назначения. |
| `type` | string | Тип ребра (см. таблицу ниже). |
| `ruleRefs` | [RuleRef[]](#ruleref) | Конкретные RBAC-правила, которые представляет это ребро. |
| `explain` | string | Человекочитаемое описание ребра. |

### Типы рёбер

| Тип | Связь (From → To) | Описание |
|---|---|---|
| `aggregates` | ClusterRole → ClusterRole | Отношение агрегации |
| `grants` | Role/ClusterRole → Binding | Привязка ссылается на эту роль |
| `subjects` | Binding → Subject | Привязка предоставляет доступ этому субъекту |
| `runsAs` | ServiceAccount → Pod | Под запущен под этим serviceAccount |
| `ownedBy` | Pod → Workload | Под принадлежит этому контроллеру воркнагрузки |

---

## RuleRef

Идентифицирует конкретное RBAC-правило, совпавшее с запросом. Полезно для трассировки — понять, почему именно эта роль была включена.

| Поле | Тип | Описание |
|---|---|---|
| `apiVersion` | string | API-версия правила (если применимо). |
| `apiGroup` | string | Совпавшая API-группа. |
| `resource` | string | Совпавший ресурс. |
| `subresource` | string | Совпавший подресурс (например, `exec` из `pods/exec`). |
| `verb` | string | Совпавший глагол. |
| `resourceNames` | string[] | Совпавшие имена ресурсов. |
| `nonResourceURLs` | string[] | Совпавшие non-resource URL. |
| `sourceObjectUID` | string | UID Role/ClusterRole, содержащего это правило (при `includeRuleMetadata: true`). |
| `sourceRuleIndex` | int | Индекс правила в массиве `rules[]` роли (при `includeRuleMetadata: true`). |

---

## ResourceMapRow

Строка сводной таблицы ресурсов. Каждая строка представляет уникальную комбинацию (apiGroup, resource, verb), найденную среди всех совпавших ролей.

| Поле | Тип | Описание |
|---|---|---|
| `apiGroup` | string | API-группа (пустая строка для core-группы). |
| `resource` | string | Имя ресурса. |
| `verb` | string | Глагол. |
| `roleCount` | int | Количество ролей, предоставляющих это разрешение. |
| `bindingCount` | int | Количество привязок, ссылающихся на эти роли. |
| `subjectCount` | int | Количество субъектов, получающих это разрешение. |

---

## Значения по умолчанию

Сводка всех значений по умолчанию, применяемых `EnsureDefaults()`:

| Поле | По умолчанию | Условие |
|---|---|---|
| `apiVersion` | `"rbacgraph.incloud.io/v1alpha1"` | Когда пусто или содержит только пробелы |
| `kind` | `"RoleGraphReview"` | Когда пусто или содержит только пробелы |
| `matchMode` | `"any"` | Когда пусто |
| `includeRuleMetadata` | `true` | Всегда устанавливается в `true` |
| `podPhaseMode` | `"active"` | Когда пусто |
| `maxPodsPerSubject` | `20` | Когда ≤ 0 |
| `maxWorkloadsPerPod` | `10` | Когда ≤ 0 |

Кроме того, если `includeWorkloads: true`, но `includePods: false`, то `includePods` автоматически устанавливается в `true`, и в ответ добавляется предупреждение.

---

## Валидация

Следующие проверки выполняются над spec:

| Поле | Правило | Ошибка |
|---|---|---|
| `matchMode` | Должно быть `"any"` или `"all"` | `invalid matchMode "<значение>"` |
| `podPhaseMode` | Должно быть `"active"`, `"running"` или `"all"` | `invalid podPhaseMode "<значение>"` |
