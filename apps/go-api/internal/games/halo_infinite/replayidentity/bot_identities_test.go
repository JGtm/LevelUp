package replayidentity

// bot_identities_test.go — UN BOT RETIRE DE LA PROJECTION SE COMPTE (lot J7.2 du
// PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, constat FK-1).
//
// Un bot NON EPINGLE par `killsource` (son slot tombe dans l espace des humains de la table du film)
// n est pas projete vers le registre d identite : ses vies ne sont nommees par personne. Ce retrait
// retirait le bot EN SILENCE ; il incremente desormais `killsource_bots_non_epingles`.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : retirer l increment du compteur dans [BotIdentities].

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/observability"
)

// ajouter rend la tranche agrandie d un element nul et un pointeur vers lui. Le type des entrees
// du roster vit dans un paquet interne du decodeur : l inference generique le nomme a notre place.
func ajouter[E any](s []E) ([]E, *E) {
	var e E
	s = append(s, e)
	return s, &s[len(s)-1]
}

func TestBotIdentities_RetraitDUnBotNonEpingleEstCompte(t *testing.T) {
	res := &decfilm.Result{}
	bots, contradictoire := ajouter(res.Roster.Bots)
	contradictoire.Slot, contradictoire.BotID, contradictoire.Name = 3, 7, "343 Contradictoire"
	bots, relais := ajouter(bots)
	relais.Slot, relais.BotID, relais.Name = 9, 16, "343 Relais"
	res.Roster.Bots = bots
	nonEpingles, nonEpingle := ajouter(res.Roster.UnpinnedBots)
	*nonEpingle = res.Roster.Bots[0]
	res.Roster.UnpinnedBots = nonEpingles

	avant := observability.LoadCounter(MetricBotsNonEpingles)
	ids := BotIdentities(res)
	if n := observability.LoadCounter(MetricBotsNonEpingles) - avant; n != 1 {
		t.Errorf("%s a bouge de %d, attendu 1 : le retrait d un bot non epingle est silencieux",
			MetricBotsNonEpingles, n)
	}
	if len(ids) != 1 || ids[0].BotID != 16 {
		t.Fatalf("identites projetees = %+v, attendu le seul bot epingle (BotID 16)", ids)
	}
}
