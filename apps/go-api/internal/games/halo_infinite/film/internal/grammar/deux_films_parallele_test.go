package grammar

// deux_films_parallele_test.go — DEUX FILMS SE DECODENT EN PARALLELE, ET RENDENT LA MEME CHOSE
// QU EN SERIE (lot 2.3 du PLAN_DECODEUR_FILM, item 2.3.3 ; critere S2 du plan).
//
// # CE QUE CE TEST PROUVE, ET CE QU IL NE PROUVE PAS
//
// IL PROUVE que le decodeur n a plus d etat partage : deux films de BUILDS DIFFERENTS decodes
// dans deux goroutines rendent, a l octet, ce que les memes decodages rendent l un apres l autre.
// C est le critere qui autorisait le retrait du verrou de paquet — le ratchet frere
// (`archlint/filmdec_package_vars_test.go`) mesure la cause (zero variable de paquet ecrite),
// celui-ci mesure l EFFET.
//
// IL NE PROUVE PAS que le decodage est thread-safe au sens large : un MEME `FilmContext` ou un
// MEME `Observation` partage entre deux goroutines reste un objet mutable ordinaire, et rien ici
// ne le permet. Ce que le lot a retire, c est l etat implicite du PROCESSUS ; l etat explicite
// d un balayage appartient a ce balayage.
//
// # POURQUOI LE CONTROLE SEQUENTIEL EST PASSE EN PREMIER
//
// Comparer les deux decodages paralleles entre eux ne dirait rien : ils pourraient se contaminer
// de la meme facon. Le temoin est la SERIE — deux decodages qui ne peuvent pas se croiser.
//
// # SOUS `-race`
//
// Ce paquet ne tire PAS DuckDB : `go test -race` y tourne sans `-gcflags=all=-d=checkptr=0`.
// La commande du gate :
//
//	go test -race -run TestDeuxFilmsEnParallele ./internal/games/halo_infinite/film/filmdec/

import (
	"fmt"
	"path/filepath"
	"sort"
	"sync"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// deuxFilmsDeBuildsDifferents : les deux mini-bobines du parallelisme. Elles sont de BUILDS
// DIFFERENTS a dessein — c est la condition qui fait diverger les profils (largeurs MPP, cadre
// d image-cle), donc celle ou un etat partage se verrait.
var deuxFilmsDeBuildsDifferents = [2]string{
	"a521164d", // HI_1_4_1
	"fb1a1a72", // HI_1_13_0, le build de l executable desassemble
}

// TestDeuxFilmsEnParallele — LE CRITERE S2.
func TestDeuxFilmsEnParallele(t *testing.T) {
	films := [2]*source.Film{}
	for i, court := range deuxFilmsDeBuildsDifferents {
		dir := filepath.Join("..", "..", "replay", "testdata", "minifilm_"+court)
		f, err := source.LoadDir(dir, nil)
		if err != nil {
			t.Fatalf("bobine %s : %v", court, err)
		}
		films[i] = f
	}

	// TEMOIN : les deux decodages, l un APRES l autre.
	var serie [2]string
	for i, f := range films {
		serie[i] = empreinteDeDecodage(t, f)
	}

	// LE TEMOIN DOIT DISCRIMINER. Deux empreintes egales rendraient le test vacuant : il
	// passerait meme si les deux goroutines decodaient le meme film. Les bobines sont de builds
	// differents, leurs empreintes doivent l etre aussi.
	if serie[0] == serie[1] {
		t.Fatalf("les deux bobines rendent la MEME empreinte : le temoin ne discrimine rien.\n  %s", serie[0])
	}
	for i, court := range deuxFilmsDeBuildsDifferents {
		t.Logf("bobine %s : %s", court, serie[i])
	}

	// MESURE : les deux decodages EN MEME TEMPS.
	var parallele [2]string
	var wg sync.WaitGroup
	for i := range films {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			parallele[i] = empreinteDeDecodage(t, films[i])
		}(i)
	}
	wg.Wait()

	for i, court := range deuxFilmsDeBuildsDifferents {
		if parallele[i] != serie[i] {
			t.Errorf("bobine %s : le decodage PARALLELE ne rend pas ce que le decodage EN SERIE "+
				"rend.\n  en serie    : %s\n  en parallele: %s\n"+
				"Un etat est partage entre les deux decodages — c est exactement ce que le lot 2.3 "+
				"a retire, et ce que `archlint/filmdec_package_vars_test.go` interdit.",
				court, serie[i], parallele[i])
		}
	}
}

// empreinteDeDecodage decode un film par les chemins de PRODUCTION qui portent le profil (bande
// de slots, decoupage d i0, fermeture d image-cle par archetype) et rend une empreinte stable et
// lisible de ce qu ils ont vu.
//
// LA FERMETURE D IMAGE-CLE EST LE BON TEMOIN : elle pose le decoupage MPP du FORMAT du film sur
// le contexte, puis marche CHAQUE record de CHAQUE archetype sous ce decoupage — et elle publie,
// par archetype, combien de records elle a vus, combien elle a fermes et sur quel composant elle
// a bute. Une largeur qui vient d un autre film deplace ces trois nombres, archetype par
// archetype : c est exactement ce qu un etat de processus melangeait. Le test refuse d ailleurs
// deux empreintes egales, pour ne pas passer sur un temoin muet (mesure : les deux bobines
// different des le premier archetype, ti=0 pour l une, ti=1 pour l autre).
func empreinteDeDecodage(t *testing.T, f *source.Film) string {
	t.Helper()
	fc := NewFilmContext(f)
	stats, err := KeyframeClosure(fc)
	if err != nil {
		return "fermeture illisible : " + err.Error()
	}
	tis := make([]int, 0, len(stats))
	for ti := range stats {
		tis = append(tis, int(ti))
	}
	sort.Ints(tis)
	out := fmt.Sprintf("slots=%d decoupage=%v", fc.BipedSlots().Count(), fc.ProfilDeBalayage().MPP)
	for _, ti := range tis {
		s := stats[uint32(ti)] //nolint:gosec // index d archetype, borne par le registre
		out += fmt.Sprintf(" | ti=%d total=%d ferme=%d bloquant=%q",
			ti, s.Total, s.Closed, s.Blocking)
	}
	return out
}
