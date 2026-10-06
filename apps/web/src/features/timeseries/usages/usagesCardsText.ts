/**
 * usagesCardsText.ts — les textes des cartes PROPRES à l'onglet « Usages » des Séries temporelles :
 * en-têtes de la grille par carte, « Contribution aux prises », « Isolement », « Usage
 * d'équipements ». Textes factuels, sans personne : le joueur est désigné par son gamertag
 * (argument `player`), l'équipe par « Équipe », le reste par « Reste de l'équipe » (garde : `textesSansPersonne.test.ts`). Parité FR / EN par le typage.
 * Fichier à part de `usagesText.ts` (surcharges des textes de l'Escouade) pour rester sous le seuil.
 */

const plural = (n: number, one: string, many: string) => (n > 1 ? many : one)

const frPct = (v: number): string => `${(Math.round(v * 10) / 10).toLocaleString('fr-FR', { maximumFractionDigits: 1 })} %`
const enPct = (v: number): string => `${(Math.round(v * 10) / 10).toLocaleString('en-GB', { maximumFractionDigits: 1 })}%`
const frDec1 = (v: number) => v.toLocaleString('fr-FR', { minimumFractionDigits: 1, maximumFractionDigits: 1 })
const enDec1 = (v: number) => v.toLocaleString('en-GB', { minimumFractionDigits: 1, maximumFractionDigits: 1 })
const frInt = (v: number) => v.toLocaleString('fr-FR')
const enInt = (v: number) => v.toLocaleString('en-GB')

export interface UsagesCardsText {
  /** « 60 % », « 44,2 % » (une décimale au plus) ; `pctIntFmt` : « 17 % ». */
  pctFmt: (v: number) => string
  pctIntFmt: (v: number) => string
  /** « 1 558 » : un compte, séparateur de milliers de la langue. */
  intFmt: (v: number) => string
  /** Légende sous la bande de « au fil des matchs » : « 90 matchs, dont 62 filmés ». */
  filCaption: (matches: number, filmed: number) => string
  /** « 03/07 » : la date d'un match sous l'axe de « au fil des matchs ». */
  dayFmt: (iso: string) => string
  maps: {
    others: string
    /** « 3 cartes · » devant le nombre de matchs de la colonne de repli. */
    otherMapsFmt: (n: number) => string
    matchesFmt: (n: number) => string
    winsFmt: (n: number) => string
    lossesFmt: (n: number) => string
    othersFmt: (n: number) => string
    /** En-tête d'infobulle d'une colonne : « Aquarius (12 matchs, 9 filmés) ». */
    tipHead: (name: string, matches: number, filmed: number) => string
  }
  /** Le nom du joueur quand le bloc ne porte pas son gamertag. */
  playerFallback: string
  mine: {
    title: string
    info: string
    rest: string
    /** « Armes de râtelier (4, repliées) ». */
    foldedFmt: (n: number) => string
    /** Le mot de l’équipe dans « JGtm 3 · équipe 7 » au bout de chaque barre (le joueur : son gamertag). */
    campWord: string
    meTip: (player: string, name: string, me: number, camp: number) => string
    restTip: (name: string, rest: number, camp: number) => string
    lossesTitle: string
    lossesFmt: (lost: number, taken: number) => string
  }
  lives: {
    title: string
    /**
     * ⓘ : les vies écartées sont comptées (`unlocated` : sans coéquipier situé ; `noRadar` : carte sans
     * portée connue ; `unpublishable` : match dont le journal des morts n'est pas publiable).
     */
    info: (unlocated: number, noRadar: number, unpublishable: number) => string
    near: string
    alone: string
    thinLegend: string
    rowSub: (lives: number) => string
    tip: (side: string, value: number, total: number, pct: string, what: 'lives' | 'kills') => string
    killsLine: (kills: number, pct: string, perLife: string) => string
    killsLineAlone: (perLife: string, kills: number) => string
    perLifeFmt: (v: number) => string
  }
  equipment: {
    title: string
    info: string
    used: string
    kept: string
    dropped: string
    thinLegend: string
    rest: string
    /** Libellés des familles hors bilan (grappin, propulseur) ; les autres viennent du bloc d'usage. */
    unmeasuredNames: Record<string, string>
    sub: (objects: number, taken: number) => string
    droppedSub: (n: number) => string
    unmeasured: string
    zeroTip: (player: string, family: string) => string
    /** « JGtm · Mur de protection\n52 servis sur 84 (61,9 %) ». */
    segTip: (who: string, n: number, part: 'used' | 'kept' | 'dropped', total: number, pct: string) => string
    restLine: (used: number, kept: number, dropped: number) => string
    restUsedShare: (pct: string) => string
    restNone: string
  }
}

const PART_FR = { used: ['servi', 'servis'], kept: ['gardé', 'gardés'], dropped: ['lâché', 'lâchés'] } as const
const PART_EN = { used: ['used', 'used'], kept: ['kept', 'kept'], dropped: ['dropped', 'dropped'] } as const

export const USAGES_CARDS_TEXT_FR: UsagesCardsText = {
  pctFmt: frPct,
  pctIntFmt: (v) => `${Math.round(v)} %`,
  intFmt: frInt,
  filCaption: (matches, filmed) => `${matches} ${plural(matches, 'match', 'matchs')}, dont ${filmed} ${plural(filmed, 'filmé', 'filmés')}`,
  dayFmt: (iso) => {
    const d = new Date(iso)
    return Number.isNaN(d.getTime()) ? '' : d.toLocaleDateString('fr-FR', { day: '2-digit', month: '2-digit' })
  },
  maps: {
    others: 'Autres cartes',
    otherMapsFmt: (n) => `${n} cartes · `,
    matchesFmt: (n) => `${n} ${plural(n, 'match', 'matchs')}`,
    winsFmt: (n) => `${n} V`,
    lossesFmt: (n) => `${n} D`,
    othersFmt: (n) => `${n} A`,
    tipHead: (name, matches, filmed) => `${name} (${matches} ${plural(matches, 'match', 'matchs')}, ${filmed} ${plural(filmed, 'filmé', 'filmés')})`,
  },
  playerFallback: 'Joueur',
  mine: {
    title: 'Contribution aux prises',
    info:
      'Objets pris par l’équipe sur les matchs filmés du périmètre : part du joueur et du reste de l’équipe, en ' +
      'comptes, par volume décroissant. Bonus perdus : gardés sans être activés, ou lâchés.',
    rest: 'Reste de l’équipe',
    foldedFmt: (n) => `(${n}, repliées)`,
    campWord: 'équipe',
    meTip: (player, name, me, camp) => `${player} · ${name}\n${me} des ${camp} prises de l’équipe`,
    restTip: (name, rest, camp) => `Reste de l’équipe · ${name}\n${rest} des ${camp} prises de l’équipe`,
    lossesTitle: 'Bonus perdus',
    lossesFmt: (lost, taken) => `${lost} sur ${taken}`,
  },
  lives: {
    title: 'Isolement',
    info: (unlocated, noRadar, unpublishable) =>
      'Vies terminées par une mort, rangées selon la distance au coéquipier le plus proche à l’instant de la ' +
      'mort (à portée de radar ou au-delà) ; barre fine : frags obtenus pendant ces vies. ' +
      (noRadar > 0
        ? `Écartées : vies sans coéquipier situé (${frInt(unlocated)}), vies d’une carte sans portée de radar connue (${frInt(noRadar)})`
        : `Écartées : vies sans coéquipier situé (${frInt(unlocated)})`) +
      (unpublishable > 0 ? `, vies d’un match au journal des morts non publiable (${frInt(unpublishable)}).` : '.'),
    near: 'À portée d’un coéquipier',
    alone: 'Isolée',
    thinLegend: 'Barre fine : frags du joueur pendant ces vies',
    rowSub: (lives) => `${frInt(lives)} ${plural(lives, 'vie terminée', 'vies terminées')} par une mort`,
    tip: (side, value, total, pct, what) => `${side}\n${frInt(value)} ${what === 'lives' ? 'vies' : 'frags'} sur ${frInt(total)} (${pct})`,
    killsLine: (kills, pct, perLife) => `frags : ${frInt(kills)} · ${pct} · ${perLife} par vie`,
    killsLineAlone: (perLife, kills) => `${perLife} par vie · ${frInt(kills)}`,
    perLifeFmt: frDec1,
  },
  equipment: {
    title: 'Usage d’équipements',
    info:
      'Équipement tenu par le joueur (réapparition comprise), par famille : servi (mur posé, charge ' +
      'consommée), gardé sans servir, lâché ; barre fine : reste de l’équipe. Seules les familles tenues dans le ' +
      'lobby sont listées ; le répulseur, sans mesure d’usage, n’a pas de ligne.',
    used: 'Servi',
    kept: 'Gardé sans servir',
    dropped: 'Lâché',
    thinLegend: 'Barre fine : reste de l’équipe',
    rest: 'Reste de l’équipe',
    unmeasuredNames: { grapple: 'Grappin', thruster: 'Propulseur' },
    sub: (objects, taken) => (objects > 0 ? `${objects} ${plural(objects, 'objet', 'objets')}, dont ${taken} pris sur la carte` : '0 objet'),
    droppedSub: (n) => `${n} ${plural(n, 'lâché', 'lâchés')}`,
    unmeasured: 'Non mesuré : ni prise ni usage publiés pour cette famille',
    zeroTip: (player, family) => `${family} : 0 objet pour ${player}`,
    segTip: (who, n, part, total, pct) => `${who}\n${n} ${PART_FR[part][n > 1 ? 1 : 0]} sur ${total} (${pct})`,
    restLine: (used, kept, dropped) =>
      `reste de l’équipe : ${used} ${PART_FR.used[used > 1 ? 1 : 0]} · ${kept} ${PART_FR.kept[kept > 1 ? 1 : 0]} · ${dropped} ${PART_FR.dropped[dropped > 1 ? 1 : 0]}`,
    restUsedShare: (pct) => `${pct} servis`,
    restNone: 'reste de l’équipe : 0 objet',
  },
}

export const USAGES_CARDS_TEXT_EN: UsagesCardsText = {
  pctFmt: enPct,
  pctIntFmt: (v) => `${Math.round(v)}%`,
  intFmt: enInt,
  filCaption: (matches, filmed) => `${matches} ${plural(matches, 'match', 'matches')}, ${filmed} filmed`,
  dayFmt: (iso) => {
    const d = new Date(iso)
    return Number.isNaN(d.getTime()) ? '' : d.toLocaleDateString('en-GB', { day: '2-digit', month: '2-digit' })
  },
  maps: {
    others: 'Other maps',
    otherMapsFmt: (n) => `${n} maps · `,
    matchesFmt: (n) => `${n} ${plural(n, 'match', 'matches')}`,
    winsFmt: (n) => `${n} W`,
    lossesFmt: (n) => `${n} L`,
    othersFmt: (n) => `${n} O`,
    tipHead: (name, matches, filmed) => `${name} (${matches} ${plural(matches, 'match', 'matches')}, ${filmed} filmed)`,
  },
  playerFallback: 'Player',
  mine: {
    title: 'Pickup contribution',
    info:
      'Items the team picked up over the filmed matches in scope: the player’s share and the rest of the ' +
      'team’s, in counts, by decreasing volume. Lost power-ups: held without being activated, or dropped.',
    rest: 'Rest of the team',
    foldedFmt: (n) => `(${n}, folded)`,
    campWord: 'team',
    meTip: (player, name, me, camp) => `${player} · ${name}\n${me} of the team’s ${camp} pickups`,
    restTip: (name, rest, camp) => `Rest of the team · ${name}\n${rest} of the team’s ${camp} pickups`,
    lossesTitle: 'Lost power-ups',
    lossesFmt: (lost, taken) => `${lost} of ${taken}`,
  },
  lives: {
    title: 'Isolation',
    info: (unlocated, noRadar, unpublishable) =>
      'Lives ended by a death, sorted by the distance to the nearest teammate at the moment of death (within ' +
      'radar range or beyond); thin bar: kills made during those lives. ' +
      (noRadar > 0
        ? `Left out: lives with no teammate located (${enInt(unlocated)}), lives on a map with no known radar range (${enInt(noRadar)})`
        : `Left out: lives with no teammate located (${enInt(unlocated)})`) +
      (unpublishable > 0 ? `, lives from a match whose kill log is not publishable (${enInt(unpublishable)}).` : '.'),
    near: 'Within range of a teammate',
    alone: 'Isolated',
    thinLegend: 'Thin bar: the player’s kills during those lives',
    rowSub: (lives) => `${enInt(lives)} ${plural(lives, 'life', 'lives')} ended by a death`,
    tip: (side, value, total, pct, what) => `${side}\n${enInt(value)} ${what === 'lives' ? 'lives' : 'kills'} of ${enInt(total)} (${pct})`,
    killsLine: (kills, pct, perLife) => `kills: ${enInt(kills)} · ${pct} · ${perLife} per life`,
    killsLineAlone: (perLife, kills) => `${perLife} per life · ${enInt(kills)}`,
    perLifeFmt: enDec1,
  },
  equipment: {
    title: 'Equipment use',
    info:
      'Equipment held by the player (spawn equipment included), by family: used (wall placed, charge spent), ' +
      'kept without use, dropped; thin bar: rest of the team. Only families held in the lobby are listed; ' +
      'the repulsor, with no usage measure, has no row.',
    used: 'Used',
    kept: 'Kept without use',
    dropped: 'Dropped',
    thinLegend: 'Thin bar: rest of the team',
    rest: 'Rest of the team',
    unmeasuredNames: { grapple: 'Grappleshot', thruster: 'Thruster' },
    sub: (objects, taken) => (objects > 0 ? `${objects} ${plural(objects, 'item', 'items')}, ${taken} picked up on the map` : '0 items'),
    droppedSub: (n) => `${n} dropped`,
    unmeasured: 'Not measured: neither pickup nor use published for this family',
    zeroTip: (player, family) => `${family}: 0 items for ${player}`,
    segTip: (who, n, part, total, pct) => `${who}\n${n} ${PART_EN[part][0]} of ${total} (${pct})`,
    restLine: (used, kept, dropped) => `rest of the team: ${used} used · ${kept} kept · ${dropped} dropped`,
    restUsedShare: (pct) => `${pct} used`,
    restNone: 'rest of the team: 0 items',
  },
}
