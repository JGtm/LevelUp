package main

// cmd_backfill_killsource_selection.go — QUELS FILMS CETTE PASSE VA DECODER, ET DANS QUEL ORDRE.
//
// Extrait de cmd_backfill_killsource.go le 2026-09-07 (revue de 7C), quand la seconde condition
// de fraicheur l'a repousse au-dela du seuil de 500 lignes. La coupure suit une frontiere
// nette : LA-BAS ce que la passe FAIT, ICI ce sur quoi elle le fait.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"levelup/go-api/internal/domain/killscope"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/sync/killcollector"
)

// bilanDeSelection : les DENOMINATEURS de la selection, qu elle seule connait.
//
// Ils existent depuis le lot 5.24.3 parce que le fichier d etat doit repondre a « il reste
// combien SUR COMBIEN » : sans eux, la passe ne sait dire que le nombre de films qu elle va
// decoder, et « 40 films a faire » ne dit pas si 1 500 ont ete sautes ou si le registre en
// compte 40. `DejaAJour` EST le compte de la reprise.
type bilanDeSelection struct {
	TotalRegistre   int
	DejaAJour       int
	SansFilmEnCache int
}

// filmsACollecter : les films du cache qui correspondent a un match du registre, tries par
// COUT CROISSANT, moins ceux qui sont deja a jour.
func filmsACollecter(
	ctx context.Context, db *sql.DB, cacheRoot string, o killsourceOptions,
) ([]filmCandidat, bilanDeSelection, error) {
	registre, err := matchsDuRegistre(ctx, db, 0)
	if err != nil {
		return nil, bilanDeSelection{}, err
	}
	dejaFaits := map[string]bool{}
	if !o.force {
		if dejaFaits, err = matchsAJour(ctx, db); err != nil {
			return nil, bilanDeSelection{}, err
		}
	}

	bilan := bilanDeSelection{TotalRegistre: len(registre)}
	out := make([]filmCandidat, 0, len(registre))
	for _, id := range registre {
		if dejaFaits[id] {
			bilan.DejaAJour++
			continue
		}
		n, ok := compterChunks(cacheRoot, id)
		if !ok {
			bilan.SansFilmEnCache++
			continue // pas de film en cache : ce match releve du producteur credit-seul
		}
		out = append(out, filmCandidat{matchID: id, chunks: n})
	}
	// LES GROS EN DERNIER. A cout egal, l identifiant departage — une passe doit etre
	// reproductible, y compris dans son ordre.
	sort.Slice(out, func(i, j int) bool {
		if out[i].chunks != out[j].chunks {
			return out[i].chunks < out[j].chunks
		}
		return out[i].matchID < out[j].matchID
	})
	if o.limit > 0 && len(out) > o.limit {
		out = out[:o.limit]
	}
	return out, bilan, nil
}

// matchsAJour : les matchs dont TOUTES les passes courantes portent leur revision de decodeur
// courante — le journal des morts, les faits d isolement ET le placement des vies.
//
// La lecture passe par les VUES `_latest` (ADR 0026) : une passe ancienne, deja supplantee, ne
// doit pas faire sauter un match. `read_path` distingue les deux producteurs — un match couvert
// par le credit-seul reste candidat au decodage de son film, et c est voulu : le film apporte la
// source du degat, que le credit ne peut pas connaitre.
//
// ─── LES FAITS D ISOLEMENT ONT LEUR PROPRE FRAICHEUR (lot 7C, 2026-09-07) ────────────────
//
// Sans la seconde condition, TOUT LE CORPUS DEJA COLLECTE resterait sans `match_lives` ni
// `match_death_context` : son journal porte deja la revision courante, donc le rattrapage le
// sauterait a jamais, et la lecture d isolement serait vide sur tout l historique sans qu aucune
// erreur ne le dise.
//
// LA CONDITION NE PORTE QUE SUR LES MATCHS QUI PEUVENT AVOIR DES FAITS, et il y a DEUX facons
// de ne pas pouvoir :
//
//	AUCUNE POSITION      un film dont la carte est hors du catalogue de bornes n en produit
//	                     pas, donc aucun fait d isolement ne peut naitre ;
//	AUCUNE EQUIPE        `match_participants.team_id` vide — « isole » se mesure ENTRE
//	                     coequipiers, et le film ne porte aucun camp. La projection se saute.
//
// SANS LA SECONDE, LA PASSE NE CONVERGE PAS : un match a positions mais sans equipe serait
// redecode a CHAQUE passe, indefiniment, pour reproduire le meme refus (constat de revue).
//
// ─── LE PLACEMENT DES VIES A LA SIENNE (plan Emprise vies, decision V12, 2026-09-29) ─────
//
// Tout match qui A DES VIES doit porter une passe de `match_life_placement_latest` a
// [killcollector.PlacementRev] : c est la seule population ou le placement peut naitre (il ne
// s ecrit qu apres des vies ecrites). Sans cette condition, le corpus deja collecte resterait
// sans placement, son journal et ses vies portant deja leurs revisions courantes.
//
// LA PASSE CONVERGE AUSSI SUR UN PONT NON PUBLIABLE (lot V2b, 2026-09-29) : un tel match a des
// vies, et le collecteur lui ecrit desormais une ligne de placement par vie (chaque vie entiere
// « non situee », rien de mesure) au lieu d'aucune — il sort donc de la liste des candidats
// comme les autres, sans regle de plus (3 matchs sur 1 519 a vies sur la copie du 2026-09-28
// etaient re-selectionnes a chaque passe avant ce lot). Le backlog AUTOMATIQUE du post-sync ne
// lit pas cette condition (`conditionBacklog`, inchange) : aucun redecodage ne part de lui-meme.
//
// LE PLACEMENT DOIT ETRE POSTERIEUR AUX VIES COURANTES (revue adversariale V5.1, R2, 2026-09-30) :
// un placement a `PlacementRev` calcule sur d anciennes vies ne rend pas le match a jour. Cas
// reel : le collecteur reecrit les vies (revision d isolement montee), puis l ecriture du
// placement echoue (compteur `killsource_placement_erreurs_ecriture`) — l ancienne passe de placement, deja a
// `PlacementRev`, ferait croire le match a jour et l Emprise lirait un placement d anciennes
// vies. LE CRITERE EST `written_at` : `decode_pass` est un tirage aleatoire (aucun ordre), alors
// que les deux persisters posent `time.Now().UTC()` UNE fois par passe (toutes les lignes d une
// passe le partagent) et que le collecteur ecrit le placement APRES les vies : sur un chemin
// sain, `placement.written_at >= vies.written_at`. `>=` (et non `>`) : deux horloges egales sont
// le meme instant, pas une preuve de peremption. Les vues `_latest` portent chacune la passe
// courante ; `MAX` n en est que l unique valeur de la passe.
func matchsAJour(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT DISTINCT e.match_id FROM match_kill_events_latest e
		WHERE e.decoder_rev = ? AND e.read_path <> ?
		  AND (
		    NOT EXISTS (SELECT 1 FROM kill_positions_latest p WHERE p.match_id = e.match_id)
		    OR EXISTS (SELECT 1 FROM match_lives_latest l
		               WHERE l.match_id = e.match_id AND l.decoder_rev = ?)
		    OR NOT EXISTS (SELECT 1 FROM match_participants mp
		                   WHERE mp.match_id = e.match_id AND mp.team_id IS NOT NULL)
		  )
		  AND (
		    NOT EXISTS (SELECT 1 FROM match_lives_latest l WHERE l.match_id = e.match_id)
		    OR EXISTS (SELECT 1 FROM match_life_placement_latest pl
		               WHERE pl.match_id = e.match_id AND pl.decoder_rev = ?
		                 AND pl.written_at >= (SELECT MAX(l.written_at) FROM match_lives_latest l
		                                       WHERE l.match_id = e.match_id))
		  )`,
		decfilm.Rev, killscope.ReadPathCreditBackfill,
		killcollector.IsolationDecoderRev, killcollector.PlacementRev)
	if err != nil {
		return nil, fmt.Errorf("matchs deja a jour: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("matchs deja a jour (scan): %w", err)
		}
		out[id] = true
	}
	return out, rows.Err()
}

// matchsDuRegistre : les matchs du registre, dans un ordre stable.
func matchsDuRegistre(ctx context.Context, db *sql.DB, limit int) ([]string, error) {
	q := `SELECT match_id FROM match_registry ORDER BY match_id`
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("registre des matchs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("registre des matchs (scan): %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// compterChunks : le nombre de chunks declares au manifeste cache. C est le PROXY DE COUT, et
// il est lu sans ouvrir un seul chunk.
func compterChunks(cacheRoot, matchID string) (int, bool) {
	court := matchID
	if i := strings.IndexByte(matchID, '-'); i > 0 {
		court = matchID[:i]
	}
	raw, err := os.ReadFile(filmcache.ManifestPath(cacheRoot, court))
	if err != nil {
		return 0, false
	}
	var m struct {
		Chunks []struct{} `json:"chunks"`
	}
	if err := json.Unmarshal(raw, &m); err != nil || len(m.Chunks) == 0 {
		return 0, false
	}
	return len(m.Chunks), true
}

// afficherPlan : le plan de passe, avec sa queue de films chers en evidence.
func afficherPlan(candidats []filmCandidat) {
	total := 0
	gros := 0
	for _, c := range candidats {
		total += c.chunks
		if c.chunks > 50 {
			gros++
		}
	}
	fmt.Printf("  chunks a decoder : %d au total, %d film(s) au-dela de 50 chunks (passes en dernier)\n",
		total, gros)
	for i, c := range candidats {
		if i >= 5 && i < len(candidats)-3 {
			continue
		}
		if i == 5 {
			fmt.Println("  ...")
		}
		fmt.Printf("  %-40s %3d chunks\n", c.matchID, c.chunks)
	}
}
