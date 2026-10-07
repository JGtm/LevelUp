/**
 * SessionColumnBody — corps d'une colonne de session (TOUT ce qui est sous le L3) :
 * bande KPI, puis QUATRE SECTIONS TITRÉES — « Bilan », « Match par match »,
 * « Frags et usages », « Détail des matchs ».
 *
 * CE COMPOSANT POSE LES TITRES, MAIS NE DÉCIDE PAS DE LEUR PLACE : chaque carte est une
 * CLÉ stable (`_sections.ts`), et ce sont les tables de ce module qui disent quelle clé ouvre
 * quel titre de groupe et quel intertitre de sous-groupe (« Ressources de la soirée »,
 * « Prendre, et s'en servir »…). Les deux rendus d'ici — pile pleine page, rangées partagées
 * de la comparaison (D16) — lisent LES MÊMES tables : un titre ne peut donc pas se poser d'un
 * côté et pas de l'autre, ce qui décalerait les deux colonnes d'une ligne.
 *
 * UN TITRE NE SE POSE JAMAIS AU-DESSUS DE RIEN : une carte n'a de clé que si elle dessine
 * quelque chose (`sessionCardsPresence`, le MÊME prédicat que la page lit pour ses rangées) —
 * un groupe ou un sous-groupe sans carte disparaît, titre compris.
 *
 * PLEINE PAGE : les paires A|B, C|D, G|H partagent une rangée (`pairGridClass`) ; l'intertitre
 * « Ressources de la soirée » porte sa couverture. COMPARAISON : une rangée par carte, chaque
 * colonne en vue compacte, la carte absente d'un côté remplacée par le marqueur « Sans équivalent
 * dans cette session ».
 *
 * LA BANDE KPI RESTE SANS TITRE, comme le Home : c'est l'en-tête de la colonne, pas une
 * section de plus. « Détail des matchs » porte le sien avec son tableau (`space-y-3`).
 *
 * Monté À L'IDENTIQUE par la colonne principale ET par le drawer compare → garantit que
 * "ce qui est sous le L3" est strictement le même rendu des deux côtés ; seules diffèrent
 * les DONNÉES (session active vs comparée) et le header L3 (navigation vs sélecteur).
 */
import { Fragment, type ReactNode } from 'react'

import type { FirstBloodPlayerSeriesDTO, IntensityMatchRow } from '@/lib/api/types'

import { DetailSection, SectionTitle } from '@/components/ui/detail-section'
import { useAppShellStore } from '@/stores/appShellStore'

import type { CompareScale } from './_compareScale'
import {
  SESSION_GROUP_TITLE_KEY,
  SESSION_SECTION_ORDER,
  SESSION_SUBGROUP_TITLE_KEY,
  groupSessionSections,
  pairSessionKeys,
  sessionRowOpenings,
  type SessionSectionKey,
  type SessionSubgroup,
} from './_sections'
import { pairGridClass, useSessionChartSections } from './_chartSections'
import { SessionCoordinationSection } from './SessionCoordinationSection'
import type { SessionColumnBlocks } from './sessionEmprise.logic'
import { SessionMatchesTable } from './SessionMatchesTable'
import { SessionRangeCard } from './SessionRangeCard'
import { SessionSummaryCard } from './SessionSummaryCard'
import { useSessionEmpriseCards } from './useSessionEmpriseCards'
import { useSessionT } from './_shared'

interface Props {
  /** Les blocs de CETTE colonne (`sessionColumnBlocks`) : session, matchs, Emprise, vies, objectif… */
  blocks: SessionColumnBlocks
  playerSlug: string
  /** Colonne divisée (drawer ouvert) → KPI abrégés, tableau compact, cartes en vue compacte. */
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
   * Mode COMPARAISON (D16) : rangées partagées avec la colonne sœur. La page passe
   * l'union ordonnée des clés des deux colonnes ; chaque clé rend ici soit la section,
   * soit le placeholder « Sans équivalent dans cette session ». Absent → pile simple
   * (vue pleine page).
   */
  rowKeys?: readonly SessionSectionKey[]
}

/** Sections de la colonne, indexées par clé stable (`_sections.ts`), et la couverture des ressources. */
function useSessionColumnSections(props: Props): {
  sections: Partial<Record<SessionSectionKey, ReactNode>>
  coverage: string | null
} {
  const t = useSessionT()
  const locale = useAppShellStore((s) => s.locale)
  const { blocks, playerSlug, compact, participationSide = 'right' } = props
  const { entry, matches, coordination, rangeProfiles } = blocks

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
  // SECTION « Frags et usages » — les cartes A à L, chacune sa clé, chacune en vue compacte quand
  // la colonne l'est (drawer ouvert = vue compacte DES DEUX CÔTÉS).
  const emprise = useSessionEmpriseCards(blocks, compact, playerSlug, locale)

  const sections: Partial<Record<SessionSectionKey, ReactNode>> = {
    summary: <SessionSummaryCard entry={entry} compact={compact} />,
    ...chartSections,
    ...emprise.cards,
    // Sections transverses de la vague 3 (D22) : « Appui reçu » puis la Portée. Deux clés
    // distinctes : ce sont deux rangées partagées, et la comparaison sert les deux (lot S).
    ...(coordination
      ? { coordination: <SessionCoordinationSection coordination={coordination} compact={compact} /> }
      : {}),
    ...(rangeProfiles
      ? {
          range: (
            <SessionRangeCard
              block={rangeProfiles}
              reference={blocks.rangeReference}
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
        <SectionTitle>{t('session.detail.matches_card')}</SectionTitle>
        <SessionMatchesTable
          matches={matches}
          playerSlug={playerSlug}
          variant={compact ? 'compact' : 'full'}
          withFriends={entry?.with_friends ?? false}
        />
      </div>
    ),
  }
  return { sections, coverage: emprise.coverage }
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

/** L'intertitre d'un sous-groupe (`h4`, sous le titre de groupe `h3`), sa couverture en petit à côté. */
function SessionSubgroupTitle({ subgroup, sub, className }: { subgroup: SessionSubgroup; sub?: string | null; className?: string }) {
  const t = useSessionT()
  return (
    <h4 className={`text-sm font-semibold text-foreground${className ? ` ${className}` : ''}`} data-session-subgroup={subgroup}>
      {t(SESSION_SUBGROUP_TITLE_KEY[subgroup])}
      {sub && <small className="ml-2 text-xs font-normal text-muted-foreground">{sub}</small>}
    </h4>
  )
}

/** Les cartes d'une suite de clés en pleine page : les paires A|B, C|D, G|H sur une rangée. */
function PairedRows({ keys, sections, compact }: { keys: SessionSectionKey[]; sections: Partial<Record<SessionSectionKey, ReactNode>>; compact: boolean }) {
  return (
    <>
      {pairSessionKeys(keys).map((pair) =>
        pair.length > 1 ? (
          <div key={pair.join('|')} className={pairGridClass(compact)} data-session-pair={pair.join('|')}>
            {pair.map((key) => (
              <Fragment key={key}>{sections[key]}</Fragment>
            ))}
          </div>
        ) : (
          <Fragment key={pair[0]}>{sections[pair[0]]}</Fragment>
        ),
      )}
    </>
  )
}

export function SessionColumnBody(props: Props) {
  const t = useSessionT()
  const { sections, coverage } = useSessionColumnSections(props)
  const { rowKeys, compact } = props

  // Vue pleine page : pile simple, dans l'ordre canonique, chaque GROUPE de clés coiffé
  // de son titre, chaque sous-groupe de son intertitre (`_sections.ts`).
  if (!rowKeys) {
    const presentes = SESSION_SECTION_ORDER.filter((key) => key in sections)
    return (
      <>
        {groupSessionSections(presentes).map((run) =>
          run.group == null ? (
            run.keys.map((key) => <Fragment key={key}>{sections[key]}</Fragment>)
          ) : (
            <DetailSection key={run.group} title={t(SESSION_GROUP_TITLE_KEY[run.group])}>
              <div className="space-y-6">
                {run.subruns.map((sub) =>
                  sub.subgroup == null ? (
                    <PairedRows key={sub.keys[0]} keys={sub.keys} sections={sections} compact={compact} />
                  ) : (
                    <div key={sub.subgroup} className="space-y-3">
                      <SessionSubgroupTitle subgroup={sub.subgroup} sub={sub.subgroup === 'resources' ? coverage : null} />
                      <div className="space-y-6">
                        <PairedRows keys={sub.keys} sections={sections} compact={compact} />
                      </div>
                    </div>
                  ),
                )}
              </div>
            </DetailSection>
          ),
        )}
      </>
    )
  }

  // Vue comparaison : une rangee de grille par cle — la colonne soeur emet les MEMES
  // cles dans le MEME ordre, donc la i-eme section de gauche et celle de droite
  // partagent la rangee (donc la hauteur, donc la ligne de titre). Le titre de groupe ET
  // l'intertitre de sous-groupe s'écrivent dans la rangée de leur PREMIÈRE clé : `rowKeys`
  // étant identique des deux côtés, les deux colonnes les posent à la même rangée. Aucune
  // couverture en petit dans cette vue (maquette).
  //
  // LA CARTE S'ÉTIRE DANS CE QUI RESTE SOUS LES TITRES, JAMAIS DANS TOUTE LA RANGÉE. Les cartes
  // de « Frags et usages » portent `h-full` (paires à hauteur égale en pleine page) : posées
  // directement sous un titre, leurs 100 % valaient la rangée ENTIÈRE, et la carte débordait
  // de la hauteur du titre sur la rangée suivante (légendes recouvertes, titres de section
  // survolés). La rangée est une colonne flexible ; la case du contenu prend le reste
  // (`flex-1`), et c'est elle que la carte remplit.
  const ouvertures = sessionRowOpenings(rowKeys)
  return (
    <>
      {rowKeys.map((key) => {
        const open = ouvertures.get(key)
        return (
          <div key={key} data-session-section={key} className="flex min-w-0 flex-col">
            {open?.group && <SectionTitle className="mb-4">{t(SESSION_GROUP_TITLE_KEY[open.group])}</SectionTitle>}
            {open?.subgroup && <SessionSubgroupTitle subgroup={open.subgroup} className="mb-3" />}
            <div className="min-h-0 min-w-0 flex-1 [&>*:only-child]:h-full">
              {key in sections ? sections[key] : <SessionSectionPlaceholder />}
            </div>
          </div>
        )
      })}
    </>
  )
}
