package replaybuild

// replis_de_la_construction_test.go — LES REPLIS QUE LA CONSTRUCTION DECLENCHE ELLE-MEME (lot J8.7,
// 2026-09-27, decision 2 du superviseur).
//
// Maillons tenus : chaque site compte sur le compteur de la construction, et son RAPPORT part dans
// `Options.ReplisHorsBalayage.Construction` — jamais dans `Options.Fallbacks`, que les faits
// persistes capturent. Le versement a l assemblage (une fois, sur les deux chemins) est tenu par
// `TestLesReplisHorsBalayageNeSeComptentQuUneFoisDepuisLesFaits`, dans le paquet de publication.
//
// MUTATION JOUEE (2026-09-27) : retirer `Construction: cat.replis.Rapport()` de
// [Builder.buildReplayOptions] fait rougir `TestLeRapportDeLaConstructionVoyageDansLesOptions`.

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/types"
	"levelup/go-api/internal/port"
)

// TestLaResolutionDesFragsCompteSesReplis : un gamertag vu avec deux xuids ne resout rien (il est
// compte comme ambigu, jamais tranche) ; un assistant sans identite compte son repli, sans rien
// changer a ce qui est publie.
func TestLaResolutionDesFragsCompteSesReplis(t *testing.T) {
	ambigu, nAmbigus := gamertagXUIDIndex([]types.Death{
		{Gamertag: "Double", XUID: 1}, {Gamertag: "Double", XUID: 2}, {Gamertag: "Double", XUID: 1},
		{Gamertag: "Victime", XUID: 3},
	})
	if _, ok := ambigu["Double"]; ok || nAmbigus != 1 || ambigu["Victime"] != 3 {
		t.Fatalf("table %v, ambigus %d — attendu Double absent, 1 ambigu, Victime -> 3", ambigu, nAmbigus)
	}
	idx, _ := gamertagXUIDIndex([]types.Death{{Gamertag: "Tueur", XUID: 1}, {Gamertag: "Victime", XUID: 3}})
	k := killDe("Tueur", "Victime", 10)
	k.Assist.Known, k.Assist.Name = true, "Inconnu"
	r := resolveKills([]decfilm.Kill{k}, idx)
	if r.assistantsNonResolus != 1 || len(r.refs) != 1 || r.refs[0].AssistKnown {
		t.Fatalf("assistant non resolu : compte %d, refs %+v", r.assistantsNonResolus, r.refs)
	}
}

// TestLaFeuilleDeMatchEcarteLesLignesSansIdentite : une ligne sans xuid quitte le tableau, un camp
// inconnu quitte la table — rien ne remplace ni l un ni l autre.
func TestLaFeuilleDeMatchEcarteLesLignesSansIdentite(t *testing.T) {
	facts := port.MatchFacts{Players: []port.MatchPlayerFact{
		{XUID: "", TeamID: 0}, {XUID: "2", TeamID: -1}, {XUID: "3", TeamID: 1},
	}}
	if got := participantsDuTableau(facts); len(got) != 2 {
		t.Fatalf("participants : %d, attendu 2", len(got))
	}
	if camps := teamByXUID(facts); len(camps) != 2 {
		t.Errorf("camps %v — attendu 2 camps", camps)
	}
}

// TestUnTitreSansTableDObjectifsCompteSonRepliAChaqueCuisson : sans table de roles, le rejeu sort
// sans etat de zone, et le repli se compte a chaque cuisson (la table memorisee ne l efface pas).
func TestUnTitreSansTableDObjectifsCompteSonRepliAChaqueCuisson(t *testing.T) {
	b := &Builder{repoRoot: t.TempDir(), titleSlug: title.DefaultSlug}
	for i := range 2 {
		fb := decfilm.NouveauCompteur()
		if zones, _ := b.matchZones(context.Background(), "m", "carte", "Arena:Strongholds", fb); zones != nil {
			t.Fatalf("titre sans table : zones %v", zones)
		}
		if got := fb.Compte(decfilm.NomCatalogueDeZonesAbsent); got != 1 {
			t.Errorf("cuisson %d : repli compte %d fois, attendu 1", i, got)
		}
	}
}

// TestLeRapportDeLaConstructionVoyageDansLesOptions : ce que la construction a compte arrive dans
// `ReplisHorsBalayage.Construction`, et le compteur des options reste vide — c est ce qui empeche
// le double compte d une republication depuis les faits.
func TestLeRapportDeLaConstructionVoyageDansLesOptions(t *testing.T) {
	b, _ := zonesTestBuilder(t)
	cat := entreesCatalogue{replis: decfilm.NouveauCompteur()}
	cat.replis.Declenche(decfilm.NomRelaisDeBotAbandonne)
	facts := port.MatchFacts{Players: []port.MatchPlayerFact{{XUID: "", TeamID: -1}}}
	opt := b.buildReplayOptions(context.Background(), decfilm.MapQuantEntry{}, facts, cat, &filmStats{})
	got := map[decfilm.Nom]int{}
	for _, d := range opt.ReplisHorsBalayage.Construction {
		got[d.Nom] = d.Declenchements
	}
	if got[decfilm.NomRelaisDeBotAbandonne] != 1 || len(got) != 1 {
		t.Errorf("rapport de construction %v, attendu le seul repli declenche (relais de bot, 1)",
			opt.ReplisHorsBalayage.Construction)
	}
	if opt.Fallbacks != nil {
		t.Error("la construction pre-remplit le compteur des options : les faits persistes le captureraient")
	}
}
