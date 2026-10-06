/**
 * useSessionEmpriseCards — LES CARTES A à L de « Frags et usages » d'une colonne de session (plan
 * PLAN_SESSIONS_EMPRISE_2026-10-06, §3 et S4.7), indexées par leur clé de section (`_sections.ts`).
 *
 * Les briques de l'Escouade et des Séries temporelles, montées avec les modèles de la colonne
 * (`sessionEmprise.logic.ts`) et le jeu de textes de SA vue (`sessionEmpriseText.ts`) : pleine page,
 * ou vue compacte du tiroir de comparaison — `compact` est transmis à CHAQUE carte, des deux côtés.
 * Une carte n'est rendue que si `sessionCardsPresence` la dit présente : la même lecture que les
 * rangées partagées de la page et que les intertitres de la colonne.
 */
import { useCallback, useMemo, type ReactNode } from 'react'

import { WeaponAccuracyChart } from '@/components/charts/WeaponAccuracyChart'
import { equipmentFamilyLabel, USAGE_TEXT } from '@/features/_shared/usage/usageI18n'
import { empriseObjectName } from '@/features/squad/emprise/objectName'
import { ProductionCard } from '@/features/squad/emprise/ProductionCard'
import { ResourceControlCard } from '@/features/squad/emprise/ResourceControlCard'
import { ResourceFilCard } from '@/features/squad/emprise/ResourceFilCard'
import { ResourceMatchGridCard } from '@/features/squad/emprise/ResourceMatchGridCard'
import { useOutcomeLabels } from '@/features/squad/emprise/useOutcomeLabels'
import { YieldCard } from '@/features/squad/emprise/YieldCard'
import { FORMES_CARDS_TEXT } from '@/features/squad/formes/cardsI18n'
import { FORMES_TEXT, type FormesText } from '@/features/squad/formes/i18n'
import { ObjectiveBalanceCard } from '@/features/squad/objectif/ObjectiveBalanceCard'
import { ObjectiveSoloSheetCard } from '@/features/squad/objectif/ObjectiveSoloSheetCard'
import { EquipmentOutcomesCard } from '@/features/timeseries/usages/EquipmentOutcomesCard'
import { LivesNearTeammateCard } from '@/features/timeseries/usages/LivesNearTeammateCard'
import { MinePickupsCard } from '@/features/timeseries/usages/MinePickupsCard'
import type { SquadEmpriseObject } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import { dominanceLabels } from '@/lib/narrative/dominance'

import type { DominanceValue, OutcomeValue } from '@/components/charts/outcomeSequence'

import {
  buildSessionEmpriseModels,
  sessionCardsPresence,
  sessionPlayerName,
  type SessionCardsPresence,
  type SessionColumnBlocks,
  type SessionEmpriseModels,
} from './sessionEmprise.logic'
import { SESSION_CARD_TEXT, type SessionCardTexts, type SessionCompactCards } from './sessionEmpriseText'
import { SessionFragBarCard } from './SessionFragBarCard'
import { SessionToolsCard } from './SessionToolsCard'

/** Hauteur de « Précision par arme » (B'), celle des cartes de frags d'avant (`ChartCard`). */
const ACCURACY_HEIGHT = 320

interface SessionEmpriseCards {
  cards: Partial<Record<keyof SessionCardsPresence, ReactNode>>
  /** « 6 matchs filmés sur 7 · … » sous « Ressources de la soirée », en pleine page ; null sans Emprise. */
  coverage: string | null
}

export function useSessionEmpriseCards(
  col: SessionColumnBlocks,
  compact: boolean,
  fallbackName: string,
  locale: Locale,
): SessionEmpriseCards {
  const view = SESSION_CARD_TEXT[locale]
  const texts = compact ? view.compact : view.full
  const cc = compact ? view.compactCards : undefined
  const usageText = USAGE_TEXT[locale]
  const unknownVehicle = texts.emprise.vehicles.unknown
  const outcomeLabels = useOutcomeLabels()
  const dominance = useMemo(() => dominanceLabels(locale), [locale])

  // Le nom d'un objet : la source unique de l'Emprise (squad/emprise/objectName.ts).
  const objectName = useCallback(
    (o: SquadEmpriseObject) => empriseObjectName(o, usageText, unknownVehicle),
    [usageText, unknownVehicle],
  )
  const { entry, matches, emprise, lives, formes } = col
  const models = useMemo(
    () => buildSessionEmpriseModels({ entry, matches, emprise, lives, formes }, objectName),
    [entry, matches, emprise, lives, formes, objectName],
  )
  const present = useMemo(() => sessionCardsPresence({ entry, matches, emprise }, models), [entry, matches, emprise, models])
  const player = sessionPlayerName(col, fallbackName)
  const playerName = useCallback(
    (xuid: string) => emprise?.players?.find((p) => p.xuid === xuid)?.gamertag ?? '',
    [emprise],
  )
  const families = FORMES_CARDS_TEXT[locale].families
  const familyLabel = useCallback((f: string) => families[f] ?? f, [families])
  const columns = FORMES_TEXT[locale].columns
  const equipmentLabel = useCallback(
    (family: string) => texts.cards.equipment.unmeasuredNames[family] ?? equipmentFamilyLabel(family, usageText),
    [texts, usageText],
  )
  const sheetName = useMemo(() => {
    const main = models.objective?.main_xuid ?? ''
    return models.objective?.squad?.find((p) => p.xuid === main)?.gamertag || player
  }, [models.objective, player])

  const ctx: CardsContext = {
    col, m: models, t: texts, cc, compact, locale, player, sheetName, objectName, playerName, familyLabel, columns, equipmentLabel,
    dominance, outcomeLabels,
  }
  const render = cardRenderers(ctx)
  const cards: SessionEmpriseCards['cards'] = {}
  for (const key of Object.keys(render) as (keyof SessionCardsPresence)[]) {
    if (present[key]) cards[key] = render[key]()
  }
  return { cards, coverage: emprise ? texts.coverage(models.coverage.filmed, models.coverage.total) : null }
}

interface CardsContext {
  col: SessionColumnBlocks
  m: SessionEmpriseModels
  t: SessionCardTexts
  cc: SessionCompactCards | undefined
  compact: boolean
  locale: Locale
  player: string
  sheetName: string
  objectName: (o: SquadEmpriseObject) => string
  playerName: (xuid: string) => string
  familyLabel: (f: string) => string
  columns: FormesText['columns']
  equipmentLabel: (family: string) => string
  dominance: Record<DominanceValue, string>
  outcomeLabels: Record<OutcomeValue, string>
}

/** Le rendu de chaque carte, dans l'ordre de la page ; appelé seulement pour une carte présente. */
function cardRenderers(x: CardsContext): Record<keyof SessionCardsPresence, () => ReactNode> {
  const { col, m, t, cc, compact, locale } = x
  const entry = col.entry
  return {
    frag_bar: () => <SessionFragBarCard entry={entry} player={x.player} locale={locale} texts={t} compact={cc?.frag} />,
    tools: () => <SessionToolsCard tools={entry?.weapon_tools} player={x.player} locale={locale} texts={t} compact={compact} />,
    weapon_accuracy: () => (
      <WeaponAccuracyChart weapons={entry?.weapon_accuracy ?? []} weaponKills={entry?.top_weapon_kills ?? []} height={ACCURACY_HEIGHT} />
    ),
    control: () => <ResourceControlCard rows={m.controlRows} t={t.emprise} compact={compact} />,
    fil: () =>
      m.fil && (
        <ResourceFilCard fil={m.fil} dominanceLabels={x.dominance} outcomeLabels={x.outcomeLabels} locale={locale} t={t.emprise} compact={compact} />
      ),
    grid: () =>
      m.grid && (
        <ResourceMatchGridCard
          grid={m.grid}
          itemName={(row) => (row.object ? x.objectName(row.object) : '')}
          playerName={x.playerName}
          dominanceLabels={x.dominance}
          outcomeLabels={x.outcomeLabels}
          locale={locale}
          t={t.emprise}
          compact={compact}
        />
      ),
    mine: () => m.mine && <MinePickupsCard mine={m.mine} itemName={x.objectName} t={t.emprise} ut={t.cards} compact={cc?.mine} />,
    production: () => <ProductionCard rows={m.production} t={t.emprise} compact={cc?.production} />,
    yield: () => <YieldCard rows={m.yieldRows} coverage={m.vehicleCoverage} t={t.emprise} />,
    lives: () => m.lives && <LivesNearTeammateCard model={m.lives} ut={t.cards} compact={cc?.lives} />,
    objective_balance: () => (
      <ObjectiveBalanceCard families={m.balance} familyLabel={x.familyLabel} columns={x.columns} t={t.objectif} compact={compact} />
    ),
    objective_sheet: () =>
      m.soloSheet && (
        <ObjectiveSoloSheetCard
          sheet={m.soloSheet}
          name={x.sheetName}
          emblemUrl={col.emblemUrl}
          familyLabel={x.familyLabel}
          columns={x.columns}
          t={t.sheet}
          compact={compact}
        />
      ),
    equipment: () => <EquipmentOutcomesCard rows={m.equipment} familyLabel={x.equipmentLabel} ut={t.cards} compact={cc?.equipment} />,
  }
}
