package replayartifacts

// usage.go — LE RÉSUMÉ D'USAGE ÉQUIPEMENT/SOCLES, PERSISTÉ AU FIL DE L'EAU.
//
// # CE QUE CE FICHIER FAIT
//
// Chaque artefact QUI VIENT D'ÊTRE RANGÉ est projeté en résumé d'usage
// (replay.BuildUsageSummary — tractions de grappin, épisodes camo/surbouclier, poses,
// prises de socle) puis persisté dans `shared.match_usage_players` +
// `shared.match_usage_films` (chantier session-usage, décision utilisateur 2026-09-04 :
// « il faut les sauvegarder en BDD lors du sync »). Jamais un second décodage de film,
// jamais une autre passe — le document vient d'être rangé, on le projette.
//
// LE DÉCLENCHEUR EST LE RANGEMENT, PAS LA CUISSON (constat A1, corrigé le 2026-09-06) :
// l'appelant est [Deriver] (derivations.go), qui est lui-même appelé par les DEUX
// rangeurs (cuisson locale et dépôt d'ouvrier) et par le rattrapage. Ce fichier ne lit
// plus le disque : il reçoit le document déjà lu et désérialisé UNE fois pour toutes les
// dérivations.
//
// # LA FORME, CALQUÉE SUR LE REPORT DU COUP D'ENVOI (t0film.go)
//
//	SUR DISQUE, PAS LE BLOB   le document vient de l'artefact TEL QU'IL EST RANGÉ (lu par
//	                          [Deriver]). `StoreArtifact` peut REFUSER les octets candidats
//	                          (garde anti-régression) et conserver l'artefact précédent :
//	                          projeter le candidat écrirait en base un résumé que le
//	                          disque ne porte pas.
//	PROJETER PUIS ÉCRIRE      toutes les projections se font AVANT d'acquérir le writer :
//	                          jamais un handle partagé tenu pendant un travail évitable.
//	SEGMENT WRITER COURT      acquis APRÈS toute cuisson, relâché aussitôt — même burst
//	                          borné (au plus maxPerCycle matchs) que le report du T0.
//	VIA `internal/persist`    l'écriture vit dans persist.UsageSummaryPersister,
//	                          INSERT-only (ADR 0019/0026/0030) ; ce fichier garde la
//	                          LECTURE et la DÉCISION.
//
// # LE GATE : UNE CAPABILITY, JAMAIS UN SLUG
//
// `film.usage_summary` (games.CapFilmUsageSummary, capabilities.toml du titre) gouverne
// la production. Un titre sans la clé — Halo 5 n'a pas de décodeur de film, donc pas
// d'artefact — ne produit RIEN, silence propre en DEBUG (ratchet
// no_slug_comparison_test.go). Le TOML est relu à chaque cycle QUI A CUIT quelque chose :
// c'est au plus une petite lecture par cycle, et elle suit les règles vivantes sans
// redémarrage (même esprit que Hook.Placement).
//
// # CE QUI N'EST PAS RÉSUMÉ ICI
//
// Les artefacts que personne ne range et que le rattrapage n'a pas encore atteints. Depuis
// le 2026-09-06 le rattrapage des dérivés existe (derivations_backlog.go) : le corpus
// déjà sur disque converge de lui-même, cinq artefacts par cycle. Le backfill CLI
// (`levelup backfill-usage-summary`) reste le moyen de le forcer d'un coup.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
)

// resumeUsagePret : une projection prête à persister.
type resumeUsagePret struct {
	matchID string
	summary replay.UsageSummary
}

// capabilityUsageArmee dit si le titre déclare `film.usage_summary`, et DIT POURQUOI
// quand la réponse est non : un TOML illisible est un INCIDENT (WARN, et `incident`
// vrai — le lot du cycle compte alors en échecs, c'est le contrat écrit de
// CompteurUsageEchecs, revue adversariale 2026-09-04) ; une clé absente est une
// configuration de titre (DEBUG, aucun compteur) — les deux se distinguent de
// « l'étape n'a jamais tourné ».
func capabilityUsageArmee(ctx context.Context, d Deps) (armee, incident bool) {
	caps, err := games.LoadCapabilityMap(d.RepoRoot, d.TitleSlug)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: résumé d'usage non produit — capabilities illisibles",
			"gamertag", d.Gamertag, "titleSlug", d.TitleSlug, "err", err)
		return false, true
	}
	if !caps.Has(games.CapFilmUsageSummary) {
		slog.DebugContext(ctx, "post-sync: résumé d'usage — titre sans la capability, rien à produire",
			"titleSlug", d.TitleSlug, "capability", string(games.CapFilmUsageSummary))
		return false, false
	}
	return true, false
}

// persisterResumesUsage projette puis écrit les résumés des artefacts rangés du lot.
// Best-effort de bout en bout, comme toute l'étape : aucun échec ne remonte au cycle,
// mais aucun ne se tait non plus.
func persisterResumesUsage(ctx context.Context, d Deps, lus []artefactLu) {
	if len(lus) == 0 {
		return
	}
	titre := ctxkeys.TitleSlug(ctx)
	if armee, incident := capabilityUsageArmee(ctx, d); !armee {
		if incident {
			// Capabilities illisibles : le lot entier du cycle est écarté — un DÉFAUT,
			// compté comme tel (le WARN seul ne nourrit aucun monitoring).
			observability.AddIntT(titre, CompteurUsageEchecs, int64(len(lus)))
		}
		return
	}
	prets := projeterResumesUsage(lus)
	echecs := 0
	if len(prets) == 0 {
		return
	}
	if d.AcquireWriter == nil {
		// DÉGRADATION VOULUE, PAS UN SILENCE : même cas que le report du T0 — un chemin
		// de sync sans writer câblé cuit ses artefacts mais ne résume rien.
		slog.WarnContext(ctx, "post-sync: résumé d'usage NON persisté (aucun writer shared câblé sur ce chemin)",
			"gamertag", d.Gamertag, "matchs", len(prets))
		observability.AddIntT(titre, CompteurUsageEchecs, int64(echecs+len(prets)))
		return
	}
	db, release, err := d.AcquireWriter(ctx)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: writer shared indisponible, résumé d'usage non persisté",
			"gamertag", d.Gamertag, "matchs", len(prets), "err", err)
		observability.AddIntT(titre, CompteurUsageEchecs, int64(echecs+len(prets)))
		return
	}
	defer release()
	p := persist.NewUsageSummaryPersister(db)
	ecrits := 0
	for i := range prets {
		if err := p.PersistPass(ctx, prets[i].matchID, &prets[i].summary); err != nil {
			slog.WarnContext(ctx, "post-sync: écriture du résumé d'usage échouée",
				"match_id", prets[i].matchID, "err", err)
			echecs++
			continue
		}
		ecrits++
	}
	observability.AddIntT(titre, CompteurUsageEcrits, int64(ecrits))
	observability.AddIntT(titre, CompteurUsageEchecs, int64(echecs))
	slog.InfoContext(ctx, "post-sync: résumé d'usage persisté",
		"gamertag", d.Gamertag, "ecrits", ecrits, "echecs", echecs)
}

// projeterResumesUsage projette tous les documents du lot, AVANT tout writer.
//
// AUCUN ÉCHEC POSSIBLE ICI depuis que [Deriver] lit et désérialise (2026-09-06) : un document
// illisible est écarté À LA LECTURE, avec son journal, et n'arrive jamais jusqu'ici.
// `BuildUsageSummary` est une fonction pure sur un document déjà en mémoire.
func projeterResumesUsage(lus []artefactLu) []resumeUsagePret {
	prets := make([]resumeUsagePret, 0, len(lus))
	for _, a := range lus {
		prets = append(prets, resumeUsagePret{matchID: a.matchID, summary: replay.BuildUsageSummary(a.doc)})
	}
	return prets
}
