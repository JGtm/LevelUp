/**
 * playerPanelLogic.ts — LE PANNEAU DE JOUEUR : qui est choisi, ce que ses trois actions font à la
 * carte, et ce qu'il affiche. Logique pure, sans React ni canvas (playerPanelLogic.test.ts).
 *
 * LE GESTE. Un clic sur une fiche de la colonne des équipes ouvre la fiche du joueur (une
 * fenêtre au Cinéma et au Plein cadre, la colonne entière au Standard) ; fermer la fiche la referme. Le panneau garde son joueur quand le temps avance
 * et change de contenu quand on clique une autre fiche. Rien de cela n'est mémorisé (D-4) : l'état
 * naît avec la page, il est tenu par elle.
 *
 * TROIS ACTIONS. Suivre cadre la carte sur le joueur à chaque image, Seul n'affiche que lui,
 * Masquer le retire. Exclusions : Seul coupe Masquer ; Suivre coupe Masquer ; Masquer coupe
 * Seul ET Suivre de ce même joueur ; Seul coupe le Suivre d'un AUTRE joueur (la carte ne se cadre
 * pas sur un joueur qu'elle ne dessine pas) ; suivre un AUTRE joueur que celui de la fiche coupe
 * Seul.
 *
 * SEUL ET MASQUER tiennent à la fiche : ils ne valent que pour son joueur, ne se basculent que
 * fiche ouverte, et ouvrir une autre fiche comme refermer la fiche les remet à zéro — un joueur
 * masqué ne reste jamais masqué sans qu'on puisse le voir. SUIVRE est la caméra, un état à part
 * (`followXuid` : qui, ou personne) : il ne dépend pas de la fiche, que l'on ferme ou change sans
 * l'éteindre, et `follow` l'accepte fiche fermée. Il s'éteint par le geste manuel de cadrage
 * (`stopFollow`), par un nouveau choix, par Masquer sur le joueur suivi, ou par Seul sur un autre joueur. Les fiches, elles, ne
 * sont jamais filtrées : un joueur présent reste dans la colonne.
 *
 * LE JOUEUR SUIVI (`trackedPlayerOf`) n'est pas dans cet état : c'est le joueur REGARDÉ par la page,
 * celui dont la page prend le point de vue (le joueur du match au départ, modifiable). Le modèle le
 * lit, il n'en tient pas un second.
 */
import { isAliveAt } from '@/lib/replay/replayLogic'
import type { KillFxEntry } from './killFx'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

/** Les trois actions du panneau sur la carte. */
export type PlayerPanelAction = 'follow' | 'solo' | 'hide'

/**
 * L'état du panneau : ouvert ou non, sur quel joueur (clé de roster), ses deux actions d'affichage
 * (Seul, Masquer), et la caméra Suivre, qui n'est pas liée à la fiche.
 */
export interface PlayerPanelState {
  open: boolean
  /** Clé de roster du joueur choisi (`ReplayPlayer.xuid` : le xuid, ou `bot:<nom>` pour un bot). */
  xuid: string | null
  /** Clé de roster du joueur que la caméra suit ; `null` = personne. Indépendant de `open` et de `xuid`. */
  followXuid: string | null
  solo: boolean
  hide: boolean
}

export const NO_PLAYER_PANEL: PlayerPanelState = {
  open: false, xuid: null, followXuid: null, solo: false, hide: false,
}

export type PlayerPanelEvent =
  /** Un clic sur la fiche d'un joueur : ouvre, remplace le joueur, ou referme si elle est déjà ouverte. */
  | { type: 'card'; xuid: string }
  /** Fermer la fiche (la croix de la fenêtre, « Équipes » de la colonne, Échap). */
  | { type: 'close' }
  /** Un des trois boutons d'action de la fiche ouverte. */
  | { type: 'toggle'; action: PlayerPanelAction }
  /** Suivre ce joueur (clé de roster), fiche ouverte ou non ; `null` = ne suivre personne. */
  | { type: 'follow'; xuid: string | null }
  /** Un geste manuel de zoom ou de déplacement sur la carte : Suivre s'arrête. */
  | { type: 'stopFollow' }

/** La fiche refermée : Seul et Masquer retombent, la caméra garde son joueur. */
function withoutCard(state: PlayerPanelState): PlayerPanelState {
  return { ...NO_PLAYER_PANEL, followXuid: state.followXuid }
}

/**
 * reducePlayerPanel applique un événement à l'état. Un événement sans effet rend le MÊME objet,
 * pour ne réveiller personne (un geste manuel de plus pendant que Suivre est déjà éteint).
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
    case 'toggle':
      return state.open ? toggleAction(state, event.action) : state
  }
}

/**
 * Suivre un joueur (ou personne) ; suivre le joueur masqué lève son masquage, et suivre un autre joueur
 * que celui de la fiche lève Seul (la carte ne dessinerait que le joueur de la fiche, hors du cadre).
 */
function followPlayer(state: PlayerPanelState, xuid: string | null): PlayerPanelState {
  if (state.followXuid === xuid) return state
  const unhide = xuid !== null && state.hide && state.xuid === xuid
  const unsolo = xuid !== null && state.solo && state.xuid !== xuid
  return {
    ...state,
    followXuid: xuid,
    hide: unhide ? false : state.hide,
    solo: unsolo ? false : state.solo,
  }
}

/**
 * Bascule une action de la fiche ouverte en appliquant les exclusions (Seul coupe Masquer et le Suivre
 * d'un autre joueur, Suivre coupe Masquer, Masquer coupe Seul et le Suivre du joueur masqué).
 */
function toggleAction(state: PlayerPanelState, action: PlayerPanelAction): PlayerPanelState {
  if (action === 'follow') return followPlayer(state, state.followXuid === state.xuid ? null : state.xuid)
  if (state[action]) return { ...state, [action]: false }
  if (action === 'hide') {
    const stillFollowed = state.followXuid === state.xuid ? null : state.followXuid
    return { ...state, hide: true, solo: false, followXuid: stillFollowed }
  }
  // Seul garde le suivi du joueur de la fiche, et éteint celui d'un autre.
  const stillFollowed = state.followXuid === state.xuid ? state.followXuid : null
  return { ...state, solo: true, hide: false, followXuid: stillFollowed }
}

/** Le joueur dont le panneau est ouvert, avec ses actions : ce que la page donne au canvas. */
export interface PlayerFocus {
  player: ReplayPlayer
  /** Vrai quand la caméra suit CE joueur (l'état du bouton « Suivre » de la fiche ; la toile lit `followedPlayerOf`). */
  follow: boolean
  solo: boolean
  hide: boolean
}

/**
 * panelFocusOf retrouve le joueur du panneau ouvert dans le roster. `null` : panneau fermé, ou
 * joueur absent du roster (un autre document) — jamais un joueur deviné.
 */
export function panelFocusOf(
  state: PlayerPanelState,
  players: readonly ReplayPlayer[],
): PlayerFocus | null {
  if (!state.open || state.xuid === null) return null
  const player = players.find((p) => p.xuid === state.xuid)
  if (!player) return null
  return { player, follow: state.followXuid === state.xuid, solo: state.solo, hide: state.hide }
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
 * ownsLifeAt dit si le joueur occupe CE slot à CETTE image : une de ses vies porte le slot et couvre
 * l'image. UN SLOT N'EST PAS UN JOUEUR : le slot de bipède est réattribué à chaque réapparition, et un
 * même slot appartient à des joueurs différents au fil du match — d'où la vie, jamais le seul slot, et
 * c'est ce qui rend la lecture juste pour un bot, qui n'a pas de xuid mais a des vies.
 */
function ownsLifeAt(player: ReplayPlayer): (slot: number, frame: number) => boolean {
  const lives = player.lives
  return (slot, frame) => lives.some((l) => l.slot === slot && isAliveAt(l, frame))
}

/**
 * shownSlotPredicate dit si ce que porte un slot à une image se dessine, pour Seul et Masquer.
 * `null` quand rien n'est filtré (ni Seul ni Masquer) : les calques n'ont alors rien à brancher.
 *
 * Il désigne les VIES du joueur choisi (cf. `ownsLifeAt`). Il sert aux éléments qui portent un slot et
 * une image DANS la vie de leur auteur : pion, nom, cône de visée, traînée, tirs, marques de tir, poussée
 * du propulseur, projectiles et grenades, traces des vies, mise en valeur. Tout le reste — effets de mort,
 * poses, véhicules, objets portés, armes au sol, chaleur — passe par `ownershipShown`, qui lit les mêmes vies.
 */
export function shownSlotPredicate(
  focus: PlayerFocus | null,
): ((slot: number, frame: number) => boolean) | null {
  if (!focus || (!focus.solo && !focus.hide)) return null
  const owns = ownsLifeAt(focus.player)
  return focus.hide ? (slot, frame) => !owns(slot, frame) : owns
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
 * Seul et Masquer pour un élément à parties. `parties` vide = l'élément n'appartient à personne, ou son
 * attribution ne se lit pas : il se dessine toujours (jamais d'attribution devinée). `shared` : les parties
 * se partagent l'élément (les occupants d'un véhicule) ; Masquer ne le retire alors que si TOUTES sont le
 * joueur masqué — sinon il emporterait avec lui un occupant que l'utilisateur n'a pas masqué.
 */
export type OwnershipShown = (parties: readonly ElementParty[], shared?: boolean) => boolean

/**
 * ownershipShown est LE prédicat de Seul et Masquer pour tout élément qui appartient à un ou plusieurs
 * joueurs, sur tous les calques. `null` quand rien n'est filtré.
 *
 * LA RÈGLE. Un élément appartient à CHAQUE joueur qui y figure. Un frag (effet de mort, mort comptée par la
 * chaleur des éliminations), une poussée, une révélation de capteur et une destruction de véhicule appartiennent
 * à leur auteur OU à leur victime : Seul les garde dès que le joueur de la fiche est l'une des parties, Masquer
 * les retire dès qu'il l'est. Seule exception, la croix de mort de la piste : elle ne suit que la victime, la
 * piste ne portant pas le tueur. Ce qui n'appartient à personne reste affiché, comme le décor ; il en va de
 * même de tout élément dont le propriétaire est inconnu (aucune partie lue), sous Seul comme sous Masquer :
 * c'est la règle des éléments à parties comme celle des pions, des projectiles et des marques des vies.
 *
 * Chaque calque ne fait que dire QUELLES sont ses parties (`model/focusFilter.ts`) : la règle n'est écrite qu'ici.
 */
export function ownershipShown(focus: PlayerFocus | null): OwnershipShown | null {
  if (!focus || (!focus.solo && !focus.hide)) return null
  const { player } = focus
  const owns = ownsLifeAt(player)
  // Le xuid du FIL d'un bot est celui de sa ligne de feuille (sa clé de film est `bot:<nom>`).
  const boardXuid = player.board?.xuid
  const isPlayer = (p: ElementParty): boolean =>
    (p.life !== undefined && owns(p.life.slot, p.life.frame)) ||
    (p.xuid !== undefined && (p.xuid === player.xuid || (!!boardXuid && p.xuid === boardXuid)))
  return (parties, shared = false) => {
    if (parties.length === 0) return true
    if (focus.solo) return parties.some(isPlayer)
    return shared ? !parties.every(isPlayer) : !parties.some(isPlayer)
  }
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
 * killFxShown rend, pour le calque des morts, le prédicat qui dit si un effet se dessine : `null` sans
 * filtre. Seul : le joueur figure dans l'effet, tueur ou victime. Masquer : il n'y figure pas.
 */
export function killFxShown(
  focus: PlayerFocus | null,
): ((e: Parameters<typeof killFxParties>[0]) => boolean) | null {
  const shown = ownershipShown(focus)
  return shown ? (e) => shown(killFxParties(e)) : null
}
