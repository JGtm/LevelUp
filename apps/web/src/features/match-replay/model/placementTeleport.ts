/**
 * placementTeleport — LES USAGES DU TRANSLOCATEUR QUANTIQUE, lus dans l'ÉVÉNEMENT DU FILM.
 *
 * CE FICHIER A CHANGÉ DE FONDATION LE 2026-09-03, et c'est le point à comprendre avant d'y
 * toucher. Il portait jusque-là DEUX lectures indirectes, toutes deux abandonnées :
 *
 *  - une HEURISTIQUE SPATIALE (`riftTeleports`) qui cherchait, dans les pistes, un joueur se
 *    déplaçant de plus de 12 m en moins de 3 images. Elle rendait 4 détections sur 39 films,
 *    et surtout : tout seuil de distance INVENTE une règle — une téléportation peut faire
 *    50 cm (mesuré : un saut de 3,24 m sur `1b2d9e08`, invisible à tous les seuils). Décision
 *    utilisateur du 2026-09-03 (« je ne veux pas de choix ou de règle arbitraire, je veux de
 *    la lecture fiable ») : SUPPRIMÉE, avec ses tests et ses imports (CLAUDE.md n°7) ;
 *  - le `spent` du calque d'équipement (`spentTranslocations`), qui date la FIN de
 *    l'équipement et non son usage : la mesure des modes de fin donne jusqu'à 16,5 s de
 *    retard, et trois `spent` du corpus n'ont aucun usage connu. Il ne subsiste QUE comme
 *    repli daté des artefacts antérieurs au schéma 38 (cf. `spentTranslocations`).
 *
 * CE QUI LES REMPLACE : `translocations[]` (schéma 38). Chaque usage émet un événement type
 * 117 dans la bobine — précision 18/18, rappel 8/8 sur 5 films (rapport R1 §4) — et sa charge
 * porte les DEUX positions du saut, lues dans l'exécutable et validées à 0,00-0,26 m
 * (rapport R6 §1). Plus aucune dérivation : l'événement EST la mesure.
 *
 * LA SÉMANTIQUE EST UN VA-ET-VIENT, et elle gouverne tout ce que le rendu affirme : la balise
 * ÉCHANGE sa position avec le joueur à chaque usage — l'arrivée d'un saut est le départ du
 * précédent, à 0,09 m près sur la mesure. Après un échange, la faille est donc au point de
 * DÉPART (`fx/fy`). Où elle se trouve AVANT le premier échange n'est lisible d'AUCUN canal
 * (négatif mesuré sur trois : R1 §1-3 et R6 §1.4) : on ne dessine donc rien avant lui — voir
 * `riftStations.ts`, qui porte cette partie.
 */
import type { ReplayDocumentReady } from './replayNormalize'
import type { XY } from './replayLogic'

/**
 * Rang du translocateur quantique dans la palette de capacités du film — FAMILLE A SEULEMENT.
 * Le rang est établi côté serveur (`internal/analysis/filmdec/translocateur_test.go`) et le
 * document le confirme : `abilityLabels` associe 11 à « translocateur quantique ». C'est le
 * REPLI des documents sans table de libellés ; partout ailleurs, `translocatorRanks` lit la
 * table — un film famille B (rangs 19-22) rendrait ce littéral muet (bug du 2026-09-02 :
 * la reconnaissance était fausse partout hors famille A, silencieusement).
 */
const RANG_TRANSLOCATEUR = 11

/**
 * translocatorRanks — les rangs de palette qui NOMMENT le translocateur dans CE document.
 *
 * La table `abilityLabels` est la seule correspondance rang→objet que l'artefact publie, et
 * elle suit la famille de palette du film (A : 11 ; B : un rang de 19-22) — la lire est donc
 * le seul chemin qui couvre les deux familles sans deviner. La reconnaissance se fait sur la
 * RACINE du mot (`transloc`), présente dans les deux locales publiées (« Translocateur
 * quantique » / « Quantum Translocator ») : comparer un libellé entier serait cassé à la
 * première retouche de traduction. Sans table, ou sans rang reconnu : le repli famille A.
 *
 * ELLE NE SERT PLUS À RECONNAÎTRE UN USAGE — l'événement 117 s'en charge — mais à lire le
 * calque d'ÉQUIPEMENT : la FIN de l'équipement (`riftStations.ts`) et le repli des artefacts
 * antérieurs au schéma 38 (`spentTranslocations`) y passent tous deux par un rang.
 */
export function translocatorRanks(
  labels: ReplayDocumentReady['abilityLabels'],
): ReadonlySet<number> {
  const ranks = new Set<number>()
  for (const [rank, label] of Object.entries(labels ?? {})) {
    const text = `${label?.fr ?? ''} ${label?.en ?? ''}`.toLowerCase()
    if (text.includes('transloc')) ranks.add(Number(rank))
  }
  if (ranks.size === 0) ranks.add(RANG_TRANSLOCATEUR)
  return ranks
}

/**
 * Un moment de translocation réduit à ce que la FICHE consomme : quelle vie, quelle image.
 * `TeleportLink` en est la version SITUÉE (le lien sur la carte a besoin des deux bouts) ;
 * une translocation dont la charge n'a pas été lue n'a que ce couple-là.
 */
export interface TranslocationMoment {
  slot: number
  frame: number
}

/** Un va-et-vient MESURÉ : la vie, l'image, le point quitté et le point atteint. */
export interface TeleportLink extends TranslocationMoment {
  /** Position quittée — c'est là que la balise se retrouve après l'échange. */
  from: XY
  /** Position atteinte par le joueur. */
  to: XY
}

/** Une téléportation du document, telle que le calque la publie. */
type Translocation = ReplayDocumentReady['translocations'][number]

/**
 * planarJump — les deux bouts du saut d'une translocation, ou `null` quand la charge n'a pas
 * été lue.
 *
 * LES SIX COORDONNÉES SONT SOLIDAIRES au contrat (armées ensemble ou pas du tout,
 * cf. `document_translocations.go`) ; le plan n'en consomme que quatre, et on les vérifie
 * toutes les quatre plutôt que de faire confiance à la solidarité — `0` est une valeur
 * légitime, `undefined` ne l'est pas, et un `?? 0` dessinerait un saut vers l'origine du monde.
 */
function planarJump(t: Translocation): { from: XY; to: XY } | null {
  const { fx, fy, tx, ty } = t
  if (fx === undefined || fy === undefined || tx === undefined || ty === undefined) return null
  return { from: { x: fx, y: fy }, to: { x: tx, y: ty } }
}

/**
 * translocationMoments — l'INSTANT de chaque usage publié, avec ou sans position.
 *
 * C'est la source de l'éclat de fiche : une translocation sans va-et-vient lisible reste un
 * usage MESURÉ (l'événement l'atteste), seule sa géométrie manque.
 */
export function translocationMoments(
  translocations: readonly Translocation[],
): TranslocationMoment[] {
  return translocations.map((t) => ({ slot: t.slot, frame: t.t }))
}

/**
 * translocationLinks — les va-et-vient SITUÉS, dans l'ordre des images.
 *
 * Une translocation sans positions n'en produit AUCUN : on date le geste sur la fiche, on ne
 * trace aucun lien sur la carte. `coverage.translocations.positioned` dit combien en portent.
 */
export function translocationLinks(
  translocations: readonly Translocation[],
): TeleportLink[] {
  const out: TeleportLink[] = []
  for (const t of translocations) {
    const jump = planarJump(t)
    if (!jump) continue
    out.push({ slot: t.slot, frame: t.t, ...jump })
  }
  out.sort((a, b) => a.frame - b.frame)
  return out
}

/**
 * hasTranslocationLayer — CE DOCUMENT A-T-IL ÉTÉ LU POUR LE CALQUE DES TÉLÉPORTATIONS ?
 *
 * DEUX PREUVES, ET LA PREMIÈRE EST LA SEULE QUI COUVRE LE SILENCE. La couverture est posée
 * SANS CONDITION par le constructeur du schéma 38 (`build.go` :
 * `doc.Coverage.Translocations = &trCov`, y compris à zéro événement) : sa présence sépare
 * exactement « le calque a tourné, il n'a rien trouvé » de « l'artefact est antérieur au
 * schéma 38 ». Une téléportation publiée est l'autre preuve — un artefact antérieur ne peut
 * pas en porter — et elle sert de filet si un producteur servait le calque sans sa couverture.
 */
export function hasTranslocationLayer(doc: ReplayDocumentReady): boolean {
  return doc.coverage?.translocations !== undefined || doc.translocations.length > 0
}

/**
 * identityIsUnknown — le rang PRÉCÉDENT annoncé par cette émission est-il hors d'usage ?
 *
 * `gap > 0` = n émissions manquent ENCORE juste avant elle, après récupération (schéma 38,
 * décision D3). `from` n'est alors ni vrai ni faux : il est INCONNU, et tout consommateur doit
 * le traiter comme tel. Le cas mesuré est le `spent` de JGtm sur `1b2d9e08`, qui porte
 * `from = 4` (le grappin) parce que sa PRISE du translocateur a été manquée : une seule
 * émission perdue aveugle le filtre.
 *
 * FOYER UNIQUE DE LA RÈGLE (CLAUDE.md n°6) : `spentTranslocations` et `riftStations` la lisent
 * ici, jamais chacun la sienne. Un artefact antérieur au schéma 38 ne publie pas `gap` — la
 * porte est alors ouverte, ce qui est exactement le comportement d'avant.
 */
export function identityIsUnknown(
  change: ReplayDocumentReady['equipmentChanges'][number],
): boolean {
  return (change.gap ?? 0) > 0
}

/**
 * spentTranslocations — REPLI DES ARTEFACTS ANTÉRIEURS AU SCHÉMA 38, et rien d'autre.
 *
 * KILL-SWITCH DATÉ (modèle `platform/duckdb/shared_reader_legacy.go`) :
 *   - bascule du défaut : 2026-09-03 — depuis le schéma 38, l'éclat vient de
 *     `translocations[]` (l'ÉVÉNEMENT), et cette fonction n'est plus jamais appelée sur un
 *     artefact qui porte ce calque (cf. `teleportMoments`) ;
 *   - retrait cible : 2026-12-01 ;
 *   - critère mesurable : plus aucun artefact servi sans `coverage.translocations`
 *     (`hasTranslocationLayer` vrai partout), c'est-à-dire re-cuisson complète du parc.
 *
 * CE QU'IL VAUT, DIT SANS FARD. Le `spent` date la FIN de l'équipement, pas l'usage : jusqu'à
 * 16,5 s de retard mesuré, et trois `spent` du corpus n'ont aucun usage connu. Il reste
 * néanmoins le SEUL signal daté d'un artefact ancien — le supprimer éteindrait l'éclat sur
 * tout le parc en production tant que la re-cuisson n'a pas eu lieu.
 *
 * IL S'ABSTIENT SUR UNE IDENTITÉ INCONNUE (`identityIsUnknown`) : un éclat AFFIRME un geste,
 * et on n'affirme pas sur un `from` que le témoin de compteur dit hors d'usage.
 */
export function spentTranslocations(
  changes: ReplayDocumentReady['equipmentChanges'],
  ranks: ReadonlySet<number>,
): TranslocationMoment[] {
  const out: TranslocationMoment[] = []
  for (const c of changes) {
    if (c.kind !== 'spent' || identityIsUnknown(c)) continue
    if (ranks.has(c.from)) out.push({ slot: c.slot, frame: c.t })
  }
  return out
}

/**
 * teleportMoments — LES INSTANTS que la fiche fait scintiller, par le meilleur canal
 * disponible pour CE document.
 *
 * UN SEUL CANAL À LA FOIS, ET C'EST LA RÈGLE : mélanger l'événement et le `spent` ferait
 * scintiller deux fois le même usage (l'un à l'instant vrai, l'autre jusqu'à 16,5 s plus
 * tard). Le calque publié gagne dès qu'il existe, le repli ne sert que là où il n'existe pas.
 */
export function teleportMoments(doc: ReplayDocumentReady): TranslocationMoment[] {
  if (hasTranslocationLayer(doc)) return translocationMoments(doc.translocations)
  return spentTranslocations(doc.equipmentChanges, translocatorRanks(doc.abilityLabels))
}

/**
 * lastTeleportAge — l'âge (en frames) du passage le plus RÉCENT d'un SLOT à cette image,
 * ou -1 s'il n'en a aucun d'advenu.
 *
 * La recherche porte sur le slot — une vie — comme tout report de lecture : un passage ne
 * peut pas survivre à son porteur, donc une fiche morte n'en portera jamais l'éclat. Un
 * passage À VENIR ne compte pas : l'éclat date un événement advenu, jamais annoncé.
 */
export function lastTeleportAge(
  teleports: readonly TranslocationMoment[],
  slot: number,
  frame: number,
): number {
  let age = -1
  for (const t of teleports) {
    if (t.slot !== slot || t.frame > frame) continue
    const a = frame - t.frame
    if (age === -1 || a < age) age = a
  }
  return age
}
