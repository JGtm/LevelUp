//go:build integration

package killcollector

// postsync_sans_killfeed_integration_test.go — UN FILM LU SANS KILL QUITTE LE BACKLOG POUR SA
// REVISION (2026-10-02).
//
// Constat du parc : huit films complets (en-tete, replications, temps forts presents) decodes
// sans aucun kill etaient redecodes a CHAQUE cycle — aucune ligne dans `match_kill_events`, aucun
// marqueur, donc candidats a vie. Tries du plus recent au plus vieux, ils occupaient les huit
// places du cycle et le backlog (7 351 matchs) ne baissait plus.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : ne plus poser `killsource_sans_killfeed_rev` sur
// l issue « sans kill-feed » ; retirer sa condition de `conditionBacklog` ; la poser pour un film
// dont le chunk des temps forts n a pas ete servi ; comparer a autre chose que `decfilm.Rev`.

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/games/halo_infinite/film/finalise"
	"levelup/go-api/internal/sync/haloclient"
)

// decodeurSansKill : le decodeur rend `ErrNoKillFeed`, l issue des films du constat.
//
// LA COUTURE, PAS UNE BOBINE : aucune mini-bobine du depot ne porte de paquet de replication de
// type 0, donc le vrai decodeur s arrete sur `ErrNoPacket` avant de lire le kill-feed. La bobine
// traverse en revanche les VRAIES portes du collecteur (cle du film, carte du catalogue), et le
// test mesure ce qui suit l issue : marqueur, backlog, cycle suivant.
func decodeurSansKill(t *testing.T) {
	t.Helper()
	dOrigine := decoderLeFilm
	decoderLeFilm = func(context.Context, string, *decfilm.Film, *decfilm.Options) (*decfilm.Result, error) {
		return nil, decfilm.ErrNoKillFeed
	}
	t.Cleanup(func() { decoderLeFilm = dOrigine })
}

// bobineSansKill : la mini-bobine de reference (cle connue, carte du catalogue), morceau des
// temps forts compris.
func bobineSansKill(t *testing.T) []haloclient.FilmChunk {
	t.Helper()
	chunks := bobineEnChunks(t, bobineCleConnue, false)
	if dernier := chunks[len(chunks)-1]; !finalise.EstTempsForts(dernier.ChunkType) {
		t.Fatalf("bobine : dernier chunk de type %d, attendu les temps forts", dernier.ChunkType)
	}
	return chunks
}

// filmsServis : une source qui sert les MEMES chunks a tout match, et compte les demandes.
type filmsServis struct {
	chunks   []haloclient.FilmChunk
	demandes []string
}

func (f *filmsServis) GetFilmChunks(_ context.Context, matchID string) ([]haloclient.FilmChunk, bool, error) {
	f.demandes = append(f.demandes, matchID)
	return f.chunks, true, nil
}

// depsSansKill : les dependances de `depsDeTest`, sous la carte de la bobine et avec un cache
// disque temporaire (le film telecharge y est archive, jamais dans le depot).
func depsSansKill(t *testing.T, db *sql.DB, films *filmsServis) PostSyncDeps {
	t.Helper()
	racine := t.TempDir()
	if err := filmcache.EnsureDirs(racine); err != nil {
		t.Fatalf("cache temporaire : %v", err)
	}
	oublierLesMatchsSansCarte()
	return PostSyncDeps{
		Fetcher:       films,
		LocalCache:    haloclient.NewLocalFilmCache(racine),
		WithRead:      func(ctx context.Context, _ string, fn func(*sql.DB)) { fn(db) },
		AcquireWriter: func(context.Context) (*sql.DB, func(), error) { return db, func() {}, nil },
		TitleSlug:     "halo_infinite",
		Gamertag:      "GT",
		MapNames:      nomsDeCarteFixes{noms: []string{carteDeLaBobine}},
	}
}

func revSansKillFeed(t *testing.T, db *sql.DB, id string) sql.NullString {
	t.Helper()
	var rev sql.NullString
	if err := db.QueryRowContext(context.Background(),
		`SELECT killsource_sans_killfeed_rev FROM match_registry WHERE match_id = ?`, id,
	).Scan(&rev); err != nil {
		t.Fatalf("lecture killsource_sans_killfeed_rev %s : %v", id, err)
	}
	return rev
}

// TestRunPostSync_SansKillFeed_QuitteLeBacklogPourSaRevision : le film est decode UNE fois, sa
// revision est enregistree, et le cycle suivant ne le redemande plus.
func TestRunPostSync_SansKillFeed_QuitteLeBacklogPourSaRevision(t *testing.T) {
	decodeurSansKill(t)
	db := baseBacklog(t)
	inscrireMatch(t, db, "muet", time.Date(2026, 10, 2, 7, 34, 0, 0, time.UTC), 0)

	films := &filmsServis{chunks: bobineSansKill(t)}
	h := NewPostSyncHook(racineDepot(t), 8)
	if ecrits := RunPostSync(context.Background(), h, depsSansKill(t, db, films), nil); ecrits != 0 {
		t.Fatalf("ecrits = %d, attendu 0 (film sans kill)", ecrits)
	}
	if len(films.demandes) != 1 {
		t.Fatalf("premier cycle : demandes = %v, attendu [muet] — le test ne mesure rien si le film "+
			"n est pas decode une fois", films.demandes)
	}

	if rev := revSansKillFeed(t, db, "muet"); !rev.Valid || rev.String != decfilm.Rev {
		t.Fatalf("killsource_sans_killfeed_rev = %v, attendu %q — l issue « sans kill-feed » n est "+
			"pas enregistree, le film reviendra a chaque cycle", rev, decfilm.Rev)
	}
	if ids, total := backlogAJour(context.Background(), db, 10, 0); total != 0 || len(ids) != 0 {
		t.Fatalf("backlog apres passe = %v (total %d), attendu vide", ids, total)
	}

	RunPostSync(context.Background(), h, depsSansKill(t, db, films), nil)
	if len(films.demandes) != 1 {
		t.Errorf("second cycle : demandes = %v — le film sans kill est redecode", films.demandes)
	}
}

// TestRunPostSync_SansTempsFortsServis_ResteCandidat : un film dont le chunk des temps forts est
// DECLARE mais n a pas ete servi (absent du disque) n est pas un film lu sans kill : rien n est
// enregistre, le match reste candidat.
func TestRunPostSync_SansTempsFortsServis_ResteCandidat(t *testing.T) {
	decodeurSansKill(t)
	db := baseBacklog(t)
	inscrireMatch(t, db, "incomplet", time.Date(2026, 10, 2, 7, 34, 0, 0, time.UTC), 0)

	chunks := bobineEnChunks(t, bobineCleConnue, false)
	films := &filmsServis{chunks: chunks[:len(chunks)-1]}
	RunPostSync(context.Background(), NewPostSyncHook(racineDepot(t), 8), depsSansKill(t, db, films), nil)
	if len(films.demandes) != 1 {
		t.Fatalf("demandes = %v, attendu [incomplet]", films.demandes)
	}
	if rev := revSansKillFeed(t, db, "incomplet"); rev.Valid {
		t.Fatalf("killsource_sans_killfeed_rev = %q sur un film sans temps forts servis — il ne "+
			"reviendrait plus avant la prochaine revision", rev.String)
	}
	if ids, _ := backlogAJour(context.Background(), db, 10, 0); len(ids) != 1 {
		t.Errorf("backlog = %v, attendu [incomplet]", ids)
	}
}

// TestBacklogAJour_SansKillFeed_RevisionAncienneResteCandidate : la colonne ne sort le match que
// pour la revision COURANTE — une revision anterieure le laisse candidat.
func TestBacklogAJour_SansKillFeed_RevisionAncienneResteCandidate(t *testing.T) {
	db := baseBacklog(t)
	t0 := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	inscrireMatch(t, db, "courant", t0, 0)
	inscrireMatch(t, db, "ancien", t0.Add(time.Hour), 0)
	for id, rev := range map[string]string{"courant": decfilm.Rev, "ancien": "killsource-1999-01-01"} {
		if _, err := db.ExecContext(context.Background(),
			`UPDATE match_registry SET killsource_sans_killfeed_rev = ? WHERE match_id = ?`, rev, id,
		); err != nil {
			t.Fatalf("update %s : %v", id, err)
		}
	}
	ids, total := backlogAJour(context.Background(), db, 10, 0)
	if total != 1 || len(ids) != 1 || ids[0] != "ancien" {
		t.Errorf("backlog = %v (total %d), attendu [ancien]", ids, total)
	}
}
