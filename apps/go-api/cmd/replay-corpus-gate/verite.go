package main

// verite.go — LE BANC DE VERITE DANS LE GATE : SON VERDICT EST CELUI DU GATE (D-5, 2026-09-30).
//
// Conception : .ai/V7.5/film_re/BANC_DE_VERITE_CONCEPTION_2026-09-30.md §6.3. Pour chaque temoin
// compare, les deux artefacts (reference et HEAD) sont notes par `internal/replayverite` avec les
// MEMES faits et le MEME oracle ; le verdict du banc (`FAUX` / `MANQUE` / `ok`) devient celui du
// temoin. Les differences `replaydiff` restent au rapport, A TITRE D'INFORMATION, sauf DEUX FILETS
// qui bloquent encore, parce qu'aucune mesure du banc ne les voit :
//
//	CALQUE DISPARU    un calque de premier niveau qui passe a zero ou disparait (une revision de
//	                  calque qui s'en va, `layers/n`, est une perte non couverte, donc un filet) ;
//	PERTE NON COUVERTE une perte `replaydiff` dans un bloc qu'AUCUNE mesure presente des deux cotes
//	                  ne couvre (`blocsCouverts`). Un bloc n'est couvert que si le banc a REELLEMENT
//	                  mesure son oracle sur ce temoin : sans oracle officiel, le score personnel
//	                  n'est pas couvert ; un O-S1 circulaire (non note) ne couvre rien.
//
// Pas de drapeau de bascule (regle 11 du depot) : le banc est livre actif.

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/replaydiff"
	"levelup/go-api/internal/replayverite"
)

// blocCouvert : les metriques `replaydiff` qu'une mesure du banc juge — une perte y est donc vue
// par le banc (faux negatif d'oracle en hausse = MANQUE), et n'a pas a bloquer une seconde fois.
type blocCouvert struct {
	mesure string // l'identifiant de la mesure dans le bulletin (un prefixe pour P-4)
	couvre func(metrique string) bool
}

// suffixeJoueur dit si une metrique `joueur/<xuid>/<s>` porte l'un des suffixes donnes.
func suffixeJoueur(metrique string, suffixes ...string) bool {
	if !strings.HasPrefix(metrique, "joueur/") {
		return false
	}
	for _, s := range suffixes {
		if strings.HasSuffix(metrique, "/"+s) {
			return true
		}
	}
	return false
}

func prefixe(p string) func(string) bool {
	return func(m string) bool { return strings.HasPrefix(m, p) }
}

// blocsCouverts : la table des blocs couverts, mesure par mesure. Tout le reste (armes, grenades,
// equipement, vehicules, portages, roster, objectifs hors captures, carte...) n'a pas d'oracle
// de manque au banc : une perte y reste BLOQUANTE.
var blocsCouverts = []blocCouvert{
	{replayverite.ScoreKills, func(m string) bool { return suffixeJoueur(m, "kills", "present") }},
	{replayverite.ScoreMorts, func(m string) bool { return suffixeJoueur(m, "deaths") }},
	{replayverite.ScoreAssists, func(m string) bool { return suffixeJoueur(m, "assists") }},
	{replayverite.ScorePersonnel, func(m string) bool { return suffixeJoueur(m, "score") }},
	{replayverite.ScoreMortsVies, func(m string) bool {
		return strings.HasPrefix(m, "tracks/vies-par-xuid/") || m == "tracks/vies-nommees" ||
			strings.HasPrefix(m, "coverage.bridge.")
	}},
	{replayverite.ScoreFinsDeVie, func(m string) bool { return m == "tracks/n" || m == "coverage.tracks.published" }},
	{replayverite.ScoreEquipes, prefixe("coverage.teams.")},
	{replayverite.PreuveFermeture, prefixe("coverage.continuousFire.")},
	{replayverite.PreuveContradic, prefixe("coverage.keyframes.")},
	{replayverite.PreuveVerdicts, prefixe("coverage.verdict.")},
}

// mesuresPresentes : les mesures NOTEES des deux cotes d'une comparaison (un score non note ou
// absent d'un cote ne couvre rien).
func mesuresPresentes(c *replayverite.Comparaison) map[string]bool {
	out := map[string]bool{}
	if c == nil {
		return out
	}
	for id, a := range c.Avant.Scores {
		if b, ok := c.Apres.Scores[id]; ok && !a.NonNote && !b.NonNote {
			out[id] = true
		}
	}
	for id := range c.Avant.Preuves {
		if _, ok := c.Apres.Preuves[id]; ok {
			out[id] = true
			if strings.HasPrefix(id, replayverite.PreuveVerdicts) {
				out[replayverite.PreuveVerdicts] = true
			}
		}
	}
	return out
}

// estCouverte dit si une metrique est jugee par une mesure presente.
func estCouverte(metrique string, presentes map[string]bool) bool {
	for _, b := range blocsCouverts {
		if presentes[b.mesure] && b.couvre(metrique) {
			return true
		}
	}
	return false
}

// estCalqueDisparu : un calque de premier niveau (`<cle>/n`, sans autre separateur) qui passe a
// zero ou disparait. Il bloque MEME dans un bloc couvert (`tracks/n` est couvert par O-V2, mais
// des pistes qui disparaissent entierement ne sont pas « une vie de moins »). Une revision de
// calque retiree (`layers/n` qui baisse) n'a pas besoin de cette regle : aucune mesure du banc ne
// couvre `layers`, sa perte est donc toujours un filet.
func estCalqueDisparu(d replaydiff.Difference) bool {
	if d.Sens != replaydiff.SensPerte && d.Sens != replaydiff.SensDisparu {
		return false
	}
	cle, ok := strings.CutSuffix(d.Metrique, "/n")
	if !ok || strings.ContainsAny(cle, "/.") {
		return false
	}
	return d.Nouveau == "" || d.Nouveau == "0"
}

// filetsDe rend les pertes `replaydiff` qui restent bloquantes (cf. l'en-tete).
func filetsDe(pertes []replaydiff.Difference, c *replayverite.Comparaison) []replaydiff.Difference {
	presentes := mesuresPresentes(c)
	var out []replaydiff.Difference
	for _, d := range pertes {
		if estCalqueDisparu(d) || !estCouverte(d.Metrique, presentes) {
			out = append(out, d)
		}
	}
	return out
}

// jugement porte les entrees du banc pour UN temoin.
type jugement struct {
	Reference, HEAD string
	Faits           domain.MatchFacts
	Oracle          *domain.MatchOracle
	RegistreAvant   replayverite.RegistreReplis
}

// juger note les deux artefacts et rend le verdict du banc. Un artefact illisible est une ERREUR
// du temoin : le gate n'a pas pu poser la question.
func juger(j jugement) (*replayverite.Comparaison, error) {
	avant, err := lireArtefactDuBanc(j.Reference)
	if err != nil {
		return nil, err
	}
	apres, err := lireArtefactDuBanc(j.HEAD)
	if err != nil {
		return nil, err
	}
	c := replayverite.Comparer(replayverite.Noter(avant, j.Faits, j.Oracle),
		replayverite.Noter(apres, j.Faits, j.Oracle), j.RegistreAvant)
	return &c, nil
}

func lireArtefactDuBanc(path string) (*replayverite.Document, error) {
	blob, err := os.ReadFile(path) //nolint:gosec // chemin d'artefact produit par ce gate
	if err != nil {
		return nil, fmt.Errorf("banc de verite : %w", err)
	}
	d, err := replayverite.LireDocument(blob)
	if err != nil {
		return nil, fmt.Errorf("banc de verite (%s) : %w", path, err)
	}
	return d, nil
}

// lireOracleDuTemoin lit `<id>.oracle.json` a cote des faits. Son absence n'est pas une erreur du
// temoin (les scores qui l'attendent sont alors absents DES DEUX COTES, sans effet sur le verdict)
// mais elle est journalisee : c'est un oracle en moins.
func lireOracleDuTemoin(ctx context.Context, factsDir, id string) *domain.MatchOracle {
	path := filepath.Join(factsDir, id+".oracle.json")
	blob, err := os.ReadFile(path) //nolint:gosec // chemin construit par ce gate
	if err != nil {
		slog.WarnContext(ctx, "replay-corpus-gate: oracle officiel absent — scores O-S3 non mesures",
			"temoin", id, "err", err)
		return nil
	}
	o, err := replayverite.LireOracle(blob)
	if err != nil {
		slog.WarnContext(ctx, "replay-corpus-gate: oracle officiel illisible — scores O-S3 non mesures",
			"temoin", id, "err", err)
		return nil
	}
	return o
}

// statutVerite rend le statut du temoin porte par le banc et les filets ("" si rien ne bloque).
func (l ligneRapport) statutVerite() string {
	if l.Verite != nil {
		switch l.Verite.Statut {
		case replayverite.StatutFaux:
			return statutFaux
		case replayverite.StatutManque:
			return statutManque
		}
	}
	if len(l.Filets) > 0 {
		return statutPerte
	}
	return ""
}

// imprimerVerite ecrit la section du banc, temoin par temoin, AVANT le detail `replaydiff`.
func imprimerVerite(ctx context.Context, w io.Writer, lignes []ligneRapport, registreAvantConnu bool) {
	var concernes []ligneRapport
	for _, l := range lignes {
		if l.Verite != nil || len(l.Filets) > 0 {
			concernes = append(concernes, l)
		}
	}
	if len(concernes) == 0 {
		return
	}
	_, _ = fmt.Fprintf(w, "\nBANC DE VERITE (%d temoin(s)) — le verdict du gate", len(concernes))
	if !registreAvantConnu {
		_, _ = fmt.Fprint(w, " ; registre des replis d'avant INCONNU : tout repli nouveau est un FAUX")
	}
	_, _ = fmt.Fprintln(w, " :")
	for _, l := range concernes {
		_, _ = fmt.Fprintln(w)
		if l.Verite != nil {
			if err := replayverite.Rendre(w, l.Temoin.ID, *l.Verite); err != nil {
				slog.WarnContext(ctx, "replay-corpus-gate: rendu du banc de verite", "temoin", l.Temoin.ID, "err", err)
			}
		}
		for _, d := range l.Filets {
			_, _ = fmt.Fprintf(w, "  [FILET] %s %s : %s -> %s (aucune mesure du banc ne couvre ce bloc)\n",
				d.Axe, d.Metrique, vide(d.Ancien), vide(d.Nouveau))
		}
	}
}

// constatJSON / veriteJSON : la section du banc au rapport JSON.
type constatJSON struct {
	Mesure string   `json:"mesure"`
	Sens   string   `json:"sens"`
	Avant  string   `json:"avant"`
	Apres  string   `json:"apres"`
	Detail []string `json:"detail,omitempty"`
}

type veriteJSON struct {
	Statut   string        `json:"statut"`
	Constats []constatJSON `json:"constats,omitempty"`
	Filets   []detailJSON  `json:"filets,omitempty"`
}

// veriteVersJSON traduit la comparaison et les filets d'un temoin ; nil si le banc n'a pas juge.
func veriteVersJSON(l ligneRapport) *veriteJSON {
	if l.Verite == nil && len(l.Filets) == 0 {
		return nil
	}
	v := &veriteJSON{Statut: string(replayverite.StatutOK), Filets: detailsVersJSON(l.Filets)}
	if l.Verite != nil {
		v.Statut = string(l.Verite.Statut)
		for _, c := range l.Verite.Constats {
			v.Constats = append(v.Constats, constatJSON{
				Mesure: c.Mesure, Sens: string(c.Sens), Avant: c.Avant, Apres: c.Apres, Detail: c.Detail,
			})
		}
	}
	return v
}
