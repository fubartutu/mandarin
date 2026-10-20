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
- Production runs on **SQLite**, which allows one writer at a time: two requests that each read
  then write can fail with "database is locked". Don't write rows that haven't changed, and don't
  fire two writing requests from the client at once (chain them).
- After changing a v2 route or model, run `mage generate:frontend-client` and commit
  `frontend/src/client/generated`. On merge conflicts there, regenerate; never hand-merge.

## Local setup

Use Node 24 (pinned in `mise.toml`; run `mise trust && mise install` in the repo). Under Node 25+
the frontend unit tests fail with `localStorage` undefined. pnpm is pinned in
`frontend/package.json`; `npx pnpm@<that version>` works without a global install.

## Upstream guidance

Vikunja's own conventions (commands, `mage`, API v2, generated clients, skills) still apply:

@AGENTS.md
