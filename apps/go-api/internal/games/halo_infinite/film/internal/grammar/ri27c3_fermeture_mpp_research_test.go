//go:build research

package grammar

// ri27c3_fermeture_mpp_research_test.go — LA PREUVE DES BAISSES DU RATCHET DE FERMETURE D IMAGE-CLE
// SOUS LE DECOUPAGE MPP DECLARE (plan de l etape 2 de la representation intermediaire, item 2.7.c3).
//
// Le ratchet (`keyframe_closure_ratchet_test.go`) mesure les bobines par build sous le decoupage que
// le film declare ([FilmContext.ResolutionMPP]), et non plus sous celui du contexte par defaut (9/5).
// Sur les bobines anciennes, des records qui fermaient ne ferment plus, d autres ferment. Cet
// instrument les liste record par record et lit pour chacun le mot de 32 bits du bloc MPP
// (`MPPWord32`, le GlobalID du tag de l objet) sous les deux decoupages, confronte au catalogue des
// tags du jeu installe : la preuve 2 de la double preuve MPP de la campagne de grammaire
// (2026-10-05). Il rend aussi, par archetype, la part des identites connues sous chaque decoupage.
//
//	RI27C3_CATALOGUE=<catalogue_tags.tsv> RI27C3_OUT=<dossier> \
//	  go test -tags=research -count=1 -run '^TestRI27c3' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"bufio"
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// ri27c3GroupesAttendus : le groupe de tag que chaque archetype porte dans son bloc MPP (mesure de la
// campagne, identique sur tous les formats).
var ri27c3GroupesAttendus = map[int][]string{
	35: {"bipd"}, 37: {"eqip"}, 38: {"bloc", "scen"}, 40: {"vehi"}, 42: {"weap"}, 43: {"mach", "ctrl"},
}

// ri27c3Cle situe un record d image-cle : chunk, rang du paquet, premier bit.
type ri27c3Cle struct{ chunk, index, debut int }

// ri27c3Record est un record borne lu sous un decoupage.
type ri27c3Record struct {
	ti, slot, desync int
	ferme            bool
	// ecart : fin de la traversee moins la frontiere visee (negatif : la marche tombe avant).
	ecart int
	mot   uint32
	motLu bool
}

// ri27c3Env lit le catalogue et le dossier de sortie ; Skip sans eux.
func ri27c3Env(t *testing.T) (map[uint32]string, string) {
	t.Helper()
	chemin, sortie := os.Getenv("RI27C3_CATALOGUE"), os.Getenv("RI27C3_OUT")
	if chemin == "" || sortie == "" {
		t.Skip("instrument : RI27C3_CATALOGUE et RI27C3_OUT requis")
	}
	if err := os.MkdirAll(sortie, 0o750); err != nil {
		t.Fatalf("sortie : %v", err)
	}
	fh, err := os.Open(chemin) //nolint:gosec // instrument, chemin donne par l environnement
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	defer fh.Close()
	cat := map[uint32]string{}
	sc := bufio.NewScanner(fh)
	for sc.Scan() {
		c := strings.Split(sc.Text(), "\t")
		g, err := strconv.ParseUint(c[0], 16, 32)
		if err != nil || len(c) < 2 {
			continue
		}
		if prev, ok := cat[uint32(g)]; !ok {
			cat[uint32(g)] = c[1]
		} else if !strings.Contains(prev, c[1]) {
			cat[uint32(g)] = prev + "+" + c[1]
		}
	}
	return cat, sortie
}

// ri27c3Identite rend le premier MPPWord32 transmis par la traversee de l etat complet du record de
// `bit`, sous le contexte de lecture `ctx`.
func ri27c3Identite(pay []byte, bit int, reg *Registry, ctx ContexteDeLecture) (uint32, bool) {
	var mot uint32
	vu := false
	ctx.Obs = &Observation{MppHook: func(f MPPField, v uint64, present bool) {
		if f == MPPWord32 && present && !vu {
			mot, vu = uint32(v), true //nolint:gosec // R(32)
		}
	}}
	WalkKeyframeFullState(pay, bit, reg, ctx)
	return mot, vu
}

// ri27c3Records rend les records bornes de la phase des images-cles de `fc`, sous son decoupage, avec
// la regle de [KeyframeClosure] (ferme : traverse jusqu au bout, preuve fermee).
func ri27c3Records(t *testing.T, fc *FilmContext) map[ri27c3Cle]ri27c3Record {
	t.Helper()
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	out := map[ri27c3Cle]ri27c3Record{}
	for p, err := range fc.ImagesCles() {
		if err != nil {
			t.Fatalf("images-cles : %v", err)
		}
		for i := 0; i+1 < len(p.Records); i++ {
			r := &p.Records[i]
			rec := ri27c3Record{ti: int(r.TI), slot: int(r.Vie.Slot), desync: int(r.Desync),
				ferme: r.Desync < 0 && r.Preuve == lecture.PreuveFerme,
				ecart: int(r.Debut+r.Bits) - int(p.Records[i+1].Debut)}
			rec.mot, rec.motLu = ri27c3Identite(p.Payload, int(r.Debut), reg, fc.ContexteDeLecture())
			out[ri27c3Cle{p.Chunk, p.Index, int(r.Debut)}] = rec
		}
	}
	return out
}

// ri27c3Groupe rend le groupe du mot dans le catalogue, « inconnu » sinon, et s il est celui que
// l archetype attend.
func ri27c3Groupe(cat map[uint32]string, ti int, mot uint32, lu bool) (string, bool) {
	if !lu {
		return "non-lu", false
	}
	g, ok := cat[mot]
	if !ok {
		return "inconnu", false
	}
	for _, a := range ri27c3GroupesAttendus[ti] {
		if strings.Contains(g, a) {
			return g, true
		}
	}
	return g, false
}

// TestRI27c3FermetureSousLeDecoupageDeclare : records d image-cle qui changent de fermeture entre le
// decoupage par defaut et le decoupage declare, et identites lues sous les deux.
func TestRI27c3FermetureSousLeDecoupageDeclare(t *testing.T) {
	cat, sortie := ri27c3Env(t)
	lignes := []string{"film\tchunk\tpaquet\tbit\tslot\tti\tsens\tdesync_defaut\tdesync_declare\t" +
		"ecart_defaut\tecart_declare\tmot_defaut\tgroupe_defaut\tattendu_defaut\tmot_declare\tgroupe_declare\tattendu_declare"}
	parts := []string{"film\tdecoupage\tti\trecords\tattendus_defaut\tattendus_declare"}
	for _, court := range closureMiniFilms() {
		film, err := source.LoadDir(filepath.Join("..", "..", "replay", "testdata", "minifilm_"+court), nil)
		if err != nil {
			t.Fatalf("bobine %s : %v", court, err)
		}
		declare := NewFilmContext(film)
		res := declare.ResolutionMPP()
		if !res.Decide() {
			t.Logf("%s : aucun decoupage resolu", court)
			continue
		}
		declare.PoserMPP(res.Widths)
		avant, apres := ri27c3Records(t, NewFilmContext(film)), ri27c3Records(t, declare)
		cles := make([]ri27c3Cle, 0, len(avant))
		for c := range avant {
			cles = append(cles, c)
		}
		slices.SortFunc(cles, func(a, b ri27c3Cle) int {
			return cmp.Or(cmp.Compare(a.chunk, b.chunk), cmp.Compare(a.index, b.index), cmp.Compare(a.debut, b.debut))
		})
		absents, parTI := 0, map[int][3]int{}
		for _, c := range cles {
			a := avant[c]
			b, ok := apres[c]
			if !ok || b.ti != a.ti || b.slot != a.slot {
				absents++
				continue
			}
			ga, oka := ri27c3Groupe(cat, a.ti, a.mot, a.motLu)
			gb, okb := ri27c3Groupe(cat, b.ti, b.mot, b.motLu)
			if _, porte := ri27c3GroupesAttendus[a.ti]; porte {
				n := parTI[a.ti]
				n[0]++
				if oka {
					n[1]++
				}
				if okb {
					n[2]++
				}
				parTI[a.ti] = n
			}
			sens := ""
			switch {
			case a.ferme && !b.ferme:
				sens = "perdu"
			case !a.ferme && b.ferme:
				sens = "gagne"
			default:
				continue
			}
			lignes = append(lignes, fmt.Sprintf("%s\t%d\t%d\t%d\t%d\t%d\t%s\t%d\t%d\t%d\t%d\t%08x\t%s\t%t\t%08x\t%s\t%t",
				court, c.chunk, c.index, c.debut, a.slot, a.ti, sens, a.desync, b.desync, a.ecart, b.ecart,
				a.mot, ga, oka, b.mot, gb, okb))
		}
		tis := make([]int, 0, len(parTI))
		for ti := range parTI {
			tis = append(tis, ti)
		}
		slices.Sort(tis)
		for _, ti := range tis {
			n := parTI[ti]
			parts = append(parts, fmt.Sprintf("%s\t%s\t%d\t%d\t%d\t%d", court, res.Widths, ti, n[0], n[1], n[2]))
		}
		t.Logf("%s : decoupage %s (provenance %v), %d records bornes, %d sans correspondant",
			court, res.Widths, res.Provenance, len(avant), absents)
	}
	for nom, l := range map[string][]string{"ri27c3_changements.tsv": lignes, "ri27c3_identites.tsv": parts} {
		if err := os.WriteFile(filepath.Join(sortie, nom), []byte(strings.Join(l, "\n")+"\n"), 0o600); err != nil {
			t.Fatalf("ecriture %s : %v", nom, err)
		}
	}
}
