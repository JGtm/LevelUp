package fallback

// registre_objectifs.go — les replis du lecteur d'ACTIONS D'OBJECTIF
// (`internal/games/halo_infinite/film/internal/facts/objectives/`) et du CONSTRUCTEUR d'artefacts
// (`internal/replaybuild/`).
//
// NOTE D'ARCHITECTURE : `internal/analysis/` n'importe JAMAIS `internal/games/{slug}/`
// (décision D9 du plan, garde-rail `archlint/no_title_package_in_analysis_test.go`, dont
// l'allowlist doit se VIDER au pas 5 de M2). Les replis d'`objectives` sont donc DÉCLARÉS
// ici et leur compteur n'est pas câblé depuis ce paquet. DEPUIS LE LOT J8.7 (2026-09-27, décision 1
// du superviseur), ils se comptent EN DONNÉES dans ce que le paquet rend déjà — le balayage du
// statborg et le résolveur d'identité par manche (`objectives.ComptesDesReplis`) — et la table de
// `replay` les verse ([siteDeVersement]). Celui qui se déclenche à la CONSULTATION
// (`repli_emission_hors_domaine_jetee`) se compte depuis le lot J8.7-bis (2026-09-28) par ÉVÉNEMENT
// DISTINCT, dans l enregistreur partagé du document
// (`objectives.ReplisALaConsultation`), versé par la même table.

const (
	pkgObjectiveEvents = "internal/games/halo_infinite/film/internal/facts/objectives/"
	pkgReplaybuild     = "internal/replaybuild/"
)

var registreObjectifsEtConstruction = []Repli{
	{
		Nom:       "repli_enregistrement_statborg_abandonne",
		Fait:      "quels enregistrements statborg d'une frame entrent dans la lecture",
		Mecanisme: "en-tete non reconnu, aucun composant decode, ou compteur hors domaine : l'enregistrement est abandonne par un `continue` muet",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{Fichier: pkgObjectiveEvents + "statborg.go", Ancre: "c.EnregistrementsAbandonnes += abandonnes"}, {Fichier: pkgReplaybuild + "matchfacts.go", Ancre: "sb.Replis.Plus(pont.Identite().ComptesDesReplis())"}, siteDeVersement("NomEnregistrementStatborgAbandonne"), {
			Fichier: pkgObjectiveEvents + "statborg.go",
			Ancre:   "if len(comps) == 0 || !statCountersInDomain(comps) {",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "le volet statborg du registre des reports (toujours ouvert) ; a defaut, " + retraitRegle4,
		CritereRetrait:  "abandons comptes par cause et rapportes a la population lue ; retrait quand le compte est nul",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_composants_statborg_arretes",
		Fait:      "quels composants d'un enregistrement statborg sont lus",
		Mecanisme: "un composant non decodable arrete la boucle : les suivants du MEME enregistrement sont perdus sans trace",
		Condition: CondLectureNonPortee,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{Fichier: pkgObjectiveEvents + "statborg.go", Ancre: "return out, h1, true"}, {Fichier: pkgObjectiveEvents + "statborg.go", Ancre: "c.ComposantsArretes += arretes"}, siteDeVersement("NomComposantsStatborgArretes"), {
			Fichier: pkgObjectiveEvents + "statborg.go",
			Ancre:   "v, w, ok := decodeStatComponent(pay, q)",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "les composants manquants portes, archetype par archetype ; a defaut, " + retraitRegle4,
		CritereRetrait:  "0 arret en milieu d'enregistrement sur les 8 builds",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_manche_zero_decretee",
		Fait:      "quelles manches d'un match sont REELLES",
		Mecanisme: "aucune manche n'est admise par les deux criteres : la manche 0 est DECRETEE reelle pour que le film reste lisible",
		Condition: CondFilmMuet,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgObjectiveEvents + "statborg.go",
			Ancre:   "out[0] = true",
		}, {
			// LE COMPTE est emis par le calque de score de `replay` (inversion de dependance : `analysis`
			// rend une donnee pure, l appelant qui porte le compteur de la cuisson la compte). Site
			// inscrit a la fusion du lot 1.9.11 (2026-09-17), le ratchet des sites l exigeait.
			Fichier: "internal/games/halo_infinite/film/replay/score_timeline.go",
			Ancre:   "fb.Declenche(fallback.NomReplayMancheZeroDecretee)",
		}},
		DatePose: dateAudit0E,
		// RESSERRE PAR LE LOT 1.9.11 (2026-09-16), ex-`repli_manches_contigues_decretees`.
		// L'entree couvrait DEUX mecanismes : la regle d'ORDRE (« les manches admises doivent
		// etre contigues depuis 0 ») et le PLANCHER (« si aucune ne l'est, la manche 0 est
		// decretee »). Seul le second est un repli : il se declenche quand le film ne donne
		// AUCUNE manche admissible, donc sur un silence. La regle d'ordre, elle, se declenche
		// sur un DESACCORD avec une lecture disponible — D14 (b) interdit d'appeler cela un
		// repli : c'est une CONTRADICTION, et le lot la publie comme telle
		// (`coverage.score.roundsContradicted`).
		//
		// POURQUOI LA REGLE D'ORDRE N'EST PAS RETIREE, ALORS QUE L'ITEM 1.9.11 LE PREVOYAIT.
		// Mesure sur les 1 351 films du cache (instrument `e1911_manches_*_research_test.go`,
		// tableaux au §5 du plan) : la retirer ajoute une manche a 24 films, TOUS du motif
		// « designateur 2, manche 1 absente », et 23 des 24 ont fini de 38 a 442 s DANS leur
		// temps reglementaire sur un mode SANS manche. Les vraies prolongations, elles, sont
		// ecrites en designateur 1 CONTIGU et la chaine les publie deja.
		CibleRetrait:    "la revision a zero difference de la chaine des manches ; a defaut, " + retraitRegle4,
		CritereRetrait:  "0 film du corpus gate ou aucune manche n'est admise (le compteur publie en `coverage.fallbacks[]` le dit) ; retrait sec au jalon suivant si le compte reste nul",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_emission_hors_domaine_jetee",
		Fait:      "quelles emissions d'un compteur nomme entrent dans la serie, ET depuis le lot J8.5 (2026-09-27) dans la progression du compteur de morts que le pont par instants de mort deroule",
		Mecanisme: "valeur negative, ou canal B hors domaine du score de mode : l emission est jetee par un `continue`, dans les DEUX marches des series (par emplacement `rawSeriesByRound`, par table `rawSeriesByKey`, meme filtre `emissionHorsDomaine`) ; compte = emissions DISTINCTES (serie, instant) consultees par le document, quel que soit le nombre de lectures (lot J8.7-bis, 2026-09-28)",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgObjectiveEvents + "named_series.go",
			Ancre:   "func emissionHorsDomaine(key statSlotKey, v types.StatValue, val int64) bool {",
		}, {
			// LES DEUX MARCHES (`rawSeriesByRound`, `rawSeriesByKey`) notent la meme ligne.
			Fichier: pkgObjectiveEvents + "named_series.go",
			Ancre:   "cons.noterEmissionJetee(key, r)",
		}, {
			Fichier: pkgObjectiveEvents + "replis_a_la_consultation.go",
			Ancre:   "EmissionsHorsDomaineJetees: len(r.emissions),",
		}, siteDeVersement("NomEmissionHorsDomaineJetee")},
		DatePose:        dateAudit0E,
		CibleRetrait:    "la conversion des series nommees ; a defaut, " + retraitRegle4,
		CritereRetrait:  "emissions jetees comptees par cause ; une emission negative est un defaut d'alignement, pas une donnee a filtrer",
		CompteurBranche: true,
	},
	// `repli_manche_du_slot_sautee` (pose le 2026-09-13) A QUITTE LE REGISTRE AU LOT J8.6 DU PLAN DE
	// SUITE D AUDIT (2026-09-27, constat FO-4), AVEC SON CODE : la branche `len(kept) == 0` de
	// `cumulateRounds` (et sa jumelle de `SeriesByRound`) ne pouvait pas se prendre — une manche
	// n entre dans la table que par une emission, et `longestRun` rend au moins un point d une suite
	// non vide (preuve : `objectives/longest_run_non_vide_test.go`). Retrait pour absence de code
	// vivant, pas pour compte nul ; la question NE17 (`longestRun` ecarte-t-il des points reels ?)
	// reste celle du filtre, pas d un repli.
	{
		Nom:       "repli_table_identite_vide",
		Fait:      "le pont slot statborg -> joueur, par les instants de mort",
		Mecanisme: "aucune mort fournie : une table VIDE est rendue, indiscernable d'un pont qui n'a rien pu resoudre",
		Condition: CondSectionAbsente,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{Fichier: pkgObjectiveEvents + "slotidentity_deaths.go", Ancre: "c.TablesIdentiteVides++"}, siteDeVersement("NomTableIdentiteVide"), {
			Fichier: pkgObjectiveEvents + "slotidentity_deaths.go",
			Ancre:   "return map[int]string{}",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "la table du film donne le lien direct a ce calque ; a defaut, " + retraitRegle4,
		CritereRetrait:  "0 pont vide faute de morts sur les 8 builds",
		CompteurBranche: true,
	},
	// `repli_emission_du_compteur_de_morts_jetee` (pose le 2026-09-13) A QUITTE LE REGISTRE AU LOT
	// J8.5 DU PLAN DE SUITE D AUDIT (2026-09-27, constat FO-3), AVEC SON CODE : les deux gardes
	// propres du pont par instants de mort (`v.B < 0 || v.B > maxDeathsPerSlot`, a plat ET par
	// manche — le second site que l audit FO-4 disait manquant) ont disparu, le pont deroulant
	// desormais la serie PUBLIEE du compteur. Une emission negative y est jetee sous
	// `repli_emission_hors_domaine_jetee` (`named_series.go`), un pas aberrant par la borne par pas
	// de la serie (`boundSteps`). Retrait pour absence de code vivant, pas pour compte nul.
	{
		Nom:       "repli_debut_de_manche_au_minimum",
		Fait:      "l'instant de debut d'une manche",
		Mecanisme: "hors consensus : le MINIMUM des instants observes fait office de debut",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{Fichier: pkgObjectiveEvents + "slotidentity_rounds.go", Ancre: "c.DebutsDeMancheAuMinimum += len(min)"}, siteDeVersement("NomDebutDeMancheAuMinimum"), {
			Fichier: pkgObjectiveEvents + "slotidentity_rounds.go",
			Ancre:   "if cur, seen := min[r.Round]; !seen || r.TimeMS < cur {",
		}},
		DatePose: dateAudit0E,
		// DÉFAUT DÉJÀ MESURÉ (audit 0.E) : 213 s d.attribution fausse sur `24dbb67d`.
		// RECIBLE PAR LE LOT 1.9.11 (2026-09-16), et la raison est mesuree : ce repli ne depend
		// PAS de la regle d ordre des manches. Il se declenche quand le CONSENSUS DE SLOTS ne
		// fixe pas le debut d une manche, et le lot a mesure que ce consensus ne separe rien du
		// fait juge — les 41 designateurs materiels du corpus de verdict sont declares par les
		// DIX slots. Il vit donc avec le fait « l instant de debut d une manche », pas avec
		// « quelles manches sont reelles », et il suit la chaine des bornes.
		CibleRetrait:    "la chaine des bornes de manche lue au consensus ; a defaut, " + retraitRegle4,
		CritereRetrait:  "0 manche du corpus gate dont le debut vienne du minimum au lieu du consensus",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_slot_abandonne_au_premier_arrive",
		Fait:      "quel joueur occupe un slot statborg pour une manche, quand la feuille en propose plusieurs",
		Mecanisme: "premier arrive, premier servi : le slot deja attribue ou le joueur deja pris est abandonne",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{Fichier: pkgObjectiveEvents + "slotidentity_rounds.go", Ancre: "replis: ri.replis.Plus(ComptesDesReplis{SlotsAbandonnes: abandonnes})"}, siteDeVersement("NomSlotAbandonneAuPremierArrive"), {
			Fichier: pkgObjectiveEvents + "slotidentity_rounds.go",
			Ancre:   "if prev, deja := fusion[slot]; deja || pris[xuid] {",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "la table du film portee a ce calque ; a defaut, " + retraitRegle4,
		CritereRetrait:  "0 abandon par premier arrive sur les 8 builds",
		CompteurBranche: true,
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
		CibleComptage:   "aucune : outil hors production (cf. HorsProduction) — ni cuisson ni passe ou se compter",
		// OUTIL HORS PRODUCTION (lot J8.7, 2026-09-27, decision 5 du superviseur) : `objectives.Extract`
		// n a qu un appelant, l outil de diagnostic `cmd/diag_weapons_v3` (via `decfilm.Extract`) ; la
		// cuisson nomme ses actions par `NamedEventsFrom` sur la famille de `ObjectiveTypeOf`. L entree
		// reste inscrite (le code existe et decide dans l outil) et sort du ratchet des compteurs.
		HorsProduction: &OutilHorsProduction{
			Date:      date0927,
			Raison:    "le seul appelant de objectives.Extract est l outil de diagnostic cmd/diag_weapons_v3 ; aucune cuisson ni passe de synchronisation ne l execute",
			Symbole:   "Extract",
			Appelants: []string{"cmd/diag_weapons_v3"},
		},
	},
	{
		Nom:       "repli_assistant_non_resolu_abandonne",
		Fait:      "l'assistant d'un frag publie dans les episodes d'equipement",
		Mecanisme: "le nom de l'assistant ne se resout pas en xuid : le champ reste vide et le frag est publie sans assistant",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{Fichier: pkgReplaybuild + "kills.go", Ancre: "r.assistantsNonResolus++"}, {Fichier: pkgReplaybuild + "kills.go", Ancre: "fb.DeclencheN(decfilm.NomAssistantNonResoluAbandonne, r.assistantsNonResolus)"}, {
			Fichier: pkgReplaybuild + "kills.go",
			Ancre:   "ref.AssistXUID, ref.AssistKnown = aXUID, true",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "la table du film nomme les indices de l assistant ; a defaut, " + retraitRegle4,
		CritereRetrait:  "0 assistant non resolu sur les 8 builds",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_repere_neutre_generique_conserve",
		Fait:      "l'icone d'une mort neutre",
		Mecanisme: "nature non etablie par l'adaptateur d'assets : le fil garde son repere generique",
		Condition: CondSectionAbsente,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{Fichier: pkgReplaybuild + "replaybuild.go", Ancre: "fb.Declenche(decfilm.NomRepereNeutreGeneriqueConserve)"}, {
			Fichier: pkgReplaybuild + "replaybuild.go",
			Ancre:   "kind, img, ok := adapter.NeutralDeathIcon(d.Source.Tag)",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "completion de la table d'assets de morts neutres",
		CritereRetrait:  "0 mort neutre sans icone nommee sur le parc",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_relais_de_bot_abandonne",
		Fait:      "quel remplacant herite des vies anonymes d'un siege",
		Mecanisme: "le xuid n'a pas la forme bid(N.0) : le relais est abandonne (un humain est nomme par le fil des morts)",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{Fichier: pkgReplaybuild + "replaybuild.go", Ancre: "fb.Declenche(decfilm.NomRelaisDeBotAbandonne)"}, {
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
		CompteurBranche: true,
	},
	{
		Nom:       "repli_catalogue_de_zones_absent",
		Fait:      "l'etat des zones d'un match",
		Mecanisme: "le titre n'a pas de table d'objectifs : le rejeu est publie SANS aucun etat de zone",
		Condition: CondSectionAbsente,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{Fichier: pkgReplaybuild + "zones.go", Ancre: "fb.Declenche(decfilm.NomCatalogueDeZonesAbsent)"}, {
			Fichier: pkgReplaybuild + "zones.go",
			Ancre:   "titre sans table d'objectifs",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "aucune (degradation gracieuse multi-titre, `ErrCapabilityNotSupported`) ; le COMPTE est ce qui manque",
		CritereRetrait:  "coverage.zones distingue « titre sans table » de « carte hors catalogue » ; retrait sans objet",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_fraicheur_des_derivations_par_taille",
		Fait:      "les derivations d'un artefact sont-elles a jour",
		Mecanisme: "comparaison de la REVISION et de la TAILLE en octets de l'artefact — compromis assume, pas une empreinte",
		Condition: CondFilmMuet,
		Ordre:     OrdreSansLecture,
		Sites: []Site{{Fichier: "internal/sync/replayartifacts/derivations_backlog.go", Ancre: "fb.Declenche(decfilm.NomFraicheurDesDerivationsParTaille)"}, {
			Fichier: pkgReplaybuild + "derivations_index.go",
			Ancre:   "return m.Rev == DerivationsRev && int64(m.ArtifactBytes) == st.Size()",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "la recuisson selective par couche : une revision par calque remplace la taille ; a defaut, " + retraitRegle4,
		CritereRetrait:  "la fraicheur se juge sur la revision de calque portee par le document, jamais sur une taille",
		CompteurBranche: true,
	},
	{
		// POSE AU LOT L3 DES RETOURS REJEU (2026-09-23, constat L3-R8 de sa revue adverse). La regle
		// « le dernier numero porte les temps forts » valait pour TOUS les films avant le lot ; elle
		// ne vaut plus que pour un film charge sans manifeste — le constructeur l accepte, le fil
		// des morts s y replie. L inscrire ici est la regle commune du plan (§4.0) : un repli est
		// nomme, date, et porte son critere de retrait.
		Nom:       "repli_temps_forts_dernier_numero",
		Fait:      "quel morceau du film porte les temps forts (fil des morts, kill-feed) quand aucun morceau n'est type par un manifeste",
		Mecanisme: "le morceau de plus grand numero present dans le repertoire est lu comme celui des temps forts ; un seul morceau type par le manifeste court-circuite ce repli (selection par le type, finalise.EstTempsForts)",
		Condition: CondSectionAbsente,
		Ordre:     OrdreApresLecture,
		Sites: []Site{
			{Fichier: pkgFilmdec + "deaths_source.go", Ancre: "return nums[len(nums)-1], true, nil"},
			{Fichier: pkgReplaybuild + "filmfacts_cuisson.go", Ancre: "func jugerFilmSansManifeste("},
			// COMPTE : le verdict rendu par la lecture et note au rapport du contexte par l etage du
			// pont d identite — UNE lecture du fil par balayage de cuisson (lot J8.7).
			{Fichier: pkgFilmdec + "pont_identite.go", Ancre: "fc.NoterReplis(ComptesDesReplis{TempsFortsAuDernierNumero: unSi(auDernierNumero)})"},
			siteDeVersement("NomTempsFortsDernierNumero"),
		},
		DatePose:        "2026-09-23",
		CibleRetrait:    "le refus des films sans manifeste par la cuisson (replaybuild.jugerFilmSansManifeste), le jour ou replay-build et les instruments qui chargent un repertoire nu (grammar.ScanFilmDeaths, descendue de replay au lot J4.2) lisent aussi son manifeste",
		CritereRetrait:  "0 repertoire de morceaux sans manifeste au cache (1 625 sur 1 625 en portent un le 2026-09-23) et 0 WARN (film SANS manifeste) de la cuisson sur une republication complete du parc",
		CompteurBranche: true,
	},
	{
		// LOT J8.4 DU PLAN DE SUITE D AUDIT (2026-09-27), CONSTATS FO-1 / RA2-4 : la voie etait
		// publiee `deduit` SANS voie (`method` vide) et son compteur de retrait etait aveugle —
		// connu depuis l audit 0.E (13/09), jamais route vers un lot. Elle se publie desormais
		// sous `residu_de_manche` (`canonical.MethodRoundResidue`) et se compte ici.
		Nom:  "repli_identite_de_slot_par_residu_de_manche",
		Fait: "le joueur d un slot statborg MUET dans une manche (ni les instants de mort ni la feuille ne l ont nomme), qui porte les compteurs et les actions d objectif de ce slot",
		Mecanisme: "le residu de la feuille sur la manche (le total du joueur moins ce que les autres manches lui attribuent) est apparie au segment K/D/A du slot quand l appariement est unique DES DEUX COTES, " +
			"segment nul refuse, jamais contre une voie plus forte ; compte = couples (manche, slot) publies sous cette voie",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgObjectiveEvents + "slotidentity_residue.go",
			Ancre:   "out.origins[round][slot] = OriginRoundResidue",
		}, {
			Fichier: "internal/games/halo_infinite/film/replay/identity_registry_section.go",
			Ancre:   "fb.Declenche(fallback.NomIdentiteDeSlotParResiduDeManche)",
		}},
		DatePose:        date0927,
		CibleRetrait:    "retrait au jalon suivant si le compte est nul au corpus gate de J11 (regle 4 de D-10, 2026-09-27) ; sinon une lecture du film qui nomme le slot muet (instant de mort ou table d identite de la manche)",
		CritereRetrait:  "0 couple (manche, slot) publie sous `residu_de_manche` sur le corpus du gate de rejeu",
		CompteurBranche: true,
	},
}
