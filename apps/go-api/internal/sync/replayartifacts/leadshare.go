package replayartifacts

// leadshare.go — LA MAINMISE D'UN MATCH, PERSISTEE AU FIL DE L'EAU.
//
// # CE QUE CE FICHIER FAIT
//
// Il LIT la courbe de score de l'artefact range (`scoreTimeline.teams[].total`), la passe au
// calcul pur `analysis/leadshare` avec la fenetre de la partie JOUEE, et transporte le resultat
// vers `persist.LeadSharePersister`. Aucune regle de meneur n'est ecrite ici : elle vit dans le
// calcul pur, partagee avec le web par un jeu de cas.
//
// # LA FENETRE, ET CE QUE LE SERVEUR SAIT DEJA DU TEMPS DU MATCH
//
// Debut = coup d'envoi MESURE dans le film (`t0FilmMs`), fin = duree du match au registre
// (`match_registry.duration_seconds`, countdown compris), bornee a la derniere image du film :
// la fenetre du rejeu 2D, telle que `leadshare.PlayedWindow` la pose. Le film deborde le match du
// countdown avant et d'une queue apres ; la fin ne vient jamais du score.
//
// # LES MODES ET LES REFUS
//
//	mode `events` (drapeau, colline, bombe)   refus `mode_events` : 3 a 5 points sur tout le match
//	                                          disent la date de la premiere prise, pas une mainmise ;
//	modes `curve` et `hidden` (Slayer, Assassin)   mesures : le score du rejeu est lu partout.
//
// Les autres refus sont ceux du calcul pur et du calque de score (`leadshare.Reason*`). TOUT REFUS
// S'ECRIT : une ligne `match` non mesuree avec sa raison, jamais un zero ni une absence.
//
// # LE MEME SCHEMA DE LECTURE QUE LES NIVEAUX D'ARMES
//
// L'identite du match (mode, duree) se lit dans le registre par un segment de LECTURE court, AVANT
// la premiere famille qui ecrit (cf. padtiers_preparation.go : la panne du 2026-09-14 au
// 2026-09-27 et pourquoi l'ordre compte). Un match dont l'identite ne se lit pas n'est ni projete
// ni marque : le rattrapage le rejouera.
//
// # POURQUOI `DerivationsRev` N'A PAS ETE MONTEE
//
// Meme choix que les familles `padtiers` et `vehicletakes` : monter la revision ferait rejouer
// toutes les derivations de tout le parc pour une seule famille neuve. Le parc existant passe par
// la commande de rattrapage de la famille (`levelup backfill-lead-share`), qui ne redecode rien ;
// les artefacts a venir passent par le fil de l'eau.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/analysis/leadshare"
	"levelup/go-api/internal/analysis/modelabel"
	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"

	"github.com/JGtm/LevelUp/libs/film/replay"
)

// IdentiteMainmise : ce que le registre dit d'un match et que la Mainmise lit.
type IdentiteMainmise struct {
	// PairName : le `pair_name` BRUT du match (mode et variante).
	PairName string
	// DurationSeconds : duree du match, countdown compris. Nil = inconnue.
	DurationSeconds *int64
}

// passeMainmisePrete : une projection prete a persister.
type passeMainmisePrete struct {
	matchID string
	batch   persist.LeadShareBatch
}

// preparationMainmise : les passes projetees AVANT tout segment d'ecriture.
type preparationMainmise struct {
	titre string
	prets []passeMainmisePrete
}

// ReglementDuTitre charge regulation.toml d'un titre depuis son resolveur de chemins. EXPORTE pour
// le backfill, qui lit le meme reglement que le fil de l'eau.
func ReglementDuTitre(pr *titlePkg.PathResolver, titleSlug string) (*mappings.RegulationSet, error) {
	return mappings.LoadRegulationFromFile(filepath.Join(pr.TitleMappingsDir(titleSlug), "regulation.toml"))
}

// IdentitesMainmise lit `pair_name` et `duration_seconds` pour un lot de matchs. Un match absent du
// registre est absent du resultat. EXPORTEE pour le backfill ; requete par cle primaire.
func IdentitesMainmise(ctx context.Context, db *sql.DB, ids []string) (map[string]IdentiteMainmise, error) {
	out := make(map[string]IdentiteMainmise, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := db.QueryContext(ctx, `SELECT match_id, COALESCE(pair_name, ''), duration_seconds
		FROM match_registry WHERE match_id IN (?`+strings.Repeat(",?", len(ids)-1)+`)`, args...)
	if err != nil {
		return nil, fmt.Errorf("mainmise: lecture match_registry: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id, pair string
		var duration sql.NullInt64
		if err := rows.Scan(&id, &pair, &duration); err != nil {
			return nil, fmt.Errorf("mainmise: scan match_registry: %w", err)
		}
		ident := IdentiteMainmise{PairName: pair}
		if duration.Valid {
			ident.DurationSeconds = &duration.Int64
		}
		out[id] = ident
	}
	return out, rows.Err()
}

// ProjeterMainmise tire d'UN document range la passe a ecrire. Rend TOUJOURS une passe (MatchID non
// vide) : « non mesure » s'ecrit avec sa raison, c'est ce qui le distingue de « zero ».
//
// EXPORTEE : le backfill rejoue EXACTEMENT cette projection sur les artefacts deja ranges.
func ProjeterMainmise(
	matchID string, doc *replay.ReplayDocument, ident IdentiteMainmise, reg *mappings.RegulationSet,
) persist.LeadShareBatch {
	out := persist.LeadShareBatch{MatchID: matchID}
	if doc != nil {
		out.DocSchema = doc.SchemaVersion
	}
	refus := func(r leadshare.Reason) persist.LeadShareBatch {
		out.Reason = string(r)
		return out
	}
	if reg.ScoreTimelineKind(modelabel.StripMapSuffix(ident.PairName)) == mappings.ScoreTimelineEvents {
		return refus(leadshare.ReasonModeEvents)
	}
	if r := refusDuCalqueDeScore(doc); r != leadshare.ReasonNone {
		return refus(r)
	}
	window, reason := leadshare.PlayedWindow(leadshare.Clock{
		OriginMS: doc.OriginMs, T0FilmMS: doc.T0FilmMs, DurationSeconds: ident.DurationSeconds,
		FrameIntervalMS: doc.FrameIntervalMS, FrameCount: doc.FrameCount,
	})
	if reason != leadshare.ReasonNone {
		return refus(reason)
	}
	res, reason := leadshare.Measure(leadshare.Input{
		Teams: equipesDuCalque(doc.ScoreTimeline), CampCount: campsDuRoster(doc.Roster),
		FrameIntervalMS: doc.FrameIntervalMS, Window: window,
	})
	if reason != leadshare.ReasonNone {
		return refus(reason)
	}
	out.Measured = true
	out.MeasuredMS, out.TieMS, out.LeadChanges, out.SilentCamps = res.MeasuredMS, res.TieMS, res.LeadChanges, res.SilentCamps
	out.Rows = make([]persist.LeadShareRow, 0, len(res.Camps))
	for _, c := range res.Camps {
		out.Rows = append(out.Rows, persist.LeadShareRow{Camp: c.Camp, LeadMS: c.LeadMS})
	}
	return out
}

// refusDuCalqueDeScore dit pourquoi le calque de score ne se lit pas, ou ReasonNone.
func refusDuCalqueDeScore(doc *replay.ReplayDocument) leadshare.Reason {
	if doc == nil || doc.ScoreTimeline == nil {
		return leadshare.ReasonScoreMissing
	}
	if cov := scoreCoverage(doc); cov != nil {
		switch {
		case cov.Truncated:
			return leadshare.ReasonScoreTruncated
		case !cov.ModeSupported:
			return leadshare.ReasonScoreNotCovered
		}
	}
	if len(doc.ScoreTimeline.Teams) == 0 {
		return leadshare.ReasonScoreNotCovered
	}
	return leadshare.ReasonNone
}

func scoreCoverage(doc *replay.ReplayDocument) *replay.ScoreCoverage {
	if doc.Coverage == nil {
		return nil
	}
	return doc.Coverage.Score
}

// equipesDuCalque convertit les series d'equipe du document en series du calcul pur.
func equipesDuCalque(tl *replay.ScoreTimeline) []leadshare.Team {
	teams := make([]leadshare.Team, 0, len(tl.Teams))
	for _, t := range tl.Teams {
		ticks := make([]leadshare.Tick, 0, len(t.Total))
		for _, p := range t.Total {
			ticks = append(ticks, leadshare.Tick{Frame: p.T, Value: p.V})
		}
		teams = append(teams, leadshare.Team{ID: t.TeamID, Totals: ticks})
	}
	return teams
}

// campsDuRoster compte les designateurs d'equipe distincts du roster du film, sans « aucune equipe »
// (-1) ni silence (champ absent) : meme lecture que `campCountOf` cote web.
func campsDuRoster(roster []replay.RosterEntry) int {
	camps := map[int]struct{}{}
	for _, r := range roster {
		if r.Team != nil && *r.Team >= 0 {
			camps[*r.Team] = struct{}{}
		}
	}
	return len(camps)
}

// capabiliteMainmiseArmee dit si le titre declare `film.lead_share`, et DIT POURQUOI quand la
// reponse est non.
func capabiliteMainmiseArmee(ctx context.Context, d Deps) (armee, incident bool) {
	return porteCapability(ctx, d, games.CapFilmLeadShare, "mainmise", nil)
}

// preparerMainmise fait tout ce qui precede l'ecriture : porte de capability, reglement du titre,
// identite des matchs (segment de LECTURE court), projection. Best-effort : aucun echec ne remonte,
// aucun ne se tait ; un match dont l'identite ne se lit pas est inscrit au bilan (pas de marque).
func preparerMainmise(ctx context.Context, d Deps, b *bilanDerivations, lus []artefactLu) preparationMainmise {
	titre := ctxkeys.TitleSlug(ctx)
	prep := preparationMainmise{titre: titre}
	if len(lus) == 0 {
		return prep
	}
	armee, incident := capabiliteMainmiseArmee(ctx, d)
	if !armee {
		if incident {
			observability.AddIntT(titre, CompteurMainmiseEchecs, int64(len(lus)))
			b.echecLot(lus)
		}
		return prep
	}
	reg, err := reglementDuTitre(d)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: mainmise — regulation.toml illisible, cette famille s'abstient "+
			"(les autres gardent leur marque)", "titleSlug", d.TitleSlug, "err", err)
		observability.AddIntT(titre, CompteurMainmiseEchecs, int64(len(lus)))
		return prep
	}
	identites, ok := lireIdentitesMainmise(ctx, d, b, titre, lus)
	if !ok {
		return prep
	}
	raisons := map[string]int{}
	for _, a := range lus {
		ident, connu := identites[a.matchID]
		if !connu {
			slog.ErrorContext(ctx, "post-sync: mainmise — match absent du registre : NON projete, NON marque",
				"titleSlug", d.TitleSlug, "match_id", a.matchID)
			observability.AddIntT(titre, CompteurMainmiseEchecs, 1)
			b.echec(a.matchID)
			continue
		}
		batch := ProjeterMainmise(a.matchID, a.doc, ident, reg)
		if !batch.Measured {
			raisons[batch.Reason]++
			observability.AddIntT(titre, CompteurMainmiseNonMesurees, 1)
		}
		prep.prets = append(prep.prets, passeMainmisePrete{matchID: a.matchID, batch: batch})
	}
	if len(raisons) > 0 {
		slog.InfoContext(ctx, "post-sync: mainmise non mesuree, par raison",
			"titleSlug", d.TitleSlug, "projetes", len(prep.prets), "raisons", raisons)
	}
	return prep
}

// lireIdentitesMainmise lit les identites du lot par un SEGMENT DE LECTURE court. ok faux = la
// lecture a echoue : le lot est inscrit en echec au bilan et compte.
func lireIdentitesMainmise(
	ctx context.Context, d Deps, b *bilanDerivations, titre string, lus []artefactLu,
) (map[string]IdentiteMainmise, bool) {
	echec := func(motif string, err error) (map[string]IdentiteMainmise, bool) {
		slog.ErrorContext(ctx, "post-sync: mainmise — identites de match illisibles : lot NON projete, "+
			"NON marque (le rattrapage le rejouera)",
			"titleSlug", d.TitleSlug, "matchs", len(lus), "motif", motif, "err", err)
		observability.AddIntT(titre, CompteurMainmiseEchecs, int64(len(lus)))
		b.echecLot(lus)
		return nil, false
	}
	if d.WithRead == nil {
		return echec("aucun segment de lecture cable", nil)
	}
	var out map[string]IdentiteMainmise
	var lecture error
	ouvert := false
	d.WithRead(ctx, "mainmise", func(sharedDB *sql.DB) {
		ouvert = true
		out, lecture = IdentitesMainmise(ctx, sharedDB, matchIDsDuLot(lus))
	})
	if !ouvert || lecture != nil {
		return echec(fmt.Sprintf("segment_ouvert=%v", ouvert), lecture)
	}
	return out, true
}

// ecrireMainmise acquiert le writer et ecrit les passes DEJA projetees.
func ecrireMainmise(ctx context.Context, d Deps, b *bilanDerivations, prep preparationMainmise) {
	titre, prets := prep.titre, prep.prets
	if len(prets) == 0 {
		return
	}
	ids := make([]string, 0, len(prets))
	for i := range prets {
		ids = append(ids, prets[i].matchID)
	}
	if d.AcquireWriter == nil {
		slog.WarnContext(ctx, "post-sync: mainmise NON persistee (aucun writer shared cable sur ce chemin)",
			"gamertag", d.Gamertag, "matchs", len(prets))
		observability.AddIntT(titre, CompteurMainmiseEchecs, int64(len(prets)))
		echecFauteDeWriter(b, ids)
		return
	}
	db, release, err := d.AcquireWriter(ctx)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: writer shared indisponible, mainmise non persistee",
			"gamertag", d.Gamertag, "matchs", len(prets), "err", err)
		observability.AddIntT(titre, CompteurMainmiseEchecs, int64(len(prets)))
		echecFauteDeWriter(b, ids)
		return
	}
	defer release()
	p := persist.NewLeadSharePersister(db)
	ecrits, echecs := 0, 0
	for i := range prets {
		if err := p.PersistPass(ctx, prets[i].batch); err != nil {
			slog.ErrorContext(ctx, "post-sync: ecriture de la mainmise echouee",
				"match_id", prets[i].matchID, "err", err)
			echecs++
			b.echec(prets[i].matchID)
			continue
		}
		ecrits++
	}
	observability.AddIntT(titre, CompteurMainmiseEcrites, int64(ecrits))
	observability.AddIntT(titre, CompteurMainmiseEchecs, int64(echecs))
	slog.InfoContext(ctx, "post-sync: mainmise persistee",
		"gamertag", d.Gamertag, "ecrits", ecrits, "echecs", echecs)
}
