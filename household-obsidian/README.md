# Mandarin for Obsidian

A private Obsidian plugin for Mandarin: your tasks in Obsidian, note lines turned into tasks, and
recipes saved as notes. It's built here and copied into the vault; it isn't published.

## Install or update

```bash
cd household-obsidian
npm install          # once
npm run build        # type-checks and writes main.js
npm test             # unit tests (add MANDARIN_TEST_URL/MANDARIN_TEST_TOKEN to also test against a server)
node install.mjs     # copies it into ~/Documents/Obsidian/HOME/.obsidian/plugins/mandarin
```

Then in Obsidian: **Settings → Community plugins → Mandarin → enable**. (Restricted mode must be off.)
After an update, toggle the plugin off and on, or restart Obsidian.

## Setting it up

1. **API token.** In Mandarin, go to **Settings → API tokens → Create**, name it "Obsidian", and tick:
   - projects: *read all*
   - tasks: *read all*, *create*, *update*
   - recipes: *read all*, *read one*, *images*
2. **Cloudflare service token** (Mandarin is behind Zero Trust, which an app can't log in to):
   1. Go to Zero Trust → **Access → Service credentials → Service Tokens → Create**, named "Obsidian".
      Copy the **Client ID** and **Client Secret**: the secret is only shown once.
   2. Open Mandarin's Access application → **Policies → Add a policy**: action **Service Auth**,
      include **Service Token: Obsidian**. Save.
3. In Obsidian, **Settings → Mandarin**: fill in the address, the API token and both service-token
   values. Press **Test**, then pick the **Project for new tasks**.

## Using it

- **Tasks sidebar:** the ✓ ribbon icon (or the command *Mandarin: Open tasks*). It shows overdue,
  today and the next 7 days. Tick to complete; click a title to open it in Mandarin.
- **Tasks in a note:** a code block:

  ````
  ```mandarin-tasks
  title: Car
  project: Car          # a project's name or id
  filter: due_date < now+30d   # optional, Vikunja filter syntax
  limit: 10
  show-done: false
  ```
  ````

- **Note line → task:** right-click a line → *Create Mandarin task*, or the command *Mandarin: Create
  task from this line*. A due date written as `📅 2026-11-03` or `due:: 2026-11-03` becomes the task's
  due date (at the time set in settings). The task links back to the note, and the line gets a
  link to the task.
- **Recipes as notes:** the command *Mandarin: Save a recipe as a note*. Pick a recipe; it's saved to
  the recipe folder with its picture in `attachments/`, the source and times in the note's
  properties, and `#recipe` plus its tags.
