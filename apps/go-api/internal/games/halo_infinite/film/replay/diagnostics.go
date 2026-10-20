package replay

// diagnostics.go — L ORCHESTRATEUR JOURNALISE CE QUE LES COUCHES ONT CONSTATE (lot J12.3, ADR 0034
// D-4).
//
// `grammar` et `facts` ne journalisent pas : ils notent des [constat.Diagnostic] dans les
// accumulateurs que l orchestrateur leur a ouverts — le contexte du film
// ([grammar.FilmContext.Diagnostics]), le compteur de replis ([fallback.Compteur.Diagnostics]),
// l enregistreur des consultations ([objectives.ReplisALaConsultation.Diagnostics]), le resultat
// du kill-feed (`killsource.Result.Diagnostics`), le recueil du statborg. L orchestrateur les
// releve et les journalise ICI, avec SON contexte : c est le seul endroit du decodeur ou un
// diagnostic devient une ligne de journal.
//
// UNE SEULE FONCTION POUR LES TROIS ORCHESTRATEURS (`film/replay`, `replaybuild`,
// `sync/killcollector`) — regle des deux copies du depot.

import (
	"context"
	"levelup/go-api/internal/games/halo_infinite/film/internal/constat"
	"log/slog"
	"slices"
)

// JournaliserDiagnostics journalise les diagnostics sous `ctx`, dans l ordre ou ils ont ete notes.
// Chaque ligne porte le code stable du diagnostic (`diagnostic=<code>`) en plus de ses attributs.
// Rien ne sort d une liste vide.
func JournaliserDiagnostics(ctx context.Context, ds []constat.Diagnostic) {
	for _, d := range ds {
		attrs := append(slices.Clip(d.Attrs), "diagnostic", string(d.Code))
		slog.Log(ctx, slog.Level(d.Niveau), d.Message, attrs...)
	}
}

// journaliserLesDiagnostics releve et journalise ce que l ASSEMBLAGE a recueilli : les replis
// (noms hors registre) et les lectures d objectifs faites sous l enregistreur des consultations.
func (a *assemblage) journaliserLesDiagnostics() {
	JournaliserDiagnostics(a.ctx, a.opt.Fallbacks.Diagnostics().Relever())
	JournaliserDiagnostics(a.ctx, a.opt.ReplisHorsBalayage.Consultations.Diagnostics().Relever())
}
