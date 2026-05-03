# **СОДЕРЖАНИЕ** {#содержание}

[**СОДЕРЖАНИЕ**](#содержание)

[**1. НАЗНАЧЕНИЕ ПРОГРАММЫ**](#назначение-программы)

[1.1. Наименование программы](#наименование-программы)

[1.2. Краткая характеристика области применения](#краткая-характеристика-области-применения)

[**2. УСЛОВИЯ ВЫПОЛНЕНИЯ ПРОГРАММЫ**](#условия-выполнения-программы)

[2.1. Рекомендуемый состав аппаратурных и программных средств в локальной среде](#рекомендуемый-состав-аппаратурных-и-программных-средств-в-локальной-среде)

[2.2. Минимальный состав аппаратурных и программных средств в продуктовой среде](#минимальный-состав-аппаратурных-и-программных-средств-в-продуктовой-среде)

[2.3. Внешние зависимости и инфраструктурные компоненты](#внешние-зависимости-и-инфраструктурные-компоненты)

[2.4. Состав сервисов системы](#состав-сервисов-системы)

[**3. ВЫПОЛНЕНИЕ ПРОГРАММЫ**](#выполнение-программы)

[3.1. Подготовка исходного кода](#подготовка-исходного-кода)

[3.2. Подготовка локальной инфраструктуры](#подготовка-локальной-инфраструктуры)

[3.3. Настройка переменных окружения и секретов для локального запуска](#настройка-переменных-окружения-и-секретов-для-локального-запуска)

[3.4. Применение миграций баз данных](#применение-миграций-баз-данных)

[3.5. Порядок запуска сервисов в локальной среде](#порядок-запуска-сервисов-в-локальной-среде)

[3.6. Запуск пользовательского frontend](#запуск-пользовательского-frontend)

[3.7. Контрольный сквозной сценарий работы оператора](#контрольный-сквозной-сценарий-работы-оператора)

[3.8. Развертывание в Kubernetes (целевой контур)](#развертывание-в-kubernetes-целевой-контур)

[3.9. Подготовка Vault и External Secrets](#подготовка-vault-и-external-secrets)

[3.10. Развертывание сервисов через Helmfile](#развертывание-сервисов-через-helmfile)

[3.11. Включение и проверка Istio API Gateway](#включение-и-проверка-istio-api-gateway)

[3.12. Обновление образов и rollout](#обновление-образов-и-rollout)

[3.13. Остановка и перезапуск компонентов](#остановка-и-перезапуск-компонентов)

[**4. СООБЩЕНИЯ ОПЕРАТОРУ**](#сообщения-оператору)

[4.1. Формат ошибок Go-сервисов (grpc-gateway)](#формат-ошибок-go-сервисов-grpc-gateway)

[4.2. Формат ошибок Rust-сервисов](#формат-ошибок-rust-сервисов)

[4.3. Основные коды ответов и интерпретация](#основные-коды-ответов-и-интерпретация)

[4.4. Типовые инциденты и действия оператора](#типовые-инциденты-и-действия-оператора)

[**5. КОНТРОЛЬ СОСТОЯНИЯ И НАБЛЮДАЕМОСТЬ**](#контроль-состояния-и-наблюдаемость)

[5.1. Проверка состояния сервисов](#проверка-состояния-сервисов)

[5.2. Логи и диагностика](#логи-и-диагностика)

[5.3. Метрики и дашборды](#метрики-и-дашборды)

[**6. СПИСОК ИСТОЧНИКОВ**](#список-источников)

[**ПРИЛОЖЕНИЕ 1. Терминология**](#приложение-1-терминология)

[**ПРИЛОЖЕНИЕ 2. Оперативные команды оператора**](#приложение-2-оперативные-команды-оператора)

[**Лист регистрации изменений**](#лист-регистрации-изменений)

1. # **НАЗНАЧЕНИЕ ПРОГРАММЫ** {#назначение-программы}

   1. ## **Наименование программы** {#наименование-программы}

      Наименование программного средства: **CryptoTax**.  
      Назначение: автоматизация обработки операций с криптоактивами, расчета налоговых обязательств и формирования итогового отчетного артефакта для пользователя.

   2. ## **Краткая характеристика области применения** {#краткая-характеристика-области-применения}

      Программа применяется в составе веб-сервиса для подготовки налоговой отчетности по операциям с криптовалютой.  
      Система поддерживает:

      1. регистрацию и аутентификацию пользователя;
      2. загрузку и нормализацию биржевых операций;
      3. стоимостную оценку транзакций в целевой фиатной валюте;
      4. формирование агрегированного представления транзакций;
      5. запуск расчета налогового задания;
      6. формирование XML-отчета (на текущем этапе — 3-НДФЛ, юрисдикция RU).

2. # **УСЛОВИЯ ВЫПОЛНЕНИЯ ПРОГРАММЫ** {#условия-выполнения-программы}

   1. ## **Рекомендуемый состав аппаратурных и программных средств в локальной среде** {#рекомендуемый-состав-аппаратурных-и-программных-средств-в-локальной-среде}

      Рекомендуется персональный компьютер со следующими характеристиками:

      1. ОС: Linux / macOS / Windows;
      2. CPU: от 4 vCPU;
      3. RAM: от 16 ГБ;
      4. свободное место на диске: от 40 ГБ;
      5. стабильный доступ в Интернет.

      Необходимые программные средства:

      1. `git`;
      2. `docker` и `docker compose`;
      3. `kubectl`;
      4. `helm`;
      5. `helmfile`;
      6. `minikube`.

   2. ## **Минимальный состав аппаратурных и программных средств в продуктовой среде** {#минимальный-состав-аппаратурных-и-программных-средств-в-продуктовой-среде}

      Минимальная целевая конфигурация для стенда:

      1. Kubernetes-кластер (минимум 2 worker-узла);
      2. совокупные ресурсы кластера: от 4 vCPU и 8 ГБ RAM;
      3. внешние или выделенные инфраструктурные инстансы PostgreSQL, Redis, RabbitMQ, MinIO;
      4. установленный Istio ingress gateway;
      5. установленный External Secrets Operator;
      6. доступ к секретам в Vault.

      Рекомендуемая конфигурация для стабильной эксплуатации:

      1. от 8 vCPU и 16 ГБ RAM в кластере;
      2. выделенные инстансы для PostgreSQL и MinIO;
      3. отдельный контур наблюдаемости (OTel + Prometheus + Loki + Grafana).

   3. ## **Внешние зависимости и инфраструктурные компоненты** {#внешние-зависимости-и-инфраструктурные-компоненты}

      Программа использует следующие инфраструктурные компоненты:

      1. PostgreSQL — постоянное хранение доменных данных;
      2. Redis — кеш и блокировки;
      3. RabbitMQ — асинхронные события импорта;
      4. MinIO (S3-совместимое хранилище) — хранение артефактов отчета и аудита;
      5. Vault + External Secrets Operator — централизованное управление секретами;
      6. Istio Gateway/VirtualService/AuthorizationPolicy — API gateway и политики доступа;
      7. OpenTelemetry Collector, Prometheus, Loki, Grafana — наблюдаемость.

   4. ## **Состав сервисов системы** {#состав-сервисов-системы}

      Backend состоит из 6 сервисов:

      1. `auth-svc` (Rust) — регистрация, логин, refresh, logout, email verify;
      2. `ledger-svc` (Rust) — импорт CSV, нормализация операций, outbox событий;
      3. `price-svc` (Go) — историческая оценка активов в фиат;
      4. `aggregation-svc` (Go) — read-model транзакций и пользовательские настройки;
      5. `tax-svc` (Go) — налоговый профиль, запуск и обработка расчетного задания;
      6. `report-svc` (Go) — генерация XML-отчета и загрузка в MinIO.

3. # **ВЫПОЛНЕНИЕ ПРОГРАММЫ** {#выполнение-программы}

   1. ## **Подготовка исходного кода** {#подготовка-исходного-кода}

      Используются два репозитория:

      1. `CryptoTax-Go` — Go-сервисы, proto-контракты, frontend, Helmfile;
      2. `CryptoTax` — Rust-сервисы `auth-svc` и `ledger-svc`.

      Пример:

      ```bash
      cd /work
      git clone https://github.com/NightRunnerEB/CryptoTax-Go.git
      git clone https://github.com/NightRunnerEB/CryptoTax.git
      ```

   2. ## **Подготовка локальной инфраструктуры** {#подготовка-локальной-инфраструктуры}

      В `CryptoTax-Go` используется `docker-compose.yml` для инфраструктурных и вспомогательных компонентов:

      ```bash
      cd /work/CryptoTax-Go
      docker compose up -d
      ```

      Поднимаются: Loki, OTel Collector, Prometheus, Grafana, RabbitMQ, MinIO, MinIO-init, Vault, Swagger UI.

      Дополнительно для полного локального запуска сервисов должны быть доступны PostgreSQL и Redis (локально, в контейнерах или как managed-сервисы).

   3. ## **Настройка переменных окружения и секретов для локального запуска** {#настройка-переменных-окружения-и-секретов-для-локального-запуска}

      Перед запуском необходимо заполнить `.env`/`.env.example` в директориях сервисов.

      Критичные параметры:

      1. для Go-сервисов — `DATABASE_URL`, `REDIS_URL`, адреса upstream-сервисов, ключи MinIO;
      2. для `price-svc` — `COINGECKO_API_KEY`;
      3. для `auth-svc` — JWT-конфигурация, SMTP-параметры, `TAX_SVC_URL`;
      4. для `ledger-svc` — `DATABASE_URL`, `worker_config.yaml` (RabbitMQ).

      Важно: для `report-svc` и `tax-svc` должен существовать bucket MinIO `tax-reports`.

   4. ## **Применение миграций баз данных** {#применение-миграций-баз-данных}

      Go-сервисы:

      ```bash
      cd services/price-svc && make migrate-up
      cd ../aggregation-svc && make migrate-up
      cd ../tax-svc && make migrate-up
      ```

      Rust-сервисы:

      ```bash
      cd /work/CryptoTax/auth-svc && make migrate-up
      cd /work/CryptoTax/ledger-svc && make migrate-up
      ```

   5. ## **Порядок запуска сервисов в локальной среде** {#порядок-запуска-сервисов-в-локальной-среде}

      Рекомендуемый порядок:

      1. `price-svc`;
      2. `aggregation-svc`;
      3. `report-svc`;
      4. `tax-svc`;
      5. `auth-svc`;
      6. `ledger-svc`.

      Команды запуска:

      ```bash
      # Go-сервисы
      cd /work/CryptoTax-Go/services/price-svc && make run-grpc
      cd /work/CryptoTax-Go/services/aggregation-svc && make run-grpc
      cd /work/CryptoTax-Go/services/report-svc && make run-grpc
      cd /work/CryptoTax-Go/services/tax-svc && make run-grpc

      # Rust-сервисы (из workspace)
      cd /work/CryptoTax && cargo run -p auth-svc
      cd /work/CryptoTax && cargo run -p ledger-svc
      ```

      Локальные порты сервисов:

      1. `auth-svc`: `8085` (HTTP);
      2. `ledger-svc`: `8086` (HTTP);
      3. `price-svc`: `8093` (gRPC);
      4. `aggregation-svc`: `8094` (gRPC), `8095` (HTTP grpc-gateway);
      5. `tax-svc`: `8096` (gRPC), `8097` (HTTP grpc-gateway);
      6. `report-svc`: `8098` (gRPC).

   6. ## **Запуск пользовательского frontend** {#запуск-пользовательского-frontend}

      ```bash
      cd /work/CryptoTax-Go/frontend
      cp .env.example .env
      npm install
      npm run dev
      ```

      По умолчанию frontend использует `VITE_API_BASE_URL=http://localhost:8080`, то есть ожидает доступ через API gateway.

   7. ## **Контрольный сквозной сценарий работы оператора** {#контрольный-сквозной-сценарий-работы-оператора}

      Базовый сценарий пользовательской проверки:

      1. регистрация нового пользователя (`/auth/register`);
      2. вход (`/auth/login`) и получение токенов;
      3. загрузка CSV (`/mexc/csv`);
      4. просмотр транзакций (`/transactions`);
      5. заполнение налогового профиля (`/tax/profile`);
      6. запуск задания (`/tax/reports:start`);
      7. получение статуса (`/tax/reports/{report_id}`);
      8. проверка наличия `report_url` в завершенном задании.

   8. ## **Развертывание в Kubernetes (целевой контур)** {#развертывание-в-kubernetes-целевой-контур}

      Для локального кластера на Minikube:

      ```bash
      minikube start
      kubectl config current-context
      ```

      Для развёртывания используется `helmfile.yaml.gotmpl` в `CryptoTax-Go`, который устанавливает все 6 сервисов.

   9. ## **Подготовка Vault и External Secrets** {#подготовка-vault-и-external-secrets}

      1. создать namespace и `vault-token`:

      ```bash
      kubectl create ns cryptotax --dry-run=client -o yaml | kubectl apply -f -
      export VAULT_TOKEN="${VAULT_DEV_ROOT_TOKEN_ID:-root}"
      kubectl -n cryptotax create secret generic vault-token \
        --from-literal=token="${VAULT_TOKEN}" \
        --dry-run=client -o yaml | kubectl apply -f -
      ```

      2. применить `SecretStore`:

      ```bash
      kubectl apply -f deploy/k8s/eso/secretstore-vault-backend.yaml
      ```

      3. убедиться, что `SecretStore` доступен:

      ```bash
      kubectl -n cryptotax get secretstore vault-backend
      ```

      Пути и обязательные группы ключей в Vault:

      1. `secret/cryptotax/price-svc`: `APP_VERSION`, `APP_ENV`, `DATABASE_URL`, `REDIS_URL`, `COINGECKO_API_KEY`, `OTEL_*`;
      2. `secret/cryptotax/aggregation-svc`: `APP_VERSION`, `APP_ENV`, `DATABASE_URL`, `REDIS_URL`, `LEDGER_SVC_BASE_URL`, `PRICE_SVC_ADDR`, `RABBITMQ_URL`, `OTEL_*`;
      3. `secret/cryptotax/tax-svc`: `APP_VERSION`, `APP_ENV`, `DATABASE_URL`, `AGGREGATION_SVC_ADDR`, `REPORT_SVC_ADDR`, `MINIO_*`, `OTEL_*`;
      4. `secret/cryptotax/report-svc`: `APP_VERSION`, `APP_ENV`, `MINIO_*`, `OTEL_*`;
      5. `secret/cryptotax/auth-svc`: `DATABASE_URL`, `REDIS_URL`, JWT/SMTP/парольные параметры, `DUMMY_PASSWORD_HASH`, `TAX_SVC_URL`;
      6. `secret/cryptotax/ledger-svc`: `DATABASE_URL`, `DB_MAX_CONNS`, `DB_CONN_TIMEOUT`.

   10. ## **Развертывание сервисов через Helmfile** {#развертывание-сервисов-через-helmfile}

      В `CryptoTax-Go`:

      ```bash
      export CRYPTOTAX_REPO_ROOT=/work/CryptoTax
      make deploy NAMESPACE=cryptotax
      make pods NAMESPACE=cryptotax
      make status NAMESPACE=cryptotax
      ```

      Если требуется развернуть только один сервис:

      ```bash
      helm upgrade --install report-svc deploy/helm/report-svc -n cryptotax --create-namespace --wait --timeout 5m
      ```

   11. ## **Включение и проверка Istio API Gateway** {#включение-и-проверка-istio-api-gateway}

      1. включить remote JWKS в Istio:

      ```bash
      kubectl -n istio-system set env deployment/istiod PILOT_JWT_ENABLE_REMOTE_JWKS=envoy
      kubectl -n istio-system rollout status deployment/istiod
      ```

      2. применить Istio-манифесты:

      ```bash
      kubectl apply -f deploy/k8s/istio/gateway.yaml
      kubectl apply -f deploy/k8s/istio/virtualservices.yaml
      kubectl apply -f deploy/k8s/istio/destinationrules.yaml
      kubectl apply -f deploy/k8s/istio/peerauthentication-strict.yaml
      kubectl apply -f deploy/k8s/istio/requestauthentication.yaml
      kubectl apply -f deploy/k8s/istio/ingress-authorizationpolicy.yaml
      kubectl apply -f deploy/k8s/istio/authorizationpolicies.yaml
      ```

      3. открыть локальный доступ к gateway:

      ```bash
      kubectl -n istio-system port-forward svc/istio-ingressgateway 8080:80
      ```

      4. проверить публичные маршруты:

      ```bash
      curl http://127.0.0.1:8080/.well-known/jwks.json
      curl -X GET http://127.0.0.1:8080/exchanges/supported
      ```

   12. ## **Обновление образов и rollout** {#обновление-образов-и-rollout}

      Пример для Minikube:

      ```bash
      cd /work/CryptoTax-Go
      minikube image build -t cryptotax/tax-svc:dev -f services/tax-svc/Dockerfile .
      minikube image build -t cryptotax/report-svc:dev -f services/report-svc/Dockerfile .
      helm upgrade --install tax-svc deploy/helm/tax-svc -n cryptotax --wait --timeout 5m
      helm upgrade --install report-svc deploy/helm/report-svc -n cryptotax --wait --timeout 5m
      kubectl -n cryptotax rollout restart deploy/tax-svc
      kubectl -n cryptotax rollout restart deploy/report-svc
      kubectl -n cryptotax rollout status deploy/tax-svc --timeout=180s
      kubectl -n cryptotax rollout status deploy/report-svc --timeout=180s
      ```

   13. ## **Остановка и перезапуск компонентов** {#остановка-и-перезапуск-компонентов}

      В Kubernetes:

      ```bash
      # остановить конкретный сервис
      kubectl -n cryptotax scale deploy/auth-svc --replicas=0

      # поднять обратно
      kubectl -n cryptotax scale deploy/auth-svc --replicas=1

      # перезапуск rollout
      kubectl -n cryptotax rollout restart deploy/auth-svc
      ```

      Все deployments namespace:

      ```bash
      make stop NAMESPACE=cryptotax
      make start NAMESPACE=cryptotax
      ```

4. # **СООБЩЕНИЯ ОПЕРАТОРУ** {#сообщения-оператору}

   1. ## **Формат ошибок Go-сервисов (grpc-gateway)** {#формат-ошибок-go-сервисов-grpc-gateway}

      Для HTTP-методов Go-сервисов (`aggregation-svc`, `tax-svc`) ошибки возвращаются в grpc-gateway формате:

      ```json
      {
        "code": 3,
        "message": "invalid argument",
        "details": [
          {
            "@type": "type.googleapis.com/google.rpc.ErrorInfo",
            "reason": "INVALID_ARGUMENT",
            "domain": "tax-svc",
            "metadata": {}
          },
          {
            "@type": "type.googleapis.com/google.rpc.BadRequest",
            "fieldViolations": [
              { "field": "inn", "description": "invalid checksum" }
            ]
          }
        ]
      }
      ```

   2. ## **Формат ошибок Rust-сервисов** {#формат-ошибок-rust-сервисов}

      `auth-svc` и `ledger-svc` возвращают ошибки в формате:

      ```json
      {
        "code": 422,
        "message": "invalid tax profile field 'inn': invalid checksum"
      }
      ```

   3. ## **Основные коды ответов и интерпретация** {#основные-коды-ответов-и-интерпретация}

      1. `200 OK` — успешное выполнение операции;
      2. `201 Created` — успешное создание ресурса (если используется сервисом);
      3. `400 Bad Request` — неверный формат входных данных, некорректные параметры;
      4. `401 Unauthorized` — отсутствует или невалиден токен;
      5. `403 Forbidden` — недостаточно прав или запрет доступа политикой;
      6. `404 Not Found` — ресурс не найден;
      7. `409 Conflict` — конфликт состояния/дублирование;
      8. `422 Unprocessable Entity` — семантическая ошибка валидации;
      9. `500 Internal Server Error` — внутренняя ошибка сервиса;
      10. `502/503` — ошибка/недоступность внешней зависимости.

   4. ## **Типовые инциденты и действия оператора** {#типовые-инциденты-и-действия-оператора}

      1. Ошибка регистрации `invalid inn` / `invalid checksum`  
         Действия: проверить корректность ИНН в профиле пользователя; убедиться, что frontend передаёт поле `inn`; повторить запрос.

      2. Ошибка в `tax-svc`: `request report render failed`  
         Действия: проверить логи `report-svc`; определить причину (`xsd validation`, `minio unavailable`, `storage response failed`).

      3. Ошибка `cannot create storage client` / `check minio bucket failed`  
         Действия: проверить доступность MinIO, ключи `MINIO_ACCESS_KEY/MINIO_SECRET_KEY`, наличие bucket `tax-reports`.

      4. `CrashLoopBackOff` у pod  
         Действия: `kubectl logs <pod> -c <container> --tail=200`; проверить секреты, переменные окружения и health probe.

      5. `UPGRADE FAILED: another operation ... is in progress`  
         Действия: дождаться завершения предыдущей операции Helm или диагностировать release через `helm history` и повторить `upgrade`.

      6. `FailedPrecondition` при рендере отчёта  
         Действия: проверить валидацию XSD в `report-svc`, входные поля NDFL payload, логи с `error.meta.details`.

5. # **КОНТРОЛЬ СОСТОЯНИЯ И НАБЛЮДАЕМОСТЬ** {#контроль-состояния-и-наблюдаемость}

   1. ## **Проверка состояния сервисов** {#проверка-состояния-сервисов}

      ```bash
      kubectl get pods -n cryptotax -o wide
      kubectl get deploy -n cryptotax
      kubectl get svc -n cryptotax
      ```

      Проверка rollout:

      ```bash
      kubectl -n cryptotax rollout status deploy/tax-svc --timeout=180s
      kubectl -n cryptotax rollout status deploy/report-svc --timeout=180s
      ```

   2. ## **Логи и диагностика** {#логи-и-диагностика}

      ```bash
      # логи deployment
      kubectl -n cryptotax logs -f deploy/tax-svc -c tax-svc --tail=200
      kubectl -n cryptotax logs -f deploy/report-svc -c report-svc --tail=200

      # логи конкретного pod
      kubectl -n cryptotax logs report-svc-<pod-id> -c report-svc --tail=300

      # описание pod (events, probes, restart reason)
      kubectl -n cryptotax describe pod <pod-name>
      ```

   3. ## **Метрики и дашборды** {#метрики-и-дашборды}

      В локальной инфраструктуре:

      1. Grafana: `http://localhost:3000` (`admin/admin`);
      2. Prometheus: `http://localhost:9090`;
      3. Loki: `http://localhost:3100`.

      Дашборды и provisioning расположены в:

      1. `observability/grafana/dashboards/*.json`;
      2. `observability/grafana/datasources.yaml`;
      3. `observability/prometheus.yml`.

6. # **СПИСОК ИСТОЧНИКОВ** {#список-источников}

1. Репозиторий `NightRunnerEB/CryptoTax-Go`.
2. Репозиторий `NightRunnerEB/CryptoTax`.
3. Документация Istio: https://istio.io/
4. Документация Kubernetes: https://kubernetes.io/
5. Документация Helmfile: https://helmfile.readthedocs.io/
6. Документация External Secrets Operator: https://external-secrets.io/
7. Документация Vault: https://developer.hashicorp.com/vault
8. Документация Docker: https://www.docker.com/
9. Документация OpenTelemetry: https://opentelemetry.io/
10. Документация Prometheus: https://prometheus.io/
11. Документация Grafana: https://grafana.com/
12. Документация Loki: https://grafana.com/oss/loki/
13. Документация MinIO: https://min.io/
14. Документация RabbitMQ: https://www.rabbitmq.com/
15. Документация PostgreSQL: https://www.postgresql.org/
16. Документация Redis: https://redis.io/

# **ПРИЛОЖЕНИЕ 1. Терминология** {#приложение-1-терминология}

1. **Микросервис** — автономный компонент системы, реализующий ограниченную бизнес-функцию.
2. **gRPC** — бинарный протокол межсервисного взаимодействия на базе Protocol Buffers.
3. **grpc-gateway** — слой трансляции gRPC-контрактов в HTTP REST.
4. **API Gateway** — единая входная точка для внешних HTTP-запросов.
5. **Service Mesh** — инфраструктурный слой сетевого взаимодействия и политик безопасности сервисов.
6. **JWT** — подписанный токен доступа пользователя.
7. **mTLS** — взаимная TLS-аутентификация между сервисами внутри mesh.
8. **Helm chart** — пакет шаблонов Kubernetes-ресурсов.
9. **Helmfile** — декларативное управление набором Helm-релизов.
10. **ExternalSecret** — ресурс синхронизации секретов из внешнего хранилища (Vault) в Kubernetes Secret.
11. **NDFL** — декларация 3-НДФЛ, формируемая `report-svc`.

# **ПРИЛОЖЕНИЕ 2. Оперативные команды оператора** {#приложение-2-оперативные-команды-оператора}

```bash
# контекст и namespace
kubectl config current-context
kubectl get ns

# базовый статус
kubectl get pods -n cryptotax -o wide
kubectl get deploy,svc -n cryptotax

# логи
kubectl -n cryptotax logs -f deploy/auth-svc --all-containers --tail=200
kubectl -n cryptotax logs -f deploy/ledger-svc --all-containers --tail=200
kubectl -n cryptotax logs -f deploy/aggregation-svc --all-containers --tail=200
kubectl -n cryptotax logs -f deploy/tax-svc --all-containers --tail=200
kubectl -n cryptotax logs -f deploy/report-svc --all-containers --tail=200

# перезапуск
kubectl -n cryptotax rollout restart deploy/auth-svc
kubectl -n cryptotax rollout restart deploy/ledger-svc
kubectl -n cryptotax rollout restart deploy/aggregation-svc
kubectl -n cryptotax rollout restart deploy/tax-svc
kubectl -n cryptotax rollout restart deploy/report-svc

# масштабирование
kubectl -n cryptotax scale deploy/auth-svc --replicas=0
kubectl -n cryptotax scale deploy/auth-svc --replicas=1

# Istio gateway локально
kubectl -n istio-system port-forward svc/istio-ingressgateway 8080:80
```

# **Лист регистрации изменений** {#лист-регистрации-изменений}

| Изм. | Номера листов (страниц) | Всего листов в документе | № документа | Входящий № сопроводительного документа и дата | Подпись | Дата |
| --- | --- | --- | --- | --- | --- | --- |
| 0 | — | — | Первичная редакция | — | — | 23.04.2026 |
