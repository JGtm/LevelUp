package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"

	"github.com/pelletier/go-toml/v2"

	"levelup/go-api/internal/i18n"
)

// assetLabels — noms de jeu officiels, par langue, versionnés dans
// config/titles/halo_5/mappings/asset_labels.toml. Clé = nom EN EXACT de l'API Metadata ;
// valeur = table par langue (`{ fr = "..." }`) : seule une langue CONNUE y figure, une langue
// absente se replie sur la donnée de l'API (jamais de traduction inventée, ADR 0040 D-4).
//
// PAS de champ Weapons : la section [weapons] a été RETIRÉE du TOML par V72-06 —
// les noms d'armes sont la source unique de weapon_names.toml, keyée par
// weapon_key (résolue via weapon_id → weapon_ids → weapon_key) et non par nom EN
// brut. Ne pas la réintroduire : le keying par nom EN est précisément la source de
// mismatch que V72-06 supprime.
type assetLabels struct {
	Medals            map[string]i18n.Text `toml:"medals"`
	Maps              map[string]i18n.Text `toml:"maps"`
	MedalDescriptions map[string]i18n.Text `toml:"medal_descriptions"`
	Commendations     map[string]i18n.Text `toml:"commendations"`
	// MapsByID — overrides de nom de map par asset_id (canvas Forge non nommés par
	// l'API officielle /maps : name_canonical vide). Distinct de Maps (keyé par nom
	// EN) car ces maps N'ONT PAS de nom EN sur lequel keyer. Appliqués en fin de run.
	MapsByID []mapIDEntry `toml:"maps_by_id"`
}

// mapIDEntry — une ligne de [[maps_by_id]] : `name = { en = "...", fr = "..." }`.
type mapIDEntry struct {
	ID   string    `toml:"id"`
	Name i18n.Text `toml:"name"`
}

// loadAssetLabels lit asset_labels.toml. Fichier ABSENT → catalogue vide (les noms retombent
// sur l'API, dégradation propre, journalisée). Fichier illisible ou table à langue inconnue →
// erreur : un seed qui continuerait sans les overrides écrirait des noms anglais en silence.
func loadAssetLabels(path string) (assetLabels, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		slog.Warn("h5-metadata-fetch: asset_labels.toml absent, aucun override de nom", "path", path)
		return assetLabels{}, nil
	}
	if err != nil {
		return assetLabels{}, fmt.Errorf("read %s: %w", path, err)
	}
	var labels assetLabels
	if err := toml.Unmarshal(raw, &labels); err != nil {
		return assetLabels{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := labels.checkLanguages(); err != nil {
		return assetLabels{}, fmt.Errorf("%s: %w", path, err)
	}
	return labels, nil
}

// checkLanguages refuse toute clé de langue absente de config/locales.toml (faute de frappe).
func (a assetLabels) checkLanguages() error {
	for section, table := range map[string]map[string]i18n.Text{
		"medals": a.Medals, "maps": a.Maps,
		"medal_descriptions": a.MedalDescriptions, "commendations": a.Commendations,
	} {
		for name, text := range table {
			if err := text.CheckKnown(); err != nil {
				return fmt.Errorf("[%s] %q : %w", section, name, err)
			}
		}
	}
	for _, m := range a.MapsByID {
		if err := m.Name.CheckKnown(); err != nil {
			return fmt.Errorf("[[maps_by_id]] %q : %w", m.ID, err)
		}
	}
	return nil
}
