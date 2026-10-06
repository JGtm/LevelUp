//go:build research

package grammar

// lr_research_test.go — SONDE DU LOT LR (campagne de grammaire, vague 3) : les listes prises au
// second rang de [debutParFermetureRangee] et ce que leur marche lit, sur la marche de la carte v2
// ([cmMarcher]). Un instrument : aucune sortie de production ne change. Elle se compile sur la base
// du lot comme sur sa tete, et se rejoue sur les deux pour comparer.
//
//	lr_par_film.tsv   par film : listes prises au second rang, records NEW traverses proprement et
//	                  DEL lus dans ces listes (ceux que la base lie ou delie, que le lot laisse) ;
//	lr_carte.tsv      ([TestLRCarte]) par paquet : ferme, records utiles fermes ;
//	lr_paquets.tsv    pour les paquets de LR_PAQUETS (`film:chunk:paquet`, separes par `;`) : le
//	                  rang du debut, chaque record, et la liaison des slots de LR_SLOTS apres le
//	                  paquet.
//
// `LR_MPP_DECLARE=1` pose sur chaque film le decoupage MPP que la grammaire resout ([lrOuvrir]),
// comme `cmd_fermeture -mpp-declare` : la marche de la sonde est alors celle du gate 2 officiel.
//
//	LR_RACINE=<film_chunks> LR_FILMS=<id,id> LR_SORTIE=<dir hors data> [LR_PAQUETS=...] \
//	  [LR_SLOTS=526,3331] [LR_MPP_DECLARE=1] go test -tags=research -count=1 -timeout 120m \
//	  -run '^TestLRSecondRang$' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
)

// lrEcouteur releve, paquet par paquet, le rang du debut que [localiserLaListe] a rendu.
type lrEcouteur struct {
	film                string
	rang                lecture.DebutDeVueB
	listes, neufs, dels int
	cibles              map[[2]int]bool
	slots               []uint32
	lignes              []string
}

func (e *lrEcouteur) debutDeChunk(int, []byte, []FilmPacket, *World) {}
func (e *lrEcouteur) finDeFilm()                                     {}

// localiser est le localisateur de production, dont le rang est garde pour l ecouteur.
func (e *lrEcouteur) localiser(int) func([]byte, *World, FrameConfig) (int, bool) {
	return func(pay []byte, w *World, cfg FrameConfig) (int, bool) {
		d, r := localiserLaListe(pay, w, cfg)
		e.rang = r
		return d, r != lecture.DebutParSignature && r != lecture.DebutNonLocalise
	}
}

func (e *lrEcouteur) paquet(c int, p *cmPaquet, w *World) {
	rang := e.rang
	e.rang = lecture.DebutEnTete
	if rang == lecture.DebutParFermetureAuBit {
		e.listes++
		for _, r := range p.recs {
			switch {
			case r.Type == recNew && r.DesyncAt == -1:
				e.neufs++
			case r.Type == recDel:
				e.dels++
			}
		}
	}
	if !e.cibles[[2]int{c, p.pk.Index}] {
		return
	}
	var liaisons []string
	for _, s := range e.slots {
		ti, lie := w.ArchetypeForSlot(s)
		liaisons = append(liaisons, fmt.Sprintf("%d:%d/%v", s, ti, lie))
	}
	e.lignes = append(e.lignes, fmt.Sprintf("%s\t%d\t%d\tdebut=%d rang=%d fermeAuBit=%v ferme=%v regle=%v\tapres: %s",
		e.film, c, p.pk.Index, p.debut, rang, p.d.FermeeAuBit, p.d.Fermee, p.d.Invariant, strings.Join(liaisons, " ")))
	for k, r := range p.recs {
		e.lignes = append(e.lignes, fmt.Sprintf("%s\t%d\t%d\trec %02d type=%d slot=%d gen=%d ti=%d bit=%d desync=%d liaison=%d",
			e.film, c, p.pk.Index, k, r.Type, r.Slot, r.ID>>30, r.TypeIndex, r.HeaderBit, r.DesyncAt, r.Liaison))
	}
}

// lrEntiers lit une liste d entiers separes par des virgules.
func lrEntiers(t *testing.T, v string) []uint32 {
	t.Helper()
	var out []uint32
	for _, x := range strings.Split(v, ",") {
		if x = strings.TrimSpace(x); x == "" {
			continue
		}
		n, err := strconv.ParseUint(x, 10, 32)
		if err != nil {
			t.Fatalf("LR_SLOTS : %q illisible", x)
		}
		out = append(out, uint32(n))
	}
	return out
}

// lrOuvrir ouvre un film comme [ltOuvrir] et, sous `LR_MPP_DECLARE=1`, lui pose le decoupage MPP
// que la grammaire resout ([FilmContext.ResolutionMPP]) quand elle le decide — la regle de
// `cmd_fermeture -mpp-declare`.
func lrOuvrir(t *testing.T, racine, id string, utiles UsagesProduit) (*cmFilm, bool) {
	t.Helper()
	f, ok := ltOuvrir(t, racine, id, utiles)
	if !ok || os.Getenv("LR_MPP_DECLARE") != "1" {
		return f, ok
	}
	if res := f.fc.ResolutionMPP(); res.Decide() {
		f.fc.PoserMPP(res.Widths)
		f.cfg = f.fc.CadreDeBalayage()
	}
	return f, ok
}

// TestLRSecondRang ecrit `lr_par_film.tsv` et `lr_paquets.tsv`.
func TestLRSecondRang(t *testing.T) {
	t.Setenv("LT_RACINE", os.Getenv("LR_RACINE"))
	t.Setenv("LT_SORTIE", os.Getenv("LR_SORTIE"))
	t.Setenv("LT_FILMS", os.Getenv("LR_FILMS"))
	racine, sortie, films := ltEntree(t)
	t.Setenv("CAMPAGNE_PAQUETS", os.Getenv("LR_PAQUETS"))
	cibles := l8Cibles(t)
	slots := lrEntiers(t, os.Getenv("LR_SLOTS"))
	utiles := cmUtiles(t)
	parFilm := []string{"film\tlistes_second_rang\tneufs_lus\tdels_lus"}
	var paquets []string
	for _, id := range films {
		f, ok := lrOuvrir(t, racine, id, utiles)
		if !ok {
			continue
		}
		e := &lrEcouteur{film: id, rang: lecture.DebutEnTete, cibles: map[[2]int]bool{}, slots: slots}
		for _, c := range cibles {
			if c.film == id {
				e.cibles[[2]int{c.chunk, c.paquet}] = true
			}
		}
		cmMarcher(f, cmVariante{tete: e.localiser}, e)
		parFilm = append(parFilm, fmt.Sprintf("%s\t%d\t%d\t%d", id, e.listes, e.neufs, e.dels))
		paquets = append(paquets, e.lignes...)
	}
	ltEcrire(t, sortie, "lr_par_film.tsv", parFilm)
	ltEcrire(t, sortie, "lr_paquets.tsv", paquets)
}

// lrCarte ecrit, paquet par paquet, le verdict de fermeture et les records utiles fermes de la
// marche de la carte ([cmMarcher]) : de quoi rejouer le gate 2 de la campagne et le comparer a la
// carte officielle, paquet par paquet.
type lrCarte struct {
	film   string
	lignes []string
}

func (e *lrCarte) debutDeChunk(int, []byte, []FilmPacket, *World) {}
func (e *lrCarte) finDeFilm()                                     {}

func (e *lrCarte) paquet(c int, p *cmPaquet, _ *World) {
	e.lignes = append(e.lignes, fmt.Sprintf("%s\t%d\t%d\t%v\t%d", e.film, c, p.pk.Index, p.d.Fermee, p.utilesFermes))
}

// TestLRCarte ecrit `lr_carte.tsv` (film, chunk, paquet, ferme, utiles fermes).
func TestLRCarte(t *testing.T) {
	t.Setenv("LT_RACINE", os.Getenv("LR_RACINE"))
	t.Setenv("LT_SORTIE", os.Getenv("LR_SORTIE"))
	t.Setenv("LT_FILMS", os.Getenv("LR_FILMS"))
	racine, sortie, films := ltEntree(t)
	utiles := cmUtiles(t)
	lignes := []string{"film\tchunk\tpaquet\tferme\tutiles_fermes"}
	for _, id := range films {
		f, ok := lrOuvrir(t, racine, id, utiles)
		if !ok {
			continue
		}
		e := &lrCarte{film: id}
		cmMarcher(f, cmVariante{}, e)
		lignes = append(lignes, e.lignes...)
	}
	ltEcrire(t, sortie, "lr_carte.tsv", lignes)
}

// lrCasReel : apres le paquet (chunk, paquet), le slot doit etre lie a `ti` (`lie`) ou ne pas
// l etre.
type lrCasReel struct {
	chunk, paquet int
	slot, ti      uint32
	lie           bool
	pourquoi      string
}

// lrCasReels porte les cas de l enquete de la representation intermediaire (lot 2.7.a0,
// 2026-10-05) sur `1c4c63c2`, sous le decoupage MPP 8/3 que le film declare : au second rang de
// la fermeture, 17:52 lisait un NEW du slot 3331 sous `ti=32`, ce qui faisait refuser le NEW
// `ti=10` de 17:172 ; 61:42 lisait un DEL du slot 526, un bipede vivant.
var lrCasReels = []lrCasReel{
	{17, 52, 3331, 32, false, "le NEW `ti=32` de 17:52 (second rang) ne lie pas"},
	{17, 172, 3331, 10, true, "le NEW `ti=10` de 17:172 se lie"},
	{61, 42, 526, 35, true, "le DEL de 61:42 (second rang) ne delie pas le bipede 526"},
}

// lrJuge verifie les cas reels au passage de la marche.
type lrJuge struct {
	t *testing.T
}

func (e *lrJuge) debutDeChunk(int, []byte, []FilmPacket, *World) {}
func (e *lrJuge) finDeFilm()                                     {}

func (e *lrJuge) paquet(c int, p *cmPaquet, w *World) {
	for _, cas := range lrCasReels {
		if cas.chunk != c || cas.paquet != p.pk.Index {
			continue
		}
		ti, lie := w.ArchetypeForSlot(cas.slot)
		if (lie && ti == cas.ti) != cas.lie {
			e.t.Errorf("%d:%d : slot %d lie=%v ti=%d — %s", c, p.pk.Index, cas.slot, lie, ti, cas.pourquoi)
		}
	}
}

// TestLRCasReelsDuSecondRang rejoue les cas reels sur `1c4c63c2` sous le decoupage MPP qu il declare
// (8/3, `LR_MPP_DECLARE=1`). ROUGE sur la base du lot (`8dfadd07e`).
func TestLRCasReelsDuSecondRang(t *testing.T) {
	racine := os.Getenv("LR_RACINE")
	if racine == "" || os.Getenv("LR_MPP_DECLARE") != "1" {
		t.Skip("LR_RACINE et LR_MPP_DECLARE=1 requis")
	}
	f, ok := lrOuvrir(t, racine, "1c4c63c2", cmUtiles(t))
	if !ok {
		t.Fatal("1c4c63c2 illisible")
	}
	cmMarcher(f, cmVariante{}, &lrJuge{t: t})
}
