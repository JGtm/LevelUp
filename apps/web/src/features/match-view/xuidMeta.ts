/**
 * xuidMeta.ts — QUI EST QUI DANS UN MATCH : nom d'affichage et camp, par xuid.
 *
 * POURQUOI CE FICHIER EXISTE. La même cascade était écrite deux fois à l'identique
 * (`MatchTugOfWarChart`, `MatchKDCumulChart`) et le rejeu 2D en réclamait une troisième.
 * Règle du dépôt : à la troisième copie on centralise ET on pose le garde-rail — ici
 * `xuidMeta.guard.test.ts`, qui interdit de réécrire la cascade ailleurs.
 *
 * LE CAMP N'EST PAS DANS LA DONNÉE, il se déduit. « Allié » veut dire « du côté du joueur
 * dont on regarde la page » : on lit le `team_side` de sa ligne, et tout le monde qui
 * partage ce camp est allié. Sans ligne pour lui, personne n'est allié — mieux vaut aucun
 * camp qu'un camp inventé, car c'est la couleur des kills qui en dépend.
 *
 * DEUX RÉGIMES DEPUIS LE 2026-09-06, et le second n'existe que pour le rejeu 2D. La page de
 * rejeu se regarde désormais PAR LES YEUX D'UN JOUEUR CHOISI (plan « frise, point de vue »),
 * qui n'est pas forcément celui de la page : elle passe ce joueur en TROISIÈME argument. Sans
 * ce troisième argument, la fonction fait exactement ce qu'elle faisait — court-circuit
 * `is_me` compris, cf. la caractérisation `xuidMeta.test.ts`. Les cinq charts de la page match
 * appellent à deux arguments et ne changent donc pas d'un iota (décision 15 du plan).
 */
import { displayPlayerName } from '@/lib/players/displayName'
import type { MatchScoreboardRow } from '@/lib/api/types'

/** Nom d'affichage (bots compris) et appartenance, par xuid. */
export type XuidMeta = ReadonlyMap<string, { gamertag: string; ally: boolean }>

/**
 * resolveXuidMeta indexe le scoreboard par xuid.
 *
 * `meXUID` désigne le joueur de la page ; sa ligne peut aussi se reconnaître à `is_me`,
 * qui reste prioritaire (une ligne marquée « moi » est alliée par définition).
 *
 * `viewpoint` (2026-09-06, rejeu 2D seulement) DÉSACTIVE ce court-circuit : quand il est
 * fourni, « allié » veut dire « du côté de CE joueur-là », un point c'est tout. Il le faut :
 * vu depuis un adversaire, le court-circuit laisserait la ligne `is_me` alliée en même temps
 * que le camp adverse — DEUX camps alliés à la fois, donc des kills des deux couleurs sur la
 * même frise (c'est le cas (c) de la caractérisation). Un `viewpoint` absent du tableau de
 * score, ou dont le camp n'est pas transmis, ne fait allié PERSONNE : aucun camp deviné, même
 * règle que partout ailleurs.
 */
export function resolveXuidMeta(
  scoreboard: MatchScoreboardRow[] | null | undefined,
  meXUID: string | null,
  viewpoint?: string | null,
): XuidMeta {
  const sb = scoreboard ?? []
  const sujet = viewpoint ?? meXUID
  const meRow = sujet ? sb.find((r) => r.xuid === sujet) : undefined
  const allyTeam = meRow?.team_side ?? null
  // LE REPLI `is_me` N'EXISTE QUE SANS POINT DE VUE : c'est la seule différence entre les deux
  // régimes, et elle tient en ce booléen. Voir l'en-tête du module.
  const repliSurMoi = viewpoint == null
  const meta = new Map<string, { gamertag: string; ally: boolean }>()
  for (const r of sb) {
    const ally = (repliSurMoi && !!r.is_me) || (allyTeam != null && r.team_side === allyTeam)
    meta.set(r.xuid, { gamertag: displayPlayerName(r.gamertag, r.xuid), ally })
  }
  return meta
}

/**
 * meXUIDOf retrouve le joueur de la page dans le scoreboard, quand l'appelant ne le tient
 * pas d'ailleurs. `is_me` est posé par le backend sur la ligne du joueur consulté.
 *
 * L'ENTRÉE EST VOLONTAIREMENT LARGE (2026-09-06) : `resolveViewpoint`, le foyer du point de
 * vue du rejeu, s'en sert pour son repli et ne manipule que `xuid` + `is_me`. Le réduire à ces
 * deux champs évite d'y recopier la recherche de la ligne « moi » — une sixième copie de la
 * lecture que le lot L2b vient précisément de supprimer.
 */
export function meXUIDOf(
  scoreboard: readonly Pick<MatchScoreboardRow, 'xuid' | 'is_me'>[] | null | undefined,
): string | null {
  return (scoreboard ?? []).find((r) => r.is_me)?.xuid ?? null
}
