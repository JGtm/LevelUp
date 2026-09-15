package fallback

// registre_replay_equipement.go — les replis du calque ÉQUIPEMENT, ARMES AU SOL, CAPACITÉS et
// INVENTAIRE (`internal/games/halo_infinite/film/replay/`).

const pkgReplay = "internal/games/halo_infinite/film/replay/"

var registreReplayEquipement = []Repli{
	{
		Nom:       "repli_piece_engendree_sans_evenement",
		Fait:      "l'origine d'une PIECE ENGENDREE (panneau de mur) qu'aucun evenement 103 ne designe",
		Mecanisme: "le manifeste du titre dit `kind = \"deployed\"` : l'objet n'existe QUE deploye, donc la pose sort `deployed` sans mesure",
		Condition: CondFilmMuet,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "equipment_origin.go",
			Ancre:   "fb.Declenche(fallback.NomPieceEngendreeSansEvenement)",
		}},
		DatePose:     "2026-09-15",
		CibleRetrait: "lot 3.x (profil par build) : la lecture des evenements de liste sur les builds anciens",
		// MESURE DU 2026-09-15 : 9 poses de panneau sur 124, et TOUTES sur les deux films de
		// build les plus anciens du corpus — `a521164d` (HI_1_4_1, 0 evenement 103 lu sur
		// 4 956 listes) et `50247b26` (v31 sans section, 2 evenements). C'est une limite de
		// BUILD, pas une incertitude : sur les builds recents la designation est de 115/115.
		CritereRetrait:  "0 declenchement sur les 8 builds, c'est-a-dire un evenement 103 lu pour chaque panneau publie",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_geste_dernier_occupant_du_match",
		Fait:      "a quel joueur crediter un geste (traction, episode, pose) pose sur un slot",
		Mecanisme: "aucune vie du slot ne couvre l'image : le DERNIER occupant connu du slot sur tout le match est credite",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "usage_summary_owners.go",
			Ancre:   "func (o usageOwners) repliDernierOccupant(slot uint32) string {",
		}},
		DatePose: dateAudit0E,
		// CIBLE REECRITE LE 2026-09-16 (revue de jalon M1) : elle nommait le lot 1.9.13, FUSIONNE
		// depuis le 2026-09-15. Le lot a bien fait son travail — il est ce qui rend le critere
		// MESURABLE et il le tient — mais le repli n'a pas ete retire avec lui.
		CibleRetrait: "retrait sec a M2 (D14 d) : le compte est DEJA nul, le corpus gate de cloture de M1 le confirme ou l'infirme sur le parc",
		// DÉFAUT MESURÉ PUIS REFERMÉ. Audit 0.E, constat N-3 de REG-R2 : 32 à 95 % des poses d'un
		// film tombaient hors de toute fenêtre publiée (153/351, 443/466, 34/105 sur trois films).
		// MESURE DU 2026-09-16, compteur câblé, les 8 builds
		// (`replay/usage_summary_replis_test.go`, fixtures d'assemblage, aucun octet de film) :
		// **0 déclenchement sur 8/8** — le recollage des vies du lot 1.9.13 a fermé le défaut.
		CritereRetrait:  "poses hors fenetre publiee a 0 sur les 8 builds apres le decoupage des vies aux morts ecrites : TENU le 2026-09-16 (0/8)",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_geste_premiere_vie_du_slot",
		Fait:      "a quel joueur crediter un geste date AVANT la premiere vie publiee du slot",
		Mecanisme: "la PREMIERE vie du slot est creditee, faute de vie precedente",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "usage_summary_owners.go",
			Ancre:   "func (o usageOwners) repliPremiereVieDuSlot(vies []usageVie) string {",
		}},
		DatePose: dateAudit0E,
		// CIBLE REECRITE LE 2026-09-16 (revue de jalon M1) : elle nommait le seul lot 1.9.13,
		// fusionne le 2026-09-15.
		CibleRetrait: "retrait sec a M2 (D14 d) : meme canal et meme mesure que repli_geste_dernier_occupant_du_match",
		// MESURE DU 2026-09-16, compteur câblé, les 8 builds
		// (`replay/usage_summary_replis_test.go`) : **0 déclenchement sur 8/8**.
		CritereRetrait:  "0 geste anterieur a la premiere vie du slot une fois les vies bornees aux apparitions ecrites : TENU le 2026-09-16 (0/8)",
		CompteurBranche: true,
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
		DatePose: dateAudit0E,
		// CIBLE REECRITE LE 2026-09-16 (revue de jalon M1) : elle nommait le lot 1.9.13, fusionne
		// le 2026-09-15 — et ce lot ne pouvait PAS la fermer : il recolle des vies, il ne
		// reconcilie pas trois canaux d'equipement entre eux.
		CibleRetrait: "M2 puis M3 (reconciliation des trois canaux prises / utilises / laches) ; PAS de retrait sec : le compte est MESURE NON NUL",
		// D14 (b) : un désaccord entre deux lectures est une CONTRADICTION, pas un repli. Elle
		// doit se compter, jamais disparaître dans un `max(0, x)`.
		// MESURE DU 2026-09-16, compteur câblé, les 8 builds
		// (`replay/usage_summary_replis_test.go`) : **30 écrasements**, sur 7 des 8 builds
		// (seul `bcb6d393` en est exempt). C'est le PREMIER chiffre de cette contradiction :
		// jusqu'ici le clamp la faisait disparaître sans trace.
		CritereRetrait:  "0 ecrasement sur les 8 builds : la somme des trois canaux est alors coherente et la garde peut tomber — NON TENU le 2026-09-16 (30 sur 8 builds)",
		CompteurBranche: true,
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
		DatePose: dateAudit0E,
		// CIBLE REECRITE LE 2026-09-16 (revue de jalon M1) : elle nommait le lot 1.9.13, fusionne
		// le 2026-09-15 sans avoir cable ce compteur — sa frequence reste donc INCONNUE, et un
		// zero n'y serait pas lisible (cf. [Repli.CompteurBranche]).
		CibleRetrait:    "M2, pas 2 (les lecteurs recoivent le profil) : cabler le compteur au meme geste, puis retrait sec si le compte est nul",
		CritereRetrait:  "0 traction dont l'accroche tombe hors de toute vie sur les 8 builds — NON MESURE (compteur non cable au 2026-09-16)",
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
		DatePose: dateAudit0E,
		// CIBLE REECRITE LE 2026-09-16 (revue de jalon M1) : meme raison que
		// repli_traction_vie_du_tir, meme geste de cablage.
		CibleRetrait:    "M2, pas 2 : cabler le compteur puis retrait sec si le compte est nul",
		CritereRetrait:  "0 traction sans vie couvrante sur les 8 builds — NON MESURE (compteur non cable au 2026-09-16)",
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
		DatePose: dateAudit0E,
		// CIBLE REECRITE LE 2026-09-16 (revue de jalon M1). Le lot 1.9.13, fusionne le
		// 2026-09-15, a REDUIT le besoin sans le fermer — gain collateral MESURE aux goldens :
		// impulsions `sans identite` 4 -> 3 sur `000d5950` et 2 -> 0 sur `11de8353`, charges
		// 20 -> 13 et 8 -> 1. Le compteur n'a pas ete cable au passage, donc le RESIDU n'a pas
		// de chiffre a lui.
		CibleRetrait:    "M2, pas 2 : cabler le compteur, mesurer le residu, puis retrait sec s'il est nul",
		CritereRetrait:  "0 elargissement necessaire sur les 8 builds — RESIDU NON MESURE (compteur non cable au 2026-09-16 ; le 1.9.13 a fait tomber les grandeurs voisines sans les annuler)",
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
