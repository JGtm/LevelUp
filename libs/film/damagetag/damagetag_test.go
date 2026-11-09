package damagetag

import (
	"sort"
	"testing"
)

// tagsAncres : les tags dont la source a ete confrontee a la VERITE TERRAIN en mode Theater
// (RE_LOG 7ter.33bis, .35, .37bis, .44, .47, .51, .56, .57). Ils DOIVENT survivre a toute
// regeneration : si l un d eux disparait du catalogue, l etiquette qui a ete confirmee par
// l utilisateur ne serait plus decodable.
var tagsAncres = map[uint32]struct {
	class Class
	name  string
}{
	0xdaa03c35: {ClassMelee, ""},              // MELEE, stable cross-film
	0x5eaa815c: {ClassArme, "Fuel Rod SPNKr"}, // 3 kills Theater sur 9b191a7f
	0x0000b29c: {ClassArme, "BR75"},           // regime faible (< 0x10000), 47 appariements
	0x4a555df8: {ClassArme, "Needler"},        // variante confirmee
	0xeea85c26: {ClassArme, "Disruptor"},      // variante Super Fiesta
	0x0d203522: {ClassObjet, ""},              // baril, etat de degat 1/7
	0x5e389b5d: {ClassObjet, ""},              // MEME baril, autre type d energie
	0x0000d627: {ClassGrenade, ""},            // grenade PLASMA (precedence gggl sur weap)
	0x119861b4: {ClassGrenade, ""},            // grenade a POINTES (correction de reference)
	0x00403594: {ClassGlobal, ""},             // chute / environnement
	0x003f582d: {ClassVehicule, ""},           // tourelle de vehicule
}

// tagsAmbigus : les deux lignes MESUREES COMME FAUSSES (RE_LOG 7ter.49 (2)d, 7ter.50 (3)1).
// Elles doivent rester marquees, et non publiables.
var tagsAmbigus = []uint32{0x88f1034c, 0x31e8d17e}

func TestCatalogueCharge(t *testing.T) {
	// Les tables GRANDISSENT d une saison a l autre : on borne par le bas, jamais a l egal.
	if Size() < 468 {
		t.Fatalf("catalogue jpt! = %d ids, attendu >= 468 (mesure du 2026-07-26)", Size())
	}
	if p := Source(); p.IDsDate == "" || p.LabelsDate == "" {
		t.Fatalf("provenance incomplete: %+v", p)
	}
}

func TestCatalogueTrieEtSansDoublon(t *testing.T) {
	l := IDs()
	if !sort.SliceIsSorted(l, func(i, j int) bool { return l[i] < l[j] }) {
		t.Fatal("le catalogue n est pas trie")
	}
	seen := map[uint32]bool{}
	for _, v := range l {
		if seen[v] {
			t.Fatalf("doublon %08x", v)
		}
		seen[v] = true
		if !IsDamageEffect(v) {
			t.Fatalf("%08x present dans IDs() mais absent de l ensemble", v)
		}
	}
}

func TestAncresTheaterPresentesEtBienClassees(t *testing.T) {
	for tag, want := range tagsAncres {
		if !IsDamageEffect(tag) {
			t.Errorf("%08x : ancre Theater absente du catalogue", tag)
			continue
		}
		l, ok := Lookup(tag)
		if !ok {
			t.Errorf("%08x : ancre Theater sans etiquette", tag)
			continue
		}
		if l.Class != want.class {
			t.Errorf("%08x : classe %q, attendue %q", tag, l.Class, want.class)
		}
		if want.name != "" && l.Name != want.name {
			t.Errorf("%08x : nom %q, attendu %q", tag, l.Name, want.name)
		}
		if !l.Publishable() {
			t.Errorf("%08x : ancre Theater non publiable (statut %q)", tag, l.Status)
		}
	}
}

func TestLignesFaussesMarqueesEtNonPubliables(t *testing.T) {
	for _, tag := range tagsAmbigus {
		l, ok := Lookup(tag)
		if !ok {
			t.Errorf("%08x : ligne connue comme fausse, absente de la table", tag)
			continue
		}
		if l.Status != StatusAmbigu {
			t.Errorf("%08x : statut %q, attendu %q — l etiquette serait affirmative et fausse",
				tag, l.Status, StatusAmbigu)
		}
		if l.Publishable() {
			t.Errorf("%08x : marquee publiable alors qu elle est mesuree comme fausse", tag)
		}
		if l.Reserve == "" {
			t.Errorf("%08x : aucune reserve ecrite", tag)
		}
	}
}

func TestCoherenceDesLignes(t *testing.T) {
	classes := map[Class]bool{ClassArme: true, ClassMelee: true, ClassGrenade: true,
		ClassVehicule: true, ClassObjet: true, ClassGlobal: true, ClassInconnu: true}
	statuts := map[Status]bool{StatusValide: true, StatusReserve: true,
		StatusAmbigu: true, StatusInconnu: true}
	for _, tag := range IDs() {
		l, ok := Lookup(tag)
		if !ok {
			t.Fatalf("%08x : id du catalogue sans ligne d etiquette", tag)
		}
		if !classes[l.Class] {
			t.Errorf("%08x : classe inconnue %q", tag, l.Class)
		}
		if !statuts[l.Status] {
			t.Errorf("%08x : statut inconnu %q", tag, l.Status)
		}
		if l.Class == ClassArme && l.Name == "" {
			t.Errorf("%08x : classe ARME sans nom", tag)
		}
		if l.Class == ClassArme && l.Status != StatusReserve {
			t.Errorf("%08x : une arme nommee doit etre SOUS_RESERVE (le nom propre n est pas garanti), statut %q",
				tag, l.Status)
		}
		if l.Class == ClassInconnu && l.Status != StatusInconnu {
			t.Errorf("%08x : classe INCONNU avec statut %q", tag, l.Status)
		}
	}
}

// TestRepartitionParClasse : le controle negatif de RE_LOG 7ter.45 / ETAT_DE_L_ART 5.3, applique
// aux 468 `jpt!` du jeu. C est le garde-fou qui detecte une regeneration cassee : une regle de
// nommage qui deraille deplace des dizaines de lignes d une classe a l autre.
func TestRepartitionParClasse(t *testing.T) {
	want := map[Class]int{ClassInconnu: 206, ClassArme: 114, ClassVehicule: 89,
		ClassObjet: 19, ClassGrenade: 17, ClassMelee: 14, ClassGlobal: 9}
	got := map[Class]int{}
	for _, tag := range IDs() {
		l, _ := Lookup(tag)
		got[l.Class]++
	}
	for c, n := range want {
		// Bornes larges (+-10 %) : les tables grandissent, mais un basculement de regle se voit.
		lo, hi := n*9/10, n*11/10+1
		if got[c] < lo || got[c] > hi {
			t.Errorf("classe %s : %d lignes, attendu ~%d (mesure du 2026-07-26, tolerance 10 %%)",
				c, got[c], n)
		}
	}
}

func TestStrong(t *testing.T) {
	// Le regime faible existe et porte des tags REELS : on ne peut pas l exclure du scan, mais
	// il ne partage pas le taux de faux positif du regime fort (1.15 % contre 1/9 000 000).
	if Strong(0x0000b29c) {
		t.Error("0000b29c (BR75) est dans le regime faible, Strong() ne doit pas le dire fort")
	}
	if !Strong(0x5eaa815c) {
		t.Error("5eaa815c (Fuel Rod SPNKr) est dans le regime fort")
	}
}

func TestLookupHorsCatalogue(t *testing.T) {
	// Valeur choisie hors catalogue : un id `jpt!` ne vaut jamais 0.
	if IsDamageEffect(0) {
		t.Error("0 ne doit pas etre un jpt!")
	}
	if _, ok := Lookup(0); ok {
		t.Error("Lookup(0) ne doit rien rendre")
	}
}
