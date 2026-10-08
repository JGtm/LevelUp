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
		DatePose:     dateVague2,
		CibleRetrait: "la lecture des evenements de liste sur les builds anciens (profil par build) ; a defaut, " + retraitRegle4,
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
		// CIBLE REECRITE UNE SECONDE FOIS LE 2026-09-16 (revue de jalon M1, RONDE 2, constat
		// F2) : elle nommait le `replay-corpus-gate` comme confirmation sur le parc. CE GATE NE
		// PEUT PAS LA PRODUIRE — il lit les ARTEFACTS, et ce repli se declenche APRES la
		// cuisson, dans `BuildUsageSummary` ; son compte n'entre donc jamais dans
		// `coverage.fallbacks[]`. Les deux instruments qui le mesurent VRAIMENT sont nommes
		// ci-dessous.
		CibleRetrait: "la lecture qui borne chaque geste a une vie publiee du slot : le compte est NON NUL sur le parc " +
			"(0 sur les 8 builds, mais 163 declenchements sur 104 des 1 227 matchs de usage-summary, vague J11.4, " +
			"2026-10-02), pas de retrait sec ; a defaut, " + retraitRegle4,
		// DÉFAUT MESURÉ PUIS REFERMÉ. Audit 0.E, constat N-3 de REG-R2 : 32 à 95 % des poses d'un
		// film tombaient hors de toute fenêtre publiée (153/351, 443/466, 34/105 sur trois films).
		// MESURE DU 2026-09-16, compteur câblé, les 8 builds
		// (`replay/usage_summary_replis_test.go`, fixtures d'assemblage, aucun octet de film) :
		// **0 déclenchement sur 8/8** — le recollage des vies du lot 1.9.13 a fermé le défaut.
		//
		// LES DEUX INSTRUMENTS QUI MESURENT CE REPLI, nommes le 2026-09-16 (ronde 2, F2) :
		// `replay/usage_summary_replis_test.go` (fixtures d'assemblage, 8 builds, chiffre
		// reproductible) et LE JOURNAL DES PASSES sur le parc — `journaliserReplisUsage`
		// (post-sync) et `journaliserReplisUsageCorpus` (backfill CLI), qui ecrivent la ligne
		// de passe MEME A ZERO (« aucun »), sans quoi un retrait sec reposerait sur une
		// absence de trace. Le `replay-corpus-gate` n'en est PAS un : il lit les artefacts.
		CritereRetrait: "poses hors fenetre publiee a 0 : sur les 8 builds " +
			"(`replay/usage_summary_replis_test.go`) TENU le 2026-09-16 (0/8) ; sur le parc, " +
			"ligne `replis de la passe` des deux producteurs a `aucun` — NON TENU le 2026-10-02 (104 matchs, 163 declenchements)",
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
		CibleRetrait: "meme canal et meme mesure que repli_geste_dernier_occupant_du_match : NON NUL sur le parc " +
			"(5 declenchements sur 3 des 1 227 matchs de usage-summary, vague J11.4, 2026-10-02) ; a defaut, " + retraitRegle4,
		// MESURE DU 2026-09-16, compteur câblé, les 8 builds
		// (`replay/usage_summary_replis_test.go`) : **0 déclenchement sur 8/8**.
		// MEMES DEUX INSTRUMENTS que `repli_geste_dernier_occupant_du_match` (ronde 2, F2) :
		// les 8 builds de `replay/usage_summary_replis_test.go`, et la ligne `replis de la
		// passe` que les deux producteurs ecrivent sur le parc, « aucun » compris.
		CritereRetrait: "0 geste anterieur a la premiere vie du slot une fois les vies bornees " +
			"aux apparitions ecrites : sur les 8 builds TENU le 2026-09-16 (0/8) ; sur le parc, " +
			"ligne `replis de la passe` a `aucun` — NON TENU le 2026-10-02 (3 matchs, 5 declenchements)",
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
		CibleRetrait: "la reconciliation des trois canaux prises / utilises / laches (le compte est MESURE NON NUL : pas de retrait sec) ; a defaut, " + retraitRegle4,
		// D14 (b) : un désaccord entre deux lectures est une CONTRADICTION, pas un repli. Elle
		// doit se compter, jamais disparaître dans un `max(0, x)`.
		// MESURE DU 2026-09-16, compteur câblé, les 8 builds
		// (`replay/usage_summary_replis_test.go`) : **30 écrasements**, sur 7 des 8 builds
		// (seul `bcb6d393` en est exempt). C'est le PREMIER chiffre de cette contradiction :
		// jusqu'ici le clamp la faisait disparaître sans trace.
		// MEMES DEUX INSTRUMENTS (ronde 2, F2), et pour celui-ci le second compte double : le
		// chiffre des 8 builds est NON NUL, donc c'est la ligne `replis de la passe` du parc
		// qui dira si la reconciliation de M3 l'a referme.
		CritereRetrait: "0 ecrasement : la somme des trois canaux est alors coherente et la garde " +
			"peut tomber — NON TENU le 2026-09-16 (30 sur les 8 builds de " +
			"`replay/usage_summary_replis_test.go`) ; sur le parc, ligne `replis de la passe` — NON TENU le 2026-10-02 (2 984 ecrasements sur 847 des 1 227 matchs)",
		CompteurBranche: true,
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
		}, {
			Fichier: pkgReplay + "document_ground_weapon_items.go",
			Ancre:   "fb.Declenche(fallback.NomLienPriseArmeAbandonne)",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot de conversion du lien de prise (table C13 : trois hypotheses de lien natif REFUTEES — le repli restera, son COMPTE PAR CAUSE est ce qui manque)",
		CritereRetrait:  "les deux causes comptees separement dans GroundWeaponItemsCoverage ; le repli lui-meme est legitime tant que le negatif tient",
		CompteurBranche: true,
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
		}, {
			Fichier: pkgReplay + "document_ability_impulses.go",
			Ancre:   "in.fb.DeclencheN(fallback.NomImpulsionFusionneeDansLeGeste, len(in.reads)-len(episodes))",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "aucune tant que le film ne borne pas un geste ; le COMPTE des fusions est ce qui manque",
		CritereRetrait:  "fusions comptees dans AbilityImpulseCoverage ; retrait si le film s'avere porter une borne de geste",
		CompteurBranche: true,
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
		}, {
			Fichier: pkgReplay + "document_ability_impulses.go",
			Ancre:   "idx.fb.Declenche(fallback.NomRangCapaciteVieElargie)",
		}},
		DatePose: dateAudit0E,
		// CIBLE REECRITE LE 2026-09-16 (revue de jalon M1). Le lot 1.9.13, fusionne le
		// 2026-09-15, a REDUIT le besoin sans le fermer — gain collateral MESURE aux goldens :
		// impulsions `sans identite` 4 -> 3 sur `000d5950` et 2 -> 0 sur `11de8353`, charges
		// 20 -> 13 et 8 -> 1. Le compteur n'a pas ete cable au passage, donc le RESIDU n'a pas
		// de chiffre a lui.
		CibleRetrait:    "mesurer le residu du compteur desormais cable ; a defaut, " + retraitRegle4,
		CritereRetrait:  "0 elargissement necessaire sur le parc — residu MESURE NON NUL le 2026-10-02 : 2 matchs / 4 elargissements sur 1 227 artefacts (.ai/V7.5/MESURES_PARC_REPLIS_NULS_2026-10-02.md)",
		CompteurBranche: true,
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
		}, {
			Fichier: pkgReplay + "bomb_carries.go",
			Ancre:   "clock.fb.DeclencheN(fallback.NomPorteurAnonymeSansFinParMort, carry.replis.porteurAnonyme)",
		}},
		DatePose:     dateAudit0E,
		CibleRetrait: "le porteur du crane lu au canal des armes tenues, et le registre d identite par la table du film ; a defaut, " + retraitRegle4,
		// Décision utilisateur du 2026-09-06 : « les vies anonymes n'existent pas ». Un porteur
		// sans xuid est un défaut de pont, pas une catégorie de donnée.
		CritereRetrait:  "0 porteur sans xuid sur les 8 builds",
		CompteurBranche: true,
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
		}, {
			Fichier: pkgReplay + "bomb_carries.go",
			Ancre:   "clock.fb.DeclencheN(fallback.NomPortageFermeALaPriseSuivante, carry.replis.priseSuivante)",
		}},
		DatePose:       dateAudit0E,
		CibleRetrait:   "le porteur du crane lu au canal des armes tenues ; a defaut, " + retraitRegle4,
		CritereRetrait: "question NE7 de la table (D) de l'audit instruite : combien de morts de porteur sont suivies d'une emission du canal ? Le repli tombe si le canal emet",
		// Le négatif « la mort ferme SANS émission » est une AFFIRMATION sans chiffre
		// (`held_object_carry.go:20-22`), pas une mesure : c'est pourquoi la condition est
		// `film_muet` mais que le critère de retrait est une mesure à faire.
		CompteurBranche: true,
	},
	{
		Nom:       "repli_plafond_grenade_par_defaut",
		Fait:      "le plafond de grenades porte par un joueur, qui borne la lecture d'inventaire",
		Mecanisme: "appelant sans plafond : DefaultGrenadeMax (2) s'applique, quel que soit le mode et la carte",
		Condition: CondInconditionnel,
		Ordre:     OrdreSansLecture,
		// DEUX SITES DEPUIS LE LOT J4.2 (2026-09-26) : la lecture d inventaire est descendue en
		// `grammar`, qui APPLIQUE le defaut et ne compte pas (ADR 0034 D-4) ; l appelant de
		// production, `replay`, le COMPTE sur la meme condition (plafond non fourni).
		Sites: []Site{
			{Fichier: pkgFilmdec + "inventory_decode.go", Ancre: "grenMax = DefaultGrenadeMax"},
			{Fichier: pkgReplay + "film_scan.go", Ancre: "s.opt.Fallbacks.Declenche(fallback.NomPlafondGrenadeParDefaut)"},
		},
		DatePose:        dateAudit0E,
		CibleRetrait:    "un plafond lu comme une donnee de mode (profil par build et par carte), pas une constante ; a defaut, " + retraitRegle4,
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
			// La decision : l entree de la carte ne porte pas ses trois largeurs (pose du contexte
			// de carte, geste unique de la grammaire).
			Fichier: pkgFilmdec + "contexte_de_carte.go",
			Ancre:   "if e.AxisWidths[0] == 0 || e.AxisWidths[1] == 0 || e.AxisWidths[2] == 0 {",
		}, {
			// Le compte et le journal, chez l appelant de la cuisson.
			Fichier: pkgReplay + "world_object_precision.go",
			Ancre:   "if pose.LargeursAbsentes {",
		}},
		DatePose:     dateAudit0E,
		CibleRetrait: "les largeurs par carte et par build, donnees de profil ; a defaut, " + retraitRegle4,
		// DÉFAUT DÉJÀ MESURÉ (audit 0.E) : le défaut conservé est celui d'UNE carte, appliqué
		// à toutes ; l'écart n'était que journalisé (slog.Warn), jamais compté.
		CritereRetrait:  "0 film cuit sur les largeurs par defaut ; le catalogue porte les largeurs de toutes les cartes du parc",
		CompteurBranche: true,
	},
	{
		// LOT J8.3 DU PLAN DE SUITE D AUDIT (2026-09-27), CONSTAT RB2-8 : la fenetre decidait
		// l origine et le lacheur d un objet au sol sans etre inscrite (D-10 regle 1). Elle est la
		// meme que celle des poses d equipement (`originDropWindowUS`, `originDropMaxDist`), qui
		// n y entre qu APRES l evenement natif ; ici aucune lecture ne la precede.
		Nom:  "repli_origine_au_sol_lachee_par_fenetre",
		Fait: "l origine `dropped` d un objet au sol (arme ou power-up) et son lacheur, publies par `weaponPads`, `coverage.groundWeapons` et le calque des armes au sol",
		Mecanisme: "une vie de bipede s acheve a moins de 200 ms (originDropWindowUS) et 1,5 m (originDropMaxDist) de la creation : l objet est classe `dropped`, le plus petit slot en fenetre est le lacheur ; " +
			"compte = apparitions ainsi classees, deux voies reunies",
		// LA QUESTION EST OUVERTE : aucun negatif n est mesure (on ne sait pas si le film ecrit
		// le lacher d une arme comme il ecrit la pose d un equipement, evenement 103). La
		// condition retenue est donc la DETTE — un lecteur a trouver —, pas `film_muet`.
		Condition: CondLectureNonPortee,
		Ordre:     OrdreSansLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "ground_weapon_rules.go",
			Ancre:   "func gwPadsClass(lives map[uint32][]equipLife, a gwPadApparition) (string, int) {",
		}, {
			Fichier: pkgReplay + "ground_weapon_pads.go",
			Ancre:   "clock.fb.DeclencheN(fallback.NomOrigineAuSolLacheeParFenetre, wc.dropped+pc.dropped)",
		}},
		DatePose:        date0927,
		CibleRetrait:    "conversion le jour ou une lecture du lacher est trouvee (evenement natif ou composant de l objet) ; a defaut, retrait au jalon suivant si le compte est nul au corpus gate de J11 (regle 4 de D-10, 2026-09-27)",
		CritereRetrait:  "origine lue dans le film pour chaque objet au sol publie `dropped`, ET 0 declenchement sur le corpus du gate de rejeu",
		CompteurBranche: true,
	},
}
