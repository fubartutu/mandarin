# Mandarin: Household App Spec

Oct 20, 2026 · Sam Primrose

## Overview and goals

One self-hosted app for the household: shared shopping lists that keep working with no signal, a recipe database with reliable web import, and full task management with private and shared lists.

- **Shopping:** fast to use in the shop, works offline, syncs when the signal returns, covers groceries and other family shopping (kids' shoes, Kmart runs).
- **Recipes:** import from recipe blogs with the exact ingredients, numbered steps and photos; add ingredients to the shopping list.
- **Menu:** plan the week's dinners from the recipes; cook from a full-screen step-by-step view.
- **Tasks:** Vikunja-level task management with due dates, reminders, saved filters and multiple views.
- **Everywhere:** looks good and works well on phone, tablet and laptop; both an iPhone and an Android phone.
- **Self-hosted:** runs in Docker on the home ZimaOS server.

Inspiration: [Vikunja](https://vikunja.io) (look, task features), [KitchenOwl](https://kitchenowl.org) (recipes, single shared grocery list), ShopShop (list feel), [Specifically Clementines](https://github.com/davideshay/groceries) (shop groups, sync visibility).

## Approach

Decision: build an all-in-one app as a **soft fork of Vikunja**, adding shopping, recipes and household mode as new modules.

- **Why Vikunja:** its task features (views, filters, reminders, CalDAV, API, shortcuts, settings) would take months to rebuild; the shopping and recipe features are smaller additions.
- **Stack:** Go backend, Vue 3 + TypeScript frontend, one repo, one container. A small Python container handles recipe scraping.
- **Soft fork rules:** new code lives in its own folders; edits to Vikunja's own files stay small and clearly marked; upstream Vikunja is merged in every few weeks for features and security fixes. Going to a hard fork later stays possible.
- **Rebrand:** own app name and logo, own theme defaults. Credit Vikunja in the README.
- **Licence:** the fork stays AGPL-3.0; source is public on GitHub. Ideas and code may be borrowed from Specifically Clementines (MIT).
- **Learning:** Sam learns Go and Vue 3 + TypeScript through the code as features are built.

## Users and household

The server has exactly one household; every user belongs to it automatically. There is no household picker anywhere in the app.

- Each person has their own login.
- **Shopping lists and recipes** always belong to the household and are shared with everyone.
- **Task projects** are private to their owner by default and can be shared with the household or individual users, using Vikunja's existing sharing.
- Tasks can be assigned to another household member.
- After the household's accounts exist, open registration is turned off.

## Tasks

Keep Vikunja's task management as it is; these are the parts Sam specifically values.

- **Views per project:** list, table and kanban. Other views (Gantt etc.) can stay but aren't priorities.
- **Navigation:** Overview, Upcoming and Projects.
- **Saved filters:** dynamic lists defined by a query, e.g. `done = false && assignees in sam`.
- Due dates, reminders, recurring tasks, labels, priorities, sub-tasks.
- API, bots, notification services and calendar (CalDAV) integration.
- The full settings panel and keyboard-shortcut workflow.

## Shopping lists

A new shopping module with a ShopShop-style list: one line per item, a big tick target, no tiles.

### The list

- Each item has a name and an optional **quantity + unit** (2, 500 g, 1 L, 3 packs) and an optional note.
- Tap to tick; ticked items grey out and drop to the bottom, with a button to clear them.
- Items are grouped by **category** (Produce, Dairy, Frozen…), each group collapsible. The All tab uses the household's category order; each shop's tab uses that shop's aisle order.
- Not just groceries: shoes, school supplies and other family shopping go on lists too.

### Shops and shop groups

- Each shop has its own list; lists are grouped, e.g. **Groceries = Aldi + Supabarn**, another group for Kmart or Target.
- Adding an item to a group puts it on every shop list in that group.
- Ticking an item at one shop ticks it off the others in the group ("bought it already").
- An item can be marked **not stocked** at a shop, so it is never added to that shop's list.
- An item can be given a **shop** when it's added (or later, when editing it): "only at Supabarn". It shows on All, labelled with the shop, and on that shop's tab, and is hidden at the other shops. Left blank, it goes to every shop as before.
- Each shop has its own **aisle order**: the order its categories come in as you walk the shop. Set in the shop manager; categories it doesn't list follow in the household order. Changing it needs signal; the order is kept on the phone, so it works offline.

### Adding items

- A **pre-loaded item catalogue** with default categories, so common items sort themselves from day one.
- Every new item the household types is saved to the catalogue.
- Typing **autocompletes** from the catalogue, frequently and recently bought items first; an item re-added keeps its last quantity, unit and category.

### Item photos

- Optional photo per item (e.g. the right brand). Stored only for items that have one; no space used otherwise.

## Offline sync

Shopping lists are offline-first: every action happens on the phone instantly and syncs when there is signal. We write our own sync on Vikunja's database, borrowing Clementines' ideas, rather than adding CouchDB.

```
Tick / add / edit → Phone database (IndexedDB) → Outbox queue → [Signal?]
                         │                            ▲           │ yes → Home server → other phone updates live
                         ▼                            └── no: keeps waiting
                 Screen updates instantly
```

1. The phone keeps a copy of the household's shopping data in its local database (IndexedDB).
2. Every tick, add or edit is written locally first, so the screen responds instantly, and is added to an outbox queue.
3. When online, the outbox is sent to the server; the phone then pulls changes made since its last sync.
4. While both phones are online, changes appear live on the other phone.
5. Conflicts resolve per field, the latest change winning, so two different edits to one item both survive.

- **Sync badge:** a small indicator shows online, offline with *n* changes waiting, or syncing.
- **Conflict log:** a rare conflict shows a warning on the badge; the log shows which change won and which lost.
- **Tasks:** online-only at first. Later, task lists become readable offline (no offline editing).

## Recipes

A household recipe database whose web import gets the exact recipe: ingredients parsed properly, numbered steps intact, one or two images.

- **Recipe record:** title, source link, servings, times, ingredients (quantity, unit, name, note), numbered steps, 1–2 images, tags.
- **Add to shopping list:** a recipe's ingredients go on the shopping list with their quantities.
- **Readable on the laptop or tablet while cooking.**

### Scraper (separate Python container)

1. Read the recipe data the page already publishes in machine-readable form (schema.org Recipe), which carries exact ingredients and steps.
2. Fall back to the recipe-scrapers library for sites needing site-specific handling.
3. Last resort: an AI model extracts the recipe; it also splits each ingredient into quantity, unit and name. Can run locally on sampc's GPU.

Every import shows a preview to correct before saving.

### Paper recipes from Paperless-ngx

Paper recipes (mostly printed: cookbook pages, magazine cut-outs) are scanned into the Paperless-ngx instance already running on the server. Mandarin imports them on demand.

- **Tag:** recipe scans get a `recipe` tag in Paperless.
- **Picker:** Add recipe → From Paperless lists the tagged documents with thumbnails; picking one starts an import.
- **Extraction:** the scraper container reads the document's OCR text through the Paperless API (address and API token in its settings) and turns it into a recipe: simple rules first (Ingredients/Method headings, lines starting with an amount), the AI model on sampc when the layout is messier. Ingredients are split into amounts the same way as web imports.
- **Preview:** the same check-and-fix screen as a web import.
- **Picture and link:** the scanned page becomes the recipe's picture, and the recipe links back to the Paperless document.
- **AI model:** runs on sampc's GPU (e.g. Ollama), shared with Sam's Whisper setup. When sampc is off, imports fall back to the simple rules.

## Menu planning

One shared household menu: what's for dinner each day of the week, planned from the recipes.

- **The plan:** one for the whole household; anyone can edit it. Online-only (no offline sync); a week already opened stays readable without signal, like tasks.
- **Entries:** each day has meal slots. An entry is either a **recipe** with its own servings (defaulting to the recipe's, e.g. "for 6 on Saturday") or a **free-text note** ("leftovers", "takeaway", "at Mum's"). A slot can hold several entries in order (main and a side).
- **Meals:** the model has breakfast, lunch and dinner; the app only shows dinner for now. Showing the others later is a small change.
- **Week view:** Monday to Sunday, with arrows to earlier and later weeks. One scrolling list of day cards on every screen size; today is highlighted and scrolled into view, past days are dimmed.
- **Planning:** "+" on a day searches the recipes or takes a note; a recipe page has a **Plan** button with a date picker. An entry's menu: move to another day, change servings, add to list, cook, delete.
- **Shopping:** ingredients go on a list one recipe at a time, when wanted: "Add to list" on a planned entry is the recipe's usual add-to-list, scaled to the entry's servings. No add-the-whole-week button.
- **Deleted recipes:** a planned entry whose recipe is deleted turns into a note with the recipe's title, so past weeks still read properly.
- **History:** past weeks are kept. No cooked/skipped marking (edit the entry instead), no copying of earlier weeks, no suggestions yet.
- **In the app:** a **Menu** entry in the menu after Recipes, a "Tonight / this week" section on Home, and Menu as a start-page choice.

## Cooking mode

A full-screen view for cooking one recipe, opened from a recipe page, or from a planned entry at that entry's servings.

- **Ingredients:** scaled to the servings, each one can be ticked off.
- **Steps:** one at a time in large text; swipe or next/back.
- **Screen stays on** while cooking mode is open (Wake Lock).
- **Timers:** a step that mentions a time ("simmer 20 minutes") gets a tap-to-start timer. Several can run at once. They run only while cooking mode is open and finish with a sound, vibration and a banner (no push).
- **Resume:** leaving and coming back to the same recipe within 6 hours returns to the same step, with the ticked ingredients and running timers. Kept on that device only.

## Push notifications

Notifications on the phones' home-screen app (and laptop browsers) through Web Push, which iPhone and iPad support for home-screen apps since iOS/iPadOS 16.4, and Android and desktop browsers always have. Vikunja has none: reminders only go by email.

- **Turning it on:** Settings → Notifications → "Turn on for this device". The permission prompt has to come from a tap (an iOS rule). Each device is listed and can be removed. A "Send a test" button checks it works.
- **Task reminders:** a push at each reminder's time, saying the task and when it's due. Tapping opens the task.
- **Daily overdue summary:** once a day at the time already set for the overdue email, a single push listing overdue tasks ("3 overdue: car rego, …").
- **Shopping list changes:** off unless turned on for a list. Changes the other person makes are batched: one push once they've stopped for about 10 minutes ("Alex added milk, eggs and 3 more to Groceries"). Never for your own changes.
- **Quiet hours (10pm–7am, on by default):** shopping summaries wait until 7am. Reminders and the overdue summary come at the times you set.
- **Per person:** each person picks which of these push, and keeps email reminders as well if wanted (each event: push, email, or both).
- **How it works:** the server has its own key pair (VAPID), made on first start and kept with the database. Each device's push address is stored per user and forgotten when the push service says it's gone. Pushes go from the server to Apple's or Google's push service directly, so Cloudflare Zero Trust isn't involved. Tapping a notification opens the app as usual.
- **Limits:** iPhone and iPad only deliver to the app added to the home screen, not to Safari tabs. Phones may delay pushes in low-power mode.

## Obsidian plugin

A private Obsidian plugin (kept in this repo, copied into the HOME vault's plugins folder; LiveSync carries it to the other machines). It talks to Mandarin's API with a Mandarin API token.

- **Tasks in Obsidian:** a sidebar with due-today and overdue tasks, and a code block for any note (e.g. a project's tasks, or a saved filter). Tasks can be ticked off from there.
- **Note → task:** a command (and right-click on a line) turns the line into a Mandarin task, with a due date if the line has one, and a link back to the note (an `obsidian://` link) in the task. The line gets a link to the task.
- **Recipes as notes:** a command picks a Mandarin recipe and saves it as a Markdown note: frontmatter (source, servings, times, tags), ingredients as a list, numbered method, the main picture saved into the vault.
- **Connection:** Mandarin is behind Cloudflare Zero Trust, which an app can't log in to. The plugin sends a Cloudflare Access service token (client ID and secret, set in its settings) with every request, so it works at home and away and Zero Trust stays in place.
- **Settings:** Mandarin address, API token, Cloudflare service token, the folder for recipe notes.

## Platforms and look

A web app installed to the home screen on both phones (no app stores), plus the full website on laptop and tablet.

- **Phone:** iPhone and Android, added to the home screen so it opens full screen like an app.
- **Tablet and laptop:** the full desktop layout, including recipes while cooking.
- **Look:** modelled on Vikunja's design and colour scheme; the shopping list follows ShopShop's simplicity.
- **Theming:** light, dark and system modes, plus theme options.
- **Store apps:** possible later by wrapping the same code (Capacitor) if the home-screen app falls short.

## Hosting and workflow

The app runs on the ZimaOS server (192.168.1.52) behind the existing Nginx Proxy Manager, with HTTPS on its own subdomain.

- **Containers:** the app (Go backend + Vue frontend in one image) and the recipe scraper (Python). SQLite to start.
- **Proxy:** Nginx Proxy Manager with Let's Encrypt and websockets on; the app's public URL setting must match the subdomain exactly.
- **Repo:** a GitHub fork of `go-vikunja/vikunja`, public. `main` mirrors upstream; work happens on a separate branch; upstream is merged in every few weeks.
- **Builds:** GitHub Actions builds the Docker image and publishes it to GitHub's container registry (ghcr.io); the server pulls it.
- **Backups:** nightly `mandarin-YYYY-MM-DD.tar.gz` (database and files) in a folder on the server, kept 14 days. An off-box copy is still open (see open questions).
- **Trial:** stock KitchenOwl and Vikunja are running now for evaluation; they're retired once the fork replaces them.

## Build phases

Groceries come first; each phase ends with something usable on the phones.

1. **Fork and pipeline:** fork on GitHub, rebrand, automated Docker build, deploy to ZimaOS in place of stock Vikunja.
2. **Household mode + basic shopping list:** single household, shopping module with the ShopShop-style view, quantities and units, categories. Online-only; first lesson through Go and Vue.
3. **Catalogue and shop groups:** pre-loaded catalogue, autocomplete, Aldi + Supabarn group with tick-once behaviour, not-stocked flag, item photos.
4. **Offline-first sync:** local database, outbox, live updates, sync badge, conflict log. Ends with a dead-zone test in the supermarket.
5. **Recipes:** recipe database, scraper container, add ingredients to the shopping list.
6. **Paper and AI import:** import printed recipes from Paperless-ngx; AI extraction on sampc for scans and for web pages without recipe data.
7. **Polish:** offline-readable tasks, theming, any trial feedback.
8. **Push notifications and Obsidian:** task reminders and the daily overdue summary on the phones, opt-in shopping list summaries, quiet hours; then the Obsidian plugin (tasks, note → task, recipes as notes).
9. **Shops, part 2:** an optional shop per item (chosen when adding or editing), and a per-shop aisle order for categories.
10. **Menu planning and cooking mode:** the shared weekly dinner menu first (planner, Home section, Plan button on recipes), then cooking mode, opened from recipes and planned entries.

## Out of scope and later

**Not included:** finances or expense tracking (from KitchenOwl), multiple households.

**Later:**

- Voice add via Siri Shortcuts or Google Assistant.
- Offline editing of tasks (not just reading).
- Store apps for iOS and Android.
- Menu: breakfast and lunch slots, copying earlier weeks, suggestions, a "tonight's dinner" push.

## Open questions

- [x] App name: **Mandarin**. Logo and app icons done (Phase 7).
- [x] GitHub account for the fork: public repo [fubartutu/mandarin](https://github.com/fubartutu/mandarin).
- [x] Dev setup: both. Code and tests run locally on the laptop (Node 24 via `mise`); GitHub Actions builds the Docker image and ZimaOS pulls it.
- [x] Meal planning and cooking mode: both wanted, in Phase 10 (menu planning first).
- [x] Backups: nightly to a folder on the server (database and files, kept 14 days). Still open: an extra copy off the box, e.g. the Synology NAS.
- [x] Notifications when the other person changes a shared list: per list, off by default, batched (Phase 8).
- [x] Vikunja trial reverse-proxy issue: resolved, Mandarin is live behind Nginx Proxy Manager.
