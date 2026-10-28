// Mandarin: plain-text bits of a task for its Kanban card (KanbanCardExtras.vue).
// Descriptions and comments are HTML from Vikunja's editor.

const BLOCKS = 'p, h1, h2, h3, h4, h5, h6, li, blockquote, pre, td'

function parse(html: string): Document {
	return new DOMParser().parseFromString(html, 'text/html')
}

const clean = (text: string | null | undefined) => (text ?? '').replace(/\s+/g, ' ').trim()

/**
 * The description's first line: the text of its first block that has any
 * (a paragraph, heading, list or checklist item), else its first line of
 * plain text. The card cuts it off with an ellipsis when it's too long.
 */
export function firstLine(description: string | null | undefined): string {
	if (!description) return ''
	const doc = parse(description)
	for (const block of doc.body.querySelectorAll(BLOCKS)) {
		// A list item's own text, not its nested list's.
		const own = block.cloneNode(true) as HTMLElement
		own.querySelectorAll('ul, ol').forEach(nested => nested.remove())
		const text = clean(own.textContent)
		if (text) return text
	}
	return clean((doc.body.innerText ?? doc.body.textContent ?? '').split('\n').find(line => line.trim()))
}

/** A comment as plain text, shortened to about `max` characters. */
export function commentText(html: string | null | undefined, max = 160): string {
	if (!html) return ''
	const doc = parse(html)
	const text = clean(doc.body.textContent)
	return text.length > max ? text.slice(0, max - 1).trimEnd() + '…' : text
}
