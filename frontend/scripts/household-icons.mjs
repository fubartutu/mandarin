// Renders Mandarin's logo and app icons from the pictures in
// src/household/assets/source. Run after changing them:
//   cd frontend && node scripts/household-icons.mjs
// Writes src/household/assets/mark.webp (the fruit on transparency, drawn in
// the page) and the icons in public/images/household.
// Uses Playwright's Chromium (a dev dependency) to draw, or the system Edge
// when Playwright's own browser isn't downloaded.
//
// Sources:
// - fruit-checkerboard.webp: the fruit with a grey-and-white checkerboard
//   painted in where the background should be see-through. cutOut() removes it.
// - app-icon.webp: the finished rounded-square app icon (transparent corners).

import {Buffer} from 'node:buffer'
import console from 'node:console'
import {mkdirSync, readFileSync, writeFileSync} from 'node:fs'
import {dirname, join} from 'node:path'
import {fileURLToPath} from 'node:url'
import {chromium} from '@playwright/test'

// Used inside page.evaluate(), which runs in the browser, not in Node.
/* global Image, OffscreenCanvas, btoa */

const frontend = join(dirname(fileURLToPath(import.meta.url)), '..')
const assets = join(frontend, 'src/household/assets')
const out = join(frontend, 'public/images/household')
mkdirSync(out, {recursive: true})

const browser = await chromium.launch().catch(() => chromium.launch({channel: 'msedge'}))

const dataUrl = (bytes, type = 'image/webp') => `data:${type};base64,${Buffer.from(bytes).toString('base64')}`
const source = name => dataUrl(readFileSync(join(assets, 'source', name)))

/**
 * Takes the checkerboard out of the fruit picture, crops to the fruit and
 * returns a square transparent WebP (`size` px) as bytes. Runs in the browser:
 * the background is every dull pixel reachable from the edge: the checkerboard
 * and the soft shadow under the fruit (saturation up to about 0.45, browner
 * close to the fruit, and dim brown right against it), where peel, flesh and
 * leaves are vivid (0.55 and up) and the peel's underside stays bright.
 * The pale cream pith is dull too but bright, so it stays. That hard mask is
 * then blurred and cut again, which rounds off its jagged steps, gives an
 * even anti-aliased outline and pulls it in a pixel past any brown fringe.
 */
async function cutOut(src, size) {
	const page = await browser.newPage()
	const base64 = await page.evaluate(async ({src, size}) => {
		const img = new Image()
		img.src = src
		await img.decode()
		const {width: w, height: h} = img
		const canvas = new OffscreenCanvas(w, h)
		const ctx = canvas.getContext('2d')
		ctx.drawImage(img, 0, 0)
		const image = ctx.getImageData(0, 0, w, h)
		const d = image.data
		const saturation = i => {
			const max = Math.max(d[i], d[i + 1], d[i + 2])
			return max ? 1 - Math.min(d[i], d[i + 1], d[i + 2]) / max : 0
		}
		const isBackground = i => {
			const s = saturation(i)
			const max = Math.max(d[i], d[i + 1], d[i + 2])
			// Grey; or not vivid and not as bright as the pith; or the dim
			// brown shadow touching the fruit (saturation up to about 0.7; the
			// peel's deep shading is redder, 0.9 and up, and dark leaf edges
			// are green).
			return s < 0.08 || (s < 0.5 && max < 232) || (max < 185 && s < 0.78 && d[i] >= d[i + 1])
		}

		// Flood fill from the edges over grey pixels.
		const bg = new Uint8Array(w * h)
		const stack = []
		for (let x = 0; x < w; x++) stack.push(x, (h - 1) * w + x)
		for (let y = 0; y < h; y++) stack.push(y * w, y * w + w - 1)
		while (stack.length) {
			const p = stack.pop()
			if (bg[p] || !isBackground(p * 4)) continue
			bg[p] = 1
			const x = p % w
			if (x > 0) stack.push(p - 1)
			if (x < w - 1) stack.push(p + 1)
			if (p >= w) stack.push(p - w)
			if (p < w * (h - 1)) stack.push(p + w)
		}

		// Specks of shadow left in the background: any patch that isn't the
		// fruit (the biggest patch) and is under 1% of its size goes too.
		const patch = new Int32Array(w * h).fill(-1)
		const sizes = []
		for (let start = 0; start < w * h; start++) {
			if (bg[start] || patch[start] >= 0) continue
			const n = sizes.length
			let count = 0
			const todo = [start]
			patch[start] = n
			while (todo.length) {
				const p = todo.pop()
				count++
				const x = p % w
				for (const q of [x > 0 ? p - 1 : -1, x < w - 1 ? p + 1 : -1, p - w, p + w]) {
					if (q >= 0 && q < w * h && !bg[q] && patch[q] < 0) {
						patch[q] = n
						todo.push(q)
					}
				}
			}
			sizes.push(count)
		}
		const biggest = Math.max(...sizes)
		for (let p = 0; p < w * h; p++) if (!bg[p] && sizes[patch[p]] < biggest / 100) bg[p] = 1

		// Smooth outline: blur the mask (three box blurs ≈ a Gaussian), then
		// map it back to alpha with a soft step a little above the middle.
		let mask = new Float32Array(w * h)
		for (let p = 0; p < w * h; p++) mask[p] = bg[p] ? 0 : 1
		const R = 3
		const boxBlur = (from, horizontal) => {
			const to = new Float32Array(w * h)
			const [n, lines, step, lineStep] = horizontal ? [w, h, 1, w] : [h, w, w, 1]
			for (let line = 0; line < lines; line++) {
				const base = line * lineStep
				let sum = 0
				for (let i = -R; i <= R; i++) sum += from[base + Math.min(n - 1, Math.max(0, i)) * step]
				for (let i = 0; i < n; i++) {
					to[base + i * step] = sum / (2 * R + 1)
					sum += from[base + Math.min(n - 1, i + R + 1) * step] - from[base + Math.max(0, i - R) * step]
				}
			}
			return to
		}
		for (let pass = 0; pass < 3; pass++) mask = boxBlur(boxBlur(mask, true), false)
		const smoothstep = (a, b, x) => {
			const t = Math.min(1, Math.max(0, (x - a) / (b - a)))
			return t * t * (3 - 2 * t)
		}

		let minX = w, minY = h, maxX = 0, maxY = 0
		for (let p = 0; p < w * h; p++) {
			const alpha = Math.round(255 * smoothstep(0.5, 0.85, mask[p]))
			d[p * 4 + 3] = alpha
			if (alpha > 0) {
				const x = p % w, y = (p / w) | 0
				minX = Math.min(minX, x); maxX = Math.max(maxX, x)
				minY = Math.min(minY, y); maxY = Math.max(maxY, y)
			}
		}
		ctx.putImageData(image, 0, 0)

		// Centre the fruit in a square with a little room around it.
		const side = Math.max(maxX - minX, maxY - minY) * 1.04
		const square = new OffscreenCanvas(size, size)
		square.getContext('2d').drawImage(canvas,
			(minX + maxX) / 2 - side / 2, (minY + maxY) / 2 - side / 2, side, side,
			0, 0, size, size)
		const blob = await square.convertToBlob({type: 'image/webp', quality: 0.9})
		const bytes = new Uint8Array(await blob.arrayBuffer())
		let s = ''
		for (const b of bytes) s += String.fromCharCode(b)
		return btoa(s)
	}, {src, size})
	await page.close()
	return Buffer.from(base64, 'base64')
}

/**
 * The page `html` (a body of width×height) as PNG bytes. `opaque` keeps the
 * white page background; otherwise it's transparent.
 */
async function shot(html, width, height = width, opaque = false) {
	const page = await browser.newPage({viewport: {width, height}})
	await page.setContent(`<body style="margin:0;width:${width}px;height:${height}px;position:relative;overflow:hidden">${html}</body>`)
	await page.locator('img').evaluateAll(imgs => Promise.all(imgs.map(img => img.decode())))
	const png = await page.screenshot({omitBackground: !opaque})
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

// The fruit on transparency: drawn in the page (MandarinMark.vue) and used for the favicons.
const fruit = await cutOut(source('fruit-checkerboard.webp'), 512)
writeFileSync(join(assets, 'mark.webp'), fruit)
const fruitImg = (style = 'width:100%;height:100%') => `<img src="${dataUrl(fruit)}" style="display:block;${style}">`

// The orange of the app icon's background, for where it has to be filled in.
const ORANGE = 'linear-gradient(135deg, #e8752a, #f0a050 60%, #f6c37a)'

// Home-screen icons: an opaque square (iOS fills transparency with black; the
// OS rounds the corners itself). The source is already rounded, so it's drawn
// a little larger than the square to crop its light rim, over the same orange
// so its see-through corners don't show.
const appIcon = source('app-icon.webp')
const home = size => shot(`<div style="position:absolute;inset:0;background:${ORANGE}"></div>
	<img src="${appIcon}" style="position:absolute;left:-3%;top:-3%;width:106%;height:106%">`, size, size, true)
write('android-chrome-192x192.png', await home(192))
write('android-chrome-512x512.png', await home(512))
write('apple-touch-icon-180x180.png', await home(180))
// Android crops maskable icons to a circle or squircle, so the fruit sits inside the middle 80%.
write('icon-maskable.png', await shot(`<div style="position:absolute;inset:0;background:${ORANGE};display:grid;place-items:center">
	${fruitImg('width:72%;height:72%')}</div>`, 512, 512, true))
// Browser tab.
const favicons = []
for (const size of [16, 32, 48]) favicons.push({size, data: await shot(fruitImg(), size)})
write('favicon.ico', ico(favicons))
write('favicon-32x32.png', favicons[1].data)
// While a time-tracking timer runs: the same with a red dot, like Vikunja's.
const dot = '<i style="position:absolute;left:0;bottom:0;width:10px;height:10px;border-radius:50%;background:#e62828;border:2px solid #fff;box-sizing:border-box"></i>'
write('favicon-tracking-32x32.png', await shot(fruitImg() + dot, 32))

await browser.close()
console.log('logo and icons written')
