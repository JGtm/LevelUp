/**
 * playerPanelLogic.ts — LE PANNEAU DE JOUEUR : qui est choisi, qui est suivi, qui est masqué, et ce que cela fait à
 * la carte. Logique pure, sans React ni canvas (playerPanelLogic.test.ts).
 *
 * LE GESTE. Un clic sur une fiche de la colonne des équipes ouvre la fiche du joueur (une fenêtre au Cinéma et au
 * Plein cadre, la colonne entière au Standard) ; fermer la fiche la referme. Le panneau garde son joueur quand le
 * temps avance et change de contenu quand on clique une autre fiche. Rien de cela n'est mémorisé (D-4) : l'état
 * naît avec la page, il est tenu par elle, et repart de zéro avec le match.
 *
 * TROIS ÉTATS INDÉPENDANTS.
 *  - LA FICHE (`open`, `xuid`) : le joueur dont la fiche est ouverte ;
 *  - SUIVRE (`followXuid`) : la caméra cadre la carte sur ce joueur à chaque image — qui, ou personne. Il
 *    s'éteint par le geste manuel de cadrage (`stopFollow`), par un nouveau choix, ou quand le joueur suivi est
 *    masqué ;
 *  - L'ENSEMBLE DES JOUEURS MASQUÉS (`hidden`, D-7) : les clés de roster des joueurs retirés de la carte (pion,
 *    traînée, nom, et tout ce qui leur appartient). Il ne dépend PAS de la fiche : ouvrir, changer ou fermer la
 *    fiche ne le touche pas. Il dure jusqu'à « Tout afficher » ou jusqu'au changement de match. Un joueur se
 *    masque ou se réaffiche par son œil (`toggleHidden`) ; « Seul » (`solo`) masque tous les autres joueurs du
 *    roster, et refait sur un état déjà « seul » il réaffiche tout (D-8). Les fiches et les lignes, elles, ne sont
 *    jamais filtrées : un joueur masqué reste dans la colonne (son œil le réaffiche).
 *
 * LES RÈGLES ENTRE SUIVRE ET MASQUER (D-15). Suivre un joueur masqué le réaffiche ; masquer le joueur suivi arrête
 * le suivi ; « Seul » arrête le suivi d'un joueur qu'il masque (la carte ne se cadre pas sur un joueur qu'elle ne
 * dessine pas).
 *
 * LE JOUEUR SUIVI (`trackedPlayerOf`) n'est pas dans cet état : c'est le joueur REGARDÉ par la page, celui dont la
 * page prend le point de vue (le joueur du match au départ, modifiable). Le modèle le lit, il n'en tient pas un second.
 */
import { isAliveAt } from '@/lib/replay/replayLogic'
import type { KillFxEntry } from './killFx'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

/** L'ensemble vide partagé : « personne n'est masqué ». Jamais modifié sur place. */
const NO_HIDDEN: ReadonlySet<string> = new Set()

/** L'état du panneau : la fiche, la caméra Suivre, et l'ensemble des joueurs masqués. */
export interface PlayerPanelState {
  open: boolean
  /** Clé de roster du joueur choisi (`ReplayPlayer.xuid` : le xuid, ou `bot:<nom>` pour un bot). */
  xuid: string | null
  /** Clé de roster du joueur que la caméra suit ; `null` = personne. Indépendant de `open` et de `xuid`. */
  followXuid: string | null
  /** Les clés de roster des joueurs masqués. Indépendant de la fiche. */
  hidden: ReadonlySet<string>
}

export const NO_PLAYER_PANEL: PlayerPanelState = {
  open: false, xuid: null, followXuid: null, hidden: NO_HIDDEN,
}

export type PlayerPanelEvent =
  /** Un clic sur la fiche d'un joueur : ouvre, remplace le joueur, ou referme si elle est déjà ouverte. */
  | { type: 'card'; xuid: string }
  /** Fermer la fiche (la croix de la fenêtre, « Équipes » de la colonne, Échap). */
  | { type: 'close' }
  /** Suivre ce joueur (clé de roster), fiche ouverte ou non ; `null` = ne suivre personne. */
  | { type: 'follow'; xuid: string | null }
  /** Un geste manuel de zoom ou de déplacement sur la carte : Suivre s'arrête. */
  | { type: 'stopFollow' }
  /** L'œil d'un joueur : le masque, ou le réaffiche s'il est masqué. */
  | { type: 'toggleHidden'; xuid: string }
  /** « Seul » sur ce joueur : masque tous les autres joueurs du roster, ou réaffiche tout s'il est déjà seul. */
  | { type: 'solo'; xuid: string; roster: readonly string[] }
  /** « Tout afficher » : plus aucun joueur masqué. */
  | { type: 'showAll' }

/**
 * isSolo dit si ce joueur est « seul » : affiché, quand tous les autres joueurs du roster sont masqués. C'est une
 * lecture de l'ensemble, pas un état : réafficher un joueur à la main sort de « Seul ». Un roster d'un seul joueur
 * n'a pas de « Seul ».
 */
export function isSolo(hidden: ReadonlySet<string>, xuid: string, roster: readonly string[]): boolean {
  if (hidden.has(xuid)) return false
  const others = roster.filter((key) => key !== xuid)
  return others.length > 0 && others.every((key) => hidden.has(key))
}

/** La fiche refermée : la caméra et les joueurs masqués restent. */
function withoutCard(state: PlayerPanelState): PlayerPanelState {
  return { ...NO_PLAYER_PANEL, followXuid: state.followXuid, hidden: state.hidden }
}

/**
 * reducePlayerPanel applique un événement à l'état. Un événement sans effet rend le MÊME objet, pour ne réveiller
 * personne (un geste manuel de plus pendant que Suivre est déjà éteint).
 */
export function reducePlayerPanel(state: PlayerPanelState, event: PlayerPanelEvent): PlayerPanelState {
  switch (event.type) {
    case 'card':
      if (state.open && state.xuid === event.xuid) return withoutCard(state)
      return { ...withoutCard(state), open: true, xuid: event.xuid }
    case 'close':
      return state.open ? withoutCard(state) : state
    case 'follow':
      return followPlayer(state, event.xuid)
    case 'stopFollow':
      return followPlayer(state, null)
    case 'toggleHidden':
      return toggleHidden(state, event.xuid)
    case 'solo':
      return soloPlayer(state, event.xuid, event.roster)
    case 'showAll':
      return state.hidden.size === 0 ? state : { ...state, hidden: NO_HIDDEN }
  }
}

/** Suivre un joueur (ou personne) ; suivre un joueur masqué le réaffiche (D-15). */
function followPlayer(state: PlayerPanelState, xuid: string | null): PlayerPanelState {
  if (state.followXuid === xuid) return state
  if (xuid === null || !state.hidden.has(xuid)) return { ...state, followXuid: xuid }
  return { ...state, followXuid: xuid, hidden: without(state.hidden, xuid) }
}

/** L'œil d'un joueur ; masquer le joueur suivi arrête le suivi (D-15). */
function toggleHidden(state: PlayerPanelState, xuid: string): PlayerPanelState {
  if (state.hidden.has(xuid)) return { ...state, hidden: without(state.hidden, xuid) }
  return {
    ...state,
    hidden: new Set(state.hidden).add(xuid),
    followXuid: state.followXuid === xuid ? null : state.followXuid,
  }
}

/**
 * « Seul » : tous les autres joueurs du roster sont masqués et le joueur choisi est affiché ; refait sur un joueur
 * déjà « seul », il réaffiche tout. Le suivi d'un joueur que « Seul » masque s'arrête (D-15).
 */
function soloPlayer(state: PlayerPanelState, xuid: string, roster: readonly string[]): PlayerPanelState {
  if (isSolo(state.hidden, xuid, roster)) return { ...state, hidden: NO_HIDDEN }
  const hidden = new Set(roster.filter((key) => key !== xuid))
  const stillFollowed = state.followXuid !== null && hidden.has(state.followXuid) ? null : state.followXuid
  return { ...state, hidden, followXuid: stillFollowed }
}

/** L'ensemble sans cette clé : un nouvel ensemble, l'ancien n'est jamais modifié. */
function without(set: ReadonlySet<string>, key: string): ReadonlySet<string> {
  const next = new Set(set)
  next.delete(key)
  return next
}

/** Le joueur dont le panneau est ouvert, avec son état d'affichage : ce que la page donne à la fiche et à l'anneau. */
export interface PlayerFocus {
  player: ReplayPlayer
  /** Vrai quand ce joueur est « seul » : tous les autres joueurs sont masqués (l'état du bouton « Seul » de la fiche). */
  solo: boolean
  /** Vrai quand ce joueur est masqué : son pion n'est pas à la carte, l'anneau de la fiche ne le désigne pas. */
  hidden: boolean
}

/**
 * panelFocusOf retrouve le joueur du panneau ouvert dans le roster. `null` : panneau fermé, ou joueur absent du
 * roster (un autre document) — jamais un joueur deviné.
 */
export function panelFocusOf(
  state: PlayerPanelState,
  players: readonly ReplayPlayer[],
): PlayerFocus | null {
  if (!state.open || state.xuid === null) return null
  const xuid = state.xuid
  const player = players.find((p) => p.xuid === xuid)
  if (!player) return null
  return {
    player,
    solo: isSolo(state.hidden, xuid, players.map((p) => p.xuid)),
    hidden: state.hidden.has(xuid),
  }
}

/**
 * Les joueurs masqués, retrouvés au roster : ce que les calques lisent pour savoir quoi retirer. `null` quand
 * personne n'est masqué — les calques n'ont alors rien à brancher et gardent leur dessin d'avant. Une clé qui n'est
 * plus au roster (un autre document) est ignorée, jamais un joueur deviné.
 */
export type HiddenPlayers = readonly ReplayPlayer[]

export function hiddenPlayersOf(
  hidden: ReadonlySet<string>,
  players: readonly ReplayPlayer[],
): HiddenPlayers | null {
  if (hidden.size === 0) return null
  const found = players.filter((p) => hidden.has(p.xuid))
  return found.length > 0 ? found : null
}

/**
 * followedPlayerOf retrouve au roster le joueur que la CAMÉRA suit (`followXuid`), fiche ouverte ou non.
 * `null` : personne n'est suivi, ou le joueur n'est plus au roster (un autre document) — jamais un joueur
 * deviné.
 */
export function followedPlayerOf(
  state: PlayerPanelState,
  players: readonly ReplayPlayer[],
): ReplayPlayer | null {
  if (state.followXuid === null) return null
  return players.find((p) => p.xuid === state.followXuid) ?? null
}

/**
 * trackedPlayerOf lit le JOUEUR SUIVI : le joueur regardé par la page, retrouvé dans le roster.
 * `watchedXuid` est la valeur du point de vue de la page (le joueur du match au départ, puis le
 * choix de l'utilisateur) : la ligne de feuille d'abord — c'est la clé que le menu de point de vue
 * rend, y compris pour un bot —, la clé de roster à défaut. `null` : personne n'est regardé, ou le
 * joueur regardé n'est pas au roster — jamais un joueur deviné.
 */
export function trackedPlayerOf(
  watchedXuid: string | null,
  players: readonly ReplayPlayer[],
): ReplayPlayer | null {
  if (!watchedXuid) return null
  return (
    players.find((p) => p.board?.xuid === watchedXuid) ??
    players.find((p) => p.xuid === watchedXuid) ??
    null
  )
}

/**
 * ownsLifeAt dit si l'un de ces joueurs occupe CE slot à CETTE image : une de ses vies porte le slot et couvre
 * l'image. UN SLOT N'EST PAS UN JOUEUR : le slot de bipède est réattribué à chaque réapparition, et un même slot
 * appartient à des joueurs différents au fil du match — d'où la vie, jamais le seul slot, et c'est ce qui rend la
 * lecture juste pour un bot, qui n'a pas de xuid mais a des vies. Les vies sont rangées par slot une fois.
 */
function ownsLifeAt(players: HiddenPlayers): (slot: number, frame: number) => boolean {
  const bySlot = new Map<number, ReplayPlayer['lives']>()
  for (const p of players) {
    for (const life of p.lives) {
      const lives = bySlot.get(life.slot)
      if (lives) lives.push(life)
      else bySlot.set(life.slot, [life])
    }
  }
  return (slot, frame) => bySlot.get(slot)?.some((life) => isAliveAt(life, frame)) ?? false
}

/**
 * shownSlotPredicate dit si ce que porte un slot à une image se dessine : un pion se dessine si son joueur n'est
 * pas masqué. `null` quand personne n'est masqué : les calques n'ont alors rien à brancher.
 *
 * Il désigne les VIES des joueurs masqués (cf. `ownsLifeAt`). Il sert aux éléments qui portent un slot et une
 * image DANS la vie de leur auteur : pion, nom, cône de visée, traînée, tirs, marques de tir, poussée du
 * propulseur, projectiles et grenades, traces des vies, mise en valeur. Tout le reste — effets de mort, poses,
 * véhicules, objets portés, armes au sol, chaleur — passe par `ownershipShown`, qui lit les mêmes vies.
 */
export function shownSlotPredicate(
  hidden: HiddenPlayers | null,
): ((slot: number, frame: number) => boolean) | null {
  if (!hidden) return null
  const owned = ownsLifeAt(hidden)
  return (slot, frame) => !owned(slot, frame)
}

/**
 * Une PARTIE d'un élément du rejeu : ce qui le rattache à un joueur. `life` est un slot et une image
 * DANS la vie du joueur (celle que `ownsLifeAt` lit) ; `xuid` est la clé du document (porteur, auteur
 * d'une action, occupant). Une partie qui porte les deux est prouvée par l'un OU l'autre : le pont
 * slot -> joueur peut manquer quand le document nomme pourtant l'occupant.
 */
export interface ElementParty {
  life?: { slot: number; frame: number }
  xuid?: string
}

/**
 * Les éléments à parties : `parties` vide = l'élément n'appartient à personne, ou son attribution ne se lit pas :
 * il se dessine toujours (jamais d'attribution devinée).
 */
export type OwnershipShown = (parties: readonly ElementParty[]) => boolean

/**
 * ownershipShown est LE prédicat des joueurs masqués pour tout élément qui appartient à un ou plusieurs joueurs,
 * sur tous les calques. `null` quand personne n'est masqué.
 *
 * LA RÈGLE (DU-3). Un élément appartient à CHAQUE joueur qui y figure, et il se dessine DÈS QU'UN de ses joueurs
 * est affiché. Un frag (effet de mort, mort comptée par la chaleur des éliminations), une poussée, une révélation
 * de capteur et une destruction de véhicule appartiennent à leur auteur ET à leur victime : masquer l'un des deux
 * laisse l'effet visible tant que l'autre est affiché, et ne le retire que lorsque TOUS ses joueurs sont masqués
 * (ce que « Seul » produit pour tout ce qui ne concerne pas le joueur seul). Les occupants d'un véhicule
 * le partagent : il se dessine tant que l'un d'eux est affiché. Seule exception, la croix de mort de la piste : elle
 * ne suit que la victime, la piste ne portant pas le tueur. Ce qui n'appartient à personne reste affiché, comme le
 * décor ; il en va de même de tout élément dont le propriétaire est inconnu (aucune partie lue).
 *
 * Chaque calque ne fait que dire QUELLES sont ses parties (`model/focusFilter.ts`) : la règle n'est écrite qu'ici.
 */
export function ownershipShown(hidden: HiddenPlayers | null): OwnershipShown | null {
  if (!hidden) return null
  const owned = ownsLifeAt(hidden)
  // Les xuid du document qui désignent un joueur masqué : le sien, et celui de sa ligne de feuille — le xuid du FIL
  // d'un bot est celui de sa ligne de feuille (sa clé de film est `bot:<nom>`).
  const hiddenXuids = new Set<string>()
  for (const player of hidden) {
    hiddenXuids.add(player.xuid)
    if (player.board?.xuid) hiddenXuids.add(player.board.xuid)
  }
  const isHidden = (p: ElementParty): boolean =>
    (p.life !== undefined && owned(p.life.slot, p.life.frame)) || (p.xuid !== undefined && hiddenXuids.has(p.xuid))
  return (parties) => parties.length === 0 || parties.some((p) => !isHidden(p))
}

/**
 * Les parties d'un effet de mort : le TUEUR et la VICTIME (un effet appartient aux deux). Une partie
 * sans vie retrouvée ni clé de fil ne prouve rien : l'autre partie décide, et sans aucune partie lue
 * l'effet se dessine.
 */
export function killFxParties(
  e: Pick<KillFxEntry, 'slot' | 'slotFrame' | 'killerXuid' | 'victimSlot' | 'victimFrame' | 'victimXuid'>,
): ElementParty[] {
  const out: ElementParty[] = []
  const killer = partyOf(e.slot === null ? undefined : { slot: e.slot, frame: e.slotFrame }, e.killerXuid)
  if (killer) out.push(killer)
  const victim = partyOf(
    e.victimSlot === null ? undefined : { slot: e.victimSlot, frame: e.victimFrame },
    e.victimXuid ?? undefined,
  )
  if (victim) out.push(victim)
  return out
}

/** Une partie, ou `null` quand ni la vie ni la clé ne se lisent. */
export function partyOf(life: ElementParty['life'], xuid: string | undefined): ElementParty | null {
  const key = xuid || undefined
  if (!life && !key) return null
  return { ...(life ? { life } : {}), ...(key ? { xuid: key } : {}) }
}

/**
 * killFxShown rend, pour le calque des morts, le prédicat qui dit si un effet se dessine : `null` sans joueur
 * masqué. L'effet appartient au tueur et à la victime : il se dessine dès que l'un des deux est affiché (DU-3).
 */
export function killFxShown(
  hidden: HiddenPlayers | null,
): ((e: Parameters<typeof killFxParties>[0]) => boolean) | null {
  const shown = ownershipShown(hidden)
  return shown ? (e) => shown(killFxParties(e)) : null
}
