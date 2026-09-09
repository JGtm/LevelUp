package filmdec

// biped_creation_film_test.go — L'INSTRUMENT DE MESURE du lecteur de création de bipède, sur un
// FILM RÉEL.
//
// Les tests bit-exacts (biped_creation_test.go) prouvent que le lecteur lit aux bonnes
// positions ; ils ne peuvent rien dire du RENDEMENT ni du plancher de faux positifs sur un vrai
// flux. C'est ce que mesure cet instrument, et il reproduit le protocole du sondage E2
// (`.ai/V7.5/film_re/SONDAGE_E2_BIPEDE_INDEX_2026-09-08.md` §2.3) :
//
//	mesure   la bande de slots `ti=35` du film
//	témoin   une bande FANTÔME de même cardinalité, décalée de 4 096 (des slots qu'aucun bipède
//	         n'occupe), passée par le MÊME décodeur — sans quoi le témoin contrôlerait une
//	         variante du code et non le code
//
// Le sondage a mesuré le témoin fantôme à ZÉRO lecture sur ses cinq films.
//
// Il est SAUTÉ sans son film : le dépôt ne versionne aucun film (des centaines de Mio), et la CI
// n'en a aucun. Usage :
//
//	BIPED_CREATION_FILM=<cache>/film_chunks/d9781168 \
//	  go test -count=1 -run TestCreationBipedeSurFilm -v ./internal/analysis/filmdec/

import (
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
)

// bipedCreationFilmEnv nomme le répertoire de chunks du film à mesurer.
const bipedCreationFilmEnv = "BIPED_CREATION_FILM"

// bandeFantomeDe rend une bande de MÊME cardinalité faite de slots qu'aucun bipède n'occupe :
// le décalage de 4 096 sort du domaine observé sans sortir des 13 bits du champ.
func bandeFantomeDe(band SlotBand) SlotBand {
	out := map[uint32]bool{}
	for _, s := range band.Slots() {
		f := (s + 4096) & 0x1fff
		if !band.Has(f) {
			out[f] = true
		}
	}
	return NewSlotBand(out)
}

func TestCreationBipedeSurFilm(t *testing.T) {
	dir := os.Getenv(bipedCreationFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", bipedCreationFilmEnv)
	}
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("chargement du film %s : %v", dir, err)
	}
	release := LockProcessDecode()
	defer release()

	fc := NewFilmContext(film)
	recs, st, err := ScanBipedCreations(fc)
	if err != nil {
		t.Fatalf("balayage des créations de bipède : %v", err)
	}
	fantomes, stF, err := ScanBipedCreationsForBand(fc, bandeFantomeDe(fc.BipedSlots()))
	if err != nil {
		t.Fatalf("balayage du témoin fantôme : %v", err)
	}

	t.Logf("== créations de bipède · %s ==", dir)
	t.Logf("bande=%d slots · ancres=%d · acceptés=%d · shapeBad=%d · signatureMismatch=%d "+
		"(mot alternatif modal %#x ×%d) · gateClosed=%d · tronqués=%d",
		st.Slots, st.Anchors, st.Accepted, st.ShapeBad, st.SignatureMismatch,
		st.OtherWord, st.OtherWordCount, st.GateClosed, st.Truncated)
	t.Logf("index de participant lus : %s", histogrammeDesIndex(recs))
	t.Logf("vies distinctes (slot, génération) désignées : %d", len(viesDesignees(recs)))
	t.Logf("TÉMOIN FANTÔME (bande de %d slots, décalage 4096) : ancres=%d acceptés=%d",
		stF.Slots, stF.Anchors, len(fantomes))

	if st.Accepted == 0 {
		t.Errorf("aucune création lue sur %d ancres : le gate de signature ne trouve jamais "+
			"%#x — ce film porte-t-il une autre représentation de bipède ? (mot alternatif "+
			"modal %#x ×%d)", st.Anchors, BipedRepresentationName, st.OtherWord, st.OtherWordCount)
	}
	if len(fantomes) != 0 {
		t.Errorf("le témoin fantôme rend %d lecture(s) : le gate a un plancher de faux positifs "+
			"non nul, la couverture mesurée n'est pas interprétable telle quelle", len(fantomes))
	}
}

// histogrammeDesIndex rend « index:compte » en ordre croissant d'index — reproductible.
func histogrammeDesIndex(recs []BipedCreation) string {
	h := map[uint32]int{}
	for _, r := range recs {
		if r.HasIndex {
			h[r.ParticipantIndex]++
		}
	}
	cles := make([]uint32, 0, len(h))
	for k := range h {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool { return cles[i] < cles[j] })
	parts := make([]string, 0, len(cles))
	for _, k := range cles {
		parts = append(parts, strconv.Itoa(int(k))+":"+strconv.Itoa(h[k]))
	}
	return strings.Join(parts, " ")
}

// viesDesignees rend l'ensemble des clés de vie que les lectures désignent. Deux lectures d'une
// MÊME vie sont attendues (le sondage en a mesuré 14 sur trois films, toutes concordantes) :
// c'est le dénominateur honnête d'une couverture par vie.
func viesDesignees(recs []BipedCreation) map[uint32]bool {
	out := map[uint32]bool{}
	for _, r := range recs {
		if r.HasIndex {
			out[r.LifeKey()] = true
		}
	}
	return out
}
