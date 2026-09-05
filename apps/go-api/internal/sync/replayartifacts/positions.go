package replayartifacts

// positions.go — LES POSITIONS JOUEURS, PROJETEES DE L'ARTEFACT (decision utilisateur 1).
//
// # CE QUE CE FICHIER FAIT, ET CE QU'IL REMPLACE
//
// `shared.match_player_positions` alimente la CARTE DE CHALEUR de la fiche de match. Elle etait
// remplie par un OUTIL DE DIAGNOSTIC (`cmd/diag_weapons_v3 -positions -write`), qui decodait
// lui-meme les positions keyframe du film et les ecrivait en DELETE-then-INSERT sur le handle de
// LECTURE du pool. Elle devient une PROJECTION DE L'ARTEFACT, ecrite par les derivations
// post-rangement comme le resume d'usage et les statistiques d'Assaut.
//
// Consequences, toutes voulues : un seul decodeur du film (celui de la cuisson), un seul regime
// d'ecriture (INSERT-only sous le lease), et des positions qui arrivent AU FIL DE L'EAU au lieu
// d'attendre qu'un operateur lance un outil.
//
// # LA CADENCE : CELLE QUE LA TABLE DECLARE, ET C'EST UNE MESURE
//
// Le document publie les trajectoires sur SA grille — une position par vie et par frame
// (100 ms). Les projeter telles quelles ecrirait 31 051 lignes PAR MATCH en moyenne sur le
// corpus local (mediane 29 167, maximum 129 096 — mesure du 2026-09-06 sur les 106 artefacts du
// cache). La table, elle, declare depuis sa creation une granularite de ~20 s (« snapshot
// type-2 », steps_shared_player_positions.go), et son unique lecteur binne en grille 20x20
// (`MatchPositionsHeatmap.tsx`) : au-dela de quelques centaines de points, chaque ligne de plus
// est du poids de base et de fil pour zero pixel.
//
// [GrainPositionsMS] fixe donc la cadence a la granularite DECLAREE de la table. Mesure du meme
// jour, meme corpus : 215 positions par match en moyenne, mediane 201, maximum 895 — l'ordre de
// grandeur de ce que la table portait. Ce n'est pas un reglage esthetique : c'est la seule
// valeur qui rende la projection equivalente a ce qu'elle remplace.
//
// # CE QUE LA PROJECTION NE FAIT PAS
//
//	ELLE N'INVENTE PAS D'EQUIPE  `Track.Team` vaut -1 dans le film (l'equipe vit en base, le
//	                             client la joint par xuid). La colonne recoit donc -1, la meme
//	                             valeur « non attribuee » que l'ancien decodeur produisait.
//	ELLE NE NOMME PAS LE JOUEUR  la table est MATCH-LEVEL par schema. Le document, lui, porte le
//	                             xuid de chaque vie : nommer le porteur est possible mais
//	                             changerait la forme d'une table deja lue — hors decision 1,
//	                             consigne en decouverte.
//	ELLE NE RECALE RIEN          `TimeMS` est sur l'axe du REJEU (frame x frameIntervalMs). La
//	                             carte de chaleur ne lit que x/y ; ecrire un axe faussement
//	                             presente comme celui du match serait pire qu'un axe assume.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
)

// GrainPositionsMS : l'ecart minimal entre deux positions retenues d'une MEME trajectoire.
//
// 20 000 ms = la granularite que le schema de la table declare (cf. l'en-tete). Le changer
// change le volume ecrit ; le mesurer avant est la seule facon de le changer honnetement.
const GrainPositionsMS = 20_000

// cadenceParDefautMS : la duree d'une frame quand le document ne la publie pas (schemas
// anterieurs au champ `frameIntervalMs`). 100 ms est la grille du rejeu depuis sa creation ;
// supposer autre chose ferait mentir l'axe de temps des vieux artefacts.
const cadenceParDefautMS = 100

// passePositionsPrete : une projection prete a persister.
type passePositionsPrete struct {
	matchID string
	batch   persist.PlayerPositionsBatch
}

// projeterPositions tire d'UN document les positions a ecrire, decimees a [GrainPositionsMS].
//
// Rend une passe VIDE (MatchID vide) quand le document ne porte aucune trajectoire — un
// artefact d'un mode non filme, ou une cuisson qui n'a rien trouve. Ce n'est pas un defaut.
func projeterPositions(matchID string, doc *replay.ReplayDocument) persist.PlayerPositionsBatch {
	cadence := doc.FrameIntervalMS
	if cadence <= 0 {
		cadence = cadenceParDefautMS
	}
	rows := make([]persist.PlayerPositionRow, 0, 256)
	for i := range doc.Tracks {
		rows = append(rows, positionsDeLaTrajectoire(&doc.Tracks[i], cadence)...)
	}
	if len(rows) == 0 {
		return persist.PlayerPositionsBatch{}
	}
	return persist.PlayerPositionsBatch{MatchID: matchID, Rows: rows}
}

// positionsDeLaTrajectoire decime UNE trajectoire : le premier point, puis un point des que
// [GrainPositionsMS] s'est ecoule depuis le precedent RETENU.
//
// LE PREMIER POINT EST TOUJOURS RETENU : une vie plus courte que le grain existe quand meme, et
// l'ecarter effacerait de la carte les joueurs qui meurent vite — precisement ceux dont les
// positions disent quelque chose.
func positionsDeLaTrajectoire(t *replay.Track, cadenceMS int) []persist.PlayerPositionRow {
	out := make([]persist.PlayerPositionRow, 0, 8)
	dernier := 0
	premier := true
	for _, p := range t.Points {
		ms := p.T * cadenceMS
		if !premier && ms-dernier < GrainPositionsMS {
			continue
		}
		out = append(out, persist.PlayerPositionRow{
			TimeMS: ms, X: p.X, Y: p.Y, Z: p.Z, Team: t.Team,
		})
		dernier, premier = ms, false
	}
	return out
}

// persisterPositions projette puis ecrit les positions des artefacts ranges du lot.
// Best-effort de bout en bout : aucun echec ne remonte au cycle, aucun ne se tait.
func persisterPositions(ctx context.Context, d Deps, lus []artefactLu) {
	if len(lus) == 0 {
		return
	}
	titre := ctxkeys.TitleSlug(ctx)
	prets := projeterPositionsDuLot(ctx, lus)
	if len(prets) == 0 {
		// AUCUNE TRAJECTOIRE DANS LE LOT : rare mais pas un defaut (mode non filme, cuisson
		// vide). Aucun writer n'est ouvert — la projection l'a vu avant, sur le document.
		return
	}
	if d.AcquireWriter == nil {
		slog.WarnContext(ctx, "post-sync: positions NON persistees (aucun writer shared cable sur ce chemin)",
			"gamertag", d.Gamertag, "matchs", len(prets))
		observability.AddIntT(titre, CompteurPositionsEchecs, int64(len(prets)))
		return
	}
	db, release, err := d.AcquireWriter(ctx)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: writer shared indisponible, positions non persistees",
			"gamertag", d.Gamertag, "matchs", len(prets), "err", err)
		observability.AddIntT(titre, CompteurPositionsEchecs, int64(len(prets)))
		return
	}
	defer release()
	p := persist.NewPlayerPositionsPersister(db)
	ecrits, echecs, lignes := 0, 0, 0
	for i := range prets {
		if err := p.PersistPass(ctx, prets[i].batch); err != nil {
			slog.ErrorContext(ctx, "post-sync: ecriture des positions echouee",
				"match_id", prets[i].matchID, "err", err)
			echecs++
			continue
		}
		ecrits++
		lignes += len(prets[i].batch.Rows)
	}
	observability.AddIntT(titre, CompteurPositionsEcrites, int64(ecrits))
	observability.AddIntT(titre, CompteurPositionsEchecs, int64(echecs))
	slog.InfoContext(ctx, "post-sync: positions persistees",
		"gamertag", d.Gamertag, "ecrits", ecrits, "echecs", echecs, "lignes", lignes)
}

// projeterPositionsDuLot projette tous les documents du lot, AVANT tout writer. Rend les passes
// NON VIDES ; un artefact sans trajectoire rend une passe vide, ce qui n'est pas un echec.
func projeterPositionsDuLot(ctx context.Context, lus []artefactLu) []passePositionsPrete {
	prets := make([]passePositionsPrete, 0, len(lus))
	for _, a := range lus {
		b := projeterPositions(a.matchID, a.doc)
		if b.MatchID == "" {
			slog.DebugContext(ctx, "post-sync: positions — artefact sans trajectoire",
				"match_id", a.matchID)
			continue
		}
		prets = append(prets, passePositionsPrete{matchID: a.matchID, batch: b})
	}
	return prets
}
