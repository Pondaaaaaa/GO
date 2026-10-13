# PZ05: PostgreSQL через database/sql

Отдельный учебный модуль с драйвером `pgx/v5/stdlib`, параметризованными запросами, репозиторием задач и транзакционной массовой вставкой.

## Быстрый запуск для Windows PowerShell

Для обычного запуска достаточно выполнить из корня репозитория:

```powershell
Set-Location D:\MAG\GO\practic
& .\practices\pz05-db\run-local.ps1
```

Скрипт сам загружает локальное окружение через `setup.ps1`, использует пользователя из `PGUSER` или `postgres` по умолчанию и спрашивает только пароль (ввод скрыт). Если роль в pgAdmin называется не `postgres`, задайте её перед запуском: `$env:PGUSER = 'имя_роли'`. Скрипт задаёт `PGPASSWORD` только на время процесса, создаёт `todo` лишь если базы ещё нет, идемпотентно создаёт таблицу `tasks`, временно формирует `DATABASE_URL`, запускает `go run .`, затем выполняет `SELECT * FROM tasks ORDER BY id` через `psql` и удаляет временные переменные окружения. Пароль и URL не записываются в файлы и не печатаются. Значения подключения по умолчанию: `127.0.0.1:5432`, база `todo`.

Если база `todo` уже создана, скрипт подключится к ней и продолжит. `CREATE TABLE IF NOT EXISTS` не меняет существующую схему и не удаляет данные; перед практическим запуском проверьте, что используете свою учебную базу.

## Подготовка базы вручную через psql (необязательно)

Клиент из локальной установки находится в `D:\programDev\bin\psql.exe`. Подключитесь к локальному серверу с запросом пароля (пароль вводится в интерактивном запросе и не попадает в команду):

```powershell
& 'D:\programDev\bin\psql.exe' -W -h 127.0.0.1 -p 5432 -U postgres -d postgres
```

В `psql` проверьте, существует ли `todo`:

```sql
SELECT datname FROM pg_database WHERE datname = 'todo';
```

Если результата нет, создайте БД, затем подключитесь к ней:

```sql
CREATE DATABASE todo;
\c todo
```

Если БД уже существует, сначала подключитесь `\c todo` и выполните `SELECT to_regclass('public.tasks');`. Если результат `tasks`, проверьте `SELECT count(*) FROM tasks;`; если в таблице есть чужие или неизвестные данные, остановитесь и согласуйте изоляцию до вставки. Если таблицы нет либо она пуста, проверьте её схему через `\d tasks` (когда таблица есть), затем используйте идемпотентное создание только для отсутствующей таблицы:

```sql
CREATE TABLE IF NOT EXISTS tasks (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    done BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
SELECT * FROM tasks ORDER BY id;
```

## Запуск на Windows PowerShell

Выполните из корня репозитория:

```powershell
Set-Location D:\MAG\GO\practic
. .\setup.ps1
Set-Location .\practices\pz05-db
go mod download
go fmt ./...
go vet ./...
go test ./...
go build ./...
```

Укажите локальный URL подключения только в текущей сессии. Команда ниже принимает значение скрыто, не выводите `$env:DATABASE_URL` и не сохраняйте его в файлы репозитория:

```powershell
$secureUrl = Read-Host 'Введите локальный DATABASE_URL' -AsSecureString
$ptr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($secureUrl)
try { $env:DATABASE_URL = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($ptr) }
finally { [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($ptr) }
go run .
Remove-Item Env:DATABASE_URL
```

Приложение использует значения `PZ05` с временной меткой, чтобы его собственные строки отличались от предыдущих запусков. Оно печатает ID, список задач, результаты `ListDone(false/true)`, `FindByID`, результат проверки отсутствующего ID, сводку `db.Stats()` и проверку отката транзакции. Если в `todo.tasks` уже есть неизвестные строки, до запуска сначала согласуйте изоляцию.

## Реализация

- `db.go`: `sql.Open("pgx", dsn)`, `PingContext` с таймаутом, лимиты пула и закрытие при ошибке подключения.
- `repository.go`: `CreateTask`, `ListTasks`, `ListDone`, `FindByID`, `CreateMany`; все значения передаются параметрами `$1`, а массовая вставка выполняется через `BeginTx`/`Commit`/`Rollback`.
- `main.go`: демонстрационный запуск и вывод статистики пула; значение DSN не печатается.

Все SQL-запросы приложения используют `context.WithTimeout`. Таблицу и БД предварительно создают отдельно командами `psql` из раздела выше.
