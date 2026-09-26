package replay

// filmfacts_fuzz_test.go — LE HARNAIS DE FUZZ DU FICHIER DE FAITS (lot J2.7, constat RA1-5,
// 2026-09-26), sur le modele de `film/internal/grammar/fuzz_records_test.go`.
//
// CE QUE CE HARNAIS GARANTIT, ET RIEN DE PLUS : [DecodeFilmFactsFile] ne panique sur AUCUNE
// entree. Un fichier de faits vient du disque — perime, tronque, corrompu — et le decodeur doit
// rendre une erreur que l appelant traite en redecodant le film. Aucune valeur n est verifiee :
// sur une entree aleatoire il n existe pas de resultat attendu.
//
// LES GRAINES vivent sous testdata/fuzz/FuzzDecodeFilmFactsFile/ au format de corpus natif de Go :
// `go test` les rejoue TOUTES a chaque execution, meme sans `-fuzz`. Ce sont des fichiers de faits
// ecrits par l encodeur de production : un fichier vide, un fichier porte par les balayages de la
// mini-bobine (`testdata/minifilm_000d5950`, trois elements par liste), et deux troncatures.
//
// CAMPAGNE (a la main, jamais en CI) :
//
//	go test ./internal/games/halo_infinite/film/replay/ -run '^$' -fuzz FuzzDecodeFilmFactsFile -fuzztime 30s -fuzzminimizetime 0s
//
// `-fuzzminimizetime 0s` N EST PAS UN CONFORT : une graine fait plusieurs Ko et le moteur passe
// sinon la campagne a MINIMISER sa premiere trouvaille — le compteur d executions reste fige a
// 0/s (mesure du 2026-09-26 : 630 executions en 20 s, contre 163 266 sans minimisation).
//
// REGENERATION DES GRAINES (porte dediee, qui ne regenere RIEN d autre) :
//
//	go test ./internal/games/halo_infinite/film/replay/ -run TestFuzzFilmFactsSeedsRegenerate -update-graines-faits

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// updateGrainesFaits est la porte de regeneration du corpus de graines de ce harnais, et d elle
// seule : le drapeau `-update` du paquet reecrit d autres references.
var updateGrainesFaits = flag.Bool("update-graines-faits", false,
	"reecrire le corpus de graines de FuzzDecodeFilmFactsFile (testdata/fuzz/)")

// grainesFaitsDir est le corpus natif Go de la cible du meme nom.
const grainesFaitsDir = "testdata/fuzz/FuzzDecodeFilmFactsFile"

// elementsParListe borne chaque liste des faits de la mini-bobine : on cherche les bornes de
// lecture, pas le volume.
const elementsParListe = 3

// FuzzDecodeFilmFactsFile : le decodeur du fichier de faits ne panique sur AUCUNE entree.
//
// L entree de catalogue est celle du film de reference : c est la cle de cuisson que portent les
// graines, donc celle qui ouvre le decodage au-dela de l en-tete.
func FuzzDecodeFilmFactsFile(f *testing.F) {
	entry, err := goldenMapQuant()
	if err != nil {
		f.Fatalf("entree de catalogue du film de reference : %v", err)
	}
	f.Add([]byte{})
	f.Add([]byte(magieFaitsDeFilm))
	f.Fuzz(func(t *testing.T, fichier []byte) {
		_, _ = DecodeFilmFactsFile(fichier, entry)
	})
}

// TestFuzzFilmFactsSeedsRegenerate : LA SEULE PORTE D ECRITURE DU CORPUS DE GRAINES.
func TestFuzzFilmFactsSeedsRegenerate(t *testing.T) {
	if !*updateGrainesFaits {
		t.Skip("regeneration du corpus de graines : passer -update-graines-faits")
	}
	graines := grainesDeFaits(t)
	if err := os.MkdirAll(grainesFaitsDir, 0o750); err != nil {
		t.Fatalf("creation de %s : %v", grainesFaitsDir, err)
	}
	for i, g := range graines {
		path := filepath.Join(grainesFaitsDir, fmt.Sprintf("seed_%02d", i))
		body := "go test fuzz v1\n[]byte(" + strconv.Quote(string(g)) + ")\n"
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatalf("ecriture de %s : %v", path, err)
		}
	}
	t.Logf("%d graine(s) reecrite(s) sous %s", len(graines), grainesFaitsDir)
}

// grainesDeFaits ecrit les graines : un fichier vide, celui de la mini-bobine, et deux
// troncatures de ce dernier (la moitie, puis l en-tete seul). Les deux premiers doivent se
// relire : une graine illisible ne franchirait pas l en-tete et n exercerait rien.
func grainesDeFaits(t *testing.T) [][]byte {
	t.Helper()
	entry := goldenEntryPourTest(t)
	vide := encoderFichierDeFaits(t, faitsVidesDeLaCarte(entry))
	mini := encoderFichierDeFaits(t, faitsDeLaMiniBobine(t, entry))
	for _, g := range [][]byte{vide, mini} {
		if _, err := DecodeFilmFactsFile(g, entry); err != nil {
			t.Fatalf("graine illisible : %v", err)
		}
	}
	entete, err := DecodeFilmFactsEntete(mini)
	if err != nil {
		t.Fatalf("en-tete de la graine de la mini-bobine : %v", err)
	}
	return [][]byte{
		vide, mini,
		append([]byte(nil), mini[:len(mini)/2]...),
		append([]byte(nil), mini[:entete.corps]...),
	}
}

func encoderFichierDeFaits(t *testing.T, g *FilmFacts) []byte {
	t.Helper()
	b, err := EncodeFilmFactsFile(&FilmFactsFile{Facts: *g,
		EmpreinteDeCle: EmpreinteDeCle(goldenEntryPourTest(t))})
	if err != nil {
		t.Fatalf("encodage d une graine : %v", err)
	}
	return b
}

// faitsDeLaMiniBobine porte les balayages que la mini-bobine supporte (cf.
// equivalence_minifilm_test.go), chaque liste bornee a `elementsParListe`.
func faitsDeLaMiniBobine(t *testing.T, entry profile.MapQuantEntry) *FilmFacts {
	t.Helper()
	g := faitsVidesDeLaCarte(entry)
	wr := entry.Range()
	var err error
	if g.Fire, err = grammar.ScanFilmFireEvents(MiniFilmDir); err != nil {
		t.Fatalf("tirs : %v", err)
	}
	if g.Grenades, err = grammar.ScanFilmGrenadeThrows(MiniFilmDir); err != nil {
		t.Fatalf("lancers de grenade : %v", err)
	}
	if g.Loadouts, err = grammar.ScanFilmKeyframeLoadouts(MiniFilmDir, loadoutFamilies()); err != nil {
		t.Fatalf("armes portees : %v", err)
	}
	if g.Inventory, _, err = ScanFilmKeyframeInventory(MiniFilmDir, loadoutFamilies(), 0, nil); err != nil {
		t.Fatalf("inventaire d image-cle : %v", err)
	}
	if g.Deaths, err = ScanFilmDeaths(MiniFilmDir); err != nil {
		t.Fatalf("morts : %v", err)
	}
	if g.PlayerIndices, err = ScanFilmPlayerIndices(MiniFilmDir, rosterFromDeaths(g.Deaths)); err != nil {
		t.Fatalf("indices joueur : %v", err)
	}
	if g.Projectiles, err = grammar.ScanFilmProjectiles(MiniFilmDir, &wr); err != nil {
		t.Fatalf("projectiles : %v", err)
	}
	g.Fire, g.Grenades = premiers(g.Fire), premiers(g.Grenades)
	g.Loadouts, g.Inventory = premiers(g.Loadouts), premiers(g.Inventory)
	g.Deaths, g.Projectiles = premiers(g.Deaths), premiers(g.Projectiles)
	for i := range g.Projectiles {
		g.Projectiles[i].Pts = premiers(g.Projectiles[i].Pts)
	}
	return g
}

// premiers rend au plus `elementsParListe` elements de `s`.
func premiers[T any](s []T) []T {
	if len(s) > elementsParListe {
		return s[:elementsParListe]
	}
	return s
}
