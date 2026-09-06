package synthetic_title_b

// film_capabilities_test.go — LA FIXTURE DE LA DEGRADATION FINE, ENFIN PROUVEE.
//
// Le lot v2(C.1) a fait declarer a ce titre les SIX cles `film.*` avec UN seul cas
// `supported` (l'artefact) et cinq `not_exposed` (ses derives), sous un commentaire
// affirmant « la divergence EST le test ». Elle ne l'etait pas : la revue adversariale C-R1
// (constat C3) a bascule `film.replay_artifact` a `not_exposed` (M3), puis supprime les six
// lignes (M4) — la suite restait VERTE dans les deux cas. Six lignes de fixture et douze de
// commentaire qui se declarent un test sans en etre un : « dead code museum ».
//
// Ce fichier ferme le trou en deux temps :
//   1. le CONTENU du TOML est asserte ligne par ligne (M3 et M4 rouges) ;
//   2. la SEMANTIQUE est verifiee sur la CapabilityMap construite depuis ce TOML : la cle
//      supportee ouvre, les cinq autres ferment — c'est ce qui prouve que ces cles sont
//      FINES et non un fourre-tout « ce titre a un film ».
//
// Les portes elles-memes (production dans `sync/replayartifacts`, loaders dans `api/wire`)
// sont exercees avec ce titre dans leurs paquets respectifs : une porte se teste ou elle
// vit, pas ici.

import (
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/mappings"
)

// filmAttendu — le contenu EXACT attendu des six lignes `film.*` du TOML de la fixture.
//
// C'est une liste ECRITE A LA MAIN, et c'est voulu : la deriver du fichier reviendrait a
// comparer le fichier a lui-meme. Un cas `supported` et cinq `not_exposed` — si un jour la
// fixture doit changer, ce tableau change dans le meme commit, ce qui rend la decision
// visible en revue.
var filmAttendu = map[games.CapabilityKey]games.CapabilityStatus{
	games.CapFilmReplayArtifact: games.CapSupported,
	games.CapFilmUsageSummary:   games.CapNotExposed,
	games.CapFilmBombStats:      games.CapNotExposed,
	games.CapFilmKillSource:     games.CapNotExposed,
	games.CapFilmWeaponShots:    games.CapNotExposed,
	games.CapFilmKillPositions:  games.CapNotExposed,
}

// capabilitiesDeLaFixture charge le capabilities.toml LIVRE du titre synthetique.
func capabilitiesDeLaFixture(t *testing.T) games.CapabilityMap {
	t.Helper()
	path := filepath.Join(repoRoot(t), "config", "titles", TitleSlug, "mappings", "capabilities.toml")
	set, err := mappings.LoadCapabilitiesFromFile(path)
	if err != nil {
		t.Fatalf("LoadCapabilitiesFromFile(%s): %v", path, err)
	}
	caps, err := games.CapabilityMapFromMappings(set)
	if err != nil {
		t.Fatalf("CapabilityMapFromMappings: %v", err)
	}
	return caps
}

// TestSkeleton_FilmCapabilitiesDeclareesTellesQuAttendues — le TOML porte les SIX cles
// `film.*`, avec exactement les statuts attendus.
//
// Ce test rougit si une ligne disparait (M4), si un statut bascule (M3), ou si une sixieme
// famille `film.*` est ajoutee au vocabulaire sans que la fixture la declare — auquel cas
// elle ne prouverait plus rien pour cette famille-la.
func TestSkeleton_FilmCapabilitiesDeclareesTellesQuAttendues(t *testing.T) {
	t.Parallel()
	caps := capabilitiesDeLaFixture(t)

	for cle, attendu := range filmAttendu {
		got, present := caps[cle]
		if !present {
			t.Errorf("la fixture ne declare PAS %q — sans elle, aucune porte de cette famille "+
				"n'est prouvee par ce titre. Remettre la ligne dans "+
				"config/titles/%s/mappings/capabilities.toml.", cle, TitleSlug)
			continue
		}
		if got != attendu {
			t.Errorf("capability %q = %q, attendu %q — la fixture existe pour porter UN cas "+
				"`supported` et cinq `not_exposed` : c'est cette divergence qui prouve que les "+
				"cles `film.*` sont FINES. Ne pas l'aligner sur halo_infinite.", cle, got, attendu)
		}
	}

	// Symetrie : aucune cle `film.*` du vocabulaire ne doit manquer au tableau ci-dessus.
	for _, cle := range games.AllCapabilityKeys() {
		if len(cle) < 5 || cle[:5] != "film." {
			continue
		}
		if _, prevu := filmAttendu[cle]; !prevu {
			t.Errorf("la famille %q existe dans games.AllCapabilityKeys() mais n'est pas "+
				"declaree par la fixture : ajouter la ligne au TOML ET l'entree a filmAttendu "+
				"(sinon la degradation de cette famille n'est prouvee nulle part).", cle)
		}
	}
}

// TestSkeleton_FilmDegradationFine — la SEMANTIQUE : `Has` ouvre pour la cle supportee et
// ferme pour les cinq autres.
//
// C'est l'invariant que le lot annonce : un titre peut produire l'ARTEFACT sans qu'aucun de
// ses derives soit cable. Si `Has` rendait vrai pour tout le monde (fourre-tout), ce test
// rougirait.
func TestSkeleton_FilmDegradationFine(t *testing.T) {
	t.Parallel()
	caps := capabilitiesDeLaFixture(t)

	if !caps.Has(games.CapFilmReplayArtifact) {
		t.Errorf("%q : Has = false, attendu true — le chemin nominal (production de "+
			"l'artefact, loaders de la Match View) doit s'ouvrir pour ce titre",
			games.CapFilmReplayArtifact)
	}
	for cle := range filmAttendu {
		if cle == games.CapFilmReplayArtifact {
			continue
		}
		if caps.Has(cle) {
			t.Errorf("%q : Has = true, attendu false — la fixture declare cette famille "+
				"`not_exposed` ; si elle ouvre quand meme, les cles `film.*` ne sont pas fines "+
				"et une seule d'entre elles ouvrirait toutes les portes", cle)
		}
	}
}
