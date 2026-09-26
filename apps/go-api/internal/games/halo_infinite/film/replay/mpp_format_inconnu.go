package replay

// mpp_format_inconnu.go — LE CABLEUR DU COMPTEUR « VERSION DE FORMAT INCONNUE » (lot 1.9.1 ter).
//
// # CE QU IL GARDE, ET POURQUOI C EST UN INCIDENT ET PAS UNE DETTE
//
// Le decoupage du bloc `object-multiplayer-properties` est keye par la VERSION DE FORMAT de
// `chunk_00` (`filmdec/film_format_version.go`). Un format que ce depot ne connait pas — 28 au
// prochain patch du jeu — n herite JAMAIS du decoupage du format voisin (D-4 d ADR 0034) : il
// bascule sur le repli `repli_largeurs_mpp_calibrees_sur_le_film`, qui MESURE le decoupage sur
// le film lui-meme.
//
// C EST LE BON DEFAUT — un patch du jeu ne doit pas eteindre le decodeur sur tout le parc neuf —
// MAIS UN REPLI SILENCIEUX SUR TOUT LE PARC EST UN INCIDENT INVISIBLE : la seule trace serait
// une derive de qualite sans cause, des semaines plus tard. D ou ce cableur, sur le patron exact
// de `publierBuildInconnu` (film_player_table.go) : `grammar` NOMME le compteur
// (`grammar.UnknownFormatExpvarPairs` -> `filmdec_unknown_format_<n>`), `replay` le CABLE.
//
// # UN COMPTEUR PAR DECLENCHEMENT, UN AVERTISSEMENT PAR FILM
//
// Le COMPTEUR s incremente a chacun des deux sites du registre — `equipment_placements.go` et
// `gwWidthsForFilm` (qui sert les armes au sol ET les vehicules) : c est un compte de
// declenchements de repli, la meme semantique que `fallback.Compteur.Declenche`.
// L AVERTISSEMENT, lui, est emis UNE SEULE FOIS PAR FILM, depuis `BuildFromFilm` — trois lignes
// identiques par cuisson noieraient le signal qu elles portent.
//
// # POURQUOI `slog.Warn` ET NON `slog.WarnContext`
//
// Il n y a AUCUN `context.Context` sur ce chemin : ni `BuildFromFilm(matchID, titleSlug, film,
// opt)`, ni son appelant `replaybuild.BuildBytes(matchID, mapNames, filmDir, facts)`, ni
// `Options` n en portent un. Les quarante et quelques appels `slog` du paquet sont tous des
// `slog.Warn` / `slog.Info` pour cette raison. Passer `context.Background()` n ajouterait aucune
// cle et romprait l uniformite du paquet ; plomber un ctx sur trois couches pour cette seule
// ligne depasse le perimetre du lot. Consigne au §4 du PLAN_DECODEUR_FILM.
//
// # LE COMPTAGE DU REGISTRE RESTE DIFFERE, ET CE N EST PAS LE MEME OBJET
//
// `Repli.CompteurBranche` reste FAUX pour `repli_largeurs_mpp_calibrees_sur_le_film` : le
// comptage des replis par `FilmContext` attend le pas 2 de M2, comme le registre l ecrit. Ce
// compteur-ci ne compte pas un repli ordinaire, il compte l evenement « le jeu a change de
// format » — il ne l attend pas.

import (
	"log/slog"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/observability"
)

// publierFormatSansProfil incremente `filmdec_unknown_format_<n>` pour un film dont la version
// de format est absente de la table de profil. Un format CONNU sans largeur relue (20, 21, 24,
// 25) n est PAS compte ici : c est l etat normal du parc ancien, pas un evenement.
func publierFormatSansProfil(format int) {
	for _, p := range grammar.UnknownFormatExpvarPairs(format) {
		observability.AddInt(p.Name, p.Value)
	}
}

// formatSansProfil dit si la version de format de ce film est absente de la table de profil, et
// rend la version lue. Un `chunk_00` illisible rend `(0, true)` : l absence de version est, elle
// aussi, une absence de profil — et elle se compte sous `filmdec_unknown_format_0`.
// IL PREND LE FILM, PAS UN `FilmContext` : il n a besoin que des huit premiers octets de
// `chunk_00`, et construire un contexte pour ca en creerait un SECOND pour le meme film — ce
// que `archlint/no_recomputed_film_context_test.go` interdit depuis le lot 1.9.2.
//
// IL DELEGUE A [grammar.MPPWidthsForFilm] DEPUIS LA REVUE M1 (2026-09-15) : la resolution du
// decoupage MPP se fait a UN SEUL endroit, celui que les deux sites de cuisson employent. Une
// seconde lecture de la meme valeur ici aurait diverge au premier format ajoute — c est
// exactement ce qui etait arrive entre ce fichier et `gwWidthsForFilm`.
func formatSansProfil(film *source.Film) (int, bool) {
	res := grammar.MPPWidthsForFilm(film)
	return res.FormatVersion, res.FormatInconnu
}

// avertirFormatSansProfil emet L UNIQUE avertissement par film. Appele par [BuildFromFilm],
// avant tout balayage, pour que la ligne precede les consequences qu elle explique.
func avertirFormatSansProfil(film *source.Film, matchID string) {
	format, sansProfil := formatSansProfil(film)
	if !sansProfil {
		return
	}
	build := ""
	if reg, ok := grammar.FilmRegistryChunk(film); ok {
		if id, err := grammar.ReadFilmIdentity(reg); err == nil {
			build = id.Build
		}
	}
	slog.Warn("version de format de chunk_00 INCONNUE de la table de profil — largeurs du bloc "+
		"de replication CALIBREES sur le film (repli repli_largeurs_mpp_calibrees_sur_le_film) ; "+
		"un patch du jeu a pu changer le format",
		"match_id", matchID, "format", format, "build", build)
}
