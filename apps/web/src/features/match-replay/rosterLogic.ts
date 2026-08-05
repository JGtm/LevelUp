/**
 * rosterLogic.ts — LA JOINTURE ENTRE LE FILM ET LA BASE, et l'état d'un joueur à une image.
 *
 * DEUX SOURCES, DEUX RÔLES, ET UNE SEULE CLÉ. Le film porte ce qui se passe — positions,
 * vies, morts, armes, bouclier — et l'identifie par XUID. La base porte qui sont les gens :
 * gamertag, équipe, K/D/A du match. Aucune des deux ne sait faire le travail de l'autre, et
 * l'artefact de rejeu n'essaie pas : il publie le xuid, qui est la seule clé sur laquelle une
 * jointure ne suppose rien.
 *
 * POURQUOI PAS UN INDEX. Ce chantier a déjà publié un « index de joueur » comme une découverte
 * avant de constater que c'était son propre tri alphabétique. Un index est un ORDRE ; le xuid
 * est une IDENTITÉ. La règle qui en est sortie tient en une ligne et vaut ici : on ne joint
 * jamais sur un rang.
 *
 * Tout ce fichier est PUR : aucun React, aucun canvas, donc testable.
 */
import type { MatchScoreboardRow } from '@/lib/api/types'

import { catalogText, type CatalogLabel } from './catalogLabel'
import type { ReplayLocale } from './i18n'
import { heldReading, isAliveAt, trackWindow } from './replayLogic'
import type {
  ReplayDocumentReady,
  ReplayInventoryReady,
  ReplayTrackReady,
} from './replayNormalize'

/**
 * ReplayPlayer — un joueur du rejeu : son identité (base) et ses vies (film).
 *
 * `board` est absent quand le xuid du film n'a aucune ligne au scoreboard. Ce n'est pas censé
 * arriver, et c'est précisément pour cela qu'on le représente au lieu de l'écraser : un joueur
 * du film introuvable dans la base est un signal, pas un détail à masquer.
 */
export interface ReplayPlayer {
  xuid: string
  /** Ligne du scoreboard du match, quand la jointure aboutit. */
  board?: MatchScoreboardRow
  /** Gamertag écrit par le FILM. Sert quand la base n'a rien à dire sur ce joueur. */
  filmName?: string
  /** Toutes les vies de ce joueur, dans l'ordre du temps. */
  lives: ReplayTrackReady[]
  /** Index de la couleur de série attribuée à ce joueur (stable sur tout le rejeu). */
  colorIndex: number
}

/**
 * playerName choisit le nom à afficher.
 *
 * LA BASE D'ABORD, LE FILM ENSUITE. Les deux disent le même gamertag, mais la base est celle
 * qui suit un changement de pseudo : le film fige le nom au jour de la partie. En cas
 * d'absence de ligne de scoreboard, le nom du film évite de tomber sur « joueur inconnu »
 * pour quelqu'un que le film nomme parfaitement.
 */
export function playerName(player: ReplayPlayer): string | null {
  return player.board?.gamertag || player.filmName || null
}

/**
 * ReplayTeamGroup — un camp. `side` reprend le libellé de la base (`team_side`) : le film ne
 * porte aucune notion d'équipe, on n'en invente donc pas la dénomination.
 */
export interface ReplayTeamGroup {
  side: string | null
  players: ReplayPlayer[]
}

/**
 * buildPlayers regroupe les vies par joueur et les joint au scoreboard.
 *
 * LES VIES ANONYMES NE SONT PAS INVENTÉES DE PROPRIÉTAIRE : une trace sans xuid n'entre dans
 * aucun joueur. Elle continue d'être dessinée sur la carte — elle existe — mais elle n'ajoute
 * ni une fiche ni une ligne à personne.
 *
 * LA COULEUR EST STABLE SUR TOUT LE REJEU : elle est attribuée AU JOUEUR, pas à la vie. Sans
 * cela, un joueur changerait de couleur à chaque réapparition (99 traces pour 8 joueurs) et
 * suivre quelqu'un des yeux deviendrait impossible.
 */
export function buildPlayers(
  doc: ReplayDocumentReady,
  scoreboard: MatchScoreboardRow[],
): ReplayPlayer[] {
  const byXUID = new Map<string, ReplayPlayer>()
  // L'ordre du roster du film donne un ordre STABLE et reproductible ; à défaut, l'ordre
  // d'apparition des traces. Jamais l'ordre d'itération d'une Map remplie au hasard.
  for (const entry of doc.roster ?? []) {
    byXUID.set(entry.xuid, {
      xuid: entry.xuid,
      filmName: entry.name,
      lives: [],
      colorIndex: byXUID.size,
    })
  }
  for (const track of doc.tracks) {
    if (!track.xuid) continue
    let p = byXUID.get(track.xuid)
    if (!p) {
      p = { xuid: track.xuid, lives: [], colorIndex: byXUID.size }
      byXUID.set(track.xuid, p)
    }
    p.lives.push(track)
  }
  const board = new Map(scoreboard.map((r) => [r.xuid, r]))
  for (const p of byXUID.values()) {
    p.board = board.get(p.xuid)
    p.lives.sort((a, b) => trackWindow(a).start - trackWindow(b).start)
  }
  return [...byXUID.values()]
}

/**
 * groupByTeam range les joueurs par camp, dans un ordre stable.
 *
 * UN JOUEUR SANS LIGNE DE SCOREBOARD N'A PAS D'ÉQUIPE et se retrouve dans un groupe à part
 * (`side: null`). Le placer arbitrairement dans un camp fabriquerait une appartenance.
 */
export function groupByTeam(players: ReplayPlayer[]): ReplayTeamGroup[] {
  const groups = new Map<string, ReplayTeamGroup>()
  for (const p of players) {
    const side = p.board?.team_side ?? null
    const key = side ?? ''
    let g = groups.get(key)
    if (!g) {
      g = { side, players: [] }
      groups.set(key, g)
    }
    g.players.push(p)
  }
  return [...groups.values()].sort((a, b) => (a.side ?? '￿').localeCompare(b.side ?? '￿'))
}

/**
 * PlayerState — ce qu'un joueur est à une image donnée, lu dans le film.
 *
 * `shield` est null quand aucune mesure n'existe dans la fenêtre de maintien : c'est
 * « on ne sait pas », à distinguer absolument d'un bouclier à zéro, qui est une mesure.
 */
export interface PlayerState {
  alive: boolean
  /** La vie en cours, quand il y en a une. */
  life: ReplayTrackReady | null
  /** Bouclier lu, avec l'âge de la lecture ; null = aucune mesure fraîche. */
  shield: { value: number; age: number } | null
  /** Nombre d'images écoulées depuis la fin de la dernière vie (mort) ; -1 s'il est en vie. */
  sinceDeath: number
  /**
   * Image de RÉAPPARITION, LUE : c'est l'image de départ de la vie suivante du même joueur, pas
   * une constante ajoutée à l'instant de la mort. -1 quand aucune vie ne suit — ce qui arrive
   * pour un survivant de fin de partie, et doit s'afficher comme une lacune, jamais comme un
   * délai deviné.
   */
  respawnFrame: number
}

/**
 * playerStateAt lit l'état d'un joueur à une image.
 *
 * LE DÉLAI DE RÉAPPARITION EST LU, PAS DÉDUIT. Mesuré sur le film de référence : 90 épisodes de
 * mort, 82 avec un retour lisible, médiane 8,0 s et 66 sur 82 exactement à 7,9-8,0 s. C'est un
 * palier net — mais mesuré sur UN match, ce qui n'en fait pas une constante du jeu. Les 8
 * épisodes sans retour restent sans délai affiché.
 */
export function playerStateAt(
  player: ReplayPlayer,
  frame: number,
  shieldHold: number,
): PlayerState {
  const live = player.lives.find((l) => isAliveAt(l, frame)) ?? null
  if (live) {
    return {
      alive: true,
      life: live,
      shield: heldReading(live.points, frame, (p) => p.sh, shieldHold),
      sinceDeath: -1,
      respawnFrame: -1,
    }
  }
  // Mort : la dernière vie CLOSE avant cette image date la mort ; la suivante, si elle existe,
  // date le retour.
  let last: ReplayTrackReady | null = null
  let next = -1
  for (const l of player.lives) {
    const w = trackWindow(l)
    if (w.end < frame) last = l
    else if (w.start > frame && next === -1) next = w.start
  }
  return {
    alive: false,
    life: null,
    shield: null,
    sinceDeath: last ? frame - trackWindow(last).end : -1,
    respawnFrame: next,
  }
}

/**
 * LoadoutReading — les armes portées et l'ÂGE de cette lecture.
 *
 * L'ÂGE EST INDISSOCIABLE DE LA VALEUR. Le loadout ne se lit qu'aux images-clés, une toutes les
 * ~20 s. Entre deux, ce qui est affiché est la dernière lecture connue, et la faire passer pour
 * l'état courant était un défaut réel : sur les 21 899 fiches d'un match, l'âge médian de cette
 * lecture est de 8,4 s, et 7,1 % seulement ont moins d'une seconde. L'estompage n'est donc pas
 * un ornement rare — il sert neuf fois sur dix.
 */
export interface LoadoutReading {
  weapons: string[]
  age: number
}

/**
 * loadoutAt rend les armes portées par un SLOT à une image, avec l'âge de la lecture.
 *
 * La recherche porte sur le slot et non sur le joueur : le loadout est écrit pour un biped, et
 * un slot est réattribué à chaque réapparition. C'est aussi ce qui rend le report SÛR — une
 * dotation ne peut pas franchir une mort, puisqu'elle ne survit pas à son porteur.
 */
export function loadoutAt(doc: ReplayDocumentReady, slot: number, frame: number): LoadoutReading | null {
  let best: LoadoutReading | null = null
  for (const l of doc.loadouts ?? []) {
    if (l.slot !== slot || l.t > frame) continue
    const age = frame - l.t
    if (!best || age < best.age) best = { weapons: l.w, age }
  }
  return best
}

/** InventoryReading — l'inventaire d'un slot et l'ÂGE de cette lecture, en frames. */
export interface InventoryReading {
  state: ReplayInventoryReady
  age: number
}

/**
 * inventoryAt rend le dernier inventaire lu pour un SLOT au plus tard à `frame`, avec son âge.
 *
 * LA RECHERCHE PORTE SUR LE SLOT, PAS SUR LE JOUEUR, et c'est ce qui rend le report SÛR : un
 * slot est réattribué à chaque réapparition, donc une dotation ne peut jamais franchir une
 * mort. Chercher par joueur ferait survivre un inventaire à son porteur.
 */
export function inventoryAt(
  doc: ReplayDocumentReady,
  slot: number,
  frame: number,
): InventoryReading | null {
  let best: InventoryReading | null = null
  for (const inv of doc.inventory ?? []) {
    if (inv.slot !== slot || inv.t > frame) continue
    const age = frame - inv.t
    if (!best || age < best.age) best = { state: inv, age }
  }
  return best
}

/**
 * grenadesCarried rend les types de grenade PORTÉS, avec leur nom et leur compteur.
 *
 * LES COMPTEURS À ZÉRO SONT ÉCARTÉS DE L'AFFICHAGE mais pas de la lecture : le tableau publié
 * est complet, et un zéro y signifie « ce type, aucune en réserve ». Montrer quatre types dont
 * trois à zéro noierait celui qui compte.
 */
export function grenadesCarried(
  state: ReplayInventoryReady,
  labels: CatalogLabel[] | undefined,
  locale: ReplayLocale,
): { rank: number; name: string; count: number }[] {
  if (!state.g) return []
  const out: { rank: number; name: string; count: number }[] = []
  state.g.forEach((count, rank) => {
    if (count <= 0) return
    // Sans table, le RANG s'affiche tel quel : c'est ce que le document dit, et c'est
    // vrai. Inventer un nom serait pire (cf. catalogLabel.ts).
    out.push({ rank, name: catalogText(labels?.[rank], locale) ?? `rang ${rank}`, count })
  })
  return out
}

/**
 * selectedGrenadeRank désigne le type ÉQUIPÉ, celui qui partira au prochain lancer.
 *
 * IL N'EST DÉDUIT QUE QUAND IL NE PEUT PAS ÊTRE AUTRE CHOSE : un seul type porté, donc c'est
 * lui. Dès qu'un joueur porte deux types, le sélecteur n'est pas dans ce que nous savons lire —
 * on rend null, et l'écran n'en désigne aucun. Deviner ici reviendrait à afficher une certitude
 * qu'on n'a pas.
 */
export function selectedGrenadeRank(state: ReplayInventoryReady): number | null {
  const carried = (state.g ?? []).map((c, r) => ({ c, r })).filter((x) => x.c > 0)
  return carried.length === 1 ? carried[0].r : null
}
