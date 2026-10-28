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
  then `git switch household && git merge main`. On a `README.md` conflict keep ours. Then
  disable any new upstream workflows (rule 5).

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
5. Upstream's own workflows are disabled in the fork's GitHub Actions settings, not deleted,
   so merges stay clean. `household` is GitHub's default branch (since 2026-10-24), so every
   upstream workflow file on it registers as active, including scheduled ones. After each upstream
   merge (it can add workflows), disable them and check that only the `Household …` ones stay active
   (`gh workflow list -R fubartutu/mandarin --all`). "Not active" errors are for ones already off:
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
  quick add, the next 7 days' menu (`menu/homeWeek.ts`), shopping lists, recently viewed recipes
  (`recipes/recent.ts`, localStorage) and current tasks. `HouseholdNavItems.vue` sits at the top of
  the menu (Home, Shopping, Recipes, Menu, then a "Tasks" divider); the Overview entry was removed
  from `Navigation.vue` (FORK). The start page (Home, Shopping, Recipes or Menu) is `household_start_page` in `frontend_settings`, picked in Settings → General
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
- **Menu planning:** `pkg/household/menu` (online-only), routes in `household_menu.go`
  (`/api/v2/menu`), frontend `frontend/src/household/menu` (`/menu/:start?`: 7 days from today,
  or from `start`; arrows move 7 days; local time, `week.ts`). One `menu_entries` table: an entry is a recipe (`recipe_id`,
  own `servings`: 0 when the recipe has none, else at least 1) or a free-text `note`, for a day's
  breakfast, lunch or dinner (only dinner shown), ordered by `position` within the meal. Error
  codes 903xx. `recipes` imports `menu`, never the other way round: deleting a recipe calls
  `menu.RecipeDeleted` in the same transaction, which turns its entries into notes. "Add to list"
  on an entry reuses the recipe page's ingredient picker scaled to the entry's servings.
- **Cooking mode:** frontend only, `frontend/src/household/cooking` (`/recipes/:id/cook`,
  `?servings=N` or `?scale=X`), teleported to `body` so it covers the top bar. Timers come from
  times in step text (`steps.ts`) and are plain data with `endsAt`; resume state is per recipe in
  localStorage for 6 hours (`resume.ts`). Wake lock via vueuse's `useWakeLock`.
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
- **Repeat rules and ends:** a repeating task's household settings are one row in `task_repeats`
  (model `TaskRepeat` in `pkg/models/household_task_repeat.go`, a new file because it uses unexported
  task helpers; routes `household_task_repeat.go`, `/api/v2/tasks/{task}/repeat`). `rule` is an RRULE
  subset (FREQ WEEKLY/MONTHLY/YEARLY, INTERVAL, BYDAY, BYMONTHDAY, BYSETPOS; rrule-go, server time
  zone) that picks the next date when the task is done: `applyTaskRepeat` (FORK call in `Task.Update`
  before buckets) sets the interval to the exact gap for that update so Vikunja's `updateDone` moves
  the dates and reminders, and the returned restore (FORK line right after `updateDone`) puts the
  stored interval back. A task with a rule keeps a stand-in `repeat_after` (7/30/365 days × interval)
  so Vikunja, CalDAV and other clients see it repeat; the frontend saves it through the task first,
  the server only sets it on tasks that don't repeat. `until`/`repeats_left` end the series
  (`repeats_left` counts the open occurrence; completing the last clears the repeat so it stays done).
  Removing the repeat removes the row. Other FORK lines: `hardDeleteTask`, `models.GetTables()`.
  Frontend `household/tasks/`: `RepeatSettingsField.vue` (FORK in `TaskDetailView.vue`, gets its
  `saveTask`), `repeatRule.ts` (rules, first date, quick add phrases) and `repeatEnd.ts`.
- **Reschedule and skip:** task rows' due-date popup is `household/tasks/RescheduleMenu.vue` (a FORK
  import swap for Vikunja's `DeferTask` in `SingleTaskInProject.vue`, same props and emit): Today,
  Tomorrow, Weekend, Next week (absolute days at the task's time, `reschedule.ts`), the calendar, and
  "Skip to <next>" on repeating tasks. Skip is `POST /api/v2/tasks/{task}/skip` (`SkipTaskOccurrence`):
  dates and reminders move as completing would, not done, `repeats_left` unchanged; past `until` the
  series ends. `GET .../repeat` returns `next` for the label. Task page: `SkipOccurrenceButton.vue`
  next to Mark done (FORK); it hands the skipped task to `seedTask`, because TaskDetailView keeps its
  own copy of the dates and would save the old ones back. `task_repeats` isn't in Vikunja's fixture
  list (`setup_tests.go`), so household tests empty it themselves.
- **Household quick add magic:** `household/tasks/quickAdd.ts` reads Todoist phrases off the title
  before Vikunja's magic (FORK lines in `useQuickAddTask.ts`), then applies them to the built task:
  calendar rules ("every mon, thu", "every weekday", "every 2nd tuesday", "every 15th", "every 3 months",
  "yearly"; swapped for Vikunja's own "weekly"/"monthly"/"yearly" so its date and reminder handling
  still runs), repeat ends, "every! 3 days" (repeat from completion; months/years are 30/365 days),
  "starting nov 1" (drops "starting" so the date becomes the first due date) and "p1"-"p4"
  (urgent/high/medium/low; Vikunja's "!3" wins). Off when quick add magic is disabled.
- **No Teams:** one household, so Teams is hidden from the UI only (backend untouched): menu entry
  (`Navigation.vue`), its shortcut listing (`shortcuts.ts`), team sharing (`ProjectSettingsShare.vue`)
  and quick-action team search and "New team" (`QuickActions.vue`), all FORK lines. The routes stay.
- **Obsidian plugin:** `household-obsidian/` (own npm project, Node 24: `npm run build`, `npm test`,
  `node install.mjs` copies it into the HOME vault). Talks to /api/v2 with a Mandarin API token
  plus Cloudflare Access service-token headers, through Obsidian's `requestUrl`. Pure text logic is in
  `markdown.ts` (tested); `api.integration.test.ts` runs against a server when MANDARIN_TEST_URL is set.
- **iPhone/iPad:** `household/ios.ts` (from `setup.ts`) adds `maximum-scale=1` to the viewport on
  Apple touch devices only, so Safari doesn't zoom into fields with text under 16px (and stay zoomed).
  iOS still allows pinch-zoom; Android would not, so it's never added there.
- **Colours:** `household/theme.scss` (imported by `setup.ts`) overrides Vikunja's `--primary-*`
  tokens with mandarin orange, checked for WCAG contrast in light and dark mode; `index.html` and
  `vite.config.ts` carry the matching `theme-color` (FORK).
- **Layout fixes to Vikunja views** go in household stylesheets imported by `setup.ts`, with selectors
  that outrank the component's scoped ones, instead of FORK edits: `household/kanban.scss` (the
  "Create a bucket" button at the bucket headers' height).
- **Kanban cards:** `household/tasks/kanban/KanbanCardExtras.vue` (one FORK insert under the title in
  `KanbanCard.vue`) shows the description's first line as a one-line subtitle (`cardText.ts`) and,
  after a 1 s hover with a real pointer, the latest 3 comments (Vikunja's `commentsQuery`, newest
  first, fetched on first hover; nothing for tasks without comments). It listens on the card it sits
  in (`closest('.task')`) instead of wrapping it; a pointerdown (drag) closes the popup. It also sets
  `data-household-urgency` (red/orange/yellow, `urgency.ts`: priority or due date, the more urgent
  wins, never on done tasks; re-checked every minute via `useGlobalNow`) on the card, which
  `household/kanban.scss` draws as an inset stripe on the left in front of Vikunja's card shadow.
- **Task row subtitles:** List view rows (and Home's current tasks, Upcoming: all `SingleTaskInProject`)
  show the same first line under the title, `household/tasks/TaskSubtitle.vue` (one FORK insert).
- **Last view per project:** Vikunja keeps it only in localStorage (`helpers/projectView.ts`), which
  logout clears. `household/projectViews.ts` (registered in `setup.ts`, no Vikunja edits) also keeps
  it in `frontend_settings.household_project_views` (`{projectId: viewId}`, type FORK in
  `helpers/userSettings.ts`): a router guard swaps in the saved view when Vikunja's redirect has none
  (view id 0), and an `afterEach` saves the shown view 1.5 s later if it changed, through Vikunja's
  own settings mutation. A deleted view falls back to Vikunja's default view.
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

## Agent skills

### Issue tracker

GitHub Issues on `fubartutu/mandarin` (always `gh ... -R fubartutu/mandarin`). See `docs/agents/issue-tracker.md`.

### Triage labels

The five default labels: `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: `GLOSSARY.md` and `docs/adr/` at the repo root, created when first needed. See `docs/agents/domain.md`.
