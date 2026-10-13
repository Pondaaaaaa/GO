# Практика №11 — Notes REST API

Отдельный Go-модуль с chi. Данные хранятся в памяти и исчезают при остановке. Архитектура: HTTP handler → service → repository. Контракт: [OpenAPI](api/openapi.yaml).

## Быстрая проверка в Windows PowerShell

Из любого каталога:

```powershell
& 'D:\MAG\GO\practic\practices\pz11-notes\run-local.ps1'
```

Скрипт подключает локальный Go, собирает приложение, выбирает свободный порт, выполняет пять операций CRUD и запросы ошибок через `curl.exe`, затем завершает только свой сервер и удаляет временные файлы.

## Ручной запуск

В первом PowerShell:

```powershell
Set-Location 'D:\MAG\GO\practic'
& .\setup.ps1
Set-Location .\practices\pz11-notes
$env:PORT = '18011' # измените, если занят
go run ./cmd/api
```

Во втором PowerShell:

```powershell
$base = 'http://127.0.0.1:18011/api/v1/notes'
curl.exe --noproxy '*' -i -X POST $base -H 'Content-Type: application/json' --data-binary '@requests/create.json'
curl.exe --noproxy '*' -i $base
curl.exe --noproxy '*' -i "$base/1"
curl.exe --noproxy '*' -i -X PATCH "$base/1" -H 'Content-Type: application/json' --data-binary '@requests/patch.json'
curl.exe --noproxy '*' -i -X DELETE "$base/1"
curl.exe --noproxy '*' -i "$base/1"
```

`PORT` по умолчанию 8080. Сервер слушает только `127.0.0.1`. Создание требует непустого `title`; PATCH принимает хотя бы одно из `title` или `content`. Пустой `content` допустим. ID — положительное целое. Неверный запрос возвращает JSON и 400, отсутствующая заметка — JSON и 404, DELETE — 204 без тела.

Проверки в каталоге модуля после `setup.ps1`:

```powershell
go fmt ./...
go vet ./...
go test ./...
go build ./...
go mod verify
```

Реальные результаты приведены в [отчёте](../../reports/pz11.md).
