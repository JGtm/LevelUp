package fallback

// registre_replay_vehicules.go — les replis du calque des VEHICULES poses par les retours du rejeu
// du 2026-09-23 (`internal/games/halo_infinite/film/replay/`). Un fichier a part parce que
// `registre_replay_identites.go`, qui porte les replis vehicules plus anciens, est a trois lignes
// du seuil de 500 du depot.

// dateRetoursRejeu : le jour du plan `PLAN_RETOURS_REJEU_2026-09-23` (lot M4a).
const dateRetoursRejeu = "2026-09-23"

var registreReplayVehicules = []Repli{
	{
		Nom:  "repli_tourelle_porteur_voisin_de_slot",
		Fait: "le vehicule qui PORTE une tourelle (piece montee `ti=40` sans aucun echantillon de position)",
		Mecanisme: "la premiere vie du slot +1 puis +2 de la tourelle, de la famille de chassis que " +
			"la table des pieces montees attend pour elle, dont la fenetre d affichage recouvre la " +
			"sienne, NEE AVEC elle (meme instant a 1 frame, meme point a 1 m quand les deux naissances " +
			"sont lues ; un voisin refuse est compte `turretCarrierBirthMismatch`) ; aucune candidate = " +
			"tourelle sans porteur, publiee et non dessinee",
		// LA RELATION EXISTE DANS LE JEU (la tourelle est attachee a son chassis) ; la sonde P1-S2
		// du 2026-09-23 n a trouve aucun lien lisible (`object-parent-state` ne rattache aucun
		// objet au vehicule sur les deux films sondes). C est une DETTE de lecteur, pas un
		// silence du film prouve : la condition le dit.
		Condition: CondLectureNonPortee,
		Ordre:     OrdreSansLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "vehicle_turrets.go",
			Ancre:   "fb.Declenche(fallback.NomTourellePorteurVoisinDeSlot)",
		}},
		DatePose:     dateRetoursRejeu,
		CibleRetrait: "lot M4b des retours du rejeu (PLAN_RETOURS_REJEU_2026-09-23) ou tout lot qui lit le parent d une piece montee dans le film",
		// MESURE DU 2026-09-23 sur les 111 documents du parc (instrument `enfants.mjs` de
		// l annexe, relu par le lot) : LAAG `dd7f9102` 45 vies, voisin `slot+1` Warthog 44/45 ;
		// roquettes `bcfb852f` 16 vies, 15/16 ; tourelles du Falcon `1a043c29` puis `f4c45d71`,
		// chaine `[1a043c29][f4c45d71][Falcon]` ; Wraith `[233c877d][001b33fc][Wraith]`. Temoin
		// independant (revue adverse du lot, meme parc) : 141 pieces posees sur 141 nees au meme
		// point (0,0 m) et au meme instant (<= 1 frame) que leur porteur.
		CritereRetrait:  "le porteur de chaque tourelle est LU dans le film sur les 8 builds, et la lecture s accorde au voisin de slot sur le parc (0 desaccord) — alors le voisinage devient un temoin, jamais une decision",
		CompteurBranche: true,
	},
}
