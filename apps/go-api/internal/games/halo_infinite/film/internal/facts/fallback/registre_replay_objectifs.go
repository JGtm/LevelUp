package fallback

// registre_replay_objectifs.go — les replis des calques DRAPEAU, ZONE (et colline) et
// BOMBE (`internal/games/halo_infinite/film/replay/`).
//
// UN FICHIER NEUF, ET C'EST LA LIMITE DE 500 LIGNES : `registre_replay_identites.go` l'avait
// franchie (509 lignes) à la fusion du jalon J9 (drapeau, bombe, zones) dans J5, chacun y ayant
// inscrit un repli (plan `.ai/V7.5/PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25.md`, lot J5.5,
// 2026-09-27). Déplacement pur : les entrées sont recopiées à l'octet, dans leur ordre.

var registreReplayObjectifs = []Repli{
	{
		Nom:       "repli_drapeau_seul_en_jeu",
		Fait:      "quel drapeau une prise concerne",
		Mecanisme: "aucun drapeau au sol ne convient : s'il n'en reste qu'UN en jeu, c'est lui",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "flag_assign.go",
			Ancre:   "if f := g.seulEnJeu(recevable); f >= 0 {",
		}, {
			Fichier: pkgReplay + "flag_assign.go",
			Ancre:   "g.fb.Declenche(fallback.NomDrapeauSeulEnJeu)",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "le drapeau qui rentre pris dans ev.flag, deja nomme en amont ; a defaut, " + retraitRegle4,
		CritereRetrait:  "0 recours a la regle « seul en jeu » une fois ev.flag lu ; coverage.flagCarries.ambiguousReturns a 0",
		CompteurBranche: true,
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
		}, {
			Fichier: pkgReplay + "flag_assign.go",
			Ancre:   "g.fb.Declenche(fallback.NomInvariantPropreDrapeauMuet)",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "l equipe du porteur est lue dans le film : le silence doit disparaitre ; a defaut, " + retraitRegle4,
		CritereRetrait:  "0 portage dont l'equipe du porteur est inconnue sur les 8 builds (coverage.flagCarries.carrierTeamUnknown a 0)",
		CompteurBranche: true,
	},
	{
		Nom: "repli_index_drapeau_zero_pour_tous",
		// LA CONDITION EST RESSERREE (2026-10-05, plan PLAN_REJEU_TOURELLES_TIRS_ZONE_RETOUR, lot
		// L1) : la base et le camp de chaque drapeau se lisent dans le film quand le catalogue se
		// tait (`flag_film_bases.go`). Le repli ne reste que sur un film sans socle catalogue ET
		// sans base lue — aucun vol localise des deux camps, ou des vols disperses.
		Fait:      "l'index de drapeau de CHAQUE portage d'un film sans socle catalogue ni base lue dans le film",
		Mecanisme: "aucun socle, ni du catalogue ni lu dans le film : tous les portages recoivent flagIndex = 0",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "flag_assign.go",
			Ancre:   "raws[i].flagIndex = 0",
		}, {
			Fichier: pkgReplay + "flag_assign.go",
			Ancre:   "ctx.fb.DeclencheN(fallback.NomIndexDrapeauZeroPourTous, len(raws))",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "une lecture des bases qui ne demande pas de vol localise des deux camps (renaissances de l'objet seules), ou la completion du catalogue de socles ; a defaut, " + retraitRegle4,
		CritereRetrait:  "0 film CTF sans socle catalogue ni base lue dans le film sur le parc (coverage.flagCarries.spawns a 0 avec flagFilm vrai)",
		CompteurBranche: true,
	},
	{
		Nom:  "repli_socle_du_film_au_centre_des_vols",
		Fait: "la POSITION d'une base de drapeau lue dans le film, sans socle du catalogue a portee",
		// La renaissance de l'objet drapeau donne le point du socle au centimetre ; le centre des
		// vols en est a une fraction de metre, assez pour poser la base, pas pour y apparier une
		// rentree de l'objet (`flagHomeExactDist`).
		Mecanisme: "aucune renaissance de l'objet (deux naissances au meme point) a moins de 3 m du centre des vols : la base est posee au centre des vols",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "flag_film_bases.go",
			Ancre:   "return flagFilmBase{owner: owner, x: c.x, y: c.y}",
		}, {
			Fichier: pkgReplay + "flag_film_bases.go",
			Ancre:   "fb.Declenche(fallback.NomSocleDuFilmAuCentreDesVols)",
		}},
		DatePose:        "2026-10-05",
		CibleRetrait:    "aucune lecture plus fine connue ; " + retraitRegle4,
		CritereRetrait:  "0 base lue dans le film sans renaissance de l'objet sur les films CTF du parc",
		CompteurBranche: true,
	},
	{
		Nom:  "repli_nombre_drapeaux_hors_catalogue_sans_passage",
		Fait: "combien de drapeaux sont en jeu quand ni le catalogue d'objectifs ni le film ne donnent de base — ce qui nomme, par l'equipe, le drapeau d'une prise (passage de main en main) et celui d'un retour credite",
		// POSE AU LOT J9.2 (2026-09-26, constat RB1-5 de l'audit du 2026-09-24). Le nombre etait
		// SUPPOSE a un (`flagSingleInPlay` rendait vrai sans socle) : toute prise d'un adversaire
		// fermait le portage en cours, tout retour credite aussi. C'est la regle sans son filtre
		// d'equipe, que la mesure du lot 6.11 chiffre a 57 portages et 775,1 s retires a tort.
		Mecanisme: "le nombre n'est pas suppose : aucun drapeau n'est nomme par l'equipe, aucun passage de main en main ni retour credite ne ferme un portage ; compte = portages non juges",
		// LA LECTURE DU FILM EST TENTEE D'ABORD (`flag_film_bases.go`, 2026-10-05) : les bases se
		// lisent aux points ou chaque camp vole. Le repli ne reste que si elle n'a pas tranche (un
		// seul camp vole, vols disperses ou sans position) sur une carte hors catalogue.
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "flag_carries_handoff.go",
			Ancre:   "fb.DeclencheN(fallback.NomNombreDrapeauxHorsCatalogueSansPassage, carries)",
		}},
		DatePose:        "2026-09-26",
		CibleRetrait:    "une lecture des bases qui ne demande pas de vol localise des deux camps (renaissances de l'objet seules), ou l'ajout de la carte au catalogue d'objectifs",
		CritereRetrait:  "0 portage non juge faute de socle sur les films CTF du parc (meme population que repli_index_drapeau_zero_pour_tous)",
		CompteurBranche: true,
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
		DatePose: dateAudit0E,
		// CIBLE REECRITE LE 2026-09-16 (revue de jalon M1) : elle nommait le lot 1.9.13, fusionne
		// le 2026-09-15.
		//
		// ET LE CRITERE N'EST PAS MESURE, CONTRAIREMENT A CE QUE LES GOLDENS LAISSENT CROIRE.
		// Le compteur est câblé et les 8 goldens d'assemblage affichent 0 — mais ce 0 dit
		// « NON EXERCÉ », pas « non déclenché » : `FilmInputs.applyTo` (`film_inputs.go`) pose
		// `opt.Flag.Marks` et JAMAIS `opt.Flag.Scanned`, donc `attachFlagCarries` prend sa
		// branche vide sur les huit fixtures, `buildFlagCarries` ne reçoit aucun portage et
		// `attachFlagCarryPositions` n'est jamais atteint. Aucun des huit films n'apporte de
		// calque drapeau. La mesure demande un film de CTF cuit en entier — corpus gate, ou un
		// fixture d'entrées portant le canal drapeau.
		CibleRetrait: "mesurer d abord (le canal drapeau n est exerce par AUCUN des 8 goldens) ; a defaut, " + retraitRegle4,
		// Deux points identiques se lisent sur la carte comme un portage immobile : le repli
		// FABRIQUE une donnée plausible, ce qui est la forme la plus difficile à repérer.
		CritereRetrait:  "0 portage dont la fin n'a pas de point publie, mesure sur un corpus qui PORTE le calque drapeau (les 8 goldens ne l'exercent pas — verifie le 2026-09-16)",
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
		CibleRetrait:    "le lien direct par la table du film ; a defaut, " + retraitRegle4,
		CritereRetrait:  "0 piste ecartee faute de pont sur les 8 builds",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_zone_camp_de_capture_deduit_de_l_issue",
		Fait:      "quel CAMP pousse la jauge d'une zone, rampe par rampe",
		Mecanisme: "aucun canal POUSSEUR n'a ete elu pour cette zone : le camp est DEDUIT de l'issue — le proprietaire juste apres le sommet d'une rampe ABOUTIE. Une rampe avortee reste alors sans camp",
		// NI `film_muet` NI `lecture_non_portee`, ET LE DIRE EST LE POINT. Le lot 5.6 a MESURE
		// que le film porte ce fait (un second canal `ti=13 tag 4` par zone : 69 rampes
		// abouties sur deux films, 69 accords, 0 desaccord) et que le decodeur le LIT deja.
		// Ce qui manque quand ce repli se declenche, c'est l'ELECTION du canal — elle exige au
		// moins deux rampes abouties concordantes, et une zone tres peu disputee n'en a pas
		// assez. Le classer `film_muet` enverrait chercher la correction dans la grammaire,
		// ou il n'y a rien a faire.
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "zone_states_capturer.go",
			Ancre:   "fb.Declenche(fallback.NomZoneCampDeCaptureDeduitDeLIssue)",
		}},
		DatePose:        "2026-09-21",
		CibleRetrait:    "un critere d'election qui n'exige pas deux rampes abouties — par exemple l'election du POUSSEUR une fois pour le film (les trois zones d'une carte partagent la meme structure de blocs) au lieu d'une election par zone",
		CritereRetrait:  "0 declenchement sur les films a zones du parc",
		CompteurBranche: true,
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
		CibleRetrait:    "la conversion du calque des collines ; a defaut, " + retraitRegle4,
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
			Fichier: pkgReplay + "zone_states_hill_owners.go",
			Ancre:   "t1 := int(^uint(0) >> 1)",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "aucune tant que le canal reste un ETAT sans emission de fin ; le COMPTE est ce qui manque",
		CritereRetrait:  "intervalles ouverts comptes dans ZonesCoverage ; retrait si une fin ecrite est etablie",
		CompteurBranche: true,
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
