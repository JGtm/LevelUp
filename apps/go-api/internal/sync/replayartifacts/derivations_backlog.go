package replayartifacts

// derivations_backlog.go — LE RATTRAPAGE DES DERIVES (constat A2 du registre v2).
//
// # Le defaut que ce fichier ferme
//
// Les derivations ne s'appliquaient qu'aux artefacts CUITS DANS LE CYCLE. Un artefact deja
// range dont le resume d'usage, les statistiques d'Assaut ou le coup d'envoi n'avaient jamais
// ete ecrits n'etait jamais resélectionné : le rattrapage de cuisson (backlog.go) tient pour
// « fait » tout match dont le FICHIER existe, et la boucle de cuisson saute sans le relire tout
// artefact deja a jour. Les derives du corpus n'avaient donc aucun chemin de reprise autre
// qu'un backfill d'operateur.
//
// # LE PREDICAT DE FRAICHEUR EST LA MARQUE, PAS LE FICHIER
//
// « Deja derive » se lit dans l'INDEX DES DERIVATIONS (`replaybuild.DerivationsUpToDate` :
// artefact present, marque presente, meme revision, meme taille), jamais dans la seule presence
// de l'artefact. C'est ce qui fait CONVERGER le rattrapage : un artefact traite est marque, donc
// il sort de la liste ; sans marque, les memes cinq artefacts reviendraient a chaque cycle.
//
// # CE QU'IL NE FAIT PAS : IL NE CUIT RIEN
//
// Il ne selectionne QUE des artefacts DEJA RANGES, et il n'appelle jamais la cuisson. Un
// artefact PERIME (schema anterieur) n'entre donc pas ici au titre de sa peremption : la
// re-cuisson du corpus local (106 artefacts sur 9 versions de schema) est un ARBITRAGE
// UTILISATEUR DATE (« la recuisson attendra », 2026-09-02, registre des reports l. 17), et ce
// lot ne le renverse pas. Un artefact perime est derive TEL QU'IL EST — ses derives valent ce
// que vaut son schema, ce qui est strictement mieux que rien, et la re-cuisson les remplacera
// le jour venu (nouvelle taille -> marque invalidee -> derivations rejouees).
//
// # LES DEUX BORNES SONT CELLES QUI EXISTENT DEJA
//
//	horizon de lecture  [BacklogHorizon] (64) matchs les plus recents de la fenetre de
//	                    retention — la meme requete que le rattrapage de cuisson.
//	plafond de travail  [maxPerCycle] (5) artefacts derives par cycle. Une derivation coute une
//	                    lecture de document et un burst writer court, pas un decodage : le
//	                    plafond est ici une PRUDENCE, pas une necessite de duree.

import (
	"context"
	"database/sql"
	"log/slog"

	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/replaybuild"
)

// rattraperDerivations rejoue les derivations des artefacts ranges dont les derives manquent.
//
// Best-effort de bout en bout, comme toute l'etape : aucune erreur ne remonte au cycle, aucune
// ne se tait. Un cycle sans segment de lecture cable ne fait rien (la selection est impossible).
func rattraperDerivations(ctx context.Context, d Deps) {
	if d.WithRead == nil {
		return
	}
	var candidats []ArtefactRange
	var restant int
	d.WithRead(ctx, "replay_derivations_backlog", func(sharedDB *sql.DB) {
		candidats, restant = candidatsDerivations(ctx, sharedDB, d)
	})
	titre := ctxkeys.TitleSlug(ctx)
	// JAUGE PUBLIEE MEME A ZERO : « tout est derive » et « le rattrapage ne tourne pas »
	// s'ecriraient autrement pareil, c'est-a-dire rien (meme regle que CompteurRetard).
	observability.SetIntT(titre, JaugeDerivationsRetard, int64(restant))
	if len(candidats) == 0 {
		return
	}
	slog.InfoContext(ctx, "post-sync: rejeu 2D — rattrapage des derives",
		"gamertag", d.Gamertag, "artefacts", len(candidats), "restant", restant)
	observability.AddIntT(titre, CompteurDerivationsRattrapees, int64(len(candidats)))
	Deriver(ctx, DerivationsDeps{
		RepoRoot: d.RepoRoot, TitleSlug: d.TitleSlug, Gamertag: d.Gamertag,
		AcquireWriter: d.AcquireWriter,
	}, candidats)
}

// candidatsDerivations lit l'horizon du registre et rend les artefacts RANGES dont les derives
// ne sont pas a jour, plafonnes a [maxPerCycle], plus le reste a faire APRES ce cycle.
//
// L'ORDRE EST CELUI DU REGISTRE (du plus recent au plus vieux), pour la meme raison que partout
// ailleurs : ce sont les matchs que l'utilisateur regarde.
func candidatsDerivations(ctx context.Context, sharedDB *sql.DB, d Deps) (work []ArtefactRange, restant int) {
	ids := lireHorizonRegistre(ctx, sharedDB, d)
	paths := titlePkg.NewPathResolver(d.RepoRoot)
	for _, id := range ids {
		p := paths.ReplayArtifactPath(d.TitleSlug, id)
		if !artefactPresent(p) {
			// AUCUN ARTEFACT : c'est le travail de la cuisson (backlog.go), pas le notre. On
			// ne compte meme pas ce match en retard de derivation — il n'a rien a deriver.
			continue
		}
		if replaybuild.DerivationsUpToDate(p) {
			continue
		}
		if len(work) < maxPerCycle {
			work = append(work, ArtefactRange{MatchID: id, Path: p})
			continue
		}
		restant++
	}
	return work, restant
}

// lireHorizonRegistre rend les identifiants de la queue recente du registre — MEME REQUETE que
// le rattrapage de cuisson (`requeteQueueRecente`), et c'est voulu : deux requetes pour la meme
// question auraient diverge au premier ajustement de la fenetre ou du marqueur terminal.
func lireHorizonRegistre(ctx context.Context, sharedDB *sql.DB, d Deps) []string {
	args := []any{bitFilmAbsent}
	if d.RetentionMonths > 0 {
		args = append(args, fenetreRetention(d.RetentionMonths))
	}
	rows, err := sharedDB.QueryContext(ctx, requeteQueueRecente(d.RetentionMonths),
		append(args, BacklogHorizon)...)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: rejeu 2D — horizon des derives illisible", "err", err)
		return nil
	}
	defer func() { _ = rows.Close() }()

	var out []string
	for rows.Next() {
		var id string
		var rawName, mapID sql.NullString
		if err := rows.Scan(&id, &rawName, &mapID); err != nil {
			slog.WarnContext(ctx, "post-sync: rejeu 2D — horizon des derives (scan)", "err", err)
			return out
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		slog.WarnContext(ctx, "post-sync: rejeu 2D — horizon des derives (rows)", "err", err)
	}
	return out
}
