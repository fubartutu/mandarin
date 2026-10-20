# Mandarin

> One self-hosted app for the household: tasks, shopping lists that work with no signal, and recipes.

Mandarin is a soft fork of **[Vikunja](https://vikunja.io)**, the open-source, self-hosted task
manager by [kolaente](https://github.com/kolaente) and the
[Vikunja contributors](https://github.com/go-vikunja/vikunja/graphs/contributors). Everything you
like about the task side (projects, list/table/kanban views, saved filters, reminders, CalDAV,
the API) is Vikunja's work. Mandarin adds household features on top:

- **Shopping lists:** a ShopShop-style list with categories, shop groups (Aldi + Supabarn) and offline-first sync.
- **Recipes:** web import of exact ingredients and steps, and "add to shopping list".
- **Household mode:** one household per server, every user in it automatically.

See [SPEC.md](SPEC.md) for the full plan and build phases. This is a personal household project.
For a task manager for yourself or your team, please use and
[support upstream Vikunja](https://vikunja.io/support/) directly.

## Running it

The Docker image is built from the `household` branch by GitHub Actions:

```
ghcr.io/<owner>/mandarin:household
```

It is a drop-in replacement for the `vikunja/vikunja` image (same port `3456`, same `/db` and
`/app/vikunja/files` volumes, same `VIKUNJA_*` environment variables), so
[Vikunja's install docs](https://vikunja.io/docs/installing) apply.

## Branches

- `main` mirrors [go-vikunja/vikunja](https://github.com/go-vikunja/vikunja) `main`, untouched.
- `household` is where Mandarin's work lives. Upstream is merged in every few weeks.

Fork-specific code lives in its own folders (e.g. `frontend/src/household/`); the few edits to
Vikunja's own files are marked with `FORK:` comments (`git grep -n "FORK:"`). See
[CLAUDE.md](CLAUDE.md).

## License

Like Vikunja, Mandarin is licensed under the
[GNU Affero General Public License v3.0 or later](LICENSE). The full source is public in this
repository. Original Vikunja code is copyright the Vikunja authors.
