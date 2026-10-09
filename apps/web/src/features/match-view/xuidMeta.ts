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
 * LA PAGE REJEU N'EN LIT QUE LES NOMS (2026-10-06). Son allégeance vient de l'équipe du FILM
 * (`lib/replay/filmAllegiance.ts`) : la table d'identité lui sert à écrire les gamertags du fil
 * et à filtrer les kills de la base, jamais à dire qui est allié. Le régime à trois arguments
 * qu'elle employait (« allié » relatif à son point de vue, 2026-09-06) a été retiré avec ce
 * dernier lecteur ; les cinq charts de la page Match appellent à deux arguments, inchangés.
 */
import { displayPlayerName } from '@/lib/players/displayName'
import type { MatchScoreboardRow } from '@/lib/api/types'
import { parseTeamSideID } from '@/lib/halo/teamNames'

/** Nom d'affichage (bots compris) et appartenance, par xuid. */
export type XuidMeta = ReadonlyMap<string, { gamertag: string; ally: boolean }>

/**
 * resolveXuidMeta indexe le scoreboard par xuid.
 *
 * `meXUID` désigne le joueur de la page ; sa ligne peut aussi se reconnaître à `is_me`,
 * qui reste prioritaire (une ligne marquée « moi » est alliée par définition — y compris en
 * mêlée générale, où personne n'a de `team_side` : le camp des AUTRES reste indéductible, celui
 * du joueur de la page ne se déduit pas, il est donné).
 */
export function resolveXuidMeta(
  scoreboard: MatchScoreboardRow[] | null | undefined,
  meXUID: string | null,
): XuidMeta {
  const sb = scoreboard ?? []
  const meRow = meXUID ? sb.find((r) => r.xuid === meXUID) : undefined
  const allyTeam = meRow?.team_side ?? null
  const meta = new Map<string, { gamertag: string; ally: boolean }>()
  for (const r of sb) {
    const ally = !!r.is_me || (allyTeam != null && r.team_side === allyTeam)
    meta.set(r.xuid, { gamertag: displayPlayerName(r.gamertag, r.xuid), ally })
  }
  return meta
}

/**
 * allyOfTeamId dit si une équipe (`teamId`, le numéro d'un côté de feuille `t{N}`, celui des
 * séries du calque de score) est du côté du joueur de la page, POUR LA PAGE MATCH.
 *
 * La page raisonne en « allié / adverse », une notion RELATIVE au joueur consulté ; le pont
 * passe par la feuille, seul endroit où le côté et le xuid coexistent, et par la cascade
 * ci-dessus (`allies`). `null` = côté introuvable ou aucun joueur reconnu : la marque prend une
 * encre neutre, jamais l'une des deux couleurs par défaut.
 *
 * LA PAGE REJEU NE L'EMPLOIE PAS : son allégeance vient de l'équipe du FILM
 * (`lib/replay/filmAllegiance.ts`, 2026-10-06). Ce lecteur de feuille vivait dans
 * `lib/replay/scoreTimeline.ts` ; il a rejoint la cascade dont il dépend quand le rejeu a cessé
 * d'en être un lecteur.
 */
export function allyOfTeamId(
  scoreboard: ReadonlyArray<Pick<MatchScoreboardRow, 'xuid' | 'team_side'>>,
  allies: ReadonlyMap<string, { ally: boolean }> | undefined,
  teamId: number,
): boolean | null {
  if (!allies) return null
  for (const row of scoreboard) {
    if (parseTeamSideID(row.team_side) !== teamId) continue
    const meta = allies.get(row.xuid)
    if (meta) return meta.ally
  }
  return null
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
