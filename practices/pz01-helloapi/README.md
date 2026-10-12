# Hello API (PZ01)

Минимальный HTTP-сервис на стандартном `net/http`. `GET /hello` возвращает `Hello, world!`, а `GET /user` — JSON с новым UUID и именем `Gopher`.

## Требования

- Windows и Go 1.27.2 или совместимая версия.
- Порт `8080` должен быть свободен (или задайте другой через `APP_PORT`).

## Запуск из исходников

В PowerShell из корня клонированного репозитория (Go должен быть установлен и доступен через `PATH`) выполните:

```powershell
Set-Location .\practices\pz01-helloapi
go run ./cmd/server
```

В этом учебном окружении локальный SDK подключается скриптом `& 'D:\MAG\GO\practic\setup.ps1'`, запущенным из корня исходного рабочего каталога; этот машинный скрипт в репозиторий не входит. При обычном клонировании используйте установленный Go из `PATH`.

Если порт `8080` занят, перед запуском задайте свободный порт, например `$env:APP_PORT = '18080'`.

Оставив сервер запущенным, откройте второе окно PowerShell и выполните:

```powershell
curl.exe -i http://localhost:8080/hello
curl.exe -i http://localhost:8080/user
```

Остановите сервер клавишами `Ctrl+C`.

## Сборка и запуск `.exe`

Из каталога практики создайте каталог сборки и соберите сервер:

```powershell
New-Item -ItemType Directory -Force .\bin | Out-Null
go build -o .\bin\helloapi.exe .\cmd\server
.\bin\helloapi.exe
```

При занятом `8080` можно запустить сервер с `$env:APP_PORT = '18080'`. В другом окне проверьте маршруты командами `curl.exe`, заменив порт на выбранный. Для завершения нажмите `Ctrl+C`. Папка `bin` и собранный `.exe` локальные и не коммитятся.

## Примеры ответов

```text
HTTP/1.1 200 OK
Hello, world!
```

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"id":"<новый UUID> на каждый запрос","name":"Gopher"}
```

## Структура

```text
go.mod
go.sum
cmd/server/main.go
README.md
```

Переменная `APP_PORT` задаёт порт; по умолчанию сервер слушает `:8080`.
