//go:build research

package grammar

// e191c_etat_par_defaut_research_test.go — LOT 1.9.1 bis, PAS 2 : LA LARGEUR D ETAT PAR DEFAUT
// IMPLIQUEE PAR LA FERMETURE, POUR LES CINQ ARCHETYPES OBJET.
//
// # LA QUESTION
//
// `FUN_142e2bfd0` lit, autour de l etat par defaut d un record d image-cle :
//
//	[108 bits d en-tete] R(32) n1 | etat par defaut (largeur w) | R(32) n2 | composants
//
// L etat par defaut precede TOUS les composants : une largeur fausse decale la marche entiere
// avant meme i0, et AUCUNE relecture de composant ne peut la rattraper. Les six composants du
// prefixe objet viennent d etre relus chez l ecrivain et sont bit-exacts ; la largeur d etat
// par defaut de `ti=37` n est, elle, validee par aucun oracle — le garde-rail `n2` ECARTE
// explicitement les archetypes 36 a 39 de son jugement (etat a branches).
//
// # CE QUE CET INSTRUMENT MESURE
//
// Il rejoue la marche en REMPLACANT le deserialiseur d etat par defaut par un DECALAGE de `w`
// bits (le bouton `keyframeFullStateTemoin.SansEtatParDefaut`, qui existe pour cela), et publie
// la fermeture pour chaque `w` du balayage. Un `w` qui fait BONDIR la fermeture designe la
// largeur que le jeu ecrit ; l ecart a la largeur portee dit de combien le deserialiseur se
// trompe. Aucune correction n est faite ici : la grammaire se relit chez l ecrivain (D13),
// l instrument ne fait que nommer la largeur a aller verifier.
//
// La LARGEUR PORTEE est mesuree a cote, sur les memes records, pour que l ecart se lise.
//
// LECTURE SEULE, sans garde d environnement (bobines versionnees).
//
//	go test ./internal/games/halo_infinite/film/filmdec/ -run '^TestE191cEtatParDefaut$' -v -count=1

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/source"
)

// e191cBalayageMax borne le balayage de largeur d etat par defaut. Le plus large etat par
// defaut connu du dossier est celui du bipede (198 bits) ; 512 laisse deux fois et demie la
// place sans faire exploser le cout.
const e191cBalayageMax = 512

// e191cEtatMesure est le resultat d un balayage pour UN archetype.
type e191cEtatMesure struct {
	Bornes   int
	ParW     map[int]int // fermetures par largeur d etat par defaut substituee
	Portee   map[int]int // largeurs consommees par le deserialiseur du depot
	FermePor int         // fermetures avec le deserialiseur du depot
}

// TestE191cEtatParDefaut publie, pour chacun des cinq archetypes objet, les largeurs d etat par
// defaut qui ferment le plus de records, et la largeur que le deserialiseur porte aujourd hui.
func TestE191cEtatParDefaut(t *testing.T) {
	t.Logf("######## PAS 2 — LARGEUR D ETAT PAR DEFAUT IMPLIQUEE (balayage 0..%d bits) ########", e191cBalayageMax)
	par := map[int]*e191cEtatMesure{}
	for _, ti := range e191cCinq {
		par[ti] = &e191cEtatMesure{ParW: map[int]int{}, Portee: map[int]int{}}
	}
	for _, court := range closureMiniFilms() {
		e191cBalayerBobine(t, court, par)
	}
	for _, ti := range e191cCinq {
		m := par[ti]
		t.Logf("  ti=%-3d %5d records bornes ; deser du depot : %5d fermes (%.2f %%), largeurs consommees %s",
			ti, m.Bornes, m.FermePor, 100*float64(m.FermePor)/float64(max(m.Bornes, 1)),
			e191cTopLargeurs(m.Portee))
		t.Logf("         meilleures largeurs substituees : %s", e191cTopLargeurs(m.ParW))
	}
}

// e191cBalayerBobine accumule le balayage d une bobine dans les mesures par archetype.
func e191cBalayerBobine(t *testing.T, court string, par map[int]*e191cEtatMesure) {
	t.Helper()
	dir := filepath.Join("..", "replay", "testdata", "minifilm_"+court)
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	fc := NewFilmContext(film)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre %s : %v", court, err)
	}
	for _, pay := range e191cPayloads(fc) {
		for _, b := range keyframeBornes(pay) {
			m := par[b.TI]
			if m == nil {
				continue
			}
			m.Bornes++
			e191cUnRecord(pay, reg, b, m)
		}
	}
}

// e191cUnRecord mesure UN record : la marche portee, puis le balayage de largeur substituee.
func e191cUnRecord(pay []byte, reg *Registry, b keyframeBorne, m *e191cEtatMesure) {
	tr := WalkKeyframeFullState(pay, b.Bit, reg, contexteDInstrument())
	if tr.DesyncAt < 0 {
		m.Portee[e191cLargeurPortee(pay, reg, b.Bit)]++
		if tr.EndBit == b.Want {
			m.FermePor++
		}
	}
	for w := 0; w <= e191cBalayageMax; w++ {
		tem := keyframeFullStateTemoin{EnTeteBits: keyframeFullStateHeaderBits + w, SansEtatParDefaut: true}
		st := walkKeyframeFullState(pay, b.Bit, reg, contexteDInstrument(), tem)
		if st.DesyncAt < 0 && st.EndBit == b.Want {
			m.ParW[w]++
		}
	}
}

// e191cLargeurPortee rend le nombre de bits que le deserialiseur d etat par defaut du depot
// consomme sur ce record : la difference entre les deux marches, l une avec l etat par defaut,
// l autre avec un decalage nul a sa place.
func e191cLargeurPortee(pay []byte, reg *Registry, bit int) int {
	br := LecteurSur(pay)
	br.SetBitPos(bit + keyframeFullStateHeaderBits)
	ti := uint32(kfReadBits(pay, bit+keyframeRecordTIBit, 6)) //nolint:gosec // 6 bits
	avant := br.BitPos()
	br.ReadBits(keyframeFullStateSizeBits)
	debut := br.BitPos()
	consumeKeyframeDefaultState(br, ti)
	_ = avant
	return br.BitPos() - debut
}

// e191cTopLargeurs colle les cinq largeurs les plus frequentes d un histogramme.
func e191cTopLargeurs(m map[int]int) string {
	type kv struct{ w, n int }
	l := make([]kv, 0, len(m))
	for w, n := range m {
		l = append(l, kv{w, n})
	}
	sort.Slice(l, func(i, j int) bool {
		if l[i].n != l[j].n {
			return l[i].n > l[j].n
		}
		return l[i].w < l[j].w
	})
	out := ""
	for i, e := range l {
		if i >= 5 {
			break
		}
		if i > 0 {
			out += " "
		}
		out += e191cPaire(e.w, e.n)
	}
	if out == "" {
		return "(aucune)"
	}
	return out
}

// e191cPaire formate une paire largeur/compte.
func e191cPaire(w, n int) string { return fmt.Sprintf("w=%d x%d", w, n) }
