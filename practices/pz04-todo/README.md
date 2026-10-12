# PZ04: ToDo API на chi

Минимальный CRUD API для задач в памяти. Модуль использует `chi/v5`, потокобезопасное хранилище, JSON-ошибки, RequestID/Recoverer, логирование запросов и CORS.

## Windows PowerShell

Откройте PowerShell в этой папке и подготовьте локальный Go SDK из корня репозитория:

```powershell
Set-Location D:\MAG\GO\practic
. .\setup.ps1
Set-Location .\practices\pz04-todo
go mod download
go fmt ./...
go vet ./...
go test ./...
go build ./...
```

Если порт `8080` занят, задайте свободный `PORT` (например, `18080`) и запустите сервер:

```powershell
$env:PORT = '18080'
go run .
```

В другом окне PowerShell выполните запросы. В примерах ниже `$base` задаёт адрес, а `$id` читается из настоящего ответа создания:

```powershell
$base = 'http://localhost:18080'
curl.exe -i "$base/health"
$created = curl.exe -sS -X POST "$base/api/tasks" -H 'Content-Type: application/json' -d '{"title":"Read Go"}'
$created
$id = ($created | ConvertFrom-Json).id
curl.exe -i "$base/api/tasks"
curl.exe -i "$base/api/tasks/$id"
curl.exe -i -X PUT "$base/api/tasks/$id" -H 'Content-Type: application/json' -d '{"title":"Read chi","done":true}'
curl.exe -i -X DELETE "$base/api/tasks/$id"
curl.exe -i "$base/api/tasks/$id"
curl.exe -i -X OPTIONS "$base/api/tasks" -H 'Origin: http://localhost:3000' -H 'Access-Control-Request-Method: POST' -H 'Access-Control-Request-Headers: Content-Type'
```

`DELETE` существующей задачи отвечает `204 No Content` без тела; повторное получение удалённого ID отвечает `404`. Ошибочные ID или JSON возвращают `400` с JSON-объектом `error`.

## Маршруты

| Метод | Путь | Успешный ответ |
| --- | --- | --- |
| GET | `/health` | `200 OK`, текст `OK` |
| GET | `/api/tasks` | `200 OK`, JSON-массив |
| POST | `/api/tasks` | `201 Created`, созданная задача |
| GET | `/api/tasks/{id}` | `200 OK`, задача |
| PUT | `/api/tasks/{id}` | `200 OK`, обновлённая задача |
| DELETE | `/api/tasks/{id}` | `204 No Content`, пустое тело |

`POST` и `PUT` принимают JSON вида `{"title":"Read Go","done":false}`. Данные живут только в памяти и сбрасываются при остановке процесса.
