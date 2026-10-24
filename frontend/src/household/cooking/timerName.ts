import {useI18n} from 'vue-i18n'

import type {CookingTimer} from './useCookingTimers'

/** How the banners and the tray name a timer: its step and the time as written. */
export function useTimerName() {
	const {t} = useI18n({useScope: 'global'})
	return (timer: CookingTimer) => t('household.cooking.timerName', {
		number: timer.stepNumber,
		label: timer.label,
	})
}
