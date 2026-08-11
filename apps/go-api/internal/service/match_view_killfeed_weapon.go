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

// decorateKillFeed pose l'équipe du tueur et l'arme du kill sur les events du feed.
//
// Modifie la tranche EN PLACE (les events sont déjà dans l'onglet assemblé). Tolère
// chaque entrée absente : scoreboard vide, sources vides, adapter nil. Aucun de ces cas
// n'est une erreur — ce sont les états nominaux d'un titre sans décodeur de film.
func decorateKillFeed(
	ctx context.Context,
	events []domain.MatchHighlightEvent,
	sources []domain.KillSourceRaw,
	scoreboard []domain.ScoreboardRaw,
	assetURL games.TitleAssetURLAdapter,
) {
	if len(events) == 0 {
		return
	}
	defer func() {
		avec, total := killFeedWeaponCoverage(events)
		slog.DebugContext(ctx, "match_view: couverture arme du kill feed",
			"kills", total, "avec_icone", avec, "sources_appariables", len(sources))
	}()
	teamByXUID := make(map[string]int, len(scoreboard))
	for _, r := range scoreboard {
		if r.TeamID != nil {
			teamByXUID[r.XUID] = *r.TeamID
		}
	}
	tagByKill := make(map[killFeedKey]uint32, len(sources))
	for _, s := range sources {
		tagByKill[killFeedKey{xuid: s.XUID, timeMS: s.TimeMS}] = s.SourceTag
	}

	for i := range events {
		e := &events[i]
		if e.ActorXUID == nil {
			continue
		}
		if team, ok := teamByXUID[*e.ActorXUID]; ok {
			t := team
			e.ActorTeamID = &t
		}
		if e.EventType != analysis.EventTypeKill || e.EventTimeMS == nil || assetURL == nil {
			continue
		}
		tag, ok := tagByKill[killFeedKey{xuid: *e.ActorXUID, timeMS: *e.EventTimeMS}]
		if !ok {
			continue
		}
		icon, ok := assetURL.KillSourceIcon(tag)
		if !ok {
			continue
		}
		e.WeaponKey = icon.WeaponKey
		e.WeaponLabel = icon.Label
		e.WeaponImageURL = icon.ImageURL
		e.WeaponImageTinted = icon.Tinted
	}
}

// killFeedWeaponCoverage compte, sur un feed décoré, les kills qui portent une icône
// d'arme et ceux qui n'en portent pas.
//
// Ce compteur EXISTE POUR ÊTRE LU : le taux de couverture de cette surface ne dépend pas
// du code mais de l'état d'avancement du décodage de film, qui bouge dans le temps (et
// qui, au 2026-08-11, est nul sur les matchs d'avril à juillet 2026). Sans mesure, une
// chute de couverture ressemblerait à une régression de rendu.
func killFeedWeaponCoverage(events []domain.MatchHighlightEvent) (avecIcone, total int) {
	for _, e := range events {
		if e.EventType != analysis.EventTypeKill {
			continue
		}
		total++
		if e.WeaponImageURL != "" {
			avecIcone++
		}
	}
	return avecIcone, total
}
