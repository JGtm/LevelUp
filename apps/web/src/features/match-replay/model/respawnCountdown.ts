/**
 * respawnCountdown.ts — LE COMPTE À REBOURS D'UN LIEU QUI RÉAPPROVISIONNE, et l'ORDRE DE SES
 * DEUX SOURCES.
 *
 * POURQUOI CE FICHIER EXISTE (CLAUDE.md n°6, et la demande du lot 5.8). Deux calques posent la
 * MÊME question au document : « ce lieu est vide, dans combien de temps son objet revient-il ? ».
 * Les SOCLES D'ARME la posent depuis le 2026-08-27 (`weaponPadTime.padRespawnAt`) ; les
 * EMPLACEMENTS DE NAISSANCE DE VÉHICULE la posent depuis le schéma 63 (`vehicleCycleTime`). La
 * règle de réponse est la même à la virgule près, et deux copies auraient divergé au premier
 * réglage — l'écart étant invisible (un chiffre plausible reste plausible).
 *
 * L'ORDRE DES SOURCES EST LA RÈGLE, ET IL VIENT D'UNE MESURE (D3, retour utilisateur du
 * 2026-08-27 : « compteur pas toujours visible ») : LE REJEU CONNAÎT LA SUITE DU FILM, donc une
 * apparition DÉJÀ VUE l'emporte toujours sur ce qu'un cycle PRÉDIT. Le cycle ne sert que pour le
 * DERNIER trou, celui qu'aucune apparition ne referme.
 *
 * SANS SOURCE, RIEN — ni zéro, ni tiret : un tiret suggérerait qu'on sait (verdict du calque des
 * socles, repris tel quel).
 *
 * CE FICHIER NE CONNAÎT NI SOCLE NI VÉHICULE, et c'est ce qui le rend partageable : il ne lit
 * que trois nombres. La question « ce lieu est-il vide ? » et celle de « depuis quand » restent
 * chez l'appelant, qui seul sait ce qu'occuper veut dire pour son objet.
 */

/**
 * Le compte à rebours ET SA PROVENANCE.
 *
 * `measured` distingue deux chiffres que rien ne séparait à l'écran : la prochaine apparition
 * VUE DANS LE FILM (exacte) et celle que le CYCLE prédit (une moyenne, qui peut tomber à côté).
 * La carte n'en fait rien — un lieu vide reste un lieu vide — mais l'infobulle le dit, et c'est
 * là que la réserve doit se lire (le « ≈ » des libellés).
 */
export interface Respawn {
  seconds: number
  measured: boolean
}

/** Les trois nombres dont la règle a besoin, et rien de plus. */
export interface RespawnSources {
  /**
   * L'image de la PROCHAINE apparition que le film montre, ou `null` quand aucune ne referme le
   * trou. Une valeur qui ne serait pas dans le futur de l'image courante est ignorée : le compte
   * mesuré est toujours positif, jamais un nombre négatif déguisé.
   */
  nextSpawn: number | null
  /**
   * L'image DEPUIS LAQUELLE le lieu est vide, et elle doit être DATÉE par la source. Pour un
   * socle c'est `tHigh`, la borne HAUTE de la disparition (partir de `tLow` avancerait la
   * prédiction d'un intervalle que le film ne date pas) ; pour un véhicule c'est `tEnd`, la seule
   * fin que le film écrive. `null` : rien à prédire depuis.
   */
  emptySince: number | null
  /** La médiane du cycle, en secondes, quand il est ÉTABLI. `undefined` sinon. */
  medianS: number | undefined
}

/**
 * respawnAt — les secondes restantes avant la réapparition, et leur provenance, ou `null`.
 *
 * L'APPELANT A DÉJÀ TRANCHÉ QUE LE LIEU EST VIDE : cette fonction ne le vérifie pas, elle ne
 * saurait pas comment. Elle refuse en revanche une durée d'image nulle ou négative (un document
 * sans cadence) et un compte épuisé — dans les deux cas il n'y a rien à annoncer.
 */
export function respawnAt(frame: number, frameMs: number, src: RespawnSources): Respawn | null {
  if (!(frameMs > 0)) return null
  if (src.nextSpawn !== null && src.nextSpawn > frame) {
    return { seconds: ((src.nextSpawn - frame) * frameMs) / 1000, measured: true }
  }
  if (src.emptySince === null || !(src.medianS !== undefined && src.medianS > 0)) return null
  const left = src.medianS - ((frame - src.emptySince) * frameMs) / 1000
  return left > 0 ? { seconds: left, measured: false } : null
}
