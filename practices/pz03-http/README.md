# PZ03 HTTP Tasks API

Учебный HTTP-сервер на стандартной библиотеке `net/http`: хранит задачи в памяти, фильтрует список по заголовку и возвращает JSON-ответы. Сторонние зависимости не используются.

## Запуск в Windows PowerShell

Нужен Go 1.21 или новее. Из корня репозитория подключите локальный SDK этой машины (при обычной установке Go в `PATH` этот шаг не нужен), затем перейдите в модуль и задайте свободный порт:

```powershell
& 'D:\MAG\GO\practic\setup.ps1'
Set-Location 'D:\MAG\GO\practic\practices\pz03-http'
$env:PORT = '18083'
.\dev.ps1 run
```

Порт по умолчанию — `8080`; на компьютере автора он занят внешним `httpd`, поэтому для проверки используется `PORT=18083`. Не останавливайте сторонний процесс.

В отдельном окне PowerShell запустите setup (если используете локальный SDK), перейдите в папку модуля и выполните запросы из [`requests.md`](requests.md). Каждый запуск сервера создаёт пустое in-memory хранилище; задачи пропадают при завершении процесса.

## Маршруты

- `GET /health` — `200 OK`, JSON со статусом `ok`.
- `GET /tasks` — список задач; необязательный `q` фильтрует заголовок по подстроке без учёта регистра.
- `POST /tasks` — создаёт задачу из JSON `{"title":"..."}`, удаляя пробелы по краям; успешный ответ — `201 Created`.
- `GET /tasks/{id}` — возвращает задачу; неверный числовой ID даёт `400`, отсутствующая задача — `404`.
- Middleware пишет метод, путь, фактический статус и длительность каждого запроса.

Неверный JSON, пустой заголовок задачи и неподдерживаемый `Content-Type` возвращают `400 Bad Request` с JSON-описанием ошибки.

## Команды разработчика

Из каталога `practices/pz03-http`:

```powershell
.\dev.ps1 run
.\dev.ps1 build
.\dev.ps1 test
```

Скрипт предоставляет цели `run`, `build` и `test`. Эквивалентные проверки: `go fmt ./...`, `go vet ./...`, `go test ./...`, `go build ./...`.

## Структура

```text
cmd/server/main.go
internal/api/handlers.go
internal/api/middleware.go
internal/api/responses.go
internal/api/handlers_test.go
internal/storage/memory.go
go.mod
dev.ps1
requests.md
README.md
```
