//go:build research

package grammar

// lt_tete_research_test.go — SONDE DU LOT LT (campagne de grammaire, vague 2) : la tete des
// listes d evenements face a la regle du masque de l ecrivain (`FUN_142e2da44`). Un instrument :
// aucune sortie de production ne change.
//
// Trois mesures, sur la marche de la carte v2 ([cmMarcher]) ou sur celle de la cuisson :
//
//	TestLTRepliParFilm  : le compteur de production `repli_debut_de_liste_ferme_au_bit`
//	                      ([ScanMarcheDesTrames]), par film ;
//	TestLTVariantes     : la marche sous des localisateurs de tete de RECHERCHE ([ltVariantes]),
//	                      paquet par paquet (sain = ferme au sens de L0), le detail des listes
//	                      prises au second rang de la fermeture et, avec `LT_DETAIL`, les records
//	                      des paquets designes ;
//	TestLTNeufs         : les records NEW lus, par archetype, regle du masque et verdict du
//	                      paquet ; `LT_TI` garde le masque des NEW d un archetype.
//
// `LT_MPP=lead/index` impose un decoupage du bloc MPP aux films hors HI_1_12_0 / HI_1_13_0 : une
// MESURE (la largeur des formats anciens est mesuree, pas lue dans le jeu), jamais un correctif.
//
// Rejouable (un film a la fois) :
//
//	LT_RACINE=<film_chunks> LT_FILMS=<id,id> LT_SORTIE=<dir hors data> LT_VARIANTES=<v,v> \
//	  go test -tags=research -count=1 -timeout 120m -run '^TestLT' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
)

// ltEntree lit les variables de la sonde.
func ltEntree(t *testing.T) (racine, sortie string, films []string) {
	t.Helper()
	racine, sortie = os.Getenv("LT_RACINE"), os.Getenv("LT_SORTIE")
	for _, x := range strings.Split(os.Getenv("LT_FILMS"), ",") {
		if x = strings.TrimSpace(x); x != "" {
			films = append(films, x)
		}
	}
	if racine == "" || sortie == "" || len(films) == 0 {
		t.Skip("LT_RACINE, LT_FILMS et LT_SORTIE requis")
	}
	abs, _ := filepath.Abs(sortie)
	for _, seg := range strings.Split(filepath.ToSlash(abs), "/") {
		if strings.EqualFold(seg, "data") {
			t.Fatalf("sortie sous data/ : %s", abs)
		}
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		t.Fatal(err)
	}
	return racine, abs, films
}

// TestLTRepliParFilm : le compteur du repli par film, tel que la cuisson le verse.
func TestLTRepliParFilm(t *testing.T) {
	racine, sortie, films := ltEntree(t)
	lignes := []string{"film\trepli_debut_de_liste_ferme_au_bit\tlistes_par_neuf_de_tete\tpaquets_evenements\tlocalises\tnon_localises"}
	for _, id := range films {
		fc, _, _ := ContexteDeFilm(racine + "/" + id)
		if fc == nil {
			t.Errorf("%s : film illisible", id)
			continue
		}
		m, err := ScanMarcheDesTrames(fc)
		if err != nil {
			t.Errorf("%s : %v", id, err)
			continue
		}
		st := m.MovementStateStats
		lignes = append(lignes, fmt.Sprintf("%s\t%d\t%d\t%d\t%d\t%d", id, m.DebutsDeListeParRepliFermeAuBit,
			st.EventPacketsNewRecordStart, st.EventPackets, st.EventPacketsLocated, st.EventPacketsUnlocated))
	}
	ltEcrire(t, sortie, "lt_repli_par_film.tsv", lignes)
}

func ltEcrire(t *testing.T, dir, nom string, lignes []string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, nom), []byte(strings.Join(lignes, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// ltVariante : un localisateur de tete de recherche. `prod` est la marche d avant le lot LT,
// `chaine` celle du lot ([pasDEssai] refuse un masque que l ecrivain n ecrit pas).
// Toutes tiennent, comme la production, l ordre de la vue B dans la chaine ([chaineJusqua]).
//
//	chaine   : la preuve par chaine refuse un pas NEW ou delta dont le masque contredit l ecrivain ;
//	rang2    : "garder" (second rang de [debutParFermetureRangee]), "aucun" (pas de second rang),
//	           "tete" (second rang refuse un candidat dont le record de tete contredit le masque),
//	           "masque" (second rang refuse une lecture dont la premiere regle contredite est une
//	           regle de masque).
type ltVariante struct {
	nom    string
	chaine bool
	rang2  string
}

var ltVariantes = map[string]ltVariante{
	"prod":          {nom: "prod", rang2: "garder"},
	"chaine":        {nom: "chaine", chaine: true, rang2: "garder"},
	"chaine+tete":   {nom: "chaine+tete", chaine: true, rang2: "tete"},
	"chaine+aucun":  {nom: "chaine+aucun", chaine: true, rang2: "aucun"},
	"chaine+masque": {nom: "chaine+masque", chaine: true, rang2: "masque"},
	"tete":          {nom: "tete", rang2: "tete"},
}

// ltDernier : ce que le localisateur de recherche a decide pour le dernier paquet.
type ltDernier struct {
	rang       lecture.DebutDeVueB
	invariant  InvariantEcrivain // verdict de la lecture d essai au second rang
	teteMasque InvariantEcrivain
	teteTI     uint32
	teteSlot   uint32
}

// ltLocaliser est [localiserLaListe] sous la variante.
func ltLocaliser(pay []byte, w *World, cfg FrameConfig, v ltVariante, der *ltDernier) (int, lecture.DebutDeVueB) {
	*der = ltDernier{}
	debut, _ := LocaliserBoucleDeRecords(pay, w, cfg, SignatureStricte)
	if debut < 0 {
		d, rang := ltFermeture(pay, candidatsDeTete(pay, len(pay)*8, w), w, cfg, v, der)
		der.rang = rang
		return d, rang
	}
	if d, ok := ltChaine(pay, debut, candidatsDeTete(pay, debut, w), w, cfg, v); ok {
		der.rang = lecture.DebutParChaine
		return d, lecture.DebutParChaine
	}
	der.rang = lecture.DebutParSignature
	return debut, lecture.DebutParSignature
}

func ltChaine(pay []byte, debut int, candidats []int, w *World, cfg FrameConfig, v ltVariante) (int, bool) {
	extra := motFacultatifDEnTete(cfg)
	essai := cfg
	essai.Obs = nil
	for _, p := range candidats {
		if p-extra < 0 || p >= debut {
			continue
		}
		pos, ok, ordre := p-extra, true, nouvelOrdreDeLaVueB()
		for n := 0; n < plafondChaineDeTete && pos < debut && ok; n++ {
			if !suitLOrdreEn(&ordre, pay, pos, extra, essai) {
				ok = false
				break
			}
			fin, bon, masque := ltPas(pay, pos, extra, w, essai)
			if !bon || fin <= pos || (v.chaine && masque != InvariantAucun) {
				ok = false
				break
			}
			pos = fin
		}
		if ok && pos == debut && suitLOrdreEn(&ordre, pay, debut, extra, essai) {
			return p - extra, true
		}
	}
	return debut, false
}

// ltPas est [pasDEssai] qui rend aussi la regle du masque contredite par le record.
func ltPas(pay []byte, pos, extra int, w *World, essai FrameConfig) (int, bool, InvariantEcrivain) {
	br := LecteurSur(pay)
	br.poserCadre(essai)
	br.SetBitPos(pos + extra)
	if br.Remaining() < woNewHeaderBits {
		return pos, false, InvariantAucun
	}
	switch readRecordType(br) {
	case recNew:
		readRecordID(br, essai.IDLowBits, essai.IDBase)
		tr := TraverseEntity(br, w.Reg, essai.NewDefaultStateBits)
		return tr.EndBit, tr.DesyncAt == -1, tr.MasqueNonEcrit
	case recDel:
		readRecordID(br, essai.IDLowBits, essai.IDBase)
		br.Skip(32)
		return br.BitPos(), true, InvariantAucun
	case recDelta:
		rec, fin, ok := TryDeltaAt(pay, pos, w, essai)
		return fin, ok, rec.Trace.MasqueNonEcrit
	}
	return pos, false, InvariantAucun
}

// ltTete lit le record NEW de tete a `p` (sans la boucle de composants au-dela du masque).
func ltTete(pay []byte, p, extra int, w *World, cfg FrameConfig) (slot, ti uint32, masque InvariantEcrivain) {
	essai := cfg
	essai.Obs = nil
	br := LecteurSur(pay)
	br.poserCadre(essai)
	br.SetBitPos(p + extra)
	if readRecordType(br) != recNew {
		return 0, 0, InvariantAucun
	}
	id := readRecordID(br, essai.IDLowBits, essai.IDBase)
	tr := TraverseEntity(br, w.Reg, essai.NewDefaultStateBits)
	return id & 0x3fffffff, tr.TypeIndex, tr.MasqueNonEcrit
}

func ltMasque(v InvariantEcrivain) bool {
	return v == InvariantMasqueHorsArchetype || v == InvariantMasqueDenseCourt || v == InvariantMasqueEparsNonCroissant
}

func ltFermeture(pay []byte, candidats []int, w *World, cfg FrameConfig, v ltVariante, der *ltDernier) (int, lecture.DebutDeVueB) {
	extra := motFacultatifDEnTete(cfg)
	auBit := -1
	var garde ltDernier
	for _, p := range candidats {
		if p-extra < 0 {
			continue
		}
		l := lectureDEssai(pay, w, cfg, p-extra)
		if l.Fermee {
			return p - extra, lecture.DebutParFermeture
		}
		if !l.FermeeAuBit || auBit >= 0 {
			continue
		}
		slot, ti, m := ltTete(pay, p-extra, extra, w, cfg)
		switch {
		case v.rang2 == "aucun":
			continue
		case v.rang2 == "tete" && m != InvariantAucun:
			continue
		case v.rang2 == "masque" && ltMasque(l.Invariant):
			continue
		}
		auBit = p - extra
		garde = ltDernier{invariant: l.Invariant, teteMasque: m, teteTI: ti, teteSlot: slot}
	}
	if auBit < 0 {
		return -1, lecture.DebutNonLocalise
	}
	*der = garde
	return auBit, lecture.DebutParFermetureAuBit
}

// ltEcouteur garde, par paquet, le verdict et la decision de tete.
type ltEcouteur struct {
	film, variante string
	der            *ltDernier
	paquets        []string
	rang2          []string
	detail         func(c, i int) bool
	details        []string
}

func (e *ltEcouteur) debutDeChunk(int, []byte, []FilmPacket, *World) {}
func (e *ltEcouteur) finDeFilm()                                     {}
func (e *ltEcouteur) paquet(c int, p *cmPaquet, _ *World) {
	rang := lecture.DebutDeVueB(0)
	evt := p.strict != -2
	if evt {
		rang = e.der.rang
	}
	e.paquets = append(e.paquets, fmt.Sprintf("%s\t%s\t%d\t%d\t%t\t%t\t%d\t%d\t%d", e.film, e.variante, c, p.pk.Index,
		p.d.FermeeAuBit, p.d.Fermee, p.d.UtilesLus, int(rang), p.debut))
	if e.detail != nil && e.detail(c, p.pk.Index) {
		var lus []string
		for _, r := range p.recs {
			lus = append(lus, fmt.Sprintf("%d:%d:%d:%d:%d", r.Type, r.Slot, r.TypeIndex, int(r.Trace.MasqueNonEcrit), int(r.Liaison)))
		}
		e.details = append(e.details, fmt.Sprintf("%s\t%s\t%d:%d\t%d\t%d\t%t\t%s\t%d\t%#x\t%s", e.film, e.variante, c, p.pk.Index,
			int(rang), p.debut, p.d.Fermee, p.d.Cause, int(p.d.Sortie), p.d.EIDRejete, strings.Join(lus, " ")))
	}
	if evt && rang == lecture.DebutParFermetureAuBit {
		var lus []string
		for _, r := range p.recs {
			lus = append(lus, fmt.Sprintf("%d:%d:%d:%d", r.Type, r.Slot, r.TypeIndex, int(r.Trace.MasqueNonEcrit)))
		}
		e.rang2 = append(e.rang2, fmt.Sprintf("%s\t%s\t%d:%d\t%d\t%d\t%d\t%s\t%s\t%s", e.film, e.variante, c, p.pk.Index,
			p.debut, e.der.teteSlot, e.der.teteTI, e.der.teteMasque, e.der.invariant, strings.Join(lus, " ")))
	}
}

// TestLTVariantes : la marche de la carte sous chaque variante, paquet par paquet.
func TestLTVariantes(t *testing.T) {
	racine, sortie, films := ltEntree(t)
	noms := strings.Split(os.Getenv("LT_VARIANTES"), ",")
	utiles := cmUtiles(t)
	paquets := []string{"film\tvariante\tchunk\tpaquet\tferme_au_bit\tferme\tutiles\trang\tdebut"}
	var details []string
	rang2 := []string{"film\tvariante\tpaquet\tdebut\ttete_slot\ttete_ti\ttete_masque\tinvariant\trecords(type:slot:ti:masque)"}
	for _, id := range films {
		for _, nom := range noms {
			v, ok := ltVariantes[strings.TrimSpace(nom)]
			if !ok {
				t.Fatalf("variante inconnue %q", nom)
			}
			f, ok := ltOuvrir(t, racine, id, utiles)
			if !ok {
				continue
			}
			der := &ltDernier{}
			e := &ltEcouteur{film: id, variante: v.nom, der: der, detail: ltFiltreDetail(os.Getenv("LT_DETAIL"))}
			cv := cmVariante{tete: func(int) func([]byte, *World, FrameConfig) (int, bool) {
				return func(pay []byte, w *World, cfg FrameConfig) (int, bool) {
					d, _ := ltLocaliser(pay, w, cfg, v, der)
					return d, d >= 0
				}
			}}
			cmMarcher(f, cv, e)
			paquets = append(paquets, e.paquets...)
			rang2 = append(rang2, e.rang2...)
			details = append(details, e.details...)
		}
	}
	ltEcrire(t, sortie, "lt_paquets.tsv", paquets)
	ltEcrire(t, sortie, "lt_rang2.tsv", rang2)
	if len(details) > 0 {
		ltEcrire(t, sortie, "lt_details.tsv", details)
	}
}

// ltFiltreDetail lit `LT_DETAIL` = `chunk:de-a[,chunk:de-a]` ; nil sans filtre.
func ltFiltreDetail(s string) func(c, i int) bool {
	type plage struct{ c, de, a int }
	var ps []plage
	for _, x := range strings.Split(s, ",") {
		var p plage
		if _, err := fmt.Sscanf(strings.TrimSpace(x), "%d:%d-%d", &p.c, &p.de, &p.a); err == nil {
			ps = append(ps, p)
		}
	}
	if len(ps) == 0 {
		return nil
	}
	return func(c, i int) bool {
		for _, p := range ps {
			if c == p.c && i >= p.de && i <= p.a {
				return true
			}
		}
		return false
	}
}

// ltNeufs compte les records NEW lus par la marche, par (film, archetype, regle du masque,
// verdict du paquet), et garde le masque des NEW `ti` = LT_TI dont le masque contredit l ecrivain.
type ltNeufs struct {
	film    string
	ti      int
	comptes map[string]int
	masques []string
}

func (e *ltNeufs) debutDeChunk(int, []byte, []FilmPacket, *World) {}
func (e *ltNeufs) finDeFilm()                                     {}
func (e *ltNeufs) paquet(c int, p *cmPaquet, w *World) {
	for i, r := range p.recs {
		if r.Type != recNew {
			continue
		}
		n := 0
		if a, ok := w.Reg.Archetype(int(r.TypeIndex)); ok {
			n = len(a.Components)
		}
		cle := fmt.Sprintf("%s\t%d\t%d\t%d\t%t\t%t\t%t", e.film, r.TypeIndex, n, int(r.Trace.MasqueNonEcrit), r.DesyncAt == -1, p.d.FermeeAuBit, p.d.Fermee)
		e.comptes[cle]++
		if int(r.TypeIndex) == e.ti {
			var comps []string
			for _, cr := range r.Trace.Comps {
				comps = append(comps, fmt.Sprintf("%d", cr.Index))
			}
			e.masques = append(e.masques, fmt.Sprintf("%s\t%d:%d\t%d\t%d\t%d\t%#x\t%t\t%d\t%t\t%t\t%d\t%s", e.film, c, p.pk.Index, i, r.Slot,
				int(r.Trace.MasqueNonEcrit), r.Trace.Mask, r.Trace.Gate, r.Trace.DefaultBits, p.d.FermeeAuBit, p.d.Fermee, r.FinBit-r.HeaderBit, strings.Join(comps, ",")))
		}
	}
}

// TestLTNeufs : les NEW lus par la marche de production (variante `chaine`), par archetype.
func TestLTNeufs(t *testing.T) {
	racine, sortie, films := ltEntree(t)
	utiles := cmUtiles(t)
	ti := -1
	if _, err := fmt.Sscanf(os.Getenv("LT_TI"), "%d", &ti); err != nil {
		ti = -1
	}
	v := ltVariantes["chaine"]
	comptes := []string{"film\tti\tn\tmasque\ttraverse\tferme_au_bit\tferme\tnombre"}
	masques := []string{"film\tpaquet\trang\tslot\tmasque\tvaleur\tporte\tdefaut\tferme_au_bit\tferme\tbits\tcomposants"}
	for _, id := range films {
		f, ok := ltOuvrir(t, racine, id, utiles)
		if !ok {
			continue
		}
		e := &ltNeufs{film: id, ti: ti, comptes: map[string]int{}}
		der := &ltDernier{}
		cv := cmVariante{tete: func(int) func([]byte, *World, FrameConfig) (int, bool) {
			return func(pay []byte, w *World, cfg FrameConfig) (int, bool) {
				d, _ := ltLocaliser(pay, w, cfg, v, der)
				return d, d >= 0
			}
		}}
		cmMarcher(f, cv, e)
		for k, n := range e.comptes {
			comptes = append(comptes, fmt.Sprintf("%s\t%d", k, n))
		}
		masques = append(masques, e.masques...)
	}
	ltEcrire(t, sortie, "lt_neufs.tsv", comptes)
	ltEcrire(t, sortie, "lt_neufs_masques.tsv", masques)
}

// ltOuvrir est [cmOuvrir] ; `LT_MPP=8/3` impose ce decoupage du bloc MPP (mesure de recherche :
// la largeur des formats 24-25 est mesuree, pas lue dans le jeu, R_VEH.md §1.4).
func ltOuvrir(t *testing.T, racine, id string, utiles UsagesProduit) (*cmFilm, bool) {
	t.Helper()
	f, ok := cmOuvrir(t, racine, id, utiles)
	if !ok {
		return f, ok
	}
	var lead, index int
	if _, err := fmt.Sscanf(os.Getenv("LT_MPP"), "%d/%d", &lead, &index); err == nil && f.build != "HI_1_13_0" && f.build != "HI_1_12_0" {
		f.cfg.Profil.MPP.Lead, f.cfg.Profil.MPP.Index = lead, index
	}
	return f, ok
}
