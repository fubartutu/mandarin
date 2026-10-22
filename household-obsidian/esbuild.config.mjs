// Bundles src/main.ts into main.js, the one file Obsidian loads.
import {build} from 'esbuild'

await build({
	entryPoints: ['src/main.ts'],
	bundle: true,
	format: 'cjs', // Obsidian loads plugins as CommonJS
	target: 'es2022',
	platform: 'browser',
	// Provided by Obsidian at runtime.
	external: ['obsidian', 'electron', '@codemirror/*', '@lezer/*'],
	outfile: 'main.js',
	logLevel: 'info',
})
