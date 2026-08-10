import { describe, expect, it } from 'vitest'
import { existsSync, readdirSync, readFileSync } from 'node:fs'
import { join, relative, resolve } from 'node:path'
import * as ts from 'typescript'

const SOURCE_ROOT = existsSync(resolve(process.cwd(), 'src'))
  ? resolve(process.cwd(), 'src')
  : resolve(process.cwd(), 'frontend', 'src')
const AUTH_SESSION_SOURCE = join(SOURCE_ROOT, 'auth', 'authSession.ts')
const PROTECTED_STORAGE_KEYS = new Set([
  'auth_token',
  'refresh_token',
  'token_expires_at',
  'auth_user',
  'auth_session_generation',
  'auth_session_head',
  'auth_session_invalidated_generation',
])
const PROTECTED_STORAGE_KEY_PREFIXES = ['auth_session_family:'] as const
const STORAGE_METHODS = new Set(['getItem', 'setItem', 'removeItem'])
const AUTH_SESSION_KEY_PROPERTIES = new Map([
  ['accessToken', 'auth_token'],
  ['refreshToken', 'refresh_token'],
  ['expiresAt', 'token_expires_at'],
  ['user', 'auth_user'],
  ['generation', 'auth_session_generation'],
  ['head', 'auth_session_head'],
])

type BoundaryViolation = 'session-storage' | 'pinia-token'

// Every storage expression understood by the detector originates from one of
// these browser globals, and every auth-store expression originates from
// useAuthStore. Keep this prefilter broader than the protected key/property
// markers so concatenated forms such as 'auth_' + 'token' remain detectable.
const BOUNDARY_SOURCE_MARKER = /\b(?:globalThis|window|localStorage|sessionStorage|useAuthStore)\b/

function productionSourceFiles(directory: string): string[] {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = join(directory, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === '__tests__' || entry.name === 'design-preview') return []
      return productionSourceFiles(path)
    }
    if (!/\.(?:ts|tsx|vue)$/.test(entry.name) || /\.(?:spec|test)\.tsx?$/.test(entry.name)) return []
    return [path]
  })
}

function scriptSource(path: string, source: string): string {
  if (!path.endsWith('.vue')) return source
  return Array.from(source.matchAll(/<script\b[^>]*>([\s\S]*?)<\/script>/gi))
    .map((match) => match[1])
    .join('\n')
}

function unwrapExpression(expression: ts.Expression): ts.Expression {
  let current = expression
  while (
    ts.isAsExpression(current)
    || ts.isTypeAssertionExpression(current)
    || ts.isParenthesizedExpression(current)
    || ts.isNonNullExpression(current)
    || ts.isSatisfiesExpression(current)
  ) {
    current = current.expression
  }
  return current
}

function staticPropertyName(
  expression: ts.PropertyAccessExpression | ts.ElementAccessExpression,
  strings: ReadonlyMap<string, string>,
): string | null {
  if (ts.isPropertyAccessExpression(expression)) return expression.name.text
  return expression.argumentExpression
    ? staticString(expression.argumentExpression, strings)
    : null
}

function staticString(
  expression: ts.Expression,
  strings: ReadonlyMap<string, string>,
  keyObjects: ReadonlySet<string> = new Set(),
): string | null {
  const current = unwrapExpression(expression)
  if (ts.isStringLiteralLike(current)) return current.text
  if (ts.isIdentifier(current)) return strings.get(current.text) ?? null
  if (
    ts.isBinaryExpression(current)
    && current.operatorToken.kind === ts.SyntaxKind.PlusToken
  ) {
    const left = staticString(current.left, strings, keyObjects)
    const right = staticString(current.right, strings, keyObjects)
    return left !== null && right !== null ? left + right : null
  }
  if (ts.isTemplateExpression(current)) {
    let value = current.head.text
    for (const span of current.templateSpans) {
      const expressionValue = staticString(span.expression, strings, keyObjects)
      if (expressionValue === null) return null
      value += expressionValue + span.literal.text
    }
    return value
  }
  if (ts.isPropertyAccessExpression(current) || ts.isElementAccessExpression(current)) {
    const property = staticPropertyName(current, strings)
    const owner = unwrapExpression(current.expression)
    if (property && ts.isIdentifier(owner) && keyObjects.has(owner.text)) {
      return AUTH_SESSION_KEY_PROPERTIES.get(property) ?? null
    }
  }
  return null
}

function staticStringPrefix(
  expression: ts.Expression,
  strings: ReadonlyMap<string, string>,
  keyObjects: ReadonlySet<string>,
): string | null {
  const exact = staticString(expression, strings, keyObjects)
  if (exact !== null) return exact

  const current = unwrapExpression(expression)
  if (
    ts.isBinaryExpression(current)
    && current.operatorToken.kind === ts.SyntaxKind.PlusToken
  ) {
    const left = staticString(current.left, strings, keyObjects)
    if (left !== null) {
      return left + (staticStringPrefix(current.right, strings, keyObjects) ?? '')
    }
    return staticStringPrefix(current.left, strings, keyObjects)
  }
  if (ts.isTemplateExpression(current)) {
    let prefix = current.head.text
    for (const span of current.templateSpans) {
      const expressionValue = staticString(span.expression, strings, keyObjects)
      if (expressionValue === null) {
        return prefix + (staticStringPrefix(span.expression, strings, keyObjects) ?? '')
      }
      prefix += expressionValue + span.literal.text
    }
    return prefix
  }
  return null
}

function isProtectedStorageKey(
  expression: ts.Expression,
  strings: ReadonlyMap<string, string>,
  keyObjects: ReadonlySet<string>,
): boolean {
  const exact = staticString(expression, strings, keyObjects)
  if (exact !== null && PROTECTED_STORAGE_KEYS.has(exact)) return true

  const prefix = staticStringPrefix(expression, strings, keyObjects)
  return prefix !== null
    && PROTECTED_STORAGE_KEY_PREFIXES.some((protectedPrefix) => (
      prefix.startsWith(protectedPrefix)
    ))
}

function boundaryViolations(path: string, source: string): Set<BoundaryViolation> {
  const script = scriptSource(path, source)
  if (!BOUNDARY_SOURCE_MARKER.test(script)) return new Set()

  const parsed = ts.createSourceFile(
    path,
    script,
    ts.ScriptTarget.Latest,
    false,
    ts.ScriptKind.TS,
  )
  const violations = new Set<BoundaryViolation>()
  const strings = new Map<string, string>()
  const windowAliases = new Set(['window', 'globalThis'])
  const storageAliases = new Set(['localStorage', 'sessionStorage'])
  const storageMethodAliases = new Set<string>()
  const authStoreFactories = new Set(['useAuthStore'])
  const storeToRefsFactories = new Set(['storeToRefs'])
  const authStoreAliases = new Set<string>()
  const authRefsAliases = new Set<string>()
  const authStateAliases = new Set<string>()
  const keyObjectAliases = new Set(['AUTH_SESSION_STORAGE_KEYS'])

  const isWindowExpression = (expression: ts.Expression): boolean => {
    const current = unwrapExpression(expression)
    return ts.isIdentifier(current) && windowAliases.has(current.text)
  }

  const isStorageExpression = (expression: ts.Expression): boolean => {
    const current = unwrapExpression(expression)
    if (ts.isIdentifier(current)) return storageAliases.has(current.text)
    if (ts.isPropertyAccessExpression(current) || ts.isElementAccessExpression(current)) {
      const property = staticPropertyName(current, strings)
      return (property === 'localStorage' || property === 'sessionStorage')
        && isWindowExpression(current.expression)
    }
    return false
  }

  const isNamedCall = (expression: ts.Expression, names: ReadonlySet<string>): boolean => {
    const current = unwrapExpression(expression)
    return ts.isCallExpression(current)
      && ts.isIdentifier(unwrapExpression(current.expression))
      && names.has((unwrapExpression(current.expression) as ts.Identifier).text)
  }

  const isAuthStoreExpression = (expression: ts.Expression): boolean => {
    const current = unwrapExpression(expression)
    if (ts.isIdentifier(current)) return authStoreAliases.has(current.text)
    return isNamedCall(current, authStoreFactories)
  }

  const isAuthRefsExpression = (expression: ts.Expression): boolean => {
    const current = unwrapExpression(expression)
    if (ts.isIdentifier(current)) return authRefsAliases.has(current.text)
    if (!isNamedCall(current, storeToRefsFactories)) return false
    const call = current as ts.CallExpression
    return call.arguments.length > 0 && isAuthStoreExpression(call.arguments[0])
  }

  const isAuthStateExpression = (expression: ts.Expression): boolean => {
    const current = unwrapExpression(expression)
    if (ts.isIdentifier(current)) return authStateAliases.has(current.text)
    if (!ts.isPropertyAccessExpression(current) && !ts.isElementAccessExpression(current)) {
      return false
    }
    return staticPropertyName(current, strings) === '$state'
      && isAuthStoreExpression(current.expression)
  }

  const classifyBinding = (declaration: ts.VariableDeclaration): void => {
    if (!declaration.initializer) return
    const initializer = unwrapExpression(declaration.initializer)

    if (ts.isIdentifier(declaration.name)) {
      const name = declaration.name.text
      const value = staticString(initializer, strings, keyObjectAliases)
      if (value !== null) strings.set(name, value)
      if (isWindowExpression(initializer)) windowAliases.add(name)
      if (isStorageExpression(initializer)) storageAliases.add(name)
      if (isAuthStoreExpression(initializer)) authStoreAliases.add(name)
      if (isAuthRefsExpression(initializer)) authRefsAliases.add(name)
      if (isAuthStateExpression(initializer)) authStateAliases.add(name)
      if (ts.isIdentifier(initializer) && authStoreFactories.has(initializer.text)) {
        authStoreFactories.add(name)
      }
      if (ts.isIdentifier(initializer) && storeToRefsFactories.has(initializer.text)) {
        storeToRefsFactories.add(name)
      }
      if (ts.isIdentifier(initializer) && keyObjectAliases.has(initializer.text)) {
        keyObjectAliases.add(name)
      }
      if (
        (ts.isPropertyAccessExpression(initializer) || ts.isElementAccessExpression(initializer))
        && isStorageExpression(initializer.expression)
        && STORAGE_METHODS.has(staticPropertyName(initializer, strings) ?? '')
      ) {
        storageMethodAliases.add(name)
      }
      return
    }

    if (!ts.isObjectBindingPattern(declaration.name)) return
    const storageSource = isStorageExpression(initializer)
    const authSource = isAuthStoreExpression(initializer) || isAuthRefsExpression(initializer)
    const authStateSource = isAuthStateExpression(initializer)
    for (const element of declaration.name.elements) {
      const property = element.propertyName
        ? ts.isIdentifier(element.propertyName) || ts.isStringLiteralLike(element.propertyName)
          ? element.propertyName.text
          : null
        : ts.isIdentifier(element.name)
          ? element.name.text
          : null
      if (!property || !ts.isIdentifier(element.name)) continue
      if (storageSource && STORAGE_METHODS.has(property)) {
        storageMethodAliases.add(element.name.text)
      }
      if (authSource && property === 'token') violations.add('pinia-token')
      if (authSource && property === '$state') authStateAliases.add(element.name.text)
      if (authStateSource && property === 'token') violations.add('pinia-token')
    }
  }

  const collectBindings = (node: ts.Node): void => {
    if (ts.isImportDeclaration(node) && ts.isStringLiteral(node.moduleSpecifier)) {
      const bindings = node.importClause?.namedBindings
      if (bindings && ts.isNamedImports(bindings)) {
        for (const element of bindings.elements) {
          const imported = element.propertyName?.text ?? element.name.text
          if (imported === 'useAuthStore') authStoreFactories.add(element.name.text)
          if (imported === 'storeToRefs') storeToRefsFactories.add(element.name.text)
          if (
            imported === 'AUTH_SESSION_STORAGE_KEYS'
            && /(?:^|\/)authSession$/.test(node.moduleSpecifier.text)
          ) {
            keyObjectAliases.add(element.name.text)
          }
        }
      }
    }

    if (ts.isVariableDeclaration(node)) classifyBinding(node)

    ts.forEachChild(node, collectBindings)
  }

  let bindingsChanged = true
  while (bindingsChanged) {
    const sizeBefore = strings.size
      + windowAliases.size
      + storageAliases.size
      + storageMethodAliases.size
      + authStoreFactories.size
      + storeToRefsFactories.size
      + authStoreAliases.size
      + authRefsAliases.size
      + authStateAliases.size
      + keyObjectAliases.size
    collectBindings(parsed)
    const sizeAfter = strings.size
      + windowAliases.size
      + storageAliases.size
      + storageMethodAliases.size
      + authStoreFactories.size
      + storeToRefsFactories.size
      + authStoreAliases.size
      + authRefsAliases.size
      + authStateAliases.size
      + keyObjectAliases.size
    bindingsChanged = sizeAfter !== sizeBefore
  }

  const inspect = (node: ts.Node): void => {
    if (ts.isCallExpression(node)) {
      const callee = unwrapExpression(node.expression)
      let storageMethodCall = ts.isIdentifier(callee) && storageMethodAliases.has(callee.text)
      if (ts.isPropertyAccessExpression(callee) || ts.isElementAccessExpression(callee)) {
        storageMethodCall = isStorageExpression(callee.expression)
          && STORAGE_METHODS.has(staticPropertyName(callee, strings) ?? '')
      }
      if (
        storageMethodCall
        && node.arguments.length > 0
        && isProtectedStorageKey(node.arguments[0], strings, keyObjectAliases)
      ) {
        violations.add('session-storage')
      }
    }

    if (ts.isPropertyAccessExpression(node) || ts.isElementAccessExpression(node)) {
      const property = staticPropertyName(node, strings)
      if (
        property === 'token'
        && (
          isAuthStoreExpression(node.expression)
          || isAuthRefsExpression(node.expression)
          || isAuthStateExpression(node.expression)
        )
      ) {
        violations.add('pinia-token')
      }
    }

    ts.forEachChild(node, inspect)
  }

  inspect(parsed)
  return violations
}

const productionBoundaryResults = productionSourceFiles(SOURCE_ROOT).map((path) => ({
  path,
  relativePath: relative(SOURCE_ROOT, path),
  violations: boundaryViolations(path, readFileSync(path, 'utf8')),
}))

describe('auth session source boundary', () => {
  it('keeps browser session storage keys behind authSession', () => {
    const violations = productionBoundaryResults
      .filter(({ path }) => path !== AUTH_SESSION_SOURCE)
      .filter(({ violations }) => violations.has('session-storage'))
      .map(({ relativePath }) => relativePath)

    expect(violations).toEqual([])
  })

  it('does not expose the Pinia token projection to production consumers', () => {
    const violations = productionBoundaryResults
      .filter(({ violations }) => violations.has('pinia-token'))
      .map(({ relativePath }) => relativePath)

    expect(violations).toEqual([])
  })

  it.each([
    [
      'constant key, storage alias and bracket call',
      "const TOKEN_KEY = 'auth_token'; const storage = window['localStorage']; storage['getItem'](TOKEN_KEY)",
      'session-storage',
    ],
    [
      'destructured storage method and imported key map',
      "import { AUTH_SESSION_STORAGE_KEYS as keys } from '@/auth/authSession'; const { removeItem: remove } = sessionStorage; remove(keys.refreshToken)",
      'session-storage',
    ],
    [
      'dynamic family key through a concatenated prefix alias',
      "const prefix = 'auth_session_family:'; const generation = getGeneration(); localStorage.getItem(prefix + generation)",
      'session-storage',
    ],
    [
      'dynamic family key through a template prefix',
      'const generation = getGeneration(); window.localStorage[`removeItem`](`auth_session_family:${generation}`)',
      'session-storage',
    ],
    [
      'concatenated storage, method and protected key names',
      "const root = globalThis; const storage = root['local' + 'Storage']; const read = storage['get' + 'Item']; read('auth_' + 'token')",
      'session-storage',
    ],
    [
      'aliased auth store property',
      'const first = useAuthStore(); const alias = first; void alias[\'token\']',
      'pinia-token',
    ],
    [
      'destructured auth store token',
      'const { token: accessToken } = useAuthStore(); void accessToken',
      'pinia-token',
    ],
    [
      'storeToRefs token destructuring through aliases',
      'const makeStore = useAuthStore; const refs = storeToRefs(makeStore()); const { token } = refs; void token',
      'pinia-token',
    ],
    [
      'direct auth store state token projection',
      'void useAuthStore().$state.token',
      'pinia-token',
    ],
    [
      'auth store state token through aliases',
      "const store = useAuthStore(); const { $state: state } = store; const alias = state; void alias['token']",
      'pinia-token',
    ],
    [
      'concatenated auth store token property',
      "const store = useAuthStore(); const property = 'to' + 'ken'; void store[property]",
      'pinia-token',
    ],
  ] as const)('detects %s', (_label, source, expected) => {
    expect(boundaryViolations('fixture.ts', source)).toContain(expected)
  })
})
