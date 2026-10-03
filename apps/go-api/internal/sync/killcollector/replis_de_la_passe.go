package killcollector

// replis_de_la_passe.go — LE COMPTEUR DES REPLIS D UNE PASSE DE FILM DU COLLECTEUR (lot J8.7 du
// PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, decision 3 du superviseur, 2026-09-27).
//
// # POURQUOI UN COMPTEUR PAR PASSE, ET PAS CELUI D UNE CUISSON
//
// Le collecteur n ecrit aucun document : ses replis (resolution des identites, ventilation des tirs,
// precision par arme, pont des positions, faits d isolement, carte du match) n ont pas de
// `coverage.fallbacks` ou voyager. Ils se comptent donc sur un compteur NE avec la passe d UN film
// ([KillSourceCollector.CollectMatch]) et publie a sa sortie, de deux facons :
//
//	expvar   un compteur par NOM de repli, `killsource_<nom>` — le prefixe des compteurs du paquet
//	         (`collector_metrics.go`), le nom tel qu au registre : une ligne de `/debug/vars` se
//	         relit au registre sans table de correspondance. Cumule sur le processus, donc sur la
//	         passe de masse.
//	journal  UNE ligne par film, `aucun` compris : c est ce qui distingue « jamais declenche » de
//	         « jamais instrumente » (D14 d).
//
// # COMMENT IL ATTEINT LES SITES
//
// Par le CONTEXTE de la passe pour les etages qui en portent un (carte du match, precision par arme,
// homonymes), et par [MatchIdentities] pour les fonctions pures qui recoivent les identites du match
// (resolution des noms, pont des positions, faits d isolement) — `IdentitiesForMatch` y recopie le
// compteur du contexte. UN SEUL compteur par film ; deux films decodes en parallele par des ouvriers
// ont chacun le leur.

import (
	"context"
	"log/slog"
	"sync"

	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/observability"
)

// prefixeReplisDeLaPasse : le prefixe des compteurs expvar par repli (ADR 0009). Celui du paquet.
const prefixeReplisDeLaPasse = "killsource_"

// cleReplisDeLaPasse : la cle de contexte de la passe. Type non exporte : aucune autre valeur ne
// peut la heurter.
type cleReplisDeLaPasse struct{}

// passeDeFilm : le compteur de la passe, et les contextes de film qu elle a ouverts (revue finale,
// 2026-10-02). Le RAPPORT D UN CONTEXTE (les replis de `grammar` et `profile` notes pendant ses
// balayages : pont d identite, pose des largeurs, lectures des porteurs) n etait verse nulle part au
// collecteur — a la cuisson, `replay` le verse a la fin du balayage. Il l est desormais UNE fois,
// a la sortie de la passe ([verserLesContextesDeLaPasse]), apres le dernier etage qui lit le contexte.
type passeDeFilm struct {
	replis    *decfilm.Compteur
	mu        sync.Mutex
	contextes []*decfilm.FilmContext
}

// avecReplisDeLaPasse rend `ctx` porteur du compteur `fb` de la passe du film.
func avecReplisDeLaPasse(ctx context.Context, fb *decfilm.Compteur) context.Context {
	return context.WithValue(ctx, cleReplisDeLaPasse{}, &passeDeFilm{replis: fb})
}

// laPasse rend la passe portee par `ctx`, ou nil hors passe.
func laPasse(ctx context.Context) *passeDeFilm {
	p, _ := ctx.Value(cleReplisDeLaPasse{}).(*passeDeFilm)
	return p
}

// replisDeLaPasse rend le compteur de la passe, ou nil hors passe (un test, un outil) : le compteur
// nil ne compte rien, et ses sites le traversent sans condition.
func replisDeLaPasse(ctx context.Context) *decfilm.Compteur {
	if p := laPasse(ctx); p != nil {
		return p.replis
	}
	return nil
}

// noterLeContexteDeLaPasse inscrit un contexte de film ouvert par la passe : son rapport sera verse
// a la sortie. Hors passe, rien.
func noterLeContexteDeLaPasse(ctx context.Context, fc *decfilm.FilmContext) {
	p := laPasse(ctx)
	if p == nil || fc == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.contextes = append(p.contextes, fc)
}

// verserLesContextesDeLaPasse verse au compteur de la passe le rapport de chaque contexte inscrit,
// UNE fois, puis les oublie. Appelee a la sortie de la passe, avant sa publication.
func verserLesContextesDeLaPasse(ctx context.Context) {
	p := laPasse(ctx)
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, fc := range p.contextes {
		replay.VerserLesReplisDuContexte(p.replis, fc)
	}
	p.contextes = nil
}

// publierReplisDeLaPasse publie les replis d UNE passe de film : un compteur expvar par nom, et la
// ligne de journal du film.
func publierReplisDeLaPasse(ctx context.Context, matchID string, fb *decfilm.Compteur) {
	// UN NOM HORS REGISTRE EST UN DIAGNOSTIC DU COMPTEUR (lot J12.3, ADR 0034 D-4) : il se journalise
	// ici, avec la passe.
	replay.JournaliserDiagnostics(ctx, fb.Diagnostics().Relever())
	rap := fb.Rapport()
	for _, r := range rap {
		observability.AddInt(prefixeReplisDeLaPasse+string(r.Nom), int64(r.Declenchements))
	}
	slog.InfoContext(ctx, "killsource: replis de la passe du film",
		"match_id", matchID, "replis", decfilm.Texte(rap))
}

// cloreLaPasse : la sortie de la passe d un film — les contextes versent leur rapport, PUIS la passe
// se publie (expvar et journal) sous le contexte de l appelant `ctx`. `matchCtx` porte la passe.
func cloreLaPasse(ctx, matchCtx context.Context, matchID string) {
	verserLesContextesDeLaPasse(matchCtx)
	publierReplisDeLaPasse(ctx, matchID, replisDeLaPasse(matchCtx))
}
