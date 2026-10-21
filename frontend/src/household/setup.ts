// Side effects for Mandarin's household modules. Importing this file (done
// once from main.ts) registers our icons, translations and a router guard
// without editing Vikunja's Icon.ts, lang/en.json or router.

import {library} from '@fortawesome/fontawesome-svg-core'
import {
	faArrowsRotate,
	faBasketShopping,
	faBookOpen,
	faCamera,
	faChevronDown,
	faChevronRight,
	faCircleCheck,
	faCloud,
	faFileLines,
	faLightbulb,
	faStore,
	faTriangleExclamation,
} from '@fortawesome/free-solid-svg-icons'
import {faCircle} from '@fortawesome/free-regular-svg-icons'

import {i18n} from '@/i18n'
import router from '@/router'
import householdEn from './i18n/en.json'
import './theme.scss' // mandarin-orange colours
import {registerOfflineStart} from './offline'

library.add(
	faArrowsRotate, faBasketShopping, faBookOpen, faCamera, faChevronDown, faChevronRight, faCircleCheck, faCloud,
	faFileLines, faLightbulb, faStore, faTriangleExclamation, faCircle,
)

// Other languages fall back to English for these keys (fallbackLocale).
i18n.global.mergeLocaleMessage('en', householdEn)

// Without signal, open on the shopping list rather than the task overview.
registerOfflineStart(router)
