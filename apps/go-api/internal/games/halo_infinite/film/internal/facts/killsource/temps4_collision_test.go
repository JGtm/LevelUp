package killsource

// temps4_collision_test.go — LE TEMPS 4 NE REECRIT JAMAIS UN INSTANT PUBLIE, ET `Covered` NE
// DEPASSE JAMAIS `RealPairs` (lot J7.4 du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, constat FK-4).
//
// # LE DEFAUT QUE CES TEMOINS FERMENT
//
// Le temps 4 (mort DE bot) ecrivait `byTime[instant]` sans regarder si l instant portait deja une
// ligne — contrairement au temps 3 et au temps 5. Et le retrait des couples fantomes du
// denominateur se calculait A PART ([decodeCtx.resolveBotDeaths] rejoue), sans regarder ce qui avait
// ete publie : un couple recolle compte fantome dont la mort de bot n etait PAS publiee (dead-state
// deja servi) gardait sa ligne humaine au numerateur. `Covered > RealPairs` devenait possible, et
// masquait une mort manquee ailleurs.
//
// # LE TEMOIN
//
// Deux kills de K sans mort en face (1000 et 1500), recolles sur les morts voisines de V et de W ;
// un dead-state lit K -> V a 1000, un autre K -> W a 1500 (le temps 1 les publie), et UN SEUL
// dead-state lit K -> bot a 1200, dans la fenetre des deux.
//
// # MUTATIONS QUI DOIVENT LES FAIRE ROUGIR
//
//   - retirer la garde `deja` de [pass.runBots] (l instant 1000 est reecrit en mort de bot) ;
//   - compter les fantomes sur les morts de bot TROUVEES au lieu des morts de bot PUBLIEES.

import "testing"

// passeAvecCollision : le decor du temoin, passe jusqu au temps 4.
func passeAvecCollision(t *testing.T) (*decodeCtx, *pass) {
	t.Helper()
	kf := &killFeed{
		events: []feedEvent{
			{timeMS: 1000, killer: "K"}, {timeMS: 1050, victim: "V"},
			{timeMS: 1500, killer: "K"}, {timeMS: 1550, victim: "W"},
		},
		names:  []string{"K", "V", "W"},
		xuidDe: map[string]uint64{},
	}
	bm := botMeta{NBots: 1, Bots: []bot{{Slot: 3, BotID: 7, Name: "343 Relais"}}}
	r := buildRoster(kf, bm, true, FilmTable{}, indexParMotif{})
	r.perm = []int{0, 1, 2, 3}
	kf.resoudreCouples(nil, r)
	if len(kf.fab) != 2 {
		t.Fatalf("temoin sans valeur : %d couples recolles, attendu 2", len(kf.fab))
	}
	c := &decodeCtx{roster: r, feed: kf, opts: DefaultOptions(), scanCands: []candidate{
		{chunk: 1, pidx: 1, bit: 10, ms: 1000, victim: 1, killer: 0, tag: 0xacd1cff4},
		{chunk: 1, pidx: 2, bit: 10, ms: 1500, victim: 2, killer: 0, tag: 0xacd1cff4},
		{chunk: 1, pidx: 3, bit: 10, ms: 1200, victim: 3, killer: 0, tag: 0xacd1cff4},
	}}
	p := &pass{ctx: c, byTime: map[int]Kill{}, botUsed: map[[3]int]bool{}, fantomes: map[int]bool{}}
	p.population(nil, c.scanCands, 0)
	p.runStrong()
	p.runBots()
	return c, p
}

// TestTemps4_NeReecritJamaisUnInstantPublie — la ligne du temps 1 reste, la collision se compte.
func TestTemps4_NeReecritJamaisUnInstantPublie(t *testing.T) {
	_, p := passeAvecCollision(t)
	for _, ms := range []int{1000, 1500} {
		if k, ok := p.byTime[ms]; !ok || k.Read.Origin != OriginCredit {
			t.Errorf("instant %d : origine %q, attendu %q — le temps 4 a reecrit une ligne publiee",
				ms, k.Read.Origin, OriginCredit)
		}
	}
	if p.collisionsBot != 2 {
		t.Errorf("collisions comptees = %d, attendu 2 — une mort de bot ecartee se compte", p.collisionsBot)
	}
	if st := p.stats(&walkResult{}); st.CollisionsDeMortDeBot != 2 {
		t.Errorf("collisions publiees = %d, attendu 2", st.CollisionsDeMortDeBot)
	}
}

// TestCouverture_CoveredNeDepassePasRealPairs — l invariant du denominateur, sur le meme temoin.
func TestCouverture_CoveredNeDepassePasRealPairs(t *testing.T) {
	c, p := passeAvecCollision(t)
	cov := c.coverage(p.kills(), p)
	if cov.Covered > cov.RealPairs {
		t.Fatalf("Covered = %d > RealPairs = %d (fantomes %d) : une ligne publiee est sortie du "+
			"denominateur, ce qui masque une mort manquee", cov.Covered, cov.RealPairs, cov.GhostPairs)
	}
	if cov.GhostPairs != 0 || cov.RealPairs != 2 || cov.Covered != 2 {
		t.Errorf("couverture = %d / %d, fantomes %d — attendu 2 / 2 et 0 : aucune mort de bot n est "+
			"publiee a un instant recolle", cov.Covered, cov.RealPairs, cov.GhostPairs)
	}
}

// TestTemps4_UnCoupleFabriqueNeMasquePasLaMortDeBot — REVUE ADVERSE DU LOT J7 (constat 1).
//
// K tue un BOT a 1000 (le kill-feed porte le kill, pas la mort) ; V se suicide a 1100 (la mort, pas
// de kill). Le recollage FABRIQUE (K, V) a 1000 ; le temps 3 y apparie le dead-state (V, V) sur la
// VICTIME SEULE et publie « V tue, credit K, source V » ; le temps 4 trouve la vraie mort de bot
// K -> bot a 1000. Une ligne que le temps 3 a posee sur un couple FABRIQUE ne confirme pas le tueur
// du couple : elle ne masque pas la mort de bot verifiee au meme instant.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : refuser toute reecriture au temps 4 (la garde de J7.4 seule).
func TestTemps4_UnCoupleFabriqueNeMasquePasLaMortDeBot(t *testing.T) {
	kf := &killFeed{
		events: []feedEvent{{timeMS: 1000, killer: "K"}, {timeMS: 1100, victim: "V"}},
		names:  []string{"K", "V"},
		xuidDe: map[string]uint64{},
	}
	bm := botMeta{NBots: 1, Bots: []bot{{Slot: 3, BotID: 7, Name: "343 Relais"}}}
	r := buildRoster(kf, bm, true, FilmTable{}, indexParMotif{})
	r.perm = []int{0, 1, 3, 2}
	kf.resoudreCouples(nil, r)
	if len(kf.fab) != 1 {
		t.Fatalf("temoin sans valeur : %d couple(s) recolle(s), attendu 1", len(kf.fab))
	}
	c := &decodeCtx{roster: r, feed: kf, opts: DefaultOptions(), scanCands: []candidate{
		{chunk: 1, pidx: 1, bit: 10, ms: 1000, victim: 3, killer: 0, tag: 0xacd1cff4}, // K -> bot
		{chunk: 1, pidx: 2, bit: 20, ms: 1100, victim: 1, killer: 1, tag: 0xacd1cff4}, // V -> V
	}}
	p := &pass{ctx: c, byTime: map[int]Kill{}, botUsed: map[[3]int]bool{}, fantomes: map[int]bool{}}
	p.population(nil, c.scanCands, 0)
	p.runStrong()
	p.runSelfSource()
	if k, ok := p.byTime[1000]; !ok || k.Read.Origin != OriginSelfSource {
		t.Fatalf("temoin sans valeur : le temps 3 n a pas publie sur le couple fabrique (%+v)", k)
	}
	p.runBots()

	k := p.byTime[1000]
	if k.Read.Origin != OriginBot || k.Victim != "343 Relais"+BotSuffix || k.Feed.Killer != "K" {
		t.Errorf("instant 1000 : victime %q, credit %q, origine %q — attendu la mort du BOT, "+
			"verifiee par le film ; « K a tue V » n a jamais eu lieu", k.Victim, k.Feed.Killer, k.Read.Origin)
	}
	if p.autoSurFabriqueRemplacees != 1 || p.collisionsBot != 0 {
		t.Errorf("remplacements %d, collisions %d — attendu 1 et 0", p.autoSurFabriqueRemplacees, p.collisionsBot)
	}
	cov := c.coverage(p.kills(), p)
	if cov.GhostPairs != 1 || cov.RealPairs != 0 || cov.Covered != 0 || cov.BotDeaths != 1 {
		t.Errorf("couverture : fantomes %d, reels %d, couverts %d, morts de bot %d — attendu 1, 0, 0, 1",
			cov.GhostPairs, cov.RealPairs, cov.Covered, cov.BotDeaths)
	}
}
