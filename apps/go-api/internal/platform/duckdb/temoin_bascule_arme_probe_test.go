// Package duckdb — temoin_bascule_arme_probe_test.go : LE TEMOIN DE BASCULE de l'arme
// du kill (etape A0 du plan .ai/V7.5/PLAN_SOURCE_UNIQUE_ARME_2026-09-01.md).
//
// # Pourquoi cette sonde existe
//
// Le lot remplace la chaine d'attribution « arme a feu » (correlation des TIRS de
// l'attaquant, table `weapon_kills`) par la SOURCE DE DEGAT lue dans l'etat de mort de la
// victime (`match_kill_events_latest.source_tag`). Affirmer que la seconde est meilleure
// ne suffit pas : il faut pouvoir le DIRE avec des nombres, avant et apres, sur le meme
// lot de matchs. C'est le role de ce fichier.
//
// # Ce qu'elle mesure, cote a cote
//
//  1. le total de frags des compteurs API (`match_participants.kills`) — la reference ;
//  2. la ventilation par classe issue de `weapon_kills` (ancienne chaine) ;
//  3. la ventilation par classe issue de la source de degat (nouvelle chaine) ;
//  4. le residu « Non attribue » des deux.
//
// Elle mesure AUSSI la concordance graphe / kill feed (D13, A0.4/A0.5) — voir
// temoin_bascule_concordance_test.go.
//
// # Elle ne s'arme que sur demande
//
// Sans `TEMOIN_ARME_SHARED`, elle est sautee : elle lit des bases REELLES, que la CI n'a
// pas. Motif des sondes du depot (cf. analysis/filmdec/sonde_*_test.go).
//
// Reproduction :
//
//	export PATH="/c/msys64/ucrt64/bin:$PATH" CGO_ENABLED=1 CC=/c/msys64/ucrt64/bin/gcc.exe
//	TEMOIN_ARME_SHARED=<copie RO de shared_matches_v2.duckdb> \
//	TEMOIN_ARME_META=<copie RO de metadata.duckdb> \
//	TEMOIN_ARME_MATCHS=200 \
//	TEMOIN_ARME_SORTIE=<fichier.md> \
//	go test ./internal/platform/duckdb/ -run TemoinBasculeArme -v -count=1
//
// Les bases sont ouvertes en LECTURE SEULE. Travailler sur une COPIE reste la regle : le
// modele mono-process interdit d'ouvrir un fichier qu'un autre process tient en RW.
package duckdb

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_infinite"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/fragdist"
)

const (
	temoinEnvShared = "TEMOIN_ARME_SHARED"
	temoinEnvMeta   = "TEMOIN_ARME_META"
	temoinEnvMatchs = "TEMOIN_ARME_MATCHS"
	temoinEnvSortie = "TEMOIN_ARME_SORTIE"
	temoinSlug      = "halo_infinite"
	temoinDefautN   = 200
)

// temoinRapport accumule les lignes du rapport. Ecrit sur stdout du test ET, si
// TEMOIN_ARME_SORTIE est donne, dans le fichier designe.
type temoinRapport struct {
	t  *testing.T
	sb strings.Builder
}

func (r *temoinRapport) ligne(format string, args ...any) {
	s := fmt.Sprintf(format, args...)
	r.t.Log(s)
	r.sb.WriteString(s)
	r.sb.WriteString("\n")
}

func (r *temoinRapport) ecrire() {
	chemin := strings.TrimSpace(os.Getenv(temoinEnvSortie))
	if chemin == "" {
		return
	}
	if err := os.WriteFile(chemin, []byte(r.sb.String()), 0o600); err != nil {
		r.t.Errorf("ecriture du rapport dans %s : %v", chemin, err)
		return
	}
	r.t.Logf("rapport ecrit : %s", chemin)
}

// temoinBase : les deux bases ouvertes en lecture seule + le PlayerDB qui les porte.
type temoinBase struct {
	pdb *PlayerDB
}

// ouvrirTemoinBase ouvre les copies read-only et monte un PlayerDB minimal — celui dont
// les repos ont besoin : le lecteur shared et la metadata.
func ouvrirTemoinBase(t *testing.T) *temoinBase {
	t.Helper()
	cheminShared := strings.TrimSpace(os.Getenv(temoinEnvShared))
	cheminMeta := strings.TrimSpace(os.Getenv(temoinEnvMeta))
	if cheminShared == "" || cheminMeta == "" {
		t.Skipf("sonde non armee : %s et %s requis", temoinEnvShared, temoinEnvMeta)
	}
	shared, err := OpenReadOnly(cheminShared)
	if err != nil {
		t.Fatalf("ouverture shared %s : %v", cheminShared, err)
	}
	t.Cleanup(func() { _ = shared.Close() })
	meta, err := OpenReadOnly(cheminMeta)
	if err != nil {
		t.Fatalf("ouverture metadata %s : %v", cheminMeta, err)
	}
	t.Cleanup(func() { _ = meta.Close() })
	return &temoinBase{pdb: &PlayerDB{
		Shared:       shared,
		SharedReader: LegacySharedReader(shared),
		Metadata:     meta,
		TitleSlug:    temoinSlug,
	}}
}

// scope : le lot de matchs mesure et les joueurs credites qu'il porte.
type temoinScope struct {
	matchIDs []string
	xuids    []string
}

// choisirScope prend les N matchs les plus RECENTS vus par au moins une des deux chaines,
// puis tous les xuids credites d'au moins un frag sur ces matchs.
//
// Prendre l'union des deux populations est la seule facon honnete de comparer : restreindre
// aux matchs de `weapon_kills` avantagerait l'ancienne chaine, et l'inverse la nouvelle.
func choisirScope(t *testing.T, b *temoinBase, n int) temoinScope {
	t.Helper()
	ctx := context.Background()
	rows, err := b.pdb.Shared.Query(ctx, `
SELECT r.match_id
FROM match_registry r
WHERE r.match_id IN (SELECT DISTINCT match_id FROM weapon_kills)
   OR r.match_id IN (SELECT DISTINCT match_id FROM match_kill_events_latest WHERE source_tag IS NOT NULL)
ORDER BY COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') DESC
LIMIT ?`, n)
	if err != nil {
		t.Fatalf("selection des matchs : %v", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan match_id : %v", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iteration des matchs : %v", err)
	}
	if len(ids) == 0 {
		t.Fatal("aucun match : la base ne porte ni weapon_kills ni source de degat")
	}
	return temoinScope{matchIDs: ids, xuids: xuidsDuScope(t, b, ids)}
}

// xuidsDuScope liste les joueurs ayant au moins un frag sur le lot. Les bots n'ont pas de
// xuid — ils sortent d'eux-memes.
func xuidsDuScope(t *testing.T, b *temoinBase, matchIDs []string) []string {
	t.Helper()
	args := make([]any, 0, len(matchIDs))
	for _, id := range matchIDs {
		args = append(args, id)
	}
	rows, err := b.pdb.Shared.Query(context.Background(),
		"SELECT DISTINCT xuid FROM match_participants WHERE xuid IS NOT NULL AND kills > 0 AND match_id IN ("+
			Placeholders(len(matchIDs))+")", args...)
	if err != nil {
		t.Fatalf("selection des xuids : %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var x string
		if err := rows.Scan(&x); err != nil {
			t.Fatalf("scan xuid : %v", err)
		}
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iteration des xuids : %v", err)
	}
	sort.Strings(out)
	return out
}

// compteursAPI lit les compteurs kill-type NATIFS du lot (la reference de tout le reste).
func compteursAPI(t *testing.T, b *temoinBase, s temoinScope) domain.FragKillTypeCounts {
	t.Helper()
	args := make([]any, 0, len(s.matchIDs)+len(s.xuids))
	for _, id := range s.matchIDs {
		args = append(args, id)
	}
	for _, x := range s.xuids {
		args = append(args, x)
	}
	var c domain.FragKillTypeCounts
	err := b.pdb.Shared.QueryRow(context.Background(), `
SELECT SUM(COALESCE(kills,0))::INTEGER,
       SUM(COALESCE(melee_kills,0))::INTEGER,
       SUM(COALESCE(grenade_kills,0))::INTEGER,
       SUM(COALESCE(assassination_kills,0))::INTEGER,
       SUM(COALESCE(ground_pound_kills,0))::INTEGER,
       SUM(COALESCE(shoulder_bash_kills,0))::INTEGER
FROM match_participants
WHERE match_id IN (`+Placeholders(len(s.matchIDs))+`) AND xuid IN (`+Placeholders(len(s.xuids))+`)`,
		args...).Scan(&c.Total, &c.Melee, &c.Grenade, &c.Assassination, &c.GroundPound, &c.ShoulderBash)
	if err != nil {
		t.Fatalf("compteurs API : %v", err)
	}
	return c
}

// TestTemoinBasculeArme est LA sonde de l'etape A0. Elle ne conclut rien toute seule : elle
// produit le tableau que le fichier de mesure commente.
func TestTemoinBasculeArme(t *testing.T) {
	b := ouvrirTemoinBase(t)
	n := temoinDefautN
	if v := strings.TrimSpace(os.Getenv(temoinEnvMatchs)); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil || parsed <= 0 {
			t.Fatalf("%s = %q : entier positif attendu", temoinEnvMatchs, v)
		}
		n = parsed
	}
	scope := choisirScope(t, b, n)
	counts := compteursAPI(t, b, scope)

	r := &temoinRapport{t: t}
	defer r.ecrire()
	r.ligne("## Perimetre mesure")
	r.ligne("")
	r.ligne("| grandeur | valeur |")
	r.ligne("|---|---:|")
	r.ligne("| matchs | %d |", len(scope.matchIDs))
	r.ligne("| joueurs credites | %d |", len(scope.xuids))
	r.ligne("| frags API (total) | %d |", counts.Total)
	r.ligne("| melee API | %d |", counts.Melee)
	r.ligne("| grenade API | %d |", counts.Grenade)
	r.ligne("")

	ancienne := ventilationAncienneChaine(t, b, scope, counts)
	nouvelle := ventilationNouvelleChaine(t, b, scope, counts)
	ecrireComparaison(r, counts, ancienne, nouvelle)
	ecrireConcordance(r, t, b, scope, counts)
}

// ventilationAncienneChaine : le sunburst tel qu'il est SERVI aujourd'hui — `weapon_kills`
// pour les armes a feu, source de degat pour les trois classes hors arsenal.
func ventilationAncienneChaine(t *testing.T, b *temoinBase, s temoinScope,
	counts domain.FragKillTypeCounts,
) domain.FragDistribution {
	t.Helper()
	repo := NewWeaponKillsRepo(b.pdb)
	rows, err := repo.LoadWeaponKillsAggregated(context.Background(), temoinSlug, port.WeaponKillFilters{
		MatchIDs: s.matchIDs, XUIDs: s.xuids, ResolveRoles: true, IncludeGrenadeMelee: true,
	})
	if err != nil {
		t.Fatalf("ancienne chaine (weapon_kills) : %v", err)
	}
	src := NewKillSourceClassRepo(b.pdb, halo_infinite.NewKillSourceRegistry())
	sources, err := src.LoadKillSourceClassesAggregated(context.Background(), temoinSlug,
		port.KillSourceClassFilters{MatchIDs: s.matchIDs, XUIDs: s.xuids})
	if err != nil {
		t.Fatalf("ancienne chaine (hors arsenal) : %v", err)
	}
	return fragdist.Build(rows, sources, counts, false)
}

// classesTriees rend les classes d'une distribution indexees par nom.
func classesTriees(d domain.FragDistribution) map[string]int {
	out := map[string]int{}
	for _, c := range d.Classes {
		out[c.Class] = c.Kills
	}
	return out
}

// ecrireComparaison pose les deux ventilations cote a cote, classe par classe, avec l'ecart.
func ecrireComparaison(r *temoinRapport, counts domain.FragKillTypeCounts,
	ancienne, nouvelle domain.FragDistribution,
) {
	a, nv := classesTriees(ancienne), classesTriees(nouvelle)
	noms := map[string]bool{}
	for k := range a {
		noms[k] = true
	}
	for k := range nv {
		noms[k] = true
	}
	ordre := make([]string, 0, len(noms))
	for k := range noms {
		ordre = append(ordre, k)
	}
	sort.Strings(ordre)

	r.ligne("## Ventilation par classe — ancienne chaine contre nouvelle")
	r.ligne("")
	r.ligne("| classe | weapon_kills (servi aujourd'hui) | source de degat | ecart |")
	r.ligne("|---|---:|---:|---:|")
	for _, c := range ordre {
		r.ligne("| %s | %d | %d | %+d |", c, a[c], nv[c], nv[c]-a[c])
	}
	r.ligne("| **total frags API** | %d | %d | |", counts.Total, counts.Total)
	r.ligne("")
	r.ligne("Residu « Non attribue » : ancienne %d (%.1f %%), nouvelle %d (%.1f %%).",
		a[domain.FragClassUnattributed], pourcent(a[domain.FragClassUnattributed], counts.Total),
		nv[domain.FragClassUnattributed], pourcent(nv[domain.FragClassUnattributed], counts.Total))
	r.ligne("")
}

// pourcent : part d'un compteur dans un total, 0 si le total est nul.
func pourcent(part, total int) float64 {
	if total <= 0 {
		return 0
	}
	return 100 * float64(part) / float64(total)
}
