import { existsSync, readFileSync, statSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { inflateSync } from 'node:zlib'

import { describe, expect, it } from 'vitest'

const frontendDirectory = resolve(dirname(fileURLToPath(import.meta.url)), '../../..')
const assetPath = (path: string): string => resolve(frontendDirectory, path)

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

  for (let offset = pngSignature.length; offset + 12 <= buffer.length; ) {
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

function inheritedStroke(element: Element): string | null {
  let current: Element | null = element

  while (current) {
    const attributeStroke = current.getAttribute('stroke')
    const styleStroke = current.getAttribute('style')?.match(/(?:^|;)\s*stroke\s*:\s*([^;]+)/i)?.[1]
    const stroke = attributeStroke ?? styleStroke
    if (stroke && stroke.trim().toLowerCase() !== 'none') return stroke
    if (current.tagName.toLowerCase() === 'svg') break
    current = current.parentElement
  }

  return null
}

describe('Snowpuff brand asset contract', () => {
  const pngAssets = [
    { path: 'public/brand/luoxue-snowpuff-3d.png', size: 1024 },
    { path: 'public/brand/luoxue-snowpuff-mark-512.png', size: 512 },
    { path: 'public/brand/luoxue-snowpuff-favicon-32.png', size: 32 },
    { path: 'public/brand/luoxue-snowpuff-touch-180.png', size: 180 },
    { path: 'public/brand/luoxue-snowpuff-extracted-512.png', size: 512 },
    { path: 'public/brand/luoxue-snowpuff-extracted-32.png', size: 32 },
    { path: 'public/brand/luoxue-snowpuff-extracted-touch-180.png', size: 180 },
    { path: 'public/logo.png', size: 512 },
  ] as const

  it('ships every planned source and compatibility asset', () => {
    const paths = [
      ...pngAssets.map((asset) => asset.path),
      'public/brand/luoxue-snowpuff-mark.svg',
      'public/brand/luoxue-snowpuff-extracted.svg',
      'public/brand/luoxue-snowpuff-exact.png',
    ]

    for (const path of paths) expect(existsSync(assetPath(path)), `${path} should exist`).toBe(true)
  })

  it.each(pngAssets)('$path is a square alpha PNG with transparent corners', ({ path, size }) => {
    const png = decodePng(readFileSync(assetPath(path)))

    expect({ width: png.width, height: png.height }).toEqual({ width: size, height: size })
    expect([3, 4, 6]).toContain(png.colorType)
    if (png.colorType === 3) expect(png.transparency).toBeDefined()

    expect([
      alphaAt(png, 0, 0),
      alphaAt(png, png.width - 1, 0),
      alphaAt(png, 0, png.height - 1),
      alphaAt(png, png.width - 1, png.height - 1),
    ]).toEqual([0, 0, 0, 0])
  })

  it('keeps the uploaded source intact and mirrors the extracted 512px mark at /logo.png', () => {
    const sourcePath = assetPath('public/brand/luoxue-snowpuff-exact.png')
    const extractedPath = assetPath('public/brand/luoxue-snowpuff-extracted-512.png')
    const compatibilityPath = assetPath('public/logo.png')
    const source = decodePng(readFileSync(sourcePath))

    expect({ width: source.width, height: source.height }).toEqual({ width: 186, height: 222 })
    expect(statSync(sourcePath).size).toBeLessThan(300 * 1024)
    expect(statSync(extractedPath).size).toBeLessThan(300 * 1024)
    expect(readFileSync(compatibilityPath).equals(readFileSync(extractedPath))).toBe(true)
  })

  it('ships a true vector extraction without embedding the source bitmap or a canvas background', () => {
    const svg = readFileSync(assetPath('public/brand/luoxue-snowpuff-extracted.svg'), 'utf8')
    const document = new DOMParser().parseFromString(svg, 'image/svg+xml')
    const root = document.documentElement

    expect(svg).toContain('viewBox="0 0 512 512"')
    expect(root.querySelectorAll('image')).toHaveLength(0)
    expect(root.querySelector('#puff')).not.toBeNull()
    expect(root.querySelectorAll('path').length).toBeGreaterThanOrEqual(3)
    expect(Array.from(root.children).some((element) => element.tagName.toLowerCase() === 'rect')).toBe(false)
  })

  it('uses only controlled soft gradients and keeps the main silhouette free of a hard stroke', () => {
    const svg = readFileSync(assetPath('public/brand/luoxue-snowpuff-mark.svg'), 'utf8')
    const document = new DOMParser().parseFromString(svg, 'image/svg+xml')
    const root = document.documentElement
    const silhouette = root.querySelector('#snowpuff')
    const gradients = Array.from(root.querySelectorAll('[id]')).filter((element) =>
      element.tagName.toLowerCase().endsWith('gradient'),
    )
    const visibleSilhouettes = Array.from(root.querySelectorAll('use')).filter(
      (element) => element.getAttribute('href') === '#snowpuff' && !element.closest('clipPath'),
    )
    const paintServerIds = Array.from(svg.matchAll(/url\(#([^)]+)\)/g), (match) => match[1])

    expect(svg).toContain('viewBox="0 0 512 512"')
    expect(svg).not.toMatch(/<(?:filter|image)\b/i)
    expect(svg).not.toMatch(/\bfilter\s*=/i)
    expect(gradients.map((gradient) => gradient.id)).toEqual(['bodyGradient', 'iceBlueBlend'])
    expect(gradients.every((gradient) => gradient.getAttribute('gradientUnits') === 'userSpaceOnUse')).toBe(
      true,
    )
    expect(paintServerIds).toEqual(['bodyGradient', 'iceBlueBlend'])
    expect(root.querySelectorAll('clipPath')).toHaveLength(0)
    expect(root.querySelectorAll('path')).toHaveLength(3)
    expect(silhouette).not.toBeNull()
    expect(visibleSilhouettes).toHaveLength(2)
    expect(inheritedStroke(silhouette!)).toBeNull()
    expect(visibleSilhouettes.every((element) => inheritedStroke(element) === null)).toBe(true)
  })
})
