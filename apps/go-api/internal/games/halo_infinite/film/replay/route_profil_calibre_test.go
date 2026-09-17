package replay

// route_profil_calibre_test.go — LA ROUTE DE LA CALIBRATION A UN GARDE-RAIL (lot 2.3.5, revue
// adversariale constat P2-6).
//
// # CE QUE LA REVUE A TROUVE
//
// La condition D1 du pilote — « la calibration de `killsource` voyage EXPLICITEMENT jusqu'au
// contexte du film » — etait tenue par le code et par AUCUN test. Deux mutations passaient au
// vert sur toute la suite :
//
//	(1) `BuildFromFilm` ignore `Options.ProfilDeBalayage`  la cuisson redecode a l'invariant,
//	    et l'heritage que le lot a rendu explicite disparait en silence ;
//	(2) l'ordre des deux poses est inverse                 [grammar.FilmContext.PoserProfilDeBalayage]
//	    remplace le profil ENTIER, donc poser la carte d'abord et le profil ensuite EFFACE les
//	    largeurs de la carte — les objets du monde se dequantifient alors aux largeurs par
//	    defaut, celles d'UNE carte, sur toutes les autres.
//
// Le gate d'equivalence ne les voyait pas davantage : les references du corpus ont ete figees
// avec le comportement courant, et une mutation qui deplace des largeurs deplacerait AUSSI les
// references si on les regenerait. Un test unitaire, lui, epingle l'intention.
//
// # POURQUOI IL N APPELLE PAS `BuildFromFilm`
//
// La mini-bobine n'a aucune image-cle de bipede (`PROVENANCE.txt`), donc `BuildFromFilm` refuse
// de rendre un document — c'est ce que `TestZeroDisqueBuildFromFilm` mesure. Ce test-ci vise le
// geste, pas la cuisson : il appelle `poserProfilPuisCarte`, la fonction que `BuildFromFilm`
// appelle, et lit le contexte apres. Les deux mutations ci-dessus le font rougir.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// TestRouteDuProfilCalibreJusquAuContexte — LE PROFIL CALIBRE ARRIVE, ET LES LARGEURS DE CARTE
// SURVIVENT.
func TestRouteDuProfilCalibreJusquAuContexte(t *testing.T) {
	film := chargerMiniBobineEnMemoire(t)
	entry, err := goldenMapQuant()
	if err != nil {
		t.Fatalf("entree de catalogue de Cliffhanger illisible : %v", err)
	}

	// UN PROFIL CALIBRE RECONNAISSABLE. Les trois valeurs sont celles que `killsource` calibre
	// (descripteur de traversee, largeur d'axe absolue, `param_4`), reglees sur des valeurs QUE
	// NI L'INVARIANT NI LE CATALOGUE NE PRODUISENT : leur presence apres coup ne peut venir que
	// de la route.
	calibre := grammar.ProfilDeBalayageParDefaut()
	calibre.Mouvement.AbsoluteAxisW = 17
	calibre.Mouvement.Traversal.IndexW = 2
	calibre.PoserParamEtat(3)
	// ET UNE QUATRIEME, QUI REND LA MUTATION D ORDRE VISIBLE. Le descripteur world-object du
	// profil par defaut est celui de `cliffhanger` — la MEME carte que l'entree de catalogue de
	// ce test : les deux coincideraient, et intervertir les deux poses ne changerait rien. On le
	// met donc a une valeur que NI l'invariant NI le catalogue ne produisent : si elle survit,
	// c'est que le profil a ete pose APRES la carte et l'a effacee.
	calibre.Mouvement.WorldObject.AxisW = [3]uint{7, 7, 7}

	fc := grammar.NewFilmContextForMap(film, &entry, nil)
	poserProfilPuisCarte(fc, "route-du-profil", Options{ProfilDeBalayage: &calibre})

	got := fc.ProfilDeBalayage()

	// (1) LE PROFIL CALIBRE EST ARRIVE.
	if got.Mouvement.AbsoluteAxisW != 17 {
		t.Errorf("largeur d'axe absolue = %d, calibree a 17.\n"+
			"La calibration de `killsource` n'atteint plus le contexte : `BuildFromFilm` ignore "+
			"`Options.ProfilDeBalayage`, ou quelque chose le remplace apres coup. C'est la "+
			"condition D1 du lot 2.2.a.", got.Mouvement.AbsoluteAxisW)
	}
	if got.Mouvement.Traversal.IndexW != 2 {
		t.Errorf("index de traversee = %d, calibre a 2 (meme cause qu'au-dessus)",
			got.Mouvement.Traversal.IndexW)
	}
	if !got.ParamEtatImpose || got.ParamEtat != 3 {
		t.Errorf("param_4 = %d (impose=%t), calibre a 3 (impose).\n"+
			"C'est la troisieme valeur que `killsource` retient, et elle voyage par le meme canal.",
			got.ParamEtat, got.ParamEtatImpose)
	}

	// (2) LES LARGEURS DE LA CARTE ONT SURVECU. Poser le profil remplace la structure ENTIERE :
	// si l'ordre des deux appels de `poserProfilPuisCarte` est inverse, elles sont effacees et
	// ce sont les largeurs par defaut qui restent.
	attendu := entry.Layout()
	wo := got.LargeursObjetDuMonde()
	if wo.AxisW != attendu.AxisW || wo.Region != attendu.Region {
		t.Errorf("largeurs des objets du monde = axes %v / region %d, attendu celles du catalogue "+
			"de Cliffhanger : axes %v / region %d.\n"+
			"ORDRE INVERSE : `PoserProfilDeBalayage` remplace le profil ENTIER, donc poser la "+
			"carte AVANT le profil efface le decoupage de la carte — et les objets du monde se "+
			"dequantifient aux largeurs par defaut, celles d'UNE carte, sur toutes les autres.",
			wo.AxisW, wo.Region, attendu.AxisW, attendu.Region)
	}

	// (3) SANS PROFIL DANS LES OPTIONS, le contexte garde le sien et la carte s'installe quand
	// meme : un appelant qui n'a pas decode le kill-feed n'est pas puni.
	fc2 := grammar.NewFilmContextForMap(film, &entry, nil)
	poserProfilPuisCarte(fc2, "route-sans-profil", Options{})
	wo2 := fc2.LargeursObjetDuMonde()
	if wo2.AxisW != attendu.AxisW || wo2.Region != attendu.Region {
		t.Errorf("sans profil dans les options : largeurs = axes %v / region %d, attendu axes %v "+
			"/ region %d — l'installation de la carte ne depend pas du profil calibre",
			wo2.AxisW, wo2.Region, attendu.AxisW, attendu.Region)
	}
}
