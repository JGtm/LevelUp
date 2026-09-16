package persist

// kill_events_merge_pairing_test.go — LES SONDES DE LA REVUE ADVERSARIALE DU LOT 2.9.
//
// Elles portent toutes sur `apparier` (`kill_events_merge_pairing.go`) et sur UN seul defaut, que
// les deux relecteurs aveugles de la ronde 1 (2026-09-16) ont trouve independamment : le repli
// « par elimination » appariait les deux residus d un instant SANS aucune identite commune. Il
// suffisait d un instant a deux morts de credit pour que la fusion, ou bien refuse LE FILM ENTIER
// (la regression meme que le lot ferme), ou bien ecrive en base l arme d une mort sur une autre
// SANS RIEN SIGNALER — `AmbiguousInstants` tombait a 0.
//
// Chaque test epingle des VALEURS (armes distinctes par ligne de film) plutot qu un simple
// « ca ne plante pas » : c est l arme qui dit QUELLE mesure a atterri sur QUELLE mort. Les
// mutations qui doivent les faire rougir sont nommees au-dessus de chacun.

import "testing"

// mortCreditDe : une mort de credit dont on choisit la victime. Les sondes de la revue
// adversariale ont toutes besoin de DEUX victimes distinctes au meme instant.
func mortCreditDe(t int, victime, xuid string) KillEventInsert {
	m := mortCredit(t)
	m.VictimGamertag, m.VictimXUID = victime, xuid
	return m
}

// mortFilmDe : une ligne de film dont on choisit la victime et l arme. L arme distingue les
// lignes entre elles : c est elle qui dit QUELLE mesure a atterri sur QUELLE mort.
func mortFilmDe(t int, xuid string, tag uint32) KillEventInsert {
	m := mortFilm(t)
	m.VictimXUID, m.SourceTag = xuid, tag
	return m
}

// TestFusionRefuseUnDoublonDeFilmSurUnInstantADeuxMorts — SONDE (A) DE LA REVUE ADVERSARIALE.
//
// credit [A, B] + film [A, A] : la premiere ligne de film s apparie a A ; la seconde n est ni
// appariee (A est deja pris) ni « autre mort » (sa victime EST celle d une mort de credit de
// l instant). Le repli par ELIMINATION l appariait alors a B — victime divergente, ERREUR RENDUE,
// LE FILM ENTIER REFUSE : exactement la regression que le lot 2.9 existe pour fermer.
//
// La seconde ligne doit etre REFUSEE et l instant compte.
func TestFusionRefuseUnDoublonDeFilmSurUnInstantADeuxMorts(t *testing.T) {
	const tagPremiere, tagSeconde = uint32(0x11111111), uint32(0x22222222)
	a := mortCreditDe(1000, "A", "xuid(a)")
	b := mortCreditDe(1000, "B", "xuid(b)")

	out, st, err := MergeCreditAndFilm(batchCredit(a, b),
		batchFilm(mortFilmDe(1000, "xuid(a)", tagPremiere), mortFilmDe(1000, "xuid(a)", tagSeconde)))
	if err != nil {
		t.Fatalf("MergeCreditAndFilm: %v\nun doublon de film fait de nouveau tomber LE FILM "+
			"ENTIER — le repli a apparie la seconde ligne a la mort de B", err)
	}
	if st.Enriched != 1 || st.AmbiguousInstants != 1 || st.Orphans != 0 {
		t.Fatalf("%d enrichies / %d instants ambigus / %d orphelins, attendu 1 / 1 / 0",
			st.Enriched, st.AmbiguousInstants, st.Orphans)
	}
	if len(out.Deaths) != 2 {
		t.Fatalf("%d morts publiees, attendu 2 — une ligne refusee ne s ajoute pas", len(out.Deaths))
	}
	if out.Deaths[0].SourceTag != tagPremiere {
		t.Errorf("la mort de A porte %#x, attendu %#x", out.Deaths[0].SourceTag, tagPremiere)
	}
	if out.Deaths[1].SourceTag != 0 {
		t.Errorf("la mort de B porte l arme %#x — elle a recu la mesure d une mort qui n est pas "+
			"la sienne, ecrite en base et servie par `_latest` sans que rien ne le signale",
			out.Deaths[1].SourceTag)
	}
}

// TestFusionRefuseUneLigneSansVictimeSurUnInstantADeuxMorts — SONDE (B) DE LA REVUE.
//
// credit [A, B] + film [A, ""] : le repli par elimination donnait a B l arme, les parts ET
// l assistant de la ligne sans victime — une mesure ecrite en base sur une mort qui n est pas la
// sienne, avec `AmbiguousInstants` a 0. Le silence etait le vrai defaut.
func TestFusionRefuseUneLigneSansVictimeSurUnInstantADeuxMorts(t *testing.T) {
	a := mortCreditDe(1000, "A", "xuid(a)")
	b := mortCreditDe(1000, "B", "xuid(b)")
	sansVictime := mortFilm(1000) // le film n a pas resolu la victime : 1 252 lignes du parc

	out, st, err := MergeCreditAndFilm(batchCredit(a, b),
		batchFilm(mortFilmDe(1000, "xuid(a)", 0x11111111), sansVictime))
	if err != nil {
		t.Fatalf("MergeCreditAndFilm: %v", err)
	}
	if st.Enriched != 1 || st.AmbiguousInstants != 1 || st.Orphans != 0 {
		t.Fatalf("%d enrichies / %d instants ambigus / %d orphelins, attendu 1 / 1 / 0 — seule la "+
			"ligne de A doit enrichir", st.Enriched, st.AmbiguousInstants, st.Orphans)
	}
	if len(out.Deaths) != 2 {
		t.Fatalf("%d morts publiees, attendu 2", len(out.Deaths))
	}
	if out.Deaths[1].SourceTag != 0 || out.Deaths[1].AssistKnown || out.Deaths[1].Diverges {
		t.Errorf("la mort de B porte tag=%#x assist_known=%v diverges=%v — elle a recu l arme, "+
			"l assistant et la divergence d une AUTRE mort", out.Deaths[1].SourceTag,
			out.Deaths[1].AssistKnown, out.Deaths[1].Diverges)
	}
}

// TestFusionRefuseDeuxMortsDeCreditDeLaMemeVictime — SONDE (C2) DU RELECTEUR L6.
//
// Deux morts de credit de la MEME victime a la MEME milliseconde : la victime ne distingue plus
// rien, donc rien ne dit laquelle des deux la ligne de film mesure. Choisir serait tirer au sort
// une mort sur deux.
//
// MUTATION QUI DOIT ROUGIR : dans `seuleMortDeLaVictime`, `return -1` (plusieurs candidates)
// devient `return trouve` (la premiere).
func TestFusionRefuseDeuxMortsDeCreditDeLaMemeVictime(t *testing.T) {
	memeVictime := mortCreditDe(1000, "A", "xuid(a)")

	out, st, err := MergeCreditAndFilm(batchCredit(memeVictime, memeVictime),
		batchFilm(mortFilmDe(1000, "xuid(a)", 0x11111111)))
	if err != nil {
		t.Fatalf("MergeCreditAndFilm: %v", err)
	}
	if st.Enriched != 0 || st.AmbiguousInstants != 1 || st.Orphans != 0 {
		t.Errorf("%d enrichies / %d instants ambigus / %d orphelins, attendu 0 / 1 / 0 — l arme a "+
			"ete attribuee au hasard a l une des deux morts",
			st.Enriched, st.AmbiguousInstants, st.Orphans)
	}
	if len(out.Deaths) != 2 || out.Deaths[0].SourceTag != 0 || out.Deaths[1].SourceTag != 0 {
		t.Errorf("%d morts, armes %#x / %#x, attendu 2 morts sans arme",
			len(out.Deaths), out.Deaths[0].SourceTag, out.Deaths[1].SourceTag)
	}
}

// TestFusionNApparieJamaisSurDeuxVictimesVides — SONDE (C3) DU RELECTEUR L6.
//
// Deux victimes vides ne sont pas « la meme victime » : ce sont deux ABSENCES. Sur un instant qui
// porte deja une autre mort de credit, apparier sur cette absence serait apparier sur rien.
//
// MUTATION QUI DOIT ROUGIR : dans `seuleMortDeLaVictime`, `if victime == ""` devient `if false`.
func TestFusionNApparieJamaisSurDeuxVictimesVides(t *testing.T) {
	sansXUID := mortCreditDe(1000, "Inconnue", "")
	autre := mortCreditDe(1000, "A", "xuid(a)")

	_, st, err := MergeCreditAndFilm(batchCredit(sansXUID, autre), batchFilm(mortFilm(1000)))
	if err != nil {
		t.Fatalf("MergeCreditAndFilm: %v", err)
	}
	if st.Enriched != 0 || st.AmbiguousInstants != 1 {
		t.Errorf("%d enrichies / %d instants ambigus, attendu 0 / 1 — deux victimes VIDES ont ete "+
			"prises pour la meme victime", st.Enriched, st.AmbiguousInstants)
	}
}

// TestFusionNEnrichitQuUneFoisSurDeuxLignesDeFilmDeLaMemeVictime — SONDE (C4) DU RELECTEUR L6.
//
// Une mort de credit, deux lignes de film de la meme victime : la PREMIERE enrichit, la seconde
// est refusee. Sans le garde `prisB[y]`, la seconde ecraserait la premiere — silencieusement, et
// l instant ne serait meme plus compte.
//
// MUTATION QUI DOIT ROUGIR : retirer `prisB[y] ||` de `seuleMortDeLaVictime`.
func TestFusionNEnrichitQuUneFoisSurDeuxLignesDeFilmDeLaMemeVictime(t *testing.T) {
	const tagPremiere, tagSeconde = uint32(0x11111111), uint32(0x22222222)

	out, st, err := MergeCreditAndFilm(batchCredit(mortCreditDe(1000, "A", "xuid(a)")),
		batchFilm(mortFilmDe(1000, "xuid(a)", tagPremiere), mortFilmDe(1000, "xuid(a)", tagSeconde)))
	if err != nil {
		t.Fatalf("MergeCreditAndFilm: %v", err)
	}
	if st.Enriched != 1 || st.AmbiguousInstants != 1 || st.Orphans != 0 {
		t.Fatalf("%d enrichies / %d instants ambigus / %d orphelins, attendu 1 / 1 / 0",
			st.Enriched, st.AmbiguousInstants, st.Orphans)
	}
	if len(out.Deaths) != 1 || out.Deaths[0].SourceTag != tagPremiere {
		t.Errorf("%d morts, arme %#x, attendu 1 mort a %#x — la seconde ligne a ecrase la "+
			"premiere", len(out.Deaths), out.Deaths[0].SourceTag, tagPremiere)
	}
}

// TestFusionNeCompteJamaisUnOrphelinDInstantLibreCommePartage — SONDE (C5) DU RELECTEUR L6.
//
// `OrphansSharedInstant` ne doit compter QUE les orphelines dont l instant porte une mort de
// credit d une autre victime. Un orphelin d instant LIBRE — les morts de bot, 968 sur 980 — n a
// rien a voir avec le defaut du lot 2.9 : les confondre noierait la population a surveiller.
//
// MUTATION QUI DOIT ROUGIR : `if v == filmOrphelinInstantPartage` devient `if true`.
func TestFusionNeCompteJamaisUnOrphelinDInstantLibreCommePartage(t *testing.T) {
	bot := mortFilm(5000)
	humain := mortFilmDe(6000, "xuid(9)", 0x33333333)
	humain.FeedKillerXUID = "xuid(8)"

	_, st, err := MergeCreditAndFilm(batchCredit(mortCredit(1000)), batchFilm(bot, humain))
	if err != nil {
		t.Fatalf("MergeCreditAndFilm: %v", err)
	}
	if st.Orphans != 2 || st.OrphansSharedInstant != 0 {
		t.Errorf("%d orphelins dont %d a instant partage, attendu 2 / 0 — un orphelin d instant "+
			"LIBRE est compte comme la population du lot 2.9, qui perd alors son sens",
			st.Orphans, st.OrphansSharedInstant)
	}
}
