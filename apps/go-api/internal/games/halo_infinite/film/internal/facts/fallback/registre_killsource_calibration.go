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
		Mecanisme: "aucune source lue : un balayage de 1 a 3 retient la largeur qui maximise le nombre de records de bipede lus sans desynchronisation ; un profil plat laisse l invariant 1",
		Condition: CondNonResolu,
		Ordre:     OrdreSansLecture,
		Sites: []Site{{
			Fichier: pkgKillsource + "calibrate.go",
			Ancre:   "res.Profil.Mouvement.Traversal.IndexW = res.PoigneeIndexW",
		}},
		DatePose: dateM3,
		// POSE APRES UNE MESURE, ET APRES UNE ERREUR. Le lot 3.4.1 a rendu les largeurs d AXE a
		// la carte et demote leur balayage en oracle ; il avait emporte avec elles cette
		// largeur-ci, qui n a AUCUNE source lue. `replay-equiv` l a vu : sur `a521164d`,
		// `ScanAbilityImpulses` rend 1 impulsion a la largeur 2 et ZERO a 1 comme a 3, et
		// l artefact perdait 14 octets. Le balayage redevient son pourvoyeur, et l entree le DIT.
		// LA GRANDEUR N EST PAS CELLE DE L INDEX DE PLAGE (`DAT_144632be0`, portee par
		// `WorldObject.IndexW` depuis le catalogue) : c est le mot que
		// `consumePositionHandleTail` lit derriere le bit de poignee (`FUN_1406d3140`).
		CibleRetrait:    "lot qui LIRA la largeur du mot de poignee chez l ecrivain (bitlen du compte de poignees, `FUN_1406d3140`) ou la fera entrer au profil par une cle que le film ecrit",
		CritereRetrait:  "la valeur vient d une lecture sur les 8 builds ; le balayage devient oracle comme celui des largeurs d axe, et `calibration.Desaccords` la compte",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_parametre_etat_record_infere",
		Fait:      "le `param_4` du moteur (largeur de trois composants) applique a tout le film",
		Mecanisme: "aucune source lue : un balayage de 0 a 5 retient la valeur qui maximise la CROISSANCE DES SLOTS ; les paquets dont les records ne se localisent pas sont ignores",
		Condition: CondNonResolu,
		Ordre:     OrdreSansLecture,
		Sites: []Site{{
			Fichier: pkgKillsource + "calibrate.go",
			Ancre:   "best, bestN = r, n",
		}},
		DatePose:        dateM3,
		CibleRetrait:    "lot qui trouvera la source LUE de `param_4` (registre ECS par composant, ou table du build)",
		CritereRetrait:  "la valeur vient d une lecture ; le balayage devient oracle comme celui des largeurs, ou disparait",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
}
