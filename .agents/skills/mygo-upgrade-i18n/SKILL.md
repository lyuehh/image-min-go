---
name: mygo-upgrade-i18n
description: Upgrade github.com/egoist/mygo across 0.x minor versions and migrate hand-rolled string tables to MyGo's native i18n (ICU catalogs + mygo generate). Encodes the testing and threading pitfalls learned from the v0.4.8 → v0.5.1 upgrade of this app, chiefly: never call main-thread-bound APIs like App.Locale()/SetLocale from a go test, or it blocks until the default 10-minute timeout.
---

# MyGo upgrade & native i18n

Workflow and pitfalls for bumping MyGo and moving UI strings onto its native
i18n. Distilled from the HOME-10 retrospective, where one avoidable mistake —
a probe test calling a main-thread API without a running app — consumed ~55%
of the total time by blocking to the default `go test` 10-minute timeout.

## The one rule that saves the most time

**Never call a main-thread-bound `mygo.App` API from a `go test`.** In 0.5
`App.Locale()`, `App.SetLocale()`, `App.Languages()` and friends route through
`onMainValue`/`postMain` and *block waiting for the main run loop*. A plain
`go test` has no running app, so the call hangs until Go's default 10-minute
test timeout fires. Symptoms: a test that "passes" only after 10 minutes, or a
timeout panic whose stack shows `onMain` / `shownLocale` / `(*Application).Locale`.

To exercise locale-dependent UI in tests, drive it through the headless tester
instead, which resolves synchronously and in milliseconds:

```go
tt := ui.NewTester(a.view, 780, 640)
tt.SetLocale("en")              // what mygo.App.SetLocale would show
// view code reads c.Locale(); catalog funcs loc.In(c.Locale()).X() follow it
```

Guard rails that would have caught this instantly:
- Run every throwaway probe with an explicit short timeout: `go test -timeout 30s`.
- Before touching a `mygo.App` method in a test, check whether its source does
  `onMainValue`/`postMain` (`go list -m -f '{{.Dir}}' github.com/egoist/mygo`,
  then read `locale.go`). If it does, it needs the run loop — use the Tester.

## Upgrading the dependency

1. List available versions from the module proxy — do not wait on web fetches
   that may be blocked: `curl -s https://proxy.golang.org/github.com/egoist/mygo/@v/list`.
   One failed `web_fetch`/`web_search` is enough signal to switch to the proxy.
2. `go get github.com/egoist/mygo@vX.Y.Z` then `go mod tidy`.
3. Read breaking-change notes before editing. For a `0.x` minor bump, read the
   relevant docs in the pinned source and the doc comments of the APIs you use
   (e.g. `locale.go` spells out that `App.Locale()` changed meaning in 0.5:
   it now returns the language the app *shows*, not the user's preferred one —
   `PreferredLanguages()` reports the old value).
4. Preview UI API drift with `go tool mygo migrate-ui .` (apply with `-write`).
5. Validate: `go build ./...`, `go vet ./...`, `go tool mygo vet .`, `go test ./...`.

## Migrating to native i18n

Replace a hand-maintained Go string table with MyGo catalogs:

1. Add i18n to `mygo.json`: `"i18n": { "dir": "locales", "source": "<sourceTag>" }`.
   The source language's catalog must hold every key; others fall back to it.
   Keep the app's existing default language as `source` (here `zh-Hans`) so
   behavior with no system match is unchanged.
2. Write one ARB/ICU JSON per language in `locales/`, named by BCP 47 tag
   (`zh-Hans.json`, `en.json`). Use ICU MessageFormat for plurals and args:
   `"imageCount": "{count, plural, one {# image} other {# images}}"`. Give
   each source message a `@key` description and typed `placeholders`.
   Pre-formatted display strings (e.g. a human size like "4 KB", a percent
   already rendered) are plain `{name}` placeholders of type `String`.
3. Generate accessors: `go tool mygo generate .` writes `locales/catalog_gen.go`.
   **Commit the generated file** with the catalogs so `go build`/`go test`
   work without the CLI.
4. In views, resolve per frame via `loc.In(c.Locale()).Message(...)` so text
   follows live language changes. Outside a view (menus, dialog titles built in
   a goroutine), use `loc.In(mygo.App.Locale())` — fine at runtime because the
   app is running; just never in a test (see the rule above).
5. Hand language selection to MyGo: call `mygo.App.SetLocale(tag)` instead of
   persisting your own language field; rebuild menus in `App.OnLocaleChanged`.
   Import the generated package for its side-effecting `init` that registers
   the catalogs: `import _ "<module>/locales"`.
6. Check catalogs with `go tool mygo i18n check .` (also run by `mygo build`).

## Verification checklist

- `go tool mygo i18n check .` reports the expected message count per language.
- `go build ./...`, `go vet ./...`, `go tool mygo vet .` clean.
- `go test -timeout 60s ./...` green, with locale cases driven by
  `Tester.SetLocale`, not `mygo.App` calls.
