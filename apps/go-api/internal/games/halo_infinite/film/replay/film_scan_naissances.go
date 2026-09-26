package replay

// film_scan_naissances.go — LE BALAYAGE DES DOTATIONS DE NAISSANCE (lot M3.2 de la campagne
// « retours rejeu », 2026-09-23).
//
// Il vit hors de `film_scan.go` pour la raison de `film_scan_mouvement.go` : le fichier frôlait
// le seuil de 500 lignes (règle 5 de CLAUDE.md). La méthode est appelée par `balayerPositions`,
// juste après les images-clés.

import (
	"log/slog"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// balayerNaissances lit la DOTATION DE NAISSANCE de chaque corps : les emplacements d'arme de son
// record NEW de création (cf. `grammar/birth_loadouts.go`).
//
// ELLE VIENT APRÈS LES CRÉATIONS, QUI L'ANCRENT, ET AVANT LES PRISES D'ARME, dont elle qualifie la
// première émission de chaque vie (`spawnSetFrom`). Une dotation dont le record ne se ferme pas
// n'est PAS publiée — aucune lecture de repli — et le refus est compté par cause.
//
// ABSENCE NON FATALE : le rejeu sort sans armes de naissance, jamais avec des armes devinées.
func (s *filmScan) balayerNaissances() {
	births, st, err := grammar.ScanBirthLoadouts(s.fc, s.in.BipedCreations)
	if err != nil {
		slog.Warn("dotations de naissance illisibles — rejeu sans armes de naissance",
			"err", err, "match_id", s.matchID)
		births, st = nil, types.BirthLoadoutStats{Creations: len(s.in.BipedCreations)}
	} else {
		slog.Info("naissance : dotations lues",
			"creations", st.Creations, "lues", st.Read, "desynchronisees", st.Desync,
			"debordantes", st.Overflow, "nonConfirmees", st.Unconfirmed,
			"sansEmplacement", st.NoWeaponComponent, "fermeesParDelta", st.ClosedByDelta,
			"fermeesParNouveauLie", st.ClosedByBoundNew,
			"fermeesParNouveauAnticipe", st.ClosedByAnticipatedNew, "match_id", s.matchID)
	}
	s.in.BirthLoadouts, s.in.BirthLoadoutStats = births, st
	s.opt.observe("birthLoadouts", s.in.BirthLoadouts)
	s.opt.observe("birthLoadouts.stats", s.in.BirthLoadoutStats)
}
