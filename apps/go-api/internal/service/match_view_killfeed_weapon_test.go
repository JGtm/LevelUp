package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

func ptrS(s string) *string  { return &s }
func ptrI64k(v int64) *int64 { return &v }
func ptrIk(v int) *int       { return &v }

// feedFixture : deux tueurs de deux équipes, trois kills et un event non-kill.
func feedFixture() []domain.MatchHighlightEvent {
	return []domain.MatchHighlightEvent{
		{EventType: analysis.EventTypeKill, EventTimeMS: ptrI64k(1000), ActorXUID: ptrS("A")},
		{EventType: analysis.EventTypeKill, EventTimeMS: ptrI64k(2000), ActorXUID: ptrS("B")},
		{EventType: analysis.EventTypeKill, EventTimeMS: ptrI64k(3000), ActorXUID: ptrS("A")},
		{EventType: analysis.EventTypeMedal, EventTimeMS: ptrI64k(4000), ActorXUID: ptrS("A")},
	}
}

func feedScoreboard() []domain.ScoreboardRaw {
	return []domain.ScoreboardRaw{
		{XUID: "A", TeamID: ptrIk(0)},
		{XUID: "B", TeamID: ptrIk(1)},
		{XUID: "C"}, // sans team_id : ne doit rien casser
	}
}

// TestDecorateKillFeed_PoseArmeEtEquipe : le chemin nominal. L'arme n'arrive que sur les
// kills appariés, l'équipe arrive sur TOUS les events dont l'acteur est au scoreboard.
func TestDecorateKillFeed_PoseArmeEtEquipe(t *testing.T) {
	events := feedFixture()
	sources := []domain.KillSourceRaw{
		{XUID: "A", TimeMS: 1000, SourceTag: 0x11},
		{XUID: "B", TimeMS: 2000, SourceTag: 0x22},
		// pas de source pour (A, 3000) : trou assumé
	}
	adapter := &stubAssetURL{killIcons: map[uint32]canonical.KillSourceIcon{
		0x11: {WeaponKey: "hinf_br75", Label: "BR75", ImageURL: "/static/x/killfeed-00.png", Tinted: true},
		0x22: {Label: "", ImageURL: "/static/x/killfeed-65.png", Tinted: true}, // melee : sans nom propre
	}}

	decorateKillFeed(context.Background(), events, killFeedInputs{
		sources: sources, scoreboard: feedScoreboard(), assetURL: adapter,
	})

	if events[0].WeaponKey != "hinf_br75" || events[0].WeaponLabel != "BR75" {
		t.Errorf("kill 0 : arme = %q/%q, attendu hinf_br75/BR75", events[0].WeaponKey, events[0].WeaponLabel)
	}
	if !events[0].WeaponImageTinted || events[0].WeaponImageURL == "" {
		t.Errorf("kill 0 : image = %q tinted=%v", events[0].WeaponImageURL, events[0].WeaponImageTinted)
	}
	if events[1].WeaponImageURL == "" || events[1].WeaponLabel != "" {
		t.Errorf("kill 1 (melee) : doit avoir une image sans nom propre, got %q/%q",
			events[1].WeaponImageURL, events[1].WeaponLabel)
	}
	if events[2].WeaponImageURL != "" {
		t.Errorf("kill 2 : sans source appariée, il ne doit PAS avoir d'image (got %q)", events[2].WeaponImageURL)
	}
	// L'équipe est posée sur tous les events, kill ou non — c'est elle qui colore le nom.
	for i, want := range []int{0, 1, 0, 0} {
		if events[i].ActorTeamID == nil || *events[i].ActorTeamID != want {
			t.Errorf("event %d : team_id = %v, attendu %d", i, events[i].ActorTeamID, want)
		}
	}
}

// TestDecorateKillFeed_AucuneArmeSurUnEventNonKill verrouille la règle qui évite le
// non-sens : une médaille ou un event de mode n'a pas d'arme, même si son acteur a tué à
// cet instant précis.
func TestDecorateKillFeed_AucuneArmeSurUnEventNonKill(t *testing.T) {
	events := []domain.MatchHighlightEvent{
		{EventType: analysis.EventTypeMedal, EventTimeMS: ptrI64k(1000), ActorXUID: ptrS("A")},
	}
	sources := []domain.KillSourceRaw{{XUID: "A", TimeMS: 1000, SourceTag: 0x11}}
	adapter := &stubAssetURL{killIcons: map[uint32]canonical.KillSourceIcon{
		0x11: {ImageURL: "/static/x/killfeed-00.png"},
	}}

	decorateKillFeed(context.Background(), events, killFeedInputs{
		sources: sources, scoreboard: feedScoreboard(), assetURL: adapter,
	})

	if events[0].WeaponImageURL != "" {
		t.Errorf("event medal : arme posée (%q) alors qu'il n'en a pas", events[0].WeaponImageURL)
	}
}

// TestDecorateKillFeed_SourceSansIconeResteSansArme : le cas des sources identifiées mais
// non traduisibles en image (véhicule, bidon, chute, nom alternatif contradictoire). Le
// pont rend faux, et RIEN ne doit être posé — surtout pas une icône par défaut.
func TestDecorateKillFeed_SourceSansIconeResteSansArme(t *testing.T) {
	events := feedFixture()
	sources := []domain.KillSourceRaw{{XUID: "A", TimeMS: 1000, SourceTag: 0xdead}}
	adapter := &stubAssetURL{killIcons: map[uint32]canonical.KillSourceIcon{}} // le pont ne connaît rien

	decorateKillFeed(context.Background(), events, killFeedInputs{
		sources: sources, scoreboard: feedScoreboard(), assetURL: adapter,
	})

	for i, e := range events {
		if e.WeaponImageURL != "" || e.WeaponKey != "" || e.WeaponLabel != "" {
			t.Errorf("event %d : décoré (%q/%q/%q) alors que le pont n'a rien rendu",
				i, e.WeaponKey, e.WeaponLabel, e.WeaponImageURL)
		}
	}
}

// TestDecorateKillFeed_AssistTroisEtats : les trois états de l'assistance, JAMAIS
// confondus. Le kill (A,1000) porte un assistant nommé avec les deux parts ; (B,2000) est
// MESURÉ sans assistant (part du tueur seule) ; (A,3000) n'a aucune entrée appariée et
// reste « on ne sait pas » — AssistState vide, aucune part. La médaille n'est pas touchée.
func TestDecorateKillFeed_AssistTroisEtats(t *testing.T) {
	events := feedFixture()
	assists := []domain.KillAssistRaw{
		{XUID: "A", TimeMS: 1000, AssistGamertag: ptrS("Bob"), AssistXUID: ptrS("B"),
			KillerDamagePct: ptrIk(63), AssistDamagePct: ptrIk(37)},
		{XUID: "B", TimeMS: 2000, KillerDamagePct: ptrIk(100)},
	}

	decorateKillFeed(context.Background(), events, killFeedInputs{
		assists: assists, scoreboard: feedScoreboard(),
	})

	named := events[0]
	if named.AssistState != domain.AssistStateNamed || named.AssistGamertag != "Bob" {
		t.Errorf("kill (A,1000) : état %q / assistant %q, attendu named/Bob",
			named.AssistState, named.AssistGamertag)
	}
	if named.KillerDamagePct == nil || *named.KillerDamagePct != 63 ||
		named.AssistDamagePct == nil || *named.AssistDamagePct != 37 {
		t.Errorf("kill (A,1000) : parts %v/%v, attendu 63/37",
			named.KillerDamagePct, named.AssistDamagePct)
	}
	if named.AssistTeamID == nil || *named.AssistTeamID != 1 {
		t.Errorf("kill (A,1000) : équipe de l'assistant %v, attendu 1 (scoreboard)", named.AssistTeamID)
	}
	none := events[1]
	if none.AssistState != domain.AssistStateNone || none.AssistGamertag != "" {
		t.Errorf("kill (B,2000) : état %q / assistant %q, attendu none/vide",
			none.AssistState, none.AssistGamertag)
	}
	if none.KillerDamagePct == nil || *none.KillerDamagePct != 100 || none.AssistDamagePct != nil {
		t.Errorf("kill (B,2000) : parts %v/%v, attendu 100/nil", none.KillerDamagePct, none.AssistDamagePct)
	}
	unknown := events[2]
	if unknown.AssistState != "" || unknown.KillerDamagePct != nil || unknown.AssistDamagePct != nil {
		t.Errorf("kill (A,3000) sans entrée : état %q parts %v/%v — un « on ne sait pas » a été "+
			"comblé", unknown.AssistState, unknown.KillerDamagePct, unknown.AssistDamagePct)
	}
	if medal := events[3]; medal.AssistState != "" {
		t.Errorf("la médaille porte un état d'assistance (%q)", medal.AssistState)
	}
}

// TestDecorateKillFeed_DegradationsGracieuses : chaque entrée peut manquer sans que la
// carte Dominance en souffre. C'est l'état NOMINAL d'un titre sans décodeur de film
// (Halo 5) et d'un match jamais passé au décodeur.
func TestDecorateKillFeed_DegradationsGracieuses(t *testing.T) {
	cas := []struct {
		nom        string
		sources    []domain.KillSourceRaw
		scoreboard []domain.ScoreboardRaw
		adapter    *stubAssetURL
	}{
		{"aucune source", nil, feedScoreboard(), &stubAssetURL{}},
		{"aucun scoreboard", []domain.KillSourceRaw{{XUID: "A", TimeMS: 1000, SourceTag: 1}}, nil, &stubAssetURL{}},
		{"adapter nil", []domain.KillSourceRaw{{XUID: "A", TimeMS: 1000, SourceTag: 1}}, feedScoreboard(), nil},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			events := feedFixture()
			in := killFeedInputs{sources: c.sources, scoreboard: c.scoreboard}
			if c.adapter != nil {
				in.assetURL = c.adapter
			}
			decorateKillFeed(context.Background(), events, in)
			for i, e := range events {
				if e.WeaponImageURL != "" {
					t.Errorf("event %d : arme posée sans adapter/source valide", i)
				}
			}
		})
	}
	// Tranche vide : pas de panique, pas d'allocation.
	decorateKillFeed(context.Background(), nil, killFeedInputs{})
}

// TestCorrectMatchViewEventsT0_RecaleAussiLesTranchesDuKillFeed : VERROU du bug du
// 2026-08-12. La correction T0 recalait d.events mais laissait killSources/killAssists
// sur l'horloge film → decorateKillFeed (clé exacte xuid+time_ms) n'appariait plus RIEN
// sur un match à T0 non nul : kill feed sans arme ni assistant, sans erreur ni log.
// Valeurs du match réel 000d5950 : T0 = 18465 ms, premier kill 35306 → 16841.
func TestCorrectMatchViewEventsT0_RecaleAussiLesTranchesDuKillFeed(t *testing.T) {
	d := matchViewData{
		events:      []domain.EventRaw{{EventType: analysis.EventTypeKill, TimeMS: ptrI64k(35306), XUID: ptrS("A")}},
		killSources: []domain.KillSourceRaw{{XUID: "A", TimeMS: 35306, SourceTag: 0x11}},
		killAssists: []domain.KillAssistRaw{{XUID: "A", TimeMS: 35306, KillerDamagePct: ptrIk(100)}},
		kvPairs:     []domain.KVPairRaw{{KillerXUID: "A", VictimXUID: "B", VictimGT: "Bob", TimeMS: 35306}},
	}
	correctMatchViewEventsT0(&d, "m1", timeline.BuildForMatchMs(600000, 18465))

	if d.events[0].TimeMS == nil || *d.events[0].TimeMS != 16841 {
		t.Fatalf("event : TimeMS = %v, attendu 16841", d.events[0].TimeMS)
	}
	if d.killSources[0].TimeMS != 16841 {
		t.Errorf("killSource : TimeMS = %d, attendu 16841 (même référentiel que les events)", d.killSources[0].TimeMS)
	}
	if d.killAssists[0].TimeMS != 16841 {
		t.Errorf("killAssist : TimeMS = %d, attendu 16841 (même référentiel que les events)", d.killAssists[0].TimeMS)
	}
	if len(d.kvPairsFeed) != 1 || d.kvPairsFeed[0].TimeMS != 16841 {
		t.Fatalf("kvPairsFeed : %+v, attendu 1 paire à 16841", d.kvPairsFeed)
	}
	if d.kvPairs[0].TimeMS != 35306 {
		t.Errorf("kvPairs (horloge brute, tug/KD) a été corrigé : %d", d.kvPairs[0].TimeMS)
	}

	// Preuve de bout en bout : après correction, la décoration apparie.
	events := []domain.MatchHighlightEvent{
		{EventType: analysis.EventTypeKill, EventTimeMS: ptrI64k(16841), ActorXUID: ptrS("A")},
	}
	adapter := &stubAssetURL{killIcons: map[uint32]canonical.KillSourceIcon{
		0x11: {WeaponKey: "hinf_br75", Label: "BR75", ImageURL: "/static/x/killfeed-00.png"},
	}}
	decorateKillFeed(context.Background(), events, killFeedInputs{
		sources: d.killSources, assists: d.killAssists, victims: d.kvPairsFeed,
		scoreboard: feedScoreboard(), assetURL: adapter,
	})
	if events[0].WeaponKey != "hinf_br75" {
		t.Errorf("l'arme ne s'apparie pas après correction T0 : %+v", events[0])
	}
	if events[0].AssistState != domain.AssistStateNone {
		t.Errorf("l'assistance ne s'apparie pas après correction T0 : état %q", events[0].AssistState)
	}
	if events[0].VictimXUID == nil || *events[0].VictimXUID != "B" {
		t.Errorf("la victime ne s'apparie pas après correction T0 : %v", events[0].VictimXUID)
	}
}

// TestDecorateKillFeed_VictimeTroisEtats : la victime suit la même discipline que
// l'arme. (A,1000) : paire unique → nommée, avec son équipe. (B,2000) : DEUX victimes
// distinctes sur la même clé (double kill au même millisecond) → AUCUNE n'est nommée,
// jamais une au hasard. (A,3000) : aucune paire → rien.
func TestDecorateKillFeed_VictimeTroisEtats(t *testing.T) {
	events := feedFixture()
	victims := []domain.KVPairRaw{
		{KillerXUID: "A", VictimXUID: "B", VictimGT: "Bob", TimeMS: 1000},
		{KillerXUID: "B", VictimXUID: "A", VictimGT: "Ana", TimeMS: 2000},
		{KillerXUID: "B", VictimXUID: "C", VictimGT: "Cid", TimeMS: 2000},
	}

	decorateKillFeed(context.Background(), events, killFeedInputs{
		victims: victims, scoreboard: feedScoreboard(),
	})

	nomme := events[0]
	if nomme.VictimXUID == nil || *nomme.VictimXUID != "B" ||
		nomme.VictimGamertag == nil || *nomme.VictimGamertag != "Bob" {
		t.Errorf("kill (A,1000) : victime %v/%v, attendu B/Bob", nomme.VictimXUID, nomme.VictimGamertag)
	}
	if nomme.VictimTeamID == nil || *nomme.VictimTeamID != 1 {
		t.Errorf("kill (A,1000) : équipe de la victime %v, attendu 1 (scoreboard)", nomme.VictimTeamID)
	}
	if conflit := events[1]; conflit.VictimXUID != nil || conflit.VictimGamertag != nil {
		t.Errorf("kill (B,2000) à double victime contradictoire : une victime a été nommée (%v)",
			conflit.VictimXUID)
	}
	if absent := events[2]; absent.VictimXUID != nil || absent.VictimTeamID != nil {
		t.Errorf("kill (A,3000) sans paire : victime posée (%v)", absent.VictimXUID)
	}
	if medal := events[3]; medal.VictimXUID != nil {
		t.Errorf("la médaille porte une victime (%v)", medal.VictimXUID)
	}
}

// TestDecorateMedalEvents_ResolutionEtRepli : la médaille résolue porte id, label,
// description et visuel ; la médaille inconnue du référentiel garde son SEUL nom brut
// (le front l'écrit en toutes lettres) ; un kill n'est jamais touché.
func TestDecorateMedalEvents_ResolutionEtRepli(t *testing.T) {
	events := []domain.MatchHighlightEvent{
		{EventType: analysis.EventTypeKill, EventTimeMS: ptrI64k(1000), ActorXUID: ptrS("A")},
		{EventType: analysis.EventTypeMedal, EventTimeMS: ptrI64k(2000), ActorXUID: ptrS("A"),
			MedalName: "Odin's Raven"},
		{EventType: analysis.EventTypeMedal, EventTimeMS: ptrI64k(3000), ActorXUID: ptrS("B"),
			MedalName: "Inconnue Totale"},
		{EventType: analysis.EventTypeMedal, EventTimeMS: ptrI64k(4000), ActorXUID: ptrS("B")},
	}
	repo := &mockMatchViewRepo{medalMetasByName: map[string]domain.MedalNameMeta{
		"Odin's Raven": {MedalNameID: 1512363953, Label: "Corbeau d'Odin", Description: "Desc"},
	}}
	adapter := &stubAssetURL{medalImg: "/static/medals/1512363953.png"}

	decorateMedalEvents(context.Background(), events, repo, adapter)

	res := events[1]
	if res.MedalNameID == nil || *res.MedalNameID != 1512363953 ||
		res.MedalLabel != "Corbeau d'Odin" || res.MedalDescription != "Desc" {
		t.Errorf("médaille résolue : %+v", res)
	}
	if res.MedalImageURL == "" {
		t.Errorf("médaille résolue sans visuel (adapter présent)")
	}
	if inc := events[2]; inc.MedalNameID != nil || inc.MedalLabel != "" || inc.MedalImageURL != "" {
		t.Errorf("médaille inconnue : décorée (%+v) alors que le référentiel ne la connaît pas", inc)
	}
	if anon := events[3]; anon.MedalNameID != nil || anon.MedalLabel != "" {
		t.Errorf("médaille sans nom brut : décorée (%+v)", anon)
	}
	if k := events[0]; k.MedalLabel != "" || k.MedalImageURL != "" {
		t.Errorf("un kill porte une identité de médaille (%+v)", k)
	}

	// Dégradations : repo nil, tranche vide — pas de panique.
	decorateMedalEvents(context.Background(), events, nil, nil)
	decorateMedalEvents(context.Background(), nil, repo, nil)
}

// TestKillFeedWeaponCoverage : le compteur ne regarde QUE les kills. Un feed de 4 events
// dont 3 kills et 1 médaille compte 3, jamais 4 — sinon le taux publié serait faux.
func TestKillFeedWeaponCoverage(t *testing.T) {
	events := feedFixture()
	events[0].WeaponImageURL = "/x.png"
	events[3].WeaponImageURL = "/y.png" // médaille : ne doit compter ni au numérateur ni au dénominateur
	avec, total := killFeedWeaponCoverage(events)
	if avec != 1 || total != 3 {
		t.Errorf("couverture = %d/%d, attendu 1/3", avec, total)
	}
}
