/**
 * SessionColumnBody — corps d'une colonne de session (TOUT ce qui est sous le L3) :
 * bande KPI, puis QUATRE SECTIONS TITRÉES — « Bilan » et « Match par match » (montées par
 * `SessionChartStack`), « Frags et usages » et « Détail des matchs » (montées ici, parce
 * qu'elles coiffent des blocs que ce composant est seul à monter).
 *
 * LA BANDE KPI RESTE SANS TITRE, comme le Home : c'est l'en-tête de la colonne, pas une
 * section de plus.
 *
 * Monté À L'IDENTIQUE par la colonne principale ET par le drawer compare → garantit que
 * "ce qui est sous le L3" est strictement le même rendu des deux côtés ; seules diffèrent
 * les DONNÉES (session active vs comparée) et le header L3 (navigation vs sélecteur).
 *
 * En mode `compact` (colonne divisée, drawer ouvert) : KPI abrégés + tableau compact —
 * exactement la "vue compacte" de la colonne principale.
 */
import type {
  FirstBloodPlayerSeriesDTO,
  IntensityMatchRow,
  SessionCompareEntry,
  SessionDetailMatchRow,
  SessionUsageBlock,
} from '@/lib/api/types'

import { DetailSection, SectionTitle } from '@/components/ui/detail-section'

import type { CompareScale } from './_compareScale'
import { SessionChartStack } from './SessionChartStack'
import { SessionFragCard } from './SessionFragCard'
import { SessionMatchesTable } from './SessionMatchesTable'
import { SessionSummaryCard } from './SessionSummaryCard'
import { SessionUsageSection } from './SessionUsageSection'
import {
  sessionFragCardHasContent,
  sessionUsageShowsSomething,
} from './sessionSectionVisibility'
import { useSessionT } from './_shared'

interface Props {
  entry: SessionCompareEntry | null
  matches: SessionDetailMatchRow[]
  playerSlug: string
  /** Colonne divisée (drawer ouvert) → KPI abrégés + tableau compact + donuts en % interne. */
  compact: boolean
  /** Côté de l'axe du profil de participation : 'right' (colonne principale) / 'left' (drawer). */
  participationSide?: 'left' | 'right'
  /** Bornes d'axe partagées A/B (mode comparaison) — fige les échelles pour comparabilité. */
  scale?: CompareScale
  /** Profil d'intensité (frags par phase) de la session — calculé côté Go (payload). */
  intensityRows?: IntensityMatchRow[]
  /** Premiers frag/mort par match de la session — calculés côté Go (payload). */
  firstBlood?: FirstBloodPlayerSeriesDTO[]
  /**
   * Bloc « usages d'équipement, armes spéciales et objectifs » — servi pour LES DEUX
   * sessions depuis le 2026-09-09 (D8) : la colonne principale reçoit `usage`, le
   * drawer `compare_usage`. Absent du payload (vieux serveur, session sans match) →
   * le composant ne rend rien, pas de bloc fantôme.
   */
  usage?: SessionUsageBlock
}

export function SessionColumnBody({
  entry,
  matches,
  playerSlug,
  compact,
  participationSide = 'right',
  scale,
  intensityRows,
  firstBlood,
  usage,
}: Props) {
  const t = useSessionT()
  // SECTION 3 — la carte des frags et les cartes d'usage se masquent chacune selon la
  // donnée : sans aucune des deux, c'est la SECTION ENTIÈRE qui disparaît, titre compris.
  // Un titre au-dessus de rien annoncerait une mesure qui n'existe pas.
  const showKillsAndUsage =
    sessionFragCardHasContent(entry) || sessionUsageShowsSomething(usage)

  return (
    <>
      <SessionSummaryCard entry={entry} compact={compact} />

      <SessionChartStack
        entry={entry}
        matches={matches}
        compact={compact}
        participationSide={participationSide}
        participationColor="compare-a"
        scale={scale}
        intensityRows={intensityRows}
        firstBlood={firstBlood}
      />

      {/* SECTION 3 — « Frags et usages » : d'où viennent les frags, et ce qu'on a ramassé,
          posé, tenu pour les obtenir. Les blocs « usages d'équipement, armes spéciales et
          objectifs » gèrent eux-mêmes leurs états indisponible / sans film ; `compact` suit
          la colonne (drawer ouvert = version compacte DES DEUX CÔTÉS, sinon les deux
          colonnes ne se compareraient pas). */}
      {showKillsAndUsage && (
        <DetailSection title={t('session.detail.section_kills_usage')}>
          <div className="space-y-6">
            <SessionFragCard entry={entry} stacked={compact} />
            <SessionUsageSection usage={usage} meLabel={playerSlug} compact={compact} />
          </div>
        </DetailSection>
      )}

      {/* SECTION 4 — « Détail des matchs », hors bloc/Card (juste un titre + le tableau).
          `space-y-3` conservé : le tableau se colle plus à son titre que des graphes. */}
      <div className="space-y-3">
        <SectionTitle>{t('session.detail.matches_card')}</SectionTitle>
        <SessionMatchesTable
          matches={matches}
          playerSlug={playerSlug}
          variant={compact ? 'compact' : 'full'}
          withFriends={entry?.with_friends ?? false}
        />
      </div>
    </>
  )
}
