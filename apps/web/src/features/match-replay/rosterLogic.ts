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
import { displayPlayerName } from '@/lib/players/displayName'

import { catalogText, type CatalogLabel } from './catalogLabel'
import type { ReplayLocale } from './i18n'
import type { PlayerMarkKind } from './playerMarks'
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
 * CE QUI EST STABLE SUR TOUT LE REJEU (couleur, marque d'identité, nom) est attribué AU
 * JOUEUR, jamais à la vie : c'est `indexBySlot` ci-dessous qui redescend l'attribut sur
 * chacune de ses vies. Sans cela, un joueur changerait de couleur à chaque réapparition
 * (99 traces pour 8 joueurs) et suivre quelqu'un des yeux deviendrait impossible.
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
    })
  }
  for (const track of doc.tracks) {
    if (!track.xuid) continue
    let p = byXUID.get(track.xuid)
    if (!p) {
      p = { xuid: track.xuid, lives: [] }
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
 * indexBySlot — LE PASSAGE « ce qui appartient au JOUEUR redescend sur chacune de ses VIES ».
 *
 * Le calque de la carte ne connaît que des SLOTS (une vie = un slot de biped, réattribué à
 * chaque réapparition) ; tout ce qui identifie durablement quelqu'un — sa couleur d'équipe,
 * sa marque « moi / ami », son nom — appartient au JOUEUR. Trois attributs, une seule
 * descente : sans ce foyer, la même boucle serait recopiée trois fois (CLAUDE.md n°6).
 *
 * Une vie SANS propriétaire (trace sans xuid) n'apparaît dans aucune table : l'appelant sert
 * alors son propre repli — encre neutre pour la couleur, aucune marque, aucun nom.
 */
export function indexBySlot<T>(
  players: readonly ReplayPlayer[],
  valueOf: (player: ReplayPlayer) => T,
): Map<number, T> {
  const bySlot = new Map<number, T>()
  for (const p of players) {
    const value = valueOf(p)
    for (const life of p.lives) bySlot.set(life.slot, value)
  }
  return bySlot
}

/**
 * colorBySlot — LA COULEUR D'UNE VIE EST CELLE DE SON PROPRIÉTAIRE, et c'est sa couleur
 * d'ÉQUIPE (décision D1 du plan d'habillage, 2026-08-16, amendée par l'utilisateur) : les
 * tokens `team-ally` / `team-enemy` que les réglages d'accessibilité peuvent surcharger,
 * jamais une couleur de série indexée sur la trace. Un joueur gardait jusque-là une teinte
 * par VIE : il changeait de couleur à chaque réapparition.
 *
 * `colorOf` rend la couleur d'un camp (l'appelant résout les tokens), `isAlly` dit le camp
 * d'un xuid (côté du joueur de la page), `neutral` sert quand on ne sait rien de l'identité.
 */
export function colorBySlot(
  players: readonly ReplayPlayer[],
  colorOf: (ally: boolean) => string,
  isAlly: (xuid: string) => boolean,
  neutral: string,
): Map<number, string> {
  return indexBySlot(players, (p) => (p.xuid ? colorOf(isAlly(p.xuid)) : neutral))
}

/** markBySlot redescend la marque d'identité (« moi », « ami ») du joueur sur ses vies. */
export function markBySlot(
  players: readonly ReplayPlayer[],
  marks: ReadonlyMap<string, PlayerMarkKind>,
): Map<number, PlayerMarkKind | undefined> {
  return indexBySlot(players, (p) => marks.get(p.xuid))
}

/**
 * nameBySlot redescend le NOM D'AFFICHAGE du joueur sur ses vies — celui des fiches et du
 * fil (`displayPlayerName`), jamais un xuid brut. Une vie sans propriétaire n'y figure pas :
 * l'étiquette de la carte reste alors vide plutôt que d'écrire « Joueur #### ».
 */
export function nameBySlot(players: readonly ReplayPlayer[]): Map<number, string> {
  return indexBySlot(players, (p) => displayPlayerName(playerName(p), p.xuid))
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
  /**
   * Bouclier : la dernière mesure de la vie avec son âge, ou 1,0 (âge 0) avant la première
   * mesure — on apparaît plein, et le flux différentiel ne retransmet que ce qui change.
   * Null UNIQUEMENT quand le document ne porte pas ce champ (cf. VitalityPresence) ou que
   * le joueur est mort.
   */
  shield: { value: number; age: number } | null
  /**
   * Santé, même contrat que le bouclier — y compris le plein d'apparition. Elle est
   * répliquée AU CHANGEMENT et rarement transmise (médiane zéro échantillon par vie sur le
   * film de référence) : dans Halo le bouclier encaisse d'abord, une vie sans mesure de
   * santé est une vie restée intacte.
   */
  health: { value: number; age: number } | null
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
 * VitalityPresence — le document porte-t-il la donnée, champ par champ ? C'est la garde
 * multi-titre du « plein au spawn » : un titre dont le film ne transmet JAMAIS une
 * vitalité ne doit pas afficher des barres pleines inventées — il n'affiche pas de barre.
 * Dégradation par ABSENCE DE DONNÉE, jamais par comparaison de slug.
 */
export interface VitalityPresence {
  shield: boolean
  health: boolean
}

/** vitalityPresence mesure, une fois par document, quels champs de vitalité existent. */
export function vitalityPresence(doc: ReplayDocumentReady): VitalityPresence {
  let shield = false
  let health = false
  for (const t of doc.tracks) {
    for (const p of t.points) {
      if (p.sh !== undefined) shield = true
      if (p.hp !== undefined) health = true
      if (shield && health) return { shield, health }
    }
  }
  return { shield, health }
}

/**
 * playerStateAt lit l'état d'un joueur à une image.
 *
 * LE REPORT DE VITALITÉ COUVRE LA VIE ENTIÈRE, et c'est une lecture juste : le flux est
 * différentiel — non retransmis veut dire INCHANGÉ — et les points appartiennent à la vie,
 * donc le report ne franchit jamais une mort. Ce qui vieillit doit se VOIR : l'âge voyage
 * avec la valeur et l'affichage l'estompe (cf. barres).
 *
 * AVANT LA PREMIÈRE MESURE D'UNE VIE, LA VALEUR JUSTE EST 1,0 : on apparaît vie et bouclier
 * PLEINS (règle du jeu), et le film ne retransmet que ce qui CHANGE — « rien d'arrivé »
 * veut dire « plein », pas « inconnu ». C'est la lecture du POC, rétablie sur décision
 * utilisateur (2026-08-12). Elle est GARDÉE par VitalityPresence : un document qui ne
 * porte jamais le champ n'affiche rien (titre sans décodage film).
 *
 * LE DÉLAI DE RÉAPPARITION EST LU, PAS DÉDUIT. Mesuré sur le film de référence : 90 épisodes de
 * mort, 82 avec un retour lisible, médiane 8,0 s et 66 sur 82 exactement à 7,9-8,0 s. C'est un
 * palier net — mais mesuré sur UN match, ce qui n'en fait pas une constante du jeu. Les 8
 * épisodes sans retour restent sans délai affiché.
 */
export function playerStateAt(
  player: ReplayPlayer,
  frame: number,
  presence: VitalityPresence,
): PlayerState {
  const live = player.lives.find((l) => isAliveAt(l, frame)) ?? null
  if (live) {
    const spawnFull = { value: 1, age: 0 }
    const shield = heldReading(live.points, frame, (p) => p.sh, Number.POSITIVE_INFINITY)
    const health = heldReading(live.points, frame, (p) => p.hp, Number.POSITIVE_INFINITY)
    return {
      alive: true,
      life: live,
      shield: shield ?? (presence.shield ? spawnFull : null),
      health: health ?? (presence.health ? spawnFull : null),
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
    health: null,
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
 *
 * AVANT LA PREMIÈRE IMAGE-CLÉ D'UNE VIE, la lecture rendue est la plus proche À VENIR du même
 * slot — donc de la MÊME vie, jamais d'une autre : c'est ce qui rend le repli sûr. L'âge est
 * alors NÉGATIF et publié tel quel : l'affichage l'estompe sur sa valeur absolue et
 * l'infobulle le dit « à venir », jamais déguisé en lecture passée. C'est la doctrine du POC
 * (readAgeAt) : 25,2 % de ses fiches affichaient des armes lues dans le futur — sans ce
 * repli, chaque début de vie dit « armes non lues » pendant jusqu'à 20 s.
 */
export function loadoutAt(doc: ReplayDocumentReady, slot: number, frame: number): LoadoutReading | null {
  const read = nearestReading(doc.loadouts ?? [], slot, frame)
  return read ? { weapons: read.value.w, age: read.age } : null
}

/**
 * nearestReading — LE FOYER CANONIQUE du report de lecture (règle ≤2 copies, CLAUDE.md n°6).
 *
 * Trois calques lisent le film par ÉCHANTILLONS ESPACÉS et doivent afficher, à une image
 * donnée, la dernière lecture connue : les armes portées, l'inventaire, et — depuis le
 * 2026-08-14 — la capacité d'armure. La troisième copie a déclenché la centralisation ; le
 * garde-rail `rosterLogic.guard.test.ts` interdit d'en réécrire une quatrième à la main.
 *
 * LA RÈGLE, IDENTIQUE POUR LES TROIS :
 *   - la recherche porte sur le SLOT, jamais sur le joueur. Un slot est réattribué à chaque
 *     réapparition : c'est ce qui rend le report SÛR, une lecture ne peut pas franchir une
 *     mort puisqu'elle ne survit pas à son porteur ;
 *   - avant la première lecture d'une vie, on rend la plus proche À VENIR du même slot, avec
 *     un âge NÉGATIF publié tel quel. L'affichage l'estompe sur sa valeur absolue et le dit
 *     « à venir » — jamais déguisé en lecture passée.
 */
export function nearestReading<T extends { slot: number; t: number }>(
  samples: readonly T[],
  slot: number,
  frame: number,
): { value: T; age: number } | null {
  let best: { value: T; age: number } | null = null
  let ahead: { value: T; age: number } | null = null
  for (const s of samples) {
    if (s.slot !== slot) continue
    const age = frame - s.t
    if (age < 0) {
      // La plus PROCHE à venir : l'âge le moins négatif.
      if (!ahead || age > ahead.age) ahead = { value: s, age }
      continue
    }
    if (!best || age < best.age) best = { value: s, age }
  }
  return best ?? ahead
}

/**
 * AbilityReading — le RANG de palette de la capacité portée, et l'âge de cette lecture.
 *
 * DEUX CANAUX, UNE SEULE GRANDEUR (cf. l'artefact, abilities.go) : `i48` transmet le rang
 * complet dans les paquets delta, environ une fois par vie ; le canal d'image-clé est dense
 * mais BORGNE — il ne voit que les rangs 16 à 23. On ne les départage pas : la lecture la
 * plus récente gagne, quel que soit son canal, parce que les deux disent la même chose.
 */
export interface AbilityReading {
  rank: number
  age: number
  src: string
}

/** abilityAt rend le dernier rang de capacité lu pour un SLOT, avec l'âge de la lecture. */
export function abilityAt(
  doc: ReplayDocumentReady,
  slot: number,
  frame: number,
): AbilityReading | null {
  const read = nearestReading(doc.abilities ?? [], slot, frame)
  return read ? { rank: read.value.r, age: read.age, src: read.value.src } : null
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
 *
 * AVANT LA PREMIÈRE IMAGE-CLÉ D'UNE VIE, la lecture rendue est la plus proche À VENIR du
 * même slot — même repli que loadoutAt, et même honnêteté : âge NÉGATIF publié tel quel,
 * estompé sur sa valeur absolue, dit « à venir » en infobulle. DÉCISION UTILISATEUR
 * (2026-08-12) : elle va AU-DELÀ du POC, qui refusait la lecture future pour les compteurs
 * (grenades, munitions) au motif qu'ils sont volatils — l'arbitrage produit est qu'une
 * dotation de spawn affichée avec son âge « à venir » informe mieux que vingt secondes de
 * vide. Le repli ne franchit jamais une mort : un slot est une vie.
 */
export function inventoryAt(
  doc: ReplayDocumentReady,
  slot: number,
  frame: number,
): InventoryReading | null {
  const read = nearestReading(doc.inventory ?? [], slot, frame)
  return read ? { state: read.value, age: read.age } : null
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
 * GrenadeSelection — le type ÉQUIPÉ, celui qui partira au prochain lancer, avec sa
 * PROVENANCE. Les trois formes ne se confondent jamais :
 *   - { rank, read: true }  : LU dans le film (sélecteur i47 de l'image-clé) ;
 *   - { rank, read: false } : DÉDUIT — un seul type porté, donc c'est lui ;
 *   - 'indeterminate'       : plusieurs types portés et sélecteur non lu — l'écran doit le
 *     DIRE (« sél. ? »), pas choisir ;
 *   - null                  : rien à désigner (compteurs non lus, ou aucun type porté).
 */
export type GrenadeSelection = { rank: number; read: boolean } | 'indeterminate' | null

/**
 * selectedGrenade désigne le type équipé.
 *
 * LA LECTURE PRIME LA DÉDUCTION : le sélecteur du film (`gs`) est publié sous garde de
 * cohérence (masque == compteurs, unanimité) — quand il est là, c'est lui. À défaut, la
 * déduction ne vaut que quand elle ne peut pas être autre chose : un seul type porté.
 * Dès qu'un joueur porte deux types sans sélecteur lu, on rend 'indeterminate' : deviner
 * reviendrait à afficher une certitude qu'on n'a pas.
 */
export function selectedGrenade(state: ReplayInventoryReady): GrenadeSelection {
  const carried = (state.g ?? []).map((c, r) => ({ c, r })).filter((x) => x.c > 0)
  if (carried.length === 0) return null
  const gs = state.gs
  if (gs !== undefined && carried.some((x) => x.r === gs)) {
    return { rank: gs, read: true }
  }
  if (carried.length === 1) return { rank: carried[0].r, read: false }
  return 'indeterminate'
}
