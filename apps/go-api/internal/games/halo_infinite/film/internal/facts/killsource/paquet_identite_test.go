package killsource

// paquet_identite_test.go — L IDENTITE DE PAQUET DECIDE L APPARIEMENT, LA FENETRE NE SERT QUE SUR
// SON SILENCE (lot 1.9.7).
//
// # POURQUOI CES TEMOINS SONT CONSTRUITS POUR DIVERGER
//
// Un test qui verifierait seulement « l instant apparie est le bon » resterait VERT si la lecture
// cessait de servir et que la fenetre rendait par hasard le meme instant — c est le cas sur 2 204
// des 2 207 appariements du corpus, ou identite et fenetre s accordent (mesure du lot, §5 du
// plan). Le temoin met donc DEUX morts du meme joueur a MOINS de 2,5 s l une de l autre, dont une
// SEULE porte l identite du kill-event : la fenetre rend la premiere, l identite rend l autre.
// C est la seule forme qui prouve que la LECTURE decide.

import "testing"

// paquetRoster : deux joueurs epingles, `A` en indice 1 et `B` en indice 2.
func paquetRoster() *roster {
	return &roster{
		names:   []string{"A", "B"},
		perm:    []int{-1, 0, 1},
		pin:     map[int]int{1: 0, 2: 1},
		seatPin: map[int]bool{1: true, 2: true},
		nPlay:   3,
	}
}

// paquetTemoin : le decor commun.
//
//	dead-state       A tue B, ECRIT dans le paquet (7, 42), a t=1500
//	feed t=1000      A tue B — ce que la FENETRE prend (premier de la fenetre, a 500 ms)
//	feed t=2000      A tue B — ce que le film ECRIT : son kill-event 85 est dans (7, 42)
//
// Les deux instants sont a 1 000 ms l un de l autre, donc tous les deux dans la demi-fenetre de
// 2,5 s : la fenetre ne peut pas les departager, l identite le peut.
func paquetTemoin() (*decodeCtx, candidate) {
	c := &decodeCtx{opts: DefaultOptions(), roster: paquetRoster(), feed: &killFeed{
		pairs: []feedEvent{
			{timeMS: 1000, killer: "A", victim: "B", victimXUID: 11},
			{timeMS: 2000, killer: "A", victim: "B", victimXUID: 22,
				paquet: paquetID{chunk: 7, pidx: 42, ok: true}},
		},
	}}
	return c, candidate{chunk: 7, pidx: 42, ms: 1500, bit: 64, victim: 2, killer: 1, cat: 1}
}

// TestLAppariementSuitLIdentiteDePaquet : LE TEMOIN PRINCIPAL. Le film ecrit le kill-event dans
// le paquet du dead-state ; c est cet instant-la qui est apparie, pas le plus proche en temps.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : supprimer la passe de LECTURE de
// [choisirParIdentitePuisFenetre] (la premiere boucle) — la fenetre reprend alors la main et rend
// t=1000. Jouee et restauree par nom au lot 1.9.7.
func TestLAppariementSuitLIdentiteDePaquet(t *testing.T) {
	c, cd := paquetTemoin()
	e, repli := c.matchExact(cd)
	if e == nil {
		t.Fatal("aucun instant apparie : les deux instants du temoin portent pourtant le couple (A, B)")
	}
	if e.timeMS != 2000 {
		t.Fatalf("instant apparie = %d, attendu 2000 — le film ECRIT le kill-event dans le paquet "+
			"(7, 42) du dead-state ; 1000 est ce que la FENETRE rendait", e.timeMS)
	}
	if repli {
		t.Error("le repli de la fenetre est signale alors que l identite a decide")
	}
	if e.victimXUID != 22 {
		t.Errorf("xuid publie = %d, attendu 22 : la ligne entiere suit l instant lu", e.victimXUID)
	}
}

// TestLaFenetreNeSertQueSurLeSilenceDeLIdentite : l ORDRE de D14 (b), par l autre bout. Sans
// identite en face, la fenetre reprend la main — et elle le DIT, pour que le compte de replis
// existe.
func TestLaFenetreNeSertQueSurLeSilenceDeLIdentite(t *testing.T) {
	c, cd := paquetTemoin()
	c.feed.pairs[1].paquet = paquetID{} // le film se tait : aucun kill-event 85 associe
	e, repli := c.matchExact(cd)
	if e == nil || e.timeMS != 1000 {
		t.Fatalf("instant apparie = %+v, attendu celui de la fenetre (1000) quand la lecture se tait", e)
	}
	if !repli {
		t.Error("le repli de la fenetre n est PAS signale : son compte serait un zero mensonger")
	}
}

// TestLIdentiteDeMauvaisPaquetNeDecidePas : l identite n apparie que le paquet EXACT. Un
// dead-state d un autre paquet ne se rattache pas a cet instant par la lecture — il ne peut le
// faire que par le repli, et le compteur le dit.
func TestLIdentiteDeMauvaisPaquetNeDecidePas(t *testing.T) {
	c, cd := paquetTemoin()
	cd.chunk, cd.pidx = 7, 43 // le paquet voisin
	e, repli := c.matchExact(cd)
	if e == nil || e.timeMS != 1000 || !repli {
		t.Fatalf("apparie = %+v (repli %v), attendu l instant de la fenetre (1000) par le REPLI : "+
			"le paquet (7, 43) n est pas celui que le film ecrit", e, repli)
	}
}

// TestLIdentiteDePaquetVoyageAvecLeCouple : LA LECTURE QUI REND TOUT LE RESTE POSSIBLE. Le
// kill-event 85 consomme par un couple du kill-feed lui laisse son identite de paquet, au lieu
// d etre jete (`pris[j] = true` et rien d autre, avant le lot).
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : retirer `res.kf.events[i].paquet = res.paquetDe(j)` de
// `consommerLesCouplesDuMemeInstant` — l identite n atteint plus `pairs`, et tout l appariement
// retombe silencieusement sur la fenetre. Jouee et restauree par nom au lot 1.9.7.
func TestLIdentiteDePaquetVoyageAvecLeCouple(t *testing.T) {
	kf := &killFeed{
		events: []feedEvent{{timeMS: 3000, killer: "A", victim: "B", victimXUID: 77}},
		xuidDe: map[string]uint64{"B": 77},
		names:  []string{"A", "B"},
	}
	rec := killEventRec{ms: 3000, chunk: 4, pidx: 19, chain: minChain,
		fields: killEventFields{killer: 1, victim: 2, assist: -1, end: 128}}
	kf.resoudreCouples([]killEventRec{rec}, paquetRoster())

	if len(kf.pairs) != 1 {
		t.Fatalf("couples = %+v, attendu le seul (A, B)", kf.pairs)
	}
	got := kf.pairs[0].paquet
	if !got.ok || got.chunk != 4 || got.pidx != 19 {
		t.Fatalf("identite de paquet du couple = %+v, attendue (4, 19) — celle du kill-event 85 "+
			"que ce couple a consomme", got)
	}
	if !got.memeQue(4, 19) || got.memeQue(4, 20) {
		t.Error("memeQue ne reconnait pas exactement le paquet ecrit")
	}
}

// TestUneIdentiteAbsenteNApparieRien : `ok` faux n est PAS un `(0, 0)` legitime. Sans cette
// distinction, tout dead-state du paquet (0, 0) s apparierait a tout instant muet.
func TestUneIdentiteAbsenteNApparieRien(t *testing.T) {
	if (paquetID{}).memeQue(0, 0) {
		t.Fatal("une identite ABSENTE apparie le paquet (0, 0) : le diagnostic « le film se tait » " +
			"est confondu avec une lecture")
	}
}

// TestLaMortDeBotSuitAussiLIdentite : le troisieme site converti. Le kill-event 85 nomme le bot
// en victime ET localise son paquet ; l appariement prend le dead-state de CE paquet.
func TestLaMortDeBotSuitAussiLIdentite(t *testing.T) {
	c := &decodeCtx{opts: DefaultOptions(), roster: &roster{
		names:   []string{"A", "Bob" + BotSuffix},
		perm:    []int{-1, 0, 5},
		pin:     map[int]int{1: 0, 5: 1},
		seatPin: map[int]bool{1: true},
		nPlay:   6,
	}}
	c.scanCands = []candidate{
		{chunk: 2, pidx: 11, ms: 5000, bit: 32, victim: 5, killer: 1},
		{chunk: 2, pidx: 90, ms: 5400, bit: 48, victim: 5, killer: 1},
	}
	m := botMatch{event: feedEvent{timeMS: 5000, killer: "A", victim: "Bob" + BotSuffix,
		paquet: paquetID{chunk: 2, pidx: 90, ok: true}}, victimeLue: 5}
	c.apparierMortDeBot(&m)

	if !m.found || m.cand.pidx != 90 {
		t.Fatalf("candidat retenu = %+v (trouve %v), attendu celui du paquet (2, 90) que le film "+
			"ecrit — (2, 11) est ce que la fenetre rendait (premier candidat)", m.cand, m.found)
	}
	if m.parLaFenetre {
		t.Error("le repli est signale alors que l identite a decide")
	}
}
