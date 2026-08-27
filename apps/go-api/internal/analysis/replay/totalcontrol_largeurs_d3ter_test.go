package replay

// totalcontrol_largeurs_d3ter_test.go — D3-ter, VERROU 1 : LE BOUTON DES LARGEURS FAIT-IL
// QUELQUE CHOSE SUR `ti=13` ?
//
// LE PROTOCOLE EST ECRIT ET COMMITE AVANT CE FICHIER (`.ai/V7.5/PLAN_OBJECTIFS_ETAT_VIVANT_
// 2026-08.md`, section « D3-ter, VERROU 1 »). Ce qui suit l'applique.
//
// # POURQUOI MESURER UNE INDEPENDANCE QU'ON PEUT LIRE
//
// Le precedent M3 est reel : sur `ti=42`, l'identite lue aux largeurs MPP par defaut rendait
// ZERO socle en silence sur les films BTB. La lecture du code dit que `ti=13` n'a pas cette
// dependance — le bloc MPP appartient aux default-states de `ti=35/36/37/38/39/42/43`, et la
// grammaire de `ti=13` porte en propre « AUCUNE DESYNCHRONISATION N'EST POSSIBLE : la largeur
// est entierement determinee par 4 bits lus dans le flux ».
//
// MAIS UN ARGUMENT DE LECTURE PEUT MANQUER UN CHEMIN INDIRECT, et le lot precedent vient de
// montrer ce que coute de supposer : « le score d'Oddball monte a ~1 Hz » etait une supposition
// de plan, jamais mesuree, et fausse. On MESURE donc : meme film, TROIS decoupages, comparaison
// a l'unite pres. Un bouton qui ne change RIEN se prouve en le tournant.
//
// # LE SECOND VOLET NOMME LE VERROU REEL
//
// L'ancrage de `scanPayload` balaie TOUTES les positions de bit et valide sur une signature
// FAIBLE. Il produit donc des faux ancrages en proportion de la taille du payload. Le
// discriminant, ecrit d'avance : les VRAIS records se concentrent sur peu de slots avec beaucoup
// de lectures ; les FAUX se dispersent sur beaucoup de slots avec une ou deux.
//
// REGIME : garde `ZONE_FILM` (repertoire de chunks), UN FILM PAR PROCESSUS, lecture seule,
// AUCUNE base.
//
//	$env:ZONE_FILM="<cache>/film_chunks/66aa5f0b"
//	go test ./internal/analysis/replay/ -run TotalControlLargeurs -v

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/filmproc"
)

const (
	// d3tSeuilChainage : le critere du verrou 1, fixe par le superviseur et recopie sans
	// modification. Au-dessus, le canal est declare lisible ; en dessous, arret.
	d3tSeuilChainage = 0.80
	// d3tTopSlots : combien de slots les plus fournis concentrent la part publiee au diagnostic.
	d3tTopSlots = 5
	// d3tSeuilConcentration : au-dela, le sous-ensemble chaine est CONCENTRE — signature de
	// vrais records, donc taux global bas = artefact d'ancrage et non lecture fausse.
	d3tSeuilConcentration = 0.80
)

// d3tLargeurs : les trois decoupages sondes. Le defaut, celui MESURE sur les films BTB du lot
// armes-au-sol, et un absurde — sans ce dernier, deux resultats identiques ne prouveraient rien
// (les deux premiers pourraient coincider par hasard sur ce film).
var d3tLargeurs = []filmdec.MPPWidths{{Lead: 9, Index: 5}, {Lead: 8, Index: 3}, {Lead: 12, Index: 7}}

// d3tReleve est ce qu'un balayage rend, reduit a ce qui se compare.
type d3tReleve struct {
	slots, records, walked, chained, lectures int
}

func (r d3tReleve) String() string {
	return fmt.Sprintf("slots=%d records=%d walked=%d chained=%d lectures=%d",
		r.slots, r.records, r.walked, r.chained, r.lectures)
}

// TestTotalControlLargeursMPP — LA SONDE DU VERROU 1. Un film par processus.
func TestTotalControlLargeursMPP(t *testing.T) {
	dir := p2aRequireFilm(t)
	short := filepath.Base(dir)
	g := filmproc.Arm("d3ter-largeurs", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE %s : %.2f Gio — sonde interrompue", short,
			float64(peak)/(1<<30))
	})
	defer func() {
		g.Disarm()
		t.Logf("%s : pic memoire observe %.2f Gio", short, float64(g.Peak())/(1<<30))
	}()

	// TEST A — LE MEME FILM SOUS TROIS DECOUPAGES.
	releves := make([]d3tReleve, 0, len(d3tLargeurs))
	var refScan filmdec.ManagedPropertyScan
	for i, w := range d3tLargeurs {
		sc, err := d3tScanSousLargeurs(dir, w)
		if err != nil {
			t.Fatalf("%s : balayage ti=13 sous largeurs %s : %v", short, w, err)
		}
		r := d3tReleve{slots: sc.Slots, records: sc.Records, walked: sc.Walked,
			chained: sc.Chained, lectures: len(sc.Reads)}
		releves = append(releves, r)
		t.Logf("%s : largeurs MPP %-5s -> %s (chainage %s)", short, w, r,
			d3tPart(r.chained, r.walked))
		if i == 0 {
			refScan = sc
		}
	}

	identiques := true
	for _, r := range releves[1:] {
		if r != releves[0] {
			identiques = false
		}
	}
	if identiques {
		t.Logf("TEST A %s : les TROIS decoupages rendent des releves IDENTIQUES A L'UNITE PRES. "+
			"Le bouton des largeurs MPP n'a AUCUN effet sur le balayage `ti=13` — l'hypothese du "+
			"precedent M3 est REFUTEE sur ce film, par la mesure et non par la lecture.", short)
	} else {
		t.Logf("TEST A %s : les decoupages DIFFERENT — `ti=13` depend donc bien des largeurs MPP, "+
			"contre ce que dit sa grammaire. C'est le resultat le plus interessant possible, et il "+
			"designe la calibration comme la voie a suivre.", short)
	}

	// TEST B — LE CRITERE DU VERROU 1, applique tel qu'ecrit.
	taux := d3tTaux(releves[0].chained, releves[0].walked)
	t.Logf("SIGNAL %s : chainage %s (seuil >= %.0f %%)", short,
		d3tPart(releves[0].chained, releves[0].walked), 100*d3tSeuilChainage)
	if taux >= d3tSeuilChainage {
		t.Logf("VERROU 1 %s : TENU — le canal est declare lisible sur ce film.", short)
	} else {
		t.Logf("VERROU 1 %s : NON TENU — sous le seuil.", short)
	}

	// TEST C — LE DIAGNOSTIC DU VERROU REEL.
	d3tDiagnostic(t, short, refScan)
}

// d3tScanSousLargeurs installe un decoupage MPP, balaye `ti=13`, et RESTAURE.
//
// LE VERROU DE PROCESSUS ET LA RESTAURATION SONT OBLIGATOIRES : ce sont des globaux de paquet, et
// une sonde qui les laisserait poses contaminerait tout ce qui suit dans le meme processus.
func d3tScanSousLargeurs(dir string, w filmdec.MPPWidths) (filmdec.ManagedPropertyScan, error) {
	release := filmdec.LockProcessDecode()
	defer release()
	prev := filmdec.SetMPPWidths(w)
	defer filmdec.SetMPPWidths(prev)
	return filmdec.ScanFilmManagedProperties(dir)
}

// d3tDiagnostic publie la CONCENTRATION du sous-ensemble chaine : le discriminant entre vrais
// records et faux ancrages, defini avant la mesure.
func d3tDiagnostic(t *testing.T, short string, sc filmdec.ManagedPropertyScan) {
	t.Helper()
	parSlot := map[uint32]int{}
	tag5 := map[uint32]map[uint64]bool{}
	chainees := 0
	for _, r := range sc.Reads {
		if !r.Chained {
			continue
		}
		chainees++
		parSlot[r.Slot]++
		if r.Field == filmdec.ManagedPropertyScalar && r.HasValue &&
			r.Tag == filmdec.ManagedPropertyTagStringID {
			if tag5[r.Slot] == nil {
				tag5[r.Slot] = map[uint64]bool{}
			}
			tag5[r.Slot][r.Value] = true
		}
	}
	comptes := make([]int, 0, len(parSlot))
	for _, n := range parSlot {
		comptes = append(comptes, n)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(comptes)))
	tete := 0
	for i := 0; i < len(comptes) && i < d3tTopSlots; i++ {
		tete += comptes[i]
	}
	valeurs := 0
	for _, v := range tag5 {
		valeurs += len(v)
	}
	t.Logf("DIAGNOSTIC %s : %d lecture(s) CHAINEE(s) sur %d, portees par %d slot(s) ; les %d "+
		"slots les plus fournis en portent %s (seuil de concentration %.0f %%) ; tag 5 chaine : "+
		"%d slot(s), %d valeur(s) distincte(s) au total",
		short, chainees, len(sc.Reads), len(parSlot), d3tTopSlots, d3tPart(tete, chainees),
		100*d3tSeuilConcentration, len(tag5), valeurs)
	if d3tTaux(tete, chainees) >= d3tSeuilConcentration {
		t.Logf("VERROU REEL %s : le sous-ensemble chaine est CONCENTRE. Le taux global bas est un "+
			"artefact de l'ANCRAGE EXHAUSTIF (signature faible balayee a toutes les positions de "+
			"bit, donc faux ancrages proportionnels a la taille du payload), PAS une lecture "+
			"fausse des vrais records.", short)
	} else {
		t.Logf("VERROU REEL %s : le sous-ensemble chaine est DISPERSE. Le bruit survit au filtre "+
			"de chainage — le verrou n'est ni les largeurs ni l'ancrage seul.", short)
	}
}

// d3tTaux rend un taux, zero sans denominateur.
func d3tTaux(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d)
}

// d3tPart rend un taux lisible. UN TAUX SANS DENOMINATEUR N'EST PAS ZERO, et il se dit.
func d3tPart(n, d int) string {
	if d == 0 {
		return "pas de denominateur"
	}
	return fmt.Sprintf("%d/%d = %.1f %%", n, d, 100*float64(n)/float64(d))
}
