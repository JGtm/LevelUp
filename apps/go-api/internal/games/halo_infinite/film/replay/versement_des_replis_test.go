package replay

// versement_des_replis_test.go — LA TABLE DE VERSEMENT, CHAMP PAR CHAMP (lot J8.7, 2026-09-27).
//
// LE MAILLON TENU : un compte que la couche basse rend en donnees ARRIVE au compteur de la cuisson,
// sous le nom de SON entree. Chaque champ source est pose seul, la table est jouee, et le compteur
// doit porter ce compte-la, une fois, sous un nom du registre dont le compteur est branche.
//
// MUTATION JOUEE (2026-09-27) : retirer la ligne `fallback.NomChunksApresTrouAbandonnes` de
// [versementsDesReplis] fait rougir `TestChaqueChampDuRapportDeGrammaireEstVerse` (« le champ
// ChunksApresTrouAbandonnes n arrive pas au compteur »).

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/killsource"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// TestChaqueChampDuRapportDeGrammaireEstVerse : chaque champ de [grammar.ComptesDesReplis] est verse
// par la table, sous UN nom, et ce nom est une entree du registre. Que l entree soit BRANCHEE est
// tenu ailleurs (`fallback.TestChaqueRepliEstCompte`, direction (E) d `archlint`).
func TestChaqueChampDuRapportDeGrammaireEstVerse(t *testing.T) {
	var zero grammar.ComptesDesReplis
	typ := reflect.TypeOf(zero)
	nomsVus := map[fallback.Nom]string{}
	for i := 0; i < typ.NumField(); i++ {
		var r grammar.ComptesDesReplis
		reflect.ValueOf(&r).Elem().Field(i).SetInt(7)
		fb := fallback.NouveauCompteur()
		versementDuBalayage(fb, grammarContexteAvec(r))
		rap := fb.Rapport()
		champ := typ.Field(i).Name
		if len(rap) != 1 || rap[0].Declenchements != 7 {
			t.Errorf("le champ %s n arrive pas au compteur sous UN nom : rapport %v", champ, rap)
			continue
		}
		if autre, deja := nomsVus[rap[0].Nom]; deja {
			t.Errorf("%s et %s sont verses sous le MEME nom %s", autre, champ, rap[0].Nom)
		}
		nomsVus[rap[0].Nom] = champ
		if _, ok := fallback.Lire(rap[0].Nom); !ok {
			t.Errorf("le champ %s est verse sous %s, qui n est pas une entree du registre", champ, rap[0].Nom)
		}
	}
}

// TestUnRapportVideNeVerseRien : une source a zero ne cree aucune ligne — le rapport ne porte que ce
// qui s est declenche.
func TestUnRapportVideNeVerseRien(t *testing.T) {
	fb := fallback.NouveauCompteur()
	verserLesReplis(fb, sourcesDeReplis{})
	if rap := fb.Rapport(); len(rap) != 0 {
		t.Fatalf("sources vides : rapport %v, attendu vide", rap)
	}
}

// TestLeRapportDuBalayageEstVerseAvantLaCaptureDesFaits : le maillon que la seule bobine du depot ne
// peut pas exercer (elle n a pas d image-cle de bipede, le balayage des positions la refuse) se tient
// sur la SOURCE — [BuildFromFilmAvecFaits] verse le rapport du contexte AVANT `faitsDuBalayage`.
// Apres, il manquerait aux faits persistes et une republication le perdrait ; absent, la cuisson ne
// le publierait jamais.
//
// MUTATION JOUEE (2026-09-27) : deplacer l appel apres `faitsDuBalayage` fait rougir ce test.
func TestLeRapportDuBalayageEstVerseAvantLaCaptureDesFaits(t *testing.T) {
	appels := appelsDansLOrdre(t, "build_from_film.go", "BuildFromFilmAvecFaits")
	iVerse, iFaits := -1, -1
	for i, a := range appels {
		switch a {
		case "versementDuBalayage":
			iVerse = i
		case "faitsDuBalayage":
			iFaits = i
		}
	}
	if iVerse < 0 || iFaits < 0 || iVerse > iFaits {
		t.Fatalf("BuildFromFilmAvecFaits : versementDuBalayage en %d, faitsDuBalayage en %d (appels %v) — le "+
			"rapport des replis du balayage doit rejoindre le compteur AVANT la capture des faits", iVerse, iFaits, appels)
	}
}

// appelsDansLOrdre rend les noms des fonctions appelees par `fonction` dans `fichier`, dans l ordre
// du source.
func appelsDansLOrdre(t *testing.T, fichier, fonction string) []string {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), fichier, nil, 0)
	if err != nil {
		t.Fatalf("parse de %s : %v", fichier, err)
	}
	var out []string
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Name.Name != fonction {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if c, ok := n.(*ast.CallExpr); ok {
				if id, ok := c.Fun.(*ast.Ident); ok {
					out = append(out, id.Name)
				}
			}
			return true
		})
	}
	return out
}

// TestChaqueCompteDuKillFeedEstVerse : chaque champ de [killsource.ReplisDuDecodage], et chacun des
// comptes que le decodeur tenait deja, arrive au compteur sous UN nom, a l assemblage.
func TestChaqueCompteDuKillFeedEstVerse(t *testing.T) {
	typ := reflect.TypeOf(killsource.ReplisDuDecodage{})
	poseurs := map[string]func(*killsource.Result){}
	for i := 0; i < typ.NumField(); i++ {
		i := i
		poseurs["Replis."+typ.Field(i).Name] = func(r *killsource.Result) {
			reflect.ValueOf(&r.Stats.Replis).Elem().Field(i).SetInt(7)
		}
	}
	poseurs["Couples.Recolles"] = func(r *killsource.Result) { r.Stats.Couples.Recolles = 7 }
	poseurs["Appariement.Fenetre"] = func(r *killsource.Result) { r.Stats.Appariement.Fenetre = 7 }
	poseurs["Assist.ParLaFenetre"] = func(r *killsource.Result) { r.Stats.Assist.ParLaFenetre = 7 }
	poseurs["Appariement.BotFenetre"] = func(r *killsource.Result) { r.Stats.Appariement.BotFenetre = 7 }
	poseurs["Appariement.NonRevendiqueeFenetre"] = func(r *killsource.Result) { r.Stats.Appariement.NonRevendiqueeFenetre = 7 }
	poseurs["Roster.FilmTable.Inferred"] = func(r *killsource.Result) { r.Roster.FilmTable.Inferred = 7 }
	for champ, poser := range poseurs {
		var ks killsource.Result
		poser(&ks)
		fb := fallback.NouveauCompteur()
		versementDeLAssemblage(fb, ReplisHorsBalayage{KillSource: &ks})
		rap := fb.Rapport()
		if len(rap) != 1 || rap[0].Declenchements != 7 {
			t.Errorf("le compte %s n arrive pas au compteur sous UN nom : rapport %v", champ, rap)
			continue
		}
		if _, ok := fallback.Lire(rap[0].Nom); !ok {
			t.Errorf("le compte %s est verse sous %s, hors du registre", champ, rap[0].Nom)
		}
	}
}

// TestChaqueCompteDesObjectifsEstVerse : chaque champ de [objectives.ComptesDesReplis] arrive au
// compteur sous UN nom, a l assemblage.
func TestChaqueCompteDesObjectifsEstVerse(t *testing.T) {
	typ := reflect.TypeOf(objectives.ComptesDesReplis{})
	for i := 0; i < typ.NumField(); i++ {
		var c objectives.ComptesDesReplis
		reflect.ValueOf(&c).Elem().Field(i).SetInt(7)
		fb := fallback.NouveauCompteur()
		versementDeLAssemblage(fb, ReplisHorsBalayage{Objectifs: c})
		rap := fb.Rapport()
		if len(rap) != 1 || rap[0].Declenchements != 7 {
			t.Errorf("le compte %s n arrive pas au compteur sous UN nom : rapport %v", typ.Field(i).Name, rap)
		}
	}
}

// TestLesReplisHorsBalayageNeSeComptentQuUneFoisDepuisLesFaits — DECISION 2 DU SUPERVISEUR.
//
// Les comptes que l appelant apporte ([Options.ReplisHorsBalayage]) se versent a l ASSEMBLAGE, jamais
// dans le rapport du balayage que les faits persistent : republier depuis les faits les reverse donc
// UNE fois, comme la cuisson du film. Les deux chemins publient le meme `coverage.fallbacks`.
//
// MUTATION JOUEE (2026-09-27) : verser aussi `opt.ReplisHorsBalayage` dans [BuildFromFacts] avant
// l assemblage (ce que ferait un pre-remplissage du compteur par l appelant) fait rougir ce test
// (« repli_chunk_du_pied_par_argmax 2, attendu 1 »).
func TestLesReplisHorsBalayageNeSeComptentQuUneFoisDepuisLesFaits(t *testing.T) {
	entry := goldenEntryPourTest(t)
	g := loadGoldenInputs(t)
	id := &profile.FilmIdentity{Build: "HI_1_13_0", FormatVersion: 27}
	repliDuBalayage := []fallback.Declenchement{{Nom: fallback.NomPlafondGrenadeParDefaut, Declenchements: 2}}
	ks := &killsource.Result{}
	ks.Stats.Replis.PiedParArgmax = 1
	ks.Stats.Couples.Recolles = 3
	horsBalayage := ReplisHorsBalayage{KillSource: ks}

	optDirect := g.options()
	optDirect.MapQuant, optDirect.FilmIdentity, optDirect.ReplisHorsBalayage = &entry, id, horsBalayage
	optDirect.Fallbacks = fallback.NouveauCompteur()
	optDirect.Fallbacks.Cumuler(repliDuBalayage)
	direct := BuildFromPositions(context.Background(), goldenFilm, "halo_infinite", g.Positions, g.Fire, optDirect)

	blob, err := EncodeFilmFactsFile(&FilmFactsFile{
		Coverage: *couvertureDuDecodeur(id), Facts: *g, Identity: identiteDeFaits(id),
		Fallbacks: repliDuBalayage, EmpreinteDeCle: EmpreinteDeCle(entry),
	})
	if err != nil {
		t.Fatalf("encodage : %v", err)
	}
	f, err := DecodeFilmFactsFile(blob, entry)
	if err != nil {
		t.Fatalf("relecture : %v", err)
	}
	rejoue := BuildFromFacts(context.Background(), goldenFilm, "halo_infinite", f, Options{MapQuant: &entry, ReplisHorsBalayage: horsBalayage})

	attendu := map[string]int{
		string(fallback.NomPlafondGrenadeParDefaut): 2, string(fallback.NomChunkDuPiedParArgmax): 1,
		string(fallback.NomCoupleRecolleSurLeVoisin): 3,
	}
	for nom, doc := range map[string]ReplayDocument{"film": direct, "faits": rejoue} {
		got := map[string]int{}
		for _, h := range doc.Coverage.Fallbacks {
			got[h.Name] = h.Hits
		}
		for n, want := range attendu {
			if got[n] != want {
				t.Errorf("chemin %s : %s %d, attendu %d", nom, n, got[n], want)
			}
		}
	}
	if renderAssembly(direct) != renderAssembly(rejoue) {
		t.Error("les deux chemins de la cuisson ne publient pas le meme document")
	}
}

// grammarContexteAvec rend un contexte de film (sans film) dont le rapport vaut `r`.
func grammarContexteAvec(r grammar.ComptesDesReplis) *grammar.FilmContext {
	fc := grammar.NewFilmContext(nil)
	fc.NoterReplis(r)
	return fc
}
