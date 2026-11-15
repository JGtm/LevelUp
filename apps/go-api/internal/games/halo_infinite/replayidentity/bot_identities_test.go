package replayidentity

// bot_identities_test.go — LA PROJECTION DES BOTS VERS LE REGISTRE D'IDENTITE.
//
// Un bot NON EPINGLE (son slot tombe dans l'espace des humains de la table du film) est passe au
// registre quand le film le nomme : c'est le registre qui tranche, par les listes de session. Seul
// un bot sans nom est refuse. Les deux cas sont comptes (`killsource_bots_non_epingles_passes` et
// `_refuses`).
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : retirer un bot non epingle nomme de la projection ;
// projeter un bot sans nom ; retirer l'un des deux increments des compteurs de [BotIdentities].

import (
	"testing"

	"levelup/go-api/internal/observability"

	"github.com/JGtm/halo-infinite-film-decoder/decfilm"
)

// ajouter rend la tranche agrandie d un element nul et un pointeur vers lui. Le type des entrees
// du roster vit dans un paquet interne du decodeur : l inference generique le nomme a notre place.
func ajouter[E any](s []E) ([]E, *E) {
	var e E
	s = append(s, e)
	return s, &s[len(s)-1]
}

func TestBotIdentities_UnBotNonEpingleNommeEstPasseUnBotSansNomEstRefuse(t *testing.T) {
	res := &decfilm.Result{}
	bots, nomme := ajouter(res.Roster.Bots)
	nomme.Slot, nomme.BotID, nomme.Name = 1, 7, "343 Nomme"
	bots, sansNom := ajouter(bots)
	sansNom.Slot, sansNom.BotID, sansNom.Name = 3, 9, ""
	bots, epingle := ajouter(bots)
	epingle.Slot, epingle.BotID, epingle.Name = 9, 16, "343 Epingle"
	bots, sansNomEpingle := ajouter(bots)
	sansNomEpingle.Slot, sansNomEpingle.BotID, sansNomEpingle.Name = 10, 17, ""
	res.Roster.Bots = bots
	nonEpingles, premier := ajouter(res.Roster.UnpinnedBots)
	*premier = res.Roster.Bots[0]
	nonEpingles, second := ajouter(nonEpingles)
	*second = res.Roster.Bots[1]
	res.Roster.UnpinnedBots = nonEpingles

	passesAvant := observability.LoadCounter(MetricBotsNonEpinglesPasses)
	refusesAvant := observability.LoadCounter(MetricBotsNonEpinglesRefuses)
	ids := BotIdentities(res)
	if n := observability.LoadCounter(MetricBotsNonEpinglesPasses) - passesAvant; n != 1 {
		t.Errorf("%s a bouge de %d, attendu 1 : le bot non epingle nomme est passe au registre",
			MetricBotsNonEpinglesPasses, n)
	}
	if n := observability.LoadCounter(MetricBotsNonEpinglesRefuses) - refusesAvant; n != 1 {
		t.Errorf("%s a bouge de %d, attendu 1 : le bot non epingle sans nom est refuse",
			MetricBotsNonEpinglesRefuses, n)
	}
	if len(ids) != 2 || ids[0].BotID != 7 || ids[1].BotID != 16 {
		t.Fatalf("identites projetees = %+v, attendu le non epingle nomme (BotID 7) puis l'epingle (BotID 16)", ids)
	}
	if ids[0].FilmIndex != 1 || ids[0].Name != "343 Nomme"+decfilm.BotSuffix {
		t.Fatalf("identite du bot non epingle = %+v, attendu l'index 1 et son nom suffixe", ids[0])
	}
}

// TestBotIdentities_LEquipeLueVoyage : l equipe que BOT_METADATA ecrit pour un bot voyage jusqu a
// l identite du rejeu, copiee (pas partagee) ; un bot dont l equipe n est pas lue n en recoit
// aucune — l absence n est pas « aucune equipe ».
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : retirer la copie de `Team` dans [BotIdentities].
func TestBotIdentities_LEquipeLueVoyage(t *testing.T) {
	res := &decfilm.Result{}
	bots, lu := ajouter(res.Roster.Bots)
	lu.Slot, lu.BotID, lu.Name = 8, 44, "343 Sandwolf"
	zero := 0
	lu.Team = &zero
	bots, muet := ajouter(bots)
	muet.Slot, muet.BotID, muet.Name = 9, 6, "343 Donos"
	res.Roster.Bots = bots

	ids := BotIdentities(res)
	if len(ids) != 2 {
		t.Fatalf("identites projetees = %+v, attendu deux bots", ids)
	}
	if ids[0].Team == nil || *ids[0].Team != 0 {
		t.Fatalf("equipe de %s = %v, attendu 0", ids[0].Name, ids[0].Team)
	}
	if ids[0].Team == res.Roster.Bots[0].Team {
		t.Fatal("l equipe projetee partage le pointeur du resultat du decodage : elle doit etre copiee")
	}
	if ids[1].Team != nil {
		t.Fatalf("equipe de %s = %d, attendu aucune (non lue)", ids[1].Name, *ids[1].Team)
	}
}

// TestBotIdentities_LeBidZeroEstUnIdentifiant : le `bid` 0 que BOT_METADATA ecrit designe un bot
// (`bid(0.0)` existe en base, `343 Ritzy` sur `ca684191` et `4db2574e`). La copie decalee de son
// nom (`43 Ritzy`, slot 0, `bid` lu 0), non epinglee, est passee au meme titre : le registre la
// laisse sans nom quand un humain est liste a ce siege, la projection n'en decide pas. Les deux
// entrees gardent leur propre index.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : projection sans `BotIDLu` (le `bid` redevient vide, les
// vies sans nom) ; cle du compteur par `bid` seul (les deux entrees se confondent).
func TestBotIdentities_LeBidZeroEstUnIdentifiant(t *testing.T) {
	res := &decfilm.Result{}
	bots, copie := ajouter(res.Roster.Bots)
	copie.Slot, copie.BotID, copie.Name = 0, 0, "43 Ritzy"
	bots, vrai := ajouter(bots)
	vrai.Slot, vrai.BotID, vrai.Name = 24, 0, "343 Ritzy"
	res.Roster.Bots = bots
	nonEpingles, nonEpingle := ajouter(res.Roster.UnpinnedBots)
	*nonEpingle = res.Roster.Bots[0]
	res.Roster.UnpinnedBots = nonEpingles

	passesAvant := observability.LoadCounter(MetricBotsNonEpinglesPasses)
	ids := BotIdentities(res)
	if n := observability.LoadCounter(MetricBotsNonEpinglesPasses) - passesAvant; n != 1 {
		t.Errorf("%s a bouge de %d, attendu 1 : seule la copie decalee est non epinglee", MetricBotsNonEpinglesPasses, n)
	}
	if len(ids) != 2 || ids[0].FilmIndex != 0 || ids[1].FilmIndex != 24 {
		t.Fatalf("identites projetees = %+v, attendu la copie (index 0) et le vrai bot (index 24)", ids)
	}
	for _, id := range ids {
		if bid := id.Bid(); bid != "bid(0.0)" {
			t.Fatalf("bid de %s = %q, attendu bid(0.0) : le zero que le film ecrit est un identifiant", id.Name, bid)
		}
	}
}
