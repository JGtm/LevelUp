/**
 * AscensionProfilTab — onglet "Profil" (index) : qui tu es en jeu.
 *
 * Restructuration 4 onglets (2026-07, DEC-3). Compose l'identité de jeu
 * (PlayerProfileV3 : identité/radar, style & engagement, performance) et les
 * tendances par contexte (patterns contextuels, solo vs escouade,
 * comportements détectés). Les leviers/pistes de progression et le coaching
 * vivent dans l'onglet "Entraînement" ; la couche Prestige dans "Objectifs".
 */
import { useAppShellStore } from '@/stores/appShellStore'
import { PlayerProfileV3 } from './profile/PlayerProfileV3'
import { usePatterns } from './queries'
import { getAscensionText } from './i18n'
import { PatternContextGrid } from './PatternContextGrid'
import { SquadVsSoloCard } from './SquadVsSoloCard'
import { BehaviorAlertList } from './BehaviorAlertList'
import { LayerSection, SectionShell } from './AscensionLayers'

export function AscensionProfilTab() {
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const playerSlug = currentPlayer?.player_slug ?? ''
  const locale = useAppShellStore((s) => s.locale)
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const t = getAscensionText(locale)

  if (!playerSlug) {
    return (
      <p className="p-6 text-sm text-muted-foreground">
        {t.profileSelectPlayer}
      </p>
    )
  }

  return (
    <div className="space-y-10">
      <LayerSection title={t.profilLayerTitle} description={t.profilLayerDescription}>
        <PlayerProfileV3 playerSlug={playerSlug} />
        <ProfilePatternsSection playerSlug={playerSlug} titleSlug={titleSlug} t={t} locale={locale} />
      </LayerSection>
    </div>
  )
}

// ─── Patterns contextuels + comportementaux (sans leviers) ───────────────────

interface ProfilePatternsSectionProps {
  playerSlug: string
  titleSlug: string
  t: ReturnType<typeof getAscensionText>
  locale: 'fr' | 'en'
}

function ProfilePatternsSection({ playerSlug, titleSlug, t, locale }: ProfilePatternsSectionProps) {
  const { data: patterns, isLoading } = usePatterns(playerSlug)
  if (isLoading) return null
  const contextPatterns = patterns?.context_patterns ?? []
  const behaviorPatterns = patterns?.behavior_patterns ?? []

  if (contextPatterns.length === 0 && behaviorPatterns.length === 0) {
    return null
  }

  return (
    <div className="space-y-6">
      {contextPatterns.length > 0 && (
        <SectionShell title={t.patternsSectionTitle}>
          <PatternContextGrid
            patterns={contextPatterns}
            t={t}
            minMatchesForSignal={patterns?.min_matches_for_signal ?? 0}
            playerSlug={playerSlug}
            titleSlug={titleSlug}
          />
          <SquadVsSoloCard patterns={contextPatterns} t={t} locale={locale} />
        </SectionShell>
      )}
      {behaviorPatterns.length > 0 && (
        <SectionShell title={t.behaviorsSectionTitle}>
          <BehaviorAlertList patterns={behaviorPatterns} t={t} />
        </SectionShell>
      )}
    </div>
  )
}
