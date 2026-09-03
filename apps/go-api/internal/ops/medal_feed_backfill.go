// Package ops — medal_feed_backfill.go : RENDRE LEUR NOM AUX MEDAILLES DEJA EN BASE.
//
// LE TROU QUE CETTE PASSE COMBLE. Depuis la bascule Collect->Persist (avril 2026),
// le collecteur ecrivait les events `medal` de highlight_events SANS `raw_json` :
// 415 matchs, 22 031 events, aucune identite. Le fil des eliminations lit
// `raw_json.medal_name` (platform/duckdb.medalNameFromRawJSON) — sans lui il
// n affiche AUCUNE medaille. LES DEUX ECRIVAINS de la table sont repares depuis le
// 2026-09-02 — le flux primaire (sync/collect.go) ET la voie
// completion/convergence (sync/engine_highlight_events.go) ; cette passe rattrape
// l existant.
//
// D OU VIENT L IDENTITE. Du film lui-meme, relu dans le cache local : le bloc event
// porte le couple (type_hint, medal_type) et ce couple designe une medaille de
// facon univoque (bijection mesuree sur 44 568 events). La passe re-parse le chunk
// highlight, apparie ses events aux lignes deja en base par (xuid, time_ms), puis
// ecrit `type_hint` (toujours) et `raw_json` (quand le couple est connu).
//
// ELLE EST HORS LIGNE ET SERIALISEE. Aucune requete reseau : un film absent du
// cache est un match CONSIGNE ET SAUTE, jamais un telechargement. Un match a la
// fois, une transaction par match — un match est entierement rattrape ou pas
// touche. Le plafond par run permet de la lancer par lots.
//
// ELLE EXIGE LE SERVEUR ARRETE : le handle est ouvert en ecriture sur la base
// partagee (mono-process, ADR 0013). L ouverture appartient a l appelant.
//
// REPRENABLE : la selection ne retient que les matchs dont il reste un event medal
// a `raw_json IS NULL`. Un match entierement resolu sort du lot. Un match dont
// certains events restent sans nom (couple jamais mesure) ou sans paire (film
// desaccorde) y revient a chaque run — c est le signal, pas un bug : ces
// evenements-la n ont pas d identite a recuperer.
package ops

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sort"
	"strconv"

	"levelup/go-api/internal/analysis"
)

// SourceChunkHighlight rend le chunk highlight brut d un match. `trouve=false`
// signifie « pas de film pour ce match » (cache incomplet, film expire cote
// serveur) : ce n est pas une erreur, c est un match a sauter.
//
// L interface existe pour que la logique d appariement soit testable SANS film
// reel ; l implementation qui lit le cache disque vit chez l appelant (CLI).
type SourceChunkHighlight interface {
	ChunkHighlight(ctx context.Context, matchID string) (data []byte, trouve bool, err error)
}

// ResolveurNomMedaille rend le nom anglais (clef de referentiel) designe par le
// couple du film, et false si le couple est inconnu. Injecte pour que `ops` ne
// porte aucun savoir title-specific : l appelant passe la table du titre.
type ResolveurNomMedaille func(typeHint, medalType int) (string, bool)

// OptionsBackfillMedailles regle une passe.
type OptionsBackfillMedailles struct {
	// Plafond borne le nombre de matchs TRAITES, c est-a-dire ceux dont le film a
	// ete relu (0 = tous). Un match saute faute de film ne consomme pas le quota :
	// un lot borne avance donc toujours, meme si la tete du lot n a aucun film.
	Plafond int
	// DryRun calcule tout et n ecrit rien.
	DryRun bool
}

// BilanBackfillMedailles compte ce qu une passe a vu et fait. Aucun compteur n est
// silencieux : chaque event candidat finit dans exactement une case.
type BilanBackfillMedailles struct {
	MatchsCandidats  int
	MatchsTraites    int
	MatchsSansFilm   int
	MatchsIllisibles int
	EventsCandidats  int
	EventsIdentifies int
	EventsSansNom    int
	EventsSansPaire  int
}

// evenementBase est une ligne medal du match en base.
//
// LES LIGNES DEJA IDENTIFIEES EN FONT PARTIE, et ce n est pas un detail : le film
// porte TOUTES les medailles du match, donc l appariement doit compter tous les
// cotes. Ne charger que les lignes sans identite ferait echouer la comparaison de
// cardinal sur un match partiellement rempli, et ce match ne serait jamais
// rattrape. Les lignes deja identifiees sont appariees puis laissees telles quelles.
type evenementBase struct {
	id            int64
	xuid          string
	timeMS        int
	dejaIdentifie bool
}

// coupleAppariement est la clef d appariement entre le film et la base.
type coupleAppariement struct {
	xuid   string
	timeMS int
}

// BackfillIdentiteMedailles rejoue les films en cache pour rendre leur nom aux
// events medal deja en base. `db` doit etre un handle EN ECRITURE sur la base
// partagee, tenu par ce seul process.
func BackfillIdentiteMedailles(
	ctx context.Context,
	db *sql.DB,
	films SourceChunkHighlight,
	resoudre ResolveurNomMedaille,
	o OptionsBackfillMedailles,
) (BilanBackfillMedailles, error) {
	var bilan BilanBackfillMedailles
	matchs, err := matchsMedaillesSansIdentite(ctx, db)
	if err != nil {
		return bilan, err
	}
	bilan.MatchsCandidats = len(matchs)
	slog.InfoContext(ctx, "backfill medailles: lot selectionne",
		"matchs_candidats", len(matchs), "plafond", o.Plafond, "dry_run", o.DryRun)

	passe := passeMedailles{db: db, films: films, resoudre: resoudre, options: o}
	for _, matchID := range matchs {
		// LE PLAFOND COMPTE LES MATCHS TRAITES, pas les matchs examines : un match
		// sans film en cache est saute sans consommer le quota, sinon un lot borne
		// pouvait ne rien faire du tout (les sans-film sont en tete d ordre).
		if o.Plafond > 0 && bilan.MatchsTraites >= o.Plafond {
			break
		}
		if err := passe.traiterMatch(ctx, matchID, &bilan); err != nil {
			return bilan, err
		}
	}
	slog.InfoContext(ctx, "backfill medailles: passe terminee",
		"matchs_candidats", bilan.MatchsCandidats, "matchs_traites", bilan.MatchsTraites,
		"matchs_sans_film", bilan.MatchsSansFilm, "matchs_illisibles", bilan.MatchsIllisibles,
		"events_candidats", bilan.EventsCandidats, "events_identifies", bilan.EventsIdentifies,
		"events_sans_nom", bilan.EventsSansNom, "events_sans_paire", bilan.EventsSansPaire,
		"dry_run", o.DryRun)
	return bilan, nil
}

// passeMedailles porte les dependances d une passe, immuables du premier au dernier
// match. Elles voyagent ensemble plutot qu en cortege de parametres.
type passeMedailles struct {
	db       *sql.DB
	films    SourceChunkHighlight
	resoudre ResolveurNomMedaille
	options  OptionsBackfillMedailles
}

// traiterMatch rattrape un match. Une erreur de LECTURE du cache ou de la base est
// fatale pour la passe (elle signale un environnement casse) ; un film absent ou un
// chunk indecodable ne l est pas — c est consigne et compte.
func (p passeMedailles) traiterMatch(
	ctx context.Context, matchID string, bilan *BilanBackfillMedailles,
) error {
	data, trouve, err := p.films.ChunkHighlight(ctx, matchID)
	if err != nil {
		return fmt.Errorf("backfill medailles: chunk highlight de %s: %w", matchID, err)
	}
	if !trouve {
		bilan.MatchsSansFilm++
		slog.InfoContext(ctx, "backfill medailles: match saute, film absent du cache", "match_id", matchID)
		return nil
	}
	// Version de film 0 : le manifeste du cache ne la porte pas, et elle ne
	// deplace que la gamertag dans le bloc (versions 39-40). L appariement se
	// fait sur le xuid — lu au bit pres hors du bloc — et sur l instant : le
	// layout de la gamertag ne l atteint pas.
	events, err := analysis.ParseHighlightEvents(data, 0)
	if err != nil {
		bilan.MatchsIllisibles++
		slog.WarnContext(ctx, "backfill medailles: match saute, chunk highlight indecodable",
			"match_id", matchID, "err", err)
		return nil
	}

	enBase, err := medaillesDuMatch(ctx, p.db, matchID)
	if err != nil {
		return err
	}
	aRattraper := 0
	for _, e := range enBase {
		if !e.dejaIdentifie {
			aRattraper++
		}
	}
	bilan.EventsCandidats += aRattraper
	corrections := apparier(ctx, enBase, events, p.resoudre, bilan)
	if !p.options.DryRun && len(corrections) > 0 {
		if err := ecrireCorrections(ctx, p.db, matchID, corrections); err != nil {
			return err
		}
	}
	bilan.MatchsTraites++
	slog.InfoContext(ctx, "backfill medailles: match rattrape",
		"match_id", matchID, "events_en_base", len(enBase), "events_a_rattraper", aRattraper,
		"corrections", len(corrections), "dry_run", p.options.DryRun)
	return nil
}

// correction est ce qu on ecrit sur une ligne : le type_hint TOUJOURS (quantite
// mesuree), le document d identite SEULEMENT si le couple est connu.
type correction struct {
	id       int64
	typeHint int
	rawJSON  *string
}

// apparier associe chaque ligne en base a l event du film de meme (xuid, time_ms).
//
// L ORDRE DANS UN GROUPE FAIT FOI. Deux medailles peuvent tomber au meme instant
// pour le meme joueur (un multi-kill et sa serie). Le collecteur inserait les
// lignes dans l ordre du scan du film, et `id` est une sequence : le n-ieme event
// du film d un groupe correspond a la n-ieme ligne du groupe. Un groupe dont les
// deux cotes n ont pas le meme cardinal n est PAS devine — ses lignes comptent
// « sans paire ».
func apparier(
	ctx context.Context, enBase []evenementBase, events []analysis.HighlightEvent,
	resoudre ResolveurNomMedaille, bilan *BilanBackfillMedailles,
) []correction {
	duFilm := map[coupleAppariement][]analysis.HighlightEvent{}
	for _, ev := range events {
		if ev.EventType != analysis.EventTypeMedal {
			continue
		}
		c := coupleAppariement{xuid: strconv.FormatUint(ev.XUID, 10), timeMS: ev.TimeMS}
		duFilm[c] = append(duFilm[c], ev)
	}
	deLaBase := map[coupleAppariement][]evenementBase{}
	for _, e := range enBase {
		c := coupleAppariement{xuid: e.xuid, timeMS: e.timeMS}
		deLaBase[c] = append(deLaBase[c], e)
	}

	out := make([]correction, 0, len(enBase))
	for c, lignes := range deLaBase {
		evs := duFilm[c]
		if len(evs) != len(lignes) {
			for _, ligne := range lignes {
				if !ligne.dejaIdentifie {
					bilan.EventsSansPaire++
				}
			}
			continue
		}
		for i, ligne := range lignes {
			if ligne.dejaIdentifie {
				continue // appariee pour tenir le compte, mais rien a lui rendre
			}
			corr := correction{id: ligne.id, typeHint: evs[i].TypeHint}
			nom, connu := resoudre(evs[i].TypeHint, evs[i].MedalType)
			if !connu {
				bilan.EventsSansNom++
				out = append(out, corr)
				continue
			}
			raw, err := analysis.MedalRawJSON(nom)
			if err != nil {
				// Defensif : la table ne rend pas de nom vide (garde-rail
				// TestBijectionNoms). Si cela arrivait, la ligne degraderait en
				// « sans nom » — mais jamais en silence.
				slog.WarnContext(ctx, "backfill medailles: document raw_json non serialisable",
					"medaille", nom, "type_hint", evs[i].TypeHint,
					"medal_type", evs[i].MedalType, "id", ligne.id, "err", err)
				bilan.EventsSansNom++
				out = append(out, corr)
				continue
			}
			corr.rawJSON = &raw
			bilan.EventsIdentifies++
			out = append(out, corr)
		}
	}
	// Le parcours d une map n a pas d ordre : on rend les corrections par `id`
	// croissant pour que la passe ecrive — et se relise — de facon reproductible.
	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	return out
}

// matchsMedaillesSansIdentite liste TOUS les matchs auxquels il manque au moins
// une identite de medaille, dans un ordre stable.
//
// AUCUN `LIMIT` ICI, ET C EST LE POINT. Borner la requete revenait a prendre les N
// premiers identifiants dans l ordre alphabetique : si aucun de ceux-la n a de film
// en cache, la passe ne traitait RIEN et un second run reprenait exactement les
// memes. Le plafond compte desormais les matchs TRAITES (cf. BackfillIdentiteMedailles) :
// les sans-film n epuisent plus le lot.
func matchsMedaillesSansIdentite(ctx context.Context, db *sql.DB) ([]string, error) {
	const q = `SELECT DISTINCT match_id FROM highlight_events
	           WHERE event_type = ? AND raw_json IS NULL
	           ORDER BY match_id`
	rows, err := db.QueryContext(ctx, q, analysis.EventTypeMedal)
	if err != nil {
		return nil, fmt.Errorf("backfill medailles: selection des matchs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("backfill medailles: selection des matchs (scan): %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// medaillesDuMatch lit TOUTES les lignes medal d un match, dans l ordre
// d insertion (`id` croissant), en marquant celles qui ont deja une identite.
func medaillesDuMatch(ctx context.Context, db *sql.DB, matchID string) ([]evenementBase, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, COALESCE(xuid, ''), COALESCE(time_ms, 0), raw_json IS NOT NULL
		FROM highlight_events
		WHERE match_id = ? AND event_type = ?
		ORDER BY id`, matchID, analysis.EventTypeMedal)
	if err != nil {
		return nil, fmt.Errorf("backfill medailles: events de %s: %w", matchID, err)
	}
	defer func() { _ = rows.Close() }()
	var out []evenementBase
	for rows.Next() {
		var e evenementBase
		if err := rows.Scan(&e.id, &e.xuid, &e.timeMS, &e.dejaIdentifie); err != nil {
			return nil, fmt.Errorf("backfill medailles: events de %s (scan): %w", matchID, err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ecrireCorrections applique les corrections d UN match, dans UNE transaction :
// un match est entierement rattrape ou pas touche.
//
// UPDATE ligne a ligne, jamais la forme bulk `UPDATE ... FROM (VALUES ...)` : c est
// elle qui declenche le bug ART #23046 en touchant N entrees d index en un
// statement (garde-rail sync/no_art_patterns_test.go). Serialisee et mono-process,
// la forme ligne a ligne est celle que la doctrine autorise.
func ecrireCorrections(ctx context.Context, db *sql.DB, matchID string, corrections []correction) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("backfill medailles: transaction de %s: %w", matchID, err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, c := range corrections {
		if _, err := tx.ExecContext(ctx,
			`UPDATE highlight_events SET raw_json = ?, type_hint = ? WHERE id = ?`,
			c.rawJSON, c.typeHint, c.id,
		); err != nil {
			return fmt.Errorf("backfill medailles: UPDATE %s/id=%d: %w", matchID, c.id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("backfill medailles: commit de %s: %w", matchID, err)
	}
	return nil
}
