/**
 * SessionColumnBody — corps d'une colonne de session (TOUT ce qui est sous le L3) :
 * bande KPI + pile de graphes + tableau "Détail des matchs".
 *
 * Monté À L'IDENTIQUE par la colonne principale ET par le drawer compare → garantit que
 * "ce qui est sous le L3" est strictement le même rendu des deux côtés ; seules diffèrent
 * les DONNÉES (session active vs comparée) et le header L3 (navigation vs sélecteur).
 *
 * En mode `compact` (colonne divisée, drawer ouvert) : KPI abrégés + tableau compact —
 * exactement la "vue compacte" de la colonne principale.
 */
import { Fragment, type ReactNode } from 'react'

import type {
  CoordinationBlock,
  FirstBloodPlayerSeriesDTO,
  IntensityMatchRow,
  MatchRangeBlock,
  RangeReferenceBlock,
  SessionCompareEntry,
  SessionDetailMatchRow,
  SessionUsageBlock,
} from '@/lib/api/types'

import type { CompareScale } from './_compareScale'
import { SESSION_SECTION_ORDER, type SessionSectionKey } from './_sections'
import { useSessionChartSections } from './_chartSections'
import { SessionCoordinationSection } from './SessionCoordinationSection'
import { SessionMatchesTable } from './SessionMatchesTable'
import { SessionRangeCard } from './SessionRangeCard'
import { SessionSummaryCard } from './SessionSummaryCard'
import { SessionUsageSection } from './SessionUsageSection'
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
   * la section n'existe pas de ce côté.
   */
  usage?: SessionUsageBlock
  /**
   * Bloc « Coordination » (riposte + appui reçu, lot N1) — `coordination` à gauche,
   * `compare_coordination` à droite (lot S) : les deux colonnes ont leurs cartes, avec
   * leurs propres données. Absent du payload → la section n'existe pas de ce côté.
   */
  coordination?: CoordinationBlock
  /**
   * Profils de PORTÉE par match (lot N2) — `range_profiles` à gauche,
   * `compare_range_profiles` à droite : les deux colonnes ont leur carte.
   */
  rangeProfiles?: MatchRangeBlock
  /**
   * Période de RÉFÉRENCE de la portée (`range_reference`, lot U) — l'axe du nuage. UN SEUL
   * bloc pour les deux colonnes : la référence dépend du filtre de la page, pas de la
   * session affichée ; chaque colonne y surligne SA session (lot W, D23-4).
   */
  rangeReference?: RangeReferenceBlock | null
  /**
   * Mode COMPARAISON (D16) : rangées partagées avec la colonne sœur. La page passe
   * l'union ordonnée des clés des deux colonnes ; chaque clé rend ici soit la section,
   * soit le placeholder « Sans équivalent dans cette session ». Absent → pile simple
   * (vue pleine page), rendu strictement identique à avant.
   */
  rowKeys?: readonly SessionSectionKey[]
}

/** Sections de la colonne, indexees par cle stable (`_sections.ts`). */
function useSessionColumnSections(props: Props): Partial<Record<SessionSectionKey, ReactNode>> {
  const t = useSessionT()
  const {
    entry,
    matches,
    playerSlug,
    compact,
    participationSide = 'right',
    usage,
    coordination,
    rangeProfiles,
    rangeReference,
  } = props

  const chartSections = useSessionChartSections({
    entry,
    matches,
    compact,
    participationSide,
    participationColor: 'compare-a',
    scale: props.scale,
    intensityRows: props.intensityRows,
    firstBlood: props.firstBlood,
  })

  return {
    summary: <SessionSummaryCard entry={entry} compact={compact} />,
    ...chartSections,
    // Blocs « usages d'équipement, armes spéciales et objectifs ». `compact` suit la
    // colonne : drawer ouvert = version compacte DES DEUX CÔTÉS, sinon les deux
    // colonnes ne se compareraient pas. Le composant gère lui-même ses états
    // indisponible / sans film ; absent du payload → la section n'existe pas.
    ...(usage
      ? { usage: <SessionUsageSection usage={usage} meLabel={playerSlug} compact={compact} /> }
      : {}),
    // Sections transverses de la vague 3 (D22) : la Coordination (deux cartes en rangée)
    // puis la Portée (une carte). Deux clés distinctes : ce sont deux rangées partagées,
    // et la comparaison sert désormais les deux (lot S).
    ...(coordination
      ? {
          coordination: (
            <SessionCoordinationSection coordination={coordination} compact={compact} />
          ),
        }
      : {}),
    ...(rangeProfiles
      ? {
          range: (
            <SessionRangeCard
              block={rangeProfiles}
              reference={rangeReference}
              meLabel={playerSlug}
              compact={compact}
              yDomain={props.scale?.rangeDelta}
            />
          ),
        }
      : {}),
    // Tableau "Détail des matchs" — hors bloc/Card (juste un titre + le tableau).
    matches: (
      <div className="space-y-3">
        <h2 className="text-base font-semibold text-foreground">{t('session.detail.matches_card')}</h2>
        <SessionMatchesTable
          matches={matches}
          playerSlug={playerSlug}
          variant={compact ? 'compact' : 'full'}
          withFriends={entry?.with_friends ?? false}
        />
      </div>
    ),
  }
}

/**
 * Placeholder D16 — la colonne soeur porte une section que celle-ci n'a pas. On garde
 * la rangee (les blocs suivants restent alignes) avec un cadre vide explicite.
 */
function SessionSectionPlaceholder() {
  const t = useSessionT()
  return (
    <div
      className="flex h-full min-h-24 flex-col items-center justify-center gap-2 rounded-lg border border-dashed border-border p-6 text-center"
      data-testid="session-section-placeholder"
    >
      <span aria-hidden="true" className="text-lg text-muted-foreground">
        —
      </span>
      <p className="text-sm text-muted-foreground">{t('session.detail.compare_no_counterpart')}</p>
    </div>
  )
}

export function SessionColumnBody(props: Props) {
  const sections = useSessionColumnSections(props)
  const { rowKeys } = props

  // Vue pleine page : pile simple, dans l'ordre canonique.
  if (!rowKeys) {
    return (
      <>
        {SESSION_SECTION_ORDER.filter((key) => key in sections).map((key) => (
          <Fragment key={key}>{sections[key]}</Fragment>
        ))}
      </>
    )
  }

  // Vue comparaison : une rangee de grille par cle — la colonne soeur emet les MEMES
  // cles dans le MEME ordre, donc la i-eme section de gauche et celle de droite
  // partagent la rangee (donc la hauteur, donc la ligne de titre).
  return (
    <>
      {rowKeys.map((key) => (
        <div
          key={key}
          data-session-section={key}
          className="min-w-0 [&>*:only-child]:h-full"
        >
          {key in sections ? sections[key] : <SessionSectionPlaceholder />}
        </div>
      ))}
    </>
  )
}
