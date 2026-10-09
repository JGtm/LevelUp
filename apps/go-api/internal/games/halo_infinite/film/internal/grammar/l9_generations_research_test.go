//go:build research

package grammar

// l9_generations_research_test.go — LOT L9 DE LA CAMPAGNE DE GRAMMAIRE : LES RECORDS QUE LA MARCHE
// D'IMAGE-CLE DU FILM DECLARE, JUGES CONTRE LE BLOC DE TYPE 1 QUI PRECEDE CHAQUE IMAGE-CLE.
//
// A jouer sur la tete puis sous une surcouche (`-overlay`) des fichiers de production d'un lot, et
// a comparer. Trois tables par film :
//
//	l9_records   une ligne par record de la marche de production ([FilmContext.MarcheDImageCle])
//	             de chaque paquet d'image-cle : position, slot, generation, archetype, et le
//	             verdict du bloc sur l'identifiant (vivant sous la meme generation, sous une
//	             autre, non vivant et ses drapeaux, chunk sans bloc) ; avec L9_TABLE=1, les
//	             entrees de [TableDeDatums] de chaque paquet en plus ;
//	l9_temoin    sur la premiere image-cle de chaque chunk, par classe d'entree du bloc et tranche
//	             de slot : combien d'entrees portent l'en-tete exact `[eid][field 0][ti < 50]`
//	             quelque part dans le payload (meme critere que [b3EnTeteDImageCle]), et combien
//	             la marche declare. Classes : `vivante g=<n>` ; `jamais-alloue` (drapeaux 0,
//	             generation 0) est le TEMOIN NEGATIF : la frequence a laquelle l'identifiant de
//	             generation 0 d'un slot que le jeu n'a jamais alloue se trouve dans le payload ;
//	l9_gen0      chaque entree vivante de generation 0 : les positions de son en-tete exact et
//	             les records declares qui les encadrent.
//
//	CAMPAGNE_RACINE=<film_chunks> CAMPAGNE_FILMS=<id,id> CAMPAGNE_SORTIE=<dir hors data> \
//	  go test -tags=research -count=1 -timeout 120m -run '^TestL9Generations$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/
//
// TestL9Slots (L9_SLOTS="film:chunk:slot,...") instruit des slots nommes : l'entree du bloc et les
// en-tetes candidats du slot que [TableDeDatums] voit.

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// l9Sorties : les lignes des trois tables.
type l9Sorties struct{ recs, temoin, gen0 []string }

// TestL9Generations ecrit les trois tables pour les films de CAMPAGNE_FILMS.
func TestL9Generations(t *testing.T) {
	racine, sortie, films := b2Env(t)
	s := &l9Sorties{
		recs:   []string{"film\tchunk\timage_cle\tbit\tslot\tgen\tti\tbloc"},
		temoin: []string{"film\tclasse\tentrees\ten_tete_present\tdeclarees\tdeclarees_a_en_tete_present"},
		gen0:   []string{"film\tchunk\tslot\tpositions_en_tete\trecord_declare_avant\trecord_declare_apres"},
	}
	for _, id := range films {
		l9UnFilm(t, racine, id, s)
	}
	b2Ecrire(t, sortie, "l9_records.tsv", s.recs)
	b2Ecrire(t, sortie, "l9_temoin.tsv", s.temoin)
	b2Ecrire(t, sortie, "l9_gen0.tsv", s.gen0)
}

// l9Compte : le temoin d'une classe d'entrees du bloc.
type l9Compte struct{ entrees, presents, declarees, declareesPresentes int }

// l9UnFilm ajoute les lignes d'un film.
func l9UnFilm(t *testing.T, racine, id string, s *l9Sorties) {
	garde := filmproc.Arm("campagne/l9", 4, func(pic uint64) {
		fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets)\n", id, pic)
		os.Exit(3)
	})
	defer garde.Disarm()
	fc, _, _ := ContexteDeFilm(racine + "/" + id)
	if fc == nil {
		t.Errorf("%s : film illisible", id)
		return
	}
	marche := fc.MarcheDImageCle()
	classes := map[string]*l9Compte{}
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		var bloc BlocDeDatums
		aBloc, k := false, 0
		for _, pk := range pks {
			if pk.Type == PacketTypeDatums {
				bl, err := LireBlocDeDatums(pk.Payload(data))
				bloc, aBloc = bl, err == nil
			}
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			recs := marche.Records(pay)
			s.ajouterRecords(id, c, k, pay, recs, bloc, aBloc)
			if k == 0 && aBloc {
				l9Temoin(classes, bloc, l9EnTetesExacts(pay), recs)
				s.gen0 = append(s.gen0, l9Gen0(id, c, pay, recs, bloc)...)
			}
			k++
		}
	}
	for nom, n := range classes {
		s.temoin = append(s.temoin, fmt.Sprintf("%s\t%s\t%d\t%d\t%d\t%d", id, nom, n.entrees, n.presents,
			n.declarees, n.declareesPresentes))
	}
	t.Logf("%s : pic %d Mio", id, garde.Peak()>>20)
}

// ajouterRecords ajoute les records d'un paquet (et, sous L9_TABLE, sa table de datums).
func (s *l9Sorties) ajouterRecords(id string, c, k int, pay []byte, recs []KeyframeRec, bloc BlocDeDatums,
	aBloc bool) {
	for _, r := range recs {
		s.recs = append(s.recs, fmt.Sprintf("%s\t%d\t%d\t%d\t%d\t%d\t%d\t%s", id, c, k, r.Bit, r.Slot, r.Gen, r.TI,
			l9Verdict(bloc, aBloc, r.Slot, r.Gen)))
	}
	if os.Getenv("L9_TABLE") == "" {
		return
	}
	table, _ := TableDeDatums(pay)
	for slot, ti := range table {
		s.recs = append(s.recs, fmt.Sprintf("%s\t%d\t%d\ttable\t%d\t-\t%d\t-", id, c, k, slot, ti))
	}
}

// l9Verdict juge un identifiant contre le bloc qui precede l'image-cle.
func l9Verdict(bloc BlocDeDatums, aBloc bool, slot, gen int) string {
	switch {
	case !aBloc:
		return "sans bloc"
	case slot >= len(bloc.Entrees):
		return "hors du bloc"
	case !bloc.Entrees[slot].Vivante() && int(bloc.Entrees[slot].Gen) == gen:
		return fmt.Sprintf("non vivant sous la meme generation (drapeaux %#x)", bloc.Entrees[slot].Drapeaux)
	case !bloc.Entrees[slot].Vivante():
		return fmt.Sprintf("non vivant sous une autre generation (drapeaux %#x)", bloc.Entrees[slot].Drapeaux)
	case int(bloc.Entrees[slot].Gen) != gen:
		return "vivant sous une autre generation"
	}
	return "vivant sous la meme generation"
}

// l9Temoin compte, par classe d'entree du bloc et tranche de slot, les entrees, celles dont
// l'en-tete exact est dans le payload (`exacts`), et celles que la marche declare.
func l9Temoin(classes map[string]*l9Compte, bloc BlocDeDatums, exacts map[uint32]bool, recs []KeyframeRec) {
	declares := map[uint32]bool{}
	for _, r := range recs {
		declares[uint32(r.Gen)<<30|uint32(r.Slot)] = true //nolint:gosec // gen < 4, slot < 8192
	}
	for slot, e := range bloc.Entrees {
		var cle string
		switch {
		case e.Vivante():
			cle = fmt.Sprintf("vivante g=%d", e.Gen)
		case e.Drapeaux == 0 && e.Gen == 0:
			cle = "jamais-alloue"
		default:
			continue
		}
		cle += " · slot " + b3Tranche(slot)
		if classes[cle] == nil {
			classes[cle] = &l9Compte{}
		}
		n := classes[cle]
		eid := uint32(e.Gen)<<30 | uint32(slot) //nolint:gosec // slot < 8192
		n.entrees++
		if exacts[eid] {
			n.presents++
		}
		if declares[eid] {
			n.declarees++
			if exacts[eid] {
				n.declareesPresentes++
			}
		}
	}
}

// l9EnTetesExacts rend, en UNE passe sur toutes les positions de bit du payload, les identifiants
// suivis d'un mot d'archetype sous le cap objet : l'ensemble des eid dont [b3EnTeteDImageCle]
// rendrait 2 (meme critere, sans rebalayer le payload pour chaque eid).
func l9EnTetesExacts(pay []byte) map[uint32]bool {
	out := map[uint32]bool{}
	total := len(pay) * 8
	for q := 0; q+64 <= total; q++ {
		if source.BitsBourres(pay, q+32, 32) < objectArchetypeCount {
			out[uint32(source.BitsBourres(pay, q, 32))] = true //nolint:gosec // 32 bits
		}
	}
	return out
}

// l9Gen0 rend, pour chaque entree vivante de generation 0 du bloc, les positions de son en-tete
// exact et les records declares (slot@bit) qui encadrent la premiere.
func l9Gen0(id string, c int, pay []byte, recs []KeyframeRec, bloc BlocDeDatums) []string {
	tries := append([]KeyframeRec(nil), recs...)
	sort.Slice(tries, func(i, j int) bool { return tries[i].Bit < tries[j].Bit })
	var out []string
	for slot, e := range bloc.Entrees {
		if !e.Vivante() || e.Gen != 0 {
			continue
		}
		pos := l9PositionsEnTete(pay, uint32(slot)) //nolint:gosec // slot < 8192
		avant, apres := "-", "-"
		if len(pos) > 0 {
			i := sort.Search(len(tries), func(i int) bool { return tries[i].Bit > pos[0] })
			if i > 0 {
				avant = fmt.Sprintf("%d g%d ti%d @%d", tries[i-1].Slot, tries[i-1].Gen, tries[i-1].TI, tries[i-1].Bit)
			}
			if i < len(tries) {
				apres = fmt.Sprintf("%d g%d ti%d @%d", tries[i].Slot, tries[i].Gen, tries[i].TI, tries[i].Bit)
			}
		}
		out = append(out, fmt.Sprintf("%s\t%d\t%d\t%v\t%s\t%s", id, c, slot, pos, avant, apres))
	}
	return out
}

// l9PositionsEnTete rend les positions (trois au plus) de l'en-tete exact de `eid`.
func l9PositionsEnTete(pay []byte, eid uint32) []int {
	var out []int
	total := len(pay) * 8
	for q := 0; q+64 <= total && len(out) < 3; q++ {
		if uint32(source.BitsBourres(pay, q, 32)) == eid && //nolint:gosec // 32 bits
			source.BitsBourres(pay, q+32, 32) < objectArchetypeCount {
			out = append(out, q)
		}
	}
	return out
}

// TestL9Slots ecrit `l9_slots.tsv` : pour chaque slot de L9_SLOTS="film:chunk:slot,...", l'entree
// du bloc qui precede chaque image-cle du chunk et les en-tetes candidats du slot que
// [TableDeDatums] voit (generation, archetype, position).
func TestL9Slots(t *testing.T) {
	racine, sortie, _ := b2Env(t)
	out := []string{"film\tchunk\timage_cle\tslot\tbloc_drapeaux\tbloc_gen\tcandidats"}
	for _, x := range strings.Split(os.Getenv("L9_SLOTS"), ",") {
		var id string
		var c, slot int
		if _, err := fmt.Sscanf(strings.ReplaceAll(x, ":", " "), "%s %d %d", &id, &c, &slot); err != nil {
			continue
		}
		if fc, _, _ := ContexteDeFilm(racine + "/" + id); fc != nil {
			out = append(out, l9UnSlot(fc, id, c, slot)...)
		}
	}
	b2Ecrire(t, sortie, "l9_slots.tsv", out)
}

// l9UnSlot rend les lignes d'un slot nomme.
func l9UnSlot(fc *FilmContext, id string, c, slot int) []string {
	data, pks, ok := fc.ChunkAt(c)
	if !ok {
		return nil
	}
	var out []string
	var bloc BlocDeDatums
	k := 0
	for _, pk := range pks {
		if pk.Type == PacketTypeDatums {
			bloc, _ = LireBlocDeDatums(pk.Payload(data))
		}
		if pk.Type != PacketTypeKeyframe {
			continue
		}
		var cands []string
		for _, cd := range candidatsDeDatum(pk.Payload(data)) {
			if int(cd.slot) == slot {
				cands = append(cands, fmt.Sprintf("g%d ti%d @%d", cd.gen, cd.ti, cd.bit))
			}
		}
		e := DatumEntry{}
		if slot < len(bloc.Entrees) {
			e = bloc.Entrees[slot]
		}
		out = append(out, fmt.Sprintf("%s\t%d\t%d\t%d\t%#x\t%d\t%s", id, c, k, slot, e.Drapeaux, e.Gen,
			strings.Join(cands, " ; ")))
		k++
	}
	return out
}
