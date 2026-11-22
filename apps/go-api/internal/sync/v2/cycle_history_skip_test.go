// Package v2 — cycle_history_skip_test.go : un match introuvable (404/410) ou abandonné par le rattrapage
// entre dans l'ensemble des matchs à ne plus demander (historystate.SkipSetKey). Il est neutre pour la
// découverte (ni connu, ni inconnu), n'est plus demandé, reste à ne plus demander après la fin de
// l'historique, et ne compte jamais comme inséré : « matchs insérés » = matchs PROUVÉS rangés.
package v2

import (
	"context"
	"slices"
	"testing"

	"levelup/go-api/internal/sync/historystate"
)

// LoadSkipSet (SkipSetLoader) : l'ensemble des matchs à ne plus demander, lu sur la base du joueur.
func (e *histEnv) LoadSkipSet(ctx context.Context, _ PlayerProfile) map[string]bool {
	return historystate.LoadSkipSet(ctx, e.db)
}

func (e *histEnv) skipSet() map[string]bool {
	return historystate.LoadSkipSet(context.Background(), e.db)
}

// markNotFound déclare ces matchs introuvables (404) côté API.
func (e *histEnv) markNotFound(ids ...string) {
	for _, id := range ids {
		e.fetcher.perMatchErr[id] = notFoundErr()
	}
}

// markDeltaOnly range la fin d'historique : le joueur est en delta pur.
func (e *histEnv) markDeltaOnly() {
	e.t.Helper()
	if !historystate.MarkComplete(context.Background(), e.db, "alice") {
		e.t.Fatal("MarkComplete")
	}
}

// rattrape joue des cycles jusqu'à la fin d'historique rangée et rend la somme des matchs insérés
// annoncés ; chaque cycle annonce exactement les matchs qu'il a rangés.
func (e *histEnv) rattrape(maxCycles int) (inserted int) {
	e.t.Helper()
	for i := 1; i <= maxCycles; i++ {
		avant := e.knownCount()
		out, err := e.cycle()
		if err != nil {
			e.t.Fatalf("cycle %d : %v", i, err)
		}
		if got, want := out.MatchesInserted, e.knownCount()-avant; got != want {
			e.t.Fatalf("cycle %d : MatchesInserted=%d, %d matchs rangés", i, got, want)
		}
		inserted += out.MatchesInserted
		if e.complete() {
			return inserted
		}
	}
	e.t.Fatalf("fin d'historique non rangée en %d cycles", maxCycles)
	return inserted
}

// Après la fin d'un rattrapage de 1 000 matchs dont 30 introuvables dispersés : les 970 autres sont
// annoncés insérés une fois, les 30 introuvables ne sont ni annoncés ni redemandés, et un cycle sans
// match neuf lit une page et ne demande aucune statistique.
func TestCycleSkip_IntrouvablesDisperses_PlusRedemandesApresLaFin(t *testing.T) {
	e := newHistEnv(t, 1000)
	var gone []string
	for i := 0; i < 30; i++ {
		gone = append(gone, e.ids[100+i*30])
	}
	e.markNotFound(gone...)

	if got := e.rattrape(10); got != 970 {
		t.Fatalf("matchs insérés annoncés pendant le rattrapage : %d, want 970", got)
	}
	if got := e.skipSet(); len(got) != 30 {
		t.Fatalf("ensemble des matchs à ne plus demander : %d, want 30", len(got))
	}
	for i := 1; i <= 2; i++ {
		appels := e.fetcher.totalCalls.Load()
		out, err := e.cycle()
		if err != nil {
			t.Fatal(err)
		}
		if len(e.client.starts) != 1 {
			t.Errorf("cycle %d après la fin : %d pages lues %v, want 1", i, len(e.client.starts), e.client.starts)
		}
		if got := e.fetcher.totalCalls.Load() - appels; got != 0 {
			t.Errorf("cycle %d après la fin : %d demandes de statistiques, want 0", i, got)
		}
		if out.MatchesInserted != 0 || out.MatchesUnknown != 0 {
			t.Errorf("cycle %d après la fin : inconnus %d, insérés %d, want 0 et 0", i, out.MatchesUnknown, out.MatchesInserted)
		}
	}
}

// 30 introuvables parmi les premières entrées : le delta lit les pages nécessaires pour atteindre 25 connus
// d'affilée (les introuvables sont neutres : ni comptés, ni remise à zéro), sans aucune demande de
// statistiques ni match annoncé inséré.
func TestCycleSkip_IntrouvablesEnTete_NeutresPourLeDelta(t *testing.T) {
	e := newHistEnv(t, 1000)
	e.markNotFound(e.ids[:30]...)
	e.rattrape(10)
	for i := 1; i <= 2; i++ {
		appels := e.fetcher.totalCalls.Load()
		out, err := e.cycle()
		if err != nil {
			t.Fatal(err)
		}
		// 30 neutres + 25 connus d'affilée = 55 entrées : trois pages de 25.
		if want := []int{0, 25, 50}; !slices.Equal(e.client.starts, want) {
			t.Errorf("cycle %d après la fin : pages lues %v, want %v", i, e.client.starts, want)
		}
		if got := e.fetcher.totalCalls.Load() - appels; got != 0 {
			t.Errorf("cycle %d après la fin : %d demandes de statistiques, want 0", i, got)
		}
		if out.MatchesInserted != 0 {
			t.Errorf("cycle %d après la fin : %d insérés annoncés, want 0", i, out.MatchesInserted)
		}
	}
}

// Un introuvable ne remet pas à zéro la suite de connus : 24 connus, un introuvable, puis un connu
// arrêtent le delta à la deuxième page (25 connus d'affilée, l'introuvable ne compte pas).
func TestCycleSkip_IntrouvableNeRemetPasLaSuiteAZero(t *testing.T) {
	e := newHistEnv(t, 100)
	e.markDeltaOnly()
	for _, id := range e.ids[1:] {
		e.known[id] = true
	}
	e.markNotFound(e.ids[0])
	if _, err := e.cycle(); err != nil { // premier cycle : la demande unique, rangée à ne plus demander
		t.Fatal(err)
	}
	if len(e.skipSet()) != 1 {
		t.Fatalf("ensemble %v, want un match", e.skipSet())
	}
	appels := e.fetcher.totalCalls.Load()
	if _, err := e.cycle(); err != nil {
		t.Fatal(err)
	}
	// Entrée 0 neutre, entrées 1 à 24 connues (24), entrée 25 connue : 25 d'affilée, deuxième page.
	if want := []int{0, 25}; !slices.Equal(e.client.starts, want) {
		t.Errorf("pages lues %v, want %v", e.client.starts, want)
	}
	if got := e.fetcher.totalCalls.Load() - appels; got != 0 {
		t.Errorf("%d demandes de statistiques, want 0", got)
	}
}

// Historique déjà complet (delta pur) : l'introuvable n'est ni compté inséré ni redemandé aux cycles suivants.
func TestCycleSkip_DeltaPur_IntrouvableRangeAuPremierCycle(t *testing.T) {
	e := newHistEnv(t, 100)
	e.markDeltaOnly()
	e.markNotFound(e.ids[0])

	out, err := e.cycle()
	if err != nil {
		t.Fatal(err)
	}
	if out.MatchesInserted != 99 || e.knownCount() != 99 {
		t.Fatalf("premier cycle : %d insérés annoncés, %d rangés, want 99 et 99", out.MatchesInserted, e.knownCount())
	}
	if got := e.skipSet(); len(got) != 1 || !got[e.ids[0]] {
		t.Fatalf("ensemble des matchs à ne plus demander : %v, want [%s]", got, e.ids[0])
	}
	appels := e.fetcher.totalCalls.Load()
	out, err = e.cycle()
	if err != nil {
		t.Fatal(err)
	}
	if got := e.fetcher.totalCalls.Load() - appels; got != 0 || out.MatchesInserted != 0 {
		t.Errorf("cycle suivant : %d demandes, %d insérés, want 0 et 0", got, out.MatchesInserted)
	}
}

// Un match abandonné (trois passes en échec) n'est plus redemandé après la fin d'historique, qui efface
// pourtant la liste des échecs : il reste dans l'ensemble des matchs à ne plus demander.
func TestCycleSkip_MatchAbandonne_PlusRedemandeApresLaFin(t *testing.T) {
	size := historystate.PassBudget() + 30
	e := newHistEnv(t, size)
	bad := e.ids[2]
	e.fetcher.perMatchErr[bad] = serverErr()
	e.rattrape(2 + maxCatchUpAttempts)

	if len(e.failures()) != 0 {
		t.Fatalf("liste des échecs %v, want vide après la fin", e.failures())
	}
	if got := e.skipSet(); len(got) != 1 || !got[bad] {
		t.Fatalf("ensemble des matchs à ne plus demander : %v, want [%s]", got, bad)
	}
	for i := 1; i <= 2; i++ {
		appels := e.fetcher.totalCalls.Load()
		out, err := e.cycle()
		if err != nil {
			t.Fatal(err)
		}
		if got := e.fetcher.totalCalls.Load() - appels; got != 0 {
			t.Errorf("cycle %d après la fin : %d demandes de statistiques, le match abandonné est redemandé", i, got)
		}
		if out.MatchesInserted != 0 || out.MatchesUnknown != 0 {
			t.Errorf("cycle %d après la fin : inconnus %d, insérés %d, want 0 et 0", i, out.MatchesUnknown, out.MatchesInserted)
		}
	}
}

// Un match récupéré mais que la persistance n'a pas rangé n'est pas annoncé inséré (preuve par la
// relecture de l'ensemble connu).
func TestCycleSkip_MatchNonRange_NEstPasCompteInsere(t *testing.T) {
	e := newHistEnv(t, 100)
	e.markDeltaOnly()
	e.dropOnPersist[e.ids[1]] = true

	out, err := e.cycle()
	if err != nil {
		t.Fatal(err)
	}
	if out.MatchesInserted != 99 || e.knownCount() != 99 {
		t.Errorf("%d insérés annoncés, %d rangés, want 99 et 99 (le match non rangé n'est pas annoncé)", out.MatchesInserted, e.knownCount())
	}
	if out.MatchesUnknown != 100 {
		t.Errorf("%d inconnus listés, want 100", out.MatchesUnknown)
	}
}

// Un orphelin de l'ensemble connu que la découverte ne demande plus n'est pas redemandé par son identifiant.
func TestAppendOrphans_IgnoreLesMatchsADeNePlusDemander(t *testing.T) {
	p := mkPlayers("alice")[0]
	got := appendOrphans(context.Background(), p, []string{"a"}, []string{"a", "gone", "b"}, map[string]bool{"gone": true})
	if want := []string{"a", "b"}; !slices.Equal(got, want) {
		t.Errorf("appendOrphans = %v, want %v", got, want)
	}
}
