package filmdec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// statborg_rounds_diag_test.go — DIAGNOSTIC de l'item A.0b.1 : ou passent les enregistrements
// statborg apres la fin de la premiere manche ?
//
// # Le fait a expliquer
//
// Sur `24dbb67d` (Oddball, 519 s), les emissions du score de mode s'ARRETENT a 290 683 ms et le
// film rend 100/78 quand l'API rend 200/121. 100/78 est la manche 1 ; la manche 2 est invisible a
// la grammaire d'ancrage (`objectiveevents.StatRecords`). Trois causes possibles, qui ne se
// distinguent que par la mesure :
//
//	(a) l'entite d'equipe est RECREEE entre les manches — slot ou GENERATION differents, et la
//	    contrainte d'en-tete de l'ancrage (2 bits constants apres le slot) la rejette ;
//	(b) les entites emettent apres 290 s, mais des composants que l'ancrage rejette — typiquement
//	    `statborg-finalized-rounds-values-stat-component` (i28..i55), dont la grammaire (R(32)
//	    masque + par bit 2 x {R(1)[si 0 : varW]}) n'est PAS celle d'i0..i27 ;
//	(c) elles n'emettent plus du tout, et le film ne porte pas la manche 2.
//
// Ce test ne conclut pas : il ecrit ce que les images-cles et la chaine voient, avant et apres la
// bascule. La grammaire d'ancrage etendue vit dans l'instrument du lot A (paquet objectiveevents).
//
// Gardes : `ROUNDS_FILM` (dossier de chunks), `ROUNDS_SPLIT_MS` (instant de bascule, defaut
// 290683), `ROUNDS_OUT` (TSV de sortie, facultatif). Un film par processus (D17).

const (
	roundsFilmEnv  = "ROUNDS_FILM"
	roundsSplitEnv = "ROUNDS_SPLIT_MS"
	roundsOutEnv   = "ROUNDS_OUT"
	// roundsDefaultSplitMS est l'instant de la derniere emission de score mesuree sur
	// `24dbb67d` en phase 0.
	roundsDefaultSplitMS = 290683
)

// TestStatborgRoundsDiag ecrit ce que voient les images-cles et la chaine, avant/apres la bascule.
func TestStatborgRoundsDiag(t *testing.T) {
	dir := os.Getenv(roundsFilmEnv)
	if dir == "" {
		t.Skipf("diagnostic des manches non arme (%s vide)", roundsFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	split := roundsDefaultSplitMS
	if v := os.Getenv(roundsSplitEnv); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &split); err != nil {
			t.Fatalf("%s=%q illisible : %v", roundsSplitEnv, v, err)
		}
	}
	n := CountFilmChunks(dir)
	if n == 0 {
		t.Fatalf("aucun chunk film dans %s", dir)
	}
	reg := registryOf(t, dir)
	ti := archetypeWithComponent(reg, statborgComponentName)
	if ti < 0 {
		t.Fatalf("aucun archetype ne declare %q", statborgComponentName)
	}
	starts := manifestStarts(t, dir)

	d := &roundsDiag{reg: reg, ti: uint32(ti), split: split, world: NewWorld(reg),
		comps: map[string]int{}, slotsBefore: map[string]int{}, slotsAfter: map[string]int{}}
	for i := 1; i <= n; i++ {
		data, err := ReadFilmChunk(dir, i)
		if err != nil {
			continue
		}
		d.chunk(data, starts[i])
	}
	report := d.report(shortOf(dir), int(d.ti))
	for _, ln := range report {
		t.Logf("ROUNDS %s", ln)
	}
	if out := os.Getenv(roundsOutEnv); out != "" {
		if err := os.WriteFile(out, []byte(joinLines(report)), 0o644); err != nil {
			t.Fatalf("ecriture de %s : %v", out, err)
		}
	}
}

// roundsDiag porte l'etat du diagnostic.
type roundsDiag struct {
	reg   *Registry
	ti    uint32
	split int
	world *World
	// keyframes accumule une ligne par image-cle : les entites statborg qu'elle declare.
	keyframes []string
	// comps compte les composants annonces par les records statborg APRES la bascule.
	comps map[string]int
	// slotsBefore / slotsAfter comptent les entites statborg vues par la chaine, par
	// « slot:generation », de part et d'autre de la bascule.
	slotsBefore, slotsAfter map[string]int
	// recBefore / recAfter comptent les records statborg des paquets PROPRES.
	recBefore, recAfter int
	// kfBefore / kfAfter comptent les images-cles de part et d'autre.
	kfBefore, kfAfter int
}

// chunk marche tous les paquets d'un chunk, l'horloge etant recalee sur le start_ms du manifeste
// (meme convention que objectiveevents.StatRecords : le premier paquet du chunk porte l'origine).
func (d *roundsDiag) chunk(data []byte, startMS int) {
	base := int64(-1)
	for _, pk := range WalkPackets(data) {
		if base < 0 {
			base = int64(pk.TimestampUS)
		}
		tMS := startMS + int((int64(pk.TimestampUS)-base)/1000)
		switch pk.Type {
		case PacketTypeKeyframe:
			d.keyframe(pk.Payload(data), tMS)
		case PacketTypeDelta:
			d.delta(pk.Payload(data), tMS)
		}
	}
}

// keyframe releve les entites de l'archetype statborg presentes dans une image-cle.
func (d *roundsDiag) keyframe(pay []byte, tMS int) {
	if tMS < d.split {
		d.kfBefore++
	} else {
		d.kfAfter++
	}
	var seen []string
	for _, r := range WalkKeyframeWorld(pay) {
		if uint32(r.TI) == d.ti {
			seen = append(seen, fmt.Sprintf("slot=%d/gen=%d", r.Slot, r.Gen))
		}
	}
	sort.Strings(seen)
	d.keyframes = append(d.keyframes, fmt.Sprintf("kf\t%d\t%d\t%v", tMS, len(seen), seen))
	d.world = WorldFromKeyframe(d.reg, pay)
}

// delta marche un paquet delta et releve les records statborg des marches PROPRES.
func (d *roundsDiag) delta(pay []byte, tMS int) {
	recs, err := DecodeFrameRecords(NewBitReader(pay), d.world, DefaultFrameConfig())
	if err != nil {
		return // les records d'un paquet desynchronise sont du bruit (mesure de la phase 0)
	}
	for _, r := range recs {
		a, ok := d.world.ArchetypeForSlot(r.Slot)
		if !ok || a != d.ti {
			continue
		}
		key := fmt.Sprintf("%d/g%d", r.Slot, r.ID>>30)
		if tMS < d.split {
			d.recBefore++
			d.slotsBefore[key]++
			continue
		}
		d.recAfter++
		d.slotsAfter[key]++
		for _, c := range r.Trace.Comps {
			d.comps[fmt.Sprintf("i%d %s", c.Index, c.Name)]++
		}
	}
}

// report rend les lignes TSV du diagnostic.
func (d *roundsDiag) report(short string, ti int) []string {
	out := []string{
		"# diagnostic des manches statborg — " + short,
		fmt.Sprintf("meta\t%s\tti=%d\tsplit_ms=%d", short, ti, d.split),
		fmt.Sprintf("images_cles\tavant=%d\tapres=%d", d.kfBefore, d.kfAfter),
		fmt.Sprintf("records_propres\tavant=%d\tapres=%d", d.recBefore, d.recAfter),
		fmt.Sprintf("slots_avant\t%s", sortedCounts(d.slotsBefore)),
		fmt.Sprintf("slots_apres\t%s", sortedCounts(d.slotsAfter)),
	}
	for _, k := range sortedKeys(d.comps) {
		out = append(out, fmt.Sprintf("comp_apres\t%s\t%d", k, d.comps[k]))
	}
	return append(out, d.keyframes...)
}

// sortedCounts rend les couples cle=n tries, pour une sortie stable.
func sortedCounts(m map[string]int) string {
	s := ""
	for _, k := range sortedKeys(m) {
		s += fmt.Sprintf("%s=%d ", k, m[k])
	}
	return s
}

func sortedKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func joinLines(lines []string) string {
	s := ""
	for _, ln := range lines {
		s += ln + "\n"
	}
	return s
}

// manifestStarts rend le start_ms de chaque chunk, lu dans le manifeste du cache film. Le
// manifeste est le frere de `film_chunks` : `<racine>/film_manifests/<short8>.json`.
func manifestStarts(t *testing.T, dir string) map[int]int {
	t.Helper()
	root := filepath.Dir(filepath.Dir(filepath.Clean(dir)))
	path := filepath.Join(root, "film_manifests", shortOf(dir)+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("manifeste illisible (%s) : %v", path, err)
	}
	var mf struct {
		Chunks []struct {
			Index   int `json:"index"`
			StartMS int `json:"start_ms"`
		} `json:"chunks"`
	}
	if err := json.Unmarshal(raw, &mf); err != nil {
		t.Fatalf("manifeste invalide (%s) : %v", path, err)
	}
	out := map[int]int{}
	for _, c := range mf.Chunks {
		out[c.Index] = c.StartMS
	}
	return out
}

// # Lire la VALEUR d'un compteur sans toucher au deserialiseur
//
// `consumeStatborgValueStat` jette ce qu'il lit, et la regle de ce lot interdit de modifier
// `traverse.go`. Mais la traversee publie `CompResult.StartBit` : la position de bit EXACTE de
// chaque composant du record. La valeur se relit donc a cette position avec la grammaire du
// compteur, depuis le test, sans poser aucun hook.
//
// Grammaire (FUN_140C18794) : deux en-tetes de 5 bits, puis deux entiers signes a largeur
// variable. Les deux en-tetes valent 0 sur 99,1 % des lectures reelles : on s'en sert comme
// controle de vraisemblance et on jette la lecture sinon.
func statValueAt(pay []byte, startBit int) (int64, bool) {
	if startBit < 0 || startBit+2*statHdrWidth+4 > len(pay)*8 {
		return 0, false
	}
	br := NewBitReader(pay)
	br.Skip(startBit)
	// Les deux en-tetes sont PUBLIES au lieu d etre exiges nuls : c est la mesure qui doit dire
	// si la chaine et l ancrage cadrent le composant au meme bit.
	h1 := int(br.ReadBits(statHdrWidth))
	h2 := int(br.ReadBits(statHdrWidth))
	v := int64(br.ReadSignedVarWidth())
	return v, h1 == 0 && h2 == 0
}

// statHdrWidth est la largeur de chacun des deux en-tetes d'un compteur statborg.
const statHdrWidth = 5

// TestStatborgRoundsValues suit la VALEUR du compteur i0 (le score de mode) par slot, telle que
// la CHAINE la positionne — la seule voie qui reste quand l'ancrage ne trouve plus rien apres la
// premiere fin de manche.
//
// Memes gardes que [TestStatborgRoundsDiag], plus `ROUNDS_COMP` (index du compteur, defaut 0).
func TestStatborgRoundsValues(t *testing.T) {
	dir := os.Getenv(roundsFilmEnv)
	if dir == "" {
		t.Skipf("suivi des valeurs non arme (%s vide)", roundsFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	comp := 0
	if v := os.Getenv("ROUNDS_COMP"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &comp); err != nil {
			t.Fatalf("ROUNDS_COMP=%q illisible : %v", v, err)
		}
	}
	n := CountFilmChunks(dir)
	reg := registryOf(t, dir)
	ti := archetypeWithComponent(reg, statborgComponentName)
	if ti < 0 {
		t.Fatalf("aucun archetype ne declare %q", statborgComponentName)
	}
	w := NewWorld(reg)
	starts := manifestStarts(t, dir)
	var lines []string
	for i := 1; i <= n; i++ {
		data, err := ReadFilmChunk(dir, i)
		if err != nil {
			continue
		}
		base := int64(-1)
		for _, pk := range WalkPackets(data) {
			if base < 0 {
				base = int64(pk.TimestampUS)
			}
			tMS := starts[i] + int((int64(pk.TimestampUS)-base)/1000)
			pay := pk.Payload(data)
			if pk.Type == PacketTypeKeyframe {
				w = WorldFromKeyframe(reg, pay)
				continue
			}
			if pk.Type != PacketTypeDelta {
				continue
			}
			lines = append(lines, valueLines(pay, w, uint32(ti), comp, tMS)...)
		}
	}
	t.Logf("VALS lignes=%d comp=%d", len(lines), comp)
	if out := os.Getenv(roundsOutEnv); out != "" {
		if err := os.WriteFile(out, []byte(joinLines(lines)), 0o644); err != nil {
			t.Fatalf("ecriture de %s : %v", out, err)
		}
	}
}

// valueLines rend une ligne par valeur de compteur relue dans un paquet delta PROPRE.
func valueLines(pay []byte, w *World, ti uint32, comp, tMS int) []string {
	recs, err := DecodeFrameRecords(NewBitReader(pay), w, DefaultFrameConfig())
	if err != nil {
		return nil // paquet desynchronise : ses records sont du bruit
	}
	var out []string
	for _, r := range recs {
		a, ok := w.ArchetypeForSlot(r.Slot)
		if !ok || a != ti {
			continue
		}
		for _, c := range r.Trace.Comps {
			if c.Index != comp {
				continue
			}
			v, clean := statValueAt(pay, c.StartBit)
			out = append(out, fmt.Sprintf("val\t%d\t%d\t%d\t%d\t%v",
				tMS, r.Slot, r.ID>>30, v, clean))
		}
	}
	return out
}
