# PZ07 — Redis SET/GET/TTL

Учебное REST API на `net/http` с отдельным клиентом `go-redis/v9`. `GET /set` сохраняет строку на 10 секунд; `GET /get` читает ключ; `GET /ttl` возвращает оставшееся время жизни. Отсутствующий ключ даёт HTTP 404, ошибки Redis — HTTP 503.

## Запуск в Windows

Требуются Go и Docker Desktop с запущенным Docker Engine. Одна команда запускается из любого каталога PowerShell, собирает API, создаёт принадлежащий только этому запуску временный контейнер `redis:7-alpine`, выбирает свободные loopback-порты, проверяет три маршрута и истечение ключа, затем завершает свой API и удаляет только собственный контейнер:

```powershell
& 'D:\MAG\GO\practic\practices\pz07-redis\run-local.ps1'
```

Пароли и `.env` не нужны. Скрипт использует уникальные имя контейнера и ключ. При ошибке очистка выполняется в `finally`; чужие контейнеры и процессы не останавливаются. Docker image должен быть доступен локально либо Docker должен иметь возможность загрузить его.

Для ручного запуска при уже работающем Redis:

```powershell
Set-Location D:\MAG\GO\practic
& .\setup.ps1
Set-Location .\practices\pz07-redis
$env:REDIS_ADDR = '127.0.0.1:6379'
$env:PORT = '8080'
go run .\cmd\server
```

В другом окне PowerShell отправьте запросы:

```powershell
$base = 'http://127.0.0.1:8080'
Invoke-RestMethod "$base/set?key=demo&value=hello"
Invoke-RestMethod "$base/get?key=demo"
Invoke-RestMethod "$base/ttl?key=demo"
```

## Проверки

```powershell
Set-Location D:\MAG\GO\practic
& .\setup.ps1
Set-Location .\practices\pz07-redis
go fmt ./...
go vet ./...
go test ./...
go build ./...
```

Тесты используют `miniredis` и проверяют запись, чтение, TTL и истечение; для проверки реального сервиса используйте `run-local.ps1`.
