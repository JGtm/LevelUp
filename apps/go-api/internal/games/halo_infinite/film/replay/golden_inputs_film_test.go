package replay

// golden_inputs_film_test.go — LE CHEMIN DU FILM : decoder les entrees, et rien d autre.
//
// Extrait de golden_inputs_test.go le 2026-09-14 (revue R1, constat R1-7). DEPLACEMENT PUR.
//
// CE CHEMIN EST UNE COPIE DE LA SEQUENCE DE BALAYAGES DE `BuildFromFilm` (decouverte D7 du
// lot 0.D) : le lot 1.0 la remplacera par une fonction partagee.

import (
	"fmt"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
)

func decodeFilmInputs(film, dir string) (*goldenInputs, error) {
	entry, err := goldenMapQuant()
	if err != nil {
		return nil, err
	}
	return decodeFilmInputsForEntry(film, dir, entry)
}

// decodeFilmInputsForEntry est le MEME decodage, pour une carte quelconque (lot 0.A.2 : un
// fixture d entrees par build, donc une carte par build). `decodeFilmInputs` en est le cas
// particulier de Cliffhanger, et le seul chemin qui change est la LECTURE DU CATALOGUE.
func decodeFilmInputsForEntry(film, dir string, entry filmdec.MapQuantEntry) (*goldenInputs, error) {
	var err error
	// MEME GESTE QUE LA PRODUCTION (cf. installWorldObjectPrecision) : les largeurs d'axe du
	// chemin world-object viennent de l'entree de catalogue, pas du defaut de paquet. Sur
	// Cliffhanger les deux coincident — c'est precisement pourquoi l'oubli avait survecu des
	// mois : le film de reference est le SEUL sur lequel il ne se voit pas.
	prev := filmdec.WorldObjectPrecision
	defer func() { filmdec.WorldObjectPrecision = prev }()
	filmdec.SetWorldObjectPrecisionFromLayout(filmdec.I0Layout{AxisW: entry.AxisWidths})
	wr := entry.Range()
	scan := filmdec.DefaultScanFilmOptions()
	scan.WorldRange = &wr
	// LE DECOUPAGE D i0 SUIT LA REGLE DE LA PRODUCTION, PAR LA MEME FONCTION (lot 0.D.7).
	//
	// CE QUE CELA CORRIGE. Ce chemin AUTO-DETECTAIT le decoupage (`ScanFilmOptions.Layout`
	// laisse nul), alors que la cuisson le fait trancher par `resolveI0Layout` — le catalogue
	// quand l entree est valide, l auto-detection en repli. Sur Live Fire les deux DIVERGENT :
	// detection `gate=5 region=0 13/12/11`, catalogue `gate=6 region=1 12/12/11`. Meme longueur
	// totale d i0, mais un bit de moins sur X au catalogue — donc un pas de quantification
	// DOUBLE a la detection — et une porte de region qui ne testait qu un bit, laissant entrer
	// des enregistrements d une AUTRE AABB. Le golden de `60ae07c4` affirmait donc des
	// coordonnees que la production ne produit pas.
	//
	// ON N APPELLE PAS `entry.Layout()` ICI : ce serait une COPIE de la regle, qui divergerait
	// le jour ou la production change d avis. On demande la regle elle-meme.
	impose := filmdec.NewFilmContextForMap(nil, &entry, nil).ImposedLayout()
	detecte := impose == nil
	if impose != nil {
		scan.Layout = impose
	}
	// L AUTO-DETECTION NE SURVIT QUE LA OU LA PRODUCTION L EMPLOIE — entree de carte invalide
	// (`axisWidths` absent, cf. resolveI0Layout). Elle est alors NOMMEE dans le fixture, pour
	// qu un lecteur sache que ces quanta ne viennent pas du catalogue.
	lay := filmdec.I0Layout{}
	if impose != nil {
		lay = *impose
	} else {
		var layErr error
		if lay, _, layErr = filmdec.DetectI0Layout(dir); layErr != nil {
			return nil, fmt.Errorf("decoupage i0 de %s : %w", dir, layErr)
		}
	}
	scan.CaptureDirs = true
	// MEME GESTE QUE LA PRODUCTION (BuildFromFilm) : les teleportations se lisent AVANT les
	// positions, parce qu elles exemptent le filtre de vitesse (decision D2), et AVEC l entree
	// de catalogue, parce que leur charge porte le va-et-vient quantifie aux bornes de la
	// carte. Sans ce geste, le fixture porterait des positions que la production ne decode plus.
	translocs := filmdec.ScanFilmTranslocatorTeleports(dir, &entry)
	scan.TeleportExemptions = filmdec.TeleportExemptionsOf(translocs)
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		return nil, err
	}
	g := &goldenInputs{
		Film: film, MapModule: entry.Module, AxisW: lay.AxisW, LayoutDetected: detecte,
		Positions: pos, Translocations: translocs,
	}
	if g.Fire, err = filmdec.ScanFilmFireEvents(dir); err != nil {
		return nil, err
	}
	if g.Loadouts, err = filmdec.ScanFilmKeyframeLoadouts(dir, loadoutFamilies()); err != nil {
		return nil, err
	}
	if g.Inventory, _, err = ScanFilmKeyframeInventory(dir, loadoutFamilies(), 0); err != nil {
		return nil, err
	}
	if g.AbilityRanks, _, err = filmdec.ScanFilmAbilityRanks(dir); err != nil {
		return nil, err
	}
	var dStats filmdec.InventoryDeltaStats
	if g.InventoryDeltas, dStats, err = filmdec.ScanFilmInventoryDeltas(dir); err != nil {
		return nil, err
	}
	// MEME GESTE QUE LA PRODUCTION (`build_from_film.go` : `opt.InventoryDeltaAmmoRefused =
	// dStats.AmmoRefused`) : le verdict du scanner voyage avec ses donnees.
	g.InventoryDeltaAmmoRefused = dStats.AmmoRefused
	// LA VERSION DU FILM, MEME GESTE QUE LA PRODUCTION (`build_from_film.go:104`) : elle est
	// publiee en `coverage.filmMajorVersion`, donc c est une entree de l assemblage.
	if film, errFilm := filmsource.LoadDir(dir, nil); errFilm == nil {
		if v, lue := filmdec.FilmMajorVersion(film); lue {
			g.FilmMajorVersion = &v
		}
	}
	if g.CamoStates, _, err = filmdec.ScanFilmCamoStates(dir); err != nil {
		return nil, err
	}
	if g.GrappleReads, _, err = filmdec.ScanFilmGrappleReads(dir); err != nil {
		return nil, err
	}
	// MEME COMPOSANT, AUTRE TAG : les impulsions de capacite passent par LA MEME fonction que
	// BuildFromFilm — le fixture porte ce que la production decode, pas une variante.
	if g.AbilityImpulses, g.AbilityImpulseStats, err = filmdec.ScanFilmAbilityImpulses(dir); err != nil {
		return nil, err
	}
	// LES CHARGES RESTANTES (v14) : la MEME fonction que BuildFromFilm, meme raison.
	if g.AbilityCharges, g.AbilityChargeStats, err = filmdec.ScanFilmAbilityCharges(dir); err != nil {
		return nil, err
	}
	if g.Placements, g.PlacementStats, err = filmdec.ScanFilmEquipmentPlacements(dir, &wr); err != nil {
		return nil, err
	}
	// Les armes au sol passent par LA MEME fonction que BuildFromFilm : le fixture porte ce que
	// la production decode, pas une variante de lecture — largeurs MPP calibrees comprises.
	g.Pads = decodeFilmPadScansDir(dir, &wr, g.PlacementStats.Calibration.Widths)
	if g.Grenades, err = filmdec.ScanFilmGrenadeThrows(dir); err != nil {
		return nil, err
	}
	if g.Projectiles, err = filmdec.ScanFilmProjectiles(dir, &wr); err != nil {
		return nil, err
	}
	if g.Deaths, err = ScanFilmDeaths(dir); err != nil {
		return nil, err
	}
	idx, err := ScanFilmPlayerIndices(dir, rosterFromDeaths(g.Deaths))
	if err != nil {
		return nil, err
	}
	table, collisions := injectiveOrEmpty(idx)
	if collisions > 0 {
		return nil, fmt.Errorf("index de joueur non injectif (%d collisions) — fixture refuse", collisions)
	}
	g.Indices = table
	// L origine d horloge est lue par la MEME fonction que BuildFromFilm : le fixture porte
	// l entree, pas une valeur recopiee a la main.
	if g.ClockOriginUS, err = ScanFilmClockOrigin(dir); err != nil {
		return nil, err
	}
	return g, nil
}

// goldenMapQuant rend l'ENTREE DE CATALOGUE de Cliffhanger : bornes ET largeurs d'axe, comme
// `replay.Options.MapQuant` les recoit en production. Les dissocier laisserait la regeneration
// armer les bornes en oubliant les largeurs — l'erreur meme que le lot du 2026-08-15 corrige.
//
// Si le catalogue change, [TestGoldenAssembly] tombe et le diff dit exactement ce qui a bouge.
func goldenMapQuant() (filmdec.MapQuantEntry, error) {
	path := filepath.Join("..", "..", "..", "..", "..", "..", "..", "data", "titles", "halo_infinite",
		"reference", "map_quant_bounds.json")
	cat, err := filmdec.LoadMapQuantCatalog(path)
	if err != nil {
		return filmdec.MapQuantEntry{}, fmt.Errorf("catalogue de bornes %s : %w", path, err)
	}
	entry, err := cat.Lookup("Cliffhanger")
	if err != nil {
		return filmdec.MapQuantEntry{}, err
	}
	return entry, nil
}

// goldenEntryPourTest rend l entree de catalogue du film de reference, ou echoue le test.
//
// Elle existe parce que le decodeur de blob EXIGE desormais cette entree (lot 0.D.3 bis) : les
// positions y sont des quanta, et sans les bornes de la carte elles ne sont pas des coordonnees.
func goldenEntryPourTest(t *testing.T) filmdec.MapQuantEntry {
	t.Helper()
	entry, err := goldenMapQuant()
	if err != nil {
		t.Fatalf("entree de catalogue du film de reference : %v", err)
	}
	return entry
}
