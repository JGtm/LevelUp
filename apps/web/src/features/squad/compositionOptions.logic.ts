/**
 * compositionOptions.logic — les coéquipiers proposés au sélecteur de composition hors de la page
 * Escouade, et l'annuaire qui traduit leurs gamertags en xuid.
 *
 * LA LISTE EST CELLE DE L'ESCOUADE : la page Escouade ne propose, en « Coéquipiers fréquents »,
 * que les AMIS DÉCLARÉS du joueur (filtre serveur `filterTopRowsToFriends`), les profils suivis
 * formant le groupe « Joueurs configurés » du `GamertagCombobox`. Les pages qui n'ont pas sa
 * réponse lourde reconstruisent la même liste à partir de lectures légères :
 *   - les amis déclarés (`/players/{slug}/friends`), en gamertags ;
 *   - les joueurs croisés (`/pages/career/encounters`), qui portent xuid et nombre de matchs ;
 *   - les profils suivis, les escouades enregistrées et les groupes, qui portent un xuid.
 *
 * AUCUN BOT : les joueurs croisés comptent les bots des parties contre l'IA (xuid `bid(N.0)`),
 * et ils occupent la majorité des cinquante premières places. Un bot ne se choisit pas comme
 * coéquipier.
 *
 * SANS AMI DÉCLARÉ (liste vide ou illisible), la liste retombe sur les coéquipiers croisés, bots
 * exclus : le sélecteur reste utilisable, ordonné par nombre de matchs joués ensemble.
 *
 * Module pur : aucune dépendance React, testable sans DOM.
 */
import type { EncounterDTO, TeammateOption } from '@/lib/api/types'

/** Préfixe du xuid d'un bot — même règle que `analysis.IsBot` côté serveur. */
const PREFIXE_XUID_BOT = 'bid('

/** estUnBot — le xuid désigne-t-il un bot ? */
export function estUnBot(xuid: string): boolean {
  return xuid.startsWith(PREFIXE_XUID_BOT)
}

/** Un joueur connu par son gamertag ET son xuid (profil suivi, membre d'escouade ou de groupe). */
export interface JoueurIdentifie {
  gamertag?: string | null
  xuid: string
}

/** Les lectures dont la liste est tirée. */
export interface SourcesDeComposition {
  /** Joueurs croisés surtout comme coéquipiers (`teammates` de la réponse). */
  coequipiers: readonly EncounterDTO[]
  /** Joueurs croisés surtout comme adversaires (`enemies`) : l'annuaire seulement. */
  adversaires: readonly EncounterDTO[]
  /** Gamertags des amis déclarés ; vide quand aucun n'est déclaré ou que la liste est illisible. */
  amis: readonly string[]
  /** Profils suivis, membres d'escouades enregistrées et de groupes. */
  identifies: readonly JoueurIdentifie[]
  /** Le joueur consulté, exclu de la liste. */
  joueur: { gamertag: string; xuid: string }
}

const cle = (gamertag: string) => gamertag.trim().toLowerCase()

/** Le tri des propositions : plus de matchs ensemble d'abord, puis l'ordre alphabétique. */
function parFrequence(a: TeammateOption, b: TeammateOption): number {
  return b.encounter_count - a.encounter_count || a.gamertag.localeCompare(b.gamertag)
}

/** estLeJoueur — la ligne désigne-t-elle le joueur consulté ? */
function estLeJoueur(sources: SourcesDeComposition, gamertag: string, xuid: string): boolean {
  const { joueur } = sources
  return (joueur.xuid !== '' && xuid === joueur.xuid) || cle(gamertag) === cle(joueur.gamertag)
}

/**
 * annuaireDeComposition — chaque joueur humain connu, avec son xuid : de quoi traduire une
 * composition en xuids. Les identités explicites (profils, escouades, groupes) priment sur les
 * rencontres ; une rencontre apporte son nombre de matchs joués ensemble.
 */
export function annuaireDeComposition(sources: SourcesDeComposition): TeammateOption[] {
  const parCle = new Map<string, TeammateOption>()
  for (const r of [...sources.coequipiers, ...sources.adversaires]) {
    if (!r.gamertag || !r.xuid || estUnBot(r.xuid) || estLeJoueur(sources, r.gamertag, r.xuid)) continue
    parCle.set(cle(r.gamertag), { gamertag: r.gamertag, xuid: r.xuid, encounter_count: r.as_teammate })
  }
  for (const j of sources.identifies) {
    const gamertag = j.gamertag ?? ''
    if (!gamertag || !j.xuid || estUnBot(j.xuid) || estLeJoueur(sources, gamertag, j.xuid)) continue
    const deja = parCle.get(cle(gamertag))
    parCle.set(cle(gamertag), { gamertag, xuid: j.xuid, encounter_count: deja?.encounter_count ?? 0 })
  }
  return [...parCle.values()]
}

/**
 * coequipiersProposes — la liste du sélecteur : les amis déclarés que l'annuaire sait traduire,
 * ou, sans ami déclaré, les coéquipiers croisés. Jamais un bot, jamais le joueur consulté.
 */
export function coequipiersProposes(
  sources: SourcesDeComposition,
  annuaire: readonly TeammateOption[],
): TeammateOption[] {
  const parCle = new Map(annuaire.map((o) => [cle(o.gamertag), o]))
  const { amis } = sources
  if (amis.length > 0) {
    const vus = new Set<string>()
    const proposes: TeammateOption[] = []
    for (const ami of amis) {
      const connu = parCle.get(cle(ami))
      if (!connu || vus.has(cle(ami))) continue
      vus.add(cle(ami))
      proposes.push(connu)
    }
    return proposes.sort(parFrequence)
  }
  const croises = new Set(sources.coequipiers.map((r) => cle(r.gamertag ?? '')))
  return annuaire.filter((o) => croises.has(cle(o.gamertag))).sort(parFrequence)
}
