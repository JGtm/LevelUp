package fallback

// registre_replay_identites.go — les replis des calques IDENTITÉ, VIES, DRAPEAU, ZONE,
// VÉHICULE, CRÂNE et BOMBE (`internal/games/halo_infinite/film/replay/`).

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
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.9.14 (le roster a l'instant T, c'est les occupants) et 1.9.13",
		CritereRetrait:  "0 recours au premier occupant sur les 8 builds : toute demande tombe dans une vie couvrante",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
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
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.6 (le registre d'identite prend la table du film comme lien direct) : le compte doit tomber avec la couverture du lien direct",
		CritereRetrait:  "coverage.identity.coverage.filmTable a 100 % et 0 piste nommee par le pont sur les 8 builds",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
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
		DatePose:     dateAudit0E,
		CibleRetrait: "lot 1.9.13 (les vies se decoupent aux morts ecrites, donc ne se recouvrent plus)",
		// Sans seuil, un recouvrement d'une seule frame l'emporte sur l'absence : le compte dit
		// combien de pistes sont nommées par un arbitrage plutôt que par une lecture.
		CritereRetrait:  "0 piste arbitree par recouvrement sur les 8 builds",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_vie_coupee_au_trou_de_replication",
		Fait:      "ou finit une vie de joueur",
		Mecanisme: "pour un joueur dont le film n'ecrit AUCUNE mort, un trou de positions de plus de lifeGapUS (5 s) ferme la vie courante et en ouvre une neuve",
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
		}},
		DatePose:     "2026-09-14",
		CibleRetrait: "lot M2 (retrait sec si le compte reste nul au corpus gate — D14 d)",
		// CE QUE LA CONVERSION A FERME, MESURE (`decoupe_des_vies_mesure_test.go`, 8 builds,
		// 2026-09-15) : sur 212 coupures decidees par le seuil, 4 portaient une mort ecrite, 0 un
		// record de creation, 0 une frontiere de manche — et 208 n'etaient justifiees PAR RIEN.
		// Les 208 sont devenues des lacunes. Le repli, lui, ne se declenche sur AUCUN des 8 builds
		// (aucune vie coupee n'appartient a un joueur sans mort ecrite) : son critere de retrait
		// est donc DEJA tenu sur cet echantillon, et le corpus gate tranchera sur le parc.
		CritereRetrait:  "0 declenchement au corpus gate ; `vies_un_echantillon_test.go` rend 0 orpheline sur les 8 builds",
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
		}},
		DatePose:     dateAudit0E,
		CibleRetrait: "lot 1.7 (l'equipe vient du film, V4) porte jusqu'a ce calque",
		// Aucune ligne, aucun compteur, aucun log : une mort disparaît du calque sans trace.
		CritereRetrait:  "l'equipe vient du film et non de la base ; 0 mort ecartee sur les 8 builds",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
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
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot de conversion du contexte d'isolement (hors famille 1.9 a ce jour)",
		CritereRetrait:  "un quatrieme etat NOMME « non lu » separe l'ignorance de la mesure, et son compte tombe a 0",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_fin_de_vie_vehicule_par_recensement",
		Fait:      "quand finit la vie d'un vehicule",
		Mecanisme: "aucune image-cle ne cesse de le recenser : la fin est posee a lastUS + vehicleCensusTolUS (20 s apres le dernier echantillon)",
		Condition: CondFilmMuet,
		Ordre:     OrdreDevantLaLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "vehicle_tracks.go",
			Ancre:   "l.hiUS = l.lastUS + vehicleCensusTolUS",
		}},
		DatePose:     dateAudit0E,
		CibleRetrait: "lot 1.9.10 (la fin de vie d'un vehicule lue au dead-state ecrit)",
		// ORDRE `devant_la_lecture` : le film ÉCRIT la destruction (`ti=40`, lisible depuis le
		// 2026-09-05 sur `wt/vehicule-deadstate`) et l'inférence décide sans elle. Défaut
		// mesuré : le ghost slot 777 de `bfecd02b` effacé à 287,4 s alors que l'utilisateur le
		// pilote encore.
		CritereRetrait:  "VehicleTrack.End porte l'instant du dead-state ; 0 fin inferee sur `bfecd02b` et sur l'echantillon court",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_cap_vehicule_vitesse_insuffisante",
		Fait:      "le cap publie d'un vehicule",
		Mecanisme: "vitesse sous vehicleMinSpeedMPS (5 m/s) : aucun cap n'est rendu et le dernier connu est reporte par l'appelant",
		Condition: CondFilmMuet,
		Ordre:     OrdreSansLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "vehicle_tracks.go",
			Ancre:   "if speed < vehicleMinSpeedMPS {",
		}},
		DatePose:     dateAudit0E,
		CibleRetrait: "aucune tant que le negatif tient (i2 REFUTE, i21 ABSENT de ti=40) ; le COMPTE des reports est ce qui manque",
		// Négatif mesuré, table (C11) de l'audit : le repli est légitime, son silence ne l'est pas.
		CritereRetrait:  "reports comptes dans VehicleCoverage ; retrait si un troisieme canal de cap est etabli",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_episode_occupation_par_trou_de_position",
		Fait:      "un joueur etait-il a bord d'un vehicule, de quand a quand, et duquel",
		Mecanisme: "aucun evenement d'embarquement ou de sortie n'explique le trou : l'episode est reconstruit du TROU de position (>= 3 s) et du vehicule le plus proche sous 1,5 m en plan",
		Condition: CondFilmMuet,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "vehicle_rides.go",
			Ancre:   "st.repli++",
		}},
		DatePose:     dateAudit0E,
		CibleRetrait: "lot 1.9.10 (la fin de vie d'un vehicule lue au dead-state ecrit) et le chantier vehicules",
		// Ce repli-ci porte DÉJÀ son nom (`st.repli`) et son compte (`episodesDeRepli` au
		// journal, `VehicleRideSrcGap` dans le document) : il entre au registre pour que sa
		// condition de retrait se lise au même endroit que les autres.
		CritereRetrait:  "0 episode de provenance `gap` sur les 8 builds une fois les evenements d'embarquement complets",
		CompteurBranche: false,
		CibleComptage:   "sans objet a court terme : le compte EXISTE deja sous `vehicleRideStats.repli` et `VehicleRideSrcGap` ; son entree dans `coverage.fallbacks` viendra au lot 1.9.10",
	},
	{
		Nom:       "repli_chassis_vehicule_marqueur_neutre",
		Fait:      "la famille de sprite d'un vehicule dont le chassis n'est pas dans la table",
		Mecanisme: "famille VIDE : le vehicule reste publie et le client dessine un marqueur neutre",
		Condition: CondFilmMuet,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "vehicle_families.go",
			Ancre:   "VALEUR INCONNUE = FAMILLE VIDE",
		}},
		DatePose:     dateAudit0E,
		CibleRetrait: "lot 1.9.9 (les tourelles automatiques bannies nommees, et dessinees comme elements de carte)",
		// Décision utilisateur du 2026-09-14 : le parc d'assets véhicules est COMPLET ; un
		// châssis absent de la table est un MISMATCH à nommer, jamais un véhicule manquant.
		CritereRetrait:  "coverage.vehicles.unknownChassis a 0 sur le parc ; tout chassis restant est nomme en table avec sa famille",
		CompteurBranche: false,
		CibleComptage:   "lot 1.9.9 (Coverage.Vehicles.UnknownChassis compte deja les chassis ; le repli entre au registre pour que le compte se lise comme un repli)",
	},
	{
		Nom:       "repli_chunk_de_replication_saute",
		Fait:      "la table index de joueur -> xuid, lue dans les chunks de replication",
		Mecanisme: "un chunk illisible ou dont la resolution est vide est SAUTE, sans distinguer les deux cas",
		Condition: CondSectionAbsente,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "player_index.go",
			Ancre:   "raw, _, ok := filmdec.FilmChunkAt(film, c)",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.6 (la table du film remplace cette voie) : le repli tombe quand la table de chunk_00 est le lien direct partout",
		CritereRetrait:  "0 chunk saute sur les 8 builds, ou la voie entiere retiree avec ses tests",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_drapeau_seul_en_jeu",
		Fait:      "quel drapeau une prise concerne",
		Mecanisme: "aucun drapeau au sol ne convient : s'il n'en reste qu'UN en jeu, c'est lui",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "flag_assign.go",
			Ancre:   "if f := g.seulEnJeu(recevable); f >= 0 {",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.9.6 (le drapeau qui rentre pris dans ev.flag, deja nomme en amont)",
		CritereRetrait:  "0 recours a la regle « seul en jeu » une fois ev.flag lu ; coverage.flagCarries.ambiguousReturns a 0",
		CompteurBranche: false,
		CibleComptage:   "lot 1.9.6",
	},
	{
		Nom:       "repli_invariant_propre_drapeau_muet",
		Fait:      "un portage designe-t-il le drapeau de l'equipe de son porteur (ce qu'aucune regle du mode n'autorise)",
		Mecanisme: "l'invariant rend FAUX des qu'une des deux equipes est inconnue : il se tait au lieu de refuser",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "flag_assign.go",
			Ancre:   "func sonPropreDrapeau(spawns []FlagSpawn, f int, equipe int, connue bool) bool {",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.7 (l'equipe vient du film) : l'equipe du porteur est desormais lue, donc le silence doit disparaitre",
		CritereRetrait:  "0 portage dont l'equipe du porteur est inconnue sur les 8 builds (coverage.flagCarries.carrierTeamUnknown a 0)",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_index_drapeau_zero_pour_tous",
		Fait:      "l'index de drapeau de CHAQUE portage d'un film sans socle catalogue",
		Mecanisme: "aucun socle : tous les portages recoivent flagIndex = 0",
		Condition: CondSectionAbsente,
		Ordre:     OrdreSansLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "flag_assign.go",
			Ancre:   "raws[i].flagIndex = 0",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot de completion du catalogue de socles (hors famille 1.9)",
		CritereRetrait:  "0 film CTF sans socle catalogue sur le parc",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_position_lacher_prend_la_prise",
		Fait:      "ou un drapeau a ete lache",
		Mecanisme: "aucun point publie a la frame de fin : la position de LACHER prend celle de la PRISE",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "flag_carries.go",
			Ancre:   "r.x1, r.y1 = p0.X, p0.Y",
		}},
		DatePose:     dateAudit0E,
		CibleRetrait: "lot 1.9.13 (les vies couvrent alors la fin du portage)",
		// Deux points identiques se lisent sur la carte comme un portage immobile : le repli
		// FABRIQUE une donnée plausible, ce qui est la forme la plus difficile à repérer.
		CritereRetrait:  "0 portage dont la fin n'a pas de point publie sur les 8 builds",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_piste_drapeau_sans_pont_ecartee",
		Fait:      "quelles pistes peuvent porter un drapeau",
		Mecanisme: "le pont ne nomme pas le slot : la piste est ecartee en silence, sans etre comptee comme le REFUS nomme voisin (AmbiguousSlot)",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "flag_carrier_tracks.go",
			Ancre:   "continue // le pont ne nomme pas ce slot : aucun porteur a inventer",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.6 (lien direct par la table du film)",
		CritereRetrait:  "0 piste ecartee faute de pont sur les 8 builds",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_zone_camp_sans_roster",
		Fait:      "a quel camp crediter une capture de zone",
		Mecanisme: "roster vide : TOUTE valeur non neutre du canal compte comme une capture du camp",
		Condition: CondSectionAbsente,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "zone_states_owner.go",
			Ancre:   "case len(teams) == 0 && v != zoneNeutralOwner:",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.7 (l'equipe vient du film) porte au calque des zones — cf. D3 (1.7), ZoneInput.TeamByXUID prend TOUJOURS l'equipe de la base",
		CritereRetrait:  "le roster des zones vient du film ; 0 film a roster vide sur les 8 builds",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_zone_proprietaire_sans_roster",
		Fait:      "quel camp possede une zone a un instant",
		Mecanisme: "roster vide : toute valeur <= 1 est tenue pour un camp",
		Condition: CondSectionAbsente,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "zone_states_owner.go",
			Ancre:   "case len(teams) == 0 && v <= 1:",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "meme cible que repli_zone_camp_sans_roster",
		CritereRetrait:  "0 film a roster vide sur les 8 builds",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_colline_votes_periode_entiere",
		Fait:      "ou se trouve la colline designee d'une periode",
		Mecanisme: "aucune rampe de capture dans la periode : les votes sont repris sur TOUTE la periode, rampes comprises ou non",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "zone_states_hill.go",
			Ancre:   "votes = hillVotes(zones, pts, p.t0, p.t1)",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot de conversion du calque des collines (hors famille 1.9 a ce jour)",
		CritereRetrait:  "0 periode sans rampe sur les films a collines du corpus",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_colline_dernier_intervalle_ouvert",
		Fait:      "jusqu'a quand court la propriete d'une colline",
		Mecanisme: "le dernier groupe court jusqu'a l'infini (borne ouverte a droite), faute d'emission de fin",
		Condition: CondFilmMuet,
		Ordre:     OrdreSansLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "zone_states_hill.go",
			Ancre:   "t1 := int(^uint(0) >> 1)",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "aucune tant que le canal reste un ETAT sans emission de fin ; le COMPTE est ce qui manque",
		CritereRetrait:  "intervalles ouverts comptes dans ZonesCoverage ; retrait si une fin ecrite est etablie",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_crane_porteur_sans_vie_nommee",
		Fait:      "un portage de crane est-il refuse faute de porteur present",
		Mecanisme: "le joueur n'a AUCUNE vie nommee : la porte laisse passer sans rien verifier",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "skull_carries.go",
			Ancre:   "func (p carrierPresence) gate(xuid string, f0, f1 int) (int, int, bool) {",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.9.5 (le porteur du crane lu au canal des armes tenues)",
		CritereRetrait:  "0 porteur sans vie nommee sur les films Oddball du corpus",
		CompteurBranche: false,
		CibleComptage:   "lot 1.9.5",
	},
	{
		Nom:       "repli_armement_bombe_debut_a_zero",
		Fait:      "l'instant de debut d'un armement de bombe commence avant la frame 0",
		Mecanisme: "la conversion en frame echoue : le debut est pose a 0",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "bomb_armings.go",
			Ancre:   "startT, ok := c.frameOf(int(r.StartMS))",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot de conversion de l'origine du rejeu (coverage.originResolved)",
		CritereRetrait:  "0 armement anterieur a la frame 0 sur les films d'Assaut du corpus",
		CompteurBranche: true,
	},
}
