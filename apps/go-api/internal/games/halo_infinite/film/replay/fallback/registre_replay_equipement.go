package fallback

// registre_replay_equipement.go — les replis du calque ÉQUIPEMENT, ARMES AU SOL, CAPACITÉS et
// INVENTAIRE (`internal/games/halo_infinite/film/replay/`).

const pkgReplay = "internal/games/halo_infinite/film/replay/"

var registreReplayEquipement = []Repli{
	{
		Nom:       "repli_origine_pose_vie_la_plus_proche",
		Fait:      "l'origine d'une pose d'equipement (deployed / dropped) quand aucune vie du poseur ne couvre l'instant de la pose",
		Mecanisme: "la vie du poseur dont l'ecart temporel a la pose est le plus faible est choisie, puis l'ecart est confronte a originDropWindowUS (200 ms)",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "equipment_placements.go",
			Ancre:   "best, bestGap := equipLife{}, ^uint64(0)",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.9.1 (origine d'une pose d'equipement)",
		CritereRetrait:  "0 declenchement sur l'echantillon court une fois la pose designee par le type 103 lue ; les 216 panneaux de mur sont deja couverts par equipmentIsSpawnedPiece",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_origine_pose_fenetre_temporelle",
		Fait:      "l'origine d'une pose d'un appareil PORTE (capteur, traqueur, ecran, champ)",
		Mecanisme: "ecart entre la creation de l'objet et le dernier point du poseur : <= originDropWindowUS (200 ms) = lacher, au-dela = deploiement",
		Condition: CondFilmMuet,
		Ordre:     OrdreSansLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "equipment_placements.go",
			Ancre:   "const originDropWindowUS = 200_000",
		}},
		DatePose:       dateAudit0E,
		CibleRetrait:   "lot 1.9.1, puis la piste ref0 de la table (D) du registre 0.E",
		CritereRetrait: "un signal ECRIT qui separe porte-deploye de porte-lache ; a ce jour le negatif est mesure (103 sur 0/31 deployed et 0/145 dropped, porte de creation ti=37 fermee sur 503/503) — le repli reste jusqu'a ce qu'une lecture existe",
		// Le compteur ne se branche pas ici : c'est le lot 1.9.1 qui ouvre ce fichier pour y
		// poser la lecture du 103 en contrôle, et qui comptera accord / repli / contradiction.
		CompteurBranche: false,
		CibleComptage:   "lot 1.9.1",
	},
	{
		Nom:       "repli_geste_dernier_occupant_du_match",
		Fait:      "a quel joueur crediter un geste (traction, episode, pose) pose sur un slot",
		Mecanisme: "aucune vie du slot ne couvre l'image : le DERNIER occupant connu du slot sur tout le match est credite",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "usage_summary.go",
			Ancre:   "return o.dernier[slot]",
		}},
		DatePose:     dateAudit0E,
		CibleRetrait: "lot 1.9.13 (une vie finit a une mort ECRITE), qui borne les vies sur les morts du film",
		// DÉFAUT DÉJÀ MESURÉ (audit 0.E, constat N-3 de REG-R2) : 32 à 95 % des poses d'un film
		// tombent hors de toute fenêtre publiée (153/351, 443/466, 34/105 sur trois films).
		CritereRetrait:  "poses hors fenetre publiee a 0 sur les 8 builds apres le decoupage des vies aux morts ecrites",
		CompteurBranche: false,
		CibleComptage:   "lot 1.9.13 (BuildUsageSummary lit un document DEJA cuit : son compteur sort de la couverture de cuisson et demande son propre canal)",
	},
	{
		Nom:       "repli_geste_premiere_vie_du_slot",
		Fait:      "a quel joueur crediter un geste date AVANT la premiere vie publiee du slot",
		Mecanisme: "la PREMIERE vie du slot est creditee, faute de vie precedente",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "usage_summary.go",
			Ancre:   "return vies[0].xuid",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.9.13",
		CritereRetrait:  "0 geste anterieur a la premiere vie du slot une fois les vies bornees aux apparitions ecrites",
		CompteurBranche: false,
		CibleComptage:   "lot 1.9.13 (meme canal que repli_geste_dernier_occupant_du_match)",
	},
	{
		Nom:       "repli_garde_equipement_negatif_a_zero",
		Fait:      "le nombre d'equipements GARDES par un joueur pour une famille",
		Mecanisme: "prises - utilises - laches, ecrase a 0 quand la soustraction est negative",
		Condition: CondContradiction,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "usage_summary_outcomes.go",
			Ancre:   "kept := taken - usageUsedOf(t, family) - t.DroppedByFamily[family]",
		}},
		DatePose:     dateAudit0E,
		CibleRetrait: "lot 1.9.13, puis retrait sec si le compte reste nul",
		// D14 (b) : un désaccord entre deux lectures est une CONTRADICTION, pas un repli. Elle
		// doit se compter, jamais disparaître dans un `max(0, x)`.
		CritereRetrait:  "0 ecrasement sur les 8 builds : la somme des trois canaux est alors coherente et la garde peut tomber",
		CompteurBranche: false,
		CibleComptage:   "lot 1.9.13 (meme canal : hors cuisson)",
	},
	{
		Nom:       "repli_traction_vie_du_tir",
		Fait:      "a quelle vie rattacher une traction de grappin",
		Mecanisme: "aucune vie ne couvre l'ACCROCHE : la vie qui couvre le TIR decide",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "grapple_lines.go",
			Ancre:   "track = lifeCovering(vies, t0)",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.9.13 (les trous de replication cessent de couper les vies)",
		CritereRetrait:  "0 traction dont l'accroche tombe hors de toute vie sur les 8 builds",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_traction_vie_la_plus_proche",
		Fait:      "a quelle vie rattacher une traction dont ni l'accroche ni le tir ne sont couverts",
		Mecanisme: "la vie la plus proche en temps est retenue, la fenetre est ensuite bornee par la vie",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "grapple_lines.go",
			Ancre:   "track = lifeNearest(vies, tAttach)",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.9.13",
		CritereRetrait:  "0 traction sans vie couvrante sur les 8 builds",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_famille_arme_identifiant_brut",
		Fait:      "le nom de la famille d'arme d'un socle",
		Mecanisme: "le catalogue ne nomme pas l'identifiant : la valeur brute 0x%08x sert de nom",
		Condition: CondFilmMuet,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "ground_weapon_rules.go",
			Ancre:   "return fmt.Sprintf(\"0x%08x\", w)",
		}},
		DatePose:     dateAudit0E,
		CibleRetrait: "aucune tant que le catalogue d'armes est incomplet ; retrait sec des que le compte est nul sur le parc",
		// Ce repli-ci est HONNÊTE (il ne fabrique aucun nom) ; ce qui manque est son COMPTE :
		// un artefact ne dit pas combien de ses familles sont des identifiants bruts.
		CritereRetrait:  "0 famille rendue sous forme 0x%08x sur le parc",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_lien_prise_arme_abandonne",
		Fait:      "quel objet au sol une prise d'arme consomme",
		Mecanisme: "famille absente, ou aucune position d'acteur a +-250 ms : le lien est abandonne SANS distinguer les deux causes",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "document_ground_weapon_items.go",
			Ancre:   "actor, ok := gwItemActorAt(bySlot, ch.Slot, ch.TimestampUS)",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot de conversion du lien de prise (table C13 : trois hypotheses de lien natif REFUTEES — le repli restera, son COMPTE PAR CAUSE est ce qui manque)",
		CritereRetrait:  "les deux causes comptees separement dans GroundWeaponItemsCoverage ; le repli lui-meme est legitime tant que le negatif tient",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_impulsion_fusionnee_dans_le_geste",
		Fait:      "deux lectures d'impulsion de capacite sont-elles UN geste ou DEUX",
		Mecanisme: "ecart <= abilityImpulseEpisodeGapUS (1 s) depuis la DERNIERE lecture : la seconde est absorbee",
		Condition: CondFilmMuet,
		Ordre:     OrdreSansLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "document_ability_impulses.go",
			Ancre:   "last[r.Slot] = r.TimestampUS",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "aucune tant que le film ne borne pas un geste ; le COMPTE des fusions est ce qui manque",
		CritereRetrait:  "fusions comptees dans AbilityImpulseCoverage ; retrait si le film s'avere porter une borne de geste",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_rang_capacite_vie_elargie",
		Fait:      "de quel equipement parle une impulsion ou une charge, quand l'instant tombe en bord de vie",
		Mecanisme: "les bornes de la vie sont elargies de lifeGapUS (5 s) des deux cotes avant de chercher le rang i48",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "document_ability_impulses.go",
			Ancre:   "if int64(at)+lifeGapUS >= l.from && int64(at) <= l.to+lifeGapUS {",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.9.13 (les vies cessent d'etre coupees par un trou de replication, donc les bords disparaissent)",
		CritereRetrait:  "0 elargissement necessaire sur les 8 builds",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_porteur_anonyme_sans_fin_par_mort",
		Fait:      "un portage d'objet tenu s'est-il termine par la mort de son porteur",
		Mecanisme: "porteur sans xuid : FinParMort est affirme FAUX, sans distinguer « pas de mort » de « porteur inconnu »",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "held_object_carry.go",
			Ancre:   "if p.XUID == 0 {",
		}},
		DatePose:     dateAudit0E,
		CibleRetrait: "lot 1.9.5 (le porteur du crane lu au canal des armes tenues) et le registre d'identite 1.6",
		// Décision utilisateur du 2026-09-06 : « les vies anonymes n'existent pas ». Un porteur
		// sans xuid est un défaut de pont, pas une catégorie de donnée.
		CritereRetrait:  "0 porteur sans xuid sur les 8 builds",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_portage_ferme_a_la_prise_suivante",
		Fait:      "quand se termine un portage d'objet tenu",
		Mecanisme: "aucune mort du porteur dans l'intervalle : la periode est fermee a la prise suivante, fut-elle d'un AUTRE slot",
		Condition: CondFilmMuet,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "held_object_carry.go",
			Ancre:   "func premiereMortDans(mortsDe map[uint64][]int, p HeldObjectPeriod, avant int) (int, bool) {",
		}},
		DatePose:       dateAudit0E,
		CibleRetrait:   "lot 1.9.5",
		CritereRetrait: "question NE7 de la table (D) de l'audit instruite : combien de morts de porteur sont suivies d'une emission du canal ? Le repli tombe si le canal emet",
		// Le négatif « la mort ferme SANS émission » est une AFFIRMATION sans chiffre
		// (`held_object_carry.go:20-22`), pas une mesure : c'est pourquoi la condition est
		// `film_muet` mais que le critère de retrait est une mesure à faire.
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_plafond_grenade_par_defaut",
		Fait:      "le plafond de grenades porte par un joueur, qui borne la lecture d'inventaire",
		Mecanisme: "appelant sans plafond : DefaultGrenadeMax (2) s'applique, quel que soit le mode et la carte",
		Condition: CondInconditionnel,
		Ordre:     OrdreSansLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "inventory_decode.go",
			Ancre:   "grenMax = DefaultGrenadeMax",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 3.x (profil par build et par carte) : un plafond est une donnee de mode, pas une constante",
		CritereRetrait:  "le plafond vient du manifeste de mode ; 0 recours au defaut sur les 8 builds",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_largeurs_axe_par_defaut_conservees",
		Fait:      "les largeurs de dequantification des objets du monde pour CE film",
		Mecanisme: "entree de catalogue sans largeurs : les largeurs par defaut sont conservees — et ce sont celles de Cliffhanger",
		Condition: CondSectionAbsente,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "world_object_precision.go",
			Ancre:   "if e.AxisWidths[0] == 0 || e.AxisWidths[1] == 0 || e.AxisWidths[2] == 0 {",
		}},
		DatePose:     dateAudit0E,
		CibleRetrait: "lot 3.4 (largeurs par carte et par build, donnees de profil)",
		// DÉFAUT DÉJÀ MESURÉ (audit 0.E) : le défaut conservé est celui d'UNE carte, appliqué
		// à toutes ; l'écart n'était que journalisé (slog.Warn), jamais compté.
		CritereRetrait:  "0 film cuit sur les largeurs par defaut ; le catalogue porte les largeurs de toutes les cartes du parc",
		CompteurBranche: true,
	},
}
