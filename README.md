# Mandarin

> One self-hosted app for the household: tasks, shopping lists that work with no signal, recipes
> and the week's menu.

Mandarin is a soft fork of **[Vikunja](https://vikunja.io)**, the open-source, self-hosted task
manager by [kolaente](https://github.com/kolaente) and the
[Vikunja contributors](https://github.com/go-vikunja/vikunja/graphs/contributors). Everything you
like about the task side (projects, list/table/kanban views, saved filters, reminders, CalDAV,
the API) is Vikunja's work. Mandarin adds household features on top.

This is a personal household project. For a task manager for yourself or your team, please use
and [support upstream Vikunja](https://vikunja.io/support/) directly.

## Contents

- [What Mandarin adds](#what-mandarin-adds)
- [Running it](#running-it)
- [Configuration](#configuration)
- [Obsidian plugin](#obsidian-plugin)
- [Development](#development)
- [Branches](#branches)
- [License](#license)

## What Mandarin adds

**Household mode.** One household per server, and every user is in it automatically. Shopping
lists and recipes are shared with everyone; task projects stay private unless shared.

**Home page.** Quick add, this week's menu (tonight's dinner first), the shopping lists, recently
viewed recipes and current tasks on one page, with Home, Shopping, Recipes and Menu at the top of
the menu. Pick which one the app opens on in Settings → General.

**Shopping lists**
- A ShopShop-style list: one line per item, a big tick target, ticked items drop to the bottom.
- Quantities and units, typed straight after the name ("flour 1kg"), or dictated with the mic
  button ("two litres of milk", "half a kilo of mince").
- Items grouped by category, from a pre-loaded catalogue that learns every new item; typing
  autocompletes, most-bought first.
- Shops as tabs on a list (e.g. Groceries = Aldi + Supabarn). Ticking at one shop ticks it at all
  of them; items can be marked *not stocked* at a shop, or tied to one shop only.
- Each shop has its own aisle order for categories.
- Optional photo per item (the right brand).
- **Offline-first:** every change happens on the phone straight away and syncs when the signal
  comes back. A sync badge shows what's waiting; a conflict log shows which edit won. Identical
  items added on two phones offline are merged.

**Recipes**
- Web import of the exact ingredients, numbered steps and pictures, using the page's recipe data,
  then [recipe-scrapers](https://github.com/hhursev/recipe-scrapers), then AI as a last resort.
- Import printed recipes scanned into [Paperless-ngx](https://docs.paperless-ngx.com/), with the
  scan as the recipe's picture and a link back to the document.
- Every import shows a preview to check and fix before saving.
- Add a recipe's ingredients to a shopping list, with their amounts.
- **Cooking mode:** full screen, with ingredients scaled to the servings and tickable, and one
  step at a time in large text (swipe, next/back or arrow keys). Times in a step ("simmer 20
  minutes") become tap-to-start timers that beep and vibrate when done; several can run at once.
  The screen stays on, and reopening the recipe within 6 hours picks up where you left off.

**Menu planning**
- One menu for the household, a week at a time (Monday to Sunday), with today highlighted.
- Plan a recipe for dinner from the Menu page's add field (it suggests recipes as you type) or
  with the Plan button on a recipe, or jot down a free-text note ("leftovers").
- Each planned recipe has its own servings. Move an entry to another day, add its ingredients to a
  shopping list at those servings, or cook it straight from the menu.
- Deleting a recipe keeps its menu entries as notes with the recipe's name.

**Tasks**
- All of Vikunja's task features, minus Teams (one household, so they're hidden in the app; the
  API is unchanged).
- A mic button to dictate into the add-task field, where the browser supports speech recognition.
- Task lists stay readable offline (editing still needs signal).

**Push notifications** (Vikunja only sends email)
- Task reminders and a daily overdue summary on phones and laptop browsers, through Web Push.
- Opt-in shopping list summaries when someone else changes a list, batched and held during quiet
  hours (10pm–7am).
- Turned on per device in Settings → Notifications. On iPhone and iPad this works for the app
  added to the home screen (iOS/iPadOS 16.4+), not Safari tabs.

**Everything else**
- Mandarin name, logo, app icons and a mandarin-orange theme (light and dark) instead of
  Vikunja's blue.
- Installs to the home screen on iPhone and Android as a full-screen app.
- Nightly backups of the database and files.

See [SPEC.md](SPEC.md) for the full plan and build phases.

## Running it

GitHub Actions builds two images from the `household` branch:

| Image | What it is |
| --- | --- |
| `ghcr.io/fubartutu/mandarin:household` | The app: Go backend and Vue frontend in one container. Also tagged `latest`, and `sha-<commit>` for rollbacks. |
| `ghcr.io/fubartutu/mandarin-scraper` | The recipe scraper (Python, FastAPI) on port `8000`. Optional: without it, recipes still save and simple ingredient parsing still works, but web and Paperless import don't. |

The app image is a drop-in replacement for `vikunja/vikunja`: same port `3456`, same `/db` and
`/app/vikunja/files` volumes, same `VIKUNJA_*` environment variables. So
[Vikunja's install docs](https://vikunja.io/docs/installing) apply, including the reverse proxy
notes (websockets on; `VIKUNJA_SERVICE_PUBLICURL` must match the address exactly).

It's built for SQLite and a single household. Once everyone's accounts exist, turn registration
off with `VIKUNJA_SERVICE_ENABLEREGISTRATION=false`.

## Configuration

On top of Vikunja's own settings:

**App container**

| Variable | Default | What it does |
| --- | --- | --- |
| `MANDARIN_SCRAPER_URL` | `http://mandarin-scraper:8000` | Where the scraper container is. |
| `MANDARIN_BACKUP_DIR` | *(off)* | Folder for nightly backups (`mandarin-YYYY-MM-DD.tar.gz`: the database plus the files folder). Backups are off without it. |
| `MANDARIN_BACKUP_KEEP_DAYS` | `14` | How many days of backups to keep. |
| `MANDARIN_BACKUP_AT` | `03:00` | When to back up, in Mandarin's time zone (`VIKUNJA_SERVICE_TIMEZONE`). |

Push notifications need no settings: the server makes its own keys on first start and keeps them
in the database, so backups cover them.

**Scraper container**

| Variable | What it does |
| --- | --- |
| `PAPERLESS_URL` | Paperless-ngx as the container reaches it, e.g. `http://paperless:8000`. Paperless import is off without this and the token. |
| `PAPERLESS_TOKEN` | A Paperless API token (your profile → API auth token). |
| `PAPERLESS_TAG` | The tag on recipe scans. Default `recipe`. |
| `PAPERLESS_PUBLIC_URL` | Paperless as your browser reaches it, for the link on a recipe. Defaults to `PAPERLESS_URL`. |
| `AI_API_KEY`, `AI_MODEL` | An Anthropic key (`sk-ant-…`) or a key for an OpenAI-compatible API, and the model to use. |
| `AI_API_URL` | The API's address. Defaults to Anthropic for an `sk-ant-` key, otherwise OpenAI. |
| `OLLAMA_URL`, `OLLAMA_MODEL` | A local Ollama instead, used when no API key is set. |

The AI is only used when the simpler methods can't read a recipe. With none of the AI settings it's
off, and imports use the rules alone.

## Obsidian plugin

[`household-obsidian/`](household-obsidian/) is a private Obsidian plugin: due and overdue tasks in
a sidebar, task lists in any note, turning a note line into a Mandarin task, and saving a recipe as a
Markdown note. It works through Cloudflare Zero Trust with a service token. See its
[README](household-obsidian/README.md) to build, install and set it up.

## Development

Mandarin is built like Vikunja (Go with `mage`, Vue 3 + TypeScript with pnpm); see
[Vikunja's development docs](https://vikunja.io/docs/development/) and [AGENTS.md](AGENTS.md). Use
Node 24 (pinned in `mise.toml`).

Fork-specific code lives in its own folders so upstream merges stay easy:

| Folder | What's there |
| --- | --- |
| `pkg/household/` | Backend modules: shopping, recipes, menu, push, backup. |
| `pkg/routes/api/v2/household_*.go` | API routes. |
| `frontend/src/household/` | Frontend: home page, shopping, recipes, cooking mode, menu, push settings, theme, branding. |
| `household-scraper/` | The recipe scraper container (tests: `pytest`). |
| `household-obsidian/` | The Obsidian plugin. |
| `.github/workflows/household-*.yml` | The two image builds. |

The few edits to Vikunja's own files are marked with `FORK:` comments (`git grep -n "FORK:"`).
[CLAUDE.md](CLAUDE.md) has the soft-fork rules and notes on each module.

## Branches

- `household` is where Mandarin's work lives, and what the images are built from.
- `main` mirrors [go-vikunja/vikunja](https://github.com/go-vikunja/vikunja) `main`, untouched.
  Upstream is merged into `household` every few weeks.

## License

Like Vikunja, Mandarin is licensed under the
[GNU Affero General Public License v3.0 or later](LICENSE). The full source is public in this
repository. Original Vikunja code is copyright the Vikunja authors. The contents of
[`desktop/`](desktop/) are Vikunja's desktop app, licensed under [GPL-3.0-or-later](desktop/LICENSE).

Background images from Unsplash are distributed under the [Unsplash License](https://unsplash.com/license),
which requires credit to the photographer and Unsplash.
