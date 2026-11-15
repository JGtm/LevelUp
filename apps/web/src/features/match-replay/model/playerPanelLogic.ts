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
import type { KillFxEntry } from './killFx'
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

function weaponsOf(rows: readonly PlayerWeaponKillRow[] | null | undefined): PanelWeapon[] {
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
