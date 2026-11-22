// iPhone and iPad fixes for Safari.
//
// Safari zooms the page in when a text field with text under 16px gets the
// focus (the Menu's add and servings fields, a new list's name, Vikunja's
// small fields), and it stays zoomed afterwards, so the page looks too big and
// cut off. Adding maximum-scale=1 to the viewport stops that zoom; on iOS it
// doesn't stop pinch-zooming (Safari ignores that part for accessibility).
// Android Chrome would block pinch-zoom, so it's only added on Apple devices.
//
// viewport-fit=cover (every device) makes Safari report the safe-area insets
// for the notch, rounded corners and home indicator; without it they are 0
// and env(safe-area-inset-*) does nothing. safeArea.scss uses them to keep
// buttons such as Save clear of the home indicator.

/** True on iPhone, iPod and iPad (iPadOS reports itself as a Mac with touch). */
export function isAppleTouch(nav: Pick<Navigator, 'userAgent' | 'maxTouchPoints'> = navigator): boolean {
	return /iPhone|iPad|iPod/.test(nav.userAgent) || (/Macintosh/.test(nav.userAgent) && nav.maxTouchPoints > 1)
}

/** The viewport content with maximum-scale=1, kept as is when it's already there. */
export function withMaximumScale(content: string): string {
	if (/maximum-scale/.test(content)) return content
	return content ? `${content},maximum-scale=1` : 'width=device-width,initial-scale=1,maximum-scale=1'
}

/** The viewport content with viewport-fit=cover, kept as is when a viewport-fit is set. */
export function withViewportFitCover(content: string): string {
	if (/viewport-fit/.test(content)) return content
	return content ? `${content},viewport-fit=cover` : 'width=device-width,initial-scale=1,viewport-fit=cover'
}

export function registerIosFixes() {
	const viewport = document.querySelector<HTMLMetaElement>('meta[name="viewport"]')
	if (!viewport) return
	let content = withViewportFitCover(viewport.content)
	if (isAppleTouch()) content = withMaximumScale(content)
	viewport.content = content
}
