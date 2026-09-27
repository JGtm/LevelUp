package killcollector

// carte_non_resolue_test.go — SANS CARTE, LE FILM EST MIS DE COTE (2026-09-27).
//
// Regle utilisateur, fermee : « Le flux du film est la seule source fiable. Pas de repli. » Jusqu a
// ce correctif, un match dont la carte n etait pas resolue etait decode aux largeurs d axe PAR
// DEFAUT (celles de Cliffhanger) : la marche des morts s y desynchronise et le scan publie a sa
// place. Le collecteur le met desormais de cote, comme un film a cle inconnue : outcome dedie,
// compteur expvar, aucun marqueur de registre, aucune ecriture.
//
// LA BOBINE est celle du build de reference (`fb1a1a72`, Banished Narrows), commise au depot : elle
// traverse la porte de la cle. Ce qu elle devient ENSUITE sous sa carte (une bobine de trois chunks
// sans paquet de replication echoue au decodeur) n est pas le sujet — la porte de la carte est en
// amont.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : retirer la porte de la carte dans `decodeFilmForMatch`
// (le film est decode et sort `sans-killfeed`) ; oublier le compteur ; ranger l outcome en erreur
// dans la synthese ; poser un marqueur de registre sur cet outcome.

import (
	"context"
	"errors"
	"strings"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/testutil"
)

// carteDeLaBobine : la carte du film dont la bobine de reference est tiree (PROVENANCE.txt).
const carteDeLaBobine = "Banished Narrows"

// nomsDeCarteFixes : le resolveur de carte reduit a une liste fixe, sans base.
type nomsDeCarteFixes struct{ noms []string }

func (n nomsDeCarteFixes) MapKeysForMatch(context.Context, string) (port.MatchMapKeys, error) {
	return port.MatchMapKeys{Names: n.noms}, nil
}

func (n nomsDeCarteFixes) MapKeysForMap(context.Context, string) (port.MatchMapKeys, error) {
	return port.MatchMapKeys{Names: n.noms}, nil
}

// catalogueDuDepot : le catalogue de bornes COMMIS, par le meme chemin que la production.
func catalogueDuDepot(t *testing.T) *decfilm.MapQuantCatalog {
	t.Helper()
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	cat, err := decfilm.LoadMapQuantCatalog(titlePkg.NewPathResolver(racine).MapQuantBoundsPath("halo_infinite"))
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	return cat
}

func collecteurDeBobine(t *testing.T) *KillSourceCollector {
	t.Helper()
	client := &clientDeBobine{chunks: bobineEnChunks(t, bobineCleConnue, false)}
	return NewKillSourceCollector(client, rosterVide{}, nil, capsAvecFilmPourCle(), 0)
}

func TestKillCollectorMetDeCoteUnFilmSansCarte(t *testing.T) {
	for _, cas := range []struct {
		nom    string
		cabler func(*KillSourceCollector)
	}{
		{"resolution de carte non cablee", func(*KillSourceCollector) {}},
		{"aucun nom de carte en base", func(c *KillSourceCollector) {
			c.WithPositionCapture(nomsDeCarteFixes{}, catalogueDuDepot(t))
		}},
		{"carte hors catalogue de bornes", func(c *KillSourceCollector) {
			c.WithPositionCapture(nomsDeCarteFixes{noms: []string{"Carte Inexistante"}}, catalogueDuDepot(t))
		}},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			col := collecteurDeBobine(t)
			cas.cabler(col)
			avant := observability.LoadCounter(metricCarteNonResolue)
			erreurs := observability.LoadCounter(metricDecodeError)

			outcome, morts, err := col.CollectMatch(context.Background(), "m1")
			if err != nil {
				t.Fatalf("CollectMatch : erreur %v — une carte non resolue est un ETAT, pas une panne", err)
			}
			if outcome != OutcomeCarteNonResolue {
				t.Fatalf("outcome = %q, attendu %q — le film a ete DECODE sans sa carte, aux largeurs "+
					"d une autre (repli interdit)", outcome, OutcomeCarteNonResolue)
			}
			if morts != 0 {
				t.Errorf("morts = %d, attendu 0 — un film mis de cote ne publie rien", morts)
			}
			if n := observability.LoadCounter(metricCarteNonResolue) - avant; n != 1 {
				t.Errorf("compteur %q : +%d, attendu +1 — la mise de cote est invisible en production",
					metricCarteNonResolue, n)
			}
			if n := observability.LoadCounter(metricDecodeError) - erreurs; n != 0 {
				t.Errorf("compteur %q : +%d, attendu 0 — une carte manquante n est pas une panne",
					metricDecodeError, n)
			}
			if _, marquer := marquerFilmParOutcome(outcome, morts); marquer {
				t.Error("un marqueur de registre est pose : le film ne reviendrait plus au rattrapage")
			}
			sum := col.CollectMatches(context.Background(), []string{"m1"})
			if sum.CarteNonResolue != 1 || sum.Errors != 0 || sum.Written != 0 || sum.NoKillFeed != 0 {
				t.Errorf("synthese : carte_non_resolue=%d erreurs=%d ecrits=%d sans_killfeed=%d, "+
					"attendu 1/0/0/0", sum.CarteNonResolue, sum.Errors, sum.Written, sum.NoKillFeed)
			}
		})
	}
}

// TestKillCollectorDecodeUnFilmSousSaCarte : LE CONTROLE NEGATIF. Sous SA carte, la bobine
// traverse la porte et va jusqu au decodeur (qui la refuse, faute de paquet de replication) : sans ce temoin, une
// porte qui ecarte TOUT passerait le test ci-dessus.
func TestKillCollectorDecodeUnFilmSousSaCarte(t *testing.T) {
	col := collecteurDeBobine(t)
	col.WithPositionCapture(nomsDeCarteFixes{noms: []string{carteDeLaBobine}}, catalogueDuDepot(t))
	avant := observability.LoadCounter(metricCarteNonResolue)

	outcome, _, err := col.CollectMatch(context.Background(), "m1")
	if outcome == OutcomeCarteNonResolue || errors.Is(err, decfilm.ErrCarteAbsente) {
		t.Fatalf("la bobine est ecartee sous SA carte (%q) : la porte refuse une carte resolue", carteDeLaBobine)
	}
	// LE DECODEUR A ETE ATTEINT : la bobine (trois chunks, pas de paquet de replication) y echoue,
	// et c est ce que la seconde moitie du message dit. Sans carte, il n etait jamais appele.
	if err == nil || !strings.Contains(err.Error(), "decodage m1") {
		t.Errorf("CollectMatch : err = %v, attendu l echec du DECODEUR (preuve que la porte de la "+
			"carte a laisse passer le film)", err)
	}
	if n := observability.LoadCounter(metricCarteNonResolue) - avant; n != 0 {
		t.Errorf("compteur %q : +%d sur une carte resolue", metricCarteNonResolue, n)
	}
}
