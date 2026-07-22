import { readdirSync, readFileSync } from 'node:fs'
import { dirname, join, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { compileTemplate, parse } from 'vue/compiler-sfc'
import { describe, expect, it } from 'vitest'

type AstAttribute = {
  type: 6
  name: string
  value?: { content?: string }
}

type AstDirective = {
  type: 7
  name: string
  arg?: { content?: string }
  exp?: { loc?: { source?: string } }
}

type AstElement = {
  type: 1
  tag: string
  props: Array<AstAttribute | AstDirective>
  children?: unknown[]
}

type ComponentUse = {
  file: string
  element: AstElement
}

const testDirectory = dirname(fileURLToPath(import.meta.url))
const sourceDirectory = resolve(testDirectory, '../../..')
const adminViewsDirectory = join(sourceDirectory, 'views/admin')
const userViewsDirectory = join(sourceDirectory, 'views/user')

function listVueFiles(directory: string): string[] {
  return readdirSync(directory, { withFileTypes: true })
    .flatMap((entry) => {
      const path = join(directory, entry.name)
      if (entry.isDirectory()) return listVueFiles(path)
      return entry.isFile() && entry.name.endsWith('.vue') ? [path] : []
    })
    .sort()
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function collectElements(node: unknown, elements: AstElement[] = []): AstElement[] {
  if (!isRecord(node)) return elements

  if (node.type === 1 && typeof node.tag === 'string' && Array.isArray(node.props)) {
    elements.push(node as AstElement)
  }

  if (Array.isArray(node.children)) {
    node.children.forEach((child) => collectElements(child, elements))
  }

  if (Array.isArray(node.branches)) {
    node.branches.forEach((branch) => collectElements(branch, elements))
  }

  return elements
}

function compileViewElements(file: string): AstElement[] {
  const source = readFileSync(file, 'utf8')
  const parsed = parse(source, { filename: file })
  expect(parsed.errors, `${relative(sourceDirectory, file)} must be a valid SFC`).toEqual([])

  const template = parsed.descriptor.template
  if (!template) return []

  const compiled = compileTemplate({
    source: template.content,
    filename: file,
    id: relative(sourceDirectory, file),
  })
  expect(compiled.errors, `${relative(sourceDirectory, file)} template must compile`).toEqual([])
  expect(compiled.ast, `${relative(sourceDirectory, file)} must expose a template AST`).toBeTruthy()

  return collectElements(compiled.ast)
}

function collectComponentUses(directory: string): ComponentUse[] {
  return listVueFiles(directory).flatMap((file) =>
    compileViewElements(file).map((element) => ({
      file: relative(sourceDirectory, file),
      element,
    })),
  )
}

function staticAttribute(element: AstElement, name: string): string | undefined {
  const attribute = element.props.find(
    (prop): prop is AstAttribute => prop.type === 6 && prop.name === name,
  )
  return attribute?.value?.content
}

function boundExpression(element: AstElement, argument: string): string | undefined {
  const directive = element.props.find(
    (prop): prop is AstDirective =>
      prop.type === 7 && prop.name === 'bind' && prop.arg?.content === argument,
  )
  return directive?.exp?.loc?.source
}

function usesDynamicAppLayout(element: AstElement): boolean {
  return element.tag === 'component' && boundExpression(element, 'is')?.includes('AppLayout') === true
}

function usesHomeClay(element: AstElement): boolean {
  return (
    staticAttribute(element, 'variant') === 'home-clay' ||
    boundExpression(element, 'variant')?.includes('home-clay') === true
  )
}

describe('AppLayout view-level home-clay contract', () => {
  it('requires every admin AppLayout use to opt into home-clay', () => {
    const uses = collectComponentUses(adminViewsDirectory)
    const staticUses = uses.filter(({ element }) => element.tag === 'AppLayout')
    const dynamicUses = uses.filter(({ element }) => usesDynamicAppLayout(element))

    expect(staticUses.length).toBeGreaterThan(0)
    staticUses.forEach(({ file, element }) => {
      expect(
        staticAttribute(element, 'variant'),
        `${file}: <AppLayout> must explicitly use variant="home-clay"`,
      ).toBe('home-clay')
    })

    const opsUse = dynamicUses.find(({ file }) => file.endsWith('views/admin/ops/OpsDashboard.vue'))
    expect(opsUse, 'OpsDashboard must retain its dynamic AppLayout wrapper').toBeDefined()

    dynamicUses.forEach(({ file, element }) => {
      expect(
        usesHomeClay(element),
        `${file}: dynamic AppLayout must bind the home-clay variant`,
      ).toBe(true)
    })

    const opsVariantExpression = boundExpression(opsUse!.element, 'variant')
    expect(opsVariantExpression).toContain('isFullscreen')
    expect(opsVariantExpression).toContain('undefined')
    expect(opsVariantExpression).toContain('home-clay')
  })

  it('lets user views inherit the Snow Clay shell without enabling the admin content adapter', () => {
    const uses = collectComponentUses(userViewsDirectory).filter(
      ({ element }) => element.tag === 'AppLayout' || usesDynamicAppLayout(element),
    )

    expect(uses.length).toBeGreaterThan(0)
    uses.forEach(({ file, element }) => {
      expect(
        usesHomeClay(element),
        `${file}: user AppLayout must inherit the shared shell without the admin content adapter`,
      ).toBe(false)
    })
  })

  it('requires every admin page shell to declare its content density', () => {
    const uses = collectComponentUses(adminViewsDirectory)
    const appLayoutFiles = new Set(
      uses
        .filter(({ element }) => element.tag === 'AppLayout' || usesDynamicAppLayout(element))
        .map(({ file }) => file),
    )
    const validKinds = new Set(['overview', 'table', 'form', 'ops'])

    appLayoutFiles.forEach((file) => {
      const elements = uses.filter((use) => use.file === file).map((use) => use.element)
      const marker = elements.find((element) => staticAttribute(element, 'data-admin-page-kind'))
      const usesTablePageLayout = elements.some((element) => element.tag === 'TablePageLayout')

      expect(
        Boolean(marker) || usesTablePageLayout,
        `${file}: page content must declare data-admin-page-kind or use TablePageLayout`,
      ).toBe(true)

      if (marker) {
        const kind = staticAttribute(marker, 'data-admin-page-kind')
        expect(validKinds.has(kind ?? ''), `${file}: invalid data-admin-page-kind="${kind}"`).toBe(true)
      }
    })
  })
})
