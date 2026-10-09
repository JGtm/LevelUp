package killsource

// sante_numerateur_test.go — LE NUMERATEUR DE SANTE NE COMPTE CHAQUE CANDIDAT QU UNE FOIS, ET NE
// COMPTE QUE DES CANDIDATS (lot J7.5 du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, constat FK-5).
//
// # LA REGLE ENFREINTE
//
// `grammar.KillSourceHealth` : « LES TROIS SE COMPTENT SUR `Candidates`, et c est ce qui autorise a
// en faire un ratio » — un seul denominateur, qui contient tout le numerateur (RE_LOG 7ter.73 (4e)).
//
// # LE DEFAUT
//
// Deux comptes la violaient. Le temps 3 comptait `UnexplainedSelf` sur TOUT `victime == tueur`, y
// compris un indice de bot que [pass.countUnexplainedBot] comptait aussi ; et ce dernier parcourait
// le SCAN ENTIER — redondants compris, qui sont hors de `Candidates` parce que la marche les lit
// deja, et sans les candidats de la MARCHE, qui y sont. Un meme dead-state lu par les deux voies
// sortait deux fois au numerateur pour une fois au denominateur ; un dead-state lu par la marche
// seule n y sortait jamais.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : laisser le temps 3 compter un candidat a indice de bot
// (cas « deux voies »), ou rendre `c.scanCands` a [pass.countUnexplainedBot] (cas « marche seule »).

import "testing"

// passeDeSante rejoue les temps de la passe sur une population donnee, sans film.
func passeDeSante(t *testing.T, walk, scan []candidate) *pass {
	t.Helper()
	kf := &killFeed{names: []string{"K", "V"}, xuidDe: map[string]uint64{}}
	bm := botMeta{NBots: 1, Bots: []bot{{Slot: 3, BotID: 7, Name: "343 Relais"}}}
	r := buildRoster(kf, bm, true, FilmTable{}, indexParMotif{})
	r.perm = []int{0, 1, 2, 3}
	c := &decodeCtx{roster: r, feed: kf, opts: DefaultOptions(), scanCands: scan}
	p := &pass{ctx: c, byTime: map[int]Kill{}, botUsed: map[[3]int]bool{}, fantomes: map[int]bool{}}
	p.population(walk, scan, 0)
	p.runStrong()
	p.runSelfSource()
	p.runBots()
	p.runBotKillers()
	p.countUnexplainedBot()
	return p
}

func TestSante_NumerateurSansDoubleComptage(t *testing.T) {
	// `victime == tueur` sur l indice du bot : ni couple ni mort de soi publiable.
	auto := candidate{chunk: 1, pidx: 1, bit: 10, ms: 1000, victim: 3, killer: 3, tag: 0xacd1cff4}
	// un tueur humain, une victime bot, et aucun kill au feed : non resolu.
	versBot := candidate{chunk: 1, pidx: 2, bit: 20, ms: 3000, victim: 3, killer: 0, tag: 0xacd1cff4}
	for _, cas := range []struct {
		nom        string
		walk, scan []candidate
	}{
		{"le meme dead-state lu par les deux voies", []candidate{auto}, []candidate{auto}},
		{"un dead-state lu par la marche seule", []candidate{versBot}, nil},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			p := passeDeSante(t, cas.walk, cas.scan)
			if len(p.all) != 1 {
				t.Fatalf("temoin sans valeur : population %d, attendu 1", len(p.all))
			}
			total := p.unexpPair + p.unexpSelf + p.unexpBot
			if total != 1 || p.unexpBot != 1 {
				t.Errorf("inexpliques = %d (couple %d, auto %d, bot %d) pour UN candidat consulte, "+
					"attendu 1 a indice de bot : le numerateur ne compte que des candidats, une fois",
					total, p.unexpPair, p.unexpSelf, p.unexpBot)
			}
		})
	}
}
