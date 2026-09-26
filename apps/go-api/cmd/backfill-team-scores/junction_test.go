package main

// junction_test.go — LA JONCTION lecture -> Decide -> UPDATE, DE BOUT EN BOUT.
//
// CE QUE CES TESTS ATTRAPENT, ET QUE RIEN D'AUTRE N'ATTRAPAIT (constat P1 de la revue
// adversariale du 2026-08-24). `Decide` était couvert, `LoadMatchIDs` aussi, mais le
// câblage entre les deux ne l'était pas : une permutation des colonnes dans le SELECT, un
// `SET team_0_score = ?, team_1_score = ?` dont les arguments seraient liés à l'envers, ou
// un `Old` lié à la place d'un `New`, passaient toute la suite EN VERT. Pire, le journal ne
// l'aurait pas montré : il est construit depuis la décision, pas depuis ce qui part
// réellement vers la base.
//
// D'où des doubles qui observent ce que DuckDB recevrait : le texte SQL exact et l'ordre
// exact des arguments liés. Aucune base réelle, aucun réseau.

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// ─── doubles ────────────────────────────────────────────────────────────────────────

// fakeFetcher rend des payloads en dur, indexés par match_id.
type fakeFetcher struct {
	payloads map[string]map[string]any
	err      error
	calls    []string
}

func (f *fakeFetcher) GetMatchStats(_ context.Context, matchID string) (map[string]any, error) {
	f.calls = append(f.calls, matchID)
	if f.err != nil {
		return nil, f.err
	}
	p, ok := f.payloads[matchID]
	if !ok {
		return nil, errors.New("payload absent du double")
	}
	return p, nil
}

// fakeReader rend des lignes de registre en dur. `sequence` permet de faire varier la
// réponse entre la phase A et la re-lecture de la phase B.
type fakeReader struct {
	rows     map[string]RegistryScores
	missing  map[string]bool
	err      error
	sequence map[string][]RegistryScores // consommée dans l'ordre si présente
	reads    []string
}

func (r *fakeReader) ReadScores(_ context.Context, matchID string) (RegistryScores, bool, error) {
	r.reads = append(r.reads, matchID)
	if r.err != nil {
		return RegistryScores{}, false, r.err
	}
	if r.missing[matchID] {
		return RegistryScores{}, false, nil
	}
	if seq, ok := r.sequence[matchID]; ok && len(seq) > 0 {
		head := seq[0]
		r.sequence[matchID] = seq[1:]
		return head, true, nil
	}
	row, ok := r.rows[matchID]
	if !ok {
		return RegistryScores{}, false, nil
	}
	return row, true, nil
}

// recordedExec capture UN appel ExecContext : le SQL et les arguments, tels quels.
type recordedExec struct {
	query string
	args  []any
}

// fakeExecer implémente `execer` — la couture réelle de l'écriture. En passant par lui,
// le test exerce le VRAI `sqlRegistry.WriteScores`, donc le vrai SQL et le vrai ordre
// d'arguments, pas une reconstitution.
type fakeExecer struct {
	calls   []recordedExec
	err     error
	rowsAff int64
}

func (e *fakeExecer) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	e.calls = append(e.calls, recordedExec{query: query, args: args})
	if e.err != nil {
		return nil, e.err
	}
	n := e.rowsAff
	if n == 0 {
		n = 1
	}
	return fakeResult{rows: n}, nil
}

type fakeResult struct{ rows int64 }

func (r fakeResult) LastInsertId() (int64, error) { return 0, errors.New("non supporté") }
func (r fakeResult) RowsAffected() (int64, error) { return r.rows, nil }

// ─── le test central ────────────────────────────────────────────────────────────────

// TestJonction_ValeursEcritesSontCellesDecidees déroule les deux phases sur le cas réel
// `7344d24f` : la base porte les ticks de zone (193/112), l'API le score affiché (200/126).
//
// Il vérifie les trois points du constat P1 :
//
//	(a) ce qui est ÉCRIT est ce qui a été DÉCIDÉ, dans le bon ordre t0/t1, avec le bon id ;
//	(b) ce qui est LU vient des bonnes colonnes (le SELECT est asserté, et
//	    TestScanRegistryScores prouve la correspondance position -> champ) ;
//	(c) ce sont les valeurs du FETCH qui gagnent, jamais celles du fichier d'entrée.
func TestJonction_ValeursEcritesSontCellesDecidees(t *testing.T) {
	const id = "7344d24f-0154-4949-80ad-e2b781c122f1"
	ctx := context.Background()

	fetcher := &fakeFetcher{payloads: map[string]map[string]any{
		// Les deux champs coexistent, comme dans le vrai payload.
		id: teamsPayload(teamWithZones(0, 200, 193), teamWithZones(1, 126, 112)),
	}}
	reader := &fakeReader{rows: map[string]RegistryScores{
		id: {Team0: ptr(193), Team1: ptr(112)},
	}}
	exec := &fakeExecer{}
	reg := sqlRegistry{ex: exec}

	var tl tally
	plan, ok := planMatch(ctx, fetcher, reader, id, &tl)
	if !ok {
		t.Fatalf("phase A n'a retenu aucune correction (tally %+v)", tl)
	}
	if plan.matchID != id {
		t.Fatalf("plan.matchID = %q, attendu %q", plan.matchID, id)
	}
	applyOne(ctx, reader, reg, plan, &tl)

	if len(exec.calls) != 1 {
		t.Fatalf("%d écritures, 1 attendue", len(exec.calls))
	}
	call := exec.calls[0]

	// (a) — le SQL émis est bien LA constante du paquet.
	//
	// Attention : cette égalité seule est une TAUTOLOGIE (elle compare la requête à la
	// constante qui l'a produite). Permuter les colonnes DANS la constante y survivrait.
	// C'est `TestUpdateLieLePremierArgumentATeam0` qui verrouille la structure de la
	// constante elle-même — défaut P1-b de la ronde 2 de revue.
	if call.query != updateScoresSQL {
		t.Errorf("SQL écrit = %q, attendu %q", call.query, updateScoresSQL)
	}
	if len(call.args) != 3 {
		t.Fatalf("%d arguments liés, 3 attendus : %v", len(call.args), call.args)
	}
	if got := call.args[0]; got != 200 {
		t.Errorf("1er argument (team_0_score) = %v, attendu 200 — colonnes permutées ?", got)
	}
	if got := call.args[1]; got != 126 {
		t.Errorf("2e argument (team_1_score) = %v, attendu 126 — colonnes permutées ?", got)
	}
	if got := call.args[2]; got != id {
		t.Errorf("3e argument (match_id) = %v, attendu %q", got, id)
	}

	// Les ANCIENNES valeurs ne doivent apparaître nulle part dans ce qui est lié : c'est
	// le scénario « Old lié à la place de New », invisible au journal.
	for i, a := range call.args {
		if a == 193 || a == 112 {
			t.Errorf("argument %d = %v : c'est une ANCIENNE valeur (base), pas la valeur décidée", i, a)
		}
	}
	if tl.fixed != 1 || tl.failed != 0 {
		t.Errorf("tally = %+v, attendu 1 corrigé / 0 échec", tl)
	}
}

// TestJonction_LeFetchGagneContreLeFichier : le TSV d'entrée porte des colonnes
// `api_t0`/`api_t1` EMPOISONNÉES ; l'outil doit écrire ce que dit le fetch, pas le fichier.
// C'est la preuve que « le fichier n'est qu'une liste » n'est pas qu'un commentaire.
func TestJonction_LeFetchGagneContreLeFichier(t *testing.T) {
	ctx := context.Background()
	const id = "abc-empoisonne"

	tsv := "match_id\tdb_t0\tdb_t1\tapi_t0\tapi_t1\n" +
		id + "\t193\t112\t999\t888\n" // 999/888 = valeurs mensongères
	ids, err := LoadMatchIDs(writeTemp(t, tsv))
	if err != nil {
		t.Fatalf("LoadMatchIDs : %v", err)
	}
	if len(ids) != 1 || ids[0] != id {
		t.Fatalf("ids = %v, attendu [%s]", ids, id)
	}

	fetcher := &fakeFetcher{payloads: map[string]map[string]any{
		id: teamsPayload(team(0, 200), team(1, 126)),
	}}
	reader := &fakeReader{rows: map[string]RegistryScores{id: {Team0: ptr(193), Team1: ptr(112)}}}
	exec := &fakeExecer{}

	var tl tally
	plan, ok := planMatch(ctx, fetcher, reader, ids[0], &tl)
	if !ok {
		t.Fatalf("aucune correction retenue")
	}
	applyOne(ctx, reader, sqlRegistry{ex: exec}, plan, &tl)

	if len(exec.calls) != 1 {
		t.Fatalf("%d écritures, 1 attendue", len(exec.calls))
	}
	for i, a := range exec.calls[0].args {
		if a == 999 || a == 888 {
			t.Fatalf("argument %d = %v : une valeur du FICHIER a été écrite — le TSV ne doit servir que de liste", i, a)
		}
	}
	if exec.calls[0].args[0] != 200 || exec.calls[0].args[1] != 126 {
		t.Errorf("écrit %v/%v, attendu 200/126 (les valeurs du fetch)", exec.calls[0].args[0], exec.calls[0].args[1])
	}
}

// TestJonction_ReLectureEvitteUneEcritureInutile : si la ligne a été corrigée entre la
// phase A et la phase B, la phase B ne réécrit pas. C'est ce qui rend l'outil idempotent.
func TestJonction_ReLectureEvitteUneEcritureInutile(t *testing.T) {
	ctx := context.Background()
	const id = "abc-deja-corrige"

	fetcher := &fakeFetcher{payloads: map[string]map[string]any{
		id: teamsPayload(team(0, 200), team(1, 126)),
	}}
	reader := &fakeReader{sequence: map[string][]RegistryScores{
		id: {
			{Team0: ptr(193), Team1: ptr(112)}, // phase A : divergent
			{Team0: ptr(200), Team1: ptr(126)}, // phase B : déjà à jour
		},
	}}
	exec := &fakeExecer{}

	var tl tally
	plan, ok := planMatch(ctx, fetcher, reader, id, &tl)
	if !ok {
		t.Fatalf("phase A aurait dû retenir une correction")
	}
	applyOne(ctx, reader, sqlRegistry{ex: exec}, plan, &tl)

	if len(exec.calls) != 0 {
		t.Fatalf("%d écriture(s) alors que la ligne était déjà à jour", len(exec.calls))
	}
	if tl.fixed != 0 || tl.identical != 1 {
		t.Errorf("tally = %+v, attendu 0 corrigé / 1 identique", tl)
	}
}

// TestJonction_AucuneEcritureSurSkipNiIdentique : les verdicts non écrivables ne doivent
// produire AUCUN appel à la base, même par inadvertance.
func TestJonction_AucuneEcritureSurSkipNiIdentique(t *testing.T) {
	ctx := context.Background()
	cases := map[string]struct {
		payload map[string]any
		cur     RegistryScores
	}{
		"identique": {teamsPayload(team(0, 50), team(1, 42)), RegistryScores{Team0: ptr(50), Team1: ptr(42)}},
		"ffa":       {teamsPayload(team(2, 30), team(3, 25)), RegistryScores{Team0: ptr(30), Team1: ptr(25)}},
		"negatif":   {teamsPayload(team(0, -1), team(1, 5)), RegistryScores{Team0: ptr(0), Team1: ptr(5)}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			const id = "abc-1"
			fetcher := &fakeFetcher{payloads: map[string]map[string]any{id: tc.payload}}
			reader := &fakeReader{rows: map[string]RegistryScores{id: tc.cur}}
			var tl tally
			if _, ok := planMatch(ctx, fetcher, reader, id, &tl); ok {
				t.Fatalf("une correction a été retenue alors qu'aucune n'était attendue")
			}
		})
	}
}

// TestJonction_MatchAbsentNeDeclencheAucunFetch : inutile de payer un appel API pour un
// match qui n'est pas au registre — et surtout, il ne doit pas être compté comme lu.
func TestJonction_MatchAbsentNeDeclencheAucunFetch(t *testing.T) {
	ctx := context.Background()
	const id = "abc-absent"
	fetcher := &fakeFetcher{payloads: map[string]map[string]any{}}
	reader := &fakeReader{missing: map[string]bool{id: true}}

	var tl tally
	if _, ok := planMatch(ctx, fetcher, reader, id, &tl); ok {
		t.Fatal("une correction a été retenue pour un match absent du registre")
	}
	if len(fetcher.calls) != 0 {
		t.Errorf("%d appel(s) API pour un match absent du registre", len(fetcher.calls))
	}
	if tl.skipped != 1 || tl.read != 0 {
		t.Errorf("tally = %+v, attendu 1 skip / 0 lu", tl)
	}
}

// TestJonction_EchecsComptes : une panne d'API et une panne d'écriture doivent compter
// comme des échecs, pas passer pour des succès silencieux.
func TestJonction_EchecsComptes(t *testing.T) {
	ctx := context.Background()
	const id = "abc-1"

	t.Run("api en panne", func(t *testing.T) {
		fetcher := &fakeFetcher{err: errors.New("503")}
		reader := &fakeReader{rows: map[string]RegistryScores{id: {Team0: ptr(1), Team1: ptr(2)}}}
		var tl tally
		if _, ok := planMatch(ctx, fetcher, reader, id, &tl); ok {
			t.Fatal("une correction a été retenue malgré l'échec du fetch")
		}
		if tl.failed != 1 {
			t.Errorf("tally = %+v, attendu 1 échec", tl)
		}
	})

	t.Run("update qui ne touche aucune ligne", func(t *testing.T) {
		reader := &fakeReader{rows: map[string]RegistryScores{id: {Team0: ptr(1), Team1: ptr(2)}}}
		exec := &fakeExecer{rowsAff: -1} // -1 : ni 0 par défaut, ni 1
		var tl tally
		plan := plannedFix{matchID: id, decision: Decision{Verdict: VerdictFix, NewTeam0: 200, NewTeam1: 126}}
		applyOne(ctx, reader, sqlRegistry{ex: exec}, plan, &tl)
		if tl.failed != 1 || tl.fixed != 0 {
			t.Errorf("tally = %+v, attendu 1 échec / 0 corrigé : un UPDATE qui ne touche pas exactement une ligne est une erreur", tl)
		}
	})
}

// ─── la lecture, colonne par colonne ────────────────────────────────────────────────

// stubScanner remplit les destinations dans un ordre CONNU. Il prouve la correspondance
// « position dans le SELECT -> champ de RegistryScores », qu'aucun autre test ne couvre.
type stubScanner struct {
	values []sql.NullInt64
	err    error
}

func (s stubScanner) Scan(dest ...any) error {
	if s.err != nil {
		return s.err
	}
	if len(dest) != len(s.values) {
		return errors.New("nombre de destinations inattendu")
	}
	for i := range dest {
		p, ok := dest[i].(*sql.NullInt64)
		if !ok {
			return errors.New("destination n'est pas un *sql.NullInt64")
		}
		*p = s.values[i]
	}
	return nil
}

func TestScanRegistryScores(t *testing.T) {
	t.Run("la 1re colonne va dans Team0", func(t *testing.T) {
		got, found, err := scanRegistryScores(stubScanner{values: []sql.NullInt64{
			{Int64: 193, Valid: true}, {Int64: 112, Valid: true},
		}})
		if err != nil || !found {
			t.Fatalf("found=%v err=%v", found, err)
		}
		if got.Team0 == nil || *got.Team0 != 193 {
			t.Errorf("Team0 = %s, attendu 193 — colonnes du SELECT permutées ?", formatScore(got.Team0))
		}
		if got.Team1 == nil || *got.Team1 != 112 {
			t.Errorf("Team1 = %s, attendu 112", formatScore(got.Team1))
		}
	})

	t.Run("NULL reste NULL, pas zero", func(t *testing.T) {
		got, found, err := scanRegistryScores(stubScanner{values: []sql.NullInt64{
			{Valid: false}, {Int64: 0, Valid: true},
		}})
		if err != nil || !found {
			t.Fatalf("found=%v err=%v", found, err)
		}
		if got.Team0 != nil {
			t.Errorf("Team0 = %s, attendu NULL — nullIntPtr confond NULL et 0", formatScore(got.Team0))
		}
		if got.Team1 == nil || *got.Team1 != 0 {
			t.Errorf("Team1 = %s, attendu 0 (valide)", formatScore(got.Team1))
		}
	})

	t.Run("ligne absente n'est pas une erreur", func(t *testing.T) {
		_, found, err := scanRegistryScores(stubScanner{err: sql.ErrNoRows})
		if err != nil {
			t.Fatalf("ErrNoRows ne doit pas remonter comme erreur : %v", err)
		}
		if found {
			t.Error("found=true sur une ligne absente")
		}
	})

	t.Run("vraie erreur remonte", func(t *testing.T) {
		if _, _, err := scanRegistryScores(stubScanner{err: errors.New("io")}); err == nil {
			t.Fatal("une erreur de scan doit remonter, pas être avalée")
		}
	})
}

// TestSelectListeLesColonnesDansLOrdreDuScan verrouille le couple SELECT/Scan : le test
// ci-dessus prouve l'ordre du Scan, celui-ci prouve que le SQL suit le même ordre.
func TestSelectListeLesColonnesDansLOrdreDuScan(t *testing.T) {
	i0 := indexIn(selectScoresSQL, "team_0_score")
	i1 := indexIn(selectScoresSQL, "team_1_score")
	if i0 < 0 || i1 < 0 {
		t.Fatalf("selectScoresSQL ne nomme pas les deux colonnes : %q", selectScoresSQL)
	}
	if i0 > i1 {
		t.Errorf("selectScoresSQL sélectionne team_1_score AVANT team_0_score : le Scan les inverserait — %q", selectScoresSQL)
	}
}

// TestUpdateLieLePremierArgumentATeam0 verrouille la STRUCTURE de `updateScoresSQL`, pas
// son égalité à elle-même.
//
// Sans lui, permuter les deux affectations dans la constante
// (`SET team_1_score = ?, team_0_score = ?`) passait toute la suite : le double de test
// compare la requête émise à cette même constante, donc la permutation était invisible —
// et les 80 lignes auraient été écrites camps inversés. Miroir exact de
// `TestSelectListeLesColonnesDansLOrdreDuScan`, pour l'écriture.
func TestUpdateLieLePremierArgumentATeam0(t *testing.T) {
	i0 := indexIn(updateScoresSQL, "team_0_score")
	i1 := indexIn(updateScoresSQL, "team_1_score")
	iMatch := indexIn(updateScoresSQL, "match_id")
	iFirstQ := indexIn(updateScoresSQL, "?")
	if i0 < 0 || i1 < 0 || iMatch < 0 || iFirstQ < 0 {
		t.Fatalf("updateScoresSQL ne porte pas les trois colonnes et un placeholder : %q", updateScoresSQL)
	}
	if i0 > i1 {
		t.Errorf("updateScoresSQL affecte team_1_score AVANT team_0_score : les arguments "+
			"liés (team0, team1) écriraient les camps INVERSÉS — %q", updateScoresSQL)
	}
	// Le 1er `?` doit tomber ENTRE team_0_score et team_1_score : c'est ce qui prouve que
	// le premier argument lié alimente bien team_0_score.
	if !(iFirstQ > i0 && iFirstQ < i1) {
		t.Errorf("le 1er placeholder n'est pas celui de team_0_score (pos %d, team_0 en %d, "+
			"team_1 en %d) : le 1er argument lié n'irait pas dans team_0_score — %q",
			iFirstQ, i0, i1, updateScoresSQL)
	}
	// match_id est le DERNIER : c'est le 3e argument lié.
	if iMatch < i1 {
		t.Errorf("match_id apparaît avant team_1_score : l'ordre des arguments liés "+
			"(team0, team1, matchID) ne correspond plus — %q", updateScoresSQL)
	}
}

func indexIn(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

// TestWriteScoresRefuseSansDroitDEcriture : en phase A l'outil n'a pas d'`execer`. Une
// écriture demandée là est un bug d'appel, pas un no-op silencieux.
func TestWriteScoresRefuseSansDroitDEcriture(t *testing.T) {
	err := sqlRegistry{}.WriteScores(context.Background(), "abc-1", 1, 2)
	if err == nil {
		t.Fatal("WriteScores sans execer doit échouer explicitement")
	}
}

// TestReadScoresRefuseSansBase : symétrique du précédent.
func TestReadScoresRefuseSansBase(t *testing.T) {
	if _, _, err := (sqlRegistry{}).ReadScores(context.Background(), "abc-1"); err == nil {
		t.Fatal("ReadScores sans base doit échouer explicitement")
	}
}

// TestFichierParDefautExisteEtEstLisible : tant que la CLI porte ce défaut, le fichier
// doit exister. S'il est déplacé ou archivé, la suite doit le dire — pas le sauter.
func TestFichierParDefautExisteEtEstLisible(t *testing.T) {
	// Depuis apps/go-api/cmd/backfill-team-scores/, la racine du dépôt est quatre
	// niveaux au-dessus (backfill-team-scores -> cmd -> go-api -> apps -> racine).
	path := filepath.Join("..", "..", "..", "..", defaultIDsFile)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("le fichier par défaut de --ids-file est introuvable (%s) : %v\n"+
			"Il est le défaut de la CLI : s'il a été déplacé ou archivé, mettre à jour "+
			"defaultIDsFile dans le même commit.", defaultIDsFile, err)
	}
	ids, err := LoadMatchIDs(path)
	if err != nil {
		t.Fatalf("le fichier par défaut doit être chargeable : %v", err)
	}
	if len(ids) != 80 {
		t.Errorf("%d ids chargés, 80 attendus (les 80 écarts mesurés le 2026-08-24)", len(ids))
	}
}
