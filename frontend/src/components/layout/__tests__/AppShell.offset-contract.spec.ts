import { readdirSync, readFileSync, statSync } from 'node:fs'
import { dirname, extname, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const srcRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../../..')
const productionExtensions = new Set(['.css', '.ts', '.vue'])
const retiredOffsetPattern = /(?<!\d)81px\b|\b5\.0625rem\b/

function collectProductionSources(directory: string): string[] {
  return readdirSync(directory).flatMap((entry) => {
    const absolutePath = resolve(directory, entry)
    const relativePath = relative(srcRoot, absolutePath).replaceAll('\\', '/')

    if (
      relativePath.includes('/__tests__/')
      || relativePath.startsWith('views/design-preview/')
      || relativePath.endsWith('.spec.ts')
      || relativePath.endsWith('.test.ts')
      || relativePath.endsWith('.d.ts')
    ) {
      return []
    }

    if (statSync(absolutePath).isDirectory()) {
      return collectProductionSources(absolutePath)
    }

    return productionExtensions.has(extname(entry)) ? [absolutePath] : []
  })
}

describe('application shell offset contract', () => {
  it('keeps retired literal header offsets out of production source', () => {
    const violations = collectProductionSources(srcRoot).flatMap((filePath) => {
      const relativePath = relative(srcRoot, filePath).replaceAll('\\', '/')

      return readFileSync(filePath, 'utf8')
        .split('\n')
        .flatMap((line, index) => (
          retiredOffsetPattern.test(line)
            ? [`${relativePath}:${index + 1}: ${line.trim()}`]
            : []
        ))
    })

    expect(violations, violations.join('\n')).toEqual([])
  })
})
