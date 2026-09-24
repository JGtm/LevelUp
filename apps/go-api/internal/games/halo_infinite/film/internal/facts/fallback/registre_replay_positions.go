package fallback

// registre_replay_positions.go — les replis de la PUBLICATION DES POSITIONS (`replay/`), poses au
// lot M1 des retours du rejeu (2026-09-23, `.ai/V7.5/PLAN_RETOURS_REJEU_2026-09-23.md` §4.4).
//
// # POURQUOI DES REPLIS, ET PAS DES REGLES
//
// Les balayages ANCRES bit a bit (`ScanBipedPositionsForBand`, `ScanVehicleCreationsForBand`)
// acceptent parfois un faux en-tete : la meme suite de bits revient sur des cartes aux
// quantifications differentes (annexe `retours_rejeu_2026-09-23/RAPPORT_positions_limbe.md` §1.2).
// La lecture qui les refuserait — valider le record ENTIER a l ancre (`TryDeltaAt` + record suivant
// valide, generation coherente) — N EST PAS PORTEE : c est l option 2 du rapport, et elle monte la
// revision de grammaire. En attendant, deux gardes MESUREES ecartent ce qui ne peut pas etre une
// position ; elles sont nommees, comptees dans la couverture du document, et leur critere de
// retrait est la livraison de cette lecture.
//
// Les deux REGLES GRAMMATICALES du meme lot (aucune position de corps avant sa creation) ne sont
// PAS ici : elles lisent le film (le record de creation), elles ne s y substituent pas.

// dateM1RetoursRejeu : le jour du lot M1 des retours du rejeu, qui pose les deux entrees.
const dateM1RetoursRejeu = "2026-09-23"

// cibleOption2Positions : la lecture qui retire les deux replis de ce fichier.
const cibleOption2Positions = "option 2 du rapport positions_limbe (porte grammaticale au decodage : " +
	"record entier valide a l ancre, record suivant valide, generation coherente), decouverte 19 du " +
	"plan des retours du rejeu du 2026-09-23"

var registreReplayPositions = []Repli{
	{
		Nom:  "repli_position_hors_emprise_ecartee",
		Fait: "quelles positions decodees sont publiees : points de trace, echantillons et naissances de vehicule",
		Mecanisme: "une position hors de l emprise jouee du film (p1..p99 plus 12 etendues centrales par axe, " +
			"la garde de `boundsOf`) ET ISOLEE (aucune chaine d instants voisins, au plus 0,5 s et 60 m/s, ne la " +
			"relie a une position dans l emprise du meme slot : une chute reelle reste publiee) est ecartee " +
			"avant toute publication et comptee",
		Condition: CondLectureNonPortee,
		Ordre:     OrdreApresLecture,
		Sites: []Site{
			{Fichier: pkgReplay + "positions_porte.go", Ancre: "fb.DeclencheN(fallback.NomPositionHorsEmpriseEcartee, cov.HorsEmprise)"},
			{Fichier: pkgReplay + "positions_porte_vehicules.go", Ancre: "fb.DeclencheN(fallback.NomPositionHorsEmpriseEcartee, ecartes)"},
		},
		DatePose:     dateM1RetoursRejeu,
		CibleRetrait: cibleOption2Positions,
		CritereRetrait: "coverage.tracks.horsEmprise, coverage.vehicles.echantillonsHorsEmprise et " +
			"coverage.vehicles.spawnsHorsEmprise a 0 sur le parc une fois la porte grammaticale livree",
		CompteurBranche: true,
	},
	{
		Nom:  "repli_echantillon_vehicule_au_travers_d_un_silence_ecarte",
		Fait: "quels echantillons de position d un vehicule sont publies",
		Mecanisme: "dans une vie de vehicule, de deux sejours de replication qui se contredisent au travers " +
			"d un silence de plus de `lifeGapUS` (deplacement de plus de 2 m), celui que son AUTRE voisin " +
			"contredit est ecarte (la naissance de la vie est la voisine gauche du premier sejour), a soutien " +
			"egal le plus court ; jamais un sejour de plus de `vehicleSejourAberrantMax` (3) echantillons. Sans " +
			"preuve, rien n est ecarte et le refus se compte (coverage.vehicles.silencesNonTranches)",
		Condition: CondLectureNonPortee,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "positions_porte_vehicules.go",
			Ancre:   "fb.DeclencheN(fallback.NomEchantillonVehiculeAuTraversDUnSilenceEcarte, b.ecartes)",
		}},
		DatePose:     dateM1RetoursRejeu,
		CibleRetrait: cibleOption2Positions,
		// LA MESURE QUI LE FONDE (2026-09-23, 111 documents) : 565 silences de plus de 5 s entre
		// deux echantillons d une meme vie, 21 avec un deplacement de plus de 2 m, et les 21
		// touchent un echantillon aberrant ; aucun deplacement reel pendant un silence.
		CritereRetrait:  "coverage.vehicles.echantillonsAuTraversDUnSilence a 0 sur le parc une fois la porte grammaticale livree",
		CompteurBranche: true,
	},
}
