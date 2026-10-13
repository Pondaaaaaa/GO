# PZ08 — MongoDB notes API

Отдельный модуль Go с официальным драйвером MongoDB и роутером chi. Приложение подключается к БД `pz8`, проверяет доступность MongoDB и создаёт уникальный индекс `unique_title` в коллекции `notes` до запуска HTTP-сервера.

## Одна команда для Windows

Запустите Docker Desktop и выполните из любого каталога PowerShell:

```powershell
& 'D:\MAG\GO\practic\practices\pz08-mongo\run-local.ps1'
```

Скрипт сам подключает локальный Go, собирает API, генерирует временные учётные данные MongoDB, создаёт уникальный Compose project и выбирает свободные порты на `127.0.0.1`. Затем проверяет пять CRUD-операций, фильтр и пагинацию, конфликт заголовка, неверный ObjectID, отсутствие документа и пустой ответ DELETE 204. После проверки он останавливает только запущенный им API и удаляет только свой Compose project с временным томом. Пароль и URI не печатаются и не сохраняются в репозитории.

## Ручной запуск через Docker Compose

Ниже приведён пример для отдельной PowerShell-сессии. Он создаёт случайные временные учётные данные и свободные порты; переменные живут только в текущем процессе PowerShell. Измените путь рабочего каталога, если репозиторий находится в другом месте.

```powershell
Set-Location D:\MAG\GO\practic
& .\setup.ps1
Set-Location .\practices\pz08-mongo
function Get-FreePort {
    $listener = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, 0)
    $listener.Start()
    try { ([System.Net.IPEndPoint]$listener.LocalEndpoint).Port }
    finally { $listener.Stop() }
}
$env:MONGO_USER = 'pz08' + [guid]::NewGuid().ToString('N').Substring(0, 8)
$env:MONGO_PASSWORD = [guid]::NewGuid().ToString('N') + [guid]::NewGuid().ToString('N')
$env:MONGO_PORT = [string](Get-FreePort)
$apiPort = Get-FreePort
while ([string]$apiPort -eq $env:MONGO_PORT) { $apiPort = Get-FreePort }
$env:MONGO_DB = 'pz8'
$env:HTTP_ADDR = "127.0.0.1:$apiPort"
$env:MONGO_URI = "mongodb://$($env:MONGO_USER):$($env:MONGO_PASSWORD)@127.0.0.1:$($env:MONGO_PORT)/?authSource=admin"
$project = 'pz08manual' + [guid]::NewGuid().ToString('N').Substring(0, 8)
docker compose --project-name $project up -d --wait
go run ./cmd/api
```

После `Ctrl+C` остановите именно этот проект и удалите временные переменные:

```powershell
docker compose --project-name $project down --volumes
Remove-Item Env:MONGO_USER,Env:MONGO_PASSWORD,Env:MONGO_PORT,Env:MONGO_URI,Env:MONGO_DB,Env:HTTP_ADDR
```

`MONGO_URI`, `MONGO_DB` и `HTTP_ADDR` обязательны при ручном запуске. `.env.example` служит только шаблоном; настоящие значения не записывайте в Git. Для штатной проверки используйте `run-local.ps1`: он выполняет уборку также при ошибке.

## API

| Метод | Маршрут | Успех |
| --- | --- | --- |
| `GET` | `/health` | `200`, `{"status":"ok"}` |
| `POST` | `/api/v1/notes` | `201`, созданная заметка |
| `GET` | `/api/v1/notes?q=&limit=&skip=` | `200`, список `items`; по умолчанию `limit=20`, `skip=0` |
| `GET` | `/api/v1/notes/{id}` | `200`, заметка |
| `PATCH` | `/api/v1/notes/{id}` | `200`, обновлённая заметка |
| `DELETE` | `/api/v1/notes/{id}` | `204`, пустое тело |

`title` обязателен при создании. PATCH принимает только переданные `title` и/или `content`; пустой или `null` заголовок запрещён. Неверный JSON, ID и параметры пагинации дают `400`; неизвестный ID — `404`; повторяющийся `title` — `409`. Поиск `q` выполняется по буквальному фрагменту заголовка без интерпретации пользовательских символов как регулярного выражения. Предел `limit`: от 1 до 100.

## Проверки Go

```powershell
Set-Location D:\MAG\GO\practic
& .\setup.ps1
Set-Location .\practices\pz08-mongo
go fmt ./...
go vet ./...
go test ./...
go build ./...
go mod verify
```
