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
	"levelup/go-api/internal/observability"
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
//
// ⚠⚠ LE NOM VIENT DE `v_gamertag_lookup`, LA VUE CANONIQUE, ET C EST UNE MESURE QUI L IMPOSE.
// `match_participants.gamertag` est vide en pratique : **27 607 lignes sur 27 989 sans nom, et
// 4 xuids nommes sur 16 996**. Une premiere version de ce roster lisait cette colonne : elle
// rendait une table quasi VIDE, donc le collecteur ecrivait 16 908 morts dont **10** portaient
// un xuid de victime. Des lignes qu aucun agregat carriere ne peut joindre — c est-a-dire
// exactement ce que ce chantier existe pour produire.
//
// La vue est la SOURCE UNIQUE du resolveur `xuid -> gamertag` du depot
// (`analysis.GamertagLookupViewSQL`), et sa cascade explique pourquoi lire une seule table ne
// suffit jamais : bot connu, puis `xuid_aliases`, puis `match_participants`, puis
// `killer_victim_pairs`, puis libelle masque. **`xuid_aliases` a cesse d etre alimente en avril
// 2026** ; ce sont les gamertags du KILL-FEED, portes par `killer_victim_pairs`, qui couvrent
// les adversaires croises depuis. Couverture mesuree : 18 219 xuids, 36 masques.
//
// AMBIGUITE : si deux participants du meme match portent le meme nom, on n en garde AUCUN.
// Ecrire les morts d un joueur sous le xuid d un autre serait pire que de n en ecrire aucun.
func (r *SharedRoster) IdentitiesForMatch(ctx context.Context, matchID string) (MatchIdentities, error) {
	parXUID, err := r.gamertagsForMatch(ctx, matchID)
	if err != nil {
		return MatchIdentities{}, err
	}
	out := MatchIdentities{
		ParXUID:    parXUID,
		ParNom:     make(map[string]string, len(parXUID)),
		ShotsFired: map[string]int{},
	}
	ambigus := map[string]bool{}
	for xuid, gt := range parXUID {
		if ambigus[gt] {
			continue
		}
		if _, deja := out.ParNom[gt]; deja {
			delete(out.ParNom, gt)
			ambigus[gt] = true
			slog.WarnContext(ctx, "killsource: deux participants portent le meme nom, aucun xuid "+
				"ne sera attribue", "match_id", matchID, "gamertag", gt)
			continue
		}
		out.ParNom[gt] = xuid
	}
	if err := r.participantsForMatch(ctx, matchID, &out); err != nil {
		return MatchIdentities{}, err
	}
	return out, nil
}

// gamertagsForMatch rend `xuid -> gamertag` pour les participants du match, par la vue
// canonique.
//
// ⚠ DEPENDANCE A CONSIGNER POUR LA BASCULE (phase 2) : `v_gamertag_lookup` lit
// `killer_victim_pairs`. Supprimer cette table sans avoir d abord rebranche la vue sur
// `match_kill_events_latest` ferait retomber les adversaires sur le libelle masque — une panne
// d affichage que rien ne signalerait, sur le chemin le plus visible du produit.
func (r *SharedRoster) gamertagsForMatch(ctx context.Context, matchID string) (map[string]string, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("SharedRoster: db nil")
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT mp.xuid, g.gamertag
		FROM match_participants mp
		JOIN v_gamertag_lookup g ON g.xuid = mp.xuid
		WHERE mp.match_id = ? AND mp.xuid IS NOT NULL AND mp.xuid <> ''
	`, matchID)
	if err != nil {
		return nil, fmt.Errorf("SharedRoster(%s): %w", matchID, err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]string, 16)
	for rows.Next() {
		var xuid string
		var gt sql.NullString
		if err := rows.Scan(&xuid, &gt); err != nil {
			return nil, fmt.Errorf("SharedRoster(%s) scan: %w", matchID, err)
		}
		if gt.Valid && gt.String != "" {
			out[xuid] = gt.String
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("SharedRoster(%s) rows: %w", matchID, err)
	}
	return out, nil
}

// participantsForMatch complete `out` avec les xuids du match et la reference `shots_fired`.
//
// ⚠ `shots_fired` NULL N EST PAS ZERO. Une colonne nulle veut dire « l API n a pas donne le
// nombre de tirs » ; zero veut dire « l API dit qu il n a pas tire ». La porte de publication
// traite les deux DIFFEREMMENT (refus faute de reference d un cote, verdict de l autre), donc la
// lecture ne doit surtout pas les confondre : une valeur nulle n entre pas dans la table.
func (r *SharedRoster) participantsForMatch(ctx context.Context, matchID string, out *MatchIdentities) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("SharedRoster: db nil")
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT xuid, shots_fired
		FROM match_participants
		WHERE match_id = ? AND xuid IS NOT NULL AND xuid <> ''
		ORDER BY xuid
	`, matchID)
	if err != nil {
		return fmt.Errorf("SharedRoster participants(%s): %w", matchID, err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var xuid string
		var shots sql.NullInt64
		if err := rows.Scan(&xuid, &shots); err != nil {
			return fmt.Errorf("SharedRoster participants(%s) scan: %w", matchID, err)
		}
		out.XUIDs = append(out.XUIDs, xuid)
		if shots.Valid {
			out.ShotsFired[xuid] = int(shots.Int64)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("SharedRoster participants(%s) rows: %w", matchID, err)
	}
	return nil
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
		// LE BUDGET N EST PAS UNE ANNULATION DE CONTEXTE, ET C EST DELIBERE. Une premiere
		// version bornait la passe en passant un `context.WithTimeout` : a son expiration, le
		// test « depassement de delai » de CollectMatch (qui exige `ctx.Err() == nil` pour
		// distinguer l abandon du match de l arret de l appelant) devenait faux, et un arret
		// NOMINAL se comptait en `killsource_erreurs_decodage`. La surface d alerte etait
		// polluee par son propre garde-fou. Ici l arret est explicite, entre deux matchs.
		if c.budget > 0 && time.Since(start) >= c.budget {
			observability.AddInt(metricBudget, 1)
			slog.InfoContext(ctx, "killsource: budget de passe epuise — le solde repart au cycle suivant",
				"budget", c.budget, "traites", sum.Written+sum.NoFilm+sum.NoKillFeed+sum.Timeouts+sum.Errors,
				"total", sum.Total)
			break
		}
		if ctx.Err() != nil {
			slog.InfoContext(ctx, "killsource: passe interrompue par l appelant",
				"traites", sum.Written+sum.NoFilm+sum.NoKillFeed+sum.Timeouts+sum.Errors,
				"total", sum.Total)
			break
		}
		outcome, deaths, err := c.CollectMatch(ctx, id)
		if err != nil {
			sum.Errors++
			continue
		}
		switch outcome {
		case OutcomeWritten:
			sum.Written++
			sum.Deaths += deaths
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
