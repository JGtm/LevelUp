/**
 * useSlotIdentity — CE QUE LE CALQUE DES JOUEURS SAIT D'UNE VIE, par slot : sa couleur
 * d'équipe, sa marque d'identité (« moi », « ami ») et le nom à écrire dessous.
 *
 * POURQUOI UN HOOK À PART. Le calcul est UNE jointure (film <-> scoreboard) et TROIS
 * descentes « joueur -> ses vies », toutes pures et déjà écrites dans `rosterLogic.ts` ;
 * ce fichier n'ajoute que la mémoïsation React. Le poser ici garde `ReplayCanvas.tsx` — qui
 * porte déjà une dette de taille gelée — à sa longueur, au lieu d'y coller vingt-cinq lignes
 * de plus (CLAUDE.md n°5).
 *
 * TOUT EST INDEXÉ PAR SLOT, jamais par rang de trace : un slot est une VIE, et son
 * propriétaire est ce qui ne change pas d'une vie à l'autre.
 *
 * UNE VIE SANS PROPRIÉTAIRE NE SE DESSINE PAS. `buildPlayers` écarte les traces sans xuid
 * (`if (!track.xuid) continue`) : leur slot n'entre donc dans aucune table, et `colorOfSlot`
 * rend `null` — la convention du calque pour « ne rien dessiner » (cf. `MarkerStyle`). Ce
 * sont les caméras et les spectateurs de fin de partie ; les replier sur l'encre neutre
 * semait des pions gris qui ne désignaient personne (retour utilisateur du 2026-08-20). La
 * marque et le nom, eux, tombent déjà sur `undefined` / `null`.
 */
import { useCallback, useMemo } from 'react'

import type { XuidMeta } from '@/features/match-view/xuidMeta'
import type { MatchScoreboardRow } from '@/lib/api/types'

import { NO_MARKS, type PlayerMarkKind } from './playerMarks'
import {
  buildPlayers, colorBySlot, markBySlot, nameBySlot, sideBySlot, type ReplayPlayer,
} from './rosterLogic'
import type { ReplayDocumentReady } from './replayNormalize'

export interface SlotIdentity {
  /**
   * Couleur d'équipe d'une vie — pour les marqueurs et les traînées. `null` = vie sans
   * propriétaire, donc RIEN à dessiner (convention `MarkerStyle.colorOfSlot`).
   */
  colorOfSlot: (slot: number) => string | null
  /** La MÊME table, brute. Elle ne porte que les slots dont le propriétaire est connu. */
  slotColors: ReadonlyMap<number, string>
  markOfSlot: (slot: number) => PlayerMarkKind | undefined
  nameOfSlot: (slot: number) => string | null
  /**
   * CAMP d'une vie (`team_side`), null quand il est inconnu. Distinct de la couleur : celle-ci
   * ne connaît que « allié / adverse » vu du joueur de la page, alors qu'opposer deux vies
   * demande leur camp réel (cf. rosterLogic.sideBySlot). Le capteur de menaces en a besoin.
   */
  sideOfSlot: (slot: number) => string | null
}

export interface SlotIdentityInput {
  doc: ReplayDocumentReady
  scoreboard: MatchScoreboardRow[] | undefined
  xuidMeta: XuidMeta | undefined
  marks: ReadonlyMap<string, PlayerMarkKind> | undefined
  /** Couleur d'un camp, tokens déjà résolus par l'appelant (ils suivent la palette). */
  teamColorOf: (ally: boolean) => string
  /**
   * Encre servie à une entrée de roster SANS xuid : elle a un slot, donc elle se dessine,
   * mais aucune équipe ne peut lui être attribuée (cf. `rosterLogic.colorBySlot`). À ne pas
   * confondre avec une vie ABSENTE des tables — celle-là ne se dessine pas du tout.
   */
  neutral: string
  /**
   * COULEURS DISTINCTES PAR JOUEUR (option du tiroir, 2026-08-24) : une palette résolue,
   * cyclée sur l'ordre STABLE des joueurs de la jointure — la couleur d'un joueur ne bouge
   * pas d'une image à l'autre ni d'une vie à l'autre. `null`/absente = couleurs d'équipe
   * (le défaut, doctrine D1).
   */
  distinctColors?: readonly string[] | null
}

/**
 * distinctSlotColors — la table slot -> couleur du mode « distinctes par joueur » : chaque
 * joueur prend la couleur de son RANG dans la jointure (cyclée au-delà de la palette), et
 * toutes ses vies la portent. Pure, testée à part du hook.
 */
export function distinctSlotColors(
  players: readonly ReplayPlayer[],
  colors: readonly string[],
): ReadonlyMap<number, string> {
  const bySlot = new Map<number, string>()
  players.forEach((p, i) => {
    const color = colors[i % colors.length]
    for (const life of p.lives) bySlot.set(life.slot, color)
  })
  return bySlot
}

export function useSlotIdentity({
  doc, scoreboard, xuidMeta, marks, teamColorOf, neutral, distinctColors,
}: SlotIdentityInput): SlotIdentity {
  // LA JOINTURE FILM <-> BASE, faite une fois : elle donne à chaque vie son propriétaire.
  const players = useMemo(() => buildPlayers(doc, scoreboard ?? []), [doc, scoreboard])
  const slotColors = useMemo(
    () =>
      distinctColors && distinctColors.length > 0
        ? distinctSlotColors(players, distinctColors)
        : colorBySlot(players, teamColorOf, (xuid) => xuidMeta?.get(xuid)?.ally ?? false, neutral),
    [players, teamColorOf, xuidMeta, neutral, distinctColors],
  )
  // PAS DE REPLI ICI : un slot absent de la table est une vie sans propriétaire (caméra,
  // spectateur de fin de partie), et `null` dit au calque de ne rien dessiner. Le repli sur
  // l'encre neutre semait des pions gris qui ne désignaient personne.
  const colorOfSlot = useCallback(
    (slot: number): string | null => slotColors.get(slot) ?? null,
    [slotColors],
  )
  const slotMarks = useMemo(() => markBySlot(players, marks ?? NO_MARKS), [players, marks])
  const markOfSlot = useCallback((slot: number) => slotMarks.get(slot), [slotMarks])
  const slotNames = useMemo(() => nameBySlot(players), [players])
  const nameOfSlot = useCallback((slot: number) => slotNames.get(slot) ?? null, [slotNames])
  const slotSides = useMemo(() => sideBySlot(players), [players])
  const sideOfSlot = useCallback((slot: number) => slotSides.get(slot) ?? null, [slotSides])

  return { colorOfSlot, slotColors, markOfSlot, nameOfSlot, sideOfSlot }
}
