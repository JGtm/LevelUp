// Package killsourceload_test — le CONTRAT de `Load`, vu du dehors.
//
// CE QUE CES TESTS PROTEGENT, ET POURQUOI CA VAUT UN FICHIER. `Load` est le foyer unique
// de six surfaces (vue match, synthese, series temporelles, sessions, explorateur cible,
// escouade) et il est BEST-EFFORT : il ne rend jamais d'erreur, il degrade. Une degradation
// est exactement ce qui ne se voit pas — le sunburst reste juste (ces kills retombent dans
// « Non attribue »), simplement moins precis, et personne ne s'en apercoit. Le seul filet
// qui reste en production est donc le LOG. C'est pour ca qu'il est teste ici comme un
// resultat, pas comme un ornement : une panne muette serait une mesure fausse a l'oeil du
// lecteur, sans un signal nulle part.
//
// AUCUNE VALEUR N'EST FIGEE. Les lignes de test sont FABRIQUEES (`rowsFor`) et toute
// attente chiffree est CALCULEE depuis ces lignes (`sumKills`). Recopier « 42 » a cote
// d'un test rendrait le test faux le jour ou la fabrique change, et — pire — vert le jour
// ou le code se met a compter autre chose que ce qu'il compte.
//
// Aucune I/O, aucune base, aucun film : un depot factice et un tampon de logs.
package killsourceload_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/killsourceload"
)

// ---------------------------------------------------------------------------
// Outillage
// ---------------------------------------------------------------------------

// fakeRepo enregistre ce qu'il a recu et rend ce qu'on lui a dit de rendre.
type fakeRepo struct {
	calls      int
	gotCtx     context.Context
	gotSlug    string
	gotFilters port.KillSourceClassFilters

	rows []port.KillSourceClassRow
	err  error
}

func (f *fakeRepo) LoadKillSourceClassesAggregated(
	ctx context.Context, slug string, filters port.KillSourceClassFilters,
) ([]port.KillSourceClassRow, error) {
	f.calls++
	f.gotCtx, f.gotSlug, f.gotFilters = ctx, slug, filters
	return f.rows, f.err
}

// rowsFor fabrique n lignes DETERMINISTES mais non figees : chaque champ derive de
// l'index. Un test qui verifie une somme la CALCULE depuis ces lignes (cf. sumKills) —
// jamais en recopiant un nombre.
func rowsFor(n int) []port.KillSourceClassRow {
	classes := []string{"equipment", "environmental"}
	out := make([]port.KillSourceClassRow, 0, n)
	for i := 1; i <= n; i++ {
		out = append(out, port.KillSourceClassRow{
			XUID:                fmt.Sprintf("xuid(%d)", 2533274000000000+i),
			WeaponKey:           fmt.Sprintf("clef_source_%d", i),
			Class:               classes[i%len(classes)],
			Label:               fmt.Sprintf("Source %d", i),
			Kills:               i * 3,
			NonPublishableKills: i,
		})
	}
	return out
}

// sumKills rend les deux totaux que le log annonce. C'est la SEULE definition de ces
// sommes cote test : si le producteur se met a compter autre chose, l'ecart se voit.
func sumKills(rows []port.KillSourceClassRow) (kills, nonPublishable int) {
	for _, r := range rows {
		kills += r.Kills
		nonPublishable += r.NonPublishableKills
	}
	return kills, nonPublishable
}

// captureLogs redirige le logger par defaut vers un tampon, au niveau DEBUG (donc TOUT
// est capture, y compris la degradation nominale). Restaure a la fin du test.
//
// Ces tests ne sont donc PAS parallelisables : `slog.SetDefault` est un etat global.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	return &buf
}

const (
	testSurface = "surface de test"
	testSlug    = "titre_de_test"
)

func someMatchIDs() []string { return []string{"m_1", "m_2", "m_3"} }
func someXUIDs() []string    { return []string{"xuid(1)", "xuid(2)"} }

// ---------------------------------------------------------------------------
// Le perimetre : ce que Load ne demande PAS au depot
// ---------------------------------------------------------------------------

// TestLoadNeSollicitePasLeDepotSansPerimetre — un scope vide n'est pas une panne, c'est
// « rien a faire » : aucun appel, et AUCUN LOG. Une ligne par surface et par requete sur
// un cas nominal noierait les vrais signaux, qui sont la seule chose qui reste en prod.
func TestLoadNeSollicitePasLeDepotSansPerimetre(t *testing.T) {
	cas := []struct {
		nom      string
		sansRepo bool
		matchIDs []string
		xuids    []string
	}{
		{nom: "depot absent (titre sans la capability)", sansRepo: true, matchIDs: someMatchIDs(), xuids: someXUIDs()},
		{nom: "aucun match", matchIDs: nil, xuids: someXUIDs()},
		{nom: "aucun joueur", matchIDs: someMatchIDs(), xuids: nil},
		{nom: "ni match ni joueur", matchIDs: nil, xuids: nil},
		{nom: "tranches vides mais non nil", matchIDs: []string{}, xuids: []string{}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			buf := captureLogs(t)
			repo := &fakeRepo{rows: rowsFor(2)}
			var arg port.KillSourceClassRepository = repo
			if c.sansRepo {
				arg = nil
			}

			got := killsourceload.Load(context.Background(), arg, testSurface, testSlug, c.matchIDs, c.xuids)

			if got != nil {
				t.Errorf("attendu nil, got %+v", got)
			}
			if !c.sansRepo && repo.calls != 0 {
				t.Errorf("le depot a ete appele %d fois, attendu 0", repo.calls)
			}
			if buf.Len() != 0 {
				t.Errorf("un scope vide doit rester SILENCIEUX, log = %q", buf.String())
			}
		})
	}
}

// TestLaGardePrecoceEtValidateDisentLaMemeChose — L'INVARIANT QUI SURVIVRA AUX EVOLUTIONS.
//
// `Load` a DEUX gardes qui protegent la meme chose (un scan complet de la table partagee) :
// son retour precoce sur scope vide, et `filters.Validate()`. Aujourd'hui elles coincident
// exactement, ce qui rend la branche « filtres invalides » inatteignable. Le jour ou
// `Validate()` gagne une regle que le retour precoce ne connait pas — un plafond de matchs,
// un format de xuid — les deux cessent de coincider, et ce test le dit AVANT que la
// divergence n'arrive en production.
//
// Le seul enonce teste est donc : le depot est appele SI ET SEULEMENT SI les filtres sont
// valides. Il ne fige aucune regle des deux cotes ; il exige qu'elles restent d'accord.
func TestLaGardePrecoceEtValidateDisentLaMemeChose(t *testing.T) {
	cas := []struct {
		nom      string
		matchIDs []string
		xuids    []string
	}{
		{nom: "perimetre complet", matchIDs: someMatchIDs(), xuids: someXUIDs()},
		{nom: "un seul match, un seul joueur", matchIDs: []string{"m_1"}, xuids: []string{"xuid(1)"}},
		{nom: "sans match", matchIDs: nil, xuids: someXUIDs()},
		{nom: "sans joueur", matchIDs: someMatchIDs(), xuids: nil},
		{nom: "sans rien", matchIDs: nil, xuids: nil},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			captureLogs(t)
			repo := &fakeRepo{rows: rowsFor(1)}
			killsourceload.Load(context.Background(), repo, testSurface, testSlug, c.matchIDs, c.xuids)

			filtresValides := port.KillSourceClassFilters{MatchIDs: c.matchIDs, XUIDs: c.xuids}.Validate() == nil
			depotAppele := repo.calls > 0
			if depotAppele != filtresValides {
				t.Errorf("depot appele = %v, filtres valides = %v — les deux gardes ont diverge",
					depotAppele, filtresValides)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// La transmission : ce que Load passe, et ce qu'il rend
// ---------------------------------------------------------------------------

// TestLoadTransmetLePerimetreSansLeReecrire — le titre et les deux listes arrivent au depot
// tels quels. Un filtre silencieusement retaille ici rendrait une mesure partielle qui a
// l'air complete.
func TestLoadTransmetLePerimetreSansLeReecrire(t *testing.T) {
	captureLogs(t)
	matchIDs, xuids := someMatchIDs(), someXUIDs()
	repo := &fakeRepo{rows: rowsFor(2)}

	killsourceload.Load(context.Background(), repo, testSurface, testSlug, matchIDs, xuids)

	if repo.calls != 1 {
		t.Fatalf("le depot devait etre appele une fois, got %d", repo.calls)
	}
	if repo.gotSlug != testSlug {
		t.Errorf("slug = %q, want %q", repo.gotSlug, testSlug)
	}
	if !reflect.DeepEqual(repo.gotFilters.MatchIDs, matchIDs) {
		t.Errorf("MatchIDs = %v, want %v", repo.gotFilters.MatchIDs, matchIDs)
	}
	if !reflect.DeepEqual(repo.gotFilters.XUIDs, xuids) {
		t.Errorf("XUIDs = %v, want %v", repo.gotFilters.XUIDs, xuids)
	}
}

// TestLoadNeMutePasLesTranchesDeLAppelant — les surfaces reutilisent leurs listes de matchs
// apres l'appel (l'escouade les partage entre plusieurs lectures). Un tri ou une
// deduplication en place ici corromprait un calcul voisin, tres loin d'ici.
func TestLoadNeMutePasLesTranchesDeLAppelant(t *testing.T) {
	captureLogs(t)
	matchIDs, xuids := someMatchIDs(), someXUIDs()
	avantM := append([]string(nil), matchIDs...)
	avantX := append([]string(nil), xuids...)

	killsourceload.Load(context.Background(), &fakeRepo{rows: rowsFor(3)},
		testSurface, testSlug, matchIDs, xuids)

	if !reflect.DeepEqual(matchIDs, avantM) {
		t.Errorf("MatchIDs mutes : %v (etait %v)", matchIDs, avantM)
	}
	if !reflect.DeepEqual(xuids, avantX) {
		t.Errorf("XUIDs mutes : %v (etait %v)", xuids, avantX)
	}
}

// TestLoadRendLesLignesTellesQuelles — Load est un CHEMIN, pas un calcul : il ne trie pas,
// ne filtre pas, n'agrege pas. L'agregation est faite cote DuckDB et le sunburst en depend.
func TestLoadRendLesLignesTellesQuelles(t *testing.T) {
	captureLogs(t)
	attendu := rowsFor(5)
	repo := &fakeRepo{rows: attendu}

	got := killsourceload.Load(context.Background(), repo, testSurface, testSlug,
		someMatchIDs(), someXUIDs())

	if !reflect.DeepEqual(got, attendu) {
		t.Errorf("lignes alterees.\n got = %+v\nwant = %+v", got, attendu)
	}
}

// TestLoadPropageLeContexte — l'annulation doit atteindre la requete DuckDB : une page
// abandonnee ne doit pas continuer a balayer la table partagee.
func TestLoadPropageLeContexte(t *testing.T) {
	captureLogs(t)
	ctx, annule := context.WithCancel(context.Background())
	annule()
	repo := &fakeRepo{rows: rowsFor(1)}

	killsourceload.Load(ctx, repo, testSurface, testSlug, someMatchIDs(), someXUIDs())

	if repo.gotCtx == nil {
		t.Fatal("aucun contexte transmis au depot")
	}
	if !errors.Is(repo.gotCtx.Err(), context.Canceled) {
		t.Errorf("le contexte transmis n'est pas celui de l'appelant (Err = %v)", repo.gotCtx.Err())
	}
}

// ---------------------------------------------------------------------------
// La degradation : le log EST le resultat
// ---------------------------------------------------------------------------

// TestCapabiliteNonSupporteeEstUnEtatNominal — un match dont le film n'a jamais ete decode
// est l'etat NOMINAL d'une grande partie du parc. Ce n'est pas une panne : rien ne doit
// partir en ERROR, sinon le vrai signal se noie dans le bruit.
//
// Les deux formes sont testees, nue et EMBALLEE : le depot enveloppe ses erreurs
// (`fmt.Errorf("...: %w", err)`), donc une comparaison `==` au lieu d'un `errors.Is`
// requalifierait tout le parc non decode en panne. C'est exactement la regression que ce
// cas verrouille.
func TestCapabiliteNonSupporteeEstUnEtatNominal(t *testing.T) {
	cas := []struct {
		nom string
		err error
	}{
		{nom: "nue", err: games.ErrCapabilityNotSupported},
		{nom: "emballee une fois", err: fmt.Errorf("depot: %w", games.ErrCapabilityNotSupported)},
		{nom: "emballee deux fois", err: fmt.Errorf("surface: %w",
			fmt.Errorf("depot: %w", games.ErrCapabilityNotSupported))},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			buf := captureLogs(t)
			// Le depot rend AUSSI des lignes : elles ne doivent pas etre servies.
			repo := &fakeRepo{rows: rowsFor(2), err: c.err}

			got := killsourceload.Load(context.Background(), repo, testSurface, testSlug,
				someMatchIDs(), someXUIDs())

			if got != nil {
				t.Errorf("attendu nil, got %+v", got)
			}
			if logged := buf.String(); strings.Contains(logged, "level=ERROR") {
				t.Errorf("le parc non decode ne doit PAS produire d'ERROR, log = %q", logged)
			}
		})
	}
}

// TestTouteAutreErreurPartEnERROR — L'INVARIANT CENTRAL DE CE FICHIER.
//
// Une panne reelle (verrou, schema en retard, DB absente) degrade le sunburst en silence :
// les kills hors arme a feu retombent dans « Non attribue » et le graphe reste plausible.
// Le seul endroit ou la panne existe encore est le log. Il doit donc porter le niveau
// ERROR, la CAUSE, et la SURFACE — sans quoi on sait qu'il y a un trou mais pas ou.
func TestTouteAutreErreurPartEnERROR(t *testing.T) {
	type erreurMaison struct{ error }
	cas := []struct {
		nom string
		err error
	}{
		{nom: "erreur nue", err: errors.New("verrou tenu par un autre process")},
		{nom: "erreur emballee", err: fmt.Errorf("depot: %w", errors.New("schema en retard"))},
		{nom: "type d'erreur maison", err: erreurMaison{errors.New("panne du connecteur")}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			buf := captureLogs(t)
			// Des lignes PARTIELLES accompagnent l'erreur : les servir donnerait une mesure
			// silencieusement amputee, ce qui est pire que pas de mesure du tout.
			repo := &fakeRepo{rows: rowsFor(4), err: c.err}

			got := killsourceload.Load(context.Background(), repo, testSurface, testSlug,
				someMatchIDs(), someXUIDs())

			if got != nil {
				t.Errorf("une lecture en erreur ne doit RIEN servir, got %+v", got)
			}
			logged := buf.String()
			if !strings.Contains(logged, "level=ERROR") {
				t.Errorf("une panne doit partir en ERROR, log = %q", logged)
			}
			if !strings.Contains(logged, c.err.Error()) {
				t.Errorf("la CAUSE doit figurer au log (%q), log = %q", c.err.Error(), logged)
			}
			if !strings.Contains(logged, testSurface) {
				t.Errorf("la SURFACE doit figurer au log, log = %q", logged)
			}
			if !strings.Contains(logged, testSlug) {
				t.Errorf("le TITRE doit figurer au log, log = %q", logged)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Le compte rendu du cas nominal
// ---------------------------------------------------------------------------

// TestLeLogAnnonceLesSommesDesLignesServies — les compteurs du log sont la SOMME des lignes
// rendues, calculee ici depuis la fabrique. C'est ce chiffre qu'on lit en production pour
// savoir si la mesure a couvert le parc attendu ; un compteur qui deriverait (compter les
// classes au lieu des kills, oublier les passes non publiables) rendrait ce diagnostic faux
// sans qu'aucun test ne bouge.
func TestLeLogAnnonceLesSommesDesLignesServies(t *testing.T) {
	for _, n := range []int{1, 3, 12} {
		t.Run(fmt.Sprintf("%d lignes", n), func(t *testing.T) {
			buf := captureLogs(t)
			rows := rowsFor(n)
			kills, nonPub := sumKills(rows)

			killsourceload.Load(context.Background(), &fakeRepo{rows: rows},
				testSurface, testSlug, someMatchIDs(), someXUIDs())

			logged := buf.String()
			attendus := map[string]int{
				"classes":         n,
				"kills":           kills,
				"non_publishable": nonPub,
				"match_count":     len(someMatchIDs()),
			}
			for cle, valeur := range attendus {
				jeton := cle + "=" + strconv.Itoa(valeur)
				if !strings.Contains(logged, jeton) {
					t.Errorf("le log doit porter %q, log = %q", jeton, logged)
				}
			}
			if !strings.Contains(logged, "level=INFO") {
				t.Errorf("le cas nominal se rend compte en INFO, log = %q", logged)
			}
		})
	}
}

// TestAucuneLigneNeProduitAucunCompteRendu — zero ligne est un resultat legitime (aucun
// kill hors arme a feu sur le scope). Le dire chaque fois noierait les comptes rendus qui
// portent une mesure ; et surtout, un « kills=0 » se lirait comme une panne alors que la
// lecture a parfaitement reussi.
func TestAucuneLigneNeProduitAucunCompteRendu(t *testing.T) {
	for _, nom := range []string{"tranche nil", "tranche vide"} {
		t.Run(nom, func(t *testing.T) {
			buf := captureLogs(t)
			repo := &fakeRepo{}
			if nom == "tranche vide" {
				repo.rows = []port.KillSourceClassRow{}
			}

			got := killsourceload.Load(context.Background(), repo, testSurface, testSlug,
				someMatchIDs(), someXUIDs())

			if len(got) != 0 {
				t.Errorf("attendu aucune ligne, got %+v", got)
			}
			if repo.calls != 1 {
				t.Errorf("le depot devait etre interroge une fois, got %d", repo.calls)
			}
			if buf.Len() != 0 {
				t.Errorf("une lecture vide et reussie doit rester silencieuse, log = %q", buf.String())
			}
		})
	}
}
