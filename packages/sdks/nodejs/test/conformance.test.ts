import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'
import {
  parseAampHeaders,
  type EmailMetadata,
} from '../src/parser.js'

interface ParserFixture {
  name: string
  input: EmailMetadata
  expected: Record<string, unknown>
}

interface ParserFixtureSuite {
  schemaVersion: string
  cases: ParserFixture[]
}

const fixturePath = new URL('../../../../conformance/fixtures/parser.json', import.meta.url)
const fixtures = JSON.parse(readFileSync(fixturePath, 'utf-8')) as ParserFixtureSuite

function canonicalProjection(
  parsed: Record<string, unknown>,
  expected: Record<string, unknown>,
): Record<string, unknown> {
  return Object.fromEntries(
    Object.keys(expected).map((key) => [key, parsed[key]]),
  )
}

describe(`shared parser conformance fixtures v${fixtures.schemaVersion}`, () => {
  for (const fixture of fixtures.cases) {
    it(fixture.name, () => {
      const parsed = parseAampHeaders(fixture.input)
      expect(parsed).not.toBeNull()

      const actual = canonicalProjection(
        parsed as unknown as Record<string, unknown>,
        fixture.expected,
      )
      expect(actual).toEqual(fixture.expected)
    })
  }
})
