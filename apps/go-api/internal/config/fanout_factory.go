// Package config — fanout_factory.go : FanoutFactory implémente port.FanoutPlayerFactory.
package config

import (
	"context"

	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
)

// FanoutFactory crée un port.FanoutRepository pour un joueur donné
// en s'appuyant sur ResolvePlayer.
type FanoutFactory struct {
	cfg *AppConfig
}

// NewFanoutFactory crée une FanoutFactory.
func NewFanoutFactory(cfg *AppConfig) *FanoutFactory {
	return &FanoutFactory{cfg: cfg}
}

// OpenForPlayer résout le gamertag en PlayerDB et retourne un FanoutRepository.
func (f *FanoutFactory) OpenForPlayer(
	ctx context.Context,
	gamertag, titleSlug string,
) (port.FanoutRepository, error) {
	pdb, err := ResolvePlayer(ctx, f.cfg, gamertag, titleSlug)
	if err != nil {
		return nil, err
	}
	return duckdb.NewFanoutRepo(pdb), nil
}
