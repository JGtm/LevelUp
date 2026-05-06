/**
 * useMediaPicker — hook léger pour gérer l'ouverture du `MediaMatchPicker`
 * depuis une vignette ou un lecteur média. Centralise le state partagé entre
 * RecentMediaRail (home), MatchMediaTab et MediaPage (galerie).
 *
 * Extrait de MediaMatchPicker.tsx pour respecter `react-refresh/only-export-components`.
 */
import { useCallback, useState } from 'react'
import type { MediaItemRow } from '@/lib/api/types'

export interface MediaPickerState {
  filePath: string
  hasCurrentMatch: boolean
}

export interface MediaPickerHandle {
  state: MediaPickerState | null
  /** Ouvre le picker pour l'item donné (mode dérivé de `item.match_id`). */
  openFor: (item: MediaItemRow) => void
  close: () => void
}

export function useMediaPicker(): MediaPickerHandle {
  const [state, setState] = useState<MediaPickerState | null>(null)
  const openFor = useCallback((item: MediaItemRow) => {
    setState({ filePath: item.file_path, hasCurrentMatch: Boolean(item.match_id) })
  }, [])
  const close = useCallback(() => setState(null), [])
  return { state, openFor, close }
}
