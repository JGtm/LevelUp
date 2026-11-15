package squadagg

// vehicle_labels.go — LE NOM DES FAMILLES DE VEHICULE QUE LE TITRE QUALIFIE (ressource véhicules de
// l'Emprise).
//
// Un nom de famille de véhicule est un NOM PROPRE du jeu (Warthog, Banshee) : il ne se traduit pas
// et la clé publiée par le calque EST ce nom ; le client l'affiche depuis la clé. Seules les
// familles que le manifeste du titre qualifie (`replay_labels.toml`, `[[vehicle_families]]` : une
// tourelle fixe, dont le nom est une DESCRIPTION) portent un libellé bilingue — le même que celui du
// rejeu 2D, lu du même catalogue. Jamais de nom en dur côté Go.

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/replaytext"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// labelCatalogue rend le catalogue de libellés du rejeu du titre, ou faux quand il manque : source
// absente, titre vide, titre sans rejeu (`games.ErrCapabilityNotSupported`, dégradation voulue) ou
// catalogue illisible (dit sous `message`). Les appelants servent alors sans libellé du titre.
func labelCatalogue(ctx context.Context, labels port.ReplayLabelSource, titleSlug, message string) (replay.LabelCatalog, bool) {
	if labels == nil || titleSlug == "" {
		return replay.LabelCatalog{}, false
	}
	cat, err := labels.LabelCatalogue(titleSlug)
	switch {
	case err == nil:
		return cat, true
	case errors.Is(err, games.ErrCapabilityNotSupported):
		slog.DebugContext(ctx, "escouade: titre sans catalogue de libellés du rejeu",
			"titleSlug", titleSlug, "err", err)
	default:
		slog.WarnContext(ctx, message, "err", err, "titleSlug", titleSlug)
	}
	return replay.LabelCatalog{}, false
}

// VehicleFamilyLabels — famille de véhicule qualifiée -> libellé dans la langue de la requête. Map
// vide (jamais nil) quand le catalogue manque : les familles gardent alors leur clé à l'écran.
func VehicleFamilyLabels(ctx context.Context, labels port.ReplayLabelSource, titleSlug, locale string) map[string]string {
	out := map[string]string{}
	cat, ok := labelCatalogue(ctx, labels, titleSlug,
		"escouade: catalogue des familles de vehicule illisible — familles non nommees")
	if !ok {
		return out
	}
	labelLocale := i18n.Parse(locale)
	for family, info := range cat.VehicleFamilies {
		label := replaytext.TexteDepuisFilm(info.Label).Pick(labelLocale)
		if label != "" {
			out[family] = label
		}
	}
	return out
}
