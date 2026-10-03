/**
 * SquadObjectiveSection — la section « Objectif » de l'onglet Contributions de l'Escouade (lot
 * L3 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26, maquette C3EW) : quatre cartes, dans
 * l'ordre d'un débrief.
 *
 *   1. « Rapport de force par famille de mode » | « Rapport de force au fil de la session »,
 *      côte à côte, même hauteur (S2) ;
 *   2. « Répartition de l'objectif dans l'escouade », pleine largeur ;
 *   3. « Rapport de force, soirée après soirée », pleine largeur.
 *
 * AUCUNE REQUÊTE NEUVE : le bloc `formes_retenues` (périmètre D2), l'historique de matchs de la
 * page (résultat, score, dominance, L3.2) et `squad_objective_history` arrivent avec la réponse
 * de `useTeammates`. Sans match à objectif dans le périmètre, la section entière se retire,
 * intertitre compris.
 */
import { useCallback, useMemo } from 'react'

import { SectionTitle } from '@/components/ui/detail-section'
import type {
  MedalDigestEntry,
  SquadFormesBlock,
  SquadMatchHistoryRow,
  SquadObjectiveHistory,
} from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import { dominanceLabels } from '@/lib/narrative/dominance'

import { FORMES_CARDS_TEXT } from '../formes/cardsI18n'
import { TEAM_REST_INK, squadPlayerInk } from '../formes/colors'
import { FORMES_TEXT } from '../formes/i18n'
import { objectiveMatches } from '../formes/model/objectives'
import { buildObjectiveBalance, buildObjectiveSheets, buildSessionFil } from './objectif.logic'
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
  const hasObjective = objectiveMatches(b).length > 0
  const balance = useMemo(() => buildObjectiveBalance(b), [b])
  const fil = useMemo(() => buildSessionFil(b, matchHistory), [b, matchHistory])
  const sheets = useMemo(() => buildObjectiveSheets(b), [b])
  const dominance = useMemo(() => dominanceLabels(locale), [locale])
  const identities = useMemo<SheetIdentity[]>(() => {
    const emblems = new Map(medalDigest.map((e) => [e.player.toLowerCase(), e.emblem_url]))
    const squad = (b.squad ?? []).map((p, i) => {
      const label = p.gamertag || (p.xuid === b.main_xuid ? mainPlayerLabel : '') || p.xuid
      return { label, color: squadPlayerInk(i), emblemUrl: emblems.get(label.toLowerCase()) ?? undefined }
    })
    return [...squad, { label: t.sheets.rest, color: TEAM_REST_INK, initial: '+' }]
  }, [b, medalDigest, mainPlayerLabel, t])

  if (!hasObjective) return null

  return (
    <section className="space-y-3" data-testid="squad-objective-section">
      <SectionTitle>{t.sectionTitle}</SectionTitle>
      <div className="grid gap-4 lg:grid-cols-2">
        <ObjectiveBalanceCard families={balance} familyLabel={familyLabel} columns={columns} t={t} />
        <ObjectiveSessionFilCard
          matches={fil}
          familyLabel={familyLabel}
          dominanceLabels={dominance}
          locale={locale}
          t={t}
        />
      </div>
      <ObjectiveSheetsCard sheets={sheets} identities={identities} familyLabel={familyLabel} columns={columns} t={t} />
      {objectiveHistory && (
        <ObjectiveEveningsCard history={objectiveHistory} familyLabel={familyLabel} locale={locale} t={t} />
      )}
    </section>
  )
}
