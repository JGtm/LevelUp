package grammar

// marche_determinisme_test.go — T6 DE LA SPECIFICATION DE LA REPRESENTATION INTERMEDIAIRE (ADR
// 0037) : deux marches du meme film rangent la meme structure, a l octet de son empreinte ; et deux
// films marches en meme temps rangent ce qu ils rangent l un apres l autre. Le second test tourne
// sous `-race` dans le job CI `film-race`, dont le filtre (`-run TestDeuxFilmsEnParallele`) le
// prend par son nom.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"sync"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// bobinesAEmpreinte : les deux bobines du depot qui portent des trames delta ET des images-cles,
// de builds differents.
var bobinesAEmpreinte = [2]string{"ks_000d5950", "ks_e5adf7b2"}

// chargerUneBobine charge la bobine du depot nommee `nom` dans [frameClosureBobines].
func chargerUneBobine(t *testing.T, nom string) *source.Film {
	t.Helper()
	for _, bo := range frameClosureBobines() {
		if bo.nom != nom {
			continue
		}
		film, err := source.LoadDir(bo.dir, nil)
		if err != nil {
			t.Fatalf("LoadDir %s : %v", bo.dir, err)
		}
		return film
	}
	t.Fatalf("bobine %s inconnue", nom)
	return nil
}

// empreinteDeLaStructure marche les deux phases d un film sous un contexte NEUF et rend l empreinte
// de tout ce que la structure range : en-tete, vues, verdict, records, composants, et le nombre
// d entites liees apres chaque trame. Elle ne touche pas `t` : elle se rend depuis une goroutine.
func empreinteDeLaStructure(film *source.Film) (string, error) {
	h := sha256.New()
	fc := contexteDeBobine(film)
	for p, err := range fc.ImagesCles() {
		if err != nil {
			return "", err
		}
		ecrirePaquet(h, p)
	}
	for p, err := range fc.Trames(nil) {
		if err != nil {
			return "", err
		}
		ecrirePaquet(h, p)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ecrirePaquet ecrit un paquet de la structure dans l empreinte.
func ecrirePaquet(h hash.Hash, p *lecture.Paquet) {
	liees := -1
	if p.Entites != nil {
		liees = p.Entites.Liees()
	}
	fmt.Fprintf(h, "%d/%d/%d/%d/%d|%+v|%+v|%+v|%+v|%v|%v|%d\n", p.Chunk, p.Index, p.Type, p.TS, p.Debut,
		p.VueA, p.VueB, p.VueC, p.Fermeture, p.Records, p.Comps, liees)
}

// TestLaStructureEstDeterministe : deux marches du meme film, chacune sous un contexte neuf, rendent
// la meme empreinte ; les deux bobines en rendent deux differentes (le temoin discrimine).
func TestLaStructureEstDeterministe(t *testing.T) {
	var empreintes [2]string
	for i, nom := range bobinesAEmpreinte {
		film := chargerUneBobine(t, nom)
		a, err := empreinteDeLaStructure(film)
		if err != nil {
			t.Fatalf("%s : %v", nom, err)
		}
		b, err := empreinteDeLaStructure(film)
		if err != nil {
			t.Fatalf("%s : %v", nom, err)
		}
		if a != b {
			t.Errorf("%s : deux marches, deux empreintes\n  %s\n  %s", nom, a, b)
		}
		empreintes[i] = a
	}
	if empreintes[0] == empreintes[1] {
		t.Fatalf("les deux bobines rendent la MEME empreinte : le temoin ne discrimine rien (%s)", empreintes[0])
	}
}

// TestDeuxFilmsEnParalleleLaStructure : deux films de builds differents marches EN MEME TEMPS, chacun
// sous son contexte, rangent la structure qu ils rangent l un apres l autre (meme raisonnement que
// [TestDeuxFilmsEnParallele] : le temoin est la serie).
func TestDeuxFilmsEnParalleleLaStructure(t *testing.T) {
	var films [2]*source.Film
	var serie, parallele [2]string
	var erreurs [2]error
	for i, nom := range bobinesAEmpreinte {
		films[i] = chargerUneBobine(t, nom)
		e, err := empreinteDeLaStructure(films[i])
		if err != nil {
			t.Fatalf("%s : %v", nom, err)
		}
		serie[i] = e
	}
	var wg sync.WaitGroup
	for i := range films {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			parallele[i], erreurs[i] = empreinteDeLaStructure(films[i])
		}(i)
	}
	wg.Wait()
	for i, nom := range bobinesAEmpreinte {
		if erreurs[i] != nil || parallele[i] != serie[i] {
			t.Errorf("%s : en serie %s, en parallele %s (%v) — un etat est partage entre les deux marches",
				nom, serie[i], parallele[i], erreurs[i])
		}
	}
}
