// Package service — match_view_killfeed_weapon.go : L'ARME DU KILL AU KILL FEED.
//
// CE QUE CE FICHIER FAIT, ET CE QU'IL NE FAIT PAS. Il pose sur chaque event `kill` de la
// carte Dominance deux décorations : l'ÉQUIPE du tueur (pour la couleur) et l'ARME qui
// l'a tué (pour l'icône). Il ne calcule rien : ni les bins, ni les vagues, ni les cumuls
// — aucun de ces calculs ne dépend de l'arme, et c'est pour ça que la décoration arrive
// APRÈS l'assemblage de l'onglet.
//
// L'APPARIEMENT, ET SON SEUL POINT DUR. `highlight_events` (le feed) et
// `match_kill_events` (le film) sont deux tables distinctes, écrites par deux producteurs
// distincts. Elles s'apparient par (tueur, instant) — mesuré sur la base de production :
// 152 009 kills de feed, appariement exact et SANS AUCUNE contradiction de source. Q21b a
// déjà écarté en amont les instants où deux morts simultanées du même tueur ne
// s'accordent pas sur l'arme.
//
// LES TROUS SONT UN RÉSULTAT, PAS UN BUG — et ils viennent presque tous de la DONNÉE, pas
// du pont. Mesure du 2026-08-11 sur les 152 009 kills du feed :
//   - 34,3 % ont une source de dégât connue et non contradictoire. De celles-là, le pont
//     en habille **98,0 %** : le résidu est le seul cas où la source désigne plusieurs
//     objets possibles (nom alternatif contradictoire, effet partagé par plusieurs
//     châssis, bidon dont la banque ne dit que le type d'énergie, chute confondue avec
//     l'environnement). Soit **33,7 % du feed** avec une icône.
//   - 47,1 % : le match a une passe de film, mais elle n'est pas publiable ligne par
//     ligne (BTB, marge de bijection nulle) ou n'a pas mesuré la source. Publier l'arme
//     y serait juste en agrégat et faux sur la ligne affichée.
//   - 18,5 % : le match n'a aucune ligne de film (jamais passé au décodeur).
//
// Dans les trois cas le feed affiche le kill SANS icône. Une icône absente est un repli ;
// une icône fausse est un mensonge, et un mensonge sur l'arme d'un kill est indétectable
// à l'œil de celui qui le lit.
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

// killFeedKey identifie une mort par son tueur et son instant. C'est la seule clé que les
// deux tables partagent : `highlight_events` ne porte pas d'identifiant de mort.
type killFeedKey struct {
	xuid   string
	timeMS int64
}

// killFeedInputs regroupe les sources de décoration du feed. Un struct, pas des
// paramètres : la décoration a gagné la VICTIME (killer_victim_pairs) et la signature
// dépassait la règle des 5 paramètres. Toute tranche peut être vide, l'adapter nil —
// ce sont les états nominaux d'un titre sans décodeur de film.
type killFeedInputs struct {
	sources []domain.KillSourceRaw
	assists []domain.KillAssistRaw
	// victims : paires killer→victim (Q20) DÉJÀ corrigées T0 (kvPairsFeed) — la clé
	// (tueur, instant) doit vivre dans le même référentiel que les events corrigés.
	victims    []domain.KVPairRaw
	scoreboard []domain.ScoreboardRaw
	assetURL   games.TitleAssetURLAdapter
}

// victimRef : la victime d'une clé (tueur, instant), et le conflit éventuel — deux
// victimes distinctes sur la même clé (double kill au même millisecond) n'en nomment
// AUCUNE, exactement la règle de Q21b pour l'arme.
//
// xuid PEUT ÊTRE VIDE : c'est une victime BOT (cf. doctrine domain.KVPairRaw). Le feed
// NOMME au lieu d'agréger, donc l'absence de xuid ne lui retire rien — le gamertag suffit
// à écrire la ligne. C'est la seule surface qui exploite une paire à xuid vide.
type victimRef struct {
	xuid     string
	gamertag string
	conflict bool
}

// memeVictime dit si une paire désigne la victime DÉJÀ retenue sur cette clé.
//
// Deux xuid renseignés se comparent par xuid — l'identité forte. Dès qu'un des deux
// manque (victime bot), il ne reste que le nom : le comparer est le seul moyen de
// distinguer deux bots tués au même instant par le même tueur. Comparer un xuid vide à
// un xuid vide déclarerait « même victime » pour deux bots différents, et le feed
// nommerait le mauvais.
func memeVictime(v *victimRef, kv *domain.KVPairRaw) bool {
	if v.xuid != "" && kv.VictimXUID != "" {
		return v.xuid == kv.VictimXUID
	}
	return v.gamertag == kv.VictimGT
}

// victimsByKill indexe les paires killer→victim par clé de mort, avec la garde
// d'unanimité.
//
// Le TUEUR bot (KillerXUID vide) reste écarté : la clé d'appariement est son xuid, et le
// feed ne porte aucun event de kill pour lui — indexer sous la clé vide ne pourrait que
// polluer les kills d'un joueur dont le xuid manquerait. La VICTIME bot, elle, entre dès
// qu'elle est nommée.
func victimsByKill(pairs []domain.KVPairRaw) map[killFeedKey]*victimRef {
	out := make(map[killFeedKey]*victimRef, len(pairs))
	for i := range pairs {
		kv := &pairs[i]
		if kv.KillerXUID == "" {
			continue
		}
		if kv.VictimXUID == "" && kv.VictimGT == "" {
			// Ni xuid ni nom : rien à afficher, et l'indexer masquerait une vraie victime
			// sur la même clé en la faisant passer pour un conflit.
			continue
		}
		key := killFeedKey{xuid: kv.KillerXUID, timeMS: kv.TimeMS}
		if v, ok := out[key]; ok {
			if !memeVictime(v, kv) {
				v.conflict = true
			}
			continue
		}
		out[key] = &victimRef{xuid: kv.VictimXUID, gamertag: kv.VictimGT}
	}
	return out
}

// decorateKillFeed pose l'équipe du tueur, l'arme du kill, l'ASSISTANCE et la VICTIME
// sur les events du feed.
//
// Modifie la tranche EN PLACE (les events sont déjà dans l'onglet assemblé). Tolère
// chaque entrée absente — aucun de ces cas n'est une erreur. Un kill sans entrée
// d'assistance appariée garde AssistState vide : ON NE SAIT PAS, et cet état-là ne
// s'écrit jamais « pas d'assistant ». Un kill sans paire appariée (ou à paire
// contradictoire) reste sans victime nommée.
func decorateKillFeed(ctx context.Context, events []domain.MatchHighlightEvent, in killFeedInputs) {
	if len(events) == 0 {
		return
	}
	defer func() {
		avec, avecHeadshot, total := killFeedWeaponCoverage(events)
		slog.DebugContext(ctx, "match_view: couverture arme du kill feed",
			"kills", total, "avec_icone", avec, "avec_headshot_connu", avecHeadshot,
			"sources_appariables", len(in.sources),
			"assists_appariables", len(in.assists), "victimes_appariables", len(in.victims))
	}()
	teamByXUID := make(map[string]int, len(in.scoreboard))
	for _, r := range in.scoreboard {
		if r.TeamID != nil {
			teamByXUID[r.XUID] = *r.TeamID
		}
	}
	// sourceByKill : la ligne ENTIÈRE (arme + headshot), pas seulement le tag — les deux
	// voyagent ensemble depuis Q21b (même garde d'unanimité, cf. domain.KillSourceRaw) et
	// Headshot se peuple INDÉPENDAMMENT de la résolution d'icône ci-dessous.
	sourceByKill := make(map[killFeedKey]domain.KillSourceRaw, len(in.sources))
	for _, s := range in.sources {
		sourceByKill[killFeedKey{xuid: s.XUID, timeMS: s.TimeMS}] = s
	}
	assistByKill := make(map[killFeedKey]*domain.KillAssistRaw, len(in.assists))
	for i := range in.assists {
		a := &in.assists[i]
		assistByKill[killFeedKey{xuid: a.XUID, timeMS: a.TimeMS}] = a
	}
	victimByKill := victimsByKill(in.victims)

	for i := range events {
		e := &events[i]
		if e.ActorXUID == nil {
			continue
		}
		if team, ok := teamByXUID[*e.ActorXUID]; ok {
			t := team
			e.ActorTeamID = &t
		}
		if e.EventType != analysis.EventTypeKill || e.EventTimeMS == nil {
			continue
		}
		key := killFeedKey{xuid: *e.ActorXUID, timeMS: *e.EventTimeMS}
		if v, ok := victimByKill[key]; ok && !v.conflict {
			decorateVictim(e, v, teamByXUID)
		}
		if a, ok := assistByKill[key]; ok {
			decorateAssist(e, a, teamByXUID)
		}
		src, ok := sourceByKill[key]
		if !ok {
			continue
		}
		// Headshot : posé dès que la source est connue et non ambiguë — PAS conditionné à la
		// résolution d'icône ni à assetURL (le tir à la tête ne dépend d'aucune table de noms).
		headshot := src.Headshot
		e.Headshot = &headshot
		if in.assetURL == nil {
			continue
		}
		icon, ok := in.assetURL.KillSourceIcon(src.SourceTag)
		if !ok {
			continue
		}
		e.WeaponKey = icon.WeaponKey
		e.WeaponLabel = icon.Label
		e.WeaponImageURL = icon.ImageURL
		e.WeaponImageTinted = icon.Tinted
	}
}

// decorateVictim pose la victime d'UN kill : son xuid, son gamertag (celui de la paire,
// jamais complété par une supposition) et son équipe si le scoreboard la connaît.
//
// CHAQUE CHAMP EST POSÉ SÉPARÉMENT, ET C'EST LE POINT. Une victime BOT n'a pas de xuid
// et n'apparaît dans aucun scoreboard : lui poser un `VictimXUID` de chaîne vide
// donnerait au front un identifiant qui ressemble à un joueur (il croiserait les marques
// et les équipes sur cette clé-là), et une équipe résolue sur la clé "" serait celle du
// premier venu. Le nom, lui, est toujours vrai — c'est tout ce que le feed a besoin
// d'écrire.
func decorateVictim(e *domain.MatchHighlightEvent, v *victimRef, teamByXUID map[string]int) {
	if v.xuid != "" {
		x := v.xuid
		e.VictimXUID = &x
		if team, ok := teamByXUID[v.xuid]; ok {
			t := team
			e.VictimTeamID = &t
		}
	}
	if v.gamertag != "" {
		gt := v.gamertag
		e.VictimGamertag = &gt
	}
}

// decorateAssist pose l'assistance d'UN kill : l'état, l'assistant s'il est nommé, son
// équipe si le scoreboard la connaît, et les parts de dégâts telles qu'elles sont
// mesurées — jamais complétées.
func decorateAssist(e *domain.MatchHighlightEvent, a *domain.KillAssistRaw, teamByXUID map[string]int) {
	e.KillerDamagePct = a.KillerDamagePct
	if a.AssistGamertag == nil {
		e.AssistState = domain.AssistStateNone
		return
	}
	e.AssistState = domain.AssistStateNamed
	e.AssistGamertag = *a.AssistGamertag
	e.AssistDamagePct = a.AssistDamagePct
	if a.AssistXUID != nil {
		if team, ok := teamByXUID[*a.AssistXUID]; ok {
			t := team
			e.AssistTeamID = &t
		}
	}
}

// killFeedWeaponCoverage compte, sur un feed décoré, les kills qui portent une icône
// d'arme, ceux dont le headshot est CONNU (Headshot non nil — vrai ou faux, cf. doctrine
// domain.MatchHighlightEvent.Headshot) et le total.
//
// Ce compteur EXISTE POUR ÊTRE LU : le taux de couverture de cette surface ne dépend pas
// du code mais de l'état d'avancement du décodage de film, qui bouge dans le temps (et
// qui, au 2026-08-11, est nul sur les matchs d'avril à juillet 2026). Sans mesure, une
// chute de couverture ressemblerait à une régression de rendu.
//
// avecHeadshot COMPTE PLUS LARGE que avecIcone (G.1) : le headshot se lit dès que la
// source est non ambiguë, sans dépendre de la résolution d'icône (cf. decorateKillFeed).
func killFeedWeaponCoverage(events []domain.MatchHighlightEvent) (avecIcone, avecHeadshot, total int) {
	for _, e := range events {
		if e.EventType != analysis.EventTypeKill {
			continue
		}
		total++
		if e.WeaponImageURL != "" {
			avecIcone++
		}
		if e.Headshot != nil {
			avecHeadshot++
		}
	}
	return avecIcone, avecHeadshot, total
}
