// Renders Mandarin's app icons from the SVGs in src/household/assets into
// public/images/household. Run after changing the logo:
//   cd frontend && node scripts/household-icons.mjs
// Uses Playwright's Chromium (a dev dependency) to draw the SVGs.

import {Buffer} from 'node:buffer'
import console from 'node:console'
import {mkdirSync, readFileSync, writeFileSync} from 'node:fs'
import {dirname, join} from 'node:path'
import {fileURLToPath} from 'node:url'
import {chromium} from '@playwright/test'

const frontend = join(dirname(fileURLToPath(import.meta.url)), '..')
const here = join(frontend, 'src/household/assets')
const out = join(frontend, 'public/images/household')
mkdirSync(out, {recursive: true})

const browser = await chromium.launch()

/**
 * One SVG drawn at width×height, the artwork at `scale` of the box, as PNG
 * bytes. `overlay` is extra HTML drawn on top.
 */
async function render(svg, width, height = width, scale = 1, overlay = '') {
	const page = await browser.newPage({viewport: {width, height}})
	const data = 'data:image/svg+xml;base64,' + readFileSync(join(here, svg)).toString('base64')
	await page.setContent(`<body style="margin:0;display:grid;place-items:center;width:${width}px;height:${height}px;position:relative">
		<img src="${data}" style="width:${scale * 100}%;height:${scale * 100}%;object-fit:contain">${overlay}</body>`)
	await page.locator('img').evaluate(img => img.decode())
	const png = await page.screenshot({omitBackground: true})
	await page.close()
	return png
}

/** A .ico holding PNG images (allowed since Windows Vista; every browser reads it). */
function ico(pngs) {
	const header = Buffer.alloc(6)
	header.writeUInt16LE(0, 0) // reserved
	header.writeUInt16LE(1, 2) // type: icon
	header.writeUInt16LE(pngs.length, 4)
	let offset = 6 + 16 * pngs.length
	const entries = pngs.map(({size, data}) => {
		const entry = Buffer.alloc(16)
		entry.writeUInt8(size >= 256 ? 0 : size, 0)
		entry.writeUInt8(size >= 256 ? 0 : size, 1)
		entry.writeUInt16LE(1, 4) // colour planes
		entry.writeUInt16LE(32, 6) // bits per pixel
		entry.writeUInt32LE(data.length, 8)
		entry.writeUInt32LE(offset, 12)
		offset += data.length
		return entry
	})
	return Buffer.concat([header, ...entries, ...pngs.map(p => p.data)])
}

const write = (name, data) => writeFileSync(join(out, name), data)

// Home-screen icons: opaque square (iOS fills transparency with black; the OS rounds the corners).
write('android-chrome-192x192.png', await render('app-icon.svg', 192))
write('android-chrome-512x512.png', await render('app-icon.svg', 512))
write('apple-touch-icon-180x180.png', await render('app-icon.svg', 180))
// Android crops maskable icons to a circle or squircle, so the fruit sits inside the middle 80%.
write('icon-maskable.png', await render('app-icon-maskable.svg', 512))
// Browser tab.
const favicons = []
for (const size of [16, 32, 48]) favicons.push({size, data: await render('mark.svg', size)})
write('favicon.ico', ico(favicons))
write('favicon-32x32.png', favicons[1].data)
// While a time-tracking timer runs: the same with a red dot, like Vikunja's.
const dot = '<i style="position:absolute;left:0;bottom:0;width:10px;height:10px;border-radius:50%;background:#e62828;border:2px solid #fff"></i>'
write('favicon-tracking-32x32.png', await render('mark.svg', 32, 32, 1, dot))

await browser.close()
console.log('icons written to', out)
