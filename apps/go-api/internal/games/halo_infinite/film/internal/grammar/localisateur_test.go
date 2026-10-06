package grammar

// localisateur_test.go — LES DEUX ORDRES DU LOCALISATEUR UNIQUE, eprouves sur la bobine reelle a
// paquets delta (`../facts/killsource/testdata/minibobine_000d5950`, cf. `object_deaths_test.go`).

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// paquetDeLocalisation : un paquet a evenements et le monde d avant lui.
type paquetDeLocalisation struct {
	pay []byte
	w   *World
}

// parcourirPaquetsAEvenements appelle `f` sur chaque paquet a evenements de la bobine, apres sa
// marche, dans le monde de la marche des trames et sous son cadre, la generation stricte posee
// comme la cuisson la pose.
func parcourirPaquetsAEvenements(t *testing.T, f func(p paquetDeLocalisation, cfg FrameConfig)) {
	t.Helper()
	film, err := source.LoadDir(bobineMarcheDir(), nil)
	if err != nil {
		t.Fatalf("bobine illisible : %v", err)
	}
	fc := NewFilmContext(film)
	bal := fc.ProfilDeBalayage()
	bal.Grammaire.GenerationStricte = true
	fc.PoserProfilDeBalayage(bal)
	m, err := fc.nouveauMarcheurDesTrames(nil)
	if err != nil {
		t.Fatalf("marche des trames : %v", err)
	}
	m.parcourir(func(tr *trameLue) bool {
		if tr.paquet.VueA.Etat == lecture.VueArretee {
			f(paquetDeLocalisation{pay: tr.paquet.Payload, w: m.monde}, m.cfg)
		}
		return true
	})
}

// TestLocaliserBoucleDeRecordsSuitLOrdreDuSite : sur chaque paquet a evenements, l ordre de la
// cuisson rend la premiere signature stricte telle quelle, et celui des sites qui lisent les morts
// la rend si sa generation est celle du monde, sinon le repli a largeur libre avec son verdict. La
// bobine porte les deux cas (signature, repli) : sans eux, le test ne garderait rien.
func TestLocaliserBoucleDeRecordsSuitLOrdreDuSite(t *testing.T) {
	signatures, replis := 0, 0
	parcourirPaquetsAEvenements(t, func(p paquetDeLocalisation, cfg FrameConfig) {
		strict := marchLocateStrict(p.pay, p.w, cfg)
		if s, libre := LocaliserBoucleDeRecords(p.pay, p.w, cfg, SignatureStricte); s != strict || libre {
			t.Errorf("ordre de la cuisson : (%d, %v), attendu (%d, false)", s, libre, strict)
		}
		attendu, attenduLibre := strict, false
		if strict < 0 || !generationDuMonde(p, cfg, strict) {
			attendu = marchLocateFallback(p.pay, p.w, cfg)
			attenduLibre = attendu >= 0
		}
		s, libre := LocaliserBoucleDeRecords(p.pay, p.w, cfg, SignaturePuisLargeurLibre)
		if s != attendu || libre != attenduLibre {
			t.Errorf("ordre des marches : (%d, %v), attendu (%d, %v)", s, libre, attendu, attenduLibre)
		}
		if libre {
			replis++
		} else if s >= 0 {
			signatures++
		}
	})
	if signatures == 0 || replis == 0 {
		t.Fatalf("la bobine ne porte pas les deux cas : %d signature(s), %d repli(s)", signatures, replis)
	}
}

// generationDuMonde : le delta essaye a `s` porte-t-il la generation que le monde lie au slot ?
func generationDuMonde(p paquetDeLocalisation, cfg FrameConfig, s int) bool {
	rec, _, ok := TryDeltaAt(p.pay, s, p.w, cfg)
	return ok && p.w.GenerationMatches(rec.ID, cfg.Profil.Grammaire.GenerationStricte)
}

// TestLocaliserBoucleDeRecordsControleLaGenerationDesMarches : quand le monde lie le slot de
// signature a une AUTRE generation, l ordre des marches ecarte la signature stricte, l ordre de la
// cuisson la garde.
func TestLocaliserBoucleDeRecordsControleLaGenerationDesMarches(t *testing.T) {
	essais := 0
	parcourirPaquetsAEvenements(t, func(p paquetDeLocalisation, cfg FrameConfig) {
		strict := marchLocateStrict(p.pay, p.w, cfg)
		if essais > 0 || strict < 0 || !generationDuMonde(p, cfg, strict) {
			return
		}
		rec, _, _ := TryDeltaAt(p.pay, strict, p.w, cfg)
		ti, _ := p.w.ArchetypeForSlot(marchSignatureSlot)
		snap := p.w.Snapshot()
		defer p.w.Restore(snap)
		p.w.BindFull(rec.ID^(1<<30), ti)
		essais++
		if s, _ := LocaliserBoucleDeRecords(p.pay, p.w, cfg, SignatureStricte); s != strict {
			t.Errorf("ordre de la cuisson : %d, attendu la signature stricte %d", s, strict)
		}
		if s, _ := LocaliserBoucleDeRecords(p.pay, p.w, cfg, SignaturePuisLargeurLibre); s == strict {
			t.Errorf("ordre des marches : la signature %d d une autre generation est gardee", s)
		}
	})
	if essais == 0 {
		t.Fatal("aucune signature stricte a la generation du monde sur la bobine : le test ne garde rien")
	}
}

// TestLaCuissonNeDemarreQueSurLaSignatureStricte : le debut de liste de la cuisson
// ([localiserLaListe]) attribue a la signature n est jamais que la premiere signature stricte —
// jamais une position du repli a largeur libre, que la cuisson n essaie pas.
func TestLaCuissonNeDemarreQueSurLaSignatureStricte(t *testing.T) {
	sansSignature := 0
	parcourirPaquetsAEvenements(t, func(p paquetDeLocalisation, cfg FrameConfig) {
		strict := marchLocateStrict(p.pay, p.w, cfg)
		if strict < 0 {
			sansSignature++
		}
		if debut, comment := localiserLaListe(p.pay, p.w, cfg); comment == lecture.DebutParSignature && debut != strict {
			t.Errorf("debut %d attribue a la signature, signature stricte %d", debut, strict)
		}
	})
	if sansSignature == 0 {
		t.Fatal("aucun paquet sans signature stricte sur la bobine : le test ne garde rien")
	}
}

// TestLeCanalDesMortsSuitLOrdreDesMarches : le site du canal des morts ([debutRecupere]) rend, sur
// chaque paquet a evenements, la position et le verdict de l ordre des marches — y compris sur les
// paquets ou l ordre de la cuisson rend autre chose (repli a largeur libre, signature d une autre
// generation). La bobine porte ces ecarts : sans eux, le test ne garderait rien.
// MUTATION — [debutRecupere] passe a l ordre de la cuisson : ROUGE.
func TestLeCanalDesMortsSuitLOrdreDesMarches(t *testing.T) {
	ecarts := 0
	parcourirPaquetsAEvenements(t, func(p paquetDeLocalisation, cfg FrameConfig) {
		attendu, attenduLibre := LocaliserBoucleDeRecords(p.pay, p.w, cfg, SignaturePuisLargeurLibre)
		if strict, _ := LocaliserBoucleDeRecords(p.pay, p.w, cfg, SignatureStricte); strict != attendu {
			ecarts++
		}
		if s, libre := debutRecupere(p.pay, p.w, cfg); s != attendu || libre != attenduLibre {
			t.Errorf("canal des morts : (%d, %v), attendu l ordre des marches (%d, %v)", s, libre, attendu,
				attenduLibre)
		}
	})
	if ecarts == 0 {
		t.Fatal("aucun paquet ou les deux ordres different sur la bobine : le test ne garde rien")
	}
}

// TestLaLargeurDeSignatureSuitLeCadre : la largeur de la signature est celle que les ecrivains
// donnent au delta sous le cadre — 35 bits au cadre par defaut, un bit par bit d identifiant bas,
// 32 de plus avec le mot facultatif. MUTATION : rendre une largeur fixe — ROUGE.
func TestLaLargeurDeSignatureSuitLeCadre(t *testing.T) {
	for _, c := range []struct {
		idLowBits int
		extra     bool
		largeur   int
	}{{13, false, 35}, {11, false, 33}, {14, false, 36}, {13, true, 67}} {
		cfg := FrameConfig{IDLowBits: c.idLowBits, HasExtraFields: c.extra}
		if got := largeurDeSignature(cfg); got != c.largeur {
			t.Errorf("IDLowBits %d, mot facultatif %v : largeur %d, attendu %d", c.idLowBits, c.extra, got, c.largeur)
		}
	}
}
