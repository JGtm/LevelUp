/**
 * SquadV2RouteHost — bridge entre la layout Squad legacy (SquadContext) et
 * la page V2 (chunk S12).
 *
 * Lit `confirmedGamertags` depuis SquadContext (peuplé par SquadLayout via
 * GamertagCombobox) et les passe à `<SquadV2Page>` comme `teammates`. Garde
 * SquadV2Page découplé du store / context pour rester testable seul.
 *
 * Cascade (experienceTypes/playlists/modes) et période sont lus depuis le store
 * squad (filterContext commité par le bouton Analyser de SquadLayout).
 */
import { useParams } from '@tanstack/react-router'

import { useSquadFilterStore } from '@/stores/squadFilterStore'
import { detectActivePreset } from '@/components/shell/FilterOmnibar'
import { useSquadContext } from '../SquadContext'

import { SquadV2Page } from './SquadV2Page'
import type { SquadPeriod } from './types'

/** Convertit un preset FilterOmnibar vers la période acceptée par le backend V2. */
function presetToSquadPeriod(presetId: ReturnType<typeof detectActivePreset>): SquadPeriod {
  switch (presetId) {
    case '7d':  return '1w'
    case '30d': return '1m'
    // Pas de preset 3 mois côté backend V2 — fallback sur tout l'historique
    case '90d': return 'all'
    default:    return 'all'
  }
}

export function SquadV2RouteHost() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const { confirmedGamertags } = useSquadContext()
  const filterContext = useSquadFilterStore((s) => s.filterContext)

  const period = presetToSquadPeriod(detectActivePreset(filterContext.period))
  const experienceTypes = filterContext.cascade?.experience_types ?? []
  const playlists = filterContext.cascade?.playlists ?? []
  const maps = filterContext.cascade?.maps ?? []
  const modes = filterContext.cascade?.modes ?? []

  return (
    <SquadV2Page
      playerSlug={playerSlug}
      teammates={confirmedGamertags}
      period={period}
      experienceTypes={experienceTypes}
      playlists={playlists}
      maps={maps}
      modes={modes}
    />
  )
}
