package replayverite

// seuils.go — CHAQUE SEUIL DU BANC, NOMME, AVEC LA MESURE QUI L'A FIXE.
//
// Mesures : 19 temoins de config/replay_corpus.toml, artefacts cuits a `b452391f7` (J6-ter,
// schema 76), lus sans decodage le 2026-09-30 (conception §7). Aucune constante n'est recopiee du
// producteur (la garde « une seule ecriture » de `film/replay/emprise_jouee_test.go`) : chaque
// valeur est justifiee ici, independamment.

const (
	// vitesseSautMaxMPS : au-dela, deux points consecutifs d'une piste sont un SAUT (V-1).
	// 100 m/s est le plafond du filtre de vitesse de la grammaire : au-dela, la grammaire rejette
	// et, apres trois rejets, se reancre — un saut publie au-dela est donc un reancrage ou un faux.
	// Un Spartan reste sous 35 m/s, un vehicule sous 26,1 m/s. Essai : 85 sauts sur 19 temoins,
	// dont 80 sur des portes de carte (portes_de_carte.go) et 1 sur une translocation.
	vitesseSautMaxMPS = 100.0

	// toleranceTranslocationImages : un saut a ±2 images d'une translocation publiee est la
	// translocation elle-meme (±200 ms, l'exemption du filtre de la grammaire au pas de 100 ms).
	toleranceTranslocationImages = 2

	// rayonPorteM : un saut dont le depart ou l'arrivee est a moins de 3 m du centre d'une porte
	// de carte mesuree est exempte. 3 m couvre les cellules de 2 m ou les arrivees se groupent
	// (Dredge : 38 arrivees en deux cellules adjacentes).
	rayonPorteM = 3.0

	// margeEmpriseObjetsM : un objet pose (arme au sol, equipement, presentoir) ou un echantillon de
	// vehicule a plus de 10 m de l'enveloppe publiee des pistes est HORS EMPRISE (V-2). Mesure : a
	// 10 m, les objets de bord des cartes Arena rentrent tous (0797ce72 4 -> 0, 111fa685 9 -> 0,
	// bf15f7ab 9 -> 0, 51ebbc0f 3 -> 0, 60ae07c4 5 -> 0) ; ce qui reste est a plus de 100 m (armes
	// `spawned` jamais ramassees, echantillons de vehicule des builds anciens).
	margeEmpriseObjetsM = 10.0

	// toleranceVieImages : une action datee a ±5 images d'une vie de son slot est DANS la vie
	// (V-3, V-5). 500 ms absorbent la datation au pas d'image et la latence de publication ; essai :
	// tirs hors vie 279 a tolerance 0, 203 a 5, sur 42 735.
	toleranceVieImages = 5

	// rayonTrajetM : un passager a plus de 3 m de son vehicule a la meme image est LOIN (V-6).
	// 3 m = le rayon d'ancrage d'un trajet mesure par le producteur (dernier rayon « gratuit »).
	// Essai : 6 echantillons au-dela sur 19 temoins, 0 au-dela de 10 m.
	rayonTrajetM = 3.0

	// toleranceEchantillonImages : un point de piste apparie a un echantillon de vehicule s'il est
	// a ±1 image (les deux flux ne sont pas echantillonnes aux memes images).
	toleranceEchantillonImages = 1

	// toleranceMortImages : une mort du statborg a l'image t appartient a une fin de vie du meme
	// joueur en [t-2, t+2] (V-8). Mesure sur 1 344 morts statborg : 982 a l'image meme de la fin
	// de vie, 304 a +1, 2 a +2 ; au-dela la distribution est eparse (56 morts).
	toleranceMortImages = 2

	// margeFinDeFilmImages : une piste qui finit a moins de 2 images de la fin du film finit AVEC
	// le film, pas par une mort (O-V2).
	margeFinDeFilmImages = 2
)
