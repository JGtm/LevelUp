package main

// cibles.go — DE UN NOM DE CARTE A CE QU'IL FAUT POUR LA CUIRE.
//
// La chaine de resolution est celle, deja eprouvee, de `cmd/mapfond-build` : le catalogue
// d'objectifs est keye par map_id (asset UGC), chaque entree declare un module de
// CATALOGUE (`aquarius_-_ranked_map`), et `himap.ChercheModuleInstalle` fait le pont vers
// le dossier de l'INSTALLATION (`ctf_aquarius`). Plusieurs entrees designent le meme
// dossier — une carte jouee en deux playlists est deux assets : on les REGROUPE, et on
// prend l'UNION de leurs ancres, comme le fait le producteur de fonds.
//
// LE NOM DE CARTE DU CORPUS (« recharge ») se relie au module par le catalogue des bornes
// de quantification, exactement comme `cmd/mappower-build` resout le module qu'il publie.
// Deux tables, deux roles : les bornes nomment, les objectifs ancrent.

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis/powerpos/geo"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/himap"
)

// Cible est une carte a cuire : son module installe, ses ancres, ses assets.
type Cible struct {
	// Carte est le nom normalise du corpus (`recharge`), la cle de sortie.
	Carte string
	// Module est le dossier du module installe (`sgh_blueprint`).
	Module string
	// CheminModule est le `.module` d'ou tirer la geometrie (variante `pc`), apres
	// redirection eventuelle (`moduleGeometrie` — le cas de Live Fire).
	CheminModule string
	// CheminModuleCarte est le `.module` PROPRE de la carte, sans redirection : c'est lui
	// qui porte la coquille de mort (`sddt`, variante `any`), meme quand la geometrie vit
	// dans un module global.
	CheminModuleCarte string
	// MapIDs sont les assets UGC qui partagent ce module, tries.
	MapIDs []string
	// Ancres sont les positions monde des objectifs, union de tous les assets.
	Ancres [][3]float64
	// Socles sont les emplacements d'armes (famille `power` ou `rack`), dedoublonnes — les
	// ressources dont on mesure la distance de deplacement. Les objectifs (Ancres) s'y
	// ajoutent a la mesure, sous la nature `objectif`.
	Socles []geo.Ressource
}

// ResoutCibles rend les cibles demandees, dans l'ordre des noms fournis.
func ResoutCibles(res *title.PathResolver, titleSlug string, cartes []string) ([]*Cible, error) {
	quant, err := decfilm.LoadMapQuantCatalog(res.MapQuantBoundsPath(titleSlug))
	if err != nil {
		return nil, fmt.Errorf("catalogue de bornes illisible : %w", err)
	}
	objectifs, err := replay.LoadMapObjectives(res.MapObjectivesPath(titleSlug))
	if err != nil {
		return nil, fmt.Errorf("catalogue d'objectifs illisible : %w", err)
	}
	pads, err := replay.LoadMapWeaponPads(res.MapWeaponPadsPath(titleSlug))
	if err != nil {
		return nil, fmt.Errorf("catalogue des socles illisible : %w", err)
	}
	parModule := regroupeParModule(objectifs)
	redirections := chargeRedirections(res, titleSlug)

	var out []*Cible
	for _, nom := range cartes {
		c, err := resoutUne(quant, parModule, pads, redirections, nom)
		if err != nil {
			slog.Warn("mapgeo: carte ecartee", "err", err, "carte", nom)
			continue
		}
		out = append(out, c)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("aucune des %d cartes demandees n'est resolvable", len(cartes))
	}
	return out, nil
}

// groupe rassemble les assets qui partagent un dossier de module installe.
type groupe struct {
	chemin string
	mapIDs []string
	ancres [][3]float64
	vues   map[[3]float64]bool
}

// regroupeParModule projette le catalogue d'objectifs sur les dossiers installes.
func regroupeParModule(cat *replay.MapObjectivesCatalog) map[string]*groupe {
	ids := make([]string, 0, len(cat.Maps))
	for id := range cat.Maps {
		ids = append(ids, id)
	}
	sort.Strings(ids) // l'ordre d'une map Go n'est pas un ordre : la sortie doit etre stable
	par := map[string]*groupe{}
	for _, id := range ids {
		e := cat.Maps[id]
		chemin, ok := himap.ChercheModuleInstalle(e.Module)
		if !ok {
			continue
		}
		cle := filepath.Base(filepath.Dir(chemin))
		g := par[cle]
		if g == nil {
			g = &groupe{chemin: chemin, vues: map[[3]float64]bool{}}
			par[cle] = g
		}
		g.mapIDs = append(g.mapIDs, id)
		for _, o := range e.Objectives {
			p := [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z}
			if g.vues[p] {
				continue
			}
			g.vues[p] = true
			g.ancres = append(g.ancres, p)
		}
	}
	return par
}

// resoutUne assemble la cible d'une carte.
func resoutUne(quant *decfilm.MapQuantCatalog, parModule map[string]*groupe,
	pads *replay.MapWeaponPadsCatalog, redirections map[string]string, nom string) (*Cible, error) {
	entree, err := quant.Lookup(nom)
	if err != nil {
		return nil, fmt.Errorf("carte absente du catalogue de bornes : %w", err)
	}
	module := entree.Module
	g := parModule[module]
	if g == nil {
		return nil, fmt.Errorf("module %q sans groupe d'objectifs (carte non installee ?)", module)
	}
	if len(g.ancres) == 0 {
		return nil, fmt.Errorf("module %q sans aucune ancre d'objectif", module)
	}
	chemin := g.chemin
	if r := redirections[module]; r != "" {
		chemin = r
	}
	c := &Cible{
		Carte: nom, Module: module, CheminModule: chemin, CheminModuleCarte: g.chemin,
		MapIDs: append([]string(nil), g.mapIDs...), Ancres: g.ancres,
	}
	sort.Strings(c.MapIDs)
	c.Socles = ressourcesDe(pads, c.MapIDs)
	return c, nil
}

// ressourcesDe rassemble les socles d'armes de tous les assets du module, dedoublonnes a
// 0,5 m pres — deux variantes de playlist declarent les MEMES socles, aux arrondis pres.
func ressourcesDe(pads *replay.MapWeaponPadsCatalog, mapIDs []string) []geo.Ressource {
	var out []geo.Ressource
	for _, id := range mapIDs {
		e, err := pads.Lookup(id)
		if err != nil {
			continue
		}
		for _, p := range e.Pads {
			r := geo.Ressource{X: p.Pos.X, Y: p.Pos.Y, Z: p.Pos.Z, Nature: nature(p.Family)}
			if dejaVue(out, r) {
				continue
			}
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].X != out[j].X {
			return out[i].X < out[j].X
		}
		return out[i].Y < out[j].Y
	})
	return out
}

// nature traduit la famille du catalogue. Le brut reste lisible : un `rack` est une arme
// de ratelier, il compte moins qu'une arme de pouvoir mais il compte.
func nature(famille string) string {
	switch strings.ToLower(famille) {
	case "power":
		return geo.NatureArmeForte
	case "powerup":
		return geo.NaturePowerup
	default:
		return geo.NatureArme
	}
}

// dejaVue dit si une ressource equivalente (< 0,5 m) est deja dans la liste.
func dejaVue(liste []geo.Ressource, r geo.Ressource) bool {
	for _, v := range liste {
		dx, dy, dz := v.X-r.X, v.Y-r.Y, v.Z-r.Z
		if dx*dx+dy*dy+dz*dz < 0.25 {
			return true
		}
	}
	return false
}
