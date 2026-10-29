package replay

// golden_equipement_test.go — LES POUSSEES ET L EQUIPEMENT PORTE DES HUIT BUILDS, FIGES A COTE DE
// LEURS ENTREES (revue adverse du lot « equipement porte », P2-1, 2026-10-15).
//
// # LE TROU QU'IL FERME
//
// Les poussees (schema des faits 12) et les lectures de l equipement porte (schema des faits 13)
// voyagent dans le COMPLEMENT de la section 1 du fichier de faits, pas dans le blob des entrees :
// les `inputs_<short8>.bin.gz` sont relus sans elles. Les goldens d assemblage et les fixtures de
// contrat publiaient donc `knockbacks`, `carriedEquipment` et `riftPoses` VIDES, et la frontiere web
// n etait traversee par aucune sortie Go de ces calques. Meme forme que les entites `ti=9`
// (golden_entites_test.go) : un fichier par build, `equipement_<short8>.bin.gz`, encode par le codec
// de production ([encodePoussees] puis [encodeEquipementPorte]), pose sur les entrees relues
// ([chargerGoldenBuild], [loadGoldenInputs]).
//
// # REGENERATION (jamais d'edition a la main)
//
// Depuis le film, par le balayage de la production ([decodeFilmInputsForEntry]), UN FILM PAR
// PROCESSUS :
//
//	REPLAY_FILM_CACHE=<repo>/data/cache/film_chunks \
//	  go test ./internal/games/halo_infinite/film/replay/ -run 'TestGoldenEquipementRegenerate/<short8>' -update
//
// Elle echoue TOUJOURS en nommant ce qu'elle a ecrit ; on relance sans les drapeaux pour verifier.

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/equipenergy"
)

// equipementPath : le fichier des poussees et de l equipement porte d un build.
func equipementPath(short8 string) string {
	return filepath.Join(goldenDir, "equipement_"+short8+".bin.gz")
}

// chargerEquipementDuBuild pose sur `g` les poussees et l equipement porte figes d un build. ABSENT =
// ECHEC : les huit builds le portent, et un build qui le perdrait publierait ces calques vides.
func chargerEquipementDuBuild(t *testing.T, short8 string, g *FilmFacts) {
	t.Helper()
	raw, err := os.ReadFile(equipementPath(short8)) //nolint:gosec // chemin construit depuis la table des builds
	if err != nil {
		t.Fatalf("equipement du build %s absent (%v) — regenerer avec %s=<cache> -run "+
			"TestGoldenEquipementRegenerate -update", short8, err, miniFilmCacheEnv)
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("equipement %s : gzip illisible : %v", short8, err)
	}
	defer func() { _ = zr.Close() }()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(zr); err != nil {
		t.Fatalf("equipement %s : decompression : %v", short8, err)
	}
	r := &greader{b: buf.Bytes()}
	g.Knockbacks, g.KnockbackStats = decodePoussees(r)
	g.CarriedEquipment, g.CarriedEquipmentStats = decodeEquipementPorte(r)
	if r.err != nil || r.off != len(r.b) {
		t.Fatalf("equipement %s : relecture invalide (err %v, %d/%d octets lus)", short8, r.err, r.off, len(r.b))
	}
}

// goldenEnergyCatalog : le catalogue de l energie des equipements que la production charge
// (`replaybuild.energieCatalog`), pour que les usages restants des goldens soient ceux du parc.
func goldenEnergyCatalog(t *testing.T) *equipenergy.Catalog {
	t.Helper()
	path := filepath.Join(racineDepot(t), "data", "titles", titreDeReference, "reference", "equipment_energy.json")
	cat, err := equipenergy.Load(path)
	if err != nil {
		t.Fatalf("catalogue de l energie des equipements %s : %v", path, err)
	}
	return cat
}

// TestGoldenEquipementRegenerate : LA SEULE PORTE D'ECRITURE des fichiers d equipement (cf. l'en-tete).
func TestGoldenEquipementRegenerate(t *testing.T) {
	cache := os.Getenv(miniFilmCacheEnv)
	switch {
	case !*updateGolden:
		t.Skip("regeneration de l equipement des builds : passer -update (et " + miniFilmCacheEnv + ")")
	case cache == "":
		t.Skip("regeneration de l equipement des builds : " + miniFilmCacheEnv + " non defini")
	}
	for _, b := range goldenBuilds() {
		t.Run(b.Short8, func(t *testing.T) {
			entry, err := b.mapQuant()
			if err != nil {
				t.Fatalf("carte %q : %v", b.Map, err)
			}
			g, err := decodeFilmInputsForEntry(b.Short8, filepath.Join(cache, b.Short8), entry)
			if err != nil {
				t.Fatalf("decodage de %s : %v", b.Short8, err)
			}
			w := &gwriter{}
			encodePoussees(w, g.Knockbacks, g.KnockbackStats)
			encodeEquipementPorte(w, g.CarriedEquipment, g.CarriedEquipmentStats)
			var out bytes.Buffer
			zw, err := gzip.NewWriterLevel(&out, gzip.BestCompression)
			if err != nil {
				t.Fatalf("gzip : %v", err)
			}
			if _, err := zw.Write(w.b); err != nil {
				t.Fatalf("gzip : %v", err)
			}
			if err := zw.Close(); err != nil {
				t.Fatalf("gzip : %v", err)
			}
			if err := os.WriteFile(equipementPath(b.Short8), out.Bytes(), 0o600); err != nil {
				t.Fatalf("ecriture %s : %v", equipementPath(b.Short8), err)
			}
			t.Fatalf("REECRIT %s (%d poussees, %d lectures d equipement porte, %d octets) — relancer sans "+
				"-update pour verifier", equipementPath(b.Short8), len(g.Knockbacks), len(g.CarriedEquipment), out.Len())
		})
	}
}
