package replay

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/testutil"
)

func writeCalloutsCatalog(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "map_callouts.json")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("ecriture fixture: %v", err)
	}
	return p
}

// TestCatalogueCalloutsRefuseUneAutreVersionDeSchema garde le verrou de version : lire
// une forme future comme l'actuelle servirait des zones silencieusement fausses.
func TestCatalogueCalloutsRefuseUneAutreVersionDeSchema(t *testing.T) {
	p := writeCalloutsCatalog(t, `{"schema_version":2,"title_slug":"halo_infinite","maps":{}}`)
	if _, err := LoadMapCallouts(p); err == nil {
		t.Fatal("catalogue en v2 : attendu une erreur de version, obtenu nil")
	}
	p1 := writeCalloutsCatalog(t, `{"schema_version":1,"title_slug":"halo_infinite","maps":{}}`)
	if _, err := LoadMapCallouts(p1); err != nil {
		t.Fatalf("catalogue en v1 : %v", err)
	}
}

// TestCalloutsLookupCarteInconnueEstUnCasNominal : une carte hors catalogue — un canevas
// Forge, qui ne pose aucune zone, ou une carte jamais extraite — est une ABSENCE. L'appelant
// doit la distinguer d'une panne de lecture pour dégrader proprement.
func TestCalloutsLookupCarteInconnueEstUnCasNominal(t *testing.T) {
	p := writeCalloutsCatalog(t, `{"schema_version":1,"maps":{}}`)
	c, err := LoadMapCallouts(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Lookup("fo08_wetland"); !errors.Is(err, ErrCalloutsUnknownMap) {
		t.Errorf("err = %v, attendu ErrCalloutsUnknownMap", err)
	}
	var nilCat *MapCalloutsCatalog
	if _, err := nilCat.Lookup("x"); !errors.Is(err, ErrCalloutsUnknownMap) {
		t.Errorf("catalogue nil : err = %v, attendu ErrCalloutsUnknownMap", err)
	}
}

// TestCalloutsLookupByID — l'espace de clés des cartes FORGE, et ses trois absences
// propres : section absente, map_id inconnu, map_id vide (le registre ne nomme pas
// toujours la carte d'un vieux match).
func TestCalloutsLookupByID(t *testing.T) {
	p := writeCalloutsCatalog(t, `{"schema_version":1,"maps":{},"maps_by_id":{`+
		`"d5c5eb4f-0dcb-4677-a866-eae0dcbfde9b":{"module":"","provenance":"mvar","zones":[`+
		`{"volume_index":312,"name":"","en":"Cave","fr":"Grotte","x":1,"y":2,"z":3,`+
		`"z_bottom":0,"z_top":6,"polygon":[[0,0],[1,0],[1,1]]}]}}}`)
	c, err := LoadMapCallouts(p)
	if err != nil {
		t.Fatal(err)
	}
	e, err := c.LookupByID("d5c5eb4f-0dcb-4677-a866-eae0dcbfde9b")
	if err != nil {
		t.Fatalf("carte Forge au catalogue : %v", err)
	}
	if e.Provenance != CalloutsProvenanceMvar || len(e.Zones) != 1 || e.Zones[0].FR != "Grotte" {
		t.Errorf("entrée inattendue : %+v", e)
	}
	if e.Module != "" {
		t.Errorf("module = %q : une carte Forge n'en a pas", e.Module)
	}
	for _, id := range []string{"", "00000000-0000-0000-0000-000000000000"} {
		if _, err := c.LookupByID(id); !errors.Is(err, ErrCalloutsUnknownMap) {
			t.Errorf("map_id %q : err = %v, attendu ErrCalloutsUnknownMap", id, err)
		}
	}
	// Catalogue SANS section Forge : l'absence est propre, pas une panne.
	vide := writeCalloutsCatalog(t, `{"schema_version":1,"maps":{}}`)
	sans, err := LoadMapCallouts(vide)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sans.LookupByID("d5c5eb4f-0dcb-4677-a866-eae0dcbfde9b"); !errors.Is(err, ErrCalloutsUnknownMap) {
		t.Errorf("section absente : err = %v, attendu ErrCalloutsUnknownMap", err)
	}
}

// TestCatalogueCalloutsLivreEstExploitable lit le catalogue REELLEMENT versionné — c'est
// lui que le service servira (patron TestCatalogueLivreEstExploitable).
//
// Invariants gardés (mesure du corpus, 2026-08) : 22 cartes, 816 zones, libellés FR et EN
// résolus 816/816, toute zone dessinée porte un polygone d'au moins 3 sommets en
// coordonnées monde plausibles, la tranche verticale est ordonnée. Ridgeline garde ses
// 28 zones dont 11 grandes — le classement du POC, rejoué aussi par
// cmd/mapcallouts-build/classify_test.go.
//
// LA RÈGLE DU DÉCOUPAGE EST FIGÉE ICI, et elle se vérifie sur l'arbre, pas sur une liste
// écrite à la main : une carte est `decoupe` SI ET SEULEMENT SI son fond est publié. C'est
// la règle « aucun découpage deviné » — une liste de cartes se serait désynchronisée à la
// première carte ajoutée.
func TestCatalogueCalloutsLivreEstExploitable(t *testing.T) {
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot introuvable : %v", err)
	}
	path := filepath.Join(root, "data", "titles", "halo_infinite", "reference", "map_callouts.json")
	cat, err := LoadMapCallouts(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cat.Maps) != 22 {
		t.Errorf("cartes = %d, attendu 22", len(cat.Maps))
	}
	fonds := filepath.Join(filepath.Dir(path), "map_backgrounds")
	total := 0
	for module, e := range cat.Maps {
		if e.Module != module {
			t.Errorf("%s : champ module = %q", module, e.Module)
		}
		attendu := CalloutsProvenanceBrut
		if _, err := os.Stat(filepath.Join(fonds, module+".png")); err == nil {
			attendu = CalloutsProvenanceDecoupe
		}
		if e.Provenance != attendu {
			t.Errorf("%s : provenance = %q, attendu %q (fond publié : %t)",
				module, e.Provenance, attendu, attendu == CalloutsProvenanceDecoupe)
		}
		total += len(e.Zones)
		for _, z := range e.Zones {
			if z.FR == "" || z.EN == "" {
				t.Errorf("%s vi=%d : libellé vide (fr=%q en=%q) — le contrat est 816/816",
					module, z.VolumeIndex, z.FR, z.EN)
			}
			if z.ZTop <= z.ZBottom {
				t.Errorf("%s vi=%d : tranche verticale inversée [%f;%f]", module, z.VolumeIndex, z.ZBottom, z.ZTop)
			}
			if len(z.Polygon) > 0 && len(z.Polygon) < 3 {
				t.Errorf("%s vi=%d : polygone à %d sommet(s)", module, z.VolumeIndex, len(z.Polygon))
			}
			for _, v := range z.Polygon {
				if v[0] < -1000 || v[0] > 1000 || v[1] < -1000 || v[1] > 1000 {
					t.Errorf("%s vi=%d : sommet hors monde plausible (%f, %f)", module, z.VolumeIndex, v[0], v[1])
				}
			}
			if z.Big && len(z.Polygon) < 3 {
				t.Errorf("%s vi=%d : zone grande sans polygone", module, z.VolumeIndex)
			}
		}
	}
	if total != 816 {
		t.Errorf("zones = %d, attendu 816", total)
	}
	ridge, err := cat.Lookup("ridgeline")
	if err != nil {
		t.Fatalf("ridgeline absente : %v", err)
	}
	grandes := 0
	for _, z := range ridge.Zones {
		if z.Big {
			grandes++
		}
	}
	if len(ridge.Zones) != 28 || grandes != 11 {
		t.Errorf("ridgeline : %d zones / %d grandes, attendu 28 / 11 (classement du POC)",
			len(ridge.Zones), grandes)
	}
	verifieBrutConserve(t, cat)
	t.Logf("catalogue livré : %d cartes, %d zones, ridgeline %d grandes", len(cat.Maps), total, grandes)
}

// TestCatalogueCalloutsForgeLivreEstExploitable — L'ORACLE SUR LA SECTION FORGE du
// catalogue versionné (produite le 2026-09-02 par `mapcallouts-build --forge-only`).
//
// Invariants gardés, tous MESURÉS sur la production et non déclarés à l'avance :
//   - la clé est un map_id (UUID), jamais un module, et l'entrée ne porte PAS de module ;
//   - la provenance est `mvar` — aucune zone Forge n'est découpée ;
//   - toute zone porte un polygone d'au moins 3 sommets : une zone Forge est un volume
//     posé, elle a toujours une forme (contrairement aux volumes secondaires du tag levl) ;
//   - la tranche verticale est ordonnée ;
//   - CHAQUE CARTE PUBLIÉE A AU MOINS UNE ZONE NOMMÉE : c'est la règle de publication de la
//     passe Forge (un calque entièrement muet sous une bascule « Zones nommées » serait du
//     bruit) — et à l'inverse une zone SANS libellé est légitime, sa géométrie est mesurée.
func TestCatalogueCalloutsForgeLivreEstExploitable(t *testing.T) {
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot introuvable : %v", err)
	}
	cat, err := LoadMapCallouts(filepath.Join(root, "data", "titles", "halo_infinite",
		"reference", "map_callouts.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cat.MapsByID) == 0 {
		t.Fatal("aucune carte Forge au catalogue : la passe Forge n'a pas été jouée")
	}
	zones, nommees := 0, 0
	for id, e := range cat.MapsByID {
		if len(id) != 36 {
			t.Errorf("clé %q : ce n'est pas un map_id (UUID de 36 caractères)", id)
		}
		if _, collision := cat.Maps[id]; collision {
			t.Errorf("%s : la même clé existe côté modules — les deux espaces se croisent", id)
		}
		if e.Module != "" {
			t.Errorf("%s : module = %q, une carte Forge n'en a pas", id, e.Module)
		}
		if e.Provenance != CalloutsProvenanceMvar {
			t.Errorf("%s : provenance = %q, attendu %q", id, e.Provenance, CalloutsProvenanceMvar)
		}
		nommeesCarte := 0
		for _, z := range e.Zones {
			zones++
			if z.FR != "" && z.EN != "" {
				nommeesCarte++
			}
			if (z.FR == "") != (z.EN == "") {
				t.Errorf("%s vi=%d : libellé résolu dans une seule langue (fr=%q en=%q)",
					id, z.VolumeIndex, z.FR, z.EN)
			}
			if len(z.Polygon) < 3 {
				t.Errorf("%s vi=%d : polygone à %d sommet(s) — une zone Forge est un volume posé",
					id, z.VolumeIndex, len(z.Polygon))
			}
			if z.ZTop <= z.ZBottom {
				t.Errorf("%s vi=%d : tranche verticale inversée [%f;%f]", id, z.VolumeIndex, z.ZBottom, z.ZTop)
			}
			for _, v := range z.Polygon {
				// Un canevas Forge tient dans ±212,5 / ±250 m (mesure des 8 installés) ;
				// la marge à 1 000 m attrape une coordonnée aberrante sans juger du canevas.
				if v[0] < -1000 || v[0] > 1000 || v[1] < -1000 || v[1] > 1000 {
					t.Errorf("%s vi=%d : sommet hors monde plausible (%f, %f)", id, z.VolumeIndex, v[0], v[1])
				}
			}
		}
		if nommeesCarte == 0 {
			t.Errorf("%s : aucune zone nommée — la carte n'aurait pas dû être publiée", id)
		}
		nommees += nommeesCarte
	}
	t.Logf("section Forge : %d cartes, %d zones, %d nommées (%.0f %%)",
		len(cat.MapsByID), zones, nommees, 100*float64(nommees)/float64(zones))
}

// verifieBrutConserve : le découpage ne PERD rien. Toute zone rognée doit retrouver son pavé
// du designer dans `cat.Brut`, sous une carte effectivement découpée.
func verifieBrutConserve(t *testing.T, cat *MapCalloutsCatalog) {
	t.Helper()
	garde := 0
	for module, zones := range cat.Brut {
		e, err := cat.Lookup(module)
		if err != nil {
			t.Errorf("brut conservé pour %q, carte absente du catalogue", module)
			continue
		}
		if e.Provenance != CalloutsProvenanceDecoupe {
			t.Errorf("%s : brut conservé alors que la provenance est %q", module, e.Provenance)
		}
		connues := map[int]bool{}
		for _, z := range e.Zones {
			connues[z.VolumeIndex] = true
		}
		for _, b := range zones {
			garde++
			if !connues[b.VolumeIndex] {
				t.Errorf("%s : brut du volume %d sans zone correspondante", module, b.VolumeIndex)
			}
			if len(b.Polygon) < 3 {
				t.Errorf("%s vi=%d : pavé brut conservé à %d sommet(s)", module, b.VolumeIndex, len(b.Polygon))
			}
		}
	}
	if garde == 0 {
		t.Error("aucun pavé brut conservé : le découpage perdrait la donnée d'origine")
	}
	t.Logf("pavés bruts conservés : %d zones sur %d cartes", garde, len(cat.Brut))
}
