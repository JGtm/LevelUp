/**
 * abilityChargeLogic.ts — LES CHARGES RESTANTES de la capacité portée (schéma 38 enrichi,
 * lot P6).
 *
 * LA SOURCE est `doc.abilityCharges` : une entrée PLATE par lecture — (t, slot, family,
 * charges) — le compteur transmis AU CHANGEMENT par le composant i56 du film (quartet haut,
 * rapport R11), attribué côté serveur par le rang de capacité de la MÊME VIE. La sémantique
 * du canal décide tout ce que ce fichier affirme :
 *
 *  - RIEN N'EST TRANSMIS AU RAMASSAGE (masque à 0 = « le moteur pose la valeur pleine ») :
 *    la première lecture est ce qui reste APRÈS le premier usage. Avant elle, la vignette
 *    dit « PLEIN » QUALITATIF, sans chiffre (décision utilisateur du 04/09) — un maximum
 *    déduit serait un chiffre inventé.
 *  - UNE BAISSE PEUT VALOIR PLUSIEURS USAGES : on affiche la LECTURE, jamais un compte
 *    d'usages dérivé.
 *  - SEULES LES FAMILLES DÉCLARÉES MESURÉES entrent dans le calque (grappin, propulseur —
 *    le répulseur n'arme jamais i56, négatif mesuré). Un équipement d'une famille que la
 *    table de ce fichier ne nomme pas n'affiche NI chiffre NI « plein » : le canal ne porte
 *    pas cette famille, rien n'est affirmé.
 *
 * TROIS JOINTURES, TOUTES TROIS OBLIGATOIRES (P6.1) :
 *
 *  1. LA VIE. Le slot de bipède est réattribué à chaque réapparition : la lecture retenue
 *     appartient à la vie qui COUVRE l'image courante (patron `isAliveAt`, le P0 de la
 *     revue P4 — une table `Map(slot → vie)` écrase les vies d'un même slot). Une lecture
 *     d'une vie précédente ne colle jamais à la suivante : on réapparaît plein.
 *  2. LE DERNIER CHANGEMENT D'ÉQUIPEMENT de la vie. Un nouveau ramassage repart à « plein » :
 *     une lecture antérieure au `taken` décrit l'équipement PRÉCÉDENT et ne colle pas au
 *     nouveau. Une lecture datée au même instant que le changement est ambiguë — traitée
 *     comme antérieure, jamais affichée sur le nouvel équipement.
 *  3. LA FAMILLE. La lecture la plus récente doit être de la MÊME famille que l'équipement
 *     que la vignette affiche (les charges d'un grappin ne collent pas à un propulseur).
 *     Si elle est d'une AUTRE famille, le canal contredit la vignette (prise manquée par le
 *     calque des changements, attribution divergente) : rien n'est affirmé — ni le chiffre
 *     de l'autre famille, ni un « plein » démenti par la lecture.
 *
 * LA CORRESPONDANCE RANG→FAMILLE N'EST PAS PUBLIÉE par l'artefact : `abilityLabels` est la
 * seule table rang→objet servie (même constat que `translocatorRanks`, placementTeleport.ts),
 * et la reconnaissance se fait donc sur la RACINE du libellé, présente dans au moins une des
 * deux locales publiées — jamais un rang en dur (la palette B déplace les rangs, mêmes
 * libellés). La table est ÉCRITE et FERMÉE, même règle que `ABILITY_IMPULSE_SOUND_STEMS` :
 * une famille absente reste muette, jamais la valeur d'une voisine.
 *
 * Tout ce fichier est PUR : aucun React, aucun réseau — même partage que `equippedLogic.ts`
 * et `changeRefine.ts`, testé sans rien monter.
 */
import { isAliveAt } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'

/**
 * LES FAMILLES MESURÉES PAR LE CANAL DES CHARGES, reconnues par la racine de leur libellé.
 *
 * Les clés sont les identifiants de famille publiés par le document (`abilityCharges[].family`,
 * vocabulaire du manifeste du titre) ; les racines couvrent les deux locales de la palette
 * (« grappin » / « Grappleshot », « propulseur » / « Thruster »). N'y entrent que les familles
 * que le titre déclare mesurées ([ability_charges] du manifeste) : ajouter une racine sans
 * canal mesuré ferait affirmer « plein » sur un équipement dont le film ne dit rien.
 */
const CHARGE_FAMILY_STEMS: Record<string, readonly string[]> = {
  grapple: ['grapp'],
  thruster: ['thrust', 'propuls'],
}

/**
 * hasAbilityChargeLayer — le document porte-t-il RÉELLEMENT le canal des charges ?
 *
 * Même construction que `hasTranslocationLayer` (placementTeleport.ts), avec un étage de
 * plus parce que la couverture des charges le publie : `coverage.abilityCharges` n'est posé
 * par le serveur que si le BALAYAGE A TOURNÉ (garde `Scanned`, build.go), et son
 * `componentAbsent` dit que le film ne déclare pas le composant i56. Trois cas, trois
 * verdicts :
 *  - couverture absente (artefact pré-38, ou balayage en échec) : rien n'a été lu — « rien
 *    transmis » n'y signifie PAS « plein », on n'affirme rien ; une lecture publiée sert de
 *    filet (un artefact antérieur ne peut pas en porter) ;
 *  - couverture posée avec `componentAbsent` : le film ne transmet pas ce canal — rien
 *    d'affirmé non plus ;
 *  - couverture posée, composant présent : le canal a parlé — zéro lecture est alors la
 *    sémantique « rien consommé », et « plein » devient une lecture, pas une invention.
 */
export function hasAbilityChargeLayer(doc: ReplayDocumentReady): boolean {
  const cov = doc.coverage?.abilityCharges
  if (cov !== undefined) return cov.componentAbsent !== true
  return doc.abilityCharges.length > 0
}

/**
 * measuredChargeFamilyOf — la famille MESURÉE de l'équipement que la vignette affiche, ou
 * null quand le canal des charges ne la porte pas (camouflage, surbouclier, répulseur,
 * rang hors table…). Null = rien d'affirmé, jamais un « plein » par défaut.
 */
export function measuredChargeFamilyOf(
  labels: ReplayDocumentReady['abilityLabels'],
  rank: number,
): string | null {
  const label = labels?.[String(rank)]
  if (!label) return null
  const text = `${label.fr ?? ''} ${label.en ?? ''}`.toLowerCase()
  for (const [family, stems] of Object.entries(CHARGE_FAMILY_STEMS)) {
    if (stems.some((stem) => text.includes(stem))) return family
  }
  return null
}

/**
 * Ce que la vignette affiche des charges :
 *  - `count` — la lecture la plus récente de la vie et de l'équipement courants, avec l'âge
 *    de la lecture (en frames) pour l'estompage, comme toute cellule de la rangée ;
 *  - `full` — « plein » qualitatif : aucune lecture depuis le ramassage, et c'est la
 *    sémantique du canal (rien transmis = rien consommé), pas un défaut d'affichage.
 */
export type AbilityChargeDisplay =
  | { kind: 'count'; charges: number; age: number }
  | { kind: 'full' }

/**
 * abilityChargesAt — les charges à afficher pour la capacité de rang `rank` que la vignette
 * du `slot` montre à l'image `frame`, ou null quand rien ne peut être affirmé (famille non
 * mesurée, aucune vie couvrante, lecture la plus récente d'une autre famille).
 */
export function abilityChargesAt(
  doc: ReplayDocumentReady,
  slot: number,
  frame: number,
  rank: number,
): AbilityChargeDisplay | null {
  // SANS le calque, rien n'est affirmé — surtout pas « plein » : sur un artefact pré-38 ou
  // un film sans composant i56, personne n'a lu le canal (constat P0 de la revue P6).
  if (!hasAbilityChargeLayer(doc)) return null
  const family = measuredChargeFamilyOf(doc.abilityLabels, rank)
  if (family === null) return null
  // La VIE qui couvre l'image — jamais `Map(slot → vie)`, qui écraserait les vies d'un même
  // slot (P0 de la revue P4 ; patron de `buildFireMarks`).
  const life = doc.tracks.find((tr) => tr.slot === slot && isAliveAt(tr, frame))
  if (!life) return null
  // Le DERNIER changement d'équipement de la vie : la borne basse des lectures recevables.
  let lastChange = -1
  for (const c of doc.equipmentChanges) {
    if (c.slot !== slot || c.t > frame || !isAliveAt(life, c.t)) continue
    if (c.t > lastChange) lastChange = c.t
  }
  // La lecture la plus récente de la FENÊTRE (même vie, après le dernier changement,
  // jamais dans le futur) — toutes familles confondues : c'est elle qu'on confronte à la
  // famille affichée, une lecture plus ancienne ne doit pas lui survivre.
  let latest: ReplayDocumentReady['abilityCharges'][number] | null = null
  for (const r of doc.abilityCharges) {
    if (r.slot !== slot || r.t > frame || r.t <= lastChange || !isAliveAt(life, r.t)) continue
    if (!latest || r.t > latest.t) latest = r
  }
  if (!latest) return { kind: 'full' }
  if (latest.family !== family) return null
  return { kind: 'count', charges: latest.charges, age: frame - latest.t }
}
