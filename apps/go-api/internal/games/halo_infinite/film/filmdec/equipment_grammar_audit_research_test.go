package filmdec

// equipment_grammar_audit_research_test.go — LOT 5, ADDENDUM A : AUDIT DE COMPLÉTUDE de la
// grammaire côté ÉQUIPEMENT, par le REGISTRE DE RÉPLICATION pris comme oracle.
//
// ## LE DOUTE À INSTRUIRE, ÉNONCÉ TEL QU'IL A ÉTÉ POSÉ
//
// « Je trouve étrange que le film contienne une grammaire d'armes et pas d'équipement — on
// pensait nos grammaires complètes, c'est peut-être faux. »
//
// C'est une hypothèse de plein rang et elle se teste, parce que le film porte son propre
// INVENTAIRE : le registre de `chunk_00` déclare, archétype par archétype, la liste ORDONNÉE
// des composants répliqués (49 archétypes porteurs, 1 067 couples — inventaire confirmé sur le
// corpus entier, cf. NOTE_COMPTE_REGISTRE_2026-08-30). Tout composant DÉCLARÉ par le film et
// que notre décodeur ne sait pas consommer est un trou de grammaire, et le registre le dit
// sans qu'on ait à le deviner.
//
// ## L'ORACLE DE « DÉCODÉ », ET POURQUOI CE N'EST PAS LA TABLE TSV
//
// Le dépôt versionne déjà `testdata/ecs_table.tsv` et trois garde-fous (G1 code<->table,
// G2 film<->table, G3 table<->document). Les rejouer prouve que la table est à jour ; cela ne
// prouve pas, à soi seul, que le CODE consomme un composant donné — c'est G1 qui fait ce lien,
// par lecture d'AST.
//
// Cet instrument prend le chemin le plus court et le plus direct : il appelle `consumeByName`,
// le dispatcheur du traverseur lui-même. Sa branche `default` rend `ported = false` — c'est
// littéralement la définition de « le décodeur ne connaît pas ce composant ». Aucune
// indirection, aucune table à croire.
//
// PRÉCAUTION DE MESURE : le lecteur de bits reçoit un tampon de zéros généreux, pour qu'aucun
// `ported = false` ne vienne d'un manque de bits plutôt que d'un manque de décodeur. Le
// contrôle NÉGATIF ci-dessous vérifie que l'instrument sait dire non.
//
// ## LES TROIS ARCHÉTYPES CONFRONTÉS, ET POURQUOI CEUX-LÀ
//
//	ti=42  ground-weapon    la RÉFÉRENCE : c'est l'archétype dont on dit la grammaire complète
//	ti=37  equipment/item   le SUSPECT : l'archétype des objets d'équipement posés
//	ti=35  biped-spartan    le porteur : c'est lui qui tient grenades et capacités
//
// Le point décisif de l'audit est l'existence d'une FEUILLE D'IDENTITÉ sur ti=37, analogue de
// celle que ti=42 porte pour une arme au sol. Elle est donc cherchée nommément.
//
// Garde `BIPED_PICKUP_FILM` : sans film, l'instrument saute (aucun effet en CI).

import (
	"os"
	"sort"
	"testing"
)

// egaFilmEnv — la même garde que tout le chantier ramassage.
const egaFilmEnv = "BIPED_PICKUP_FILM"

// egaArchetypes — les archétypes audités, avec le rôle que l'audit leur donne.
var egaArchetypes = []struct {
	TI   int
	Role string
}{
	{42, "ground-weapon — LA RÉFÉRENCE (grammaire réputée complète)"},
	{37, "equipment / item — LE SUSPECT"},
	{35, "biped-spartan — LE PORTEUR (grenades, capacités)"},
}

// egaIdentite est le composant qui porte l'IDENTITÉ d'un objet du monde : le mot de 32 bits
// dont le manifeste du titre tire le GlobalID de tag. Le chantier des socles l'a établi pour
// ti=42 (arme au sol) et le chantier des poses pour ti=37 (`eqip`) — la question de l'audit est
// de savoir si les deux archétypes le déclarent bel et bien tous les deux.
const egaIdentite = "object-multiplayer-properties-component"

// egaBits est la taille du tampon de zéros servi à chaque décodeur, en octets. Large devant
// tout composant connu (le plus gourmand, i0 en forme lourde, consomme 59 bits).
const egaBits = 4096

// TestAuditGrammaireEquipementParLeRegistre — ADDENDUM A. Confronter, archétype par archétype,
// ce que le FILM DÉCLARE à ce que le DÉCODEUR SAIT LIRE.
//
// VERDICTS ÉCRITS AVANT LA MESURE :
//
//	A1 — ti=37 ne porte AUCUN composant déclaré que `consumeByName` refuse. Un seul refus
//	     suffirait à donner raison au doute : ce serait un trou de grammaire côté équipement.
//	A2 — ti=37 déclare la feuille d'identité `object-multiplayer-properties-component`,
//	     comme ti=42. Si elle manquait à l'un des deux, l'asymétrie serait structurelle.
//	A3 — CONTRÔLE NÉGATIF : un nom de composant inventé DOIT être refusé. Sans lui, « zéro
//	     refus » ne vaut rien — l'instrument pourrait dire oui à tout.
//	A4 — le même audit sur ti=35 et ti=42, pour situer ti=37 : si les trois archétypes ont
//	     le même taux, il n'y a pas d'asymétrie du tout.
func TestAuditGrammaireEquipementParLeRegistre(t *testing.T) {
	dir := os.Getenv(egaFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", egaFilmEnv)
	}
	// ReadFilmChunk DECOMPRESSE : ParseRegistryChunk ne le fait plus (lot 1).
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 illisible : %v", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	t.Logf("== ADDENDUM A — AUDIT DE COMPLÉTUDE PAR LE REGISTRE · %s ==", dir)

	// A3 d'abord : un instrument qui ne sait pas dire non ne mesure rien.
	if _, _, ported := consumeByName(NewBitReader(make([]byte, egaBits)),
		"composant-qui-n-existe-pas-component", 37, 0); ported {
		t.Fatal("CONTRÔLE NÉGATIF EN ÉCHEC : le dispatcheur accepte un composant inventé — l'audit ne prouverait rien")
	}
	t.Log("VERDICT A3 (contrôle négatif) : TENU — un composant inventé est refusé")

	type bilan struct {
		declares, consommes int
		refuses             []string
		identite            bool
	}
	bilans := map[int]*bilan{}
	for _, a := range egaArchetypes {
		arch, ok := reg.Archetype(a.TI)
		if !ok {
			t.Fatalf("archétype %d absent du registre de ce film", a.TI)
		}
		b := &bilan{}
		bilans[a.TI] = b
		t.Logf("-- ti=%d · %s · %d composant(s) déclaré(s) --", a.TI, a.Role, len(arch.Components))
		for i, name := range arch.Components {
			b.declares++
			if name == egaIdentite {
				b.identite = true
			}
			_, _, ported := consumeByName(NewBitReader(make([]byte, egaBits)), name, uint32(a.TI), arch.Level(i))
			if ported {
				b.consommes++
				continue
			}
			b.refuses = append(b.refuses, name)
			t.Logf("   i=%-3d %-56s  << DÉCLARÉ, NON CONSOMMÉ >>", i, name)
		}
		t.Logf("   bilan ti=%d : %d/%d consommés (%.1f %%) · feuille d'identité %q : %v",
			a.TI, b.consommes, b.declares,
			100*float64(b.consommes)/float64(max(b.declares, 1)), egaIdentite, b.identite)
	}

	e37, e42, e35 := bilans[37], bilans[42], bilans[35]
	t.Logf("VERDICT A1 (ti=37 sans aucun composant déclaré non consommé) : %v — %d refus %v",
		len(e37.refuses) == 0, len(e37.refuses), e37.refuses)
	t.Logf("VERDICT A2 (feuille d'identité déclarée sur ti=37 ET ti=42) : %v — ti=37 %v · ti=42 %v",
		e37.identite && e42.identite, e37.identite, e42.identite)
	t.Logf("VERDICT A4 (pas d'asymétrie entre les trois) : ti=42 %d/%d · ti=37 %d/%d · ti=35 %d/%d",
		e42.consommes, e42.declares, e37.consommes, e37.declares, e35.consommes, e35.declares)
}

// TestAuditRegistreComposantsEquipement — le VOCABULAIRE que le registre déclare, côté
// équipement et grenades, TOUS archétypes confondus.
//
// POURQUOI CETTE SECONDE PASSE. A1/A2 répondent « le décodeur lit-il tout ce que ti=37
// déclare ? ». Cette passe-ci répond à la question voisine, et c'est elle qui pourrait
// révéler une grammaire ignorée : « le registre déclare-t-il, QUELQUE PART, un composant qui
// parlerait d'équipement ou de grenades et que nous n'aurions jamais regardé ? ». Le registre
// est un espace de noms EN CLAIR : il se dépouille, il ne se devine pas.
//
// VERDICT A5 : tout composant dont le nom cite équipement/grenade/inventaire est consommé, et
// on publie la liste avec l'archétype porteur — le lecteur juge sur pièces.
func TestAuditRegistreComposantsEquipement(t *testing.T) {
	dir := os.Getenv(egaFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", egaFilmEnv)
	}
	// ReadFilmChunk DECOMPRESSE : ParseRegistryChunk ne le fait plus (lot 1).
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 illisible : %v", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}

	// Les racines cherchées. `pickup` et `inventory` sont là pour attraper un éventuel
	// composant de ramassage que personne n'aurait cherché.
	racines := []string{"equipment", "grenade", "inventory", "pickup", "ammo", "item-"}
	type porteur struct {
		ti, i int
	}
	vus := map[string][]porteur{}
	total, refuses := 0, 0
	var listeRefuses []string
	for _, arch := range reg.Archetypes {
		for i, name := range arch.Components {
			if !egaCite(name, racines) {
				continue
			}
			total++
			vus[name] = append(vus[name], porteur{arch.Index, i})
			if _, _, ported := consumeByName(NewBitReader(make([]byte, egaBits)),
				name, uint32(arch.Index), arch.Level(i)); !ported {
				refuses++
				listeRefuses = append(listeRefuses, name)
			}
		}
	}
	noms := make([]string, 0, len(vus))
	for n := range vus {
		noms = append(noms, n)
	}
	sort.Strings(noms)
	t.Logf("== ADDENDUM A (2e passe) — LE VOCABULAIRE ÉQUIPEMENT/GRENADE DU REGISTRE · %s ==", dir)
	t.Logf("%d nom(s) distinct(s) pour %d couple(s) archétype x composant", len(noms), total)
	for _, n := range noms {
		p := vus[n]
		tis := map[int]bool{}
		for _, x := range p {
			tis[x.ti] = true
		}
		keys := make([]int, 0, len(tis))
		for k := range tis {
			keys = append(keys, k)
		}
		sort.Ints(keys)
		t.Logf("   %-56s  x%-3d  archétypes %v", n, len(p), keys)
	}
	t.Logf("VERDICT A5 (tout composant du vocabulaire équipement/grenade est consommé) : %v — %d refus %v",
		refuses == 0, refuses, listeRefuses)
}

// egaCite dit si un nom de composant contient l'une des racines cherchées.
func egaCite(name string, racines []string) bool {
	for _, r := range racines {
		if len(r) <= len(name) && egaContient(name, r) {
			return true
		}
	}
	return false
}

// egaContient est un `strings.Contains` local — l'instrument n'importe rien pour si peu.
func egaContient(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
