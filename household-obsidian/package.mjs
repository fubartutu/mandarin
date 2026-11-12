// Zips the built plugin into release/mandarin-obsidian.zip, for installing on another computer:
// unzip it into <vault>/.obsidian/plugins/ (it holds a "mandarin" folder).
// Run with `npm run package` (builds first). Pure Node, so it works on Windows without a zip tool.
import {mkdirSync, readFileSync, writeFileSync} from 'node:fs'
import {crc32, deflateRawSync} from 'node:zlib'

const files = ['manifest.json', 'main.js', 'styles.css']
const locals = []
const centrals = []
let offset = 0

for (const file of files) {
	const data = readFileSync(file)
	const packed = deflateRawSync(data)
	const name = Buffer.from(`mandarin/${file}`)
	const crc = crc32(data)

	// Local file header (zip spec 4.3.7). A fixed date keeps the zip identical between builds.
	const local = Buffer.alloc(30)
	local.writeUInt32LE(0x04034b50, 0)
	local.writeUInt16LE(20, 4) // version needed
	local.writeUInt16LE(8, 8) // deflate
	local.writeUInt16LE(0x21, 12) // date: 1980-01-01
	local.writeUInt32LE(crc, 14)
	local.writeUInt32LE(packed.length, 18)
	local.writeUInt32LE(data.length, 22)
	local.writeUInt16LE(name.length, 26)
	locals.push(local, name, packed)

	// Central directory entry (4.3.12): the same facts plus where the local header starts.
	const central = Buffer.alloc(46)
	central.writeUInt32LE(0x02014b50, 0)
	central.writeUInt16LE(20, 4) // version made by
	central.writeUInt16LE(20, 6)
	central.writeUInt16LE(8, 10)
	central.writeUInt16LE(0x21, 14)
	central.writeUInt32LE(crc, 16)
	central.writeUInt32LE(packed.length, 20)
	central.writeUInt32LE(data.length, 24)
	central.writeUInt16LE(name.length, 28)
	central.writeUInt32LE(offset, 42)
	centrals.push(central, name)

	offset += local.length + name.length + packed.length
}

const directory = Buffer.concat(centrals)
const end = Buffer.alloc(22) // end of central directory (4.3.16)
end.writeUInt32LE(0x06054b50, 0)
end.writeUInt16LE(files.length, 8)
end.writeUInt16LE(files.length, 10)
end.writeUInt32LE(directory.length, 12)
end.writeUInt32LE(offset, 16)

mkdirSync('release', {recursive: true})
writeFileSync('release/mandarin-obsidian.zip', Buffer.concat([...locals, directory, end]))
console.log('Wrote release/mandarin-obsidian.zip')
