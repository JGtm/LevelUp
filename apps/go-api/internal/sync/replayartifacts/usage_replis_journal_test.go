package replayartifacts

// usage_replis_journal_test.go — LE PRODUCTEUR DE PASSES JOURNALISE-T-IL LES REPLIS ?
//
// # LE DEFAUT QUE CE FICHIER FERME (revue de jalon M1, ronde 2, constat F2)
//
// `UsageMatchSummary.Fallbacks` etait ECRIT par `BuildUsageSummary` et LU PAR AUCUN CODE DE
// PRODUCTION : les deux producteurs de passes journalisaient `EquipmentChanges` et ignoraient
// ce champ. Trois entrees du registre des replis portaient pourtant une cible de retrait qui
// nommait le `replay-corpus-gate` — un instrument qui NE PEUT PAS produire cette mesure,
// puisqu'il lit les artefacts et que ces replis-la se declenchent APRES la cuisson.
//
// Le journal des passes EST l'instrument. Ce fichier verifie qu'il existe vraiment, aux deux
// grains, et qu'il part bien du POINT D'ENTREE du producteur — pas seulement de son helper.
//
// # MUTATION QUI DOIT LE FAIRE ROUGIR
//
// Retirer la ligne `journaliserReplisUsage(ctx, d, prets)` de `persisterResumesUsage` :
// `TestProducteurPostSyncJournaliseLesReplis` rougit. Jouee et restauree par NOM le 2026-09-16.

// `capturerJournal` (tampon JSON) vit dans `journal_test.go`, meme paquet : une seconde copie
// aurait diverge (regle 6 du depot). Les assertions portent donc sur la forme JSON du handler.

import (
	"context"
	"strings"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/games/halo_infinite/film/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// resumeAvecReplis forge une projection dont le rapport de replis porte ces declenchements.
func resumeAvecReplis(matchID string, r ...fallback.Declenchement) resumeUsagePret {
	return resumeUsagePret{
		matchID: matchID,
		summary: replay.UsageSummary{Match: replay.UsageMatchSummary{Fallbacks: r}},
	}
}

// TestJournalDesReplisAuxDeuxGrains : une passe ecrit une ligne par match declenchant, ET une
// ligne de passe qui porte le cumul.
func TestJournalDesReplisAuxDeuxGrains(t *testing.T) {
	buf := capturerJournal(t)
	journaliserReplisUsage(context.Background(), Deps{Gamertag: "GT", TitleSlug: "halo_infinite"},
		[]resumeUsagePret{
			resumeAvecReplis("m1",
				fallback.Declenchement{Nom: fallback.NomGardeEquipementNegatifAZero, Declenchements: 2}),
			resumeAvecReplis("m2"),
			resumeAvecReplis("m3",
				fallback.Declenchement{Nom: fallback.NomGardeEquipementNegatifAZero, Declenchements: 1},
				fallback.Declenchement{Nom: fallback.NomGestePremiereVieDuSlot, Declenchements: 4}),
		})
	sortie := buf.String()
	for _, attendu := range []string{
		`"match_id":"m1","replis":"repli_garde_equipement_negatif_a_zero=2"`,
		`"match_id":"m3"`,
		`"matchs":3,"matchsConcernes":2`,
		`"replis":"repli_garde_equipement_negatif_a_zero=3 repli_geste_premiere_vie_du_slot=4"`,
	} {
		if !strings.Contains(sortie, attendu) {
			t.Errorf("le journal ne porte pas %q.\nSortie :\n%s", attendu, sortie)
		}
	}
	// LE MATCH SANS REPLI N'ECRIT PAS SA LIGNE : un lot entierement lu ne doit rien noyer.
	if strings.Contains(sortie, `"match_id":"m2"`) {
		t.Errorf("m2 n'a declenche aucun repli : il ne doit pas avoir de ligne par match.\n%s", sortie)
	}
}

// TestJournalDesReplisDitAucun : la ligne de PASSE s'ecrit meme quand rien ne s'est declenche.
//
// C'EST L'ASSERTION QUI PORTE D14 (d). Un silence ferait lire « jamais declenche » la ou il n'y
// a que « jamais instrumente », et un retrait sec reposerait alors sur une absence de trace.
func TestJournalDesReplisDitAucun(t *testing.T) {
	buf := capturerJournal(t)
	journaliserReplisUsage(context.Background(), Deps{Gamertag: "GT", TitleSlug: "halo_infinite"},
		[]resumeUsagePret{resumeAvecReplis("m1"), resumeAvecReplis("m2")})
	sortie := buf.String()
	if !strings.Contains(sortie, `"matchs":2,"matchsConcernes":0,"replis":"aucun"`) {
		t.Errorf("la ligne de passe doit dire `aucun` a zero declenchement.\nSortie :\n%s", sortie)
	}
}

// TestProducteurPostSyncJournaliseLesReplis : LE CABLAGE, depuis le point d'entree.
//
// Il appelle `persisterResumesUsage` — le producteur lui-meme — sur un artefact dont la
// projection DECLENCHE un repli, avec `AcquireWriter` nil (aucune base n'est ouverte : la
// journalisation precede l'acquisition du writer, et c'est voulu — une passe qui ne peut pas
// persister doit quand meme DIRE ce qu'elle a mesure).
func TestProducteurPostSyncJournaliseLesReplis(t *testing.T) {
	buf := capturerJournal(t)
	doc := docProjectionAvecRepli()
	ctx := ctxkeys.WithTitleSlug(context.Background(), "halo_infinite")
	b := &bilanDerivations{}
	persisterResumesUsage(ctx, Deps{
		RepoRoot:  racineDepot(t),
		TitleSlug: "halo_infinite",
		Gamertag:  "GT",
	}, b, []artefactLu{{matchID: "m-cablage", path: "artefact.json", doc: doc}})

	sortie := buf.String()
	if !strings.Contains(sortie, "replis de la passe") {
		t.Fatalf("le producteur post-sync n'ecrit AUCUNE ligne de passe : le canal `Fallbacks` "+
			"est de nouveau sans lecteur (constat F2).\nSortie :\n%s", sortie)
	}
	if !strings.Contains(sortie, `"match_id":"m-cablage"`) ||
		!strings.Contains(sortie, string(fallback.NomGardeEquipementNegatifAZero)) {
		t.Errorf("le repli declenche par la projection n'atteint pas le journal.\nSortie :\n%s", sortie)
	}
}

// docProjectionAvecRepli : un document dont `BuildUsageSummary` declenche
// `repli_garde_equipement_negatif_a_zero` — une prise de camouflage pour DEUX activations,
// donc une garde a -1 que le clamp ecrase.
func docProjectionAvecRepli() *replay.ReplayDocument {
	doc := &replay.ReplayDocument{
		SchemaVersion:   replay.SchemaVersion,
		FrameIntervalMS: 100,
		FrameCount:      1000,
		Roster:          []replay.RosterEntry{{XUID: "111", FilmIndex: 0}},
		Tracks:          []replay.Track{{Slot: 1, XUID: "111", StartFrame: 0, EndFrame: 900}},
		AbilityLabels:   map[string]replay.Label{"8": {En: "Camo", Fr: "Camo", Family: "powerup_camo"}},
		EquipmentChanges: []replay.EquipmentChange{
			{Slot: 1, T: 10, Kind: replay.EquipmentTaken, R: 8, From: replay.NoAbilityRank},
		},
	}
	for i := 0; i < 2; i++ {
		t0 := 100 + i*100
		doc.EquipmentEpisodes = append(doc.EquipmentEpisodes,
			replay.EquipmentEpisode{Slot: 1, Fam: replay.EquipFamilyCamo, T0: t0, T1: t0 + 10})
	}
	return doc
}
