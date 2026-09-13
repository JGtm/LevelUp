package filmdec

// mesure_entete_ti9_test.go — MESURE M : L'EN-TETE D'UN RECORD D'IMAGE-CLE EST-IL PROPRE AU
// TYPE D'ENTITE ?
//
// LA THESE MESUREE (fork ChaseWoodhams/LevelUp, commit 2a8b21a51). Le decodeur suppose un
// en-tete de 64 bits pour TOUS les types d'entite (`keyframeHeaderBits`), valeur heritee d'un
// utilitaire ecrit pour le BIPED. Le fork affirme que cet en-tete est PROPRE AU TYPE, et que
// ti=9 (« managed-player ») en demande 47 : ce serait le seul prefixe, sur 300 essayes, qui
// rende HUIT entites porteuses d'un `managed-player-team-designator-component` (i0, R(4)) a
// valeur STABLE, reparties 4-4 sur tout le film, dans six films. Et a 47 bits le biped
// (ti=35) passerait de ~19 736 composants decodes a ZERO.
//
// LA GRAMMAIRE REJOUEE. Le fork lit un record d'image-cle comme :
//
//	[en-tete: H bits][etat par defaut de l'archetype][1 bit porte has-components][masque + composants]
//
// H = 47 pour ti=9 (prefixe total mesure 61, moins les 14 bits de `consumeDefaultStateTI9` =
// V(1) + 6 + 6 + 1), H = 64 pour le biped. La marche reutilise les deserialiseurs et la
// boucle de composants de PRODUCTION (`consumeKeyframeDefaultState`, `decodeDeltaWithArch`) :
// rien n'est recopie, seul l'offset de depart change. AUCUN code de production n'est modifie
// par cette mesure.
//
// LE PIEGE QUE CE BANC REFUSE (documente par le fork). « 0 desync » ne prouve RIEN : un
// masque lu a zero fait sortir la boucle immediatement, sans consommer un seul composant.
// Toute ligne de ce banc publie donc le nombre de composants decodes A COTE du compte de
// desyncs, et un prefixe sans composant decode n'est jamais compte comme candidat.
//
// LA VERITE EXTERNE. Les films choisis sont des matchs d'arene 4v4 dont la table des scores
// donne la repartition (4 joueurs d'equipe 0, 4 d'equipe 1, tous presents du debut a la fin).
// C'est la seule grandeur visee dont on connaisse la valeur par ailleurs.
//
// LANCEMENT (les films ne sont pas versionnes ; sans la garde, tout se saute) :
//
//	MESURE_TI9_ROOT=<depot>/data/cache/film_chunks \
//	MESURE_TI9_IDS=4a93f0e2,8b512df2,ce083875 \
//	  go test ./internal/analysis/filmdec/ -run MesureEnteteTI9 -v -timeout 60m
//
// Le balayage de prefixes (contre-epreuve d'unicite) demande en plus MESURE_TI9_BALAYAGE=1.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const (
	mesureTI9RootEnv     = "MESURE_TI9_ROOT"
	mesureTI9IDsEnv      = "MESURE_TI9_IDS"
	mesureTI9BalayageEnv = "MESURE_TI9_BALAYAGE"
	// mesureTI9Designator : le composant i0 de ti=9, consomme R(4) par `traverse.go`.
	mesureTI9Designator = "managed-player-team-designator-component"
	// mesureTI9DesignatorBits : largeur du designateur (FUN_140f581e8).
	mesureTI9DesignatorBits = 4
	// mesureTI9These : la largeur d'en-tete que le fork attribue a ti=9.
	mesureTI9These = 47
	// mesureTI9AttenduEntites / mesureTI9AttenduParEquipe : le critere du fork, et la verite
	// de la table des scores des films retenus.
	mesureTI9AttenduEntites   = 8
	mesureTI9AttenduParEquipe = 4
)

// mesureEnteteCandidats : les largeurs d'en-tete confrontees. 47 = la these du fork ; 64 =
// `keyframeHeaderBits`, la lecture historique ; 108 = `keyframeFullStateHeaderBits`, celle de
// FUN_142e2bfd0 (cf. keyframe_fullstate_loop.go).
var mesureEnteteCandidats = []int{mesureTI9These, keyframeHeaderBits, keyframeFullStateHeaderBits}

// mesureEnteteTypes : les deux types confrontes. 9 porte le designateur d'equipe ; 35 est le
// biped, dont l'effondrement a 47 bits serait la contre-epreuve de la these « par type ».
var mesureEnteteTypes = []int{9, bipedDefaultStateTypeIndex}

// mesureFilm porte un film ouvert.
type mesureFilm struct {
	id     string
	dir    string
	reg    *Registry
	chunks int
}

// mesureCell est le releve d'un couple (en-tete, type d'entite) sur tous les films.
type mesureCell struct {
	records   int // records d'image-cle de ce type rencontres
	traverses int // records dont l'archetype est connu et la marche jouee
	desync    int // marches arretees sur un composant non porte
	vides     int // marches SANS AUCUN composant decode (le piege : ce n'est pas un succes)
	comps     int // composants decodes, tous records confondus
	designs   int // lectures du designateur d'equipe
}

// mesureObs rassemble les lectures du designateur pour une entite (film, slot).
type mesureObs struct {
	valeurs map[uint64]int
	premier int // rang du chunk de la premiere lecture
	dernier int
}

func mesureOuvre(t *testing.T) []mesureFilm {
	t.Helper()
	root := os.Getenv(mesureTI9RootEnv)
	if root == "" {
		t.Skipf("%s absent : banc de mesure saute (les films ne sont pas versionnes)", mesureTI9RootEnv)
	}
	var out []mesureFilm
	for _, id := range strings.Split(os.Getenv(mesureTI9IDsEnv), ",") {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		dir := filepath.Join(root, id)
		raw, err := ReadFilmChunk(dir, 0)
		if err != nil {
			t.Fatalf("%s : chunk_00 illisible (%v)", id, err)
		}
		reg, err := ParseRegistryChunk(raw)
		if err != nil {
			t.Fatalf("%s : registre illisible (%v)", id, err)
		}
		out = append(out, mesureFilm{id: id, dir: dir, reg: reg, chunks: CountFilmChunks(dir)})
	}
	if len(out) == 0 {
		t.Fatalf("%s vide : preciser les identifiants de films", mesureTI9IDsEnv)
	}
	return out
}

// mesureMarche rejoue UN record d'image-cle avec un en-tete de `hdr` bits. Rend la trace et
// les valeurs de designateur qu'elle porte, relues a `CompResult.StartBit` — position fixee
// AVANT le deserialiseur du composant. Le decodeur ne CAPTURE pas ce composant, et la mesure
// n'a pas a le lui faire capturer : elle relit les memes bits.
func mesureMarche(pay []byte, r KeyframeRec, reg *Registry, hdr int) (EntityTrace, []uint64, bool) {
	arch, ok := reg.Archetype(r.TI)
	if !ok {
		return EntityTrace{DesyncAt: -1}, nil, false
	}
	br := NewBitReader(pay)
	br.SetBitPos(r.Bit + hdr)
	consumeKeyframeDefaultState(br, uint32(r.TI))
	br.ReadBit() // porte has-components
	tr := decodeDeltaWithArch(br, arch, uint32(r.TI))
	var vals []uint64
	for _, c := range tr.Comps {
		if c.Name == mesureTI9Designator && c.Ported {
			vals = append(vals, kfReadBits(pay, c.StartBit, mesureTI9DesignatorBits))
		}
	}
	return tr, vals, true
}

// mesureParcours applique `visite` a chaque record d'image-cle du film.
func mesureParcours(f mesureFilm, visite func(chunk int, pay []byte, r KeyframeRec)) {
	for c := 1; c <= f.chunks; c++ {
		chunk, err := ReadFilmChunk(f.dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(chunk) {
			if p.Type != PacketTypeKeyframe {
				continue
			}
			pay := p.Payload(chunk)
			for _, r := range WalkKeyframeWorld(pay) {
				visite(c, pay, r)
			}
		}
	}
}

func mesureTypeVise(ti int) bool {
	for _, v := range mesureEnteteTypes {
		if v == ti {
			return true
		}
	}
	return false
}

func TestMesureEnteteTI9(t *testing.T) {
	films := mesureOuvre(t)
	defer LockProcessDecode()()

	cells := map[[2]int]*mesureCell{}
	obs := map[string]map[int]*mesureObs{} // film -> slot -> lectures (en-tete 47, ti 9)
	for _, f := range films {
		obs[f.id] = map[int]*mesureObs{}
	}
	for _, f := range films {
		mesureParcours(f, func(c int, pay []byte, r KeyframeRec) {
			if !mesureTypeVise(r.TI) {
				return
			}
			for _, hdr := range mesureEnteteCandidats {
				mesureCompte(cells, hdr, r, pay, f, c, obs)
			}
		})
	}
	mesureRapportCellules(t, cells)
	mesureRapportDesignateurs(t, films, obs)
}

// mesureCompte remplit la cellule (hdr, ti) pour un record, et releve le designateur quand la
// combinaison mesuree est celle de la these (en-tete 47 sur ti=9).
func mesureCompte(cells map[[2]int]*mesureCell, hdr int, r KeyframeRec, pay []byte,
	f mesureFilm, chunk int, obs map[string]map[int]*mesureObs) {
	key := [2]int{hdr, r.TI}
	cell := cells[key]
	if cell == nil {
		cell = &mesureCell{}
		cells[key] = cell
	}
	cell.records++
	tr, vals, ok := mesureMarche(pay, r, f.reg, hdr)
	if !ok {
		return
	}
	cell.traverses++
	cell.comps += len(tr.Comps)
	if tr.DesyncAt >= 0 {
		cell.desync++
	}
	if len(tr.Comps) == 0 {
		cell.vides++
	}
	cell.designs += len(vals)
	if hdr != mesureTI9These || r.TI != 9 || len(vals) == 0 {
		return
	}
	o := obs[f.id][r.Slot]
	if o == nil {
		o = &mesureObs{valeurs: map[uint64]int{}, premier: chunk}
		obs[f.id][r.Slot] = o
	}
	o.dernier = chunk
	for _, v := range vals {
		o.valeurs[v]++
	}
}

func mesureRapportCellules(t *testing.T, cells map[[2]int]*mesureCell) {
	t.Helper()
	t.Logf("")
	t.Logf("=== A. MARCHE DES RECORDS D'IMAGE-CLE, par largeur d'en-tete et par type ===")
	t.Logf("%8s %5s %10s %10s %9s %9s %10s %13s", "en-tete", "ti", "records", "traverses",
		"desync", "vides", "comps", "designateurs")
	keys := make([][2]int, 0, len(cells))
	for k := range cells {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(a, b int) bool {
		if keys[a][1] != keys[b][1] {
			return keys[a][1] < keys[b][1]
		}
		return keys[a][0] < keys[b][0]
	})
	for _, k := range keys {
		c := cells[k]
		t.Logf("%8d %5d %10d %10d %9d %9d %10d %13d", k[0], k[1], c.records, c.traverses,
			c.desync, c.vides, c.comps, c.designs)
	}
}

// mesureRapportDesignateurs publie la stabilite par entite puis la repartition, et TRANCHE :
// le critere du fork est 8 entites stables reparties 4-4.
func mesureRapportDesignateurs(t *testing.T, films []mesureFilm, obs map[string]map[int]*mesureObs) {
	t.Helper()
	t.Logf("")
	t.Logf("=== B. DESIGNATEUR D'EQUIPE (en-tete %d, ti=9) : une ligne par entite ===", mesureTI9These)
	t.Logf("%10s %8s %10s %9s %8s %8s  %s", "film", "slot", "lectures", "stable", "chunk1",
		"chunkN", "valeurs vues")
	verdicts := map[string]string{}
	for _, f := range films {
		verdicts[f.id] = mesureRapportFilm(t, f, obs[f.id])
	}
	t.Logf("")
	t.Logf("=== C. VERDICT PAR FILM (critere du fork : 8 entites stables, 4-4) ===")
	for _, f := range films {
		t.Logf("%10s : %s", f.id, verdicts[f.id])
	}
}

func mesureRapportFilm(t *testing.T, f mesureFilm, slotsObs map[int]*mesureObs) string {
	t.Helper()
	slots := make([]int, 0, len(slotsObs))
	for s := range slotsObs {
		slots = append(slots, s)
	}
	sort.Ints(slots)
	repart := map[uint64]int{}
	stables := 0
	for _, s := range slots {
		o := slotsObs[s]
		n, val, stable := mesureResume(o)
		if stable {
			stables++
			repart[val]++
		}
		t.Logf("%10s %8d %10d %9v %8d %8d  %s", f.id, s, n, stable, o.premier, o.dernier,
			mesureHisto(o.valeurs))
	}
	t.Logf("%10s  -> %d entites, %d stables, repartition %s", f.id, len(slots), stables,
		mesureHisto(repart))
	return mesureVerdictFilm(len(slots), stables, repart)
}

func mesureResume(o *mesureObs) (n int, val uint64, stable bool) {
	for v, c := range o.valeurs {
		n += c
		val = v
	}
	return n, val, len(o.valeurs) == 1
}

func mesureHisto(m map[uint64]int) string {
	keys := make([]uint64, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(a, b int) bool { return keys[a] < keys[b] })
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%d x%d", k, m[k]))
	}
	return strings.Join(parts, " · ")
}

func mesureVerdictFilm(entites, stables int, repart map[uint64]int) string {
	if entites == 0 {
		return "AUCUNE entite : le designateur n'est jamais lu"
	}
	if entites != mesureTI9AttenduEntites || stables != mesureTI9AttenduEntites {
		return fmt.Sprintf("NON TENU : %d entites (%d stables), attendu %d stables",
			entites, stables, mesureTI9AttenduEntites)
	}
	if len(repart) != 2 {
		return fmt.Sprintf("NON TENU : %d valeurs distinctes, attendu 2", len(repart))
	}
	for _, c := range repart {
		if c != mesureTI9AttenduParEquipe {
			return fmt.Sprintf("NON TENU : repartition %s, attendu %d-%d", mesureHisto(repart),
				mesureTI9AttenduParEquipe, mesureTI9AttenduParEquipe)
		}
	}
	return "TENU : 8 entites stables, 4-4"
}

// mesureBalayage accumule, pour un prefixe donne, les composants decodes et les lectures du
// designateur par (film, slot).
type mesureBalayage struct {
	comps int
	obs   map[string]map[int]map[uint64]int
}

func (b *mesureBalayage) note(film string, slot int, v uint64) {
	if b.obs[film] == nil {
		b.obs[film] = map[int]map[uint64]int{}
	}
	if b.obs[film][slot] == nil {
		b.obs[film][slot] = map[uint64]int{}
	}
	b.obs[film][slot][v]++
}

// TestMesureEnteteTI9Balayage est la CONTRE-EPREUVE D'UNICITE : si 47 est le bon prefixe, il
// doit etre le SEUL, sur la plage essayee, a rendre huit entites stables reparties 4-4. Le
// balayage est lent (une marche par prefixe et par record) et porte donc sa propre garde.
func TestMesureEnteteTI9Balayage(t *testing.T) {
	if os.Getenv(mesureTI9BalayageEnv) == "" {
		t.Skipf("%s absent : balayage de prefixes saute", mesureTI9BalayageEnv)
	}
	films := mesureOuvre(t)
	defer LockProcessDecode()()
	maxW := 300
	if v, err := strconv.Atoi(os.Getenv("MESURE_TI9_MAX")); err == nil && v > 0 {
		maxW = v
	}
	sc := make([]*mesureBalayage, maxW+1)
	for i := range sc {
		sc[i] = &mesureBalayage{obs: map[string]map[int]map[uint64]int{}}
	}
	for _, f := range films {
		mesureParcours(f, func(_ int, pay []byte, r KeyframeRec) {
			if r.TI != 9 {
				return
			}
			for w := 0; w <= maxW; w++ {
				tr, vals, ok := mesureMarche(pay, r, f.reg, w)
				if !ok {
					continue
				}
				sc[w].comps += len(tr.Comps)
				for _, v := range vals {
					sc[w].note(f.id, r.Slot, v)
				}
			}
		})
	}
	mesureRapportBalayage(t, films, sc, maxW)
}

func mesureRapportBalayage(t *testing.T, films []mesureFilm, sc []*mesureBalayage, maxW int) {
	t.Helper()
	t.Logf("")
	t.Logf("=== D. BALAYAGE DES PREFIXES 0..%d (ti=9) : candidats qui tiennent le critere 4-4 ===", maxW)
	t.Logf("%8s %10s  %s", "en-tete", "comps", "verdict par film")
	tenus, vivants := 0, 0
	for w := 0; w <= maxW; w++ {
		// Piege « 0 desync sans composant » : un prefixe qui ne decode RIEN n'est pas un
		// candidat, quelle que soit la proprete apparente de sa marche.
		if sc[w].comps == 0 {
			continue
		}
		vivants++
		lignes := make([]string, 0, len(films))
		ok := true
		for _, f := range films {
			v := mesureVerdictBalayage(sc[w].obs[f.id])
			if !strings.HasPrefix(v, "TENU") {
				ok = false
			}
			lignes = append(lignes, f.id+":"+v)
		}
		if !ok {
			continue
		}
		tenus++
		t.Logf("%8d %10d  %s", w, sc[w].comps, strings.Join(lignes, " | "))
	}
	t.Logf("prefixes qui decodent AU MOINS UN composant : %d sur %d essayes", vivants, maxW+1)
	t.Logf("prefixes qui tiennent le critere sur TOUS les films : %d", tenus)
}

func mesureVerdictBalayage(slots map[int]map[uint64]int) string {
	repart := map[uint64]int{}
	stables := 0
	for _, vals := range slots {
		if len(vals) != 1 {
			continue
		}
		stables++
		for v := range vals {
			repart[v]++
		}
	}
	return mesureVerdictFilm(len(slots), stables, repart)
}
