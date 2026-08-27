/**
 * weaponBurstSpecs.ts — LES ARMES DONT UN TIR DU FILM SE JOUE EN RAFALE, ET AVEC QUEL ÉCART.
 *
 * LE RETOUR QUI OUVRE CE FICHIER (utilisateur, 2026-08-27, mot pour mot) : « L'arme MA40 est
 * une auto donc les sons joués doivent chacun tirer 3 balles, un fire event pour cette arme
 * c'est une rafale de 3 balles (comportement normalement par défaut pour les armes
 * automatiques) ».
 *
 * POURQUOI LA LECTURE ET PAS L'ASSET (décision D7 du plan des retours du 2026-08-27). Re-cuire
 * un `.wav` en salve de trois coups serait plus simple — et ce serait perdre deux choses. La
 * première est la VARIATION INTERNE : le moteur du jeu tire volume et hauteur dans les
 * fourchettes RANGED de la bank À CHAQUE BALLE, jamais à chaque pression de détente ; une salve
 * cuite figerait les trois coups au même échantillon, au même gain, à la même hauteur, et cela
 * s'entend immédiatement (c'est exactement le défaut que `weaponSoundLogic.ts` existe pour
 * corriger). La seconde est la DOCTRINE DES ASSETS : un fichier livré est UN COUP reconstitué
 * selon la sémantique prouvée des banks (`replaySound.ts`, RECETTE_SONS_ARMES) — y coller une
 * salve ferait mentir la seule règle qui rend ces fichiers vérifiables. Le lecteur programme
 * donc N départs du MÊME fichier sur UNE seule voix logique, chacun avec SON tirage
 * (`ReplayAudioPlayer.playBurst`). Aucun asset n'est touché, aucun n'est régénéré.
 *
 * D'OÙ VIENT L'ÉCART DE 33 ms — MESURÉ, PAS CHOISI. Instrument :
 * `apps/go-api/internal/analysis/replay/weapon_burst_research_test.go` (test de recherche sous
 * garde `WEAPON_CADENCE_CORPUS`, aucun code de production), passé sur les 38 artefacts locaux,
 * 39 749 tirs publiés. La MA40 y porte 19 388 tirs, découpés en 1 417 salves d'au moins quatre
 * tirs (18 828 tirs) ; l'intervalle MOYEN par salve a pour médiane 100,0 ms (p10 80,0 ms,
 * p90 133,3 ms). Trois balles à tenir dans cet intervalle donnent 100,0 / 3 = 33,3 ms, arrondi
 * à 33.
 *
 * LA CENSURE QU'IL A FALLU CONTOURNER, parce qu'elle change le chiffre : le film date ses
 * événements À L'IMAGE, et une image dure 100 ms. Les écarts bruts entre deux tirs ne valent
 * donc que 0, 100, 200 ms... — la médiane brute de la MA40 tombe pile sur 100 ms avec 14 % de
 * zéros, et diviser une telle valeur par trois aurait donné une précision imaginaire. C'est
 * l'intervalle MOYEN PAR SALVE qui résout sous l'image (les arrondis des bornes se divisent par
 * le nombre d'intervalles), et c'est lui qui commande ici.
 *
 * CE QUE LA MESURE DIT D'AUTRE, ET QU'IL FAUT SAVOIR AVANT D'ÉTENDRE : à 100 ms d'intervalle
 * médian, les tirs MA40 que le film publie arrivent déjà à la cadence RÉELLE de l'arme. Un tir
 * du film n'est donc pas, dans la donnée, un paquet de trois balles ; la rafale est une
 * décision de MISE EN SCÈNE, prise par l'utilisateur, et c'est son oreille qui la valide au
 * gate d'écoute (« les votes priment sur tout critère », RECETTE_SONS_ARMES §5).
 *
 * LES CANDIDATS À L'EXTENSION SONT NOMMÉS ICI ET N'Y SONT PAS INSCRITS — ils attendent ce même
 * vote, une ligne de table chacun, jamais un nouveau mécanisme :
 *  - `hinf_ma5k_avenger` (automatique, asset à 1 transitoire, salves à 88,9 ms médian) ;
 *  - `hinf_pulse_carbine` (salve native de trois dans le jeu, mais asset à 1 transitoire
 *    seulement — 112,5 ms médian : c'est le candidat le plus légitime après la MA40) ;
 *  - `hinf_vk78_commando` (automatique, asset à 1 transitoire, 181,8 ms médian) ;
 *  - `hinf_needler` (automatique, asset à 1 transitoire, 100,0 ms médian).
 *
 * DEUX ARMES SONT EXCLUES PAR LA MESURE, PAS PAR L'OREILLE, et il faut le dire ici pour que
 * personne ne les ajoute par symétrie : l'asset du `hinf_br75` porte DÉJÀ TROIS transitoires
 * d'attaque — sa salve native est dans le fichier, et lui appliquer une rafale de trois ferait
 * entendre NEUF coups. Le `hinf_sentinel_beam` en porte trois aussi, mais c'est un faisceau
 * tenu : il n'a pas de coup à répéter.
 */

/** Ce qu'une rafale vaut pour une arme : combien de balles, et à quel écart. */
export interface WeaponBurstSpec {
  coups: number
  ecartMs: number
}

/**
 * PÉRIMÈTRE INITIAL : la MA40 SEULE (décision D7). Toute autre arme se joue exactement comme
 * avant — un tir, un départ. Une clé ajoutée ici sans vote d'écoute serait une décision de
 * mise en scène prise à la place de l'utilisateur.
 *
 * Le garde-rail voisin (`weaponBurstSpecs.guard.test.ts`) tient les invariants : une clé qui ne
 * serait pas un stem livré ferait une rafale muette, et une rafale plus longue que le plafond
 * de coupe serait tronquée sans que rien ne le dise.
 */
export const WEAPON_BURST_SPECS: Readonly<Record<string, WeaponBurstSpec>> = {
  hinf_ma40_ar: { coups: 3, ecartMs: 33 },
}
