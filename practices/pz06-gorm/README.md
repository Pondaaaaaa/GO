# PZ06: GORM, relations, and REST API

This is an independent Go module using GORM with PostgreSQL and a chi HTTP router. It migrates `users`, `notes`, `tags`, and `note_tags`, demonstrates User-to-Notes 1:N and Note-to-Tags M:N relations, and exposes JSON endpoints.

## One-command local run on Windows

From any PowerShell working directory:

```powershell
& 'D:\MAG\GO\practic\practices\pz06-gorm\run-local.ps1'
```

The script loads the repository-local Go setup, finds `psql.exe`, uses `PGUSER` or `postgres`, and prompts only for the PostgreSQL password with hidden input. It creates the separate `pz6_gorm` database only when absent, refuses unrelated public tables, builds and starts the API on a free loopback port, checks the real HTTP responses, prints the GORM schema using `psql \d`, then stops only the server process it started and restores temporary environment variables. Password and DSN are not printed or written to the repository.

If your PostgreSQL role is not `postgres`, set it before running:

```powershell
$env:PGUSER = 'your_role'
& 'D:\MAG\GO\practic\practices\pz06-gorm\run-local.ps1'
Remove-Item Env:PGUSER
```

PostgreSQL must be accepting TCP connections at `127.0.0.1:5432`; the role needs permission to create `pz6_gorm` if it does not exist and to create tables in that database. The script does not modify PZ05's `todo` database. Practice records use unique per-run email and tag names.

## API

| Method | Path | Result |
| --- | --- | --- |
| `GET` | `/health` | `200`, `{"status":"ok"}` |
| `POST` | `/users` | `201`, created user; duplicate email returns `409` |
| `GET` | `/users/{id}` | `200`, user with related notes (demonstrates 1:N) |
| `POST` | `/notes` | `201`, note linked to an existing user and named tags |
| `GET` | `/notes/{id}` | `200`, note with preloaded author and tags |

Invalid JSON/fields return `400`; missing users/notes return `404`; unexpected database failures return a generic `500` JSON response. The error response does not expose raw database details.

Example requests (use actual IDs from the responses):

```powershell
$base = 'http://127.0.0.1:8080'
$user = Invoke-RestMethod -Method Post -Uri "$base/users" -ContentType 'application/json' -Body (@{name='Example'; email='example@example.test'} | ConvertTo-Json)
$note = Invoke-RestMethod -Method Post -Uri "$base/notes" -ContentType 'application/json' -Body (@{title='First note'; content='GORM relations'; userId=$user.id; tags=@('go','gorm')} | ConvertTo-Json)
Invoke-RestMethod -Method Get -Uri "$base/notes/$($note.id)"
```

The standalone sample assumes the server has been started separately and that `DB_DSN` is configured only in that process environment. For the full reproducible database scenario, use `run-local.ps1` above.

## Local checks

```powershell
Set-Location D:\MAG\GO\practic
& .\setup.ps1
Set-Location .\practices\pz06-gorm
go fmt ./...
go vet ./...
go test ./...
go build ./...
```
