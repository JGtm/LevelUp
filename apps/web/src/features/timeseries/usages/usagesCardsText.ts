/**
 * usagesCardsText.ts — les textes des cartes PROPRES à l'onglet « Usages » des Séries temporelles :
 * en-têtes de la grille par carte, « Mes prises dans mon camp », « Mes vies : près d'un coéquipier
 * ou seul », « Équipement pris, et ce que j'en ai fait » (maquette v4, `renderGrid`, `renderMine`,
 * carte « Mes vies », `renderEquip`). FR mot pour mot de la maquette ; parité FR / EN par le typage.
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
  mine: {
    title: string
    info: string
    me: string
    rest: string
    /** « Armes de râtelier (4, repliées) ». */
    foldedFmt: (n: number) => string
    /** Les mots de « moi 3 · camp 7 » au bout de chaque barre. */
    meWord: string
    campWord: string
    meTip: (name: string, me: number, camp: number) => string
    restTip: (name: string, rest: number, camp: number) => string
    lossesTitle: string
    lossesFmt: (lost: number, taken: number) => string
  }
  lives: {
    title: string
    /** ⓘ : les vies écartées sont comptées (`unlocated` : sans coéquipier situé ; `noRadar` : carte sans portée connue). */
    info: (unlocated: number, noRadar: number) => string
    near: string
    alone: string
    thinLegend: string
    rowLabel: string
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
    me: string
    rest: string
    /** Libellés des familles hors bilan (grappin, propulseur) ; les autres viennent du bloc d'usage. */
    unmeasuredNames: Record<string, string>
    sub: (objects: number, taken: number) => string
    droppedSub: (n: number) => string
    unmeasured: string
    zeroTip: (family: string) => string
    /** « Moi · Mur de protection\n52 servis sur 84 (61,9 %) ». */
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
  mine: {
    title: 'Mes prises dans mon camp',
    info:
      'Chaque objet pris par mon camp : ma part et celle du reste du camp, en comptes, triés par volume de mon ' +
      'camp. Une répartition, pas un classement. Les bonus perdus sont ceux gardés sans être activés ou lâchés.',
    me: 'Moi',
    rest: 'Reste de mon camp',
    foldedFmt: (n) => `(${n}, repliées)`,
    meWord: 'moi',
    campWord: 'camp',
    meTip: (name, me, camp) => `Moi · ${name}\n${me} des ${camp} prises de mon camp`,
    restTip: (name, rest, camp) => `Reste de mon camp · ${name}\n${rest} des ${camp} prises de mon camp`,
    lossesTitle: 'Bonus perdus',
    lossesFmt: (lost, taken) => `${lost} sur ${taken}`,
  },
  lives: {
    title: 'Mes vies : près d’un coéquipier ou seul',
    info: (unlocated, noRadar) =>
      'Chaque vie est rangée selon la distance au coéquipier le plus proche au moment de la mort : à moins ' +
      'd’une portée de radar, ou au-delà. La barre épaisse partage mes vies, la barre fine les frags obtenus ' +
      'pendant ces vies. ' +
      (noRadar > 0
        ? `Les vies terminées sans aucun coéquipier situé sont écartées (${frInt(unlocated)} ici), comme celles d’une carte sans portée de radar connue (${frInt(noRadar)}).`
        : `Les vies terminées sans aucun coéquipier situé sont écartées (${frInt(unlocated)} ici).`),
    near: 'Près d’un coéquipier',
    alone: 'Seul',
    thinLegend: 'Barre fine : mes frags pendant ces vies',
    rowLabel: 'Mes vies',
    rowSub: (lives) => `${frInt(lives)} ${plural(lives, 'vie terminée', 'vies terminées')} par une mort`,
    tip: (side, value, total, pct, what) => `${side}\n${frInt(value)} ${what === 'lives' ? 'vies' : 'frags'} sur ${frInt(total)} (${pct})`,
    killsLine: (kills, pct, perLife) => `frags : ${frInt(kills)} · ${pct} · ${perLife} par vie`,
    killsLineAlone: (perLife, kills) => `${perLife} par vie · ${frInt(kills)}`,
    perLifeFmt: frDec1,
  },
  equipment: {
    title: 'Équipement pris, et ce que j’en ai fait',
    info:
      'Pour chaque famille, ce que sont devenus mes objets : servis (posé pour le mur, charge consommée pour les ' +
      'autres), gardés sans servir, lâchés. Les comptes portent sur tout l’équipement tenu, celui de réapparition ' +
      'compris ; le sous-libellé dit combien en ont été pris sur la carte. La barre fine donne les mêmes trois ' +
      'parts pour le reste de mon camp. Le répulseur n’a pas de ligne : aucun canal ne mesure son usage.',
    used: 'Servi',
    kept: 'Gardé sans servir',
    dropped: 'Lâché',
    thinLegend: 'Barre fine : reste de mon camp',
    me: 'Moi',
    rest: 'Reste de mon camp',
    unmeasuredNames: { grapple: 'Grappin', thruster: 'Propulseur' },
    sub: (objects, taken) => (objects > 0 ? `${objects} ${plural(objects, 'objet', 'objets')}, dont ${taken} pris sur la carte` : '0 objet'),
    droppedSub: (n) => `${n} ${plural(n, 'lâché', 'lâchés')}`,
    unmeasured: 'Non mesuré : ni prise ni usage publiés pour cette famille',
    zeroTip: (family) => `${family} : 0 objet pour moi`,
    segTip: (who, n, part, total, pct) => `${who}\n${n} ${PART_FR[part][n > 1 ? 1 : 0]} sur ${total} (${pct})`,
    restLine: (used, kept, dropped) =>
      `reste de mon camp : ${used} ${PART_FR.used[used > 1 ? 1 : 0]} · ${kept} ${PART_FR.kept[kept > 1 ? 1 : 0]} · ${dropped} ${PART_FR.dropped[dropped > 1 ? 1 : 0]}`,
    restUsedShare: (pct) => `${pct} servis`,
    restNone: 'reste de mon camp : 0 objet',
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
  mine: {
    title: 'My pickups within my side',
    info:
      'Each item my side picked up: my share and the rest of the side’s, in counts, sorted by my side’s volume. ' +
      'A split, not a ranking. Lost power-ups are those held without being activated, or dropped.',
    me: 'Me',
    rest: 'Rest of my side',
    foldedFmt: (n) => `(${n}, folded)`,
    meWord: 'me',
    campWord: 'side',
    meTip: (name, me, camp) => `Me · ${name}\n${me} of my side’s ${camp} pickups`,
    restTip: (name, rest, camp) => `Rest of my side · ${name}\n${rest} of my side’s ${camp} pickups`,
    lossesTitle: 'Lost power-ups',
    lossesFmt: (lost, taken) => `${lost} of ${taken}`,
  },
  lives: {
    title: 'My lives: near a teammate or alone',
    info: (unlocated, noRadar) =>
      'Each life is sorted by the distance to the nearest teammate at the moment of death: within one radar ' +
      'range, or beyond. The thick bar splits my lives, the thin bar the kills made during those lives. ' +
      (noRadar > 0
        ? `Lives that ended with no teammate located are left out (${enInt(unlocated)} here), as are those on a map with no known radar range (${enInt(noRadar)}).`
        : `Lives that ended with no teammate located are left out (${enInt(unlocated)} here).`),
    near: 'Near a teammate',
    alone: 'Alone',
    thinLegend: 'Thin bar: my kills during those lives',
    rowLabel: 'My lives',
    rowSub: (lives) => `${enInt(lives)} ${plural(lives, 'life', 'lives')} ended by a death`,
    tip: (side, value, total, pct, what) => `${side}\n${enInt(value)} ${what === 'lives' ? 'lives' : 'kills'} of ${enInt(total)} (${pct})`,
    killsLine: (kills, pct, perLife) => `kills: ${enInt(kills)} · ${pct} · ${perLife} per life`,
    killsLineAlone: (perLife, kills) => `${perLife} per life · ${enInt(kills)}`,
    perLifeFmt: enDec1,
  },
  equipment: {
    title: 'Equipment picked up, and what I did with it',
    info:
      'For each family, what became of my items: used (placed for the wall, charge spent for the others), kept ' +
      'without use, dropped. The counts cover all equipment held, spawn equipment included; the sub-label says ' +
      'how many were picked up on the map. The thin bar gives the same three shares for the rest of my side. ' +
      'The repulsor has no row: no channel measures its use.',
    used: 'Used',
    kept: 'Kept without use',
    dropped: 'Dropped',
    thinLegend: 'Thin bar: rest of my side',
    me: 'Me',
    rest: 'Rest of my side',
    unmeasuredNames: { grapple: 'Grappleshot', thruster: 'Thruster' },
    sub: (objects, taken) => (objects > 0 ? `${objects} ${plural(objects, 'item', 'items')}, ${taken} picked up on the map` : '0 items'),
    droppedSub: (n) => `${n} dropped`,
    unmeasured: 'Not measured: neither pickup nor use published for this family',
    zeroTip: (family) => `${family}: 0 items for me`,
    segTip: (who, n, part, total, pct) => `${who}\n${n} ${PART_EN[part][0]} of ${total} (${pct})`,
    restLine: (used, kept, dropped) => `rest of my side: ${used} used · ${kept} kept · ${dropped} dropped`,
    restUsedShare: (pct) => `${pct} used`,
    restNone: 'rest of my side: 0 items',
  },
}
