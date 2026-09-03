package main

// forge_fetch.go — D'OÙ VIENNENT LES VARIANTES DE CARTE FORGE.
//
// LA LISTE EST VERSIONNÉE, PAS DEVINÉE : `.ai/V7.5/cartes/inventaire_rotation_ugc_*.json`
// recense l'univers des cartes atteignables par la rotation matchmaking (104 cartes au
// 2026-08-27, dont 83 Forge), avec pour chacune son map_id, sa version, ses fichiers et son
// PRÉFIXE DE STOCKAGE BLOB. Cet inventaire a coûté un balayage authentifié du Discovery
// UGC ; il est au dépôt pour n'être payé qu'une fois.
//
// LE TÉLÉCHARGEMENT, LUI, EST ANONYME. Le stockage blob répond sans jeton (seul le
// Discovery en exige un) : `mapcatalog.Client.FetchFileAt` suffit, avec des jetons nil. Un
// `.mvar` pèse ~100 Ko et le cache est HORS DÉPÔT (.gitignore) — c'est le catalogue produit
// qui est versionné, jamais sa matière première.
//
// LE REJEU N'APPELLE JAMAIS CE CHEMIN : il lit `map_callouts.json`. La production d'une
// donnée de référence a le droit d'aller sur le réseau, sa lecture non (même règle que
// mapobj-build / map_objectives.json).

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"levelup/go-api/internal/mapcatalog"
)

// fichierVarianteForge : le fichier de variante qui porte les objets POSÉS par le créateur.
// L'autre entrée de la liste `mvar` est le canevas, qui ne pose aucune zone nommée (mesuré
// sur les 8 canevas installés).
const fichierVarianteForge = "map.mvar"

// familleForge : la valeur du champ `famille` de l'inventaire pour une carte communautaire.
const familleForge = "forge"

// inventaireUGC est la forme du fichier d'inventaire (champs utilisés uniquement).
type inventaireUGC struct {
	SchemaVersion int         `json:"schema_version"`
	GenereLe      string      `json:"genere_le"`
	Cartes        []carteUGC  `json:"cartes"`
	Total         json.Number `json:"total"`
}

// carteUGC est une carte de la rotation.
type carteUGC struct {
	MapID      string   `json:"map_id"`
	VersionID  string   `json:"version_id"`
	Nom        string   `json:"nom"`
	Famille    string   `json:"famille"`
	Mvar       []string `json:"mvar"`
	BlobPrefix string   `json:"blob_prefix"`
}

// chargeInventaire lit l'inventaire versionné et rend les cartes FORGE qui exposent une
// variante `map.mvar`.
func chargeInventaire(path string) ([]carteUGC, error) {
	blob, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("inventaire UGC illisible (%s) : %w", path, err)
	}
	var inv inventaireUGC
	if err := json.Unmarshal(blob, &inv); err != nil {
		return nil, fmt.Errorf("inventaire UGC invalide (%s) : %w", path, err)
	}
	var out []carteUGC
	for _, c := range inv.Cartes {
		if c.Famille != familleForge || c.MapID == "" {
			continue
		}
		for _, f := range c.Mvar {
			if f == fichierVarianteForge {
				out = append(out, c)
				break
			}
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("inventaire UGC (%s) : aucune carte Forge exposant %s", path, fichierVarianteForge)
	}
	return out, nil
}

// telechargeVariantes complète le cache local : une requête par carte MANQUANTE, jamais de
// re-téléchargement de ce qu'on a déjà.
//
// Un échec est COMPTÉ et journalisé, jamais fatal : une carte retirée du stockage UGC ne
// doit pas empêcher les 82 autres d'entrer au catalogue.
func telechargeVariantes(ctx context.Context, cibles []carteUGC, cache string, delai time.Duration) (int, int) {
	if err := os.MkdirAll(cache, 0o755); err != nil {
		slog.Error("cache des variantes", "err", err, "cache", cache)
		return 0, len(cibles)
	}
	client := mapcatalog.NewClient(nil)
	pris, echecs := 0, 0
	for _, c := range cibles {
		dest := filepath.Join(cache, c.MapID+".mvar")
		if _, err := os.Stat(dest); err == nil {
			continue
		}
		blob, err := client.FetchFileAt(ctx, c.BlobPrefix, fichierVarianteForge)
		if err != nil {
			slog.Warn("variante non téléchargée", "nom", c.Nom, "map_id", c.MapID, "err", err)
			echecs++
			continue
		}
		if err := os.WriteFile(dest, blob, 0o644); err != nil {
			slog.Warn("variante non écrite", "nom", c.Nom, "dest", dest, "err", err)
			echecs++
			continue
		}
		pris++
		if delai > 0 {
			time.Sleep(delai)
		}
	}
	return pris, echecs
}
