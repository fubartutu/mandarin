// What changed in Mandarin, in plain words for the household, shown on the
// What's new page (/whats-new). Newest first, one entry per day something
// shipped. Add a line here with every change people would notice; leave out
// refactors, docs and CI. Plain English only (no i18n): it's our own app.

export type ChangeKind = 'new' | 'improved' | 'fixed'

export interface Change {
	kind: ChangeKind
	text: string
}

export interface ChangelogEntry {
	/** Local date, YYYY-MM-DD. */
	date: string
	changes: Change[]
}

export const changelog: ChangelogEntry[] = [
	{
		date: '2026-10-29',
		changes: [
			{kind: 'new', text: 'This What\'s new page, in the menu under your name. Home shows a note when there\'s something new.'},
		],
	},
	{
		date: '2026-10-28',
		changes: [
			{kind: 'new', text: 'Kanban cards show a coloured stripe for urgent tasks (high priority or due soon).'},
			{kind: 'new', text: 'Kanban cards and task rows show the first line of the description under the title.'},
			{kind: 'new', text: 'Hover over a Kanban card for a moment to see its latest comments.'},
			{kind: 'improved', text: 'Each project remembers its view (List, Kanban, …) on every device, even after signing out.'},
			{kind: 'improved', text: 'The Menu shows the next 7 days from today.'},
			{kind: 'fixed', text: 'iPhones no longer zoom in when you tap a small field.'},
			{kind: 'fixed', text: 'The top bar no longer shows an old project\'s name on Shopping, Recipes and Menu.'},
		],
	},
	{
		date: '2026-10-27',
		changes: [
			{kind: 'new', text: 'The due date button on a task offers Today, Tomorrow, Weekend and Next week in one tap.'},
			{kind: 'new', text: 'Skip on a repeating task moves it to the next time without marking it done.'},
			{kind: 'new', text: 'Repeat tasks on calendar rules: every Monday and Thursday, every weekday, the 2nd Tuesday, the 15th of each month.'},
			{kind: 'new', text: 'A repeat can end on a date or after a number of times.'},
			{kind: 'new', text: 'Quick add understands "every mon, thu", "every! 3 days" (counted from when it\'s done), "starting nov 1" and p1 to p4.'},
			{kind: 'fixed', text: 'Kanban\'s "Create a bucket" lines up with the bucket titles.'},
		],
	},
	{
		date: '2026-10-24',
		changes: [
			{kind: 'new', text: 'Menu: plan the week\'s dinners from recipes or notes, and put the ingredients on a shopping list.'},
			{kind: 'new', text: 'Cooking mode: a recipe step by step on a full screen, with timers from the steps and the screen kept on.'},
		],
	},
	{
		date: '2026-10-23',
		changes: [
			{kind: 'new', text: 'Home page with this week\'s menu, the shopping lists, recipes and today\'s tasks.'},
			{kind: 'new', text: 'Pick the page Mandarin opens on in Settings → General.'},
			{kind: 'new', text: 'Dictate tasks and shopping items with the microphone button ("two litres of milk").'},
			{kind: 'improved', text: 'Teams are hidden: there\'s one household.'},
		],
	},
	{
		date: '2026-10-22',
		changes: [
			{kind: 'new', text: 'Each shopping item can belong to a shop, and each shop has its own aisle order.'},
			{kind: 'new', text: 'Push notifications for reminders, overdue tasks and shopping lists (Settings → Notifications).'},
			{kind: 'new', text: 'Obsidian plugin for tasks and recipes.'},
			{kind: 'new', text: 'Tasks you\'ve opened can still be read without signal.'},
			{kind: 'new', text: 'Nightly backups of everything.'},
			{kind: 'improved', text: 'Mandarin orange instead of Vikunja blue.'},
			{kind: 'improved', text: 'Updated to Vikunja 2.7.1.'},
			{kind: 'fixed', text: 'The logo sometimes vanished from the top bar.'},
		],
	},
	{
		date: '2026-10-21',
		changes: [
			{kind: 'new', text: 'Recipe box: import recipes from websites and add their ingredients to a shopping list.'},
			{kind: 'new', text: 'Import printed recipes scanned into Paperless, with AI to read them.'},
			{kind: 'new', text: 'The shopping list works offline on phones and catches up when signal returns. The same item added on two phones becomes one.'},
			{kind: 'new', text: 'Mandarin\'s own logo, icons and name throughout.'},
			{kind: 'improved', text: 'Amounts typed after the item ("flour 1kg") are understood.'},
		],
	},
	{
		date: '2026-10-20',
		changes: [
			{kind: 'new', text: 'Shopping lists with categories, shops, suggestions as you type, "not stocked" marks and item photos.'},
		],
	},
]

/** Entries newer than the date someone last saw, for the Home notice. */
export function unseenEntries(entries: readonly ChangelogEntry[], lastSeen: string): ChangelogEntry[] {
	// YYYY-MM-DD strings sort the same as the dates they stand for.
	return entries.filter(entry => entry.date > lastSeen)
}

/** "2026-10-27" as a local date (new Date('2026-10-27') would be midnight UTC). */
export function entryDate(date: string): Date {
	const [year, month, day] = date.split('-').map(Number)
	return new Date(year, month - 1, day)
}
