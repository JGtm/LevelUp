package duckdb

// match_view_repo_options.go — LE CONSTRUCTEUR ET LES OPTIONS DE CABLAGE de MatchViewRepo,
// plus les deux resolutions qui en decoulent (`viewer`, `sharedRead`).
//
// POURQUOI CE FICHIER EXISTE : DEPLACEMENT, PAS REECRITURE. match_view_repo.go franchissait le
// seuil de 500 lignes (513) apres l arrivee de l option `WithBombStats`. Tout ce qui CONFIGURE
// le repo en sort tel quel, sans une ligne de logique changee ; le fichier d origine garde le
// type, les lectures de base et les helpers de resolution d assets.
//
// LA FRONTIERE : ici, ce que le wiring INJECTE (classifieur de source de degat, viewer de
// session, capabilities du titre, overrides de libelle, reader shared, taxonomie de modes) et
// comment chaque champ se resout quand il n a pas ete injecte. La-bas, ce que le repo LIT.
//
// TOUTES LES OPTIONS RENDENT LE REPO : le cablage se chaine (`NewMatchViewRepo(...).With...()`),
// et une option non appelee a un repli DOCUMENTE — jamais un comportement devine.

import (
	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/port"
)

// viewer retourne le liker dont l'état de like doit être servi dans l'onglet
// Médias. Même repli — et mêmes raisons — que MediaRepo.viewer : sans joueur
// courant en session (instance mono-utilisateur), la page consultée est celle du
// joueur local, qui est donc le viewer.
func (r *MatchViewRepo) viewer() string {
	if r.viewerSlug != "" {
		return r.viewerSlug
	}
	if r.pdb == nil {
		return ""
	}
	return r.pdb.Gamertag
}

// NewMatchViewRepo crée un MatchViewRepo.
func NewMatchViewRepo(pdb *PlayerDB, xuid string) *MatchViewRepo {
	return &MatchViewRepo{pdb: pdb, xuid: xuid}
}

// WithKillSourceClassifier injecte le traducteur de source de degat du titre. nil (ou
// non appele) : les armes du match restent lues dans `v_weapon_kills`.
func (r *MatchViewRepo) WithKillSourceClassifier(c port.KillSourceClassifier) *MatchViewRepo {
	r.killSourceClassifier = c
	return r
}

// WithViewer injecte le slug du joueur qui consulte la page (session HTTP), qui
// détermine l'état `liked` des médias associés au match. Vide ou non appelé :
// repli documenté dans viewer().
func (r *MatchViewRepo) WithViewer(slug string) *MatchViewRepo {
	r.viewerSlug = slug
	return r
}

// WithBombStats active la lecture des STATISTIQUES D'ASSAUT reconstruites du film
// (`match_bomb_stats_latest`, capability `film.bomb_stats`). Câblé au wiring depuis la
// CapabilityMap du titre. Retourne le repo pour chaînage.
func (r *MatchViewRepo) WithBombStats(enabled bool) *MatchViewRepo {
	r.bombStats = enabled
	return r
}

// WithPlaylistCategoryStrip active/désactive le retrait du préfixe de catégorie
// matchmaking du libellé de playlist (CapPlaylistCategoryStrip). Câblé au wiring
// depuis la CapabilityMap du titre. Retourne le repo pour chaînage.
func (r *MatchViewRepo) WithPlaylistCategoryStrip(enabled bool) *MatchViewRepo {
	r.stripPlaylistCategory = enabled
	return r
}

// WithPlaylistLabelOverrides injecte la table data-driven des overrides de
// libellé de playlist (nom brut -> libellé court, playlist_labels.toml). nil = no-op.
// Retourne le repo pour chaînage.
func (r *MatchViewRepo) WithPlaylistLabelOverrides(overrides map[string]string) *MatchViewRepo {
	r.playlistLabelOverrides = overrides
	return r
}

// WithSharedReader injecte un SharedReader override pour les lectures shared (pilote
// snapshot scoped). Retourne le repo pour chaînage. nil = no-op (reste sur pdb.SharedReadDB()).
func (r *MatchViewRepo) WithSharedReader(sr SharedReader) *MatchViewRepo {
	r.sharedReader = sr
	return r
}

// WithModeTaxonomy injecte la classification des modes du titre (préfixes pair_name
// par catégorie) pour le filtrage neighbors. Sans injection, la clause ModeCategory
// est omise (dégradation gracieuse). Câblé au wiring depuis games/halo_infinite (F15-2).
func (r *MatchViewRepo) WithModeTaxonomy(t analysis.ModeTaxonomy) *MatchViewRepo {
	r.modeTax = t
	return r
}

// sharedRead retourne le SharedReader effectif : l'override snapshot s'il est câblé
// (et que la requête n'a pas basculé sur le live), sinon le reader live du pool
// (pdb.SharedReadDB()). forceLive prime : voir le champ (fallback snapshot-miss).
func (r *MatchViewRepo) sharedRead() SharedReader {
	if r.sharedReader != nil && !r.forceLive {
		return r.sharedReader
	}
	return r.pdb.SharedReadDB()
}
