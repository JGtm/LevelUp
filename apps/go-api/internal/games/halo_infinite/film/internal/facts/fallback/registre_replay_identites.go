package fallback

// registre_replay_identites.go — les replis des calques IDENTITÉ, VIES et VÉHICULE
// (`internal/games/halo_infinite/film/replay/`). Les replis des PLACES et des PRÉSENCES du roster
// vivent dans `registre_replay_places.go` depuis le lot M2.3 ; ceux des calques DRAPEAU, ZONE,
// CRÂNE et BOMBE dans `registre_replay_objectifs.go` depuis le lot J5.5 (2026-09-27).

var registreReplayIdentites = []Repli{
	{
		Nom:       "repli_identite_premier_occupant_du_siege",
		Fait:      "quel joueur occupait un slot a un instant donne, pour un lecteur qui demande un xuid",
		Mecanisme: "aucune vie nommee ne couvre l'instant : le PREMIER occupant nomme du siege est servi (sauf slot ambigu, ou le registre se tait)",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "identity_registry_pont.go",
			// L'ANCRE EST LA GARDE, PAS LA LECTURE : citer le champ que ce site lit ferait
			// rougir `archlint/no_identity_bridge_outside_registry_test.go`, qui interdit ce
			// nom hors du registre d'identite — commentaires exclus, chaines comprises.
			Ancre: "if r.SlotAmbiguous[slot] {",
		}, {
			Fichier: pkgReplay + "identity_registry.go",
			Ancre:   "r.fb.Declenche(fallback.NomIdentitePremierOccupantDuSiege)",
		}},
		DatePose: dateAudit0E,
		// CIBLE CORRIGEE AU LOT 1.9.14 (2026-09-15) : ce repli sert `XUIDAt` sur un siege dont
		// aucune vie nommee ne couvre l'instant. Ni 1.9.13 (les vies finissent a une mort ecrite)
		// ni 1.9.14 (le siege d'une fiche) ne le retirent — aucun des deux ne change ce que le
		// pont sait d'un slot a un instant. La cible est le lot qui fermera ce trou-la.
		CibleRetrait:    "le lot qui donnera une vie nommee a tout instant d'un siege occupe (suite du registre d'identite, M2 ou M3)",
		CritereRetrait:  "0 recours au premier occupant sur les 8 builds : toute demande tombe dans une vie couvrante",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_nom_piste_par_le_pont",
		Fait:      "le xuid d'une piste publiee qui ne porte pas de nom lu",
		Mecanisme: "le pont slot -> joueur est interroge, et son verdict devient le nom de la piste",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "published_tracks.go",
			Ancre:   "return strconv.FormatUint(x, 10)",
		}, {
			Fichier: pkgReplay + "build_inventaire.go",
			Ancre:   "a.opt.Fallbacks.DeclencheN(fallback.NomNomPisteParLePont, pistesNommeesParLePont(a.doc.Tracks, a.reg.PontEpure()))",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.6 (le registre d'identite prend la table du film comme lien direct) : le compte doit tomber avec la couverture du lien direct",
		CritereRetrait:  "coverage.identity.coverage.filmTable a 100 % et 0 piste nommee par le pont sur les 8 builds",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_identite_piste_meilleur_recouvrement",
		Fait:      "a quelle vie nommee appartient une piste quand PLUSIEURS vies du meme slot la recouvrent",
		Mecanisme: "la vie dont le recouvrement temporel avec la piste est le plus grand gagne, sans aucun seuil minimal",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "identity.go",
			Ancre:   "bestOverlap, bestXUID = ov, l.xuid",
		}},
		DatePose: dateAudit0E,
		// CIBLE REECRITE LE 2026-09-16 (revue de jalon M1). Elle nommait le lot 1.9.13, fusionne
		// le 2026-09-15 : le lot a bien FERMÉ le défaut (les vies se découpent aux morts écrites,
		// donc ne se recouvrent plus) et le critère est TENU — 0 déclenchement sur les 8 goldens
		// d'assemblage, bloc « REPLIS DECLENCHES », alors que `nameTracksByLives` y tourne sur
		// 99 à 144 pistes par film. L'entrée ne SORT pourtant pas encore, pour deux raisons
		// écrites :
		//
		//	(1) D14 (d) retire au JALON SUIVANT ce dont le compte est nul À LA CLÔTURE d'un jalon,
		//	    et sur LE CORPUS : les 8 goldens sont 8 films, le corpus gate en porte 14 et le
		//	    parc 1 351. Le gate de clôture de M1 tranche ;
		//	(2) le « site » n'est pas du code mort séparable. L'ancre est la SÉLECTION par
		//	    recouvrement maximal, qui nomme CHAQUE piste ; ce qui est un repli, c'est
		//	    l'arbitrage à partir du deuxième candidat, et il n'a pas de branche à lui.
		//	    Retirer l'entrée retirerait donc le COMPTEUR — le seul instrument qui prouve que
		//	    le zéro dure — en laissant l'arbitrage anonyme dans le code, ce que D14 (a)
		//	    interdit. Le retrait propre est une CONVERSION (abstention explicite dès deux
		//	    candidats), et elle appartient à M2.
		CibleRetrait: "M2 : abstention explicite des deux candidats, puis retrait de l'entree (le compte est nul depuis le 2026-09-15)",
		// Sans seuil, un recouvrement d'une seule frame l'emporte sur l'absence : le compte dit
		// combien de pistes sont nommées par un arbitrage plutôt que par une lecture.
		CritereRetrait:  "0 piste arbitree par recouvrement sur les 8 builds : TENU le 2026-09-16 (0/8, goldens d'assemblage) ; reste a confirmer au corpus gate",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_vie_coupee_au_trou_de_replication",
		Fait:      "ou finit une vie de joueur",
		Mecanisme: "un trou de positions de plus de lifeGapUS (5 s) ferme la vie courante et en ouvre une neuve — pour le seul joueur dont le film n'ecrit AUCUNE mort (site 1), ou pour TOUTES les vies quand la table d'index des joueurs est vide (site 2)",
		// CONVERTI AU LOT 1.9.13 (2026-09-15), `film_muet / devant_la_lecture` ->
		// `film_muet / apres_lecture`. La decoupe LIT desormais ce que le film ecrit — mort
		// appariee au fil, record de creation de bipede dans le trou, frontiere de manche — et le
		// trou de replication devient une LACUNE de la MEME vie (`coverage.tracks.gaps`). Le seuil
		// ne decide plus QUE la ou les trois lectures se taisent ensemble, c'est-a-dire pour un
		// joueur que rien ne tue de tout le film : rien ne borne alors ses vies.
		Condition: CondFilmMuet,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "lives_decoupe.go",
			Ancre:   "in.fb.Declenche(fallback.NomVieCoupeeAuTrouDeReplication)",
		}, {
			// SECOND SITE, INSCRIT A LA REVUE DE JALON M1 (2026-09-15, lentille D13). Il
			// existait depuis le lot 1.9.13 et le registre ne le citait pas : la publication
			// des traces se rabat sur l ECHAFAUDAGE (`buildLifeSpans`) quand le registre
			// d identite ne rend AUCUNE vie — c est-a-dire pour TOUTES les vies du film, et
			// non pour celles d un joueur que rien ne tue.
			//
			// SA CONDITION N EST PAS CELLE DE L ENTREE, et c est le fait qui a manque : le
			// film n est pas muet sur les morts, c est `IdentityInput.PlayerIndices.ByXUID`
			// qui est VIDE — `buildOwnersFromTracks` sort alors avant toute decoupe. Le geste
			// de retrait n est donc pas le meme : lire la table d index, pas mieux lire le
			// fil des morts.
			Fichier:   pkgReplay + "tracks_publication.go",
			Ancre:     "in.fb.DeclencheN(fallback.NomVieCoupeeAuTrouDeReplication, coupuresDuSeuil(vies))",
			Condition: CondSectionAbsente,
		}},
		DatePose:     "2026-09-14",
		CibleRetrait: "lot M2 (retrait sec si le compte reste nul au corpus gate — D14 d)",
		// CE QUE LA CONVERSION A FERME, MESURE (`decoupe_des_vies_mesure_test.go`, 8 builds,
		// 2026-09-15) : sur 212 coupures decidees par le seuil, 4 portaient une mort ecrite, 0 un
		// record de creation, 0 une frontiere de manche — et 208 n'etaient justifiees PAR RIEN.
		// Les 208 sont devenues des lacunes. Le repli, lui, ne se declenche sur AUCUN des 8 builds
		// (aucune vie coupee n'appartient a un joueur sans mort ecrite) : son critere de retrait
		// est donc DEJA tenu sur cet echantillon, et le corpus gate tranchera sur le parc.
		// LE COMPTE EST UN NOMBRE DE COUPURES, PAS DE PASSAGES (revue M1, 2026-09-15) : le site 2
		// appelle `DeclencheN` avec le nombre de vies que le seuil a coupees. Un `hits: 1` pour
		// 212 coupures sous-decrivait le fait au point de rendre D14 (d) inoperant — un compte
		// qui ne bouge pas avec la population qu il decrit ne mesure rien.
		CritereRetrait:  "0 coupure decidee par le seuil au corpus gate ; `vies_un_echantillon_test.go` rend 0 orpheline sur les 8 builds",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_mort_ecartee_hors_equipe_de_base",
		Fait:      "une mort entre-t-elle dans le contexte d'isolement publie",
		Mecanisme: "la victime n'est dans aucune equipe de la feuille de match : la mort est ecartee ENTIEREMENT",
		Condition: CondSectionAbsente,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "death_context.go",
			Ancre:   "if !dansUneEquipe {",
		}, {
			Fichier: pkgReplay + "death_context.go",
			Ancre:   "e.Fallbacks.Declenche(fallback.NomMortEcarteeHorsEquipeDeBase)",
		}},
		DatePose:     dateAudit0E,
		CibleRetrait: "lot 1.7 (l'equipe vient du film, V4) porte jusqu'a ce calque",
		// Aucune ligne, aucun compteur, aucun log : une mort disparaît du calque sans trace.
		CritereRetrait:  "l'equipe vient du film et non de la base ; 0 mort ecartee sur les 8 builds",
		CompteurBranche: false,
		CibleComptage:   comptageCollecteur,
	},
	{
		Nom:       "repli_coequipier_hors_de_vue_par_defaut",
		Fait:      "l'etat d'un coequipier a l'instant d'une mort",
		Mecanisme: "ni visible ni en attente : « hors de vue » par defaut de fin de fonction — indistinct de « canal non lu »",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "death_context.go",
			Ancre:   "return EtatHorsDeVue",
		}, {
			Fichier: pkgReplay + "death_context.go",
			Ancre:   "e.Fallbacks.Declenche(fallback.NomCoequipierHorsDeVueParDefaut)",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot de conversion du contexte d'isolement (hors famille 1.9 a ce jour)",
		CritereRetrait:  "un quatrieme etat NOMME « non lu » separe l'ignorance de la mesure, et son compte tombe a 0",
		CompteurBranche: false,
		CibleComptage:   comptageCollecteur,
	},
	// RETIRE LE 2026-09-16 (lot 1.9.10) : `repli_fin_de_vie_vehicule_par_recensement`. La fin de
	// vie d'un véhicule se LIT au composant `object-dead-state` de `ti=40`
	// (`grammar.ScanObjectDeaths`), et la borne « dernier recensement + 20 s » a disparu du
	// code avec son ancre (`vehicle_tracks.go`, `assignVehicleWindows`). Ce que la mesure a
	// établi avant le retrait : le repli se déclenchait EXACTEMENT sur les vies que la dernière
	// image-clé recense encore, c'est-à-dire celles qui finissent AVEC le film — 88 vies sur
	// 295, 20 artefacts du parc. Elles sont désormais publiées `end = "film_end"`, qui est une
	// lecture et non une inférence. D14 (d) appliqué : le repli sort du registre avec son code.
	{
		Nom:  "repli_cap_vehicule_vitesse_insuffisante",
		Fait: "le cap publie d'un vehicule",
		// LE REPLI A CHANGE DE RANG le 2026-09-21 (lot 5.4.3) : il n'est plus la source PRINCIPALE
		// du cap, il est le SECOND recours. Le film ECRIT l'avant du chassis — la perpendiculaire
		// reconstruite du couple (vecteur haut, angle de roulis) d'`i2` — et le cap en sort sur le
		// mode dont la reconstruction est prouvee. La deduction par la velocite ne sert plus que
		// la ou ce mode est absent : chemin « delta » (aucun angle absolu ecrit) et mode 0, REFUTE
		// par la mesure sur `a349fea8` (mediane 95,5 deg contre un temoin a 94,5).
		Mecanisme: "le film ne rend pas de cap sur ce chemin (mode non publie ou roulis relatif) : le cap est DEDUIT de la velocite i1, et sous vehicleMinSpeedMPS (5 m/s) le dernier connu est reporte par l'appelant",
		Condition: CondFilmMuet,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "vehicle_heading.go",
			Ancre:   "return vehicleVelocityHeadingOf(p)",
		}, {
			Fichier: pkgReplay + "build_vehicles.go",
			Ancre:   "clock.fb.DeclencheN(fallback.NomCapVehiculeVitesseInsuffisante, logVehicleHeadingSource(scan.Positions))",
		}},
		DatePose: dateAudit0E,
		// LE NEGATIF QUI TENAIT ICI EST TOMBE : « i2 REFUTE » datait du lot 5.2b.2, qui avait
		// mesure le vecteur HAUT en croyant mesurer l'avant. La cible de retrait est donc
		// desormais REELLE et datee, et le compte qui manquait existe.
		CibleRetrait:    "le mode 0 d'i2 rendu publiable (sa direction lue n'est pas verticale sur les vieux builds : |z| median 0,585 contre 0,979 — a instruire avant tout elargissement), ou la reconstruction du chemin delta par registre d'etat par entite",
		CritereRetrait:  "part de `capParVelocite` nulle au journal `rejeu : source du cap des vehicules` sur le parc",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_episode_occupation_par_trou_de_position",
		Fait:      "un joueur etait-il a bord d'un vehicule, de quand a quand, et duquel",
		Mecanisme: "aucun evenement d'embarquement ou de sortie n'explique le trou : l'episode est reconstruit du TROU de position (>= 3 s) et du vehicule le plus proche sous 1,5 m en plan",
		Condition: CondFilmMuet,
		Ordre:     OrdreApresLecture,
		// DEUX SITES DEPUIS LE LOT 5.10 : le compteur est DECLARE avec les autres statistiques
		// d assemblage (`vehicle_rides.go`) et INCREMENTE a l etape qui pose le repli
		// (`vehicle_rides_build.go`, sortie du meme fichier par deplacement pur).
		Sites: []Site{
			{Fichier: pkgReplay + "vehicle_rides.go", Ancre: "repli int"},
			{Fichier: pkgReplay + "vehicle_rides_build.go", Ancre: "b.st.repli++"},
			{Fichier: pkgReplay + "build_vehicles.go", Ancre: "clock.fb.DeclencheN(fallback.NomEpisodeOccupationParTrouDePosition, st.repli)"},
		},
		DatePose:     dateAudit0E,
		CibleRetrait: "lot 2.2 (M2, les lecteurs recoivent le profil : cablage des compteurs) puis le chantier vehicules — le lot 1.9.10 (2026-09-16) a lu la fin de vie au dead-state SANS convertir l episode d occupation par trou de position (cible reecrite a sa fusion)",
		// Ce repli-ci porte DÉJÀ son nom (`st.repli`) et son compte — et DEPUIS LE SCHEMA 67 il
		// le porte jusqu'au document : `rides[].src = "proximity"` et `coverage.vehicles.
		// ridesProximity` disent, par épisode et en total, ce qui est DÉDUIT plutôt que LU. Le
		// lot 5.10 a aussi posé sa borne : un épisode de repli n'est publié que si aucune
		// lecture d'`object-parent-state` de la même vie ne le contredit.
		CritereRetrait:  "0 episode de provenance `proximity` sur les 8 builds une fois les montees a bord lues sur tous les sieges",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_chassis_vehicule_marqueur_neutre",
		Fait:      "la famille de sprite d'un vehicule dont le chassis n'est pas dans la table",
		Mecanisme: "famille VIDE : le vehicule reste publie et le client dessine un marqueur neutre",
		// LOT 1.9.9 (2026-09-16) : la condition passe de `film_muet` a
		// `chassis_absent_de_la_table`. Le film N EST PAS muet — il ecrit le mot d identite du
		// chassis dans le default-state du record de creation `ti=40`, et le decodeur le LIT
		// (c'est `Coverage.Vehicles.WithChassis`). Ce qui manque est NOTRE table. Classer ce
		// repli en `film_muet` envoyait chercher la correction du mauvais cote de la frontiere.
		Condition: CondChassisAbsentDeLaTable,
		Ordre:     OrdreApresLecture,
		Sites: []Site{
			{
				Fichier: pkgReplay + "vehicle_families.go",
				Ancre:   "VALEUR INCONNUE = FAMILLE VIDE",
			},
			{
				Fichier: pkgReplay + "document_vehicles_coverage.go",
				Ancre:   "fb.Declenche(fallback.NomChassisVehiculeMarqueurNeutre)",
			},
		},
		DatePose:     dateAudit0E,
		CibleRetrait: "lot 2.2 (M2) puis le chantier vehicules — 22 chassis restes sans piece ecrite au lot 1.9.9 (2026-09-16) : chaque chassis prouve entre en table sous sa famille, ce repli tombe quand il n en reste aucun (cible reecrite a la fusion, le lot 1.9.10 n en nommait aucun)",
		// Décision utilisateur du 2026-09-14 : le parc d'assets véhicules est COMPLET ; un
		// châssis absent de la table est un MISMATCH à nommer, jamais un véhicule manquant.
		// Le lot 1.9.9 a NOMME le premier d'entre eux (`0x038df01a`, la tourelle automatique
		// bannie) ; la mesure du meme lot en a releve d'autres, consignes en §4 du plan et NON
		// traites (regle 7) — d'ou une cible de retrait qui n'est plus 1.9.9.
		CritereRetrait:  "coverage.vehicles.unknownChassis a 0 sur le parc ; tout chassis restant est nomme en table avec sa famille",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_chunk_de_replication_saute",
		Fait:      "la table index de joueur -> xuid, lue dans les chunks de replication",
		Mecanisme: "un chunk illisible ou dont la resolution est vide est SAUTE, sans distinguer les deux cas",
		Condition: CondSectionAbsente,
		Ordre:     OrdreApresLecture,
		// LA LECTURE EST DESCENDUE DE `replay` EN `grammar` AU LOT J4.2 (2026-09-26) : meme ancre,
		// sans le qualificatif de paquet.
		Sites: []Site{{
			Fichier: pkgFilmdec + "player_index.go",
			Ancre:   "raw, _, ok := FilmChunkAt(film, c)",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.6 (la table du film remplace cette voie) : le repli tombe quand la table de chunk_00 est le lien direct partout",
		CritereRetrait:  "0 chunk saute sur les 8 builds, ou la voie entiere retiree avec ses tests",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:  "repli_identite_vie_par_occupation_du_corps",
		Fait: "le joueur d une piste publiee que ni la lecture (creation, morts, table) ni les fermetures, bots et relais n ont nommee",
		Mecanisme: "la vie nommee du MEME CORPS (slot, generation) qui precede la piste, sinon celle qui la suit, sinon le pont par slot quand les records du slot ne divergent pas ; " +
			"abstention entre deux occupants differents (lot J5.4 : borne au corps, RA2-2)",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "unnamed_lives.go",
			Ancre:   "reg.fb.Declenche(fallback.NomIdentiteVieParOccupationDuCorps)",
		}},
		DatePose:        date0927,
		CibleRetrait:    "J11 du plan PLAN_SUITE_AUDIT_DECODEUR_FILM (gates de corpus) : lecture de l identite des vies restantes (fin de vie sans mort ecrite, bots sans entite)",
		CritereRetrait:  "0 piste nommee par occupation sur le corpus du gate de rejeu",
		CompteurBranche: true,
	},
}
