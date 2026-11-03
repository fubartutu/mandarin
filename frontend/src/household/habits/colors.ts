// The colours a habit can have. Stored without '#' (the API's format);
// '' means the app's mandarin orange.

export const HABIT_COLORS = ['', 'e0457b', '8b5cf6', '3b82f6', '0ea5a4', '22a35a', 'c58a00'] as const

/** The habit's colour as a CSS variable, for the tick button and grid. */
export function colorStyle(color: string): Record<string, string> {
	return {'--habit-color': color ? `#${color}` : 'var(--primary)'}
}
