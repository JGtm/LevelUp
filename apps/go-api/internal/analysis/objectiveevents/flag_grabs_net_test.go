package objectiveevents

// flag_grabs_net_test.go — LES CINQ CAS QUE LA REGLE DOIT TRANCHER, plus les refus.
//
// Les fixtures sont ecrites en MILLISECONDES et la fenetre de reference est celle mesuree le
// 2026-09-13 : 1,5 s. Les deux fixtures 1,4 s / 1,6 s encadrent la coupure — ce sont elles qui
// verrouillent le SENS de la comparaison (`<=` replie, `>` compte).

import (
	"testing"
	"time"
)

const fenetreRef = 1500 * time.Millisecond

// carry / carryOpen / dropped / home : des constructeurs courts, pour que les fixtures se
// lisent comme des chronologies et non comme des litteraux de struct.
func carry(t0, t1 int, xuid string) FlagSpan {
	return FlagSpan{State: FlagSpanCarried, StartMS: t0, EndMS: t1, XUID: xuid}
}

func carryOpen(t0, t1 int, xuid string) FlagSpan {
	return FlagSpan{State: FlagSpanCarriedOpen, StartMS: t0, EndMS: t1, XUID: xuid}
}

func dropped(t0, t1 int) FlagSpan {
	return FlagSpan{State: FlagSpanDropped, StartMS: t0, EndMS: t1}
}

func home(t0, t1 int) FlagSpan {
	return FlagSpan{State: FlagSpanHome, StartMS: t0, EndMS: t1}
}

// netOf rend le couple (brut, net) d un joueur.
func netOf(t *testing.T, res FlagGrabsNetResult, xuid string) (int, int) {
	t.Helper()
	for _, p := range res.Players {
		if p.XUID == xuid {
			return p.Raw, p.Net
		}
	}
	return 0, 0
}

func TestNetFlagGrabs_JonglageReplie(t *testing.T) {
	// A prend, lache a 10 000, reprend 800 ms plus tard, relache, reprend encore : UN geste.
	tr := FlagTrack{Team: 0, Spans: []FlagSpan{
		carry(5_000, 10_000, "A"),
		dropped(10_000, 10_800),
		carry(10_800, 14_000, "A"),
		dropped(14_000, 15_000),
		carry(15_000, 20_000, "A"),
	}}
	res := NetFlagGrabs(nil, []FlagTrack{tr}, fenetreRef)
	if !res.Measured || res.WindowMS != 1500 {
		t.Fatalf("resultat non mesure ou fenetre fausse : %+v", res)
	}
	brut, net := netOf(t, res, "A")
	if brut != 3 || net != 1 {
		t.Fatalf("jonglage : brut=%d net=%d, attendu 3 et 1", brut, net)
	}
}

func TestNetFlagGrabs_PasseDeMainCompte(t *testing.T) {
	// A lache, B reprend 300 ms plus tard : c est une passe, pas un jonglage. Puis A reprend
	// 300 ms apres B — le portage precedent n est pas le sien, sa prise compte aussi.
	tr := FlagTrack{Team: 1, Spans: []FlagSpan{
		carry(1_000, 4_000, "A"),
		carry(4_300, 6_000, "B"),
		carry(6_300, 9_000, "A"),
	}}
	res := NetFlagGrabs(nil, []FlagTrack{tr}, fenetreRef)
	if brut, net := netOf(t, res, "A"); brut != 2 || net != 2 {
		t.Fatalf("passe de main (A) : brut=%d net=%d, attendu 2 et 2", brut, net)
	}
	if brut, net := netOf(t, res, "B"); brut != 1 || net != 1 {
		t.Fatalf("passe de main (B) : brut=%d net=%d, attendu 1 et 1", brut, net)
	}
}

func TestNetFlagGrabs_RepriseApresRetourAuSocleCompte(t *testing.T) {
	// Le drapeau RENTRE CHEZ LUI entre les deux portages du meme joueur. L ecart est de
	// 400 ms — sous la fenetre —, et pourtant la reprise est une VRAIE prise.
	tr := FlagTrack{Team: 0, Spans: []FlagSpan{
		carry(1_000, 4_000, "A"),
		home(4_000, 4_400),
		carry(4_400, 8_000, "A"),
	}}
	res := NetFlagGrabs(nil, []FlagTrack{tr}, fenetreRef)
	if brut, net := netOf(t, res, "A"); brut != 2 || net != 2 {
		t.Fatalf("retour au socle : brut=%d net=%d, attendu 2 et 2", brut, net)
	}
	// TEMOIN : la MEME chronologie sans l etat `home` replie bien la seconde prise — c est
	// donc le socle qui tranche, pas la duree.
	sans := FlagTrack{Team: 0, Spans: []FlagSpan{
		carry(1_000, 4_000, "A"),
		dropped(4_000, 4_400),
		carry(4_400, 8_000, "A"),
	}}
	res2 := NetFlagGrabs(nil, []FlagTrack{sans}, fenetreRef)
	if brut, net := netOf(t, res2, "A"); brut != 2 || net != 1 {
		t.Fatalf("temoin sans socle : brut=%d net=%d, attendu 2 et 1", brut, net)
	}
}

func TestNetFlagGrabs_FilmTronqueNeReplieRien(t *testing.T) {
	// Un portage OUVERT court jusqu a la fin de l axe : sa fin est une borne haute. La prise
	// suivante du MEME joueur reste NETTE, meme a 100 ms d ecart apparent.
	tr := FlagTrack{Team: 0, Spans: []FlagSpan{
		carryOpen(1_000, 30_000, "A"),
		carry(30_100, 33_000, "A"),
	}}
	res := NetFlagGrabs(nil, []FlagTrack{tr}, fenetreRef)
	if brut, net := netOf(t, res, "A"); brut != 2 || net != 2 {
		t.Fatalf("film tronque : brut=%d net=%d, attendu 2 et 2", brut, net)
	}
}

func TestNetFlagGrabs_CoupureA1400Et1600(t *testing.T) {
	// LA COUPURE : 1,4 s replie (ecart <= fenetre), 1,6 s compte.
	cas := []struct {
		nom       string
		ecartMS   int
		netAttend int
	}{
		{"1,4 s : jonglage", 1_400, 1},
		{"1,5 s : la borne elle-meme, repliee", 1_500, 1},
		{"1,6 s : une vraie reprise", 1_600, 2},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			tr := FlagTrack{Spans: []FlagSpan{
				carry(1_000, 5_000, "A"),
				carry(5_000+c.ecartMS, 9_000+c.ecartMS, "A"),
			}}
			res := NetFlagGrabs(nil, []FlagTrack{tr}, fenetreRef)
			if brut, net := netOf(t, res, "A"); brut != 2 || net != c.netAttend {
				t.Fatalf("%s : brut=%d net=%d, attendu 2 et %d", c.nom, brut, net, c.netAttend)
			}
		})
	}
}

func TestNetFlagGrabs_DeuxDrapeauxNeSeReplientPas(t *testing.T) {
	// Le meme joueur enchaine deux drapeaux DIFFERENTS a 200 ms : deux prises, deux nettes.
	a := FlagTrack{Team: 0, Spans: []FlagSpan{carry(1_000, 4_000, "A")}}
	b := FlagTrack{Team: 1, Spans: []FlagSpan{carry(4_200, 7_000, "A")}}
	res := NetFlagGrabs(nil, []FlagTrack{a, b}, fenetreRef)
	if brut, net := netOf(t, res, "A"); brut != 2 || net != 2 {
		t.Fatalf("deux drapeaux : brut=%d net=%d, attendu 2 et 2", brut, net)
	}
}

func TestNetFlagGrabs_PortageSansXuidResteLePrecedent(t *testing.T) {
	// Un portage que le pont n a pas nomme ne compte a personne, MAIS il reste le portage
	// precedent : la prise de A qui le suit ne se replie pas sur sa propre prise d avant.
	tr := FlagTrack{Spans: []FlagSpan{
		carry(1_000, 4_000, "A"),
		carry(4_200, 5_000, ""),
		carry(5_200, 8_000, "A"),
	}}
	res := NetFlagGrabs(nil, []FlagTrack{tr}, fenetreRef)
	if brut, net := netOf(t, res, "A"); brut != 2 || net != 2 {
		t.Fatalf("portage anonyme intercalaire : brut=%d net=%d, attendu 2 et 2", brut, net)
	}
}

func TestNetFlagGrabs_FenetreAbsenteNePubliePas(t *testing.T) {
	tr := FlagTrack{Spans: []FlagSpan{carry(1_000, 4_000, "A"), carry(4_100, 6_000, "A")}}
	res := NetFlagGrabs(nil, []FlagTrack{tr}, 0)
	if res.Measured {
		t.Fatal("fenetre absente : le resultat se declare mesure")
	}
	if len(res.Players) != 0 {
		t.Fatalf("fenetre absente : %d joueurs publies, attendu 0 (jamais des zeros)", len(res.Players))
	}
}

func TestNetFlagGrabs_OuverturesDeLOracle(t *testing.T) {
	evs := []NamedEvent{
		{Stat: StatFlagGrabs}, {Stat: StatFlagGrabs}, {Stat: StatFlagSteals},
		{Stat: StatFlagCaptures}, {Stat: StatFlagReturns},
	}
	res := NetFlagGrabs(evs, nil, fenetreRef)
	if res.Openings != 3 {
		t.Fatalf("openings=%d, attendu 3 (2 prises + 1 vol)", res.Openings)
	}
}

func TestNetFlagGrabs_OrdreDEntreeIndifferent(t *testing.T) {
	// La MEME chronologie, entree a l envers, rend le meme compte.
	spans := []FlagSpan{
		carry(15_000, 20_000, "A"),
		carry(10_800, 14_000, "A"),
		carry(5_000, 10_000, "A"),
	}
	res := NetFlagGrabs(nil, []FlagTrack{{Spans: spans}}, fenetreRef)
	if brut, net := netOf(t, res, "A"); brut != 3 || net != 1 {
		t.Fatalf("ordre inverse : brut=%d net=%d, attendu 3 et 1", brut, net)
	}
}

func TestNetFlagGrabs_NetJamaisSuperieurAuBrut(t *testing.T) {
	// Le controle 1 du rapport d etape 0, joue sur une chronologie melangee.
	tr := FlagTrack{Spans: []FlagSpan{
		carry(1_000, 3_000, "A"), carry(3_200, 5_000, "A"), carry(5_100, 6_000, "B"),
		home(6_000, 6_500), carry(6_500, 9_000, "B"), carryOpen(9_200, 40_000, "A"),
	}}
	for _, w := range []time.Duration{time.Second, fenetreRef, 8 * time.Second} {
		res := NetFlagGrabs(nil, []FlagTrack{tr}, w)
		for _, p := range res.Players {
			if p.Net > p.Raw {
				t.Fatalf("fenetre %v : %s net=%d > brut=%d", w, p.XUID, p.Net, p.Raw)
			}
		}
	}
}
