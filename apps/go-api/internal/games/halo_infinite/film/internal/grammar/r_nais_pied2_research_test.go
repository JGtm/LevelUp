//go:build research

package grammar

// r_nais_pied2_research_test.go — CHANTIER « NAIS », R-L1 (b) : UNE FERMETURE APRES REJET
// SIGNE-T-ELLE LA VRAIE FIN DE LA VUE B ?
//
// Les queues de bits des 4 598 paquets fermes apres un rejet (`r_nais_pied.tsv`) montrent, AVANT
// l en-tete rejete, des motifs d entrees de vue C (`1 00 0 iiiii 1 ...`). Deux mesures, chacune
// avec son temoin :
//
//	auto-synchronisation  sur les paquets fermes APRES TERMINATEUR (la vue C y est au bon bit), la
//	                      vue C relue depuis un bit DECALE de k (1..24) ferme-t-elle encore le
//	                      paquet ? Si oui, une fermeture au bit pres ne prouve pas que la vue C a
//	                      commence au bon bit ;
//	debut anterieur       sur les paquets fermes APRES REJET, existe-t-il un bit t AVANT l en-tete
//	                      rejete, precede de `000` (le terminateur de la vue B), d ou la vue C ferme
//	                      aussi le paquet ? Ses entrees contiennent-elles, en suffixe, celles que la
//	                      marche a lues ? Temoin : la meme recherche sur les paquets fermes apres
//	                      terminateur, en amont de leur vrai terminateur.
//
//	CAMPAGNE_RACINE=<film_chunks> CAMPAGNE_FILMS=<id,id> CAMPAGNE_SORTIE=<dir hors data> \
//	  go test -tags=research -count=1 -timeout 120m -run '^TestRNaisPiedResync$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"testing"
	"time"

	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// rnVueCDepuis relit la vue C depuis `t` : ferme-t-elle le paquet, et ses index d entrees.
func rnVueCDepuis(pay []byte, cfg FrameConfig, t int) (bool, []int) {
	br := LecteurSur(pay)
	cfg.Obs = nil
	br.poserCadre(cfg)
	br.SetBitPos(t)
	c := consumeVueC(br, len(pay)*8)
	if !c.Porte || !vueCFermee(pay, br.BitPos()) {
		return false, nil
	}
	idx := make([]int, len(c.Entrees))
	for i, e := range c.Entrees {
		idx[i] = e.Index
	}
	return true, idx
}

// rnSuffixe : `b` est un suffixe de `a`.
func rnSuffixe(a, b []int) bool {
	if len(b) > len(a) {
		return false
	}
	for i := range b {
		if a[len(a)-len(b)+i] != b[i] {
			return false
		}
	}
	return true
}

// rnPied2 ecoute la marche de reference.
type rnPied2 struct {
	f   *cmFilm
	chk *cmCollecteur
	t   cmTables
}

func (x *rnPied2) debutDeChunk(c int, _ []byte, _ []FilmPacket, _ *World) { x.chk.chunk = c }
func (x *rnPied2) finDeFilm()                                             {}

func (x *rnPied2) paquet(_ int, p *cmPaquet, _ *World) {
	if !p.d.Fermee || p.d.DebutVueB < 0 {
		return
	}
	juge := "contredit"
	if len(cmContredit(x.chk, p)) == 0 {
		juge = "sain"
	}
	var lus []int
	for _, e := range p.d.VueC.Entrees {
		lus = append(lus, e.Index)
	}
	switch {
	case p.d.Sortie == SortieVueBTerminateur && p.d.Index%16 == 0: // temoin echantillonne : un paquet sur 16
		n := 0
		for k := 1; k <= 24; k++ {
			if ok, _ := rnVueCDepuis(p.pay, x.f.cfg, p.d.FinVueB+k); ok {
				n++
			}
		}
		x.t.un("auto_sync_apres_terminateur", cmJoindre(juge, fmt.Sprintf("decalages qui ferment : %s", rnClasseN(n))))
		x.t.un("debut_anterieur_apres_terminateur", cmJoindre(juge, x.anterieur(p, p.d.FinVueB-3, lus)))
	case p.d.Sortie.EstUnRejet():
		x.t.un("debut_anterieur_apres_rejet", cmJoindre(juge, x.anterieur(p, p.d.FinVueB-rnLargeurEnTete(x.f.cfg), lus)))
	}
}

// anterieur cherche, avant `borne`, le bit t le plus proche de la borne precede de `000` d ou la
// vue C ferme le paquet, et compare ses entrees a celles que la marche a lues.
func (x *rnPied2) anterieur(p *cmPaquet, borne int, lus []int) string {
	for t := borne - 1; t >= p.d.DebutVueB+3; t-- {
		if source.BitsTolerants(p.pay, t-3, 3) != 0 {
			continue
		}
		ok, idx := rnVueCDepuis(p.pay, x.f.cfg, t)
		if !ok {
			continue
		}
		rel := "entrees de la marche en suffixe"
		if !rnSuffixe(idx, lus) {
			rel = "entrees de la marche PAS en suffixe"
		}
		return cmJoindre("debut anterieur qui ferme", rnClasseDecalage(borne-t), rel,
			fmt.Sprintf("entrees en plus : %s", rnClasseN(len(idx)-len(lus))))
	}
	return "aucun debut anterieur qui ferme"
}

func rnClasseN(n int) string {
	switch {
	case n < 0:
		return "<0"
	case n <= 2:
		return fmt.Sprint(n)
	case n <= 8:
		return "3-8"
	}
	return ">8"
}

// TestRNaisPiedResync joue les deux mesures sur les films de CAMPAGNE_FILMS.
func TestRNaisPiedResync(t *testing.T) {
	racine, sortie, films := b2Env(t)
	utiles := cmUtiles(t)
	out := []string{"film\tbuild\ttable\tcle\tn"}
	for _, id := range films {
		func() {
			garde := filmproc.Arm("r_nais/pied2", 4, func(pic uint64) {
				fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets)\n", id, pic)
				os.Exit(3)
			})
			defer garde.Disarm()
			debut := time.Now()
			f, ok := cmOuvrir(t, racine, id, utiles)
			if !ok {
				return
			}
			b := cmLireBlocs(f)
			x := &rnPied2{f: f, chk: nouveauCollecteur(f, b, nil), t: cmTables{}}
			cmMarcher(f, cmVariante{}, x)
			for nom, m := range x.t {
				for _, k := range cmCles(m) {
					out = append(out, rnTab(id, f.build, nom, k, m[k].n))
				}
			}
			t.Logf("%s %s : pic %d Mio, %s", id, f.build, garde.Peak()>>20, time.Since(debut).Round(time.Second))
		}()
	}
	b2Ecrire(t, sortie, "r_nais_pied_resync.tsv", out)
}
