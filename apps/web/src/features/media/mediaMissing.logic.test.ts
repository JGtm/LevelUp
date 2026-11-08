import { describe, expect, it } from 'vitest'
import type { MediaMissingItem } from '@/lib/api/types'
import { getMediaModalsText } from './i18n-modals'
import {
  findMissingItem,
  folderNoticeKind,
  formatCaptureDelta,
  formatCaptureTime,
} from './mediaMissing.logic'

const ITEMS: MediaMissingItem[] = [
  { basename: 'a.mp4', file_path: '/files/a.mp4', kind: 'clip' },
  { basename: 'b.png', file_path: '/files/b.png', kind: 'screenshot' },
]

describe('formatCaptureTime', () => {
  it.each([null, undefined, '', 'pas une date'])('rend un tiret pour %j', (value) => {
    expect(formatCaptureTime(value, 'fr')).toBe('—')
  })

  it('formate un instant valide, sans tiret', () => {
    const out = formatCaptureTime('2026-04-26T14:30:00Z', 'fr')
    expect(out).not.toBe('—')
    expect(out).toMatch(/\d/)
  })

  it('suit la langue demandée', () => {
    const fr = formatCaptureTime('2026-04-26T14:30:00Z', 'fr')
    const en = formatCaptureTime('2026-04-26T14:30:00Z', 'en')
    expect(fr).not.toBe(en)
  })
})

describe('formatCaptureDelta', () => {
  const fr = getMediaModalsText('fr').missing
  const en = getMediaModalsText('en').missing

  it('rend vide quand le serveur ne calcule pas d’écart', () => {
    expect(formatCaptureDelta(null, fr)).toBe('')
    expect(formatCaptureDelta(undefined, fr)).toBe('')
  })

  it('arrondit à la minute, signe ignoré', () => {
    expect(formatCaptureDelta(125, fr)).toBe('±2 min')
    expect(formatCaptureDelta(-125, en)).toBe('±2 min')
  })

  it('moins d’une minute', () => {
    expect(formatCaptureDelta(20, fr)).toBe('< 1 min')
    expect(formatCaptureDelta(0, en)).toBe('< 1 min')
  })
})

describe('findMissingItem', () => {
  it('retrouve la fiche par son chemin', () => {
    expect(findMissingItem(ITEMS, '/files/b.png')?.basename).toBe('b.png')
  })

  it('rend null sans chemin ou quand la fiche a quitté la liste', () => {
    expect(findMissingItem(ITEMS, null)).toBeNull()
    expect(findMissingItem(ITEMS, '/files/c.mp4')).toBeNull()
    expect(findMissingItem([], '/files/a.mp4')).toBeNull()
  })
})

describe('folderNoticeKind', () => {
  it('aucun bandeau quand tout va bien', () => {
    expect(folderNoticeKind({ rootUnavailable: false, bulkPending: 0 })).toBeNull()
  })

  it('retrait massif en attente', () => {
    expect(folderNoticeKind({ rootUnavailable: false, bulkPending: 12 })).toBe('bulk-pending')
  })

  it('le dossier inaccessible passe avant le retrait massif', () => {
    expect(folderNoticeKind({ rootUnavailable: true, bulkPending: 12 })).toBe('root-unavailable')
    expect(folderNoticeKind({ rootUnavailable: true, bulkPending: 0 })).toBe('root-unavailable')
  })
})
