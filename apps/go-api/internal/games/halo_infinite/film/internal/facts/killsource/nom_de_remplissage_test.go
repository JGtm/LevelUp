package killsource

// nom_de_remplissage_test.go — UN NOM DE REMPLISSAGE N EST JAMAIS PUBLIE (lot J7.1 du
// PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, constat FK-2, decision DT-6).
//
// # LE DEFAUT QUE CES TEMOINS FERMENT
//
// [buildRoster] fabrique des noms `?N` pour rendre le probleme d affectation carre, et
// [roster.nameOf] rend `?` hors bijection. Le paquet traitait les deux comme un SILENCE dans
// [roster.originOf] (prefixe `?`), mais [decodeCtx.fillAssist] ne rejetait que la chaine `?`
// exacte : un indice d assistant pose par l inference sur un nom de remplissage publiait `?10`
// comme assistant NOMME (`b1ad85eb`, reel), ecrit en base par le collecteur et affiche sur la page
// du match. Deux regles contradictoires dans le meme paquet ; il n en reste qu une,
// [estNomDeRemplissage].
//
// # MUTATIONS QUI DOIVENT LES FAIRE ROUGIR
//
//   - remettre `c.roster.nameOf(f.assist) == "?"` dans [decodeCtx.fillAssist] ;
//   - faire rendre `nom == "?"` a [estNomDeRemplissage] (le `?N` redevient un nom) ;
//   - retirer la garde de [pass.nomPubliable] dans le temps 4.

import "testing"

// rosterAvecRemplissage : un roster construit par [buildRoster] ou l inference a pose un nom de
// remplissage sur un indice. Deux humains au kill-feed, un bot au slot 3 : l indice 2 n a aucun
// nom, [buildRoster] fabrique `?3` (la position du nom dans `names`), et la bijection le range
// sur l indice 2 — exactement la forme de `b1ad85eb`.
func rosterAvecRemplissage(t *testing.T) *roster {
	t.Helper()
	kf := &killFeed{names: []string{"TUEUR", "VICTIME"}}
	bm := botMeta{NBots: 1, Bots: []bot{{Slot: 3, BotID: 7, Name: "343 Relais"}}}
	r := buildRoster(kf, bm, true, FilmTable{}, indexParMotif{})
	// names = [TUEUR VICTIME "343 Relais [bot]" ?3] ; indice 3 epingle sur le bot (position 2).
	r.perm = []int{0, 1, 3, 2}
	if got := r.nameOf(2); got != "?3" {
		t.Fatalf("temoin sans valeur : l indice 2 porte %q, attendu le nom de remplissage ?3", got)
	}
	return r
}

// TestAssist_NomDeRemplissageNestJamaisPublie — LE TEMOIN DE FK-2.
func TestAssist_NomDeRemplissageNestJamaisPublie(t *testing.T) {
	c := &decodeCtx{roster: rosterAvecRemplissage(t), opts: DefaultOptions()}
	kills := []Kill{{TimeMS: 10_000, Victim: "VICTIME", Feed: FeedTruth{Killer: "TUEUR", Present: true}}}
	s := &assistScan{recs: []killEventRec{
		{ms: 10_000, fields: killEventFields{victim: 1, killer: 0, assist: 2, killerPct: 70, assistPct: 30}},
	}}
	st := c.attachAssists(kills, s)

	k := kills[0]
	if k.Assist.Name != "" {
		t.Errorf("assistant publie = %q : un nom de REMPLISSAGE est publie comme assistant nomme", k.Assist.Name)
	}
	if k.Assist.Rejected != AssistRejectRoster {
		t.Errorf("rejet = %q, attendu %q : l indice ne designe aucun joueur du roster retenu",
			k.Assist.Rejected, AssistRejectRoster)
	}
	if st.Named != 0 || st.RejectedRoster != 1 {
		t.Errorf("compteurs : nommes %d, hors-roster %d — attendu 0, 1", st.Named, st.RejectedRoster)
	}
	// La part de l assistant est MESUREE (le champ etait present) : c est son porteur qu on refuse
	// de nommer, et le collecteur n ecrit pas de part sans assistant nomme (cf. [fillAssist]).
	if !k.AssistDamage.Known {
		t.Error("la part de l assistant doit rester mesuree : le champ etait present")
	}
}

// TestOrigineDUnNomDeRemplissageEstLeSilence : la provenance publiee dit « silence » pour un
// indice que seul le remplissage nomme — le meme predicat que la publication.
func TestOrigineDUnNomDeRemplissageEstLeSilence(t *testing.T) {
	r := rosterAvecRemplissage(t)
	if got := r.originOf(2); got != OriginNone {
		t.Errorf("provenance de l indice 2 = %q, attendu %q", got, OriginNone)
	}
	if got := r.originOf(3); got != OriginBotMeta {
		t.Errorf("provenance de l indice 3 = %q, attendu %q", got, OriginBotMeta)
	}
}

// TestEstNomDeRemplissage : la liste FERMEE de ce que le predicat reconnait. Un gamertag Xbox ne
// porte jamais `?` ; le nom de repli `xuid:` et le suffixe de bot ne sont PAS du remplissage.
func TestEstNomDeRemplissage(t *testing.T) {
	for nom, attendu := range map[string]bool{
		"?": true, "?3": true, "?10": true,
		"Zeus Herd": false, XUIDNamePrefix + "2533274": false, "343 Relais" + BotSuffix: false, "": false,
	} {
		if got := estNomDeRemplissage(nom); got != attendu {
			t.Errorf("estNomDeRemplissage(%q) = %v, attendu %v", nom, got, attendu)
		}
	}
}

// TestMortDeBot_NomDeRemplissageNestJamaisPublie : le temps 4 publie la victime depuis le roster
// de replication, pas depuis le kill-feed. Un nom qui tomberait dans l espace de remplissage (un
// nom BOT_METADATA illisible commencant par `?`) n est pas publie, et le refus se compte.
func TestMortDeBot_NomDeRemplissageNestJamaisPublie(t *testing.T) {
	kf := &killFeed{names: []string{"TUEUR", "VICTIME"}}
	bm := botMeta{NBots: 1, Bots: []bot{{Slot: 3, BotID: 7, Name: "?illisible"}}}
	r := buildRoster(kf, bm, true, FilmTable{}, indexParMotif{})
	r.perm = []int{0, 1, 3, 2}
	kf.orphK = []feedEvent{{timeMS: 5_000, killer: "TUEUR"}}
	// Le temps 5, symetrique : une mort du feed SANS kill, infligee par le meme indice de bot.
	kf.orphD = []feedEvent{{timeMS: 6_000, victim: "VICTIME"}}
	c := &decodeCtx{roster: r, feed: kf, opts: DefaultOptions(),
		scanCands: []candidate{{chunk: 1, pidx: 4, bit: 40, ms: 5_000, victim: 3, killer: 0, tag: 0xacd1cff4}}}
	p := &pass{ctx: c, byTime: map[int]Kill{}, botUsed: map[[3]int]bool{},
		all: []sourcedCandidate{{candidate{chunk: 1, pidx: 5, bit: 50, ms: 6_000, victim: 1, killer: 3,
			tag: 0xacd1cff4}, PathWalk}}}
	p.runBots()
	p.runBotKillers()

	for _, k := range p.kills() {
		if estNomDeRemplissage(k.Victim) || estNomDeRemplissage(k.Feed.Killer) {
			t.Errorf("ligne publiee avec un nom de remplissage : victime %q, tueur %q", k.Victim, k.Feed.Killer)
		}
	}
	if p.nomsDeRemplissage != 2 {
		t.Errorf("refus comptes = %d, attendu 2 (temps 4 et 5) — un refus qui ne se compte pas est "+
			"une perte silencieuse", p.nomsDeRemplissage)
	}
	if p.botStats.Matched != 1 || p.botKillerStats.Matched != 1 {
		t.Fatalf("temoin sans valeur : apparies temps 4 = %d, temps 5 = %d, attendu 1 et 1",
			p.botStats.Matched, p.botKillerStats.Matched)
	}
}
