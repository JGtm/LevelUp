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
	"levelup/go-api/internal/domain"
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
		out[id] = zonesPuresDuCatalogue(store.ZonesDeLaCarte(ctx, id))
	}
	return out
}

// zonesPuresDuCatalogue projette les zones du port vers le type du paquet d'algo (seconde copie
// de cette projection, apres celle du service Tactique ; une troisieme impose de la centraliser).
func zonesPuresDuCatalogue(zones []domain.ZoneNommee) []tactical.ZoneNommee {
	out := make([]tactical.ZoneNommee, 0, len(zones))
	for _, z := range zones {
		out = append(out, tactical.ZoneNommee{
			Nom: z.Nom, X: z.X, Y: z.Y,
			Polygone: z.Polygone, Parties: z.Parties, Trous: z.Trous,
			ZBas: z.ZBas, ZHaut: z.ZHaut, VolumeIndex: z.VolumeIndex,
		})
	}
	return out
}

// fonctionZone rend la fonction « indice de zone d'un point » des echantillons : l'indice de
// volume de la zone qui nomme le point, -1 si la carte n'a pas de zone ou si aucune ne le nomme.
func fonctionZone(zones map[string][]tactical.ZoneNommee) func(string, tactical.Point3) int {
	return func(carte string, p tactical.Point3) int {
		zs := zones[carte]
		if len(zs) == 0 {
			return -1
		}
		r, ok := tactical.NommerZone(p.X, p.Y, []float64{p.Z}, zs)
		if !ok {
			return -1
		}
		return r.Zone.VolumeIndex
	}
}
