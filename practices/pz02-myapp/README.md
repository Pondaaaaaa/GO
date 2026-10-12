# MyApp — структура Go-проекта (PZ02)

Небольшой HTTP-сервис, показывающий разделение точки входа, внутренней логики приложения, обработчиков и вспомогательных функций.

## Запуск

Нужен Go 1.21 или новее. На этой рабочей копии перед первым запуском подключите локальный SDK командой `& '.\setup.ps1'` из корня репозитория. Скрипт machine-specific и не публикуется; при обычном клонировании пропустите его и используйте Go из `PATH`. Затем выполните из корня:

```powershell
Set-Location .\practices\pz02-myapp
$env:APP_PORT = '18082'
go run ./cmd/myapp
```

По заданию порт по умолчанию — `8080`. На компьютере автора он занят внешним `httpd`, поэтому для локального запуска и примеров используется `APP_PORT=18082`. Чтобы использовать стандартный порт, удалите переменную (`Remove-Item Env:APP_PORT`) перед запуском.

Во втором PowerShell окне из корня репозитория повторно подключите SDK командой `& '.\setup.ps1'`, если Go не добавлен в `PATH` (для чистого клона с установленным Go эту команду пропустите), затем перейдите в модуль и вызовите маршруты:

```powershell
& '.\setup.ps1'
Set-Location .\practices\pz02-myapp
curl.exe --noproxy '*' -i http://127.0.0.1:18082/
curl.exe --noproxy '*' -i http://127.0.0.1:18082/ping
curl.exe --noproxy '*' -i -H 'X-Request-Id: demo-123' http://127.0.0.1:18082/ping
curl.exe --noproxy '*' -i http://127.0.0.1:18082/fail
```

Остановите сервер `Ctrl+C`. Сервер журналирует время UTC, удалённый адрес, метод и путь запроса.

## Маршруты

- `GET /` — `Hello, Go project structure!`.
- `GET /ping` — JSON со статусом `ok` и текущим временем UTC в RFC3339.
- `GET /fail` — пример ошибки `400 Bad Request` в JSON.
- Каждый ответ содержит `X-Request-Id`: входной идентификатор сохраняется, иначе генерируется новый 16-символьный hex ID.

## Проверки и сборка

Из каталога модуля:

```powershell
go fmt ./...
go vet ./...
go build ./...
```

Для необязательной сборки исполняемого файла: `go build -o .\myapp.exe .\cmd\myapp`.

## Структура

```text
cmd/myapp/main.go
internal/app/app.go
internal/app/handlers/ping.go
utils/logger.go
utils/httpjson.go
go.mod
.gitignore
README.md
```
