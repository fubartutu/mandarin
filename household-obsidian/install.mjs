// Copies the built plugin into an Obsidian vault:
//   node install.mjs [vault folder]   (default: ~/Documents/Obsidian/HOME)
// Then in Obsidian: Settings → Community plugins → enable "Mandarin".
import {copyFileSync, existsSync, mkdirSync} from 'node:fs'
import {homedir} from 'node:os'
import {join} from 'node:path'

const vault = process.argv[2] ?? join(homedir(), 'Documents/Obsidian/HOME')
if (!existsSync(join(vault, '.obsidian'))) {
	console.error(`No Obsidian vault at ${vault}`)
	process.exit(1)
}
const target = join(vault, '.obsidian/plugins/mandarin')
mkdirSync(target, {recursive: true})
for (const file of ['manifest.json', 'main.js', 'styles.css']) {
	copyFileSync(file, join(target, file))
}
console.log(`Installed into ${target}`)
