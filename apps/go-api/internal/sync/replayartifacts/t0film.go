package replayartifacts

// t0film.go — LE COUP D'ENVOI MESURE DANS LE FILM, REPORTE AU REGISTRE AU FIL DE L'EAU.
//
// # CE QUE CE FICHIER FERME
//
// L'artefact fraichement cuit publie `t0FilmMs` : le coup d'envoi du match date par le premier
// mouvement des joueurs (cf. `analysis/replay/t0_film.go`). Mais tout ce qui date sur l'axe du
// MATCH hors du rejeu — le premier frag, la duree jouable, la vue de match — lit
// `match_registry.real_start_time`, ESTIME des `first_joined_time` de l'API : degenere a ~0 ms
// sur 10-15 % des matchs. Sans ce report, chaque nouveau match repartait donc avec le mauvais
// T0 jusqu'a la prochaine passe de `cmd/backfill_t0_film`.
//
// # LA FORME DE L'ECRITURE, ET POURQUOI ELLE EST CELLE-LA
//
//	SEGMENT WRITER COURT   acquis APRES la cuisson, jamais pendant un decodage. Un decodage
//	                       dure des secondes a des minutes ; tenir un writer partage pendant
//	                       ce temps est exactement ce que le decoupage du paquet interdit.
//	VIA `internal/persist` l'UPDATE lui-meme vit dans `persist.T0FilmPersister`, parce que
//	                       `match_registry` est une table match-of-record : le garde-rail
//	                       `shared_write_guard_test.go` n'en autorise l'ecriture que depuis le
//	                       paquet d'orchestration ou depuis une allowlist de dettes. Une
//	                       ecriture NEUVE n'a rien a faire dans une liste de dettes. Ce
//	                       fichier-ci garde la LECTURE et la DECISION ; le persister ecrit.
//	ROW-BY-ROW             un `UPDATE ... WHERE match_id = ?` par match, sequentiel. La forme
//	                       bulk sur `match_registry` est le declencheur direct du bug ART
//	                       #23046 et `internal/sync/no_art_patterns_test.go` la refuse.
//	AU PLUS `maxPerCycle`  le lot du cycle est plafonne a cinq artefacts : le burst est borne
//	                       par construction.
//
// LE SEGMENT DE LECTURE EST DEJA RELACHE quand on arrive ici (`selectionnerLeTravail` rend la
// main avant `buildAll`) : `SharedAccess.Write` refuserait un burst avec un Read en vol
// (garde anti-deadlock du drain). Les deux lectures que ce fichier fait — le start canonique et
// l'etat courant du T0 — se font donc SUR LE HANDLE WRITER, dans le meme segment.
//
// # LA GARDE : ON NE REECRIT PAS CE QUI EST DEJA LA
//
// Un match deja marque `film_movement` a la MEME valeur ne se reecrit pas. Sans cette garde,
// un artefact re-cuit (reparation d'un artefact appauvri, montee de schema) rejouerait un
// UPDATE identique a chaque passage — une ecriture par cycle sur une table critique, pour rien.
//
// # CE QUI N'EST PAS REPORTE ICI, ET POURQUOI
//
// Seuls les artefacts CUITS DANS CE CYCLE. Un artefact deja a jour est saute par `buildAll`
// sans etre relu, et le chemin « ouvrier » ne cuit rien du tout. La reparation de l'historique
// — tout le corpus deja sur disque — est le travail de `cmd/backfill_t0_film`, hors ligne et
// sous le controle de l'operateur.

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
)

// rapportT0Film : un match dont l'artefact range publie un coup d'envoi mesure, en
// millisecondes sur l'axe du MATCH (meme axe que `header.t0_ms`).
type rapportT0Film struct {
	matchID  string
	t0FilmMs int64
}

// resultatT0Film : ce qu'un match a donne au report.
type resultatT0Film int

const (
	t0FilmEcrit resultatT0Film = iota
	t0FilmDejaLa
	t0FilmEchec
)

// LA LECTURE DU CHAMP `t0FilmMs` A DEMENAGE (2026-09-06) : elle vit dans `rapportsT0`
// (derivations.go), qui la tire du document DEJA LU par [Deriver] — une seule lecture de
// l'artefact range pour les quatre derivations, au lieu d'une par famille.
//
// SUR DISQUE, ET NON DANS LE BLOB CANDIDAT : la regle n'a pas bouge. `StoreArtifact` peut
// REFUSER l'ecriture (garde anti-regression) et conserver l'artefact precedent ; reporter la
// valeur du candidat ecrirait en base un coup d'envoi que le disque ne porte pas.
//
// Un document sans `t0FilmMs` — schema anterieur au champ, ou detecteur qui a REFUSE de dater
// le coup d'envoi (piege omitempty documente sur le champ) — ne donne aucun rapport : les deux
// cas veulent la meme chose, ne rien ecrire.

// reporterT0Film ecrit les coups d'envoi mesures du cycle dans `match_registry`.
//
// Best-effort de bout en bout, comme toute l'etape : aucun echec ne remonte au cycle, mais
// aucun ne se tait non plus.
func reporterT0Film(ctx context.Context, d Deps, rapports []rapportT0Film) {
	if len(rapports) == 0 {
		return
	}
	titre := ctxkeys.TitleSlug(ctx)
	if d.AcquireWriter == nil {
		// DEGRADATION VOULUE, PAS UN SILENCE : un chemin de sync sans writer cable cuit ses
		// artefacts mais laisse le registre sur le T0 de l'API. Il faut le voir.
		slog.WarnContext(ctx, "post-sync: rejeu 2D — coup d'envoi mesure NON reporte au registre "+
			"(aucun writer shared cable sur ce chemin)",
			"gamertag", d.Gamertag, "matchs", len(rapports))
		observability.AddIntT(titre, CompteurT0FilmEchecs, int64(len(rapports)))
		return
	}
	db, release, err := d.AcquireWriter(ctx)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: rejeu 2D — writer shared indisponible, coup d'envoi non reporte",
			"gamertag", d.Gamertag, "matchs", len(rapports), "err", err)
		observability.AddIntT(titre, CompteurT0FilmEchecs, int64(len(rapports)))
		return
	}
	defer release()
	p := persist.NewT0FilmPersister(db)
	ecrits, dejaLa, echecs := 0, 0, 0
	for _, r := range rapports {
		switch ecrireUnT0Film(ctx, db, p, r) {
		case t0FilmEcrit:
			ecrits++
		case t0FilmDejaLa:
			dejaLa++
		case t0FilmEchec:
			echecs++
		}
	}
	observability.AddIntT(titre, CompteurT0FilmReportes, int64(ecrits))
	observability.AddIntT(titre, CompteurT0FilmDejaLa, int64(dejaLa))
	observability.AddIntT(titre, CompteurT0FilmEchecs, int64(echecs))
	slog.InfoContext(ctx, "post-sync: rejeu 2D — coup d'envoi du film reporte au registre",
		"gamertag", d.Gamertag, "reportes", ecrits, "deja_a_jour", dejaLa, "echecs", echecs)
}

// ecrireUnT0Film lit l'etat courant du T0 du match puis le corrige si besoin. La lecture et
// l'ecriture passent par le MEME handle writer, dans le meme segment court.
func ecrireUnT0Film(ctx context.Context, db *sql.DB, p *persist.T0FilmPersister,
	r rapportT0Film) resultatT0Film {
	var startUTC time.Time
	var realStart sql.NullTime
	var qualite string
	// Start CANONIQUE, jamais `start_time` brut (ratchet archlint/no_raw_start_time_literal).
	err := db.QueryRowContext(ctx, `
		SELECT `+analysis.SQLStartTimeCanonical("")+` AS start_utc,
		       real_start_time,
		       COALESCE(t0_quality, '')
		FROM match_registry WHERE match_id = ?`, r.matchID).Scan(&startUTC, &realStart, &qualite)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: rejeu 2D — etat du T0 illisible, coup d'envoi non reporte",
			"match_id", r.matchID, "err", err)
		return t0FilmEchec
	}
	nouveau := startUTC.UTC().Add(time.Duration(r.t0FilmMs) * time.Millisecond)
	if qualite == string(timeline.T0QualityFilmMovement) &&
		realStart.Valid && realStart.Time.UTC().Equal(nouveau) {
		return t0FilmDejaLa
	}
	if err := p.MarkT0Film(ctx, r.matchID, nouveau,
		string(timeline.T0QualityFilmMovement)); err != nil {
		slog.WarnContext(ctx, "post-sync: rejeu 2D — report du coup d'envoi echoue",
			"match_id", r.matchID, "err", err)
		return t0FilmEchec
	}
	return t0FilmEcrit
}
