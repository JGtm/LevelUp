package grammar

// object_deaths_test.go — LA MARCHE DES MORTS D OBJET, eprouvee sur une bobine REELLE et sur des
// entrees degradees.
//
// LA BOBINE EST CELLE DE `killsource` (`../killsource/testdata/minibobine_000d5950`), et c est
// voulu : c est la SEULE fixture versionnee du depot qui porte un PREFIXE CONTIGU de chunks avec
// leurs paquets delta. Les sept mini-bobines de `replay/testdata` ne portent que des paquets
// d image-cle et le pied — une marche n y a rien a derouler, et une bobine de paquets
// cherry-pickes n y decode AUCUNE mort (mesure inscrite dans leur PROVENANCE.txt).
//
// AUCUNE DE CES ENTREES NE VIENT DU CACHE DE FILMS : le paquet reste jouable sans le parc.

import (
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/source"
)

// bobineMarcheDir : la bobine a paquets delta, chez `killsource`.
func bobineMarcheDir() string {
	return filepath.Join("..", "killsource", "testdata", "minibobine_000d5950")
}

// TestScanObjectDeathsSurBobineReelle : la marche rend-elle des morts BIPEDES sur une bobine qui
// en porte, et zero mort de VEHICULE sur une carte d arene qui n en declare aucun ?
//
// LES DEUX MOITIES COMPTENT. La premiere dit que la marche marche (une mort de bipede est un
// fait du corpus) ; la seconde est le TEMOIN NEGATIF du gate G2 du lot V13 — un instrument qui
// fabrique des vehicules la ou le film n en declare pas est un instrument faux.
func TestScanObjectDeathsSurBobineReelle(t *testing.T) {
	film, err := source.LoadDir(bobineMarcheDir(), nil)
	if err != nil {
		t.Fatalf("bobine illisible : %v", err)
	}
	fc := NewFilmContext(film)
	deaths, st, err := ScanObjectDeaths(fc)
	if err != nil {
		t.Fatalf("ScanObjectDeaths : %v", err)
	}
	if st.Deltas == 0 {
		t.Fatalf("la bobine ne porte aucun paquet delta : la marche n a rien a derouler")
	}
	t.Logf("cadre retenu idLow=%d amorce=%d · %d images-cles · %d paquets delta · %d marches"+
		" (%d a events, %d localises) · %d morts",
		st.Config.IDLowBits, st.Config.PacketPreambleBits, st.Keyframes, st.Deltas,
		st.Packets, st.EventPackets, st.LocatedPackets, len(deaths))
	parTI := map[uint32]int{}
	for _, d := range deaths {
		parTI[d.TypeIndex]++
	}
	const bipedTI = uint32(35)
	if parTI[bipedTI] == 0 {
		t.Errorf("0 mort de bipede sur une bobine qui en porte : la marche n atteint pas `i11`"+
			" (records bipedes marches %d, propres %d, masques declarant le dead-state %d)",
			st.Records[bipedTI], st.CleanRecords[bipedTI], st.MaskDeclared[bipedTI])
	}
	if n := parTI[uint32(VehicleTypeIndex)]; n != 0 {
		t.Errorf("%d morts de vehicule sur une carte d arene sans `ti=40` : l instrument en"+
			" fabrique", n)
	}
	if st.Records[bipedTI] == 0 {
		t.Errorf("aucun record bipede marche : les denominateurs de couverture sont muets")
	}
}

// TestScanObjectDeathsSansPaquetDelta : une bobine qui ne porte que des images-cles et le pied
// rend une liste VIDE et AUCUNE erreur — « rien a derouler » n est pas une panne.
func TestScanObjectDeathsSansPaquetDelta(t *testing.T) {
	dir := filepath.Join("..", "replay", "testdata", "minifilm_e5adf7b2")
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("mini-bobine illisible : %v", err)
	}
	deaths, st, err := ScanObjectDeaths(NewFilmContext(film))
	if err != nil {
		t.Fatalf("ScanObjectDeaths : %v", err)
	}
	if len(deaths) != 0 {
		t.Errorf("%d morts rendues sans un seul paquet delta", len(deaths))
	}
	if st.Deltas != 0 {
		t.Errorf("Deltas=%d : la mini-bobine est censee n en porter aucun", st.Deltas)
	}
}

// TestMarcheSurPayloadTronque : LA BORNE DE BOUCLE, eprouvee sur une entree coupee.
//
// Le localisateur et la marche bornent leurs boucles sur `len(payload)*8`. Un payload TRONQUE —
// un chunk coupe, un paquet dont l en-tete ment — doit rendre un resultat, jamais paniquer. Le
// test coupe un payload REEL a toutes les longueurs d une grille, y compris zero et un octet.
func TestMarcheSurPayloadTronque(t *testing.T) {
	film, err := source.LoadDir(bobineMarcheDir(), nil)
	if err != nil {
		t.Fatalf("bobine illisible : %v", err)
	}
	fc := NewFilmContext(film)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	_, deltas := marchPacketsOf(fc)
	if len(deltas) == 0 {
		t.Fatalf("aucun paquet delta a tronquer")
	}
	pay := deltas[0].payload
	cfg := DefaultFrameConfig()
	for _, n := range []int{0, 1, 2, 3, 7, 16, 64, len(pay) / 2, len(pay) - 1} {
		if n < 0 || n > len(pay) {
			continue
		}
		w := NewWorld(reg)
		coupe := pay[:n]
		start, _, ok := marchStartOf(coupe, w, cfg)
		if !ok {
			continue
		}
		if recs := marchRecordsOf(coupe, w, cfg, start); len(recs) < 0 {
			t.Fatalf("longueur negative")
		}
	}
}

// TestAcceptationQueueDesynchronisee : LA REGLE DU LOT, sur la table de decision.
//
// Un dead-state est accepte quand le record est entierement porte, OU quand la rupture est
// STRICTEMENT APRES l index du composant. Une rupture AVANT ou SUR le composant le rejette : les
// bits n ont pas ete lus au bon endroit. C est la difference entre lire 21 a 27 morts de
// vehicule par film et n en lire aucune.
func TestAcceptationQueueDesynchronisee(t *testing.T) {
	const ti = uint32(40)
	h := &objectDeathHarvest{idx: map[uint32]int{ti: 11}, st: &ObjectDeathStats{}}
	for _, cas := range []struct {
		nom        string
		desyncAt   int
		veutOK     bool
		veutQueue  bool
		pourquoiOK string
	}{
		{"record entierement porte", -1, true, false, "rien n a rompu"},
		{"rupture a i30 (apres i11)", 30, true, true, "la tete est lue, la queue est inconnue"},
		{"rupture a i12 (apres i11)", 12, true, true, "la tete est lue"},
		{"rupture a i11 (SUR le composant)", 11, false, false, "le composant lui-meme n a pas ete porte"},
		{"rupture a i5 (avant i11)", 5, false, false, "le curseur est decale avant le composant"},
		{"rupture a i0", 0, false, false, "rien n a ete consomme"},
	} {
		r := &FrameRecord{TypeIndex: ti, DesyncAt: cas.desyncAt}
		queue, ok := h.accept(r)
		if ok != cas.veutOK || queue != cas.veutQueue {
			t.Errorf("%s : accept = (queue=%v, ok=%v), attendu (queue=%v, ok=%v) — %s",
				cas.nom, queue, ok, cas.veutQueue, cas.veutOK, cas.pourquoiOK)
		}
	}
}

// TestAcceptationSansComposantAuRegistre : un archetype dont le registre ne nomme PAS le
// dead-state ne beneficie jamais de la tolerance de queue — sans index, « apres » n a pas de
// sens, et accepter serait supposer.
func TestAcceptationSansComposantAuRegistre(t *testing.T) {
	h := &objectDeathHarvest{idx: map[uint32]int{7: -1}, st: &ObjectDeathStats{}}
	if _, ok := h.accept(&FrameRecord{TypeIndex: 7, DesyncAt: 30}); ok {
		t.Errorf("un record desynchronise d un archetype sans dead-state au registre a ete accepte")
	}
	if _, ok := h.accept(&FrameRecord{TypeIndex: 7, DesyncAt: -1}); !ok {
		t.Errorf("un record ENTIEREMENT PORTE a ete refuse : le filtre strict doit rester ouvert")
	}
}

// TestDedupObjectDeaths : une entite ne meurt qu une fois a un instant donne, et la QUALITE LA
// MEILLEURE gagne — plusieurs vues de replication republient le meme dead-state, et une vue
// tardive qui rompt ne doit pas degrader une lecture propre.
func TestDedupObjectDeaths(t *testing.T) {
	in := []ObjectDeath{
		{TimestampUS: 100, Slot: 777, Gen: 1, TailDesync: true},
		{TimestampUS: 100, Slot: 777, Gen: 1, TailDesync: false},
		{TimestampUS: 100, Slot: 778, Gen: 1},
		{TimestampUS: 200, Slot: 777, Gen: 1},
		{TimestampUS: 100, Slot: 777, Gen: 2},
	}
	out := dedupObjectDeaths(in)
	if len(out) != 4 {
		t.Fatalf("%d morts apres deduplication, attendu 4 : %+v", len(out), out)
	}
	for _, d := range out {
		if d.TimestampUS == 100 && d.Slot == 777 && d.Gen == 1 && d.TailDesync {
			t.Errorf("la version a QUEUE INCONNUE a ete retenue alors qu une lecture propre existait")
		}
	}
	for i := 1; i < len(out); i++ {
		if out[i-1].TimestampUS > out[i].TimestampUS {
			t.Errorf("la sortie n est pas triee par instant : %+v", out)
		}
	}
}

// TestProfilDuCadreEstDomineOuDeclare — LE GARDE-FOU DE LA CALIBRATION, sur les deux bobines
// versionnees qui portent des paquets delta.
//
// CE QU IL TIENT : la largeur retenue est soit DOMINANTE (elle bat son dauphin d un facteur
// `calibDominationMin` sur les paquets a evenements LOCALISES), soit DECLAREE par defaut. Le
// silence — retenir une largeur au departage par records propres, critere REFUTE par le lot V13 —
// n est plus une issue possible.
//
// LA TABLE EST LA MESURE, et elle est collee au journal du test : le verdict n en est que la
// conclusion.
func TestProfilDuCadreEstDomineOuDeclare(t *testing.T) {
	for _, cas := range []struct {
		dir        string
		veutDefaut bool
		pourquoi   string
	}{
		{bobineMarcheDir(), false,
			"profil franc mesure le 2026-09-16 : idLow=13 localise 142/147, le dauphin 23/147 (facteur 6,2)"},
		{filepath.Join("..", "killsource", "testdata", "minibobine_e5adf7b2"), true,
			"profil PLAT mesure le 2026-09-16 : les six largeurs localisent 0 paquet sur 54"},
	} {
		film, err := source.LoadDir(cas.dir, nil)
		if err != nil {
			t.Fatalf("%s : bobine illisible : %v", filepath.Base(cas.dir), err)
		}
		fc := NewFilmContext(film)
		reg, err := fc.Registry()
		if err != nil {
			t.Fatalf("%s : registre illisible : %v", filepath.Base(cas.dir), err)
		}
		kfs, deltas := marchPacketsOf(fc)
		cfg, parDefaut, meilleur, dauphin := calibrateFrameConfig(reg, kfs, deltas, fc.CadreDeBalayage())
		t.Logf("%s : %d paquets delta · retenu idLow=%d amorce=%d · localises %d/%d · dauphin %d"+
			" · cadre par defaut : %v", filepath.Base(cas.dir), len(deltas), cfg.IDLowBits,
			cfg.PacketPreambleBits, meilleur.located, meilleur.events, dauphin.located, parDefaut)
		if parDefaut != cas.veutDefaut {
			t.Errorf("%s : cadreParDefaut=%v, attendu %v — %s",
				filepath.Base(cas.dir), parDefaut, cas.veutDefaut, cas.pourquoi)
		}
		if cfg.PacketPreambleBits != DefaultPacketPreambleBits {
			t.Errorf("%s : amorce=%d — l amorce est une propriete PROUVEE du format"+
				" (DefaultPacketPreambleBits = %d), elle ne se balaye pas (D13)",
				filepath.Base(cas.dir), cfg.PacketPreambleBits, DefaultPacketPreambleBits)
		}
		if parDefaut && cfg.IDLowBits != DefaultFrameConfig().IDLowBits {
			t.Errorf("%s : profil plat mais largeur %d retenue au lieu du defaut %d — on ne devine pas",
				filepath.Base(cas.dir), cfg.IDLowBits, DefaultFrameConfig().IDLowBits)
		}
	}
}
