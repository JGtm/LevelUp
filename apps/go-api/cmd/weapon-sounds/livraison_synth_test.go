package main

// livraison_synth_test.go — LE JEU D'ENTREES SYNTHETIQUE du mode `livrer`, et lui seul.
//
// POURQUOI IL EST GENERE PAR DU CODE ET NON VERSIONNE. Les `.wav` sources du chantier
// sons-armes n'existent plus sur aucun poste (livraison du 2026-08-16, seuls les `.json`
// survivent) et pesaient des centaines de mega-octets. Ce generateur reconstruit, en
// quelques dizaines de lignes deterministes, un chantier miniature qui exerce TOUS les
// chemins de decision du mode : les deux roles confirmes, le vote de coup, le vote
// d'evenement en repli, le dedoublonnage variante/base, l'introuvable, le sans-fichier,
// mono et stereo, 44,1 kHz et 48 kHz, `_EMBARQUES`, et les fourchettes RANGED degenerees.
//
// LES SORTIES ATTENDUES, ELLES, NE SONT PAS GENEREES : `testdata/livraison/goldens/` a ete
// produit UNE FOIS par `_outils/livraison.py` (le script Python d'origine, hors depot) sur
// l'arborescence que ce fichier ecrit. Regenerer un golden avec le code Go annulerait la
// preuve — cf. l'en-tete de livraison_golden_test.go pour la procedure.

import (
	"encoding/binary"
	"os"
	"path/filepath"
)

// --- fabrique de `.wav` PCM 16 bits ---

// livraisonSynthPCM rend des echantillons DETERMINISTES (generateur congruentiel lineaire),
// d'amplitude assez large pour que le melange des couches depasse 30 000 et declenche donc
// l'attenuation de `livraisonEcrireCoup` : un jeu d'entrees silencieux rendrait le test
// aveugle a l'arrondi du mixage, qui est precisement ce qu'il doit garder.
func livraisonSynthPCM(graine uint32, n int) []int16 {
	out := make([]int16, n)
	x := graine
	for i := range out {
		x = x*1664525 + 1013904223
		out[i] = int16(int32((x>>16)%60000) - 30000)
	}
	return out
}

// livraisonSynthWav ecrit un `.wav` PCM canonique (en-tete de 44 octets, le meme que celui
// que le module `wave` de Python produit).
func livraisonSynthWav(chemin string, canaux, taux, trames int, graine uint32) error {
	ech := livraisonSynthPCM(graine, trames*canaux)
	corps := make([]byte, len(ech)*2)
	for i, v := range ech {
		binary.LittleEndian.PutUint16(corps[i*2:], uint16(v))
	}
	if err := os.MkdirAll(filepath.Dir(chemin), 0o755); err != nil {
		return err
	}
	return livraisonEcrireWavBrut(chemin, canaux, taux, 16, corps)
}

// livraisonSynthFichier : un `.wav` du chantier miniature.
type livraisonSynthFichier struct {
	Rel    string // chemin relatif a la racine du chantier
	Canaux int
	Taux   int
	Trames int
	Graine uint32
}

// livraisonSynthFichiers : les sources du chantier miniature.
//
// Chaque ligne porte le chemin de decision qu'elle exerce ; les retirer ou les renommer
// change les goldens, donc la preuve.
var livraisonSynthFichiers = []livraisonSynthFichier{
	// Covenant_provoker : l'unique arme RENDUE (melange de couches, generateur MT19937).
	{"Covenant_provoker/0.10s_101.wav", 2, 48000, 4800, 11}, // couche A, candidat 1
	{"Covenant_provoker/0.10s_102.wav", 2, 48000, 4800, 22}, // couche A, candidat 2
	{"Covenant_provoker/0.20s_201.wav", 2, 44100, 8820, 33}, // couche B : 44,1 kHz => rejetee APRES tirage
	{"Covenant_provoker/0.08s_302.wav", 1, 48000, 3840, 44}, // couche C : MONO, duplique en stereo
	// Forerunner_sentinelbeam : le `.wem` du role n'existe QUE dans `_EMBARQUES`.
	{"Forerunner_sentinelbeam/0.05s_777.wav", 2, 48000, 2400, 55},
	{"Forerunner_sentinelbeam_EMBARQUES/0.12s_503433748.wav", 2, 48000, 5760, 66},
	// UNSC_assaultrifle : la source LONGUE (> 1,25 s) et a 44,1 kHz — le temoin de la
	// troncature a 1,2 s, format d'origine preserve.
	{"UNSC_assaultrifle/1.30s_401.wav", 1, 44100, 57330, 77},
	{"UNSC_assaultrifle_infectee/0.06s_701.wav", 2, 48000, 2880, 88},
	{"Banished_mangler/0.09s_555.wav", 2, 48000, 4320, 99},
	{"Covenant_needler/0.07s_666.wav", 2, 48000, 3360, 111},
	{"UNSC_relatifdrive/0.04s_801.wav", 2, 48000, 1920, 122},
}

// --- les cinq fichiers de `_donnees` ---

// livraisonSynthLot1 : deux armes seulement (les seules dont un EVENEMENT est consulte).
//
// `sb_010_wea_cv_provoker.pck` et `sb_010_wea_un_assaultrifle.pck` sont des chemins Windows
// ABSOLUS, comme ceux du chantier reel.
const livraisonSynthLot1 = `{"armes": [
  {"pck": "C:\\Steam\\SFX\\sb_010_wea_cv_provoker.pck", "evenements": [
    {"id_event": 3140584475, "couches": [
      {"wems_candidats": [101, 102], "gains_db": {"101": -3.0, "102": -3.0},
       "variation": {"volume_db": {"bas": -2, "haut": 2}, "pitch_cents": {"bas": -50, "haut": 50}, "gain_db_couche": 4.5}},
      {"wems_candidats": [999], "gains_db": {}},
      {"wems_candidats": [201], "gains_db": {"201": 0}},
      {"wems_candidats": [301, 302], "gains_db": {"302": -1.0},
       "variation": {"volume_db": {"bas": -7, "haut": 7}, "pitch_cents": {"bas": -7, "haut": 7}, "gain_db_couche": 4.5}}
    ]}
  ]},
  {"pck": "C:\\Steam\\SFX\\sb_010_wea_un_assaultrifle.pck", "evenements": [
    {"id_event": 2863267841, "couches": [
      {"wems_candidats": [401], "gains_db": {},
       "variation": {"volume_db": {"bas": -60, "haut": 60}, "pitch_cents": {"bas": -60, "haut": 60}, "gain_db_couche": 9.0}}
    ]},
    {"id_event": 2863267842, "couches": [
      {"wems_candidats": [401], "gains_db": {},
       "variation": {"volume_db": {"bas": -1, "haut": 1}, "pitch_cents": {"bas": 0, "haut": 0}, "gain_db_couche": 4.5}},
      {"wems_candidats": [401], "gains_db": {},
       "variation": {"volume_db": {"bas": -9, "haut": 9}, "pitch_cents": {"bas": -9, "haut": 9}, "gain_db_couche": 4.5}}
    ]}
  ]}
]}`

// livraisonSynthLot2 : les fourchettes de repli, et TOUS les cas de formatage de nombre.
//
// `C:sb_010_wea_un_relatifdrive.pck` est un chemin RELATIF AU LECTEUR (pas de separateur
// apres le `:`) : `ntpath.basename` le coupe apres le prefixe de lecteur, un `LastIndexAny`
// naif ne le coupe pas du tout et perd l'arme (constat C8 de la revue R1).
const livraisonSynthLot2 = `[
  {"pck": "C:\\Steam\\SFX\\sb_010_wea_fr_sentinelbeam.pck",
   "variation": {"volume_db": {"bas": -3.0, "haut": 2.0}, "pitch_cents": {"bas": 0.0, "haut": 1e3}, "gain_db_couche": 0.0}},
  {"pck": "C:\\Steam\\SFX\\sb_010_wea_bt_mangler.pck",
   "variation": {"volume_db": {"bas": 0, "haut": 0}, "pitch_cents": {"bas": -85, "haut": 80}, "gain_db_couche": 0.0}},
  {"pck": "C:\\Steam\\SFX\\sb_010_wea_cv_needler.pck",
   "variation": {"volume_db": {"bas": -0, "haut": 1.5}, "pitch_cents": {"bas": 1234567.5, "haut": 1e-5}, "gain_db_couche": 0.0}},
  {"pck": "C:sb_010_wea_un_relatifdrive.pck",
   "variation": {"volume_db": {"bas": -4.25, "haut": 0}, "gain_db_couche": 0.0}}
]`

// livraisonSynthManifeste : dix entrees — dont une sans cle et une sans vote, toutes deux
// hors des candidats, et la variante qui partage la cle de sa base.
const livraisonSynthManifeste = `{"armes": [
  {"dossier": "Covenant_provoker", "cle": "hinf_ravager", "nom_fr": "Provocateur"},
  {"dossier": "Forerunner_sentinelbeam", "cle": "hinf_sentinel_beam", "nom_fr": "Rayon de sentinelle"},
  {"dossier": "UNSC_assaultrifle", "cle": "hinf_ma40_ar", "nom_fr": "MA40 AR"},
  {"dossier": "UNSC_assaultrifle_infectee", "cle": "hinf_ma40_ar", "nom_fr": "MA40 AR (infectee)"},
  {"dossier": "Banished_mangler", "cle": "hinf_mangler", "nom_fr": "Mutilateur"},
  {"dossier": "Covenant_needler", "cle": "hinf_needler", "nom_fr": "Needler"},
  {"dossier": "UNSC_relatifdrive", "cle": "hinf_relatifdrive", "nom_fr": "Arme au chemin relatif"},
  {"dossier": "UNSC_sidekick", "cle": "hinf_mk50_sidekick", "nom_fr": "MK50 Sidekick"},
  {"dossier": "UNSC_bulldog", "cle": "hinf_cqs48_bulldog", "nom_fr": "CQS48 Bulldog"},
  {"dossier": "UNSC_sanscle", "cle": null, "nom_fr": "Sans cle canonique"},
  {"dossier": "UNSC_sansvote", "cle": "hinf_sansvote", "nom_fr": "Sans vote"}
]}`

// livraisonSynthCoups : trois rendus pour le meme (mode, perspective) — la boucle Python
// n'a PAS de `break`, c'est donc le DERNIER qui gagne.
const livraisonSynthCoups = `{
  "UNSC_assaultrifle": {"rendus": [
    {"mode": 0, "perspective": "3p", "event": "aaaa0003"},
    {"mode": 0, "perspective": "1p", "event": "aaaa0001"},
    {"mode": 0, "perspective": "1p", "event": "aaaa0002"}
  ]}
}`

// livraisonSynthVotes : l'ordre du tableau est DELIBEREMENT contraire a l'ordre de tri
// (3p avant 1p, m1 avant m0) — le tri par `clefCoup` doit le remettre a l'endroit.
//
// Le premier vote de `Covenant_needler` porte un exemple VIDE : Python rend une valeur
// fausse et passe au vote suivant (constat C7 de la revue R1).
const livraisonSynthVotes = `{"votes": [
  {"arme": "UNSC_assaultrifle", "groupe": "_coup_m0_3p", "vote": "garder", "exemples_retenus": ["9.99s_inexistant.wav"]},
  {"arme": "UNSC_assaultrifle", "groupe": "_coup_m0_1p", "vote": "favori", "exemples_retenus": ["1.30s_401.wav"]},
  {"arme": "UNSC_assaultrifle", "groupe": "_coup_m1_1p", "vote": "rejeter", "exemples_retenus": ["9.99s_rejete.wav"]},
  {"arme": "UNSC_assaultrifle_infectee", "groupe": "_coup_m0_1p", "vote": "garder", "exemples_retenus": ["0.06s_701.wav"]},
  {"arme": "Banished_mangler", "groupe": "ev_00000001", "vote": "garder", "exemples_retenus": ["_prive.wav"]},
  {"arme": "Banished_mangler", "groupe": "ev_00c0ffee", "vote": "garder", "exemples_retenus": ["0.09s_555.wav"]},
  {"arme": "Covenant_needler", "groupe": "_coup_m1_1p", "vote": "garder", "exemples_proposes": ["0.07s_666.wav"]},
  {"arme": "Covenant_needler", "groupe": "_coup_m0_1p", "vote": "garder", "exemples_retenus": [""]},
  {"arme": "UNSC_relatifdrive", "groupe": "_coup_m0_1p", "vote": "garder", "exemples_retenus": ["0.04s_801.wav"]},
  {"arme": "UNSC_sidekick", "groupe": "_coup_m0_1p", "vote": "garder", "exemples_retenus": ["0.10s_absent.wav"]},
  {"arme": "UNSC_bulldog", "groupe": "_coup_m0_1p", "vote": "garder", "exemples_retenus": [], "exemples_proposes": []}
]}`

// livraisonSynthDonnees : les cinq fichiers de `_donnees`, par nom.
var livraisonSynthDonnees = map[string]string{
	"lot1.json":        livraisonSynthLot1,
	"lot2.json":        livraisonSynthLot2,
	"manifeste.json":   livraisonSynthManifeste,
	"coups.json":       livraisonSynthCoups,
	"votes-final.json": livraisonSynthVotes,
}

// livraisonEcrireJeuSynthetique materialise le chantier miniature sous `racine` : les
// dossiers d'armes avec leurs `.wav`, et `_donnees` avec ses cinq fichiers.
func livraisonEcrireJeuSynthetique(racine string) error {
	for _, f := range livraisonSynthFichiers {
		chemin := filepath.Join(racine, filepath.FromSlash(f.Rel))
		if err := livraisonSynthWav(chemin, f.Canaux, f.Taux, f.Trames, f.Graine); err != nil {
			return err
		}
	}
	donnees := filepath.Join(racine, "_donnees")
	if err := os.MkdirAll(donnees, 0o755); err != nil {
		return err
	}
	for nom, contenu := range livraisonSynthDonnees {
		if err := os.WriteFile(filepath.Join(donnees, nom), []byte(contenu), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// livraisonEcrireDepotSynthetique prepare le depot cible : le dossier des sons avec ses DEUX
// temoins — un `hinf_*.wav` perime que le miroir doit effacer, et un son d'evenement du pack
// utilisateur que le miroir ne doit JAMAIS toucher.
func livraisonEcrireDepotSynthetique(depot string) error {
	sons := filepath.Join(depot, "static", "sounds", "halo_infinite")
	if err := os.MkdirAll(sons, 0o755); err != nil {
		return err
	}
	if err := livraisonSynthWav(filepath.Join(sons, "hinf_perime.wav"), 2, 48000, 480, 1); err != nil {
		return err
	}
	if err := livraisonSynthWav(filepath.Join(sons, "melee_kill.wav"), 2, 48000, 480, 2); err != nil {
		return err
	}
	return os.MkdirAll(filepath.Join(depot, "apps", "web", "src", "features", "match-replay"), 0o755)
}
