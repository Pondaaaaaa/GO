# Практика №15 — unit-тесты Go

Самостоятельный модуль: mathx.Sum/Divide, stringsx.Clip и Service.FindIDByEmail. Тесты используют testing, testify v1.7.0 (assert/require) и локальный stub UserRepo. PostgreSQL, Docker, сервер и пароль не нужны.

## Проверка в PowerShell

```powershell
Set-Location 'D:\MAG\GO\practic'
& .\setup.ps1
Set-Location .\practices\pz15-tests
go mod download
go test ./...
go test -v ./internal/...
go test -cover ./...
go test '-coverprofile=coverage.out' ./...
go tool cover '-html=coverage.out' '-o=../../reports/logs/pz15/coverage.html'
go tool cover '-func=coverage.out'
go fmt ./...
go vet ./...
go build ./...
go mod verify
Start-Process (Resolve-Path '..\..\reports\logs\pz15\coverage.html').Path
Remove-Item -LiteralPath coverage.out
```

Первый go mod download требует сети, если зависимостей ещё нет в локальном кэше. Флаги cover с именами файлов заключены в кавычки для воспроизводимости в PowerShell. coverage.out — временный файл, исключён корневым .gitignore; в Git сохраняется только HTML и небольшие подлинные текстовые выводы в [reports/logs/pz15](../../reports/logs/pz15/).

## Контракт и ограничения

- Sum складывает int; Divide выполняет целочисленное деление с усечением к нулю и возвращает 0/ошибку `divide by zero` при b=0. Переполнение int не обрабатывается отдельно.
- Clip считает **байты**, отрицательный max заменяет на 0. Может разрезать Unicode и вернуть невалидный UTF-8: Clip("П",1) — один байт 0xD0. Для пользовательского Unicode-текста нужна другая функция с учётом границ символов.
- FindIDByEmail передаёт email репозиторию без нормализации, возвращает ID при успехе и 0/исходную ошибку при неуспехе. Stub расположен только в тестовом файле, проверяет точный аргумент и один вызов; БД не используется.

Фактическое покрытие mathx, stringsx, service — 100,0% statements в каждом пакете. Это свидетельство выполнения кода выбранными тестами, а не доказательство отсутствия всех возможных ошибок. [Отчёт](../../reports/pz15.md), [настоящий HTML покрытия](../../reports/logs/pz15/coverage.html). Необязательные benchmark и тест паники не добавлялись.
