package main

// cmd_backfill_replay_repair.go — LE MODE `--repair-impoverished` : REPARER LE CACHE APPAUVRI.
//
// # LE TROU QU'IL FERME
//
// Le garde de fraicheur post-sync (`replayartifacts.etatArtefact`) ne re-enfile / ne reconstruit un
// artefact APPAUVRI (cuit sans les faits de match -> `scoreTimeline.players` vide) que pour les
// matchs INSERES du cycle courant. Un artefact deja appauvri present dans le cache n'est donc
// reparable par AUCUN chemin — il n'est jamais re-vu. Des la premiere passe d'ouvrier sans faits, le
// cache s'empoisonne A DEMEURE (dette `PLAN_OUVRIER_DISTANT.md` §5ter, ouverte AVANT activation).
//
// # CE N'EST PAS UN SECOND CHEMIN DE CUISSON — C'EST UNE SELECTION
//
// Ce fichier ne decode RIEN. Il ne fait que changer QUELS artefacts existants le parent re-cuit :
// l'ENFANT (`cmd_backfill_replay_child.go`), `Builder.BuildMatch` et le garde anti-regression
// `writeArtifactBytes` sont INTACTS. L'enfant lit deja ses faits et construit avec ; le garde au
// point d'ecriture unique interdit deja toute retrogradation. Le mode ne peut donc QU'AMELIORER.
//
// # LE PREDICAT, IDENTIQUE A CELUI DU POST-SYNC (`replayartifacts.etatArtefact`)
//
// On re-cuit un artefact SI ET SEULEMENT SI il est au schema COURANT, SANS compteurs de joueur, ET
// que la base porte des lignes de match (`len(facts.Players) > 0`). Les trois vacuites LEGITIMES
// (film sans entites, appariement ambigu, aucun compteur dans la fenetre — cf. l'en-tete
// d'`ArtifactHasPlayerCounters`) restent SAUTEES quand la base n'a pas de joueurs : re-cuire ne
// donnerait rien, et decoder pour rien est precisement ce que le blindage memoire cherche a eviter.
//
// # LES FAITS SE LISENT PAR LE MEME PORT QUE L'OUVRIER, EN LECTURE COURTE
//
// `duckdb.NewReplayFactsRepo` + `FactsForMatch` : le MEME comptage que celui que l'enfant obtiendra,
// donc zero divergence. La base shared est ouverte en LECTURE et RELACHEE avant tout decodage — on ne
// tient jamais un handle partage pendant les dizaines de secondes que coute un film.

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sort"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/replaybuild"
)

// etatReparation : ce que vaut, pour le mode reparation, l'artefact deja sur disque d'un match.
type etatReparation int

const (
	// reparationACuire : a jour, SANS compteurs de joueur, ET la base a des lignes — a re-cuire.
	reparationACuire etatReparation = iota
	// reparationDejaComplet : a jour AVEC compteurs de joueur — rien a faire.
	reparationDejaComplet
	// reparationVacuiteLegitime : a jour, sans compteurs, mais la base n'a AUCUN joueur — re-cuire
	// ne donnerait rien (une des trois vacuites legitimes). Saute.
	reparationVacuiteLegitime
	// reparationHorsSchema : artefact d'une version anterieure — domaine de `--only-existing`
	// ordinaire d'apres bump, PAS de ce mode.
	reparationHorsSchema
	// reparationSansArtefact : aucun artefact sur disque — rien a reparer.
	reparationSansArtefact
)

// classerReparation dit ce qu'il faut faire de l'artefact au chemin donne.
//
// `joueursEnBase` est APPELE PARESSEUSEMENT : on ne lit la base que pour un artefact deja juge a jour
// ET appauvri — un artefact riche, hors schema ou absent ne coute aucune lecture DuckDB.
func classerReparation(path string, joueursEnBase func() int) etatReparation {
	if _, err := os.Stat(path); err != nil {
		return reparationSansArtefact
	}
	if !replaybuild.ArtifactUpToDate(path) {
		return reparationHorsSchema
	}
	if replaybuild.ArtifactHasPlayerCounters(path) {
		return reparationDejaComplet
	}
	if joueursEnBase() > 0 {
		return reparationACuire
	}
	return reparationVacuiteLegitime
}

// selectionnerReparables ventile les candidats et rend le lot a re-cuire, trie et borne.
//
// PUR (le resolveur `joueursDe` est injecte) : testable sans base. Le wrapper `passeReparation` lui
// fournit la lecture DuckDB reelle.
func selectionnerReparables(
	candidats []replayCandidat, pr *titlePkg.PathResolver, o replayBackfillOptions,
	report *replayBackfillReport, joueursDe func(matchID string) int,
) []replayCandidat {
	aFaire := make([]replayCandidat, 0, len(candidats))
	for _, c := range candidats {
		path := pr.ReplayArtifactPath(o.titleSlug, c.matchID)
		switch classerReparation(path, func() int { return joueursDe(c.matchID) }) {
		case reparationACuire:
			aFaire = append(aFaire, c)
		case reparationDejaComplet:
			report.dejaComplets++
		case reparationVacuiteLegitime:
			report.vacuitesLegitimes++
		case reparationHorsSchema:
			report.horsSchemaCourant++
		case reparationSansArtefact:
			report.sansArtefact++
		}
	}
	return trierEtBornerReplay(aFaire, o.limit)
}

// passeReparation : le wrapper DuckDB du mode reparation. Ouvre la base shared en LECTURE COURTE,
// relachee AVANT tout decodage (defer), et fournit a la selection le comptage des joueurs en base.
//
// LA BASE EST ESSENTIELLE A LA DECISION (distinguer appauvri-reparable de vacuite legitime) : son
// ouverture en echec est une ERREUR DURE, jamais une degradation silencieuse qui masquerait un cache
// empoisonne (contrairement a metadata, best-effort ailleurs).
func passeReparation(
	ctx context.Context, cfg *config.AppConfig, candidats []replayCandidat,
	o replayBackfillOptions, report *replayBackfillReport,
) ([]replayCandidat, error) {
	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	sharedPath := pr.SharedDBPath(o.titleSlug)
	db, release, err := duckdb.OpenReadForQuery(sharedPath)
	if err != nil {
		return nil, fmt.Errorf("mode reparation : open shared RO (%s): %w (serveur en ecriture ? reessayer)", sharedPath, err)
	}
	defer release()

	repo := duckdb.NewReplayFactsRepo(db)
	joueursDe := func(matchID string) int {
		facts, ferr := repo.FactsForMatch(ctx, matchID)
		if ferr != nil {
			// Un match illisible DEGRADE ce match seul (traite comme vacuite legitime : non re-cuit,
			// donc aucun decodage gache), journalise — jamais avale, jamais un echec de passe.
			slog.WarnContext(ctx, "mode reparation : faits de match illisibles — match traite comme vacuite legitime (non re-cuit)",
				"err", ferr, "match_id", matchID)
			return 0
		}
		return len(facts.Players)
	}
	return selectionnerReparables(candidats, pr, o, report, joueursDe), nil
}

// trierEtBornerReplay trie la passe par cout croissant (les gros films en dernier, meme regle que
// backfill-killsource) puis applique `--limit`. A cout egal, l'identifiant departage — une passe doit
// etre reproductible, y compris dans son ordre. EXTRAIT pour etre partage entre la passe ordinaire
// (`filtrerEtTrierReplay`) et le mode reparation (`selectionnerReparables`).
func trierEtBornerReplay(aFaire []replayCandidat, limit int) []replayCandidat {
	sort.Slice(aFaire, func(i, j int) bool {
		if aFaire[i].chunks != aFaire[j].chunks {
			return aFaire[i].chunks < aFaire[j].chunks
		}
		return aFaire[i].matchID < aFaire[j].matchID
	})
	if limit > 0 && len(aFaire) > limit {
		aFaire = aFaire[:limit]
	}
	return aFaire
}
