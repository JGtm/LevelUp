/**
 * useMatchEmprise — LES CARTES DE L'EMPRISE de l'onglet « Armes et terrain » (plan
 * PLAN_MATCHVIEW_EMPRISE_2026-10-06, cartes D, E, G, H, I) : les modèles (`matchEmprise.logic.ts`),
 * les textes (`matchEmpriseText.ts`) et la palette des joueurs du match, assemblés une fois ; chaque
 * carte n'est rendue que si le prédicat de présence la dit présente — la même lecture que l'onglet
 * pour poser l'intertitre. « Prises par joueur » est posé à même la section, comme sur l'Emprise de
 * l'Escouade : intertitre avec son aide, fiches sans cadre.
 */
import { useCallback, useMemo, type ReactNode } from 'react'

import { SectionTitle } from '@/components/ui/detail-section'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { USAGE_TEXT } from '@/features/_shared/usage/usageI18n'
import { empriseObjectName } from '@/features/squad/emprise/objectName'
import { PickupSheetsCard, type PickupIdentity } from '@/features/squad/emprise/PickupSheetsCard'
import { ProductionCard } from '@/features/squad/emprise/ProductionCard'
import { YieldCard } from '@/features/squad/emprise/YieldCard'
import type { MatchEmpriseBlock, MatchLivesNearTeammate, MatchRosterRow, MatchScoreboardRow, SquadEmpriseObject } from '@/lib/api/types'
import { resolveToken } from '@/lib/accessibility'
import type { Locale } from '@/lib/i18n/locale'

import { buildMatchPlayerColors } from './colors'
import {
  buildMatchEmpriseModels,
  matchCoverage,
  matchEmpriseCards,
  type MatchEmpriseCards,
  type MatchProductionPending,
  type MatchYieldPending,
} from './matchEmprise.logic'
import { MATCH_EMPRISE_TEXT, type MatchOwnText } from './matchEmpriseText'
import { MatchLivesCard } from './MatchLivesCard'
import { MatchResourceControlCard } from './MatchResourceControlCard'

interface Input {
  emprise: MatchEmpriseBlock | null | undefined
  lives: MatchLivesNearTeammate | null | undefined
  scoreboard: MatchScoreboardRow[]
  roster: MatchRosterRow[]
  meXUID: string | null
  friendGamertags: readonly string[]
  locale: Locale
}

interface MatchEmprise {
  present: MatchEmpriseCards
  /** « film décodé · 8 joueurs présents à la fin » ; null sans film. */
  coverage: string | null
  control: ReactNode
  sheets: ReactNode
  production: ReactNode
  yieldCard: ReactNode
  lives: ReactNode
}

export function useMatchEmprise(x: Input): MatchEmprise {
  const texts = MATCH_EMPRISE_TEXT[x.locale]
  const usageText = USAGE_TEXT[x.locale]
  const unknownVehicle = texts.emprise.vehicles.unknown
  const objectName = useCallback((o: SquadEmpriseObject) => empriseObjectName(o, usageText, unknownVehicle), [usageText, unknownVehicle])
  const models = useMemo(() => buildMatchEmpriseModels(x.emprise, x.lives, objectName), [x.emprise, x.lives, objectName])
  const present = useMemo(() => matchEmpriseCards(models), [models])
  const palette = useMemo(
    () => buildMatchPlayerColors(x.scoreboard, x.meXUID, x.friendGamertags, x.roster),
    [x.scoreboard, x.meXUID, x.friendGamertags, x.roster],
  )
  const inkOf = useCallback((xuid: string) => palette.hexByXUID.get(xuid) ?? resolveToken('team-ally'), [palette])
  const identities = useMemo<PickupIdentity[]>(
    () => (models.sheets?.owners ?? []).map((o) => ({ label: o.gamertag, color: inkOf(o.xuid ?? '') })),
    [models.sheets, inkOf],
  )
  const cov = matchCoverage(x.emprise, x.scoreboard)
  const own = texts.own
  return {
    present,
    coverage: cov ? own.coverage(cov.present) : null,
    control: present.control && <MatchResourceControlCard control={models.control} objectName={objectName} t={texts.emprise} own={own} />,
    sheets: present.sheets && models.sheets && (
      <section className="space-y-2" data-testid="match-emprise-sheets-section">
        <SectionTitle className="flex items-center gap-1.5">
          {texts.emprise.sheets.title}
          <InfoTooltip content={texts.emprise.sheets.info} />
        </SectionTitle>
        <PickupSheetsCard sheets={models.sheets} identities={identities} itemName={(line) => objectName(line.object)} bare t={texts.emprise} />
      </section>
    ),
    production: present.production && (
      <ProductionCard
        rows={models.production.rows}
        pending={models.production.pending.map((p) => productionPending(p, own))}
        notes={Object.fromEntries(models.production.noPickupNote.map((r) => [r, own.production.powerNoPickup]))}
        t={texts.emprise}
      />
    ),
    yieldCard: present.yield && (
      <YieldCard rows={models.yield.rows} pending={models.yield.pending.map((p) => yieldPending(p, own, texts.emprise.production.exposure.effect_ms.fmt))} t={texts.emprise} />
    ),
    lives: present.lives && models.lives && (
      <MatchLivesCard lives={models.lives} inkOf={inkOf} meXUID={x.meXUID} ut={texts.cards} />
    ),
  }
}

function productionPending(p: MatchProductionPending, own: MatchOwnText) {
  return { resource: p.resource, text: own.production[p.reason], exposure: p.exposure }
}

function yieldPending(p: MatchYieldPending, own: MatchOwnText, effect: (ms: number) => string) {
  const r = p.reason
  const text = r.kind === 'noEffect' ? own.yield.noEffect(r.team, effect(r.teamEffectMs), r.teamKills) : own.yield.noPickup(r.team, r.us, r.them)
  return { resource: p.resource, text }
}
