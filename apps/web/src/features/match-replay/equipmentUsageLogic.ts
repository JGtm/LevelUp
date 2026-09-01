/**
 * equipmentUsageLogic.ts — CE QUE CHAQUE JOUEUR A FAIT DE SON ÉQUIPEMENT, agrégé sur tout le
 * match depuis le document de rejeu DÉJÀ SERVI.
 *
 * ZÉRO GO, ZÉRO SCHÉMA, ZÉRO RE-CUISSON (patron `PLAN_TEMPS_MORT_WEB.md`) : tout ce qui suit se
 * lit dans les calques que l'artefact publie déjà. Le rejeu montre ces gestes IMAGE PAR IMAGE ;
 * ce module les COMPTE, ce que le rejeu ne fait pas — un joueur qui a posé six murs ne se voit
 * pas en regardant six instants séparés de trois minutes.
 *
 * QUATRE CANAUX ATTRIBUÉS, ET UN CINQUIÈME QUI NE L'EST PAS. C'est la ligne de partage de tout
 * ce fichier :
 *   - `grappleLines[].slot`      — les TRACTIONS de grappin, la seule ACTIVATION de capacité que
 *                                  le film mesure et attribue ;
 *   - `equipmentEpisodes[]`      — les épisodes d'ÉTAT ACTIF (camouflage, surbouclier) : leur
 *                                  nombre et leur durée cumulée. C'est un PROXY, et sa réserve
 *                                  est affichée à l'écran (cf. `EquipmentUsageEpisode`) ;
 *   - `equipmentPlacements[]` `deployed` — les DÉPLOIEMENTS, par famille ;
 *   - `equipmentPlacements[]` `dropped`  — ce qu'on LÂCHE en mourant, par famille ;
 *   - `grenades[]`               — les lancers, par type. ATTENTION, LA CLÉ N'EST PAS LA MÊME :
 *                                  `Grenade.slot` est « le biped lanceur QUAND IL EST CONNU
 *                                  (0 sinon) », l'auteur est `Grenade.i`, l'index de joueur
 *                                  ÉCRIT dans le film (cf. grenades.go, et `grenadeThrowActive`
 *                                  qui joint déjà par là). Mesuré sur quatre témoins du cache :
 *                                  65/70, 108/143, 123/130 lancers portent un slot ABSENT des
 *                                  pistes — joindre par le slot perdrait la quasi-totalité des
 *                                  lancers, ou pire, les verserait tous au propriétaire du
 *                                  slot 0 sur un film où ce slot existe ;
 *   - `padPickups[]` x `weaponPads[]` de famille `powerup_*` — les SOCLES DE BONUS VIDÉS. Ce
 *     canal-là reste au niveau du MATCH, il ne descend sur aucune ligne de joueur. Depuis le
 *     schéma 30 (2026-08-31) `padPickups[].xuid` PEUT être renseigné (l'événement natif porte
 *     son ramasseur), mais cet écran n'a pas été repensé pour l'exploiter : ne pas descendre
 *     reste le comportement VOULU, et ce n'est plus une impossibilité, c'est un choix. Le
 *     ramasseur NOMMÉ est le sujet d'un tableau à part — `padControlLogic.ts` / la section
 *     « Contrôle des armes spéciales », juste sous celle-ci dans l'onglet Chronologie.
 *
 * RÉPULSEUR ET PROPULSEUR N'ONT AUCUNE GRANDEUR ICI, et c'est une absence de DONNÉE, pas un
 * oubli : le film ne publie aucun canal d'activation pour ces deux capacités (le chantier qui
 * le cherche est en cours). Une colonne vide affirmerait « zéro utilisation » là où la vérité
 * est « non mesuré ».
 *
 * LE PONT SLOT -> JOUEUR -> ÉQUIPE EST CELUI DU REJEU, réutilisé tel quel (`buildPlayers`,
 * `indexBySlot`, `groupByTeam` de rosterLogic) : un slot est une VIE, son propriétaire est le
 * xuid, et l'ÉQUIPE vient du scoreboard — `Track.Team` vaut -1 partout, le film n'en porte
 * aucune. Un joueur du film SANS ligne de scoreboard garde donc sa ligne, sans équipe : le trou
 * se montre, il ne se comble pas.
 *
 * Tout est PUR : aucun React, aucun canvas, aucune couleur, AUCUNE LANGUE — ce fichier compte,
 * il ne nomme rien. La mise en colonnes et les libellés vivent dans `equipmentUsageColumns.ts`
 * (extrait le 2026-08-25, seuil de taille), le rendu dans `MatchEquipmentUsageSection.tsx`.
 */
import type { MatchScoreboardRow } from '@/lib/api/types'
import { displayPlayerName } from '@/lib/players/displayName'

import { EQUIP_FAMILY_CAMO, EQUIP_FAMILY_OVERSHIELD } from './equipmentFx'
import { PLACEMENT_RENDER, placementIsDeployedObject } from './equipmentPlacementsLayer'
import { placementIsDroppedPower, PLACEMENT_DROPPED_FAMILIES } from './placementDropped'
import type { ReplayDocumentReady } from './replayNormalize'
import { frameToMs } from './replayLogic'
import { buildPlayers, groupByTeam, indexBySlot, playerName, type ReplayPlayer } from './rosterLogic'
import { padEquipmentFamilyOf, type PadEquipmentFamilyKey } from './weaponPadFamilies'

/** Les deux familles dont l'ÉTAT ACTIF est mesuré (identifiants stables du document). */
export const EPISODE_FAMILIES = [EQUIP_FAMILY_CAMO, EQUIP_FAMILY_OVERSHIELD] as const
export type EquipmentEpisodeFamily = (typeof EPISODE_FAMILIES)[number]

/**
 * Un état actif cumulé : combien d'épisodes, combien de temps en tout, et combien de frags
 * DU PORTEUR pendant ces épisodes.
 *
 * COMPTE ET DURÉE SE LISENT ENSEMBLE : six épisodes d'une seconde et un épisode de six
 * secondes ne racontent pas la même partie, et le nombre seul ne les distingue pas.
 *
 * `kills` EST UNE SOMME SUR DES EPISODES, PAS UNE MESURE DE MATCH À ELLE SEULE : elle vaut
 * zéro aussi bien quand le porteur n'a rien tué sous l'effet que quand la jointure n'a pas pu
 * être tentée pour ce match — c'est `EquipmentUsageCoverage.killsRead` (document entier) qui
 * distingue les deux, jamais ce champ seul (PLAN_RETOURS_UTILISATEUR_2026-08-29 §LOT F.2).
 */
export interface EquipmentUsageEpisode {
  count: number
  /** Durée cumulée, en millisecondes (les épisodes sont datés en frames). */
  ms: number
  /** Frags DU PORTEUR pendant SES épisodes de cette famille (`EquipmentEpisode.k`, sommés). */
  kills: number
}

/** Les grandeurs comptées, sans identité — la ligne d'un joueur comme le total d'une équipe. */
export interface EquipmentUsageTally {
  /** Tractions de grappin (`grappleLines`). */
  grapplePulls: number
  /** Par famille mesurée (`camo`, `overshield`). */
  episodes: Record<string, EquipmentUsageEpisode>
  /** Déploiements, par famille du document. */
  deployed: Record<string, number>
  /** Objets lâchés à la mort, par famille du document. */
  dropped: Record<string, number>
  /** Lancers de grenade, par RANG du catalogue du document (`grenadeLabels[rank]`). */
  grenades: Record<number, number>
}

/** La ligne d'un joueur : son identité, son camp, ses grandeurs. */
export interface EquipmentUsageRow extends EquipmentUsageTally {
  xuid: string
  /** Nom d'affichage (`displayPlayerName`), jamais un xuid brut. */
  name: string
  /** `team_side` du scoreboard ; `null` = joueur du film absent du scoreboard. */
  side: string | null
}

/** Un camp et ses joueurs, avec son total. `side` null = les joueurs sans ligne de scoreboard. */
export interface EquipmentUsageTeam {
  side: string | null
  players: EquipmentUsageRow[]
  total: EquipmentUsageTally
}

/**
 * LES COLONNES QUE LA DONNÉE JUSTIFIE, et elles seules. Une famille qu'aucun joueur n'a
 * employée n'a pas de colonne : une colonne de zéros occupe la largeur d'une mesure sans en
 * être une.
 */
export interface EquipmentUsageColumns {
  grapple: boolean
  episodes: EquipmentEpisodeFamily[]
  deployed: string[]
  dropped: string[]
  grenades: number[]
}

/**
 * LES DÉNOMINATEURS, repris de `doc.coverage` — jamais recalculés ici. « 4 tractions » ne se
 * juge pas sans savoir combien de vies le calque a lues, et un artefact dont un calque n'a
 * rien lu se distingue ainsi d'un match où personne n'a rien fait.
 *
 * Tous les blocs de couverture sont OPTIONNELS au contrat (un artefact ancien n'en porte pas) :
 * l'absence se lit zéro, et l'écran ne montre alors aucun dénominateur.
 */
export interface EquipmentUsageCoverage {
  /** `equipment.tracksTotal` : vies publiées, les seules où un épisode peut exister. */
  tracksTotal: number
  /** `equipment.camoLives` / `overshieldLives` : vies portant au moins un épisode. */
  episodeLives: Record<string, number>
  /** `grapple.pulls` / `pullLives` : tractions publiées, et vies en portant au moins une. */
  grapplePulls: number
  grapplePullLives: number
  /** `placements.byFamilyOrigin` : le croisement famille x origine, clé `famille/origine`. */
  placementsByFamilyOrigin: Record<string, number>
  /** `groundWeapons.powerupPads` : socles de bonus publiés — le dénominateur des vidages. */
  powerupPads: number
  /**
   * `equipment.killsRead` : les frags/assistances sous effet actif ont-ils pu être MESURÉS
   * pour ce match ? Faux = jointure non tentée (killsource non décodé, porte de publication
   * ligne-par-ligne fermée, ou origine d'horloge non établie) — DISTINCT d'un compte à zéro
   * mesuré. La cellule de colonne « Frags sous <famille> » lit CE champ, jamais
   * `episodes[fam].kills` seul, pour choisir entre un nombre et « — ».
   */
  killsRead: boolean
}

/** Le résultat complet : par joueur, par équipe, et ce qui reste au niveau du match. */
export interface EquipmentUsage {
  byPlayer: EquipmentUsageRow[]
  byTeam: EquipmentUsageTeam[]
  columns: EquipmentUsageColumns
  coverage: EquipmentUsageCoverage
  /**
   * SOCLES DE BONUS VIDÉS, par famille — ANONYME et au niveau du MATCH. Ne jamais descendre sur
   * une ligne de joueur : le ramasseur n'est pas publié (cf. en-tête).
   */
  powerupPickups: Record<string, number>
  powerupPickupsTotal: number
  /**
   * CE QUI EST MESURÉ MAIS SANS PROPRIÉTAIRE : gestes dont le slot n'appartient à aucun joueur
   * (caméras, spectateurs de fin de partie), ou pose sans poseur mesuré (`owner` -1). Compté
   * pour que la somme des lignes ne mente pas sur le total du film.
   */
  unattributed: EquipmentUsageTally
  /** Faux = aucune grandeur non nulle : l'écran ne doit rien rendre (double porte). */
  hasData: boolean
}

/** Un compteur vide. Chaque appel rend un NOUVEL objet : les tables ne se partagent pas. */
function emptyTally(): EquipmentUsageTally {
  return { grapplePulls: 0, episodes: {}, deployed: {}, dropped: {}, grenades: {} }
}

/** Incrémente une case de table, en la créant au besoin. */
function bump<K extends string | number>(table: Record<K, number>, key: K, by = 1): void {
  table[key] = (table[key] ?? 0) + by
}

/** Ajoute un épisode (un de plus, sa durée, ses frags) à une table par famille. */
function addEpisode(
  table: Record<string, EquipmentUsageEpisode>,
  fam: string,
  ms: number,
  kills: number,
): void {
  const cur = table[fam] ?? { count: 0, ms: 0, kills: 0 }
  table[fam] = { count: cur.count + 1, ms: cur.ms + Math.max(0, ms), kills: cur.kills + kills }
}

/** Verse `src` dans `dst` — le total d'une équipe est la somme de ses lignes, rien de plus. */
function mergeTally(dst: EquipmentUsageTally, src: EquipmentUsageTally): void {
  dst.grapplePulls += src.grapplePulls
  for (const [fam, ep] of Object.entries(src.episodes)) {
    const cur = dst.episodes[fam] ?? { count: 0, ms: 0, kills: 0 }
    dst.episodes[fam] = { count: cur.count + ep.count, ms: cur.ms + ep.ms, kills: cur.kills + ep.kills }
  }
  for (const [fam, n] of Object.entries(src.deployed)) bump(dst.deployed, fam, n)
  for (const [fam, n] of Object.entries(src.dropped)) bump(dst.dropped, fam, n)
  for (const [rank, n] of Object.entries(src.grenades)) bump(dst.grenades, Number(rank), n)
}

/** Vrai si le compteur porte au moins une grandeur non nulle. */
export function tallyIsEmpty(t: EquipmentUsageTally): boolean {
  if (t.grapplePulls > 0) return false
  if (Object.values(t.episodes).some((e) => e.count > 0)) return false
  return ![t.deployed, t.dropped, t.grenades].some((m) => Object.values(m).some((n) => n > 0))
}

/**
 * LES FAMILLES DONT UN DÉPLOIEMENT EST UN GESTE, et pourquoi la liste n'est pas « toutes ».
 *
 * `PLACEMENT_RENDER` porte déjà la décision, mesurée et écrite : une famille à `null` y est une
 * famille CONNUE qu'on ne dessine pas parce que ce n'est pas un objet posé sur le terrain — les
 * quatre grenades (leurs poses `deployed` sont des LANCERS, déjà comptés par `grenades[]`, les
 * compter deux fois ferait deux colonnes d'un même geste) et les trois capacités `grapple`,
 * `thruster`, `repulsor` (l'appareil lui-même, pas un objet ; le grappin a sa colonne de
 * TRACTIONS, les deux autres n'ont aucun canal d'activation mesuré).
 *
 * Réutiliser cette table plutôt qu'en écrire une deuxième est la règle des ≤ 2 copies : le jour
 * où une famille change de statut, elle change ici aussi, sans qu'on y pense.
 */
function isDeployableFamily(family: string): boolean {
  return PLACEMENT_RENDER[family] != null
}

/**
 * buildEquipmentUsage — l'agrégation complète, en une passe par calque.
 *
 * `scoreboard` peut manquer (chargement, titre sans tableau des scores) : les joueurs existent
 * alors tous sans camp, et le tableau les range sous « sans équipe ». Aucun camp n'est deviné.
 */
export function buildEquipmentUsage(
  doc: ReplayDocumentReady,
  scoreboard: MatchScoreboardRow[] | undefined,
): EquipmentUsage {
  // SEULS LES JOUEURS QUE LE FILM A VUS VIVRE ont une ligne (cf. `teamsOf`) : la table des
  // compteurs se borne aux mêmes, sinon un geste attribué à une entrée de roster sans piste
  // disparaîtrait de l'écran SANS entrer dans les orphelins — et la somme mentirait.
  const players = buildPlayers(doc, scoreboard ?? []).filter((p) => p.lives.length > 0)
  const ownerOfSlot = indexBySlot(players, (p) => p)
  const tallies = new Map<string, EquipmentUsageTally>()
  const unattributed = emptyTally()

  /** Le compteur d'un joueur, créé à la demande ; celui des gestes orphelins sans lui. */
  const tallyOf = (owner: ReplayPlayer | undefined): EquipmentUsageTally => {
    if (!owner) return unattributed
    let t = tallies.get(owner.xuid)
    if (!t) {
      t = emptyTally()
      tallies.set(owner.xuid, t)
    }
    return t
  }
  /** Le compteur du propriétaire d'une VIE (clé : le slot de piste). */
  const tallyOfSlot = (slot: number): EquipmentUsageTally => tallyOf(ownerOfSlot.get(slot))
  /**
   * Le compteur d'un joueur désigné par son INDEX DE FILM — l'autre clé du document, et la
   * seule que les lancers de grenade portent vraiment (cf. en-tête). Le roster fait le pont,
   * comme `ReplayTeams` le fait déjà pour le badge de lancer.
   */
  const byFilmIndex = new Map(
    doc.roster.map((entry) => [entry.filmIndex, players.find((p) => p.xuid === entry.xuid)]),
  )
  const tallyOfFilmIndex = (index: number): EquipmentUsageTally => tallyOf(byFilmIndex.get(index))

  for (const line of doc.grappleLines) tallyOfSlot(line.slot).grapplePulls += 1

  for (const e of doc.equipmentEpisodes) {
    // Une famille hors des deux mesurées n'a ni libellé ni sens établi : elle n'entre pas.
    if (!(EPISODE_FAMILIES as readonly string[]).includes(e.fam)) continue
    addEpisode(tallyOfSlot(e.slot).episodes, e.fam, frameToMs(e.t1 - e.t0, doc), e.k ?? 0)
  }

  for (const p of doc.equipmentPlacements) {
    // `owner` -1 = aucun bipède contemporain à moins de 3 m : la pose est réelle, son auteur
    // ne l'est pas. Elle rejoint les gestes sans propriétaire plutôt qu'une ligne au hasard.
    const t = p.owner >= 0 ? tallyOfSlot(p.owner) : unattributed
    if (placementIsDeployedObject(p) && isDeployableFamily(p.family)) bump(t.deployed, p.family)
    else if (placementIsDroppedPower(p)) bump(t.dropped, p.family)
  }

  // LE LANCER PORTE SON AUTEUR : `i`, l'index de joueur du film — jamais `slot` (cf. en-tête).
  for (const g of doc.grenades) bump(tallyOfFilmIndex(g.i).grenades, g.rank)

  const byTeam = teamsOf(players, tallies)
  const byPlayer = byTeam.flatMap((g) => g.players)
  const powerupPickups = countPowerupPickups(doc)
  const powerupPickupsTotal = Object.values(powerupPickups).reduce((a, b) => a + b, 0)

  return {
    byPlayer,
    byTeam,
    columns: columnsOf(byPlayer),
    coverage: coverageOf(doc),
    powerupPickups,
    powerupPickupsTotal,
    unattributed,
    hasData: powerupPickupsTotal > 0 || byPlayer.some((r) => !tallyIsEmpty(r)),
  }
}

/**
 * teamsOf range les joueurs par camp et somme chaque camp.
 *
 * Le filtre « au moins une vie » a déjà été appliqué par l'appelant (cf. `buildEquipmentUsage`) :
 * une entrée de roster sans aucune vie n'a été mesurée sur aucun canal, et une ligne de zéros la
 * ferait passer pour quelqu'un qui n'a rien fait. L'ordre est celui du roster du film (stable).
 */
function teamsOf(
  players: ReplayPlayer[],
  tallies: Map<string, EquipmentUsageTally>,
): EquipmentUsageTeam[] {
  return groupByTeam(players).map((group) => {
    const total = emptyTally()
    const rows = group.players.map((p) => {
      const tally = tallies.get(p.xuid) ?? emptyTally()
      mergeTally(total, tally)
      return {
        ...tally,
        xuid: p.xuid,
        name: displayPlayerName(playerName(p), p.xuid),
        side: group.side,
      }
    })
    return { side: group.side, players: rows, total }
  })
}

/**
 * columnsOf retient les colonnes qu'au moins un joueur justifie, dans un ordre ÉCRIT.
 *
 * L'ordre vient des tables de référence (`PLACEMENT_RENDER`, `PLACEMENT_DROPPED_FAMILIES`, le
 * rang du catalogue de grenades) et non de l'ordre de rencontre dans le film : deux matchs
 * doivent présenter leurs colonnes dans le même ordre, sans quoi les comparer devient un
 * exercice de relecture.
 */
function columnsOf(rows: EquipmentUsageRow[]): EquipmentUsageColumns {
  const used = (pick: (r: EquipmentUsageRow) => Record<string, number> | Record<number, number>) => {
    const keys = new Set<string>()
    for (const r of rows) for (const [k, n] of Object.entries(pick(r))) if (n > 0) keys.add(k)
    return keys
  }
  const deployedUsed = used((r) => r.deployed)
  const droppedUsed = used((r) => r.dropped)
  const grenadesUsed = used((r) => r.grenades)
  return {
    grapple: rows.some((r) => r.grapplePulls > 0),
    episodes: EPISODE_FAMILIES.filter((f) => rows.some((r) => (r.episodes[f]?.count ?? 0) > 0)),
    deployed: Object.keys(PLACEMENT_RENDER).filter((f) => deployedUsed.has(f)),
    dropped: PLACEMENT_DROPPED_FAMILIES.filter((f) => droppedUsed.has(f)),
    grenades: [...grenadesUsed].map(Number).sort((a, b) => a - b),
  }
}

/**
 * countPowerupPickups croise les occupations achevées avec les socles de BONUS.
 *
 * `padPickups[].pad` est un INDEX dans `weaponPads` (ordre stable garanti côté build) : un
 * index hors bornes ne compte pour rien plutôt que pour un socle voisin. La famille se lit par
 * la table écrite `PAD_EQUIPMENT_FAMILIES` (`padEquipmentFamilyOf`), jamais par un test de
 * préfixe — un socle d'ARME n'est pas un socle de bonus.
 *
 * UNE OCCUPATION EST UN VIDAGE, PAS UN RAMASSAGE NOMMÉ. Le compte dit combien de fois un socle
 * s'est vidé ; il ne dit pas qui, et rien ici ne doit le laisser croire.
 */
function countPowerupPickups(doc: ReplayDocumentReady): Record<string, number> {
  const out: Record<string, number> = {}
  for (const pick of doc.padPickups) {
    const pad = doc.weaponPads[pick.pad]
    if (!pad) continue
    const family: PadEquipmentFamilyKey | null = padEquipmentFamilyOf(pad.weapon)
    if (family) bump(out, family)
  }
  return out
}

/** tallyTotal — le nombre de gestes d'un compteur, tous canaux confondus. */
export function tallyTotal(t: EquipmentUsageTally): number {
  const sum = (m: Record<string, number> | Record<number, number>) =>
    Object.values(m).reduce((a: number, b: number) => a + b, 0)
  const episodes = Object.values(t.episodes).reduce((a, e) => a + e.count, 0)
  return t.grapplePulls + episodes + sum(t.deployed) + sum(t.dropped) + sum(t.grenades)
}

/** coverageOf recopie les dénominateurs du document — aucun n'est recalculé ni deviné. */
function coverageOf(doc: ReplayDocumentReady): EquipmentUsageCoverage {
  const cov = doc.coverage
  const equip = cov?.equipment
  const grapple = cov?.grapple
  return {
    tracksTotal: equip?.tracksTotal ?? 0,
    episodeLives: {
      [EQUIP_FAMILY_CAMO]: equip?.camoLives ?? 0,
      [EQUIP_FAMILY_OVERSHIELD]: equip?.overshieldLives ?? 0,
    },
    grapplePulls: grapple?.pulls ?? 0,
    grapplePullLives: grapple?.pullLives ?? 0,
    placementsByFamilyOrigin: cov?.placements?.byFamilyOrigin ?? {},
    powerupPads: cov?.groundWeapons?.powerupPads ?? 0,
    killsRead: equip?.killsRead ?? false,
  }
}
