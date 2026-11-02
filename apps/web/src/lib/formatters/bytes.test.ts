import { describe, expect, it } from 'vitest'

import { formatBytes } from './bytes'

describe('formatBytes', () => {
  it('unités décimales par langue : o / Ko / Mo / Go en FR, B / KB / MB / GB en EN', () => {
    expect(formatBytes(512, 'fr')).toBe('512 o')
    expect(formatBytes(512, 'en')).toBe('512 B')
    expect(formatBytes(1_500, 'fr')).toMatch(/^1,5 Ko$/)
    expect(formatBytes(1_500, 'en')).toBe('1.5 KB')
    expect(formatBytes(2_000_000, 'fr')).toMatch(/Mo$/)
    expect(formatBytes(3_000_000_000, 'en')).toBe('3 GB')
    expect(formatBytes(undefined, 'fr')).toBe('—')
  })
})
