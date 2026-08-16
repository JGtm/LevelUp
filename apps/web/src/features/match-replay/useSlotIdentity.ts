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
 * propriétaire est ce qui ne change pas d'une vie à l'autre. Une vie sans propriétaire tombe
 * sur les replis — encre neutre, aucune marque, aucun nom.
 */
import { useCallback, useMemo } from 'react'

import type { XuidMeta } from '@/features/match-view/xuidMeta'
import type { MatchScoreboardRow } from '@/lib/api/types'

import { NO_MARKS, type PlayerMarkKind } from './playerMarks'
import { buildPlayers, colorBySlot, markBySlot, nameBySlot } from './rosterLogic'
import type { ReplayDocumentReady } from './replayNormalize'

export interface SlotIdentity {
  /** Couleur d'équipe d'une vie, repli neutre compris — pour les marqueurs et les traînées. */
  colorOfSlot: (slot: number) => string
  /** La MÊME table, brute : les effets de mort veulent distinguer « pas de couleur » du repli. */
  slotColors: ReadonlyMap<number, string>
  markOfSlot: (slot: number) => PlayerMarkKind | undefined
  nameOfSlot: (slot: number) => string | null
}

export interface SlotIdentityInput {
  doc: ReplayDocumentReady
  scoreboard: MatchScoreboardRow[] | undefined
  xuidMeta: XuidMeta | undefined
  marks: ReadonlyMap<string, PlayerMarkKind> | undefined
  /** Couleur d'un camp, tokens déjà résolus par l'appelant (ils suivent la palette). */
  teamColorOf: (ally: boolean) => string
  /** Encre servie quand l'identité est inconnue : ni équipe inventée, ni point invisible. */
  neutral: string
}

export function useSlotIdentity({
  doc, scoreboard, xuidMeta, marks, teamColorOf, neutral,
}: SlotIdentityInput): SlotIdentity {
  // LA JOINTURE FILM <-> BASE, faite une fois : elle donne à chaque vie son propriétaire.
  const players = useMemo(() => buildPlayers(doc, scoreboard ?? []), [doc, scoreboard])
  const slotColors = useMemo(
    () => colorBySlot(players, teamColorOf, (xuid) => xuidMeta?.get(xuid)?.ally ?? false, neutral),
    [players, teamColorOf, xuidMeta, neutral],
  )
  const colorOfSlot = useCallback(
    (slot: number) => slotColors.get(slot) ?? neutral,
    [slotColors, neutral],
  )
  const slotMarks = useMemo(() => markBySlot(players, marks ?? NO_MARKS), [players, marks])
  const markOfSlot = useCallback((slot: number) => slotMarks.get(slot), [slotMarks])
  const slotNames = useMemo(() => nameBySlot(players), [players])
  const nameOfSlot = useCallback((slot: number) => slotNames.get(slot) ?? null, [slotNames])

  return { colorOfSlot, slotColors, markOfSlot, nameOfSlot }
}
