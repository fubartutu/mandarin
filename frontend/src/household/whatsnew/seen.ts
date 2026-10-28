// Which changelog entries this device has seen, so Home can point at new ones.

import {computed} from 'vue'
import {useStorage} from '@vueuse/core'

import {changelog, unseenEntries} from './changelog'

export function useWhatsNew() {
	// The newest entry date seen on the What's new page ('' = never opened).
	const lastSeen = useStorage('mandarin.whatsnew.seen', '')
	const unseen = computed(() => unseenEntries(changelog, lastSeen.value))

	function markSeen() {
		const newest = changelog[0]?.date
		if (newest && newest !== lastSeen.value) lastSeen.value = newest
	}

	return {unseen, markSeen}
}
