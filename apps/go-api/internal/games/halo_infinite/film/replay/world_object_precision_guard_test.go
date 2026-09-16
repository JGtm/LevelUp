package replay

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/replay/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/source"
)

// world_object_precision_guard_test.go — GARDE-RAIL du correctif du 2026-08-15.
//
// CE QU'IL GARDE. `grammar.WorldObjectPrecision` est un GLOBAL de paquet dont le défaut EST
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

// TestInstallWorldObjectPrecision : le mécanisme. Pose les largeurs de l'entrée SUR LE CONTEXTE
// du film, et n'en pose AUCUNE ailleurs — un second contexte garde l'invariant.
//
// LA RESTAURATION A DISPARU AU LOT 2.3, et c'est le point : il n'y a plus rien à rendre. Le
// profil meurt avec le contexte, donc aucun film ne peut contaminer le suivant, et deux films
// peuvent se décoder en parallèle.
func TestInstallWorldObjectPrecision(t *testing.T) {
	prev := grammar.ProfilDeBalayageParDefaut().LargeursObjetDuMonde()

	entry := grammar.MapQuantEntry{Module: "ctf_bazaar", AxisWidths: [3]uint{17, 17, 16}}
	if entry.AxisWidths == prev.AxisW {
		t.Fatal("le cas de test doit différer de l'invariant du profil, sinon il ne mesure rien")
	}
	fc := grammar.NewFilmContextForMap(nil, &entry, nil)
	installWorldObjectPrecision(fc, "testdata", fallback.NouveauCompteur())
	if got := fc.LargeursObjetDuMonde().AxisW; got != entry.AxisWidths {
		t.Fatalf("largeurs NON POSÉES sur le contexte : %v, attendu %v (celles de la carte du match)",
			got, entry.AxisWidths)
	}
	if got := grammar.NewFilmContext(nil).LargeursObjetDuMonde(); got != prev {
		t.Fatalf("un AUTRE contexte a vu les largeurs de ce film (%v au lieu de %v) : le profil "+
			"a fui hors du contexte", got.AxisW, prev.AxisW)
	}
}

// TestInstallWorldObjectPrecisionKeepsDefaultWithoutWidths : une entrée sans largeurs (catalogue
// antérieur au champ, entrée fabriquée à la main) garde le défaut. La dégradation est LOGGÉE
// par l'installateur — jamais silencieuse.
func TestInstallWorldObjectPrecisionKeepsDefaultWithoutWidths(t *testing.T) {
	prev := grammar.ProfilDeBalayageParDefaut().LargeursObjetDuMonde()

	sansLargeurs := grammar.MapQuantEntry{Module: "sans_largeurs"}
	fc := grammar.NewFilmContextForMap(nil, &sansLargeurs, nil)
	installWorldObjectPrecision(fc, "testdata", fallback.NouveauCompteur())
	if got := fc.LargeursObjetDuMonde(); got != prev {
		t.Fatalf("largeurs à zéro posées (%v) : le décodeur lirait des champs de 0 bit", got.AxisW)
	}
}

// TestBuildFromFilmRefusesWithoutMapQuant : sans entrée de catalogue, aucun document. Bornes et
// largeurs venant du MÊME champ, l'état « bornes armées, largeurs oubliées » n'existe pas —
// c'est la raison d'être du champ unique `Options.MapQuant`.
func TestBuildFromFilmRefusesWithoutMapQuant(t *testing.T) {
	film, err := source.LoadDir(MiniFilmDir, nil)
	if err != nil {
		t.Fatalf("mini-bobine illisible : %v", err)
	}
	if _, err := BuildFromFilm("minifilm", "halo_infinite", film, Options{}); err == nil {
		t.Fatal("BuildFromFilm a produit un document sans entrée de catalogue : les positions " +
			"ne seraient que des quanta déquantifiés au hasard")
	}
}

// TestBuildFromFilmWiresWorldObjectPrecision : le branchement. `BuildFromFilm` doit poser les
// largeurs DEPUIS `opt.MapQuant`, SUR LE CONTEXTE DU FILM, et AVANT l'étage de balayage qui les
// consomme.
//
// CE QUE LE LOT 2.3 A CHANGÉ, ET CE QU'IL N'A PAS CHANGÉ. Le `defer` et la restauration ont
// disparu avec l'état de processus : le profil meurt avec le contexte, il n'y a plus rien à
// rendre — et c'est ce qui autorise deux films à se décoder en parallèle. L'ORDRE, lui, reste
// l'invariant : poser APRÈS avoir ouvert le contexte, AVANT le premier balayage.
//
// DEPUIS LA REVUE DU LOT (2.3.5), LA POSE EST UNE FONCTION NOMMÉE. `BuildFromFilm` appelle
// `poserProfilPuisCarte(fc, …)`, qui pose le profil calibré PUIS les largeurs de la carte — le
// second ordre, celui des deux poses ENTRE ELLES, est épinglé par
// `TestRouteDuProfilCalibreJusquAuContexte` (`route_profil_calibre_test.go`), qui le MESURE au
// lieu de le lire. Ce garde-ci reste sur le texte, et il garde les DEUX corps : la pose doit
// être appelée depuis `BuildFromFilm` entre l'ouverture du contexte et le balayage, et poser
// les largeurs sur `fc`.
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
	// largeurs viennent de la CARTE DU MATCH et pas de l'invariant du profil —, seulement de
	// chemin : l'installateur recoit desormais le CONTEXTE, sur lequel il pose.
	pose := regexp.MustCompile(`
\s*poserProfilPuisCarte\(fc, `)
	if !pose.MatchString(body) {
		t.Fatal("BuildFromFilm n'appelle plus poserProfilPuisCarte(fc, …) : les largeurs d'axe " +
			"ne sont plus posées sur le contexte du film, et les objets du monde de TOUTES les " +
			"cartes repassent en silence aux largeurs de Cliffhanger (invariant du profil). " +
			"Mesuré le 2026-08-15 : la part d'échantillons de projectile dans l'emprise des " +
			"bipèdes tombe de ~99 % à 0,09-65 % hors Cliffhanger")
	}
	// LA POSE ELLE-MÊME doit toujours atteindre `installWorldObjectPrecision` sur le contexte :
	// une `poserProfilPuisCarte` vidée de son second geste passerait le contrôle ci-dessus.
	corpsPose, ok := funcBody(string(src), "func poserProfilPuisCarte(")
	if !ok {
		t.Fatal("poserProfilPuisCarte introuvable dans build_from_film.go : ce garde-rail ne " +
			"garde plus rien")
	}
	if !regexp.MustCompile(`
\s*installWorldObjectPrecision\(fc, `).MatchString(corpsPose) {
		t.Fatal("poserProfilPuisCarte ne pose plus les largeurs d'axe sur le contexte du film " +
			"(mêmes conséquences mesurées que ci-dessus)")
	}
	poseInstall := pose.FindStringIndex(body)[0]
	ouverture := strings.Index(body, "grammar.NewFilmContextForMap(")
	if ouverture < 0 || ouverture > poseInstall {
		t.Fatal("les largeurs sont posées avant que le contexte du film n'existe : le profil de " +
			"balayage est un CHAMP du contexte depuis le lot 2.3, il n'y a rien à poser avant lui")
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

// TestDecoupageForceSuitLesOptions : `decoupageForce` — que `BuildFromFilm` passe au constructeur
// du contexte — rend EXACTEMENT le champ `Layout` que `scanFilmInputs` compose pour son compte.
//
// LES DEUX SE SONT SÉPARÉS AU LOT 2.1 : le contexte est désormais ouvert dans `BuildFromFilm`,
// donc avant que `scanFilmInputs` n'ait composé ses `ScanFilmOptions`. Une divergence entre les
// deux ferait décoder le film sous un découpage que l'appelant croyait avoir forcé, en silence.
func TestDecoupageForceSuitLesOptions(t *testing.T) {
	force := grammar.I0Layout{GateBits: 7, AxisW: [3]uint{12, 12, 11}, Region: 1}
	cas := []struct {
		nom     string
		opt     Options
		attendu *grammar.I0Layout
	}{
		{"sans options de balayage", Options{}, grammar.DefaultScanFilmOptions().Layout},
		{"options sans découpage", Options{Scan: &grammar.ScanFilmOptions{}}, nil},
		{"découpage forcé", Options{Scan: &grammar.ScanFilmOptions{Layout: &force}}, &force},
	}
	for _, c := range cas {
		// On rejoue EXACTEMENT les deux lignes par lesquelles `scanFilmInputs` compose `s.scan` :
		// le défaut du paquet, écrasé par `*opt.Scan` quand l'appelant en fournit un.
		attendu := grammar.DefaultScanFilmOptions()
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
// Le profil du FILM porte la carte du match ; le profil de BALAYAGE du contexte porte ce que le
// décodeur applique. Les deux doivent rendre les MÊMES largeurs pendant tout le décodage.
//
// LE TEST A CHANGE DE CIBLE AU LOT 2.3 : il confrontait le profil du film à la VARIABLE DE
// PAQUET (double écriture datée). Celle-ci a disparu ; il confronte désormais le profil du film
// au profil de balayage du contexte, qui est ce que les lecteurs portent.
func TestProfilEgaleGlobalesWorldObject(t *testing.T) {
	cartes := []grammar.MapQuantEntry{
		{Module: "ctf_bazaar", AxisWidths: [3]uint{17, 17, 16}},
		{Module: "live_fire", AxisWidths: [3]uint{12, 12, 11}, Region: 1, RegionIndexBits: 2},
		{Module: "cliffhanger", AxisWidths: [3]uint{13, 13, 14}},
	}
	for _, e := range cartes {
		entry := e
		fc := grammar.NewFilmContextForMap(nil, &entry, nil)
		installWorldObjectPrecision(fc, "testdata", fallback.NouveauCompteur())
		attendu, got := fc.Profile().Map().Layout(), fc.LargeursObjetDuMonde()
		if got.AxisW != attendu.AxisW || got.Region != attendu.Region {
			t.Errorf("%s : le profil du film dit {axes %v région %d}, le profil de balayage du "+
				"contexte dit {axes %v région %d}", entry.Module, attendu.AxisW, attendu.Region,
				got.AxisW, got.Region)
		}
	}
}
