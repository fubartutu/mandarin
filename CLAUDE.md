# Mandarin — soft fork of Vikunja

Mandarin is a household app (tasks + shopping lists + recipes) built as a **soft fork** of
[Vikunja](https://github.com/go-vikunja/vikunja). See [SPEC.md](SPEC.md) for what we're building
and the build phases. Sam is learning Go and Vue 3 + TypeScript through this project: explain
non-obvious Go/Vue idioms briefly when writing code.

## Branches and remotes

- `upstream` = `go-vikunja/vikunja`, `origin` = Sam's fork.
- `main` is a clean mirror of `upstream/main`. Never commit to it; only fast-forward it.
- All our work happens on `household` (Docker image is built from it by
  `.github/workflows/household-docker.yml`).
- Syncing upstream every few weeks:
  `git fetch upstream && git switch main && git merge --ff-only upstream/main && git push origin main`,
  then `git switch household && git merge main`. On a `README.md` conflict keep ours.

## Soft-fork rules

1. **New code in its own folders.** Fork-only code lives under `household`-named paths, e.g.
   `frontend/src/household/`, `pkg/household/`, `.github/workflows/household-*.yml`.
   New features (shopping, recipes, household mode) are new modules, not rewrites of Vikunja's.
2. **Edits to Vikunja's files stay minimal and are marked.** When a Vikunja file must change
   (a hook, an import, a route registration), make the smallest change that delegates into our
   folder and mark it so it's easy to find during upstream merges:
   - Go / TS / JS: `// FORK: <why>`
   - Vue templates / HTML: `<!-- FORK: <why> -->`
   - YAML / shell: `# FORK: <why>`

   `git grep -n "FORK:"` lists every touch point. Keep that list short.
3. **AGPL-3.0 stays.** Don't remove or change `LICENSE`, copyright headers or Vikunja credits.
   The source stays public. Code borrowed from Specifically Clementines (MIT) keeps its notice.
4. **Don't touch the license system** in `pkg/license/` (see upstream's guidance below).
5. Upstream's own CI workflows are disabled in the fork's GitHub Actions settings, not deleted,
   so merges stay clean. `ci` was disabled on 2026-10-20; the others only fire on PRs, issues or
   schedules (off on forks), so they never registered. If one ever runs, disable it with:
   `for f in .github/workflows/*.yml; do case $f in *household*) ;; *) gh workflow disable $(basename $f) -R fubartutu/mandarin;; esac; done`

## Where fork code goes

Go forces a few compromises (one folder = one package, and unexported helpers can't be used from
another package), so in practice:

- **Models and logic:** `pkg/household/<module>/` (e.g. `pkg/household/shopping`). Permission is
  `household.IsMember` (any user; never link shares). Error codes start at 90001.
- **New files inside Vikunja packages** (no edits, so no merge conflicts) where Go requires it:
  v2 routes `pkg/routes/api/v2/household_<module>.go`, migrations `pkg/migration/<timestamp>.go`,
  webtests `pkg/webtests/household_<module>_test.go`. Our package must never import `pkg/migration`
  (import cycle).
- **Wiring (FORK lines):** tables in `schemaBeans()` in `pkg/migration/migration.go` (fresh
  installs skip migrations, so new tables must be listed there too); start-up seeding in
  `pkg/initialize/init.go`.
- **Frontend:** `frontend/src/household/`. Routes are spread into `src/router/index.ts`, menu
  entries come from `HouseholdNavItems.vue` in `Navigation.vue`, and `household/setup.ts` (imported
  from `main.ts`) registers icons and merges `household/i18n/en.json`. Don't edit Vikunja's
  `lang/en.json` or `Icon.ts`.
- **Home and start page:** `household/HomeView.vue` replaces Vikunja's Overview on `/` (added in
  `setup.ts` with `router.addRoute` under the same name `home`, so `router/index.ts` is untouched):
  quick add, shopping lists, recently viewed recipes (`recipes/recent.ts`, localStorage) and current
  tasks. `HouseholdNavItems.vue` sits at the top of the menu (Home, Shopping, Recipes, then a "Tasks"
  divider); the Overview entry was removed from `Navigation.vue` (FORK). The start page (Home,
  Shopping or Recipes) is `household_start_page` in `frontend_settings`, picked in Settings → General
  (`components/StartPageField.vue`, FORK in `GeneralSettingsForm.vue`) and applied by
  `startPage.ts` only when the app opens or after sign-in.
- Production runs on **SQLite**, which allows one writer at a time: two requests that each read
  then write can fail with "database is locked". Don't write rows that haven't changed, and don't
  fire two writing requests from the client at once (chain them).
- **Offline sync (shopping):** phones keep a copy in IndexedDB and send queued ops to
  `POST /api/v2/shopping/sync` (`pkg/household/shopping/sync.go`, `pull.go`;
  `frontend/src/household/shopping/sync/`). Every server-side write to shopping data must call
  `recordChange` (or `recordItemChange`) in the same transaction, or phones never see it. Item edits
  must keep `FieldTimes` up to date (latest change wins per field). Custom handlers that write must
  call `events.DispatchPending(ctx, s)` after commit, so open apps get the websocket "changed" push.
  An add identical to an unticked item is merged into it (`merge.go`); its uid becomes an alias
  that `ApplySyncOps` resolves, so look items up by resolved uid in new op types.
- **Shops:** a shop is a tab (view) on its list, not a list of its own. An item's `shop_id` (0 = any)
  is a synced field like `category_id` (unknown shops become 0; deleting a shop frees its items).
  A shop's `category_order` (aisle order, JSON on `shopping_shops`) arrives with its list; the
  frontend applies it with `inAisleOrder`, unlisted categories following in household order.
- **Recipes:** `pkg/household/recipes` (online-only; ingredients, steps, tags and image ids are
  JSON columns on `recipes`), routes in `household_recipes.go`, frontend `frontend/src/household/recipes`.
  Web import and ingredient parsing go to the scraper container, `household-scraper/` (Python,
  FastAPI, recipe-scrapers, ingredient-parser-nlp; tests: `pytest` there), reached at
  `MANDARIN_SCRAPER_URL` (default `http://mandarin-scraper:8000`). Saving never depends on it: the
  Go simple parser fills in when it's down. Calls to it (and image downloads) happen before the DB
  session opens. "Add to list" goes through the shopping sync engine (`shopping/adding.ts`).
  Paperless-ngx scans and the AI (Anthropic or an OpenAI-compatible API via AI_API_KEY/AI_MODEL, else Ollama on sampc) are scraper-side only (`paperless.py`,
  `text_recipe.py` rules, `ai.py`; settings in `settings.py`); Mandarin proxies them under
  `/recipes/paperless`, and a scan's page picture is `paperless:documents/<id>/image`.
- **Offline tasks (read-only):** `registerOfflineApi()` in `household/sw.ts` keeps every JSON
  GET answer from /api/v1 and /api/v2 (NetworkFirst, 8 s, 500 entries) in the `mandarin-api`
  cache, served without signal. Deleted on logout (FORK in `stores/auth.ts`) and when another user
  signs in (`offlineTasks.ts`). `OfflineBanner.vue` is its own mini app so it shows on every page.
  Endpoints that must never come from a cache go in `NOT_CACHED` there.
- **Offline start:** small FORK hooks in `stores/auth.ts`, `stores/base.ts`, `Ready.vue`,
  `client/queryClient.ts` and `sw.ts` delegate to `household/offline.ts` and `household/sw.ts`.
  Upstream's `sw.ts` precached twice, and the empty second copy wiped the cache on activate. That
  line is disabled (FORK); keep it disabled after merges, or the app won't open offline.
- **Backups:** `pkg/household/backup` (FORK line in `pkg/initialize/init.go`): nightly
  `VACUUM INTO` copy of the SQLite database plus the files folder, as `mandarin-YYYY-MM-DD.tar.gz`
  in `MANDARIN_BACKUP_DIR`, pruned after `MANDARIN_BACKUP_KEEP_DAYS` (14). Times follow
  `service.timezone`. Don't VACUUM through `db.NewSession()`: it opens a transaction.
- **Branding:** the mandarin mark and illustrations are SVGs in `frontend/src/household/assets`;
  Vikunja's llama pictures are swapped out by FORK lines (Ready, NoAuthWrapper, ShowTasks,
  MigrationHandler). Home-screen and tab icons live in `public/images/household`, rendered by
  `node scripts/household-icons.mjs`; index.html, vite.config.ts and useTimeTrackingFavicon.ts
  point there. The logo and mark drawn in the page are Vue components
  (`household/components/MandarinLogo.vue`, `MandarinMark.vue`) using `useId()` for gradient ids:
  the logo is drawn twice, and shared ids made it vanish when the first copy was hidden.
  English text that says "Vikunja" is overridden in `household/i18n/en.json` ("Powered by
  Vikunja" stays as credit). `household/branding.test.ts` fails when an upstream merge adds or
  changes such a string: add the override it names.
- **Push notifications:** `pkg/household/push` (Web Push via webpush-go; VAPID keys made on first
  start and kept in `push_vapid_keys`, so backups cover them). Reminders and the overdue summary come
  through a hook in Vikunja's `notifications.Notify` (new file `pkg/notifications/household.go`, one
  FORK line in `notification.go`, one in `task_overdue_reminder.go`); shopping summaries through
  `shopping.OnItemAdded`. `push.Init()` must run after `cron.Init()` and before the reminder crons.
  Frontend: `household/push/` (settings page added with `router.addRoute`, menu FORK in
  `views/user/Settings.vue`) and `registerPush()` in `household/sw.ts`. Headless Chromium can't
  subscribe (no push service); Playwright Firefox with `dom.push.enabled` can, end to end.
- **Dictation:** `household/dictation/` (browser speech recognition, Web Speech API). A mic
  button next to the add-task field (FORK in `components/tasks/AddTask.vue`) and the shopping
  add bar (`AddItemBar.vue`), only shown where the browser has speech recognition; what's said
  goes into the field for quick add magic or `parseItemInput` to parse.
- **No Teams:** one household, so Teams is hidden from the UI only (backend untouched): menu entry
  (`Navigation.vue`), its shortcut listing (`shortcuts.ts`), team sharing (`ProjectSettingsShare.vue`)
  and quick-action team search and "New team" (`QuickActions.vue`), all FORK lines. The routes stay.
- **Obsidian plugin:** `household-obsidian/` (own npm project, Node 24: `npm run build`, `npm test`,
  `node install.mjs` copies it into the HOME vault). Talks to /api/v2 with a Mandarin API token
  plus Cloudflare Access service-token headers, through Obsidian's `requestUrl`. Pure text logic is in
  `markdown.ts` (tested); `api.integration.test.ts` runs against a server when MANDARIN_TEST_URL is set.
- **Colours:** `household/theme.scss` (imported by `setup.ts`) overrides Vikunja's `--primary-*`
  tokens with mandarin orange, checked for WCAG contrast in light and dark mode; `index.html` and
  `vite.config.ts` carry the matching `theme-color` (FORK).
- **API-token permissions:** since upstream v2.7.1, start-up panics if two routes would share a
  token permission key. A new household resource with both a list and a read-one route must be
  added to `crudResources` in `pkg/models/api_routes.go` (FORK block). The household webtests catch it.
- After changing a v2 route or model, run `mage generate:frontend-client` and commit
  `frontend/src/client/generated`. On merge conflicts there, regenerate; never hand-merge.

## Local setup

Use Node 24 (pinned in `mise.toml`; run `mise trust && mise install` in the repo). Under Node 25+
the frontend unit tests fail with `localStorage` undefined. pnpm is pinned in
`frontend/package.json`; `npx pnpm@<that version>` works without a global install.

## Upstream guidance

Vikunja's own conventions (commands, `mage`, API v2, generated clients, skills) still apply:

@AGENTS.md
