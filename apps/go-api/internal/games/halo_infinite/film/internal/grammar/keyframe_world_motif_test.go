package grammar

// keyframe_world_motif_test.go — LA RECHERCHE D ANCRES PAR MOTIF REND EXACTEMENT CE QUE RENDAIT LA
// BOUCLE D AVANT (2026-09-28, temps CI du paquet).
//
// [kfRecherche.suivante] lisait 32 bits a chaque position de bit de sa fenetre ; elle saute
// desormais, par [source.PremierMotif64], aux seules positions que [motifDAncre] retient, et y
// rejoue le meme test. La boucle d avant est recopiee ICI, a l identique a son nom pres
// ([kfRecherche.suivanteOracle]) : elle est l ORACLE. Chaque cas compare l issue ENTIERE
// (ancre, decision, fin, trainee, refutations, contradiction) et la liste des candidats de la
// fenetre — qui nourrit l election, la preuve et [kfRecherche.ecartes].
//
// MUTATIONS JOUEES LE 2026-09-28, TOUTES ROUGES : un bit de slot de trop dans `Nuls` de
// [motifDAncre] (des ancres sautees) ; la borne basse de slot decalee d un rang ; `if p != q`
// retire (une trainee de sentinelles qui survit a une position sautee) ; une sentinelle de trop
// comptee par [kfRecherche.sentinelles] (seul le test synthetique la voit : les bobines n ont
// pas de fin de table a cheval sur une borne).

import (
	"math/rand"
	"path/filepath"
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

func (r *kfRecherche) suivanteOracle(from, prevSlot int) kfIssue {
	iss := kfIssue{at: -1}
	best := kfCand{consecutive: -1, gen: 1 << 30, slot: 1 << 30, bit: 1 << 30}
	exact := -1
	end := min(from+r.maxWin, r.total)
	r.cands = r.cands[:0]
	sentStreak := 0
	for q := from; q < end && q+64 <= r.total; q++ {
		id := source.BitsBourres(r.buf, q, 32)
		if id == kfSent {
			if sentStreak++; sentStreak >= 2048 {
				iss.fin = true
				break
			}
			continue
		}
		sentStreak = 0
		s, ti, g, ok := kfAnchorFromID(r.buf, q, id, prevSlot, r.total)
		if !ok {
			continue
		}
		if s == prevSlot+1 && g == 1 {
			return kfIssue{at: q, dec: kfVoisin} // consécutif gen-1 : non ambigu
		}
		if exact < 0 && g == 1 && ti == BipedTypeIndex {
			exact = q
		}
		r.cands = append(r.cands, kfCandidat{gen: g, slot: s, bit: q, ti: ti})
		cand := kfCand{gen: g, slot: s, bit: q}
		if s == prevSlot+1 {
			cand.consecutive = 1
		}
		if iss.at < 0 || cand.betterThan(best) {
			iss.at, best = q, cand
		}
	}
	if iss.at >= 0 && r.preuve != nil {
		if elu, refutes := r.elire(prevSlot); elu >= 0 {
			iss.at, iss.refutes = elu, refutes
		} else {
			iss.contradictoire = true // constat DFIX-R8 : plus de retombée muette
		}
	}
	switch {
	case exact >= 0 && (iss.at < 0 || exact <= iss.at):
		// `<=` : quand l'élection retient elle-même l'en-tête exact, c'est lui qui la justifie —
		// la décision se compte en recalage, et `Elections` ne compte que les choix du repli.
		iss.at, iss.dec = exact, kfRecalage
	case iss.at >= 0:
		iss.dec = kfElection // repli nomme `repli_ancre_d_image_cle_par_election`
	default:
		iss.traine = sentStreak
	}
	return iss
}

// comparerSuivante joue la recherche et son oracle au meme point et exige la meme issue et les
// memes candidats. Les deux partagent la memoire des preuves (pure : une position, un verdict).
func comparerSuivante(t *testing.T, nom string, r *kfRecherche, from, prevSlot int) {
	t.Helper()
	want := r.suivanteOracle(from, prevSlot)
	wantCands := append([]kfCandidat(nil), r.cands...)
	got := r.suivante(from, prevSlot)
	if got != want || !reflect.DeepEqual(append([]kfCandidat(nil), r.cands...), wantCands) {
		t.Fatalf("%s, depart %d, slot precedent %d :\n  issue      %+v (%d candidats)\n  oracle     %+v (%d candidats)",
			nom, from, prevSlot, got, len(r.cands), want, len(wantCands))
	}
}

// TestMotifDAncreCouvreLesGardesDeLAncre : toute fenetre que [kfAnchorFromID] accepte sous un
// `prevSlot`, et toute sentinelle, sont retenues par [motifDAncre] de ce `prevSlot`.
func TestMotifDAncreCouvreLesGardesDeLAncre(t *testing.T) {
	rng := rand.New(rand.NewSource(0x4B46))
	retenues := 0
	for i := 0; i < 200_000; i++ {
		gen, slot := uint64(rng.Intn(4)), uint64(rng.Intn(kfTableCap+64))
		id := gen<<30 | slot
		if i%7 == 0 {
			id = uint64(rng.Uint32())
		}
		mot := uint64(rng.Intn(kfArchMax + 20))
		if i%11 == 0 {
			mot = uint64(rng.Uint32())
		}
		if i%13 == 0 {
			id = kfSent
		}
		w := &bitWriter{}
		w.bits(id, 32)
		w.bits(mot, 32)
		w.bits(0, 8)
		prev := -1
		if i%3 != 0 {
			prev = rng.Intn(kfTableCap + 16)
		}
		if i%5 == 0 { // au ras de la borne : slot = prev + 1, + 2, + 3 (frontieres de puissance de deux)
			prev = int(id&0x3FFFFFFF) - 1 - rng.Intn(3)
		}
		retenue := source.PremierMotif64(w.buf, 0, 1, motifDAncre(prev)) == 0
		_, _, _, ok := kfAnchorFromID(w.buf, 0, id, prev, len(w.buf)*8)
		if (ok || id == kfSent) && !retenue {
			t.Fatalf("id %#x, mot %#x : slot precedent %d : ancre acceptee (ok=%v) ou sentinelle, mais motif muet", id, mot, prev, ok)
		}
		if retenue {
			retenues++
		}
	}
	if retenues == 0 {
		t.Fatal("aucune fenetre retenue : le test ne mesure rien")
	}
}

// payloadSynthetiqueDAncres ecrit un payload d image-cle factice : records vrais et faux,
// zones nulles, bruit, trainees de sentinelles de longueurs variees (sous et sur 2 048).
func payloadSynthetiqueDAncres(rng *rand.Rand) []byte {
	w := &bitWriter{}
	w.bit(0)
	slot := uint64(rng.Intn(40))
	for k := 0; k < 12; k++ {
		switch rng.Intn(6) {
		case 0, 1:
			slot += uint64(1 + rng.Intn(3))
			kfEcrireRecord(w, uint64(1+rng.Intn(3)), slot, uint64(rng.Intn(kfArchMax+4)), rng.Intn(300))
		case 2:
			w.bits(0, rng.Intn(400))
		case 3:
			for i := rng.Intn(40); i > 0; i-- {
				w.bits(rng.Uint64(), 64)
			}
		case 4:
			for i := []int{3, 70, 2047, 2048, 2100}[rng.Intn(5)]; i > 0; i-- {
				w.bits(kfSent, 32)
			}
			w.bits(rng.Uint64(), rng.Intn(33))
		default:
			w.bits(uint64(rng.Intn(4))<<30|uint64(rng.Intn(kfTableCap)), 32) // fausse ancre
			w.bits(uint64(rng.Intn(64)), 32)
		}
	}
	return w.buf
}

// TestSuivanteEgaleLaBoucleDAvantSurTamponsSynthetiques : tous les departs, plusieurs slots
// precedents, plusieurs fenetres (dont des fenetres courtes, pour les frontieres de fenetre et la
// reprise sur trainee), avec et sans preuve (preuve vide : seul l ordre de l election joue).
func TestSuivanteEgaleLaBoucleDAvantSurTamponsSynthetiques(t *testing.T) {
	rng := rand.New(rand.NewSource(0x5C1E))
	cas := 0
	for n := 0; n < 12; n++ {
		buf := payloadSynthetiqueDAncres(rng)
		total := len(buf) * 8
		for _, fen := range []int{64, 500, 3000, total} {
			for _, preuve := range []*PreuveDImageCle{nil, {}} {
				r := &kfRecherche{buf: buf, total: total, maxWin: fen, preuve: preuve, prouves: map[int]bool{}}
				for from := 0; from < total; from += 1 + rng.Intn(600) {
					for _, prev := range []int{-1, rng.Intn(60), rng.Intn(kfTableCap)} {
						comparerSuivante(t, "synthetique", r, from, prev)
						cas++
					}
				}
			}
		}
	}
	if cas < 1_000 {
		t.Fatalf("%d cas seulement", cas)
	}
	t.Logf("%d cas confrontes", cas)
}

// TestSuivanteEgaleLaBoucleDAvantSurLesBobines : sur les payloads d image-cle des bobines du
// depot, aux points ou la marche de production appelle la recherche — le debut du payload, et
// l etat qui suit chaque record (bit + 64, slot du record) — sous la preuve du film. Un record sur
// `pas` : l oracle coute ce que coutait la marche d avant, et ce test tourne sous la couverture de
// la CI. Toutes les bobines par build, et les deux bobines contigues de `killsource`.
func TestSuivanteEgaleLaBoucleDAvantSurLesBobines(t *testing.T) {
	const pas = 61
	dirs := []string{
		filepath.Join("..", "facts", "killsource", "testdata", "minibobine_000d5950"),
		filepath.Join("..", "facts", "killsource", "testdata", "minibobine_e5adf7b2"),
	}
	for _, court := range closureMiniFilms() {
		dirs = append(dirs, filepath.Join("..", "..", "replay", "testdata", "minifilm_"+court))
	}
	appels := 0
	for _, dir := range dirs {
		film, err := source.LoadDir(dir, nil)
		if err != nil {
			t.Fatalf("%s : %v", dir, err)
		}
		fc := NewFilmContext(film)
		marche := fc.MarcheDImageCle()
		for i, pay := range e191cPayloads(fc) {
			r := &kfRecherche{buf: pay, total: len(pay) * 8, maxWin: kfScanFenetreBits,
				preuve: marche.preuve, prouves: map[int]bool{}}
			nom := filepath.Base(dir) + " image-cle " + string(rune('0'+i%10))
			comparerSuivante(t, nom, r, 1, -1)
			appels++
			for k, rec := range marche.Records(pay) {
				if k%pas == i%pas {
					comparerSuivante(t, nom, r, rec.Bit+64, rec.Slot)
					appels++
				}
			}
		}
	}
	if appels < 100 {
		t.Fatalf("%d appels seulement : les bobines ne portent plus d image-cle ?", appels)
	}
	t.Logf("%d appels confrontes", appels)
}
