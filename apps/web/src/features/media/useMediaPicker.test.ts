/**
 * useMediaPicker — hook qui pilote l'ouverture du MediaMatchPicker depuis
 * une vignette ou un lecteur. Les 3 callers (home/match-view/galerie)
 * dépendent de la même API minimale.
 */
import { describe, expect, it } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import { useMediaPicker } from './useMediaPicker'
import type { MediaItemRow } from '@/lib/api/types'

function makeItem(overrides: Partial<MediaItemRow>): MediaItemRow {
  return {
    basename: 'x.mp4',
    file_path: '/x.mp4',
    kind: 'clip',
    thumbnail_path: null,
    match_id: 'match-1',
    capture_end_utc: null,
    match_start_time: null,
    section: 'mine',
    owner_gamertag: null,
    map_name: null,
    mode_name: null,
    liked: false,
    like_count: 0,
    ...overrides,
  }
}

describe('useMediaPicker', () => {
  it('démarre avec state = null', () => {
    const { result } = renderHook(() => useMediaPicker())
    expect(result.current.state).toBeNull()
  })

  it('openFor() avec un item ayant match_id → state.hasCurrentMatch = true', () => {
    const { result } = renderHook(() => useMediaPicker())
    const item = makeItem({ file_path: '/clip.mp4', match_id: 'match-42' })
    act(() => {
      result.current.openFor(item)
    })
    expect(result.current.state).toEqual({
      filePath: '/clip.mp4',
      hasCurrentMatch: true,
    })
  })

  it('openFor() avec un item sans match_id → state.hasCurrentMatch = false', () => {
    const { result } = renderHook(() => useMediaPicker())
    const item = makeItem({ file_path: '/orphan.mp4', match_id: null })
    act(() => {
      result.current.openFor(item)
    })
    expect(result.current.state).toEqual({
      filePath: '/orphan.mp4',
      hasCurrentMatch: false,
    })
  })

  it('close() remet state à null', () => {
    const { result } = renderHook(() => useMediaPicker())
    act(() => {
      result.current.openFor(makeItem({ file_path: '/a.mp4' }))
    })
    expect(result.current.state).not.toBeNull()
    act(() => {
      result.current.close()
    })
    expect(result.current.state).toBeNull()
  })

  it('openFor() puis openFor() avec un autre item → state remplacé', () => {
    const { result } = renderHook(() => useMediaPicker())
    act(() => {
      result.current.openFor(makeItem({ file_path: '/a.mp4', match_id: 'm1' }))
    })
    act(() => {
      result.current.openFor(makeItem({ file_path: '/b.mp4', match_id: null }))
    })
    expect(result.current.state).toEqual({
      filePath: '/b.mp4',
      hasCurrentMatch: false,
    })
  })

  it('openFor et close gardent une identité stable entre renders (utile pour memoization aval)', () => {
    const { result, rerender } = renderHook(() => useMediaPicker())
    const firstOpenFor = result.current.openFor
    const firstClose = result.current.close
    rerender()
    expect(result.current.openFor).toBe(firstOpenFor)
    expect(result.current.close).toBe(firstClose)
  })
})
