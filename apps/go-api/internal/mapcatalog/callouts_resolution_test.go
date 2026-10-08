package mapcatalog

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// Identifiants des témoins : une carte Forge présente aux deux catalogues, une seulement au
// généré, une carte Forge connue PAR SON MAP_ID au catalogue des bornes (cas Narrows), et une
// AUTRE version de la carte Forge versionnée (même nom, autre map_id).
const (
	idVersionnee   = "11111111-0000-0000-0000-000000000001"
	idGeneree      = "22222222-0000-0000-0000-000000000002"
	idParBornes    = "33333333-0000-0000-0000-000000000003"
	idAutreVersion = "44444444-0000-0000-0000-000000000004"
	idNativeParID  = "55555555-0000-0000-0000-000000000005"
)

// indexStub : l'index des identités déclarées par les fonds publiés.
type indexStub map[string]string

func (i indexStub) Lookup(nom string) (string, bool) { c, ok := i[nom]; return c, ok }

func entreeNommee(module, en string) replay.MapCalloutsEntry {
	e := entreeZonesTest(en)
	e.Module = module
	return e
}

// sourcesTemoins monte les quatre sources de la cascade.
func sourcesTemoins() SourcesDeZones {
	return SourcesDeZones{
		Versionne: &replay.MapCalloutsCatalog{
			Maps: map[string]replay.MapCalloutsEntry{
				"ridgeline":     entreeNommee("ridgeline", "Horseshoe"),
				"btb_highpower": entreeNommee("btb_highpower", "Gold Base"),
			},
			MapsByID: map[string]replay.MapCalloutsEntry{idVersionnee: entreeNommee("", "Versionnee")},
		},
		Genere: &replay.MapCalloutsCatalog{MapsByID: map[string]replay.MapCalloutsEntry{
			idGeneree:    entreeNommee("", "Generee"),
			idVersionnee: entreeNommee("", "NE DOIT PAS PRIMER"),
			idParBornes:  entreeNommee("", "Narrows"),
		}},
		Bornes: &decfilm.MapQuantCatalog{Maps: map[string]decfilm.MapQuantEntry{
			"cliffhanger": {Module: "ridgeline"},
			"dynasty":     {Module: "fo08_wetland"}, // canevas : aucune zone
			idParBornes:   {Module: "fo13_frost"},
			idNativeParID: {Module: "ridgeline"},
		}},
		Identites: indexStub{
			"Highpower Sentry Defense": "btb_highpower",
			"Shiro":                    idVersionnee, // un fond Forge : clé = map_id
		},
	}
}

// TestResoudreSuitLaCascade — chaque essai, et l'origine qu'il déclare.
func TestResoudreSuitLaCascade(t *testing.T) {
	s := sourcesTemoins()
	cas := []struct {
		nom     string
		id      IdentitesDeCarte
		en      string
		origine OrigineZones
	}{
		{"carte intégrée par son nom", IdentitesDeCarte{MapID: "x", Noms: []string{"Cliffhanger - Ranked"}}, "Horseshoe", OrigineModule},
		{"map_id au catalogue des bornes, sans aucun nom", IdentitesDeCarte{MapID: idNativeParID}, "Horseshoe", OrigineModule},
		{"carte Forge versionnée", IdentitesDeCarte{MapID: idVersionnee, Noms: []string{"Dynasty"}}, "Versionnee", OrigineVersionne},
		{"carte Forge rattrapée", IdentitesDeCarte{MapID: idGeneree}, "Generee", OrigineGenere},
		{"map_id connu des bornes, canevas sans zone", IdentitesDeCarte{MapID: idParBornes}, "Narrows", OrigineGenere},
		{"nom de variante déclaré par un fond", IdentitesDeCarte{MapID: "y", Noms: []string{"Highpower Sentry Defense"}}, "Gold Base", OrigineIdentiteDeclaree},
	}
	for _, c := range cas {
		e, origine, ok := s.Resoudre(c.id)
		if !ok || origine != c.origine || e.Zones[0].EN != c.en {
			t.Errorf("%s : ok=%v origine=%q zone=%q ; attendu %q par %q", c.nom, ok, origine, firstEN(e), c.en, c.origine)
		}
	}
}

// TestResoudreNeRapprochePasDeuxMapIDParLeNom — une AUTRE version d'une carte Forge (même nom,
// autre map_id) n'hérite pas des zones de la version cataloguée : ni par le catalogue, ni par
// l'index des fonds, dont la clé Forge est un map_id absent de l'espace des modules.
func TestResoudreNeRapprochePasDeuxMapIDParLeNom(t *testing.T) {
	s := sourcesTemoins()
	if e, origine, ok := s.Resoudre(IdentitesDeCarte{MapID: idAutreVersion, Noms: []string{"Shiro"}}); ok {
		t.Fatalf("autre version de Shiro résolue par %q : %+v", origine, e)
	}
}

// TestResoudreSansSources — des sources toutes absentes rendent une absence, jamais un panic.
func TestResoudreSansSources(t *testing.T) {
	if _, _, ok := (SourcesDeZones{}).Resoudre(IdentitesDeCarte{MapID: idGeneree, Noms: []string{"Cliffhanger"}}); ok {
		t.Fatal("aucune source : résolution inattendue")
	}
}

func firstEN(e replay.MapCalloutsEntry) string {
	if len(e.Zones) == 0 {
		return ""
	}
	return e.Zones[0].EN
}
