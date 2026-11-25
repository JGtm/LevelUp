/**
 * chatterLines.ts — LES RÉPLIQUES DES SPARTANS d'un match : lesquelles s'affichent, à quelle famille elles appartiennent, et
 * lesquelles passent les filtres. Logique pure ; les libellés vivent dans `i18n/chatter.i18n.ts`.
 *
 * CE QUI S'AFFICHE. Une ligne du document (`chatter[]`) devient une réplique affichable quand le décodeur la NOMME (`name`
 * absent = build dont la disposition n'est pas mesurée : ni affichée ni comptée comme manquante), qu'elle a un joueur qui parle
 * (un bot, ou une unité sans piste, n'a pas de pion où la poser ni de ligne à son nom), et que son son est dans la table des
 * répliques PARLÉES (les cris de bouclier percé, d'atterrissage ou de recul n'y sont pas : décision du 2026-11-24).
 * La clé du texte est l'identifiant du son, jamais l'indice de réplique : il change de sens d'un build à l'autre.
 *
 * LA FAMILLE se lit sur le NOM que le film donne à CETTE réplique, pas sur son son : un son sort parfois sous un autre nom
 * (9 % des répliques du parc). Cinq familles, premier critère qui correspond.
 *
 * LA FRÉQUENCE se mesure sur le match : une phrase (un son porte une phrase et une voix) dite quatre fois ou plus est
 * « fréquente », une phrase dite deux fois ou moins est « rare » (règle de la maquette des réglages validée le 2026-11-24).
 */
import type { Locale } from '@/lib/i18n/locale'
import { pickText } from '@/lib/i18n/pickText'

import { chatterLanguage, chatterVoiceStem, type ChatterTable } from './chatterTable'
import { ownershipShown, partyOf, type HiddenPlayers } from './playerPanelLogic'

/** Les familles, dans l'ordre où le panneau les propose. */
export const CHATTER_FAMILY_IDS = ['enemy', 'shields', 'squadDown', 'praise', 'other'] as const
export type ChatterFamily = (typeof CHATTER_FAMILY_IDS)[number]

/**
 * La famille d'un nom court de réplique (`seeingenemynearby`, `shieldsdepleted`…), premier critère qui correspond :
 * un ennemi repéré (`seeingenemy…`), un bouclier (le nom dit `shields`), un équipier tombé (`lamentsquaddeath…`), une
 * félicitation (`praise…`), tout le reste.
 */
export function chatterFamilyOf(name: string): ChatterFamily {
  if (name.startsWith('seeingenemy')) return 'enemy'
  if (name.includes('shields')) return 'shields'
  if (name.startsWith('lamentsquaddeath')) return 'squadDown'
  if (name.startsWith('praise')) return 'praise'
  return 'other'
}

/** Les réglages de fréquence, dans l'ordre où le panneau les propose. */
export const CHATTER_FREQUENCIES = ['all', 'noFrequent', 'rare'] as const
export type ChatterFrequency = (typeof CHATTER_FREQUENCIES)[number]

/** Une phrase dite au moins ce nombre de fois dans le match est « fréquente ». */
export const CHATTER_FREQUENT_FROM = 4
/** Une phrase dite au plus ce nombre de fois dans le match est « rare ». */
export const CHATTER_RARE_UP_TO = 2

/** Une réplique affichable. */
export interface ChatterLine {
  /** Clé stable de rendu : le rang de la ligne dans le document. */
  key: string
  /** L'image du film (axe de `Point.T`) où le message est écrit. */
  frame: number
  /** Le joueur qui parle (xuid du document). */
  xuid: string
  /** L'identifiant du son : la clé de la table. */
  sound: string
  family: ChatterFamily
  /** Le texte, dans la langue des répliques (celle de l'interface, à défaut l'anglais). */
  text: string
  /** Le stem du fichier de voix dans cette même langue, ou `null` quand elle n'est pas versée. */
  voice: string | null
  /** Combien de fois cette phrase est dite dans tout le match (les lignes affichables seulement). */
  count: number
}

/** Une ligne du document, réduite à ce que la lecture a besoin d'y voir. */
export interface ChatterSource {
  t: number
  player?: string
  sound?: string
  name?: string
}

/** Les répliques affichables d'un match, dans l'ordre du film. */
export function buildChatterLines(
  chatter: readonly ChatterSource[],
  table: ChatterTable,
  locale: Locale,
): ChatterLine[] {
  const lines: ChatterLine[] = []
  chatter.forEach((c, index) => {
    if (!c.name || !c.player || !c.sound) return
    const entry = table.get(c.sound)
    if (!entry) return
    const language = chatterLanguage(entry, locale)
    if (language === null) return
    lines.push({
      key: `chatter-${index}`,
      frame: c.t,
      xuid: c.player,
      sound: c.sound,
      family: chatterFamilyOf(c.name),
      text: pickText(entry.texte, locale),
      voice: entry.audio.includes(language) ? chatterVoiceStem(c.sound, language) : null,
      count: 0,
    })
  })
  const counts = new Map<string, number>()
  for (const l of lines) counts.set(l.sound, (counts.get(l.sound) ?? 0) + 1)
  return lines.map((l) => ({ ...l, count: counts.get(l.sound) ?? 0 }))
}

/** Ce que les filtres du panneau demandent : les familles montrées et la fréquence. */
export interface ChatterFilter {
  families: Readonly<Record<ChatterFamily, boolean>>
  frequency: ChatterFrequency
}

/** Tout passe : les cinq familles, toutes fréquences — l'état d'ouverture. */
export const CHATTER_FILTER_ALL: ChatterFilter = {
  families: { enemy: true, shields: true, squadDown: true, praise: true, other: true },
  frequency: 'all',
}

/** La réplique passe-t-elle la famille et la fréquence demandées ? */
export function passesChatterFilter(line: Pick<ChatterLine, 'family' | 'count'>, filter: ChatterFilter): boolean {
  if (!filter.families[line.family]) return false
  if (filter.frequency === 'noFrequent') return line.count < CHATTER_FREQUENT_FROM
  if (filter.frequency === 'rare') return line.count <= CHATTER_RARE_UP_TO
  return true
}

/** Les répliques qui passent les filtres. Liste neuve, l'entrée intacte. */
export function filterChatterLines(lines: readonly ChatterLine[], filter: ChatterFilter): ChatterLine[] {
  return lines.filter((l) => passesChatterFilter(l, filter))
}

/** Le nombre de répliques par famille (sans filtre de fréquence : le compte dit ce que la famille contient). */
export function countByFamily(lines: readonly ChatterLine[]): Record<ChatterFamily, number> {
  const out: Record<ChatterFamily, number> = { enemy: 0, shields: 0, squadDown: 0, praise: 0, other: 0 }
  for (const l of lines) out[l.family]++
  return out
}

/**
 * Le prédicat qui dit si la réplique d'un joueur se montre : une réplique est à son joueur, elle suit les joueurs masqués
 * (« Seul » masque tous les autres) comme tout ce qui appartient à quelqu'un. `null` sans joueur masqué : tout se montre.
 */
export function chatterShownFor(hidden: HiddenPlayers | null): ((line: Pick<ChatterLine, 'xuid'>) => boolean) | null {
  const shown = ownershipShown(hidden)
  if (!shown) return null
  return (line) => {
    const party = partyOf(undefined, line.xuid)
    return party === null || shown([party])
  }
}
