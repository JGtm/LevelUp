package fallback

// registre_filmdec.go — les replis des INFÉRENCES de `grammar` qui décident en production.
//
// PÉRIMÈTRE, TEL QUE L'AUDIT 0.E L'A FERMÉ : la GRAMMAIRE de `grammar` (largeurs de composants,
// désérialiseurs, états par défaut) est HORS AXE — elle vient du jeu et se corrige chez
// l'écrivain, pas par un repli. N'entrent ici que les fonctions qui DÉCIDENT à la place d'une
// lecture : découpage d'i0, bande de slots, largeurs installées, comblements, abandons de
// chunks.
//
// CONTRAINTE DE CÂBLAGE : le ratchet `archlint/filmdec_package_vars_test.go` gèle le nombre de
// variables de paquet de `grammar`, et le critère S1 du plan vise ZÉRO variable mutable dans le
// décodeur. Un compteur de replis ne s'y câble donc pas par une globale : il passera par le
// `FilmContext` (D1 : un seul objet par film, le profil résolu à la construction) au pas 2 de
// M2, qui est le lot où les lecteurs reçoivent ce porteur.

const pkgFilmdec = "internal/games/halo_infinite/film/internal/grammar/"

// pkgProfile : la couche `profile`, nee au lot 2.5.b. Un repli de ce registre a son site dans la
// DONNEE et non dans le lecteur — le decoupage MPP par defaut — et son ancre a suivi le SYMBOLE,
// pas le fichier d origine : c est la constante qui dit le repli, ou qu elle vive.
const pkgProfile = "internal/games/halo_infinite/film/internal/profile/"

// comptageParFilmContext : la raison, écrite une fois, du câblage différé des replis `grammar`.
const comptageParFilmContext = "pas 2 de M2 (les lecteurs recoivent le profil via FilmContext : le seul porteur par film sans variable de paquet, cf. S1)"

var registreFilmdec = []Repli{
	{
		Nom:       "repli_cadre_de_marche_par_defaut_conserve",
		Fait:      "la largeur du champ bas d identifiant de record (IDLowBits) sous laquelle la marche des morts d objet deroule CE film",
		Mecanisme: "profil de calibration PLAT (aucune largeur candidate ne domine son dauphin d un facteur 2 sur les paquets a evenements localises) : le cadre par defaut est conserve",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgFilmdec + "object_deaths_calibrate.go",
			Ancre:   "return base, true, meilleur, dauphin",
		}, {
			// Le COMPTE est emis par le calque des vehicules, qui recoit le verdict de la
			// calibration (site inscrit a la fusion du lot 1.9.10, 2026-09-16 : le ratchet des
			// sites pose par la revue M1 l exigeait).
			Fichier: "internal/games/halo_infinite/film/replay/vehicle_tracks.go",
			Ancre:   "clock.fb.Declenche(fallback.NomCadreDeMarcheParDefautConserve)",
		}},
		DatePose:     "2026-09-16",
		CibleRetrait: "M3 (le cadre de la boucle de records devient une donnee de PROFIL par build, comme les largeurs du bloc MPP)",
		// ORDRE `apres_lecture` : le balayage des six largeurs candidates tourne D ABORD, sur
		// l oracle de la signature du slot 123 ; ce repli n entre que si AUCUNE ne se detache.
		// CE QU IL EMPECHE, ET C EST MESURE (2026-09-16) : sur `minibobine_e5adf7b2` les six
		// cadres localisent 0 paquet sur 54. Sans ce repli, le code retenait quand meme un cadre
		// — au departage par records propres, c est-a-dire par le critere que le lot V13 a
		// REFUTE (`idLow=10` rend 492 records dits propres et localise 0 paquet sur 12). Un cadre
		// faux consomme des bits sans rien decoder de vrai : il fabrique des morts.
		CritereRetrait:  "0 film cuit au cadre par defaut ; les sept builds portent leur IDLowBits au profil",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_largeurs_mpp_calibrees_sur_le_film",
		Fait:      "les deux largeurs du bloc object-multiplayer-properties (FUN_14080cfe8), qui precede TOUS les composants dans l etat par defaut de ti=36, 37, 38, 39, 42 et 43",
		Mecanisme: "CalibrateMPPWidthsOf MESURE le decoupage sur le film et l INSTALLE, au lieu de le lire au profil de la version de format",
		Condition: CondFormatSansProfilRelu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgFilmdec + "equipment_placements.go",
			Ancre:   "cal, ok := CalibrateMPPWidthsOf(fc, wr, band, spans)",
		}, {
			Fichier: "internal/games/halo_infinite/film/replay/build_ground_weapons.go",
			Ancre:   "return calibrees",
		}},
		DatePose:     dateVague2,
		CibleRetrait: "l executable d un build <= HI_1_11_0, relu comme FUN_141fd72c0 l a ete pour HI_1_13_0 — ou un profil mesure par un oracle valide au-dessus du seuil de coincidence",
		// POURQUOI CE REPLI EXISTE, ET POURQUOI CE N'EST PAS UNE DETTE ORDINAIRE.
		// La grammaire du bloc MPP est VERSIONNEE PAR LA VERSION DE FORMAT de `chunk_00`
		// (`+4`, mesure du 2026-09-15, lot 1.9.1 ter — la cle etait dite « par build » au
		// 1.9.1 bis, et le chargeur du jeu a tranche : c'est le format). La coupure est dans
		// `]25, 27]`, et la version MAJEURE du film ne la donne pas (`e5adf7b2` et `bcb6d393`
		// sont tous deux `v=40` et tombent de part et d'autre). Le format 27 porte sa largeur AU
		// PROFIL, RELUE (9/5, `FUN_141fd72c0` litteral `141fd72de`) : sur lui la calibration ne
		// s'applique plus. Les formats 20, 21, 24 et 25 n'ont pas de valeur relue, et les deux
		// oracles internes au film SE CONTREDISENT : `n2` designe 8/3 (part modale 0,988 a
		// 0,996) mais la FERMETURE descend de 246 a 182 records si on le pose. Poser l'un ou
		// l'autre serait une decision deguisee en mesure ; la calibration reste donc, NOMMEE,
		// comptee et datee.
		//
		// ORDRE `apres_lecture` : le profil se resout D'ABORD (`chunk_00+4`), et la calibration
		// n'entre que si ce format n'a pas de largeur relue. Un film sans section
		// d'identification tombe dans le meme cas — il porte le format 20.
		//
		// UN FORMAT INCONNU (28 au prochain patch du jeu) TOMBE ICI AUSSI, ET C'EST VOULU : le
		// repli tient le parc neuf au lieu de l'eteindre. Son declenchement est compte A PART,
		// par `grammar.UnknownFormatExpvarPairs` (`filmdec_unknown_format_<n>`, cable dans
		// `replay/mpp_format_inconnu.go`) et signale par UN avertissement par film — parce qu'un
		// changement de format du jeu est un EVENEMENT, pas un repli ordinaire, et qu'il ne peut
		// pas attendre le comptage differe ci-dessous. Branche de production gardee par
		// `replay/mpp_format_inconnu_test.go`, mutation verifiee dans les deux sens.
		CritereRetrait:  "les formats du parc portent leur largeur MPP au profil ; 0 recours a la calibration sur le parc",
		CompteurBranche: false,
		CibleComptage:   comptageParFilmContext,
	},
	{
		Nom:       "repli_i0_porte_et_region_par_defaut",
		Fait:      "la largeur de la porte d'i0 et l'index de region attendu, quand AUCUNE entree de catalogue ne les impose",
		Mecanisme: "GateBits est force a DefaultI0GateBits (5) et Region a 0 : l'auto-detection ne sait pas voir un index de region de plus d'un bit",
		Condition: CondCarteAbsenteDuCatalogue,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgFilmdec + "i0_layout.go",
			Ancre:   "GateBits: profile.DefaultI0GateBits,",
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
		// CE QUE LE LOT 1.9.4 A FERME (2026-09-15), ET CE QUI SUBSISTE.
		//
		// FERME : `DetectI0Layout` n'a plus AUCUN appelant de production. Son dernier,
		// `grammar.DetectFilmMapEntry`, identifiait la CARTE par la signature de ses largeurs
		// d'axe ; il est supprime, la carte vient du nom de match, et le ratchet est le retrait de
		// l'entree d'allowlist dans `archlint/no_recomputed_film_context_test.go`.
		//
		// SUBSISTE, ET C'EST POURQUOI L'ENTREE RESTE : `DetectI0LayoutOf` — la forme FILM — garde
		// deux appelants de production, `FilmContext.I0Layout` et `offline_biped_band.bipedI0Layout`,
		// tous deux atteints quand AUCUNE entree de catalogue n'est imposee. C'est exactement la
		// condition de ce repli, et le geste qui le retire reste le meme : donner sa carte au
		// catalogue (lot 3.x, profil par carte). L'ancre, elle, ne bouge pas.
		//
		// MESURE DU LOT 1.9.2 (17 films, 14 temoins du corpus gate + les 8 builds) : catalogue et
		// auto-detection donnent le MEME decoupage sur 15 films ; les deux films Live Fire
		// divergent (`gate=6 region=1 12/12/11` contre `gate=5 region=0 13/12/11`), et la porte
		// de region du catalogue ecarte 26 enregistrements sur 267 400 (`60ae07c4`) et 11 sur
		// 146 860 (`0797ce72`) qui appartiennent a une AUTRE region de compression.
		CibleRetrait:    "lot 3.x (profil par carte) — la moitie `DetectI0Layout` est faite au lot 1.9.4, restent les deux appelants de `DetectI0LayoutOf`",
		CritereRetrait:  "aucun chemin de production n'appelle DetectI0LayoutOf ; le decoupage vient du catalogue sur les 8 builds",
		CompteurBranche: false,
		CibleComptage:   comptageParFilmContext,
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
		Nom:       "repli_largeurs_monde_par_defaut_conservees",
		Fait:      "les largeurs d'axe des objets du monde installees pour ce film",
		Mecanisme: "layout non detecte : le defaut est conserve plutot que d'installer des zeros",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgFilmdec + "profil_balayage.go",
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
			Fichier: pkgFilmdec + "profil_balayage.go",
			Ancre:   "if l.GateBits > profile.I0SpineBits+profile.I0UseDefaultBits {",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 3.4",
		CritereRetrait:  "la largeur d'index vient du profil de la carte ; 0 recours au defaut",
		CompteurBranche: false,
		CibleComptage:   comptageParFilmContext,
	},
	// `repli_largeur_absolue_uniforme` EST RETIRE LE 2026-09-17 (lot 3.4.1-a), ET SON CRITERE
	// EST TENU : « les trois axes prennent leur largeur du profil ; le reglage global uniforme
	// est retire avec ses tests ». Le champ `profile.MovementProfile.AbsoluteAxisW` n'existe
	// plus, ni son accesseur `Lecteur.absoluteAxisW` — le chemin absolu d i0 lit les trois
	// largeurs PAR AXE du descripteur de la carte ([profile.MapQuantEntry.PrecisionAbsolue]),
	// la meme table que le chemin world-object. Le temoin de mutation qui le prouve est
	// `grammar.TestProfilDePositionChangeLaConsommationDeBits`, cas `WorldObject.AxisW` : sous
	// l uniforme il mesurait 49 bits des deux cotes, il mesure desormais 47 contre 57.
	{
		Nom:       "repli_largeurs_mpp_par_defaut",
		Fait:      "les largeurs du bloc de proprietes multijoueur (MPP), donc l'alignement de tout l'etat par defaut",
		Mecanisme: "9 et 5 bits par defaut, employes des que la calibration du film n'a rien pose",
		Condition: CondInconditionnel,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgProfile + "mpp_widths.go",
			Ancre:   "const mppLeadParDefaut = 9",
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
	{
		Nom:       "repli_amorce_grenade_profil_de_reference",
		Fait:      "l amorce du record de creation de projectile — sa largeur, sa valeur et la position du champ d index de l auteur — sous laquelle les lancers de grenade de CE film se lisent",
		Mecanisme: "la clef du film (build de la section 2, ou version majeure pour les films sans section) n a pas de ligne dans la table du profil : la grammaire du build de REFERENCE s applique (24 bits, 0x40C00, index a +103)",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgProfile + "grenade.go",
			Ancre:   "func AmorceGrenadeDeReference() AmorceGrenade {",
		}, {
			Fichier: pkgFilmdec + "grenade_events.go",
			Ancre:   "g := grammaireDeReference()",
		}},
		DatePose: dateM3,
		// POURQUOI LE REFUS SERAIT PIRE QUE LE REPLI. D-4 d ADR 0034 interdit de lire un film au
		// profil du build VOISIN ; ici la reference n est pas un voisin choisi au jugement, c est
		// la grammaire de 1 269 des 1 351 films du cache. Refuser eteindrait les lancers de tout
		// le parc au premier patch du jeu, c est-a-dire exactement le defaut que le lot 3.3.1
		// vient de fermer sur les builds anciens.
		CibleRetrait: "retrait sec des que la table du profil couvre toutes les clefs du parc : le prochain patch du jeu ajoute sa clef d amorce en meme temps que son empreinte de registre",
		// Le compte n est pas branche pour la meme raison que les autres replis de `grammar` : le
		// decodeur ne porte pas de compteur de replis, et le cablage passe par le `FilmContext`.
		// D ici la, le declenchement sort en AVERTISSEMENT par film, avec la clef refusee.
		CritereRetrait:  "0 film cuit sous le profil de reference ; chaque clef du parc porte sa ligne d amorce",
		CompteurBranche: false,
		CibleComptage:   comptageParFilmContext,
	},
	{
		Nom:       "repli_controle_corruption_section_absente",
		Fait:      "le CONTROLE DE CORRUPTION PAR COMPOSANT — le bit de `chunk_00 + 0x0CB45C` qui decide si chaque composant present d un corps est suivi d un R(1) de garde et, si ce bit vaut 1, d un R(32) sentinelle 0x0bcddcba",
		Mecanisme: "le film ne porte PAS de section d identification, donc pas ce bit : la grammaire garde son invariant (faux — la valeur du singleton du jeu a la construction, `FUN_140eff23c`)",
		Condition: CondSectionAbsente,
		// APRES LECTURE, et l ordre est le fait : [ReadFilmIdentity] TOURNE d abord et rend
		// [ErrNoFilmIdentity] — c est son echec, pas une decision prise avant elle, qui ouvre ce
		// repli (D14 (b) : lire d abord, se replier ensuite).
		Ordre: OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgFilmdec + "profil_balayage.go",
			Ancre:   "func grammaireSousFilm(g GrammaireBalayage, p profile.Profile) (GrammaireBalayage, bool) {",
		}, {
			Fichier: pkgFilmdec + "controle_corruption_du_film.go",
			Ancre:   "func (c *FilmContext) ControleDeCorruptionRepli() bool {",
		}, {
			Fichier: pkgKillsource + "decode.go",
			Ancre:   "if !c.calib.ControleDeCorruptionLu {",
		}},
		DatePose: "2026-09-22",
		// POSE PAR LE LOT 5.18.2, QUI TRANSFORME UN DEFAUT MUET EN LECTURE. Jusque-la
		// `GrammaireBalayage.ControleDeCorruption` etait faux par defaut et n avait d ecrivain
		// qu un instrument : le decodeur ne lisait PAS ce que le film declare. Il le lit
		// desormais, et ce repli nomme le seul cas ou le film ne declare rien.
		//
		// LA POPULATION EST CONNUE ET BORNEE : les 5 films du cache sans section
		// d identification (`03af54c3`, `13b00e35`, `47d20b5d`, `50247b26`, `a349fea8`, format
		// 20), que la production met DEJA de cote ([profile.ErrUnknownBuild]). Sur les 1 605
		// autres, le bit est LU et vaut zero — l invariant conserve est donc, pour ces cinq-la,
		// la valeur que tout le parc declare.
		CibleRetrait:    "lot qui donnera un profil aux films de format 20 (section d identification absente) : le bit se lira alors a une position derivee de la version de format plutot que de l ancre de la chaine de build",
		CritereRetrait:  "0 film cuit sans declaration de ce bit ; les cinq films de format 20 portent leur position au profil",
		CompteurBranche: false,
		CibleComptage:   comptageParFilmContext,
	},
}
