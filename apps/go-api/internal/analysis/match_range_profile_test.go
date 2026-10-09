package analysis

// match_range_profile_test.go — ce que ces tests verrouillent, dans l'ordre d'importance :
//
//  1. LA MÉDIANE DU LOBBY N'EST PAS LA MOYENNE DES MÉDIANES PAR JOUEUR. C'est l'erreur la
//     plus facile du fichier et la moins visible à l'écran : les deux nombres se ressemblent
//     tant que les effectifs sont égaux, et divergent exactement quand ils ne le sont pas —
//     c'est-à-dire tout le temps, la couverture des positions étant inégale d'un joueur à
//     l'autre.
//  2. un joueur sans frag mesuré est ABSENT, jamais publié à zéro (un zéro se lirait « il
//     fragge au corps à corps ») ;
//  3. l'écart au lobby est SIGNÉ, dans le bon sens ;
//  4. un match sans aucun frag mesuré n'est pas rendu.

import (
	"math"
	"testing"
	"time"
)

const mrpMatch = "m_portee_001"

func mrpKill(matchID, killer string, timeMS int64, dist float64) MeasuredKill {
	return MeasuredKill{
		MatchID: matchID, KillerXUID: killer, TimeMS: timeMS,
		Side: SideKiller, DistanceM: dist,
	}
}

func mrpScope(ids ...string) []MatchRangeMatch {
	out := make([]MatchRangeMatch, 0, len(ids))
	for i, id := range ids {
		out = append(out, MatchRangeMatch{
			MatchID:  id,
			PlayedAt: time.Date(2026, 9, 21, 20, i, 0, 0, time.UTC),
			MapName:  "Live Fire",
		})
	}
	return out
}

func mrpPresque(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// TestMatchRangeProfiles_MedianeLobbySurLesFrags : LE test du lot.
//
// A frague à 10, 10, 10, 10, 10 m (5 frags) ; B frague à 40 m (1 frag).
//
//	médiane du lobby SUR LES FRAGS     : la série triée est 10,10,10,10,10,40 -> 10 m ;
//	moyenne des médianes PAR JOUEUR    : (10 + 40) / 2 = 25 m.
//
// Un référentiel à 25 m dirait que A joue 15 m TROP COURT alors qu'il joue exactement à la
// distance de son lobby. C'est cet écart-là que la ligne du bas mesure.
func TestMatchRangeProfiles_MedianeLobbySurLesFrags(t *testing.T) {
	kills := []MeasuredKill{
		mrpKill(mrpMatch, "A", 1, 10), mrpKill(mrpMatch, "A", 2, 10),
		mrpKill(mrpMatch, "A", 3, 10), mrpKill(mrpMatch, "A", 4, 10),
		mrpKill(mrpMatch, "A", 5, 10),
		mrpKill(mrpMatch, "B", 6, 40),
	}
	got := MatchRangeProfiles(MatchRangeInput{
		Kills:        kills,
		Matches:      mrpScope(mrpMatch),
		Publish:      map[string]string{"A": "Aya", "B": "Bo"},
		PublishOrder: []string{"A", "B"},
	})
	if len(got) != 1 {
		t.Fatalf("profils = %d, want 1 : %+v", len(got), got)
	}
	p := got[0]
	if !mrpPresque(p.LobbyMedianM, 10) {
		t.Errorf("LobbyMedianM = %v, want 10 (mediane DES FRAGS ; 25 = la moyenne des "+
			"medianes par joueur, qui est le piege)", p.LobbyMedianM)
	}
	if p.LobbyMeasured != 6 {
		t.Errorf("LobbyMeasured = %d, want 6 (tous les frags du match)", p.LobbyMeasured)
	}
	if len(p.Players) != 2 {
		t.Fatalf("joueurs = %d, want 2 : %+v", len(p.Players), p.Players)
	}
	if p.Players[0].XUID != "A" || p.Players[0].Gamertag != "Aya" || p.Players[0].Measured != 5 {
		t.Errorf("joueur[0] = %+v, want A/Aya/5 frags (ordre de PublishOrder)", p.Players[0])
	}
	// L'écart est SIGNÉ : A joue AU NIVEAU de son lobby, B nettement plus loin.
	if !mrpPresque(p.Players[0].LobbyDeltaM, 0) {
		t.Errorf("A LobbyDeltaM = %v, want 0", p.Players[0].LobbyDeltaM)
	}
	if !mrpPresque(p.Players[1].LobbyDeltaM, 30) {
		t.Errorf("B LobbyDeltaM = %v, want +30 (40 - 10)", p.Players[1].LobbyDeltaM)
	}
}

// TestMatchRangeProfiles_EcartNegatif : le signe tient aussi vers le bas — un joueur qui
// fragge plus court que son lobby a un écart NÉGATIF (la ligne de front du nuage).
func TestMatchRangeProfiles_EcartNegatif(t *testing.T) {
	kills := []MeasuredKill{
		mrpKill(mrpMatch, "A", 1, 5),
		mrpKill(mrpMatch, "B", 2, 25), mrpKill(mrpMatch, "B", 3, 35),
	}
	got := MatchRangeProfiles(MatchRangeInput{
		Kills: kills, Matches: mrpScope(mrpMatch),
		Publish: map[string]string{"A": "Aya"}, PublishOrder: []string{"A"},
	})
	if len(got) != 1 || len(got[0].Players) != 1 {
		t.Fatalf("profils/joueurs inattendus : %+v", got)
	}
	// Lobby : 5, 25, 35 -> médiane 25. A est 20 m en dessous.
	if !mrpPresque(got[0].LobbyMedianM, 25) {
		t.Fatalf("LobbyMedianM = %v, want 25", got[0].LobbyMedianM)
	}
	if !mrpPresque(got[0].Players[0].LobbyDeltaM, -20) {
		t.Errorf("LobbyDeltaM = %v, want -20", got[0].Players[0].LobbyDeltaM)
	}
	// Le lobby porte les 3 frags, même si un SEUL joueur est publié : c'est tout l'objet
	// du référentiel.
	if got[0].LobbyMeasured != 3 {
		t.Errorf("LobbyMeasured = %d, want 3 (le lobby entier, pas le joueur publie)",
			got[0].LobbyMeasured)
	}
}

// TestMatchRangeProfiles_JoueurSansFragMesureAbsent : le joueur publiable qui n'a aucun frag
// mesuré sur ce match n'y a PAS de ligne — jamais une médiane à zéro.
func TestMatchRangeProfiles_JoueurSansFragMesureAbsent(t *testing.T) {
	got := MatchRangeProfiles(MatchRangeInput{
		Kills:        []MeasuredKill{mrpKill(mrpMatch, "A", 1, 12)},
		Matches:      mrpScope(mrpMatch),
		Publish:      map[string]string{"A": "Aya", "Z": "Zero"},
		PublishOrder: []string{"A", "Z"},
	})
	if len(got) != 1 {
		t.Fatalf("profils = %d, want 1", len(got))
	}
	if len(got[0].Players) != 1 || got[0].Players[0].XUID != "A" {
		t.Fatalf("joueurs = %+v, want la seule ligne de A (Z sans frag mesure = absent)",
			got[0].Players)
	}
}

// TestMatchRangeProfiles_MatchSansMesureAbsent : un match dont aucun frag n'est mesuré (film
// non décodé) n'a pas de référentiel — il sort de la liste, il n'y figure pas à zéro.
func TestMatchRangeProfiles_MatchSansMesureAbsent(t *testing.T) {
	got := MatchRangeProfiles(MatchRangeInput{
		Kills:        []MeasuredKill{mrpKill("m_decode", "A", 1, 12)},
		Matches:      mrpScope("m_decode", "m_sans_film"),
		Publish:      map[string]string{"A": "Aya"},
		PublishOrder: []string{"A"},
	})
	if len(got) != 1 || got[0].MatchID != "m_decode" {
		t.Fatalf("profils = %+v, want le seul match decode", got)
	}
}

// TestMatchRangeProfiles_CoteVictimeIgnore : une mesure lue côté victime est LE MÊME frag.
// La compter déplacerait la médiane du lobby — ici, la faire passer de 10 à 15 m.
func TestMatchRangeProfiles_CoteVictimeIgnore(t *testing.T) {
	victime := mrpKill(mrpMatch, "A", 1, 40)
	victime.Side = SideVictim
	got := MatchRangeProfiles(MatchRangeInput{
		Kills:        []MeasuredKill{mrpKill(mrpMatch, "A", 2, 10), victime},
		Matches:      mrpScope(mrpMatch),
		Publish:      map[string]string{"A": "Aya"},
		PublishOrder: []string{"A"},
	})
	if len(got) != 1 || got[0].LobbyMeasured != 1 || !mrpPresque(got[0].LobbyMedianM, 10) {
		t.Fatalf("profil = %+v, want 1 frag mesure a 10 m (le cote victime est le MEME frag)",
			got)
	}
}

// TestMatchRangeProfiles_OrdreDeterministe : un xuid publiable absent de PublishOrder est
// publié quand même, après les autres et par ordre de xuid — jamais perdu, jamais au gré
// d'un parcours de map (l'ordre des séries du nuage changerait d'une requête à l'autre).
func TestMatchRangeProfiles_OrdreDeterministe(t *testing.T) {
	kills := []MeasuredKill{
		mrpKill(mrpMatch, "A", 1, 10), mrpKill(mrpMatch, "C", 2, 20),
		mrpKill(mrpMatch, "B", 3, 30),
	}
	got := MatchRangeProfiles(MatchRangeInput{
		Kills: kills, Matches: mrpScope(mrpMatch),
		Publish:      map[string]string{"A": "Aya", "B": "Bo", "C": "Cy"},
		PublishOrder: []string{"C"},
	})
	if len(got) != 1 {
		t.Fatalf("profils = %d, want 1", len(got))
	}
	want := []string{"C", "A", "B"}
	for i, x := range want {
		if got[0].Players[i].XUID != x {
			t.Fatalf("ordre = %+v, want %v", got[0].Players, want)
		}
	}
}
