package replay

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
	"levelup/go-api/internal/games/halo_infinite/film/replay/fallback"
)

// world_object_precision_guard_test.go — GARDE-RAIL du correctif du 2026-08-15.
//
// CE QU'IL GARDE. `filmdec.WorldObjectPrecision` est un GLOBAL de paquet dont le défaut EST
// l'entrée `cliffhanger` du catalogue. Pendant des mois AUCUN chemin de production ne
// l'écrasait : toutes les autres cartes déquantifiaient leurs objets du monde aux largeurs de
// Cliffhanger, en silence. `BuildFromFilm` installe désormais les largeurs de l'entrée de
// catalogue DU MATCH et les restaure au retour.
//
// POURQUOI DEUX TESTS ET PAS UN. Un global oublié ne se voit pas : il faut garder les DEUX
// moitiés du correctif, et elles se cassent séparément.
//   - le MÉCANISME (installe puis restaure) : [TestInstallWorldObjectPrecision] ;
//   - le BRANCHEMENT (BuildFromFilm l'appelle, sous le verrou, depuis `opt.MapQuant`) :
//     [TestBuildFromFilmWiresWorldObjectPrecision], qui lit la source.
//
// Le branchement ne peut pas se vérifier en exécutant `BuildFromFilm` : le seul film versionné
// (`MiniFilmDir`) est une fenêtre de paquets DELTA sans image-clé de bipède, et le décodage
// s'arrête avant les objets du monde. Un test de source EST ici la seule garde toujours active
// — et c'est la forme que le dépôt emploie déjà pour ce genre d'invariant.

// TestInstallWorldObjectPrecision : le mécanisme. Installe les largeurs de l'entrée, les rend
// visibles pendant l'appel, et les restaure ensuite.
func TestInstallWorldObjectPrecision(t *testing.T) {
	prev := filmdec.WorldObjectPrecision
	t.Cleanup(func() { filmdec.WorldObjectPrecision = prev })

	entry := filmdec.MapQuantEntry{Module: "ctf_bazaar", AxisWidths: [3]uint{17, 17, 16}}
	if entry.AxisWidths == prev.AxisW {
		t.Fatal("le cas de test doit différer du défaut de paquet, sinon il ne mesure rien")
	}
	restore := installWorldObjectPrecision(filmdec.ResolveProfile(nil, &entry), "testdata", nil)
	if got := filmdec.WorldObjectPrecision.AxisW; got != entry.AxisWidths {
		t.Fatalf("largeurs NON INSTALLÉES : %v, attendu %v (celles de la carte du match)",
			got, entry.AxisWidths)
	}
	restore()
	if filmdec.WorldObjectPrecision != prev {
		t.Fatalf("largeurs NON RESTAURÉES : %v, attendu %v — un global non rendu contamine le "+
			"film suivant décodé dans le même process",
			filmdec.WorldObjectPrecision.AxisW, prev.AxisW)
	}
}

// TestInstallWorldObjectPrecisionKeepsDefaultWithoutWidths : une entrée sans largeurs (catalogue
// antérieur au champ, entrée fabriquée à la main) garde le défaut. La dégradation est LOGGÉE
// par l'installateur — jamais silencieuse.
func TestInstallWorldObjectPrecisionKeepsDefaultWithoutWidths(t *testing.T) {
	prev := filmdec.WorldObjectPrecision
	t.Cleanup(func() { filmdec.WorldObjectPrecision = prev })

	sansLargeurs := filmdec.MapQuantEntry{Module: "sans_largeurs"}
	restore := installWorldObjectPrecision(filmdec.ResolveProfile(nil, &sansLargeurs), "testdata", nil)
	if filmdec.WorldObjectPrecision != prev {
		t.Fatalf("largeurs à zéro installées (%v) : le décodeur lirait des champs de 0 bit",
			filmdec.WorldObjectPrecision.AxisW)
	}
	restore()
	if filmdec.WorldObjectPrecision != prev {
		t.Fatalf("restauration fautive : %v, attendu %v",
			filmdec.WorldObjectPrecision.AxisW, prev.AxisW)
	}
}

// TestBuildFromFilmRefusesWithoutMapQuant : sans entrée de catalogue, aucun document. Bornes et
// largeurs venant du MÊME champ, l'état « bornes armées, largeurs oubliées » n'existe pas —
// c'est la raison d'être du champ unique `Options.MapQuant`.
func TestBuildFromFilmRefusesWithoutMapQuant(t *testing.T) {
	film, err := filmsource.LoadDir(MiniFilmDir, nil)
	if err != nil {
		t.Fatalf("mini-bobine illisible : %v", err)
	}
	if _, err := BuildFromFilm("minifilm", "halo_infinite", film, Options{}); err == nil {
		t.Fatal("BuildFromFilm a produit un document sans entrée de catalogue : les positions " +
			"ne seraient que des quanta déquantifiés au hasard")
	}
}

// TestBuildFromFilmWiresWorldObjectPrecision : le branchement. `BuildFromFilm` doit installer
// les largeurs DEPUIS `opt.MapQuant`, en DIFFÉRÉ (donc restaurer), APRÈS avoir pris le verrou
// de décodage — le descripteur est un global, deux films décodés en parallèle se voleraient
// leurs largeurs — et AVANT l'étage de balayage, qui les consomme.
func TestBuildFromFilmWiresWorldObjectPrecision(t *testing.T) {
	src, err := os.ReadFile("build_from_film.go")
	if err != nil {
		t.Fatal(err)
	}
	body, ok := funcBody(string(src), "func BuildFromFilm(")
	if !ok {
		t.Fatal("BuildFromFilm introuvable dans build_from_film.go : ce garde-rail ne garde plus rien")
	}
	// LA SOURCE DES LARGEURS EST LE PROFIL DEPUIS LE LOT 2.1 (item 2.1.3), et c'est la MEME
	// valeur : `fc.Profile().Map()` est l'entree `opt.MapQuant` que `NewFilmContextForMap` a
	// recue deux lignes plus haut. Ce que le garde exige n'a pas change de nature — que les
	// largeurs viennent de la CARTE DU MATCH et pas du defaut de paquet —, seulement de chemin.
	install := regexp.MustCompile(`defer\s+installWorldObjectPrecision\(fc\.Profile\(\)`)
	if !install.MatchString(body) {
		t.Fatal("BuildFromFilm n'installe plus les largeurs d'axe depuis le profil du film : les " +
			"objets du monde de TOUTES les cartes repassent en silence aux largeurs de " +
			"Cliffhanger (défaut de paquet). Mesuré le 2026-08-15 : la part d'échantillons de " +
			"projectile dans l'emprise des bipèdes tombe de ~99 % à 0,09-65 % hors Cliffhanger")
	}
	poseInstall := install.FindStringIndex(body)[0]
	lock := strings.Index(body, "filmdec.LockProcessDecode()")
	if lock < 0 || lock > poseInstall {
		t.Fatal("l'installation des largeurs précède la prise du verrou de décodage : le " +
			"descripteur est un global de paquet, il ne s'écrit que sous le verrou")
	}
	// L'ORDRE AVEC L'ÉTAGE DE BALAYAGE (revue R1, constat R1-3). Le garde ne vérifiait que
	// « verrou puis installation » : déplacer `installWorldObjectPrecision` d'une SEULE ligne
	// après `scanFilmInputs` laissait `replay` ET `archlint` verts, alors que les ~27 balayages
	// se seraient faits aux largeurs par défaut. C'est l'ORDRE des trois symboles qui est
	// l'invariant, pas la présence de chacun.
	balayage := strings.Index(body, "scanFilmInputs(")
	if balayage < 0 {
		t.Fatal("BuildFromFilm n'appelle plus scanFilmInputs : l'étage de balayage a été " +
			"renommé ou remis en ligne — déplacer ce garde-rail avec lui")
	}
	if poseInstall > balayage {
		t.Fatal("les largeurs d'axe sont installées APRÈS l'étage de balayage : les ~27 " +
			"balayages déquantifient alors leurs objets du monde aux largeurs par DÉFAUT " +
			"(celles de Cliffhanger), et sur une carte à plus de deux régions ils lisent " +
			"leurs trois axes un bit trop tôt (cf. lot B-bis, 2026-09-12)")
	}
}

// funcBody rend le corps de la fonction dont la signature commence par head, du `{` ouvrant à
// l'accolade fermante de colonne 0 (convention gofmt).
func funcBody(src, head string) (string, bool) {
	i := strings.Index(src, head)
	if i < 0 {
		return "", false
	}
	rest := src[i:]
	if end := strings.Index(rest, "\n}\n"); end >= 0 {
		return rest[:end], true
	}
	return rest, true
}

// installWorldObjectPrecisionDeCarte est L'ENVELOPPE DES INSTRUMENTS : elle prend une entrée de
// catalogue et résout autour d'elle le profil des invariants.
//
// Elle existe parce que la production, depuis le lot 2.1, passe un [filmdec.Profile] et non plus
// une entrée de carte : les quatorze instruments de ce paquet qui installent des largeurs pour
// balayer un film n'ont, eux, aucune raison de résoudre un profil complet. Le film est nil — les
// trois clés ne servent pas ici, seule la CARTE est lue par l'installateur.
func installWorldObjectPrecisionDeCarte(e filmdec.MapQuantEntry, matchID string,
	fb *fallback.Compteur) func() {
	return installWorldObjectPrecision(filmdec.ResolveProfile(nil, &e), matchID, fb)
}

// TestDecoupageForceSuitLesOptions : `decoupageForce` — que `BuildFromFilm` passe au constructeur
// du contexte — rend EXACTEMENT le champ `Layout` que `scanFilmInputs` compose pour son compte.
//
// LES DEUX SE SONT SÉPARÉS AU LOT 2.1 : le contexte est désormais ouvert dans `BuildFromFilm`,
// donc avant que `scanFilmInputs` n'ait composé ses `ScanFilmOptions`. Une divergence entre les
// deux ferait décoder le film sous un découpage que l'appelant croyait avoir forcé, en silence.
func TestDecoupageForceSuitLesOptions(t *testing.T) {
	force := filmdec.I0Layout{GateBits: 7, AxisW: [3]uint{12, 12, 11}, Region: 1}
	cas := []struct {
		nom     string
		opt     Options
		attendu *filmdec.I0Layout
	}{
		{"sans options de balayage", Options{}, filmdec.DefaultScanFilmOptions().Layout},
		{"options sans découpage", Options{Scan: &filmdec.ScanFilmOptions{}}, nil},
		{"découpage forcé", Options{Scan: &filmdec.ScanFilmOptions{Layout: &force}}, &force},
	}
	for _, c := range cas {
		// On rejoue EXACTEMENT les deux lignes par lesquelles `scanFilmInputs` compose `s.scan` :
		// le défaut du paquet, écrasé par `*opt.Scan` quand l'appelant en fournit un.
		attendu := filmdec.DefaultScanFilmOptions()
		if c.opt.Scan != nil {
			attendu = *c.opt.Scan
		}
		got := decoupageForce(c.opt)
		if got != attendu.Layout || got != c.attendu {
			t.Errorf("%s : decoupageForce rend %v, `scanFilmInputs` composerait %v (attendu %v)",
				c.nom, got, attendu.Layout, c.attendu)
		}
	}
}

// TestProfilEgaleGlobalesWorldObject : LE VOLET WORLD-OBJECT DE LA DOUBLE ÉCRITURE (item 2.1.3).
//
// Le profil du film porte la carte du match ; la globale de paquet porte ce que le décodeur
// applique. Tant que [doubleEcritureGlobales] est vrai, les deux doivent rendre les MÊMES
// largeurs pendant tout le décodage — c'est ce qui autorisera le lot 2.2 à basculer les lecteurs
// sur le profil sans changer un seul bit lu.
func TestProfilEgaleGlobalesWorldObject(t *testing.T) {
	if !doubleEcritureGlobales {
		t.Skip("double écriture retirée (lot 2.3) : ce test devient le critère « 0 globale »")
	}
	prev := filmdec.WorldObjectPrecision
	t.Cleanup(func() { filmdec.WorldObjectPrecision = prev })
	cartes := []filmdec.MapQuantEntry{
		{Module: "ctf_bazaar", AxisWidths: [3]uint{17, 17, 16}},
		{Module: "live_fire", AxisWidths: [3]uint{12, 12, 11}, Region: 1, RegionIndexBits: 2},
		{Module: "cliffhanger", AxisWidths: [3]uint{13, 13, 14}},
	}
	for _, e := range cartes {
		entry := e
		prof := filmdec.ResolveProfile(nil, &entry)
		restore := installWorldObjectPrecision(prof, "testdata", nil)
		attendu, got := prof.Map().Layout(), filmdec.WorldObjectPrecision
		restore()
		if got.AxisW != attendu.AxisW || got.Region != attendu.Region {
			t.Errorf("%s : le profil dit {axes %v région %d}, la globale installée dit "+
				"{axes %v région %d}", entry.Module, attendu.AxisW, attendu.Region,
				got.AxisW, got.Region)
		}
	}
}
