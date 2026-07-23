import { describe, expect, it } from 'vitest'

import type { AdminLogTail } from '@/lib/api/types'
import { nextLogCursor } from './logsCursor'

function page(partial: Partial<AdminLogTail>): AdminLogTail {
  return {
    module: 'sync',
    generated_at: '2026-07-23T10:00:00Z',
    entries: [],
    scanned_bytes: 0,
    truncated: false,
    has_more: false,
    ...partial,
  }
}

describe('nextLogCursor', () => {
  it('renvoie next_offset quand il reste des lignes plus anciennes', () => {
    expect(nextLogCursor(page({ has_more: true, next_offset: 4096 }))).toBe(4096)
  })

  it('renvoie undefined au début de fichier (has_more false)', () => {
    // has_more false = plus rien à charger, même si un offset traîne.
    expect(nextLogCursor(page({ has_more: false, next_offset: 4096 }))).toBeUndefined()
  })

  it('renvoie undefined si next_offset absent ou 0 (garde anti-boucle depuis la fin)', () => {
    expect(nextLogCursor(page({ has_more: true }))).toBeUndefined()
    expect(nextLogCursor(page({ has_more: true, next_offset: 0 }))).toBeUndefined()
  })
})
