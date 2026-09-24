package fallback

// registre_replay_vehicules.go — les replis du calque des VEHICULES poses par les retours du rejeu
// du 2026-09-23 (`internal/games/halo_infinite/film/replay/`, et le DECOR DE CARTE decide a la
// requete par `internal/service/replay_vehicle_scenery*.go`, lot M7). Un fichier a part parce que
// `registre_replay_identites.go`, qui porte les replis vehicules plus anciens, est a trois lignes
// du seuil de 500 du depot.

// dateRetoursRejeu : le jour du plan `PLAN_RETOURS_REJEU_2026-09-23` (lot M4a).
const dateRetoursRejeu = "2026-09-23"

// dateDecorM7 : le jour du lot M7 (decor de carte hors de la zone jouable) et de sa reprise apres
// revue adverse (constat RR-M7-03 : ses deux replis n etaient pas inscrits ici).
const dateDecorM7 = "2026-09-24"

// siteDecorM7 : la regle du decor, service de rejeu (hors du decodeur : le registre accepte les
// sites hors du paquet film, comme `internal/sync/killcollector/`).
const siteDecorM7 = "internal/service/replay_vehicle_scenery_rule.go"

// COMPTE DES DEUX REPLIS DU DECOR : ils vivent A LA REQUETE, pas a la cuisson, et `coverage.fallbacks`
// est une donnee de CUISSON que la requete ne reecrit pas. Leur compte est publie dans le calque de
// requete lui-meme, par vie et par document : `vehicleScenery.hidden[].reason = below_played_floor`
// et `vehicleScenery.zoneUnknown`. Le compteur est donc BRANCHE au site, sans `fb.Declenche`.

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
	{
		Nom:  "repli_decor_sous_le_sol_foule_du_match",
		Fait: "une vie de vehicule POSEE (cinq conditions de L1.3) est-elle sous la zone jouable, en hauteur ?",
		Mecanisme: "la pose est a plus de 1,5 m (`sceneryFloorToleranceM`) sous le SOL FOULE du match : " +
			"la plus basse altitude ou une piste de joueur reste au moins 1 s (`sceneryStandMinMs`) dans " +
			"une bande de 0,3 m (`sceneryStandMaxDZ`) ; une chute ne s y tient jamais. Raison publiee " +
			"`below_played_floor`. DEPEND DU MATCH, et c est ecrit : des joueurs qui se tiennent plus bas " +
			"que la pose la rendent (il y a alors du jeu a sa hauteur) ; un match ecourte peut lire un sol " +
			"plus haut (1,42 m sur deux matchs de Lattice de 767 et 882 points)",
		// MESURE DU 2026-09-24 sur les 107 documents du parc (schema 69) : le sol foule d une carte
		// varie de 0,34 m au plus d un match a l autre (hors deux matchs ecourtes), la ou la plus
		// basse position publiee variait de 33,5 m (Streets) ; 260 vies en jeu ramenees a une pose
		// seule : la plus basse est 0,10 m sous le sol foule (socle de Launch Site) ; le seul decor
		// sous le sol est a 3,03 m (Wasp de Goliath, d8b13ec2, un seul match de Goliath au parc).
		// AUCUNE reference versionnee ne porte l altitude de la matiere : le sidecar du fond n en
		// publie qu une par carte (`playLevelZ`), et les ancres d objectifs ne bornent pas le sol
		// (negatif mesure : sur Banished Narrows les joueurs se tiennent 3,29 m sous la plus basse).
		Condition: CondCarteAbsenteDuCatalogue,
		Ordre:     OrdreSansLecture,
		Sites: []Site{{
			Fichier: siteDecorM7,
			Ancre:   "case floorKnown && s.Z < floor-sceneryFloorToleranceM:",
		}},
		DatePose:        dateDecorM7,
		CibleRetrait:    "le fond de carte publie porte l altitude de sa matiere (cuisson cmd/mapfond-build), lue par la regle a la place du match",
		CritereRetrait:  "sur le parc, le test de hauteur lu dans la reference de carte masque exactement les decors que ce repli masque (Wasp de Goliath) et 0 vie en jeu ramenee a une pose seule, sans lire aucune position de joueur",
		CompteurBranche: true,
	},
	{
		Nom:  "repli_decor_carte_sans_zone_affiche",
		Fait: "une vie de vehicule POSEE (cinq conditions de L1.3) est-elle hors de la zone jouable, sur une carte sans zone connue ?",
		Mecanisme: "la carte n a pas de fond publie (ou son image est absente ou illisible) : aucune " +
			"vie n est masquee, zone publiee `unknown`, chaque candidate comptee dans `zoneUnknown`",
		// MESURE DU 2026-09-24 : 0 candidate en zone inconnue au parc (les trois cartes qui portent
		// une pose seule — Starboard, Goliath, Behemoth — ont un fond publie). Le repli est le choix
		// sur de la decision utilisateur : ne jamais masquer un vehicule qu on ne sait pas situer.
		Condition: CondCarteAbsenteDuCatalogue,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: siteDecorM7,
			Ancre:   "// repli_decor_carte_sans_zone_affiche : aucune zone, rien de masque, compte.",
		}},
		DatePose:        dateDecorM7,
		CibleRetrait:    "toute carte jouee a un fond publie (campagne de cuisson des fonds, cmd/mapfond-build)",
		CritereRetrait:  "sur le parc de production, `vehicleScenery.zoneUnknown` vaut 0 sur tous les documents servis pendant un trimestre",
		CompteurBranche: true,
	},
}
