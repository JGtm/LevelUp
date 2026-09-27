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
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/types"
	"levelup/go-api/internal/port"
)

// TestLaResolutionDesFragsCompteSesReplis : un gamertag vu avec deux xuids, un assistant sans
// identite — chacun compte une fois, sans rien changer a ce qui est publie.
func TestLaResolutionDesFragsCompteSesReplis(t *testing.T) {
	idx, divergences := gamertagXUIDIndex([]types.Death{
		{Gamertag: "Tueur", XUID: 1}, {Gamertag: "Tueur", XUID: 2}, {Gamertag: "Tueur", XUID: 1},
		{Gamertag: "Victime", XUID: 3},
	})
	if idx["Tueur"] != 1 || divergences != 1 {
		t.Fatalf("premier xuid %d, divergences %d — attendu 1 et 1 (le premier gagne, l ecart se compte)",
			idx["Tueur"], divergences)
	}
	k := killDe("Tueur", "Victime", 10)
	k.Assist.Known, k.Assist.Name = true, "Inconnu"
	r := resolveKills([]decfilm.Kill{k}, idx)
	if r.assistantsNonResolus != 1 || len(r.refs) != 1 || r.refs[0].AssistKnown {
		t.Fatalf("assistant non resolu : compte %d, refs %+v", r.assistantsNonResolus, r.refs)
	}
}

// TestLaFeuilleDeMatchCompteSesRetraits : une ligne sans xuid quitte le tableau, un camp inconnu
// quitte la table — et chacun se compte.
func TestLaFeuilleDeMatchCompteSesRetraits(t *testing.T) {
	facts := port.MatchFacts{Players: []port.MatchPlayerFact{
		{XUID: "", TeamID: 0}, {XUID: "2", TeamID: -1}, {XUID: "3", TeamID: 1},
	}}
	fb := decfilm.NouveauCompteur()
	if got := participantsDuTableau(facts, fb); len(got) != 2 {
		t.Fatalf("participants : %d, attendu 2", len(got))
	}
	if got := fb.Compte(decfilm.NomParticipantSansXuidRetire); got != 1 {
		t.Errorf("participant sans xuid : %d, attendu 1", got)
	}
	if camps, retires := tableDesCamps(facts); retires != 1 || len(camps) != 2 {
		t.Errorf("camps %v, retires %d — attendu 2 camps et 1 retrait", camps, retires)
	}
}

// TestUnTitreSansTableDObjectifsCompteSonRepliAChaqueCuisson : sans table de roles, le rejeu sort
// sans etat de zone, et le repli se compte a chaque cuisson (la table memorisee ne l efface pas).
func TestUnTitreSansTableDObjectifsCompteSonRepliAChaqueCuisson(t *testing.T) {
	b := &Builder{repoRoot: t.TempDir(), titleSlug: title.DefaultSlug}
	for i := 0; i < 2; i++ {
		fb := decfilm.NouveauCompteur()
		if zones, _ := b.matchZones("m", "carte", "Arena:Strongholds", fb); zones != nil {
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
	opt := b.buildReplayOptions(decfilm.MapQuantEntry{}, facts, cat, &filmStats{})
	got := map[decfilm.Nom]int{}
	for _, d := range opt.ReplisHorsBalayage.Construction {
		got[d.Nom] = d.Declenchements
	}
	for _, nom := range []decfilm.Nom{decfilm.NomRelaisDeBotAbandonne, decfilm.NomParticipantSansXuidRetire,
		decfilm.NomCampInconnuRetireDeLaTable} {
		if got[nom] != 1 {
			t.Errorf("%s : %d dans le rapport de construction, attendu 1 (rapport %v)", nom, got[nom],
				opt.ReplisHorsBalayage.Construction)
		}
	}
	if opt.Fallbacks != nil {
		t.Error("la construction pre-remplit le compteur des options : les faits persistes le captureraient")
	}
}
