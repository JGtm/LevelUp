package grammar

// keyframe_fullstate_loop_test.go — INSTRUMENT DU LOT R7-e
// (cf. .ai/V7.5/replay2d/PLAN_R7E_BOUCLE_ETAT_COMPLET.md).
//
// LA QUESTION : la boucle d'ETAT COMPLET du jeu (`FUN_142e2bfd0` -> `FUN_1428e2b68` ->
// `FUN_142e2c690`, lue par R7-d) portee TELLE QUELLE sur le payload type-2 atterrit-elle
// bit-exact ? La variable qui reste, allumee seule :
//
//	(c) le CONTROLE par composant — `R(1) [+R(32)]` sous le drapeau film
//
// QUATRE VARIABLES ONT ETE TRANCHEES ET LIVREES, ET ELLES ONT DISPARU DE LA MATRICE :
//
//	(a) « le niveau du composant `i` est celui de l entree `i` » — lot 1.2 (2026-09-14) : c est
//	    la lecture du registre (`registry.go`), avec `KeyframeFullStateOpt.LevelShift`.
//	(b) l EN-TETE par entite (108 bits + deux `R(32)` de taille, contre 64) — lot 1.4
//	    (2026-09-14) : c est la lecture de production (`WalkKeyframeFullState`, sans argument de
//	    cadre), avec `KeyframeFullStateOpt` tout entier.
//	(d) `DAT_144e61ea0` et (e) la grammaire de l'ECRIVAIN d'`i0` — plan LK (2026-10-08) : la marche
//	    d'etat complet pose la portee ([Lecteur.portee]), sous laquelle la branche absolue d'`i0`
//	    lit la forme de l'ecrivain ([consumeAbsoluSousLaPortee]). L'ancienne ligne (d+e) est la REF.
//
// Elles ont disparu parce que le CHOIX a disparu, pas parce qu on aurait cesse de mesurer.
//
// CE QU'IL NE FAIT PAS : il ne publie AUCUNE donnee, n'ecrit RIEN sur disque, ne touche a
// aucun schema. LECTURE SEULE, garde par KF35_ROOT (meme garde que R7-a/R7-b/R7-d).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 KF35_ROOT=<repo>/data/cache/film_chunks \
//	  go test ./internal/games/halo_infinite/film/internal/grammar/ -run '^TestKF7E' -timeout 90m -v

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
	"os"
	"sort"
	"testing"
)

// kf7eInflate rend le chunk_00 DEFLATE d'un film, tel que `ParseRegistryChunk` le lit — mais
// en OCTETS BRUTS : c'est la seule facon d'avoir confronte l'ancien cadrage de `registry.go`
// (`[u32 kind][u32 flags][nom @ +8]` depuis l'octet 0) a celui que `FUN_142e2c690` lit en
// memoire (`[nom @ +0x00][u32 niveau @ +0x100]`, entree de 0x104 depuis l'octet 8, 64 par
// archetype) — ce dernier etant la lecture de production depuis le lot 1.2.
func kf7eInflate(t *testing.T, dir string) []byte {
	t.Helper()
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 illisible : %v", err)
	}
	if len(raw) < 2 || raw[0] != 0x78 {
		return raw
	}
	zr, err := zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("zlib : %v", err)
	}
	dec, err := io.ReadAll(zr)
	if err != nil && len(dec) == 0 {
		t.Fatalf("inflate : %v", err)
	}
	return dec
}

// kf7eCString rend la chaine NUL-terminee a `off`, bornee a `max` octets.
func kf7eCString(d []byte, off, max int) string {
	if off < 0 || off >= len(d) {
		return ""
	}
	end := min(off+max, len(d))
	if i := bytes.IndexByte(d[off:end], 0); i >= 0 {
		return string(d[off : off+i])
	}
	return string(d[off:end])
}

// TestKF7ETableLayout a tranche la variable (a) SANS supposer : il lit les octets bruts du bloc
// d'archetype du bipede et confronte les DEUX cadrages. Verdict, livre au lot 1.2 : le « kind »
// que l'ancien cadrage lisait en `+0` est TOUJOURS nul — queue de bourrage du nom precedent — et
// le « flags » qu'il lisait en `+4` etait le NIVEAU du composant PRECEDENT. Il reste ici comme
// TEMOIN SUR LE CORPUS DE RECHERCHE (garde `KF35_ROOT`, films reels) ; le temoin qui tourne sans
// cache de films, sur les bobines versionnees, est `TestRegistreNiveauxVoisinsCensus`.
func TestKF7ETableLayout(t *testing.T) {
	root := os.Getenv(kf35RootEnv)
	if root == "" {
		t.Skipf("%s absent : instrument de mesure saute", kf35RootEnv)
	}
	d := kf7eInflate(t, root+"/"+kf35OracleFilms[0])
	t.Logf("chunk_00 inflate : %d octets · %% 0x4100 = %d · (len-8) %% 0x4100 = %d · blocs %d",
		len(d), len(d)%archetypeBlockSize, (len(d)-8)%archetypeBlockSize, len(d)/archetypeBlockSize)

	base := bipedDefaultStateTypeIndex * archetypeBlockSize
	nonZeroKind, nonZeroTail := 0, 0
	for s := range archetypeBlockSlots {
		off := base + s*registrySlotSize
		if off+registrySlotSize > len(d) {
			break
		}
		kind := binary.LittleEndian.Uint32(d[off:])
		flags := binary.LittleEndian.Uint32(d[off+4:])
		nameGo := kf7eCString(d, off+8, registrySlotSize-8)
		// Layout du JEU : l'entree k commence au nom que registry.go lit en +8, et son
		// niveau est le u32 situe 0x100 octets plus loin.
		lvlGame := uint32(0)
		if off+8+0x100+4 <= len(d) {
			lvlGame = binary.LittleEndian.Uint32(d[off+8+0x100:])
		}
		if kind != 0 {
			nonZeroKind++
		}
		if nameGo == "" {
			continue
		}
		nonZeroTail++
		if s < 12 || s >= 60 {
			t.Logf("  slot %2d | registry.go kind=%d flags=%d | JEU niveau(+0x100)=%d | %s",
				s, kind, flags, lvlGame, nameGo)
		}
	}
	t.Logf("  -> %d slots nommes · %d slots avec kind != 0 (layout registry.go)", nonZeroTail, nonZeroKind)
}

// ---------------------------------------------------------------------------------------
// LA MESURE — la REF, puis la variable qui reste.
// ---------------------------------------------------------------------------------------

// kf7eCase est UNE configuration mesuree : un libelle, les options de marche, et les
// bascules globales qu'elle installe.
type kf7eCase struct {
	Label string
	Corr  bool // (c) le controle par composant du mode film
}

// kf7eTally compte ce qu'une configuration a rencontre sur un film. Memes denominateurs et
// memes definitions que `kf35Tally` (R7-a/R7-b) : `bounded` est le denominateur publie.
type kf7eTally struct {
	bounded, exact, chained, desync, lost int
	consumed, absGaps                     []int
	breaks                                map[string]int
}

func newKF7ETally() kf7eTally { return kf7eTally{breaks: map[string]int{}} }

func (k kf7eTally) rate() float64 {
	if k.bounded == 0 {
		return 0
	}
	return 100 * float64(k.exact+k.chained) / float64(k.bounded)
}

// kf7eWalkOne rejoue le corps d'UN record sous la configuration donnee, puis mesure.
func kf7eWalkOne(f kf35Film, pay []byte, b kf35Bound, c kf7eCase, tal *kf7eTally) {
	tr := WalkKeyframeFullState(pay, b.Rec.Bit, f.Reg, contexteDInstrument())
	if tr.DesyncAt >= 0 {
		tal.desync++
		return
	}
	tal.consumed = append(tal.consumed, tr.EndBit-b.Rec.Bit)
	if tr.EndBit == b.Want {
		tal.exact++
		return
	}
	gap := b.Want - tr.EndBit
	if gap < 0 {
		gap = -gap
	}
	tal.absGaps = append(tal.absGaps, gap)
	tal.breaks[kf35Break(tr, b.Want)]++
	if kf7eChain(f, pay, tr.EndBit, b) {
		tal.chained++
		return
	}
	tal.lost++
}

// kf7eChain enchaine la marche SOUS LA MEME CONFIGURATION jusqu a la frontiere visee : c est
// le rattrapage des records que le filtre fort du balayeur ne voit pas (meme borne que R7-a).
// La configuration ne s y passe plus en argument depuis le lot 1.4 : le cadre est fixe, et les
// trois bascules qui restent sont des globales de process, deja installees par `kf7ePass`.
func kf7eChain(f kf35Film, pay []byte, from int, b kf35Bound) bool {
	total := len(pay) * 8
	pos, prev := from, b.Rec.Slot
	for range kf35ChainMax {
		if pos == b.Want {
			return true
		}
		if pos > b.Want || pos+keyframeHeaderBits > total {
			return false
		}
		h, ok := readKeyframeHeader(pay, pos, total)
		if !ok || h.Slot <= prev {
			return false
		}
		tr := WalkKeyframeFullState(pay, pos, f.Reg, contexteDInstrument())
		if tr.DesyncAt >= 0 {
			return false
		}
		pos, prev = tr.EndBit, h.Slot
	}
	return false
}

// kf7ePass mesure UNE configuration sur UN film, bascules globales installees et restaurees.
func kf7ePass(f kf35Film, c kf7eCase) kf7eTally {
	defer poserBasculeDInstrument(func(g *GrammaireBalayage) { g.ControleDeCorruption = c.Corr })()
	tal := newKF7ETally()
	for _, pay := range f.Pays {
		for _, b := range kf35BoundedRecs(pay) {
			tal.bounded++
			kf7eWalkOne(f, pay, b, c, &tal)
		}
	}
	return tal
}

// kf7eCases construit la matrice A/B des variables QUI RESTENT.
//
// LA VARIABLE (b) — le CADRE (en-tete 108, mots de taille, etat par defaut) — A ETE TRANCHEE ET
// LIVREE au lot 1.4 (2026-09-14), comme (a) l'avait ete au lot 1.2 : ce n'est plus une option,
// c'est la lecture de production (`WalkKeyframeFullState`, sans argument de cadre). Les quatre
// lignes qui la balayaient (REF en-tete 64, b1, b2, b3) ont donc disparu de cette matrice —
// parce que le CHOIX a disparu, pas parce qu'on aurait cesse de mesurer. Toutes les lignes
// ci-dessous lisent desormais le meme cadre, celui du jeu, sous la portee que la marche pose avec
// la branche absolue d i0 de l ecrivain (d, e : la REF depuis le plan LK) ; ce qui varie est ce
// qui reste ouvert : le controle par composant (c).
func kf7eCases() []kf7eCase {
	return []kf7eCase{
		{Label: "REF    cadre d etat complet, portee et i0 de l ecrivain"},
		{Label: "(c)    + controle par composant", Corr: true},
	}
}

// TestKF7EFullStateLoop est LA MESURE de la phase 2 : chaque configuration, sur les trois
// films du corpus ferme, largeurs de la carte installees et trous neutralises.
func TestKF7EFullStateLoop(t *testing.T) {
	films := kf35Films(t)

	defer poserBasculeDInstrument(func(g *GrammaireBalayage) { g.SimStateComplet = true })()

	for _, f := range films {
		kf7eOneFilm(t, f)
	}
}

// kf7eOneFilm joue toute la matrice sur UN film.
func kf7eOneFilm(t *testing.T, f kf35Film) {
	t.Helper()
	_, restorePrec := kf35bInstallPrecision(t, f.Name)
	defer restorePrec()
	stubbed, restoreStubs := kf35ApplyStubs(f, kf7dVariant)
	defer restoreStubs()
	t.Logf("======== %s — composants neutralises : %d ========", f.Name, len(stubbed))

	for _, c := range kf7eCases() {
		tal := kf7ePass(f, c)
		t.Logf("  %-42s exactes %4d · chainees %4d · desync %4d / %4d | ATTERRISSAGE %5.2f %%"+
			" | longueur MEDIANE %5d · ecart absolu MEDIAN %5d",
			c.Label, tal.exact, tal.chained, tal.desync, tal.bounded, tal.rate(),
			kf35Median(tal.consumed), kf35Median(tal.absGaps))
		kf7eLogBreaks(t, tal.breaks, 4)
	}
}

// kf7eLogBreaks publie les `n` points de decrochage les plus frequents.
func kf7eLogBreaks(t *testing.T, hist map[string]int, n int) {
	t.Helper()
	if len(hist) == 0 {
		return
	}
	type kv struct {
		k string
		n int
	}
	xs := make([]kv, 0, len(hist))
	for k, v := range hist {
		xs = append(xs, kv{k, v})
	}
	sort.Slice(xs, func(i, j int) bool { return xs[i].n > xs[j].n })
	if len(xs) > n {
		xs = xs[:n]
	}
	for _, x := range xs {
		t.Logf("        %-62s %4d fois", x.k, x.n)
	}
}
