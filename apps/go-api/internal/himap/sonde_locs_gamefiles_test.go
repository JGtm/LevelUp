package himap

// SONDE (2026-08-27) — LE TAG `locs` (location_name_globals_definition) EST-IL LE
// VOCABULAIRE QUE LES .mvar REFERENCENT ?
//
// Contexte : notre catalogue de zones nommees ne couvre que les 22 cartes natives, dont la
// geometrie vient du tag `levl` du .module. L'utilisateur affirme que le jeu affiche des
// callouts sur TOUTES les cartes, Forge comprises. Deux briques a verifier :
//
//  1. le tag `locs` de globals-rtx-new.module porte-t-il un bloc de StringId, et lesquels ?
//  2. ces StringId sont-ils ceux de callouts_i18n.csv (donc bien le meme vocabulaire) ?
//
// La sonde ne conclut pas : elle mesure et journalise. Aucun octet n'est ecrit.

import (
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/replay/mapvar"
	"levelup/go-api/internal/himodule"
)

func moduleGlobals(t *testing.T) string {
	t.Helper()
	racine, err := DeployRoot()
	if err != nil {
		t.Skipf("pas d installation : %v", err)
	}
	p := filepath.Join(racine, "any", "globals", "globals-rtx-new.module")
	if _, err := os.Stat(p); err != nil {
		t.Skipf("globals absent : %v", err)
	}
	return p
}

// TestSondeTagLocsStructure — combien de tags `locs`, quelle taille, quels StringId ?
func TestSondeTagLocsStructure(t *testing.T) {
	p := moduleGlobals(t)
	m, err := himodule.Open(p)
	if err != nil {
		t.Fatalf("ouvrir %s : %v", p, err)
	}
	fichiers := m.Files("locs")
	t.Logf("%d tag(s) locs dans %s", len(fichiers), filepath.Base(p))
	if len(fichiers) == 0 {
		t.Skip("aucun locs")
	}
	brut, err := m.Extract(fichiers[0])
	if err != nil {
		t.Fatalf("extraire locs : %v", err)
	}
	t.Logf("taille du tag locs : %d octets", len(brut))

	// Balayage brut : tous les u32 du tag, pour voir la densite de StringId plausibles.
	// On compare directement au vocabulaire des callouts natifs.
	vocab := chargeVocabCallouts(t)
	t.Logf("vocabulaire callouts natifs : %d string_id distincts", len(vocab))

	trouves := map[uint32]bool{}
	for off := 0; off+4 <= len(brut); off++ {
		v := binary.LittleEndian.Uint32(brut[off:])
		if vocab[v] {
			if !trouves[v] {
				t.Logf("  locs+0x%X : string_id 0x%08X (= %s)", off, v, vocabNom(vocab, v))
			}
			trouves[v] = true
		}
	}
	t.Logf("=> %d/%d string_id de callouts natifs presents dans le tag locs", len(trouves), len(vocab))
}

// vocabCallouts porte les string_id et un nom de conception par id.
var vocabNoms = map[uint32]string{}

func chargeVocabCallouts(t *testing.T) map[uint32]bool {
	t.Helper()
	p := filepath.Join("..", "..", "..", "..", "data", "titles", "halo_infinite", "reference", "callouts_i18n.csv")
	f, err := os.Open(p)
	if err != nil {
		t.Skipf("catalogue i18n absent : %v", err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comma = ';'
	r.FieldsPerRecord = -1
	lignes, err := r.ReadAll()
	if err != nil {
		t.Fatalf("csv : %v", err)
	}
	out := map[uint32]bool{}
	for i, l := range lignes {
		if i == 0 || len(l) < 5 {
			continue
		}
		s := strings.TrimPrefix(strings.TrimSpace(l[2]), "0x")
		v, err := strconv.ParseUint(s, 16, 32)
		if err != nil {
			continue
		}
		out[uint32(v)] = true
		vocabNoms[uint32(v)] = l[3] + " / " + l[4]
	}
	return out
}

func vocabNom(_ map[uint32]bool, v uint32) string {
	if n, ok := vocabNoms[v]; ok {
		return n
	}
	return "?"
}

// TestCalloutStringIDEstUnMurmur — le string_id d'une zone nommee est-il le murmur3 de son
// nom de conception (avec ou sans le prefixe « named location ») ?
func TestCalloutStringIDEstUnMurmur(t *testing.T) {
	p := filepath.Join("..", "..", "..", "..", "data", "titles", "halo_infinite", "reference", "callouts_i18n.csv")
	f, err := os.Open(p)
	if err != nil {
		t.Skipf("catalogue absent : %v", err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comma = ';'
	r.FieldsPerRecord = -1
	lignes, _ := r.ReadAll()
	var nOK, nOKBrut, nTotal int
	var exemples []string
	for i, l := range lignes {
		if i == 0 || len(l) < 9 {
			continue
		}
		s := strings.TrimPrefix(strings.TrimSpace(l[2]), "0x")
		v, err := strconv.ParseUint(s, 16, 32)
		if err != nil {
			continue
		}
		nTotal++
		conception := strings.TrimSpace(l[3])
		brut := strings.TrimSpace(l[8])
		if hachageChaine(conception) == uint32(v) {
			nOK++
			if len(exemples) < 3 {
				exemples = append(exemples, fmt.Sprintf("%q -> 0x%08X", conception, v))
			}
		}
		if hachageChaine(brut) == uint32(v) {
			nOKBrut++
		}
		if hachageChaine(strings.ReplaceAll(conception, " ", "_")) == uint32(v) {
			nOK += 0
		}
	}
	t.Logf("callouts : %d lignes ; murmur3(nom_conception) colle %d fois ; murmur3(nom_brut) colle %d fois",
		nTotal, nOK, nOKBrut)
	for _, e := range exemples {
		t.Logf("  %s", e)
	}
}

// TestStringIDsDuMvarContreVocabulaireCallouts — LE CROISEMENT DEMANDE.
// On collecte TOUS les mots de 32 bits du .mvar (au niveau octet, sans grammaire, pour ne
// rien rater), puis on les confronte au vocabulaire des zones nommees natives.
func TestStringIDsDuMvarContreVocabulaireCallouts(t *testing.T) {
	vocab := chargeVocabCallouts(t)
	dossier := filepath.Join("C:\\", "Users", "Guillaume", "Projects", "LevelUp", ".ai", "re_dump", "mapvar")
	for _, nom := range []string{"isolation_map.mvar", "aquarius_map.mvar", "streets_map.mvar", "cliffhanger_map.mvar", "recharge_map.mvar", "live_fire_map.mvar"} {
		brut, err := os.ReadFile(filepath.Join(dossier, nom))
		if err != nil {
			continue
		}
		trouves := map[uint32]int{}
		for off := 0; off+4 <= len(brut); off++ {
			v := binary.LittleEndian.Uint32(brut[off:])
			if vocab[v] {
				trouves[v]++
			}
		}
		if len(trouves) == 0 {
			t.Logf("%-24s %8d octets : AUCUN string_id de zone nommee (sur %d du vocabulaire)", nom, len(brut), len(vocab))
			continue
		}
		ids := make([]uint32, 0, len(trouves))
		for v := range trouves {
			ids = append(ids, v)
		}
		sort.Slice(ids, func(i, j int) bool { return trouves[ids[i]] > trouves[ids[j]] })
		t.Logf("%-24s %8d octets : %d string_id de zone nommee trouves", nom, len(brut), len(trouves))
		for i, v := range ids {
			if i >= 10 {
				break
			}
			t.Logf("    0x%08X x%d  (%s)", v, trouves[v], vocabNom(vocab, v))
		}
	}
}

// hachageChaine : le murmur3 canonique du depot (mapvar.LabelHash), en u32.
func hachageChaine(s string) uint32 { return uint32(mapvar.LabelHash(s)) }

// ---------------------------------------------------------------------------
// LE CROISEMENT CORRECT : les entiers du .mvar sont des VARINT ZIGZAG Bond.
// Un balayage octet a octet du fichier ne peut PAS les retrouver — il faut decoder
// l'arbre et comparer les valeurs DECODEES.
// ---------------------------------------------------------------------------

// vocabulaireLocs rend tous les mots de 32 bits du tag locs (le bloc racine du tag est
// un tableau dense de StringId : mesure du test de structure ci-dessus).
func vocabulaireLocs(t *testing.T) map[uint32]bool {
	t.Helper()
	m, err := himodule.Open(moduleGlobals(t))
	if err != nil {
		t.Fatalf("ouvrir globals : %v", err)
	}
	fichiers := m.Files("locs")
	if len(fichiers) == 0 {
		t.Skip("aucun locs")
	}
	brut, err := m.Extract(fichiers[0])
	if err != nil {
		t.Fatalf("extraire locs : %v", err)
	}
	out := map[uint32]bool{}
	for off := 0; off+4 <= len(brut); off += 4 {
		out[binary.LittleEndian.Uint32(brut[off:])] = true
	}
	t.Logf("tag locs : %d octets, %d mots de 32 bits distincts (vocabulaire brut, bruit compris)", len(brut), len(out))
	return out
}

// TestStructureTagLocs — navigation de la struct-table : combien de blocs, quels comptes.
func TestStructureTagLocs(t *testing.T) {
	m, err := himodule.Open(moduleGlobals(t))
	if err != nil {
		t.Fatalf("ouvrir globals : %v", err)
	}
	fichiers := m.Files("locs")
	if len(fichiers) == 0 {
		t.Skip("aucun locs")
	}
	brut, err := m.Extract(fichiers[0])
	if err != nil {
		t.Fatalf("extraire : %v", err)
	}
	ti, err := meilleurTagInfo(brut)
	if err != nil {
		t.Logf("struct-table illisible (%v) — on reste sur le balayage brut", err)
		return
	}
	t.Logf("locs : %d octets, headerSize=%d deps=%d dataBlocks=%d structs=%d",
		len(brut), ti.headerSize, ti.deps, ti.dataBlocks, ti.structs)
	ri, err := ti.rootBlockIndex()
	if err != nil {
		t.Logf("root block : %v", err)
		return
	}
	abs, size := ti.blockAbs(ri)
	t.Logf("root block #%d : abs=0x%X taille=%d", ri, abs, size)
	for _, l := range liensBlocs(ti) {
		t.Logf("  bloc %d -> %d @ owner+0x%X : count=%d", l.owner, l.target, l.fieldOff, compteChamp(ti, l))
	}
}

// TestEntiersDuMvarContreVocabulaireLocs — LE TEST DECISIF.
// Tous les entiers DECODES de l'arbre Bond d'une variante, confrontes au vocabulaire locs.
func TestEntiersDuMvarContreVocabulaireLocs(t *testing.T) {
	vocab := vocabulaireLocs(t)
	vocabNatif := chargeVocabCallouts(t)
	dossier := filepath.Join("C:\\", "Users", "Guillaume", "Projects", "LevelUp", ".ai", "re_dump", "mapvar")
	cartes := []string{
		"isolation_map.mvar", "isolation_fo08_wetland.mvar",
		"aquarius_map.mvar", "streets_map.mvar", "cliffhanger_map.mvar",
		"recharge_map.mvar", "live_fire_map.mvar", "bazaar_map.mvar",
		"argyle_map.mvar", "smallhalla_map.mvar",
	}
	for _, nom := range cartes {
		brut, err := os.ReadFile(filepath.Join(dossier, nom))
		if err != nil {
			continue
		}
		root, err := mapvar.DecodeRoot(brut)
		if err != nil {
			t.Logf("%-32s ILLISIBLE : %v", nom, err)
			continue
		}
		vals := map[uint32]int{}
		collecteEntiers(root, vals)
		nVocab, nNatif := 0, 0
		var exemples []string
		for v, n := range vals {
			if vocab[v] {
				nVocab++
				if len(exemples) < 8 {
					exemples = append(exemples, fmt.Sprintf("0x%08X x%d", v, n))
				}
			}
			if vocabNatif[v] {
				nNatif++
			}
		}
		t.Logf("%-32s %6d entiers distincts ; %d dans le vocabulaire locs ; %d dans les zones nommees natives",
			nom, len(vals), nVocab, nNatif)
		for _, e := range exemples {
			t.Logf("      %s", e)
		}
	}
}

func collecteEntiers(v mapvar.Value, out map[uint32]int) {
	switch {
	case v.Fields != nil:
		for _, f := range v.Fields {
			collecteEntiers(f, out)
		}
	case v.Items != nil:
		for _, it := range v.Items {
			collecteEntiers(it, out)
		}
	case v.Pairs != nil:
		for _, kv := range v.Pairs {
			collecteEntiers(kv.Key, out)
			collecteEntiers(kv.Val, out)
		}
	}
	if v.Int != 0 {
		out[uint32(v.Int)]++
	}
	if v.Uint != 0 {
		out[uint32(v.Uint)]++
	}
}

// ---------------------------------------------------------------------------
// LE VOCABULAIRE locs EXACT (778 entrees, pas le balayage brut) ET LA LOCALISATION
// DES OCCURRENCES DANS L'ARBRE DE LA VARIANTE.
// ---------------------------------------------------------------------------

// VocabulaireLocsExact lit le bloc de 778 StringId du tag locs via la struct-table.
func vocabulaireLocsExact(t *testing.T) (map[uint32]bool, int) {
	t.Helper()
	m, err := himodule.Open(moduleGlobals(t))
	if err != nil {
		t.Fatalf("ouvrir globals : %v", err)
	}
	fichiers := m.Files("locs")
	if len(fichiers) == 0 {
		t.Skip("aucun locs")
	}
	brut, err := m.Extract(fichiers[0])
	if err != nil {
		t.Fatalf("extraire : %v", err)
	}
	ti, err := meilleurTagInfo(brut)
	if err != nil {
		t.Fatalf("struct-table : %v", err)
	}
	liens := liensBlocs(ti)
	if len(liens) == 0 {
		t.Fatal("aucun TagBlock")
	}
	l := liens[0]
	n := compteChamp(ti, l)
	abs, size := ti.blockAbs(l.target)
	stride := 0
	if n > 0 {
		stride = size / n
	}
	t.Logf("bloc des noms de lieu : abs=0x%X taille=%d n=%d stride=%d", abs, size, n, stride)
	out := map[uint32]bool{}
	for i := 0; i < n && abs+i*stride+4 <= len(brut); i++ {
		out[binary.LittleEndian.Uint32(brut[abs+i*stride:])] = true
	}
	return out, n
}

// TestVocabulaireLocsExact — combien d'entrees, combien couvrent les callouts natifs.
func TestVocabulaireLocsExact(t *testing.T) {
	vocab, n := vocabulaireLocsExact(t)
	natif := chargeVocabCallouts(t)
	commun := 0
	var manquants []string
	for v := range natif {
		if vocab[v] {
			commun++
		} else if len(manquants) < 10 {
			manquants = append(manquants, fmt.Sprintf("0x%08X (%s)", v, vocabNom(natif, v)))
		}
	}
	t.Logf("locs : %d entrees declarees, %d StringId distincts lus", n, len(vocab))
	t.Logf("zones nommees natives : %d string_id distincts, %d couverts par locs", len(natif), commun)
	for _, m := range manquants {
		t.Logf("  absent de locs : %s", m)
	}
}

// TestOuSontLesStringIdDeLieuDansLaVariante — pour chaque valeur du vocabulaire locs
// rencontree dans l'arbre, on rend le CHEMIN de champ exact.
func TestOuSontLesStringIdDeLieuDansLaVariante(t *testing.T) {
	vocab, _ := vocabulaireLocsExact(t)
	natif := chargeVocabCallouts(t)
	dossier := filepath.Join("C:\\", "Users", "Guillaume", "Projects", "LevelUp", ".ai", "re_dump", "mapvar")
	entrees, err := os.ReadDir(dossier)
	if err != nil {
		t.Skipf("dump absent : %v", err)
	}
	totalCartes, cartesAvecOccurrence, totalOccurrences := 0, 0, 0
	parChemin := map[string]int{}
	for _, e := range entrees {
		nom := e.Name()
		if !strings.HasSuffix(nom, ".mvar") {
			continue
		}
		brut, err := os.ReadFile(filepath.Join(dossier, nom))
		if err != nil {
			continue
		}
		root, err := mapvar.DecodeRoot(brut)
		if err != nil {
			continue
		}
		totalCartes++
		trouves := map[string][]uint32{}
		cherche("root", root, vocab, trouves)
		if len(trouves) == 0 {
			continue
		}
		cartesAvecOccurrence++
		for chemin, vals := range trouves {
			parChemin[chemin] += len(vals)
			totalOccurrences += len(vals)
		}
		if cartesAvecOccurrence <= 12 {
			t.Logf("--- %s ---", nom)
			chemins := make([]string, 0, len(trouves))
			for c := range trouves {
				chemins = append(chemins, c)
			}
			sort.Strings(chemins)
			for _, c := range chemins {
				var b []string
				for i, v := range trouves[c] {
					if i >= 6 {
						break
					}
					etiquette := ""
					if natif[v] {
						etiquette = " = " + vocabNom(natif, v)
					}
					b = append(b, fmt.Sprintf("0x%08X%s", v, etiquette))
				}
				t.Logf("   %-24s x%d : %s", c, len(trouves[c]), strings.Join(b, ", "))
			}
		}
	}
	t.Logf("=== %d variantes balayees, %d portent au moins une valeur du vocabulaire locs, %d occurrences ===",
		totalCartes, cartesAvecOccurrence, totalOccurrences)
	chemins := make([]string, 0, len(parChemin))
	for c := range parChemin {
		chemins = append(chemins, c)
	}
	sort.Slice(chemins, func(i, j int) bool { return parChemin[chemins[i]] > parChemin[chemins[j]] })
	for _, c := range chemins {
		t.Logf("  %-24s %d occurrences", c, parChemin[c])
	}
}

// cherche parcourt l'arbre et note, par chemin de champ (listes agregees par []),
// les valeurs entieres appartenant au vocabulaire. Les petits entiers (< 0x10000) sont
// ecartes : ce sont des comptes et des index, pas des StringId, et le vocabulaire en
// contient quelques-uns par accident de balayage.
func cherche(chemin string, v mapvar.Value, vocab map[uint32]bool, out map[string][]uint32) {
	switch {
	case v.Fields != nil:
		for id, f := range v.Fields {
			cherche(fmt.Sprintf("%s/%d", chemin, id), f, vocab, out)
		}
	case v.Items != nil:
		for _, it := range v.Items {
			cherche(chemin+"[]", it, vocab, out)
		}
	case v.Pairs != nil:
		for _, kv := range v.Pairs {
			cherche(chemin+"{k}", kv.Key, vocab, out)
			cherche(chemin+"{v}", kv.Val, vocab, out)
		}
	}
	for _, x := range []uint32{uint32(v.Int), uint32(v.Uint)} {
		if x >= 0x10000 && vocab[x] {
			out[chemin] = append(out[chemin], x)
		}
	}
}

// ---------------------------------------------------------------------------
// CE QUE SONT LES OBJETS PORTEURS DE #8/4 : LES ZONES NOMMEES DE LA VARIANTE.
// ---------------------------------------------------------------------------

type zoneVariante struct {
	Index    int
	TypeID   int32
	StringID uint32
	Pos      [3]float64
	Forme    string
	Labels   []int32
}

// zonesNommeesDeVariante extrait les objets qui portent un StringId de lieu au chemin
// #8/4[]/0/0 (mesure : 4151 occurrences sur 100 des 257 variantes du dump).
func zonesNommeesDeVariante(root mapvar.Value) []zoneVariante {
	objs, ok := root.Field(3)
	if !ok {
		return nil
	}
	var out []zoneVariante
	for i, o := range objs.Items {
		bag, ok := o.Field(8)
		if !ok {
			continue
		}
		lst, ok := bag.Field(4)
		if !ok || len(lst.Items) == 0 {
			continue
		}
		st, ok := lst.Items[0].Field(0)
		if !ok {
			continue
		}
		sid, ok := st.Field(0)
		if !ok {
			continue
		}
		z := zoneVariante{Index: i, StringID: uint32(sid.Int)}
		if t, ok := o.Field(2); ok {
			if id, ok := t.Field(0); ok {
				z.TypeID = int32(id.Int)
			}
		}
		if p, ok := o.Field(3); ok {
			for k := uint16(0); k < 3; k++ {
				if c, ok := p.Field(k); ok {
					z.Pos[k] = c.Float
				}
			}
		}
		z.Forme = formeDeObjet(bag)
		z.Labels = labelsDeObjet(bag)
		out = append(out, z)
	}
	return out
}

func formeDeObjet(bag mapvar.Value) string {
	gp, ok := bag.Field(0)
	if !ok || len(gp.Items) == 0 {
		return ""
	}
	f, ok := gp.Items[0].Field(0)
	if !ok || len(f.Items) == 0 {
		return ""
	}
	s := f.Items[0]
	fam := int64(-1)
	if v, ok := s.Field(0); ok {
		fam = v.Int
	}
	dim := func(id uint16) float64 {
		v, ok := s.Field(id)
		if !ok {
			return 0
		}
		w, ok := v.Field(0)
		if !ok {
			return 0
		}
		return float64(w.Int) / 65536.0
	}
	nom := map[int64]string{2: "cylindre", 3: "boite"}[fam]
	if nom == "" {
		nom = fmt.Sprintf("fam%d", fam)
	}
	return fmt.Sprintf("%s %.1fx%.1f h%.1f/%.1f", nom, dim(5), dim(6), dim(7), dim(8))
}

func labelsDeObjet(bag mapvar.Value) []int32 {
	gp, ok := bag.Field(0)
	if !ok || len(gp.Items) == 0 {
		return nil
	}
	l, ok := gp.Items[0].Field(9)
	if !ok {
		return nil
	}
	var out []int32
	for _, e := range l.Items {
		if h, ok := e.Field(0); ok {
			out = append(out, int32(h.Int))
		}
	}
	return out
}

// TestZonesNommeesDesVariantes — le recensement complet sur le dump.
func TestZonesNommeesDesVariantes(t *testing.T) {
	vocab, _ := vocabulaireLocsExact(t)
	natif := chargeVocabCallouts(t)
	dossier := filepath.Join("C:\\", "Users", "Guillaume", "Projects", "LevelUp", ".ai", "re_dump", "mapvar")
	entrees, err := os.ReadDir(dossier)
	if err != nil {
		t.Skipf("dump absent : %v", err)
	}
	typesPorteurs := map[int32]int{}
	formes := map[string]int{}
	nCartes, nCartesAvec, nZones, nDansVocab := 0, 0, 0, 0
	type ligne struct {
		nom string
		n   int
	}
	var lignes []ligne
	for _, e := range entrees {
		if !strings.HasSuffix(e.Name(), ".mvar") {
			continue
		}
		brut, err := os.ReadFile(filepath.Join(dossier, e.Name()))
		if err != nil {
			continue
		}
		root, err := mapvar.DecodeRoot(brut)
		if err != nil {
			continue
		}
		nCartes++
		zs := zonesNommeesDeVariante(root)
		if len(zs) == 0 {
			continue
		}
		nCartesAvec++
		nZones += len(zs)
		lignes = append(lignes, ligne{e.Name(), len(zs)})
		for _, z := range zs {
			typesPorteurs[z.TypeID]++
			if vocab[z.StringID] {
				nDansVocab++
			}
			f := strings.SplitN(z.Forme, " ", 2)[0]
			formes[f]++
		}
	}
	t.Logf("=== %d variantes, %d portent des zones nommees, %d zones au total, %d dont le StringId est dans locs ===",
		nCartes, nCartesAvec, nZones, nDansVocab)
	t.Logf("familles de forme : %v", formes)
	tids := make([]int32, 0, len(typesPorteurs))
	for k := range typesPorteurs {
		tids = append(tids, k)
	}
	sort.Slice(tids, func(i, j int) bool { return typesPorteurs[tids[i]] > typesPorteurs[tids[j]] })
	for _, k := range tids {
		t.Logf("  type_id %d : %d zones", k, typesPorteurs[k])
	}
	sort.Slice(lignes, func(i, j int) bool { return lignes[i].n > lignes[j].n })
	for i, l := range lignes {
		if i >= 25 {
			t.Logf("  ... (%d variantes de plus)", len(lignes)-25)
			break
		}
		t.Logf("  %-40s %d zones", l.nom, l.n)
	}
	_ = natif
}

// TestZonesNommeesIsolationDetail — le detail objet par objet sur la carte Forge Isolation.
func TestZonesNommeesIsolationDetail(t *testing.T) {
	vocab, _ := vocabulaireLocsExact(t)
	natif := chargeVocabCallouts(t)
	for _, nom := range []string{"isolation_map.mvar", "isolation_2e_asset_map.mvar", "aquarius_map.mvar", "streets_map.mvar", "live_fire_map.mvar", "recharge_map.mvar", "argyle_map.mvar"} {
		p := filepath.Join("C:\\", "Users", "Guillaume", "Projects", "LevelUp", ".ai", "re_dump", "mapvar", nom)
		brut, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		root, err := mapvar.DecodeRoot(brut)
		if err != nil {
			continue
		}
		zs := zonesNommeesDeVariante(root)
		t.Logf("=== %s : %d zones nommees ===", nom, len(zs))
		for _, z := range zs {
			etiquette := "(inconnu du catalogue natif)"
			if n, ok := vocabNoms[z.StringID]; ok {
				etiquette = n
			}
			dansVocab := "HORS locs"
			if vocab[z.StringID] {
				dansVocab = "dans locs"
			}
			t.Logf("  obj #%-5d type=%-12d sid=0x%08X %-9s pos=(%.1f, %.1f, %.1f) %-28s labels=%v  %s",
				z.Index, z.TypeID, z.StringID, dansVocab, z.Pos[0], z.Pos[1], z.Pos[2], z.Forme, z.Labels, etiquette)
		}
		_ = natif
	}
}

// TestCouvertureZonesNommeesVariantes — combien de noms distincts, combien deja resolus.
func TestCouvertureZonesNommeesVariantes(t *testing.T) {
	vocab, _ := vocabulaireLocsExact(t)
	natif := chargeVocabCallouts(t)
	dossier := filepath.Join("C:\\", "Users", "Guillaume", "Projects", "LevelUp", ".ai", "re_dump", "mapvar")
	entrees, err := os.ReadDir(dossier)
	if err != nil {
		t.Skipf("dump absent : %v", err)
	}
	sids := map[uint32]int{}
	champs4 := map[uint16]int{}
	var sansZone []string
	nCartes := 0
	for _, e := range entrees {
		if !strings.HasSuffix(e.Name(), ".mvar") {
			continue
		}
		brut, err := os.ReadFile(filepath.Join(dossier, e.Name()))
		if err != nil {
			continue
		}
		root, err := mapvar.DecodeRoot(brut)
		if err != nil {
			continue
		}
		nCartes++
		zs := zonesNommeesDeVariante(root)
		if len(zs) == 0 {
			sansZone = append(sansZone, e.Name())
		}
		for _, z := range zs {
			sids[z.StringID]++
		}
		// quels champs vivent sous #8/4[] ?
		if objs, ok := root.Field(3); ok {
			for _, o := range objs.Items {
				bag, ok := o.Field(8)
				if !ok {
					continue
				}
				l, ok := bag.Field(4)
				if !ok {
					continue
				}
				for _, it := range l.Items {
					for id := range it.Fields {
						champs4[id]++
					}
				}
			}
		}
	}
	nDansLocs, nNommables := 0, 0
	for s := range sids {
		if vocab[s] {
			nDansLocs++
		}
		if natif[s] {
			nNommables++
		}
	}
	t.Logf("%d variantes ; %d StringId de lieu DISTINCTS employes ; %d dans le vocabulaire locs (778) ; %d deja nommes par callouts_i18n.csv",
		nCartes, len(sids), nDansLocs, nNommables)
	t.Logf("champs presents sous #8/4[] : %v", champs4)
	t.Logf("%d variantes SANS zone nommee", len(sansZone))
	for i, n := range sansZone {
		if i >= 30 {
			t.Logf("  ... (%d de plus)", len(sansZone)-30)
			break
		}
		t.Logf("  %s", n)
	}
}

// TestIsolationSidsDansLeTableauLocs — pour chaque zone d'Isolation, l'INDICE et l'OFFSET
// exacts de son StringId dans le tableau de 778 entrees du tag locs. C'est la verification
// « au moins une valeur se resout contre le vocabulaire global » demandee, faite sur les 18.
func TestIsolationSidsDansLeTableauLocs(t *testing.T) {
	m, err := himodule.Open(moduleGlobals(t))
	if err != nil {
		t.Fatalf("globals : %v", err)
	}
	fichiers := m.Files("locs")
	if len(fichiers) == 0 {
		t.Skip("aucun locs")
	}
	tag, err := m.Extract(fichiers[0])
	if err != nil {
		t.Fatalf("extraire : %v", err)
	}
	const abs, n, stride = 0x120, 778, 4
	indice := map[uint32]int{}
	for i := 0; i < n; i++ {
		v := binary.LittleEndian.Uint32(tag[abs+i*stride:])
		if _, vu := indice[v]; !vu {
			indice[v] = i
		}
	}
	chargeVocabCallouts(t)
	brut, err := os.ReadFile(filepath.Join("C:\\", "Users", "Guillaume", "Projects", "LevelUp", ".ai", "re_dump", "mapvar", "isolation_map.mvar"))
	if err != nil {
		t.Skipf("isolation absente : %v", err)
	}
	root, err := mapvar.DecodeRoot(brut)
	if err != nil {
		t.Fatalf("mvar : %v", err)
	}
	zs := zonesNommeesDeVariante(root)
	nTrouves := 0
	for _, z := range zs {
		i, ok := indice[z.StringID]
		if !ok {
			t.Logf("  obj #%-5d sid=0x%08X  ABSENT du tableau locs", z.Index, z.StringID)
			continue
		}
		nTrouves++
		t.Logf("  obj #%-5d sid=0x%08X  locs[%3d] @ 0x%03X  %s",
			z.Index, z.StringID, i, abs+i*stride, vocabNoms[z.StringID])
	}
	t.Logf("=> %d/%d StringId d'Isolation resolus dans le tableau locs de 778 entrees", nTrouves, len(zs))
}
