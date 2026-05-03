#!/usr/bin/env python3

from __future__ import annotations

import copy
import sys
import zipfile
from pathlib import Path
import xml.etree.ElementTree as ET


W_NS = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
XML_NS = "http://www.w3.org/XML/1998/namespace"
ET.register_namespace("w", W_NS)


def w(tag: str) -> str:
    return f"{{{W_NS}}}{tag}"


def paragraph_text(paragraph: ET.Element) -> str:
    return "".join(node.text or "" for node in paragraph.findall(".//" + w("t"))).strip()


def make_paragraph(template: ET.Element, text: str) -> ET.Element:
    paragraph = copy.deepcopy(template)

    for child in list(paragraph):
        if child.tag != w("pPr"):
            paragraph.remove(child)

    run = None
    for node in template:
        if node.tag == w("r"):
            run = copy.deepcopy(node)
            break

    if run is None:
        run = ET.Element(w("r"))

    for child in list(run):
        if child.tag != w("rPr"):
            run.remove(child)

    text_node = ET.Element(w("t"))
    if text.startswith(" ") or text.endswith(" ") or "  " in text:
        text_node.set(f"{{{XML_NS}}}space", "preserve")
    text_node.text = text
    run.append(text_node)
    paragraph.append(run)

    return paragraph


def main() -> int:
    if len(sys.argv) != 2:
        print("usage: apply_tp_docx.py <docx-path>", file=sys.stderr)
        return 1

    docx_path = Path(sys.argv[1])
    if not docx_path.exists():
        print(f"file not found: {docx_path}", file=sys.stderr)
        return 1

    with zipfile.ZipFile(docx_path, "r") as src:
        xml_data = src.read("word/document.xml")
        other_files = {name: src.read(name) for name in src.namelist() if name != "word/document.xml"}

    root = ET.fromstring(xml_data)
    body = root.find(".//" + w("body"))
    if body is None:
        print("word/document.xml does not contain w:body", file=sys.stderr)
        return 1

    paragraphs = [node for node in body if node.tag == w("p")]

    section_template = next((p for p in paragraphs if paragraph_text(p) == "ВВЕДЕНИЕ"), None)
    subsection_template = next((p for p in paragraphs if paragraph_text(p) == "Наименование программы"), None)
    body_template = next(
        (p for p in paragraphs if paragraph_text(p).startswith("Разрабатываемая система предназначена")),
        None,
    )
    anchor = next((p for p in paragraphs if paragraph_text(p) == "ТЕКСТ ПРОГРАММЫ"), None)

    if section_template is None or subsection_template is None or body_template is None or anchor is None:
        print("failed to find template paragraphs in document", file=sys.stderr)
        return 1

    content = [
        ("subsection", "Полный текст программы"),
        (
            "body",
            "Полный текст программы расположен в двух репозиториях проекта CryptoTax: "
            "`NightRunnerEB/CryptoTax-Go` [1] и `NightRunnerEB/CryptoTax` [2]. Такое разделение "
            "связано с выбранной гетерогенной архитектурой: часть сервисов реализована на языке Go, "
            "а часть — на языке Rust.",
        ),
        (
            "body",
            "В репозитории `CryptoTax-Go` расположены общие контракты и основная часть backend-"
            "инфраструктуры. В корне репозитория находятся директории `api` с proto-описаниями "
            "gRPC-контрактов, `gen` со сгенерированным кодом, `pkg` с общими пакетами, `services` "
            "с исходными текстами сервисов, `docs` с архитектурной и API-документацией, `frontend` "
            "с веб-интерфейсом, а также `docker-compose.yml` для локального запуска системы.",
        ),
        (
            "body",
            "Внешний доступ к системе организован через HTTP API. Для пользовательских сценариев "
            "используются REST-маршруты, документированные в OpenAPI-спецификациях, а для внутренних "
            "высоконагруженных вызовов между сервисами используются gRPC и Protocol Buffers [3], [4]. "
            "Долгие и потенциально отказоустойчивые этапы обработки вынесены в асинхронный контур "
            "с использованием RabbitMQ [5].",
        ),
        ("subsection", "2.1 Общие пакеты и структура сервисов"),
        (
            "body",
            "Общие Go-пакеты расположены в директории `pkg` репозитория `CryptoTax-Go`. Они "
            "используются несколькими сервисами и обеспечивают единообразную работу с инфраструктурой "
            "и служебными механизмами.",
        ),
        (
            "body",
            "В проекте выделены следующие общие пакеты: `pkg/logger` — структурированное логирование; "
            "`pkg/postgres` — подключение к PostgreSQL [6]; `pkg/redis` — подключение к Redis [7]; "
            "`pkg/minio` — работа с S3-совместимым объектным хранилищем MinIO [8]; `pkg/telemetry` — "
            "инициализация OpenTelemetry [9]; `pkg/in-memory` — in-memory реализации для "
            "вспомогательных сценариев.",
        ),
        (
            "body",
            "Каждый Go-сервис имеет схожую внутреннюю структуру. В директории `cmd` расположена "
            "точка входа в сервис. Директория `internal` содержит основную реализацию и, как правило, "
            "делится на `app`, `config`, `domain`, `usecases`, `infra`, `server` и `worker`."
            " В директории `db` располагаются SQL-миграции и SQL-запросы для `sqlc`.",
        ),
        ("subsection", "2.2 Сервис аутентификации (auth-svc)"),
        (
            "body",
            "Сервис `auth-svc` расположен в репозитории `CryptoTax` и реализован на языке Rust [10]. "
            "Он отвечает за регистрацию пользователей, вход, обновление токенов доступа, завершение "
            "сессии и подтверждение адреса электронной почты.",
        ),
        (
            "body",
            "Исходный код сервиса расположен в директории `auth-svc/src`. Внутри выделены слои "
            "маршрутов, доменной логики, репозиториев и инфраструктуры базы данных. Сервис "
            "предоставляет REST API для пользовательских операций аутентификации и выступает "
            "источником данных о пользователе и tenant-контексте.",
        ),
        (
            "body",
            "При обращении пользователя к защищённым маршрутам `auth-svc` участвует в схеме доступа "
            "через выпуск и проверку JWT. Полученный идентификатор пользователя и tenant далее "
            "используются другими сервисами для изоляции данных и выполнения операций только в пределах "
            "соответствующего аккаунта.",
        ),
        ("subsection", "2.3 Сервис импорта и нормализации транзакций (ledger-svc)"),
        (
            "body",
            "Сервис `ledger-svc` расположен в репозитории `CryptoTax` и реализован на языке Rust. "
            "Он отвечает за приём CSV-файлов из внешних источников, валидацию строк, нормализацию "
            "операций и сохранение результата импорта.",
        ),
        (
            "body",
            "В составе сервиса выделены маршруты HTTP для загрузки файлов и получения результатов "
            "импорта, парсеры, специфичные для поддерживаемых бирж и форматов выгрузок, доменные "
            "структуры нормализованной транзакции и слой доступа к PostgreSQL.",
        ),
        (
            "body",
            "Сервис преобразует разнородные CSV-форматы к единой внутренней модели транзакции. "
            "На этом этапе определяются время операции, источник, тип действия, входящий актив, "
            "исходящий актив, комиссия и вспомогательные метаданные. После завершения успешного "
            "импорта `ledger-svc` публикует событие, по которому downstream-компоненты могут начать "
            "дальнейшую обработку данных.",
        ),
        ("subsection", "2.4 Сервис исторического прайсинга (price-svc)"),
        (
            "body",
            "Сервис `price-svc` расположен в директории `services/price-svc` репозитория "
            "`CryptoTax-Go`. Он реализован на языке Go и предназначен для стоимостной оценки "
            "криптовалютных операций в выбранной фиатной валюте.",
        ),
        (
            "body",
            "Сервис получает запросы по gRPC, разрешает тикеры активов в канонические идентификаторы "
            "внешнего источника цен, запрашивает исторические котировки и выполняет пересчёт в "
            "целевую фиатную валюту. Для хранения исторических данных используется PostgreSQL, "
            "для ускорения повторных вычислений — Redis.",
        ),
        (
            "body",
            "Внутри `price-svc` выделены подсистемы `internal/server`, `internal/usecases`, "
            "`internal/coingecko`, `internal/fiatfx` и `internal/infra`. Сервис проектируется как "
            "отдельный вычислительный контур, поскольку прайсинг зависит от внешних провайдеров, "
            "подвержен ограничениям по частоте запросов и требует кэширования.",
        ),
        ("subsection", "2.5 Сервис агрегации и read-model (aggregation-svc)"),
        (
            "body",
            "Сервис `aggregation-svc` расположен в директории `services/aggregation-svc` репозитория "
            "`CryptoTax-Go`. Он принимает нормализованные транзакции после импорта, запрашивает для "
            "них стоимостную оценку у `price-svc`, сохраняет обогащённые данные и предоставляет "
            "read-model для пользовательских запросов.",
        ),
        (
            "body",
            "Основные задачи сервиса: потребление события о завершении импорта, загрузка "
            "нормализованных транзакций из `ledger-svc`, батч-запрос стоимости операций в "
            "`price-svc`, сохранение агрегированных транзакций в собственной базе и предоставление "
            "API для списков транзакций, tenant-настроек и справочника поддерживаемых фиатных валют.",
        ),
        (
            "body",
            "В доменной модели `aggregation-svc` каждая агрегированная транзакция хранит не только "
            "исходные данные операции, но и результат стоимостной оценки: входящие, исходящие и "
            "комиссионные legs с криптовалютным количеством и фиатной стоимостью. Это делает сервис "
            "промежуточным, но критически важным слоем между импортом сырой истории операций и "
            "последующим налоговым расчётом.",
        ),
        ("subsection", "2.6 Сервис налогового расчёта (tax-svc)"),
        (
            "body",
            "Сервис `tax-svc` расположен в директории `services/tax-svc` репозитория `CryptoTax-Go`. "
            "Он реализует прикладную логику налогового расчёта, управление профилем налогоплательщика, "
            "создание и обработку заданий на расчёт, а также формирование итогового audit-результата.",
        ),
        (
            "body",
            "Внешний REST API сервиса публикуется через `grpc-gateway`. Сервис предоставляет методы "
            "для получения и обновления профиля налогоплательщика, запуска нового расчёта за налоговый "
            "период, получения статуса задания и получения списка ранее созданных заданий.",
        ),
        (
            "body",
            "После создания задания `tax-svc` переводит его в очередь на обработку. Фоновый воркер "
            "сервиса атомарно забирает задание, запрашивает готовые агрегированные транзакции через "
            "gRPC у `aggregation-svc`, выбирает движок расчёта в зависимости от налоговой политики и "
            "юрисдикции, а затем формирует результат расчёта. Результат включает события учёта, лоты, "
            "строки списания, итоговую налоговую сводку и audit-артефакты.",
        ),
        (
            "body",
            "Для поддержки расширяемости в сервисе выделен слой `engines`: каждая юрисдикция "
            "оформляется как отдельный движок расчёта с единым интерфейсом. Дополнительно `tax-svc` "
            "интегрирован с объектным хранилищем MinIO, в котором сохраняются audit-артефакты и иные "
            "файлы, а в базе данных хранятся только стабильные object keys.",
        ),
        ("subsection", "2.7 Сервис формирования отчётных документов (report-svc)"),
        (
            "body",
            "Сервис `report-svc` расположен в директории `services/report-svc` репозитория "
            "`CryptoTax-Go`. Он выделен как отдельный сервис формирования отчётных документов на "
            "основе уже подготовленного результата расчёта.",
        ),
        (
            "body",
            "В целевом варианте сервис принимает структурированный результат из `tax-svc`, формирует "
            "итоговый XML-документ, сохраняет его в объектное хранилище и возвращает object key, "
            "который далее используется `tax-svc` как ссылка на сформированный отчёт.",
        ),
        (
            "body",
            "Выделение генерации отчётных файлов в отдельный сервис позволяет изолировать шаблоны и "
            "форматирование отчётов от налогового движка, упростить расширение под новые виды "
            "документов и не смешивать вычислительную бизнес-логику и файловую генерацию в одном "
            "компоненте.",
        ),
        ("subsection", "2.8 API Gateway и пользовательский frontend"),
        (
            "body",
            "Внешний HTTP-контур системы строится вокруг Nginx и REST-маршрутов, публикуемых "
            "сервисами. Для Go-сервисов REST-доступ строится поверх `grpc-gateway`, что позволяет "
            "поддерживать единый gRPC-контракт внутри системы и при этом отдавать удобное HTTP API "
            "наружу.",
        ),
        (
            "body",
            "Контракты внешнего и внутреннего API документированы в директории `docs/api`. "
            "Веб-интерфейс проекта расположен в директории `frontend` и реализован на React [11] "
            "с использованием Vite [12]. Он предназначен для загрузки и просмотра импортов, выбора "
            "биржи и пользовательских настроек, просмотра агрегированных транзакций, запуска "
            "налогового расчёта и получения статуса задания.",
        ),
        ("subsection", "2.9 Инфраструктура запуска и наблюдаемость"),
        (
            "body",
            "Для локального запуска и отладки в проекте используется `docker-compose.yml`, в котором "
            "описан запуск инфраструктурных зависимостей и вспомогательных компонентов. В системе "
            "используются PostgreSQL, Redis, RabbitMQ, MinIO, а также стек наблюдаемости на базе "
            "OpenTelemetry, Prometheus, Loki и Grafana [9], [13], [14], [15].",
        ),
        (
            "body",
            "Каталог `observability` содержит конфигурации коллекторов телеметрии, Prometheus и "
            "Grafana. Такой подход позволяет контролировать состояние сервисов, анализировать "
            "структурированные логи и отслеживать производительность ключевых RPC и HTTP-методов.",
        ),
        ("section", "3 СПИСОК ИСТОЧНИКОВ"),
        ("body", "1. Репозиторий проекта `CryptoTax-Go`. URL: https://github.com/NightRunnerEB/CryptoTax-Go"),
        ("body", "2. Репозиторий проекта `CryptoTax`. URL: https://github.com/NightRunnerEB/CryptoTax"),
        ("body", "3. gRPC Documentation. URL: https://grpc.io/docs/"),
        ("body", "4. Protocol Buffers Documentation. URL: https://protobuf.dev/"),
        ("body", "5. RabbitMQ Documentation. URL: https://www.rabbitmq.com/documentation.html"),
        ("body", "6. PostgreSQL Documentation. URL: https://www.postgresql.org/docs/"),
        ("body", "7. Redis Documentation. URL: https://redis.io/docs/"),
        ("body", "8. MinIO Documentation. URL: https://min.io/docs/"),
        ("body", "9. OpenTelemetry Documentation. URL: https://opentelemetry.io/docs/"),
        ("body", "10. The Rust Programming Language. URL: https://doc.rust-lang.org/book/"),
        ("body", "11. React Documentation. URL: https://react.dev/"),
        ("body", "12. Vite Documentation. URL: https://vite.dev/guide/"),
        ("body", "13. Prometheus Documentation. URL: https://prometheus.io/docs/"),
        ("body", "14. Grafana Documentation. URL: https://grafana.com/docs/"),
        ("body", "15. Loki Documentation. URL: https://grafana.com/docs/loki/latest/"),
    ]

    body_children = list(body)
    insert_index = body_children.index(anchor) + 1

    new_nodes = []
    for kind, text in content:
        if kind == "section":
            new_nodes.append(make_paragraph(section_template, text))
        elif kind == "subsection":
            new_nodes.append(make_paragraph(subsection_template, text))
        else:
            new_nodes.append(make_paragraph(body_template, text))

    for offset, node in enumerate(new_nodes):
        body.insert(insert_index + offset, node)

    updated_xml = ET.tostring(root, encoding="utf-8", xml_declaration=True)

    tmp_path = docx_path.with_suffix(docx_path.suffix + ".tmp")
    with zipfile.ZipFile(tmp_path, "w", compression=zipfile.ZIP_DEFLATED) as dst:
        for name, payload in other_files.items():
            dst.writestr(name, payload)
        dst.writestr("word/document.xml", updated_xml)

    tmp_path.replace(docx_path)
    print(f"updated {docx_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
