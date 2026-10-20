//go:build research

package grammar

// campagne_l0_research_test.go — MESURES PREALABLES DU LOT L0 (campagne de grammaire, vague 1) :
// les regles de l ecrivain qu une lecture fermee au bit pres contredit, avant de les faire entrer
// dans la definition de la fermeture (`ecrivain_invariants.go`).
//
//	L0.6   paquets fermes au bit pres APRES une sortie de vue B par rejet, ventiles par le juge des
//	       trois invariants (sain / contredit), par l existence d un debut de vue C ANTERIEUR qui
//	       ferme aussi le paquet (la mesure de R-L1 (b), `r_nais_pied2_research_test.go`), par le
//	       reste du payload derriere l en-tete rejete (moins de 9 bits : la place minimale qu un
//	       DELTA reel laisse derriere son en-tete, corps de 5 bits, terminateurs de 3 et 1 bits) et
//	       par la classe de l eid au bloc de type 1 ;
//	L0.7   records dont le masque contredit `FUN_142e2da44`, par etat du paquet, et l accord du
//	       juge relu (`campagne_invariants_research_test.go`) avec la regle lue a la traversee ;
//	L0.8   records DEL et leur mot de 32 bits, par etat du paquet (`FUN_142f304a8` n ecrit un mot
//	       non nul que pour l archetype 0x10 ; cet archetype est celui du DATUM, que le paquet ne
//	       porte pas : la ventilation par archetype a ete mesuree une fois sous instrumentation
//	       temporaire, `.ai/V7.5/film_re/campagne_grammaire_2026-10-01/LOT_L0.md`).
//
// La marche est celle de la carte (`cmMarcher`), sous la definition de la fermeture en vigueur ;
// l etat d un paquet se lit sur `FermeeAuBit`, jamais sur `Fermee`. La mesure prealable du lot L0
// a ete jouee definition neutralisee (verdict « ferme » = ferme au bit pres, cf. `LOT_L0.md`).
//
//	CAMPAGNE_RACINE=<film_chunks> CAMPAGNE_FILMS=<id,id> CAMPAGNE_SORTIE=<dir hors data> \
//	  go test -tags=research -count=1 -timeout 120m -run '^TestCampagneL0Mesures$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/filmproc"
)

// l0Mesure ecoute la marche d un film.
type l0Mesure struct {
	f   *cmFilm
	b   *cmBlocs
	chk *cmCollecteur
	rn  *rnPied2
	t   cmTables
	// paquets : une ligne par paquet ferme au bit pres apres rejet, sans debut anterieur.
	paquets []string
}

func (x *l0Mesure) debutDeChunk(c int, _ []byte, _ []FilmPacket, _ *World) {
	x.chk.chunk, x.rn.chk.chunk = c, c
}
func (x *l0Mesure) finDeFilm() {}

// etatL0 : l etat d un paquet pour les tables du lot.
func etatL0(p *cmPaquet, juge []string) string {
	switch {
	case !p.d.FermeeAuBit:
		return "non ferme au bit"
	case len(juge) == 0:
		return "ferme au bit, sain au juge"
	}
	return "ferme au bit, contredit au juge"
}

// classeDeReste classe le reste du payload derriere l en-tete rejete.
func l0ClasseDeReste(r int) string {
	switch {
	case r < 9:
		return "reste < 9 (aucun DELTA reel ne tient)"
	case r <= 64:
		return "reste 9-64"
	}
	return "reste > 64"
}

func (x *l0Mesure) paquet(_ int, p *cmPaquet, _ *World) {
	if p.d.DebutVueB < 0 {
		return
	}
	juge := cmContredit(x.chk, p)
	etat := etatL0(p, juge)
	x.records(p, etat)
	x.accord(p, juge)
	if !p.d.FermeeAuBit || !p.d.Sortie.EstUnRejet() {
		return
	}
	var lus []int
	for _, e := range p.d.VueC.Entrees {
		lus = append(lus, e.Index)
	}
	ant := x.rn.anterieur(p, p.d.FinVueB-rnLargeurEnTete(x.f.cfg), lus)
	aAnterieur := "debut anterieur qui ferme"
	if ant == "aucun debut anterieur qui ferme" {
		aAnterieur = ant
	}
	reste := p.d.Bits - p.d.FinVueB
	classe := x.b.naissance(p.d.Chunk, p.d.EIDRejete, slices.Contains(p.d.NeufsLus, p.d.EIDRejete&0x3fffffff))
	x.t.un("l0_6_rejets_fermes", cmJoindre(etat, aAnterieur, l0ClasseDeReste(reste)))
	x.t.un("l0_6_rejets_fermes_classe", cmJoindre(etat, aAnterieur, classe))
	if aAnterieur != ant {
		return
	}
	x.paquets = append(x.paquets, rnTab(x.f.id, x.f.build, p.d.Chunk, p.d.Index, etat, reste,
		fmt.Sprintf("%#x", p.d.EIDRejete), classe, p.d.DernierLu, len(p.recs), len(p.d.VueC.Entrees)))
}

// records compte, par etat du paquet, les records NEW/DELTA a masque non ecrivable et les DEL.
func (x *l0Mesure) records(p *cmPaquet, etat string) {
	for _, mot := range l0MotsDeDel(x.f, p) {
		x.t.un("l0_8_dels", cmJoindre(etat, "DEL lus"))
		if mot != 0 {
			x.t.un("l0_8_dels", cmJoindre(etat, "DEL a mot non nul"))
		}
	}
	for _, r := range p.recs {
		if r.Type == recDel || (r.Trace.EndBit == 0 && len(r.Trace.Comps) == 0) {
			continue // un DEL n a pas de masque ; un DELTA infere n a pas lu le sien
		}
		x.t.un("l0_7_masques", cmJoindre(etat, nomDeTypeDeRecord(r.Type), "records a masque lu"))
		if r.Trace.MasqueNonEcrit != InvariantAucun {
			x.t.un("l0_7_masques", cmJoindre(etat, nomDeTypeDeRecord(r.Type), r.Trace.MasqueNonEcrit.String()))
		}
	}
}

// accord confronte, sur les paquets fermes au bit pres, le juge relu et les regles de production.
func (x *l0Mesure) accord(p *cmPaquet, juge []string) {
	if !p.d.FermeeAuBit {
		return
	}
	prod := map[string]bool{}
	for v := range NombreDInvariants {
		if p.d.Invariants&(uint32(1)<<uint(v)) == 0 {
			continue
		}
		switch InvariantEcrivain(v) {
		case InvariantOrdreVueB:
			prod["ordre"] = true
		case InvariantMasqueHorsArchetype, InvariantMasqueDenseCourt, InvariantMasqueEparsNonCroissant:
			prod["masque"] = true
		case InvariantVueCKind, InvariantVueCTropDEntrees, InvariantVueCIndex, InvariantVueCEnTete,
			InvariantVueCCodeAnalogique:
			prod["vue C"] = true
		}
	}
	for _, j := range []string{"ordre", "masque", "vue C"} {
		x.t.un("l0_accord_juge_production", cmJoindre(j, fmt.Sprintf("juge %t", slices.Contains(juge, j)),
			fmt.Sprintf("production %t", prod[j])))
	}
}

// TestCampagneL0Mesures joue les mesures prealables du lot L0 sur les films de CAMPAGNE_FILMS.
func TestCampagneL0Mesures(t *testing.T) {
	racine, sortie, films := b2Env(t)
	utiles := cmUtiles(t)
	out := []string{"film\tbuild\ttable\tcle\tn"}
	detail := []string{"film\tbuild\tchunk\tpaquet\tetat\treste_apres_entete\teid_rejete\tclasse_au_bloc\tdernier_lu\trecords\tentrees_vue_c"}
	for _, id := range films {
		func() {
			garde := filmproc.Arm("campagne/l0", 4, func(pic uint64) {
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
			x := &l0Mesure{f: f, b: b, chk: nouveauCollecteur(f, b, nil), t: cmTables{},
				rn: &rnPied2{f: f, chk: nouveauCollecteur(f, b, nil), t: cmTables{}}}
			cmMarcher(f, cmVariante{}, x)
			for nom, m := range x.t {
				for _, k := range cmCles(m) {
					out = append(out, rnTab(id, f.build, nom, k, m[k].n))
				}
			}
			detail = append(detail, x.paquets...)
			t.Logf("%s %s : pic %d Mio, %s", id, f.build, garde.Peak()>>20, time.Since(debut).Round(time.Second))
		}()
	}
	b2Ecrire(t, sortie, "l0_mesures.tsv", out)
	b2Ecrire(t, sortie, "l0_rejets_sans_debut_anterieur.tsv", detail)
}

// l0MotsDeDel relit, dans l ordre du flux, le mot de 32 bits de chaque DEL de la vue B : un record
// commence ou finit le precedent (un NEW ou un DELTA a la fin de sa trace, un DEL apres son mot).
// La relecture s arrete sur un DELTA infere, dont la fin n est pas connue.
func l0MotsDeDel(f *cmFilm, p *cmPaquet) []uint64 {
	pos, extra := p.d.DebutVueB, motFacultatifDEnTete(f.cfg)
	var out []uint64
	for _, r := range p.recs {
		switch r.Type {
		case recDel:
			br := LecteurSur(p.pay)
			br.SetBitPos(pos + extra)
			readRecordType(br)
			readRecordID(br, f.cfg.IDLowBits, f.cfg.IDBase)
			out = append(out, br.ReadBits(32))
			pos = br.BitPos()
		case recNew, recDelta:
			if r.Trace.EndBit == 0 {
				return out
			}
			pos = r.Trace.EndBit
		}
	}
	return out
}

// l0Detail ecoute la marche et decrit les paquets nommes.
type l0Detail struct {
	f      *cmFilm
	cibles map[[2]int]bool
	lignes []string
}

func (x *l0Detail) debutDeChunk(int, []byte, []FilmPacket, *World) {}
func (x *l0Detail) finDeFilm()                                     {}

func (x *l0Detail) paquet(c int, p *cmPaquet, _ *World) {
	if !x.cibles[[2]int{c, p.d.Index}] {
		return
	}
	x.lignes = append(x.lignes, fmt.Sprintf("%s %d:%d debut=%d strict=%d ferme_au_bit=%t regles=%#x sortie=%s fin_vue_b=%d bits=%d",
		x.f.id, c, p.d.Index, p.debut, p.strict, p.d.FermeeAuBit, p.d.Invariants, p.d.Sortie, p.d.FinVueB, p.d.Bits))
	for k, r := range p.recs {
		n := -1
		if a, ok := x.f.reg.Archetype(int(r.TypeIndex)); ok {
			n = len(a.Components)
		}
		x.lignes = append(x.lignes, fmt.Sprintf("  #%d %s slot=%d eid=%#x ti=%d n=%d masque=%#x %s fin=%d desync=%d",
			k, nomDeTypeDeRecord(r.Type), r.Slot, r.ID, r.TypeIndex, n, r.Trace.Mask, r.Trace.MasqueNonEcrit, r.Trace.EndBit, r.DesyncAt))
	}
}

// TestCampagneL0Paquets decrit les paquets de CAMPAGNE_PAQUETS (`film:chunk:paquet,...`).
func TestCampagneL0Paquets(t *testing.T) {
	racine, sortie, _ := b2Env(t)
	utiles := cmUtiles(t)
	parFilm := map[string]map[[2]int]bool{}
	for _, s := range strings.Split(os.Getenv("CAMPAGNE_PAQUETS"), ",") {
		var id string
		var c, i int
		if _, err := fmt.Sscanf(strings.ReplaceAll(s, ":", " "), "%s %d %d", &id, &c, &i); err != nil {
			continue
		}
		if parFilm[id] == nil {
			parFilm[id] = map[[2]int]bool{}
		}
		parFilm[id][[2]int{c, i}] = true
	}
	var out []string
	for _, id := range slices.Sorted(maps.Keys(parFilm)) {
		f, ok := cmOuvrir(t, racine, id, utiles)
		if !ok {
			continue
		}
		x := &l0Detail{f: f, cibles: parFilm[id]}
		cmMarcher(f, cmVariante{}, x)
		out = append(out, x.lignes...)
	}
	if err := os.WriteFile(sortie+"/l0_paquets_detail.txt", []byte(strings.Join(out, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}
