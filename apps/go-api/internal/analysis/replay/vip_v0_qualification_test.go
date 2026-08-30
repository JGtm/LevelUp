package replay

// vip_v0_qualification_test.go — LOT RESOLUTION, PHASE V0 : LA QUALIFICATION DU CORPUS VIP.
//
// LE PROTOCOLE EST COMMITE AVANT CE FICHIER (`.ai/V7.5/PROTOCOLE_RESOLUTION_VIP_ASSAUT.md`,
// section R2, et `PROTOCOLE_REMESURE_ODDBALL_VIP.md` §0.2/§3). Deux questions par film,
// aucune supposee — meme discipline qu'A0.2 (Assaut), sans la question des sites (le VIP n'a
// pas d'objet porte au sol) :
//
//	BORNES  la carte est-elle au catalogue de quantification (`map_quant_bounds.json`) ?
//	        Sans elles le film est indecodable en coordonnees monde — il sort avec sa raison.
//	PONT    part des slots de bipede nommes par le pont d'identite — la precondition >= 50 %
//	        heritee du lot O (`d8PontMinimum`), MEME instrument que la qualification Assaut.
//
// Le VIP se JUGE ensuite au statborg (le film ne porte pas le bit VIP, §2.10) : cette phase
// ne fait que QUALIFIER le corpus. Le gate d'identite du composant est au confront dedie.
//
// REGIME : gardes `ATT_FILM` (racine du cache) + `VIP_FILM` (identifiant court), UN FILM PAR
// PROCESSUS, lecture seule, AUCUNE base.
//
//	$env:ATT_FILM="<repo>/data/cache"; $env:VIP_FILM="00761d27"
//	go test ./internal/analysis/replay/ -run VIPV0Qualification -v

import (
	"math"
	"os"
	"testing"

	"levelup/go-api/internal/filmproc"
)

// vipFilmEnv designe LE film qualifie par ce processus — une boucle sur trois films dans un
// seul processus est exactement ce que la doctrine de l'executeur borne interdit.
const vipFilmEnv = "VIP_FILM"

// TestVIPV0Qualification — la qualification d'UN film du corpus VIP.
func TestVIPV0Qualification(t *testing.T) {
	root := attRequireRoot(t)
	id := os.Getenv(vipFilmEnv)
	if id == "" {
		t.Skipf("mesure non demandee : %s vide (identifiant court du film VIP)", vipFilmEnv)
	}
	g := filmproc.Arm("vip-v0", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE %s : %.2f Gio — qualification interrompue, ce film "+
			"sort NON QUALIFIE", id, float64(peak)/(1<<30))
	})
	defer func() {
		g.Disarm()
		t.Logf("%s : pic memoire observe %.2f Gio (plafond souple %d Gio)",
			id, float64(g.Peak())/(1<<30), filmproc.MeasureLimitGiB)
	}()

	if _, ok := objOpenFilm(t, root, id); !ok {
		t.Fatalf("%s : film absent du cache (%s=%q)", id, attFilmEnv, root)
	}

	// (1) BORNES de quantification — sans elles, le verdict tombe ici.
	wr, lay, okB := d6Bornes(t, root, id)
	if !okB {
		t.Logf("VERDICT %s : EXCLU — bornes de quantification absentes du catalogue "+
			"(film indecodable en coordonnees monde)", id)
		return
	}

	// (2) PONT bipede — la precondition du lot O, meme calcul que `d8Charge` / A0.2.
	pos, err := d6Positions(objChunkDir(root, id), wr, lay)
	if err != nil {
		t.Fatalf("%s : positions de bipede illisibles : %v", id, err)
	}
	tracks := indexBySlot(pos)
	pont := objBridgeOf(t, root, id)
	nommes := 0
	for slot := range tracks {
		if _, ok := pont.SlotXUID[slot]; ok {
			nommes++
		}
	}
	part := float64(nommes) / math.Max(float64(len(tracks)), 1)
	t.Logf("%s : pont %d/%d slot(s) nomme(s) = %.1f %% (plancher %.0f %%)",
		id, nommes, len(tracks), 100*part, 100*d8PontMinimum)

	if part < d8PontMinimum {
		t.Logf("VERDICT %s : EXCLU — pont %.1f %% sous le plancher de %.0f %%", id, 100*part,
			100*d8PontMinimum)
		return
	}
	t.Logf("VERDICT %s : ADMIS — bornes OK, pont %.1f %% OK", id, 100*part)
}
