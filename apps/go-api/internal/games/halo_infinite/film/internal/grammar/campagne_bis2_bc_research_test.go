//go:build research

package grammar

// campagne_bis2_bc_research_test.go — MESURES BIS 2 DE LA CAMPAGNE DE GRAMMAIRE (2026-10-01),
// POINT 24 DE LA CRITIQUE DE COMPLETUDE : LA FOURCHE `+0x74` DU BLOC 0xbc DE LA VUE C (T5-3).
//
// Un instrument de recherche : aucun fichier de production n est touche, `grammar.Rev` ne bouge
// pas. La marche est celle de la carte v2 ([cmMarcher], copie de recherche de
// [FrameClosureDetaillee]) ; la production arrete la vue C sur le bit `b` d une entree kind 0
// ([ArretVueCBlocBC]). Pour CHAQUE paquet ainsi arrete, la vue C est RELUE depuis la fin de la vue
// B ([PaquetDeCarte.FinVueB], la ou [consumeVueC] a commence) avec le bloc 0xbc PORTE
// (`FUN_141fdae44(lecteur, bloc, 1)`, relu dans Ghidra le 2026-10-01, lecture seule) sous les deux
// formes de la fourche, et sous deux temoins de hasard :
//
//	r96   +0x74 = R(96) brut            (`FUN_1406d676c(lecteur, +0x74, 0x60)`, 141fdb102)
//	e420  +0x74 = FUN_14076e420(0x10)   (141fdb0f7 ; niveau 0x10 sous DAT_145121140 != 1)
//	r95, r97 : la forme r96 a UN bit pres (temoins : ce qu une largeur fausse ferme par hasard)
//
// Grammaire portee (lecteur ET ecrivain `FUN_1406d1ba4` lus par T5 §5 ; lecteur relu ici) :
//
//	R(5) flags ; R(2) ; R(17) lacet ; R(16) tangage ; R(1) [varint cat 1]   (param_3 = 1)
//	si flags & 2 :
//	   R(1) [varint cat 1]                          FUN_142f2e63c(+0x18, 1)
//	   R(2) ; R(3) ; c = R(3) ; c x { R(1) [varint cat 0] ; R(4) ; R(4) }
//	   R(5) ; R(3) ; R(1) [R(2) si 0] ; R(1) [R(2) si 0]       FUN_1424ccc74, FUN_1406d01fc
//	   R(6) ; R(3)                                 FUN_140c6a638
//	   R(3) ; R(1) [R(6) si 0]                     FUN_1406d0ff0
//	   FOURCHE +0x74
//	   R(1) [R(19) si 0] ; R(8)                    FUN_140c1e79c
//	   R(1) [R(19) R(10) si 0]                     FUN_14076d528(0x13, 0xa)
//	   R(1) [R(19) R(8) si 0]                      FUN_14076d528(0x13, 0x8)
//	   p = FUN_1408f0ac4(cat 0) ; si p : R(11)     FUN_142f0ec48 (+0xb8 = !p)
//	   R(7)                                        FUN_142f21dd4
//
// Rejouable (un film a la fois, plafond 4 Gio) :
//
//	CAMPAGNE_RACINE=<film_chunks> CAMPAGNE_FILMS=<id,id> CAMPAGNE_SORTIE=<dir hors data> \
//	  go test -tags=research -count=1 -timeout 120m -run '^TestCampagneBis2BlocBC$' \
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
)

// b2Forme est une forme de lecture du champ `+0x74` du bloc 0xbc.
type b2Forme int

const (
	b2R96 b2Forme = iota
	b2E420
	b2R95
	b2R97
	b2NombreDeFormes
)

func (f b2Forme) String() string {
	return [...]string{"r96", "e420", "r95-temoin", "r97-temoin"}[f]
}

// b2Bloc : ce que la lecture d un bloc 0xbc a rendu.
type b2Bloc struct {
	flags, tangage uint64
	// invalide : le bloc contredit la validite de l ecrivain (`FUN_1407699d0`, partie lisible
	// sans etat : flags & 0x1f != 0 quand flags & 2 == 0 ; tangage dans [-pi/2 ; pi/2] quand
	// flags & 1, codes 16384..49152 sur 16 bits).
	invalide bool
}

// b2LireBlocBC lit `FUN_141fdae44(lecteur, bloc, 1)` sous la forme `f`.
func b2LireBlocBC(br *Lecteur, f b2Forme) b2Bloc {
	var b b2Bloc
	b.flags = br.ReadBits(5) // FUN_1424ccc74
	br.ReadBits(2)           // +0x01, forme courte (DAT_145121140 != 1)
	br.ReadBits(17)          // FUN_1406d84b4 0x11 lacet
	b.tangage = br.ReadBits(16)
	if br.ReadBit() { // param_3 = 1
		readVarWidthInt(br, 1)
	}
	if b.flags&2 != 0 {
		if br.ReadBit() { // FUN_142f2e63c(+0x18, cat 1)
			readVarWidthInt(br, 1)
		}
		br.ReadBits(2)                   // FUN_14080cb98
		br.ReadBits(3)                   // FUN_1406d0f20
		c := br.ReadBits(3)              // FUN_1424d0f48
		for k := uint64(0); k < c; k++ { // FUN_142f26754
			if br.ReadBit() { // FUN_142f2e63c(cat 0)
				readVarWidthInt(br, 0)
			}
			br.ReadBits(4) // FUN_142ed0674
			br.ReadBits(4) // FUN_14101d200
		}
		br.ReadBits(5) // FUN_1424ccc74
		br.ReadBits(3) // FUN_1406d01fc : FUN_1406d0f20
		for k := 0; k < 2; k++ {
			if !br.ReadBit() { // FUN_1406d00ec
				br.ReadBits(2)
			}
		}
		br.ReadBits(6) // FUN_140c6a638
		br.ReadBits(3)
		br.ReadBits(3) // FUN_1406d0ff0
		if !br.ReadBit() {
			br.ReadBits(6)
		}
		switch f {
		case b2R96:
			br.ReadBits(rawVec3Bits)
		case b2R95:
			br.ReadBits(rawVec3Bits - 1)
		case b2R97:
			br.ReadBits(rawVec3Bits + 1)
		case b2E420:
			lireE420(br, niveauPosition)
		}
		consumeCompressedDir140c1e79c(br)
		if !br.ReadBit() { // FUN_14076d528(0x13, 0xa)
			br.ReadBits(19)
			br.ReadBits(10)
		}
		if !br.ReadBit() { // FUN_14076d528(0x13, 0x8)
			br.ReadBits(19)
			br.ReadBits(8)
		}
		if present, _ := consume1408f0ac4(br, 0); present { // FUN_142f0ec48
			br.ReadBits(11)
		}
		br.ReadBits(7) // FUN_142f21dd4
	}
	switch {
	case b.flags&2 == 0 && b.flags&0x1f == 0:
		b.invalide = true
	case b.flags&1 != 0 && (b.tangage < 16384 || b.tangage > 49152):
		b.invalide = true
	}
	return b
}

// b2Lecture : la relecture de la vue C d un paquet sous une forme.
type b2Lecture struct {
	fermee          bool
	entrees, blocs  int
	blocsInvalides  int
	blocsFlags2     int
	arretKind, plaf bool
	debordement     bool
}

// b2RelireVueC relit la vue C depuis `debut` (copie de [consumeVueC] + [lireEntreeDeControle], le
// bloc 0xbc porte).
func b2RelireVueC(pay []byte, debut int, cfg FrameConfig, f b2Forme) b2Lecture {
	essai := cfg
	essai.Obs = nil // une relecture ne publie rien
	br := LecteurSur(pay)
	br.poserCadre(essai)
	br.SetBitPos(debut)
	frameLen := len(pay) * 8
	var out b2Lecture
	for tour := 0; tour < plafondToursVueC; tour++ {
		if !placeDisponible(br, frameLen, 1) {
			out.debordement = true
			return out
		}
		if !br.ReadBit() {
			out.fermee = vueCFermee(pay, br.BitPos())
			return out
		}
		if !placeDisponible(br, frameLen, LargeurKindVueC) {
			out.debordement = true
			return out
		}
		switch br.ReadBits(LargeurKindVueC) {
		case kindVueCNeant:
			continue
		case kindVueCControle:
		default:
			out.arretKind = true
			return out
		}
		e := EntreeDeControle{Champs: champsAbsents()}
		if br.ReadBit() {
			br.ReadBits(largeurIndexCdc04)
		}
		br.ReadBits(largeurIndexControle)
		if br.ReadBit() {
			if _, ok := consumeEntreeControle(br, frameLen, &e.Champs); !ok {
				out.debordement = true
				return out
			}
		}
		if !placeDisponible(br, frameLen, 1) {
			out.debordement = true
			return out
		}
		if br.ReadBit() {
			b := b2LireBlocBC(br, f)
			out.blocs++
			if b.invalide {
				out.blocsInvalides++
			}
			if b.flags&2 != 0 {
				out.blocsFlags2++
			}
			if br.BitPos() > frameLen {
				out.debordement = true
				return out
			}
		}
		out.entrees++
	}
	out.plaf = true
	return out
}

// b2Compte : les comptes d un film sous une forme.
type b2Compte struct {
	fermes, utiles, entrees, blocs, invalides, flags2, exclusifs int
	fermesInvalides                                              int
	parSortie                                                    map[string]int
}

// b2Ecouteur relit la vue C des paquets arretes par le bloc 0xbc.
type b2Ecouteur struct {
	cfg                         FrameConfig
	arretes, arretesSortieRejet int
	utilesArretes               int
	comptes                     [b2NombreDeFormes]b2Compte
	paquets                     []string
	film, build                 string
}

func (e *b2Ecouteur) debutDeChunk(int, []byte, []FilmPacket, *World) {}
func (e *b2Ecouteur) finDeFilm()                                     {}

func (e *b2Ecouteur) paquet(c int, p *cmPaquet, _ *World) {
	if p.d.Fermee || !p.d.VueCAtteinte || p.d.VueC.Arret != ArretVueCBlocBC {
		return
	}
	e.arretes++
	e.utilesArretes += p.d.UtilesEnJeu
	if p.d.Sortie.EstUnRejet() {
		e.arretesSortieRejet++
	}
	var lus [b2NombreDeFormes]b2Lecture
	for f := b2Forme(0); f < b2NombreDeFormes; f++ {
		lus[f] = b2RelireVueC(p.pay, p.d.FinVueB, e.cfg, f)
	}
	for f := b2Forme(0); f < b2NombreDeFormes; f++ {
		l, k := lus[f], &e.comptes[f]
		if !l.fermee {
			continue
		}
		k.fermes++
		k.utiles += p.d.UtilesEnJeu
		k.entrees += l.entrees
		k.blocs += l.blocs
		k.invalides += l.blocsInvalides
		k.flags2 += l.blocsFlags2
		if l.blocsInvalides > 0 {
			k.fermesInvalides++
		}
		if k.parSortie == nil {
			k.parSortie = map[string]int{}
		}
		k.parSortie[p.d.Sortie.String()]++
		autre := b2E420
		if f == b2E420 {
			autre = b2R96
		}
		if (f == b2R96 || f == b2E420) && !lus[autre].fermee {
			k.exclusifs++
		}
	}
	etat := func(f b2Forme) string {
		if lus[f].fermee {
			return "F"
		}
		return "-"
	}
	e.paquets = append(e.paquets, fmt.Sprintf("%s\t%s\t%d\t%d\t%s\t%d\t%s\t%s\t%s\t%s\t%d\t%d",
		e.film, e.build, c, p.d.Index, p.d.Sortie, p.d.UtilesEnJeu, etat(b2R96), etat(b2E420),
		etat(b2R95), etat(b2R97), lus[b2R96].blocs, lus[b2E420].blocs))
}

// TestCampagneBis2BlocBC mesure la fermeture des paquets arretes par le bloc 0xbc sous les deux
// formes de la fourche `+0x74`.
func TestCampagneBis2BlocBC(t *testing.T) {
	racine, sortie, films := b2Env(t)
	utiles := cmUtiles(t)
	lignes := []string{"film\tbuild\tforme\tarretes_bc\tarretes_apres_rejet\tutiles_en_jeu_arretes\t" +
		"fermes\tutiles_fermes\tentrees_fermees\tblocs_lus\tblocs_invalides\tfermes_avec_bloc_invalide\t" +
		"blocs_flags2\tfermes_exclusifs\tfermes_apres_terminateur\tfermes_apres_rejet\tpaquets_fermes_ref\tpaquets"}
	detail := []string{"film\tbuild\tchunk\tpaquet\tsortie_vue_b\tutiles_en_jeu\tr96\te420\tr95\tr97\tblocs_r96\tblocs_e420"}
	for _, id := range films {
		garde := filmproc.Arm("campagne/bis2-bc", 4, func(pic uint64) {
			fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets)\n", id, pic)
			os.Exit(3)
		})
		debut := time.Now()
		f, ok := cmOuvrir(t, racine, id, utiles)
		if !ok {
			garde.Disarm()
			continue
		}
		e := &b2Ecouteur{cfg: f.cfg, film: id, build: f.build}
		rep, _, _ := cmMarcher(f, cmVariante{}, e)
		e.cfg = f.cfg
		for fo := b2Forme(0); fo < b2NombreDeFormes; fo++ {
			k := e.comptes[fo]
			lignes = append(lignes, fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d",
				id, f.build, fo, e.arretes, e.arretesSortieRejet, e.utilesArretes, k.fermes, k.utiles,
				k.entrees, k.blocs, k.invalides, k.fermesInvalides, k.flags2, k.exclusifs,
				k.parSortie["terminateur"], k.parSortie["rejet hors datum"]+k.parSortie["rejet de vue"],
				rep.PaquetsFermes, rep.Paquets))
		}
		detail = append(detail, e.paquets...)
		t.Logf("%s %s : %d paquets arretes par le bloc 0xbc ; fermes r96 %d, e420 %d, r95 %d, r97 %d ; pic %d Mio, %s",
			id, f.build, e.arretes, e.comptes[b2R96].fermes, e.comptes[b2E420].fermes,
			e.comptes[b2R95].fermes, e.comptes[b2R97].fermes, garde.Peak()>>20, time.Since(debut).Round(time.Second))
		garde.Disarm()
	}
	b2Ecrire(t, sortie, "mb2_bc.tsv", lignes)
	b2Ecrire(t, sortie, "mb2_bc_paquets.tsv", detail)
}

// b2Env lit l environnement commun des mesures bis 2.
func b2Env(t *testing.T) (racine, sortie string, films []string) {
	t.Helper()
	racine, sortie = os.Getenv("CAMPAGNE_RACINE"), os.Getenv("CAMPAGNE_SORTIE")
	for _, x := range strings.Split(os.Getenv("CAMPAGNE_FILMS"), ",") {
		if x = strings.TrimSpace(x); x != "" {
			films = append(films, x)
		}
	}
	if racine == "" || sortie == "" || len(films) == 0 {
		t.Skip("CAMPAGNE_RACINE, CAMPAGNE_FILMS et CAMPAGNE_SORTIE requis")
	}
	abs, err := filepath.Abs(sortie)
	if err != nil {
		t.Fatal(err)
	}
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

// b2Ecrire ecrit une table (en-tete puis lignes triees).
func b2Ecrire(t *testing.T, dir, nom string, lignes []string) {
	t.Helper()
	if len(lignes) == 0 {
		return
	}
	corps := append([]string(nil), lignes[1:]...)
	sort.Strings(corps)
	brut := lignes[0] + "\n" + strings.Join(corps, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(dir, nom), []byte(brut), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestCampagneBis2BlocBCVecteurs : les vecteurs V4/V5 de T5 §7.3 et un bloc a flags & 2 construit
// d apres l ecrivain, lus par [b2LireBlocBC].
func TestCampagneBis2BlocBCVecteurs(t *testing.T) {
	// V4 : idx 0, a = 0, b = 1, bloc flags = 1, +0x01 = 0, lacet 0, tangage 0x8000, +0x10 absent ;
	// terminateur au bit 52.
	pay := []byte{0x80, 0x21, 0x00, 0x00, 0x10, 0x00, 0x00}
	l := b2RelireVueC(pay, 0, FrameConfig{Profil: ProfilDeBalayageParDefaut()}, b2R96)
	if !l.fermee || l.blocs != 1 || l.entrees != 1 || l.blocsInvalides != 0 {
		t.Fatalf("V4 : %+v, attendu fermee, 1 entree, 1 bloc valide", l)
	}
	// V5 : flags = 0 -> invalide chez l ecrivain.
	pay = []byte{0x80, 0x20, 0x00, 0x00, 0x10, 0x00, 0x00}
	l = b2RelireVueC(pay, 0, FrameConfig{Profil: ProfilDeBalayageParDefaut()}, b2R96)
	if l.blocs != 1 || l.blocsInvalides != 1 {
		t.Fatalf("V5 : %+v, attendu 1 bloc invalide", l)
	}
	// Bloc flags = 2 (aucun champ optionnel), forme r96 : 5+2+17+16+1 + 1+2+3+3 + 5+3+1+1 + 6+3
	// + 3+1 + 96 + 1+8 + 1 + 1 + 1 + 7 = 188 bits, tous les bits optionnels a 1 sauf la porte de
	// FUN_1408f0ac4 et les portes de presence a 0.
	w := &b2Ecrivain{}
	w.ecrire(2, 5)
	w.ecrire(0, 2)
	w.ecrire(0, 17)
	w.ecrire(0x8000, 16)
	w.ecrire(0, 1) // +0x10 absent
	w.ecrire(0, 1) // +0x18 absent
	w.ecrire(0, 2)
	w.ecrire(0, 3)
	w.ecrire(0, 3) // c = 0
	w.ecrire(0, 5)
	w.ecrire(0, 3)
	w.ecrire(1, 1)
	w.ecrire(1, 1)
	w.ecrire(0, 6)
	w.ecrire(0, 3)
	w.ecrire(0, 3)
	w.ecrire(1, 1)
	w.ecrire(0, 96)
	w.ecrire(1, 1)
	w.ecrire(0, 8)
	w.ecrire(1, 1)
	w.ecrire(1, 1)
	w.ecrire(0, 1)
	w.ecrire(0, 7)
	if w.n != 188 {
		t.Fatalf("vecteur construit sur %d bits, attendu 189", w.n)
	}
	br := LecteurSur(w.octets())
	br.poserCadre(FrameConfig{Profil: ProfilDeBalayageParDefaut()})
	b := b2LireBlocBC(br, b2R96)
	if br.BitPos() != 188 || b.flags != 2 || b.invalide {
		t.Fatalf("bloc flags=2 : %d bits lus, %+v ; attendu 188, flags 2, valide", br.BitPos(), b)
	}
}

// b2Ecrivain ecrit des bits poids fort d abord.
type b2Ecrivain struct {
	bits []bool
	n    int
}

func (w *b2Ecrivain) ecrire(v uint64, n int) {
	for i := n - 1; i >= 0; i-- {
		w.bits = append(w.bits, v>>uint(i)&1 == 1)
	}
	w.n += n
}

func (w *b2Ecrivain) octets() []byte {
	out := make([]byte, (len(w.bits)+7)/8+16)
	for i, b := range w.bits {
		if b {
			out[i/8] |= 1 << uint(7-i%8)
		}
	}
	return out
}
