//go:build research

package grammar

// lr_tetes_research_test.go — SONDE DU LOT LR : les records NEW qu une autre marche (la base du
// lot) a pris pour tete d une liste, relus sous la grammaire du lot. Pour chaque tete de LR_TETES
// (un fichier : `film chunk paquet bit` par ligne, separes par des tabulations), la sonde relit le
// record NEW au bit donne, dans le paquet donne, au moment ou la marche de la carte y passe : son
// archetype, le verdict du lecteur d etat de creation du jeu ([EntityTrace.EtatIllisible]) et les
// champs publies du bloc MPP. Elle dit enfin si le mot de 32 bits du bloc (`MPPWord32`) est CONNU :
// porte par un record NEW lisible d un paquet ferme du meme film. Un instrument : aucune sortie de
// production ne change.
//
//	LR_RACINE=<film_chunks> LR_SORTIE=<dir hors data> LR_TETES=<fichier> [LR_MPP_DECLARE=1] \
//	  go test -tags=research -count=1 -timeout 120m -run '^TestLRTetes$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

// lrRelu : ce que la relecture d un record NEW rend.
type lrRelu struct {
	neuf, illisible bool
	ti              uint32
	desync          int
	mpp             [MPPFieldCount]uint64
}

// lrRelire relit le record NEW qui commence au bit `bit` de `pay`.
func lrRelire(pay []byte, bit int, w *World, cfg FrameConfig) lrRelu {
	var r lrRelu
	essai := cfg
	essai.Obs = NouvelleObservation()
	essai.Obs.MppHook = func(f MPPField, v uint64, present bool) {
		if present {
			r.mpp[f] = v
		}
	}
	br := LecteurSur(pay)
	br.poserCadre(essai)
	br.SetBitPos(bit + motFacultatifDEnTete(cfg))
	if readRecordType(br) != recNew {
		return r
	}
	readRecordID(br, essai.IDLowBits, essai.IDBase)
	tr := TraverseEntity(br, w.Reg, essai.NewDefaultStateBits)
	r.neuf, r.illisible, r.ti, r.desync = true, tr.EtatIllisible, tr.TypeIndex, tr.DesyncAt
	return r
}

// lrTetesEcouteur relit les tetes d un film et recense les mots MPP connus.
type lrTetesEcouteur struct {
	film   string
	cfg    FrameConfig
	tetes  map[[2]int][]int
	connus map[uint64]bool
	relus  []string
	mots   []uint64
}

func (e *lrTetesEcouteur) debutDeChunk(int, []byte, []FilmPacket, *World) {}
func (e *lrTetesEcouteur) finDeFilm()                                     {}

func (e *lrTetesEcouteur) paquet(c int, p *cmPaquet, w *World) {
	for _, bit := range e.tetes[[2]int{c, p.pk.Index}] {
		r := lrRelire(p.pay, bit, w, e.cfg)
		e.relus = append(e.relus, fmt.Sprintf("%s\t%d\t%d\t%d\tneuf=%v\tti=%d\tdesync=%d\tetat_illisible=%v\tmot9=%d\tmot32=%#x\tvariante=%#x\tqueue=%#x",
			e.film, c, p.pk.Index, bit, r.neuf, r.ti, r.desync, r.illisible, r.mpp[MPPWord9], r.mpp[MPPWord32],
			r.mpp[MPPVariantName], r.mpp[MPPTailName]))
		e.mots = append(e.mots, r.mpp[MPPWord32])
	}
	if !p.d.Fermee {
		return
	}
	for _, rec := range p.recs {
		if rec.Type != recNew || rec.DesyncAt != -1 {
			continue
		}
		if r := lrRelire(p.pay, rec.HeaderBit, w, e.cfg); r.neuf && !r.illisible && r.mpp[MPPWord32] != 0 {
			e.connus[r.mpp[MPPWord32]] = true
		}
	}
}

// lrLireTetes lit LR_TETES, par film.
func lrLireTetes(t *testing.T) map[string]map[[2]int][]int {
	t.Helper()
	brut, err := os.ReadFile(os.Getenv("LR_TETES"))
	if err != nil {
		t.Skip("LR_TETES requis")
	}
	out := map[string]map[[2]int][]int{}
	for _, l := range strings.Split(strings.TrimSpace(string(brut)), "\n") {
		c := strings.Split(strings.TrimSpace(l), "\t")
		if len(c) != 4 {
			continue
		}
		var v [3]int
		for i := range v {
			if v[i], err = strconv.Atoi(c[i+1]); err != nil {
				t.Fatalf("LR_TETES : %q illisible", l)
			}
		}
		if out[c[0]] == nil {
			out[c[0]] = map[[2]int][]int{}
		}
		out[c[0]][[2]int{v[0], v[1]}] = append(out[c[0]][[2]int{v[0], v[1]}], v[2])
	}
	return out
}

// TestLRTetes ecrit `lr_tetes.tsv`.
func TestLRTetes(t *testing.T) {
	parFilm := lrLireTetes(t)
	racine, sortie := os.Getenv("LR_RACINE"), os.Getenv("LR_SORTIE")
	if racine == "" || sortie == "" {
		t.Skip("LR_RACINE et LR_SORTIE requis")
	}
	utiles := cmUtiles(t)
	lignes := []string{"film\tchunk\tpaquet\tbit\tneuf\tti\tdesync\tetat_illisible\tmot9\tmot32\tvariante\tqueue\tmot32_connu"}
	for id, tetes := range parFilm {
		f, ok := lrOuvrir(t, racine, id, utiles)
		if !ok {
			continue
		}
		e := &lrTetesEcouteur{film: id, cfg: f.cfg, tetes: tetes, connus: map[uint64]bool{}}
		cmMarcher(f, cmVariante{}, e)
		for i, l := range e.relus {
			lignes = append(lignes, fmt.Sprintf("%s\t%v", l, e.connus[e.mots[i]]))
		}
	}
	if err := os.MkdirAll(sortie, 0o750); err != nil {
		t.Fatal(err)
	}
	ltEcrire(t, sortie, "lr_tetes.tsv", lignes)
}
