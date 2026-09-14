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
 * TOUS LES DÉNOMINATEURS DU BLOC SONT LES SIENS — couverture ET parités (correctif de revue,
 * 2026-09-14). Ils venaient du résumé d'usage voisin, qui porte sur un AUTRE périmètre : une
 * passe distincte, sur d'autres matchs. La carte annonçait donc une couverture qu'elle n'avait
 * pas, et posait son trait de parité au mauvais endroit dès que les deux divergeaient — ce
 * qu'elles font par construction.
 *
 * MÊME FORME QUE LES DEUX AUTRES RANGÉES : `buildCountsGrid` fait les barres, l'axe et les
 * textes. Rien de neuf à l'écran — une rangée de plus, dans le vocabulaire existant.
 *
 * Pur : aucun React, aucune couleur, aucune langue — les libellés arrivent par `UsageText`.
 */
import type { SessionUsagePadTiersBlock } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import type { UsageCountsRowInput } from './usageCountsModel'
import { buildGaugeRow, type UsageGaugeRowModel } from './usageGaugeModel'
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

/** Une ligne de niveau du contrat, telle que le serveur la sert. */
type PadTierLine = NonNullable<SessionUsagePadTiersBlock['tiers']>[number]

/** Le libellé d'un niveau ; une valeur inconnue du contrat garde sa clé, jamais un nom voisin. */
export function padTierLabel(tier: string, t: UsageText): string {
  return Object.hasOwn(t.padTierLabels, tier) ? t.padTierLabels[tier as UsagePadTier] : tier
}

/** Les lignes de niveau SERVIES, dans l'ordre écrit. */
function lignesOrdonnees(block: SessionUsagePadTiersBlock | null | undefined): PadTierLine[] {
  if (block == null || (block.tiers ?? []).length === 0) return []
  const parNiveau = new Map((block.tiers ?? []).map((tier) => [tier.tier, tier]))
  const out: PadTierLine[] = []
  for (const tier of USAGE_PAD_TIER_ORDER) {
    const ligne = parNiveau.get(tier)
    if (ligne != null) out.push(ligne)
  }
  return out
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
  return lignesOrdonnees(block).map((ligne) => ({
    key: ligne.tier,
    label: padTierLabel(ligne.tier, t),
    taken: ligne.player_total,
    hint: hintDesArmes(ligne, t),
  }))
}

/** Le détail par arme d'un niveau, déjà composé : « Armes de puissance — S7 Sniper 4, SPNKr 2 ». */
function hintDesArmes(tier: PadTierLine, t: UsageText): string | undefined {
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
 * Rend une liste VIDE quand il n'y a RIEN à signaler — tous les matchs mesurés ont des socles,
 * une carte connue, et aucun mode aléatoire. Une note permanente qui répète « tout va bien » ne
 * se lit plus.
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
  if (block.matches_random_starts > 0) {
    notes.push(t.padTierRandomStartsFmt(block.matches_random_starts))
  }
  return notes
}

/**
 * padTiersCoverage — LA COUVERTURE DU BLOC, et c'est LA SIENNE.
 *
 * La carte affichait celle du RÉSUMÉ D'USAGE (`usage.matches_measured`), qui porte sur un autre
 * périmètre : une passe distincte, sur d'autres matchs. Les deux divergent par construction, et
 * la carte annonçait alors une couverture qu'elle n'avait pas (revue du 2026-09-14).
 *
 * Rend `null` quand le bloc ne connaît pas son propre dénominateur : mieux vaut ne rien écrire
 * que citer celui du voisin.
 */
export function padTiersCoverage(
  block: SessionUsagePadTiersBlock | null | undefined,
  t: UsageText,
): string | null {
  if (block == null || block.matches_total <= 0) return null
  return t.measuredFooterFmt(block.matches_measured, block.matches_total)
}

/**
 * Ce dont les lignes de jauge ont besoin EN PLUS du bloc : la langue, et rien d'autre.
 *
 * LES PARITÉS NE SONT PLUS PASSÉES PAR L'APPELANT (revue du 2026-09-14) : c'étaient celles du
 * résumé d'usage voisin, donc d'un AUTRE périmètre. Le bloc porte désormais les siennes,
 * calculées par le serveur sur ses propres matchs.
 */
export interface PadTierGaugeOptions {
  t: UsageText
  locale: Locale
}

/**
 * buildPadTierGaugeRows — les MÊMES lignes, dans la forme « trois jauges » de la page Sessions.
 *
 * DEUX FORMES, UN SEUL ORDRE ET UN SEUL DÉTAIL : cette fonction et `buildPadTierRows`
 * partagent `lignesOrdonnees` et `hintDesArmes`. Deux pages qui rangeraient les niveaux dans
 * deux ordres, ou qui nommeraient les armes de deux façons, se liraient comme deux mesures.
 */
export function buildPadTierGaugeRows(
  block: SessionUsagePadTiersBlock | null | undefined,
  opts: PadTierGaugeOptions,
): UsageGaugeRowModel[] {
  if (block == null) return []
  return lignesOrdonnees(block).map((ligne) =>
    buildGaugeRow({
      key: `tier-${ligne.tier}`,
      label: padTierLabel(ligne.tier, opts.t),
      shares: ligne,
      // LES PARITÉS DU BLOC, calculées sur SON périmètre par le serveur.
      teamParityPct: block.team_parity_pct,
      lobbyParityPct: block.lobby_parity_pct,
      teamOfLobbyParityPct: block.team_of_lobby_parity_pct,
      hint: hintDesArmes(ligne, opts.t),
      t: opts.t,
      locale: opts.locale,
    }),
  )
}
