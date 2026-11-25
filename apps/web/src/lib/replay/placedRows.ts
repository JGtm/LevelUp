/**
 * placedRows.ts — LES LIGNES DU FILM QU'UN BOT ÉCRIT, rattachées à leur joueur par la table du roster.
 *
 * UN BOT N'A PAS D'IDENTIFIANT. Le décodeur désigne son élimination ou son action de score par sa PLACE (l'index de
 * joueur du film, `roster[].filmIndex`), sans `xuid` ; de même pour la victime d'une élimination (`victimPlace`). La cible
 * d'une action de score n'est pas lue : l'API ne sert pas sa place. La frontière du document les ramène à la clé du film de leur joueur
 * (`ReplayPlayer.xuid` : le xuid d'un humain, `bot:<nom>` pour un bot) par `placeOwnerResolver`, la seule lecture d'une
 * place : tout le dossier du rejeu lit ensuite ces lignes comme celles d'un humain, par une seule clé d'identité.
 *
 * ON NE DEVINE PAS. Une ligne dont l'auteur ne se résout pas (sans `xuid`, place que le roster ne porte pas ou qu'il ne
 * départage pas) est écartée : aucune vie anonyme. Une victime qui ne se résout pas laisse la ligne, sans victime.
 *
 * Module pur : ni DOM, ni texte d'interface, ni couleur.
 */
import type { ReplayDocument } from '@/lib/api/types'

import type { ReplayEliminationReady, ReplayScoreActionReady } from './replayReadyTypes'

/** Le résolveur de places du roster (`placeOwnerResolver`). */
type OwnerOf = (place: number, frame: number) => string | null

type RawElimination = NonNullable<ReplayDocument['eliminations']>[number]
type RawScoreAction = NonNullable<ReplayDocument['scoreActions']>[number]

/** Un identifiant du document, sinon la clé de roster de la place qui le remplace, sinon rien. */
function keyOf(xuid: string | undefined, place: number | undefined, frame: number, ownerOf: OwnerOf): string | undefined {
  if (xuid) return xuid
  return place === undefined ? undefined : (ownerOf(place, frame) ?? undefined)
}

/** Les éliminations du document, tueur et victime ramenés à leur clé du film ; une ligne sans tueur résolu est écartée. */
export function eliminationsWithKeys(rows: readonly RawElimination[], ownerOf: OwnerOf): ReplayEliminationReady[] {
  const out: ReplayEliminationReady[] = []
  for (const { xuid, place, victim, victimPlace, ...rest } of rows) {
    const killer = keyOf(xuid, place, rest.t, ownerOf)
    if (killer === undefined) continue
    out.push({ ...rest, xuid: killer, victim: keyOf(victim, victimPlace, rest.t, ownerOf) })
  }
  return out
}

/** Les actions de score du document, destinataire ramené à sa clé du film ; une ligne sans destinataire résolu est écartée. */
export function scoreActionsWithKeys(rows: readonly RawScoreAction[], ownerOf: OwnerOf): ReplayScoreActionReady[] {
  const out: ReplayScoreActionReady[] = []
  for (const { xuid, place, ...rest } of rows) {
    const recipient = keyOf(xuid, place, rest.t, ownerOf)
    if (recipient === undefined) continue
    out.push({ ...rest, xuid: recipient })
  }
  return out
}
