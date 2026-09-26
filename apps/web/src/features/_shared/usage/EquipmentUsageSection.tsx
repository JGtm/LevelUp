/**
 * EquipmentUsageSection.tsx — L'ORCHESTRATEUR du bloc « servi ou gâché » en variante
 * COMPTES (décision P9, PLAN_EQUIPEMENT_GACHIS_2026-09-09, étapes E5.8-E5.10/E6.2-E6.4).
 *
 * QUATRE CARTES SUR DEUX RANGÉES (le gabarit `SectionCard`) — découpage du 2026-09-13.
 * Chaque vue portait jusqu'ici un sous-titre à l'intérieur d'une carte à deux étages : la
 * barre et son donut se lisaient comme un seul objet, alors qu'ils répondent à deux
 * questions (« qu'ai-je fait de ce que j'ai pris » et « quelle part du lobby était pour
 * moi »). Le sous-titre de chaque vue est devenu le TITRE de sa carte :
 *   1. « Usages d'équipement » (barres) | « Ma part de l'équipement du lobby » (donut) ;
 *   2. « Contrôle des armes spéciales » (barres) | « Ma part des armes spéciales du lobby ».
 * Le tir n'est pas mesuré au grain des socles (P5/E6.1) : la barre de la seconde rangée est
 * un compte simple, sans pile d'issues. En mode squad, les deux donuts disent « Notre part ».
 *
 * DEUX MODES, une seule différence : la BASE des lignes de la barre équipement.
 *   - 'solo' (Synthèse) : une ligne par FAMILLE (`usage.families`, déjà triée côté Go).
 *   - 'squad' (Escouade) : une ligne par COÉQUIPIER SUIVI (`usage.players`, toutes
 *     familles confondues) — la barre armes spéciales, elle, est TOUJOURS construite
 *     depuis `usage.players` sur les deux pages : il n'existe aucune ventilation des
 *     prises de socle par famille d'arme (Go, `internal/domain/equipment_usage.go`).
 *
 * TROIS AJUSTEMENTS DU 2026-09-21 (lot A2) :
 *   - UNE SEULE AIDE PAR CARTE, VISIBLE : l'aide d'en-tête (invisible, sur le libellé), les
 *     notes sous la grille et le pied de couverture fusionnent dans l'infobulle (i) du
 *     titre (`usageCardTitle`). Trois mécanismes disaient la même méthode à trois endroits ;
 *   - LE CORPS DES CARTES À GRILLE EST CENTRÉ VERTICALEMENT (`flex-1 justify-center`) : une
 *     carte étirée par sa voisine plus haute laissait sa grille collée en haut ;
 *   - AUCUN BLOC D'UNE RANGÉE NE SE MASQUE (D8) : vide, il reste affiché et NOMME sa cause
 *     (`UsageEmptyNotice`) — escamoté, il laisse la rangée bancale et se lit comme un bug.
 *
 * Aucun calcul de part ici : `usageCountsModel.ts` (barres) et
 * `usageEquipmentPartiesModel.ts` (donuts) le font.
 */
import { SectionCard } from '@/components/ui/section-card'
import type { EquipmentUsageBlock, EquipmentUsagePlayerLine, SessionUsageSquadPlayer } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { UsageCountsGrid } from './UsageCountsGrid'
import { UsageEmptyNotice } from './UsageEmptyNotice'
import { UsageEquipmentDonutCard } from './UsageEquipmentDonutCard'
import { usageAvailability } from './usageAvailability'
import { usageCardTitle } from './usageCardTitle'
import { buildCountsGrid, type UsageCountsRowInput, type UsageCountsRowModel } from './usageCountsModel'
import {
  buildPadTierRows,
  isCollapsedTierRowKey,
  padTiersCoverage,
  padTiersNotes,
} from './usagePadTiersModel'
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

/** Le corps d'une carte à grille : centré verticalement quand la voisine l'étire. */
function CardBody({ children }: { children: React.ReactNode }) {
  return <div className="flex flex-1 flex-col justify-center p-3">{children}</div>
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

/** Une barre entièrement à zéro n'est pas une mesure : elle vaut une grille vide (D8). */
function hasMeasure(rows: UsageCountsRowInput[]): boolean {
  return rows.some((r) => r.taken > 0)
}

interface CardContentProps {
  usage: EquipmentUsageBlock
  mode: 'solo' | 'squad'
  t: UsageText
  locale: Locale
}

/** La rangée « équipement » : les comptes à gauche, la part du lobby à droite. */
function EquipmentCards({ usage, mode, t, locale }: CardContentProps) {
  const rows = mode === 'solo' ? familyRows(usage, t) : playerEquipmentRows(usage, t)
  const grid = buildCountsGrid(rows, { t, locale, unit: 'equipment' })
  const donut = buildPartiesDonutModel(usage.equipment_parties, usage.tracked_players ?? [], t, locale)
  const donutTitle = mode === 'solo' ? t.viewEquipmentPartsSolo : t.viewEquipmentPartsSquad
  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
      <SectionCard
        title={t.blockEquipment}
        label={t.blockEquipment}
        titleAdornment={usageCardTitle(t.cardHintEquipmentCounts)}
      >
        <CardBody>
          {hasMeasure(rows) ? (
            <UsageCountsGrid grid={grid} />
          ) : (
            <UsageEmptyNotice reason="no-film" t={t} />
          )}
        </CardBody>
      </SectionCard>
      <SectionCard
        title={donutTitle}
        label={donutTitle}
        titleAdornment={usageCardTitle(t.cardHintEquipmentCounts)}
      >
        <CardBody>
          {donut != null ? (
            <UsageEquipmentDonutCard model={donut} />
          ) : (
            <UsageEmptyNotice reason="no-film" t={t} />
          )}
        </CardBody>
      </SectionCard>
    </div>
  )
}

/** La rangée « armes spéciales » : mêmes deux questions, sur les prises de socle. */
function PadControlCards({ usage, mode, t, locale }: CardContentProps) {
  const rows = playerWeaponRows(usage, t)
  const grid = buildCountsGrid(rows, { t, locale, unit: 'weapon' })
  const donut = buildPartiesDonutModel(usage.weapon_pad_parties, usage.tracked_players ?? [], t, locale)
  const donutTitle = mode === 'solo' ? t.viewWeaponPartsSolo : t.viewWeaponPartsSquad
  // ZÉRO SOCLE SUR UNE MESURE FAITE, ce n'est pas « aucun film » : c'est un mode qui
  // n'allume aucun socle (Super Fiesta). Deux causes, deux phrases (D8).
  const empty = <UsageEmptyNotice reason="no-pads" t={t} />
  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
      <SectionCard
        title={t.blockPadControl}
        label={t.blockPadControl}
        titleAdornment={usageCardTitle(t.cardHintWeaponCounts)}
      >
        <CardBody>{hasMeasure(rows) ? <UsageCountsGrid grid={grid} /> : empty}</CardBody>
      </SectionCard>
      {/* LA CARTE SE REND TOUJOURS (2026-09-19), en PARALLÈLE de l'équipement : sans
          `weapon_pad_parties` (ou à zéro), elle porte son titre et dit que rien n'est
          mesuré. Escamotée, elle laissait la rangée bancale et se lisait comme un bug. */}
      <SectionCard
        title={donutTitle}
        label={donutTitle}
        titleAdornment={usageCardTitle(t.cardHintWeaponCounts)}
      >
        <CardBody>{donut != null ? <UsageEquipmentDonutCard model={donut} /> : empty}</CardBody>
      </SectionCard>
    </div>
  )
}

/**
 * La rangée « niveaux d'armes » : les MÊMES prises, rangées par niveau — puissance, terrain,
 * puis les armes de base DANS UN DÉPLIABLE FERMÉ (D2). Le détail par arme au survol.
 *
 * RANGÉE ABSENTE PLUTÔT QUE VIDE : sans bloc `pad_tiers` (aucun match du scope n'a été projeté
 * par la passe des niveaux), rien ne se rend. « Pas encore mesuré » ne se dessine pas comme
 * « aucune prise » — et c'est une SECTION entière, pas un bloc d'une rangée (D8).
 *
 * PAS DE DONUT ICI, et ce n'est pas un oubli : la question « quelle part du lobby était pour
 * moi » est DÉJÀ celle de la rangée au-dessus, sur les mêmes prises.
 */
function PadTierCards({ usage, t, locale }: CardContentProps) {
  const rows = buildPadTierRows(usage.pad_tiers, t)
  if (usage.pad_tiers == null) return null
  // `sort: false` : l'ordre des niveaux est ÉCRIT (puissance, terrain, base), jamais le
  // volume — un classement dont l'ordre bouge d'une session à l'autre ne se compare pas.
  const grid = buildCountsGrid(rows, { t, locale, unit: 'weapon', sort: false })
  // Les lignes repliées sortent de la grille CONSTRUITE, jamais d'un second appel : l'axe
  // des comptes doit rester le même pour les lignes visibles et les lignes dépliées.
  const visible: UsageCountsRowModel[] = []
  const collapsed: UsageCountsRowModel[] = []
  for (const row of grid.rows) (isCollapsedTierRowKey(row.key) ? collapsed : visible).push(row)
  const baseCount = collapsed.length
  return (
    <SectionCard
      title={t.blockPadTiers}
      label={t.blockPadTiers}
      titleAdornment={usageCardTitle(
        t.cardHintPadTiers,
        ...padTiersNotes(usage.pad_tiers, t),
        padTiersCoverage(usage.pad_tiers, t),
      )}
    >
      <CardBody>
        {rows.length > 0 ? (
          <UsageCountsGrid
            grid={{ ...grid, rows: visible }}
            collapsedRows={collapsed}
            collapsedLabel={baseCount > 0 ? t.padTierBaseToggleFmt(baseCount) : undefined}
          />
        ) : (
          <UsageEmptyNotice reason="no-pads" t={t} />
        )}
      </CardBody>
    </SectionCard>
  )
}

export function EquipmentUsageSection({ usage, mode, t, locale }: EquipmentUsageSectionProps) {
  const availability = usageAvailability(usage)
  if (availability.kind === 'hidden' || usage == null) return null
  if (availability.kind === 'empty') {
    return (
      <>
        <SectionCard title={t.blockUnavailableTitle} label={t.blockUnavailableTitle}>
          <CardBody>
            <UsageEmptyNotice
              reason={usage.available ? 'no-film' : 'load-failed'}
              t={t}
            />
          </CardBody>
        </SectionCard>
        {/* LA RANGÉE DES NIVEAUX A SA PROPRE DISPONIBILITÉ (revue du 2026-09-14). Elle vient
            d'une AUTRE passe, sur d'autres matchs : un résumé d'usage vide ne prouve rien de
            ses niveaux, et l'avaler ici masquerait une mesure qui existe. */}
        <PadTierCards usage={usage} mode={mode} t={t} locale={locale} />
      </>
    )
  }
  return (
    <>
      <EquipmentCards usage={usage} mode={mode} t={t} locale={locale} />
      <PadControlCards usage={usage} mode={mode} t={t} locale={locale} />
      <PadTierCards usage={usage} mode={mode} t={t} locale={locale} />
    </>
  )
}
