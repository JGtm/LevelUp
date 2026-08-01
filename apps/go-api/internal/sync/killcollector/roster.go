package killcollector

// roster.go — la resolution `gamertag -> xuid`, et la passe multi-matchs.
//
// POURQUOI CETTE RESOLUTION EXISTE : le film ne porte AUCUN xuid cote replication — il ne rend
// que des NOMS. Le rattachement au joueur se fait donc contre le roster du match, en base. Son
// echec n est PAS une erreur : un bot n a pas de xuid, et un nom que le roster ne connait pas
// est une donnee (la ligne s ecrit avec un xuid NULL).

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// SharedRoster : la resolution nom -> xuid depuis `shared.match_participants`.
//
// La table est celle du sync primaire : elle est deja peuplee quand le collecteur passe (c est
// meme la raison pour laquelle le collecteur est une passe SEPAREE — il travaille sur des matchs
// deja au registre).
type SharedRoster struct {
	db *sql.DB
}

// NewSharedRoster construit la resolution. `db` peut etre un handle LECTURE (le collecteur
// n ecrit pas par ce chemin) — utiliser `OpenReadForQuery` si la DB est potentiellement tenue
// en RW par le process (jamais `OpenReadOnly` force : erreurs « different configuration »).
func NewSharedRoster(db *sql.DB) *SharedRoster { return &SharedRoster{db: db} }

// RosterForMatch rend `gamertag -> xuid` pour les participants du match.
//
// ⚠ La cle est le GAMERTAG parce que c est la seule quantite que le film rende. Un gamertag qui
// change entre le match et la collecte ne se rattachera pas — c est une limite connue, et elle
// se lit dans la donnee (xuid NULL), pas dans un silence.
func (r *SharedRoster) RosterForMatch(ctx context.Context, matchID string) (map[string]string, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("SharedRoster: db nil")
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT gamertag, xuid
		FROM match_participants
		WHERE match_id = ? AND gamertag IS NOT NULL AND gamertag <> ''
	`, matchID)
	if err != nil {
		return nil, fmt.Errorf("SharedRoster(%s): %w", matchID, err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]string, 16)
	for rows.Next() {
		var gt string
		var xuid sql.NullString
		if err := rows.Scan(&gt, &xuid); err != nil {
			return nil, fmt.Errorf("SharedRoster(%s) scan: %w", matchID, err)
		}
		if xuid.Valid && xuid.String != "" {
			out[gt] = xuid.String
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("SharedRoster(%s) rows: %w", matchID, err)
	}
	return out, nil
}

// KillSourceSummary : ce qu une passe multi-matchs a produit. Tous les compteurs sont des
// ENTIERS et chacun correspond a un outcome — pas de « erreurs » fourre-tout qui melangerait un
// film absent (normal) et une base injoignable (panne).
type KillSourceSummary struct {
	Total       int
	Written     int
	Deaths      int
	NoFilm      int
	NoKillFeed  int
	Timeouts    int
	Errors      int
	NotSupport  int
	ElapsedTime time.Duration
}

// CollectMatches : la passe de fond, EN SERIE.
//
// LA SERIE N EST PAS UNE SIMPLIFICATION. Les parametres de replication du decodeur sont des
// globaux de paquet ; `killsource.Decode` serialise deja par un verrou. Lancer N goroutines ne
// ferait qu empiler des appelants sur ce verrou — meme debit, plus de memoire, et un ordre
// d abandon imprevisible. On boucle.
//
// Une erreur sur UN match n arrete pas la passe : elle est comptee et journalisee. Seul l arret
// de l appelant (`ctx`) interrompt — et il rend la synthese de ce qui a ete fait, pas une erreur.
func (c *KillSourceCollector) CollectMatches(ctx context.Context, matchIDs []string) KillSourceSummary {
	start := time.Now()
	sum := KillSourceSummary{Total: len(matchIDs)}

	for _, id := range matchIDs {
		if ctx.Err() != nil {
			slog.InfoContext(ctx, "killsource: passe interrompue par l appelant",
				"traites", sum.Written+sum.NoFilm+sum.NoKillFeed+sum.Timeouts+sum.Errors,
				"total", sum.Total)
			break
		}
		outcome, err := c.CollectMatch(ctx, id)
		if err != nil {
			sum.Errors++
			continue
		}
		switch outcome {
		case OutcomeWritten:
			sum.Written++
		case OutcomeNoFilm:
			sum.NoFilm++
		case OutcomeNoKillFeed:
			sum.NoKillFeed++
		case OutcomeTimeout:
			sum.Timeouts++
		case OutcomeNotSupported:
			sum.NotSupport++
		}
	}

	sum.ElapsedTime = time.Since(start)
	slog.InfoContext(ctx, "killsource: passe terminee",
		"total", sum.Total, "ecrits", sum.Written, "films_absents", sum.NoFilm,
		"sans_killfeed", sum.NoKillFeed, "abandons_delai", sum.Timeouts,
		"erreurs", sum.Errors, "capability_absente", sum.NotSupport,
		"duration", sum.ElapsedTime)
	return sum
}
