// Side effects for Mandarin's household modules. Importing this file (done
// once from main.ts) registers our icons and translations without editing
// Vikunja's Icon.ts or lang/en.json.

import {library} from '@fortawesome/fontawesome-svg-core'
import {
	faBasketShopping,
	faCamera,
	faChevronDown,
	faChevronRight,
	faCircleCheck,
	faStore,
} from '@fortawesome/free-solid-svg-icons'
import {faCircle} from '@fortawesome/free-regular-svg-icons'

import {i18n} from '@/i18n'
import householdEn from './i18n/en.json'

library.add(faBasketShopping, faCamera, faChevronDown, faChevronRight, faCircleCheck, faStore, faCircle)

// Other languages fall back to English for these keys (fallbackLocale).
i18n.global.mergeLocaleMessage('en', householdEn)
