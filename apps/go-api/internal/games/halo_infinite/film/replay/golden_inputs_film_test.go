package replay

// golden_inputs_film_test.go — LE CHEMIN DU FILM : decoder les entrees, et rien d autre.
//
// Extrait de golden_inputs_test.go le 2026-09-14 (revue R1, constat R1-7). DEPLACEMENT PUR.
//
// # CE CHEMIN N EST PLUS UNE COPIE (lot 1.0, 2026-09-14)
//
// Il l a ete : la sequence de balayages de `BuildFromFilm` y etait recopiee a la main, et les
// deux copies avaient diverge (decouverte D7 du lot 0.D) — cinq canaux entiers manquaient au
// fixture (`WeaponChanges`, `Pickups`, `EquipmentChanges`, `Vehicles`, `BipedCreations`), la
// lunette aussi, et les largeurs d axe du chemin world-object s installaient par un AUTRE geste
// que celui de la production : `SetWorldObjectPrecisionFromLayout(I0Layout{AxisW: ...})` ne pose
// NI la largeur d index de region NI la region cataloguee, la ou `installWorldObjectPrecision`
// passe par `entry.Layout()` et pose les trois. Sur Live Fire (`60ae07c4`, region 1 sur 2 bits),
// le fixture lisait donc ses objets du monde un bit trop tot — le defaut meme que le lot B-bis
// avait corrige en production le 2026-09-12.
//
// Il APPELLE desormais `scanFilmInputs` — l etage de balayage de la production — sous les DEUX
// memes gestes que `BuildFromFilm` : le verrou de decodage et les largeurs d axe de la carte.
//
// # LES OPTIONS DE BALAYAGE VIENNENT DE LA FEUILLE DE MATCH (revue R1, constat R1-1)
//
// Une CINQUIEME divergence restait : le fixture passait `RosterXUIDs` NUL. L etage lit
// `rosterOf(deaths, opt.RosterXUIDs)` avant `ScanPlayerIndices` — avec `nil`, un joueur a ZERO
// MORT n est dans aucune mort, donc dans aucun roster, donc dans aucune table d index, et il
// manquait au golden (cas mesure sur `3372e7eb` : 6 joueurs publies pour 8). La fidelite etait
// aveugle : elle compare deux assemblages batis sur les MEMES options. Le fixture lit desormais
// le `<short8>.facts.json` du corpus d equivalence et applique LA REGLE DE LA PRODUCTION
// (`RosterXUIDsOf`, cf. roster_xuids.go), celle-la meme que `replaybuild.rosterXUIDs` appelle.
//
// # CE QUE LE FIXTURE N A TOUJOURS PAS, ET C EST DECLARE
//
// Les entrees d `Options` que seule la BASE fournit — catalogue de zones, garde de mode du
// drapeau / de la bombe, enregistrements du statborg, tableau des participants, bots declares,
// relais, actions d objectif, libelles, geometrie, points d apparition, morts neutres et couples
// de frags. Elles n entrent PAS dans l etage de balayage, a une exception pres : les trois
// GARDES DE MODE (`Flag`, `Zone`, `Bomb`) commandent trois balayages, qui restent donc muets au
// fixture. C est la seule divergence de DECODAGE qui subsiste, et elle est nommee ici comme dans
// `champsNonTransportes`.

import (
	"encoding/json"
	"fmt"
	"os"
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
	// LE FILM SE CHARGE UNE FOIS, comme en production (`replaybuild.BuildBytes`) : c est ce
	// chargement-la que l etage de balayage consomme, et c est lui aussi qui porte la version
	// majeure du film.
	charge, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return nil, fmt.Errorf("chargement du film %s : %w", dir, err)
	}
	// LES DEUX GESTES DE LA PRODUCTION, PAR LES MEMES FONCTIONS (cf. `BuildFromFilm`) : le
	// verrou de decodage, puis les largeurs d axe du chemin world-object installees DEPUIS
	// L ENTREE DE CATALOGUE — largeurs, largeur d index de region ET region, que seule
	// `MapQuantEntry.Layout()` porte toutes les trois.
	release := filmdec.LockProcessDecode()
	defer release()
	roster, err := rosterDeLaFeuille(film)
	if err != nil {
		return nil, err
	}
	// L ORDRE EST CELUI DE `BuildFromFilm` DEPUIS LE LOT 2.1 : le contexte du film (donc son
	// PROFIL, resolu a la construction) s ouvre AVANT l installation des largeurs, et c est le
	// profil que l installateur lit. Un fixture qui garderait l ancien ordre ne mesurerait plus
	// ce que la production fait.
	opt := Options{MapQuant: &entry, RosterXUIDs: roster}
	fc := filmdec.NewFilmContextForMap(charge, opt.MapQuant, decoupageForce(opt))
	defer installWorldObjectPrecision(fc.Profile(), film, nil)()
	in, err := scanFilmInputs(film, charge, fc, opt)
	if err != nil {
		return nil, err
	}
	lay, detecte, err := decoupageDuFixture(charge, entry)
	if err != nil {
		return nil, err
	}
	return &goldenInputs{
		Film: film, MapModule: entry.Module, AxisW: lay.AxisW, LayoutDetected: detecte,
		FilmInputs: in,
	}, nil
}

// decoupageDuFixture rend le decoupage d i0 EMPLOYE par le balayage, et s il vient de
// l auto-detection.
//
// LA REGLE EST CELLE DE LA PRODUCTION, PAR LA MEME FONCTION (lot 0.D.7). Ce chemin
// AUTO-DETECTAIT le decoupage (`ScanFilmOptions.Layout` laisse nul), alors que la cuisson le fait
// trancher par `resolveI0Layout` — le catalogue quand l entree est valide, l auto-detection en
// repli. Sur Live Fire les deux DIVERGENT : detection `gate=5 region=0 13/12/11`, catalogue
// `gate=6 region=1 12/12/11`. Meme longueur totale d i0, mais un bit de moins sur X au catalogue
// — donc un pas de quantification DOUBLE a la detection — et une porte de region qui ne testait
// qu un bit, laissant entrer des enregistrements d une AUTRE AABB.
//
// ON N APPELLE PAS `entry.Layout()` ICI : ce serait une COPIE de la regle, qui divergerait le
// jour ou la production change d avis. On demande la regle elle-meme.
//
// L AUTO-DETECTION NE SURVIT QUE LA OU LA PRODUCTION L EMPLOIE — entree de carte invalide
// (`axisWidths` absent, cf. resolveI0Layout). Elle est alors NOMMEE dans le fixture, pour qu un
// lecteur sache que ces quanta ne viennent pas du catalogue.
func decoupageDuFixture(charge *filmsource.Film, entry filmdec.MapQuantEntry) (filmdec.I0Layout, bool, error) {
	if impose := filmdec.NewFilmContextForMap(nil, &entry, nil).ImposedLayout(); impose != nil {
		return *impose, false, nil
	}
	lay, _, err := filmdec.DetectI0LayoutOf(charge)
	if err != nil {
		return filmdec.I0Layout{}, true, fmt.Errorf("decoupage i0 auto-detecte : %w", err)
	}
	return lay, true, nil
}

// rosterDeLaFeuille rend le roster d appoint du film, LU dans son `<short8>.facts.json` et
// projete par la REGLE DE LA PRODUCTION ([RosterXUIDsOf]).
//
// ON NE RELIT QU UN CHAMP, et c est deliberé : `replaybuild.FactsFile` (la forme unique du
// fichier) vit dans un paquet qui IMPORTE `replay`, donc hors de portee d un test de ce paquet.
// Le risque que la doctrine du fichier unique combat — une copie du type qui perd un champ EN
// SILENCE — n existe pas ici : un `players[].xuid` qui cesserait d etre lu rendrait un roster
// VIDE, ce que la garde ci-dessous refuse, et les huit goldens perdraient des joueurs. L echec
// est bruyant par construction.
//
// LE REFUS EST LA GARDE : les huit films du fixture ont tous une feuille peuplee. Un roster vide
// ne peut donc etre qu une lecture cassee, jamais un fait.
func rosterDeLaFeuille(film string) ([]uint64, error) {
	path := filepath.Join(goldenDir, "equivalence", film+".facts.json")
	raw, err := os.ReadFile(path) //nolint:gosec // chemin construit depuis la table des builds
	if err != nil {
		return nil, fmt.Errorf("feuille de match du film %s : %w", film, err)
	}
	var feuille struct {
		Players []struct {
			XUID string `json:"xuid"`
		} `json:"players"`
	}
	if err := json.Unmarshal(raw, &feuille); err != nil {
		return nil, fmt.Errorf("feuille de match du film %s invalide : %w", film, err)
	}
	xuids := make([]string, 0, len(feuille.Players))
	for _, p := range feuille.Players {
		xuids = append(xuids, p.XUID)
	}
	roster := RosterXUIDsOf(xuids)
	if len(roster) == 0 {
		return nil, fmt.Errorf("feuille de match du film %s : AUCUN xuid exploitable sur %d "+
			"ligne(s) — la forme de %s a-t-elle change ? un roster vide ferait disparaitre du "+
			"golden les joueurs a zero mort, en silence", film, len(feuille.Players), path)
	}
	return roster, nil
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
