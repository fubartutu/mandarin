// Times mentioned in a recipe step ("simmer 20 minutes"), offered as timers
// in cooking mode.

export interface DetectedTimer {
	seconds: number
	label: string // the matched text as written, e.g. "20–25 minutes"
	index: number // where it starts in the step text
}

// Longest spellings first, so "mins" isn't read as "min" plus a stray "s".
// The lookahead stops a unit running into a word: "2 heaped" isn't 2 hours.
const UNIT = String.raw`(hours?|hrs?|h|minutes?|mins?|seconds?|secs?|s)(?![a-z])`

const VULGAR: Record<string, number> = {'¼': 1 / 4, '⅓': 1 / 3, '½': 1 / 2, '⅔': 2 / 3, '¾': 3 / 4}
const WORDS: Record<string, number> = {
	a: 1, an: 1, one: 1, two: 2, three: 3, four: 4, five: 5, six: 6, seven: 7, eight: 8, nine: 9,
	ten: 10, eleven: 11, twelve: 12, fifteen: 15, twenty: 20, thirty: 30, forty: 40, sixty: 60,
}
// "half an hour", "a half-hour"; then the words above, "forty-five" before "forty".
const WORD_NUMBER = String.raw`\b(?:half\s+an?|a\s+half|half|forty[\s-]five|an?|one|two|three|four|five|six|seven|eight|nine|ten|eleven|twelve|fifteen|twenty|thirty|forty|sixty)\b`
// "1½", "1 1/2", "1/2", "1.5", "2,5", "¾" or plain "20". Mixed numbers first,
// so "1 1/2" isn't read as just the "1/2".
const DIGIT_NUMBER = String.raw`\d+\s*[¼⅓½⅔¾]|\d+\s+\d+\/\d+|\d+\/\d+|\d+(?:[.,]\d+)?|[¼⅓½⅔¾]`
const NUMBER = `${DIGIT_NUMBER}|${WORD_NUMBER}`

// A range ("20–25 minutes") counts from its lower bound, the first number.
const RANGE_TO = String.raw`(?:\s*[-–—]\s*|\s+to\s+)(?:${NUMBER})`
// Number and unit: "20 minutes", "20min", "a 20-minute walk".
const GAP = String.raw`(?:\s*|-)`

// The `g` flag lets exec() walk the text from lastIndex; `y` (sticky) only
// matches exactly at lastIndex, for the parts that must follow straight on.
const FIRST = new RegExp(String.raw`(?<![\d.,/])(${NUMBER})(?:${RANGE_TO})?${GAP}${UNIT}`, 'gi')
// The rest of a compound: "1 hour 30 minutes", "1 hr and 15 min".
const NEXT = new RegExp(String.raw`(?:\s*,\s*|\s+and\s+|\s*)(${NUMBER})${GAP}${UNIT}`, 'iy')
// "an hour and a half"
const AND_A_HALF = /\s+and\s+a\s+half\b/iy
// "2h30": minutes written straight after the h, with no unit of their own.
const BARE_MINUTES = /(\d{1,2})(?![\d.,/°%a-z])/iy

function parseNumber(text: string): number {
	const lower = text.toLowerCase().trim()
	if (lower.includes('half')) return 0.5
	if (/^forty[\s-]five$/.test(lower)) return 45
	if (lower in WORDS) return WORDS[lower]
	let total = 0
	for (const part of lower.split(/\s+|(?=[¼⅓½⅔¾])/)) {
		if (part in VULGAR) total += VULGAR[part]
		else if (part.includes('/')) {
			const [top, bottom] = part.split('/').map(Number)
			total += top / bottom
		} else total += Number(part.replace(',', '.'))
	}
	return total
}

function unitSeconds(unit: string): number {
	switch (unit.toLowerCase()[0]) {
		case 'h': return 3600
		case 'm': return 60
		default: return 1
	}
}

// Written-out numbers only go with written-out units: "a s" or "an h" is
// noise, and "a second egg" is an ordinal.
function isWordNoise(number: string, unit: string): boolean {
	if (!/^[a-z]/i.test(number)) return false
	return unit.length === 1 || unit.toLowerCase() === 'second'
}

/** Every time in the text, in order. Repeats of the same length are left out. */
export function findTimers(text: string): DetectedTimer[] {
	const found: DetectedTimer[] = []
	FIRST.lastIndex = 0
	for (let match = FIRST.exec(text); match; match = FIRST.exec(text)) {
		if (isWordNoise(match[1], match[2])) {
			// Try again from the next character: "a second 5 minutes" still has a time.
			FIRST.lastIndex = match.index + 1
			continue
		}
		let unit = unitSeconds(match[2])
		let seconds = parseNumber(match[1]) * unit
		let end = match.index + match[0].length

		AND_A_HALF.lastIndex = end
		if (AND_A_HALF.test(text)) {
			seconds += unit / 2
			end = AND_A_HALF.lastIndex
		}

		// Each further part must be a smaller unit: hours, then minutes, then seconds.
		for (;;) {
			NEXT.lastIndex = end
			const next = NEXT.exec(text)
			if (next && unitSeconds(next[2]) < unit && !isWordNoise(next[1], next[2])) {
				unit = unitSeconds(next[2])
				seconds += parseNumber(next[1]) * unit
				end = NEXT.lastIndex
				continue
			}
			BARE_MINUTES.lastIndex = end
			const bare = unit === 3600 && /h$/i.test(text.slice(0, end)) && BARE_MINUTES.exec(text)
			if (bare) {
				unit = 60
				seconds += Number(bare[1]) * 60
				end = BARE_MINUTES.lastIndex
				continue
			}
			break
		}

		FIRST.lastIndex = end
		seconds = Math.round(seconds)
		if (seconds > 0 && !found.some(timer => timer.seconds === seconds)) {
			found.push({seconds, label: text.slice(match.index, end), index: match.index})
		}
	}
	return found
}

/** A countdown as mm:ss, or h:mm:ss from an hour up. */
export function formatRemaining(seconds: number): string {
	const total = Math.max(0, Math.floor(seconds))
	const pad = (n: number) => String(n).padStart(2, '0')
	const h = Math.floor(total / 3600)
	const m = Math.floor(total / 60) % 60
	const s = total % 60
	return h > 0 ? `${h}:${pad(m)}:${pad(s)}` : `${pad(m)}:${pad(s)}`
}
