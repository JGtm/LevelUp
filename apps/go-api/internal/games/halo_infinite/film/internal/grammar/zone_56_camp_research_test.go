//go:build research

package grammar

// zone_56_camp_research_test.go — LOT 5.6, POINT 1 : CE QUE `ti=23` PORTE VRAIMENT.
//
// # LA QUESTION, ET CE QUE L ECRIVAIN A DEJA REPONDU
//
// Le lot cherche LE CAMP QUI POUSSE LA JAUGE D UNE ZONE, lu dans le film au lieu d etre deduit
// de l issue de la rampe (couche de publication, lot 5.2a.3). Le suspect designe au brief etait
// l archetype `zones` `ti=23` (`selectable-zone-data-component`, 32 instances, `deser_non_cable`),
// dont la table ECS dit qu il porte « l identifiant, la POSITION et l ETAT » d une zone de mode.
//
// L ECRIVAIN A ETE LU D ABORD (Ghidra, lecture seule), et il REFUTE « l etat » :
//
//	FUN_142ed6cec(desc, flux, etat)                        // porteur, 1 instruction utile
//	  -> FUN_141454340(etat->zones + desc->index * 0x14)   // 20 octets par zone
//	       FUN_14080dec4(flux, "zone-name", base+0x00)     // R(32) : identifiant de chaine
//	       FUN_14076e494(flux, base+0x04, 0x10, 0, 1, 0)   // position ABSOLUE quantifiee
//	       *(base+0x10) = FUN_1407f1ff4(flux, "associated-participant-handle")
//	                        -> FUN_1407f2058 : R(1) ; si 0 -> R(5) ; sinon 0xFFFFFFFF
//
// Soit, par zone : un identifiant de chaine, une position, et un HANDLE DE PARTICIPANT sur
// 5 bits (32 participants). Ni jauge, ni camp, ni etat de capture — et le 33e composant de
// l archetype (`i32 participant-spawn-availability-mask-data-component`) dit de quoi cette
// famille parle : la SELECTION DE ZONE DE REAPPARITION, pas la capture.
//
// # POURQUOI CETTE MESURE EXISTE QUAND MEME
//
// Doctrine du chantier : aucun negatif ne se publie sans mesure etalonnee. Le MASQUE de
// composants voyage en tete du record, AVANT toute charge : il se lit donc sans porter le
// deserialiseur et sans jamais desyncer (recette du lot 5.5.1 point (c)). Cette passe ventile
// ce que le film DECLARE de `ti=23`, index par index, sur un film a zones.
//
// Elle decode aussi, sur les records a masque SINGLETON `i0..i31`, les 32 premiers bits de la
// charge — le `zone-name` — parce que c est la seule valeur dont la largeur est certaine sans
// porter la suite : un identifiant de chaine stable par zone est la signature d une LISTE DE
// ZONES, un identifiant qui varie a chaque record serait celle d un etat.
//
// # LES LARGEURS D AXE DE LA CARTE SONT INSTALLEES (piege 5.3, paye deux fois)
//
// `NewFilmContextForMap` + `PoserLargeursObjetDuMondeDepuisDecoupage` via `avantContexteDe`.
// Elles ne servent pas au masque, elles servent a ne pas mesurer sous un profil different de
// celui de la marche de production.
//
// # CE QUE CETTE PASSE NE PEUT PAS DIRE, ET ELLE LE DIT
//
// `matchWorldObjectRecord` est un CHERCHEUR D ANCRES sur la bande des slots `ti=23` recensee aux
// images-cles : un record dont le slot n apparait a AUCUNE image-cle est invisible ici, et une
// ancre fortuite est possible (la garde est le masque strictement croissant). Le denominateur
// est donc publie a cote de chaque chiffre, avec la bande FANTOME comme temoin de hasard.
//
// # REGIME
//
//	ZONE56_FILM=<abs>/data/cache/film_chunks/396cfc92 ZONE56_CARTE=Illusion \
//	  go test -tags=research -count=1 -v -timeout 60m \
//	    -run '^TestZone56MasqueTi23$' ./internal/games/halo_infinite/film/internal/grammar/
//
// UN SEUL FILM PAR INVOCATION, aucune base DuckDB, aucun artefact ecrit.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// zone56TypeIndex est l archetype `zones (selectable-zone-data)`.
const zone56TypeIndex = 23

// zone56Index — les index dont la ventilation est commentee, parce que la question porte sur eux.
var zone56Index = map[int]string{
	0:  "i0  selectable-zone-data (zone 0 : nom + position + handle de participant)",
	1:  "i1  selectable-zone-data (zone 1)",
	2:  "i2  selectable-zone-data (zone 2)",
	32: "i32 participant-spawn-availability-mask (non porte, deser_addr VIDE)",
}

// zone56Acc accumule ce qu une bande rend.
type zone56Acc struct {
	records   int
	slots     map[uint32]bool
	parIndex  map[int]int
	singleton map[int]int
	// noms porte, par index de composant, les valeurs distinctes des 32 premiers bits de la
	// charge sur les records a masque singleton.
	noms map[int]map[uint32]int
}

func newZone56Acc() *zone56Acc {
	return &zone56Acc{
		slots: map[uint32]bool{}, parIndex: map[int]int{}, singleton: map[int]int{},
		noms: map[int]map[uint32]int{},
	}
}

func (a *zone56Acc) add(rec WorldObjectRecord, pay []byte) {
	a.records++
	a.slots[rec.Slot] = true
	for _, i := range rec.Idx {
		a.parIndex[i]++
	}
	if len(rec.Idx) != 1 {
		return
	}
	i := rec.Idx[0]
	a.singleton[i]++
	if i > 31 || rec.After+32 > len(pay)*8 {
		return
	}
	if a.noms[i] == nil {
		a.noms[i] = map[uint32]int{}
	}
	a.noms[i][uint32(PeekBits(pay, rec.After, 32))]++
}

// TestZone56MasqueTi23 — CE QUE LE FILM DECLARE DE `ti=23`.
func TestZone56MasqueTi23(t *testing.T) {
	dir, carte := os.Getenv("ZONE56_FILM"), os.Getenv("ZONE56_CARTE")
	if dir == "" || carte == "" {
		t.Skip("instrument de mesure : ZONE56_FILM et ZONE56_CARTE requis")
	}
	ti := zone56TypeIndex
	if v := os.Getenv("ZONE56_TI"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			t.Fatalf("ZONE56_TI illisible : %v", err)
		}
		ti = n // ETALON : un archetype TEMOIN (13, 40) doit rendre une bande NON VIDE
	}
	fc, _ := avantContexteDe(t, dir, carte)
	if fc == nil {
		t.Fatalf("contexte illisible")
	}
	if restore, err := InstallFilmFormatMPP(fc); err == nil {
		defer restore()
	}
	kf := ScanWorldObjectKeyframes(fc, ti)
	t.Logf("%s (carte %s) : bande ti=%d = %d slots, %d vies recensees, %d images-cles",
		filepath.Base(dir), carte, ti, len(kf.Band), len(kf.SeenUS), len(kf.TimesUS))
	zone56LogVies(t, kf)
	if len(kf.Band) == 0 {
		t.Log("AUCUN slot aux images-cles : le film ne recense pas cet archetype")
		return
	}
	fantome := zone56BandeFantome(kf.Band)
	reel := zone56Balayer(t, fc, kf.Band)
	faux := zone56Balayer(t, fc, fantome)
	t.Logf("--- VENTILATION DU MASQUE : %d records sur la bande REELLE (%d slots), "+
		"%d sur la bande FANTOME (%d slots, temoin de hasard) ---",
		reel.records, len(kf.Band), faux.records, len(fantome))
	zone56Rapport(t, reel, faux)
}

// zone56LogVies rend les vies recensees : slot, generation, nombre d images-cles qui les voient.
func zone56LogVies(t *testing.T, kf WorldObjectKeyframes) {
	t.Helper()
	if len(kf.SeenUS) == 0 {
		return
	}
	cles := make([]types.EquipmentLifeKey, 0, len(kf.SeenUS))
	for k := range kf.SeenUS {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool {
		if cles[i].Slot != cles[j].Slot {
			return cles[i].Slot < cles[j].Slot
		}
		return cles[i].Gen < cles[j].Gen
	})
	for _, k := range cles {
		v := kf.SeenUS[k]
		t.Logf("  vie slot=%d gen=%d : %d images-cles, de %.1f s a %.1f s",
			k.Slot, k.Gen, len(v), float64(v[0])/1e6, float64(v[len(v)-1])/1e6)
	}
}

// zone56BandeFantome rend une bande de MEME TAILLE, decalee hors des slots reels : le temoin de
// hasard du chercheur d ancres (recette de `ti13_variant_test.go`).
func zone56BandeFantome(band map[uint32]bool) map[uint32]bool {
	out := map[uint32]bool{}
	for s := range band {
		for d := uint32(4096); d < 8192; d += 1 {
			c := (s + d) & 0x1fff
			if !band[c] && !out[c] {
				out[c] = true
				break
			}
		}
	}
	return out
}

// zone56Balayer marche tous les paquets delta et ventile les records ancres sur `band`.
func zone56Balayer(t *testing.T, fc *FilmContext, band map[uint32]bool) *zone56Acc {
	t.Helper()
	acc := newZone56Acc()
	if len(band) == 0 {
		return acc
	}
	film := fc.Film()
	for _, c := range FilmChunkNumbers(film) {
		data, pks, ok := FilmChunkAt(film, c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			limit := len(pay)*8 - (worldObjectHeaderBits + worldObjectIndexBits)
			for p := 0; p <= limit; p++ {
				rec, ok := matchWorldObjectRecord(pay, p, band)
				if !ok {
					continue
				}
				p = rec.After
				acc.add(rec, pay)
			}
		}
	}
	return acc
}

// zone56Rapport colle la ventilation index par index, puis les identifiants de chaine lus.
func zone56Rapport(t *testing.T, reel, faux *zone56Acc) {
	t.Helper()
	idx := make([]int, 0, len(reel.parIndex))
	for i := range reel.parIndex {
		idx = append(idx, i)
	}
	sort.Ints(idx)
	t.Log("| index | records reels | part | singletons | records fantomes | role |")
	t.Log("|---|---:|---:|---:|---:|---|")
	for _, i := range idx {
		part := 0.0
		if reel.records > 0 {
			part = 100 * float64(reel.parIndex[i]) / float64(reel.records)
		}
		t.Logf("| i%d | %d | %.1f %% | %d | %d | %s |",
			i, reel.parIndex[i], part, reel.singleton[i], faux.parIndex[i], zone56Role(i))
	}
	zone56RapportNoms(t, reel)
}

func zone56Role(i int) string {
	if r, ok := zone56Index[i]; ok {
		return r
	}
	if i <= 31 {
		return fmt.Sprintf("i%d selectable-zone-data (zone %d)", i, i)
	}
	return fmt.Sprintf("i%d hors des 33 composants declares", i)
}

// zone56RapportNoms colle, par index, les identifiants de chaine distincts vus en tete de charge.
func zone56RapportNoms(t *testing.T, a *zone56Acc) {
	t.Helper()
	if len(a.noms) == 0 {
		t.Log("aucun record a masque SINGLETON : aucune charge n est localisable sans porter la grammaire")
		return
	}
	idx := make([]int, 0, len(a.noms))
	for i := range a.noms {
		idx = append(idx, i)
	}
	sort.Ints(idx)
	t.Log("--- `zone-name` (R(32)) SUR LES RECORDS SINGLETON : valeurs distinctes par index ---")
	for _, i := range idx {
		vals := make([]uint32, 0, len(a.noms[i]))
		for v := range a.noms[i] {
			vals = append(vals, v)
		}
		sort.Slice(vals, func(x, y int) bool { return a.noms[i][vals[x]] > a.noms[i][vals[y]] })
		n := len(vals)
		if n > 5 {
			n = 5
		}
		var s string
		for _, v := range vals[:n] {
			s += fmt.Sprintf(" 0x%08x(%d)", v, a.noms[i][v])
		}
		t.Logf("  i%d : %d valeurs distinctes sur %d records —%s",
			i, len(vals), a.singleton[i], s)
	}
}

// ---------------------------------------------------------------------------------------------
// LOT 5.6, POINT 2 bis : LE NOM DE CHAQUE CANAL ti=13
// ---------------------------------------------------------------------------------------------
//
// La correlation du point 2 (paquet `replay`) designe, par zone, DEUX canaux `tag 4` a valeurs
// de camp : l un est le PROPRIETAIRE, l autre vaut, PENDANT une rampe, le camp qui la remporte.
// Un port qui les distinguerait par ARITHMETIQUE DE SLOT serait une coincidence figee ; le jeu,
// lui, les distingue par leur NOM.
//
// `ti=13 i0 managed-object-property-name-component` est un R(32) DEJA PORTE, marche a chaque
// record et jete (`zone_state_scan.go` : « personne ne le consomme »). Cette passe le recolte
// par le hook de sonde, et rend le nom dominant PAR SLOT.
//
//	ZONE56_FILM=<abs>/data/cache/film_chunks/396cfc92 ZONE56_CARTE=Illusion \
//	  go test -tags=research -count=1 -v -run '^TestZone56NomsTi13$' ./...grammar/

// TestZone56NomsTi13 — QUEL NOM PORTE CHAQUE SLOT.
func TestZone56NomsTi13(t *testing.T) {
	dir, carte := os.Getenv("ZONE56_FILM"), os.Getenv("ZONE56_CARTE")
	if dir == "" || carte == "" {
		t.Skip("instrument de mesure : ZONE56_FILM et ZONE56_CARTE requis")
	}
	fc, _ := avantContexteDe(t, dir, carte)
	if fc == nil {
		t.Fatalf("contexte illisible")
	}
	if restore, err := InstallFilmFormatMPP(fc); err == nil {
		defer restore()
	}
	arch, err := fc.managedPropertyArchetype()
	if err != nil {
		t.Fatalf("archetype ti=13 : %v", err)
	}
	noms, tags := zone56RecolterNoms(fc, arch)
	slots := make([]uint32, 0, len(noms))
	for s := range noms {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	t.Logf("--- NOM DE PROPRIETE (ti=13 i0, R(32)) PAR SLOT — %d slots ---", len(slots))
	t.Log("| slot | nom dominant | emissions | autres noms | tags vus |")
	t.Log("|---:|---|---:|---:|---|")
	for _, s := range slots {
		dom, n := uint32(0), 0
		for v, c := range noms[s] {
			if c > n {
				dom, n = v, c
			}
		}
		t.Logf("| %d | 0x%08x | %d | %d | %s |",
			s, dom, n, len(noms[s])-1, zone56TagsVus(tags[s]))
	}
}

// zone56RecolterNoms marche les records ti=13 avec les desers DE PRODUCTION et recolte, par
// slot, les valeurs de `i0` (hook de sonde) et les tags du variant (hook de propriete).
func zone56RecolterNoms(fc *FilmContext, arch Archetype) (
	map[uint32]map[uint32]int, map[uint32]map[int]int,
) {
	noms := map[uint32]map[uint32]int{}
	tags := map[uint32]map[int]int{}
	var slot uint32
	obs := NouvelleObservation()
	obs.ProbeHook = func(_ uint32, comp ProbeComponent, values []uint64) {
		if comp != ProbeManagedObjectPropertyName || len(values) == 0 {
			return
		}
		if noms[slot] == nil {
			noms[slot] = map[uint32]int{}
		}
		noms[slot][uint32(values[0])]++
	}
	obs.ManagedPropertyHook = func(_ ManagedPropertyField, values []uint64) {
		if len(values) == 0 {
			return
		}
		if tags[slot] == nil {
			tags[slot] = map[int]int{}
		}
		tags[slot][int(values[0])]++
	}
	prof := fc.ProfilDeBalayage()
	band := observedSlotBand(fc, ManagedPropertyTypeIndex)
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			limit := len(pay)*8 - (worldObjectHeaderBits + worldObjectIndexBits)
			for p := 0; p <= limit; p++ {
				rec, ok := matchWorldObjectRecord(pay, p, band)
				if !ok {
					continue
				}
				slot = rec.Slot
				zone56MarcherRecord(pay, rec, arch, prof, obs)
				p = rec.After
			}
		}
	}
	return noms, tags
}

// zone56MarcherRecord marche les composants du masque d un record, et s arrete au premier non
// porte — meme regle que la marche de production.
func zone56MarcherRecord(pay []byte, rec WorldObjectRecord, arch Archetype,
	prof ProfilDeBalayage, obs *Observation,
) {
	total := len(pay) * 8
	at := rec.After
	for _, id := range rec.Idx {
		name := arch.component(id)
		if name == "" || at > total {
			return
		}
		br := LecteurSur(pay)
		br.PoserContexte(ContexteDeLecture{Profil: prof, Obs: obs})
		br.SetBitPos(at)
		_, _, ported := consumeByName(br, name, ManagedPropertyTypeIndex, arch.Level(id))
		if !ported || br.BitPos() > total {
			return
		}
		at = br.BitPos()
	}
}

// zone56TagsVus colle les tags d un slot, les plus frequents d abord.
func zone56TagsVus(m map[int]int) string {
	if len(m) == 0 {
		return "-"
	}
	ks := make([]int, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(i, j int) bool { return m[ks[i]] > m[ks[j]] })
	var s string
	for _, k := range ks {
		s += fmt.Sprintf(" %d(x%d)", k, m[k])
	}
	return s
}
