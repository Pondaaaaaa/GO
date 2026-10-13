# Практика №12 — документация Notes API

Самостоятельный модуль на основе №11: chi, память, handler → service → repository. Все пять CRUD-операций документированы аннотациями Go. `swag v1.16.6` генерирует **Swagger 2.0** в `docs/`; `http-swagger/v2 v2.0.2` обслуживает UI и спецификацию.

## Одна команда для запуска и проверки

Из любого каталога Windows PowerShell 5.1 или PowerShell 7:

```powershell
& 'D:\MAG\GO\practic\practices\pz12-notes\run-local.ps1'
```

Скрипт подключает локальный Go через setup.ps1, выбирает свободный loopback-порт, собирает и запускает свой .exe. Через curl.exe проверяет пять CRUD-операций, ошибки 400/404, JSON-заголовки, пустое тело 204, UI и пять операций спецификации. В finally завершает свой процесс, удаляет свою временную папку и восстанавливает PORT. Docker, PostgreSQL и пароли не нужны.

Чтобы оставить сервер работающим для настоящего снимка:

```powershell
& 'D:\MAG\GO\practic\practices\pz12-notes\run-local.ps1' -KeepOpen
```

После проверок откройте напечатанный URL http://127.0.0.1:<порт>/docs/index.html. Снимите окно браузера с названием API и пятью операциями. В терминале нажмите Enter: скрипт остановит сервер и очистит файлы. Не завершайте терминал принудительно.

## Повторная генерация

```powershell
& 'D:\MAG\GO\practic\practices\pz12-notes\generate-docs.ps1'
```

При первом запуске скрипт устанавливает github.com/swaggo/swag/cmd/swag@v1.16.6 в отдельный каталог локального GOPATH; нужен доступ к Go proxy. Версию существующего исполняемого файла проверяет через go version -m, поэтому не зависит от swag в PATH. У модуля v1.16.6 команда swag --version сообщает v1.16.4; версия установки подтверждается метаданными Go. Повторный запуск использует проверенный локальный бинарник.

Команда из корня модуля (swag — бинарник, установленный скриптом с указанной версией):

```powershell
swag init --dir ./cmd/api,./internal/core,./internal/http --generalInfo main.go --parseInternal --output ./docs
```

Файлы docs/docs.go, docs/swagger.json, docs/swagger.yaml генерируются автоматически и хранятся в Git. После изменения аннотаций выполните генерацию, проверки и просмотрите diff. run-local.ps1 использует уже сгенерированные файлы.

## Маршруты и формат

| Метод и путь | Успех | Ошибки |
| --- | --- | --- |
| POST /api/v1/notes | 201 Note, заголовок Location | 400, 500 |
| GET /api/v1/notes | 200 массив Note, включая [] | 500 |
| GET /api/v1/notes/{id} | 200 Note | 400, 404, 500 |
| PATCH /api/v1/notes/{id} | 200 Note с updatedAt | 400, 404, 500 |
| DELETE /api/v1/notes/{id} | 204 без тела | 400, 404, 500 |

Тела успеха и ошибок имеют application/json; charset=utf-8, кроме пустого 204. Ошибка: {"error":"invalid request"}, {"error":"note not found"} или {"error":"internal server error"}. ID — положительное целое. POST требует непустой после обрезки пробелов title. PATCH принимает хотя бы одно ненулевое поле title или content; пустой content допустим. Неизвестные поля, лишний JSON, неверный Content-Type и тело больше 1 МиБ получают 400. Эти смысловые ограничения описаны в аннотациях; Swagger 2.0 не выражает все правила PATCH машинной схемой.

У списка нет page, limit, q и X-Total-Count. Данные сбрасываются при перезапуске. Аутентификация не реализована, UI доступна на локальном интерфейсе.

## Ручной запуск и curl в Windows

В первом терминале:

```powershell
Set-Location 'D:\MAG\GO\practic'
& .\setup.ps1
Set-Location .\practices\pz12-notes
$env:PORT = '18012' # выберите другой, если занят
go run ./cmd/api
```

Во втором терминале:

```powershell
Set-Location 'D:\MAG\GO\practic\practices\pz12-notes'
$base = 'http://127.0.0.1:18012/api/v1/notes'
curl.exe --noproxy '*' -i -X POST $base -H 'Content-Type: application/json' --data-binary '@requests/create.json'
curl.exe --noproxy '*' -i $base
curl.exe --noproxy '*' -i "$base/1"
curl.exe --noproxy '*' -i -X PATCH "$base/1" -H 'Content-Type: application/json' --data-binary '@requests/patch.json'
curl.exe --noproxy '*' -i -X DELETE "$base/1"
```

Файлы запросов исключают проблемы экранирования JSON в Windows PowerShell. UI — http://127.0.0.1:18012/docs/index.html, спецификация — /docs/doc.json. В спецификации нет фиксированного host: UI использует текущий адрес и /api/v1.

После setup.ps1 из каталога модуля:

```powershell
go fmt ./...
go vet ./...
go test ./...
go build ./...
go mod verify
```

## Swagger 2.0 и OpenAPI 3.0.3

Файл [api/openapi.yaml практики №11](../pz11-notes/api/openapi.yaml) — отдельная вручную подготовленная спецификация OpenAPI 3.0.3. Она остаётся в №11 и не генерируется этим скриптом. Для №12 источником служат аннотации; результат — Swagger 2.0 с definitions, basePath и body parameters, а не OpenAPI 3.0 с components.schemas и requestBody. Автоматической синхронизации двух форматов нет. Исходный модуль №11 не изменён.

Сведения о синтаксисе: [swag](https://github.com/swaggo/swag), [http-swagger](https://github.com/swaggo/http-swagger). [Отчёт](../../reports/pz12.md), [подлинный HTTP-лог](../../reports/logs/pz12-http.txt).
