package profile

// registre_empreintes.go — LES EMPREINTES DU REGISTRE ECS, PAR CLEF ECRITE DU FILM (lot 3.1.1).
//
// # CE QUE CETTE TABLE EST, ET POURQUOI ELLE EST UNE RECOPIE
//
// Le registre ECS d un film (`chunk_00`) porte les NOMS des composants et leur ORDRE : c est lui
// qui route toute la grammaire du decodeur. Il est bit-a-bit identique d un film a l autre DANS
// UN MEME BUILD, et il change d un build a l autre. Son empreinte — un FNV-1a 64 bits sur les
// entrees nommees — est donc l identite de la grammaire des composants du film.
//
// La SOURCE de ces neuf lignes est le catalogue versionne
// `data/titles/halo_infinite/reference/film_profiles.json`, section `registryFingerprints`,
// ecrite par `cmd/film-profiles-build` au lot 3.2.1. LE DECODEUR NE PEUT PAS LA LIRE : ce serait
// un import d un paquet de catalogue depuis un calque, que le sens unique interdit (ADR 0034,
// D-1). La table du profil la RECOPIE donc, et le test de conformite
// `filmprofile.TestCatalogueConformeALaTableDesEmpreintes` prouve la recopie ligne pour ligne,
// valeur contre valeur — y compris les preuves, qui sont donc ecrites une fois pour deux
// lecteurs et ne peuvent pas diverger.
//
// # DEUX FAITS MESURES QU IL FAUT LIRE AVANT DE S ETONNER
//
//	L EMPREINTE N EST PAS UNIQUE. Trois paires de clefs partagent la leur : `HI_1_8_0` et
//	`HI_1_9_0` ; `HI_1_12_0` et `HI_1_13_0` ; `HI_1_4_1` et les films de majeure 33 sans
//	section (D2 (3.2)). L unicite porte sur la CLEF, jamais sur l empreinte — et c est
//	exactement ce qui fait exister le statut `presumee` (cf. classement_registre.go).
//
//	UNE CLEF `majeure` PEUT SELECTIONNER UN FILM QUI ECRIT UN BUILD. `majeure=33` couvre les
//	films sans section de cette version, mais aussi `a521164d`, qui ecrit `HI_1_4_1` : les deux
//	portent la meme empreinte. [EmpreinteRegistreAttendue] rend la plus specifique — le build.
//
// # `Statut` N EST PAS LE STATUT DE LA CLASSIFICATION, ET LES CONFONDRE EST LE PIEGE
//
// [StatutCatalogue] dit jusqu ou la MESURE porte (combien de films de la clef ont ete lus). La
// classification a l execution ([StatutRegistre], classement_registre.go) dit ce qu une
// empreinte LUE DANS UN FILM vaut contre cette table. Deux vocabulaires, deux fichiers.

// LES NEUF CLEFS ET LA DATE DE LA MESURE, NOMMEES UNE FOIS. Une table de donnees repete par
// nature sa clef ; les nommer garde la table lisible ET tient la regle 6 du depot (a la
// troisieme copie, on centralise — `goconst` compte les copies du PAQUET, pas celles du
// fichier : `profile_table.go` et `grenade.go` portent deja ces memes clefs).
//
// LES VALEURS SONT DES LITTERAUX et non `clefBuild(buildHI1131)` : le test de conformite lit ce
// fichier par analyse syntaxique et resout les constantes de FICHIER, pas les appels de
// fonction. Leur accord avec les autres tables est garde par
// [TestLaTableDesEmpreintesCouvreLesNeufClefs], qui interroge la table PAR les clefs du profil.
const (
	cleBuildHI1131 = "build=HI_1_13_0"
	cleBuildHI1120 = "build=HI_1_12_0"
	cleBuildHI1110 = "build=HI_1_11_0"
	cleBuildHI1100 = "build=HI_1_10_0"
	cleBuildHI190  = "build=HI_1_9_0"
	cleBuildHI180  = "build=HI_1_8_0"
	cleBuildHI141  = "build=HI_1_4_1"
	cleMajeure33   = "majeure=33"
	cleMajeure31   = "majeure=31"
	// dateDesEmpreintes : le jour ou les neuf empreintes ont ete mesurees (lot 3.2.1).
	dateDesEmpreintes = "2026-09-16"
)

// StatutCatalogue dit jusqu ou la mesure d une empreinte PORTE.
//
//	connue    l empreinte est lue sur au moins deux films distincts de la clef, ou sur le seul
//	          film que le cache en porte — la population est couverte.
//	presumee  un seul film de la clef a ete lu alors que le cache en porte d autres : rien ne
//	          contredit la valeur, rien ne prouve encore qu elle vaut pour toute la clef.
type StatutCatalogue string

// Les deux statuts de couverture, et rien d autre.
const (
	StatutCatalogueConnue   StatutCatalogue = "connue"
	StatutCataloguePresumee StatutCatalogue = "presumee"
)

// EmpreinteRegistre est UNE empreinte de registre ECS, pour UNE clef ecrite du film.
type EmpreinteRegistre struct {
	// Cle est la clef ECRITE que cette empreinte couvre : `build=<id>` ou `majeure=<n>`.
	Cle string
	// Empreinte est le FNV-1a 64 bits des entrees nommees du registre.
	Empreinte uint64
	// Blocs est le nombre de blocs d archetype ; SlotsNommes le nombre d entrees NOMMEES
	// hachees — le denominateur de l empreinte. Sans lui, une empreinte differente ne se
	// distingue pas d un registre tronque.
	Blocs       int
	SlotsNommes int
	// Statut est la COUVERTURE de la mesure (cf. [StatutCatalogue]), jamais la classification.
	Statut StatutCatalogue
	// Source est la provenance de la ligne, comme pour toute valeur de profil.
	Source Provenance
	// Temoins sont les films sur lesquels l empreinte a ete LUE, en forme courte (8 hex).
	Temoins []string
	// Preuve est l instrument, le journal ou l oracle qui a rendu la valeur ; Date le jour de
	// la mesure.
	Preuve string
	Date   string
}

// EmpreintesRegistre rend la table complete, DANS L ORDRE DU CATALOGUE — c est cet ordre que le
// test de conformite compare rang par rang, et c est lui qui fait la priorite de
// [EmpreinteRegistreAttendue] entre deux clefs `majeure=` correspondantes.
//
// ELLE EST UNE FONCTION ET NON UNE `var` DE PAQUET, comme [TableProfil] : le ratchet
// `archlint/TestFilmdecPackageVarsNeCroitPas` gele l etat global mutable du decodeur, et une
// table de donnees n a aucune raison d en etre. Chaque appel rend une tranche neuve.
func EmpreintesRegistre() []EmpreinteRegistre {
	out := make([]EmpreinteRegistre, 0, 9)
	out = append(out, empreintesDesBuildsRecents()...)
	out = append(out, empreintesDesBuildsAnciens()...)
	out = append(out, empreintesSansSectionDIdentification()...)
	return out
}

// empreintesDesBuildsRecents : les quatre builds dont le cache porte le plus de films
// (`HI_1_13_0` a `HI_1_10_0`). Les deux premiers PARTAGENT leur empreinte : la frontiere du
// registre n est pas celle des builds (D2 (3.2)).
func empreintesDesBuildsRecents() []EmpreinteRegistre {
	return []EmpreinteRegistre{
		{
			Cle: cleBuildHI1131, Empreinte: 0x36ca8c3d2a2f9b88,
			Blocs: 50, SlotsNommes: 1067,
			Statut: StatutCatalogueConnue, Source: ProvenanceMesuree,
			Temoins: []string{"fb1a1a72", "0797ce72", "4f77afc1", "51ebbc0f", "bf15f7ab", "bfecd02b", "c75f33b8", "d9781168"},
			Preuve: "build de REFERENCE (grammar.KnownRegistryFingerprint). Lecture directe de chunk_00 " +
				"de la mini-bobine commise film/replay/testdata/minifilm_fb1a1a72 " +
				"(ParseRegistryChunk + RegistryFingerprint) : 0x36ca8c3d2a2f9b88, 50 blocs, 1067 " +
				"slots nommes. Corrobore par le SILENCE de l alerte sur les huit temoins HI_1_13_0 " +
				"du corpus gate du 2026-09-16 (journal de cuisson work_m1_cloture/logs/general.log " +
				": aucun WARN `empreinte du registre ECS du film INCONNUE` pour ces huit match_id, " +
				"dont le build est ecrit en clair dans le meme journal) — 1 123 films de ce build " +
				"au cache (lot H)",
			Date: dateDesEmpreintes,
		},
		{
			Cle: cleBuildHI1120, Empreinte: 0x36ca8c3d2a2f9b88,
			Blocs: 50, SlotsNommes: 1067,
			Statut: StatutCataloguePresumee, Source: ProvenanceMesuree,
			Temoins: []string{"bcb6d393"},
			Preuve: "lecture directe de chunk_00 de la mini-bobine commise minifilm_bcb6d393 : MEME " +
				"empreinte que HI_1_13_0, 50 blocs, 1067 slots. Corrobore par le silence de l " +
				"alerte sur bcb6d393 au corpus gate du 2026-09-16. FAIT NOUVEAU : la frontiere du " +
				"registre n est pas celle des grenades — HI_1_12_0 est CONFORME au registre de " +
				"reference, la population `grammaire inconnue` s arrete donc avant HI_1_12_0. " +
				"`presumee` : un seul film lu sur les 146 de ce build au cache (lot H)",
			Date: dateDesEmpreintes,
		},
		{
			Cle: cleBuildHI1110, Empreinte: 0x8879e2b6746ba047,
			Blocs: 49, SlotsNommes: 1031,
			Statut: StatutCataloguePresumee, Source: ProvenanceMesuree,
			Temoins: []string{"e5adf7b2"},
			Preuve: "lecture directe de chunk_00 de la mini-bobine commise minifilm_e5adf7b2 : " +
				"0x8879e2b6746ba047, 49 blocs, 1031 slots. Meme valeur au corpus gate du 2026-09-16 " +
				"(WARN empreinte=9834140534605324359, blocs=49, slots_non_vides=1031, temoin " +
				"e5adf7b2). `presumee` : un seul film lu sur les 39 de ce build au cache (lot H)",
			Date: dateDesEmpreintes,
		},
		{
			Cle: cleBuildHI1100, Empreinte: 0x9b6397b3ad58e258,
			Blocs: 49, SlotsNommes: 1031,
			Statut: StatutCatalogueConnue, Source: ProvenanceMesuree,
			Temoins: []string{"111fa685", "084a804d"},
			Preuve: "lecture directe de chunk_00 de la mini-bobine commise minifilm_111fa685 : " +
				"0x9b6397b3ad58e258, 49 blocs, 1031 slots. DEUX films distincts au corpus gate du " +
				"2026-09-16 rendent la meme valeur (WARN empreinte=11196959896536408664 compte DEUX " +
				"fois — la deduplication est par processus, et chaque temoin est cuit dans le sien " +
				"; les deux match_id de build HI_1_10_0 du journal sont 111fa685 et 084a804d). 26 " +
				"films de ce build au cache (lot H)",
			Date: dateDesEmpreintes,
		},
	}
}

// empreintesDesBuildsAnciens : `HI_1_9_0`, `HI_1_8_0` et `HI_1_4_1`. Les deux premiers partagent
// leur empreinte, et celle du troisieme est aussi celle de la majeure 33 sans section.
func empreintesDesBuildsAnciens() []EmpreinteRegistre {
	return []EmpreinteRegistre{
		{
			Cle: cleBuildHI190, Empreinte: 0x33c7e724716d8cc5,
			Blocs: 49, SlotsNommes: 1031,
			Statut: StatutCatalogueConnue, Source: ProvenanceMesuree,
			Temoins: []string{"11de8353"},
			Preuve: "lecture directe de chunk_00 de la mini-bobine commise minifilm_11de8353 ET du film " +
				"complet 11de8353 du cache : 0x33c7e724716d8cc5, 49 blocs, 1031 slots — valeur " +
				"IDENTIQUE a celle de HI_1_8_0, mesure neuve (aucun temoin HI_1_9_0 n avait jamais " +
				"ete cuit au corpus gate). `connue` : le cache ne porte qu UN film de ce build (lot " +
				"H) et il est lu",
			Date: dateDesEmpreintes,
		},
		{
			Cle: cleBuildHI180, Empreinte: 0x33c7e724716d8cc5,
			Blocs: 49, SlotsNommes: 1031,
			Statut: StatutCataloguePresumee, Source: ProvenanceMesuree,
			Temoins: []string{"60ae07c4"},
			Preuve: "lecture directe de chunk_00 de la mini-bobine commise minifilm_60ae07c4 : " +
				"0x33c7e724716d8cc5, 49 blocs, 1031 slots. Meme valeur au corpus gate du 2026-09-16 " +
				"(WARN empreinte=3731204960007589061, temoin 60ae07c4). `presumee` : un seul film " +
				"lu sur les 10 de ce build au cache (lot H)",
			Date: dateDesEmpreintes,
		},
		{
			Cle: cleBuildHI141, Empreinte: 0x40531a0d86ce90ce,
			Blocs: 49, SlotsNommes: 1033,
			Statut: StatutCatalogueConnue, Source: ProvenanceMesuree,
			Temoins: []string{"a521164d"},
			Preuve: "lecture directe de chunk_00 de la mini-bobine commise minifilm_a521164d ET du film " +
				"complet a521164d du cache : 0x40531a0d86ce90ce, 49 blocs, 1033 slots — le SEUL " +
				"registre du corpus a 1 033 slots nommes. `connue` : le cache ne porte qu UN film " +
				"de ce build (lot H) et il est lu. Mesure neuve : aucun temoin HI_1_4_1 n avait " +
				"jamais ete cuit au corpus gate",
			Date: dateDesEmpreintes,
		},
	}
}

// empreintesSansSectionDIdentification : les deux familles de films dont `chunk_00` ne porte
// AUCUNE section d identification. Elles n ecrivent pas de build, et la version majeure est la
// seule clef qu elles portent — meme regle que [tableProfilGrenadeSansSection].
func empreintesSansSectionDIdentification() []EmpreinteRegistre {
	return []EmpreinteRegistre{
		{
			Cle: cleMajeure33, Empreinte: 0x40531a0d86ce90ce,
			Blocs: 49, SlotsNommes: 1033,
			Statut: StatutCataloguePresumee, Source: ProvenanceMesuree,
			Temoins: []string{"a349fea8"},
			Preuve: "films de version majeure 33 dont chunk_00 ne porte AUCUNE section d identification " +
				": ils n ecrivent pas de build, la majeure est leur seule clef. Lecture directe de " +
				"chunk_00 du film a349fea8 du cache : 0x40531a0d86ce90ce, 49 blocs, 1033 slots — " +
				"MEME registre que HI_1_4_1, ce qui NOMME enfin la grammaire de ces films. Meme " +
				"valeur au corpus gate du 2026-09-16 (WARN empreinte=4635077086892953806, blocs=49, " +
				"slots_non_vides=1033, temoin a349fea8). `presumee` : un seul des deux films sans " +
				"section de cette majeure est lu (lot H). Cette clef selectionne aussi a521164d, " +
				"qui ecrit HI_1_4_1 : meme empreinte, aucune contradiction, et l entree `build=` " +
				"prime",
			Date: dateDesEmpreintes,
		},
		{
			Cle: cleMajeure31, Empreinte: 0xba34fa35f781d1a7,
			Blocs: 49, SlotsNommes: 1029,
			Statut: StatutCataloguePresumee, Source: ProvenanceMesuree,
			Temoins: []string{"50247b26"},
			Preuve: "films de version majeure 31 sans section d identification — la plus ancienne " +
				"grammaire du cache. Lecture directe de chunk_00 du film 50247b26 : " +
				"0xba34fa35f781d1a7, 49 blocs, 1029 slots : une empreinte JAMAIS mesuree avant ce " +
				"lot (aucun temoin de cette famille n a jamais ete cuit au corpus gate) et le seul " +
				"registre du corpus a 1 029 slots. `presumee` : un seul des trois films de cette " +
				"majeure est lu (lot H)",
			Date: dateDesEmpreintes,
		},
	}
}
