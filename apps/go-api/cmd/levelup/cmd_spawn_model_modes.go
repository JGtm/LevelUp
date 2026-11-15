package main

// cmd_spawn_model_modes.go — LES FAMILLES DE MODES DE LA MESURE DE REAPPARITION, et le registre
// des matchs qui les porte.
//
// La table qui range une variante de jeu dans une famille est une donnee du titre
// (config/titles/{slug}/mappings/spawn_families.toml, `mappings.SpawnFamilySet`) : cette commande
// ne connait aucun nom de mode. Un match sans famille (BTB, rotation Husky Raid, variante qui ne
// ressemble a rien) n'est pas mesure.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/mappings"
)

// fichierFamilles est le nom du fichier de familles dans le dossier des mappings d'un titre.
const fichierFamilles = "spawn_families.toml"

// famillesDuTitre charge la table des familles de modes du titre. Un titre qui declare la
// prevision sans declarer de famille est une erreur de configuration : la commande s'arrete en le
// disant, elle ne mesure pas un jeu de modes qu'elle ne sait pas nommer.
func famillesDuTitre(cfg *config.AppConfig, slug string) (*mappings.SpawnFamilySet, error) {
	path := filepath.Join(cfg.PathResolver().TitleMappingsDir(slug), fichierFamilles)
	set, err := mappings.LoadSpawnFamiliesFromFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("le titre %s n'a pas de familles de modes (%s absent) : aucune mesure possible", slug, fichierFamilles)
	}
	if err != nil {
		return nil, fmt.Errorf("familles de modes de %s : %w", slug, err)
	}
	return set, nil
}

// infoMatch est ce que le registre sait d'un match.
type infoMatch struct {
	matchID, carteID, carteNom, variante, categorie, famille string
}

// registreDesMatchs lit le registre des matchs JcJ, indexe par la forme COURTE du match_id (la
// cle des artefacts).
func registreDesMatchs(ctx context.Context, db *sql.DB, familles *mappings.SpawnFamilySet) (map[string]infoMatch, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT match_id, COALESCE(map_id, ''), COALESCE(map_name, ''),
		       COALESCE(game_variant_name, ''), COALESCE(mode_category, '')
		FROM match_registry
		WHERE COALESCE(is_firefight, FALSE) = FALSE`)
	if err != nil {
		return nil, fmt.Errorf("registre des matchs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]infoMatch{}
	for rows.Next() {
		var m infoMatch
		if err := rows.Scan(&m.matchID, &m.carteID, &m.carteNom, &m.variante, &m.categorie); err != nil {
			return nil, fmt.Errorf("registre des matchs (scan): %w", err)
		}
		m.famille = familles.FamilyOf(m.variante, m.categorie)
		out[titlePkg.FilmShortMatchID(m.matchID)] = m
	}
	return out, rows.Err()
}

// variantesDuParc compte, par (famille, variante, categorie), les matchs du registre dont
// l'artefact existe : de quoi verifier que les familles disent ce qu'on croit.
func variantesDuParc(registre map[string]infoMatch, shorts []string) []string {
	type cle struct{ famille, variante, categorie string }
	n := map[cle]int{}
	for _, s := range shorts {
		if m, ok := registre[s]; ok {
			n[cle{m.famille, m.variante, m.categorie}]++
		}
	}
	lignes := make([]string, 0, len(n))
	for k, v := range n {
		f := k.famille
		if f == "" {
			f = "(hors mesure)"
		}
		lignes = append(lignes, fmt.Sprintf("%-14s | %-40s | %-14s | %5d", f, k.variante, k.categorie, v))
	}
	sort.Strings(lignes)
	return lignes
}
