# Практика №13 — профилирование Fib

Самостоятельный стенд без внешних Go-зависимостей. Сервер слушает только 127.0.0.1. Собственный ServeMux явно подключает pprof.Index, Profile, Cmdline, Symbol и Trace; Index обслуживает heap/goroutine и другие именованные профили. Исходные практики №11 и №12 не изменяются.

## Основная команда в Windows PowerShell

Из любого каталога:

```powershell
& 'D:\MAG\GO\practic\practices\pz13-pprof\run-local.ps1'
```

Скрипт подключает setup.ps1, создаёт собственную временную папку, устанавливает GOMAXPROCS=1, собирает две версии с обычными символами Go (без -s/-w). Сначала измеряет slow, затем собирает fast и повторяет сценарий. Сервер сам выбирает свободный loopback-порт через listen на порту 0; адрес берётся из лога. Не используются чужие процессы или порты.

В каждой версии: один проверочный /work, CPU-профиль 12 секунд, затем 24 последовательных запроса /work с n=38 и интервалом 500 мс (2 запроса/с, concurrency=1). Проверяется результат 39088169. Нагрузчик сообщает фактическое время, среднее HTTP-время, максимум и число опоздавших запусков. Затем отдельный /heap-work выделяет и удерживает 4 МиБ; снимается /debug/pprof/heap?gc=1. Эта аллокация не является частью Fib и происходит после CPU-профиля. Анализируются CPU top/list и Heap inuse_space/alloc_space, строятся реальные SVG при наличии dot. Ручные таймеры измеряют вызов Fib, исключая запись HTTP-ответа, и содержат собственные накладные расходы таймера.

После двух сценариев запускаются обе функции через `go test -bench=. -benchmem -count=3 ./...` на одинаковом n=30. Вход читается из переменной, результат сохраняется в глобальный Sink и проверяется; вычисление остаётся наблюдаемым. Бенчмарки функций не измеряют HTTP, профилировщик или логирование.

Все свои фоновые процессы завершаются в finally, GOMAXPROCS и PATH восстанавливаются, временные exe и бинарные профили удаляются. Для сохранения небольших выводов и SVG используйте отдельную папку:

```powershell
& 'D:\MAG\GO\practic\practices\pz13-pprof\run-local.ps1' -OutputDirectory 'D:\MAG\GO\.local\pz13-results'
```

## Живой pprof UI и настоящий снимок

```powershell
& 'D:\MAG\GO\practic\practices\pz13-pprof\run-local.ps1' -OpenUI
```

После измерений скрипт запускает свой pprof UI на свободном loopback-порту и печатает URL. Откройте его в браузере, выберите View → Graph (или `/ui/graph`), снимите настоящий граф до оптимизации. По Enter в терминале UI останавливается, профили удаляются. Не закрывайте терминал принудительно. Для чтения Fib в UI можно выбрать View → Source; для консольного list используется сохранённый текстовый вывод.

Graphviz dot необходим для графа, но не для top/list/Heap/benchmarks. Скрипт ищет dot в PATH и локальном `D:\MAG\GO\.local\graphviz-16.1.0`. При отсутствии честно сообщает, что граф не построен. Установите официальный Graphviz из [страницы загрузки](https://graphviz.org/download/) либо распакуйте ZIP в этот локальный каталог и повторите. SDK/Graphviz и бинарные профили не входят в Git. В этом окружении использована официальная переносная версия 16.1.0 с проверкой опубликованного SHA256.

## Ручной запуск

```powershell
Set-Location 'D:\MAG\GO\practic'
& .\setup.ps1
Set-Location .\practices\pz13-pprof
go run ./cmd/api -port 0 -n 38
```

Обычный запуск использует FibFast; напечатанный порт подставьте вместо `<port>`. `/work` — вычисление, `/debug/pprof/` — настоящий индекс профилей, `/heap-work` — отдельный учебный буфер. n ограничен 0..40, порт 0..65535. Для медленного бинарника:

```powershell
go build -ldflags '-X main.defaultMode=slow' -o "$env:TEMP\pz13-slow.exe" ./cmd/api
& "$env:TEMP\pz13-slow.exe" -port 0 -n 38
```

Команды получения и анализа с реальным адресом (нагрузку запускать параллельно CPU-запросу):

```powershell
curl.exe --noproxy '*' -f -o cpu.pprof 'http://127.0.0.1:<port>/debug/pprof/profile?seconds=12'
go tool pprof -top "$env:TEMP\pz13-slow.exe" cpu.pprof
go tool pprof -list=Fib "$env:TEMP\pz13-slow.exe" cpu.pprof
go tool pprof -svg -output cpu.svg "$env:TEMP\pz13-slow.exe" cpu.pprof
curl.exe --noproxy '*' -f -o heap.pprof 'http://127.0.0.1:<port>/debug/pprof/heap?gc=1'
go tool pprof -sample_index=inuse_space -top "$env:TEMP\pz13-slow.exe" heap.pprof
go tool pprof -sample_index=alloc_space -top "$env:TEMP\pz13-slow.exe" heap.pprof
```

Ручные временные файлы создавайте вне репозитория и удаляйте после анализа. Основной скрипт делает это автоматически.

## Проверки

После setup.ps1 из корня модуля:

```powershell
go fmt ./...
go vet ./...
go test ./...
go build ./...
go test -bench=. -benchmem ./...
```

Сравнивайте CPU при одинаковых n, расписании нагрузки и длительности профиля. Сэмплирование может не заметить очень быструю функцию; отсутствие её в top не означает нулевое время. Heap — выборочные оценки, inuse_space показывает живую память, alloc_space — накопленные выделения; оба Fib в данном алгоритме не требуют heap-аллокаций. [Отчёт](../../reports/pz13.md) содержит фактические результаты.
