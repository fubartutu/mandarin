import {describe, expect, it} from 'vitest'

import {taskRemindersFromSettings} from '@/helpers/userSettings'

// Upstream bug fixed in the fork (FORK line in helpers/userSettings.ts):
// Settings → "Default reminders for quick add" showed saved reminders as
// "Add a reminder…" because they came back without relative_to.
describe('quick-add default reminders', () => {
	it('come back relative to the due date, so they can be shown', () => {
		expect(taskRemindersFromSettings([{relative_period: -86400}])).toEqual([
			{relative_period: -86400, relative_to: 'due_date'},
		])
	})
})
