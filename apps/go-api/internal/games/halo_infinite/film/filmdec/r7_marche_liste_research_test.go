package filmdec

// r7_marche_liste_research_test.go — lot R7 : LE MARCHEUR DE LISTE et ses temoins.
//
// La liste d'evenements d'un paquet delta est `[1 bit config][( 1 R(7) type refs charge )* 0]`.
// Le marcheur consomme les evenements l'un apres l'autre en s'appuyant sur deux tables
// sourcees de l'exe : les domaines des references (`r7_grammaire_research_test.go`) et les
// largeurs de charge (`r7_charges*_research_test.go`).
//
// TROIS TEMOINS DE CADRAGE, ecrits AVANT toute mesure (methode imposee du plan) :
//
//  1. FIN DE LISTE, contre un TEMOIN DECALE. Une marche juste se termine sur un bit de
//     continuation a 0 ; une marche fausse derive et bute sur un type opaque ou un type
//     >= 123. On mesure le taux de fin propre au bon cadrage CONTRE le meme decodage decale
//     de +1 et de +3 bits sur les MEMES paquets. SEUIL ECRIT D'AVANCE : >= 90 % de fins
//     propres parmi les marches non bloquees au bon cadrage, et un ecart d'au moins un
//     facteur 2 avec chaque temoin decale.
//  2. TAUX D'OPACITE PAR TYPE. La distribution des types rencontres doit reproduire celle du
//     recensement des TETES (dont le cadrage est certain : config + continuation + R(7)).
//     Un desalignement rendrait des types quasi uniformes. SEUIL ECRIT D'AVANCE : les cinq
//     types les plus vus en position >= 2 doivent appartenir aux dix premiers du recensement.
//  3. ORACLE DE TRAME (le juge, celui de NOTE_MODELE_EVENEMENTS). Apres le dernier evenement,
//     la trame de records doit se decoder et ALLER LOIN. On mesure la profondeur moyenne et
//     le taux de fermeture au bon cadrage CONTRE un temoin volontairement decale de +3 bits.
//     SEUIL ECRIT D'AVANCE : facteur >= 3 sur la profondeur (le lot damage_aftermath avait
//     mesure 13 ; 3 est une marge prudente).
//
// LECTURE SEULE, skip par defaut, CGO_ENABLED=0, balayage borne (R7_CHUNKS).

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// r7Stop : pourquoi la marche s'est arretee.
type r7Stop int

const (
	r7StopFin         r7Stop = iota // bit de continuation a 0 : liste finie proprement
	r7StopOpaque                    // type sans grammaire fermee : on ne sait pas avancer
	r7StopSansDomaine               // type absent de la table des domaines (impossible : 123/123)
	r7StopTypeInconnu               // type >= 123 : impossible, cadrage refute
	r7StopBuffer                    // fin du payload atteinte
)

// r7Ev : un evenement lu dans la liste.
type r7Ev struct {
	Typ      int
	Pos      int // rang dans la liste, 1 = tete
	BitDebut int // bit du R(7) de type
	BitFin   int // bit apres la charge
	Ref0     uint64
	HasRef0  bool
}

// r7Marche parcourt la liste d'evenements d'un paquet delta a partir du bit 1 (le bit de
// configuration est deja consomme). Rend les evenements lus, la raison d'arret, le type
// opaque le cas echeant, et le bit ou commence la trame (valide seulement si stop==r7StopFin).
func r7Marche(pay []byte, ctx r7Ctx) ([]r7Ev, r7Stop, int, int) {
	return r7MarcheDecalee(pay, ctx, 0)
}

// r7MarcheDecalee est la marche avec un decalage volontaire de `decale` bits AJOUTE APRES LA
// CHARGE de chaque evenement — le TEMOIN NEGATIF du temoin 1. Le decalage est place la et
// pas au depart : decaler l'entree ferait lire un bit de continuation a 0 des le debut, donc
// une « liste vide » qui se declarerait trivialement « fin propre » — un temoin sans valeur.
// Place apres la charge, il attaque exactement ce que ce lot pretend avoir etabli : la
// LARGEUR des charges. decale=0 est la marche reelle.
func r7MarcheDecalee(pay []byte, ctx r7Ctx, decale int) ([]r7Ev, r7Stop, int, int) {
	br := NewBitReader(pay)
	br.Skip(1) // bit de configuration
	var evs []r7Ev
	for pos := 1; ; pos++ {
		if br.Remaining() < 1 {
			return evs, r7StopBuffer, 0, br.BitPos()
		}
		if !br.ReadBit() { // bit de continuation : 0 = fin de liste
			return evs, r7StopFin, 0, br.BitPos()
		}
		if br.Remaining() < 7 {
			return evs, r7StopBuffer, 0, br.BitPos()
		}
		debut := br.BitPos()
		typ := int(br.ReadBits(7))
		if typ >= 123 {
			return evs, r7StopTypeInconnu, typ, br.BitPos()
		}
		ref0, has0, ok := r7RefsSkip(br, typ)
		if !ok {
			return evs, r7StopSansDomaine, typ, br.BitPos()
		}
		if !r7SkipCharge(br, typ, ctx) {
			return evs, r7StopOpaque, typ, br.BitPos()
		}
		br.Skip(decale) // temoin negatif : 0 pour la marche reelle
		if br.BitPos() > len(pay)*8 {
			return evs, r7StopBuffer, typ, br.BitPos()
		}
		evs = append(evs, r7Ev{Typ: typ, Pos: pos, BitDebut: debut, BitFin: br.BitPos(),
			Ref0: ref0, HasRef0: has0})
	}
}

// r7Bilan agrege une campagne de marche.
type r7Bilan struct {
	paquets      int // paquets delta a liste non vide
	listesVides  int
	fins         int
	opaques      map[int]int
	refsImposs   map[int]int
	typesInconnu int
	buffers      int
	profondeurs  map[int]int         // longueur de liste -> nombre de listes (marches finies)
	parPosition  map[int]map[int]int // position -> type -> compte
	evenements   int
}

func newR7Bilan() *r7Bilan {
	return &r7Bilan{opaques: map[int]int{}, refsImposs: map[int]int{},
		profondeurs: map[int]int{}, parPosition: map[int]map[int]int{}}
}

func (b *r7Bilan) ajoute(evs []r7Ev, stop r7Stop, typ int) {
	b.paquets++
	b.evenements += len(evs)
	for _, e := range evs {
		if b.parPosition[e.Pos] == nil {
			b.parPosition[e.Pos] = map[int]int{}
		}
		b.parPosition[e.Pos][e.Typ]++
	}
	switch stop {
	case r7StopFin:
		b.fins++
		b.profondeurs[len(evs)]++
	case r7StopOpaque:
		b.opaques[typ]++
	case r7StopSansDomaine:
		b.refsImposs[typ]++
	case r7StopTypeInconnu:
		b.typesInconnu++
	case r7StopBuffer:
		b.buffers++
	}
}

func (b *r7Bilan) fusionne(o *r7Bilan) {
	b.paquets += o.paquets
	b.listesVides += o.listesVides
	b.fins += o.fins
	b.typesInconnu += o.typesInconnu
	b.buffers += o.buffers
	b.evenements += o.evenements
	for k, v := range o.opaques {
		b.opaques[k] += v
	}
	for k, v := range o.refsImposs {
		b.refsImposs[k] += v
	}
	for k, v := range o.profondeurs {
		b.profondeurs[k] += v
	}
	for p, m := range o.parPosition {
		if b.parPosition[p] == nil {
			b.parPosition[p] = map[int]int{}
		}
		for k, v := range m {
			b.parPosition[p][k] += v
		}
	}
}

// r7Compte formate une distribution type -> compte, triee par compte decroissant.
func r7Compte(m map[int]int, top int) string {
	type kv struct{ k, v int }
	var l []kv
	for k, v := range m {
		l = append(l, kv{k, v})
	}
	sort.Slice(l, func(i, j int) bool {
		if l[i].v != l[j].v {
			return l[i].v > l[j].v
		}
		return l[i].k < l[j].k
	})
	var parts []string
	for i, e := range l {
		if top > 0 && i >= top {
			parts = append(parts, fmt.Sprintf("... (+%d autres)", len(l)-top))
			break
		}
		parts = append(parts, fmt.Sprintf("%d %s:%d", e.k, r7Noms[e.k], e.v))
	}
	return strings.Join(parts, " · ")
}

func (b *r7Bilan) rapport(t *testing.T, echelle string) {
	t.Helper()
	t.Logf("== %s : %d listes non vides · %d fins propres (%.1f %%) · %d evenements lus ==",
		echelle, b.paquets, b.fins, 100*float64(b.fins)/float64(max(1, b.paquets)), b.evenements)
	t.Logf("   arrets : opaque %d · sans domaine %d · type >=123 %d · buffer %d",
		sommeMap(b.opaques), sommeMap(b.refsImposs), b.typesInconnu, b.buffers)
	if len(b.opaques) > 0 {
		t.Logf("   types OPAQUES (a fermer) : %s", r7Compte(b.opaques, 15))
	}
	var prof []int
	for k := range b.profondeurs {
		prof = append(prof, k)
	}
	sort.Ints(prof)
	var pp []string
	for _, k := range prof {
		pp = append(pp, fmt.Sprintf("%d:%d", k, b.profondeurs[k]))
	}
	t.Logf("   longueur des listes fermees {%s}", strings.Join(pp, " "))
	var poss []int
	for p := range b.parPosition {
		poss = append(poss, p)
	}
	sort.Ints(poss)
	for _, p := range poss {
		if p > 6 {
			continue
		}
		t.Logf("   position %d : %s", p, r7Compte(b.parPosition[p], 12))
	}
}

func sommeMap(m map[int]int) int {
	s := 0
	for _, v := range m {
		s += v
	}
	return s
}

// r7CtxDeCarte construit le contexte de marche depuis une entree du catalogue de production.
func r7CtxDeCarte(e r6CatEntry) r7Ctx {
	return r7Ctx{
		etendues:   [3]float64{e.Max[0] - e.Min[0], e.Max[1] - e.Min[1], e.Max[2] - e.Min[2]},
		regionBits: 1, // DAT_144632be0 : 1 quand la carte n'a qu'une region (mesure R6, 18/18)
		hasMap:     true,
	}
}

// r7Profil : un candidat de carte pour la calibration.
type r7Profil struct {
	Nom string
	Ctx r7Ctx
}

// r7ProfilsCarte rend les profils d'etendues DISTINCTS du catalogue de production
// (`map_quant_bounds.json`), avec un nom representatif. C'est le jeu de candidats de la
// calibration de carte : des valeurs venues de cartes REELLES, jamais un balayage libre.
func r7ProfilsCarte(t *testing.T) []r7Profil {
	t.Helper()
	catPath := os.Getenv(r7CatEnv)
	if catPath == "" {
		return nil
	}
	vus := map[string]bool{}
	var out []r7Profil
	noms := make([]string, 0)
	cat := r6LireCatalogue(t, catPath)
	for nom := range cat {
		noms = append(noms, nom)
	}
	sort.Strings(noms)
	for _, nom := range noms {
		e := cat[nom]
		if e.AxisWidths[0] == 22 && e.AxisWidths[1] == 22 && e.AxisWidths[2] == 22 {
			continue // pseudo-entree « defaut moteur »
		}
		ctx := r7CtxDeCarte(e)
		cle := fmt.Sprintf("%d/%d/%d", r7BitsAxe(ctx.etendues[0], 16),
			r7BitsAxe(ctx.etendues[1], 16), r7BitsAxe(ctx.etendues[2], 16))
		if vus[cle] {
			continue
		}
		vus[cle] = true
		out = append(out, r7Profil{nom + " [" + cle + "]", ctx})
	}
	return out
}

// r7Cartes construit le contexte par film depuis R7_MAPS ("id8=nomCarte,...") et R7_CAT.
func r7Cartes(t *testing.T) map[string]r7Ctx {
	t.Helper()
	out := map[string]r7Ctx{}
	catPath, maps := os.Getenv(r7CatEnv), os.Getenv(r7MapsEnv)
	if catPath == "" || maps == "" {
		return out
	}
	cat := r6LireCatalogue(t, catPath)
	for _, kv := range strings.Split(maps, ",") {
		i := strings.IndexByte(kv, '=')
		if i <= 0 {
			continue
		}
		id, nom := strings.TrimSpace(kv[:i]), strings.TrimSpace(kv[i+1:])
		e, ok := cat[nom]
		if !ok {
			t.Logf("carte %q inconnue du catalogue (film %s)", nom, id)
			continue
		}
		out[id] = r7CtxDeCarte(e)
	}
	return out
}

// r7Chunks borne le balayage (R7_CHUNKS, 0 = tous).
func r7Chunks(dir string) int {
	n := CountFilmChunks(dir)
	if v := os.Getenv("R7_CHUNKS"); v != "" {
		if k, err := strconv.Atoi(v); err == nil && k > 0 && k < n {
			return k
		}
	}
	return n
}

// TestR7Marche marche la liste complete de tous les paquets delta du parc et publie les
// temoins 1 et 2. Le temoin 3 (oracle de trame) est dans TestR7OracleTrame.
func TestR7Marche(t *testing.T) {
	root, ids := r7Films(t)
	cartes := r7Cartes(t)
	parc := newR7Bilan()
	temoins := map[int]*r7Bilan{1: newR7Bilan(), 3: newR7Bilan()}
	for _, id := range ids {
		dir := filepath.Join(root, id)
		n := r7Chunks(dir)
		if n == 0 {
			t.Logf("film %s : aucun chunk", id)
			continue
		}
		ctx := cartes[id]
		film := newR7Bilan()
		for c := 1; c <= n; c++ {
			data, err := ReadFilmChunk(dir, c)
			if err != nil {
				continue
			}
			for _, pk := range WalkPackets(data) {
				if pk.Type != PacketTypeDelta || pk.Size < 2 {
					continue
				}
				pay := pk.Payload(data)
				if pay[0]&0x40 == 0 { // bit de continuation a 0 : liste vide
					film.listesVides++
					continue
				}
				evs, stop, typ, _ := r7Marche(pay, ctx)
				film.ajoute(evs, stop, typ)
				for d, b := range temoins {
					de, ds, dt, _ := r7MarcheDecalee(pay, ctx, d)
					b.ajoute(de, ds, dt)
				}
			}
		}
		film.rapport(t, "FILM "+id)
		parc.fusionne(film)
	}
	t.Logf("")
	parc.rapport(t, "PARC")
	// Temoin 1 : fin de liste au bon cadrage CONTRE un decalage de +1 / +3 bits APRES la charge.
	taux := func(b *r7Bilan) float64 {
		return 100 * float64(b.fins) / float64(max(1, b.paquets-sommeMap(b.opaques)))
	}
	t.Logf("TEMOIN 1 (fin de liste apres charge) : cadrage JUSTE %.2f %% (%d/%d) ; "+
		"decale +1 bit %.2f %% (%d/%d) ; decale +3 bits %.2f %% (%d/%d)",
		taux(parc), parc.fins, parc.paquets-sommeMap(parc.opaques),
		taux(temoins[1]), temoins[1].fins, temoins[1].paquets-sommeMap(temoins[1].opaques),
		taux(temoins[3]), temoins[3].fins, temoins[3].paquets-sommeMap(temoins[3].opaques))
	t.Logf("   seuils ecrits d'avance : >= 90 %% au cadrage juste, et facteur >= 2 contre chaque temoin")
	// Temoin 2 : plausibilite de la distribution des types en position >= 2.
	tous := map[int]int{}
	for pos, m := range parc.parPosition {
		if pos < 2 {
			continue
		}
		for typ, n := range m {
			tous[typ] += n
		}
	}
	t.Logf("TEMOIN 2 (distribution en position >= 2, %d evenements) : %s",
		sommeMap(tous), r7Compte(tous, 12))
}
