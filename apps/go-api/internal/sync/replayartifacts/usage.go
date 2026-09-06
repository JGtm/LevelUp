package replayartifacts

// usage.go — LE RÉSUMÉ D'USAGE ÉQUIPEMENT/SOCLES, PERSISTÉ AU FIL DE L'EAU.
//
// # CE QUE CE FICHIER FAIT
//
// Chaque artefact CUIT DANS CE CYCLE est projeté en résumé d'usage
// (replay.BuildUsageSummary — tractions de grappin, épisodes camo/surbouclier, poses,
// prises de socle) puis persisté dans `shared.match_usage_players` +
// `shared.match_usage_films` (chantier session-usage, décision utilisateur 2026-09-04 :
// « il faut les sauvegarder en BDD lors du sync »). C'est L'ÉTAPE QUI CONSTRUIT
// L'ARTEFACT qui porte le résumé : jamais un second décodage de film, jamais une autre
// passe — le document vient d'être rangé, on le projette.
//
// # LA FORME, CALQUÉE SUR LE REPORT DU COUP D'ENVOI (t0film.go)
//
//	SUR DISQUE, PAS LE BLOB   la projection lit l'artefact TEL QU'IL EST RANGÉ.
//	                          `StoreArtifact` peut REFUSER les octets candidats (garde
//	                          anti-régression) et conserver l'artefact précédent :
//	                          projeter le candidat écrirait en base un résumé que le
//	                          disque ne porte pas — même doctrine que lireT0FilmArtefact.
//	PROJETER PUIS ÉCRIRE      toutes les projections (lecture fichier + parse JSON) se
//	                          font AVANT d'acquérir le writer : jamais un handle partagé
//	                          tenu pendant une E/S disque évitable.
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
// Seuls les artefacts CUITS DANS CE CYCLE. Un artefact déjà à jour est sauté par
// `buildAll` sans être relu, et le chemin « ouvrier » ne cuit rien localement. Le corpus
// déjà sur disque relève du backfill CLI (`levelup backfill-usage-summary`), hors ligne
// et sous le contrôle de l'opérateur.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
)

// artefactCuit : un match dont l'artefact a été cuit ce cycle. Il alimente TOUTES les
// projections post-cuisson du paquet — le résumé d'usage (usage.go) ET les statistiques
// d'Assaut (bombstats.go) —, parce qu'elles lisent le MÊME fichier rangé : deux listes
// séparées auraient divergé au premier crochet ajouté.
type artefactCuit struct {
	matchID string
	// path : l'artefact RANGÉ SUR DISQUE (StoredArtifact.Path) — la seule source que la
	// projection accepte (voir l'en-tête).
	path string
}

// resumeUsagePret : une projection réussie, prête à persister.
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

// projeterResumeUsage lit UN artefact rangé et le projette. Une erreur est un échec de
// CE match, jamais du cycle.
func projeterResumeUsage(path string) (replay.UsageSummary, error) {
	doc, err := lireDocumentRange(path)
	if err != nil {
		return replay.UsageSummary{}, err
	}
	return replay.BuildUsageSummary(doc), nil
}

// persisterResumesUsage projette puis écrit les résumés des artefacts cuits du cycle.
// Best-effort de bout en bout, comme toute l'étape : aucun échec ne remonte au cycle,
// mais aucun ne se tait non plus.
func persisterResumesUsage(ctx context.Context, d Deps, rapports []artefactCuit) {
	if len(rapports) == 0 {
		return
	}
	titre := ctxkeys.TitleSlug(ctx)
	if armee, incident := capabilityUsageArmee(ctx, d); !armee {
		if incident {
			// Capabilities illisibles : le lot entier du cycle est écarté — un DÉFAUT,
			// compté comme tel (le WARN seul ne nourrit aucun monitoring).
			observability.AddIntT(titre, CompteurUsageEchecs, int64(len(rapports)))
		}
		return
	}
	prets, echecs := projeterResumesUsage(ctx, d, rapports)
	if len(prets) == 0 {
		observability.AddIntT(titre, CompteurUsageEchecs, int64(echecs))
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

// projeterResumesUsage projette tous les artefacts du lot, AVANT tout writer. Rend les
// projections réussies et le compte d'échecs (déjà journalisés, un par match).
func projeterResumesUsage(ctx context.Context, d Deps, rapports []artefactCuit) ([]resumeUsagePret, int) {
	prets := make([]resumeUsagePret, 0, len(rapports))
	echecs := 0
	for _, r := range rapports {
		s, err := projeterResumeUsage(r.path)
		if err != nil {
			slog.WarnContext(ctx, "post-sync: artefact rangé mais résumé d'usage impossible",
				"gamertag", d.Gamertag, "match_id", r.matchID, "err", err)
			echecs++
			continue
		}
		prets = append(prets, resumeUsagePret{matchID: r.matchID, summary: s})
	}
	return prets, echecs
}
