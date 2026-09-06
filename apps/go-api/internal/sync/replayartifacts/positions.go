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
// # L'EQUIPE VIENT DE LA BASE, PAR LE XUID (correction du constat C5, revue A-R1)
//
// LE FILM NE PORTE PAS L'EQUIPE : `Track.Team` vaut -1 sur tout artefact produit par le
// decodeur d'aujourd'hui (`analysis/replay/build.go` la pose sans condition, et le roster le
// dit : « ce qu'il ne donne PAS, et que seule la base porte : l'equipe »). La projection ne
// l'invente donc pas — elle la JOINT, par le xuid que le document nomme sur chaque vie, contre
// `match_participants` : la meme jointure que celle que le client fait pour colorer un
// tableau. Un xuid absent (bot, vie que le fil des morts n'a pas nommee, match hors registre)
// reste a -1, valeur PLEINE que le lecteur sait lire.
//
// Ce que ce fichier a d'abord ecrit — « -1, la meme valeur non attribuee que l'ancien decodeur
// produisait » — etait FAUX, et c'est ce que la revue a releve : l'ancien decodeur appelait
// `positions.assignTeamsBestEffort`, qui attribuait 0/1 des qu'un ecart franc separait deux
// groupes sur l'axe X. Un DEVINEMENT SPATIAL, jamais l'equipe reelle — mais pas -1 non plus. La
// consequence mesurable etait le filtre Global / Equipe A / Equipe B de la carte de chaleur
// (`MatchPositionsHeatmap.tsx`), qui ne s'affiche que si au moins une position porte une equipe :
// il serait devenu du code mort pour toute donnee projetee.
//
// L'ARTEFACT PRIME QUAND IL PORTE L'EQUIPE : un titre dont le film la replique verra sa valeur
// transportee telle quelle, et la base ne servira qu'aux vies restees a -1.
//
// # CE QUE LA PROJECTION NE FAIT PAS
//
//	ELLE NE NOMME PAS LE JOUEUR  la table est MATCH-LEVEL par schema. Le xuid sert a poser
//	                             l'equipe puis il est jete : le publier changerait la forme
//	                             d'une table deja lue — hors decision 1, consigne en decouverte.
//	ELLE NE RECALE RIEN          `TimeMS` est sur l'axe du REJEU (frame x frameIntervalMs). La
//	                             carte de chaleur ne lit que x/y ; ecrire un axe faussement
//	                             presente comme celui du match serait pire qu'un axe assume.

import (
	"context"
	"database/sql"
	"log/slog"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
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

// EquipeInconnue : la valeur que porte une position dont on ne sait pas le camp. C'est une
// valeur PLEINE, pas un trou : le lecteur (`MatchPositionsHeatmap.tsx`) la reconnait et range
// ces positions dans le seul filtre « Global ».
const EquipeInconnue = -1

// passePositionsPrete : une projection prete a persister.
type passePositionsPrete struct {
	matchID string
	batch   persist.PlayerPositionsBatch
	// porteurs : le xuid de la vie dont vient CHAQUE ligne de `batch.Rows`, dans le meme
	// ordre. Il ne part JAMAIS en base — la table est match-level par schema. Il ne sert qu'a
	// poser l'equipe, qui vit en base (cf. l'en-tete), une fois le writer acquis. Vide pour
	// une vie que le fil des morts n'a pas nommee.
	porteurs []string
}

// projeterPositions tire d'UN document les positions a ecrire, decimees a [GrainPositionsMS].
//
// Rend une passe VIDE (matchID vide) quand le document ne porte aucune trajectoire — un
// artefact d'un mode non filme, ou une cuisson qui n'a rien trouve. Ce n'est pas un defaut.
//
// L'EQUIPE N'EST PAS RESOLUE ICI : le document ne la porte pas, et la resoudre demanderait la
// base — or cette fonction est PURE, et c'est ce qui permet de projeter tout le lot AVANT
// d'acquerir le moindre segment d'ecriture. Elle emporte le xuid de chaque ligne ;
// [appliquerEquipes] fait la jointure dans le segment court.
func projeterPositions(matchID string, doc *replay.ReplayDocument) passePositionsPrete {
	cadence := doc.FrameIntervalMS
	if cadence <= 0 {
		cadence = cadenceParDefautMS
	}
	rows := make([]persist.PlayerPositionRow, 0, 256)
	porteurs := make([]string, 0, 256)
	for i := range doc.Tracks {
		t := &doc.Tracks[i]
		lignes := positionsDeLaTrajectoire(t, cadence)
		rows = append(rows, lignes...)
		for range lignes {
			porteurs = append(porteurs, t.XUID)
		}
	}
	if len(rows) == 0 {
		return passePositionsPrete{}
	}
	return passePositionsPrete{
		matchID:  matchID,
		batch:    persist.PlayerPositionsBatch{MatchID: matchID, Rows: rows},
		porteurs: porteurs,
	}
}

// positionsDeLaTrajectoire decime UNE trajectoire : le premier point, puis un point des que
// [GrainPositionsMS] s'est ecoule depuis le precedent RETENU.
//
// LE PREMIER POINT EST TOUJOURS RETENU : une vie plus courte que le grain existe quand meme, et
// l'ecarter effacerait de la carte les joueurs qui meurent vite — precisement ceux dont les
// positions disent quelque chose.
//
// `Team` PREND CE QUE LE DOCUMENT PORTE, y compris -1 : un titre dont le film replique l'equipe
// prime sur la base (cf. l'en-tete). Les -1 sont remplis par [appliquerEquipes].
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

// appliquerEquipes pose l'EQUIPE de chaque position, lue en base par le xuid de son porteur.
//
// LA LECTURE SE FAIT SUR LE HANDLE WRITER, DANS LE MEME SEGMENT COURT — meme regle et meme
// raison que le report du coup d'envoi (t0film.go) : le segment de LECTURE est deja relache
// quand les derivations tournent, et `SharedAccess.Write` refuserait un burst avec un Read en
// vol. Un `port.ReplayFactsRepo`, pas une requete de plus : c'est deja LE lecteur de « ce que
// la base sait du match » pour le rejeu, camps compris.
//
// Rend le nombre de lignes effectivement situees — c'est ce que le journal publie. Une lecture
// qui echoue degrade CE match seul : ses positions restent a [EquipeInconnue], ce qui vaut
// exactement ce qu'elles valaient avant.
func appliquerEquipes(ctx context.Context, db *sql.DB, prets []passePositionsPrete) int {
	var repo port.ReplayFactsRepo = duckdbpkg.NewReplayFactsRepo(db)
	situees := 0
	for i := range prets {
		facts, err := repo.FactsForMatch(ctx, prets[i].matchID)
		if err != nil {
			slog.WarnContext(ctx, "post-sync: positions — camps illisibles, equipes non attribuees",
				"match_id", prets[i].matchID, "err", err)
			continue
		}
		equipes := make(map[string]int, len(facts.Players))
		for _, j := range facts.Players {
			if j.XUID != "" && j.TeamID >= 0 {
				equipes[j.XUID] = j.TeamID
			}
		}
		situees += poserEquipes(&prets[i], equipes)
	}
	return situees
}

// poserEquipes remplit les equipes INCONNUES d'une passe. Une ligne que le document a deja
// situee n'est jamais retouchee, et un xuid absent de la table reste a [EquipeInconnue].
func poserEquipes(p *passePositionsPrete, equipes map[string]int) int {
	n := 0
	for j := range p.batch.Rows {
		if p.batch.Rows[j].Team != EquipeInconnue {
			continue
		}
		if e, ok := equipes[p.porteurs[j]]; ok {
			p.batch.Rows[j].Team = e
			n++
		}
	}
	return n
}

// persisterPositions projette puis ecrit les positions des artefacts ranges du lot.
// Best-effort de bout en bout : aucun echec ne remonte au cycle, aucun ne se tait.
func persisterPositions(ctx context.Context, d Deps, b *bilanDerivations, lus []artefactLu) {
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
		echecPositions(b, prets)
		return
	}
	db, release, err := d.AcquireWriter(ctx)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: writer shared indisponible, positions non persistees",
			"gamertag", d.Gamertag, "matchs", len(prets), "err", err)
		observability.AddIntT(titre, CompteurPositionsEchecs, int64(len(prets)))
		echecPositions(b, prets)
		return
	}
	defer release()
	// L'EQUIPE, AVANT L'ECRITURE ET DANS LE MEME SEGMENT (constat C5) : le film ne la porte
	// pas, la base si, et le filtre Global / Equipe A / Equipe B de la carte de chaleur ne
	// s'affiche que si au moins une position en porte une.
	situees := appliquerEquipes(ctx, db, prets)
	p := persist.NewPlayerPositionsPersister(db)
	ecrits, echecs, lignes := 0, 0, 0
	for i := range prets {
		if err := p.PersistPass(ctx, prets[i].batch); err != nil {
			slog.ErrorContext(ctx, "post-sync: ecriture des positions echouee",
				"match_id", prets[i].matchID, "err", err)
			echecs++
			b.echec(prets[i].matchID)
			continue
		}
		ecrits++
		lignes += len(prets[i].batch.Rows)
	}
	observability.AddIntT(titre, CompteurPositionsEcrites, int64(ecrits))
	observability.AddIntT(titre, CompteurPositionsEchecs, int64(echecs))
	slog.InfoContext(ctx, "post-sync: positions persistees",
		"gamertag", d.Gamertag, "ecrits", ecrits, "echecs", echecs, "lignes", lignes,
		"lignes_situees", situees)
}

// echecPositions enregistre au bilan que ces passes n'ont pas ete persistees faute de writer :
// sans cette trace, la marque de derivation se poserait sur un match dont RIEN n'a ete ecrit
// (constat C1 de la revue A-R1).
func echecPositions(b *bilanDerivations, prets []passePositionsPrete) {
	b.writerIndisponible()
	for i := range prets {
		b.echec(prets[i].matchID)
	}
}

// projeterPositionsDuLot projette tous les documents du lot, AVANT tout writer. Rend les passes
// NON VIDES ; un artefact sans trajectoire rend une passe vide, ce qui n'est pas un echec.
func projeterPositionsDuLot(ctx context.Context, lus []artefactLu) []passePositionsPrete {
	prets := make([]passePositionsPrete, 0, len(lus))
	for _, a := range lus {
		p := projeterPositions(a.matchID, a.doc)
		if p.matchID == "" {
			slog.DebugContext(ctx, "post-sync: positions — artefact sans trajectoire",
				"match_id", a.matchID)
			continue
		}
		prets = append(prets, p)
	}
	return prets
}
