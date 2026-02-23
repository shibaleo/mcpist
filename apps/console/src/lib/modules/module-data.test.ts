import { describe, it, expect } from 'vitest'
import {
  isDangerous,
  getLocalizedText,
  getModuleDisplayName,
  getModuleIcon,
  getModuleDescription,
  getToolDescription,
} from './module-data'
import type { ToolDef, ModuleDef } from './module-data'

describe('isDangerous', () => {
  it('returns true when destructiveHint is true (default) and readOnlyHint is false', () => {
    const tool: ToolDef = {
      id: 'notion:delete_page',
      name: 'delete_page',
      descriptions: { 'en-US': 'Delete a page' },
      annotations: { destructiveHint: true, readOnlyHint: false },
    }
    expect(isDangerous(tool)).toBe(true)
  })

  it('returns true when annotations are empty (defaults apply)', () => {
    const tool: ToolDef = {
      id: 'notion:update',
      name: 'update',
      descriptions: {},
      annotations: {},
    }
    expect(isDangerous(tool)).toBe(true)
  })

  it('returns false when readOnlyHint is true', () => {
    const tool: ToolDef = {
      id: 'notion:search',
      name: 'search',
      descriptions: {},
      annotations: { readOnlyHint: true },
    }
    expect(isDangerous(tool)).toBe(false)
  })

  it('returns false when destructiveHint is explicitly false', () => {
    const tool: ToolDef = {
      id: 'notion:create',
      name: 'create',
      descriptions: {},
      annotations: { destructiveHint: false },
    }
    expect(isDangerous(tool)).toBe(false)
  })
})

describe('getLocalizedText', () => {
  it('returns ja-JP by default', () => {
    const texts = { 'ja-JP': '検索', 'en-US': 'Search' }
    expect(getLocalizedText(texts)).toBe('検索')
  })

  it('falls back to en-US when requested lang is missing', () => {
    const texts = { 'en-US': 'Search' }
    expect(getLocalizedText(texts, 'ja-JP')).toBe('Search')
  })

  it('returns empty string when texts is undefined', () => {
    expect(getLocalizedText(undefined)).toBe('')
  })

  it('returns empty string when no matching lang', () => {
    const texts = { 'fr-FR': 'Rechercher' }
    expect(getLocalizedText(texts, 'ja-JP')).toBe('')
  })

  it('returns specified language', () => {
    const texts = { 'ja-JP': '検索', 'en-US': 'Search' }
    expect(getLocalizedText(texts, 'en-US')).toBe('Search')
  })
})

describe('getModuleDescription', () => {
  it('delegates to getLocalizedText with module descriptions', () => {
    const mod: ModuleDef = {
      id: 'notion',
      name: 'Notion',
      status: 'active',
      descriptions: { 'ja-JP': 'ノート', 'en-US': 'Notes' },
      tools: [],
    }
    expect(getModuleDescription(mod)).toBe('ノート')
    expect(getModuleDescription(mod, 'en-US')).toBe('Notes')
  })
})

describe('getToolDescription', () => {
  it('delegates to getLocalizedText with tool descriptions', () => {
    const tool: ToolDef = {
      id: 'notion:search',
      name: 'search',
      descriptions: { 'ja-JP': '検索する', 'en-US': 'Search' },
      annotations: {},
    }
    expect(getToolDescription(tool)).toBe('検索する')
    expect(getToolDescription(tool, 'en-US')).toBe('Search')
  })
})

describe('getModuleDisplayName', () => {
  it('returns display name for known module', () => {
    expect(getModuleDisplayName('notion')).toBe('Notion')
    expect(getModuleDisplayName('github')).toBe('GitHub')
    expect(getModuleDisplayName('google_calendar')).toBe('Google Calendar')
  })

  it('returns moduleId for unknown module', () => {
    expect(getModuleDisplayName('unknown_module')).toBe('unknown_module')
  })
})

describe('getModuleIcon', () => {
  it('returns icon for known module', () => {
    expect(getModuleIcon('notion')).toBe('file-text')
    expect(getModuleIcon('github')).toBe('github')
  })

  it('returns "box" for unknown module', () => {
    expect(getModuleIcon('unknown_module')).toBe('box')
  })
})
