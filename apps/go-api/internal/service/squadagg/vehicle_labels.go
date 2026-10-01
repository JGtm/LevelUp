package squadagg

// vehicle_labels.go — LE NOM DES FAMILLES DE VEHICULE QUE LE TITRE QUALIFIE (ressource véhicules de
// l'Emprise, plan `.ai/PLAN_EMPRISE_VEHICULES_2026-09-28.md`, lot L7.3).
//
// Un nom de famille de véhicule est un NOM PROPRE du jeu (Warthog, Banshee) : il ne se traduit pas
// et la clé publiée par le calque EST ce nom ; le client l'affiche depuis la clé. Seules les
// familles que le manifeste du titre qualifie (`replay_labels.toml`, `[[vehicle_families]]` : une
// tourelle fixe, dont le nom est une DESCRIPTION) portent un libellé bilingue — le même que celui du
// rejeu 2D, lu du même catalogue. Jamais de nom en dur côté Go.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/games/halo_infinite/replaylabels"
)

// VehicleFamilyLabels — famille de véhicule qualifiée -> libellé dans la langue de la requête. Map
// vide (jamais nil) quand le catalogue manque : les familles gardent alors leur clé à l'écran.
func VehicleFamilyLabels(ctx context.Context, repoRoot, titleSlug, locale string) map[string]string {
	out := map[string]string{}
	if repoRoot == "" || titleSlug == "" {
		return out
	}
	cat, err := replaylabels.Catalogue(repoRoot, titleSlug)
	if err != nil {
		slog.WarnContext(ctx, "escouade: catalogue des familles de vehicule illisible — familles non nommees",
			"err", err, "titleSlug", titleSlug)
		return out
	}
	frPreferred := locale != "en"
	for family, info := range cat.VehicleFamilies {
		label := info.En
		if frPreferred && info.Fr != "" {
			label = info.Fr
		}
		if label != "" {
			out[family] = label
		}
	}
	return out
}
