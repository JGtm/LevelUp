/**
 * SquadObjectiveSection — l'objectif dans l'onglet Emprise de l'Escouade (arrivé de Contributions
 * le 2026-10-07 ; cartes du lot L3 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26, maquette
 * C3EW). Deux sections, dans l'ordre d'un débrief :
 *
 *   1. « Objectif » (aide du rapport de force sur l'intertitre) : les cadres « Rapport de force »
 *      par famille de mode, posés à même la section, légende dessous | « Rapport de force au fil
 *      de la session », côte à côte ; puis « Rapport de force, soirée après soirée », pleine
 *      largeur ;
 *   2. « Répartition de l'objectif dans l'escouade » (aide sur l'intertitre) : une fiche par
 *      joueur de l'escouade, à même la section comme les médailles. Pas de fiche du reste du camp.
 *
 * AUCUNE REQUÊTE NEUVE : le bloc `formes_retenues` (périmètre D2), l'historique de matchs de la
 * page (résultat, score, dominance, L3.2) et `squad_objective_history` arrivent avec la réponse
 * de `useTeammates`. Sans match à objectif dans le périmètre, les deux sections se retirent,
 * intertitres compris (`hasSquadObjective`).
 */
import { useCallback, useMemo } from 'react'

import { SectionTitle } from '@/components/ui/detail-section'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import type {
  MedalDigestEntry,
  SquadFormesBlock,
  SquadMatchHistoryRow,
  SquadObjectiveHistory,
} from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import { dominanceLabels } from '@/lib/narrative/dominance'

import { FORMES_CARDS_TEXT } from '../formes/cardsI18n'
import { FORMES_TEXT } from '../formes/i18n'
import { useSquadPlayerPalette } from '../useSquadPlayerPalette'
import { buildObjectiveBalance, buildObjectiveSheets, buildSessionFil, hasSquadObjective, squadSheetsOnly } from './objectif.logic'
import { ObjectiveBalanceCard } from './ObjectiveBalanceCard'
import { ObjectiveEveningsCard } from './ObjectiveEveningsCard'
import { ObjectiveSessionFilCard } from './ObjectiveSessionFilCard'
import { ObjectiveSheetsCard, type SheetIdentity } from './ObjectiveSheetsCard'
import { OBJECTIF_TEXT } from './objectifStrings'

interface Props {
  block: SquadFormesBlock | null | undefined
  matchHistory: SquadMatchHistoryRow[]
  objectiveHistory: SquadObjectiveHistory | null | undefined
  /** Les emblèmes des joueurs (fiches de médailles), par gamertag. */
  medalDigest: MedalDigestEntry[]
  /** Le nom du joueur de la page (`main_player`), dernier repli de son libellé. */
  mainPlayerLabel: string
  locale: Locale
}

const EMPTY_BLOCK: SquadFormesBlock = { available: false, matches_measured: 0, matches_total: 0 }

export function SquadObjectiveSection({
  block,
  matchHistory,
  objectiveHistory,
  medalDigest,
  mainPlayerLabel,
  locale,
}: Props) {
  const t = OBJECTIF_TEXT[locale]
  const columns = FORMES_TEXT[locale].columns
  const families = FORMES_CARDS_TEXT[locale].families
  const familyLabel = useCallback((f: string) => families[f] ?? f, [families])
  const b = block ?? EMPTY_BLOCK
  const balance = useMemo(() => buildObjectiveBalance(b), [b])
  const fil = useMemo(() => buildSessionFil(b, matchHistory), [b, matchHistory])
  const sheets = useMemo(() => squadSheetsOnly(buildObjectiveSheets(b)), [b])
  const dominance = useMemo(() => dominanceLabels(locale), [locale])
  const { inkOf } = useSquadPlayerPalette()
  const identities = useMemo<SheetIdentity[]>(() => {
    const emblems = new Map(medalDigest.map((e) => [e.player.toLowerCase(), e.emblem_url]))
    return (b.squad ?? []).map((p) => {
      const label = p.gamertag || (p.xuid === b.main_xuid ? mainPlayerLabel : '') || p.xuid
      return { label, color: inkOf(label), emblemUrl: emblems.get(label.toLowerCase()) ?? undefined }
    })
  }, [b, medalDigest, mainPlayerLabel, inkOf])

  if (!hasSquadObjective(b)) return null

  return (
    <>
      <section className="space-y-2" data-testid="squad-objective-section">
        <SectionTitle className="flex items-center gap-1.5">
          {t.sectionTitle}
          <InfoTooltip content={t.balance.info} />
        </SectionTitle>
        <div className="space-y-4">
          <div className="grid gap-4 lg:grid-cols-2">
            <ObjectiveBalanceCard families={balance} familyLabel={familyLabel} columns={columns} t={t} bare />
            <ObjectiveSessionFilCard
              matches={fil}
              familyLabel={familyLabel}
              dominanceLabels={dominance}
              locale={locale}
              t={t}
            />
          </div>
          {objectiveHistory && (
            <ObjectiveEveningsCard history={objectiveHistory} familyLabel={familyLabel} locale={locale} t={t} />
          )}
        </div>
      </section>
      <section className="space-y-2" data-testid="squad-objective-sheets-section">
        <SectionTitle className="flex items-center gap-1.5">
          {t.sheets.title}
          <InfoTooltip content={t.sheets.info} />
        </SectionTitle>
        <ObjectiveSheetsCard sheets={sheets} identities={identities} familyLabel={familyLabel} columns={columns} t={t} />
      </section>
    </>
  )
}
