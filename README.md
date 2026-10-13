# Практические занятия по Go

В репозитории собраны **16 самостоятельных практик**: от первого HTTP-сервиса до интеграционных тестов API с PostgreSQL. Каждая работа находится в отдельном Go-модуле и имеет собственный README с командами запуска. Для каждой практики подготовлены отчёты в Markdown и Word.

## Быстрый старт на Windows

Установите [Git](https://git-scm.com/install/windows) и [Go](https://go.dev/dl/) **1.27.2 или новее**. Обычно Go доступен в новом окне PowerShell через `PATH`; шаблон настройки ниже поддерживает и локальную установку рядом с репозиторием. Для практик с дополнительными сервисами смотрите таблицу требований ниже.

```powershell
git clone https://github.com/Pondaaaaaa/GO.git
Set-Location .\GO
Copy-Item .\setup.example.ps1 .\setup.ps1
& .\setup.ps1
go version
```

Если Git не нужен, на странице репозитория выберите **Code → Download ZIP**, распакуйте архив, откройте PowerShell в полученной папке `GO-main` и выполняйте команды выше начиная с `Copy-Item`.

`setup.example.ps1` — переносимый шаблон: он использует Go из `PATH`, а при его отсутствии ищет локальную установку рядом с репозиторием в `..\.local\go\bin`. Скопированный `setup.ps1` остаётся только на вашем компьютере и исключён из Git. Если Go не найден, установите его и откройте новое окно PowerShell.

Для первого знакомства запустите практику №1 из корня клона:

```powershell
Set-Location .\practices\pz01-helloapi
go mod download
go run ./cmd/server
```

В другом окне проверьте `curl.exe -i http://127.0.0.1:8080/hello`. Если порт 8080 занят, перед `go run` задайте `$env:APP_PORT = '18080'` и замените порт в запросе. Подробности и остальные маршруты — в [README практики №1](practices/pz01-helloapi/README.md).

## Навигация по практикам

| № | Тема | Код и запуск | Отчёты |
| --- | --- | --- | --- |
| 01 | Первый HTTP-сервис | [PZ01](practices/pz01-helloapi/README.md) | [Markdown](reports/pz01.md) · [Word](reports/word/pz01.docx) |
| 02 | Структура Go-проекта | [PZ02](practices/pz02-myapp/README.md) | [Markdown](reports/pz02.md) · [Word](reports/word/pz02.docx) |
| 03 | API задач на `net/http` | [PZ03](practices/pz03-http/README.md) | [Markdown](reports/pz03.md) · [Word](reports/word/pz03.docx) |
| 04 | Chi и CRUD ToDo | [PZ04](practices/pz04-todo/README.md) | [Markdown](reports/pz04.md) · [Word](reports/word/pz04.docx) |
| 05 | PostgreSQL и `database/sql` | [PZ05](practices/pz05-db/README.md) | [Markdown](reports/pz05.md) · [Word](reports/word/pz05.docx) |
| 06 | GORM и связи моделей | [PZ06](practices/pz06-gorm/README.md) | [Markdown](reports/pz06.md) · [Word](reports/word/pz06.docx) |
| 07 | Redis, SET/GET/TTL | [PZ07](practices/pz07-redis/README.md) | [Markdown](reports/pz07.md) · [Word](reports/word/pz07.docx) |
| 08 | MongoDB и CRUD заметок | [PZ08](practices/pz08-mongo/README.md) | [Markdown](reports/pz08.md) · [Word](reports/word/pz08.docx) |
| 09 | Регистрация, вход и bcrypt | [PZ09](practices/pz09-auth/README.md) | [Markdown](reports/pz09.md) · [Word](reports/word/pz09.docx) |
| 10 | JWT и права доступа | [PZ10](practices/pz10-auth/README.md) | [Markdown](reports/pz10.md) · [Word](reports/word/pz10.docx) |
| 11 | REST API заметок | [PZ11](practices/pz11-notes/README.md) | [Markdown](reports/pz11.md) · [Word](reports/word/pz11.docx) |
| 12 | Swagger-документация Notes API | [PZ12](practices/pz12-notes/README.md) | [Markdown](reports/pz12.md) · [Word](reports/word/pz12.docx) |
| 13 | Профилирование и бенчмарки | [PZ13](practices/pz13-pprof/README.md) | [Markdown](reports/pz13.md) · [Word](reports/word/pz13.docx) |
| 14 | Оптимизация PostgreSQL | [PZ14](practices/pz14-notes-db/README.md) | [Markdown](reports/pz14.md) · [Word](reports/word/pz14.docx) |
| 15 | Unit-тесты и покрытие | [PZ15](practices/pz15-tests/README.md) | [Markdown](reports/pz15.md) · [Word](reports/word/pz15.docx) |
| 16 | Интеграционные тесты с Docker | [PZ16](practices/pz16-integration/README.md) | [Markdown](reports/pz16.md) · [Word](reports/word/pz16.docx) |

## Что нужно для разных работ

| Практики | Дополнительные инструменты |
| --- | --- |
| 01–04, 10–13, 15 | Достаточно Go. Для графа профиля в №13 дополнительно полезен [Graphviz](https://graphviz.org/download/); остальные проверки работают без него. |
| 05, 06, 09, 14 | Локальный [PostgreSQL](https://www.postgresql.org/download/windows/) с `psql.exe` в `PATH`. Скрипты по умолчанию подключаются к `127.0.0.1:5432`, используют роль `postgres` и запрашивают её пароль; роль должна иметь право создать учебную БД. При другой роли задайте `$env:PGUSER`. |
| 07, 08, 16 | [Docker Desktop](https://docs.docker.com/desktop/setup/install/windows/) с запущенным Linux Engine; для №08 и №16 нужна команда `docker compose`. Контейнеры этих практик создаются и удаляются их скриптами. |

Проверка установленных инструментов:

```powershell
go version
git --version
psql --version
docker version
docker compose version
```

Команды для неустановленных дополнительных инструментов выдадут ошибку — установите их только перед соответствующей практикой. Для загрузки Go-зависимостей, Docker-образов и генератора Swagger в №12 при первом запуске нужен доступ к сети.

## Как запускать остальные работы

Откройте **README нужной практики** по ссылке в таблице: там указаны её команда запуска, маршруты и проверки. Многие практики содержат `run-local.ps1`, который сам подготавливает учебную среду, выполняет сценарий и завершает свои процессы. Например, из корня репозитория:

```powershell
& .\practices\pz16-integration\run-local.ps1
```

Каждая папка `practices/pzNN-*` — отдельный Go-модуль. Команды `go mod download`, `go test ./...` и `go build ./...` выполняйте **внутри выбранной папки**, а не в корне репозитория. В примерах некоторых внутренних README указан исходный путь автора `D:\MAG\GO\practic`; после клонирования заменяйте его путём к своему клону. Для `run-local.ps1` удобнее использовать относительный вызов из примера выше.

Исходные задания DOCX не опубликованы. Краткое описание требований находится в [`context/`](context/INDEX.md), промпты для Codex — в [`prompts/`](prompts/). Подлинные скриншоты и технические логи, когда они есть, расположены в [`reports/images/`](reports/images/) и [`reports/logs/`](reports/logs/). Пароли, локальные настройки и установленный Go в репозиторий не входят.
