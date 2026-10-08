package killsource

// morts_de_bot_test.go — LES MORTS QUI TOUCHENT UN BOT : un dead-state ne decrit qu une mort, une
// mort de bot peut venir de sa propre source, le recollage ne prend pas une mort que le film donne a
// un autre tueur, et le nom d un bot se lit a l instant de la ligne.
//
// Chaque test nomme la mutation qui doit le faire rougir.

import "testing"

// rosterABots : K (indice 0) et V (indice 1) epingles par la table du film, deux bots epingles par
// BOT_METADATA sur les indices 3 et 4.
func rosterABots() *roster {
	return &roster{
		names:   []string{"K", "V", "343 A" + BotSuffix, "343 B" + BotSuffix},
		perm:    []int{0, 1, -1, 2, 3},
		pin:     map[int]int{0: 0, 1: 1, 3: 2, 4: 3},
		seatPin: map[int]bool{0: true, 1: true},
		nPlay:   5,
	}
}

// passeDesBots : la passe hybride jouee jusqu au temps 5 sur un feed et des dead-states donnes.
func passeDesBots(kf *killFeed, r *roster, recs []killEventRec, cands []candidate) *pass {
	kf.xuidDe = map[string]uint64{}
	kf.resoudreCouples(recs, r, cands)
	c := &decodeCtx{roster: r, feed: kf, opts: DefaultOptions(), scanCands: cands, film: &film{}}
	p := &pass{ctx: c, byTime: map[int]Kill{}, botUsed: map[[3]int]bool{}, fantomes: map[int]bool{}}
	p.population(nil, cands, 0)
	p.runStrong()
	p.runSelfSource()
	p.runBots()
	p.runBotKillers()
	return p
}

// TestDeuxKillsVoisinsSurDeuxBotsGardentChacunLeurMort — K tue le bot A a 1000 et le bot B a 2600 :
// chaque kill est dans la fenetre de l autre dead-state. Les deux lignes sortent, chacune avec SON
// bot.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : retirer le complement de [pass.runBots] (le second kill retrouve
// au premier candidat de sa fenetre le dead-state du premier, deja servi, et perd sa ligne).
func TestDeuxKillsVoisinsSurDeuxBotsGardentChacunLeurMort(t *testing.T) {
	kf := &killFeed{events: []feedEvent{{timeMS: 1000, killer: "K"}, {timeMS: 2600, killer: "K"}},
		names: []string{"K", "V"}}
	p := passeDesBots(kf, rosterABots(), nil, []candidate{
		{chunk: 1, pidx: 1, bit: 10, ms: 996, victim: 3, killer: 0, tag: 0xacd1cff4},
		{chunk: 1, pidx: 2, bit: 10, ms: 2596, victim: 4, killer: 0, tag: 0xacd1cff4},
	})
	for ms, bot := range map[int]string{1000: "343 A" + BotSuffix, 2600: "343 B" + BotSuffix} {
		k, ok := p.byTime[ms]
		if !ok || k.Victim != bot || k.Feed.Killer != "K" || k.Read.Origin != OriginBot {
			t.Errorf("instant %d : publie %v, victime %q, credit %q, origine %q — attendu %q tue par K",
				ms, ok, k.Victim, k.Feed.Killer, k.Read.Origin, bot)
		}
	}
}

// TestUnBotTueParSaPropreSourcePublieLeCreditDuFeed — K est credite du kill d un bot dont le
// dead-state designe le bot lui-meme (chute, source globale) : la ligne sort, credit K, divergence
// levee — le temps 3 de l hybride, pour la population des bots.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : retirer la seconde passe (`deLaVictime`) de
// [decodeCtx.affecterLesMortsDeBot].
func TestUnBotTueParSaPropreSourcePublieLeCreditDuFeed(t *testing.T) {
	kf := &killFeed{events: []feedEvent{{timeMS: 1000, killer: "K"}}, names: []string{"K", "V"}}
	p := passeDesBots(kf, rosterABots(), nil, []candidate{
		{chunk: 1, pidx: 1, bit: 10, ms: 998, victim: 3, killer: 3, tag: 0x00403594},
	})
	k, ok := p.byTime[1000]
	if !ok || k.Victim != "343 A"+BotSuffix || k.Feed.Killer != "K" || !k.Diverges {
		t.Fatalf("instant 1000 : publie %v, victime %q, credit %q, divergence %v — attendu le bot A, "+
			"credit K, divergence levee", ok, k.Victim, k.Feed.Killer, k.Diverges)
	}
}

// TestLeRecollageNePrendPasUneMortQueLeFilmDonneAUnAutreTueur — K tue (kill sans mort) a 1000, V
// meurt (mort sans kill) a 1050, et le kill-event 85 de 1048 ecrit « le bot A tue V ». Le kill de K
// ne se recolle pas sur la mort de V : elle reste orpheline, et le temps 5 la publie tuee par A.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : retirer [resolveurDeCouples.mortEcriteDUnAutreTueur] du
// recollage.
func TestLeRecollageNePrendPasUneMortQueLeFilmDonneAUnAutreTueur(t *testing.T) {
	kf := &killFeed{events: []feedEvent{{timeMS: 1000, killer: "K"}, {timeMS: 1050, victim: "V"}},
		names: []string{"K", "V"}}
	recs := []killEventRec{{ms: 1048, chunk: 1, pidx: 2, bit: 7, fields: killEventFields{killer: 3, victim: 1}}}
	p := passeDesBots(kf, rosterABots(), recs, []candidate{
		{chunk: 1, pidx: 2, bit: 30, ms: 1048, victim: 1, killer: 3, tag: 0xacd1cff4},
	})
	if len(kf.fab) != 0 || len(kf.orphK) != 1 || len(kf.orphD) != 1 {
		t.Fatalf("recolles %d, kills orphelins %d, morts orphelines %d — attendu 0, 1, 1", len(kf.fab),
			len(kf.orphK), len(kf.orphD))
	}
	k, ok := p.byTime[1050]
	if !ok || k.Victim != "V" || k.Feed.Killer != "343 A"+BotSuffix || k.Read.Origin != OriginBotKiller {
		t.Errorf("instant 1050 : publie %v, victime %q, tueur %q, origine %q — attendu V tue par le bot A",
			ok, k.Victim, k.Feed.Killer, k.Read.Origin)
	}
}

// TestUneMortNeSeRecollePasDeuxFois — deux kills sans mort (K a 1000, K a 1100) et une seule mort
// sans kill (V a 1200) : le premier kill la prend, le second reste orphelin.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : retirer `res.prisMort[i+d]` de la garde du recollage.
func TestUneMortNeSeRecollePasDeuxFois(t *testing.T) {
	kf := &killFeed{events: []feedEvent{{timeMS: 1000, killer: "K"}, {timeMS: 1100, killer: "K"},
		{timeMS: 1200, victim: "V"}}, names: []string{"K", "V"}, xuidDe: map[string]uint64{}}
	kf.resoudreCouples(nil, rosterABots(), nil)
	if len(kf.fab) != 1 || kf.fab[0].timeMS != 1000 || len(kf.orphK) != 1 || kf.orphK[0].timeMS != 1100 {
		t.Fatalf("recolles %+v, orphelins %+v — attendu le seul kill de 1000 recolle, celui de 1100 "+
			"orphelin", kf.fab, kf.orphK)
	}
}

// TestLeNomDuBotSeLitALInstant — deux bots se succedent sur l indice 3 : A declare de 10 a 20 ms,
// B a partir de 50 ms. Une ligne a 15 ms nomme A, une ligne a 60 ms nomme B ; entre les deux, le
// film se tait et le nom du roster (le dernier declare) reste.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : publier le nom du roster sans lire les declarations
// ([pass.nomPubliableA] reduit a [pass.nomPubliable]).
func TestLeNomDuBotSeLitALInstant(t *testing.T) {
	kf := &killFeed{names: []string{"K"}}
	bm := botMeta{NBots: 1, Bots: []bot{
		{Slot: 3, BotID: 1, Name: "343 A", declarations: []BotDeclaration{{FromUS: 10_000, ToUS: 20_000}}},
		{Slot: 3, BotID: 2, Name: "343 B", declarations: []BotDeclaration{{FromUS: 50_000}}},
	}}
	r := buildRoster(kf, bm, true, FilmTable{}, indexParMotif{})
	r.perm = []int{0, -1, -1, 1}
	p := &pass{ctx: &decodeCtx{roster: r, film: &film{}}}
	for ms, attendu := range map[int]string{15: "343 A" + BotSuffix, 60: "343 B" + BotSuffix,
		30: "343 B" + BotSuffix} {
		if nom, ok := p.nomPubliableA(3, ms); !ok || nom != attendu {
			t.Errorf("a %d ms : %q (%v), attendu %q", ms, nom, ok, attendu)
		}
	}
	if nom, ok := p.ctx.nomDuBotA(3, 30); ok {
		t.Errorf("a 30 ms aucune declaration ne couvre l instant, et %q est rendu", nom)
	}
}

// rosterAvecW : K, V et W epingles par la table du film, le bot A epingle par BOT_METADATA.
func rosterAvecW() *roster {
	return &roster{
		names:   []string{"K", "V", "W", "343 A" + BotSuffix},
		perm:    []int{0, 1, 2, 3},
		pin:     map[int]int{0: 0, 1: 1, 2: 2, 3: 3},
		seatPin: map[int]bool{0: true, 1: true, 2: true},
		nPlay:   4,
	}
}

// verifierLigne : la ligne publiee a `ms` porte ce tueur, cette victime et cette origine.
func verifierLigne(t *testing.T, p *pass, ms int, tueur, victime string, origine Origin) {
	t.Helper()
	k, ok := p.byTime[ms]
	if !ok || k.Feed.Killer != tueur || k.Victim != victime || k.Read.Origin != origine {
		t.Errorf("instant %d : publie %v, tueur %q, victime %q, origine %q — attendu %q tue %q (%q)",
			ms, ok, k.Feed.Killer, k.Victim, k.Read.Origin, tueur, victime, origine)
	}
}

// TestUnKillDejaExpliqueNePrendPasLaMortDeBotDUnAutreKill — K tue V (kill 1000, mort 1001 recollee,
// publiee au temps 1) puis un bot (kill seul 1010). Le dead-state du bot (1004) est plus proche du
// kill 1000, qui porte deja sa ligne : il revient au kill 1010, comme avant le lot.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : affecter les morts de bot du plus proche au plus lointain sur
// TOUS les kills, deja expliques compris, au lieu de la premiere passe puis du complement.
func TestUnKillDejaExpliqueNePrendPasLaMortDeBotDUnAutreKill(t *testing.T) {
	kf := &killFeed{events: []feedEvent{{timeMS: 1000, killer: "K"}, {timeMS: 1001, victim: "V"},
		{timeMS: 1010, killer: "K"}}, names: []string{"K", "V"}}
	p := passeDesBots(kf, rosterABots(), nil, []candidate{
		{chunk: 1, pidx: 1, bit: 10, ms: 999, victim: 1, killer: 0, tag: 0xacd1cff4},
		{chunk: 1, pidx: 2, bit: 10, ms: 1004, victim: 3, killer: 0, tag: 0xacd1cff4},
	})
	verifierLigne(t, p, 1000, "K", "V", OriginCredit)
	verifierLigne(t, p, 1010, "K", "343 A"+BotSuffix, OriginBot)
}

// TestUneMortDeBotNeRemplacePasLaLigneDUnKillVoisin — la variante au temps 3 : la ligne de V a 1000
// vient de sa propre source, et le dead-state du bot tombe a 1000 a la milliseconde. Le kill seul de
// 1010 le prend toujours ; la ligne de V reste.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : la meme que ci-dessus.
func TestUneMortDeBotNeRemplacePasLaLigneDUnKillVoisin(t *testing.T) {
	kf := &killFeed{events: []feedEvent{{timeMS: 1000, killer: "K"}, {timeMS: 1001, victim: "V"},
		{timeMS: 1010, killer: "K"}}, names: []string{"K", "V"}}
	p := passeDesBots(kf, rosterABots(), nil, []candidate{
		{chunk: 1, pidx: 1, bit: 10, ms: 1001, victim: 1, killer: 1, tag: 0xacd1cff4},
		{chunk: 1, pidx: 2, bit: 10, ms: 1000, victim: 3, killer: 0, tag: 0xacd1cff4},
	})
	verifierLigne(t, p, 1000, "K", "V", OriginSelfSource)
	verifierLigne(t, p, 1010, "K", "343 A"+BotSuffix, OriginBot)
}

// TestLeRecollageNePrendPasUneMortQueLeDeadStateDonneAUnAutreTueur — deux kills de K (1000, 1001),
// deux morts sans kill (V 1002, W 1003), aucun kill-event ; le dead-state dit « K tue V » et « le bot
// A tue W ». Le second kill ne va pas chercher W au-dela de V : la ligne « A tue W » reste.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : retirer [resolveurDeCouples.mortLueDUnAutreTueur] du recollage
// ET y passer a la mort suivante sur une mort refusee (chacune des deux gardes suffit ici).
func TestLeRecollageNePrendPasUneMortQueLeDeadStateDonneAUnAutreTueur(t *testing.T) {
	kf := &killFeed{events: []feedEvent{{timeMS: 1000, killer: "K"}, {timeMS: 1001, killer: "K"},
		{timeMS: 1002, victim: "V"}, {timeMS: 1003, victim: "W"}}, names: []string{"K", "V", "W"}}
	p := passeDesBots(kf, rosterAvecW(), nil, []candidate{
		{chunk: 1, pidx: 1, bit: 10, ms: 1000, victim: 1, killer: 0, tag: 0xacd1cff4},
		{chunk: 1, pidx: 2, bit: 10, ms: 1002, victim: 2, killer: 3, tag: 0xacd1cff4},
	})
	for _, c := range kf.fab {
		if c.victim == "W" {
			t.Errorf("couple fabrique (%q, W) a %d : le dead-state donne W au bot A", c.killer, c.timeMS)
		}
	}
	verifierLigne(t, p, 1003, "343 A"+BotSuffix, "W", OriginBotKiller)
}

// TestLeRecollageNePrendPasUneMortQueLeDeadStateDonneAUnBot — K tue (kill seul a 1000), W meurt (mort
// seule a 1001), aucun kill-event ; le seul dead-state de W dit « le bot A tue W ». Le kill de K ne
// se recolle pas sur la mort de W, et le temps 5 la publie tuee par A.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : retirer [resolveurDeCouples.mortLueDUnAutreTueur] du recollage.
func TestLeRecollageNePrendPasUneMortQueLeDeadStateDonneAUnBot(t *testing.T) {
	kf := &killFeed{events: []feedEvent{{timeMS: 1000, killer: "K"}, {timeMS: 1001, victim: "W"}},
		names: []string{"K", "W"}}
	p := passeDesBots(kf, rosterAvecW(), nil, []candidate{
		{chunk: 1, pidx: 2, bit: 10, ms: 1001, victim: 2, killer: 3, tag: 0xacd1cff4},
	})
	if len(kf.fab) != 0 {
		t.Errorf("couples fabriques %+v : le dead-state donne W au bot A", kf.fab)
	}
	verifierLigne(t, p, 1001, "343 A"+BotSuffix, "W", OriginBotKiller)
}

// TestLeRecollageNeVaPasChercherLaMortSuivante — deux kills de K (1000, 1001), deux morts sans kill
// (V 1002, W 1003) ; W meurt de sa propre source et personne ne revendique sa mort. Le premier kill
// prend V ; le second, dont la premiere mort voisine est deja prise, reste orphelin : la mort de W
// reste a personne, comme le film le dit.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : passer a la mort suivante (`continue`) au lieu de s arreter
// (`break`) sur une mort refusee dans [resolveurDeCouples.repliRecollageSurLeVoisin].
func TestLeRecollageNeVaPasChercherLaMortSuivante(t *testing.T) {
	kf := &killFeed{events: []feedEvent{{timeMS: 1000, killer: "K"}, {timeMS: 1001, killer: "K"},
		{timeMS: 1002, victim: "V"}, {timeMS: 1003, victim: "W"}}, names: []string{"K", "V", "W"}}
	p := passeDesBots(kf, rosterAvecW(), nil, []candidate{
		{chunk: 1, pidx: 1, bit: 10, ms: 1000, victim: 1, killer: 0, tag: 0xacd1cff4},
		{chunk: 1, pidx: 2, bit: 10, ms: 1002, victim: 2, killer: 2, tag: 0x00403594},
	})
	p.runUnclaimed()
	if len(kf.fab) != 1 || kf.fab[0].victim != "V" {
		t.Errorf("couples fabriques %+v : seul (K, V) a 1000 est attendu", kf.fab)
	}
	if len(p.unclaimed) != 1 || p.unclaimed[0].TimeMS != 1003 || p.unclaimed[0].Victim != "W" {
		t.Errorf("morts non revendiquees %+v : attendu celle de W a 1003", p.unclaimed)
	}
}
