//go:build research

package replaybuild

// m1_parc_depuis_faits_research_test.go — INSTRUMENT DE PARC du lot M1 des retours du rejeu
// (2026-09-23) : RECONSTRUIRE LES DOCUMENTS DEPUIS LES FAITS PERSISTES, puis COMPARER deux
// reconstructions calque par calque.
//
// # POURQUOI UN TEST ET PAS `backfill-replay`
//
// `backfill-replay` lit la base partagee (faits du match) et, faute de faits frais, DECODE le film
// puis ECRIT ses faits dans `data/cache/film_facts` — deux gestes interdits a cet instrument. Ici :
//
//	LES FAITS SONT RELUS A LA MAIN, depuis un depot en LECTURE SEULE (`M1_FAITS`), et un fichier
//	perime ou illisible est SAUTE et compte : aucune branche de ce test ne decode un film, aucune
//	n ecrit ailleurs que dans `M1_SORTIE` ;
//	LES FAITS DU MATCH SONT VIDES (`port.MatchFacts{}`) : la comparaison base / branche porte sur
//	les MEMES entrees des deux cotes, et ce lot ne touche que la publication des positions ;
//	LES CATALOGUES viennent du depot du code teste (`testutil.RepoRoot`), c est-a-dire du code de
//	la base puis de la branche.
//
// Un seul processus, SEQUENTIEL (un document en memoire a la fois), sous `GOMEMLIMIT` et sous la
// voie film de la campagne. REPRENABLE : un document deja present dans `M1_SORTIE` est saute.
//
//	M1_FAITS=<depot>/data/cache/film_facts/halo_infinite M1_ARTEFACTS=<depot>/data/cache/replays/halo_infinite \
//	M1_CARTES=<maps.tsv> M1_SORTIE=<dossier> GOMEMLIMIT=3GiB \
//	  go test -tags research -count=1 -run '^TestM1ParcDepuisLesFaits$' -v -timeout 60m ./internal/replaybuild/
//
//	M1_AVANT=<dossier base> M1_APRES=<dossier branche> M1_RAPPORT=<fichier.json> \
//	  go test -tags research -count=1 -run '^TestM1ParcComparer$' -v ./internal/replaybuild/

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/replaydiff"
	"levelup/go-api/internal/testutil"
)

var m1ArtefactRe = regexp.MustCompile(`^[0-9a-f]{8}\.json$`)

func TestM1ParcDepuisLesFaits(t *testing.T) {
	faits, artefacts, cartes, sortie := os.Getenv("M1_FAITS"), os.Getenv("M1_ARTEFACTS"),
		os.Getenv("M1_CARTES"), os.Getenv("M1_SORTIE")
	if faits == "" || artefacts == "" || cartes == "" || sortie == "" {
		t.Skip("instrument de parc : M1_FAITS, M1_ARTEFACTS, M1_CARTES et M1_SORTIE requis")
	}
	repoRoot, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine repo : %v", err)
	}
	b, err := NewBuilder(repoRoot, title.DefaultSlug)
	if err != nil {
		t.Fatalf("builder : %v", err)
	}
	carteDe := m1LireCartes(t, cartes)
	if err := os.MkdirAll(sortie, 0o750); err != nil {
		t.Fatal(err)
	}
	entrees, err := os.ReadDir(artefacts)
	if err != nil {
		t.Fatal(err)
	}
	faitsOK, sautes := 0, map[string]string{}
	for _, e := range entrees {
		if !m1ArtefactRe.MatchString(e.Name()) {
			continue
		}
		court := strings.TrimSuffix(e.Name(), ".json")
		if _, err := os.Stat(filepath.Join(sortie, e.Name())); err == nil {
			faitsOK++ // REPRISE : deja reconstruit par une passe precedente (ecriture atomique)
			continue
		}
		if raison := m1Reconstruire(b, m1Entree{
			artefact: filepath.Join(artefacts, e.Name()), faits: filepath.Join(faits, court+".filmfacts.bin"),
			carte: carteDe[court], sortie: filepath.Join(sortie, e.Name()),
		}); raison != "" {
			sautes[court] = raison
			continue
		}
		faitsOK++
	}
	t.Logf("PARC : %d documents reconstruits depuis les faits, %d sautes", faitsOK, len(sautes))
	for c, r := range sautes {
		t.Logf("   saute %s : %s", c, r)
	}
}

type m1Entree struct{ artefact, faits, carte, sortie string }

// m1Reconstruire rend "" quand le document est ecrit, la raison du saut sinon.
func m1Reconstruire(b *Builder, in m1Entree) string {
	if in.carte == "" {
		return "carte inconnue de la table"
	}
	matchID, err := m1MatchID(in.artefact)
	if err != nil {
		return "matchId illisible : " + err.Error()
	}
	entry, err := b.ResolveMapEntry([]string{in.carte})
	if err != nil {
		return err.Error()
	}
	blob, err := os.ReadFile(in.faits)
	if err != nil {
		return "faits absents : " + err.Error()
	}
	entete, err := replay.DecodeFilmFactsEntete(blob)
	if err == nil {
		err = entete.Utilisable(entry)
	}
	if err != nil {
		return "faits perimes : " + err.Error()
	}
	f, err := replay.DecodeFilmFactsFile(blob, entry)
	if err != nil {
		return "faits illisibles : " + err.Error()
	}
	ctx := context.Background()
	src := entreesDeCuisson{faits: f, statborg: f.Statborg, deaths: filmDeaths{list: f.Facts.Deaths}, kills: f.Kills}
	facts := port.MatchFacts{}
	stats := assemblerFilmStats(ctx, matchID, src.statborg, facts, src.deaths)
	if stats.score != nil {
		stats.score.TargetScore, _ = b.regulation.ScoreTarget(facts.GameVariantName)
		stats.score.HoldTicksPerPoint, _ = b.regulation.HoldTicksPerPoint(facts.GameVariantName)
	}
	cat := b.collecterEntreesCatalogue(matchID, []string{in.carte}, facts, &stats, src)
	doc, err := b.documentDeLaCuisson(ctx, matchID, b.buildReplayOptions(entry, facts, cat, &stats), src)
	if err != nil {
		return err.Error()
	}
	built, err := b.serialiserDocument(matchID, entry, doc, true, time.Now())
	if err != nil {
		return err.Error()
	}
	// ECRITURE ATOMIQUE : une passe interrompue ne laisse jamais un document tronque que la
	// reprise tiendrait pour fait.
	if err := os.WriteFile(in.sortie+".tmp", built.Blob, 0o600); err != nil {
		return "ecriture : " + err.Error()
	}
	if err := os.Rename(in.sortie+".tmp", in.sortie); err != nil {
		return "ecriture : " + err.Error()
	}
	return ""
}

// m1MatchID lit le `matchId` d un artefact publie sans le charger en entier.
func m1MatchID(chemin string) (string, error) {
	f, err := os.Open(chemin) //nolint:gosec // instrument de parc, chemin de l operateur
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	dec := json.NewDecoder(bufio.NewReader(f))
	for {
		tok, err := dec.Token()
		if err != nil {
			return "", err
		}
		if s, ok := tok.(string); ok && s == "matchId" {
			var id string
			if err := dec.Decode(&id); err != nil {
				return "", err
			}
			return id, nil
		}
	}
}

// m1LireCartes lit la table `id8 \t carte \t ...` (copie de la base du 2026-09-23).
func m1LireCartes(t *testing.T, chemin string) map[string]string {
	t.Helper()
	blob, err := os.ReadFile(chemin) //nolint:gosec // instrument de parc
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, l := range strings.Split(string(blob), "\n")[1:] {
		c := strings.Split(strings.TrimRight(l, "\r"), "\t")
		if len(c) >= 2 && c[0] != "" {
			out[c[0]] = c[1]
		}
	}
	return out
}

// TestM1ParcComparer confronte deux reconstructions document par document (empreintes de
// `internal/replaydiff`) et agrege les ecarts par mesure.
func TestM1ParcComparer(t *testing.T) {
	avant, apres, rapport := os.Getenv("M1_AVANT"), os.Getenv("M1_APRES"), os.Getenv("M1_RAPPORT")
	if avant == "" || apres == "" {
		t.Skip("instrument de parc : M1_AVANT et M1_APRES requis")
	}
	entrees, err := os.ReadDir(avant)
	if err != nil {
		t.Fatal(err)
	}
	type agr struct {
		Docs  int      `json:"docs"`
		Delta float64  `json:"delta"`
		Ex    []string `json:"exemples"`
	}
	parMesure := map[string]*agr{}
	docsTouches, n := 0, 0
	for _, e := range entrees {
		if !m1ArtefactRe.MatchString(e.Name()) {
			continue
		}
		da, err := replaydiff.LireDocument(filepath.Join(avant, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		db, err := replaydiff.LireDocument(filepath.Join(apres, e.Name()))
		if err != nil {
			t.Errorf("%s absent de la branche : %v", e.Name(), err)
			continue
		}
		n++
		r := replaydiff.Comparer(replaydiff.Empreindre(da), replaydiff.Empreindre(db))
		if len(r.Differences) > 0 {
			docsTouches++
		}
		for _, d := range r.Differences {
			k := d.Axe + "/" + m1Generaliser(d.Metrique)
			a := parMesure[k]
			if a == nil {
				a = &agr{}
				parMesure[k] = a
			}
			a.Docs++
			a.Delta += d.Delta
			if len(a.Ex) < 4 {
				a.Ex = append(a.Ex, fmt.Sprintf("%s %s:%s->%s", e.Name()[:8], d.Metrique, d.Ancien, d.Nouveau))
			}
		}
	}
	cles := make([]string, 0, len(parMesure))
	for k := range parMesure {
		cles = append(cles, k)
	}
	sort.Strings(cles)
	t.Logf("COMPARAISON : %d documents, %d avec au moins un ecart", n, docsTouches)
	for _, k := range cles {
		a := parMesure[k]
		t.Logf("   %-70s docs=%3d delta=%10.2f  %v", k, a.Docs, a.Delta, a.Ex)
	}
	if rapport != "" {
		blob, _ := json.MarshalIndent(parMesure, "", " ")
		if err := os.WriteFile(rapport, blob, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// m1Generaliser remplace les identifiants (xuid, slot) d une metrique par `*` pour agreger.
var m1Chiffres = regexp.MustCompile(`[0-9]{3,}`)

func m1Generaliser(m string) string { return m1Chiffres.ReplaceAllString(m, "*") }
