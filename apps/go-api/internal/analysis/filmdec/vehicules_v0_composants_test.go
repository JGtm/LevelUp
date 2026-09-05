package filmdec

// vehicules_v0_composants_test.go — INSTRUMENT JETABLE du lot V0 (cadrage vehicules, 2026-08-31).
//
// DEUX QUESTIONS, DEUX LECTURES.
//
//  1. LE REGISTRE DU FILM dit-il la meme chose que `testdata/ecs_table.tsv` pour `ti=40` ? La
//     table est une piece du depot, datee ; le `chunk_00` du film est la source. On les
//     confronte au lieu de croire la table sur parole.
//
//  2. QUELS COMPOSANTS de `ti=40` le FLUX porte-t-il reellement ? La table dit lesquels sont
//     portes par le decodeur ; elle ne dit pas lesquels arrivent. Un composant non porte qui
//     n'arrive jamais ne bloque rien ; un composant non porte present dans un record sur deux
//     est le prochain deserialiseur a ecrire. C'est cette frequence qui ordonne le lot V1.
//
// SELECTIVITE : le comptage n'accepte QUE les records deja acceptes par le balayage de
// production (`matchWorldObjectRecord` sur la bande de l'archetype, i0 present, position
// dequantifiee valide, avance apres le record accepte). Un histogramme construit sur des ancres
// non validees compterait surtout des faux positifs de balayage bit a bit.
//
// LECTURE SEULE : aucun fichier ecrit, aucune base ouverte. A supprimer a la cloture du lot V0.
//
//	CGO_ENABLED=0 V0_CHUNK_DIRS=<cache>/film_chunks/8a049c50 \
//	  go test ./internal/analysis/filmdec/ -run TestV0Composants -v -timeout 60m

import (
	"os"
	"sort"
	"strings"
	"testing"
)

// v0ChunkDirsEnv porte les repertoires de chunks a lire (separes par des virgules).
const v0ChunkDirsEnv = "V0_CHUNK_DIRS"

// v0VehiculeTI est l'archetype vehicule presume. Presume est le mot juste : c'est ce test qui
// le confronte au registre du film.
const v0VehiculeTI = 40

// v0Dirs rend les repertoires demandes.
func v0Dirs(t *testing.T) []string {
	t.Helper()
	v := os.Getenv(v0ChunkDirsEnv)
	if v == "" {
		t.Skipf("mesure non demandee : %s vide", v0ChunkDirsEnv)
	}
	var out []string
	for _, s := range strings.Split(v, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// v0Detail arme le listage composant par composant (sinon seule la ligne de synthese sort).
const v0DetailEnv = "V0_DETAIL"

// TestV0ComposantsRegistre confronte le registre du film a la table ECS pour `ti=40`, ET publie
// l'EMPREINTE du registre : la table de reference vaut pour UN build, et le corpus vise est
// posterieur de plusieurs mois aux films sur lesquels la grammaire a ete etablie.
func TestV0ComposantsRegistre(t *testing.T) {
	for _, dir := range v0Dirs(t) {
		raw, err := ReadFilmChunk(dir, 0)
		if err != nil {
			t.Logf("%s : chunk_00 illisible : %v", dir, err)
			continue
		}
		reg, err := ParseRegistryChunk(raw)
		if err != nil {
			t.Logf("%s : registre illisible : %v", dir, err)
			continue
		}
		fp := RegistryFingerprint(reg)
		arch, ok := reg.Archetype(v0VehiculeTI)
		if !ok {
			t.Logf("V0 %s — empreinte %#x — archetype %d ABSENT du registre", dir, fp, v0VehiculeTI)
			continue
		}
		t.Logf("V0 %s — empreinte %#x (reference %#x, %v) — ti=%d porte %d composants, i33=%q i47=%q",
			dir, fp, KnownRegistryFingerprint, fp == KnownRegistryFingerprint,
			v0VehiculeTI, len(arch.Components),
			v0NomAt(arch.Components, 33), v0NomAt(arch.Components, 47))
		if os.Getenv(v0DetailEnv) == "" {
			continue
		}
		for i, c := range arch.Components {
			t.Logf("    i%-2d %s", i, c)
		}
	}
}

// v0NomAt rend le nom du composant d'index i, ou une marque d'absence.
func v0NomAt(comps []string, i int) string {
	if i < len(comps) {
		return comps[i]
	}
	return "(absent)"
}

// TestV0ComposantsFlux histogramme les composants presents dans les records delta de `ti=40`.
func TestV0ComposantsFlux(t *testing.T) {
	for _, dir := range v0Dirs(t) {
		v0FluxUnFilm(t, dir)
	}
}

// v0FluxUnFilm mesure UN film.
func v0FluxUnFilm(t *testing.T, dir string) {
	t.Helper()
	n := CountFilmChunks(dir)
	if n == 0 {
		t.Logf("%s : aucun chunk", dir)
		return
	}
	noms := v0NomsComposants(dir)
	band := worldObjectSlotBandDir(dir, n, v0VehiculeTI)
	if len(band) == 0 {
		t.Logf("V0 %s — aucun slot ti=%d aux images-cles : rien a compter", dir, v0VehiculeTI)
		return
	}
	hist := map[int]int{}
	records := 0
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(data) {
			if p.Type != PacketTypeDelta {
				continue
			}
			r, h := v0ScanPayload(p.Payload(data), band)
			records += r
			for k, v := range h {
				hist[k] += v
			}
		}
	}
	idx := make([]int, 0, len(hist))
	for k := range hist {
		idx = append(idx, k)
	}
	sort.Ints(idx)
	t.Logf("V0 %s — %d records delta ti=%d acceptes (bande %d slots)", dir, records, v0VehiculeTI, len(band))
	for _, k := range idx {
		nom := "(hors archetype)"
		if k < len(noms) {
			nom = noms[k]
		}
		t.Logf("    i%-2d %6d records (%5.1f %%)  %s", k, hist[k], 100*float64(hist[k])/float64(records), nom)
	}
}

// v0NomsComposants rend les noms des composants de `ti=40` d'apres le registre du film.
func v0NomsComposants(dir string) []string {
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		return nil
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		return nil
	}
	arch, ok := reg.Archetype(v0VehiculeTI)
	if !ok {
		return nil
	}
	return arch.Components
}

// v0ScanPayload reprend EXACTEMENT l'acceptation de `scanProjectileRecords` (meme ancre, meme
// exigence d'i0, meme rejet de position, meme avance) et rend en plus le masque de chaque
// record accepte. Rien n'est ajoute a la selectivite : c'est le meme filtre, observe.
func v0ScanPayload(pay []byte, band map[uint32]bool) (int, map[int]int) {
	hist := map[int]int{}
	records := 0
	posBits := projPosBits()
	// Bornes du monde neutres : la position ne sert ici qu'au filtre de porte / quantum sature,
	// pas a une coordonnee publiee. Un intervalle [0,1] par axe suffit et ne change aucun rejet.
	wr := Vec3Range{{Min: 0, Max: 1}, {Min: 0, Max: 1}, {Min: 0, Max: 1}}
	limit := len(pay)*8 - (worldObjectHeaderBits + worldObjectIndexBits + posBits)
	for p := 0; p <= limit; p++ {
		rec, ok := matchWorldObjectRecord(pay, p, band)
		if !ok || rec.Idx[0] != 0 {
			continue
		}
		if _, ok := decodeWorldObjectPos(pay, rec.After, &wr); !ok {
			continue
		}
		records++
		for _, i := range rec.Idx {
			hist[i]++
		}
		p += posBits
	}
	return records, hist
}
