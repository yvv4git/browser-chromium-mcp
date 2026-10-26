# browser-chromium-mcp

MCP-сервер для управления нативным Chromium через Chrome DevTools Protocol (CDP) с использованием [chromedp](https://github.com/chromedp/chromedp).

## Возможности

- **Управление браузером**: запуск Chromium или подключение к уже запущенному по CDP URL
- **Изолированные сессии**: несколько независимых browser context'ов через `sessionKey`
- **Множественные вкладки**: управление вкладками в рамках сессии
- **Навигация и действия**: `open_tab`, `navigate`, `back`, `forward`, `refresh`, `click`, `type`, `press`, `scroll`
- **Инспекция страницы**: `snapshot` (ARIA-снимок с элементами и `refs`), `links`, `evaluate`, `network_requests`, `stats`
- **Скриншоты**: `screenshot` (viewport или full-page), сохранение в PNG/JPEG с настройкой `quality`/`max_width`
- **Транспорты**: `stdio` и `streamable HTTP`
- **Гибкая конфигурация**: TOML, переменные окружения, CLI-флаги

## Требования

- Go 1.26+
- Chromium/Google Chrome (для запуска через `exec_path` или в headless-режиме)

## Установка

```bash
git clone https://github.com/yvv4git/browser-chromium-mcp.git
cd browser-chromium-mcp
go build ./...
```

## Конфигурация

Скопируйте пример конфига и при необходимости отредактируйте:

```bash
cp config.example.toml config.toml
```

Основные параметры:

| Параметр | Окружение | Описание | По умолчанию |
|---|---|---|---|
| `serve.cdp_url` | `SERVE_CDP_URL` | CDP URL для подключения к существующему Chromium. Если не задан - запускается новый экземпляр. | - |
| `serve.exec_path` | `SERVE_EXEC_PATH` | Путь к исполняемому файлу Chromium/Chrome. | - |
| `serve.headless` | `SERVE_HEADLESS` | Запуск в headless-режиме. | `true` |
| `serve.user_data_dir` | `SERVE_USER_DATA_DIR` | Путь к профилю пользователя. | - |
| `serve.transport` | `SERVE_TRANSPORT` | Транспорт: `stdio` или `http`. | `stdio` |
| `serve.http_addr` | `SERVE_HTTP_ADDR` | Адрес для HTTP-транспорта. | `:8080` |
| `serve.window_width`/`serve.window_height` | `SERVE_WINDOW_WIDTH`/`SERVE_WINDOW_HEIGHT` | Размер окна браузера. | `1280`/`800` |
| `log.level` | `LOG_LEVEL` | Уровень логирования (`debug`, `info`, `warn`, `error`). | `info` |

## Запуск

### Режим stdio (по умолчанию)

```bash
go run ./cmd/serve serve
# или после сборки
./browser-chromium-mcp serve
```

### Режим streamable HTTP

```bash
./browser-chromium-mcp serve --transport http --http-addr :8080
```

Или в `config.toml`:

```toml
[serve]
transport = "http"
http_addr = ":8080"
```

### Подключение к существующему Chromium

Запустите Chromium с включённым CDP:

```bash
chromium --remote-debugging-port=9222 --user-data-dir=/tmp/chrome-profile
```

Затем настройте подключение:

```toml
[serve]
cdp_url = "http://127.0.0.1:9222"
```

## Инструменты MCP

| Инструмент | Описание |
|---|---|
| `health` | Статус браузера, кол-во сессий, подключение |
| `open_tab` | Открыть новую вкладку в сессии (создаст сессию при необходимости) |
| `navigate` | Перейти по URL в указанной вкладке |
| `back`/`forward`/`refresh` | Навигация по истории |
| `list_tabs` | Список вкладок в сессии |
| `close_tab` | Закрыть вкладку |
| `close_session` | Закрыть сессию и все её вкладки |
| `snapshot` | ARIA-снимок страницы с элементами и ссылками на refs |
| `click`/`type`/`press` | Взаимодействие с элементами (по `ref` из snapshot или `selector`) |
| `scroll` | Скролл страницы (`up`/`down`/`left`/`right`) |
| `links` | Список ссылок на странице с пагинацией |
| `screenshot` | Скриншот (viewport или full-page), можно сохранить в файл с `quality`/`max_width` |
| `evaluate` | Выполнить JavaScript и получить результат |
| `network_requests` | Список сетевых запросов на вкладке |
| `stats` | Статистика вкладки (URL, title, посещённые URL, кол-во вызовов, кол-во refs) |

Все инструменты используют `sessionKey` для изоляции сессий и `tabId` для указания вкладки.

## Разработка

```bash
# Сборка
go build ./...

# Тесты
go test ./...

# Линтинг
go vet ./...
```

## Лицензия

[Apache License 2.0](LICENSE)
