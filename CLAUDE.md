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
   so merges stay clean. GitHub only lets you disable a workflow once it's registered, so after
   the first `main` push from an upstream sync (which runs upstream `ci.yml` once), run:
   `for f in .github/workflows/*.yml; do case $f in *household*) ;; *) gh workflow disable $(basename $f) -R fubartutu/mandarin;; esac; done`

## Upstream guidance

Vikunja's own conventions (commands, `mage`, API v2, generated clients, skills) still apply:

@AGENTS.md
