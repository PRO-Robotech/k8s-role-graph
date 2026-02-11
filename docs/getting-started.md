# Быстрый старт

## Предварительные требования

| Требование | Версия | Примечание |
|---|---|---|
| Кластер Kubernetes | v1.28+ | Любой конформный кластер (kind, k3s, EKS, GKE, AKS и т.д.) |
| kubectl | v1.28+ | Должен быть настроен для работы с целевым кластером |
| cert-manager | v1.12+ | Необходим для автоматической выдачи TLS-сертификатов |
| kustomize | v5.0+ | Или используйте встроенный `kubectl apply -k` |

### Установка cert-manager (если ещё не установлен)

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml
kubectl wait --for=condition=Available deployment/cert-manager -n cert-manager --timeout=120s
kubectl wait --for=condition=Available deployment/cert-manager-webhook -n cert-manager --timeout=120s
```

## Деплой

Примените базовую kustomization — все ресурсы будут созданы в namespace `rbac-graph-system`:

```bash
kubectl apply -k deploy/kustomize/base
```

Создаются следующие ресурсы:

| Ресурс | Имя | Описание |
|---|---|---|
| Namespace | `rbac-graph-system` | Выделенный namespace для всех компонентов |
| ServiceAccount | `rbacgraph-apiserver` | Идентификатор агрегированного API-сервера |
| ServiceAccount | `rbacgraph-web` | Идентификатор веб-фронтенда |
| ClusterRole + Binding | `rbacgraph-apiserver-rbac-reader` | Доступ на чтение RBAC, подов и воркнагрузок по всему кластеру |
| ClusterRole + Binding | `rbacgraph-web-query` | Разрешение на создание `rolegraphreviews` |
| ClusterRoleBinding | `rbacgraph-apiserver-auth-delegator` | Делегирование аутентификации к kube-apiserver |
| RoleBinding | `rbacgraph-apiserver-auth-reader` | Чтение конфигурации аутентификации из `kube-system` |
| ClusterIssuer | `rbacgraph-selfsigned-root` | Самоподписанный корень для цепочки cert-manager |
| Certificate | `rbacgraph-ca` | CA-сертификат (ротация каждые 90 дней) |
| Issuer | `rbacgraph-ca-issuer` | Выдаёт серверные сертификаты от CA |
| Certificate | `rbacgraph-serving-cert` | TLS-сертификат для API-сервера (ротация каждые 90 дней) |
| Deployment | `rbacgraph-apiserver` | Агрегированный API-сервер (1 реплика) |
| Service | `rbacgraph-apiserver` | ClusterIP-сервис на порту 443 |
| Deployment | `rbacgraph-web` | Веб-фронтенд-прокси (1 реплика) |
| Service | `rbacgraph-web` | ClusterIP-сервис на порту 80 |
| APIService | `v1alpha1.rbacgraph.incloud.io` | Регистрация API в kube-apiserver |

## Проверка деплоя

### 1. Убедитесь, что поды запущены

```bash
kubectl get pods -n rbac-graph-system
```

Ожидаемый вывод:

```
NAME                                    READY   STATUS    RESTARTS   AGE
rbacgraph-apiserver-...                 1/1     Running   0          30s
rbacgraph-web-...                       1/1     Running   0          30s
```

### 2. Проверьте статус APIService

```bash
kubectl get apiservice v1alpha1.rbacgraph.incloud.io
```

Колонка `AVAILABLE` должна показывать `True`:

```
NAME                                SERVICE                                  AVAILABLE   AGE
v1alpha1.rbacgraph.incloud.io       rbac-graph-system/rbacgraph-apiserver    True        1m
```

Если показывается `False`, проверьте, что cert-manager выдал серверный сертификат:

```bash
kubectl get certificate -n rbac-graph-system
```

### 3. Проверьте готовность API

```bash
kubectl get --raw /apis/rbacgraph.incloud.io/v1alpha1
```

Должен вернуться документ обнаружения API-группы.

## Первый запрос

Отправьте минимальный запрос для поиска всех ролей, предоставляющих доступ к `pods/exec`:

```bash
kubectl create --raw /apis/rbacgraph.incloud.io/v1alpha1/rolegraphreviews -f - <<'EOF'
{
  "apiVersion": "rbacgraph.incloud.io/v1alpha1",
  "kind": "RoleGraphReview",
  "metadata": {"name": "my-first-query"},
  "spec": {
    "selector": {
      "apiGroups": [""],
      "resources": ["pods/exec"],
      "verbs": ["get", "create"]
    }
  }
}
EOF
```

Ответ содержит:
- **`status.graph.nodes`** — найденные роли, привязки и субъекты в виде узлов графа
- **`status.graph.edges`** — связи между узлами (grants, subjects и т.д.)
- **`status.resourceMap`** — сводная таблица найденных API-ресурсов
- **`status.matchedRoles`**, **`status.matchedBindings`**, **`status.matchedSubjects`** — счётчики

Для удобного чтения используйте `jq`:

```bash
kubectl create --raw /apis/rbacgraph.incloud.io/v1alpha1/rolegraphreviews -f request.json | jq .
```

## Доступ к веб-интерфейсу

Пробросьте порт веб-сервиса на локальную машину:

```bash
kubectl port-forward -n rbac-graph-system svc/rbacgraph-web 8080:80
```

Откройте [http://localhost:8080](http://localhost:8080) в браузере. Веб-интерфейс предоставляет интерактивную визуализацию графа и поддерживает упрощённые форматы запросов (см. [Руководство по запросам](query-guide.md)).

## Что дальше

- [Справочник API](api-reference.md) — полная спецификация всех полей запроса/ответа
- [Руководство по запросам](query-guide.md) — практические рецепты и упрощённые форматы
- [Справочник CLI](cli-reference.md) — все флаги командной строки и эндпоинты
- [Архитектура](architecture.md) — как компоненты работают вместе
