package main

// livraison.go — mode `livrer` : LIVRAISON des sons d'armes du rejeu 2D, fusion avec le
// moteur sonore de feat/v75 (etape 7 de `.ai/V7.5/RECETTE_SONS_ARMES.md`).
//
// PORTAGE FIDELE de `_outils/livraison.py` (archive Desktop du chantier sons-armes, HORS
// DEPOT, lecture seule), au meme titre que pck_dump.go a ferme le maillon `akpk_unpack.py`
// (constat H2 du registre `.ai/AUDIT_V75_DEPUIS_V7.3.0_2026-09-05.md`). MEME ALGORITHME,
// MEMES STRUCTURES DE DONNEES (lot1.json, lot2.json, manifeste.json, coups.json,
// votes-final.json, produits par les etapes 2/3/5/6 de la recette — hors depot, non portes
// par ce lot), MEME GENERATEUR PSEUDO-ALEATOIRE (livraison_mt19937.go, verifie bit a bit
// contre CPython) pour l'unique arme rendue par evenement plutot que copiee depuis un
// fichier vote (Covenant_provoker -> hinf_ravager).
//
// CE QUE CE MODE REMPLACE, ET RIEN D'AUTRE : les fichiers d'ARMES (`hinf_*.wav`) de
// `static/sounds/halo_infinite/`. Les sons d'evenements du pack utilisateur (lancers,
// explosions, melee_kill, camo_*, overshield_*) ne sont JAMAIS touches. Il GENERE AUSSI
// `weaponSoundVariations.ts` : les fourchettes RANGED (volume, hauteur) par stem, extraites
// des banks du jeu.
//
// Usage : -mode livrer -donnees <dossier _donnees> [-sons <racine du chantier sons>]
//
//	[-depot <racine du depot cible>]
//
// -sons par defaut = le parent de -donnees (les dossiers d'armes et `_donnees` sont
// SIBLINGS sous la racine du chantier, comme dans l'archive Desktop d'origine). -depot par
// defaut = title.FindRepoRoot().
//
// Armes NON livrables et pourquoi : le moteur joint par cle canonique du registre
// (weapon_names.toml). Le Mutilator, les tourelles et les armes de PNJ n'y ont pas
// d'entree : leurs sons votes restent dans l'archive Desktop, pas dans l'app.

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// --- ROLES CONFIRMES PAR L'UTILISATEUR (handoff sons-armes, section 7) ---

type livraisonRole struct {
	Genre string // "rendre_event" ou "wem_source"
	Val   uint32
}

var livraisonRoles = map[string]livraisonRole{
	"Covenant_provoker":       {"rendre_event", 0xbb31841b}, // tir 3 coups = LE son du rejeu
	"Forerunner_sentinelbeam": {"wem_source", 503433748},    // le court = LE son du rejeu
}

// --- forme JSON partagee (variation RANGED) ---

// livraisonPlage porte le TEXTE JSON d'origine (json.Number) plutot qu'un float64 : la
// sortie TS doit reproduire "%s" % valeur de Python a l'identique (un entier JSON reste un
// entier a l'affichage, un flottant garde son point).
type livraisonPlage struct {
	Bas  json.Number `json:"bas"`
	Haut json.Number `json:"haut"`
}

func (p livraisonPlage) basF() float64 {
	v, _ := p.Bas.Float64()
	return v
}

func (p livraisonPlage) hautF() float64 {
	v, _ := p.Haut.Float64()
	return v
}

type livraisonVariation struct {
	VolumeDB     *livraisonPlage `json:"volume_db"`
	PitchCents   *livraisonPlage `json:"pitch_cents"`
	GainDBCouche float64         `json:"gain_db_couche"`
}

// livraisonVariationOut : la variation FILTREE (fourchettes non degenerees seulement) —
// forme rendue par variationDe en Python (un dict avec 0, 1 ou 2 cles).
type livraisonVariationOut struct {
	VolumeDB   *livraisonPlage
	PitchCents *livraisonPlage
}

// --- lot1.json ---

type livraisonLot1Root struct {
	Armes []livraisonLot1Arme `json:"armes"`
}

type livraisonLot1Arme struct {
	PCK        string               `json:"pck"`
	Evenements []livraisonLot1Event `json:"evenements"`
}

type livraisonLot1Event struct {
	IDEvent uint32                `json:"id_event"`
	Couches []livraisonLot1Couche `json:"couches"`
}

type livraisonLot1Couche struct {
	WemsCandidats []uint32            `json:"wems_candidats"`
	GainsDB       map[string]float64  `json:"gains_db"`
	Variation     *livraisonVariation `json:"variation"`
}

// --- lot2.json (tableau racine) ---

type livraisonLot2Arme struct {
	PCK       string              `json:"pck"`
	Variation *livraisonVariation `json:"variation"`
}

// --- manifeste.json ---

type livraisonManifesteRoot struct {
	Armes []livraisonManifesteArme `json:"armes"`
}

type livraisonManifesteArme struct {
	Dossier string  `json:"dossier"`
	Cle     *string `json:"cle"`
	NomFr   *string `json:"nom_fr"`
}

// --- coups.json (objet racine : dossier -> entree) ---

type livraisonCoupsEntree struct {
	Rendus []livraisonCoupRendu `json:"rendus"`
}

type livraisonCoupRendu struct {
	Mode        int    `json:"mode"`
	Perspective string `json:"perspective"`
	Event       string `json:"event"`
}

// --- votes-final.json ---

type livraisonVotesRoot struct {
	Votes []livraisonVote `json:"votes"`
}

type livraisonVote struct {
	Arme             string   `json:"arme"`
	Groupe           string   `json:"groupe"`
	Vote             string   `json:"vote"`
	ExemplesRetenus  []string `json:"exemples_retenus"`
	ExemplesProposes []string `json:"exemples_proposes"`
}

// --- donnees et etat, regroupes pour ne pas depasser 5 parametres de fonction ---

type livraisonDonnees struct {
	Lot1           map[string]livraisonLot1Arme
	Lot2           map[string]livraisonLot2Arme
	Manifeste      map[string]livraisonManifesteArme
	OrdreManifeste []string // ordre d'apparition dans manifeste.json, dossiers dedupliques
	Coups          map[string]livraisonCoupsEntree
	Votes          []livraisonVote
}

type livraisonChemins struct {
	SonsRacine string
	Cible      string
	Tmp        string
}

type livraisonEtat struct {
	Livres     map[string]string // cle canonique -> dossier qui l'a livree
	Variations map[string]*livraisonVariationOut
	Lignes     []string
}

// --- joli : normalise le nom d'un .pck en dossier canonique (coups_lot.py:joli) ---

var joliRe = regexp.MustCompile(`^sb_010_(wea|tur|whizby)_(un|cv|bt|fr|pl|prj)_(.+)$`)

var joliFactions = map[string]string{
	"un": "UNSC", "cv": "Covenant", "bt": "Banished",
	"fr": "Forerunner", "pl": "Divers", "prj": "Projectiles",
}

var joliPrefixes = map[string]string{"wea": "", "tur": "Tourelle_", "whizby": "Whizby_"}

func joliDossier(pck string) string {
	base := strings.TrimSuffix(filepath.Base(pck), filepath.Ext(pck))
	m := joliRe.FindStringSubmatch(base)
	if m == nil {
		return strings.ReplaceAll(base, "sb_010_", "")
	}
	k, f, n := m[1], m[2], m[3]
	fac, ok := joliFactions[f]
	if !ok {
		fac = f
	}
	return fac + "_" + joliPrefixes[k] + n
}

// --- votes ---

func livraisonVotesDe(dossier string, votes []livraisonVote) []livraisonVote {
	var out []livraisonVote
	for _, v := range votes {
		if v.Arme == dossier && (v.Vote == "garder" || v.Vote == "favori") {
			out = append(out, v)
		}
	}
	return out
}

func livraisonFichierDuVote(v livraisonVote) (string, bool) {
	if len(v.ExemplesRetenus) > 0 {
		return v.ExemplesRetenus[0], true
	}
	if len(v.ExemplesProposes) > 0 {
		return v.ExemplesProposes[0], true
	}
	return "", false
}

var clefCoupRe = regexp.MustCompile(`^_coup_m(\d+)_(1p|3p)$`)

// livraisonClefCoup rend la cle de tri (mode, rang perspective ; 1p avant 3p) — port de
// coups_lot.py:clefCoup. (99, 9) si le groupe ne correspond pas au patron.
func livraisonClefCoup(groupe string) (int, int) {
	m := clefCoupRe.FindStringSubmatch(groupe)
	if m == nil {
		return 99, 9
	}
	mode, _ := strconv.Atoi(m[1])
	rang := 1
	if m[2] == "1p" {
		rang = 0
	}
	return mode, rang
}

// livraisonChoix : la source choisie pour un dossier, et le contexte qui va avec — port du
// triplet (source, evHex, groupe) que choixDossier() rend en Python. Source vide = "SANS
// FICHIER" ; prefixe "__RENDRE__<hex>" = evenement a rendre plutot que fichier a copier
// (meme convention litterale que le script Python, pour ne pas re-arbitrer le control flow).
type livraisonChoix struct {
	Source string
	EvHex  string
	Groupe string
}

// livraisonChoixDossier decide QUELLE source livrer un dossier — port de
// livraison.py:choixDossier. Ordre de decision : role confirme, puis vote de coup (trie par
// mode/perspective), puis vote d'evenement en repli.
func livraisonChoixDossier(dossier, racine string, coups map[string]livraisonCoupsEntree, votes []livraisonVote) (livraisonChoix, error) {
	if role, ok := livraisonRoles[dossier]; ok {
		return livraisonChoixParRole(dossier, racine, role)
	}

	vs := livraisonVotesDe(dossier, votes)

	var coupVotes []livraisonVote
	for _, v := range vs {
		if strings.HasPrefix(v.Groupe, "_coup_") {
			coupVotes = append(coupVotes, v)
		}
	}
	sort.SliceStable(coupVotes, func(i, j int) bool {
		mi, pi := livraisonClefCoup(coupVotes[i].Groupe)
		mj, pj := livraisonClefCoup(coupVotes[j].Groupe)
		if mi != mj {
			return mi < mj
		}
		return pi < pj
	})
	for _, v := range coupVotes {
		f, ok := livraisonFichierDuVote(v)
		if !ok {
			continue
		}
		evHex := livraisonEvenementDuCoup(dossier, v.Groupe, coups)
		return livraisonChoix{Source: dossier + "/" + f, EvHex: evHex, Groupe: v.Groupe}, nil
	}

	for _, v := range vs {
		if !strings.HasPrefix(v.Groupe, "ev_") {
			continue
		}
		f, ok := livraisonFichierDuVote(v)
		if !ok || strings.HasPrefix(f, "_") {
			continue
		}
		chemin := f
		if !strings.Contains(f, "/") {
			chemin = dossier + "/" + f
		}
		return livraisonChoix{Source: chemin, EvHex: strings.TrimPrefix(v.Groupe, "ev_"), Groupe: v.Groupe}, nil
	}
	return livraisonChoix{}, nil
}

// livraisonChoixParRole traite les deux ARMES A ROLE CONFIRME (section 7 du handoff) :
// Covenant_provoker rend un evenement, Forerunner_sentinelbeam pointe un .wem source.
func livraisonChoixParRole(dossier, racine string, role livraisonRole) (livraisonChoix, error) {
	if role.Genre == "rendre_event" {
		hex := fmt.Sprintf("%08x", role.Val)
		return livraisonChoix{Source: "__RENDRE__" + hex, EvHex: hex, Groupe: "role:tir_3_coups"}, nil
	}
	index, err := livraisonIndexWems(racine, dossier)
	if err != nil {
		return livraisonChoix{}, err
	}
	p, ok := index[role.Val]
	if !ok {
		return livraisonChoix{}, nil
	}
	rel, err := filepath.Rel(racine, p)
	if err != nil {
		return livraisonChoix{}, err
	}
	return livraisonChoix{Source: filepath.ToSlash(rel), Groupe: "role:tir_court"}, nil
}

// livraisonEvenementDuCoup retrouve l'evenement (hex) associe a un vote "_coup_mN_1p|3p" dans
// coups.json — meme recherche que la boucle `for r in coups.get(dossier).get("rendus")` de
// livraison.py:choixDossier. Rend "" si rien ne correspond (comme `ev = None`).
func livraisonEvenementDuCoup(dossier, groupe string, coups map[string]livraisonCoupsEntree) string {
	m := clefCoupRe.FindStringSubmatch(groupe)
	if m == nil {
		return ""
	}
	modeN, _ := strconv.Atoi(m[1])
	persp := m[2]
	entry, ok := coups[dossier]
	if !ok {
		return ""
	}
	var ev string
	for _, r := range entry.Rendus {
		if r.Mode == modeN && r.Perspective == persp {
			ev = r.Event
		}
	}
	return ev
}
