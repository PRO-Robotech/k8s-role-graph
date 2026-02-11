# Архитектура

## Компоненты

```
┌──────────────────────────────────────────────────────────────────┐
│                        Кластер Kubernetes                        │
│                                                                  │
│  ┌─────────────┐      ┌──────────────────┐      ┌────────────┐  │
│  │ kube-apiserver│◄────│  APIService      │      │ cert-manager│  │
│  │             │      │  (v1alpha1.      │      │            │  │
│  │             │      │   rbacgraph.     │      │  ┌────────┐│  │
│  │             │      │   incloud.io)    │      │  │CA + TLS││  │
│  │             │      └────────┬─────────┘      │  │ серт-ты││  │
│  └──────┬──────┘               │                │  └────┬───┘│  │
│         │                      │                │       │    │  │
│         │              ┌───────▼──────────┐     │       │    │  │
│         │              │ rbacgraph-       │◄────┼───────┘    │  │
│         │              │ apiserver        │     │            │  │
│         │              │  ┌────────────┐  │     └────────────┘  │
│         │              │  │  Indexer    │  │                     │
│         │              │  │ (информеры) │  │                     │
│  watch/list ◄──────────│  └────────────┘  │                     │
│         │              │  ┌────────────┐  │                     │
│         │              │  │  Engine     │  │                     │
│         │              │  │  (запросы)  │  │                     │
│         │              │  └────────────┘  │                     │
│         │              └───────▲──────────┘                     │
│         │                      │                                │
│         │              ┌───────┴──────────┐                     │
│         │              │ rbacgraph-web    │                     │
│  прокси через ◄────────│ (фронтенд-прокси)│                     │
│  kube-apiserver        └───────▲──────────┘                     │
│                                │                                │
└────────────────────────────────┼────────────────────────────────┘
                                 │
                          ┌──────┴──────┐
                          │  Браузер /  │
                          │  curl / CI  │
                          └─────────────┘
```

### rbacgraph-apiserver

Агрегированный API-сервер, построенный на фреймворке Kubernetes `GenericAPIServer`. Регистрирует API-группу `rbacgraph.incloud.io/v1alpha1` и обрабатывает запросы на создание `RoleGraphReview`.

- **Без персистентного хранилища**: API в стиле review — обрабатывает запросы синхронно, etcd не нужен
- **Без admission**: Admission-контроллеры отключены
- **Делегированная аутентификация**: Аутентификация и авторизация делегируются к kube-apiserver

### rbacgraph-web

HTTP-прокси, который отдаёт React-фронтенд и перенаправляет POST-запросы `/api/query` к агрегированному API-серверу через kube-apiserver кластера. Принимает упрощённые форматы запросов (см. [Руководство по запросам](query-guide.md#форматы-запросов)) и оборачивает их в полные объекты `RoleGraphReview` перед проксированием.

### Indexer

Фоновый компонент внутри процесса apiserver. Поддерживает in-memory снэпшот RBAC-состояния кластера и рантайм-объектов, используя Kubernetes-информеры (watch + list).

### Engine

Stateless-процессор запросов. Получая снэпшот и spec, вычисляет совпавший граф и карту ресурсов.

---

## Путь запроса

```
1. Клиент отправляет POST на /api/query (веб) или kubectl create --raw (apiserver)
2. Веб-сервер декодирует запрос, заполняет умолчания, проксирует к kube-apiserver
3. kube-apiserver аутентифицирует/авторизует, маршрутизирует к rbacgraph-apiserver
4. REST storage rbacgraph-apiserver конвертирует internal типы ↔ v1alpha1
5. Engine.Query(snapshot, spec) выполняет:
   a. snapshot.CandidateRoleIDs(selector) — поиск кандидатов-ролей через индекс
   b. buildRBACGraph(roleIDs) — матчинг правил, построение узлов/рёбер для ролей, привязок, субъектов
   c. expandRuntimeChain() — если includePods/includeWorkloads, добавление узлов подов/воркнагрузок
   d. finalize() — вычисление счётчиков, свёртка resourceMap, сортировка графа
6. Ответ возвращается со status, содержащим граф + resourceMap + счётчики
```

---

## Индексируемые ресурсы

Indexer наблюдает и кэширует следующие ресурсы Kubernetes:

| API-группа | Ресурсы | Назначение |
|---|---|---|
| `rbac.authorization.k8s.io` | Roles, ClusterRoles, RoleBindings, ClusterRoleBindings | Основной RBAC-граф |
| _(core)_ | Pods | Цепочка рантайма (serviceAccount → pod) |
| `apps` | Deployments, ReplicaSets, StatefulSets, DaemonSets | Цепочка воркнагрузок (pod → владелец) |
| `batch` | Jobs, CronJobs | Цепочка воркнагрузок (pod → владелец) |

Indexer поддерживает единый атомарный `Snapshot`, который пересобирается при каждом событии add/update/delete. Снэпшот иммутабелен после построения — конкурентные запросы читают из него без блокировок.

---

## Требования RBAC

### rbacgraph-apiserver

API-серверу необходим кластерный доступ на чтение к индексируемым ресурсам:

| ClusterRole | Правила |
|---|---|
| `rbacgraph-apiserver-rbac-reader` | `get`, `list`, `watch` на Roles, ClusterRoles, RoleBindings, ClusterRoleBindings, Pods, Deployments, ReplicaSets, StatefulSets, DaemonSets, Jobs, CronJobs |

Также необходимо делегирование аутентификации/авторизации:

| Привязка | Роль | Назначение |
|---|---|---|
| `rbacgraph-apiserver-auth-delegator` (ClusterRoleBinding) | `system:auth-delegator` | Делегирование аутентификации токенов к kube-apiserver |
| `rbacgraph-apiserver-auth-reader` (RoleBinding в `kube-system`) | `extension-apiserver-authentication-reader` | Чтение конфигурации аутентификации (CA клиента, конфигурация requestheader) |

### rbacgraph-web

Веб-серверу нужно только разрешение на создание review-запросов:

| ClusterRole | Правила |
|---|---|
| `rbacgraph-web-query` | `create` на `rolegraphreviews` в `rbacgraph.incloud.io` |

---

## TLS и cert-manager

Агрегированный API-сервер требует TLS. Деплой использует cert-manager для автоматизации жизненного цикла сертификатов:

```
┌─────────────────────────┐
│ ClusterIssuer            │
│ rbacgraph-selfsigned-root│  (самоподписанный)
└────────────┬────────────┘
             │ выдаёт
             ▼
┌─────────────────────────┐
│ Certificate              │
│ rbacgraph-ca             │  (CA, isCA: true)
│ → Secret: rbacgraph-ca   │  (90 дней, обновление за 15 дней)
└────────────┬────────────┘
             │
             ▼
┌─────────────────────────┐
│ Issuer                   │
│ rbacgraph-ca-issuer      │  (CA issuer, читает из секрета rbacgraph-ca)
└────────────┬────────────┘
             │ выдаёт
             ▼
┌─────────────────────────┐
│ Certificate              │
│ rbacgraph-serving-cert   │  (серверный сертификат)
│ → Secret: rbacgraph-     │  (90 дней, обновление за 15 дней)
│   serving-tls            │
│                          │
│ DNS SAN:                 │
│  - rbacgraph-apiserver   │
│  - ...rbac-graph-system  │
│  - ...svc                │
│  - ...svc.cluster.local  │
└─────────────────────────┘
```

Объект `APIService` использует аннотацию `cert-manager.io/inject-ca-from`, чтобы cert-manager автоматически внедрял CA bundle из Certificate `rbacgraph-ca` в поле `caBundle` APIService. Это позволяет kube-apiserver проверять TLS-сертификат агрегированного API-сервера.

### Ротация сертификатов

Все сертификаты имеют срок действия 90 дней с окном обновления 15 дней. cert-manager обрабатывает ротацию автоматически — ручное вмешательство не требуется.

---

## Namespace

Все компоненты разворачиваются в namespace `rbac-graph-system`. Исключения: `ClusterIssuer` и `APIService` — кластерные ресурсы, а RoleBinding `auth-reader` находится в `kube-system`.
