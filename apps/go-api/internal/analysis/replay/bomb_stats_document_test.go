package replay

import "testing"

// bomb_stats_document_test.go — LE CÂBLAGE de `bomb_carriers_killed`, et sa seule vraie
// question : SUR QUELLE HORLOGE les couples arrivent.
//
// Le noyau est déjà tenu par `TestBombStatsCarriersKilled` (bomb_stats_test.go) — il joint des
// entrées supposées bien datées. Ce qui n'était tenu NULLE PART, c'est le pas d'avant : que
// `attachBombStats` passe `opt.MatchKills` TEL QUEL, sans lui appliquer `FilmToMatchOffsetMS`.
//
// LE TEST EST DONC BÂTI POUR ÉCHOUER SI QUELQU'UN AJOUTE LE PONT : le décalage y vaut
// −5 000 ms (`FilmClockOriginUS` 1 000 000 µs, `DeathOffsetMS` 6 000), c'est-à-dire un ordre de
// grandeur au-dessus de la période de portage elle-même. Un kill converti par erreur sortirait
// de la période et le compte tomberait à zéro ; un kill hors période converti par erreur y
// entrerait. Un décalage réaliste (33-114 ms mesurés) n'aurait pas cette propriété — c'est pour
// ça qu'il est ici volontairement énorme.

// bombDocOptions rend les options minimales pour armer `attachBombStats` : la garde de mode, le
// pont d'horloge des ARMEMENTS (celui qui ne doit PAS toucher aux kills) et les couples.
func bombDocOptions(kills MatchKillsInput) (Options, OwnerReport) {
	opt := Options{
		Bomb:              BombInput{CarryScanned: true},
		FilmClockOriginUS: 1_000_000,
		MatchKills:        kills,
	}
	own := OwnerReport{SlotXUID: map[uint32]uint64{1: 7}, DeathOffsetMS: 6000}
	return opt, own
}

// TestAttachBombStatsKillsHorlogeDuMatch : un couple daté sur l'horloge du match tombe dans la
// période telle qu'elle est, sans recalage — et la borne de fin ne s'étend que de la tolérance
// mesurée (`bombCarrierKillToleranceMS`), pas d'une milliseconde de plus.
func TestAttachBombStatsKillsHorlogeDuMatch(t *testing.T) {
	carry := bombCarryDe(bombPeriode(7, 10_000, 12_000))
	fermeture := int64(12_000 + bombCarrierKillToleranceMS)
	cases := []struct {
		nom   string
		kills []KillRef
		veut  map[string]int
	}{
		{
			nom:   "en plein portage : le tueur est credite",
			kills: []KillRef{{KillerXUID: 5, VictimXUID: 7, TimeMS: 11_000}},
			veut:  map[string]int{"5": 1, "7": 0},
		},
		{
			nom:   "exactement a la borne toleree : encore credite",
			kills: []KillRef{{KillerXUID: 5, VictimXUID: 7, TimeMS: fermeture}},
			veut:  map[string]int{"5": 1},
		},
		{
			nom:   "1 ms apres la borne toleree : plus personne n est credite",
			kills: []KillRef{{KillerXUID: 5, VictimXUID: 7, TimeMS: fermeture + 1}},
			veut:  map[string]int{"7": 0},
		},
		{
			nom:   "suicide du porteur : ne credite personne",
			kills: []KillRef{{KillerXUID: 7, VictimXUID: 7, TimeMS: 11_000}},
			veut:  map[string]int{"7": 0},
		},
	}
	for _, c := range cases {
		t.Run(c.nom, func(t *testing.T) {
			doc := ReplayDocument{MatchID: "m"}
			opt, own := bombDocOptions(MatchKillsInput{Read: true, Kills: c.kills})
			attachBombStats(&doc, opt, own, carry)
			if doc.BombStats == nil {
				t.Fatal("aucune statistique posée sur le document")
			}
			if !doc.BombStats.Coverage.KillsRead {
				t.Fatal("KillsRead = false alors que la source a été lue")
			}
			for xuid, veut := range c.veut {
				assertBombInt(t, *doc.BombStats, xuid,
					func(p BombPlayerStats) *int { return p.CarriersKilled },
					veut, "carriers_killed")
			}
		})
	}
}

// TestAttachBombStatsKillsNonLus : source non lue = champ ABSENT chez tous, jamais un zéro. Le
// document publie `KillsRead=false` pour le dire — « on n'a pas regardé ».
func TestAttachBombStatsKillsNonLus(t *testing.T) {
	doc := ReplayDocument{MatchID: "m"}
	// Des couples SONT fournis : c'est le témoin de lecture, et lui seul, qui décide.
	opt, own := bombDocOptions(MatchKillsInput{
		Read:  false,
		Kills: []KillRef{{KillerXUID: 5, VictimXUID: 7, TimeMS: 11_000}},
	})
	attachBombStats(&doc, opt, own, bombCarryDe(bombPeriode(7, 10_000, 12_000)))
	if doc.BombStats == nil {
		t.Fatal("aucune statistique posée sur le document")
	}
	if doc.BombStats.Coverage.KillsRead {
		t.Fatal("KillsRead = true alors que Read valait false")
	}
	assertBombAbsent(t, *doc.BombStats,
		func(p BombPlayerStats) bool { return p.CarriersKilled != nil }, "carriers_killed")
}

// TestAttachBombStatsKillsDenominateur : le dénominateur publié est le nombre de couples FOURNIS,
// et rien d'autre — les couples que le producteur a perdus (`Dropped`) n'y entrent pas, ils sont
// journalisés à côté. Sans ce test, un `Dropped` qui se mettrait à gonfler `Kills` passerait.
func TestAttachBombStatsKillsDenominateur(t *testing.T) {
	doc := ReplayDocument{MatchID: "m"}
	opt, own := bombDocOptions(MatchKillsInput{
		Read:    true,
		Kills:   []KillRef{{KillerXUID: 5, VictimXUID: 7, TimeMS: 11_000}},
		Dropped: 4,
	})
	attachBombStats(&doc, opt, own, bombCarryDe(bombPeriode(7, 10_000, 12_000)))
	if doc.BombStats == nil {
		t.Fatal("aucune statistique posée sur le document")
	}
	cov := doc.BombStats.Coverage
	if cov.Kills != 1 || cov.KillsOnCarrier != 1 {
		t.Fatalf("couverture = {kills %d, surPorteur %d}, attendu {1, 1} — `Dropped` (4) ne doit"+
			" gonfler ni le dénominateur ni le numérateur", cov.Kills, cov.KillsOnCarrier)
	}
}
