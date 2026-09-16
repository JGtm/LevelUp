package grammar

// player_table_corpus_test.go — LE LECTEUR DE PRODUCTION REPRODUIT L'ORACLE DES INSTRUMENTS SUR
// TOUT LE CACHE (lot 1.5.4).
//
// # CE QUE CE TEST CONFRONTE, ET POURQUOI IL NE SE CONTENTE PAS D'UN INVARIANT
//
// Les instruments de recherche (`rsChaine` / `rsDelta`, `residus_slots_research_test.go`) sont
// L'ORACLE de ce lot : ils ont mesure, le 2026-09-13 puis le 2026-09-14, que la table se lit
// exactement — 32 slots, occupes plus vacants, sur 1 351 films sur 1 351. Le lecteur de
// production doit rendre la MEME chose, film par film. Un test qui ne verifierait que
// « occupes + vacants == 32 » laisserait passer un lecteur qui trouve 32 slots ailleurs ; la
// confrontation a l'oracle, elle, compare les COMPTES.
//
// LA DIFFERENCE ATTENDUE, ET ELLE EST UNE DECISION, PAS UN DEFAUT : l'instrument calibre la
// transposition SUR LE FILM (`rsDelta`), donc il lit aussi les 5 films sans section
// d'identification. Le lecteur de production les REFUSE (`ErrNoFilmIdentity` puis
// [profile.ErrUnknownBuild]), parce qu'un build inconnu ne se lit jamais au profil du build voisin
// (D-4, ADR 0034). Ces 5 films sont nommes ci-dessous et comptes a part.
//
// # LA GARDE
//
// `CHUNK00_CORPUS` est la RACINE du cache de films ; `CHUNK00_FILMS` reste accepte pour une
// petite liste. LA RACINE N'EST PAS UNE COMMODITE : une liste de 1 351 chemins absolus separes
// par `;` pese une centaine de kilo-octets, au-dela de la borne d'une variable d'environnement
// Windows (32 767 caracteres) — la forme en liste ne peut PAS porter le corpus entier.
//
//	CGO_ENABLED=0 CHUNK00_CORPUS="C:/.../data/cache/film_chunks" \
//	  go test ./internal/games/halo_infinite/film/filmdec/ -run TableJoueursCorpus -v -timeout 60m

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/profile"
	"levelup/go-api/internal/games/halo_infinite/film/source"
)

// corpusPlancher : en dessous de ce nombre de films lus, le test ne mesure rien et le dit. Le
// cache en porte 1 351 le 2026-09-14 ; le plancher est pose bas pour qu'un cache partiel reste
// utilisable, assez haut pour qu'un repertoire presque vide ne passe pas pour une preuve.
const corpusPlancher = 100

// filmsSansSection : les films du cache dont `chunk_00` ne porte AUCUNE section
// d'identification, mesures le 2026-09-14. La liste est ecrite ici ET dans le commentaire
// d'[ErrNoFilmIdentity] : si elle change, les deux se corrigent ensemble.
func filmsSansSection() []string {
	return []string{"03af54c3", "13b00e35", "47d20b5d", "50247b26", "a349fea8"}
}

// bilanCorpus : ce que la passe mesure.
type bilanCorpus struct {
	lus, fermes, sansSection, buildInconnu int
	occupes, vacants                       int
	accordOracle, ecartOracle              int
	contradictions, calibrageFaux          int
	invisibles, filmsInvisibles            int
	intercales                             []string
	divergents                             []divergence
	sansSectionVus                         []string
	parBuild                               map[string]int
}

// TestTableJoueursCorpus confronte le lecteur de production a l'oracle des instruments.
func TestTableJoueursCorpus(t *testing.T) {
	dirs := corpusFilms(t)
	b := &bilanCorpus{parBuild: map[string]int{}}
	for _, dir := range dirs {
		mesurerFilmCorpus(t, dir, b)
	}
	publierBilanCorpus(t, b)
	verifierBilanCorpus(t, b)
}

// corpusFilms rend les repertoires de film, depuis `CHUNK00_CORPUS` (racine) ou `CHUNK00_FILMS`.
func corpusFilms(t *testing.T) []string {
	t.Helper()
	racine := os.Getenv("CHUNK00_CORPUS")
	if racine == "" {
		return chunk00Films(t, "CHUNK00_FILMS")
	}
	entrees, err := os.ReadDir(racine)
	if err != nil {
		t.Fatalf("lecture de %s : %v", racine, err)
	}
	var out []string
	for _, e := range entrees {
		if e.IsDir() {
			out = append(out, filepath.Join(racine, e.Name()))
		}
	}
	return out
}

// mesurerFilmCorpus lit un film par le lecteur de PRODUCTION, puis par l'ORACLE, et range le
// resultat au bilan.
func mesurerFilmCorpus(t *testing.T, dir string, b *bilanCorpus) {
	t.Helper()
	nom := filepath.Base(dir)
	brut, err := os.ReadFile(filepath.Join(dir, "chunk_00.bin")) //nolint:gosec // corpus local
	if err != nil {
		return
	}
	d := source.Inflate(brut)
	b.lus++
	id, err := ReadFilmIdentity(d)
	if errors.Is(err, ErrNoFilmIdentity) {
		b.sansSection++
		b.sansSectionVus = append(b.sansSectionVus, nom)
		return
	}
	if err != nil {
		t.Errorf("%s : identite illisible (%v) — aucun film du cache ne doit l'etre", nom, err)
		return
	}
	b.parBuild[id.Build]++
	slots, rep, err := ReadPlayerTable(d, id)
	if errors.Is(err, profile.ErrUnknownBuild) {
		b.buildInconnu++
		t.Errorf("%s : build %q absent du profil — le profil doit couvrir les sept builds du "+
			"cache (D-4 : ajouter la ligne avec sa provenance, ne jamais lire au plus proche)",
			nom, id.Build)
		return
	}
	if err != nil {
		t.Errorf("%s (%s) : %v", nom, id.Build, err)
		return
	}
	rangerFilmCorpus(t, nom, slots, rep, d, b)
}

// rangerFilmCorpus verifie les invariants d'un film lu et le confronte a l'oracle.
func rangerFilmCorpus(t *testing.T, nom string, slots []PlayerSlot, rep PlayerTableReport,
	d []byte, b *bilanCorpus) {
	t.Helper()
	b.fermes++
	b.occupes += rep.Occupied
	b.vacants += rep.Vacant
	if rep.Occupied+rep.Vacant != playerTableSlots {
		t.Errorf("%s : %d + %d slots, la borne de l'ecrivain en impose %d", nom, rep.Occupied,
			rep.Vacant, playerTableSlots)
	}
	if rep.GapsContradict > 0 {
		b.contradictions += rep.GapsContradict
		t.Errorf("%s : %d ecart(s) en contradiction avec la grammaire", nom, rep.GapsContradict)
	}
	if !rep.CalibrationAgrees {
		b.calibrageFaux++
		t.Errorf("%s (%s) : le calibrage lu sur le film vaut %+d, le profil dit %+d", nom,
			rep.Build, rep.FilmDeltaBits, rep.ProfileDeltaBits)
	}
	if rep.GapsHidden > 0 {
		b.invisibles += rep.GapsHidden
		b.filmsInvisibles++
		t.Logf("  %s : %d enregistrement(s) INVISIBLE(S) au balayage, lus par la grammaire "+
			"(%d candidats reels pour %d enregistrements)", nom, rep.GapsHidden,
			rep.CandidatesReal, rep.Occupied)
	}
	if rep.InterleavedVacant {
		b.intercales = append(b.intercales, nom)
	}
	for _, s := range slots {
		if !gamertagImprimable(s.Gamertag) || s.XUID <= slotXuidLo || s.XUID >= slotXuidHi {
			t.Errorf("%s slot %d : gamertag %q / XUID %d", nom, s.FilmIndex, s.Gamertag, s.XUID)
		}
	}
	// L'ORACLE : le lecteur par grammaire des instruments, calibre SUR LE FILM.
	delta, _, _ := rsDelta(d)
	oracle, oracleVacants := rsChaine(d, delta)
	if len(oracle) == rep.Occupied && oracleVacants == rep.Vacant {
		b.accordOracle++
		return
	}
	b.ecartOracle++
	b.divergents = append(b.divergents, divergence{nom: nom, production: rep.Occupied,
		oracle: len(oracle), vacantsProd: rep.Vacant, vacantsOracle: oracleVacants,
		coupures: coupuresDeGrappe(d)})
	if rep.Occupied < len(oracle) {
		t.Errorf("%s : production %d occupes, ORACLE %d — le lecteur de production ne doit "+
			"JAMAIS en lire moins que l'instrument", nom, rep.Occupied, len(oracle))
	}
}

// divergence : un film ou la production et l'oracle ne rendent pas le meme compte.
type divergence struct {
	nom                        string
	production, oracle         int
	vacantsProd, vacantsOracle int
	coupures                   int
}

// coupuresDeGrappe compte les ecarts du balayage CORRIGE (sans regroupement) qui depassent le
// seuil `s3rEcartMax` des instruments.
//
// C'EST L'EXPLICATION ATTENDUE D'UNE DIVERGENCE, et elle se verifie au lieu de se supposer :
// `rsChaine` part du regroupement terminal de `profilRosterBalaye`, lequel remonte tant que
// l'ecart au precedent reste sous 40 000 bits. Un slot VACANT intercale ajoute 16 499 bits a
// l'ecart entre deux enregistrements (sur le build courant) et le pousse au-dela du seuil : le
// regroupement perd alors TOUTE LA TETE de la table. Le lecteur de production n'a pas de seuil,
// donc il ne perd rien.
func coupuresDeGrappe(d []byte) int {
	crit := profilRosterCritCorrigee()
	crit.EcartMax = 0
	hs := profilRosterBalaye(d, crit)
	n := 0
	for i := 0; i+1 < len(hs); i++ {
		if hs[i+1].bit-hs[i].bit > s3rEcartMax {
			n++
		}
	}
	return n
}

// publierBilanCorpus imprime la mesure, build par build.
func publierBilanCorpus(t *testing.T, b *bilanCorpus) {
	t.Helper()
	var builds []string
	for k := range b.parBuild {
		builds = append(builds, k)
	}
	sort.Slice(builds, func(i, j int) bool { return b.parBuild[builds[i]] > b.parBuild[builds[j]] })
	t.Logf("%d chunk_00 lus", b.lus)
	for _, k := range builds {
		octets, _ := profile.PersonnalisationOctets(k)
		t.Logf("  %-12s %4d film(s) ; bloc de personnalisation %d o (%+d bits)", k,
			b.parBuild[k], octets, profile.PersoDeltaBits(octets))
	}
	sort.Strings(b.sansSectionVus)
	sort.Strings(b.intercales)
	t.Logf("=== BILAN === %d film(s) lus a 32 slots (%d occupes + %d vacants) ; "+
		"%d/%d en accord avec l'ORACLE des instruments ; %d contradiction(s) de grammaire ; "+
		"%d calibrage(s) en desaccord avec le profil ; "+
		"%d enregistrement(s) invisible(s) au balayage sur %d film(s)",
		b.fermes, b.occupes, b.vacants, b.accordOracle, b.fermes, b.contradictions,
		b.calibrageFaux, b.invisibles, b.filmsInvisibles)
	t.Logf("=== MIS DE COTE === %d film(s) sans section d'identification (%s) ; "+
		"%d film(s) a build inconnu", b.sansSection, strings.Join(b.sansSectionVus, " "),
		b.buildInconnu)
	t.Logf("=== VACANT INTERCALE === %d film(s) (%s) : les seuls ou « rang absolu » et "+
		"« index parmi les occupes » divergent", len(b.intercales),
		strings.Join(b.intercales, " "))
}

// verifierBilanCorpus applique les criteres du lot.
func verifierBilanCorpus(t *testing.T, b *bilanCorpus) {
	t.Helper()
	if b.lus < corpusPlancher {
		t.Fatalf("%d film(s) lus : sous le plancher de %d, ce test ne mesure rien", b.lus,
			corpusPlancher)
	}
	if b.fermes+b.sansSection != b.lus {
		t.Errorf("%d films lus, %d fermes + %d sans section : le compte ne tombe pas juste",
			b.lus, b.fermes, b.sansSection)
	}
	for _, dv := range b.divergents {
		if dv.coupures == 0 {
			t.Errorf("%s : production %d occupes, oracle %d, et AUCUNE coupure de grappe ne "+
				"l explique — la divergence n est pas comprise", dv.nom, dv.production, dv.oracle)
			continue
		}
		t.Logf("  %-10s production %2d + %2d vacants, oracle %2d + %2d : %d ecart(s) du balayage "+
			"au-dela de %d bits — le regroupement de l instrument perd la tete de la table",
			dv.nom, dv.production, dv.vacantsProd, dv.oracle, dv.vacantsOracle, dv.coupures,
			s3rEcartMax)
	}
	sort.Strings(b.sansSectionVus)
	if attendu := filmsSansSection(); strings.Join(b.sansSectionVus, " ") !=
		strings.Join(attendu, " ") {
		t.Errorf("films sans section d'identification : %v, attendu %v — si le cache a change, "+
			"corriger `filmsSansSection` ET le commentaire d'ErrNoFilmIdentity dans le meme commit",
			b.sansSectionVus, attendu)
	}
}
