# Практика №10 — JWT, middleware и права доступа

Отдельный Go-модуль. Пользователи хранятся в памяти с bcrypt-хэшами: `admin@example.com` и `user@example.com`, учебный пароль обоих — `secret123`. База данных и Docker не нужны. Данные и список отозванных refresh-токенов сбрасываются при перезапуске.

## Одна команда для запуска и проверки

Из любого каталога PowerShell:

```powershell
& 'D:\MAG\GO\practic\practices\pz10-auth\run-local.ps1'
```

Скрипт подключает локальный Go, выбирает свободный порт, генерирует временный секрет JWT, запускает сервер, проверяет login, защищённые маршруты, RBAC, ABAC, refresh и повтор старого refresh. Значения токенов и секрет не выводятся. По окончании сервер останавливается, переменные окружения восстанавливаются.

## Ручной запуск

```powershell
Set-Location 'D:\MAG\GO\practic'
& .\setup.ps1
Set-Location .\practices\pz10-auth
$env:JWT_SECRET = 'replace-with-a-random-secret-of-at-least-32-bytes'
$env:JWT_TTL = '15m'
$env:JWT_REFRESH_TTL = '168h'
$env:APP_PORT = '18010'
go run ./cmd/server
```

`JWT_SECRET` обязателен и должен содержать не менее 32 байт. `APP_PORT` по умолчанию 8080, `JWT_TTL` — 15m, `JWT_REFRESH_TTL` — 168h; `APP_HOST` по умолчанию 127.0.0.1. Для реального сервиса нужны HTTPS, безопасное хранение ключа и постоянное хранилище отзыва токенов.

## Маршруты

| Маршрут | Доступ | Результат |
| --- | --- | --- |
| `POST /api/v1/login` | публичный | `access_token`, `refresh_token`, `token_type`, `expires_in` |
| `POST /api/v1/refresh` | refresh-токен в JSON | новая пара; старый refresh отозван |
| `GET /api/v1/me` | Bearer access, admin/user | `id`, `email`, `role` |
| `GET /api/v1/admin/stats` | Bearer access, admin | статистика |
| `GET /api/v1/users/{id}` | Bearer access | admin — любой профиль, user — только свой |

Пример получения токена вручную (не помещайте вывод с токеном в публичный отчёт):

```powershell
$login = curl.exe --noproxy '*' -sS -X POST http://127.0.0.1:18010/api/v1/login -H 'Content-Type: application/json' -d '{"email":"admin@example.com","password":"secret123"}' | ConvertFrom-Json
$token = $login.access_token
curl.exe --noproxy '*' -i http://127.0.0.1:18010/api/v1/me -H "Authorization: Bearer $token"
```

Повторяемый сценарий и точные статусы задокументированы в [отчёте](../../reports/pz10.md) и [логе](../../reports/logs/pz10-http.txt).
