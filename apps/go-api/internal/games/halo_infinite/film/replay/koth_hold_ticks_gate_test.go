package replay

// koth_hold_ticks_gate_test.go — LE GATE DES TICS DE COLLINE : `comp 23 A` reproduit-il ENCORE
// `zone_scoring_ticks` de l'API, joueur par joueur ?
//
// # POURQUOI CE FICHIER EXISTE (lot 6.7 phase B2, item 3 ; audit L10)
//
// `hill_hold_ticks.go:8-12` affirme que le compteur « reproduit `ZonesStats.StrongholdScoringTicks`
// de l'API EXACTEMENT, joueur par joueur, sur 31 joueurs de 4 films ». Cette affirmation datait du
// 2026-08-30 et n'etait tenue par AUCUN test qui echoue : `TestCollineStatborgE1Bis` est un
// RELEVE — il journalise ses desaccords et passe quand meme. Une affirmation d'exactitude sans
// assertion est une affirmation qui derive en silence.
//
// # CE QU'IL AJOUTE AU RELEVE
//
//	le RELEVE (E1-bis)   balaye 64 composants et journalise ceux qui collent — il CHERCHE
//	le GATE (ici)        ne connait que `comp 23 A` et ECHOUE sur le moindre desaccord — il TIENT
//
// Il porte en plus le critere du lot : **aucun joueur au-dessus de son oracle**. Un compteur qui
// SOUS-estime est une couverture incomplete ; un compteur qui SUR-estime publie des tics qui
// n'ont pas eu lieu, et c'est le defaut que l'audit poursuit sur tous les calques d'objectif.
//
// # LE CORPUS EST PASSE DE QUATRE A SIX FILMS LE 2026-09-11
//
// Chasm (`606d9844`) et Shogun (`8076f97f`) etaient au cache depuis toujours, mais leur cuisson
// echouait « carte hors catalogue ([]) ». L'audit supposait un catalogue sans formes de colline :
// c'est FAUX — les six cartes portent 5 ou 6 volumes `hill`, tous a forme. La cause ecrite dans
// `E4_cuisson_koth.log` est la bonne : « match absent du registre — identite de carte non
// resolue » (`sql: no rows in result set`). Les matchs y sont depuis, les six films cuisent.
//
// REGIME : garde `ZONE_FILM` (repertoire de chunks), UN FILM PAR PROCESSUS, lecture seule,
// AUCUNE base ouverte — le meme que tous les instruments de ce paquet.
//
//	$env:ZONE_FILM="<cache>/film_chunks/606d9844"
//	go test ./internal/games/halo_infinite/film/replay/ -run KothHoldTicks -v

import (
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// kothTicksDesaccord est un joueur dont le compteur du film contredit l'oracle de l'API.
type kothTicksDesaccord struct {
	xuid   string
	slot   int
	publie int
	oracle int
}

// kothTicksVerdict confronte `comp 23 A` a l'oracle, joueur par joueur, et rend les desaccords.
//
// PURE ET SANS `testing` : c'est ce qui permet de la rejouer sur un oracle MUTE dans le meme
// processus, et donc de prouver que le comparateur mord (cf. le sous-test de mutation).
//
// Un slot SANS EMISSION est la lecture « zero » du compteur, pas une absence de mesure : le
// joueur n'a jamais tenu la colline. C'est la meme convention que la phase 2 du releve E1-bis.
func kothTicksVerdict(
	recs []objectiveevents.StatRecord, oracle []e1bJoueur,
) (apparies int, desaccords []kothTicksDesaccord) {
	lines := make([]objectiveevents.PlayerLine, 0, len(oracle))
	attendu := map[string]int{}
	for _, j := range oracle {
		attendu[j.xuid] = j.tics
		if j.kills < 0 {
			continue // bot sans ligne de match : aucun pont possible, et c'est dit
		}
		lines = append(lines, objectiveevents.PlayerLine{
			XUID: j.xuid, Kills: j.kills, Deaths: j.deaths, Assists: j.assists,
		})
	}
	identity := objectiveevents.SlotIdentityFrom(recs, lines)
	series := objectiveevents.SeriesTotal(recs, holdTicksComponent, false)
	slots := make([]int, 0, len(identity))
	for s := range identity {
		slots = append(slots, s)
	}
	sort.Ints(slots)
	for _, slot := range slots {
		xuid := identity[slot]
		publie := 0
		if pts := series[slot]; len(pts) > 0 {
			publie = int(pts[len(pts)-1].Value)
		}
		apparies++
		if publie != attendu[xuid] {
			desaccords = append(desaccords,
				kothTicksDesaccord{xuid: xuid, slot: slot, publie: publie, oracle: attendu[xuid]})
		}
	}
	return apparies, desaccords
}

// TestKothHoldTicksGate — LE GATE. Un film par processus, via `ZONE_FILM`.
func TestKothHoldTicksGate(t *testing.T) {
	dir := p2aRequireFilm(t)
	short := filepath.Base(dir)
	oracle, ok := e1bOracle[short]
	if !ok {
		t.Skipf("film %s hors corpus de tics de colline (aucun oracle gele pour lui)", short)
	}
	recs := objectiveevents.StatRecords(p2aBobine(t, dir))
	if len(recs) == 0 {
		t.Fatalf("%s : aucun enregistrement de statistiques — rien a confronter", short)
	}

	apparies, desaccords := kothTicksVerdict(recs, oracle)
	t.Logf("KOTH-TICS\t%s\tcomp 23 A : %d slot(s) apparie(s) sur %d ligne(s) d'oracle, %d desaccord(s)",
		short, apparies, len(oracle), len(desaccords))
	if apparies == 0 {
		t.Fatalf("%s : AUCUN slot apparie — le pont d'identite n'a rien rendu, le gate serait vide",
			short)
	}
	dessus := 0
	for _, d := range desaccords {
		sens := "SOUS"
		if d.publie > d.oracle {
			sens = "AU-DESSUS"
			dessus++
		}
		t.Errorf("KOTH-TICS\t%s\tslot %d (%s) : publie %d, oracle %d — %s",
			short, d.slot, d.xuid, d.publie, d.oracle, sens)
	}
	if dessus > 0 {
		t.Errorf("KOTH-TICS\t%s\t%d joueur(s) AU-DESSUS de leur oracle — tolerance 0", short, dessus)
	}

	// MUTATION — le comparateur mord-il ? On rejoue le MEME verdict sur un oracle ou UN joueur
	// porte un tic de plus. Sans cette contre-epreuve, un comparateur qui rendrait toujours
	// « aucun desaccord » passerait le gate sur n'importe quel film.
	t.Run("mutation", func(t *testing.T) {
		mute := make([]e1bJoueur, len(oracle))
		copy(mute, oracle)
		for i := range mute {
			if mute[i].kills < 0 {
				continue // un bot n'est pas apparie : le muter ne prouverait rien
			}
			mute[i].tics++
			break
		}
		if _, d := kothTicksVerdict(recs, mute); len(d) == 0 {
			t.Fatalf("%s : un oracle MUTE d'un tic ne produit aucun desaccord — le comparateur ne mord pas",
				short)
		}
	})
}
