// Copies the built plugin into an Obsidian vault:
//   node install.mjs [vault folder]   (default: the vault called HOME that Obsidian knows about)
// Then in Obsidian: Settings → Community plugins → enable "Mandarin".
import {copyFileSync, existsSync, mkdirSync, readFileSync} from 'node:fs'
import {homedir} from 'node:os'
import {basename, join} from 'node:path'

// Obsidian lists its vaults in obsidian.json, in a different place on each system.
function homeVault() {
	const config = {
		win32: join(process.env.APPDATA ?? '', 'obsidian'),
		darwin: join(homedir(), 'Library/Application Support/obsidian'),
	}[process.platform] ?? join(homedir(), '.config/obsidian')
	try {
		const {vaults} = JSON.parse(readFileSync(join(config, 'obsidian.json'), 'utf8'))
		const found = Object.values(vaults).find(v => basename(v.path) === 'HOME')
		if (found) return found.path
	} catch {
		// Obsidian not set up here: fall back below.
	}
	return join(homedir(), 'Documents/Obsidian/HOME')
}

const vault = process.argv[2] ?? homeVault()
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
