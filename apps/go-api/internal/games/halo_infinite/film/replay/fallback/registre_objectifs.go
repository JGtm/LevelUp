package fallback

// registre_objectifs.go — les replis du lecteur d'ACTIONS D'OBJECTIF
// (`internal/analysis/objectiveevents/`) et du CONSTRUCTEUR d'artefacts
// (`internal/replaybuild/`).
//
// NOTE D'ARCHITECTURE : `internal/analysis/` n'importe JAMAIS `internal/games/{slug}/`
// (décision D9 du plan, garde-rail `archlint/no_title_package_in_analysis_test.go`, dont
// l'allowlist doit se VIDER au pas 5 de M2). Les replis d'`objectiveevents` sont donc DÉCLARÉS
// ici et leur compteur ne peut pas être câblé depuis ce paquet : il viendra par inversion de
// dépendance (un port défini côté `analysis/`, l'implémentation injectée par l'appelant) au lot
// qui convertit le fait. C'est écrit entrée par entrée dans CibleComptage.

const (
	pkgObjectiveEvents = "internal/analysis/objectiveevents/"
	pkgReplaybuild     = "internal/replaybuild/"
	// comptageStatborg : la cible de cablage commune aux replis du lecteur statborg — un port
	// defini cote `analysis/`, dont l'appelant fournit l'implementation (cf. note d'architecture).
	comptageStatborg = "lot de conversion statborg (port cote analysis/)"
	// comptageLot1911 : les replis que le lot 1.9.11 ouvre deja (desiignateur de manche).
	comptageLot1911 = "lot 1.9.11"
)

var registreObjectifsEtConstruction = []Repli{
	{
		Nom:       "repli_enregistrement_statborg_abandonne",
		Fait:      "quels enregistrements statborg d'une frame entrent dans la lecture",
		Mecanisme: "en-tete non reconnu, aucun composant decode, ou compteur hors domaine : l'enregistrement est abandonne par un `continue` muet",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgObjectiveEvents + "statborg.go",
			Ancre:   "if len(comps) == 0 || !statCountersInDomain(comps) {",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "volet statborg du registre des reports (toujours ouvert) ; hors famille 1.9 a ce jour",
		CritereRetrait:  "abandons comptes par cause et rapportes a la population lue ; retrait quand le compte est nul",
		CompteurBranche: false,
		CibleComptage:   comptageStatborg + ", cf. note d'architecture",
	},
	{
		Nom:       "repli_composants_statborg_arretes",
		Fait:      "quels composants d'un enregistrement statborg sont lus",
		Mecanisme: "un composant non decodable arrete la boucle : les suivants du MEME enregistrement sont perdus sans trace",
		Condition: CondLectureNonPortee,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgObjectiveEvents + "statborg.go",
			Ancre:   "v, w, ok := decodeStatComponent(pay, q)",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 3.6 (les composants manquants, archetype par archetype)",
		CritereRetrait:  "0 arret en milieu d'enregistrement sur les 8 builds",
		CompteurBranche: false,
		CibleComptage:   comptageStatborg,
	},
	{
		Nom:       "repli_manches_contigues_decretees",
		Fait:      "quelles manches d'un match sont REELLES",
		Mecanisme: "les manches admises doivent etre contigues depuis 0 ; si aucune ne l'est, la manche 0 est DECRETEE reelle",
		Condition: CondNonResolu,
		Ordre:     OrdreDevantLaLecture,
		Sites: []Site{{
			Fichier: pkgObjectiveEvents + "statborg.go",
			Ancre:   "out[0] = true",
		}},
		DatePose:     dateAudit0E,
		CibleRetrait: "lot 1.9.11 (le designateur de manche lu tel que le film l'ecrit)",
		// ORDRE `devant_la_lecture` : sur `fb1a1a72` (CTF:Arena, 814 s pour 720 s de temps
		// réglementaire), 148 records statborg portent le désignateur `2` et la garde publie
		// 1 manche parce que la manche 1 est absente (0.D.1 bis). La garde JETTE ce que le film
		// écrit.
		CritereRetrait:  "coverage.score.rounds repasse a la valeur ECRITE ; la garde tombe avec ses tests une fois la semantique de `2` etablie",
		CompteurBranche: false,
		CibleComptage:   comptageLot1911,
	},
	{
		Nom:       "repli_emission_hors_domaine_jetee",
		Fait:      "quelles emissions d'un compteur nomme entrent dans la serie",
		Mecanisme: "valeur negative, ou canal B hors domaine du score de mode : l'emission est jetee par un `continue`",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgObjectiveEvents + "named_series.go",
			Ancre:   "if val < 0 {",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot de conversion des series nommees (hors famille 1.9 a ce jour)",
		CritereRetrait:  "emissions jetees comptees par cause ; une emission negative est un defaut d'alignement, pas une donnee a filtrer",
		CompteurBranche: false,
		CibleComptage:   comptageStatborg,
	},
	{
		Nom:       "repli_manche_du_slot_sautee",
		Fait:      "la serie cumulee d'un slot pour une manche",
		Mecanisme: "la plus longue sous-suite non decroissante est vide : la manche entiere du slot est sautee",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgObjectiveEvents + "named_series.go",
			Ancre:   "kept := longestRun(pts, false)",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "question NE17 de la table (D) de l'audit instruite (`longestRun` ecarte-t-il des points reels ?)",
		CritereRetrait:  "0 manche de slot sautee sur les films multi-manches du corpus",
		CompteurBranche: false,
		CibleComptage:   comptageStatborg,
	},
	{
		Nom:       "repli_table_identite_vide",
		Fait:      "le pont slot statborg -> joueur, par les instants de mort",
		Mecanisme: "aucune mort fournie : une table VIDE est rendue, indiscernable d'un pont qui n'a rien pu resoudre",
		Condition: CondSectionAbsente,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgObjectiveEvents + "slotidentity_deaths.go",
			Ancre:   "return map[int]string{}",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.6 porte a ce calque (la table du film donne le lien direct)",
		CritereRetrait:  "0 pont vide faute de morts sur les 8 builds",
		CompteurBranche: false,
		CibleComptage:   comptageStatborg,
	},
	{
		Nom:       "repli_mort_sans_xuid_ignoree",
		Fait:      "quelles morts alimentent le fil par joueur",
		Mecanisme: "une mort sans xuid est ignoree par un `continue`",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgObjectiveEvents + "slotidentity_deaths.go",
			Ancre:   "if d.XUID == \"\" {",
		}},
		DatePose:     dateAudit0E,
		CibleRetrait: "lot 1.8 porte jusqu'a ce calque",
		// Contredit la décision utilisateur du 2026-09-06 (« les vies anonymes n'existent
		// pas ; une vie est un humain ou un bot ») : une mort sans xuid est un défaut de
		// nommage, à réparer à la source.
		CritereRetrait:  "0 mort sans xuid sur les 8 builds",
		CompteurBranche: false,
		CibleComptage:   comptageStatborg,
	},
	{
		Nom:       "repli_emission_du_compteur_de_morts_jetee",
		Fait:      "la progression du compteur de morts d'un slot",
		Mecanisme: "valeur hors [0, maxDeathsPerSlot] : l'emission est jetee par un `continue`",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgObjectiveEvents + "slotidentity_deaths.go",
			Ancre:   "if !ok || v.B < 0 || v.B > maxDeathsPerSlot {",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot de conversion statborg",
		CritereRetrait:  "0 emission hors domaine sur les 8 builds (une valeur hors domaine est un defaut d'alignement)",
		CompteurBranche: false,
		CibleComptage:   comptageStatborg,
	},
	{
		Nom:       "repli_debut_de_manche_au_minimum",
		Fait:      "l'instant de debut d'une manche",
		Mecanisme: "hors consensus : le MINIMUM des instants observes fait office de debut",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgObjectiveEvents + "slotidentity_rounds.go",
			Ancre:   "if cur, seen := min[r.Round]; !seen || r.TimeMS < cur {",
		}},
		DatePose:     dateAudit0E,
		CibleRetrait: "lot 1.9.11 (le designateur de manche lu tel que le film l'ecrit)",
		// DÉFAUT DÉJÀ MESURÉ (audit 0.E) : 213 s d'attribution fausse sur `24dbb67d`.
		CritereRetrait:  "les bornes de manche viennent du designateur ecrit ; 0 recours au minimum sur les films multi-manches",
		CompteurBranche: false,
		CibleComptage:   comptageLot1911,
	},
	{
		Nom:       "repli_slot_abandonne_au_premier_arrive",
		Fait:      "quel joueur occupe un slot statborg pour une manche, quand la feuille en propose plusieurs",
		Mecanisme: "premier arrive, premier servi : le slot deja attribue ou le joueur deja pris est abandonne",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgObjectiveEvents + "slotidentity_rounds.go",
			Ancre:   "if _, deja := fusion[slot]; deja || pris[xuid] {",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.6 porte a ce calque",
		CritereRetrait:  "0 abandon par premier arrive sur les 8 builds",
		CompteurBranche: false,
		CibleComptage:   comptageStatborg,
	},
	{
		Nom:       "repli_instant_sur_la_premiere_manche",
		Fait:      "a quelle manche appartient un instant anterieur a toute manche connue",
		Mecanisme: "il retombe sur la PREMIERE manche",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgObjectiveEvents + "slotidentity_rounds.go",
			Ancre:   "round := ri.starts[0].round",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.9.11",
		CritereRetrait:  "0 instant anterieur a la premiere manche une fois les bornes lues au designateur",
		CompteurBranche: false,
		CibleComptage:   comptageLot1911,
	},
	{
		Nom:       "repli_famille_objectif_vide",
		Fait:      "la famille d'objectif d'un match, donc les actions nommees qu'il publiera",
		Mecanisme: "aucun mot-cle reconnu dans le nom de variante : famille vide, et le match ne publie AUCUNE action nommee",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgObjectiveEvents + "extract.go",
			Ancre:   "return nil, ctl",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "question NE13 de la table (D) : le classement par `strings.Contains` sur le nom de variante est hors doctrine multi-titre (skill `halo-modes`)",
		CritereRetrait:  "la famille vient du manifeste de titre ; 0 match a famille vide sur le parc",
		CompteurBranche: false,
		CibleComptage:   "lot de conversion des familles d'objectif (port cote analysis/)",
	},
	{
		Nom:       "repli_assistant_non_resolu_abandonne",
		Fait:      "l'assistant d'un frag publie dans les episodes d'equipement",
		Mecanisme: "le nom de l'assistant ne se resout pas en xuid : le champ reste vide et le frag est publie sans assistant",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplaybuild + "kills.go",
			Ancre:   "ref.AssistXUID, ref.AssistKnown = aXUID, true",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.8 porte (la table du film nomme les indices)",
		CritereRetrait:  "0 assistant non resolu sur les 8 builds",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_gamertag_premier_xuid_gagne",
		Fait:      "le xuid porte par un gamertag, pour la resolution des frags",
		Mecanisme: "en cas de divergence, le PREMIER vu gagne",
		Condition: CondContradiction,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplaybuild + "kills.go",
			Ancre:   "if _, seen := out[d.Gamertag]; !seen {",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.8 porte : un gamertag qui porte deux xuids est une contradiction a compter, pas a trancher",
		CritereRetrait:  "0 divergence gamertag -> xuid sur les 8 builds",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_mort_neutre_sans_xuid_abandonnee",
		Fait:      "quelles morts neutres entrent dans le fil publie",
		Mecanisme: "victime sans xuid : la mort est abandonnee (elle ne rencontrerait aucune piste)",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplaybuild + "replaybuild.go",
			Ancre:   "if d.VictimXUID == 0 {",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.8 porte jusqu'au constructeur",
		CritereRetrait:  "0 mort neutre sans xuid sur les 8 builds",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_repere_neutre_generique_conserve",
		Fait:      "l'icone d'une mort neutre",
		Mecanisme: "nature non etablie par l'adaptateur d'assets : le fil garde son repere generique",
		Condition: CondSectionAbsente,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplaybuild + "replaybuild.go",
			Ancre:   "kind, img, ok := adapter.NeutralDeathIcon(d.Source.Tag)",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "completion de la table d'assets de morts neutres",
		CritereRetrait:  "0 mort neutre sans icone nommee sur le parc",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_relais_de_bot_abandonne",
		Fait:      "quel remplacant herite des vies anonymes d'un siege",
		Mecanisme: "le xuid n'a pas la forme bid(N.0) : le relais est abandonne (un humain est nomme par le fil des morts)",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplaybuild + "replaybuild.go",
			Ancre:   "if _, err := fmt.Sscanf(p.XUID, \"bid(%d.0)\", &id); err != nil {",
		}},
		DatePose: dateAudit0E,
		// CIBLE CORRIGEE AU LOT 1.9.14 (2026-09-15) : ce lot a mesure le critere et il est TENU —
		// sur les quatre temoins, les 95 entrees de roster ont TOUTES un siege publie, donc zero
		// remplacant humain sans siege. Mais ce repli ne porte PAS sur le siege : il porte sur le
		// NOMMAGE des vies anonymes d'un relais de bot (`replaybuild.botSuccessions`), que le lot
		// 1.9.14 ne touche pas. L'ancienne cible confondait les deux faits ; elle est remplacee
		// par celle qui correspond au mecanisme.
		CibleRetrait:    "le lot qui fera nommer les vies d'un relais par le registre d'identite plutot que par la participation de la base",
		CritereRetrait:  "0 vie anonyme restante sur un siege de bot relaye, sur les 8 builds, sans passer par `Succession`",
		CompteurBranche: false,
		CibleComptage:   "le meme lot : le compteur n'a de sens qu'au site qui subsistera",
	},
	{
		Nom:       "repli_participant_sans_xuid_retire",
		Fait:      "quels joueurs de la feuille de match entrent dans le tableau des participants",
		Mecanisme: "un joueur sans xuid est retire du tableau par un `continue`",
		Condition: CondSectionAbsente,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplaybuild + "matchfacts.go",
			Ancre:   "func participantsDuTableau(facts port.MatchFacts) []replay.Participant {",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.6.3 (la cuisson HORS LIGNE publie un roster complet, sans base) porte au tableau",
		CritereRetrait:  "0 joueur retire du tableau sur les 8 builds",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_camp_inconnu_retire_de_la_table",
		Fait:      "le camp de chaque joueur, pour le controle et les calques qui lisent encore la base",
		Mecanisme: "un camp inconnu (-1) n'entre PAS dans la table, ce qui le rend indistinct d'un joueur absent",
		Condition: CondSectionAbsente,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplaybuild + "matchfacts.go",
			Ancre:   "if p.TeamID < 0 {",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.7 (V4) porte aux zones — cf. D3 (1.7) : ZoneInput.TeamByXUID prend TOUJOURS l'equipe de la base",
		CritereRetrait:  "les calques prennent l'equipe du film ; la table de base ne sert plus que de controle",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_catalogue_de_zones_absent",
		Fait:      "l'etat des zones d'un match",
		Mecanisme: "le titre n'a pas de table d'objectifs : le rejeu est publie SANS aucun etat de zone",
		Condition: CondSectionAbsente,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplaybuild + "zones.go",
			Ancre:   "titre sans table d'objectifs",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "aucune (degradation gracieuse multi-titre, `ErrCapabilityNotSupported`) ; le COMPTE est ce qui manque",
		CritereRetrait:  "coverage.zones distingue « titre sans table » de « carte hors catalogue » ; retrait sans objet",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_fraicheur_des_derivations_par_taille",
		Fait:      "les derivations d'un artefact sont-elles a jour",
		Mecanisme: "comparaison de la REVISION et de la TAILLE en octets de l'artefact — compromis assume, pas une empreinte",
		Condition: CondFilmMuet,
		Ordre:     OrdreSansLecture,
		Sites: []Site{{
			Fichier: pkgReplaybuild + "derivations_index.go",
			Ancre:   "return m.Rev == DerivationsRev && int64(m.ArtifactBytes) == st.Size()",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 4.4 (la recuisson selective par couche) : une revision par calque remplace la taille",
		CritereRetrait:  "la fraicheur se juge sur la revision de calque portee par le document, jamais sur une taille",
		CompteurBranche: false,
		CibleComptage:   "lot 4.4",
	},
}
