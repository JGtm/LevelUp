package replay

// ground_weapon_pads_aggregate_test.go — LE CRITERE DE CATALOGUE (item 1.4 du plan) : un socle
// est-il une propriete de la CARTE, ou une propriete du MATCH ?
//
// LA QUESTION SE TRANCHE ENTRE DEUX FILMS DE LA MEME CARTE, et pas autrement. Si les socles
// mesures dans un film se retrouvent dans un autre film de la meme carte, ils appartiennent a
// la carte et peuvent devenir une donnee de REFERENCE versionnee (`map_weapon_pads.json`, au
// meme endroit et au meme format d'ecriture que `map_objectives.json`). Sinon ils restent PAR
// MATCH, et le registre le dit. Le critere ecrit avant la mesure : sur une carte vue dans >= 2
// films, >= 80 % des socles d'un film retrouvent un socle de MEME famille a < 1 m dans l'autre.
//
// POURQUOI UN SECOND PROCESSUS. Un seul decodage filmdec par process : chaque film est mesure
// separement par `TestGroundWeaponPads`, qui ecrit ses apparitions en clair (lignes `APPAR`).
// Cet agregateur ne decode RIEN — il relit ces sorties et rejoue la MEME regle de grappe
// (`gwPadsCluster`), donc les grappes qu'il compare sont celles que l'instrument a publiees.
//
// LECTURE SEULE. SOUS GARDE D'ENVIRONNEMENT (GW_PADS_AGG).
//
// USAGE (depuis apps/go-api) :
//
//	GW_PADS_AGG=<scratch>/pads_01e1f945.log,<scratch>/pads_75f1188f.log \
//	  go test ./internal/analysis/replay/ -run '^TestGroundWeaponPadsAggregate$' -timeout 10m -v

import (
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const gwPadsAggEnv = "GW_PADS_AGG" // liste de fichiers de sortie, separes par des virgules

// gwPadsOverlapMin est la part de socles d'un film qui doivent se retrouver dans l'autre pour
// que les socles soient une propriete de la CARTE. 0,80 — le critere tranche du plan (1.4),
// ecrit avant la mesure.
const gwPadsOverlapMin = 0.80

// gwPadsMatchRadiusM est la distance en deca de laquelle deux socles de films differents sont
// LE MEME socle. 1 m, comme le rayon de grappe : deux socles plus proches que le rayon
// n'auraient pas ete separes a l'interieur d'un film non plus.
const gwPadsMatchRadiusM = 1.0

func TestGroundWeaponPadsAggregate(t *testing.T) {
	raw := os.Getenv(gwPadsAggEnv)
	if raw == "" {
		t.Skipf("%s absent : agregation sautee", gwPadsAggEnv)
	}
	byFilm, mapOf := gwPadsReadLogs(t, strings.Split(raw, ","))
	if len(byFilm) == 0 {
		t.Fatalf("aucune ligne APPAR lue dans %s", raw)
	}
	byMap := map[string][]string{}
	films := make([]string, 0, len(byFilm))
	for f := range byFilm {
		films = append(films, f)
	}
	sort.Strings(films)
	for _, f := range films {
		byMap[mapOf[f]] = append(byMap[mapOf[f]], f)
	}
	maps := make([]string, 0, len(byMap))
	for m := range byMap {
		maps = append(maps, m)
	}
	sort.Strings(maps)

	// LES DEUX JEUX DE CANDIDATS SONT COMPARES, comme dans l'instrument par film : le critere
	// ecrit (`spawned`) et celui que l'item 1.0 designe (`at_rest`, sans vie delta).
	for _, set := range []string{gwPadsSetSpawned, gwPadsSetAtRest} {
		socles := map[string][]gwPadCluster{}
		for _, f := range films {
			sel := gwPadsSelect(byFilm[f], set)
			socles[f] = gwPadsKeep(gwPadsCluster(sel, gwPadRadiusM), gwPadMinHits)
			t.Logf("JEU %s · film %s · carte %s · apparitions %d · candidates %d · SOCLES %d",
				set, f, mapOf[f], len(byFilm[f]), len(sel), len(socles[f]))
		}
		gwPadsCompareMaps(t, set, maps, byMap, socles)
	}
}

// gwPadsSelect rend les candidates d'un jeu : `spawned` (item 1.1) ou `at_rest` (item 1.0 —
// `spawned` ET sans vie delta).
func gwPadsSelect(app []gwPadApparition, set string) []gwPadApparition {
	out := make([]gwPadApparition, 0, len(app))
	for _, a := range app {
		if a.Class != gwClassSpawned || (set == gwPadsSetAtRest && a.HasDelta) {
			continue
		}
		out = append(out, a)
	}
	return out
}

// gwPadsCompareMaps publie, carte par carte, le recouvrement des socles entre films — dans les
// DEUX SENS. Un seul sens suffirait a masquer un film pauvre : dix socles retrouves dans une
// carte qui en porte trente, c'est 100 % dans un sens et 33 % dans l'autre.
func gwPadsCompareMaps(
	t *testing.T, set string, maps []string, byMap map[string][]string,
	socles map[string][]gwPadCluster,
) {
	t.Helper()
	pairs, held := 0, 0
	for _, m := range maps {
		fs := byMap[m]
		if len(fs) < 2 {
			t.Logf("1.4 [%s] CARTE %s — vue dans %d film : le critere de catalogue ne s'y"+
				" applique pas", set, m, len(fs))
			continue
		}
		for i := 0; i < len(fs); i++ {
			for j := i + 1; j < len(fs); j++ {
				a, b := fs[i], fs[j]
				ab, an := gwPadsOverlap(socles[a], socles[b], true)
				ba, bn := gwPadsOverlap(socles[b], socles[a], true)
				ok := gwPadsRatio(ab, an) >= gwPadsOverlapMin && gwPadsRatio(ba, bn) >= gwPadsOverlapMin
				pairs++
				if ok {
					held++
				}
				t.Logf("1.4 [%s] CARTE %s — %s -> %s %s · %s -> %s %s · critere >= %.0f %% dans"+
					" les DEUX sens : %s", set, m, a, b, gwPadsPart(ab, an), b, a,
					gwPadsPart(ba, bn), 100*gwPadsOverlapMin, gwPadsVerdict(ok))
				pab, pan := gwPadsOverlap(socles[a], socles[b], false)
				pba, pbn := gwPadsOverlap(socles[b], socles[a], false)
				t.Logf("1.4 [%s] CARTE %s — MEME MESURE, FAMILLE IGNOREE (la POSITION seule) :"+
					" %s -> %s %s · %s -> %s %s", set, m, a, b, gwPadsPart(pab, pan), b, a,
					gwPadsPart(pba, pbn))
				gwPadsLogUnmatched(t, set, m, a, b, socles[a], socles[b])
			}
		}
	}
	t.Logf("1.4 [%s] VERDICT — paires de films de meme carte %d · tenant le critere %d (%s)",
		set, pairs, held, gwPadsPart(held, pairs))
}

// gwPadsOverlap compte les socles de `a` qui retrouvent un socle de meme nature a moins de
// `gwPadsMatchRadiusM` dans `b`. `byFamily` exige en plus la MEME famille.
//
// LES DEUX MESURES SONT PUBLIEES, ET CE N'EST PAS UN SEUIL QU'ON ABAISSE : ce sont deux
// QUESTIONS distinctes. « Le socle est-il au meme endroit ? » et « porte-t-il la meme arme ? »
// ne recoivent pas la meme reponse, et c'est le fait qui compte. Le critere du plan (item 1.4)
// reste celui a famille exigee ; la mesure a famille ignoree dit ce que le desaccord signifie.
func gwPadsOverlap(a, b []gwPadCluster, byFamily bool) (hit, total int) {
	for _, p := range a {
		total++
		for _, q := range b {
			if p.Kind != q.Kind || (byFamily && p.Family != q.Family) {
				continue
			}
			if gwPadsDist(p.X, p.Y, p.Z, q.X, q.Y, q.Z) <= gwPadsMatchRadiusM {
				hit++
				break
			}
		}
	}
	return hit, total
}

// gwPadsLogUnmatched liste les socles ORPHELINS d'une paire : ce sont eux qui disent POURQUOI
// le critere tient ou non. Un compte sans le detail ne se relit pas.
func gwPadsLogUnmatched(t *testing.T, set, m, a, b string, pa, pb []gwPadCluster) {
	t.Helper()
	for _, p := range pa {
		found := false
		for _, q := range pb {
			if p.Kind == q.Kind && p.Family == q.Family &&
				gwPadsDist(p.X, p.Y, p.Z, q.X, q.Y, q.Z) <= gwPadsMatchRadiusM {
				found = true
				break
			}
		}
		if !found {
			t.Logf("    ORPHELIN\t%s\t%s\t%s (absent de %s)\t%s\t%s\t%.2f\t%.2f\t%.2f\tx%d",
				set, m, a, b, p.Kind, p.Family, p.X, p.Y, p.Z, len(p.TS))
		}
	}
}

func gwPadsRatio(k, n int) float64 {
	if n == 0 {
		return 0
	}
	return float64(k) / float64(n)
}

func gwPadsVerdict(ok bool) string {
	if ok {
		return "TENU"
	}
	return "NON TENU"
}

// gwPadsReadLogs relit les lignes `APPAR` des sorties de `TestGroundWeaponPads`. Une ligne mal
// formee est COMPTEE et signalee, jamais ignoree en silence : un agregat qui perd des lignes
// sans le dire rend un recouvrement flatteur.
func gwPadsReadLogs(t *testing.T, paths []string) (map[string][]gwPadApparition, map[string]string) {
	t.Helper()
	byFilm, mapOf, bad := map[string][]gwPadApparition{}, map[string]string{}, 0
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		blob, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("sortie illisible (%s) : %v", p, err)
		}
		for _, line := range strings.Split(string(blob), "\n") {
			i := strings.Index(line, "APPAR\t")
			if i < 0 {
				continue
			}
			film, mapName, a, ok := gwPadsParseAppar(line[i+len("APPAR\t"):])
			if !ok {
				bad++
				continue
			}
			byFilm[film] = append(byFilm[film], a)
			mapOf[film] = mapName
		}
	}
	if bad > 0 {
		t.Errorf("%d lignes APPAR mal formees : l'agregat serait partiel", bad)
	}
	return byFilm, mapOf
}

// gwPadsParseAppar decoupe une ligne APPAR : film, carte, nature, famille, x, y, z, t, classe,
// vie delta.
func gwPadsParseAppar(s string) (film, mapName string, a gwPadApparition, ok bool) {
	f := strings.Split(strings.TrimRight(s, "\r"), "\t")
	if len(f) != 10 {
		return "", "", a, false
	}
	x, e1 := strconv.ParseFloat(f[4], 32)
	y, e2 := strconv.ParseFloat(f[5], 32)
	z, e3 := strconv.ParseFloat(f[6], 32)
	ts, e4 := strconv.ParseUint(f[7], 10, 64)
	if e1 != nil || e2 != nil || e3 != nil || e4 != nil {
		return "", "", a, false
	}
	a = gwPadApparition{
		Kind: f[2], Family: f[3],
		X: float32(x), Y: float32(y), Z: float32(z),
		TUS: ts, Class: f[8], HasDelta: f[9] == "true",
	}
	return f[0], f[1], a, true
}
