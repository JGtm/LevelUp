//go:build research

package grammar

// campagne_bis1_research_test.go — MESURES BIS 1 DE LA CAMPAGNE DE GRAMMAIRE (2026-10-01) : les
// mesures que la critique de completude (`CRITIQUE_COMPLETUDE_1.md`, points 10 a 13, 21 a 23, 29)
// a trouvees manquantes autour du lot L1 et de l item 1.3. Un instrument de recherche : aucun
// fichier de production n est touche, aucune sortie ne change.
//
//	entrees    l item 1.3 : un denominateur des entrees de controle INDEPENDANT de la fermeture,
//	           par la regle de l ecrivain (T5 : au plus une entree par joueur) et les joueurs du
//	           film (table du film, joueurs geres `ti=9` lies au debut du chunk) ;
//	juge       chaque paquet ferme d une marche passe aux invariants de l ecrivain (ordre de la vue
//	           B, masques, vue C : [cmCollecteur.invariants]) ; les paquets GAGNES contre la
//	           reference sont ventiles en contredits / non contredits ;
//	regions    l oracle-NEW restreint a UNE region de la sonde M1 a la fois ;
//	L1a        le localisateur « bande OU bloc de type 1 » dont les candidats, la chaine et (au
//	           niveau « paquet ») le paquet decode passent les invariants de l ecrivain.
//
// Rejouable (un film a la fois, plafond 4 Gio) :
//
//	CAMPAGNE_RACINE=<film_chunks> CAMPAGNE_FILMS=<id,id> CAMPAGNE_SORTIE=<dir hors data> \
//	  go test -tags=research -count=1 -timeout 180m -run '^TestCampagneMesuresBis1$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// cmEssaiDepuis marche UN paquet depuis `debut` sur le monde `w` (que l appelant restaure) et le
// rend comme la marche de la carte le rendrait.
func cmEssaiDepuis(f *cmFilm, pay []byte, w *World, debut int) *cmPaquet {
	md := marcheDetaillee{mesureDesTrames: nouvelleMesureDesTrames(f.reg, f.utiles, f.cfg)}
	p := &cmPaquet{pay: pay, strict: -2, d: PaquetDeCarte{Bits: len(pay) * 8, DebutVueB: -1, FinVueB: -1}}
	md.vueC = LectureVueC{}
	recs, rangs, l := md.marcherParRangs(pay, w, debut, &p.d)
	enTete := debut == md.cfg.PacketPreambleBits && md.cfg.PacketPreambleBits >= 1
	pm := paquetMarche{enTete: enTete, recs: recs, rangs: rangs, vueC: l}
	md.classer(pm)
	p.d.Fermee = l.Fermee
	if !l.Fermee {
		p.d.Cause = md.bloquantDuPaquet(pm).nom
	}
	p.recs, p.debut = recs, debut
	return p
}

// cmMasqueSainEn : le record a `pos` (debut du record, mot facultatif compris) a un masque que
// l ecrivain peut ecrire. Un NEW lit son `R(6)` ; un DELTA prend l archetype du monde.
func cmMasqueSainEn(f *cmFilm, pay []byte, pos int, w *World) (typ int, slot uint32, sain bool) {
	essai := f.cfg
	essai.Obs = nil
	br := LecteurSur(pay)
	br.poserCadre(essai)
	br.SetBitPos(pos + motFacultatifDEnTete(essai))
	typ = readRecordType(br)
	id := readRecordID(br, essai.IDLowBits, essai.IDBase)
	slot = id & 0x3fffffff
	var ti uint32
	switch typ {
	case recNew:
		ti = cmPrefixeNeuf(br, essai.NewDefaultStateBits)
		if ti >= objectArchetypeCount {
			return typ, slot, false
		}
	case recDelta:
		var ok bool
		if ti, ok = w.ArchetypeForSlot(slot); !ok {
			return typ, slot, true // pas de masque jugeable
		}
		if br.ReadBit() {
			br.Skip(7)
		}
	default:
		return typ, slot, true
	}
	arch, ok := f.reg.Archetype(int(ti))
	if !ok {
		return typ, slot, false
	}
	return typ, slot, violationDeMasque(cmLireMasque(br), len(arch.Components)) == "sain"
}

// cmChaineSaine : la chaine de tete de `pos` a `debut` respecte l ordre de l ecrivain (NEW* puis
// DELTA*, slots croissants, aucun DEL) et chacun de ses records a un masque ecrivable.
func cmChaineSaine(f *cmFilm, pay []byte, pos, debut int, w *World) bool {
	essai := f.cfg
	essai.Obs = nil
	extra := motFacultatifDEnTete(essai)
	var h []cmEnTete
	for n := 0; n < plafondChaineDeTete && pos < debut; n++ {
		typ, slot, sain := cmMasqueSainEn(f, pay, pos, w)
		if !sain {
			return false
		}
		h = append(h, cmEnTete{typ: typ, slot: slot})
		fin, ok := pasDEssai(pay, pos, extra, w, essai)
		if !ok || fin <= pos {
			return false
		}
		pos = fin
	}
	return violationDOrdre(h, true) == "ordre respecte"
}

// cmTeteInv : le localisateur de L1a. Candidats : bande de production, OU (si `bloc`)
// allocation au bloc de type 1 ; un candidat n est garde que si son NEW a un masque ecrivable.
// Chaine : en plus de finir au debut strict, elle doit respecter l ordre et les masques.
// Fermeture (localisateur strict muet) : le paquet ferme doit en plus ne contredire aucun
// invariant. `paquet` : sur le chemin de la chaine aussi, le paquet decode depuis le debut
// retenu ne doit contredire aucun invariant, sinon le debut strict est garde.
func cmTeteInv(f *cmFilm, b *cmBlocs, bloc, paquet bool) func(c int) func([]byte, *World, FrameConfig) (int, bool) {
	chk := nouveauCollecteur(f, b, nil)
	return func(c int) func([]byte, *World, FrameConfig) (int, bool) {
		n, aSuivant := b.suivant[c]
		alloue := func(slot, gen uint32) bool {
			if !bloc {
				return false
			}
			g := uint8(gen) //nolint:gosec // deux bits
			cur, _ := b.entree(c, slot)
			if cur.Vivante() && cur.Gen == g {
				return true
			}
			if !aSuivant {
				return false
			}
			nx, ok := b.entree(n, slot)
			return ok && cmAlloueSous(nx, g) && !cmAlloueSous(cur, g)
		}
		return func(pay []byte, w *World, cfg FrameConfig) (int, bool) {
			extra := motFacultatifDEnTete(cfg)
			cand := func(fin int) []int {
				var out []int
				for p := 0; p+woNewHeaderBits <= fin; p++ {
					slot, ti, ok := enteteNeufEn(pay, p)
					if !ok || p-extra < 0 {
						continue
					}
					h := LireHandle(pay, p+woNewTypeBits)
					if !w.anticipee.SlotDeLArchetype(slot, ti) && !alloue(h.Slot, h.Gen) {
						continue
					}
					if _, _, sain := cmMasqueSainEn(f, pay, p-extra, w); sain {
						out = append(out, p)
					}
				}
				return out
			}
			propre := func(debut int) bool {
				snap := w.Snapshot()
				q := cmEssaiDepuis(f, pay, w, debut)
				w.Restore(snap)
				chk.chunk = c
				return len(cmContredit(chk, q)) == 0
			}
			strict := marchLocateStrict(pay, w, cfg)
			if strict < 0 {
				for _, p := range cand(len(pay) * 8) {
					essai := cfg
					obs := NouvelleObservation()
					ferme := false
					obs.VueControleHook = func(l LectureVueC) { ferme = l.Fermee }
					essai.Obs = obs
					snap := w.Snapshot()
					DecodeFrameViewsCurseur(pay, w, essai, MovementStateViews, p-extra)
					w.Restore(snap)
					if ferme && propre(p-extra) {
						return p - extra, true
					}
				}
				return -1, false
			}
			for _, p := range cand(strict) {
				if p >= strict || !chaineJusqua(pay, p-extra, strict, extra, w, cfg) {
					continue
				}
				if !cmChaineSaine(f, pay, p-extra, strict, w) {
					continue
				}
				if paquet && !propre(p-extra) {
					return strict, false
				}
				return p - extra, true
			}
			return strict, false
		}
	}
}

// cmNomsDeRegion : les regions de l oracle, dans l ordre du rapport.
var cmNomsDeRegion = []struct{ region, nom string }{
	{"(i) tete d un paquet a evenements, avant le debut", "oracle-(i)"},
	{"(ii) paquet a evenements non localise", "oracle-(ii)"},
	{"(iii') apres l arret de la vue B : rejet hors datum", "oracle-(iii')-rejet"},
	{"(iii') apres l arret de la vue B : ouverte", "oracle-(iii')-ouverte"},
	{"(iii) NEW lu desynchronise", "oracle-(iii)"},
}

// cmUnion fusionne des tables de liaisons.
func cmUnion(ms ...map[[2]int][]cmLiaison) map[[2]int][]cmLiaison {
	out := map[[2]int][]cmLiaison{}
	for _, m := range ms {
		for k, v := range m {
			out[k] = append(out[k], v...)
		}
	}
	return out
}

// TestCampagneMesuresBis1 joue les mesures bis 1 sur les films de CAMPAGNE_FILMS.
func TestCampagneMesuresBis1(t *testing.T) {
	racine, sortie := os.Getenv("CAMPAGNE_RACINE"), os.Getenv("CAMPAGNE_SORTIE")
	var films []string
	for _, x := range strings.Split(os.Getenv("CAMPAGNE_FILMS"), ",") {
		if x = strings.TrimSpace(x); x != "" {
			films = append(films, x)
		}
	}
	if racine == "" || sortie == "" || len(films) == 0 {
		t.Skip("CAMPAGNE_RACINE, CAMPAGNE_FILMS et CAMPAGNE_SORTIE requis")
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
	utiles := cmUtiles(t)
	tables := map[string][]string{}
	for _, id := range films {
		cmBis1UnFilm(t, racine, id, utiles, tables)
	}
	tetes := map[string]string{
		"variantes": "film\tbuild\tvariante\tpaquets\tfermes\tutiles_fermes\tutiles_lus\thors_cadre\tlistes_non_localisees\t" +
			"liaisons\tgagnes\tperdus\tgagnes_contredits\tfermes_contredits\tentrees_fermees\tentrees_utiles_fermees\n",
		"juge":    "film\tbuild\tvariante\tcle\tpaquets\tutiles_en_jeu\n",
		"entrees": "film\tbuild\tmesure\tvaleur\n",
		"rejets":  "film\tbuild\tcle\tn\tpaquets\thors_cadre\tfermes\ten_jeu\n",
	}
	for nom, lignes := range tables {
		sort.Strings(lignes)
		brut := tetes[nom] + strings.Join(lignes, "\n") + "\n"
		if err := os.WriteFile(filepath.Join(abs, "mb_"+nom+".tsv"), []byte(brut), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// cmBis1UnFilm mesure un film.
func cmBis1UnFilm(t *testing.T, racine, id string, utiles UsagesProduit, tables map[string][]string) {
	garde := filmproc.Arm("campagne/bis1", 4, func(pic uint64) {
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
	wr := profile.QuantRangeCEBiped()
	cre, _, _ := ScanVehicleCreations(f.fc, &wr)
	col := nouveauCollecteur(f, b, cre)
	ent := cmNouvellesEntrees(f)
	jref := cmNouveauJuge(f, b, nil)
	rep, _, _ := cmMarcher(f, cmVariante{}, cmMux{col, ent, jref})
	ligne := func(nom string, r FrameClosureReport, nl int, j *cmJuge) {
		tables["variantes"] = append(tables["variantes"], fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d",
			id, f.build, nom, r.Paquets, r.PaquetsFermes, r.Utiles.RecordsFermes, r.Utiles.Records,
			r.Bloquants[CauseHorsCadre].Paquets, r.ListesNonLocalisees, nl, j.gagnes, j.perdus,
			j.gagnesContredits, j.fermesContre, j.entrees, j.entreesUtiles))
		for _, k := range cmCles(j.t["juge"]) {
			x := j.t["juge"][k]
			tables["juge"] = append(tables["juge"], fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%d", id, f.build, nom, k, x.n, x.enJeu))
		}
	}
	ligne("reference", rep, 0, jref)
	compter := func(m map[[2]int][]cmLiaison) int {
		n := 0
		for _, l := range m {
			n += len(l)
		}
		return n
	}
	type variante struct {
		nom string
		v   cmVariante
		nl  int
	}
	vs := []variante{{"oracle-NEW", cmVariante{oracle: col.oracle}, compter(col.oracle)}}
	connues := map[string]bool{}
	for _, x := range cmNomsDeRegion {
		connues[x.region] = true
		m := col.parRegion[x.region]
		vs = append(vs, variante{x.nom, cmVariante{oracle: m}, compter(m)})
	}
	var autres []string
	for reg := range col.parRegion {
		if !connues[reg] {
			autres = append(autres, reg)
		}
	}
	sort.Strings(autres)
	for _, reg := range autres {
		m := col.parRegion[reg]
		vs = append(vs, variante{"oracle-autre : " + reg, cmVariante{oracle: m}, compter(m)})
	}
	i12 := cmUnion(col.parRegion[cmNomsDeRegion[0].region], col.parRegion[cmNomsDeRegion[1].region])
	vs = append(vs,
		variante{"oracle-(i)+(ii)", cmVariante{oracle: i12}, compter(i12)},
		variante{"oracle-bloc", cmVariante{oracle: col.oracleBloc}, compter(col.oracleBloc)},
		variante{"tete-bloc", cmVariante{tete: cmTeteBloc(b)}, 0},
		variante{"oracle-bloc+tete-bloc", cmVariante{oracle: col.oracleBloc, tete: cmTeteBloc(b)}, compter(col.oracleBloc)},
		variante{"bande+inv", cmVariante{tete: cmTeteInv(f, b, false, false)}, 0},
		variante{"tete-bloc+inv", cmVariante{tete: cmTeteInv(f, b, true, false)}, 0},
		variante{"tete-bloc+inv+paquet", cmVariante{tete: cmTeteInv(f, b, true, true)}, 0},
	)
	for _, x := range vs {
		j := cmNouveauJuge(f, b, col.statut)
		j.contreRef, j.utilesRef = jref.contreRef, jref.utilesRef
		r, _, _ := cmMarcher(f, x.v, j)
		ligne(x.nom, r, x.nl, j)
	}
	tables["entrees"] = append(tables["entrees"], ent.lignes(id, f.build)...)
	rj := cmTables{}
	for _, x := range col.rangs {
		lie := "non lie"
		if x.lie {
			lie = "lie par l oracle"
		}
		rj.add("rejets", cmJoindre(x.region, x.distance, lie, x.desync), x.compte)
	}
	for _, k := range cmCles(rj["rejets"]) {
		x := rj["rejets"][k]
		tables["rejets"] = append(tables["rejets"], fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d",
			id, f.build, k, x.n, x.paquets, x.horsCadre, x.fermes, x.enJeu))
	}
	t.Logf("%s %s : %d marches ; pic %d Mio, %s", id, f.build, len(vs)+1, garde.Peak()>>20, time.Since(debut).Round(time.Second))
}
