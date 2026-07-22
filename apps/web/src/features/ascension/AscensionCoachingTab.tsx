// cross-feature-allow: tab orchestrateur Entraînement — agrège les composants
// coach (proposals) et les pistes de progression Ascension.
/**
 * AscensionCoachingTab — onglet "Entraînement" (couche coaching d'amélioration).
 *
 * Restructuration 4 onglets (2026-07, DEC-3). Composition (du plus actionnable
 * au plus analytique) :
 *   1. CoachFocusCard       — cap du moment
 *   2. CoachProposalsCard   — suggestions proactives
 *   3. CampaignTracker      — campagne en cours (si active)
 *   4. ProgressionSection   — pistes de progression (leviers LUSR + défis suggérés)
 *   5. LeverList            — leviers calibrés (issus des patterns)
 *
 * L'identité/profil (PlayerProfileV3) et les patterns contextuels vivent
 * désormais dans l'onglet "Profil" (index). La navigation Coach → création de
 * campagne reste sur le même écran (modale StartCampaign).
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
import { ProgressionSection } from './profile/ProgressionSection'
import { useActiveCampaign, usePlayerProfile } from './profile/queries'
import { usePatterns } from './queries'
import { getAscensionText } from './i18n'
import { LeverList } from './LeverList'
import { LayerSection, SectionShell } from './AscensionLayers'

export function AscensionCoachingTab() {
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const playerSlug = currentPlayer?.player_slug ?? ''
  const locale = useAppShellStore((s) => s.locale)
  const t = getAscensionText(locale)
  const coachT = getCoachStrings(locale)
  const { data: settings } = useSettings()
  // Défaut ON (DEC-2) : optimiste pendant le chargement des settings. Un opt-out
  // explicite renvoie false et le hint d'activation réapparaît.
  const proactiveEnabled = settings?.coach_proactive_mode ?? true

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
        <ProgressionLeadsSection
          playerSlug={playerSlug}
          onStartCampaign={hasActiveCampaign ? undefined : openStartCampaign}
        />
        <CalibratedLeversSection playerSlug={playerSlug} t={t} />
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

// ─── Pistes de progression (leviers LUSR + défis suggérés) ───────────────────

interface ProgressionLeadsSectionProps {
  playerSlug: string
  onStartCampaign?: (axis: string, axisKind: AxisKind) => void
}

function ProgressionLeadsSection({ playerSlug, onStartCampaign }: ProgressionLeadsSectionProps) {
  const { data: profile, isLoading } = usePlayerProfile(playerSlug, 30)
  if (isLoading || !profile || !profile.has_enough_data) return null
  return (
    <ProgressionSection
      leverages={profile.leverages}
      suggestions={profile.suggested_challenges}
      onStartCampaign={onStartCampaign}
    />
  )
}

// ─── Leviers calibrés (issus des patterns) ───────────────────────────────────

interface CalibratedLeversSectionProps {
  playerSlug: string
  t: ReturnType<typeof getAscensionText>
}

function CalibratedLeversSection({ playerSlug, t }: CalibratedLeversSectionProps) {
  const { data: patterns, isLoading } = usePatterns(playerSlug)
  if (isLoading) return null
  const levers = patterns?.levers ?? []
  if (levers.length === 0) return null
  return (
    <SectionShell title={t.leversSectionTitle}>
      <LeverList levers={levers} t={t} />
    </SectionShell>
  )
}
