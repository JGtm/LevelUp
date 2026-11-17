package main

// cmd_spawn_model_zones.go — LES ZONES NOMMEES DES CARTES MESUREES.
//
// Les zones sont celles que lit l'onglet Tactique (`service.NewTacticalCalloutsStore`, meme
// cascade module puis asset UGC) ; le nom d'un point est celui que `tactical.NommerZone` lui
// donnerait. La base partagee est lue par la poignee deja ouverte en lecture : aucune connexion
// de plus.

import (
	"context"
	"database/sql"

	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/service"
)

// lecteurPartage adapte une poignee de lecture au contrat SharedReader du lecteur de cartes.
type lecteurPartage struct{ db *sql.DB }

func (l lecteurPartage) Get(context.Context) (*sql.DB, func(), error) {
	return l.db, func() {}, nil
}

// zonesDesCartes rend les zones nommees de chaque carte demandee (liste vide si la carte n'est
// pas au catalogue).
func zonesDesCartes(ctx context.Context, repoRoot, slug string, shared *sql.DB, cartes []string) map[string][]tactical.ZoneNommee {
	store := service.NewTacticalCalloutsStore(repoRoot, slug,
		duckdb.NewReplayMapRepo(lecteurPartage{db: shared}, nil))
	out := make(map[string][]tactical.ZoneNommee, len(cartes))
	for _, id := range cartes {
		out[id] = tactical.ZonesDepuisLePort(store.ZonesDeLaCarte(ctx, id))
	}
	return out
}
