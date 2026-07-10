// cross-feature-allow: tab orchestrateur Entraînement — agrège les composants
// coach (proposals) et profil/patterns Ascension.
/**
 * AscensionCoachingTab — tab "Entraînement" (couche coaching d'amélioration).
 *
 * Extrait de AscensionProfileTab lors du split en 2 onglets (2026-06-08).
 * Composition (verticale, du plus actionnable au plus analytique) :
 *   1. CoachProposalsCard — suggestions proactives
 *   2. CampaignTracker     — campagne en cours (si active)
 *   3. PlayerProfileV3     — identité, style, performance, leviers + CTAs
 *   4. Patterns contextuels + comportementaux + leviers calibrés
 *
 * La navigation Coach → création de campagne reste sur le même écran (modale
 * StartCampaign).
 */
import { useState } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSettings } from '@/features/settings/queries'
import { CoachProposalsCard } from '@/features/coach/CoachProposalsCard'
import { getCoachStrings } from '@/features/coach/i18n'
import type { AxisKind } from '@/lib/playerProfile'
import { CoachFocusCard } from './CoachFocusCard'
import { CampaignTracker } from './campaign/CampaignTracker'
import { StartCampaignModal } from './campaign/StartCampaignModal'
import { PlayerProfileV3 } from './profile/PlayerProfileV3'
import { useActiveCampaign } from './profile/queries'
import { usePatterns } from './queries'
import { getAscensionText } from './i18n'
import { PatternContextGrid } from './PatternContextGrid'
import { SquadVsSoloCard } from './SquadVsSoloCard'
import { BehaviorAlertList } from './BehaviorAlertList'
import { LeverList } from './LeverList'
import { LayerSection, SectionShell } from './AscensionLayers'

export function AscensionCoachingTab() {
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const playerSlug = currentPlayer?.player_slug ?? ''
  const locale = useAppShellStore((s) => s.locale)
  const t = getAscensionText(locale)
  const coachT = getCoachStrings(locale)
  const { data: settings } = useSettings()
  const proactiveEnabled = settings?.coach_proactive_mode ?? false

  const { data: activeCampaign } = useActiveCampaign(playerSlug)
  const hasActiveCampaign = !!activeCampaign && activeCampaign.status === 'active'

  const [campaignModal, setCampaignModal] = useState<{ open: boolean; axis: string; axisKind: AxisKind }>(
    { open: false, axis: '', axisKind: 'lusr_component' },
  )
  const openStartCampaign = (axis: string, axisKind: AxisKind) =>
    setCampaignModal({ open: true, axis, axisKind })

  if (!playerSlug) {
    return (
      <p className="p-6 text-sm text-muted-foreground">
        {t.coachingSelectPlayer}
      </p>
    )
  }

  return (
    <div className="space-y-10">
      {/* ─── Couche Ascension (coaching s'appuyant sur Prestige) ──────────── */}
      <LayerSection title={t.ascensionLayerTitle} description={t.ascensionLayerDescription}>
        <CoachFocusCard playerSlug={playerSlug} />
        <div id="coach-proposals">
          <CoachProposalsCard playerSlug={playerSlug} proactiveEnabled={proactiveEnabled} t={coachT} />
        </div>
        {hasActiveCampaign && <CampaignTracker playerSlug={playerSlug} campaign={activeCampaign} />}
        <PlayerProfileV3
          playerSlug={playerSlug}
          onStartCampaign={hasActiveCampaign ? undefined : openStartCampaign}
        />
        <PatternsSection playerSlug={playerSlug} t={t} />
      </LayerSection>

      <StartCampaignModal
        open={campaignModal.open}
        playerSlug={playerSlug}
        axis={campaignModal.axis}
        axisKind={campaignModal.axisKind}
        onOpenChange={(open) => setCampaignModal((s) => ({ ...s, open }))}
      />
    </div>
  )
}

// ─── Patterns ────────────────────────────────────────────────────────────────

interface PatternsSectionProps {
  playerSlug: string
  t: ReturnType<typeof getAscensionText>
}

function PatternsSection({ playerSlug, t }: PatternsSectionProps) {
  const { data: patterns, isLoading } = usePatterns(playerSlug)
  if (isLoading) return null
  const contextPatterns = patterns?.context_patterns ?? []
  const behaviorPatterns = patterns?.behavior_patterns ?? []
  const levers = patterns?.levers ?? []

  if (contextPatterns.length === 0 && behaviorPatterns.length === 0 && levers.length === 0) {
    return null
  }

  return (
    <div className="space-y-6">
      {contextPatterns.length > 0 && (
        <SectionShell title={t.patternsSectionTitle}>
          <PatternContextGrid patterns={contextPatterns} t={t} />
          <SquadVsSoloCard patterns={contextPatterns} t={t} />
        </SectionShell>
      )}
      {behaviorPatterns.length > 0 && (
        <SectionShell title={t.behaviorsSectionTitle}>
          <BehaviorAlertList patterns={behaviorPatterns} t={t} />
        </SectionShell>
      )}
      {levers.length > 0 && (
        <SectionShell title={t.leversSectionTitle}>
          <LeverList levers={levers} t={t} />
        </SectionShell>
      )}
    </div>
  )
}
