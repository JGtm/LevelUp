package grammar

// localisateur_test.go — LES ORDRES DU LOCALISATEUR UNIQUE, eprouves sur les bobines reelles a
// paquets delta (`../facts/killsource/testdata/minibobine_000d5950` et `minibobine_e5adf7b2`, cf.
// `object_deaths_test.go`), et sa signature haute frequence sur des paquets de la premiere dont le
// premier delta est recopie sur un autre objet de l archetype, comme l ecrirait le jeu.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// marchLocateStrict rend la premiere position qui porte la signature du slot 123, -1 si aucune :
// le premier etage de la cuisson ([SignatureStricte]). Aide des tests et des sondes de recherche.
func marchLocateStrict(pay []byte, w *World, cfg FrameConfig) int {
	s, _ := marchLocateSignatures(pay, w, cfg)
	return s
}

// bobineE5adf7b2Dir : la seconde bobine reelle ; ses paquets a evenements ne portent aucune
// signature du slot 123.
func bobineE5adf7b2Dir() string {
	return filepath.Join("..", "facts", "killsource", "testdata", "minibobine_e5adf7b2")
}

// paquetDeLocalisation : un paquet a evenements et le monde d avant lui.
type paquetDeLocalisation struct {
	pay []byte
	w   *World
}

// parcourirPaquetsAEvenements appelle `f` sur chaque paquet a evenements de la bobine `dir`, dans
// le monde de la marche des morts d objet, sous le cadre qu elle calibre et la generation stricte
// des marches qui lisent les morts. Le monde n a pas de table anticipee : aucun candidat de tete,
// la cuisson n y trouve aucune chaine ([candidatsDeTete]).
func parcourirPaquetsAEvenements(t *testing.T, dir string, f func(p paquetDeLocalisation, cfg FrameConfig)) {
	t.Helper()
	parcourirLaBobine(t, dir, false, f)
}

// parcourirLaBobine est [parcourirPaquetsAEvenements] ; `avecTable` pose sur le monde la table
// anticipee du film, comme la marche des trames de la cuisson : les candidats de tete existent.
func parcourirLaBobine(t *testing.T, dir string, avecTable bool, f func(p paquetDeLocalisation, cfg FrameConfig)) {
	t.Helper()
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("bobine illisible : %v", err)
	}
	fc := NewFilmContext(film)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	kfs, deltas := marchPacketsOf(fc)
	cfg, _, _, _ := calibrateFrameConfig(reg, kfs, deltas, fc.CadreDeBalayage())
	cfg.Profil.Grammaire.GenerationStricte = true
	tl := newMarchTimeline(reg, kfs)
	if avecTable {
		tl.w.PoserTableAnticipee(ConstruireTableAnticipee(fc))
	}
	for _, d := range deltas {
		w := tl.advanceTo(d.timestampUS)
		if marchHasEvents(d.payload) {
			f(paquetDeLocalisation{pay: d.payload, w: w}, cfg)
		}
	}
}

// TestLocaliserBoucleDeRecordsSuitLOrdreDuSite : sur chaque paquet a evenements, l ordre de la
// cuisson rend la premiere signature du slot 123 telle quelle, et celui des marches la rend si sa
// generation est celle du monde, sinon le repli a largeur libre avec son verdict. La bobine porte
// les deux cas (signature, repli) : sans eux, le test ne garderait rien.
func TestLocaliserBoucleDeRecordsSuitLOrdreDuSite(t *testing.T) {
	signatures, replis := 0, 0
	parcourirPaquetsAEvenements(t, bobineMarcheDir(), func(p paquetDeLocalisation, cfg FrameConfig) {
		strict := marchLocateStrict(p.pay, p.w, cfg)
		if strict >= 0 {
			if rec, _, _ := TryDeltaAt(p.pay, strict, p.w, cfg); rec.Slot != marchSignatureSlot {
				t.Errorf("signature du slot 123 a %d sur le slot %d", strict, rec.Slot)
			}
		}
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

// TestLaSignatureHauteFrequenceNeLitQueLArchetype : sur les deux bobines reelles, aucun paquet ne
// porte de signature haute frequence la ou le slot 123 n en porte pas. `e5adf7b2` porte pourtant,
// dans des paquets sans signature du slot 123, des deltas de 35 bits a composant unique sur
// d autres archetypes : un predicat qui ne reconnaitrait pas l archetype les prendrait.
func TestLaSignatureHauteFrequenceNeLitQueLArchetype(t *testing.T) {
	for _, dir := range []string{bobineMarcheDir(), bobineE5adf7b2Dir()} {
		sans123 := 0
		parcourirPaquetsAEvenements(t, dir, func(p paquetDeLocalisation, cfg FrameConfig) {
			if marchLocateStrict(p.pay, p.w, cfg) < 0 {
				sans123++
			}
			if s, _ := LocaliserBoucleDeRecords(p.pay, p.w, cfg, SignatureHauteFrequence); s >= 0 {
				rec, _, _ := TryDeltaAt(p.pay, s, p.w, cfg)
				t.Errorf("%s : signature haute frequence a %d sur le slot %d (archetype %d)",
					filepath.Base(dir), s, rec.Slot, rec.TypeIndex)
			}
		})
		if sans123 == 0 {
			t.Fatalf("%s : aucun paquet sans signature du slot 123 : le test ne garde rien", filepath.Base(dir))
		}
	}
}

// poserBits ecrit `v` sur `n` bits a `pos`, bit de poids fort d abord (la convention de
// [source.BitAt]).
func poserBits(pay []byte, pos, n int, v uint64) {
	for i := 0; i < n; i++ {
		p := pos + i
		masque := byte(1) << (7 - uint(p&7))
		if (v>>uint(n-1-i))&1 == 1 {
			pay[p>>3] |= masque
		} else {
			pay[p>>3] &^= masque
		}
	}
}

// recopieSurUnAutreSlot rend une copie du paquet ou le delta de signature (slot 123, a `s`) porte
// le premier slot `x > 123` que le monde ne lie pas, et lie `x` a l archetype du slot 123 sous
// l identifiant `id` ; `restaurer` rend le monde d avant. Faux si le slot 123 n est pas lie a
// l archetype `high-frequency`.
func recopieSurUnAutreSlot(p paquetDeLocalisation, cfg FrameConfig, s int,
	id func(rec FrameRecord, x uint32) uint32) (pay []byte, x uint32, restaurer func(), ok bool) {
	rec, _, _ := TryDeltaAt(p.pay, s, p.w, cfg)
	if ti, lie := p.w.ArchetypeForSlot(marchSignatureSlot); !lie || ti != archetypeHauteFrequence {
		return nil, 0, nil, false
	}
	for x = marchSignatureSlot + 1; ; x++ {
		if _, lie := p.w.ArchetypeForSlot(x); !lie {
			break
		}
	}
	pay = append([]byte(nil), p.pay...)
	debutID := s + 1 // le type d un DELTA tient en un bit ([readRecordType])
	if cfg.HasExtraFields {
		debutID += 32
	}
	poserBits(pay, debutID, cfg.IDLowBits, uint64(x-cfg.IDBase))
	snap := p.w.Snapshot()
	p.w.BindFull(id(rec, x), archetypeHauteFrequence)
	return pay, x, func() { p.w.Restore(snap) }, true
}

// TestLaSignatureHauteFrequenceLocaliseUnAutreObjetDeLArchetype : l ecrivain des objets de
// l archetype `high-frequency` est le meme pour tous ; un premier delta du slot 123 recopie sur un
// autre slot que le monde lie a cet archetype, a la meme generation, est la meme signature. Le slot
// 123 n en porte alors aucune : l ordre de la cuisson et le dernier etage de la cuisson le disent,
// les marches et la cuisson demarrent sur lui. Sous une AUTRE generation, les marches l ecartent.
func TestLaSignatureHauteFrequenceLocaliseUnAutreObjetDeLArchetype(t *testing.T) {
	memeGeneration := func(rec FrameRecord, x uint32) uint32 { return rec.ID&^0x3fffffff | x }
	autreGeneration := func(rec FrameRecord, x uint32) uint32 { return (rec.ID^1<<30)&^0x3fffffff | x }
	vecteurs, anterieures := 0, 0
	parcourirPaquetsAEvenements(t, bobineMarcheDir(), func(p paquetDeLocalisation, cfg FrameConfig) {
		s := marchLocateStrict(p.pay, p.w, cfg)
		if s < 0 || !generationDuMonde(p, cfg, s) {
			return
		}
		pay, x, restaurer, ok := recopieSurUnAutreSlot(p, cfg, s, memeGeneration)
		if !ok {
			return
		}
		defer restaurer()
		if rec, _, lu := TryDeltaAt(pay, s, p.w, cfg); !lu || rec.Slot != x {
			t.Fatalf("recopie a %d : slot relu %d, attendu %d", s, rec.Slot, x)
		}
		if got, _ := LocaliserBoucleDeRecords(pay, p.w, cfg, SignatureStricte); got != -1 {
			t.Errorf("slot %d : la signature du slot 123 est encore vue a %d", x, got)
		}
		got, _ := LocaliserBoucleDeRecords(pay, p.w, cfg, SignatureHauteFrequence)
		if got != s {
			// Une position ANTERIEURE peut porter la meme forme sur un objet de l archetype (le
			// slot recopie, ou un autre que la bobine lie a l archetype) : c est la premiere
			// signature haute frequence, et le vecteur ne dit rien de plus.
			if got < 0 || got > s || !premiereSignatureAnterieure(pay, got, p.w, cfg) {
				t.Errorf("slot %d : signature haute frequence a %d, attendue a %d", x, got, s)
			}
			anterieures++
			return
		}
		vecteurs++
		if got, libre := LocaliserBoucleDeRecords(pay, p.w, cfg, SignaturePuisLargeurLibre); got != s || libre {
			t.Errorf("slot %d : ordre des marches (%d, %v), attendu (%d, false)", x, got, libre, s)
		}
		if d, comment := localiserLaListe(pay, p.w, cfg); d != s || comment != lecture.DebutParSignature {
			t.Errorf("slot %d : cuisson (%d, %v), attendu (%d, signature)", x, d, comment, s)
		}
		restaurer()
		pay, _, restaurer, _ = recopieSurUnAutreSlot(p, cfg, s, autreGeneration)
		if got, _ := LocaliserBoucleDeRecords(pay, p.w, cfg, SignaturePuisLargeurLibre); got == s {
			t.Errorf("slot %d : la signature d une autre generation est gardee a %d", x, got)
		}
	})
	if vecteurs == 0 {
		t.Fatal("aucun paquet a signature du slot 123 lie a l archetype high-frequency : le test ne garde rien")
	}
	t.Logf("%d vecteur(s), %d precede(s) par une autre signature haute frequence", vecteurs, anterieures)
}

// premiereSignatureAnterieure : le delta lu a `s` est-il une signature haute frequence (forme de la
// signature, autre slot que le 123, archetype `high-frequency` a la generation du monde) ?
func premiereSignatureAnterieure(pay []byte, s int, w *World, cfg FrameConfig) bool {
	rec, fin, lu := TryDeltaAt(pay, s, w, cfg)
	return lu && rec.Slot != marchSignatureSlot && formeDeSignature(rec, s, fin) &&
		signeLaHauteFrequence(rec, w, cfg)
}

// generationDuMonde : le delta essaye a `s` porte-t-il la generation que le monde lie au slot ?
func generationDuMonde(p paquetDeLocalisation, cfg FrameConfig, s int) bool {
	rec, _, ok := TryDeltaAt(p.pay, s, p.w, cfg)
	return ok && aLaGenerationDuMonde(rec, p.w, cfg)
}

// TestLocaliserBoucleDeRecordsControleLaGenerationDesMarches : quand le monde lie le slot de
// signature a une AUTRE generation, l ordre des marches ecarte la signature stricte, l ordre de la
// cuisson la garde.
func TestLocaliserBoucleDeRecordsControleLaGenerationDesMarches(t *testing.T) {
	essais := 0
	parcourirPaquetsAEvenements(t, bobineMarcheDir(), func(p paquetDeLocalisation, cfg FrameConfig) {
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
// ([localiserLaListe]) attribue a la signature n est jamais qu une signature stricte — celle du
// slot 123, ou, sans elle, la signature haute frequence — jamais une position du repli a largeur
// libre, que la cuisson n essaie pas.
func TestLaCuissonNeDemarreQueSurLaSignatureStricte(t *testing.T) {
	sansSignature := 0
	parcourirPaquetsAEvenements(t, bobineMarcheDir(), func(p paquetDeLocalisation, cfg FrameConfig) {
		strict := marchLocateStrict(p.pay, p.w, cfg)
		if strict < 0 {
			sansSignature++
			strict, _ = LocaliserBoucleDeRecords(p.pay, p.w, cfg, SignatureHauteFrequence)
		}
		if debut, comment := localiserLaListe(p.pay, p.w, cfg); comment == lecture.DebutParSignature && debut != strict {
			t.Errorf("debut %d attribue a la signature, signature stricte %d", debut, strict)
		}
	})
	if sansSignature == 0 {
		t.Fatal("aucun paquet sans signature stricte sur la bobine : le test ne garde rien")
	}
}

// TestLaCuissonEssaieLaFermetureAvantLaHauteFrequence : dans [localiserLaListe], la signature du
// slot 123, puis la fermeture par NEW de tete, puis la signature haute frequence, chacune une fois.
// Aucune bobine du depot ne porte un paquet que la fermeture et la signature haute frequence
// localisent toutes deux : l ordre se lit donc dans le source.
func TestLaCuissonEssaieLaFermetureAvantLaHauteFrequence(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "debut_de_liste.go", nil, 0)
	if err != nil {
		t.Fatalf("analyse de debut_de_liste.go : %v", err)
	}
	var etapes []string
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "localiserLaListe" {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if c, ok := n.(*ast.CallExpr); ok {
				if e := etapeDeLaCuisson(c); e != "" {
					etapes = append(etapes, e)
				}
			}
			return true
		})
	}
	attendu := []string{"SignatureStricte", "debutParFermetureRangee", "SignatureHauteFrequence"}
	if len(etapes) != len(attendu) {
		t.Fatalf("etapes de la cuisson %v, attendu %v", etapes, attendu)
	}
	for i := range attendu {
		if etapes[i] != attendu[i] {
			t.Fatalf("etapes de la cuisson %v, attendu %v", etapes, attendu)
		}
	}
}

// etapeDeLaCuisson nomme l etape qu un appel de [localiserLaListe] essaie : l ordre passe au
// localisateur, ou la fermeture par NEW de tete ; "" pour un autre appel.
func etapeDeLaCuisson(c *ast.CallExpr) string {
	id, ok := c.Fun.(*ast.Ident)
	if !ok {
		return ""
	}
	switch id.Name {
	case "debutParFermetureRangee":
		return id.Name
	case "LocaliserBoucleDeRecords":
		if len(c.Args) == 4 {
			if ordre, ok := c.Args[3].(*ast.Ident); ok {
				return ordre.Name
			}
		}
	}
	return ""
}
