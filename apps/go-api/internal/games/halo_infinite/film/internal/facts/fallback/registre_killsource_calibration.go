package fallback

// registre_killsource_calibration.go — CE QUE LA CALIBRATION DECIDE FAUTE DE SOURCE LUE.
//
// # POURQUOI CES DEUX ENTREES ONT LEUR FICHIER (lot 3.4.1, 2026-09-17)
//
// SCISSION DE TAILLE, ET DE SENS. De taille d abord : `registre_killsource.go` repassait les
// 500 lignes du depot en accueillant la seconde. De sens ensuite, et c est ce qui fait qu elle
// tient : ces deux entrees repondent a la MEME question, et c est la derniere que le lot 3.4
// laisse ouverte — QUELLES VALEURS le decodeur decide encore par balayage, maintenant que les
// largeurs d axe viennent de la carte.
//
// # CE QUI LES REUNIT, ET CE QUI LES SEPARE DES AUTRES
//
// Un repli du registre decide un FAIT a la place de la lecture du film. Ces deux-la decident
// une LARGEUR, c est-a-dire la facon de lire — et une largeur fausse ne se voit pas : elle
// deplace tout ce qui suit dans le record, en silence. C est le mode de defaillance le plus
// cher du decodeur (24 bits de deficit sur i0 ont produit des comptes de grenade a 255), et
// c est pourquoi les deux portent une cible NOMMEE plutot qu un « a voir ».
//
// # CE QUE LE LOT 3.4.1 A FERME, ET CE QU IL A ROUVERT
//
// FERME : les largeurs d AXE et la largeur d INDEX DE PLAGE viennent du catalogue de la carte
// (`profile.MapQuantEntry.PrecisionAbsolue`, loi verifiee 79 cartes sur 79), et le balayage qui
// les devinait est devenu un ORACLE qui COMPTE ses desaccords. Les entrees
// `repli_largeur_absolue_uniforme`, `repli_calibration_paquet_exclu` et
// `repli_calibration_paquet_non_localise` sont sorties du registre avec lui.
//
// ROUVERT, ET PAR UNE MESURE : ce meme lot avait emporte la largeur du MOT DE POIGNEE, qui
// n a aucune source lue. `replay-equiv` l a vu sur `a521164d` (une impulsion publiee perdue),
// et elle revient ici sous son propre nom — pas sous celui de l index de plage, avec lequel
// elle avait ete confondue (D5 (3.4.1), fermee).

var registreKillsourceCalibration = []Repli{
	{
		Nom:       "repli_largeur_mot_de_poignee_inferee",
		Fait:      "la largeur du mot de poignee lu dans la queue de position d un record (`Traversal.IndexW`)",
		Mecanisme: "aucune source lue : les trois largeurs candidates sont scorees AU TRIPLET LU de la carte (le monde que la production decode) sur le nombre de records de bipede lus sans desynchronisation ; la meilleure n est retenue QUE si elle domine la mediane d un facteur 2, sinon l invariant 1 tient",
		Condition: CondNonResolu,
		Ordre:     OrdreSansLecture,
		Sites: []Site{{
			Fichier: pkgKillsource + "calibrate.go",
			Ancre:   "res.Profil.Mouvement.Traversal.IndexW = retenu",
		}},
		DatePose: dateM3,
		// POSE APRES UNE MESURE, RE-MOTIVEE APRES UNE SECONDE (2026-09-17, voie (a1) du pilote).
		//
		// PREMIERE MESURE : le lot 3.4.1 a rendu les largeurs d AXE a la carte et demote leur
		// balayage en oracle ; il avait emporte avec elles cette largeur-ci, qui n a AUCUNE
		// source lue, et elle retombait a l invariant sans que rien ne le dise.
		//
		// SECONDE MESURE, ET C EST ELLE QUI DONNE SA FORME A CETTE ENTREE : le balayage qui
		// « decidait » scorait ses candidats sous une largeur d axe UNIFORME que la production
		// avait cesse de lire, et son critere est AVEUGLE a la grandeur sur les films instruits
		// — `a521164d` rend 272 / 272 / 272 au triplet lu, `64e8adfa` 61 / 61 / 61. La valeur
		// posee sortait donc d un ex aequo tranche par un tri instable, et elle voyageait
		// jusqu au rejeu (`profilDeBalayageDeLaCuisson`), ou elle deplacait `abilityImpulses`,
		// `grappleReads.stats` et les tapis sur 14 films sur 20. Le balayage ne decide plus ce
		// qu il ne separe pas : l invariant tient, et CETTE ENTREE EST CE QUI LE DIT.
		//
		// LA GRANDEUR N EST PAS CELLE DE L INDEX DE PLAGE (`DAT_144632be0`, portee par
		// `WorldObject.IndexW` depuis le catalogue) : c est le mot que
		// `consumePositionHandleTail` lit derriere le bit de poignee (`FUN_1406d3140`).
		CibleRetrait:    "lot qui RELEVERA la largeur du mot de poignee CHEZ L ECRIVAIN — le bitlen du compte de poignees de `FUN_1406d3140` — et la fera entrer au profil par une cle que le film ecrit, comme la loi des largeurs d axe y est entree",
		CritereRetrait:  "`calibration.PoigneeDiscriminee` cesse d etre le juge : la valeur vient d une lecture sur les 8 builds du corpus, le balayage devient ORACLE (il compte ses desaccords avec elle, il n ecrit plus), et `replay-equiv` ne bouge sur AUCUNE des trois etapes derriere i0 (`abilityImpulses`, `grappleReads.stats`, `pads`) sur les 20 films",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
}
