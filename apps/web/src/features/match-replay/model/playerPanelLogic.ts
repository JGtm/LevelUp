/**
 * playerPanelLogic.ts — LE PANNEAU DE JOUEUR : qui est choisi, ce que ses trois actions font à la
 * carte, et ce qu'il affiche. Logique pure, sans React ni canvas (playerPanelLogic.test.ts).
 *
 * LE GESTE. Un clic sur une fiche de la colonne des équipes ouvre un panneau à la place du fil
 * d'événements ; « Retour au fil » le referme. Le panneau garde son joueur quand le temps avance
 * et change de contenu quand on clique une autre fiche. Rien de cela n'est mémorisé (D-4) : l'état
 * naît avec la page, il est tenu par elle.
 *
 * TROIS ACTIONS, UNE SEULE CHOSE À LA FOIS. Suivre cadre la carte sur le joueur à chaque image,
 * Seul n'affiche que lui, Masquer le retire. Exclusions (D-7) : Seul coupe Masquer ; Suivre coupe
 * Masquer ; Masquer coupe Seul ET Suivre. Elles s'appliquent à UN joueur à la fois : ouvrir le
 * panneau d'un autre joueur, comme le refermer, remet les trois à zéro — aucun état d'affichage
 * ne survit à son seul bouton, et un joueur masqué ne reste jamais masqué sans qu'on puisse le
 * voir. Les fiches, elles, ne sont jamais filtrées : un joueur présent reste dans la colonne.
 *
 * CE QUE LE PANNEAU AFFICHE. Les TOTAUX DE FIN DE MATCH lus sur la ligne de feuille du contexte
 * (D-8) : précision, dégâts, détail du score, objectif, armes. Aucune précision « à l'instant » —
 * le film ne la publie pas. Une valeur que la feuille ne porte pas est une LACUNE (`null`), jamais
 * un zéro ; une section dont aucune valeur n'est lue n'existe pas ; un joueur sans ligne de feuille
 * (non publié, ou bot que la feuille ne reconnaît pas) n'a aucune section, et le panneau le dit.
 * Les compteurs frags / morts / assistances de la tête restent ceux de l'INSTANT, ceux de la fiche.
 */
import type { MatchScoreboardRow, PlayerWeaponKillRow } from '@/lib/api/types'
import { isAliveAt } from '@/lib/replay/replayLogic'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

/** Les trois actions du panneau sur la carte. */
export type PlayerPanelAction = 'follow' | 'solo' | 'hide'

/** L'état du panneau : ouvert ou non, sur quel joueur (clé de roster), et ses trois actions. */
export interface PlayerPanelState {
  open: boolean
  /** Clé de roster du joueur choisi (`ReplayPlayer.xuid` : le xuid, ou `bot:<nom>` pour un bot). */
  xuid: string | null
  follow: boolean
  solo: boolean
  hide: boolean
}

export const NO_PLAYER_PANEL: PlayerPanelState = {
  open: false, xuid: null, follow: false, solo: false, hide: false,
}

export type PlayerPanelEvent =
  /** Un clic sur la fiche d'un joueur : ouvre, remplace le joueur, ou referme si elle est déjà ouverte. */
  | { type: 'card'; xuid: string }
  /** « Retour au fil ». */
  | { type: 'close' }
  /** Un des trois boutons d'action. */
  | { type: 'toggle'; action: PlayerPanelAction }
  /** Un geste manuel de zoom ou de déplacement sur la carte : Suivre s'arrête. */
  | { type: 'stopFollow' }

/**
 * reducePlayerPanel applique un événement à l'état. Un événement sans effet rend le MÊME objet,
 * pour ne réveiller personne (un geste manuel de plus pendant que Suivre est déjà éteint).
 */
export function reducePlayerPanel(state: PlayerPanelState, event: PlayerPanelEvent): PlayerPanelState {
  switch (event.type) {
    case 'card':
      if (state.open && state.xuid === event.xuid) return NO_PLAYER_PANEL
      return { open: true, xuid: event.xuid, follow: false, solo: false, hide: false }
    case 'close':
      return state.open ? NO_PLAYER_PANEL : state
    case 'stopFollow':
      return state.follow ? { ...state, follow: false } : state
    case 'toggle':
      return state.open ? toggleAction(state, event.action) : state
  }
}

/** Bascule une action en appliquant les exclusions de D-7. */
function toggleAction(state: PlayerPanelState, action: PlayerPanelAction): PlayerPanelState {
  const next = !state[action]
  if (!next) return { ...state, [action]: false }
  if (action === 'hide') return { ...state, hide: true, solo: false, follow: false }
  return { ...state, [action]: true, hide: false }
}

/** Le joueur dont le panneau est ouvert, avec ses actions : ce que la page donne au canvas. */
export interface PlayerFocus {
  player: ReplayPlayer
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
  return player ? { player, follow: state.follow, solo: state.solo, hide: state.hide } : null
}

/**
 * shownSlotPredicate dit si ce que porte un slot à une image se dessine, pour Seul et Masquer.
 * `null` quand rien n'est filtré (ni Seul ni Masquer) : les calques n'ont alors rien à brancher.
 *
 * UN SLOT N'EST PAS UN JOUEUR : le slot de bipède est réattribué à chaque réapparition, et un même
 * slot appartient à des joueurs différents au fil du match. Le prédicat désigne donc les VIES du
 * joueur choisi — son slot ET l'image qu'elle couvre — et c'est ce qui le rend juste pour un bot,
 * qui n'a pas de xuid mais a des vies.
 *
 * IL NE SERT QU'AUX ÉLÉMENTS PROPRES AU JOUEUR (pion, nom, cône de visée, traînée, tirs, marques de
 * tir, poussée du propulseur). Les morts, les objectifs, les véhicules et les objets posés sont des
 * événements du match : ni Seul ni Masquer ne les retirent.
 */
export function shownSlotPredicate(
  focus: PlayerFocus | null,
): ((slot: number, frame: number) => boolean) | null {
  if (!focus || (!focus.solo && !focus.hide)) return null
  const lives = focus.player.lives
  const owns = (slot: number, frame: number) => lives.some((l) => l.slot === slot && isAliveAt(l, frame))
  return focus.hide ? (slot, frame) => !owns(slot, frame) : owns
}

// ---------------------------------------------------------------------------------------------
// Ce que le panneau affiche
// ---------------------------------------------------------------------------------------------

/** Les champs d'objectif de la feuille que le panneau sait nommer : une table FERMÉE, clés de l'API. */
export const OBJECTIVE_ROW_KEYS = [
  'flag_captures', 'flag_returns', 'flag_grabs', 'flag_carriers_killed', 'flag_capture_assists',
  'zone_captures', 'zone_offensive_kills', 'zone_defensive_kills',
  'bomb_grabs', 'bomb_arms', 'bomb_detonations', 'bomb_carriers_killed',
  'skull_grabs', 'skull_carriers_killed',
] as const

export type ObjectiveRowKey = (typeof OBJECTIVE_ROW_KEYS)[number]

export type PanelRowKey =
  | 'precision' | 'shotsHit' | 'shotsFired'
  | 'damageDealt' | 'damageTaken'
  | 'score' | 'headshotKills' | 'powerWeaponKills' | 'meleeKills' | 'grenadeKills'
  | ObjectiveRowKey

/** Une ligne de liste : sa clé de libellé, son format, sa valeur — `null` = non lue (une lacune). */
export interface PanelRow {
  key: PanelRowKey
  format: 'integer' | 'percent'
  value: number | null
}

/** Une arme de la feuille : combien d'éliminations le joueur y a faites. */
export interface PanelWeapon {
  weaponId: number
  /** `null` = arme hors référentiel : la liste écrit son numéro, jamais le nom d'une autre. */
  label: string | null
  imageUrl: string | null
  tinted: boolean
  kills: number
}

export type PanelSectionId = 'precision' | 'damage' | 'score' | 'objective' | 'weapons'

export interface PanelSection {
  id: PanelSectionId
  rows: PanelRow[]
  /** Seule la section d'armes en porte. */
  weapons?: PanelWeapon[]
}

export interface PlayerPanelSheet {
  /** Faux = aucune ligne de feuille pour ce joueur : le panneau l'écrit au lieu de sections vides. */
  hasSheet: boolean
  sections: PanelSection[]
}

/** Le nombre d'armes listées : les plus meurtrières, comme le tiroir du tableau de score. */
const WEAPONS_LISTED = 5

function row(key: PanelRowKey, value: number | null | undefined, format: PanelRow['format'] = 'integer'): PanelRow {
  const v = value == null || !Number.isFinite(value) ? null : value
  return { key, format, value: v === null || format === 'percent' ? v : Math.round(v) }
}

/** Une section n'existe que si au moins une de ses valeurs est lue. */
function section(id: PanelSectionId, rows: PanelRow[], weapons?: PanelWeapon[]): PanelSection[] {
  if (!rows.some((r) => r.value !== null) && !weapons?.length) return []
  return [weapons ? { id, rows, weapons } : { id, rows }]
}

function objectiveRows(objective: MatchScoreboardRow['objective']): PanelRow[] {
  if (!objective) return []
  const stats = objective as unknown as Record<string, unknown>
  return OBJECTIVE_ROW_KEYS.filter((k) => typeof stats[k] === 'number').map((k) => row(k, stats[k] as number))
}

function weaponsOf(rows: readonly PlayerWeaponKillRow[] | undefined): PanelWeapon[] {
  return [...(rows ?? [])]
    .filter((w) => w.kills > 0)
    .sort((a, b) => b.kills - a.kills)
    .slice(0, WEAPONS_LISTED)
    .map((w) => ({
      weaponId: w.weapon_id,
      label: w.label ?? null,
      imageUrl: w.image_url ?? null,
      tinted: w.image_tinted ?? false,
      kills: w.kills,
    }))
}

/**
 * playerPanelSheet compose les sections du panneau depuis la ligne de feuille du joueur, dans
 * l'ordre de la maquette : précision, dégâts, détail du score, objectif, armes.
 *
 * `grenadeKills` n'a de ligne que si la feuille PORTE le champ (absent d'un titre qui ne le publie
 * pas) : une ligne qui n'existe pas dans le titre n'est pas une lacune, elle n'a pas lieu d'être.
 */
export function playerPanelSheet(board: MatchScoreboardRow | undefined): PlayerPanelSheet {
  if (!board) return { hasSheet: false, sections: [] }
  const score = [
    row('score', board.score),
    row('headshotKills', board.headshot_kills),
    row('powerWeaponKills', board.power_weapon_kills),
    row('meleeKills', board.melee_kills),
    ...(board.grenade_kills === undefined ? [] : [row('grenadeKills', board.grenade_kills)]),
  ]
  const weapons = weaponsOf(board.weapon_kills)
  return {
    hasSheet: true,
    sections: [
      ...section('precision', [
        row('precision', board.accuracy, 'percent'),
        row('shotsHit', board.shots_hit),
        row('shotsFired', board.shots_fired),
      ]),
      ...section('damage', [row('damageDealt', board.damage_dealt), row('damageTaken', board.damage_taken)]),
      ...section('score', score),
      ...section('objective', objectiveRows(board.objective)),
      ...section('weapons', [], weapons),
    ],
  }
}
