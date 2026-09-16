package grammar

// i0_catalogue_mutation_test.go — LOT 1.9.2 : LE CATALOGUE DÉCIDE, ET LE FAUSSER SE VOIT.
//
// # CE QUE CE FICHIER TIENT
//
// Le découpage d'i0 est une DONNÉE DE PROFIL (D-3 d'ADR 0034) : il vient du catalogue de carte
// (`MapQuantEntry.Layout`), jamais d'une mesure sur le film. Un test de non-régression qui se
// contenterait de compter des positions ne dirait PAS d'où vient le découpage : tant que
// l'auto-détection décide, fausser le catalogue ne change rien et le vert ne prouve rien.
//
// Ce fichier joue donc la MUTATION que le lot exige : une entrée de catalogue décalée d'UN BIT
// doit faire changer la lecture.
//
// # LES BITS SONT LA RÉFÉRENCE, LE CATALOGUE EST LA VARIABLE
//
// C'EST TOUT LE POINT, et la première version de ce fichier l'avait manqué : elle écrivait les
// bits SOUS le découpage du catalogue, puis les relisait sous le même — écriture et lecture
// bougeaient ensemble, et fausser le catalogue restait VERT (mutation jouée le 2026-09-15,
// `axisWidths` de Live Fire 12 -> 13 : deux paquets verts). Les bits sont donc écrits sous un
// découpage FIGÉ dans ce fichier — la valeur du catalogue au 2026-09-15, avec sa source — comme
// un film de Live Fire l'aurait fait ; c'est l'ENTRÉE DE CATALOGUE qui varie.
//
// Ils sont SYNTHÉTIQUES, et écrits avec la grammaire du dépôt (`writeBipedHeaderEtMasque`,
// offline_biped_test.go — jamais une copie : un test qui re-décrirait la grammaire testerait sa
// propre copie). Les mini-bobines versionnées ne pouvaient pas servir : elles ne portent AUCUN
// paquet delta (mesure du lot 1.9.1 bis, §5 du plan), donc aucune position de bipède. Les bits
// écrits ici sont d'ailleurs plus forts qu'un film — la région de chaque enregistrement est
// CHOISIE, donc ce que la porte doit écarter est connu à l'avance.
//
// # POURQUOI LIVE FIRE
//
// Seule carte du catalogue dont la région jouée n'est pas la première du bloc structure-BSP :
// 4 régions déclarées, arène en région 1, index sur DEUX bits (`gate=6 region=1 12/12/11`).
// L'auto-détection ne sait pas voir un index de plus d'un bit et rend `gate=5 region=0 13/12/11`
// — MÊME longueur totale d'i0 (41 bits), donc la marche avance pareil, mais la porte de région
// ne teste qu'un bit contre zéro et accepte AUSSI des enregistrements d'une AUTRE région, dont
// les quanta sont exprimés dans une autre AABB.
//
//	go test ./internal/games/halo_infinite/film/filmdec/ -run I0Catalogue -v -count=1

import (
	"testing"
)

// recordBipedSynthetique : un enregistrement bipède écrit à la main, sous UN découpage donné.
//
// STRUCTURE PLUTÔT QUE SEPT PARAMÈTRES (seuil du dépôt : 5), et surtout : le découpage et la
// région sont des CHAMPS, ce qui est tout l'objet du fichier — on écrit les bits d'une carte, on
// les relit sous un découpage qui peut être le bon ou le muté.
type recordBipedSynthetique struct {
	// Lay est le découpage sous lequel les bits sont ÉCRITS.
	Lay I0Layout
	// Region est la valeur écrite dans le champ d'index de région (largeur `GateBits - 4`).
	Region uint64
	Slot   uint32
	// Tag est la génération du handle (1 = ce que `RequireTag1` exige) ; MaskCount le nombre
	// d'index annoncés par le masque.
	Tag, MaskCount uint64
	// Q sont les trois quanta d'axe.
	Q [3]uint64
}

// ecrire pose l'en-tête, le masque et le composant i0 de cet enregistrement.
func (r recordBipedSynthetique) ecrire(w *bitWriter) {
	writeBipedHeaderEtMasque(w, r.Slot, r.Tag, r.MaskCount)
	const preGate = i0SpineBits + i0UseDefaultBits
	w.bits(0, preGate) // i0 absolu : spine + useDefault nuls
	w.bits(r.Region, r.Lay.GateBits-preGate)
	for ax := 0; ax < 3; ax++ {
		w.bits(r.Q[ax], int(r.Lay.AxisW[ax]))
	}
}

// decoupageDeReferenceLiveFire : le découpage sous lequel le témoin de bits est ÉCRIT.
//
// C'EST LA VALEUR DU CATALOGUE AU 2026-09-15 (`map_quant_bounds.json`, entrée « live fire » :
// `axisWidths [12 12 11]`, `region 1`, `regionIndexBits 2`), FIGÉE ICI parce qu'elle joue le rôle
// du FILM : les octets d'un film de Live Fire ne changent pas quand le catalogue change. Si le
// catalogue venait à dire autre chose pour cette carte, `TestI0CatalogueEstLaSourceDuDecoupage`
// rougit — et c'est le comportement voulu : un changement de découpage de carte est une décision,
// pas un effet de bord.
var decoupageDeReferenceLiveFire = I0Layout{GateBits: 6, AxisW: [3]uint{12, 12, 11}, Region: 1}

// liveFireEntry rend l'entrée de catalogue de Live Fire, LUE DANS LE CATALOGUE VERSIONNÉ.
func liveFireEntry(t *testing.T) MapQuantEntry {
	t.Helper()
	cat, err := LoadMapQuantCatalog(e191bCatalogue())
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	e, err := cat.Lookup("Live Fire")
	if err != nil {
		t.Fatalf("entree Live Fire : %v", err)
	}
	return e
}

// slotTemoin : le slot des cinq enregistrements du témoin.
const slotTemoin uint32 = 517

// payloadTemoin écrit CINQ enregistrements sous [decoupageDeReferenceLiveFire] : trois dans
// l'arène (région 1) et deux dans d'autres régions (0 et 2). Une lecture juste n'en rend que
// trois.
func payloadTemoin() []byte {
	lay := decoupageDeReferenceLiveFire
	w := &bitWriter{}
	w.bits(0, 11) // bruit de tête : le balayage est bit à bit, pas aligné octet
	for _, r := range []recordBipedSynthetique{
		{Lay: lay, Region: 1, Slot: slotTemoin, Tag: 1, MaskCount: 3, Q: [3]uint64{1000, 1200, 700}},
		{Lay: lay, Region: 0, Slot: slotTemoin, Tag: 1, MaskCount: 3, Q: [3]uint64{2000, 900, 500}},
		{Lay: lay, Region: 1, Slot: slotTemoin, Tag: 1, MaskCount: 3, Q: [3]uint64{1010, 1210, 705}},
		{Lay: lay, Region: 2, Slot: slotTemoin, Tag: 1, MaskCount: 3, Q: [3]uint64{3000, 800, 400}},
		{Lay: lay, Region: 1, Slot: slotTemoin, Tag: 1, MaskCount: 3, Q: [3]uint64{1020, 1220, 710}},
	} {
		r.ecrire(w)
	}
	w.bits(0, 64) // queue : le balayage doit pouvoir dépasser le dernier enregistrement
	return w.buf
}

// lireSousDecoupage rejoue le balayage PUR du dépôt (`ScanBipedRecords`) sur le témoin de bits,
// sous le découpage `lu`.
func lireSousDecoupage(lu I0Layout, rng Vec3Range) []BipedPosition {
	opt := DefaultScanFilmOptions()
	opt.WorldRange = &rng
	return ScanBipedRecords(payloadTemoin(), NewSlotBand(map[uint32]bool{slotTemoin: true}), lu, opt,
		ContexteParDefaut())
}

// TestI0CatalogueEstLaSourceDuDecoupage : l'entrée de catalogue de Live Fire dit EXACTEMENT ce
// sous quoi le témoin de bits a été écrit. C'est le premier maillon — sans lui, toutes les
// lectures de ce fichier compareraient un film à un référentiel qui a bougé.
func TestI0CatalogueEstLaSourceDuDecoupage(t *testing.T) {
	e := liveFireEntry(t)
	if got := e.Layout(); got != decoupageDeReferenceLiveFire {
		t.Fatalf("le catalogue donne %s a Live Fire, le temoin de bits est ecrit sous %s.\n"+
			"Si le catalogue a change VOLONTAIREMENT (nouvelles bornes mesurees, region corrigee), "+
			"reecrire `decoupageDeReferenceLiveFire` dans le meme commit et dire pourquoi. Sinon, "+
			"c'est le catalogue qui est faux : le decoupage d'une carte ne change pas tout seul.",
			got, decoupageDeReferenceLiveFire)
	}
	if e.EffectiveRegionIndexBits() < 2 {
		t.Fatalf("Live Fire porte %d bit(s) d'index de region : ce fichier exige la SEULE carte a "+
			"plus de deux regions ; si le catalogue a change, choisir une autre carte temoin",
			e.EffectiveRegionIndexBits())
	}
}

// TestI0CatalogueEcarteLesAutresRegions : sous le découpage DU CATALOGUE, seuls les trois
// enregistrements de la région jouée sont lus ; sous celui que l'auto-détection rend, la porte
// d'un seul bit en laisse passer davantage.
//
// C'EST LE DÉFAUT DU 2026-09-03 REPRODUIT SUR DES BITS CHOISIS : sur `60ae07c4`, cette même porte
// acceptait 26 enregistrements sur 267 400 qui n'appartiennent pas à l'arène (mesure du lot
// 1.9.2, §5 du plan).
func TestI0CatalogueEcarteLesAutresRegions(t *testing.T) {
	e := liveFireEntry(t)
	catalogue := e.Layout()
	rng := e.Range()

	got := lireSousDecoupage(catalogue, rng)
	if len(got) != 3 {
		t.Fatalf("decoupage du catalogue (%s) : %d enregistrement(s) lus, attendu 3 (les seuls de "+
			"la region jouee)", catalogue, len(got))
	}

	// LE DÉCOUPAGE QUE L'AUTO-DÉTECTION REND SUR CETTE CARTE : même longueur totale d'i0, porte
	// d'un seul bit, région attendue 0 (cf. i0_layout.go). Écrit ici comme TÉMOIN DE MESURE, pas
	// comme une valeur de production.
	detecte := I0Layout{GateBits: DefaultI0GateBits, AxisW: [3]uint{
		decoupageDeReferenceLiveFire.AxisW[0] + 1,
		decoupageDeReferenceLiveFire.AxisW[1],
		decoupageDeReferenceLiveFire.AxisW[2]}}
	if detecte.TotalBits() != catalogue.TotalBits() {
		t.Fatalf("le temoin de detection ne fait pas la meme longueur d'i0 (%d contre %d) : la "+
			"comparaison ne porterait plus sur la seule porte de region",
			detecte.TotalBits(), catalogue.TotalBits())
	}
	if laxiste := lireSousDecoupage(detecte, rng); len(laxiste) <= len(got) {
		t.Errorf("la porte d'un seul bit lit %d enregistrement(s), le catalogue %d : le temoin "+
			"n'expose plus la difference que ce lot ferme", len(laxiste), len(got))
	}
}

// TestI0CatalogueMutationDUnBitFaitRougir : LA MUTATION DU LOT. Une entrée de catalogue dont
// `axisWidths` est décalée d'un bit, ou dont `regionIndexBits` est rabaissée, ne lit PLUS les
// mêmes enregistrements du MÊME témoin de bits — donc le catalogue décide bel et bien, et
// l'auto-détection ne peut plus « rattraper » en silence.
func TestI0CatalogueMutationDUnBitFaitRougir(t *testing.T) {
	e := liveFireEntry(t)
	rng := e.Range()
	reference := lireSousDecoupage(e.Layout(), rng)
	if len(reference) == 0 {
		t.Fatal("la lecture de reference est vide : la mutation ne prouverait rien")
	}

	for _, cas := range []struct {
		nom    string
		muter  func(*MapQuantEntry)
		raison string
	}{
		{"axisWidths X decale d'un bit", func(m *MapQuantEntry) { m.AxisWidths[0]++ },
			"un bit de plus sur X double le pas de quantification et decale les deux autres axes"},
		{"axisWidths Z decale d'un bit", func(m *MapQuantEntry) { m.AxisWidths[2]-- },
			"la longueur totale d'i0 change : la marche n'avance plus au meme endroit"},
		{"regionIndexBits rabaissee a 1", func(m *MapQuantEntry) { m.RegionIndexBits = 1 },
			"la porte de region ne teste plus qu'un bit : elle laisse passer une autre region"},
		{"region attendue remise a 0", func(m *MapQuantEntry) { m.Region = 0 },
			"la porte attend la region 0 : elle ecarte l'arene et garde ce qui n'en est pas"},
	} {
		mute := e
		cas.muter(&mute)
		if got := lireSousDecoupage(mute.Layout(), rng); memeLecture(reference, got) {
			t.Errorf("%s : la lecture est IDENTIQUE a la reference (%d enregistrements) — le "+
				"decoupage ne vient donc pas du catalogue. %s", cas.nom, len(got), cas.raison)
		}
	}
}

// memeLecture compare deux lectures sur ce qui compte ici : le nombre d'enregistrements et leurs
// quanta.
func memeLecture(a, b []BipedPosition) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Slot != b[i].Slot || a[i].Q != b[i].Q {
			return false
		}
	}
	return true
}
