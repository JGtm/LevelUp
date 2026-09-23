package duckdb

// weapon_range_repo_fenetres_test.go — LA PORTÉE MESURÉE EST PAYÉE AU PÉRIMÈTRE, EN UNE
// LECTURE (lot L5a du plan perf, 2026-09-23).
//
// Ce que ces tests verrouillent, et pourquoi chacun peut échouer :
//
//  1. aucune fenêtre `_latest` (journal, positions, entames) ne voit plus que les lignes du
//     scope, pour les trois lectures du repo. Le `e.match_id IN (...)` ne traversait pas la
//     jointure jusqu'à la vue des positions : elle se calculait sur la table entière, sans
//     qu'aucun chiffre ne bouge (cf. fenetres_perimetre_helpers_test.go) ;
//  2. la lecture des DEUX côtés en une requête rend EXACTEMENT ce que rendaient les deux
//     lectures d'un côté — l'oracle est la composition d'un côté par la production elle-même
//     (buildWeaponRangeQuery, toujours servie à LoadMatchRangeKills), sur un corpus qui
//     exerce toutes les gardes (double frag, unanimité, publiable, bot, suicide, hors scope,
//     gamertag à deux xuid, gamertag inconnu, tous les joueurs).

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/killscope"
	"levelup/go-api/internal/port"
)

// classifieurFenetres range chaque source dans sa propre arme : la classification n'est pas
// l'objet de ces tests, la sélection des lignes l'est.
type classifieurFenetres struct{}

func (classifieurFenetres) KillSourceRegistryKey(tag uint32) (string, bool) {
	return fmt.Sprintf("arme_%d", tag), true
}

// fragMesure : une mort à poser dans le journal, avec sa source ; ses positions se posent à
// part (poserPositions), la même ligne dans `kill_positions` et `kill_openings`.
type fragMesure struct {
	match, tueur, victime string // victime vide = bot (NULL)
	ms                    int
	tag                   uint32
	publiable             bool
}

func poserFrag(t *testing.T, pdb *PlayerDB, f fragMesure) {
	t.Helper()
	var victime any
	if f.victime != "" {
		victime = f.victime
	}
	tacExec(t, pdb, `INSERT INTO match_kill_events
		(match_id, decode_pass, decoder_rev, publishable, time_ms,
		 victim_gamertag, victim_xuid, feed_killer_gamertag, feed_killer_xuid,
		 feed_present, assist_known, source_tag, source_category, read_path, read_origin)
		VALUES (?, 'pass_v1', 'rev_test', ?, ?, 'Victime', ?, 'Tueur', ?, TRUE, FALSE, ?, 'None', ?,
		        'credit-concordant')`,
		f.match, f.publiable, f.ms, victime, f.tueur, int64(f.tag), killscope.ReadPathFilmWalk)
}

// poserPositions place le frag (match, tueur, instant) dans les DEUX tables de positions.
// La position dépend de l'instant : chaque frag a sa propre distance.
func poserPositions(t *testing.T, pdb *PlayerDB, match, tueur string, ms int) {
	t.Helper()
	d := float64(ms) / 1000
	for _, table := range []string{"kill_positions", "kill_openings"} {
		tacExec(t, pdb, `INSERT INTO `+table+`
			(match_id, decode_pass, killer_xuid, time_ms, killer_x, killer_y, killer_z, victim_x, victim_y, victim_z)
			VALUES (?, 'pass_test', ?, ?, 0.0, 0.0, 1.0, ?, 0.0, 0.0)`,
			match, tueur, ms, d)
	}
}

// lignesMesurees rend une empreinte triée d'une lecture — le multiset qui fait foi. L'ordre
// de sortie d'une lecture n'est pas un contrat (agrégat par hachage, sans ORDER BY).
func lignesMesurees(ks []analysis.MeasuredKill) []string {
	out := make([]string, 0, len(ks))
	for _, k := range ks {
		out = append(out, fmt.Sprintf("%s|%s|%s|%d|%s|%.6f|%.6f",
			k.Side, k.MatchID, k.KillerXUID, k.TimeMS, k.WeaponKey, k.DistanceM, k.DeltaZ))
	}
	sort.Strings(out)
	return out
}

// deuxLecturesDAutrefois : l'ORACLE — un côté puis l'autre, chacun composé par
// buildWeaponRangeQuery et traduit par toMeasuredKills, comme loadBothSides le faisait avant
// le lot L5a.
func deuxLecturesDAutrefois(
	t *testing.T, repo *WeaponRangeRepo, f port.WeaponRangeFilters, table measuredPositionsTable,
) []analysis.MeasuredKill {
	t.Helper()
	ctx := context.Background()
	db, release, err := repo.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		t.Fatalf("lecteur partagé : %v", err)
	}
	defer release()
	out := make([]analysis.MeasuredKill, 0)
	for _, cote := range []struct {
		colonne string
		cote    analysis.Side
	}{{weaponRangeKillerColumn, analysis.SideKiller}, {weaponRangeVictimColumn, analysis.SideVictim}} {
		q, args := buildWeaponRangeQuery(table, cote.colonne, f)
		measured, err := queryMeasuredKills(ctx, db, q, args, "oracle")
		if err != nil {
			t.Fatalf("oracle %s : %v", cote.cote, err)
		}
		out = append(out, repo.toMeasuredKills(ctx, measured, cote.cote, "oracle")...)
	}
	return out
}

// TestWeaponRange_UneLecture_PariteAvecLesDeuxLectures : la lecture des deux côtés en une
// requête rend le MÊME multiset que les deux lectures d'un côté, pour les deux tables et
// quatre désignations du joueur.
func TestWeaponRange_UneLecture_PariteAvecLesDeuxLectures(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	const moi, moiBis, adv, tiers, bot = "x-moi", "x-moi-bis", "x-adv", "x-tiers", ""
	frags := []fragMesure{
		{"w01", moi, adv, 1000, 10, true},    // mon frag
		{"w01", adv, moi, 2000, 11, true},    // ma mort
		{"w01", tiers, adv, 3000, 12, true},  // ni l'un ni l'autre
		{"w01", moi, bot, 4000, 10, true},    // mon frag sur un bot (victime NULL)
		{"w01", moiBis, adv, 5000, 14, true}, // l'autre xuid du même gamertag
		{"w02", moi, moi, 1000, 13, true},    // je me tue : les DEUX côtés
		{"w02", moi, adv, 2000, 10, true},    // double frag, même arme : exclu
		{"w02", moi, tiers, 2000, 10, true},
		{"w02", moi, adv, 3000, 10, true}, // unanimité violée : exclu
		{"w02", moi, tiers, 3000, 11, true},
		{"w03", moi, adv, 1000, 10, false}, // non publiable : exclu
		{"w03", adv, moi, 2000, 15, true},
		{"w04", moi, adv, 1000, 10, true}, // hors scope
	}
	for _, f := range frags {
		poserFrag(t, pdb, f)
	}
	for _, p := range []struct {
		match, tueur string
		ms           int
	}{
		{"w01", moi, 1000}, {"w01", adv, 2000}, {"w01", tiers, 3000}, {"w01", moi, 4000},
		{"w01", moiBis, 5000}, {"w02", moi, 1000}, {"w02", moi, 2000}, {"w02", moi, 3000},
		{"w03", moi, 1000}, {"w03", adv, 2000}, {"w04", moi, 1000},
	} {
		poserPositions(t, pdb, p.match, p.tueur, p.ms)
	}
	for _, alias := range [][2]string{{moi, "Moi"}, {moiBis, "Moi"}, {adv, "Adversaire"}} {
		tacExec(t, pdb, `INSERT INTO xuid_aliases (xuid, gamertag) VALUES (?, ?)`, alias[0], alias[1])
	}

	repo := NewWeaponRangeRepo(pdb, classifieurFenetres{})
	scope := []string{"w01", "w02", "w03"}
	cas := []struct {
		nom    string
		f      port.WeaponRangeFilters
		lignes int // taille attendue sur la table des positions, pour que la parité ne soit pas vide
	}{
		{"xuid", port.WeaponRangeFilters{MatchIDs: scope, XUIDs: []string{moi}}, 6},
		{"gamertag a deux xuid", port.WeaponRangeFilters{MatchIDs: scope, Gamertag: "Moi"}, 7},
		{"gamertag inconnu", port.WeaponRangeFilters{MatchIDs: scope, Gamertag: "Personne"}, 0},
		{"tous les joueurs", port.WeaponRangeFilters{MatchIDs: scope, AllPlayers: true}, 14},
	}
	for _, c := range cas {
		for _, table := range []measuredPositionsTable{positionsAtKill, positionsAtOpening} {
			got, err := repo.loadBothSides(context.Background(), pdb.TitleSlug, c.f, table, "test")
			if err != nil {
				t.Fatalf("%s / %s : %v", c.nom, table, err)
			}
			want := lignesMesurees(deuxLecturesDAutrefois(t, repo, c.f, table))
			gotLignes := lignesMesurees(got)
			if fmt.Sprint(gotLignes) != fmt.Sprint(want) {
				t.Errorf("%s / %s : une lecture\n  %v\nmais deux lectures\n  %v", c.nom, table, gotLignes, want)
			}
			if len(gotLignes) != c.lignes {
				t.Errorf("%s / %s : %d lignes, want %d — le corpus n'exerce pas ce qu'il prétend :\n  %v",
					c.nom, table, len(gotLignes), c.lignes, gotLignes)
			}
		}
	}
}

// seedFenetresPortee monte `n` matchs de trois frags mesurés chacun (positions ET entames).
func seedFenetresPortee(t *testing.T, pdb *PlayerDB, n int) []string {
	t.Helper()
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	ids := make([]string, 0, n)
	for i := 1; i <= n; i++ {
		id := fmt.Sprintf("p%02d", i)
		ids = append(ids, id)
		tacMatch(t, pdb, id, tacCarteA, base.Add(time.Duration(i)*time.Hour))
		tacParticipant(t, pdb, id, tacXUIDMoi, 0, domain.OutcomeWin)
		for k := 0; k < mortsParMatchFenetres; k++ {
			ms := 1000 * (k + 1)
			tueur, victime := tacXUIDMoi, tacXUIDAdv
			if k%2 == 1 {
				tueur, victime = tacXUIDAdv, tacXUIDMoi
			}
			poserFrag(t, pdb, fragMesure{id, tueur, victime, ms, 10, true})
			poserPositions(t, pdb, id, tueur, ms)
		}
	}
	return ids
}

// TestWeaponRange_Perimetre_FenetresBornees : DEUX matchs demandés sur DIX ; chaque fenêtre des
// trois lectures du repo ne doit voir que les lignes de ces deux-là.
func TestWeaponRange_Perimetre_FenetresBornees(t *testing.T) {
	b := newBaseNotee(t)
	ids := seedFenetresPortee(t, b.pdb, 10)
	repo := NewWeaponRangeRepo(b.pdb, classifieurFenetres{})
	ctx := context.Background()
	f := port.WeaponRangeFilters{MatchIDs: ids[:2], XUIDs: []string{tacXUIDMoi}}
	borne := 2 * mortsParMatchFenetres
	b.carnet.vider()

	kills, err := repo.LoadWeaponRange(ctx, b.pdb.TitleSlug, f)
	if err != nil {
		t.Fatalf("LoadWeaponRange: %v", err)
	}
	if len(kills) != borne {
		t.Fatalf("LoadWeaponRange = %d frags, want %d (chaque frag du scope, de mon côté tueur OU victime)",
			len(kills), borne)
	}
	exigerFenetresBornees(t, b, "LoadWeaponRange", borne, 1)

	if _, err := repo.LoadWeaponOpening(ctx, b.pdb.TitleSlug, f); err != nil {
		t.Fatalf("LoadWeaponOpening: %v", err)
	}
	exigerFenetresBornees(t, b, "LoadWeaponOpening", borne, 1)

	lobby, err := repo.LoadMatchRangeKills(ctx, b.pdb.TitleSlug,
		port.WeaponRangeFilters{MatchIDs: ids[:2], AllPlayers: true})
	if err != nil {
		t.Fatalf("LoadMatchRangeKills: %v", err)
	}
	if len(lobby.Kills) != borne || lobby.KillsTotal != borne {
		t.Fatalf("LoadMatchRangeKills = %d frags / %d publiables, want %d / %d",
			len(lobby.Kills), lobby.KillsTotal, borne, borne)
	}
	// Frags mesurés + dénominateur : deux requêtes, chacune sur une vue `_latest`.
	exigerFenetresBornees(t, b, "LoadMatchRangeKills", borne, 2)
}
