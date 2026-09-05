package replayartifacts

// bombstats.go — LES STATISTIQUES D'OBJECTIF DE L'ASSAUT, PERSISTEES AU FIL DE L'EAU.
//
// # CE QUE CE FICHIER FAIT, ET CE QU'IL NE FAIT PAS
//
// Chaque artefact CUIT DANS CE CYCLE porte deja ses statistiques d'Assaut : elles sont
// calculees A LA CUISSON (`replay.attachBombStats`, appele par `BuildFromPositions`), la ou les
// quatre sources vivent en pleine fidelite — chronologie de portage en MILLISECONDES, recalage
// film -> match, armements dates, actions d'objectif nommees. Ce fichier ne RECALCULE rien : il
// lit le document RANGE et transporte `doc.bombStats` / `doc.bombEvents` vers
// `persist.BombStatsPersister`.
//
// POURQUOI PAS UN CALCUL ICI. Le document publie le portage en FRAMES (grille de 100 ms), sans
// les periodes non pontees ni la distinction lacher/mort ; il ne publie ni le recalage
// d'horloge, ni la paire tueur/victime. Reconstruire `BombStatsInput` a partir de lui ferait un
// SECOND decodeur du meme fait, avec une precision moindre — exactement ce que l'en-tete de
// `replay/bomb_stats.go` condamne.
//
// # LA FORME, CALQUEE SUR usage.go (chantier session-usage)
//
//	SUR DISQUE, PAS LE BLOB   la projection lit l'artefact TEL QU'IL EST RANGE (StoreArtifact
//	                          peut REFUSER les octets candidats et garder le precedent).
//	PROJETER PUIS ECRIRE      toutes les lectures de fichier se font AVANT d'acquerir le writer.
//	SEGMENT WRITER COURT      acquis APRES toute cuisson, relache aussitot.
//	VIA `internal/persist`    INSERT-only (ADR 0019/0026/0030) ; ce fichier garde la LECTURE et
//	                          la DECISION.
//
// # LE GATE : UNE CAPABILITY, JAMAIS UN SLUG
//
// `film.bomb_stats` (games.CapFilmBombStats, capabilities.toml du titre). Halo 5 ne la declare
// pas — pas de decodeur de film, donc pas d'artefact : rien n'est produit, silence propre en
// DEBUG (ratchet no_slug_comparison_test.go).
//
// # LES TROIS SILENCES, ET ILS SE DISENT
//
//	film absent / artefact illisible   la projection echoue pour CE match : WARN + compteur ;
//	mode non-Assaut                    le document ne porte AUCUN `bombStats` (la garde de
//	                                   famille est a la cuisson) : rien a ecrire, DEBUG ;
//	capability absente                 rien n'est produit du tout, DEBUG.

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
)

// passeBombePrete : une projection réussie, prête à persister.
type passeBombePrete struct {
	matchID string
	batch   persist.BombStatsBatch
}

// capabilityBombeArmee dit si le titre déclare `film.bomb_stats`, et DIT POURQUOI quand la
// réponse est non — même contrat que capabilityUsageArmee : un TOML illisible est un INCIDENT
// (WARN + compteur d'échecs), une clé absente est une configuration de titre (DEBUG).
func capabilityBombeArmee(ctx context.Context, d Deps) (armee, incident bool) {
	caps, err := games.LoadCapabilityMap(d.RepoRoot, d.TitleSlug)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: stats d'Assaut non produites — capabilities illisibles",
			"gamertag", d.Gamertag, "titleSlug", d.TitleSlug, "err", err)
		return false, true
	}
	if !caps.Has(games.CapFilmBombStats) {
		slog.DebugContext(ctx, "post-sync: stats d'Assaut — titre sans la capability, rien à produire",
			"titleSlug", d.TitleSlug, "capability", string(games.CapFilmBombStats))
		return false, false
	}
	return true, false
}

// projeterStatsBombe lit UN artefact rangé et en tire la passe à écrire. Rend une passe VIDE
// (MatchID vide) quand il n'y a RIEN à écrire — document sans calque de bombe (le cas NORMAL de
// tout match hors famille bomb), ou film d'Assaut dont aucune source n'a rien rendu. Ni l'un
// ni l'autre n'est un défaut, et aucun ne doit se journaliser comme tel.
func projeterStatsBombe(matchID, path string) (persist.BombStatsBatch, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return persist.BombStatsBatch{}, fmt.Errorf("lecture artefact: %w", err)
	}
	var doc replay.ReplayDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return persist.BombStatsBatch{}, fmt.Errorf("parse artefact: %w", err)
	}
	if doc.BombStats == nil {
		return persist.BombStatsBatch{}, nil
	}
	b := bombBatchDuDocument(matchID, doc.BombStats, doc.BombEvents)
	if len(b.Players) == 0 && len(b.Events) == 0 {
		// FILM D'ASSAUT DONT AUCUNE SOURCE N'A RIEN RENDU (aucun pont slot->xuid, ou calque
		// d'armement retenu par la garde 2 ET aucune période de portage). Le persister
		// refuserait déjà d'écrire une passe vide — « écrire zéro ligne serait indistinguable
		// d'un match sans Assaut » —, mais il le dirait en WARN à chaque cycle. On l'écarte
		// ici, au même titre qu'un mode hors Assaut : ce n'est pas un défaut.
		return persist.BombStatsBatch{}, nil
	}
	return b, nil
}

// bombBatchDuDocument transporte le bloc du document vers la forme du persister. AUCUN calcul :
// les pointeurs voyagent tels quels — `nil` reste `nil`, et s'écrira NULL.
//
// LA PROVENANCE DE CHAQUE FAIT EST POSÉE ICI, et une seule fois : `match_objective_events`
// refuse un fait dont la `source` ou la `confidence` manque (« un fait qui ne dit pas d'où il
// vient laisse un lecteur lui prêter la précision qu'il veut »). Les deux valeurs viennent des
// constantes de `replay` — jamais un littéral de plus dans ce fichier.
func bombBatchDuDocument(matchID string, stats *replay.BombMatchStats,
	events []replay.BombEvent) persist.BombStatsBatch {
	out := persist.BombStatsBatch{MatchID: matchID}
	out.Players = make([]persist.BombPlayerStatsRow, 0, len(stats.Players))
	for _, p := range stats.Players {
		out.Players = append(out.Players, persist.BombPlayerStatsRow{
			XUID: p.XUID, Detonations: p.Detonations, Arms: p.Arms, Grabs: p.Grabs,
			TimeAsCarrierSeconds: p.TimeAsCarrierSeconds, CarriersKilled: p.CarriersKilled,
		})
	}
	out.Events = make([]persist.BombEventRow, 0, len(events))
	for _, e := range events {
		src, conf := replay.BombEventProvenance(e.Type)
		out.Events = append(out.Events, persist.BombEventRow{
			EventType: e.Type, TimeMS: e.TimeMS, XUID: e.XUID,
			Source: src, Confidence: conf,
		})
	}
	return out
}

// persisterStatsBombe projette puis écrit les statistiques d'Assaut des artefacts cuits du
// cycle. Best-effort de bout en bout : aucun échec ne remonte au cycle, aucun ne se tait.
func persisterStatsBombe(ctx context.Context, d Deps, rapports []artefactCuit) {
	if len(rapports) == 0 {
		return
	}
	titre := ctxkeys.TitleSlug(ctx)
	if armee, incident := capabilityBombeArmee(ctx, d); !armee {
		if incident {
			observability.AddIntT(titre, CompteurBombStatsEchecs, int64(len(rapports)))
		}
		return
	}
	prets, echecs := projeterStatsBombeDuLot(ctx, d, rapports)
	if len(prets) == 0 {
		// AUCUN match d'Assaut dans le lot est le cas NORMAL et majoritaire : on ne compte
		// que les échecs RÉELS, et le journal ne dit rien de plus.
		observability.AddIntT(titre, CompteurBombStatsEchecs, int64(echecs))
		return
	}
	if d.AcquireWriter == nil {
		slog.WarnContext(ctx, "post-sync: stats d'Assaut NON persistées (aucun writer shared câblé sur ce chemin)",
			"gamertag", d.Gamertag, "matchs", len(prets))
		observability.AddIntT(titre, CompteurBombStatsEchecs, int64(echecs+len(prets)))
		return
	}
	db, release, err := d.AcquireWriter(ctx)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: writer shared indisponible, stats d'Assaut non persistées",
			"gamertag", d.Gamertag, "matchs", len(prets), "err", err)
		observability.AddIntT(titre, CompteurBombStatsEchecs, int64(echecs+len(prets)))
		return
	}
	defer release()
	p := persist.NewBombStatsPersister(db)
	ecrits := 0
	for i := range prets {
		if err := p.PersistPass(ctx, prets[i].batch); err != nil {
			slog.ErrorContext(ctx, "post-sync: écriture des stats d'Assaut échouée",
				"match_id", prets[i].matchID, "err", err)
			echecs++
			continue
		}
		ecrits++
	}
	observability.AddIntT(titre, CompteurBombStatsEcrits, int64(ecrits))
	observability.AddIntT(titre, CompteurBombStatsEchecs, int64(echecs))
	slog.InfoContext(ctx, "post-sync: stats d'Assaut persistées",
		"gamertag", d.Gamertag, "ecrits", ecrits, "echecs", echecs)
}

// projeterStatsBombeDuLot projette tous les artefacts du lot, AVANT tout writer. Rend les
// passes NON VIDES et le compte d'échecs (déjà journalisés, un par match). Un match qui n'est
// pas de la famille bomb rend une passe vide : ce n'est pas un échec, c'est un silence attendu.
func projeterStatsBombeDuLot(ctx context.Context, d Deps, rapports []artefactCuit) ([]passeBombePrete, int) {
	prets := make([]passeBombePrete, 0, len(rapports))
	echecs := 0
	for _, r := range rapports {
		b, err := projeterStatsBombe(r.matchID, r.path)
		if err != nil {
			slog.WarnContext(ctx, "post-sync: artefact rangé mais stats d'Assaut illisibles",
				"gamertag", d.Gamertag, "match_id", r.matchID, "err", err)
			echecs++
			continue
		}
		if b.MatchID == "" {
			// PAS UN ÉCHEC, ET C EST LE CAS MAJORITAIRE : un match qui n est pas de la
			// famille bomb n a aucun calque de bombe au document. Le compter en défaut
			// noierait les vrais dans le bruit de chaque cycle.
			slog.DebugContext(ctx, "post-sync: stats d'Assaut — artefact sans calque de bombe (mode hors Assaut)",
				"match_id", r.matchID)
			continue
		}
		prets = append(prets, passeBombePrete{matchID: r.matchID, batch: b})
	}
	return prets, echecs
}
