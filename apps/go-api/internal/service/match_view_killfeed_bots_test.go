// Package service — match_view_killfeed_bots_test.go : LA FRONTIÈRE ENTRE NOMMER ET AGRÉGER.
//
// Une ligne de la canonique dont le xuid est NULL décrit un BOT (cf. doctrine
// domain.KVPairRaw). Le scanner la rend avec un xuid vide, et deux traitements opposés
// s'appliquent à partir de là :
//
//   - le KILL FEED la NOMME — un journal cite ce qui s'est passé, et « tué 343 Razzle [bot] »
//     est vrai même sans identifiant ;
//   - les AGRÉGATS (Dominance, courbe K/D, antagonistes, némésis) l'IGNORENT — ils
//     répondent sur des JOUEURS, et agréger sous la clé "" fusionnerait tous les bots
//     d'un match en un acteur fantôme.
//
// Ces tests verrouillent les deux moitiés ENSEMBLE : c'est leur opposition qui est la
// règle, et une seule des deux moitiés testée laisserait l'autre dériver.
package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

const botRazzle = "343 Razzle [bot]"

// ---------------------------------------------------------------------------
// victimsByKill / decorateVictim — le feed nomme la victime bot
// ---------------------------------------------------------------------------

// TestVictimsByKill_VictimeBotIndexeeParSonNom : une victime sans xuid entre dans l'index
// dès qu'elle est nommée. Le TUEUR bot, lui, reste dehors : la clé d'appariement est le
// xuid du tueur, et le feed ne porte aucun event de kill pour un bot.
func TestVictimsByKill_VictimeBotIndexeeParSonNom(t *testing.T) {
	pairs := []domain.KVPairRaw{
		{KillerXUID: "A", VictimXUID: "", VictimGT: botRazzle, TimeMS: 1000},
		{KillerXUID: "", KillerGT: botRazzle, VictimXUID: "A", VictimGT: "Ana", TimeMS: 2000},
		{KillerXUID: "A", VictimXUID: "B", VictimGT: "Bob", TimeMS: 3000},
	}

	out := victimsByKill(pairs)

	if len(out) != 2 {
		t.Fatalf("attendu 2 clés (les deux kills DU joueur A), obtenu %d : %+v", len(out), out)
	}
	bot, ok := out[killFeedKey{xuid: "A", timeMS: 1000}]
	if !ok {
		t.Fatal("le kill (A,1000) sur un bot n'est pas indexé — la victime nommée est perdue")
	}
	if bot.xuid != "" || bot.gamertag != botRazzle || bot.conflict {
		t.Errorf("victime bot = %+v, attendu xuid vide + %q sans conflit", bot, botRazzle)
	}
	if _, ok := out[killFeedKey{xuid: "", timeMS: 2000}]; ok {
		t.Error("le TUEUR bot a été indexé sous la clé vide — aucun event de feed ne l'y attend")
	}
}

// TestVictimsByKill_DeuxBotsSurLaMemeCle : la garde d'unanimité tient SANS xuid. Deux bots
// différents tués au même instant par le même tueur ne peuvent pas se départager
// autrement que par leur nom ; sans cette comparaison, deux xuid vides passeraient pour
// la même victime et le feed en nommerait une au hasard.
func TestVictimsByKill_DeuxBotsSurLaMemeCle(t *testing.T) {
	pairs := []domain.KVPairRaw{
		{KillerXUID: "A", VictimGT: botRazzle, TimeMS: 1000},
		{KillerXUID: "A", VictimGT: "343 Brutus [bot]", TimeMS: 1000},
	}

	out := victimsByKill(pairs)

	v, ok := out[killFeedKey{xuid: "A", timeMS: 1000}]
	if !ok {
		t.Fatal("clé (A,1000) absente")
	}
	if !v.conflict {
		t.Errorf("deux bots distincts au même instant : conflit non détecté (%+v)", v)
	}
}

// TestVictimsByKill_MemeBotDeuxFois : le miroir du test précédent. Deux lignes qui
// désignent LE MÊME bot ne sont pas une contradiction — le feed doit encore le nommer.
func TestVictimsByKill_MemeBotDeuxFois(t *testing.T) {
	pairs := []domain.KVPairRaw{
		{KillerXUID: "A", VictimGT: botRazzle, TimeMS: 1000},
		{KillerXUID: "A", VictimGT: botRazzle, TimeMS: 1000},
	}

	v, ok := victimsByKill(pairs)[killFeedKey{xuid: "A", timeMS: 1000}]
	if !ok {
		t.Fatal("clé (A,1000) absente")
	}
	if v.conflict {
		t.Errorf("le même bot deux fois a été pris pour un conflit (%+v)", v)
	}
}

// TestVictimsByKill_VictimeSansIdentiteEcartee : ni xuid ni nom = rien à afficher. Indexer
// cette ligne-là ferait pire que ne rien faire : elle passerait pour un conflit avec la
// vraie victime de la même clé, qui cesserait alors d'être nommée.
func TestVictimsByKill_VictimeSansIdentiteEcartee(t *testing.T) {
	pairs := []domain.KVPairRaw{
		{KillerXUID: "A", VictimXUID: "B", VictimGT: "Bob", TimeMS: 1000},
		{KillerXUID: "A", TimeMS: 1000}, // ligne sans identité de victime
	}

	v, ok := victimsByKill(pairs)[killFeedKey{xuid: "A", timeMS: 1000}]
	if !ok {
		t.Fatal("clé (A,1000) absente")
	}
	if v.conflict || v.xuid != "B" {
		t.Errorf("une ligne sans identité a effacé la vraie victime : %+v", v)
	}
}

// TestDecorateVictim_BotNiXuidNiEquipe : la décoration d'une victime bot pose SON NOM et
// rien d'autre. Un `VictimXUID` de chaîne vide donnerait au front un identifiant qui
// ressemble à un joueur ; une équipe résolue sur la clé "" serait celle du premier venu.
func TestDecorateVictim_BotNiXuidNiEquipe(t *testing.T) {
	e := &domain.MatchHighlightEvent{EventType: analysis.EventTypeKill}
	// Le scoreboard porte volontairement une entrée à xuid vide : si decorateVictim
	// interrogeait la map avec "", il lui collerait cette équipe-là.
	teams := map[string]int{"": 7, "B": 1}

	decorateVictim(e, &victimRef{xuid: "", gamertag: botRazzle}, teams)

	if e.VictimGamertag == nil || *e.VictimGamertag != botRazzle {
		t.Errorf("gamertag de la victime bot = %v, attendu %q", e.VictimGamertag, botRazzle)
	}
	if e.VictimXUID != nil {
		t.Errorf("VictimXUID posé (%q) sur une victime SANS identité", *e.VictimXUID)
	}
	if e.VictimTeamID != nil {
		t.Errorf("VictimTeamID posé (%d) — résolu sur la clé vide", *e.VictimTeamID)
	}
}

// TestDecorateKillFeed_KillSurBotNommeAuFil : NON-RÉGRESSION DE BOUT EN BOUT (le bug du
// 2026-09-02). Sur un match à bot, le kill humain→bot doit porter le nom du bot et AUCUN
// xuid, pendant que le kill humain→humain garde son identité complète.
func TestDecorateKillFeed_KillSurBotNommeAuFil(t *testing.T) {
	events := []domain.MatchHighlightEvent{
		{EventType: analysis.EventTypeKill, EventTimeMS: ptrI64k(1000), ActorXUID: ptrS("A")},
		{EventType: analysis.EventTypeKill, EventTimeMS: ptrI64k(2000), ActorXUID: ptrS("A")},
	}
	victims := []domain.KVPairRaw{
		{KillerXUID: "A", VictimXUID: "", VictimGT: botRazzle, TimeMS: 1000},
		{KillerXUID: "A", VictimXUID: "B", VictimGT: "Bob", TimeMS: 2000},
	}

	decorateKillFeed(context.Background(), events, killFeedInputs{
		victims: victims, scoreboard: feedScoreboard(),
	})

	surBot := events[0]
	if surBot.VictimGamertag == nil || *surBot.VictimGamertag != botRazzle {
		t.Errorf("kill (A,1000) : victime = %v, attendu %q", surBot.VictimGamertag, botRazzle)
	}
	if surBot.VictimXUID != nil {
		t.Errorf("kill (A,1000) : VictimXUID = %q, attendu nil (un bot n'a pas de xuid)", *surBot.VictimXUID)
	}
	if surBot.VictimTeamID != nil {
		t.Errorf("kill (A,1000) : VictimTeamID = %d, attendu nil (absent du scoreboard)", *surBot.VictimTeamID)
	}
	surHumain := events[1]
	if surHumain.VictimXUID == nil || *surHumain.VictimXUID != "B" ||
		surHumain.VictimGamertag == nil || *surHumain.VictimGamertag != "Bob" {
		t.Errorf("kill (A,2000) : victime %v/%v, attendu B/Bob — la ligne de bot a contaminé "+
			"l'appariement", surHumain.VictimXUID, surHumain.VictimGamertag)
	}
	if surHumain.VictimTeamID == nil || *surHumain.VictimTeamID != 1 {
		t.Errorf("kill (A,2000) : équipe de la victime %v, attendu 1", surHumain.VictimTeamID)
	}
}

// ---------------------------------------------------------------------------
// Agrégats — D2 : humains seulement
// ---------------------------------------------------------------------------

// botPairs : une paire humaine, une mort infligée PAR un bot, une mort DE bot. Seule la
// première décrit un affrontement entre deux joueurs.
func botPairs() []domain.KVPairRaw {
	return []domain.KVPairRaw{
		{KillerXUID: "A", KillerGT: "Ana", VictimXUID: "B", VictimGT: "Bob", TimeMS: 1000},
		{KillerXUID: "", KillerGT: botRazzle, VictimXUID: "A", VictimGT: "Ana", TimeMS: 2000},
		{KillerXUID: "A", KillerGT: "Ana", VictimXUID: "", VictimGT: botRazzle, TimeMS: 3000},
	}
}

// TestBuildTugEvents_BotsIgnores : la Dominance oppose DEUX CAMPS. Un bot n'appartient à
// aucun des deux, et le compter le verserait mécaniquement au camp adverse (isAlly est
// faux par défaut) — la carte lirait alors une pression ennemie qui n'existe pas.
func TestBuildTugEvents_BotsIgnores(t *testing.T) {
	got := buildTugEvents(botPairs(), "A")

	if len(got) != 1 {
		t.Fatalf("attendu 1 event (la seule paire humaine), obtenu %d : %+v", len(got), got)
	}
	if got[0].TimeMS != 1000 || !got[0].IsAlly {
		t.Errorf("event = %+v, attendu la paire A→B à 1000 côté allié", got[0])
	}
}

// TestBuildKDEvents_BotsIgnores : la garde n'est PAS redondante avec la comparaison à
// myXUID. La mort infligée par le bot porte un VictimXUID égal au viewer — sans la garde
// elle creuserait la courbe K/D d'un joueur qui n'a pas été tué par un joueur.
func TestBuildKDEvents_BotsIgnores(t *testing.T) {
	got := buildKDEvents(botPairs(), "A")

	if len(got) != 1 {
		t.Fatalf("attendu 1 event, obtenu %d : %+v", len(got), got)
	}
	if !got[0].IsKill || got[0].TimeMS != 1000 {
		t.Errorf("event = %+v, attendu le frag A→B à 1000", got[0])
	}
}

// TestBuildKillerVictimPairs_BotsIgnores : les antagonistes nomment des adversaires. Une
// paire à xuid vide y créerait une ligne d'un joueur fantôme agrégeant tous les bots.
func TestBuildKillerVictimPairs_BotsIgnores(t *testing.T) {
	got := buildKillerVictimPairs(botPairs(), nil)

	if len(got) != 1 {
		t.Fatalf("attendu 1 paire humaine, obtenu %d : %+v", len(got), got)
	}
	if got[0].KillerXUID != "A" || got[0].VictimXUID != "B" {
		t.Errorf("paire = %+v, attendu A→B", got[0])
	}
}

// TestSynthesizeEventRawFromKVPairs_BotsIgnores : le fallback title-agnostic (titres dont
// highlight_events ne porte que des médailles, ex. Halo 5) reconstruit des events kill et
// death depuis les paires. Un acteur de chaîne vide y deviendrait le tueur — ou le mort —
// de toutes les lignes de bot du match.
func TestSynthesizeEventRawFromKVPairs_BotsIgnores(t *testing.T) {
	got := synthesizeEventRawFromKVPairs(botPairs(), "m1")

	if len(got) != 2 {
		t.Fatalf("attendu 2 events (1 kill + 1 death de la seule paire humaine), obtenu %d : %+v",
			len(got), got)
	}
	for i, e := range got {
		if e.XUID == nil || *e.XUID == "" {
			t.Errorf("event %d : acteur %v — un xuid vide a été synthétisé", i, e.XUID)
		}
		if e.TimeMS == nil || *e.TimeMS != 1000 {
			t.Errorf("event %d : instant %v, attendu 1000", i, e.TimeMS)
		}
	}
}
