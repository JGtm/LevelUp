/**
 * EquipmentUsageSection.tsx — L'ORCHESTRATEUR du bloc « servi ou gâché » en variante
 * COMPTES (décision P9, PLAN_EQUIPEMENT_GACHIS_2026-09-09, étapes E5.8-E5.10/E6.2-E6.4).
 *
 * DEUX CARTES (le gabarit `SectionCard`, même patron que `SessionUsageSection`) :
 *   1. « Usages d'équipement » — une ligne par grandeur (barre en comptes, P9) puis le
 *      donut « part de l'équipement du lobby » (P10/P11) ;
 *   2. « Contrôle des armes spéciales » — même paire, sur les prises de socle. Le tir
 *      n'est pas mesuré à ce grain (P5/E6.1) : la barre y est un compte simple, sans
 *      pile d'issues.
 *
 * DEUX MODES, une seule différence : la BASE des lignes de la barre équipement.
 *   - 'solo' (Synthèse) : une ligne par FAMILLE (`usage.families`, déjà triée côté Go).
 *   - 'squad' (Escouade) : une ligne par COÉQUIPIER SUIVI (`usage.players`, toutes
 *     familles confondues) — la barre armes spéciales, elle, est TOUJOURS construite
 *     depuis `usage.players` sur les deux pages : il n'existe aucune ventilation des
 *     prises de socle par famille d'arme (Go, `internal/domain/equipment_usage.go`).
 *     Sur Solo, `players` ne porte que le joueur de la route (+ les amis globalement
 *     configurés qui apparaissent dans le scope, cf. `ResolveScopeFriends`) — la barre
 *     y compte donc peu de lignes, mais ce n'est jamais zéro tant qu'il y a un
 *     ramassage mesuré.
 *
 * LA DONNÉE ARRIVE DANS LA RÉPONSE EXISTANTE (aucune query neuve) : `equipment_usage`
 * sur `SynthesisPageResponse` et (à terme — cf. lib/api/types.ts) `TeammatesPageResponse`.
 *
 * Aucun calcul de part ici : `usageCountsModel.ts` (barres) et
 * `usageEquipmentPartiesModel.ts` (donuts) le font. Ce fichier n'assemble que les deux
 * ensembles de lignes depuis `EquipmentUsageBlock` et pose le chrome des cartes.
 */
import { SectionCard } from '@/components/ui/section-card'
import { HeaderLabelTooltip } from '@/lib/table/columnMeta'
import type { EquipmentUsageBlock, EquipmentUsagePlayerLine, SessionUsageSquadPlayer } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { UsageCountsGrid } from './UsageCountsGrid'
import { UsageEquipmentDonutCard } from './UsageEquipmentDonutCard'
import { usageAvailability } from './usageAvailability'
import { buildCountsGrid, type UsageCountsRowInput } from './usageCountsModel'
import { buildPartiesDonutModel } from './usageEquipmentPartiesModel'
import { equipmentFamilyLabel, type UsageText } from './usageI18n'

export interface EquipmentUsageSectionProps {
  /** Le bloc `equipment_usage` de la réponse de page — absent : rien ne se rend. */
  usage: EquipmentUsageBlock | null | undefined
  /** 'solo' (Synthèse, une ligne par famille) ou 'squad' (Escouade, une ligne par
   *  coéquipier suivi) — cf. l'en-tête du fichier. */
  mode: 'solo' | 'squad'
  t: UsageText
  locale: Locale
}

function ViewTitle({ children }: { children: string }) {
  return (
    <h4 className="mb-2 text-3xs font-semibold uppercase tracking-wider text-muted-foreground">
      {children}
    </h4>
  )
}

/** Même gabarit de bandeau de titre que `SessionUsageSection` (aide sur le libellé). */
function cardTitleAdornment(measured: string, hint: string) {
  return (label: string) => (
    <span className="flex items-baseline gap-2">
      <HeaderLabelTooltip text={hint} focusable>
        <span>{label}</span>
      </HeaderLabelTooltip>
      <span className="text-3xs font-medium normal-case text-muted-foreground tabular-nums">{measured}</span>
    </span>
  )
}

/** Le libellé d'un sujet (moi ou un coéquipier suivi) — « Moi » pour le joueur de la
 *  route (toujours en tête de `players[]`, garanti par le contrat Go), le gamertag du
 *  coéquipier sinon (jointure par xuid contre `tracked_players`). */
function playerLabel(xuid: string, mainXuid: string | undefined, trackedPlayers: SessionUsageSquadPlayer[], t: UsageText): string {
  if (xuid === mainXuid) return t.donutMe
  return trackedPlayers.find((p) => p.xuid === xuid)?.gamertag ?? xuid
}

function familyRows(usage: EquipmentUsageBlock, t: UsageText): UsageCountsRowInput[] {
  return (usage.families ?? []).map((f) => ({
    key: f.family_key,
    label: equipmentFamilyLabel(f.family_key, t),
    taken: f.taken,
    outcomes: {
      used: f.used,
      kept: f.kept,
      dropped: f.dropped,
      teammates_used_rate_pct: f.teammates_used_rate_pct,
      opponents_used_rate_pct: f.opponents_used_rate_pct,
    },
  }))
}

function playerEquipmentRows(usage: EquipmentUsageBlock, t: UsageText): UsageCountsRowInput[] {
  const players = usage.players ?? []
  const mainXuid = players[0]?.xuid
  const tracked = usage.tracked_players ?? []
  return players.map((p: EquipmentUsagePlayerLine) => ({
    key: p.xuid,
    label: playerLabel(p.xuid, mainXuid, tracked, t),
    taken: p.taken,
    outcomes: {
      used: p.used,
      kept: p.kept,
      dropped: p.dropped,
      teammates_used_rate_pct: p.teammates_used_rate_pct,
      opponents_used_rate_pct: p.opponents_used_rate_pct,
    },
  }))
}

/** La barre « armes spéciales » — TOUJOURS par sujet, sur les deux pages (P5/E6.1 :
 *  aucune ventilation par famille d'arme au grain période). Le tir n'étant pas mesuré à
 *  ce grain, une ligne n'a pas d'`outcomes` : un aplat simple, jamais un zéro inventé. */
function playerWeaponRows(usage: EquipmentUsageBlock, t: UsageText): UsageCountsRowInput[] {
  const players = usage.players ?? []
  const mainXuid = players[0]?.xuid
  const tracked = usage.tracked_players ?? []
  return players.map((p: EquipmentUsagePlayerLine) => ({
    key: p.xuid,
    label: playerLabel(p.xuid, mainXuid, tracked, t),
    taken: p.pad_pickups,
  }))
}

interface CardContentProps {
  usage: EquipmentUsageBlock
  mode: 'solo' | 'squad'
  t: UsageText
  locale: Locale
}

function EquipmentCard({ usage, mode, t, locale }: CardContentProps) {
  const rows = mode === 'solo' ? familyRows(usage, t) : playerEquipmentRows(usage, t)
  const grid = buildCountsGrid(rows, { t, locale, unit: 'equipment' })
  const donut = buildPartiesDonutModel(
    usage.equipment_parties,
    usage.tracked_players ?? [],
    t.donutEquipmentCenterLabel,
    t,
    locale,
  )
  return (
    <SectionCard
      title={t.blockEquipment}
      label={t.blockEquipment}
      titleAdornment={cardTitleAdornment(
        t.measuredFmt(usage.matches_measured, usage.matches_total),
        t.cardHintEquipmentCounts,
      )}
    >
      <div className="flex flex-col gap-4 p-3">
        <div>
          <ViewTitle>{mode === 'solo' ? t.viewCountsSolo : t.viewCountsSquad}</ViewTitle>
          <UsageCountsGrid grid={grid} />
        </div>
        {donut != null && (
          <div>
            <ViewTitle>{mode === 'solo' ? t.viewEquipmentPartsSolo : t.viewEquipmentPartsSquad}</ViewTitle>
            <UsageEquipmentDonutCard model={donut} />
          </div>
        )}
      </div>
    </SectionCard>
  )
}

function PadControlCard({ usage, mode, t, locale }: CardContentProps) {
  const rows = playerWeaponRows(usage, t)
  const grid = buildCountsGrid(rows, { t, locale, unit: 'weapon' })
  const donut = buildPartiesDonutModel(
    usage.weapon_pad_parties,
    usage.tracked_players ?? [],
    t.donutWeaponCenterLabel,
    t,
    locale,
  )
  return (
    <SectionCard
      title={t.blockPadControl}
      label={t.blockPadControl}
      titleAdornment={cardTitleAdornment(
        t.measuredFmt(usage.matches_measured, usage.matches_total),
        t.cardHintWeaponCounts,
      )}
    >
      <div className="flex flex-col gap-4 p-3">
        <div>
          <ViewTitle>{mode === 'solo' ? t.viewCountsSolo : t.viewCountsSquad}</ViewTitle>
          <UsageCountsGrid grid={grid} />
        </div>
        {donut != null && (
          <div>
            <ViewTitle>{mode === 'solo' ? t.viewWeaponPartsSolo : t.viewWeaponPartsSquad}</ViewTitle>
            <UsageEquipmentDonutCard model={donut} />
          </div>
        )}
      </div>
    </SectionCard>
  )
}

export function EquipmentUsageSection({ usage, mode, t, locale }: EquipmentUsageSectionProps) {
  const availability = usageAvailability(usage, t)
  if (availability.kind === 'hidden' || usage == null) return null
  if (availability.kind === 'empty') {
    return (
      <SectionCard title={t.blockUnavailableTitle} label={t.blockUnavailableTitle}>
        <p className="px-3 pb-3 pt-3 text-sm text-muted-foreground">{availability.message}</p>
      </SectionCard>
    )
  }
  return (
    <>
      <EquipmentCard usage={usage} mode={mode} t={t} locale={locale} />
      <PadControlCard usage={usage} mode={mode} t={t} locale={locale} />
    </>
  )
}
