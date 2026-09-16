package profile

// profile_table.go — LA TABLE DU PROFIL : UNE LIGNE PAR CLE ECRITE DANS LE FILM.
//
// # LA CLE DU PROFIL EST ECRITE DANS LE FILM, ET IL Y EN A TROIS
//
// Fait tranche par l utilisateur le 2026-09-16 : « le film est autoportant ; il ne depend que
// de lui-meme pour expliquer au mode Theater comment le lire ». Un profil n est donc JAMAIS un
// reglage externe qui refuserait un film : c est une TABLE indexee par ce que le film ECRIT.
// Le depot en connait trois cles, et elles ne disent pas la meme chose :
//
//	VERSION DE FORMAT   `chunk_00+4` ([FilmFormatVersionFromHeader]). Elle dit la GRAMMAIRE DES
//	                    BITS : largeur du bloc de registre, cardinal de la table par type,
//	                    decoupage du bloc `object-multiplayer-properties`. C est la cle que le
//	                    LECTEUR du jeu consulte (`FUN_14299ab50`, `FUN_1428e1c0c`).
//	BUILD               la chaine en clair de la section 2 ([ReadFilmIdentity]). Elle dit la
//	                    TAILLE DES STRUCTURES DE CONTENU : le bloc de personnalisation d un
//	                    slot de joueur. Elle ne se deduit PAS du format (le format 24 porte
//	                    trois builds, dont deux largeurs de personnalisation differentes).
//	VERSION MAJEURE     l u32 de `chunk_00+0` ([FilmMajorVersionFromHeader]). Elle dit
//	                    l IMPLANTATION DU GAMERTAG dans un bloc d evenement de temps fort.
//
// Une quatrieme entree n est pas une cle du film mais une donnee du MATCH — l entree de
// catalogue de la CARTE ([MapQuantEntry]) — et elle entre au profil par le constructeur.
//
// # LA PROVENANCE EST UNE COLONNE, PAS UNE POLITESSE (D3 du plan, D-3 d ADR 0034)
//
// Chaque valeur porte D OU ELLE VIENT, et il y a exactement trois provenances :
//
//	RELUE     l executable est ouvert, la fonction est citee (Ghidra, base 0x140000000).
//	MESUREE   l executable de ce build n est pas disponible : la valeur se mesure sur un film
//	          temoin par un oracle INTERNE au film, et la ligne cite l oracle et son score.
//	PRESUMEE  ni l un ni l autre. La valeur est celle que le depot applique depuis toujours,
//	          sans preuve ecrite. [TestProfilPresumes] les liste, une par une : c est le
//	          registre de ce qui reste a etablir, et il ne grossit pas en silence.
//
// # CE QUE CETTE TABLE N EST PAS
//
// Elle n est pas un CATALOGUE VERSIONNE (D12) : aucun fichier, aucune ecriture a l execution.
// Elle n est pas non plus une MESURE : rien ici ne se lit sur le film au moment du decodage —
// ce qui se lit sur le film, ce sont les trois CLES, et elles seules.

// LES LIBELLES REPETES DE LA TABLE, NOMMES UNE FOIS. Une table de donnees repete par nature sa
// cle et son champ ; les nommer garde la table lisible ET tient la regle 6 du depot (a la
// troisieme copie, on centralise).
const (
	// cleToutes : la valeur ne depend d AUCUNE cle ecrite dans le film.
	cleToutes = "toutes"
	// champSlotsPerso : la largeur du bloc de personnalisation d un slot de joueur.
	champSlotsPerso = "Slots.PersoBytes"
	// champImplantation : l implantation du gamertag dans un bloc d evenement de temps fort.
	champImplantation = "Highlight.Implantation"
)

// LES DATES AUXQUELLES CES PROVENANCES ONT ETE ETABLIES, nommees par ce qu elles datent.
const (
	// dateEcrivainSection2 : releve du serialiseur de `chunk_00` chez l ecrivain (Ghidra).
	dateEcrivainSection2 = "2026-09-12"
	// dateTranspositionSlots : mesure des transpositions de slot par build sur le cache.
	dateTranspositionSlots = "2026-09-13"
	// dateProfilPose : le lot 2.1, jour ou ces valeurs entrent au profil avec leur provenance.
	dateProfilPose = "2026-09-17"
)

// Provenance dit d ou vient une valeur de profil. Cf. l en-tete du fichier.
type Provenance string

// Les trois provenances, et rien d autre.
const (
	// ProvenanceRelue : l executable est ouvert et la fonction est citee.
	ProvenanceRelue Provenance = "relue"
	// ProvenanceMesuree : la valeur est mesuree sur un film temoin par un oracle interne.
	ProvenanceMesuree Provenance = "mesuree"
	// ProvenancePresumee : ni relue ni mesuree. [TestProfilPresumes] la liste.
	ProvenancePresumee Provenance = "presumee"
)

// LigneProfil est UNE ligne de la table : une valeur, sa cle, sa provenance et sa preuve.
type LigneProfil struct {
	// Cle est la cle ECRITE qui selectionne cette ligne : `format=27`, `build=HI_1_13_0`,
	// `majeure>=41`, ou `toutes` quand la valeur ne depend d aucune cle connue.
	Cle string
	// Champ nomme ce que la ligne pose, dans le vocabulaire du [Profile].
	Champ string
	// Valeur est la valeur posee, ecrite pour etre lue par un humain.
	Valeur string
	// Source est la provenance (cf. l en-tete).
	Source Provenance
	// Preuve est la fonction Ghidra (provenance relue), ou le film temoin et son oracle
	// (provenance mesuree), ou ce qui manque (provenance presumee).
	Preuve string
	// Date est le jour ou cette provenance a ete etablie, `AAAA-MM-JJ`.
	Date string
}

// TableProfil rend la table complete, dans l ordre des trois cles puis des invariants.
//
// ELLE EST UNE FONCTION ET NON UNE `var` DE PAQUET : le ratchet
// `archlint/TestFilmdecPackageVarsNeCroitPas` gele l etat global mutable de `filmdec`, et une
// table de donnees n a aucune raison d en etre. Chaque appel rend une tranche neuve.
//
//nolint:funlen // une table de donnees : la decouper masquerait la lecture ligne a ligne
func TableProfil() []LigneProfil {
	out := make([]LigneProfil, 0, 32)
	out = append(out, tableProfilFormat()...)
	out = append(out, tableProfilBuild()...)
	out = append(out, tableProfilMajeure()...)
	out = append(out, tableProfilInvariants()...)
	return out
}

// tableProfilFormat : les lignes keyees par la VERSION DE FORMAT (`chunk_00+4`).
func tableProfilFormat() []LigneProfil {
	return []LigneProfil{
		{
			Cle: "format=27", Champ: "MPP", Valeur: "lead=9 index=5",
			Source: ProvenanceRelue,
			Preuve: "FUN_14080cfe8 (bloc object-multiplayer-properties) : champ de tete R(9) " +
				"par FUN_141fd72c0 (`141fd72de : ADD dword ptr [RCX + 0x2c],0x9`), champ " +
				"d index R(5) inline ; aucune branche de version dans le bloc",
			Date: "2026-09-15",
		},
		{
			Cle: "format=20,21,24,25", Champ: "MPP", Valeur: "INDETERMINEE (aucune largeur posee)",
			Source: ProvenanceMesuree,
			Preuve: "les deux oracles internes se contredisent : `n2` designe 8/3 (part modale " +
				"0,988 a 0,996 contre 0,304 a 0,522) et la fermeture designe 10/5 la ou " +
				"l ecrivain est certain — cf. build_profile.go. La case reste VIDE : c est " +
				"l etat reel de la connaissance",
			Date: "2026-09-15",
		},
	}
}

// tableProfilBuild : les lignes keyees par le BUILD (section 2 de `chunk_00`).
//
// Les largeurs elles-memes vivent dans [personnalisationOctets], avec un commentaire de
// provenance par ligne : cette table les REFERENCE, elle ne les recopie pas — une seconde copie
// divergerait au premier build ajoute (regle 6 du depot).
func tableProfilBuild() []LigneProfil {
	return []LigneProfil{
		{
			Cle: "build=HI_1_13_0", Champ: champSlotsPerso, Valeur: "1852",
			Source: ProvenanceRelue,
			Preuve: "FUN_1407edea8 : `LEA R8,[RDI+0xcc0] ; R9D=0x39e0` ; fermeture par le " +
				"serialiseur delta FUN_1407ec27c (`actionPose` a +0x738, plus 4 = 0x73C)",
			Date: dateEcrivainSection2,
		},
		{
			Cle: "build=HI_1_12_0", Champ: champSlotsPerso, Valeur: "1852",
			Source: ProvenanceMesuree,
			Preuve: "transposition `mesure - predit` de 0 bit, valeur unique sur les 146 films " +
				"HI_1_12_0 du cache (oracle TestResidusSlotChaineCorpus)",
			Date: "2026-09-14",
		},
		{
			Cle: "build=HI_1_11_0", Champ: champSlotsPerso, Valeur: "1492",
			Source: ProvenanceMesuree,
			Preuve: "transposition -2 880 bits, valeur unique sur les 39 films du cache ; coupe " +
				"autour du bloc de queue verifiee sur `00ba2e1c`",
			Date: dateTranspositionSlots,
		},
		{
			Cle: "build=HI_1_10_0", Champ: champSlotsPerso, Valeur: "1492",
			Source: ProvenanceMesuree,
			Preuve: "transposition -2 880 bits, valeur unique sur les 26 films du cache ; coupe " +
				"verifiee sur `084a804d`",
			Date: dateTranspositionSlots,
		},
		{
			Cle: "build=HI_1_9_0", Champ: champSlotsPerso, Valeur: "1312",
			Source: ProvenanceMesuree,
			Preuve: "transposition -4 320 bits, unique film `11de8353` du cache, coupe verifiee",
			Date:   dateTranspositionSlots,
		},
		{
			Cle: "build=HI_1_8_0", Champ: champSlotsPerso, Valeur: "1312",
			Source: ProvenanceMesuree,
			Preuve: "transposition -4 320 bits, valeur unique sur les 10 films du cache ; coupe " +
				"verifiee sur `0a247154`",
			Date: dateTranspositionSlots,
		},
		{
			Cle: "build=HI_1_4_1", Champ: champSlotsPerso, Valeur: "2052",
			Source: ProvenanceMesuree,
			Preuve: "transposition +1 600 bits, unique film `a521164d` du cache ; le +200 ne " +
				"ferme sur aucun pas connu — cause au registre des reports",
			Date: dateTranspositionSlots,
		},
	}
}

// tableProfilMajeure : les lignes keyees par la VERSION MAJEURE (`chunk_00+0`).
//
// TROIS IMPLANTATIONS, DEUX VALEURS : `<= 38` et `>= 41` posent le gamertag en tete du bloc
// d evenement, `39-40` le decalent de douze octets. La BRANCHE qui les applique vit en un seul
// endroit, `analysis.decodeEventBytes` ; ce que le profil apporte, c est la CLE et le NOM.
func tableProfilMajeure() []LigneProfil {
	return []LigneProfil{
		{
			Cle: "majeure<=38", Champ: champImplantation, Valeur: "gamertag_en_tete (b[0:32])",
			Source: ProvenanceMesuree,
			Preuve: "port de spnkr/film/highlight_events.py, confirme sur le cache : le roster " +
				"humain se ferme sur les films de majeure <= 38",
			Date: dateEcrivainSection2,
		},
		{
			Cle: "majeure=39,40", Champ: champImplantation, Valeur: "gamertag_decale_12 (b[12:44])",
			Source: ProvenanceMesuree,
			Preuve: "211 films du cache (mars-novembre 2025) : sous l implantation en tete le " +
				"roster tombait a 2 noms distincts pour 24 a 27 joueurs " +
				"(.ai/RAPPORT_BTB_2025_ABSTENTION_2026-09-12.md)",
			Date: dateEcrivainSection2,
		},
		{
			Cle: "majeure>=41", Champ: champImplantation, Valeur: "gamertag_en_tete (b[0:32])",
			Source: ProvenanceMesuree,
			Preuve: "1 117 films du cache portent 0x29 (41) et rendent un roster ferme sous " +
				"l implantation en tete",
			Date: dateEcrivainSection2,
		},
	}
}

// tableProfilInvariants : les valeurs que le depot applique a TOUS les films.
//
// « Toutes » n est pas un blanc-seing : c est une ligne comme une autre, avec sa provenance. La
// plupart sont PRESUMEES — c est-a-dire que la variabilite par build n a jamais ete mesuree —
// et [TestProfilPresumes] les tient a l oeil. Les faire varier par build est le lot 3.
//
//nolint:funlen // une table de donnees : la decouper masquerait la lecture ligne a ligne
func tableProfilInvariants() []LigneProfil {
	return []LigneProfil{
		{
			Cle: cleToutes, Champ: "Keyframe.EnTeteBits", Valeur: "108",
			Source: ProvenanceRelue,
			Preuve: "FUN_142e2bfd0 : R(32) id, R(32) typeIndex, R(32), R(4) (FUN_142e29cf8), R(8)",
			Date:   "2026-08-17",
		},
		{
			Cle: cleToutes, Champ: "Keyframe.MotDeTailleBits", Valeur: "32 (deux mots : n1, n2)",
			Source: ProvenanceRelue,
			Preuve: "FUN_142e2bfd0 : deux `if (0 < (int)uVar7)` gardant vtable[0x60] puis " +
				"vtable[0x88] — comparaison SIGNEE",
			Date: "2026-09-15",
		},
		{
			Cle: cleToutes, Champ: "Movement.Traversal", Valeur: "IndexW=1 AxisW=6/6/6",
			Source: ProvenancePresumee,
			Preuve: "largeurs capturees a l execution (Cheat Engine) sur UNE session ; aucune " +
				"lecture chez l ecrivain, aucune mesure par build",
			Date: dateProfilPose,
		},
		{
			Cle: cleToutes, Champ: "Movement.AbsoluteAxisW", Valeur: "14",
			Source: ProvenancePresumee,
			Preuve: "uniforme applique a defaut de table par index de plage ; `absAxisWFor` " +
				"retombe dessus (repli inscrit au registre, ancre `if w := br.absoluteAxisW(); w > 0 {`)",
			Date: dateProfilPose,
		},
		{
			Cle: cleToutes, Champ: "Movement.WorldObject", Valeur: "IndexW=1 AxisW=13/13/14 Region=0",
			Source: ProvenanceMesuree,
			Preuve: "largeurs du chemin world-object : le defaut EST l entree `cliffhanger` de " +
				"`map_quant_bounds.json` (verifie par `map_bounds_test.go`) ; la carte du match " +
				"les remplace par `SetWorldObjectPrecisionFromLayout`, depuis les bornes du " +
				"catalogue et non l auto-detection (accord catalogue / film : 7 films sur 7)",
			Date: "2026-08-15",
		},
		{
			Cle: cleToutes, Champ: "Movement.DeltaQuantum", Valeur: "0.01383",
			Source: ProvenanceMesuree,
			Preuve: "pas d un cran de position repliquee en DELTA par i0, mesure sur le film ; " +
				"la range delta vaut DeltaQuantum * 2^AxisW",
			Date: dateProfilPose,
		},
		{
			Cle: cleToutes, Champ: "Movement.DeltaAxisWidth", Valeur: "14",
			Source: ProvenancePresumee,
			Preuve: "largeur d axe du chemin DELTA axis-width ; ambiguite 6 contre 14 jamais " +
				"tranchee chez l ecrivain",
			Date: dateProfilPose,
		},
		{
			Cle: cleToutes, Champ: "Movement.Range", Valeur: "QuantRangeCEBiped",
			Source: ProvenancePresumee,
			Preuve: "DAT_14462cbe0[0] capture a l execution sur UNE carte ; la range par CARTE " +
				"vient du catalogue (Profile.Map), celle-ci est le defaut de paquet",
			Date: dateProfilPose,
		},
		{
			Cle: cleToutes, Champ: "Movement.FullPrecision", Valeur: "false",
			Source: ProvenanceRelue,
			Preuve: "miroir du global de configuration DAT_145121140, teste `== 1` par " +
				"FUN_14080cfe8 sans consommer de bit ; faux = la configuration de rejeu",
			Date: dateProfilPose,
		},
		{
			Cle: cleToutes, Champ: "Movement.DeltaHasHandleTail", Valeur: "false",
			Source: ProvenanceRelue,
			Preuve: "miroir du champ d execution bVar16 = (precIndex != -1) ; faux = le cas " +
				"dominant du film, et ce n est PAS un bit du flux",
			Date: dateProfilPose,
		},
		{
			Cle: cleToutes, Champ: "Movement.CalibratedSkip", Valeur: "false",
			Source: ProvenancePresumee,
			Preuve: "bascule de calibration d i0 (saut de 47 ou 101 bits) ; jamais activee en " +
				"production, jamais mesuree par build",
			Date: dateProfilPose,
		},
		{
			Cle: cleToutes, Champ: "Movement.MobilityActionExtraBits", Valeur: "0",
			Source: ProvenancePresumee,
			Preuve: "bits supplementaires d action de mobilite ; zero par defaut, aucune " +
				"lecture chez l ecrivain",
			Date: dateProfilPose,
		},
	}
}

// ImplantationGamertag est le NOM de l implantation du gamertag pour une version majeure.
//
// EXPORTEE AU LOT 2.5.b, ET LA RAISON EST MECANIQUE : c est le type du champ PUBLIC
// [HighlightProfile.Implantation]. Tant que le type et son lecteur vivaient dans le meme paquet,
// le champ se comparait a `implantationEnTete` sans que le type soit nommable de l exterieur ;
// depuis que le profil est une couche a part, un appelant qui lit ce champ ne pouvait plus
// nommer ce a quoi il le compare. Un champ exporte dont le type ne l est pas est un champ qu on
// ne peut que regarder.
type ImplantationGamertag string

// Les deux implantations mesurees sur le cache (cf. [tableProfilMajeure]).
const (
	// ImplantationEnTete : le gamertag occupe `b[0:32]` du bloc d evenement.
	ImplantationEnTete ImplantationGamertag = "gamertag_en_tete"
	// ImplantationDecale12 : le gamertag occupe `b[12:44]`.
	ImplantationDecale12 ImplantationGamertag = "gamertag_decale_12"
)

// majeureDecalageBascule / majeureDecalageFin bornent la plage de versions majeures qui
// decalent le gamertag de douze octets. Hors de `[39, 40]`, il est en tete.
const (
	majeureDecalageBascule = 39
	majeureDecalageFin     = 40
	// gamertagDecalageOctets : le decalage de l implantation `39-40`, en octets.
	gamertagDecalageOctets = 12
)

// implantationDuGamertag rend l implantation du gamertag pour une version majeure, et son
// decalage en octets.
//
// LA VERSION INCONNUE (`0`, film sans registre) TOMBE EN TETE, et c est le comportement
// historique que `analysis.ParseHighlightEvents` lui applique deja : le profil le NOMME au lieu
// de le laisser implicite, et [HighlightProfile.Lue] dit que la cle n a pas ete lue.
func implantationDuGamertag(majeure int) (ImplantationGamertag, int) {
	if majeure >= majeureDecalageBascule && majeure <= majeureDecalageFin {
		return ImplantationDecale12, gamertagDecalageOctets
	}
	return ImplantationEnTete, 0
}
