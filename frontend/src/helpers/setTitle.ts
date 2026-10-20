import {APP_NAME} from '@/household/branding' // FORK: app name in page titles
export function setTitle(title : undefined | string) {
	document.title = (typeof title === 'undefined' || title === '')
		? APP_NAME
		: `${title} | ${APP_NAME}`
}
