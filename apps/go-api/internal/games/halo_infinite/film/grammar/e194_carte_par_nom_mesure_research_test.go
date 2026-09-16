//go:build research

package grammar

// e194_carte_par_nom_mesure_research_test.go — LOT 1.9.4, LA MESURE AVANT DE CODER.
//
// # LA QUESTION, POSÉE AVANT TOUTE LIGNE DE PRODUCTION
//
// L'IDENTITÉ de la carte d'un film se décidait, AVANT CE LOT, par une SIGNATURE DE LARGEURS :
// `DetectFilmMapEntry` — supprimée par le lot — lisait le découpage d'i0 dans le film
// ([DetectI0Layout]), puis retenait l'entrée de catalogue dont les `axisWidths` coïncidaient, et
// n'acceptait que s'il y en avait EXACTEMENT UNE. Le nom de carte du match, lui, est une DONNÉE :
// le collecteur le résolvait déjà par la base pour l'autre passe, et le catalogue est indexé par
// ce nom.
//
// Trois populations rendent la signature incapable de décider. Les deux premières étaient
// attendues, la TROISIÈME est ce que cette mesure a trouvé :
//
//	les CARTES JUMELLES — deux entrées du catalogue qui partagent leurs largeurs d'axe. La
//	  signature y rend plusieurs candidats, `len(hits) != 1`, et les distances de touche sont
//	  désactivées EN SILENCE (audit 0.E, A2 ; rapport F.0 §0.3 et §6 réserve 2, qui en comptait
//	  SIX — la mesure en compte 68 sur 79) ;
//	les cartes dont la signature ne retrouve AUCUNE entrée ;
//	les cartes dont la signature retrouve UNE entrée, ET QUE CE N'EST PAS LA BONNE. Live Fire
//	  est de celles-là : l'auto-détection impute son bit d'index de région à l'axe X et rend
//	  `13/12/11`, la signature d'`aquarius`. D2 (1.9.2) croyait les distances désactivées ; elles
//	  étaient calculées dans l'AABB d'une autre carte.
//
// # CE QUE CET INSTRUMENT MESURE, ET CE QU'IL NE MESURE PAS
//
// [TestE194CartesJumellesDuCatalogue] ne lit QUE le catalogue versionné, sans film, et rend la
// liste des classes d'équivalence de signature. C'est la mesure (c) du lot.
//
// [TestE194SignatureContreNomDeMatch] croise, film par film, la carte que la SIGNATURE rend et
// celle que le NOM DE MATCH donne : accord, désaccord, signature ambiguë, signature hors
// catalogue. C'est la mesure (b). Garde `CHUNK00_FILMS` (répertoires absolus séparés par `;`) ;
// la table des cartes est celle, versionnée, de l'instrument du lot 1.9.2 (`e192CarteDuFilm`) —
// une seule table pour les deux lots, jamais une seconde copie.
//
// # TAG `research` (2026-09-15) : POURQUOI LA MESURE (c) NE TOURNE PAS EN CI NON PLUS
//
// Ce fichier porte `//go:build research` comme tout instrument du dépôt : le job de couverture
// CI a dépassé ses 600 s sur `filmdec` à cause des instruments, qui tournent désormais sous
// `go test -tags research`. La mesure (c) n'a pourtant besoin d'aucun film et coûte 0,06 s — la
// règle s'applique quand même, parce qu'un tag posé « sauf exceptions » ne se tient pas.
//
// CE QUI GARDE LE PRÉSUPPOSÉ EN CI, LUI, C'EST UN AUTRE TEST : dans
// `sync/killcollector/hits_carte_par_nom_test.go`, `TestJumellesDuCatalogueSontIndistinguablesParSignature`
// vérifie sur le catalogue versionné que les deux cartes du témoin de mutation partagent bien
// leur signature et non leurs bornes. Sans lui, le tag ferait disparaître du run par défaut la
// seule vérification que la mutation du lot a un sens.
//
// Aucune écriture, aucune base, aucun artefact, aucune cuisson : la détection ne balaye que les
// six premiers chunks (`detectMaxChunks`), le reste du film n'est pas parcouru.
//
//	CHUNK00_FILMS='C:/.../film_chunks/bcb6d393;C:/.../film_chunks/60ae07c4' \
//	  CGO_ENABLED=0 go test -tags research ./internal/games/halo_infinite/film/filmdec/ \
//	  -run '^TestE194SignatureContreNomDeMatch$' -v -count=1 -timeout 3600s

import (
	"fmt"
	"levelup/go-api/internal/games/halo_infinite/film/profile"
	"path/filepath"
	"sort"
	"testing"
)

// e194Classe : une classe d'équivalence de signature du catalogue — les cartes qu'une signature
// de largeurs ne sait pas distinguer.
type e194Classe struct {
	Largeurs   [3]uint
	RegionBits []uint
	Cartes     []string
}

// e194ClassesDeSignature groupe les entrées du catalogue par largeurs d'axe — LA CLÉ EXACTE que
// `DetectFilmMapEntry` comparait (`e.AxisWidths == lay.AxisW`), et rien d'autre.
//
// LES BITS D'INDEX DE RÉGION SONT RELEVÉS MAIS NE GROUPENT PAS, et c'est délibéré : la signature
// ne les consulte pas. Les afficher dit si une clé PLUS FINE séparerait la classe — elle ne le
// ferait que si le découpage détecté portait la bonne porte, ce que la mesure (b) réfute sur
// Live Fire.
func e194ClassesDeSignature(cat *MapQuantCatalog) []e194Classe {
	par := map[[3]uint][]string{}
	bits := map[[3]uint]map[uint]bool{}
	for nom, e := range cat.Maps {
		par[e.AxisWidths] = append(par[e.AxisWidths], nom)
		if bits[e.AxisWidths] == nil {
			bits[e.AxisWidths] = map[uint]bool{}
		}
		bits[e.AxisWidths][e.EffectiveRegionIndexBits()] = true
	}
	out := make([]e194Classe, 0, len(par))
	for w, noms := range par {
		sort.Strings(noms)
		var rb []uint
		for b := range bits[w] {
			rb = append(rb, b)
		}
		sort.Slice(rb, func(i, j int) bool { return rb[i] < rb[j] })
		out = append(out, e194Classe{Largeurs: w, RegionBits: rb, Cartes: noms})
	}
	sort.Slice(out, func(i, j int) bool {
		if len(out[i].Cartes) != len(out[j].Cartes) {
			return len(out[i].Cartes) > len(out[j].Cartes)
		}
		return fmt.Sprint(out[i].Largeurs) < fmt.Sprint(out[j].Largeurs)
	})
	return out
}

// TestE194CartesJumellesDuCatalogue — MESURE (c) : les cartes qu'une signature ne distingue pas.
func TestE194CartesJumellesDuCatalogue(t *testing.T) {
	cat, err := LoadMapQuantCatalog(e191bCatalogue())
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	classes := e194ClassesDeSignature(cat)
	jumelees, classesJumelles := 0, 0
	t.Logf("######## 1.9.4 — CLASSES D'ÉQUIVALENCE DE SIGNATURE, %d cartes au catalogue ########",
		len(cat.Maps))
	for _, c := range classes {
		if len(c.Cartes) < 2 {
			continue
		}
		classesJumelles++
		jumelees += len(c.Cartes)
		t.Logf("  largeurs %v  bits de region %v  %d cartes : %v",
			c.Largeurs, c.RegionBits, len(c.Cartes), c.Cartes)
	}
	t.Logf("  TOTAL : %d classes ambigues, %d cartes indistinguables par signature sur %d",
		classesJumelles, jumelees, len(cat.Maps))
	t.Logf("  (une carte de classe ambigue rendait `len(hits) != 1` dans DetectFilmMapEntry : " +
		"distances de touche desactivees en silence)")
	if jumelees == 0 {
		t.Error("aucune carte jumelle mesuree : le catalogue ou la cle de groupement a change, " +
			"le gain annonce par le lot (6 cartes jumelles) ne se verifie plus")
	}
}

// e194Ligne : une ligne de la mesure (b), un film.
type e194Ligne struct {
	Film, Carte string
	// ParNom : l'entrée que le NOM DE MATCH donne, et son erreur.
	ParNom    MapQuantEntry
	ErrNom    error
	Detecte   profile.I0Layout
	ErrDetect error
	// Candidats : les cartes du catalogue dont les largeurs égalent la signature détectée.
	Candidats []string
}

// Verdict classe la ligne : accord, desaccord, ambigue (jumelles) ou hors catalogue.
func (l e194Ligne) Verdict() string {
	switch {
	case l.ErrDetect != nil:
		return "signature illisible"
	case len(l.Candidats) == 0:
		return "hors catalogue"
	case len(l.Candidats) > 1:
		return "ambigue"
	case l.ErrNom != nil:
		return "nom hors catalogue"
	case NormalizeMapName(l.Candidats[0]) == NormalizeMapName(l.Carte):
		return "accord"
	default:
		return "desaccord"
	}
}

// TestE194SignatureContreNomDeMatch — MESURE (b) : la carte que la signature rend contre celle
// que le nom de match donne, film par film.
func TestE194SignatureContreNomDeMatch(t *testing.T) {
	dirs := chunk00Films(t, "CHUNK00_FILMS")
	cat, err := LoadMapQuantCatalog(e191bCatalogue())
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	var lignes []e194Ligne
	for _, dir := range dirs {
		l, ok := e194Mesure(t, cat, dir)
		if !ok {
			continue
		}
		lignes = append(lignes, l)
	}
	sort.Slice(lignes, func(i, j int) bool { return lignes[i].Film < lignes[j].Film })
	e194Rapport(t, lignes)
}

// e194Mesure remplit la ligne d'un film : carte nommée, entrée du catalogue, signature détectée
// et candidats de cette signature.
func e194Mesure(t *testing.T, cat *MapQuantCatalog, dir string) (e194Ligne, bool) {
	t.Helper()
	l := e194Ligne{Film: filepath.Base(dir)}
	carte, ok := e192CarteDuFilm[l.Film]
	if !ok {
		t.Logf("  %-10s SAUTÉ : film absent de la table des cartes versionnée", l.Film)
		return l, false
	}
	l.Carte = carte
	l.ParNom, l.ErrNom = cat.Lookup(carte)
	lay, _, derr := detectI0Layout(dir)
	l.Detecte, l.ErrDetect = lay, derr
	if derr != nil {
		return l, true
	}
	for nom, e := range cat.Maps {
		if e.AxisWidths == lay.AxisW {
			l.Candidats = append(l.Candidats, nom)
		}
	}
	sort.Strings(l.Candidats)
	return l, true
}

// e194Rapport colle le tableau et les totaux par verdict.
func e194Rapport(t *testing.T, lignes []e194Ligne) {
	t.Helper()
	t.Logf("######## 1.9.4 — SIGNATURE DE LARGEURS CONTRE NOM DE MATCH, %d films ########", len(lignes))
	t.Logf("  %-10s %-22s %-30s %-30s %-6s %-18s %s",
		"film", "carte (nom de match)", "catalogue par nom", "signature detectee", "cands", "verdict", "candidats")
	totaux := map[string]int{}
	for _, l := range lignes {
		parNom := l.ParNom.Layout().String()
		if l.ErrNom != nil {
			parNom = "ABSENTE DU CATALOGUE"
		}
		sig := "ERREUR"
		if l.ErrDetect == nil {
			sig = l.Detecte.String()
		}
		v := l.Verdict()
		totaux[v]++
		t.Logf("  %-10s %-22s %-30s %-30s %-6d %-18s %v",
			l.Film, l.Carte, parNom, sig, len(l.Candidats), v, l.Candidats)
	}
	var cles []string
	for k := range totaux {
		cles = append(cles, k)
	}
	sort.Strings(cles)
	for _, k := range cles {
		t.Logf("  VERDICT %-18s : %d film(s)", k, totaux[k])
	}
	t.Logf("  LECTURE : tout verdict autre qu'« accord » est un film dont les distances de touche " +
		"sont desactivees en silence aujourd'hui, et que le nom de match resout.")
}
