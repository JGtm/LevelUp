package killsource

// roster_releve_test.go — UN INDICE DE BOT N EST UN BOT QUE PENDANT SES DECLARATIONS
// (roster_releve.go).
//
// Le decor reprend le temoin du parc `0762ae7c` : le bot de l indice 8 est declare de 20 a 80 s du
// film, puis un humain (ici `Chivo`) reprend l indice ; le motif de xuid le lit a l indice 8. Son
// kill a 227 994 ms (fiche du film a 227 992 : tueur 8, victime 1) etait publie « tueur-bot » au nom
// du bot parti, a 1 ms de la ligne du credit : une mort servie deux fois. Les instants sont ici en
// millisecondes de killsource, l horloge du film en microsecondes (`tsBase` = 0).

import "testing"

// toutLeFilm : une declaration ouverte des l instant 0, jamais close — un bot present tout le film.
var toutLeFilm = []BotDeclaration{{FromUS: 0}}

// botsDuFilmEntier : BOT_METADATA declarant un bot sur chacun de `slots` pendant tout le film (une
// declaration ouverte des l instant 0) — le decor des tests qui ne portent pas sur les instants.
func botsDuFilmEntier(slots ...int) botMeta {
	var bm botMeta
	for _, s := range slots {
		bm.Bots = append(bm.Bots, bot{Slot: s, Name: "bot", declarations: toutLeFilm})
	}
	return bm
}

// rosterAReleve : K (indice 0) et M (indice 1) a la table du film, le bot `343 Berg` declare sur
// l indice 8 de 20 a 80 s, et le motif de xuid qui lit `Chivo` a l indice 8.
func rosterAReleve(t *testing.T) *roster {
	t.Helper()
	kf := &killFeed{names: []string{"K", "M", "Chivo"}}
	bm := botMeta{NBots: 1, Bots: []bot{{Slot: 8, BotID: 46, Name: "343 Berg",
		declarations: []BotDeclaration{{FromUS: 20_000_000, ToUS: 80_000_000}}}}}
	table := FilmTable{Seats: map[int]string{0: "K", 1: "M"}, Refusal: FilmTableRead}
	motif := indexParMotif{nomParIndex: map[int]string{0: "K", 1: "M", 8: "Chivo"}}
	r := buildRoster(kf, bm, true, table, motif)
	r.perm = []int{0, 1, -1, -1, -1, -1, -1, -1, 3}
	return r
}

// TestLeSuccesseurDUnBotTientSonIndiceApresLaDerniereDeclaration — les lectures datees de l indice 8.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : [roster.relever] qui rend toujours faux (la lecture du motif
// redevient une contradiction, l indice reste au bot pour tout le film), ou [roster.isBotIndexA]
// reduit a [roster.isBotIndex].
func TestLeSuccesseurDUnBotTientSonIndiceApresLaDerniereDeclaration(t *testing.T) {
	r := rosterAReleve(t)
	if r.table.MotifReleve != 1 || r.table.MotifContradict != 0 {
		t.Fatalf("releve %d / contradiction %d, attendu 1 / 0 : le motif nomme le successeur du bot",
			r.table.MotifReleve, r.table.MotifContradict)
	}
	for _, c := range []struct {
		ms        int
		bot, lu   bool
		nom, nomA string
	}{
		{ms: 10_000, bot: false, lu: false, nomA: "343 Berg" + BotSuffix},                            // avant la declaration : silence
		{ms: 30_000, bot: true, lu: true, nom: "343 Berg" + BotSuffix, nomA: "343 Berg" + BotSuffix}, // pendant
		{ms: 227_992, bot: false, lu: true, nom: "Chivo", nomA: "Chivo"},                             // apres : le successeur
	} {
		if got := r.isBotIndexA(8, c.ms); got != c.bot {
			t.Errorf("a %d ms : bot = %v, attendu %v", c.ms, got, c.bot)
		}
		nom, lu := r.nomEpingleA(8, c.ms)
		if lu != c.lu || nom != c.nom {
			t.Errorf("a %d ms : nom lu %q (%v), attendu %q (%v)", c.ms, nom, lu, c.nom, c.lu)
		}
		if got := r.nameOfA(8, c.ms); got != c.nomA {
			t.Errorf("a %d ms : nom %q, attendu %q", c.ms, got, c.nomA)
		}
	}
	_, libres := r.freeSlots()
	for _, p := range libres {
		if r.names[p] == "Chivo" {
			t.Errorf("Chivo est un nom libre : le successeur tient deja l indice 8")
		}
	}
}

// TestUnBotDeclareJusquALaFinNEstPasReleve — un bot encore declare au dernier paquet (`ToUS` = 0)
// garde son indice : la lecture du motif reste une contradiction comptee.
func TestUnBotDeclareJusquALaFinNEstPasReleve(t *testing.T) {
	kf := &killFeed{names: []string{"K", "M", "Chivo"}}
	bm := botMeta{NBots: 1, Bots: []bot{{Slot: 8, BotID: 46, Name: "343 Berg",
		declarations: []BotDeclaration{{FromUS: 20_000_000}}}}}
	table := FilmTable{Seats: map[int]string{0: "K", 1: "M"}, Refusal: FilmTableRead}
	r := buildRoster(kf, bm, true, table, indexParMotif{nomParIndex: map[int]string{8: "Chivo"}})
	if r.table.MotifReleve != 0 || r.table.MotifContradict != 1 {
		t.Errorf("releve %d / contradiction %d, attendu 0 / 1", r.table.MotifReleve, r.table.MotifContradict)
	}
	if !r.isBotIndexA(8, 900_000) {
		t.Errorf("le bot declare jusqu a la fin ne tient plus son indice a 900 s")
	}
}

// TestLeSuccesseurSigneSonKillAuLieuDuBotParti — le temoin `0762ae7c` de bout en bout : la fiche du
// film (tueur 8, victime 1) nomme Chivo, le couple (Chivo, M) est lu, et la mort de M n est plus
// publiee « tueur-bot » au nom du bot parti.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : [roster.relever] qui rend toujours faux.
func TestLeSuccesseurSigneSonKillAuLieuDuBotParti(t *testing.T) {
	kf := &killFeed{events: []feedEvent{{timeMS: 227_994, killer: "Chivo"},
		{timeMS: 227_995, victim: "M", victimXUID: 2}}, names: []string{"K", "M", "Chivo"}}
	r := rosterAReleve(t)
	p := passeDesBots(kf, r, []killEventRec{coupleRec(227_992, 8, 1)},
		[]candidate{{chunk: 3, pidx: 7, bit: 10, ms: 227_992, victim: 1, killer: 8, tag: 0xacd1cff4}})
	verifierLigne(t, p, 227_994, "Chivo", "M", OriginCredit)
	for ms, k := range p.byTime {
		if k.Read.Origin == OriginBotKiller {
			t.Errorf("ligne tueur-bot a %d ms (tueur %q) : l indice 8 n est plus au bot", ms, k.Feed.Killer)
		}
	}
}

// TestAucuneLigneTueurBotHorsDesDeclarations — le bot de l indice 8 est parti a 80 s et personne
// n est lu a sa place : une mort que le dead-state donne a l indice 8 apres son depart ne se publie
// pas « tueur-bot ».
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : [roster.isBotIndexA] reduit a [roster.isBotIndex].
func TestAucuneLigneTueurBotHorsDesDeclarations(t *testing.T) {
	kf := &killFeed{events: []feedEvent{{timeMS: 227_995, victim: "M", victimXUID: 2}},
		names: []string{"K", "M"}}
	bm := botMeta{NBots: 1, Bots: []bot{{Slot: 8, BotID: 46, Name: "343 Berg",
		declarations: []BotDeclaration{{FromUS: 20_000_000, ToUS: 80_000_000}}}}}
	table := FilmTable{Seats: map[int]string{0: "K", 1: "M"}, Refusal: FilmTableRead}
	r := buildRoster(kf, bm, true, table, indexParMotif{})
	r.perm = []int{0, 1, -1, -1, -1, -1, -1, -1, 2}
	cand := candidate{chunk: 3, pidx: 7, bit: 10, ms: 227_992, victim: 1, killer: 8, tag: 0xacd1cff4}
	p := passeDesBots(kf, r, nil, []candidate{cand})
	if k, ok := p.byTime[227_995]; ok && k.Read.Origin == OriginBotKiller {
		t.Errorf("ligne tueur-bot publiee a 227 995 ms (tueur %q) hors de toute declaration", k.Feed.Killer)
	}
	cand.ms = 50_000 // pendant la declaration, la mort infligee par le bot se publie
	kf2 := &killFeed{events: []feedEvent{{timeMS: 50_001, victim: "M", victimXUID: 2}}, names: []string{"K", "M"}}
	p2 := passeDesBots(kf2, r, nil, []candidate{cand})
	verifierLigne(t, p2, 50_001, "343 Berg"+BotSuffix, "M", OriginBotKiller)
}

// TestLaReleveGardeAssezDeNomsLibres — le successeur tient un nom sans tenir d indice a lui : le
// roster doit garder au moins autant de noms libres que d indices libres, sans quoi la bijection
// (ses relances melangent les noms libres sur les indices libres) sort de ses tableaux.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : remplir `names` jusqu a `nPlay` seulement dans [buildRoster]
// (panique de [solveBijection], temoin du parc `0ae53571`).
func TestLaReleveGardeAssezDeNomsLibres(t *testing.T) {
	r := rosterAReleve(t)
	libres, noms := r.freeSlots()
	if len(noms) < len(libres) {
		t.Fatalf("%d noms libres pour %d indices libres", len(noms), len(libres))
	}
	perm, _ := solveBijection(r, nil, nil, 3)
	if len(perm) != r.nPlay {
		t.Errorf("bijection de %d indices, attendu %d", len(perm), r.nPlay)
	}
}
