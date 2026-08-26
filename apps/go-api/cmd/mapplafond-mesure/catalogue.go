package main

// catalogue.go — CE QUE L'INSTRUMENT LIT, ET RIEN D'AUTRE.
//
// Trois fichiers de reference VERSIONNES, plus l'installation du jeu :
//
//	map_backgrounds/{module}.json   calage du fond publie + sol joue (`playLevelZ`)
//	map_callouts.json               paves du designer (tag `levl`) — la reconnaissance de carte
//	map_objectives.json             ancres d'objectifs — le cadre et le sol joue de la cuisson
//
// Le corpus de rejeux, lui, est lu FILM PAR FILM (cf. frequentation.go). Rien n'ouvre une
// base, rien ne va sur le reseau.
//
// PERIMETRE : les cartes NATIVES seulement. Une carte FORGE n'a AUCUNE zone nommee (mesure sur
// les 8 canevas installes, cf. `replay.ErrCalloutsUnknownMap`) : la reconnaissance de carte n'a
// alors aucun appui et lui attribuerait un module au hasard. Elles sont ECARTEES, et le
// rapport le dit — l'absence est un fait de construction, pas une lacune de l'instrument.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/himap"
)

// cheminDumpPOC : le decoupage du POC pour Ridgeline. Le catalogue de callouts ne conserve
// PAS le pave brut de cette carte (son decoupage vient du POC, pas de la chaine universelle,
// cf. `MapCalloutsCatalog.Brut`) — sans ce dump, Ridgeline se reconnait sur ses contours
// DECOUPES, plus petits, donc avec un score plus severe. Fichier versionne.
var cheminDumpPOC = filepath.Join(".ai", "V7.5", "dumps", "callout_zones_ridgeline_clipped.json")

// moduleRidgeline : la carte dont les paves bruts vivent dans le dump du POC.
const moduleRidgeline = "ridgeline"

// sources rassemble les catalogues lus une fois pour toutes.
type sources struct {
	res      *title.PathResolver
	slug     string
	callouts *replay.MapCalloutsCatalog
	ancres   map[string][][3]float64
	brutPOC  map[int][][2]float64
}

// chargeCartes construit la table des cartes mesurables : un fond publie, des paves de
// designer, des ancres d'objectifs, et un `.module` installe. Rend aussi, en clair, les
// cartes ECARTEES et leur motif — une carte absente du tableau sans raison ecrite serait
// indiscernable d'un trou de l'instrument.
func chargeCartes(racine, slug string) ([]carte, []string, error) {
	res := title.NewPathResolver(racine)
	callouts, err := replay.LoadMapCallouts(res.MapCalloutsPath(slug))
	if err != nil {
		return nil, nil, err
	}
	objectifs, err := replay.LoadMapObjectives(res.MapObjectivesPath(slug))
	if err != nil {
		return nil, nil, err
	}
	bornes, err := filmdec.LoadMapQuantCatalog(res.MapQuantBoundsPath(slug))
	if err != nil {
		return nil, nil, err
	}
	s := &sources{res: res, slug: slug, callouts: callouts,
		ancres:  ancresParModule(objectifs, bornes),
		brutPOC: chargeBrutPOC(filepath.Join(racine, cheminDumpPOC))}
	modules := make([]string, 0, len(callouts.Maps))
	for m := range callouts.Maps {
		modules = append(modules, m)
	}
	sort.Strings(modules) // l'ordre d'une map Go n'est pas un ordre : sans ca le rapport bouge
	var cartes []carte
	var ecartees []string
	for _, m := range modules {
		c, motif := s.carteDe(m)
		if motif != "" {
			ecartees = append(ecartees, fmt.Sprintf("%s (%s)", m, motif))
			continue
		}
		cartes = append(cartes, c)
	}
	return cartes, ecartees, nil
}

// carteDe assemble une carte, ou dit POURQUOI elle n'est pas mesurable.
func (s *sources) carteDe(module string) (carte, string) {
	meta := s.res.MapBackgroundMetaPath(s.slug, module)
	if _, err := os.Stat(meta); err != nil {
		return carte{}, "pas de fond publie"
	}
	bg, err := replay.LoadMapBackground(meta)
	if err != nil {
		return carte{}, fmt.Sprintf("calage illisible : %v", err)
	}
	chemin, ok := himap.ChercheModuleInstalle(module)
	if !ok {
		return carte{}, "module non installe localement"
	}
	c := carte{module: module, noms: bg.MapNames, chemin: chemin,
		calage: bg.Calibration, niveauJeu: bg.Stats.PlayLevelZ, ancres: s.ancres[module]}
	for _, z := range s.callouts.Maps[module].Zones {
		c.prismes = append(c.prismes, prisme{nom: nomZone(z), poly: s.paveBrut(module, z),
			zBas: z.ZBottom, zHaut: z.ZTop, grande: z.Big})
		if z.Big {
			c.grandes++
		}
	}
	if len(c.prismes) == 0 {
		return carte{}, "aucune zone nommee"
	}
	if len(c.ancres) == 0 {
		return carte{}, "aucune ancre d'objectif"
	}
	return c, ""
}

// paveBrut rend le PAVE DU DESIGNER d'une zone : le dump du POC pour Ridgeline, le brut
// conserve par le catalogue pour les cartes passees par la chaine universelle, et a defaut le
// polygone servi (une carte encore brute est deja son propre pave).
func (s *sources) paveBrut(module string, z replay.CalloutZone) [][2]float64 {
	if module == moduleRidgeline {
		if p, ok := s.brutPOC[z.VolumeIndex]; ok && len(p) >= 3 {
			return p
		}
	}
	for _, b := range s.callouts.Brut[module] {
		if b.VolumeIndex == z.VolumeIndex {
			return b.Polygon
		}
	}
	return z.Polygon
}

// chargeBrutPOC lit les paves bruts du dump de Ridgeline. Un dump absent ou illisible n'est
// PAS fatal : la reconnaissance retombe sur les contours decoupes, plus severes. Rendre une
// table vide est donc une degradation, et elle est journalisee par l'appelant du rapport.
func chargeBrutPOC(chemin string) map[int][][2]float64 {
	out := map[int][][2]float64{}
	blob, err := os.ReadFile(chemin) //nolint:gosec // fichier versionne du depot, lecture seule
	if err != nil {
		return out
	}
	var d struct {
		Zones []struct {
			VolumeIndex int `json:"volumeIndex"`
			Brut        struct {
				Polygone [][2]float64 `json:"polygone"`
			} `json:"brut"`
		} `json:"zones"`
	}
	if json.Unmarshal(blob, &d) != nil {
		return out
	}
	for _, z := range d.Zones {
		if len(z.Brut.Polygone) >= 3 {
			out[z.VolumeIndex] = z.Brut.Polygone
		}
	}
	return out
}

// ancresParModule regroupe les ancres d'objectifs par dossier de module installe.
//
// MEME REGROUPEMENT QUE `cmd/mapfond-build` (ciblesNatives) et pour la meme raison : le
// catalogue porte plusieurs entrees pour un meme dossier (une carte classee et sa version non
// classee), et la cuisson prend l'UNION de leurs ancres, dedupliquee par position — un doublon
// pese sur la mediane qui donne le sol joue sans etre une mesure de plus.
func ancresParModule(cat *replay.MapObjectivesCatalog, bornes *filmdec.MapQuantCatalog) map[string][][3]float64 {
	ids := make([]string, 0, len(cat.Maps))
	for id := range cat.Maps {
		ids = append(ids, id)
	}
	sort.Strings(ids) // l'ordre d'une map Go n'est pas un ordre
	out := map[string][][3]float64{}
	connues := map[string]map[[3]float64]bool{}
	for _, id := range ids {
		e := cat.Maps[id]
		cle := moduleDeLEntree(e, bornes)
		if cle == "" {
			continue
		}
		if connues[cle] == nil {
			connues[cle] = map[[3]float64]bool{}
		}
		for _, o := range e.Objectives {
			a := [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z}
			if !connues[cle][a] {
				connues[cle][a] = true
				out[cle] = append(out[cle], a)
			}
		}
	}
	return out
}

// moduleDeLEntree rend le dossier installe d'une entree du catalogue d'objectifs, ou "".
//
// DEUX LECTURES, ET LA SECONDE N'EST PAS UN CONFORT. Le catalogue re-tire du reseau le
// 2026-08-25 (commit `d50f3b728`) porte `module: "map"` et `mvar_file: "map.mvar"` sur 63 de
// ses 73 entrees : le nom du module ne dit plus rien pour elles, et l'appariement par jetons
// (`himap.ChercheModuleInstalle`) les declare toutes non installees. Le lien qui SURVIT est
// celui du catalogue de bornes, `nom affiche -> module` — la meme table, declaree une seule
// fois, que le service du fond de carte emploie en production
// (`internal/service/replay_map_background.go`).
func moduleDeLEntree(e replay.MapObjectivesEntry, bornes *filmdec.MapQuantCatalog) string {
	if chemin, ok := himap.ChercheModuleInstalle(e.Module); ok {
		return filepath.Base(filepath.Dir(chemin))
	}
	entree, err := bornes.Lookup(e.PublicName)
	if err != nil || entree.Module == "" {
		return ""
	}
	if chemin, ok := himap.ChercheModuleInstalle(entree.Module); ok {
		return filepath.Base(filepath.Dir(chemin))
	}
	return ""
}

// nomZone rend le nom JOUEUR d'une zone : le libelle francais quand il existe, l'anglais
// sinon, et en dernier recours le nom de CONCEPTION. Jamais vide — une zone sans nom dans le
// rapport serait indiscernable d'une absence de zone.
func nomZone(z replay.CalloutZone) string {
	for _, n := range []string{z.FR, z.EN, z.Name} {
		if s := strings.TrimSpace(n); s != "" {
			return s
		}
	}
	return fmt.Sprintf("volume %d", z.VolumeIndex)
}

// nomsLisibles rend les noms affiches d'une carte, en une chaine courte pour le rapport.
func (c *carte) nomsLisibles() string {
	vus := map[string]bool{}
	var out []string
	for _, n := range c.noms {
		n = strings.TrimSpace(n)
		if n == "" || vus[strings.ToLower(n)] || strings.EqualFold(n, c.module) {
			continue
		}
		vus[strings.ToLower(n)] = true
		out = append(out, n)
		if len(out) == 2 {
			break
		}
	}
	return strings.Join(out, ", ")
}
