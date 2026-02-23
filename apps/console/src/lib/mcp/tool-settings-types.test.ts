import { describe, it, expect } from 'vitest'
import { toToolSettingsMap, toModuleDescriptionsMap } from './tool-settings-types'

describe('toToolSettingsMap', () => {
  it('groups settings by module_name', () => {
    const settings = [
      { module_name: 'notion', tool_id: 'notion:search', enabled: true },
      { module_name: 'notion', tool_id: 'notion:delete_page', enabled: false },
      { module_name: 'github', tool_id: 'github:list_issues', enabled: true },
    ]
    const result = toToolSettingsMap(settings)
    expect(result).toEqual({
      notion: { 'notion:search': true, 'notion:delete_page': false },
      github: { 'github:list_issues': true },
    })
  })

  it('returns empty object for empty array', () => {
    expect(toToolSettingsMap([])).toEqual({})
  })

  it('overwrites duplicate tool_id within same module', () => {
    const settings = [
      { module_name: 'notion', tool_id: 'notion:search', enabled: true },
      { module_name: 'notion', tool_id: 'notion:search', enabled: false },
    ]
    const result = toToolSettingsMap(settings)
    expect(result.notion['notion:search']).toBe(false)
  })
})

describe('toModuleDescriptionsMap', () => {
  it('converts array to map', () => {
    const descriptions = [
      { module_name: 'notion', description: 'Notion workspace' },
      { module_name: 'github', description: 'GitHub repos' },
    ]
    const result = toModuleDescriptionsMap(descriptions)
    expect(result).toEqual({
      notion: 'Notion workspace',
      github: 'GitHub repos',
    })
  })

  it('returns empty object for empty array', () => {
    expect(toModuleDescriptionsMap([])).toEqual({})
  })
})
