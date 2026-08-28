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

import type { PlayerMarkKind } from './playerMarks'
import { heldReading, isAliveAt, trackWindow } from './replayLogic'
import type { ReplayDocumentReady, ReplayTrackReady } from './replayNormalize'

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
 * aucun joueur. Elle n'ajoute donc ni fiche ni ligne à personne — et son slot n'entrant dans
 * aucune des tables ci-dessous, le calque ne la dessine pas non plus (`colorOfSlot` rend
 * `null`, cf. useSlotIdentity) : ce sont les caméras et les spectateurs de fin de partie.
 *
 * CE QUI IDENTIFIE QUELQU'UN (couleur, marque d'identité, nom) appartient AU JOUEUR, jamais à
 * la vie ; le calque le résout par SLOT et par IMAGE (`buildSlotOwnership` / `ownerAtFrame`
 * ci-dessous, dérivés `colorResolver`/`markResolver`/`nameResolver`). Sans cela, un joueur
 * changerait de couleur à chaque réapparition (99 traces pour 8 joueurs), et — un slot de
 * biped étant réattribué entre manches — un slot montrerait un seul joueur pour tout le match :
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
 * indexBySlot — AGRÉGAT MATCH : « ce qui appartient au JOUEUR redescend sur chacune de ses
 * VIES », effondré en UNE valeur par slot (DERNIER GAGNANT sur l'ordre des joueurs puis des
 * vies).
 *
 * C'EST UN AGRÉGAT SUR TOUT LE MATCH, PAS UNE LECTURE PAR IMAGE. Il sert au comptage d'usage
 * d'équipement (`equipmentUsageLogic`), où l'on veut « le joueur du slot » toutes vies
 * confondues. Pour le RENDU par image, ne pas s'en servir : un slot de biped est réattribué
 * entre réapparitions ET entre manches, il n'a donc pas UN propriétaire mais le propriétaire
 * de la vie qui l'occupe À CETTE IMAGE — c'est `buildSlotOwnership` / `ownerAtFrame`. (En
 * multi-manche, cet agrégat attribue au dernier propriétaire du slot : dette connue, assumée
 * ici parce que le comptage d'usage est au niveau du match, pas de l'image.)
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
 * SlotOwnership — QUI OCCUPE UN SLOT À UNE IMAGE. Le remède à l'effondrement d'`indexBySlot`
 * pour le rendu : un slot de biped est réattribué entre réapparitions ET entre manches, son
 * propriétaire n'est donc pas une constante du match mais la VIE qui couvre l'image lue.
 *
 * SANS AMBIGUÏTÉ : les vies d'un slot sont disjointes (un biped a un seul occupant à la fois),
 * `buildPlayers` les a triées dans le temps, et la recherche rend la première vie qui couvre
 * l'image. Un slot libre entre deux vies — ou avant la première, ou après la dernière — n'a
 * pas de propriétaire à cette image : `ownerAtFrame` rend `null`, le calque ne dessine rien.
 */
export interface SlotOwnership {
  ownerAtFrame(slot: number, frame: number): ReplayPlayer | null
  /**
   * ownerAtFrameOrLast — la vie qui COUVRE l'image, sinon la vie la plus récente TERMINÉE avant
   * elle (la « vie juste précédente »). Réservé aux consommateurs de FRONTIÈRE dont l'événement
   * est daté À l'INSTANT OÙ LE PROPRIÉTAIRE VIENT DE QUITTER LE SLOT : un objet LÂCHÉ à la mort
   * porte `t0 = finVie + 1` (le poseur n'occupe plus le slot), et un kill posthume/échange
   * peut tomber une image après la fin de vie du tueur. `ownerAtFrame` y rendrait `null` (donc
   * une encre neutre au lieu de la couleur d'équipe) : c'est un TROU à combler par la vie qui
   * précède, pas par le dernier-gagnant du match — en multi-manche, le résultat reste le joueur
   * de LA manche concernée (SHROOM s'il vient de mourir, DinoR00 sinon). NE PAS l'employer pour
   * le rendu continu des marqueurs/vies : eux doivent rester `null` dans les trous.
   */
  ownerAtFrameOrLast(slot: number, frame: number): ReplayPlayer | null
}

/** Une vie possédée, réduite à sa fenêtre et à son propriétaire (l'index n'a besoin de rien d'autre). */
interface OwnedLife {
  start: number
  end: number
  player: ReplayPlayer
}

/**
 * buildSlotOwnership indexe, par slot, les vies qui l'ont occupé — triées par début — et rend
 * le résolveur `ownerAtFrame`. Construit UNE FOIS par document (mémoïsé par l'appelant) ; la
 * résolution est un balayage court (une poignée de vies par slot) avec sortie anticipée dès
 * qu'une vie commence après l'image.
 */
export function buildSlotOwnership(players: readonly ReplayPlayer[]): SlotOwnership {
  const bySlot = new Map<number, OwnedLife[]>()
  for (const p of players) {
    for (const life of p.lives) {
      const w = trackWindow(life)
      const entry: OwnedLife = { start: w.start, end: w.end, player: p }
      const lives = bySlot.get(life.slot)
      if (lives) lives.push(entry)
      else bySlot.set(life.slot, [entry])
    }
  }
  for (const lives of bySlot.values()) lives.sort((a, b) => a.start - b.start)
  return {
    ownerAtFrame(slot, frame) {
      const lives = bySlot.get(slot)
      if (!lives) return null
      for (const l of lives) {
        // Triées par début : dès qu'une vie commence après l'image, aucune de la suite ne la couvre.
        if (frame < l.start) break
        if (frame <= l.end) return l.player
      }
      return null
    },
    ownerAtFrameOrLast(slot, frame) {
      const lives = bySlot.get(slot)
      if (!lives) return null
      let last: ReplayPlayer | null = null
      for (const l of lives) {
        if (frame < l.start) break // aucune vie couvrante ni précédente au-delà : on garde `last`
        if (frame <= l.end) return l.player // vie couvrante
        last = l.player // vie entièrement AVANT l'image : on retient la plus récente
      }
      return last // la vie juste précédente (le lâcheur / le tueur), ou null si aucune
    },
  }
}

/**
 * colorResolver — LA COULEUR D'UNE VIE EST CELLE DE SON PROPRIÉTAIRE À CETTE IMAGE, et c'est sa
 * couleur d'ÉQUIPE (décision D1 du plan d'habillage, 2026-08-16, amendée par l'utilisateur) :
 * les tokens `team-ally` / `team-enemy` que les réglages d'accessibilité peuvent surcharger,
 * jamais une couleur de série indexée sur la trace. Un joueur gardait jusque-là une teinte par
 * VIE (il changeait à chaque réapparition) ; puis une teinte figée par SLOT, qui montrait un
 * seul joueur pour tout le match quand un slot est réattribué entre manches. La couleur est
 * désormais résolue À L'IMAGE, via le propriétaire de la vie qui occupe alors le slot.
 *
 * `colorOf` rend la couleur d'un camp (l'appelant résout les tokens), `isAlly` dit le camp d'un
 * xuid (côté du joueur de la page), `neutral` sert quand on ne sait rien de l'identité. `null`
 * = aucun propriétaire à cette image (slot libre, ou vie anonyme) : le calque ne dessine rien.
 */
export function colorResolver(
  ownership: SlotOwnership,
  colorOf: (ally: boolean) => string,
  isAlly: (xuid: string) => boolean,
  neutral: string,
): (slot: number, frame: number) => string | null {
  return (slot, frame) => teamColorOfOwner(ownership.ownerAtFrame(slot, frame), colorOf, isAlly, neutral)
}

/**
 * colorResolverOrLast — MÊME couleur d'équipe, mais résolue via `ownerAtFrameOrLast` : pour les
 * consommateurs de FRONTIÈRE (couleur d'un objet LÂCHÉ à la mort, couleur d'un effet de mort)
 * dont l'événement est daté à l'instant où le propriétaire vient de quitter le slot. Un objet
 * lâché porte `t0 = finVie + 1` : la résolution stricte y rendrait `null` → encre neutre au lieu
 * de la couleur du lâcheur. NE PAS l'employer pour les marqueurs/vies (rendu continu).
 */
export function colorResolverOrLast(
  ownership: SlotOwnership,
  colorOf: (ally: boolean) => string,
  isAlly: (xuid: string) => boolean,
  neutral: string,
): (slot: number, frame: number) => string | null {
  return (slot, frame) => teamColorOfOwner(ownership.ownerAtFrameOrLast(slot, frame), colorOf, isAlly, neutral)
}

/** teamColorOfOwner — la couleur d'équipe d'un propriétaire (neutre pour une entrée sans xuid, null s'il n'y en a pas). */
function teamColorOfOwner(
  p: ReplayPlayer | null,
  colorOf: (ally: boolean) => string,
  isAlly: (xuid: string) => boolean,
  neutral: string,
): string | null {
  if (!p) return null
  return p.xuid ? colorOf(isAlly(p.xuid)) : neutral
}

/**
 * sideResolver — LE CAMP D'UNE VIE À UNE IMAGE, celui de son propriétaire à cette image
 * (`team_side`, comme `groupByTeam`).
 *
 * PAS LE DRAPEAU « allié », qui est relatif au joueur de la page et range tous les autres dans
 * un seul camp — faux dès qu'il y a plus de deux équipes (mêlée générale, BTB à quatre camps).
 * Ce que le capteur de menaces doit savoir, c'est si DEUX vies s'opposent ; `team_side` le dit.
 *
 * Sans propriétaire à cette image (slot libre) ou sans ligne de scoreboard, une vie n'a PAS de
 * camp (null) : ni alliée ni ennemie de personne, et rien ne l'affirmera à sa place. Chaîne
 * vide = absence (le DTO l'écrit pour un camp non résolu).
 */
export function sideResolver(
  ownership: SlotOwnership,
): (slot: number, frame: number) => string | null {
  return (slot, frame) => ownership.ownerAtFrame(slot, frame)?.board?.team_side || null
}

/**
 * markResolver rend la marque d'identité (« moi », « ami ») du propriétaire de la vie qui
 * occupe le slot à cette image ; `undefined` s'il n'y a pas de propriétaire ou pas de marque.
 */
export function markResolver(
  ownership: SlotOwnership,
  marks: ReadonlyMap<string, PlayerMarkKind>,
): (slot: number, frame: number) => PlayerMarkKind | undefined {
  return (slot, frame) => {
    const p = ownership.ownerAtFrame(slot, frame)
    return p ? marks.get(p.xuid) : undefined
  }
}

/**
 * nameResolver rend le NOM D'AFFICHAGE du propriétaire de la vie qui occupe le slot à cette
 * image — celui des fiches et du fil (`displayPlayerName`), jamais un xuid brut. Aucun
 * propriétaire à cette image → `null` : l'étiquette de la carte reste alors vide plutôt que
 * d'écrire « Joueur #### » ou, pire, le nom d'un joueur d'une AUTRE manche.
 */
export function nameResolver(
  ownership: SlotOwnership,
): (slot: number, frame: number) => string | null {
  return (slot, frame) => {
    const p = ownership.ownerAtFrame(slot, frame)
    return p ? displayPlayerName(playerName(p), p.xuid) : null
  }
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
