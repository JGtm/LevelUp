//go:build research

package grammar

// lp_temoin_research_test.go — LOT LP DE LA CAMPAGNE DE GRAMMAIRE : L IMAGE-CLE JUGEE PAR LE BLOC DE
// TYPE 1 QUI LA PRECEDE (`.ai/V7.5/film_re/campagne_grammaire_2026-10-01/LOT_LP.md`).
//
// Chaque test lit la marche d image-cle DU CONTEXTE ([FilmContext.MarcheDImageCle]) et la marche de
// trames ([FilmContext.Trames]) de l arbre compile : joue sur la tete, il decrit la marche de
// reference ; joue sous la surcouche du lot (`-overlay`), la marche sous temoin. Les deux sorties se
// comparent hors ligne. Le bloc de type 1 est lu ici par [LireBlocDeDatums], sans rien du lot.
//
//	TestLPTemoins     lp_paquets.tsv (paquets d image-cle par bloc qui les precede) et
//	                  lp_records.tsv (ancres declarees, classees par l entree de leur slot au bloc :
//	                  meme generation ou non, vivante au sens de `FUN_1408f1730` ou non, drapeaux)
//	TestLPSlots       LP_SLOTS="film:chunk:slot,..." : chronique d un slot sur [chunk-2, chunk+2]
//	TestLPBandes      LP_BANDES="film:ti:slot,..." : la bande de l archetype que la table anticipee
//	                  donne ([TableAnticipee.SlotDeLArchetype]) et ses declarations
//	TestLPPaquets     LP_PAQUETS="film:chunk:index,..." : debut de liste et records d un paquet delta
//	TestLPTetes       LP_LISTE=<fichier « film TAB chunk:index »> : le NEW de tete de chaque liste et
//	                  l entree de son slot aux blocs du chunk et du chunk suivant
//
//	CAMPAGNE_RACINE=<film_chunks> CAMPAGNE_FILMS=<id,id> CAMPAGNE_SORTIE=<dir hors data> \
//	  go test -tags=research -count=1 -timeout 120m -run '^TestLPTemoins$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
)

// lpBloc : la table de datums d un bloc de type 1, nil quand il n y en a pas de lisible.
type lpBloc []DatumEntry

// lpLireBloc lit un bloc de type 1.
func lpLireBloc(pay []byte) lpBloc {
	if len(pay) == 0 {
		return nil
	}
	b, err := LireBlocDeDatums(pay)
	if err != nil {
		return nil
	}
	return b.Entrees
}

// entree decrit l entree d un slot.
func (b lpBloc) entree(slot int) string {
	if b == nil || slot < 0 || slot >= len(b) {
		return "sans bloc"
	}
	return fmt.Sprintf("drapeaux %#x g%d", b[slot].Drapeaux, b[slot].Gen)
}

// lpContexte ouvre un film sous le plafond memoire de la campagne.
func lpContexte(t *testing.T, racine, id string) (*FilmContext, func()) {
	garde := filmproc.Arm("campagne/lp", 4, func(pic uint64) {
		fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets)\n", id, pic)
		os.Exit(3)
	})
	fc, _, _ := ContexteDeFilm(racine + "/" + id)
	if fc == nil {
		t.Errorf("%s : film illisible", id)
	}
	return fc, garde.Disarm
}

// lpListe lit une liste « film:a:b » separee par des virgules.
func lpListe(env string) [][3]string {
	var out [][3]string
	for _, x := range strings.Split(os.Getenv(env), ",") {
		if p := strings.Split(strings.TrimSpace(x), ":"); len(p) == 3 {
			out = append(out, [3]string{p[0], p[1], p[2]})
		}
	}
	return out
}

// lpEntier lit un entier d une liste.
func lpEntier(s string) int {
	var n int
	_, _ = fmt.Sscanf(s, "%d", &n)
	return n
}

// TestLPTemoins ecrit lp_paquets.tsv et lp_records.tsv pour les films de CAMPAGNE_FILMS.
func TestLPTemoins(t *testing.T) {
	racine, sortie, films := b2Env(t)
	paquets := []string{"film\tclasse\tpaquets"}
	records := []string{"film\tclasse\trecords"}
	for _, id := range films {
		fc, fin := lpContexte(t, racine, id)
		if fc != nil {
			p, r := lpUnFilm(fc, id)
			paquets, records = append(paquets, p...), append(records, r...)
		}
		fin()
	}
	b2Ecrire(t, sortie, "lp_paquets.tsv", paquets)
	b2Ecrire(t, sortie, "lp_records.tsv", records)
}

// lpUnFilm rend les lignes d un film.
func lpUnFilm(fc *FilmContext, id string) (paquets, records []string) {
	marche := fc.MarcheDImageCle()
	cp, cr := map[string]int{}, map[string]int{}
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		var bloc lpBloc
		lu, juste := false, false
		for _, pk := range pks {
			if pk.Type == PacketTypeDatums {
				bloc, lu, juste = lpLireBloc(pk.Payload(data)), true, true
				continue
			}
			if pk.Type != PacketTypeKeyframe {
				juste = false
				continue
			}
			cp[lpClassePaquet(lu, juste, bloc)]++
			juste = false
			for _, r := range marche.Records(pk.Payload(data)) {
				cr[lpClasse(bloc, r)]++
			}
		}
	}
	for k, n := range cp {
		paquets = append(paquets, fmt.Sprintf("%s\t%s\t%d", id, k, n))
	}
	for k, n := range cr {
		records = append(records, fmt.Sprintf("%s\t%s\t%d", id, k, n))
	}
	return paquets, records
}

// lpClassePaquet classe un paquet d image-cle par le bloc qui le precede dans son chunk.
func lpClassePaquet(lu, juste bool, bloc lpBloc) string {
	switch {
	case !lu:
		return "sans bloc avant lui"
	case bloc == nil:
		return "bloc illisible"
	case juste:
		return "bloc juste avant"
	}
	return "bloc plus haut dans le chunk"
}

// lpClasse classe une ancre par l entree de son slot au bloc.
func lpClasse(bloc lpBloc, r KeyframeRec) string {
	if bloc == nil {
		return "sans bloc"
	}
	if r.Slot >= len(bloc) {
		return "hors du bloc"
	}
	e := bloc[r.Slot]
	etat, gen := "non vivante", "meme generation"
	if e.Vivante() {
		etat = "vivante"
	}
	if int(e.Gen) != r.Gen {
		gen = "autre generation"
	}
	return fmt.Sprintf("%s · %s · drapeaux %#x", gen, etat, e.Drapeaux)
}

// TestLPSlots ecrit lp_slots.tsv : la chronique de chaque slot de LP_SLOTS="film:chunk:slot,...".
func TestLPSlots(t *testing.T) {
	racine, sortie, _ := b2Env(t)
	out := []string{"film\tslot\tchunk\timage_cle\tbloc\tancres\ttable_a_position_libre\tanticipee"}
	for _, x := range lpListe("LP_SLOTS") {
		fc, fin := lpContexte(t, racine, x[0])
		if fc != nil {
			out = append(out, lpUnSlot(fc, x[0], lpEntier(x[1]), lpEntier(x[2]))...)
		}
		fin()
	}
	b2Ecrire(t, sortie, "lp_slots.tsv", out)
}

// lpUnSlot rend la chronique d un slot nomme.
func lpUnSlot(fc *FilmContext, id string, chunk, slot int) []string {
	marche := fc.MarcheDImageCle()
	ta := ConstruireTableAnticipee(fc)
	var anticipee []string
	for g := range 4 {
		if ti, decl, ok := ta.ArchetypeApres(uint32(g)<<30|uint32(slot), chunk-3); ok { //nolint:gosec // slot < 8192
			anticipee = append(anticipee, fmt.Sprintf("g%d ti%d (chunk %d)", g, ti, decl))
		}
	}
	var out []string
	for c := chunk - 2; c <= chunk+2; c++ {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		var bloc lpBloc
		k := 0
		for _, pk := range pks {
			if pk.Type == PacketTypeDatums {
				bloc = lpLireBloc(pk.Payload(data))
			}
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			table, _ := TableDeDatums(pay)
			ti, porte := table[uint32(slot)] //nolint:gosec // slot < 8192
			tab := "-"
			if porte {
				tab = fmt.Sprintf("ti%d", ti)
			}
			out = append(out, fmt.Sprintf("%s\t%d\t%d\t%d\t%s\t%s\t%s\t%s", id, slot, c, k, bloc.entree(slot),
				lpAncres(marche.Records(pay), slot), tab, strings.Join(anticipee, " ; ")))
			k++
		}
	}
	return out
}

// lpAncres decrit les ancres d un slot.
func lpAncres(recs []KeyframeRec, slot int) string {
	var s []string
	for _, r := range recs {
		if r.Slot == slot {
			s = append(s, fmt.Sprintf("g%d ti%d @%d", r.Gen, r.TI, r.Bit))
		}
	}
	if len(s) == 0 {
		return "-"
	}
	return strings.Join(s, " ; ")
}

// TestLPBandes ecrit lp_bandes.tsv pour chaque « film:ti:slot » de LP_BANDES : les bornes de la
// bande de l archetype `ti`, si `slot` y est, et chaque declaration de `ti` (ou d un slot voisin de
// `slot`) avec l entree du bloc qui la juge.
func TestLPBandes(t *testing.T) {
	racine, sortie, _ := b2Env(t)
	out := []string{"film\tti\tslot\tligne"}
	for _, x := range lpListe("LP_BANDES") {
		fc, fin := lpContexte(t, racine, x[0])
		if fc != nil {
			out = append(out, lpUneBande(fc, x[0], uint32(lpEntier(x[1])), uint32(lpEntier(x[2])))...) //nolint:gosec // ti < 50, slot < 8192
		}
		fin()
	}
	b2Ecrire(t, sortie, "lp_bandes.tsv", out)
}

// lpUneBande rend les lignes d une bande.
func lpUneBande(fc *FilmContext, id string, ti, slot uint32) []string {
	marche := fc.MarcheDImageCle()
	ta := ConstruireTableAnticipee(fc)
	var out []string
	vus := map[uint32]bool{}
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		var bloc lpBloc
		for _, pk := range pks {
			if pk.Type == PacketTypeDatums {
				bloc = lpLireBloc(pk.Payload(data))
			}
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range marche.Records(pk.Payload(data)) {
				s, x := uint32(r.Slot), uint32(r.TI) //nolint:gosec // slot < 8192, ti < 50
				if x == ti {
					vus[s] = true
				}
				if x == ti || (s+16 >= slot && slot+16 >= s) {
					out = append(out, fmt.Sprintf("%s\t%d\t%d\tdeclaration chunk %d slot %d g%d ti%d @%d bloc %s", id, ti,
						slot, c, r.Slot, r.Gen, r.TI, r.Bit, bloc.entree(r.Slot)))
				}
			}
		}
	}
	lo, hi := uint32(1<<31), uint32(0)
	for s := range filledSlotMap(vus) {
		lo, hi = min(lo, s), max(hi, s)
	}
	return append(out, fmt.Sprintf("%s\t%d\t%d\tbande : bornes %d..%d, slot dans la bande : %v", id, ti, slot, lo, hi,
		ta.SlotDeLArchetype(slot, ti)))
}

// lpCibles lit les paquets nommes « film:chunk:index » de LP_PAQUETS, ou ceux du fichier LP_LISTE
// (« film TAB chunk:index » par ligne).
func lpCibles(env string) map[string]map[[2]int]bool {
	out := map[string]map[[2]int]bool{}
	ajouter := func(id string, c, i int) {
		if out[id] == nil {
			out[id] = map[[2]int]bool{}
		}
		out[id][[2]int{c, i}] = true
	}
	if env == "LP_LISTE" {
		brut, err := os.ReadFile(os.Getenv("LP_LISTE"))
		if err != nil {
			return out
		}
		for _, l := range strings.Split(string(brut), "\n") {
			if p := strings.FieldsFunc(l, func(r rune) bool { return r == '\t' || r == ':' }); len(p) == 3 {
				ajouter(p[0], lpEntier(p[1]), lpEntier(p[2]))
			}
		}
		return out
	}
	for _, x := range lpListe(env) {
		ajouter(x[0], lpEntier(x[1]), lpEntier(x[2]))
	}
	return out
}

// TestLPPaquets ecrit lp_paquets_traces.tsv : pour chaque paquet delta de LP_PAQUETS, le debut de
// sa liste ([lecture.DebutDeVueB]) et les records lus (slot, generation, archetype, genre, liaison).
func TestLPPaquets(t *testing.T) {
	racine, sortie, _ := b2Env(t)
	out := []string{"film\tpaquet\tdebut\trecords"}
	for id, paquets := range lpCibles("LP_PAQUETS") {
		fc, fin := lpContexte(t, racine, id)
		if fc != nil {
			for p, err := range fc.Trames(nil) {
				if err != nil || !paquets[[2]int{p.Chunk, p.Index}] {
					continue
				}
				var recs []string
				for _, r := range p.Records {
					recs = append(recs, fmt.Sprintf("%d g%d ti%d genre%d liaison%d", r.Vie.Slot, r.Vie.Gen, r.TI, r.Genre, r.Liaison))
				}
				out = append(out, fmt.Sprintf("%s\t%d:%d\t%d\t%s", id, p.Chunk, p.Index, p.Debut, strings.Join(recs, " ; ")))
			}
		}
		fin()
	}
	b2Ecrire(t, sortie, "lp_paquets_traces.tsv", out)
}

// TestLPTetes ecrit lp_tetes.tsv : pour chaque paquet de LP_LISTE, le record NEW de tete de sa liste
// et l entree de son slot aux blocs de type 1 de son chunk et du chunk suivant. Une creation du jeu
// porte la generation qui suit celle du slot (`FUN_142f2e598`) ; le bloc suivant montre le slot sous
// cette generation (alloue ou libere), sauf allocations successives entre les deux images-cles.
func TestLPTetes(t *testing.T) {
	racine, sortie, _ := b2Env(t)
	out := []string{"film\tpaquet\tdebut\ttete\tbloc_courant\tbloc_suivant\tverdict"}
	for id, paquets := range lpCibles("LP_LISTE") {
		fc, fin := lpContexte(t, racine, id)
		if fc != nil {
			out = append(out, lpTetesDuFilm(fc, id, paquets)...)
		}
		fin()
	}
	b2Ecrire(t, sortie, "lp_tetes.tsv", out)
}

// lpTetesDuFilm rend les lignes d un film.
func lpTetesDuFilm(fc *FilmContext, id string, paquets map[[2]int]bool) []string {
	blocs := map[int]lpBloc{}
	for _, c := range fc.ChunkNumbers() {
		if data, pks, ok := fc.ChunkAt(c); ok {
			for _, pk := range pks {
				if pk.Type == PacketTypeDatums && blocs[c] == nil {
					blocs[c] = lpLireBloc(pk.Payload(data))
				}
			}
		}
	}
	var out []string
	for p, err := range fc.Trames(nil) {
		if err != nil || !paquets[[2]int{p.Chunk, p.Index}] {
			continue
		}
		tete, cour, suiv, verdict := "-", "-", "-", "sans record NEW de tete"
		if len(p.Records) > 0 && p.Records[0].Genre == lecture.GenreNeuf {
			r := p.Records[0]
			slot := int(r.Vie.Slot)
			tete = fmt.Sprintf("%d g%d ti%d", slot, r.Vie.Gen, r.TI)
			cour, suiv = blocs[p.Chunk].entree(slot), blocs[p.Chunk+1].entree(slot)
			verdict = lpVerdict(blocs[p.Chunk+1], slot, int(r.Vie.Gen))
		}
		out = append(out, fmt.Sprintf("%s\t%d:%d\t%d\t%s\t%s\t%s\t%s", id, p.Chunk, p.Index, p.Debut, tete, cour, suiv, verdict))
	}
	return out
}

// lpVerdict classe le slot d une creation (slot, gen) au bloc du chunk suivant.
func lpVerdict(suivant lpBloc, slot, gen int) string {
	switch {
	case suivant == nil || slot >= len(suivant):
		return "pas de bloc suivant"
	case int(suivant[slot].Gen) == gen && suivant[slot].Drapeaux&DatumAlloue != 0:
		return "bloc suivant : meme generation, allouee"
	case int(suivant[slot].Gen) == gen:
		return "bloc suivant : meme generation, liberee"
	}
	return "bloc suivant : autre generation"
}

// TestLPDebuts ecrit lp_debuts.tsv : par film de CAMPAGNE_FILMS, le nombre de paquets delta par debut
// de liste ([lecture.DebutDeVueB]) ; le rang [lecture.DebutParFermetureAuBit] est le compte du repli
// `repli_debut_de_liste_ferme_au_bit`.
func TestLPDebuts(t *testing.T) {
	racine, sortie, films := b2Env(t)
	out := []string{"film\tdebut\tpaquets"}
	for _, id := range films {
		fc, fin := lpContexte(t, racine, id)
		n := map[lecture.DebutDeVueB]int{}
		if fc != nil {
			for p, err := range fc.Trames(nil) {
				if err == nil {
					n[p.Debut]++
				}
			}
		}
		fin()
		for d, k := range n {
			out = append(out, fmt.Sprintf("%s\t%d\t%d", id, d, k))
		}
	}
	b2Ecrire(t, sortie, "lp_debuts.tsv", out)
}
