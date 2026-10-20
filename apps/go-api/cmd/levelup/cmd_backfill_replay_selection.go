package main

// cmd_backfill_replay_selection.go — `backfill-replay` : LA SELECTION DES FILMS A CUIRE
// (`--since`, `--match`, `--demo`).
//
// Sans ces options la passe prend tout le cache (des heures). Elles bornent la population AVANT
// le filtre « deja a jour », le tri par cout et `--limit` : un pilote `--limit 25 --since ...`
// livre 25 constructions reelles parmi les films retenus. La cuisson, elle, ne change pas : un
// processus par film sous plafond memoire (`filmproc`).
//
// SEMANTIQUE
//
//	--since AAAA-MM-JJ   films dont le match a commence a cette date UTC (minuit inclus) ou apres,
//	                     date CANONIQUE du registre (analysis.SQLStartTimeCanonical). Un match
//	                     sans date est ECARTE et COMPTE (« sans date canonique »), jamais cuit.
//	--match id1,id2,...  ces matchs seulement : identifiants complets ou prefixes univoques d au
//	                     moins 8 caracteres, resolus contre le registre (meme resolveur que
//	                     backfill-killsource et backfill-vehicle-takes) ; refus sinon.
//	--demo               les rejeux figes (`replay_matches`) des manifestes config/demo/*/*.json
//	                     du `--title` courant (ops.DemoReplayMatchIDs) : aucun identifiant recopie.
//
// COMBINAISONS. `--since` et `--demo` s ADDITIONNENT (union : « les films depuis telle date, plus
// ceux de la demo quelle que soit leur date »). `--match` est une liste exacte : il s emploie
// SEUL, avec `--since` ou `--demo` il est refuse. Ces options ne se combinent pas avec `--one`
// (forme enfant) ni `--purge-non-geres` (qui ne cuit rien) : refus plutot que drapeau ignore.
// Elles se combinent avec `--dry-run`, `--limit`, `--force`, `--only-existing` et
// `--repair-impoverished`.
//
// UN IDENTIFIANT DEMANDE (`--match`, `--demo`) QUI N ABOUTIT A AUCUN FILM CUISABLE (absent du
// cache, du registre, mode non gere, film illisible) est NOMME dans le bilan : une selection qui
// cuit moins que demande ne le tait jamais.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/platform/duckdb"
)

// formatDateSince : la forme de `--since`.
const formatDateSince = "2006-01-02"

// parserDepuisReplay lit `--since` : vide = pas de borne (zero), sinon minuit UTC du jour.
func parserDepuisReplay(brut string) (time.Time, error) {
	brut = strings.TrimSpace(brut)
	if brut == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(formatDateSince, brut)
	if err != nil {
		return time.Time{}, fmt.Errorf("--since %q : date attendue au format AAAA-MM-JJ (%w)", brut, err)
	}
	return t, nil
}

// selectionActive : au moins une des options de selection est posee.
func (o replayBackfillOptions) selectionActive() bool {
	return !o.depuis.IsZero() || strings.TrimSpace(o.match) != "" || o.demo
}

// validerSelectionReplay : les incompatibilites des options de selection, AVANT d ouvrir quoi que ce soit.
func validerSelectionReplay(o replayBackfillOptions) error {
	if o.match != "" && len(decouperMatchs(o.match)) == 0 {
		return fmt.Errorf("--match %q : aucun identifiant (liste separee par des virgules)", o.match)
	}
	if !o.selectionActive() {
		return nil
	}
	if o.match != "" && (o.demo || !o.depuis.IsZero()) {
		return fmt.Errorf("--match est une liste exacte : il s emploie seul (ni --since ni --demo ; " +
			"--since et --demo, eux, s additionnent)")
	}
	if strings.TrimSpace(o.one) != "" {
		return fmt.Errorf("--since, --match et --demo n ont pas de sens avec --one : il cuit CE film, il ne selectionne rien")
	}
	if o.purgeNonGeres {
		return fmt.Errorf("--since, --match et --demo n ont pas de sens avec --purge-non-geres : " +
			"la purge ne cuit rien et ne lit pas cette selection")
	}
	return nil
}

// selectionReplay : la selection resolue, prete a juger un match.
type selectionReplay struct {
	// depuis : borne basse (minuit UTC) ; zero = pas de borne de date.
	depuis time.Time
	// dates : debut canonique UTC par match_id du registre ; un match absent n a pas de date.
	dates map[string]time.Time
	// voulus : match_id en minuscules demandes par `--match` / `--demo` ; nil = pas de liste.
	voulus map[string]bool
}

// issueSelection : ce que la selection decide d UN match.
type issueSelection int

const (
	selectionRetenue issueSelection = iota
	// selectionHorsPerimetre : le match n est ni dans la liste ni apres la borne de date.
	selectionHorsPerimetre
	// selectionSansDate : une borne de date est posee et le match n en a pas ; ecarte, compte a part.
	selectionSansDate
)

// juger decide d un match. Une liste (`--demo`) retient le match quelle que soit sa date
// (union avec `--since`) ; sinon la borne de date tranche, et l absence de date ecarte.
func (s selectionReplay) juger(matchID string) issueSelection {
	if s.voulus[strings.ToLower(matchID)] {
		return selectionRetenue
	}
	if s.depuis.IsZero() {
		return selectionHorsPerimetre
	}
	debut, ok := s.dates[matchID]
	switch {
	case !ok:
		return selectionSansDate
	case debut.Before(s.depuis):
		return selectionHorsPerimetre
	default:
		return selectionRetenue
	}
}

// preparerSelectionReplay resout les options contre le registre (`--match`), les manifestes de
// demo (`--demo`) et les dates canoniques (`--since`). Rend la selection inactive (zero) quand
// aucune option n est posee. Lectures courtes, relachees avant tout lancement d enfant.
func preparerSelectionReplay(
	ctx context.Context, pr *titlePkg.PathResolver, o replayBackfillOptions, registre map[string]registreEntry,
) (selectionReplay, error) {
	sel := selectionReplay{depuis: o.depuis}
	if !o.selectionActive() {
		return sel, nil
	}
	if !o.depuis.IsZero() {
		dates, err := datesCanoniquesDuRegistre(ctx, pr, o.titleSlug)
		if err != nil {
			return sel, err
		}
		sel.dates = dates
	}
	if o.match != "" {
		ids := make([]string, 0, len(registre))
		for _, e := range registre {
			ids = append(ids, e.matchID)
		}
		sort.Strings(ids)
		resolus, err := resoudreMatchs(ids, o.match)
		if err != nil {
			return sel, err
		}
		sel.voulus = ajouterMinuscules(sel.voulus, clesDe(resolus))
	}
	if o.demo {
		ids, err := ops.DemoReplayMatchIDs(pr, o.titleSlug)
		if err != nil {
			return sel, fmt.Errorf("--demo : %w", err)
		}
		sel.voulus = ajouterMinuscules(sel.voulus, ids)
	}
	return sel, nil
}

// ajouterMinuscules ajoute ids (en minuscules) a l ensemble, cree au besoin.
func ajouterMinuscules(ensemble map[string]bool, ids []string) map[string]bool {
	if ensemble == nil {
		ensemble = make(map[string]bool, len(ids))
	}
	for _, id := range ids {
		ensemble[strings.ToLower(id)] = true
	}
	return ensemble
}

func clesDe(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// datesCanoniquesDuRegistre lit le debut canonique (UTC) de chaque match du registre. Un match
// sans date canonique est ABSENT du resultat. Lecture courte (OpenReadForQuery), relachee au
// retour.
func datesCanoniquesDuRegistre(ctx context.Context, pr *titlePkg.PathResolver, titleSlug string) (map[string]time.Time, error) {
	sharedPath := pr.SharedDBPath(titleSlug)
	db, release, err := duckdb.OpenReadForQuery(sharedPath)
	if err != nil {
		return nil, fmt.Errorf("open shared RO (%s): %w", sharedPath, err)
	}
	defer release()
	return lireDatesCanoniques(ctx, db)
}

// lireDatesCanoniques : la requete, separee de l ouverture pour se tester sur une base de test.
func lireDatesCanoniques(ctx context.Context, db *sql.DB) (map[string]time.Time, error) {
	rows, err := db.QueryContext(ctx, `SELECT match_id, `+analysis.SQLStartTimeCanonical("")+` FROM match_registry`)
	if err != nil {
		return nil, fmt.Errorf("dates canoniques du registre: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]time.Time{}
	for rows.Next() {
		var id string
		var debut sql.NullTime
		if err := rows.Scan(&id, &debut); err != nil {
			return nil, fmt.Errorf("dates canoniques du registre (scan): %w", err)
		}
		if debut.Valid {
			out[id] = debut.Time.UTC()
		}
	}
	return out, rows.Err()
}

// appliquerSelectionReplay compte l issue d UN match dans le bilan et dit s il est retenu. Un
// match sans date ecarte par `--since` est journalise : il n est jamais ecarte en silence.
func appliquerSelectionReplay(
	ctx context.Context, sel selectionReplay, matchID string, report *replayBackfillReport,
) bool {
	switch sel.juger(matchID) {
	case selectionHorsPerimetre:
		report.horsPerimetre++
		return false
	case selectionSansDate:
		report.sansDate++
		slog.WarnContext(ctx, "backfill-replay: match sans date canonique — ecarte par --since", "match_id", matchID)
		return false
	default:
		return true
	}
}

// noterDemandesSansFilm releve les identifiants demandes (`--match`, `--demo`) qui n ont donne
// aucun candidat : ils sont nommes dans le bilan.
func noterDemandesSansFilm(sel selectionReplay, candidats []replayCandidat, report *replayBackfillReport) {
	if len(sel.voulus) == 0 {
		return
	}
	trouves := make(map[string]bool, len(candidats))
	for _, c := range candidats {
		trouves[strings.ToLower(c.matchID)] = true
	}
	for id := range sel.voulus {
		if !trouves[id] {
			report.demandesSansFilm = append(report.demandesSansFilm, id)
		}
	}
	sort.Strings(report.demandesSansFilm)
}

// afficherBilanSelectionReplay : ce que la selection a ecarte, et ce qu elle n a pas trouve.
// Muet quand aucune option de selection n est posee.
func afficherBilanSelectionReplay(o replayBackfillOptions, r replayBackfillReport) {
	if !o.selectionActive() {
		return
	}
	var criteres []string
	if !o.depuis.IsZero() {
		criteres = append(criteres, "--since "+o.depuis.Format(formatDateSince))
	}
	if o.match != "" {
		criteres = append(criteres, "--match")
	}
	if o.demo {
		criteres = append(criteres, "--demo")
	}
	fmt.Printf("selection (%s) : %d hors perimetre, %d sans date canonique (ecartes), %d demandes sans film cuisable\n",
		strings.Join(criteres, " + "), r.horsPerimetre, r.sansDate, len(r.demandesSansFilm))
	for _, id := range r.demandesSansFilm {
		fmt.Printf("  demande sans film cuisable : %s (hors cache, hors registre, mode non gere ou film illisible)\n", id)
	}
}
