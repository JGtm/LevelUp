package filmdec

// objectif_ti11_masque_test.go — LE RECENSEMENT QUI CHIFFRE LE PORTAGE DE `ti=11`.
//
// # LA QUESTION, ET POURQUOI ELLE SE POSE AVANT D'OUVRIR GHIDRA
//
// L'archetype `ti=11` porte la jauge d'objectif : `managed-objective-progress-component` (i12)
// et son seuil `managed-objective-required-progress-component` (i13). Il est couvert **0 sur 34**
// par `consumeByName` — deux deserialiseurs existent (`components_batch3.go`, i2 et i9) mais sans
// appelant.
//
// La boucle de composants est SEQUENTIELLE : `traverseComponentLoop` consomme les composants
// PRESENTS dans l'ordre du masque, et s'arrete au premier qui n'est pas porte (`DesyncAt`). Pour
// LIRE i12, il faut donc porter tous les composants presents AVANT lui — mais SEULEMENT ceux qui
// sont effectivement presents. Le devis du chantier n'est donc pas « treize deserialiseurs » par
// principe : c'est ce que le masque dit, et le masque se lit SANS aucun deserialiseur.
//
// # POURQUOI LE MASQUE EST LISIBLE ALORS QUE LE CORPS NE L'EST PAS
//
// Le prologue d'un record vaut `R(6) typeIndex ; default-state ; R(1) gate ; masque`. Le
// default-state de `ti=11` est PORTE (`defaultStateDeserByTI[11] = consumeVersionPrefix`,
// FUN_14110d4d8, « V seul »). Le masque tombe donc a une position connue, et `TraverseEntity` le
// publie (`EntityTrace.Mask`) AVANT d'entrer dans la boucle qui, elle, va desynchroniser.
//
// # LE CONTROLE QUI REND LA MESURE HONNETE
//
// Un masque lu au mauvais endroit est du bruit, et du bruit se compte aussi bien qu'un signal.
// Le garde-fou est le DOMAINE : `ti=11` a 34 composants, donc tout bit de rang >= 34 est
// impossible. La part de masques HORS DOMAINE est rendue film par film ; au-dela de quelques
// pourcents, la mesure ne dit rien et il faut douter du prologue avant de douter du reste.
//
// # LE VOISIN A LIRE ENSUITE
//
// `objectif_ti11_oracle_test.go` porte l'ORACLE DE LARGEUR et le balayage A/B des bascules. Le
// recensement dit COMBIEN de deserialiseurs porter ; l'oracle dit si on les lit JUSTE. Les deux
// se lisent dans cet ordre.
//
// # LE RESULTAT D'ENSEMBLE, ecrit ici parce qu'il conditionne tout usage de ti=11 (2026-09-01)
//
// Le portage lui-meme est acquis : 2 211 records marches jusqu'au bout contre 884 avant. Mais la
// COUVERTURE n'est pas la JUSTESSE, et l'oracle le montre sans ambiguite :
//
//	records a UN seul composant        25 % de chainage
//	records a PLUSIEURS composants      0 %
//
// Si les largeurs etaient justes, la fin d'un record tomberait sur l'en-tete du suivant a presque
// 100 % — la probabilite qu'une position quelconque passe `readKeyframeHeader` est de l'ordre de
// 1e-5, donc 25 % est un vrai signal, mais un signal INSUFFISANT. Et le fait que meme les records
// LES PLUS SIMPLES plafonnent a 25 % dit que le probleme n'est pas (seulement) dans les
// composants : il est EN AMONT — prologue de record, ou faux ancrages rendus par le balayeur.
//
// La calibration a balaye les deux bascules disponibles (`filmComponentCorruptionCheck` x
// `newRecordTailBits`) : AUCUNE combinaison ne depasse 25 % / 13,6 %. Negatif net, critere ecrit
// avant la mesure (60 % sur les records a plusieurs composants).
//
// **NE PAS EXPLOITER LES VALEURS DE ti=11 AVANT QUE CE CHAINAGE NE MONTE.**
//
// REGIME : garde `ASSAUT_CACHE`. Aucune base, aucun reseau, sentinelle memoire armee.
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/filmdec/ -run ObjectifTi11 -v -timeout 40m

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/filmproc"
)

// ti11ArchIndex est l'archetype des objectifs geres, et ti11Composants sa taille (le registre
// la redonne a chaque film ; la constante ne sert qu'au controle de domaine).
const (
	ti11ArchIndex  = 11
	ti11Composants = 34
)

// ti11Corpus : les neuf films d'Assaut fouilles depuis le lot A, et QUATRE TEMOINS de modes ou
// la progression d'objectif existe a l'ecran. Sans temoin, un recensement vide ne se distingue
// pas d'un instrument casse.
var ti11Corpus = []struct{ id, mode string }{
	{"2ce58582", "TEMOIN Strongholds classe"},
	{"696a9d7c", "TEMOIN Strongholds arene"},
	{"7f1bbf06", "TEMOIN KOTH arene"},
	{"cde26226", "TEMOIN CTF arene"},
	{"9f57c612", "Assaut"},
	{"c75f33b8", "Assaut"},
	{"df8fcbef", "Assaut"},
	{"34bb3bc8", "Assaut"},
	{"1c01e34f", "Assaut"},
	{"ce083875", "Assaut"},
	{"35b75a31", "Assaut"},
	{"69b16f5d", "Assaut"},
	{"3d58eb37", "Assaut"},
}

// ti11Bilan agrege un film.
type ti11Bilan struct {
	records     int
	horsDomaine int
	presence    [64]int // presence[i] = nombre de records portant le composant i
	desync      map[int]int
	avecJauge   int
	compagnons  map[int]int // parmi les records portant i12 : les autres composants presents
}

func nouveauTi11Bilan() *ti11Bilan {
	return &ti11Bilan{desync: map[int]int{}, compagnons: map[int]int{}}
}

// TestObjectifTi11Masques recense les masques de presence des records `ti=11`.
func TestObjectifTi11Masques(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	g := filmproc.Arm("TestObjectifTi11Masques", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — recensement interrompu", float64(peak)/(1<<30))
	})
	defer func() {
		g.Disarm()
		t.Logf("pic memoire observe : %.2f Gio (plafond souple %d Gio)",
			float64(g.Peak())/(1<<30), filmproc.MeasureLimitGiB)
	}()

	global := nouveauTi11Bilan()
	for _, f := range ti11Corpus {
		dir := filepath.Join(cache, "film_chunks", f.id)
		reg, err := ti11Registre(dir)
		if err != nil {
			t.Logf("%-9s %-26s registre illisible (%v) — saute", f.id, f.mode, err)
			continue
		}
		b := ti11RecenserFilm(dir, reg)
		t.Logf("%-9s %-26s %4d record(s) ti=11, %d hors domaine, jauge i12 sur %d",
			f.id, f.mode, b.records, b.horsDomaine, b.avecJauge)
		if b.records > 0 {
			t.Logf("           presence : %s", ti11Histogramme(reg, b.presence[:], b.records))
			t.Logf("           premier composant bloquant : %s", ti11Desync(reg, b.desync))
		}
		ti11Fusionner(global, b)
	}

	t.Logf("########## BILAN — %d record(s) ti=11, %d hors domaine (%.1f %%), %d portant la jauge i12",
		global.records, global.horsDomaine, ti11Part(global.horsDomaine, global.records), global.avecJauge)
	if global.records == 0 {
		return
	}
	reg, err := ti11Registre(filepath.Join(cache, "film_chunks", ti11Corpus[0].id))
	if err == nil {
		t.Logf("PRESENCE GLOBALE : %s", ti11Histogramme(reg, global.presence[:], global.records))
		t.Logf("BLOCAGE GLOBAL   : %s", ti11Desync(reg, global.desync))
		if global.avecJauge > 0 {
			t.Logf("DEVIS DU PORTAGE — composants presents AVEC la jauge i12 : %s",
				ti11Histogramme(reg, ti11Tableau(global.compagnons), global.avecJauge))
		}
	}
}

// ti11Registre lit et analyse le chunk 0 d'un film.
func ti11Registre(dir string) (*Registry, error) {
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		return nil, err
	}
	return ParseRegistryChunk(raw)
}

// ti11RecenserFilm parcourt les images-cles d'un film et releve le masque de chaque record ti=11.
func ti11RecenserFilm(dir string, reg *Registry) *ti11Bilan {
	b := nouveauTi11Bilan()
	n := CountFilmChunks(dir)
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			for _, r := range WalkKeyframeWorld(pay) {
				if r.TI != ti11ArchIndex {
					continue
				}
				ti11ReleverRecord(b, pay, reg, r.Bit)
			}
		}
	}
	return b
}

// ti11ReleverRecord rejoue le prologue d'UN record et range son masque.
func ti11ReleverRecord(b *ti11Bilan, pay []byte, reg *Registry, bit int) {
	br := NewBitReader(pay)
	br.SetBitPos(bit + keyframeRecordTIBit)
	tr := TraverseEntity(br, reg, 0)
	if tr.TypeIndex != ti11ArchIndex {
		return
	}
	b.records++
	if tr.Mask>>ti11Composants != 0 {
		b.horsDomaine++
		return
	}
	for i := 0; i < ti11Composants; i++ {
		if tr.Mask>>uint(i)&1 == 1 {
			b.presence[i]++
		}
	}
	b.desync[tr.DesyncAt]++
	if tr.Mask>>12&1 == 1 {
		b.avecJauge++
		for i := 0; i < ti11Composants; i++ {
			if tr.Mask>>uint(i)&1 == 1 {
				b.compagnons[i]++
			}
		}
	}
}

// ti11Fusionner cumule le bilan d'un film dans le bilan global.
func ti11Fusionner(dst, src *ti11Bilan) {
	dst.records += src.records
	dst.horsDomaine += src.horsDomaine
	dst.avecJauge += src.avecJauge
	for i := range src.presence {
		dst.presence[i] += src.presence[i]
	}
	for k, v := range src.desync {
		dst.desync[k] += v
	}
	for k, v := range src.compagnons {
		dst.compagnons[k] += v
	}
}

// ti11Tableau convertit un comptage par index en tableau dense.
func ti11Tableau(m map[int]int) []int {
	out := make([]int, 64)
	for k, v := range m {
		if k >= 0 && k < 64 {
			out[k] = v
		}
	}
	return out
}

// ti11Histogramme rend les composants presents, du plus frequent au moins frequent.
func ti11Histogramme(reg *Registry, presence []int, total int) string {
	type l struct {
		i, n int
	}
	var ls []l
	for i, n := range presence {
		if n > 0 {
			ls = append(ls, l{i, n})
		}
	}
	if len(ls) == 0 {
		return "(aucun)"
	}
	sort.Slice(ls, func(a, c int) bool {
		if ls[a].n != ls[c].n {
			return ls[a].n > ls[c].n
		}
		return ls[a].i < ls[c].i
	})
	arch, _ := reg.Archetype(ti11ArchIndex)
	var sb strings.Builder
	for k, x := range ls {
		if k > 0 {
			sb.WriteString(", ")
		}
		fmt.Fprintf(&sb, "i%d%s %d (%.0f %%)", x.i, ti11Nom(arch, x.i), x.n, ti11Part(x.n, total))
	}
	return sb.String()
}

// ti11Desync rend la distribution du premier composant non porte.
func ti11Desync(reg *Registry, m map[int]int) string {
	ks := make([]int, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(a, c int) bool { return m[ks[a]] > m[ks[c]] })
	arch, _ := reg.Archetype(ti11ArchIndex)
	var sb strings.Builder
	for k, i := range ks {
		if k > 0 {
			sb.WriteString(", ")
		}
		if i < 0 {
			fmt.Fprintf(&sb, "AUCUN (record entierement porte) %d", m[i])
			continue
		}
		fmt.Fprintf(&sb, "i%d%s %d", i, ti11Nom(arch, i), m[i])
	}
	return sb.String()
}

// ti11Nom rend le nom court du composant d'index i, ou la chaine vide s'il est hors registre.
func ti11Nom(arch Archetype, i int) string {
	if i < 0 || i >= len(arch.Components) {
		return ""
	}
	nom := strings.TrimPrefix(arch.Components[i], "managed-objective-")
	return " " + strings.TrimSuffix(nom, "-component")
}

// ti11Part rend un pourcentage, zero si le total est nul.
func ti11Part(n, total int) float64 {
	if total == 0 {
		return 0
	}
	return 100 * float64(n) / float64(total)
}
