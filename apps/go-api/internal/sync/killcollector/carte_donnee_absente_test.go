package killcollector

// carte_donnee_absente_test.go — UNE DONNEE ABSENTE N EST PAS UNE PANNE, UNE PANNE N EST PAS UNE
// DONNEE ABSENTE (revue adverse du correctif J7 « carte obligatoire », 2026-09-27).
//
// Deux sens, deux tests.
//
//  1. Un match dont `match_registry.map_id` ET `map_name` sont vides (ou absent du registre) fait
//     rendre `ErrMatchMapUnknown` au resolveur de production (`platform/duckdb.ReplayMapRepo`).
//     C est une DONNEE qui manque : le match doit etre ecarte avant le telechargement, et mis de
//     cote s il arrive au decodeur — jamais range en erreur de decodage, ce qui le faisait garder,
//     telecharger et compter en `killsource_erreurs_decodage` a CHAQUE cycle, sans fin.
//     MUTATION QUI DOIT LE FAIRE ROUGIR : retirer `port.ErrMatchMapUnknown` de `carteNonResolue`.
//
//  2. Une erreur de LECTURE generique (base indisponible, delai) n ecarte PAS le match : ce n est
//     pas une carte absente, et l ecarter le sortirait de la liste de travail pour une panne
//     passagere. MUTATION QUI DOIT LE FAIRE ROUGIR : remplacer `carteNonResolue(err)` par
//     `err != nil` dans `RetenirLesMatchsAvecCarte`.

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
)

// cartesEnErreur : le resolveur de carte qui rend toujours la meme erreur.
type cartesEnErreur struct{ err error }

func (c cartesEnErreur) MapKeysForMatch(context.Context, string) (port.MatchMapKeys, error) {
	return port.MatchMapKeys{}, c.err
}

func (c cartesEnErreur) MapKeysForMap(context.Context, string) (port.MatchMapKeys, error) {
	return port.MatchMapKeys{}, c.err
}

func TestRetenirEcarteUnMatchSansIdentiteDeCarteEnBase(t *testing.T) {
	col := collecteurDeBobine(t)
	// LA SENTINELLE DU RESOLVEUR DE PRODUCTION, telle qu il la rend (non enveloppee).
	col.WithPositionCapture(cartesEnErreur{err: duckdb.ErrMatchMapUnknown}, catalogueDuDepot(t))

	retenus, ecartes := col.RetenirLesMatchsAvecCarte(context.Background(), []string{"m1", "m2"}, 8)
	if len(retenus) != 0 || len(ecartes) != 2 {
		t.Fatalf("retenus=%v ecartes=%v, attendu aucun retenu et 2 ecartes — un match sans identite "+
			"de carte en base est garde et TELECHARGE a chaque cycle", retenus, ecartes)
	}

	erreurs := observability.LoadCounter(metricDecodeError)
	outcome, _, err := col.CollectMatch(context.Background(), "m1")
	if err != nil || outcome != OutcomeCarteNonResolue {
		t.Fatalf("CollectMatch : outcome=%q err=%v, attendu %q sans erreur — une donnee absente est "+
			"rangee en PANNE", outcome, err, OutcomeCarteNonResolue)
	}
	if n := observability.LoadCounter(metricDecodeError) - erreurs; n != 0 {
		t.Errorf("compteur %q : +%d, attendu 0", metricDecodeError, n)
	}
	sum := col.CollectMatches(context.Background(), []string{"m1"})
	if sum.CarteNonResolue != 1 || sum.Errors != 0 {
		t.Errorf("synthese : carte_non_resolue=%d erreurs=%d, attendu 1/0", sum.CarteNonResolue, sum.Errors)
	}
}

func TestRetenirGardeUnMatchSurPanneDeLecture(t *testing.T) {
	col := collecteurDeBobine(t)
	panne := fmt.Errorf("replay map: lecture match_registry: %w", errors.New("delai depasse"))
	col.WithPositionCapture(cartesEnErreur{err: panne}, catalogueDuDepot(t))
	avant := observability.LoadCounter(metricCarteAvantTelechargement)

	retenus, ecartes := col.RetenirLesMatchsAvecCarte(context.Background(), []string{"m1", "m2"}, 8)
	if len(retenus) != 2 || len(ecartes) != 0 {
		t.Fatalf("retenus=%v ecartes=%v, attendu les 2 retenus — une panne passagere de lecture a "+
			"SORTI les matchs de la liste de travail", retenus, ecartes)
	}
	if n := observability.LoadCounter(metricCarteAvantTelechargement) - avant; n != 0 {
		t.Errorf("compteur %q : +%d, attendu 0", metricCarteAvantTelechargement, n)
	}
}
