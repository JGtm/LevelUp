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
import { buildPadTierRows, padTiersCoverage, padTiersNotes } from './usagePadTiersModel'
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

/**
 * Le bandeau de titre : le libellé PORTEUR DE SON AIDE, rien d'autre.
 *
 * LE COMPTEUR « Matchs mesurés N/M » A QUITTÉ LES QUATRE TITRES le 2026-09-13 (demande
 * utilisateur) : répété quatre fois, il encombrait autant de bandeaux pour une seule
 * information. La couverture s'écrit désormais UNE fois par rangée, en pied de la carte
 * de gauche (`measuredFooter`).
 */
function cardTitleWithHint(hint: string) {
  return (label: string) => (
    <HeaderLabelTooltip text={hint} focusable>
      <span>{label}</span>
    </HeaderLabelTooltip>
  )
}

/** La couverture de mesure de la rangée, en pied de sa première carte. */
function measuredFooter(usage: EquipmentUsageBlock, t: UsageText) {
  return (
    <div className="border-t border-border px-3 py-2">
      <p className="text-xs text-muted-foreground">
        {t.measuredFooterFmt(usage.matches_measured, usage.matches_total)}
      </p>
    </div>
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

/** La rangée « équipement » : les comptes à gauche, la part du lobby à droite. */
function EquipmentCards({ usage, mode, t, locale }: CardContentProps) {
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
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
      <SectionCard
        title={t.blockEquipment}
        label={t.blockEquipment}
        titleAdornment={cardTitleWithHint(t.cardHintEquipmentCounts)}
        footer={measuredFooter(usage, t)}
      >
        <div className="p-3">
          <UsageCountsGrid grid={grid} />
        </div>
      </SectionCard>
      {donut != null && (
        <SectionCard
          title={mode === 'solo' ? t.viewEquipmentPartsSolo : t.viewEquipmentPartsSquad}
          label={mode === 'solo' ? t.viewEquipmentPartsSolo : t.viewEquipmentPartsSquad}
          titleAdornment={cardTitleWithHint(t.cardHintEquipmentCounts)}
        >
          <div className="p-3">
            <UsageEquipmentDonutCard model={donut} />
          </div>
        </SectionCard>
      )}
    </div>
  )
}

/** La rangée « armes spéciales » : mêmes deux questions, sur les prises de socle. */
function PadControlCards({ usage, mode, t, locale }: CardContentProps) {
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
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
      <SectionCard
        title={t.blockPadControl}
        label={t.blockPadControl}
        titleAdornment={cardTitleWithHint(t.cardHintWeaponCounts)}
        footer={measuredFooter(usage, t)}
      >
        <div className="p-3">
          <UsageCountsGrid grid={grid} />
        </div>
      </SectionCard>
      {donut != null && (
        <SectionCard
          title={mode === 'solo' ? t.viewWeaponPartsSolo : t.viewWeaponPartsSquad}
          label={mode === 'solo' ? t.viewWeaponPartsSolo : t.viewWeaponPartsSquad}
          titleAdornment={cardTitleWithHint(t.cardHintWeaponCounts)}
        >
          <div className="p-3">
            <UsageEquipmentDonutCard model={donut} />
          </div>
        </SectionCard>
      )}
    </div>
  )
}

/**
 * La rangée « niveaux d'armes » : les MEMES prises, rangées par niveau — armes de base, de
 * terrain, de puissance. Une ligne par niveau, le détail par arme au survol.
 *
 * RANGEE ABSENTE PLUTOT QUE VIDE : sans bloc `pad_tiers` (aucun match du scope n'a été projeté
 * par la passe des niveaux), rien ne se rend. « Pas encore mesuré » ne se dessine pas comme
 * « aucune prise ».
 *
 * PAS DE DONUT ICI, et ce n'est pas un oubli : la question « quelle part du lobby était pour
 * moi » est DEJA celle de la rangée au-dessus, sur les mêmes prises. Un second donut ne dirait
 * rien de plus.
 */
function PadTierCards({ usage, t, locale }: CardContentProps) {
  const rows = buildPadTierRows(usage.pad_tiers, t)
  if (rows.length === 0) return null
  // `sort: false` : l'ordre des niveaux est ECRIT (base, terrain, puissance...), jamais le
  // volume — un classement dont l'ordre bouge d'une session à l'autre ne se compare pas.
  const grid = buildCountsGrid(rows, { t, locale, unit: 'weapon', sort: false })
  const notes = padTiersNotes(usage.pad_tiers, t)
  return (
    <SectionCard
      title={t.blockPadTiers}
      label={t.blockPadTiers}
      titleAdornment={cardTitleWithHint(t.cardHintPadTiers)}
      footer={padTiersFooter(usage, t)}
    >
      <div className="p-3">
        <UsageCountsGrid grid={grid} />
        {notes.map((note) => (
          <p key={note} className="pt-2 text-3xs text-muted-foreground">
            {note}
          </p>
        ))}
      </div>
    </SectionCard>
  )
}

export function EquipmentUsageSection({ usage, mode, t, locale }: EquipmentUsageSectionProps) {
  const availability = usageAvailability(usage, t)
  if (availability.kind === 'hidden' || usage == null) return null
  if (availability.kind === 'empty') {
    return (
      <>
        <SectionCard title={t.blockUnavailableTitle} label={t.blockUnavailableTitle}>
          <p className="px-3 pb-3 pt-3 text-sm text-muted-foreground">{availability.message}</p>
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

/** La couverture de la rangée des niveaux — LA SIENNE, jamais celle du résumé d'usage. */
function padTiersFooter(usage: EquipmentUsageBlock, t: UsageText) {
  const texte = padTiersCoverage(usage.pad_tiers, t)
  if (texte == null) return undefined
  return (
    <div className="border-t border-border px-3 py-2">
      <p className="text-xs text-muted-foreground">{texte}</p>
    </div>
  )
}
