package killsource

// replis_du_decodage_test.go — LES COMPTES DE REPLIS DU DECODEUR DE MORTS (lot J8.7, 2026-09-27).
//
// Maillons tenus ici : chaque etage qui decide un repli le COMPTE, et [decodeCtx.replisDuResultat]
// porte chaque compte dans [Stats.Replis]. Le dernier maillon — des statistiques au compteur de la
// cuisson — est tenu par `replay/versement_des_replis_test.go`.
//
// MUTATION JOUEE (2026-09-27) : retirer `ChainesArretees: c.killEvents.chainesArretees` de
// [decodeCtx.replisDuResultat] fait rougir `TestReplisDuResultatPorteChaqueCompte` (« le champ
// ChainesArretees reste a zero »).

import (
	"context"
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// TestReplisDuResultatPorteChaqueCompte : chaque etage alimente, chaque champ de [ReplisDuDecodage]
// est non nul — un champ ajoute au type que la composition oublierait reste a zero et rougit.
func TestReplisDuResultatPorteChaqueCompte(t *testing.T) {
	c := &decodeCtx{
		walkRes:    &walkResult{desync: 1, horsRoster: 3, horsEnum: 4, largeurLibre: 5},
		roster:     &roster{nomsInventes: 6},
		feed:       &killFeed{},
		killEvents: &assistScan{chainesArretees: 8, rattrapes: 9},
		calib:      calibration{CarteLue: false, ControleDeCorruptionLu: false, PoigneeDecidee: true},
	}
	kills := []Kill{{Victim: "A", Feed: FeedTruth{Killer: "B"}}}
	unclaimed := []UnclaimedDeath{{}}
	r := c.replisDuResultat(kills, unclaimed, false)
	v := reflect.ValueOf(r)
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).Int() == 0 {
			t.Errorf("le champ %s reste a zero : son etage ne le verse pas au resultat", v.Type().Field(i).Name)
		}
	}
	if r.LibellesAutres != 2 {
		t.Errorf("libelles « Autres » %d (attendu 2 : une mort revendiquee, une orpheline)", r.LibellesAutres)
	}
	if got := c.replisDuResultat(nil, nil, true); got.SondeNonLancee != 0 {
		t.Errorf("sonde lancee comptee comme repli : %d", got.SondeNonLancee)
	}
}

// TestLeFiltreDeCredibiliteCompteSesRejets : chaque rejet du filtre de credibilite se compte sous
// SA cause, et un dead-state credible n en compte aucun.
func TestLeFiltreDeCredibiliteCompteSesRejets(t *testing.T) {
	res := &walkResult{deads: []deadRecord{
		{slot: 15, dead: types.DeadState{EnumA: 0, EnumB: 1, Val0c: 2}},  // credible
		{slot: 15, dead: types.DeadState{EnumA: 9, EnumB: 1}},            // victime hors roster
		{slot: 15, dead: types.DeadState{EnumA: 0, EnumB: -1}},           // tueur hors roster
		{slot: 15, dead: types.DeadState{EnumA: 0, EnumB: 1, Val0c: 12}}, // categorie hors enumeration
	}}
	res.selectCredible(&roster{nPlay: 4})
	if len(res.credible) != 1 || res.horsRoster != 2 || res.horsEnum != 1 {
		t.Fatalf("credibles %d, hors roster %d, hors enumeration %d — attendu 1, 2, 1",
			len(res.credible), res.horsRoster, res.horsEnum)
	}
}

// TestLeDecodageDeLaBobineV40PorteSesReplis : sur de vrais octets, les deux replis que tout decodage
// declenche (pied par argmax, type de chunk perdu) sont comptes, et le compte voyage dans le resultat.
//
// La bobine se decode SOUS SA CARTE (Fragmentation) : sans carte, [Decode] refuse le film depuis le
// 2026-09-27 ([ErrCarteAbsente]), et `repli_carte_absente_largeurs_par_defaut` n existe plus.
func TestLeDecodageDeLaBobineV40PorteSesReplis(t *testing.T) {
	carte := carteDuCatalogue(t, "Fragmentation")
	opts := DefaultOptions()
	opts.Carte = &carte
	res, err := Decode(context.Background(), miniBobineV40Film, chargerMiniBobineV40(t), &opts)
	if err != nil {
		t.Fatalf("decodage de la bobine v40 : %v", err)
	}
	if res.Stats.Replis.PiedParArgmax != 1 || res.Stats.Replis.TypeDeChunkPerdu != 1 {
		t.Errorf("pied par argmax %d, type de chunk perdu %d — attendu 1 et 1 par decodage",
			res.Stats.Replis.PiedParArgmax, res.Stats.Replis.TypeDeChunkPerdu)
	}
}
