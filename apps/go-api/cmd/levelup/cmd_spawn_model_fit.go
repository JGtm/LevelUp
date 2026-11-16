package main

// cmd_spawn_model_fit.go — `levelup spawn-model fit` : ajuste les poids par famille de modes et
// ecrit le fichier de reference versionne (PathResolver.SpawnModelPath).
//
// Le fichier porte, avec chaque jeu de poids, l'effectif et les scores de la validation croisee
// par match : ce qui permet de relire, sans relancer la mesure, ce que vaut la prevision d'un
// mode. Une famille sous effectifMinFamille reapparitions n'a pas de poids propres (journalise) :
// le choix des poids d'un match retombe alors sur la famille « tous » (tactical.PoidsPourFamille).
//
// Le fichier de reference ne s'ecrit que sur le parc entier et toutes les familles (voir
// verifierAjustementComplet). La commande exacte qui le reproduit est dans docs/COMMANDS.md.

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"time"

	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/atomicfile"
)

// effectifMinFamille : sous ce nombre de reapparitions retenues, les poids d'une famille ne
// seraient que du bruit.
const effectifMinFamille = 500

const permFichierPoids = 0o644

// verifierAjustementComplet refuse un ajustement partiel qui ecraserait le fichier de reference :
// avec `--famille` ou `--limit`, les autres familles seraient perdues et « tous » serait calcule sur
// une seule famille ou sur un extrait du parc. Un ajustement partiel s'ecrit ailleurs (`--sortie`).
func verifierAjustementComplet(o spawnModelOptions) error {
	if o.sortie != "" || (o.famille == "" && o.limite == 0) {
		return nil
	}
	return fmt.Errorf("ajustement partiel (--famille %q, --limit %d) : le fichier de reference ne se reecrit "+
		"que sur le parc entier et toutes les familles ; passer --sortie <fichier> pour ecrire ailleurs",
		o.famille, o.limite)
}

// definitionDesPoids dit ce que le fichier mesure (voir domain.PoidsReapparition.Definition).
func definitionDesPoids(jeu *jeuReapparitions, o spawnModelOptions) string {
	portee := "toutes les familles de modes du titre"
	if o.famille != "" || o.limite > 0 {
		portee = fmt.Sprintf("AJUSTEMENT PARTIEL (famille %q, %d artefacts au plus)", o.famille, o.limite)
	}
	return fmt.Sprintf("reapparitions apres mort lues dans les artefacts de rejeu, %s. Lues avant filtres : %d par la lecture "+
		"du film (%d documents sans lecture du film n'en donnent aucune). Retenue : une reapparition dont la carte est vue dans au moins %d matchs "+
		"de la famille, qui a au moins un adversaire vivant, et qui a au moins deux points candidats vus dans au moins %d AUTRES matchs "+
		"que le sien. Chaque famille est ajustee sur ses propres matchs, « tous » sur toutes. Les effectifs (reapparitions, cartes, "+
		"matchs) comptent les reapparitions retenues ; les scores sont ceux de la validation croisee a %d plis par match",
		portee, jeu.bilan.reapparitions, jeu.bilan.sansLectureFilm, o.matchsMinCarte, tactical.MatchsMinCandidat, tactical.PlisValidation)
}

func ajusterEtEcrire(ctx context.Context, cfg *config.AppConfig, o spawnModelOptions) error {
	if err := verifierAjustementComplet(o); err != nil {
		return err
	}
	familles, err := famillesDemandees(o)
	if err != nil {
		return err
	}
	jeu, _, release, err := donneesDuParc(ctx, cfg, o, familles)
	if err != nil {
		return err
	}
	defer release()
	fichier := domain.PoidsReapparition{
		Revision: domain.SpawnModelRevision, Date: time.Now().Format("2006-01-02"),
		Definition: definitionDesPoids(jeu, o), Familles: map[string]domain.PoidsFamille{},
	}
	groupes := map[string]map[string]bool{domain.FamilleToutesModes: familles}
	for f := range familles {
		groupes[f] = map[string]bool{f: true}
	}
	for nom, fams := range groupes {
		if p, ok := poidsDeGroupe(ctx, jeu, nom, fams, o); ok {
			fichier.Familles[nom] = p
		}
	}
	chemin := o.sortie
	if chemin == "" {
		chemin = cfg.PathResolver().SpawnModelPath(o.titleSlug)
	}
	return ecrirePoids(ctx, chemin, fichier)
}

// poidsDeGroupe ajuste un groupe de familles et mesure ce qu'il vaut en validation croisee. Les
// effectifs rendus sont ceux des echantillons retenus.
func poidsDeGroupe(ctx context.Context, jeu *jeuReapparitions, nom string, fams map[string]bool,
	o spawnModelOptions) (domain.PoidsFamille, bool) {
	evts := jeu.evenementsDeFamilles(fams)
	reglages := tactical.ReglagesEchantillons{MatchsMinCarte: o.matchsMinCarte}
	echs := tactical.ConstruireEchantillons(evts, tactical.CandidatsParCarte(evts), reglages)
	if len(echs) < effectifMinFamille {
		slog.WarnContext(ctx, "spawn-model fit : famille sous l'effectif minimal, pas de poids propres",
			"famille", nom, "reapparitions", len(echs), "minimum", effectifMinFamille)
		return domain.PoidsFamille{}, false
	}
	w := tactical.Ajuster(echs, func(tactical.Echantillon) bool { return true }, tactical.ToutesLesVariables)
	m := tactical.Evaluer(echs, tactical.ScoresValides(echs, tactical.ToutesLesVariables))
	f := tactical.Evaluer(echs, tactical.ScoresValides(echs, tactical.FrequenceSeule))
	cartes, matchs := map[string]bool{}, map[string]bool{}
	for _, s := range echs {
		cartes[s.Carte], matchs[s.Match] = true, true
	}
	return domain.PoidsFamille{
		Poids: poidsArrondis(tactical.PoidsNommes(w)), Reapparitions: len(echs), Cartes: len(cartes), Matchs: len(matchs),
		Scores: domain.ScoresReapparition{
			Top1Hasard: arrondi1(m.Hasard1), Top1Frequence: arrondi1(f.Top1), Top1: arrondi1(m.Top1),
			Top3: arrondi1(m.Top3), Rayon1Hasard: arrondi1(m.HasardRayon1), Rayon3Hasard: arrondi1(m.HasardRayon3),
			Rayon3: arrondi1(m.Rayon3),
		},
	}, true
}

func arrondi1(v float64) float64 { return math.Round(v*10) / 10 }

// poidsArrondis rend les poids au dix-millieme : le fichier est versionne, et seize decimales
// de bruit numerique feraient un diff a chaque ajustement.
func poidsArrondis(w map[string]float64) map[string]float64 {
	for k, v := range w {
		w[k] = math.Round(v*1e4) / 1e4
	}
	return w
}

// ecrirePoids depose le fichier, atomiquement, dans le dossier de reference du depot.
func ecrirePoids(ctx context.Context, path string, p domain.PoidsReapparition) error {
	raw, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("serialisation des poids: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creation du dossier de reference: %w", err)
	}
	if err := atomicfile.WriteFile(ctx, path, append(raw, '\n'), permFichierPoids); err != nil {
		return fmt.Errorf("ecriture des poids: %w", err)
	}
	slog.InfoContext(ctx, "spawn-model fit : poids ecrits", "path", path, "familles", len(p.Familles))
	fmt.Printf("poids ecrits : %s (%d familles)\n", path, len(p.Familles))
	return nil
}
