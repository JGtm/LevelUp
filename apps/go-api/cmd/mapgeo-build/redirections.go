package main

// redirections.go — QUAND LA GEOMETRIE D'UNE CARTE N'EST PAS DANS SON MODULE.
//
// UNE carte de l'installation est dans ce cas : `sgh_interlock` (Live Fire) ne porte aucun
// tag sbsp, sa geometrie vit dans `pc/globals/common-rtx-new.module`. Le producteur de
// fonds le sait depuis le 2026-08-27 et le declare en DONNEE, dans `map_fond_reglages.json`
// (champ `moduleGeometrie`). On lit la MEME donnee plutot que de redecouvrir le cas :
// deux producteurs qui divergeraient sur l'endroit ou est la geometrie d'une carte
// produiraient deux cartes differentes de la meme carte.
//
// SEUL CE CHAMP EST LU. Les autres reglages du fichier (echelle, ecretage, comblement)
// sont des choix d'IMAGE ; ils n'ont pas de sens pour une analyse geometrique, qui ne
// dessine pas la carte mais la mesure.

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/himap"
)

// fichierReglages est la vue MINIMALE du fichier de reglages des fonds.
type fichierReglages struct {
	Cartes map[string]struct {
		ModuleGeometrie string `json:"moduleGeometrie"`
	} `json:"cartes"`
}

// chargeRedirections rend, par module installe, le chemin absolu du module qui porte sa
// geometrie. Vide quand le fichier manque : aucune carte n'est alors redirigee, ce qui est
// le cas de toutes sauf une, et le journal le dit.
func chargeRedirections(res *title.PathResolver, titleSlug string) map[string]string {
	chemin := res.MapFondReglagesPath(titleSlug)
	blob, err := os.ReadFile(chemin)
	if err != nil {
		slog.Warn("mapgeo: reglages de fond illisibles — aucune redirection de geometrie",
			"err", err, "path", chemin)
		return nil
	}
	var f fichierReglages
	if err := json.Unmarshal(blob, &f); err != nil {
		slog.Warn("mapgeo: reglages de fond invalides — aucune redirection de geometrie",
			"err", err, "path", chemin)
		return nil
	}
	racine, err := himap.DeployRoot()
	if err != nil {
		slog.Warn("mapgeo: installation introuvable — aucune redirection de geometrie", "err", err)
		return nil
	}
	out := map[string]string{}
	for cle, c := range f.Cartes {
		if c.ModuleGeometrie == "" {
			continue
		}
		out[cle] = filepath.Join(racine, filepath.FromSlash(c.ModuleGeometrie))
		slog.Info("mapgeo: geometrie prise dans un AUTRE module", "carte", cle, "module", c.ModuleGeometrie)
	}
	return out
}
