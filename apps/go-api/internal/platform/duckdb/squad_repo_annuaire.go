// Package duckdb — squad_repo_annuaire.go : L'ANNUAIRE DES NOMS des lectures qui ne joignent plus
// v_gamertag_lookup. Escouade (lot perf L2) : Q29 LoadTopTeammates, Q32 LoadImpactEvents, Q32b
// LoadMainTeamParticipants. Carriere (lot perf L7) : Q26 GetTopEncountersGlobal, Q27 GetRivals,
// Q10 GetEncounters ; Comparer : GetLocalStats — le fichier garde son nom d'origine, et sa ligne
// DEBUG le sien (`squad_annuaire`), pour toutes.
//
// # LE DEFAUT SUPPRIME (lots perf L2 et L7, 2026-09-23)
//
// Ces lectures portaient `LEFT JOIN v_gamertag_lookup`. Aucun filtre ne se pousse dans
// cette vue (agregats en FULL OUTER JOIN) : elle etait materialisee EN ENTIER a chaque
// jointure, 3 s par evaluation sur la base de production. Escouade : six evaluations par page
// (Q32 etait lue quatre fois), Q29 3,0-3,3 s AVEC la jointure, 32 ms sans. Carriere : trois par
// ouverture (Q26, Q27 deux fois), rencontres 10,7 s et rivaux 10,1 s a la mesure de campagne.
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
// Les lectures AGREGEES de la Carriere et de Comparer (une ligne par joueur, tous matchs
// confondus) lisent l'annuaire sur les matchs de l'historique du joueur (QMatchsDuJoueurTpl ; pour
// Comparer, celui du joueur compare, que la lecture agrege en entier) ; les rivaux, venus du
// kill-feed, portent en plus UN match du duel par ligne (`match_rencontre`) : la jambe kill-feed y
// trouve un adversaire qu'aucune ligne participant ne connait.
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
//
// Carriere (L7), meme copie, sur TOUS les joueurs croises ou affrontes par les cinq joueurs
// suivis (58 353 couples joueur / croise, pas seulement les lignes servies) : zero ecart pour
// quatre d'entre eux ; 8 chez Nuzzles, « Joueur #### » la ou la vue trouve un nom HORS de ses
// 7 190 matchs (ses participants n'ont pas de gamertag), aucun dans une ligne servie. Les 269
// rivaux sans alias ni nom de participant (JGtm, Madina97294, Chocoboflor) y ont le nom de la vue.
// Comparer : zero ecart sur les 53 061 xuids de la base (la lecture couvre tout leur historique).
//
// # LA PORTEE BASE (lot A du plan perf « lectures par perimetre », 2026-09-26, ADR 0036 I1)
//
// Lecteurs : vue match Q12 GetMatchScoreboard, Q21 GetMatchEvents, Q23 GetMatchEncounters
// (match_view_repo_scoreboard.go, match_view_repo_noms.go ; Q23b n'a plus de nom du tout),
// GamertagRepo.ResolveGamertags (evenements de match), Relations Q28 GetRelations et heatmap Q29
// GetRelationsHeatmap. Restreint aux matchs de la lecture, l'annuaire perdait des noms que la vue
// trouve AILLEURS (11 couples (match, joueur) de la vue match, 2 lignes de Relations). En portee
// base (nommerLignesPorteeBase), alias et participants se lisent sur toute la base (le MAX de la
// vue, predicats sur des TABLES), le kill-feed d'abord sur les matchs de la lecture, puis sur toute
// la base pour les seuls xuids encore sans nom (lireKillFeedBase).
//
// Mesure sur la copie de production du 2026-09-23 (2 threads, 512 Mo) : zero ecart de nom sur les
// 93 000 lignes de Q12 et les 1 311 594 events de Q21 (tous les matchs), les 2 625 lignes de Q23 /
// Q23b et les 1 853 xuids de ResolveGamertags (279 couples match / joueur), les 12 289 lignes de
// Relations et les 1 265 de la heatmap (cinq joueurs suivis, periode entiere et 30 matchs). Sans le
// dernier repli : exactement les 11 + 2 noms perdus. Cout : l'annuaire d'un match coute moins de
// 10 ms ; le repli toute la base evalue la fenetre `_latest` du journal canonique entiere (3 a 7 s,
// davantage sous charge) — il part pour 23 des 1 160 matchs de JGtm, 4 688 des 7 190 de Nuzzles et
// les Relations de Nuzzles (journal P2 du plan).
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
// chemin par lequel les lectures de l'en-tete nomment leurs lignes.
func nommerLignes[T any](ctx context.Context, db *sql.DB, matchIDs []string, lignes []T, acces accesLigne[T]) error {
	return nommerLignesSelon(ctx, db, lectureANommer{matchIDs: matchIDs}, lignes, acces)
}

// nommerLignesPorteeBase : nommerLignes en « portée base » (lot A, DA.3 — vue match, Relations) :
// même collecte, même cascade, l'annuaire lu comme la vue (cf. lectureANommer.porteeBase).
func nommerLignesPorteeBase[T any](ctx context.Context, db *sql.DB, matchIDs []string, lignes []T, acces accesLigne[T]) error {
	return nommerLignesSelon(ctx, db, lectureANommer{matchIDs: matchIDs, porteeBase: true}, lignes, acces)
}

// nommerLignesSelon : le chemin commun des deux portées. `lecture` porte les matchs lus et la
// portée ; les xuids (et leur match) sont collectés ici, sur les lignes.
func nommerLignesSelon[T any](ctx context.Context, db *sql.DB, lecture lectureANommer, lignes []T, acces accesLigne[T]) error {
	if len(lignes) == 0 {
		return nil
	}
	lecture.xuids = make([]string, len(lignes))
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
	// porteeBase : l'annuaire lit les alias et les participants sur TOUTE la base (le MAX de la
	// vue), puis le kill-feed des matchs de la lecture, puis celui de toute la base pour les
	// seuls xuids encore sans nom — la semantique de la vue, ecarts (i) a (iii) de DA.3 / DA.4
	// mis a part. Faux : portee de la lecture (lots L2, L7), inchangee.
	porteeBase bool
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
// Sans xuid a nommer, ou sans match en portee de la lecture : annuaire vide (Resolve rend alors
// le nom des bots et le libelle masque, rien d'autre — un appelant qui lit des lignes a toujours
// des matchs). En portee base, sans match : alias, participants et kill-feed de toute la base.
func annuaireDeLecture(ctx context.Context, db *sql.DB, lecture lectureANommer) (analysis.AnnuaireGamertags, error) {
	a := analysis.AnnuaireGamertags{
		Alias:        map[string]string{},
		Participants: map[string]string{},
		KillFeed:     map[string]string{},
	}
	cherches := xuidsANommer(lecture.xuids)
	if len(cherches) == 0 || (len(lecture.matchIDs) == 0 && !lecture.porteeBase) {
		return a, nil
	}
	debut := time.Now()
	if err := lireAliasEtParticipants(ctx, db, cherches, lecture, &a); err != nil {
		return a, err
	}
	restants := xuidsSansNom(cherches, a)
	matchsKF, err := lireKillFeedDeLaLecture(ctx, db, lecture, restants, &a)
	if err != nil {
		return a, err
	}
	var horsLecture []string
	if lecture.porteeBase {
		if horsLecture = xuidsSansNom(restants, a); len(horsLecture) > 0 {
			if err := lireKillFeedBase(ctx, db, horsLecture, &a); err != nil {
				return a, err
			}
		}
	}
	slog.DebugContext(ctx, "squad_annuaire",
		"xuids", len(cherches), "sans_alias_ni_participant", len(restants),
		"nommes_par_kill_feed", len(a.KillFeed), "matchs", len(lecture.matchIDs),
		"matchs_kill_feed", len(matchsKF), "portee_base", lecture.porteeBase,
		"kill_feed_toute_la_base", len(horsLecture), "duration_ms", time.Since(debut).Milliseconds())
	return a, nil
}

// lireKillFeedDeLaLecture lit le niveau 4 sur les matchs de la lecture pour les xuids restants
// (ceux ou la lecture les a rencontres, cf. matchsKillFeed) ; rend ces matchs.
func lireKillFeedDeLaLecture(
	ctx context.Context, db *sql.DB, lecture lectureANommer, restants []string, a *analysis.AnnuaireGamertags,
) ([]string, error) {
	if len(restants) == 0 || len(lecture.matchIDs) == 0 {
		return nil, nil
	}
	matchsKF, err := lecture.matchsKillFeed(ctx, db, restants)
	if err != nil || len(matchsKF) == 0 {
		return matchsKF, err
	}
	return matchsKF, lireKillFeed(ctx, db, restants, matchsKF, a)
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

// xuidsSansNom : les xuids qu'aucun niveau lu jusqu'ici ne nomme (les bots n'y sont jamais :
// xuidsANommer les a ecartes).
func xuidsSansNom(xuids []string, a analysis.AnnuaireGamertags) []string {
	var out []string
	for _, x := range xuids {
		if !a.Nomme(x) {
			out = append(out, x)
		}
	}
	return out
}

// lireAliasEtParticipants lit les niveaux 2 et 3 (une requete) : participants des matchs de la
// lecture, ou de toute la base en portee base.
func lireAliasEtParticipants(
	ctx context.Context, db *sql.DB, xuids []string, lecture lectureANommer, a *analysis.AnnuaireGamertags,
) error {
	q := analysis.AnnuaireNomsBaseSQL(Placeholders(len(xuids)))
	args := make([]any, 0, 2*len(xuids)+len(lecture.matchIDs))
	args = append(args, ToAnySlice(xuids)...)
	args = append(args, ToAnySlice(xuids)...)
	if !lecture.porteeBase {
		q = analysis.AnnuaireNomsSQL(Placeholders(len(xuids)), Placeholders(len(lecture.matchIDs)))
		args = append(args, ToAnySlice(lecture.matchIDs)...)
	}
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

// lireKillFeedBase lit le niveau 4 sur TOUTE la base pour les xuids que rien d'autre ne nomme
// (portee base, DA.3) : la jambe de la vue sans restriction de match — la fenetre `_latest` du
// journal canonique y est evaluee en entier (cout mesure au journal P2 du plan).
func lireKillFeedBase(ctx context.Context, db *sql.DB, xuids []string, a *analysis.AnnuaireGamertags) error {
	rows, err := db.QueryContext(ctx, analysis.AnnuaireKillFeedBaseSQL(Placeholders(len(xuids))), ToAnySlice(xuids)...)
	if err != nil {
		return fmt.Errorf("annuaire (kill-feed, toute la base): %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var xuid, gamertag string
		if err := rows.Scan(&xuid, &gamertag); err != nil {
			return fmt.Errorf("annuaire (kill-feed, toute la base) scan: %w", err)
		}
		a.KillFeed[xuid] = gamertag
	}
	return rows.Err()
}
