package killsource

// roster_espace_humains_test.go — L ESPACE DES HUMAINS VIENT DU FILM, JAMAIS DU KILL-FEED (lot J7.2
// du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, constat FK-1).
//
// # LE DEFAUT QUE CES TEMOINS FERMENT
//
// [buildRoster] bornait l espace des humains au NOMBRE DE NOMS du kill-feed. Un remplacant humain
// ajoute un nom au feed sans ajouter de place au match (les places sont finies : un partant libere
// la sienne) ; il poussait donc la borne au-dela du slot du bot de relais, qui cessait d etre
// epingle — sans journal ni compteur. Ses morts et celles qu il inflige se perdaient, ses
// assistances sortaient en `?N`, et un second indice libre suffisait a fermer la publication ligne
// par ligne de tout le match. `b1ad85eb` a exactement cette structure (huit sieges 0..7, un bot au
// slot 8, un remplacant a l indice 10) ; le BTB mesure deja 2 bots epingles sur 8 declares.
//
// # MUTATIONS QUI DOIVENT LES FAIRE ROUGIR
//
//   - borner [roster.pinBots] par `r.nHumans` (le nombre de noms du feed) au lieu de
//     [borneDesHumainsDuFilm] ;
//   - retirer `BotsNonEpingles` de [decodeCtx.coverage].

import "testing"

// feedAvecRemplacant : huit joueurs de depart et le remplacant, tous nommes par le kill-feed.
func feedAvecRemplacant() *killFeed {
	return &killFeed{names: []string{"A", "B", "C", "D", "E", "F", "G", "H", "Remplacant"}}
}

// sieges0a7 : la table de `chunk_00`, ecrite a l ouverture : huit sieges, 0..7.
func sieges0a7() FilmTable {
	return FilmTable{Build: "b", Seats: map[int]string{
		0: "A", 1: "B", 2: "C", 3: "D", 4: "E", 5: "F", 6: "G", 7: "H",
	}}
}

// TestRoster_RemplacantHumainNeDesepinglePasLeBotDeRelais — LE TEMOIN DE FK-1, a la structure de
// `b1ad85eb`.
func TestRoster_RemplacantHumainNeDesepinglePasLeBotDeRelais(t *testing.T) {
	bm := botMeta{NBots: 1, Bots: []bot{{Slot: 8, BotID: 7, Name: "343 Relais"}}}
	motif := indexParMotif{nomParIndex: map[int]string{10: "Remplacant"}, lectures: 27}
	r := buildRoster(feedAvecRemplacant(), bm, true, sieges0a7(), motif)

	if len(r.unpinned) != 0 {
		t.Fatalf("bots non epingles = %+v : le remplacant du kill-feed a desepingle le bot de relais",
			r.unpinned)
	}
	if !r.isBotIndex(8) {
		t.Error("l indice 8 n est pas un bot : ses morts et celles qu il inflige seraient perdues")
	}
	if nom, ok := r.nomEpingle(8); !ok || nom != "343 Relais"+BotSuffix {
		t.Errorf("indice 8 epingle = %q (%v), attendu le bot de relais", nom, ok)
	}
	if nom, ok := r.nomEpingle(10); !ok || nom != "Remplacant" {
		t.Errorf("indice 10 epingle = %q (%v), attendu le remplacant", nom, ok)
	}
	// LA BORNE N EST PAS LE NOMBRE DE NOMS DU FEED : c est le nombre de places que la table donne.
	if r.borneHumains != 8 {
		t.Errorf("borne des humains = %d, attendu 8 (les sieges 0..7 de la table)", r.borneHumains)
	}
}

// TestRoster_BotNonEpingleEstCompte : un bot dont le slot tombe sur un siege HUMAIN de la table
// contredit la table ; il n est pas epingle, et la perte se COMPTE dans la couverture publiee.
func TestRoster_BotNonEpingleEstCompte(t *testing.T) {
	bm := botMeta{NBots: 2, Bots: []bot{
		{Slot: 3, BotID: 7, Name: "343 Contradictoire"},
		{Slot: 9, BotID: 16, Name: "343 Relais"},
	}}
	r := buildRoster(feedAvecRemplacant(), bm, true, sieges0a7(), indexParMotif{})
	if len(r.unpinned) != 1 || r.unpinned[0].Slot != 3 {
		t.Fatalf("bots non epingles = %+v, attendu le seul bot du slot 3", r.unpinned)
	}
	if !r.isBotIndex(9) {
		t.Error("le bot du slot 9, hors des sieges humains, doit rester epingle")
	}
	c := &decodeCtx{roster: r, feed: &killFeed{}, opts: DefaultOptions()}
	cov := c.coverage(nil, &pass{ctx: c})
	if cov.BotsNonEpingles != 1 {
		t.Errorf("couverture : bots non epingles = %d, attendu 1 — la perte doit etre publiee",
			cov.BotsNonEpingles)
	}
}

// TestRoster_SansTableAucunBotNEstDesepingle : sans table lue, le film ne donne aucune borne des
// humains — aucune contradiction ne peut etre etablie, et le nombre de noms du kill-feed n en
// tient JAMAIS lieu. BOT_METADATA, la lecture la plus eprouvee, epingle ses bots.
func TestRoster_SansTableAucunBotNEstDesepingle(t *testing.T) {
	bm := botMeta{NBots: 1, Bots: []bot{{Slot: 2, BotID: 7, Name: "343 Relais"}}}
	r := buildRoster(feedAvecRemplacant(), bm, true, FilmTable{Refusal: FilmTableNoSection}, indexParMotif{})
	if len(r.unpinned) != 0 || !r.isBotIndex(2) {
		t.Errorf("sans table : bots non epingles %+v, indice 2 bot=%v — attendu le bot epingle",
			r.unpinned, r.isBotIndex(2))
	}
}

// TestRoster_BotSurSiegeVacantIntercaleResteEpingle — REVUE ADVERSE DU LOT J7 (constat 3).
//
// La table a un siege VACANT intercale (etat mesure sur 13 films du cache, dont `111fa685`,
// `a521164d`, `11de8353`) : le siege 3 est libre, et BOT_METADATA y declare un bot. Le slot tombe
// sous le nombre de sieges, mais aucun HUMAIN ne tient ce siege : rien ne contredit le bot.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : retirer la condition « siege nomme » de [roster.pinBots].
func TestRoster_BotSurSiegeVacantIntercaleResteEpingle(t *testing.T) {
	kf := &killFeed{names: []string{"A", "B", "C", "E", "F", "G", "H", "I"}}
	table := FilmTable{Build: "b", InterleavedVacant: true, Seats: map[int]string{
		0: "A", 1: "B", 2: "C", 4: "E", 5: "F", 6: "G", 7: "H", 8: "I"}}
	bm := botMeta{NBots: 1, Bots: []bot{{Slot: 3, BotID: 7, Name: "343 Relais"}}}
	r := buildRoster(kf, bm, true, table, indexParMotif{})
	if _, nomme := table.Seats[3]; nomme || r.borneHumains <= 3 {
		t.Fatalf("temoin sans valeur : siege 3 nomme=%v, borne %d", nomme, r.borneHumains)
	}
	if len(r.unpinned) != 0 || !r.isBotIndex(3) {
		t.Errorf("bot du siege vacant 3 : non epingles %+v, indice 3 bot=%v — attendu epingle : aucun "+
			"humain ne tient ce siege", r.unpinned, r.isBotIndex(3))
	}
	if r.table.BotConflict != 0 {
		t.Errorf("BotConflict = %d, attendu 0 : le siege est vacant", r.table.BotConflict)
	}
}
