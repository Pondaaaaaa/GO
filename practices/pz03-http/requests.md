# Проверка HTTP API в PowerShell

Запустите сервер в первом окне из каталога `practices/pz03-http`:

```powershell
$env:PORT = '18083'
.\dev.ps1 run
```

Во втором окне задайте тот же порт, подготовьте базовый URL и выполните запросы по порядку. Примеры используют `curl.exe`, а `--noproxy '*'` направляет локальные запросы прямо к серверу.

## Запросы

1. Проверка здоровья — ожидается `200 OK`:

   ```powershell
   $env:PORT = '18083'
   $base = "http://127.0.0.1:$env:PORT"
   curl.exe --noproxy '*' -i "$base/health"
   ```

2. Создание задачи — ожидается `201 Created`. Команда извлекает ID из созданного JSON, чтобы использовать его ниже:

   ```powershell
   $createResponse = curl.exe --noproxy '*' --silent --show-error --write-out "`nHTTP_STATUS=%{http_code}" -X POST "$base/tasks" -H 'Content-Type: application/json' --data-raw '{"title":"  Buy Milk  "}'
   $createLines = $createResponse -split "`n"
   $task = $createLines[0] | ConvertFrom-Json
   $taskID = $task.id
   $createResponse
   ```

3. Список задач — ожидается `200 OK` и массив с созданной задачей:

   ```powershell
   curl.exe --noproxy '*' -i "$base/tasks"
   ```

4. Фильтр `q=mIlK` без учёта регистра — ожидается `200 OK` и задача `Buy Milk`:

   ```powershell
   curl.exe --noproxy '*' -i "$base/tasks?q=mIlK"
   ```

5. Получение созданной задачи по фактическому ID — ожидается `200 OK`:

   ```powershell
   curl.exe --noproxy '*' -i "$base/tasks/$taskID"
   ```

6. Неподдерживаемый тип тела — ожидается `400 Bad Request`:

   ```powershell
   curl.exe --noproxy '*' -i -X POST "$base/tasks" -H 'Content-Type: text/plain' --data-raw 'not JSON'
   ```

7. Нечисловой ID — ожидается `400 Bad Request`:

   ```powershell
   curl.exe --noproxy '*' -i "$base/tasks/not-a-number"
   ```

8. Числовой ID, отсутствующий в свежем in-memory хранилище — ожидается `404 Not Found`:

   ```powershell
   curl.exe --noproxy '*' -i "$base/tasks/999999"
   ```

Тесты `go test ./...` также проверяют некорректный JSON и пустой `title` — оба случая должны возвращать `400 Bad Request`.
