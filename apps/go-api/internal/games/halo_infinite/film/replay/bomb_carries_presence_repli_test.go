package replay

// bomb_carries_presence_repli_test.go — LA PORTE DE PRESENCE PARTAGEE COMPTE CHAQUE CALQUE SOUS SON
// NOM (lot J8.7-bis du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, 2026-09-28).
//
// CE QUE LE LOT A CONSTATE : le crane et la bombe traversent la MEME porte (`carrierPresence.gate`),
// et son abstention « porteur sans aucune vie nommee » est un repli. Le calque du crane passait son
// compteur a la porte ; celui de la bombe passait nil — un portage de bombe sans vie nommee sortait
// SANS ETRE COMPTE, ni sous son nom ni sous celui du crane. La porte compte desormais en donnees, et
// chaque calque verse sous le nom de SON repli.
//
// MUTATIONS JOUEES (2026-09-28) : retirer le versement de `attachBombCarries` fait rougir le cas bombe
// (« 0, attendu 1 ») ; verser la bombe sous `NomCranePorteurSansVieNommee` le fait rougir aussi
// (le crane compte, la bombe non).

import (
	"context"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// replisDe rend le rapport d un compteur en table nom -> declenchements.
func replisDe(fb *fallback.Compteur) map[fallback.Nom]int {
	out := map[fallback.Nom]int{}
	for _, d := range fb.Rapport() {
		out[d.Nom] = d.Declenchements
	}
	return out
}

// TestLaPorteDePresenceCompteLaBombeSousSonNom : un portage de bombe dont le porteur n a aucune vie
// publiee passe la porte, et se compte sous `repli_bombe_porteur_sans_vie_nommee` — jamais sous le
// nom du crane.
func TestLaPorteDePresenceCompteLaBombeSousSonNom(t *testing.T) {
	fb := fallback.NouveauCompteur()
	doc := ReplayDocument{MatchID: "test", Coverage: &Coverage{}}
	opt := Options{Bomb: BombInput{CarryScanned: true}, WeaponChanges: []types.HeldWeaponChange{
		bombChange(10_000_000, 3, bombHeldFamily, grammar.NoWeaponVariant), // prise a 10 s
		bombChange(12_000_000, 3, bombTestOther, bombHeldFamily),           // lacher a 12 s
	}}
	// Le siege 3 est nomme par le pont (xuid 111), mais AUCUNE piste n est publiee : la porte
	// s abstient faute de vie nommee du porteur.
	attachBombCarries(context.Background(), &doc, opt, regDeTest(nil, map[uint32]uint64{3: 111}, nil),
		replayClock{origin: 0, step: 1000, frames: 100_000, fb: fb}, nil)
	if len(doc.BombCarries) != 1 {
		t.Fatalf("portages de bombe publies : %+v, attendu 1", doc.BombCarries)
	}
	got := replisDe(fb)
	if got[fallback.NomBombePorteurSansVieNommee] != 1 || got[fallback.NomCranePorteurSansVieNommee] != 0 {
		t.Fatalf("bombe : %d sous son nom et %d sous celui du crane, attendu 1 et 0 (rapport %v)",
			got[fallback.NomBombePorteurSansVieNommee], got[fallback.NomCranePorteurSansVieNommee], got)
	}
}

// TestLaPorteDePresenceCompteLeCraneSousSonNom : le pendant crane — quatre trains, aucun porteur n a
// de vie publiee, quatre declenchements sous le nom du crane, zero sous celui de la bombe.
func TestLaPorteDePresenceCompteLeCraneSousSonNom(t *testing.T) {
	recs, deaths := skullFixture()
	fb := fallback.NouveauCompteur()
	doc := ReplayDocument{MatchID: "test", Coverage: &Coverage{}}
	opt := Options{Skull: SkullInput{Scanned: true, Records: recs,
		Identity: objectives.ResolveRoundIdentity(recs, deaths, nil)}}
	attachSkullCarries(context.Background(), &doc, opt, IdentityRegistry{},
		replayClock{origin: 0, step: 1000, frames: 100_000, fb: fb}, nil)
	if len(doc.SkullCarries) != 4 {
		t.Fatalf("portages de crane publies : %d, attendu 4", len(doc.SkullCarries))
	}
	got := replisDe(fb)
	if got[fallback.NomCranePorteurSansVieNommee] != 4 || got[fallback.NomBombePorteurSansVieNommee] != 0 {
		t.Fatalf("crane : %d sous son nom et %d sous celui de la bombe, attendu 4 et 0 (rapport %v)",
			got[fallback.NomCranePorteurSansVieNommee], got[fallback.NomBombePorteurSansVieNommee], got)
	}
}
