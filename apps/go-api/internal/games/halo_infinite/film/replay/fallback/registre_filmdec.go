package fallback

// registre_filmdec.go — les replis des INFÉRENCES de `filmdec` qui décident en production.
//
// PÉRIMÈTRE, TEL QUE L'AUDIT 0.E L'A FERMÉ : la GRAMMAIRE de `filmdec` (largeurs de composants,
// désérialiseurs, états par défaut) est HORS AXE — elle vient du jeu et se corrige chez
// l'écrivain, pas par un repli. N'entrent ici que les fonctions qui DÉCIDENT à la place d'une
// lecture : découpage d'i0, bande de slots, largeurs installées, comblements, abandons de
// chunks.
//
// CONTRAINTE DE CÂBLAGE : le ratchet `archlint/filmdec_package_vars_test.go` gèle le nombre de
// variables de paquet de `filmdec`, et le critère S1 du plan vise ZÉRO variable mutable dans le
// décodeur. Un compteur de replis ne s'y câble donc pas par une globale : il passera par le
// `FilmContext` (D1 : un seul objet par film, le profil résolu à la construction) au pas 2 de
// M2, qui est le lot où les lecteurs reçoivent ce porteur.

const pkgFilmdec = "internal/games/halo_infinite/film/filmdec/"

// comptageParFilmContext : la raison, écrite une fois, du câblage différé des replis `filmdec`.
const comptageParFilmContext = "pas 2 de M2 (les lecteurs recoivent le profil via FilmContext : le seul porteur par film sans variable de paquet, cf. S1)"

var registreFilmdec = []Repli{
	{
		Nom:       "repli_i0_porte_et_region_par_defaut",
		Fait:      "la largeur de la porte d'i0 et l'index de region attendu, quand AUCUNE entree de catalogue ne les impose",
		Mecanisme: "GateBits est force a DefaultI0GateBits (5) et Region a 0 : l'auto-detection ne sait pas voir un index de region de plus d'un bit",
		Condition: CondCarteAbsenteDuCatalogue,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgFilmdec + "i0_layout.go",
			Ancre:   "GateBits: DefaultI0GateBits,",
		}},
		DatePose: dateAudit0E,
		// RETROGRADE AU LOT 1.9.2 (2026-09-15), `inconditionnel / devant_la_lecture` ->
		// `carte_absente_du_catalogue / apres_lecture`, ET LE RATCHET DES SEPT DESCEND A SIX.
		//
		// CE QUI A CHANGE, SUR PIECES. Les deux chemins de `sync/killcollector` construisaient
		// leurs reglages de balayage avec `ScanFilmOptions.Layout` nil, donc `DetectI0LayoutOf`
		// DECIDAIT du decoupage — devant un catalogue qui, lui, savait
		// (`positions.go` tenait deja l'entree de carte ; `hits.go` la retrouvait par signature
		// et n'en gardait que les bornes). Ils imposent desormais `MapQuantEntry.Layout()`,
		// comme le chemin de cuisson le fait depuis le 2026-09-03. La lecture du CATALOGUE
		// precede donc partout, et ce repli n'entre plus que la ou le catalogue se tait.
		//
		// CE QUI SUBSISTE, ET POURQUOI L'ENTREE NE SORT PAS. `DetectI0Layout` a un dernier
		// appelant de production : `filmdec/weapon_hit_distance_resolver.go`, ou la signature de
		// largeurs sert a IDENTIFIER la carte (le fait decide y est la carte, pas le decoupage —
		// `repli_distances_de_touche_desactivees` en porte la consequence). Le lot 1.9.4 le
		// supprime ; c'est lui qui pourra retirer cette entree.
		//
		// MESURE DU LOT 1.9.2 (17 films, 14 temoins du corpus gate + les 8 builds) : catalogue et
		// auto-detection donnent le MEME decoupage sur 15 films ; les deux films Live Fire
		// divergent (`gate=6 region=1 12/12/11` contre `gate=5 region=0 13/12/11`), et la porte
		// de region du catalogue ecarte 26 enregistrements sur 267 400 (`60ae07c4`) et 11 sur
		// 146 860 (`0797ce72`) qui appartiennent a une AUTRE region de compression.
		CibleRetrait:    "lot 1.9.4 (la carte du film vient du nom de match : dernier appel de production de DetectI0Layout) puis lot 3.x (profil par carte)",
		CritereRetrait:  "aucun chemin de production n'appelle DetectI0LayoutOf ni DetectI0Layout ; le decoupage vient du catalogue sur les 8 builds",
		CompteurBranche: false,
		CibleComptage:   lot194,
	},
	{
		Nom:       "repli_bande_bipede_comblee",
		Fait:      "quels slots de replication sont des bipedes de joueur",
		Mecanisme: "la bande observee est COMBLEE sur tout [min, max], jusqu'au domaine de 13 bits",
		Condition: CondInconditionnel,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgFilmdec + "offline_biped_band.go",
			Ancre:   "func fillSlotBand(s map[uint32]bool) SlotBand { return NewSlotBand(filledSlotMap(s)) }",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 3.5 (la bande de slots bipede par build)",
		CritereRetrait:  "la bande vient du profil du build ; 0 comblement sur les 8 builds",
		CompteurBranche: false,
		CibleComptage:   "lot 3.5",
	},
	{
		Nom:       "repli_composant_hors_table_parametre_un",
		Fait:      "le parametre d'etat d'un composant absent de la table mesuree",
		Mecanisme: "la valeur 1 est rendue par defaut (0 etait un choix « conservateur » jamais mesure, et faux)",
		Condition: CondLectureNonPortee,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgFilmdec + "traverse.go",
			Ancre:   "func paramForComponent(name string) uint32 {",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 3.6 (les composants manquants, archetype par archetype)",
		CritereRetrait:  "0 composant hors table sur les 8 builds (la table `ecs_table.tsv` couvre alors tout ce que les films portent)",
		CompteurBranche: false,
		CibleComptage:   comptageParFilmContext,
	},
	{
		Nom:       "repli_largeurs_monde_par_defaut_conservees",
		Fait:      "les largeurs d'axe des objets du monde installees pour ce film",
		Mecanisme: "layout non detecte : le defaut est conserve plutot que d'installer des zeros",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgFilmdec + "traverse.go",
			Ancre:   "if l.AxisW[0] == 0 || l.AxisW[1] == 0 || l.AxisW[2] == 0 {",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 3.4 (les largeurs sont une donnee de la carte et du build)",
		CritereRetrait:  "les largeurs viennent du profil ; 0 conservation du defaut sur les 8 builds",
		CompteurBranche: false,
		CibleComptage:   comptageParFilmContext,
	},
	{
		Nom:       "repli_index_de_region_largeur_un",
		Fait:      "la largeur de l'index de region dans i0",
		Mecanisme: "la porte est trop courte pour porter un index : la largeur 1 par defaut est conservee",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgFilmdec + "traverse.go",
			Ancre:   "if l.GateBits > i0SpineBits+i0UseDefaultBits {",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 3.4",
		CritereRetrait:  "la largeur d'index vient du profil de la carte ; 0 recours au defaut",
		CompteurBranche: false,
		CibleComptage:   comptageParFilmContext,
	},
	{
		Nom:       "repli_largeur_absolue_uniforme",
		Fait:      "la largeur de quantification des trois axes d'un chemin absolu i0",
		Mecanisme: "une largeur UNIFORME (14) s'applique aux trois axes des que le reglage global est pose — ce qui est toujours vrai en production",
		Condition: CondInconditionnel,
		Ordre:     OrdreDevantLaLecture,
		Sites: []Site{{
			Fichier: pkgFilmdec + "position_capture.go",
			Ancre:   "if absoluteAxisW > 0 {",
		}},
		DatePose:     dateAudit0E,
		CibleRetrait: "lot 3.4 (largeurs par carte et par build)",
		// ORDRE `devant_la_lecture` : le global de precision des objets du monde porte les
		// largeurs PAR AXE de la carte, et le reglage global uniforme les ecrase pour les trois.
		// (Le nom de ce global n'est pas cite ici : `filmdec/world_object_precision_guard_test.go`
		// exige que tout fichier qui le mentionne dise d'ou il tient ses largeurs, et un registre
		// documentaire n'en lit aucune.)
		CritereRetrait:  "les trois axes prennent leur largeur du profil ; le reglage global uniforme est retire avec ses tests",
		CompteurBranche: false,
		CibleComptage:   comptageParFilmContext,
	},
	{
		Nom:       "repli_largeurs_mpp_par_defaut",
		Fait:      "les largeurs du bloc de proprietes multijoueur (MPP), donc l'alignement de tout l'etat par defaut",
		Mecanisme: "9 et 5 bits par defaut, employes des que la calibration du film n'a rien pose",
		Condition: CondInconditionnel,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgFilmdec + "default_state.go",
			Ancre:   "var mppLeadBits = 9",
		}},
		DatePose:     dateAudit0E,
		CibleRetrait: "lot 3.x (profil par build : les largeurs MPP sont une donnee du build)",
		// La voie ÉQUIPEMENT journalise quand elle retombe sur le défaut ; la voie SOCLES se
		// tait. Deux chemins, une seule constante, un seul silence.
		CritereRetrait:  "les largeurs MPP viennent du profil du build ; 0 recours au defaut sur les 8 builds",
		CompteurBranche: false,
		CibleComptage:   comptageParFilmContext,
	},
	{
		Nom:       "repli_chunks_apres_trou_abandonnes",
		Fait:      "quels chunks de replication d'un film sont lus",
		Mecanisme: "un trou dans la numerotation arrete l'enumeration : les chunks 8..N d'un film dont le 7 manque sont ABANDONNES",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgFilmdec + "film_chunks.go",
			Ancre:   "break // trou de numerotation : meme arret que l'ancien CountFilmChunks",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot de conversion du manifeste (le manifeste dit quels chunks existent ; un trou est une donnee, pas une borne)",
		CritereRetrait:  "0 film du parc dont l'enumeration s'arrete avant le dernier chunk du manifeste",
		CompteurBranche: false,
		CibleComptage:   comptageParFilmContext,
	},
	{
		Nom:       "repli_ancre_sans_vie_delta_ecartee",
		Fait:      "sur quelles ancres la calibration des largeurs de creation d'equipement s'appuie",
		Mecanisme: "une vie que les paquets delta n'ont pas vue est ecartee AVANT l'increment du denominateur : les ancres ecartees sont invisibles",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgFilmdec + "equipment_creation_width.go",
			Ancre:   "continue // une vie que les paquets delta n'ont pas vue ne peut rien arbitrer",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 3.x (largeurs de creation par build, la calibration disparait)",
		CritereRetrait:  "les largeurs viennent du profil ; la calibration est retiree avec ses tests",
		CompteurBranche: false,
		CibleComptage:   comptageParFilmContext,
	},
	{
		Nom:       "repli_registre_inconnu_sans_lecteur_de_troncature",
		Fait:      "la cause annoncee quand le registre d'un chunk_00 ne correspond a aucun build connu",
		Mecanisme: "un seul signal sort (« registre inconnu »), alors que Registry.Truncated et TruncatedBytes nomment une AUTRE cause — un tampon coupe — que personne ne lit",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgFilmdec + "registry_fingerprint.go",
			Ancre:   "func warnUnknownRegistry(fp uint64, blocks, slots int) {",
		}},
		DatePose:     "2026-09-14",
		CibleRetrait: "lot 3.1 (build inconnu actif, profil comme donnee fabriquee) : un build inconnu devient une erreur typee, une troncature en est une autre",
		// Découverte D6 (1.2) : la troncature est NOMMÉE dans `Registry` mais n'a AUCUN lecteur,
		// et le seul signal qui sort en exploitation attribue la mauvaise cause.
		CritereRetrait:  "les deux causes sont distinguees a la sortie (erreur typee `build_inconnu` contre `tronque`) et comptees separement",
		CompteurBranche: false,
		CibleComptage:   "lot 3.1",
	},
}
