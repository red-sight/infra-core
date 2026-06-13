# Ревью архитектуры — ветка `logto`

> Дата: 2026-06-11. Статус: findings зафиксированы, исправления не начаты.

## Общая оценка

Архитектура «самособирающейся» инфраструктуры цельная и хорошо задокументированная: labels → discovery → OpenAPI → генерация KrakenD → RBAC через Logto. Разделение ответственности правильное (EnvironmentAdapter, registrar/gateway/logto/openapi), хеш-детект изменений, reconcile-loop против stale-событий, M2M-секрет передаётся через volume, а не env, multi-stage образы со `scratch`.

Блокирующие для прода классы проблем: **fail-open авторизация на старте**, **гонки вокруг krakend.json** и **неработоспособность Swarm-деплоя как такового**.

## Критичное

### 1. Fail-open RBAC: окно без проверки скоупов

`resolveRoles` возвращает `nil`, когда Logto недоступен или креды ещё не записаны (`registrator/internal/gateway/gateway.go:304-306`), а `reload` это явно проглатывает: `registrator/main.go:155-158` — «scope→roles unavailable, KrakenD will use no role restrictions». Пока `logto-init` не отработал (или Logto лежит), эндпоинты с `x-infra-scopes` доступны **любому валидному JWT**.

**Фикс:** если у сервиса объявлены скоупы, а маппинг недоступен — либо не генерировать конфиг (оставить старый), либо ставить deny-sentinel, как уже сделано для немаппленных скоупов (`__no_role_configured__`, `gateway.go:346`).

**Статус: исправлено (Фаза 0, 2026-06-13).** `resolveRoles` теперь fail-closed: при объявленных скоупах возвращает deny-sentinel, когда маппинг недоступен (`scopeRoles == nil`) и когда пересечение в режиме `matcher=all` пусто (ранее ветка `all` тоже текла fail-open). Лог в `reload` переформулирован — больше не «no role restrictions», а «deny all until mapping available». Покрыто тестом `TestResolveRoles`.

### 2. Гонка двух `reload` и неатомарная запись конфига

`reload` вызывается из двух горутин — debounce-таймера (`main.go:82-84`) и poll-тикера (`main.go:111-119`) — без синхронизации. `krakend.json` и `openapi.json` пишутся через `os.WriteFile` напрямую (`gateway.go:103`, `openapi.go:70`), плюс generate делает read-modify-write того же файла (`gateway.go:241-256`). KrakenD при рестарте может прочитать недописанный JSON.

**Фикс:** мьютекс вокруг `reload` + атомарная запись (temp-файл + `rename`).

**Статус: исправлено (Фаза 0, 2026-06-13).** `reloadMu` сериализует `reload` между debounce-таймером и poll-тикером; запись конфига и спеки идёт через общий `fsutil.AtomicWrite` (temp в той же директории + `rename`) в `gateway` и `openapi`. Покрыто `TestAtomicWrite`.

### 3. Перезапуск KrakenD сломан в Swarm; рестарт = downtime

`main.go:172` ищет контейнер по метке `com.docker.compose.service=krakend` — в Swarm контейнеры несут `com.docker.swarm.service.name`, рестарт не найдёт цель. Рестарт (а не hot-reload) гейтвея на каждое изменение топологии — обрыв всех in-flight запросов; AGENTS.md пишет «writes and reloads», реализация делает restart. Если конфиг записан, а рестарт не удался — конфиг и работающий гейтвей расходятся без ретрая (`main.go:172-176`, только лог).

**Фикс (решено 2026-06-12):** механизм перезапуска — часть `EnvironmentAdapter`, без отдельного компонента. Compose — рестарт контейнера (как сейчас); Swarm — `ServiceUpdate` с инкрементом `ForceUpdate` (эквивалент `docker service update --force`): rolling-рестарт, с `update_config: order: start-first` одна реплика перезапускается почти без даунтайма. Авто-reload сохраняется в обоих режимах — в Swarm набор эндпоинтов меняется только при явном `stack deploy`, так что перезапуск гейтвея — ожидаемое завершение деплоя. Балансировка реплик и здоровье тасков — зона Swarm (VIP/IPVS), registrator их не отслеживает.

### 4. Swarm-стек нефункционален

В `docker-compose.swarm.yml` нет `logto-init` — в проде Logto никогда не инициализируется: ни API resource (без него JWT не имеют `aud` и KrakenD отвергает всё), ни M2M-креды для Registrator. Также отсутствуют service-core и admin, нет TLS-конфигурации для entrypoint `:443`, нет resource limits.

### 5. M2M-секрет с правами по умолчанию

`scripts/logto/init.js:345-346` пишет `registrator-m2m.json` (plaintext `clientSecret`) без `mode` — файл получает 0644.

**Фикс:** `{ mode: 0o600 }` и `mkdirSync(..., { mode: 0o700 })`; для Swarm — Docker secrets.

### 6. `/admin/organizations` доступен любому аутентифицированному пользователю

service-core не объявляет `x-infra-scopes` (только `x-infra-protected`), хотя скоуп `read:organizations` существует в `logto.config.yaml` и назначен роли admin. `list()` (`service-core/internal/organization/organization.go:79-105`) не фильтрует по `x-organization-id` — любой залогиненный пользователь видит все организации. Нарушение собственного контракта (service-a его соблюдает через `@InfraAuth`) и утечка данных.

## Важное

- **Бэкенды слепо доверяют `x-user-id` / `x-user-roles` / `x-organization-id`**, все сервисы в одной плоской сети `infra`. KrakenD корректно отрезает клиентские заголовки (`input_headers: ["Authorization"]`), но любой контейнер в сети может обратиться к бэкенду напрямую, минуя гейтвей. Зафиксировать trust boundary в AGENTS.md; дальше — сегментация сетей (бэкенды в отдельной сети, доступной только KrakenD) либо подписанный заголовок/секрет от гейтвея.
- **Нет таймаутов на исходящих HTTP**: `http.Get` для спек сервисов (`gateway.go:110`, `openapi/openapi.go:157`) и Logto-клиент. Один зависший сервис блокирует весь reload-цикл; токен Logto фетчится под мьютексом (`logto/logto.go:110-147`).
- **N+1 к Logto**: roles → по одному запросу скоупов на роль, последовательно (`logto.go:149-187`). При росте числа ролей станет узким местом каждого reload.
- **Дубликаты путей между сервисами не детектируются** (`gateway.go:58`) — второй сервис с тем же `path+method` молча теряется.
- **Секреты в env**: дефолтные `postgres`/`changeme` в `.env`, креды Postgres в env у всех сервисов. Для Compose-дева нормально, для Swarm — Docker secrets и обязательность сильных паролей без дефолта.
- **Невалидный `infra.port` молча падает в 3000** (`registrar/compose.go`, serviceFromLabels) — оператор не узнает об опечатке в метке.

## Зафиксированное / минорное

Уже известные и корректно описанные в AGENTS.md: OpenAPI 3.1/3.0 nullable-конфликт, `depends_on`-гэп `logto-init`, KrakenD zero-downtime отложен.

Из мелкого:

- Токены admin SPA в localStorage (дефолт Logto SDK — стандартный для SPA компромисс с XSS-риском); `VITE_API_BASE_URL` нигде не используется — интеграция SPA с API не дописана.
- Logto admin-консоль в dev на `0.0.0.0:3002` — лучше `127.0.0.1:3002:3002`.
- Debounce-таймер не гасится при shutdown.

## Приоритет исправлений

1. Fail-closed для скоупов при недоступном Logto (п.1) — правка `resolveRoles` + `Generate`.
2. Мьютекс на `reload` + атомарная запись через rename (п.2) — небольшая правка, убирает целый класс отказов.
3. `x-infra-scopes: [read:organizations]` на service-core + тенант-фильтрация (п.6).
4. Swarm: добавить `logto-init` (одноразовый запуск), TLS, secrets, resource limits; починить рестарт KrakenD по Swarm-метке (п.3, 4).
5. `mode: 0o600` на M2M-файл (п.5) — однострочник.
6. Таймауты на все исходящие HTTP-клиенты.

Пункты 1, 2, 5 — маленькие диффы с большим эффектом.
