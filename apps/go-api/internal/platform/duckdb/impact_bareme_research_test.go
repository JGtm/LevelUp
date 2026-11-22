//go:build research

package duckdb

// impact_bareme_research_test.go — TÉMOIN DU BARÈME de la matrice d'impact de l'Escouade : les
// points par match et par joueur d'une escouade, sous le barème en vigueur (squadimpact.Weight) et
// sous un barème candidat (table `baremeCandidat` ci-dessous), avec le rôle « Os de verre ».
//
// Les entrées sont celles du service (`teammates.rolesDesMatchs`) : Q32 (événements, repli des
// frags reconstitués compris), Q32b (équipe alliée complète du joueur principal), Q32e (journal
// des morts, badge « Voleur »), puis squadimpact.RolesOfMatch. Écarts avec le service :
//   - la population est « tous les matchs où le joueur principal partage son équipe avec au moins
//     un des coéquipiers retenus », sans filtre de page (dates, modes, sessions) ni population
//     « escouade complète » ; elle est lue comme UN seul groupe Q32 ;
//   - le T0 du match n'est pas retranché (les rôles n'en dépendent pas : constante par match) ;
//   - les coéquipiers retenus sont ceux qui partagent le plus de matchs avec le joueur principal.
//
// « Os de verre » : par match mesuré (fichier IMPACT_BRIS, sortie de TestMedaillesBouclierBris), le
// membre de l'équipe alliée au plus grand nombre de bris ; à égalité : moins de recharges, puis
// xuid croissant ; le rôle ne compte que s'il tombe sur un membre de l'escouade.
//
// LECTURE SEULE, SUR UNE COPIE de la base partagée.
//
//	IMPACT_SHARED=<copie shared_matches_v2.duckdb> IMPACT_JOUEUR=<gamertag> IMPACT_BRIS=<tsv> \
//	  go test -tags research -count=1 -run '^TestImpactBareme$' -v ./internal/platform/duckdb/

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/squadimpact"
)

const (
	impactRoleOsDeVerre  = "glass_jaw"
	impactPoidsOsDeVerre = -0.5
	impactNbCoequipiers  = 3
)

// baremeCandidat : les poids qui changent ; un rôle absent garde squadimpact.Weight.
var baremeCandidat = map[string]float64{
	squadimpact.RoleFirstGroupDeath: -0.5,
	squadimpact.RoleLastGroupKill:   -0.5,
	squadimpact.RoleKamikaze:        -0.5,
}

type impactBris struct{ bris, recharges int }

type impactCumul struct {
	matchs, matchsMesures       int
	ancien, candidat, osDeVerre float64
	roles                       map[string]int
}

func TestImpactBareme(t *testing.T) {
	shared, joueur := os.Getenv("IMPACT_SHARED"), os.Getenv("IMPACT_JOUEUR")
	if shared == "" || joueur == "" {
		t.Skip("IMPACT_SHARED et IMPACT_JOUEUR requis")
	}
	db, err := sql.Open("duckdb", shared+"?access_mode=read_only")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()

	var main string
	if err := db.QueryRow(`SELECT xuid FROM match_participants WHERE lower(gamertag) = lower(?)
		GROUP BY xuid ORDER BY count(*) DESC LIMIT 1`, joueur).Scan(&main); err != nil {
		t.Fatalf("xuid de %s : %v", joueur, err)
	}
	squad := map[string]string{main: joueur}
	ordre := []string{main}
	rows, err := db.Query(`SELECT p.xuid, arg_max(p.gamertag, p.match_id), count(*) AS n
		FROM match_participants p JOIN match_participants m
		  ON m.match_id = p.match_id AND m.xuid = ? AND p.team_id = m.team_id
		WHERE p.xuid <> ? AND p.xuid IS NOT NULL AND p.xuid <> ''
		GROUP BY p.xuid ORDER BY n DESC, p.xuid LIMIT ?`, main, main, impactNbCoequipiers)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var x, gt string
		var n int
		if err := rows.Scan(&x, &gt, &n); err != nil {
			t.Fatal(err)
		}
		squad[x] = gt
		ordre = append(ordre, x)
		t.Logf("coequipier retenu : %s (%d matchs dans l equipe de %s)", gt, n, joueur)
	}
	rows.Close()

	// Population : matchs où le principal partage son équipe avec au moins un coéquipier retenu.
	args := []any{main}
	for _, x := range ordre[1:] {
		args = append(args, x)
	}
	rows, err = db.Query(`SELECT DISTINCT p.match_id FROM match_participants p JOIN match_participants m
		  ON m.match_id = p.match_id AND m.xuid = ? AND p.team_id = m.team_id
		WHERE p.xuid IN (`+placeholderList(len(ordre)-1)+`) ORDER BY 1`, args...)
	if err != nil {
		t.Fatal(err)
	}
	var matchIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		matchIDs = append(matchIDs, id)
	}
	rows.Close()
	t.Logf("population : %d matchs", len(matchIDs))

	events := impactEvenements(ctx, t, db, matchIDs)
	allies := impactAllies(ctx, t, db, main, matchIDs)
	voleurs := impactVoleurs(ctx, t, db, matchIDs, squad)
	bris := impactLireBris(t, os.Getenv("IMPACT_BRIS"))

	cumul := map[string]*impactCumul{}
	for _, x := range ordre {
		cumul[x] = &impactCumul{roles: map[string]int{}}
	}
	complet := map[string]*impactCumul{}
	for _, x := range ordre {
		complet[x] = &impactCumul{roles: map[string]int{}}
	}
	matchsMesures, osAttribues, osHorsEscouade := 0, 0, 0
	for _, mid := range matchIDs {
		presents := 0
		for _, a := range allies[mid] {
			if _, ok := squad[a.XUID]; ok {
				presents++
			}
		}
		cibles := []map[string]*impactCumul{cumul}
		if presents == len(ordre) {
			cibles = append(cibles, complet)
		}
		verre, mesure := impactOsDeVerre(allies[mid], bris[mid])
		if mesure {
			matchsMesures++
		}
		if verre != "" {
			if _, ok := squad[verre]; ok {
				osAttribues++
			} else {
				osHorsEscouade++
			}
		}
		attrs := squadimpact.RolesOfMatch(squadimpact.MatchInput{
			Events: events[mid], Allies: allies[mid], Thief: voleurs[mid],
		}, impactIdentite(squad))
		for _, c := range cibles {
			for _, a := range allies[mid] {
				if j := c[a.XUID]; j != nil {
					j.matchs++
					if mesure {
						j.matchsMesures++
					}
				}
			}
			for _, a := range attrs {
				j := c[a.Player]
				w, _ := squadimpact.Weight(a.Role)
				j.ancien += w
				if nw, ok := baremeCandidat[a.Role]; ok {
					w = nw
				}
				j.candidat += w
				j.roles[a.Role]++
			}
			if j := c[verre]; j != nil {
				j.osDeVerre += impactPoidsOsDeVerre
				j.roles[impactRoleOsDeVerre]++
			}
		}
	}
	t.Logf("matchs avec mesure de bouclier : %d sur %d ; Os de verre sur l escouade : %d ; sur un allie hors escouade : %d",
		matchsMesures, len(matchIDs), osAttribues, osHorsEscouade)
	for _, bloc := range []struct {
		nom string
		c   map[string]*impactCumul
	}{{"au moins un coequipier retenu dans l equipe", cumul}, {"escouade complete dans l equipe", complet}} {
		t.Logf("=== population : %s", bloc.nom)
		for _, x := range ordre {
			j := bloc.c[x]
			n := float64(max(j.matchs, 1))
			t.Logf("%-18s matchs %4d (mesures %4d) | points par match : ancien %+.3f ; bareme candidat seul %+.3f ; candidat + Os de verre %+.3f | totaux %+.1f / %+.1f / %+.1f",
				squad[x], j.matchs, j.matchsMesures, j.ancien/n, j.candidat/n, (j.candidat+j.osDeVerre)/n,
				j.ancien, j.candidat, j.candidat+j.osDeVerre)
			var parts []string
			for _, r := range append(squadimpact.RoleOrder(), impactRoleOsDeVerre) {
				parts = append(parts, fmt.Sprintf("%s=%d", r, j.roles[r]))
			}
			t.Logf("    roles : %s", strings.Join(parts, " "))
		}
	}
}

// impactIdentite : RolesOfMatch nomme la ligne par la valeur de squad ; ici la ligne est le xuid.
func impactIdentite(squad map[string]string) map[string]string {
	out := make(map[string]string, len(squad))
	for x := range squad {
		out[x] = x
	}
	return out
}

func impactEvenements(ctx context.Context, t *testing.T, db *sql.DB, matchIDs []string) map[string][]analysis.ImpactEvent {
	t.Helper()
	args := appendStrArgs(nil, matchIDs)
	lignes, err := scanImpactEvents(ctx, db, fmt.Sprintf(Q32SquadImpactEventsTemplate, placeholderList(len(matchIDs))), args)
	if err != nil {
		t.Fatal(err)
	}
	if besoin := analysis.ImpactMatchesNeedingKVFallback(lignes, [][]string{matchIDs}); len(besoin) > 0 {
		t.Logf("repli des frags reconstitues : %d matchs", len(besoin))
		kv, err := (&SquadRepo{}).loadKVPairsOn(ctx, db, besoin)
		if err != nil {
			t.Fatal(err)
		}
		lignes = mergeImpactRowsByTime(lignes, synthesizeImpactRowsFromKVPairs(kv))
	}
	out := map[string][]analysis.ImpactEvent{}
	for _, r := range lignes {
		out[r.MatchID] = append(out[r.MatchID], analysis.ImpactEvent{TimeMS: r.TimeMS, EventType: r.EventType, ActorXUID: r.XUID})
	}
	return out
}

func impactAllies(ctx context.Context, t *testing.T, db *sql.DB, main string, matchIDs []string) map[string][]analysis.ParticipantSnap {
	t.Helper()
	args := appendStrArgs([]any{main}, matchIDs)
	lignes, err := scanMainTeamParticipants(ctx, db, fmt.Sprintf(Q32bMainTeamParticipantsTemplate, placeholderList(len(matchIDs))), args)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][]analysis.ParticipantSnap{}
	for _, a := range lignes {
		out[a.MatchID] = append(out[a.MatchID], analysis.ParticipantSnap{
			XUID: a.XUID, Outcome: a.Outcome, Kills: a.Kills, Deaths: a.Deaths, Assists: a.Assists,
		})
	}
	return out
}

func impactVoleurs(ctx context.Context, t *testing.T, db *sql.DB, matchIDs []string, squad map[string]string) map[string]*analysis.ImpactBadge {
	t.Helper()
	amis := map[string]bool{}
	var xuids []string
	for x := range squad {
		amis[x] = true
		xuids = append(xuids, x)
	}
	sort.Strings(xuids)
	q, args := buildSquadKillLogQuery(matchIDs, xuids)
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	lignes, err := scanSquadKillLog(rows)
	if err != nil {
		t.Fatal(err)
	}
	parMatch := map[string][]analysis.KillLogEntry{}
	for _, r := range lignes {
		parMatch[r.MatchID] = append(parMatch[r.MatchID], analysis.KillLogEntry{
			TimeMS: r.TimeMS, KillerXUID: r.KillerXUID, VictimXUID: r.VictimXUID,
			AssistXUID: r.AssistXUID, KillerDamagePct: r.KillerDamagePct,
		})
	}
	out := map[string]*analysis.ImpactBadge{}
	for mid, e := range parMatch {
		if b := analysis.ComputeThiefBadge(e, amis); b != nil {
			out[mid] = b
		}
	}
	return out
}

func impactLireBris(t *testing.T, chemin string) map[string]map[string]impactBris {
	t.Helper()
	out := map[string]map[string]impactBris{}
	if chemin == "" {
		t.Log("IMPACT_BRIS non pose : Os de verre non mesure")
		return out
	}
	f, err := os.Open(chemin)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		c := strings.Split(sc.Text(), "\t")
		if len(c) != 4 {
			continue
		}
		b, _ := strconv.Atoi(c[2])
		r, _ := strconv.Atoi(c[3])
		if out[c[0]] == nil {
			out[c[0]] = map[string]impactBris{}
		}
		out[c[0]][c[1]] = impactBris{b, r}
	}
	return out
}

// impactOsDeVerre rend le xuid de l'allié au plus grand nombre de bris (vide si aucun bris), et si
// le match porte une mesure.
func impactOsDeVerre(allies []analysis.ParticipantSnap, bris map[string]impactBris) (string, bool) {
	if len(bris) == 0 {
		return "", false
	}
	best, bb := "", impactBris{}
	for _, a := range allies {
		b, ok := bris[a.XUID]
		if !ok || b.bris == 0 {
			continue
		}
		if best == "" || b.bris > bb.bris || (b.bris == bb.bris && (b.recharges < bb.recharges ||
			(b.recharges == bb.recharges && a.XUID < best))) {
			best, bb = a.XUID, b
		}
	}
	return best, true
}
