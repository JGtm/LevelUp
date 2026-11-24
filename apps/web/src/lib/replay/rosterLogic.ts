/**
 * rosterLogic.ts — LA JOINTURE ENTRE LE FILM ET LA BASE, et l'état d'un joueur à une image.
 *
 * DEUX SOURCES, DEUX RÔLES, ET UNE SEULE CLÉ. Le film porte ce qui se passe — positions,
 * vies, morts, armes, bouclier, et l'ÉQUIPE de chaque joueur (`roster[].team`, la seule
 * source d'appartenance ET d'allégeance de la page, cf. `ReplayPlayer.team` et
 * `filmAllegiance.ts`) — et l'identifie par XUID. La base porte qui sont les gens : gamertag,
 * côté de feuille (qui NOMME un camp), K/D/A du match. Aucune des deux ne sait faire le travail
 * de l'autre, et l'artefact de rejeu n'essaie pas : il publie le xuid, qui est la seule clé sur
 * laquelle une jointure ne suppose rien.
 *
 * POURQUOI PAS UN INDEX. Ce chantier a déjà publié un « index de joueur » comme une découverte
 * avant de constater que c'était son propre tri alphabétique. Un index est un ORDRE ; le xuid
 * est une IDENTITÉ. La règle qui en est sortie tient en une ligne et vaut ici : on ne joint
 * jamais sur un rang.
 *
 * Tout ce fichier est PUR : aucun React, aucun canvas, donc testable.
 */
import type { MatchScoreboardRow, ReplayRosterEntry } from '@/lib/api/types'
import { displayPlayerName, stripBotSuffix } from '@/lib/players/displayName'

import type { PlayerMarkKind } from './playerMarks'
import { groupByCamp, type ReplayCamp } from './replayCamps'
import { trackWindow } from './replayLogic'
import type { ReplayDocumentReady, ReplayTrackReady } from './replayNormalize'

/**
 * ReplayPlayer — un joueur du rejeu : son identité (base) et ses vies (film).
 *
 * `board` est absent quand le xuid du film n'a aucune ligne au scoreboard. Ce n'est pas censé
 * arriver, et c'est précisément pour cela qu'on le représente au lieu de l'écraser : un joueur
 * du film introuvable dans la base est un signal, pas un détail à masquer.
 */
export interface ReplayPlayer {
  /**
   * Identité du joueur : le xuid en décimal, ou — pour un BOT, qui n'en a pas — la clé
   * synthétique `bot:<nom>` (schéma 36). La clé ne sort jamais à l'écran : elle sert les
   * Map de ce fichier et des résolveurs, le nom affiché vient de `filmName`/`board`.
   */
  xuid: string
  /** Vrai pour un bot déclaré par le film (roster/tracks `bot`). */
  bot?: boolean
  /** Ligne du scoreboard du match, quand la jointure aboutit. */
  board?: MatchScoreboardRow
  /** Gamertag écrit par le FILM. Sert quand la base n'a rien à dire sur ce joueur. */
  filmName?: string
  /**
   * L'ÉQUIPE DU JOUEUR : le désignateur que le FILM écrit pour son entrée de roster
   * (`roster[].team`, schéma 57 ; `-1` = « aucune équipe » d'un mode sans camps, une lecture).
   * C'est la SEULE source d'appartenance de la page Rejeu (ADR 0034 D-9) — tous ses camps en
   * sortent (`replayCamps.ts`), et toute son encre allié / adverse (`filmAllegiance.ts`). Absente
   * = le film la tait, ou le joueur n'a pas d'entrée : il n'entre dans aucun camp, n'a pas
   * d'encre de camp, et la feuille de match ne la remplace pas.
   */
  team?: number
  /** Toutes les vies de ce joueur, dans l'ordre du temps. */
  lives: ReplayTrackReady[]
}

/** Clé d'identité d'un bot dans les tables de ce fichier — jamais affichée. */
export function botKey(name: string): string {
  return `bot:${name}`
}

/**
 * rosterEntryKey — la clé d'identité d'une entrée de roster, DANS LE MÊME ESPACE que
 * `ReplayPlayer.xuid` construit par `buildPlayers` ci-dessous : le xuid quand il existe,
 * sinon `botKey(name)` pour un bot (`RosterEntry.xuid` vaut `''` pour un bot, jamais un xuid
 * réel — schéma 36). Un appelant qui compare `entry.xuid` directement à `player.xuid` ne fait
 * jamais matcher un bot (constat P2-5, audit vies anonymes 2026-09-06 : `equipmentUsageLogic`
 * et `playerCardReadings` en portaient chacun une occurrence). Centralisée ici pour que les
 * deux jointures roster -> joueur du dépôt qui en avaient besoin partagent une seule
 * dérivation (règle des ≤ 2 copies, CLAUDE.md n°6). `seatLogic.filmIndexByIdentity` portait une
 * 3e copie inline (correcte, mais non centralisée) : migrée le 2026-09-10 — garde-rail grep :
 * `apps/web/src/lib/replay/rosterEntryKey_no_new_copies.guard.test.ts`. `buildPlayers` la
 * consomme aussi : c'est par elle que l'équipe d'une entrée rejoint son joueur.
 */
export function rosterEntryKey(entry: Pick<ReplayRosterEntry, 'xuid' | 'bot' | 'name'>): string {
  return entry.xuid || (entry.bot && entry.name ? botKey(entry.name) : '')
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
 * ReplayTeamGroup — un camp DU FILM et ses joueurs (`replayCamps.ts` : le désignateur décide,
 * le côté de feuille de ses membres ne fait que le nommer).
 */
export interface ReplayTeamGroup extends ReplayCamp {
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
  // UN BOT N'A PAS DE XUID (schéma 36) : son identité de table est `bot:<nom>` — deux bots
  // homonymes fusionneraient, et c'est assumé : le film ne donne rien de plus discriminant.
  // L'ÉQUIPE EST POSÉE ICI ET NULLE PART AILLEURS (cf. `ReplayPlayer.team`) : le désignateur de
  // l'entrée, jointe par la même clé que partout (`rosterEntryKey`). Un joueur que seules ses
  // vies nomment n'a pas d'entrée, donc pas d'équipe.
  for (const entry of doc.roster ?? []) {
    const key = rosterEntryKey(entry)
    if (!key) continue
    byXUID.set(key, {
      xuid: key,
      bot: entry.bot || undefined,
      filmName: entry.name,
      team: entry.team ?? undefined,
      lives: [],
    })
  }
  for (const track of doc.tracks) {
    const key = track.xuid || (track.bot ? botKey(track.bot) : '')
    if (!key) continue
    let p = byXUID.get(key)
    if (!p) {
      p = { xuid: key, bot: track.bot ? true : undefined, filmName: track.bot, lives: [] }
      byXUID.set(key, p)
    }
    p.lives.push(track)
  }
  const board = new Map(scoreboard.map((r) => [r.xuid, r]))
  // LA JOINTURE D'UN BOT SE FAIT PAR GAMERTAG, faute de xuid commun : la base identifie ses
  // bots par `is_bot` (xuid `bid(N.0)`, gamertag résolu SANS suffixe), le film par le nom
  // suffixé « [bot] » — la clé de jointure est le nom NU des deux côtés. Un nom absent du
  // scoreboard laisse `board` vide — le bot reste sans ligne de feuille, jamais une ligne
  // devinée ; son équipe, elle, est celle que le film écrit (`team`, posée plus haut).
  const boardByName = new Map(
    scoreboard.filter((r) => r.is_bot).map((r) => [stripBotSuffix(r.gamertag), r]),
  )
  for (const p of byXUID.values()) {
    p.board = p.bot ? boardByName.get(stripBotSuffix(p.filmName ?? '')) : board.get(p.xuid)
    p.lives.sort((a, b) => trackWindow(a).start - trackWindow(b).start)
  }
  return [...byXUID.values()]
}

/**
 * groupByTeam range les joueurs par camp DU FILM (`ReplayPlayer.team`), camps dans l'ordre des
 * désignateurs, joueurs dans l'ordre du roster.
 *
 * UN JOUEUR DONT LE FILM TAIT L'ÉQUIPE N'ENTRE DANS AUCUN CAMP — il n'y a pas de groupe « à
 * part » : c'est un défaut de source (cf. `replayCamps.ts`). Un joueur SANS LIGNE de feuille,
 * lui, est dans le camp que le film lui donne ; la feuille ne fait que nommer ce camp.
 */
export function groupByTeam(players: readonly ReplayPlayer[]): ReplayTeamGroup[] {
  return groupByCamp(players, (p) => p.team, (p) => [p.board]).map(({ team, side, members }) => ({
    team,
    side,
    players: members,
  }))
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
   * de LA manche concernée (SHROOM s'il vient de mourir, J900001 sinon). NE PAS l'employer pour
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
 * `colorOf` rend la couleur d'un camp (l'appelant résout les tokens), `allyOf` dit l'allégeance
 * d'un JOUEUR vue de la référence (`FilmAllegiance.ofPlayer` : l'équipe du FILM, jamais la
 * feuille), `neutral` sert quand elle est inconnue (`null` : le film tait l'équipe du joueur, ou
 * celle de la référence). `null` en sortie = aucun propriétaire à cette image (slot libre, ou vie
 * anonyme) : le calque ne dessine rien.
 */
export function colorResolver(
  ownership: SlotOwnership,
  colorOf: (ally: boolean) => string,
  allyOf: (player: ReplayPlayer) => boolean | null,
  neutral: string,
): (slot: number, frame: number) => string | null {
  return (slot, frame) => teamColorOfOwner(ownership.ownerAtFrame(slot, frame), colorOf, allyOf, neutral)
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
  allyOf: (player: ReplayPlayer) => boolean | null,
  neutral: string,
): (slot: number, frame: number) => string | null {
  return (slot, frame) => teamColorOfOwner(ownership.ownerAtFrameOrLast(slot, frame), colorOf, allyOf, neutral)
}

/** teamColorOfOwner — la couleur d'équipe d'un propriétaire (neutre sans allégeance, null s'il n'y en a pas). */
function teamColorOfOwner(
  p: ReplayPlayer | null,
  colorOf: (ally: boolean) => string,
  allyOf: (player: ReplayPlayer) => boolean | null,
  neutral: string,
): string | null {
  if (!p) return null
  const ally = allyOf(p)
  return ally === null ? neutral : colorOf(ally)
}

/**
 * campResolver — LE CAMP D'UNE VIE À UNE IMAGE, celui de son propriétaire à cette image : son
 * équipe du FILM (`ReplayPlayer.team`). Il sert l'OPPOSITION entre deux vies (capteur de menaces,
 * zones d'une fiche, poses), pas l'encre : celle-là est relative à la référence
 * (`filmAllegiance.ts`).
 *
 * PAS LE DRAPEAU « allié », qui est relatif au joueur regardé et range tous les autres dans un
 * seul camp — faux dès qu'il y a plus de deux équipes (BTB à quatre camps). Ce que le capteur
 * doit savoir, c'est si DEUX vies s'opposent ; leurs désignateurs le disent.
 *
 * Sans propriétaire à cette image (slot libre), ou quand le film tait son équipe ou n'en donne
 * aucune (`-1`, mode sans camps), une vie n'a PAS de camp (null) : ni alliée ni ennemie de
 * personne, et la feuille ne l'affirmera pas à sa place.
 */
export function campResolver(
  ownership: SlotOwnership,
): (slot: number, frame: number) => number | null {
  return (slot, frame) => {
    const team = ownership.ownerAtFrame(slot, frame)?.team
    return team !== undefined && team >= 0 ? team : null
  }
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
 * nameByXuidResolver — LE MÊME NOM D'AFFICHAGE, mais demandé PAR XUID et sans image.
 *
 * POURQUOI IL EXISTE (2026-09-02, calque des véhicules). `nameResolver` répond à « qui occupe CE
 * SLOT à CETTE IMAGE » — la seule question juste pour un pion, dont l'identité n'existe que par
 * la vie qui occupe le slot. Mais un ÉPISODE D'OCCUPATION de véhicule (`VehicleRide`) nomme son
 * occupant LUI-MÊME, par xuid, et cette affirmation-là ne dépend d'aucune trace de bipède : elle
 * survit au cas — observé sur film — où la trace du slot n'a pas été jointe à un joueur
 * (`buildPlayers` l'écarte alors, cf. `if (!track.xuid) continue`) et où le pont slot->joueur ne
 * peut donc rien rendre.
 *
 * XUID INCONNU DU ROSTER -> `null` : on ne fabrique pas un nom à partir d'un identifiant qu'aucun
 * joueur du match ne porte.
 */
export function nameByXuidResolver(
  players: readonly ReplayPlayer[],
): (xuid: string) => string | null {
  const byXuid = new Map(players.map((p) => [p.xuid, p]))
  return (xuid) => {
    const p = byXuid.get(xuid)
    return p ? displayPlayerName(playerName(p), p.xuid) : null
  }
}

/**
 * colorByXuidResolver — LA MÊME COULEUR D'ÉQUIPE, mais demandée PAR XUID et sans image.
 *
 * POURQUOI IL EXISTE (2026-09-02, teinte des véhicules). `colorResolver` répond à « de quelle
 * couleur est la vie qui occupe CE SLOT à CETTE IMAGE » — la seule question juste pour un pion.
 * Mais un ÉPISODE D'OCCUPATION de véhicule nomme son occupant LUI-MÊME (`VehicleRide.xuid`), et
 * pendant l'épisode le bipède ne réplique plus : le pont slot->joueur est alors MUET là où il
 * faudrait qu'il parle — c'est exactement le défaut que le contrat de `VehicleRide.xuid`
 * annonçait pourtant résolu (« c'est lui qui donne sa couleur au véhicule »). Résoudre par xuid
 * tient cette promesse ; le pont par slot reste le REPLI, comme pour le nom.
 *
 * XUID INCONNU DU ROSTER -> `null` : aucun joueur du match ne le porte, on ne lui invente pas
 * d'équipe. Xuid connu dont l'allégeance est inconnue (le film tait son équipe, ou celle de la
 * référence) -> encre neutre (même règle que le pont par slot : le joueur existe, son camp non).
 */
export function colorByXuidResolver(
  players: readonly ReplayPlayer[],
  colorOf: (ally: boolean) => string,
  allyOf: (player: ReplayPlayer) => boolean | null,
  neutral: string,
): (xuid: string) => string | null {
  const byXuid = new Map(players.map((p) => [p.xuid, p]))
  return (xuid) => teamColorOfOwner(byXuid.get(xuid) ?? null, colorOf, allyOf, neutral)
}
