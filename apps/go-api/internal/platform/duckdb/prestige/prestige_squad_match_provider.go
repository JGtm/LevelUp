// Package duckdb — SquadMatchProvider pour le titre Halo Infinite.
//
// Implémente prestige.SquadMatchProvider : pour un roster d'escouade, retourne
// les derniers matchs où TOUT le roster a joué, avec tous les participants (pour
// la règle no-overlap) et la métrique du défi par membre. Lecture sur
// shared_matches_v2.duckdb via un duckdb.SharedReader (coordination RO↔RW).

package prestige

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/prestige"
)

// ModeTranslatorFR traduit une liste de libellés de mode EN normalisés vers leur
// forme FR (metadata.mode_name_tr, lang='fr'). Signature alignée sur
// duckdb.SquadRepo.LoadModeTranslationsFR — SOURCE UNIQUE du littéral SQL
// mode_name_tr : ne pas redupliquer la requête ici (règle ≤2 copies).
type ModeTranslatorFR func(ctx context.Context, modeENs []string) (map[string]string, error)

// PlaylistTranslatorFR traduit une liste de playlist_id (UUID metadata) vers leur
// nom FR (metadata.asset_translations, asset_type='playlist'). Signature alignée
// sur duckdb.SquadRepo.LoadAssetTranslationsFR("playlist", ids) — même résolveur
// par IDENTIFIANT que la page Carrière (ResolveAssetNamesBulk), qui comble le
// trou de données où match_registry.playlist_name_fr est vide pour certaines
// playlists (« Quick Play », « Big Team Battle » — V72-10 suite).
type PlaylistTranslatorFR func(ctx context.Context, playlistIDs []string) (map[string]string, error)

// PrestigeSquadMatchProvider lit match_participants pour l'évaluation des défis
// d'escouade.
type PrestigeSquadMatchProvider struct {
	reader               duckdb.SharedReader
	translateModesFR     ModeTranslatorFR     // optionnel : résolution FR des modes de l'indice escouade
	translatePlaylistsFR PlaylistTranslatorFR // optionnel : résolution FR des playlists par id (comble le trou name_fr vide)
}

// NewPrestigeSquadMatchProvider construit le provider depuis un duckdb.SharedReader
// (passer pdb.SharedReadDB() côté caller, comme HaloBaselineProvider).
func NewPrestigeSquadMatchProvider(reader duckdb.SharedReader) *PrestigeSquadMatchProvider {
	return &PrestigeSquadMatchProvider{reader: reader}
}

// WithModeTranslatorFR injecte la résolution FR canonique des modes de l'indice
// escouade (« Slayer » → « Assassin »), pour que l'API serve des libellés prêts à
// afficher en contexte FR (parité home / match-view / historique). Best-effort :
// sans traducteur, ou s'il échoue, l'indice reste en EN (jamais vide). Chaînable.
func (p *PrestigeSquadMatchProvider) WithModeTranslatorFR(fn ModeTranslatorFR) *PrestigeSquadMatchProvider {
	p.translateModesFR = fn
	return p
}

// WithPlaylistTranslatorFR injecte la résolution FR par IDENTIFIANT des
// playlists de l'indice escouade (comble le trou « Quick Play »/« Big Team
// Battle » dont match_registry.playlist_name_fr est vide — V72-10 suite, même
// mécanisme que career : résolution par playlist_id via asset_translations,
// jamais par nom). Best-effort : sans traducteur, ou s'il échoue, le libellé
// COALESCE(playlist_name_fr, playlist_name) existant est conservé (jamais
// vide). Chaînable.
func (p *PrestigeSquadMatchProvider) WithPlaylistTranslatorFR(fn PlaylistTranslatorFR) *PrestigeSquadMatchProvider {
	p.translatePlaylistsFR = fn
	return p
}

var _ prestige.SquadMatchProvider = (*PrestigeSquadMatchProvider)(nil)

// SquadMatchMetrics retourne les `limit` derniers matchs où tout le roster a
// joué (start_time >= since), chacun avec ses participants (Xuids) + la métrique
// par membre (Values). since borne le bas (created_at du défi + fenêtre) — un
// since zéro désactive la borne.
func (p *PrestigeSquadMatchProvider) SquadMatchMetrics(ctx context.Context, rosterXUIDs []string, _ string, metric string, limit int, since time.Time) ([]prestige.SquadMatchMetric, error) {
	if len(rosterXUIDs) == 0 {
		return nil, nil
	}
	col := mapMetricToColumn(metric)
	if col == "" {
		return nil, nil // métrique non mappée → pas d'évaluation possible
	}
	if limit <= 0 {
		limit = 50
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	db, release, err := p.reader.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("PrestigeSquadMatchProvider: shared reader: %w", err)
	}
	defer release()

	candidates, err := p.candidateMatches(ctx, db, rosterXUIDs, limit, since)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	return p.participantsWithMetric(ctx, db, candidates, col)
}

// candidateMatches retourne les match_id (les plus récents) où TOUT le roster a
// joué : COUNT(DISTINCT xuid parmi le roster) == taille du roster. Si since est
// non nul, seuls les matchs dont le start_time canonique est >= since sont
// retenus (borne basse Lot 4 — pas de complétion rétroactive).
func (p *PrestigeSquadMatchProvider) candidateMatches(ctx context.Context, db *sql.DB, roster []string, limit int, since time.Time) ([]string, error) {
	args := toAnyArgs(roster)
	sinceClause := ""
	if !since.IsZero() {
		sinceClause = " AND " + duckdb.StartTimeCanonicalSQL("mr") + " >= ?"
		args = append(args, since)
	}
	q := fmt.Sprintf(`
		SELECT mp.match_id
		FROM match_participants mp
		JOIN match_registry mr ON mr.match_id = mp.match_id
		WHERE mp.xuid IN (%s)%s
		GROUP BY mp.match_id, `+duckdb.StartTimeCanonicalSQL("mr")+`
		HAVING COUNT(DISTINCT mp.xuid) = %d
		ORDER BY `+duckdb.StartTimeCanonicalSQL("mr")+` DESC
		LIMIT %d
	`, sqlInPlaceholders(len(roster)), sinceClause, len(roster), limit)

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("PrestigeSquadMatchProvider.candidateMatches: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan candidate: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// participantsWithMetric retourne, pour les matchs donnés, TOUS les participants
// + la valeur de la colonne métrique par xuid.
func (p *PrestigeSquadMatchProvider) participantsWithMetric(ctx context.Context, db *sql.DB, matchIDs []string, col string) ([]prestige.SquadMatchMetric, error) {
	q := fmt.Sprintf(`
		SELECT mp.match_id, mp.xuid, COALESCE(CAST(%s AS DOUBLE), 0)
		FROM match_participants mp
		WHERE mp.match_id IN (%s)
	`, col, sqlInPlaceholders(len(matchIDs)))

	rows, err := db.QueryContext(ctx, q, toAnyArgs(matchIDs)...)
	if err != nil {
		return nil, fmt.Errorf("PrestigeSquadMatchProvider.participantsWithMetric: %w", err)
	}
	defer rows.Close()

	byMatch := make(map[string]*prestige.SquadMatchMetric)
	var order []string
	for rows.Next() {
		var matchID, xuid string
		var val float64
		if err := rows.Scan(&matchID, &xuid, &val); err != nil {
			return nil, fmt.Errorf("scan participant: %w", err)
		}
		sm, ok := byMatch[matchID]
		if !ok {
			sm = &prestige.SquadMatchMetric{MatchID: matchID, Values: map[string]float64{}}
			byMatch[matchID] = sm
			order = append(order, matchID)
		}
		sm.Xuids = append(sm.Xuids, xuid)
		sm.Values[xuid] = val
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]prestige.SquadMatchMetric, 0, len(order))
	for _, id := range order {
		out = append(out, *byMatch[id])
	}
	return out, nil
}

// uuidLabelRE détecte un label non résolu (UUID brut) à écarter de l'indice
// d'affichage (miroir du filtre UUID côté front pour les options de cascade).
var uuidLabelRE = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// SquadUsualContexts retourne les playlists/modes dominants (top 2) parmi les
// `limit` derniers matchs où TOUT le roster a joué ensemble. Labels résolus FR
// via v_match_full ; vides et UUID non résolus écartés. Lecture seule.
func (p *PrestigeSquadMatchProvider) SquadUsualContexts(ctx context.Context, rosterXUIDs []string, _ string, limit int) (playlists, modes []string, err error) {
	if len(rosterXUIDs) == 0 {
		return nil, nil, nil
	}
	if limit <= 0 {
		limit = 60
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	db, release, err := p.reader.Get(ctx)
	if err != nil {
		slog.WarnContext(ctx, "SquadUsualContexts: shared reader indisponible (indice escouade dégradé)",
			"err", err, "roster_size", len(rosterXUIDs))
		return nil, nil, fmt.Errorf("SquadUsualContexts: shared reader: %w", err)
	}
	defer release()

	q := fmt.Sprintf(`
		WITH cm AS (
			SELECT mp.match_id,
			       MAX(`+duckdb.StartTimeCanonicalSQL("mr")+`) AS st
			FROM match_participants mp
			JOIN match_registry mr ON mr.match_id = mp.match_id
			WHERE mp.xuid IN (%s)
			GROUP BY mp.match_id
			HAVING COUNT(DISTINCT mp.xuid) = %d
			ORDER BY st DESC
			LIMIT %d
		)
		SELECT
			COALESCE(r.playlist_id, '')                                    AS playlist_id,
			COALESCE(NULLIF(r.playlist_name_fr, ''), r.playlist_name, '') AS playlist,
			COALESCE(NULLIF(r.pair_name_fr, ''), r.pair_name, '')         AS mode
		FROM cm
		JOIN v_match_full r ON r.match_id = cm.match_id
	`, sqlInPlaceholders(len(rosterXUIDs)), len(rosterXUIDs), limit)

	rows, err := db.QueryContext(ctx, q, toAnyArgs(rosterXUIDs)...)
	if err != nil {
		slog.WarnContext(ctx, "SquadUsualContexts: échec requête (indice escouade dégradé)",
			"err", err, "roster_size", len(rosterXUIDs))
		return nil, nil, fmt.Errorf("SquadUsualContexts: query: %w", err)
	}
	defer rows.Close()

	tally := newPlaylistTally()
	mdCount := map[string]int{}
	for rows.Next() {
		var plID, pl, md string
		if err := rows.Scan(&plID, &pl, &md); err != nil {
			return nil, nil, fmt.Errorf("SquadUsualContexts: scan: %w", err)
		}
		// pair_name est un libellé composé mode+carte (« Slayer on Bazaar ») :
		// normaliser pour ne servir que le mode (V72-10, parité match view).
		md = analysis.NormalizeModeLabel(md)
		if pl != "" && !uuidLabelRE.MatchString(pl) {
			tally.add(plID, pl)
		}
		if md != "" && !uuidLabelRE.MatchString(md) {
			mdCount[md]++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	topLabels, idByLabel := tally.top(2)
	playlists = p.applyPlaylistTranslationsFR(ctx, topLabels, idByLabel)
	modes = p.applyModeTranslationsFR(ctx, topNByFreq(mdCount, 2))
	return playlists, modes, nil
}

// playlistTally compte les playlists de l'indice escouade par IDENTITÉ —
// playlist_id quand il est connu, libellé en dernier repli — et non par libellé
// (contre-revue V7.2, 2026-07-25).
//
// Pourquoi : deux libellés peuvent désigner la MÊME playlist (renommage
// saisonnier, playlist_name_fr renseigné sur une partie des lignes seulement,
// casse/espaces divergents dans le registre). Compter par libellé éclatait alors
// une playlist dominante en deux entrées sous-comptées, capables de sortir du
// top 2 au profit d'une playlist réellement moins jouée — ou d'y apparaître deux
// fois. La traduction FR reste appliquée APRÈS le comptage, par id
// (applyPlaylistTranslationsFR), donc elle ne peut plus scinder ni fusionner un
// compte.
type playlistTally struct {
	count   map[string]int    // clé d'identité → occurrences
	labelBy map[string]string // clé d'identité → libellé représentatif (le premier vu)
	idBy    map[string]string // clé d'identité → playlist_id ("" si identité = libellé)
}

func newPlaylistTally() *playlistTally {
	return &playlistTally{
		count:   map[string]int{},
		labelBy: map[string]string{},
		idBy:    map[string]string{},
	}
}

// add comptabilise une occurrence. id vide (registre sans playlist_id) ⇒ repli sur
// le libellé, qui redevient la seule identité disponible.
func (t *playlistTally) add(id, label string) {
	key := id
	if key == "" {
		key = label
	}
	t.count[key]++
	if _, ok := t.labelBy[key]; !ok {
		t.labelBy[key] = label
		t.idBy[key] = id
	}
}

// top retourne les n playlists les plus jouées (libellés représentatifs, ordre
// topNByFreq) et la table libellé → playlist_id attendue par
// applyPlaylistTranslationsFR.
func (t *playlistTally) top(n int) (labels []string, idByLabel map[string]string) {
	keys := topNByFreq(t.count, n)
	labels = make([]string, 0, len(keys))
	idByLabel = make(map[string]string, len(keys))
	for _, k := range keys {
		label := t.labelBy[k]
		labels = append(labels, label)
		if id := t.idBy[k]; id != "" {
			idByLabel[label] = id
		}
	}
	return labels, idByLabel
}

// applyModeTranslationsFR remplace chaque libellé de mode EN normalisé par sa
// traduction FR (mode_name_tr) si disponible ; conserve l'EN sinon — un libellé
// n'est JAMAIS vidé. Best-effort : traducteur nil ou en erreur → modes inchangés
// (dégradation gracieuse, indice servi en anglais plutôt qu'absent). Un mode sans
// entrée mode_name_tr[fr] (trou de couverture) reste en EN = rattrapage données.
func (p *PrestigeSquadMatchProvider) applyModeTranslationsFR(ctx context.Context, modes []string) []string {
	if p.translateModesFR == nil || len(modes) == 0 {
		return modes
	}
	fr, err := p.translateModesFR(ctx, modes)
	if err != nil {
		slog.WarnContext(ctx, "SquadUsualContexts: traduction FR des modes indisponible (indice servi en anglais)",
			"err", err, "modes", modes)
		return modes
	}
	out := make([]string, len(modes))
	for i, m := range modes {
		if t, ok := fr[m]; ok && strings.TrimSpace(t) != "" {
			out[i] = t
		} else {
			out[i] = m
		}
	}
	return out
}

// applyPlaylistTranslationsFR remplace chaque libellé de playlist de l'indice
// escouade par sa traduction FR résolue par IDENTIFIANT (metadata.asset_translations,
// même résolveur — ResolveAssetNamesBulk — que la page Carrière), quand disponible.
// Comble le trou de données : la requête SQL sert COALESCE(playlist_name_fr,
// playlist_name) sur match_registry, où playlist_name_fr est vide pour certaines
// playlists (« Quick Play », « Big Team Battle » — V72-10 suite). Priorité :
// traduction par id > name_fr de match_registry (déjà encodé dans idByLabel/label)
// > name EN. Best-effort : traducteur nil, id inconnu/absent, ou en erreur →
// libellé COALESCE existant conservé (un libellé n'est jamais vidé).
func (p *PrestigeSquadMatchProvider) applyPlaylistTranslationsFR(ctx context.Context, playlists []string, idByLabel map[string]string) []string {
	if p.translatePlaylistsFR == nil || len(playlists) == 0 {
		return playlists
	}
	ids := make([]string, 0, len(playlists))
	for _, label := range playlists {
		if id := idByLabel[label]; id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return playlists
	}
	fr, err := p.translatePlaylistsFR(ctx, ids)
	if err != nil {
		slog.WarnContext(ctx, "SquadUsualContexts: traduction FR des playlists indisponible (indice servi en anglais/partiel)",
			"err", err, "playlists", playlists)
		return playlists
	}
	out := make([]string, len(playlists))
	for i, label := range playlists {
		if id, ok := idByLabel[label]; ok {
			if t, ok2 := fr[id]; ok2 && strings.TrimSpace(t) != "" {
				out[i] = t
				continue
			}
		}
		out[i] = label
	}
	return out
}

// topNByFreq retourne les n clés les plus fréquentes (fréquence desc, départage
// alphabétique pour un ordre déterministe).
func topNByFreq(counts map[string]int, n int) []string {
	type kv struct {
		k string
		v int
	}
	arr := make([]kv, 0, len(counts))
	for k, v := range counts {
		arr = append(arr, kv{k, v})
	}
	sort.Slice(arr, func(i, j int) bool {
		if arr[i].v != arr[j].v {
			return arr[i].v > arr[j].v
		}
		return arr[i].k < arr[j].k
	})
	if n > len(arr) {
		n = len(arr)
	}
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, arr[i].k)
	}
	return out
}

// sqlInPlaceholders construit "?,?,…" pour une clause IN de n éléments.
func sqlInPlaceholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

// toAnyArgs convertit []string en []any pour QueryContext.
func toAnyArgs(xs []string) []any {
	args := make([]any, len(xs))
	for i, x := range xs {
		args[i] = x
	}
	return args
}
