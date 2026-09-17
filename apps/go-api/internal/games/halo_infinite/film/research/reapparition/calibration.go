//go:build research

package reapparition

// calibration.go — CE QUI DONNE LE DROIT DE PUBLIER UNE ADRESSE NEUVE.
//
// La chaine du descripteur est mecanique, donc elle peut etre mecaniquement fausse : un offset
// de famille different, une table de pointeurs au lieu d'un accesseur, une image d'une autre
// version du jeu. La seule defense est la CALIBRATION : rejouer la chaine sur des composants
// dont le depot connait deja l'adresse de l'ecrivain, et n'accorder aucune confiance au reste si
// l'une d'elles rate.
//
// LES SIX TEMOINS, ET D'OU VIENT CHAQUE ADRESSE :
//
//	quatre de `NOTE_3_6_METHODE_DESCRIPTEURS_2026-09-16.md` § 3 (les calibrations de la note) ;
//	deux de `NOTE_3_6_TI12_GRAMMAIRES_A_2026-09-17.md` § 15 et § 16 — les deux minuteurs manuels
//	de navpoint, qui sont justement les candidats du lot 3.7 : les lire par la chaine et retomber
//	sur l'adresse de la note prouve la chaine SUR LE TERRAIN MEME de la question posee.
//
// Le septieme (`ti=11 i0`) n'est PAS un temoin : `ecs_table.tsv` et `PISTE_A_ti11_color.md` en
// donnent deux adresses differentes (`FUN_142edbac8` et `FUN_142ed5a6c`). L'instrument le
// resout comme les autres et le rapport publie l'ecart au lieu de le trancher d'autorite.

// Temoin est un composant dont l'adresse d'ecrivain est deja etablie par le depot.
type Temoin struct {
	Nom      string
	Attendue uint64
	Source   string
}

// Calibrations porte les six temoins de la passe.
var Calibrations = []Temoin{
	{
		Nom: "managed-player-back-button-scoreboard-flair-component", Attendue: 0x142ed5af4,
		Source: "NOTE_3_6_METHODE_DESCRIPTEURS §3 (ti=9 i3)",
	},
	{
		Nom: "managed-navpoint-sub-type-component", Attendue: 0x1410e0cac,
		Source: "NOTE_3_6_METHODE_DESCRIPTEURS §3 (ti=12 i0)",
	},
	{
		Nom: "managed-navpoint-radial-progress", Attendue: 0x140fc8d14,
		Source: "NOTE_3_6_METHODE_DESCRIPTEURS §3 (ti=12 i14)",
	},
	{
		Nom: "device-position-animation-name-component", Attendue: 0x1410156e4,
		Source: "NOTE_3_6_METHODE_DESCRIPTEURS §3 (ti=43 i19)",
	},
	{
		Nom: "managed-navpoint-manual-timer-initial-duration-component", Attendue: 0x142ed5194,
		Source: "NOTE_3_6_TI12_GRAMMAIRES_A §15 (ti=12 i11)",
	},
	{
		Nom: "managed-navpoint-manual-timer-current-duration-component", Attendue: 0x142ed512c,
		Source: "NOTE_3_6_TI12_GRAMMAIRES_A §16 (ti=12 i12)",
	},
}

// Cible est un composant que le lot 3.7 interroge, avec la raison de l'interroger.
type Cible struct {
	Archetype string
	Nom       string
	Pourquoi  string
}

// Cibles porte les composants du lot, dans l'ordre de la question.
//
// L'ORDRE N'EST PAS DECORATIF : les trois premiers sont la piste des OBJECTIFS ouverte par le
// verdict de `objectif_ti11_minuteurs_verdict_test.go` (« la VALEUR du compte a rebours, si elle
// existe, est derriere l'index — dans ti=0 i15 »), les suivants sont la piste des VEHICULES
// (distributeurs, filtres de reapparition), et les derniers sont les temoins de negatif : les
// seuls composants de l'archetype vehicule dont le nom contient un mot de temps.
var Cibles = []Cible{
	{"ti=0 i15 · ti=2 i15", "managed-engine-timers-component",
		"le BASSIN de minuteurs ; ti=11 i0 et ti=12 i10 n'en portent que l'INDEX"},
	{"ti=11 i0", "managed-objective-timers-component",
		"le couple d'index de minuteur de l'objectif (porte, fige — verdict 2026-09-01)"},
	{"ti=12 i10", "managed-navpoint-timers-component",
		"le meme couple d'index, cote navpoint"},
	{"ti=11 i32", "managed-objective-outro-phase-duration-component",
		"une DUREE d'objectif, portee, jamais interpretee"},
	{"ti=43 i37", "device-object-dispenser-timer-component",
		"le minuteur du DISTRIBUTEUR d'objets — le seul minuteur de generateur nomme de l'image"},
	{"ti=43 i36", "device-dispenser-state-component",
		"l'etat du distributeur ; `object_dispenser_primary_respawning` est une chaine du jeu"},
	{"ti=43 i32", "device-dispenser-state-flags-component",
		"les drapeaux du meme distributeur"},
	{"ti=43 i31", "device-dispenser-monitors-changed-component",
		"ce que le distributeur SURVEILLE (l'objet qu'il doit refaire apparaitre)"},
	{"ti=20 i2", "spawn-filter-filters-component",
		"le seul composant non porte de l'archetype des filtres de reapparition"},
	{"ti=40 i37", "vehicle-emp-timer-component",
		"TEMOIN DE NEGATIF : le seul composant de l'archetype vehicule dont le nom porte un temps"},
	{"ti=29 i0", "managed-object-participant-respawn-block-component",
		"le blocage de reapparition d'un JOUEUR — pour ecarter la confusion avec l'objet"},
	{"ti=0 i11 · ti=2 i11", "game-engine-soft-ceilings-component",
		"LE BLOQUANT UNIQUE de la route vers le bassin (113/113 records, mesure 3.7)"},
	{"ti=0 i13 · ti=2 i13", "game-engine-disabled-kill-volume-flags-component",
		"le bloquant suivant presume sur la meme route"},
	{"ti=0 i14 · ti=2 i14", "GameEngineComposerLetterboxComponent",
		"le dernier composant avant i15 ; nom en CamelCase, pas en kebab"},
	{"ti=0 i9", "game-engine-screen-sequence-component",
		"TEMOIN : porte, sur la meme route, pour recaler la grammaire des voisins"},
	{"ti=0 i5", "game-engine-round-timer-component",
		"TEMOIN POSITIF : un minuteur PORTE et exploite, pour comparer les grammaires"},
}
