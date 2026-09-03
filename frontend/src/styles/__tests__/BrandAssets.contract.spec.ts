import { createHash } from 'node:crypto'
import { existsSync, readFileSync, readdirSync, statSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { inflateSync } from 'node:zlib'

import { describe, expect, it } from 'vitest'

const frontendDirectory = resolve(dirname(fileURLToPath(import.meta.url)), '../../..')
const assetPath = (path: string): string => resolve(frontendDirectory, path)
const canonicalLogoPath = assetPath('public/logo.png')

const pngSignature = Buffer.from([137, 80, 78, 71, 13, 10, 26, 10])
const channelsByColorType: Record<number, number> = {
  0: 1,
  2: 3,
  3: 1,
  4: 2,
  6: 4,
}

interface PngData {
  width: number
  height: number
  colorType: number
  transparency?: Buffer
  pixels: Buffer
}

function paethPredictor(left: number, above: number, upperLeft: number): number {
  const estimate = left + above - upperLeft
  const leftDistance = Math.abs(estimate - left)
  const aboveDistance = Math.abs(estimate - above)
  const upperLeftDistance = Math.abs(estimate - upperLeft)

  if (leftDistance <= aboveDistance && leftDistance <= upperLeftDistance) return left
  if (aboveDistance <= upperLeftDistance) return above
  return upperLeft
}

function decodePng(buffer: Buffer): PngData {
  if (!buffer.subarray(0, pngSignature.length).equals(pngSignature)) {
    throw new Error('Expected a PNG signature')
  }

  let width = 0
  let height = 0
  let bitDepth = 0
  let colorType = -1
  let interlace = -1
  let transparency: Buffer | undefined
  const imageData: Buffer[] = []

  for (let offset = pngSignature.length; offset + 12 <= buffer.length;) {
    const length = buffer.readUInt32BE(offset)
    const chunkEnd = offset + length + 12
    if (chunkEnd > buffer.length) throw new Error('PNG chunk extends beyond the file')

    const type = buffer.toString('ascii', offset + 4, offset + 8)
    const data = buffer.subarray(offset + 8, offset + 8 + length)

    if (type === 'IHDR') {
      width = data.readUInt32BE(0)
      height = data.readUInt32BE(4)
      bitDepth = data[8]
      colorType = data[9]
      interlace = data[12]
    } else if (type === 'tRNS') {
      transparency = Buffer.from(data)
    } else if (type === 'IDAT') {
      imageData.push(Buffer.from(data))
    }

    offset = chunkEnd
    if (type === 'IEND') break
  }

  if (!width || !height || imageData.length === 0) throw new Error('PNG is missing required image data')
  if (bitDepth !== 8) throw new Error(`Only 8-bit PNG assets are supported, received ${bitDepth}-bit`)
  if (interlace !== 0) throw new Error('Brand PNG assets must be non-interlaced')

  const channels = channelsByColorType[colorType]
  if (!channels) throw new Error(`Unsupported PNG color type: ${colorType}`)

  const rowLength = width * channels
  const encoded = inflateSync(Buffer.concat(imageData))
  if (encoded.length !== (rowLength + 1) * height) {
    throw new Error('PNG scanline data does not match its dimensions')
  }

  const pixels = Buffer.alloc(rowLength * height)
  for (let y = 0; y < height; y += 1) {
    const encodedRow = y * (rowLength + 1)
    const decodedRow = y * rowLength
    const filter = encoded[encodedRow]

    for (let x = 0; x < rowLength; x += 1) {
      const raw = encoded[encodedRow + x + 1]
      const left = x >= channels ? pixels[decodedRow + x - channels] : 0
      const above = y > 0 ? pixels[decodedRow - rowLength + x] : 0
      const upperLeft = y > 0 && x >= channels ? pixels[decodedRow - rowLength + x - channels] : 0

      let predictor = 0
      if (filter === 1) predictor = left
      else if (filter === 2) predictor = above
      else if (filter === 3) predictor = Math.floor((left + above) / 2)
      else if (filter === 4) predictor = paethPredictor(left, above, upperLeft)
      else if (filter !== 0) throw new Error(`Unsupported PNG filter: ${filter}`)

      pixels[decodedRow + x] = (raw + predictor) & 0xff
    }
  }

  return { width, height, colorType, transparency, pixels }
}

function alphaAt(png: PngData, x: number, y: number): number {
  const channels = channelsByColorType[png.colorType]
  const pixelOffset = (y * png.width + x) * channels

  if (png.colorType === 3) return png.transparency?.[png.pixels[pixelOffset]] ?? 255
  if (png.colorType === 4) return png.pixels[pixelOffset + 1]
  if (png.colorType === 6) return png.pixels[pixelOffset + 3]
  return 255
}

describe('canonical brand asset contract', () => {
  it('ships the approved 512px transparent PNG exactly at /logo.png', () => {
    expect(existsSync(canonicalLogoPath)).toBe(true)

    const bytes = readFileSync(canonicalLogoPath)
    const png = decodePng(bytes)

    expect({ width: png.width, height: png.height }).toEqual({ width: 512, height: 512 })
    expect(statSync(canonicalLogoPath).size).toBeLessThan(300 * 1024)
    expect(createHash('sha256').update(bytes).digest('hex')).toBe(
      '95fab1ff815329f29a552e1af3166752159928279a0f536aa333b2b579cfc86a',
    )
    expect([
      alphaAt(png, 0, 0),
      alphaAt(png, png.width - 1, 0),
      alphaAt(png, 0, png.height - 1),
      alphaAt(png, png.width - 1, png.height - 1),
    ]).toEqual([0, 0, 0, 0])
  })

  it('does not ship superseded snowflake or snowpuff logo variants', () => {
    const legacyAssets = readdirSync(assetPath('public/brand')).filter((name) => (
      /^luoxue-(?:snowflake|snowpuff)/i.test(name)
    ))

    expect(legacyAssets).toEqual([])
  })
})
