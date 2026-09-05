package replaybuild

// grenade_ambigu_sweep_test.go — LE DENOMINATEUR ELARGI DE LA PHASE 0.
//
// # POURQUOI CE SECOND INSTRUMENT EXISTE
//
// Le corpus d'artefacts ne compte que 23 films, soit 63 morts par grenade. Un compte de ZERO
// sur 63 ne DIT PAS « moins de 1 % » : la regle de trois borne la vraie proportion a 4,8 %
// avec 95 % de confiance. Arreter un chantier sur cette base serait conclure au-dela de ce que
// la mesure porte — la faute exacte que le seuil du plan cherche a interdire.
//
// Or la question de la phase 0 (« quelle part des morts par grenade porte l'un des deux tags
// AMBIGU ») n'a besoin QUE du film : le tag se lit dans le dead-state de la victime. L'artefact
// n'est necessaire qu'a la JOINTURE de la phase 1. Le denominateur peut donc s'elargir a tout
// le cache de films sans rien construire.
//
// # GARDE
//
// `FILM_SWEEP` = nombre de films a balayer (« tout » pour la totalite). Absent : le test SAUTE.
// Le cache n'est pas versionne, la CI ne l'a pas, et un balayage coute ~7 s par film. Les films
// sont pris dans l'ORDRE LEXICAL de leur identifiant court, qui est un prefixe d'UUID : la
// tranche est arbitraire, donc non biaisee par la carte, le mode ou la date.
//
// Aucune base n'est ouverte, ni en lecture ni en ecriture.

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
	"levelup/go-api/internal/games/halo_infinite/film/killsource"
)

// tailleDuBalayage lit `FILM_SWEEP`, ou saute le test.
func tailleDuBalayage(t *testing.T) int {
	t.Helper()
	v := strings.TrimSpace(os.Getenv("FILM_SWEEP"))
	if v == "" {
		t.Skip("FILM_SWEEP non defini : balayage large du cache de films non demande")
	}
	if strings.EqualFold(v, "tout") {
		return -1
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		t.Fatalf("FILM_SWEEP doit etre un entier positif ou « tout » (recu %q)", v)
	}
	return n
}

// TestPhase0Bis_BalayageLargeDesTagsAmbigus — LA MESURE QUI REND LE GATE 0 DECISIF.
//
// Elle ne remplace pas la phase 0 sur le corpus d'artefacts : elle en elargit le seul compte
// dont la precision decide, celui des deux tags AMBIGU.
func TestPhase0Bis_BalayageLargeDesTagsAmbigus(t *testing.T) {
	limite := tailleDuBalayage(t)
	cache := cacheDesFilms(t)
	dirs, err := os.ReadDir(cache)
	if err != nil {
		t.Fatalf("lecture du cache de films : %v", err)
	}
	var courts []string
	for _, d := range dirs {
		if d.IsDir() {
			courts = append(courts, d.Name())
		}
	}
	sort.Strings(courts)
	if limite > 0 && limite < len(courts) {
		courts = courts[:limite]
	}
	b := nouveauBalayage()
	for _, court := range courts {
		balayerUnFilm(&b, cache, court)
	}
	if b.films == 0 {
		t.Fatal("aucun film decodable dans le cache")
	}
	publierBalayage(t, &b, len(courts))
}

// balayerUnFilm decode un film et ventile ses morts. Un echec n'arrete pas le balayage : un
// film tronque est le cas NOMINAL d'un cache partiel, et il est COMPTE plutot que tu.
func balayerUnFilm(b *balayage, cache, court string) {
	src, err := filmsource.LoadDir(filepath.Join(cache, court), nil)
	if err != nil {
		b.echecs++
		return
	}
	res, err := killsource.Decode(context.Background(), court, src, nil)
	if err != nil {
		b.echecs++
		return
	}
	b.films++
	b.compterResultat(res, court)
}

// publierBalayage rend les comptes et la borne superieure de la regle de trois.
//
// LA BORNE EST LA RAISON D ETRE DE CE FICHIER : un numerateur nul ne se lit pas seul. Tant que
// `3 / grenades` reste au-dessus du seuil du plan, la mesure ne tranche PAS le gate — elle dit
// seulement qu'on n'a rien vu.
func publierBalayage(t *testing.T, b *balayage, demandes int) {
	t.Helper()
	t.Logf("PHASE 0 bis — %d film(s) demande(s), %d decode(s), %d echec(s) ; %d mort(s) au total",
		demandes, b.films, b.echecs, b.morts)
	b.publierVentilation(t)
	t.Logf("  dont VALIDE %d, AMBIGU %d, autre statut %d", b.valides, b.ambigues, b.autresStatuts)
	t.Logf("AMBIGU : %d / %d = %.2f %% des morts par grenade, sur %d film(s). "+
		"Borne haute a 95 %% (Wilson) : %.2f %%. SEUIL DU PLAN : 1 %%.",
		b.ambigues, b.grenades, part(b.ambigues, b.grenades), len(b.filmsAvecAmbiguite),
		borneHauteWilson(b.ambigues, b.grenades))
	b.publierFilmsAmbigus(t)
}

// TestBorneHauteWilson — le calcul de precision qui decide du gate doit lui-meme etre verifie.
// Il tourne en CI : aucun corpus, aucune garde.
func TestBorneHauteWilson(t *testing.T) {
	cas := []struct {
		nom     string
		x, n    int
		attendu float64
	}{
		{"corpus d'artefacts, aucune occurrence", 0, 63, 5.75},
		{"balayage large, une occurrence", 1, 696, 0.81},
		{"balayage large, aucune occurrence", 0, 696, 0.55},
		{"denominateur nul", 0, 0, 0},
	}
	for _, c := range cas {
		got := borneHauteWilson(c.x, c.n)
		if diff := got - c.attendu; diff > 0.02 || diff < -0.02 {
			t.Errorf("%s : borne %d/%d = %.4f %%, attendu %.2f %%", c.nom, c.x, c.n, got, c.attendu)
		}
	}
}
