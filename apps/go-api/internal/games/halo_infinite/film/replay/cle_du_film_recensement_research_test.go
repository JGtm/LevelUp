package replay

// cle_du_film_recensement_research_test.go — LE RECENSEMENT DES CLEFS DU PARC, SANS DECODER
// (lot 3.1.1, point de depart de la politique « cle inconnue = film mis de cote »).
//
// # CE QU IL MESURE, ET POURQUOI IL FALLAIT LE MESURER AVANT D ECRIRE LA POLITIQUE
//
// Mettre un film de cote est un changement de COMPORTEMENT : des films aujourd hui cuits ne le
// seraient plus. Le seul chiffre qui dit ce que la politique coute est donc le nombre de films
// du cache dont la clef ECRITE est absente de la table de profil. La decision de l utilisateur
// du 2026-09-17 (V20 (2)) est prise SUR ce chiffre : « on peut les ignorer, ca depend du
// volume ».
//
// # IL NE DECODE RIEN, ET C EST LA CONTRAINTE QUI L A DESSINE
//
// Un seul decodage a la fois tient sur ce poste, et le parc compte plus de 1 300 films. Ce banc
// lit `chunk_00` SEUL — la version de format, la version majeure, la section d identification —
// par [CleDeChunk0]. Aucun flux de bits, aucun autre chunk, aucun verrou solo.
//
// USAGE :
//
//	CGO_ENABLED=0 CLE_FILM_PARC=<racine portant data/> \
//	  go test ./internal/games/halo_infinite/film/replay -run TestRecensementDesClefsDuParc -v

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// envParcDesClefs : la racine du parc (celle qui porte `data/cache/film_chunks`).
const envParcDesClefs = "CLE_FILM_PARC"

// TestRecensementDesClefsDuParc compte les films du cache PAR CLEF ECRITE, et nomme ceux que la
// table de profil ne connait pas.
//
// IL NE JUGE PAS : il n echoue que si le parc est illisible. Le nombre de clefs inconnues est un
// FAIT du jour, pas un invariant — le jour ou le jeu change de build, il monte, et c est
// precisement ce que la politique attend.
func TestRecensementDesClefsDuParc(t *testing.T) {
	racine := os.Getenv(envParcDesClefs)
	if racine == "" {
		t.Skip("recensement des clefs : " + envParcDesClefs + " non defini")
	}
	dirs, err := filepath.Glob(filepath.Join(filmcache.ChunksRoot(filepath.Join(racine, "data", "cache")), "*"))
	if err != nil || len(dirs) == 0 {
		t.Fatalf("aucun film au cache sous %q (%v)", racine, err)
	}
	parClef := map[string]int{}
	empreinteDeLaClef := map[string]string{}
	var refuses []string
	var illisibles int
	for _, d := range dirs {
		chunk0, ok := chunk00DuCache(d)
		if !ok {
			illisibles++
			continue
		}
		cle := CleDeChunk0(chunk0)
		clef := cle.Ecrite
		if clef == "" {
			clef = "(aucune clef lisible)"
		}
		parClef[clef]++
		if _, vue := empreinteDeLaClef[clef]; !vue {
			empreinteDeLaClef[clef] = empreinteDuRegistreDeChunk0(chunk0)
		}
		if cle.Refusee() {
			refuses = append(refuses, filepath.Base(d)+" "+clef+" "+empreinteDeLaClef[clef])
		}
	}
	rapporterLesClefs(t, len(dirs), illisibles, parClef, empreinteDeLaClef, refuses)
}

// empreinteDuRegistreDeChunk0 rend l empreinte du registre ECS et ses deux denominateurs, sous
// la forme du catalogue (`0x%016x`). Elle sert a joindre le recensement a la table des
// empreintes de `film_profiles.json` sans ouvrir un second outil.
func empreinteDuRegistreDeChunk0(chunk0 []byte) string {
	reg, err := grammar.ParseRegistryChunk(chunk0)
	if err != nil {
		return "(registre illisible)"
	}
	return fmt.Sprintf("0x%016x blocs=%d slots=%d", grammar.RegistryFingerprint(reg),
		len(reg.Archetypes), grammar.RegistryNamedSlots(reg))
}

// rapporterLesClefs ecrit le tableau du recensement — une ligne par clef, puis le verdict.
func rapporterLesClefs(t *testing.T, films, illisibles int, parClef map[string]int,
	empreintes map[string]string, refuses []string) {
	t.Helper()
	clefs := make([]string, 0, len(parClef))
	for k := range parClef {
		clefs = append(clefs, k)
	}
	sort.Slice(clefs, func(i, j int) bool { return parClef[clefs[i]] > parClef[clefs[j]] })
	t.Logf("RECENSEMENT DES CLEFS — %d film(s) au cache, %d sans chunk_00 lisible", films, illisibles)
	for _, k := range clefs {
		t.Logf("  %-24s %5d   %s", k, parClef[k], empreintes[k])
	}
	if len(refuses) == 0 {
		t.Logf("VERDICT : %d film(s) a clef INCONNUE — la politique n ecarte rien aujourd hui", 0)
		return
	}
	sort.Strings(refuses)
	t.Logf("VERDICT : %d film(s) a clef INCONNUE, la politique les ecarterait :", len(refuses))
	for _, r := range refuses {
		t.Logf("  %s", r)
	}
}

// chunk00DuCache lit et decompresse le `chunk_00` d un repertoire de film du cache. Faux = le
// film n en porte pas (bobine partielle) ou le fichier est illisible.
func chunk00DuCache(dir string) ([]byte, bool) {
	brut, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("chunk_%02d.bin", 0)))
	if err != nil {
		return nil, false
	}
	inflate := source.Inflate(brut)
	if len(inflate) == 0 {
		return nil, false
	}
	return inflate, true
}
