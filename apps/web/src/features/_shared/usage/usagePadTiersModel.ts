/**
 * usagePadTiersModel.ts — LES PRISES DE SOCLE PAR NIVEAU D'ARME, en lignes de grille.
 *
 * CE QUE LA PAGE NE SAVAIT PAS DIRE. Le bloc « contrôle des armes spéciales » comptait toutes
 * les prises de socle à égalité : reprendre son fusil d'assaut posé sur un râtelier y pesait
 * autant que rafler le lance-roquettes du socle central. Une ligne par NIVEAU remet chaque
 * prise à sa place — armes de base, de terrain, de puissance.
 *
 * LE NIVEAU VIENT DU SERVEUR, ET DU SERVEUR SEUL. Il se mesure sur la CARTE (l'emplacement que
 * le fichier Forge pose, croisé au socle du match) et sur l'équipement de départ du film ; ce
 * module ne le recalcule pas, il le met en forme. Mesure du 2026-09-14 : juger au nom ou au
 * rôle de l'arme produirait 10 % de faux niveaux (70 socles sur 669 portent une arme de rôle
 * « lourd » sur un râtelier).
 *
 * MÊME FORME QUE LES DEUX AUTRES RANGÉES : `buildCountsGrid` fait les barres, l'axe et les
 * textes. Rien de neuf à l'écran — une rangée de plus, dans le vocabulaire existant.
 *
 * Pur : aucun React, aucune couleur, aucune langue — les libellés arrivent par `UsageText`.
 */
import type { SessionUsagePadTiersBlock } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { buildGaugeRow, type UsageGaugeRowModel } from './usageGaugeModel'
import type { UsageCountsRowInput } from './usageCountsModel'
import type { UsageText } from './usageI18n'

/**
 * L'ORDRE DE LECTURE DES NIVEAUX, écrit — le même que `domain.PadTierOrder` côté Go.
 *
 * Il n'est PAS trié par volume, et c'est délibéré : un classement dont l'ordre change d'une
 * session à l'autre ne se compare pas d'un écran au suivant. C'est aussi pourquoi l'appelant
 * passe `sort: false` à `buildCountsGrid`.
 */
export const USAGE_PAD_TIER_ORDER = ['base', 'terrain', 'puissance', 'bonus', 'non_classe'] as const

export type UsagePadTier = (typeof USAGE_PAD_TIER_ORDER)[number]

/** Le libellé d'un niveau ; une valeur inconnue du contrat garde sa clé, jamais un nom voisin. */
export function padTierLabel(tier: string, t: UsageText): string {
  return Object.hasOwn(t.padTierLabels, tier) ? t.padTierLabels[tier as UsagePadTier] : tier
}

/**
 * buildPadTierRows — une ligne par niveau SERVI, dans l'ordre écrit.
 *
 * Un niveau que le bloc ne publie pas n'a pas de ligne : une ligne à zéro dirait « aucune
 * prise à ce niveau » là où la vérité est « ce niveau n'existe pas sur ce scope » (un mode à
 * départs aléatoires n'a pas d'arme de base, une carte hors référence n'a ni terrain ni
 * puissance).
 *
 * LE DÉTAIL PAR ARME PART AU SURVOL, pas dans le libellé : trois à huit armes par niveau
 * rendraient la ligne illisible, et les masquer entièrement ferait perdre la seule information
 * qui dit CE QU'ON a contrôlé.
 */
export function buildPadTierRows(
  block: SessionUsagePadTiersBlock | null | undefined,
  t: UsageText,
): UsageCountsRowInput[] {
  if (block == null || (block.tiers ?? []).length === 0) return []
  const parNiveau = new Map((block.tiers ?? []).map((tier) => [tier.tier, tier]))
  const rows: UsageCountsRowInput[] = []
  for (const tier of USAGE_PAD_TIER_ORDER) {
    const ligne = parNiveau.get(tier)
    if (ligne == null) continue
    rows.push({
      key: tier,
      label: padTierLabel(tier, t),
      taken: ligne.player_total,
      hint: hintDesArmes(ligne, t),
    })
  }
  return rows
}

/** Le détail par arme d'un niveau, déjà composé : « Armes de puissance — S7 Sniper 4, SPNKr 2 ». */
function hintDesArmes(
  tier: NonNullable<SessionUsagePadTiersBlock['tiers']>[number],
  t: UsageText,
): string | undefined {
  const armes = (tier.weapons ?? []).filter((w) => w.player_pickups > 0)
  if (armes.length === 0) return undefined
  // La clé sert de repli quand le catalogue du titre ne connaît pas la famille : on n'affiche
  // JAMAIS un nom approchant (même règle que le catalogue du rejeu).
  const detail = armes
    .map((w) => `${w.family_label || w.family_key} ${w.player_pickups}`)
    .join(', ')
  return `${padTierLabel(tier.tier, t)} — ${detail}`
}

/**
 * LA NOTE DE MESURE de la rangée, en une phrase : ce que les quatre dénominateurs du bloc
 * disent, et que rien d'autre ne dit.
 *
 * Rend `null` quand il n'y a RIEN à signaler — tous les matchs mesurés ont des socles, une
 * carte connue, et aucun mode aléatoire. Une note permanente qui répète « tout va bien » ne se
 * lit plus.
 */
export function padTiersNotes(
  block: SessionUsagePadTiersBlock | null | undefined,
  t: UsageText,
): string[] {
  if (block == null) return []
  const notes: string[] = []
  const sansSocle = block.matches_measured - block.matches_with_pads
  if (sansSocle > 0) notes.push(t.padTierNoPadsFmt(sansSocle))
  const horsReference = block.matches_with_pads - block.matches_tiers_established
  if (horsReference > 0) notes.push(t.padTierUnmeasuredFmt(horsReference))
  if (block.matches_random_starts > 0) notes.push(t.padTierRandomStartsFmt(block.matches_random_starts))
  return notes
}

/** Ce dont les lignes de jauge ont besoin en plus du bloc (parités du scope, langue). */
export interface PadTierGaugeOptions {
  teamParityPct: number | null | undefined
  lobbyParityPct: number | null | undefined
  teamOfLobbyParityPct: number | null | undefined
  t: UsageText
  locale: Locale
}

/**
 * buildPadTierGaugeRows — les MÊMES lignes, dans la forme « trois jauges » de la page Sessions.
 *
 * DEUX FORMES, UN SEUL ORDRE ET UN SEUL DÉTAIL : cette fonction et `buildPadTierRows`
 * partagent `USAGE_PAD_TIER_ORDER` et `hintDesArmes`. Deux pages qui rangeraient les niveaux
 * dans deux ordres, ou qui nommeraient les armes de deux façons, se liraient comme deux
 * mesures.
 */
export function buildPadTierGaugeRows(
  block: SessionUsagePadTiersBlock | null | undefined,
  opts: PadTierGaugeOptions,
): UsageGaugeRowModel[] {
  if (block == null || (block.tiers ?? []).length === 0) return []
  const parNiveau = new Map((block.tiers ?? []).map((tier) => [tier.tier, tier]))
  const rows: UsageGaugeRowModel[] = []
  for (const tier of USAGE_PAD_TIER_ORDER) {
    const ligne = parNiveau.get(tier)
    if (ligne == null) continue
    rows.push(
      buildGaugeRow({
        key: `tier-${tier}`,
        label: padTierLabel(tier, opts.t),
        shares: ligne,
        teamParityPct: opts.teamParityPct,
        lobbyParityPct: opts.lobbyParityPct,
        teamOfLobbyParityPct: opts.teamOfLobbyParityPct,
        hint: hintDesArmes(ligne, opts.t),
        t: opts.t,
        locale: opts.locale,
      }),
    )
  }
  return rows
}
