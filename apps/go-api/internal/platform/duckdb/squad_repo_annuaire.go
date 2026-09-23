// Package duckdb — squad_repo_annuaire.go : L'ANNUAIRE DES NOMS des lectures Escouade (Q29
// LoadTopTeammates, Q32 LoadImpactEvents, Q32b LoadMainTeamParticipants).
//
// # LE DEFAUT SUPPRIME (lot perf L2, 2026-09-23)
//
// Les trois lectures portaient `LEFT JOIN v_gamertag_lookup`. Aucun filtre ne se pousse dans
// cette vue (agregats en FULL OUTER JOIN) : elle etait materialisee EN ENTIER a chaque
// jointure, 3 s par evaluation sur la base de production et six evaluations par page
// Escouade (Q32 etait lue quatre fois). Mesure : Q29 3,0-3,3 s AVEC la jointure, 32 ms sans.
//
// # CE QUI NE CHANGE PAS : LA SOURCE DES NOMS
//
// Chaque lecture charge son annuaire UNE fois : pour les xuids qu'elle a rencontres, sur les
// matchs qu'elle lit, les noms que chaque niveau de la vue canonique leur donnerait — alias,
// participants (MAX), puis kill-feed (MAX) pour les seuls xuids que les deux premiers
// niveaux laissent sans nom. Le SQL de ces niveaux vient d'`analysis` (celui du kill-feed est
// la jambe MEME de la vue, un seul generateur) et la cascade s'applique en Go
// (analysis.AnnuaireGamertags.Resolve) : bot connu, alias, participant, kill-feed, puis
// « Joueur #### ». Les consommateurs de ces lectures hors Escouade (SquadService legacy,
// coequipiers de session de l'accueil) recoivent donc les memes noms qu'avant.
//
// # LES ECARTS POSSIBLES AVEC LA VUE, NOMMES
//
// La vue prend le MAX des participants et du kill-feed sur TOUTE la base ; l'annuaire, sur les
// matchs de la lecture (D2.1) — et, pour le kill-feed, sur ceux ou la lecture a RENCONTRE le
// xuid quand ses lignes portent leur match (Q32, Q32b ; cf. matchsKillFeed). Les deux ne
// different que pour un xuid SANS alias, dont le nom VARIE d'un match a l'autre — mesure sur la
// copie de production du 2026-09-23 : zero ecart sur les 50 du top, les 796 allies et les
// 2 393 joueurs de lobby des 604 matchs « avec amis » de JGtm, ni sur les 119 + 29 xuids que
// seul le kill-feed nomme parmi les 579 matchs JGtm + Madina97294. Et un xuid de bot absent
// de toutes les sources prend le nom du bot la ou la jointure rendait le libelle masque
// (aucun bot dans highlight_events, seule lecture ou ce cas pouvait se produire).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis"
)

// accesLigne : ce que nommerLignes lit et ecrit sur une ligne de lecture.
type accesLigne[T any] struct {
	xuid func(T) string
	// match : le match de la ligne. nil quand la lecture est AGREGEE (Q29 : une ligne par
	// coequipier, tous matchs confondus) — la jambe kill-feed lit alors tous ses matchs.
	match  func(T) string
	nommer func(*T, string)
}

// nommerLignes pose le nom d'affichage de chaque ligne d'une lecture : collecte ses xuids,
// charge l'annuaire UNE fois sur les matchs de la lecture, applique la cascade. C'est le seul
// chemin par lequel Q29, Q32 et Q32b nomment leurs lignes.
func nommerLignes[T any](ctx context.Context, db *sql.DB, matchIDs []string, lignes []T, acces accesLigne[T]) error {
	if len(lignes) == 0 {
		return nil
	}
	lecture := lectureANommer{matchIDs: matchIDs, xuids: make([]string, len(lignes))}
	if acces.match != nil {
		lecture.matchs = make([]string, len(lignes))
	}
	for i := range lignes {
		lecture.xuids[i] = acces.xuid(lignes[i])
		if acces.match != nil {
			lecture.matchs[i] = acces.match(lignes[i])
		}
	}
	noms, err := annuaireDeLecture(ctx, db, lecture)
	if err != nil {
		return err
	}
	for i := range lignes {
		acces.nommer(&lignes[i], noms.Resolve(lecture.xuids[i]))
	}
	return nil
}

// lectureANommer : les xuids d'une lecture, le match de chacun quand la ligne le porte, et
// les matchs lus.
type lectureANommer struct {
	xuids    []string
	matchs   []string // matchs[i] = match de xuids[i] ; nil = inconnu (lecture agregee)
	matchIDs []string
}

// matchsKillFeed rend les matchs ou la jambe kill-feed cherche les xuids restants : ceux ou la
// lecture les a RENCONTRES. Les lignes portent leur match (Q32, Q32b) : lu sur les lignes. La
// lecture est agregee (Q29, une ligne par coequipier) : lu dans match_participants, parmi les
// matchs de la lecture — une requete sur une TABLE, qui pousse ses deux predicats.
//
// Mesure sur la copie de production (2026-09-23, JGtm + Madina97294, 579 matchs) : 119 xuids de
// lobby sans alias ni participant nommes, rencontres dans 34 matchs — memes 118 noms qu'en
// lisant les 579 matchs, 86 ms au lieu de 452 (le cout de la jambe suit la liste de matchs).
func (l lectureANommer) matchsKillFeed(ctx context.Context, db *sql.DB, restants []string) ([]string, error) {
	if l.matchs == nil {
		return matchsDesParticipants(ctx, db, restants, l.matchIDs)
	}
	return l.matchsDesLignes(restants), nil
}

// matchsDesParticipants : parmi `matchIDs`, les matchs ou au moins un des xuids a joue.
func matchsDesParticipants(ctx context.Context, db *sql.DB, xuids, matchIDs []string) ([]string, error) {
	q := "SELECT DISTINCT match_id FROM match_participants WHERE xuid IN (" + Placeholders(len(xuids)) +
		") AND match_id IN (" + Placeholders(len(matchIDs)) + ")"
	args := append(ToAnySlice(xuids), ToAnySlice(matchIDs)...)
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("annuaire (matchs des restants): %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("annuaire (matchs des restants) scan: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// matchsDesLignes : les matchs des lignes dont le xuid est parmi `restants`.
func (l lectureANommer) matchsDesLignes(restants []string) []string {
	cherches := make(map[string]struct{}, len(restants))
	for _, x := range restants {
		cherches[x] = struct{}{}
	}
	vus := map[string]struct{}{}
	var out []string
	for i, x := range l.xuids {
		if _, ok := cherches[x]; !ok {
			continue
		}
		if _, deja := vus[l.matchs[i]]; !deja {
			vus[l.matchs[i]] = struct{}{}
			out = append(out, l.matchs[i])
		}
	}
	return out
}

// annuaireDeLecture charge l'annuaire des xuids d'une lecture, restreint a ses matchs.
//
// `db` est la connexion shared DEJA acquise par la lecture (aucun second Get sous timeout).
// Sans xuid a nommer ou sans match : annuaire vide (Resolve rend alors le nom des bots et le
// libelle masque, rien d'autre — un appelant qui lit des lignes a toujours des matchs).
func annuaireDeLecture(ctx context.Context, db *sql.DB, lecture lectureANommer) (analysis.AnnuaireGamertags, error) {
	a := analysis.AnnuaireGamertags{
		Alias:        map[string]string{},
		Participants: map[string]string{},
		KillFeed:     map[string]string{},
	}
	cherches := xuidsANommer(lecture.xuids)
	if len(cherches) == 0 || len(lecture.matchIDs) == 0 {
		return a, nil
	}
	debut := time.Now()
	if err := lireAliasEtParticipants(ctx, db, cherches, lecture.matchIDs, &a); err != nil {
		return a, err
	}
	restants := xuidsSansAliasNiParticipant(cherches, a)
	var matchsKF []string
	if len(restants) > 0 {
		var err error
		if matchsKF, err = lecture.matchsKillFeed(ctx, db, restants); err != nil {
			return a, err
		}
		if len(matchsKF) > 0 {
			if err := lireKillFeed(ctx, db, restants, matchsKF, &a); err != nil {
				return a, err
			}
		}
	}
	slog.DebugContext(ctx, "squad_annuaire",
		"xuids", len(cherches), "sans_alias_ni_participant", len(restants),
		"nommes_par_kill_feed", len(a.KillFeed), "matchs", len(lecture.matchIDs),
		"matchs_kill_feed", len(matchsKF), "duration_ms", time.Since(debut).Milliseconds())
	return a, nil
}

// xuidsANommer rend les xuids distincts a chercher en base. Les bots en sont exclus : leur nom
// se deduit du xuid seul (niveau 1 de la vue, avant tout alias).
func xuidsANommer(xuids []string) []string {
	vus := make(map[string]struct{}, len(xuids))
	out := make([]string, 0, len(xuids))
	for _, x := range xuids {
		if analysis.IsBot(x) {
			continue
		}
		if _, deja := vus[x]; deja {
			continue
		}
		vus[x] = struct{}{}
		out = append(out, x)
	}
	return out
}

// xuidsSansAliasNiParticipant : les xuids que seul le kill-feed pourrait encore nommer.
func xuidsSansAliasNiParticipant(xuids []string, a analysis.AnnuaireGamertags) []string {
	var out []string
	for _, x := range xuids {
		if a.Alias[x] == "" && a.Participants[x] == "" {
			out = append(out, x)
		}
	}
	return out
}

// lireAliasEtParticipants lit les niveaux 2 et 3 (une requete).
func lireAliasEtParticipants(
	ctx context.Context, db *sql.DB, xuids, matchIDs []string, a *analysis.AnnuaireGamertags,
) error {
	q := analysis.AnnuaireNomsSQL(Placeholders(len(xuids)), Placeholders(len(matchIDs)))
	args := make([]any, 0, 2*len(xuids)+len(matchIDs))
	args = append(args, ToAnySlice(xuids)...)
	args = append(args, ToAnySlice(xuids)...)
	args = append(args, ToAnySlice(matchIDs)...)
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("annuaire (alias, participants): %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var niveau, xuid, gamertag string
		if err := rows.Scan(&niveau, &xuid, &gamertag); err != nil {
			return fmt.Errorf("annuaire (alias, participants) scan: %w", err)
		}
		switch niveau {
		case analysis.AnnuaireNiveauAlias:
			a.Alias[xuid] = gamertag
		case analysis.AnnuaireNiveauParticipant:
			a.Participants[xuid] = gamertag
		}
	}
	return rows.Err()
}

// lireKillFeed lit le niveau 4 pour les xuids restants (une requete, la jambe de la vue).
func lireKillFeed(
	ctx context.Context, db *sql.DB, xuids, matchIDs []string, a *analysis.AnnuaireGamertags,
) error {
	q := analysis.AnnuaireKillFeedSQL(Placeholders(len(xuids)), Placeholders(len(matchIDs)))
	args := make([]any, 0, analysis.AnnuaireKillFeedJambes*len(matchIDs)+len(xuids))
	for range analysis.AnnuaireKillFeedJambes {
		args = append(args, ToAnySlice(matchIDs)...)
	}
	args = append(args, ToAnySlice(xuids)...)
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("annuaire (kill-feed): %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var xuid, gamertag string
		if err := rows.Scan(&xuid, &gamertag); err != nil {
			return fmt.Errorf("annuaire (kill-feed) scan: %w", err)
		}
		a.KillFeed[xuid] = gamertag
	}
	return rows.Err()
}
