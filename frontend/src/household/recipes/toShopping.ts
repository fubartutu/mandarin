// Turning a recipe's ingredient line into a shopping item: what you'd write
// on a list, not what the recipe says. "400 g can chickpeas, drained" becomes
// "Chickpeas", 400 "g can", in Pantry.
//
// Names the household already has in its catalogue keep their catalogue name
// and category (the household put them there). Only new names get a guessed
// category: from catalogue entries that share their words ("chickpeas" and
// "Canned chickpeas"), else from a list of keywords. The guess is sent as the
// item's category, and the server then remembers it for that name.

import type {CatalogEntry, ShoppingCategoryResponse} from '../shopping/queries'
import type {ItemInput} from '../shopping/adding'
import type {IngredientLine} from './queries'
import {scaledQuantity} from './scaling'

// --- Reading a line the server couldn't split -----------------------------

const VULGAR: Record<string, number> = {'½': .5, '⅓': 1 / 3, '⅔': 2 / 3, '¼': .25, '¾': .75, '⅛': .125}

// Units as the server's simple parser stores them (pkg/household/recipes/ingredients.go).
const UNITS: Record<string, string> = {
	g: 'g', gram: 'g', grams: 'g', kg: 'kg', kilo: 'kg', kilos: 'kg', kilogram: 'kg', kilograms: 'kg',
	mg: 'mg', ml: 'mL', millilitre: 'mL', millilitres: 'mL', milliliter: 'mL', milliliters: 'mL',
	l: 'L', litre: 'L', litres: 'L', liter: 'L', liters: 'L',
	tsp: 'tsp', teaspoon: 'tsp', teaspoons: 'tsp', tbsp: 'tbsp', tablespoon: 'tbsp', tablespoons: 'tbsp',
	cup: 'cup', cups: 'cups', oz: 'oz', lb: 'lb', lbs: 'lb',
	can: 'can', cans: 'cans', tin: 'tin', tins: 'tins', packet: 'packet', packets: 'packets',
	pack: 'pack', packs: 'packs', bunch: 'bunch', bunches: 'bunches', clove: 'clove', cloves: 'cloves',
	pinch: 'pinch', handful: 'handful', slice: 'slice', slices: 'slices', sprig: 'sprig', sprigs: 'sprigs',
}

// "2 1/2", "1/2", "1.5" or "1,5", then an optional "½", then the rest.
const LEADING_AMOUNT = /^(\d+\s+\d+\/\d+|\d+\/\d+|\d+(?:[.,]\d+)?)?\s*([½⅓⅔¼¾⅛])?\s*(.*)$/

function toNumber(raw: string | undefined): number {
	if (!raw) return 0
	const [whole, fraction] = raw.includes(' ') ? raw.split(/\s+/) : ['', raw]
	const part = (s: string) => {
		if (!s.includes('/')) return Number(s.replace(',', '.')) || 0
		const [top, bottom] = s.split('/').map(Number)
		return bottom ? top / bottom : 0
	}
	return (whole ? part(whole) : 0) + part(fraction)
}

/** The line split the way the server's simple parser would (for lines saved unsplit). */
export function splitLine(text: string): Pick<IngredientLine, 'quantity' | 'unit' | 'name'> {
	const m = LEADING_AMOUNT.exec(text.trim())
	const quantity = m ? toNumber(m[1]) + (VULGAR[m[2] ?? ''] ?? 0) : 0
	let rest = m ? m[3] : text.trim()
	let unit = ''
	if (quantity > 0) {
		// "200g butter" and "200 g butter" both have a unit.
		const [word, ...after] = rest.split(/\s+/)
		const known = UNITS[word.replace(/\.$/, '').toLowerCase()]
		if (known && after.length > 0) {
			unit = known
			rest = after.join(' ')
		}
		rest = rest.replace(/^of\s+/i, '')
	}
	return {quantity, unit, name: rest.trim() || text.trim()}
}

// --- Tidying the name ------------------------------------------------------

// What a thing is bought in: "can chickpeas" → unit "can", name "chickpeas".
// Also "x 400g can …" and "400g can …" (the size goes with the container).
const CONTAINER = /^(?:(x|×)\s*)?(?:(\d+(?:[.,]\d+)?)\s*(g|kg|ml|l)\s+)?(cans?|tins?|jars?|packets?|packs?|bottles?|bags?|boxes?|tubs?|blocks?|sachets?|punnets?|bunch|bunches)\s+(?:of\s+)?(.+)$/i
const CANNED = /^(cans?|tins?)$/i

// Words that describe how much or how it's prepared, dropped from the front…
const LEADING_WORDS = new Set([
	'large', 'small', 'medium', 'big', 'extra', 'virgin', 'heaped', 'level', 'generous', 'good', 'quality',
	'fresh', 'freshly', 'finely', 'roughly', 'thinly', 'coarsely', 'chopped', 'diced', 'sliced', 'minced',
	'crushed', 'grated', 'shredded', 'peeled', 'cooked', 'uncooked', 'ripe', 'boneless', 'skinless', 'raw',
	'dried', 'whole', 'organic', 'free-range', 'about', 'approx', 'approximately',
])
// …and from the end ("chickpeas drained and rinsed").
const TRAILING_WORDS = new Set([
	'drained', 'rinsed', 'chopped', 'diced', 'sliced', 'minced', 'crushed', 'grated', 'peeled', 'halved',
	'quartered', 'trimmed', 'softened', 'melted', 'beaten', 'sifted', 'washed', 'optional', 'and', 'or',
	'finely', 'roughly', 'thinly',
])
const TRAILING_PHRASES = /\s+(?:to serve|to taste|for serving|for garnish|to garnish|or more|plus extra|plus more)\b.*$/i

const words = (s: string) => s.split(/\s+/).filter(Boolean)
const capitalise = (s: string) => s.charAt(0).toUpperCase() + s.slice(1)

/** The name with the note, brackets and trailing preparation words taken off. */
function baseName(name: string): string {
	let s = name.split(',')[0]
	s = s.replace(/\([^)]*\)/g, ' ').replace(/[()]/g, ' ').replace(TRAILING_PHRASES, '')
	const list = words(s)
	while (list.length > 1 && TRAILING_WORDS.has(list[list.length - 1].toLowerCase())) list.pop()
	return list.join(' ')
}

/** Shorter and shorter names, dropping describing words from the front. */
function candidates(name: string): string[] {
	const list = words(name)
	const out = [list.join(' ')]
	while (list.length > 1 && LEADING_WORDS.has(list[0].toLowerCase())) {
		list.shift()
		out.push(list.join(' '))
	}
	return out
}

// --- Finding it in the catalogue -------------------------------------------

/** One word without a plural ending: "tomatoes" → "tomato", "berries" → "berry". */
function singular(word: string): string {
	const w = word.toLowerCase()
	if (w.length > 4 && w.endsWith('ies')) return w.slice(0, -3) + 'y'
	if (w.length > 4 && /(oes|ches|shes|xes|sses)$/.test(w)) return w.slice(0, -2)
	if (w.length > 3 && w.endsWith('s') && !w.endsWith('ss')) return w.slice(0, -1)
	return w
}

const key = (name: string) => words(name).map(singular).join(' ')

function findByKey(catalog: CatalogEntry[], name: string): CatalogEntry | undefined {
	const k = key(name)
	return catalog.find(e => e.name.toLowerCase() === name.toLowerCase()) ??
		catalog.find(e => key(e.name) === k)
}

/** A category from catalogue entries that share the name's words, when they agree on one. */
function categoryFromSimilar(catalog: CatalogEntry[], name: string): number {
	const mine = words(name).map(singular)
	const similar = catalog.filter(e => {
		if (!e.category_id) return false
		const theirs = words(e.name).map(singular)
		return theirs.every(w => mine.includes(w)) || mine.every(w => theirs.includes(w))
	})
	const pick = (entries: CatalogEntry[]) => {
		const ids = new Set(entries.map(e => e.category_id))
		return ids.size === 1 ? [...ids][0] : 0
	}
	// "chicken" is in several aisles (Whole chicken, Chicken stock); the ones
	// ending in the same word ("Whole chicken", "Sliced chicken") may agree.
	const last = mine[mine.length - 1]
	return pick(similar) || pick(similar.filter(e => singular(words(e.name).pop() ?? '') === last))
}

// Keywords for names the catalogue has never seen, by default category title.
// A household that renamed a category simply gets no guess for it.
const KEYWORDS: [string, string[]][] = [
	['Frozen', ['frozen']],
	['Meat & Seafood', [
		'chicken', 'beef', 'pork', 'lamb', 'veal', 'mince', 'steak', 'sausage', 'bacon', 'pancetta', 'chorizo',
		'prosciutto', 'fish', 'salmon', 'tuna', 'prawn', 'shrimp', 'barramundi', 'snapper', 'squid', 'mussel',
		'turkey', 'duck', 'brisket', 'ribs', 'fillet', 'thigh', 'breast', 'drumstick', 'cutlet',
	]],
	['Dairy & Eggs', [
		'milk', 'cream', 'butter', 'cheese', 'cheddar', 'parmesan', 'mozzarella', 'ricotta', 'mascarpone',
		'halloumi', 'feta', 'yoghurt', 'yogurt', 'egg', 'buttermilk', 'creme', 'crème',
	]],
	['Bakery', ['bread', 'roll', 'bun', 'baguette', 'tortilla', 'wrap', 'pita', 'naan', 'brioche', 'sourdough']],
	['Fruit & Veg', [
		'onion', 'shallot', 'garlic', 'ginger', 'carrot', 'potato', 'tomato', 'capsicum', 'pepper', 'chilli',
		'chili', 'zucchini', 'eggplant', 'broccoli', 'broccolini', 'cauliflower', 'cabbage', 'lettuce', 'spinach',
		'kale', 'rocket', 'celery', 'cucumber', 'mushroom', 'pumpkin', 'squash', 'leek', 'corn', 'pea', 'bean',
		'asparagus', 'beetroot', 'radish', 'fennel', 'avocado', 'lemon', 'lime', 'orange', 'apple', 'pear',
		'banana', 'berry', 'strawberry', 'blueberry', 'raspberry', 'mango', 'grape', 'herb', 'parsley',
		'coriander', 'cilantro', 'basil', 'mint', 'dill', 'chive', 'thyme', 'rosemary', 'sage', 'lemongrass',
	]],
	['Pantry', [
		'chickpea', 'lentil', 'rice', 'pasta', 'spaghetti', 'penne', 'noodle', 'flour', 'sugar', 'oil', 'vinegar',
		'stock', 'broth', 'sauce', 'paste', 'salt', 'cumin', 'paprika', 'turmeric', 'cinnamon', 'nutmeg',
		'oregano', 'spice', 'seasoning', 'powder', 'honey', 'syrup', 'mustard', 'mayonnaise', 'ketchup',
		'breadcrumb', 'panko', 'couscous', 'quinoa', 'polenta', 'cornflour', 'cornstarch', 'yeast', 'cocoa',
		'vanilla', 'essence', 'extract', 'nut', 'almond', 'cashew', 'peanut', 'walnut', 'sesame', 'seed',
		'coconut', 'oat', 'raisin', 'sultana', 'date', 'bicarb', 'gelatine', 'tahini', 'miso', 'curry',
	]],
]

/** The category id a keyword in the name points to; canned things are Pantry. */
function categoryFromKeywords(name: string, canned: boolean, categories: ShoppingCategoryResponse[]): number {
	const id = (title: string) => categories.find(c => c.title.toLowerCase() === title.toLowerCase())?.id ?? 0
	if (canned) return id('Pantry')
	const mine = words(name).map(singular)
	for (const [title, keys] of KEYWORDS) {
		if (keys.some(k => mine.includes(k))) return id(title)
	}
	return 0
}

// --- Putting it together ---------------------------------------------------

/** The shopping item for one ingredient line at a recipe scale factor. */
export function ingredientToItem(
	line: IngredientLine,
	factor: number,
	catalog: CatalogEntry[],
	categories: ShoppingCategoryResponse[],
): ItemInput {
	// Lines saved before they were split (or that the parser gave up on)
	// have the whole text as their name: split them here.
	const unsplit = !line.name.trim() || line.name.trim() === line.text.trim()
	const parts = unsplit ? {...line, ...splitLine(line.text)} : line
	let quantity = parts.quantity > 0 ? scaledQuantity(parts, factor) : 0
	let unit = parts.quantity > 0 ? parts.unit : ''

	// "can chickpeas" → the can goes with the amount: 400 "g can".
	let name = baseName(parts.name)
	let canned = CANNED.test(unit)
	const container = CONTAINER.exec(name)
	if (container) {
		const [, times, size, sizeUnit, what, rest] = container
		const amount = [times ? '×' : '', size ? `${size}${sizeUnit.toLowerCase()}` : '', what.toLowerCase()]
			.filter(Boolean).join(' ')
		if (quantity === 0) quantity = 1
		unit = [unit, amount].filter(Boolean).join(' ')
		name = rest
		canned = canned || CANNED.test(what)
	}

	const names = candidates(name)
	const shortest = names[names.length - 1]
	if (canned) {
		// A can of tomatoes isn't the fresh tomatoes in Fruit & Veg: use the
		// household's "Canned tomatoes" when there is one.
		const fresh = names.map(n => findByKey(catalog, n)).find(Boolean)
		const tinned = names.flatMap(n => [`canned ${n}`, `tinned ${n}`]).map(n => findByKey(catalog, n)).find(Boolean)
		if (tinned && fresh && fresh.category_id !== tinned.category_id) {
			return {name: tinned.name, quantity, unit, category_id: tinned.category_id}
		}
	}
	const guess = (n: string) => canned
		? categoryFromKeywords(n, true, categories) || categoryFromSimilar(catalog, n)
		: categoryFromSimilar(catalog, n) || categoryFromKeywords(n, false, categories)
	for (const candidate of names) {
		const known = findByKey(catalog, candidate)
		// Its catalogue name, so it lands in the household's own category
		// (a guess only for entries that never got one).
		if (known) return {name: known.name, quantity, unit, category_id: known.category_id || guess(known.name)}
	}
	return {name: capitalise(shortest || line.text.trim()), quantity, unit, category_id: guess(shortest)}
}
