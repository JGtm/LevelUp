package replayartifacts

// coupdenvoi.go — LA PUBLICATION D'UN REJEU RÉÉCRIT LE COUP D'ENVOI DE SON MATCH (ADR 0050).
//
// Le document rangé publie `t0FilmMs`, le coup d'envoi du match en ms depuis son début canonique.
// Chaque artefact qui passe par [Deriver] — cuisson locale, dépôt d'un ouvrier, file de
// régénération et génération à la demande (par le dépôt), rattrapage des dérivés — fait résoudre son
// match par `kickoff.Resolve` : la source `film` quand le document publie un coup d'envoi, la
// suivante sinon, puis le temps de jeu de chaque joueur sur ce coup d'envoi. Le champ est pris TEL
// QUE LE DOCUMENT LE DONNE, sur l'artefact rangé (jamais le blob candidat, cf. derivations.go).
//
// LECTURE AVANT TOUT SEGMENT D'ÉCRITURE : l'état du match se lit par le segment de lecture, avec les
// autres préparations ; le writer partagé n'est acquis que si un match change quelque chose. Un
// artefact déjà reporté ne réécrit rien : la résolution rend ce que la base porte déjà. Un chemin sans
// segment de lecture câblé lit l'état sur le segment d'écriture de la passe, au moment d'écrire.

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/sync/kickoff"
)

// Compteurs de la famille (ADR 0009), par titre. CompteurCoupDEnvoiEcrits : matchs réécrits ;
// CompteurCoupDEnvoiAJour : matchs qui portaient déjà ce que la résolution décide ;
// CompteurCoupDEnvoiEchecs : état illisible, writer indisponible ou écriture refusée — un défaut,
// jamais un état normal (un match absent du registre n'en est pas un : kickoff.CompteurAbsentsDuRegistre).
const (
	CompteurCoupDEnvoiEcrits = "postsync_replay_coup_d_envoi_ecrits_total"
	CompteurCoupDEnvoiAJour  = "postsync_replay_coup_d_envoi_a_jour_total"
	CompteurCoupDEnvoiEchecs = "postsync_replay_coup_d_envoi_echecs_total"
)

// errWriterSansHandle : l'acquisition a réussi sans rendre de base (shared indisponible).
var errWriterSansHandle = errors.New("segment d'ecriture sans base")

// coupsDEnvoi : ce que la préparation rend à l'écriture.
type coupsDEnvoi struct {
	// plans : les matchs résolus par le segment de lecture.
	plans []kickoff.Plan
	// surWriter : aucun segment de lecture n'est câblé sur ce chemin ; l'état des matchs de films se
	// lit alors sur le segment d'écriture de la passe, au moment d'écrire.
	surWriter bool
	films     map[string]kickoff.Film
}

// preparerCoupsDEnvoi lit l'état rangé des matchs du lot par le segment de lecture et résout leur
// coup d'envoi. Une lecture impossible fait échouer le lot (aucune marque : le rattrapage des
// dérivés le rejouera) ; un match absent du registre est signalé, sans échec.
func preparerCoupsDEnvoi(ctx context.Context, d Deps, b *bilanDerivations, lus []artefactLu) coupsDEnvoi {
	if len(lus) == 0 {
		return coupsDEnvoi{}
	}
	films := make(map[string]kickoff.Film, len(lus))
	for _, a := range lus {
		films[a.matchID] = kickoff.FilmPublished(a.doc.T0FilmMs)
	}
	if d.WithRead == nil {
		slog.WarnContext(ctx, "post-sync: coup d'envoi — aucun segment de lecture cable : l'etat des matchs se lira "+
			"sur le segment d'ecriture de la passe", "titleSlug", d.TitleSlug, "matchs", len(lus))
		return coupsDEnvoi{surWriter: true, films: films}
	}
	var plans []kickoff.Plan
	var missing []string
	var lecture error
	ouvert := false
	d.WithRead(ctx, "coup_d_envoi", func(sharedDB *sql.DB) {
		ouvert = true
		plans, missing, lecture = kickoff.Prepare(ctx, sharedDB, d.TitleSlug, films)
	})
	if !ouvert || lecture != nil {
		slog.ErrorContext(ctx, "post-sync: coup d'envoi — etat des matchs illisible : lot NON reporte, NON marque "+
			"(le rattrapage le rejouera)", "titleSlug", d.TitleSlug, "matchs", len(lus),
			"segment_ouvert", ouvert, "err", lecture)
		observability.AddIntT(d.TitleSlug, CompteurCoupDEnvoiEchecs, int64(len(lus)))
		b.echecLot(lus)
		return coupsDEnvoi{}
	}
	signalerAbsents(ctx, d, missing)
	return coupsDEnvoi{plans: plans}
}

// signalerAbsents journalise et compte les rejeux dont le match n'est pas au registre. Ce n'est pas
// un échec : le report corrige, il ne crée jamais, et rien ne changera au passage suivant ; la
// dérivation se marque comme jouée, sans quoi les familles append-only se rejoueraient à chaque cycle.
func signalerAbsents(ctx context.Context, d Deps, missing []string) {
	for _, id := range missing {
		slog.WarnContext(ctx, "post-sync: coup d'envoi — match absent du registre : rien a reporter",
			"titleSlug", d.TitleSlug, "match_id", id)
	}
	observability.AddIntT(d.TitleSlug, kickoff.CompteurAbsentsDuRegistre, int64(len(missing)))
}

// ecrireCoupsDEnvoi écrit les plans qui changent quelque chose, dans le segment d'écriture de la
// passe, acquis seulement s'il y a lieu (toujours quand l'état doit se lire sur lui).
func ecrireCoupsDEnvoi(ctx context.Context, d Deps, b *bilanDerivations, c coupsDEnvoi) {
	var aEcrire []string
	for id := range c.films {
		aEcrire = append(aEcrire, id)
	}
	for _, p := range c.plans {
		if !p.Unchanged() {
			aEcrire = append(aEcrire, p.MatchID)
		}
	}
	if len(aEcrire) == 0 {
		observability.AddIntT(d.TitleSlug, CompteurCoupDEnvoiAJour, int64(len(c.plans)))
		return
	}
	db, release, ok := writerDesCoupsDEnvoi(ctx, d, b, aEcrire)
	if !ok {
		return
	}
	defer release()
	plans := c.plans
	if c.surWriter {
		var missing []string
		var err error
		if plans, missing, err = kickoff.Prepare(ctx, db, d.TitleSlug, c.films); err != nil {
			slog.ErrorContext(ctx, "post-sync: coup d'envoi — etat des matchs illisible sur le segment d'ecriture : "+
				"lot NON reporte, NON marque", "titleSlug", d.TitleSlug, "matchs", len(c.films), "err", err)
			observability.AddIntT(d.TitleSlug, CompteurCoupDEnvoiEchecs, int64(len(c.films)))
			for id := range c.films {
				b.echec(id)
			}
			return
		}
		signalerAbsents(ctx, d, missing)
	}
	persisterCoupsDEnvoi(ctx, d, b, db, plans)
}

// writerDesCoupsDEnvoi acquiert le segment d'écriture de la passe ; sans lui, les matchs ids échouent
// faute de writer (aucune marque du lot).
func writerDesCoupsDEnvoi(ctx context.Context, d Deps, b *bilanDerivations, ids []string) (*sql.DB, func(), bool) {
	if d.AcquireWriter == nil {
		slog.WarnContext(ctx, "post-sync: coup d'envoi NON reporte (aucun writer shared cable sur ce chemin)",
			"gamertag", d.Gamertag, "matchs", len(ids))
		observability.AddIntT(d.TitleSlug, CompteurCoupDEnvoiEchecs, int64(len(ids)))
		echecFauteDeWriter(b, ids)
		return nil, nil, false
	}
	db, release, err := d.AcquireWriter(ctx)
	if err == nil && db == nil {
		if release != nil {
			release()
		}
		err = errWriterSansHandle
	}
	if err != nil {
		slog.WarnContext(ctx, "post-sync: writer shared indisponible, coup d'envoi non reporte",
			"gamertag", d.Gamertag, "matchs", len(ids), "err", err)
		observability.AddIntT(d.TitleSlug, CompteurCoupDEnvoiEchecs, int64(len(ids)))
		echecFauteDeWriter(b, ids)
		return nil, nil, false
	}
	return db, release, true
}

// persisterCoupsDEnvoi écrit les plans par le persisteur unique et compte les issues.
func persisterCoupsDEnvoi(ctx context.Context, d Deps, b *bilanDerivations, db *sql.DB, plans []kickoff.Plan) {
	aJour := 0
	for _, p := range plans {
		if p.Unchanged() {
			aJour++
		}
	}
	ecrits, echecs := kickoff.Persist(ctx, db, d.TitleSlug, plans)
	for _, id := range echecs {
		b.echec(id)
	}
	observability.AddIntT(d.TitleSlug, CompteurCoupDEnvoiEcrits, int64(ecrits))
	observability.AddIntT(d.TitleSlug, CompteurCoupDEnvoiAJour, int64(aJour))
	observability.AddIntT(d.TitleSlug, CompteurCoupDEnvoiEchecs, int64(len(echecs)))
	slog.InfoContext(ctx, "post-sync: rejeu 2D — coup d'envoi du rejeu reporte au registre",
		"gamertag", d.Gamertag, "ecrits", ecrits, "a_jour", aJour, "echecs", len(echecs))
}
