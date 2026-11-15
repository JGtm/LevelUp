package main

// cmd_spawn_model_fit.go — `levelup spawn-model fit` : ajuste les poids par famille de modes et
// ecrit le fichier de reference versionne (PathResolver.SpawnModelPath).
//
// Le fichier porte, avec chaque jeu de poids, l'effectif et les scores de la validation croisee
// par match : ce qui permet de relire, sans relancer la mesure, ce que vaut la prevision d'un
// mode. Une famille sous effectifMinFamille reapparitions n'a pas de poids propres (journalise) :
// le service retombe sur la famille « tous ».

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

// effectifMinFamille : sous ce nombre de reapparitions mesurees, les poids d'une famille ne
// seraient que du bruit.
const effectifMinFamille = 500

const permFichierPoids = 0o644

func ajusterEtEcrire(ctx context.Context, cfg *config.AppConfig, o spawnModelOptions) error {
	familles, err := famillesDemandees(o)
	if err != nil {
		return err
	}
	jeu, _, release, err := donneesDuParc(ctx, cfg, o, familles, o.vue)
	if err != nil {
		return err
	}
	defer release()
	masque := tactical.ToutesLesVariables
	if o.vue {
		masque = tactical.AvecVisibilite
	}
	fichier := domain.PoidsReapparition{
		Revision: domain.SpawnModelRevision, Date: time.Now().Format("2006-01-02"),
		Definition: fmt.Sprintf("reapparitions apres mort lues dans les artefacts : %d par la lecture du film, %d par le repli sur l'ecart de 4 a 15 s",
			jeu.bilan.reapFilm, jeu.bilan.reapRepli),
		Familles: map[string]domain.PoidsFamille{},
	}
	groupes := map[string]map[string]bool{domain.FamilleToutesModes: familles}
	for f := range familles {
		groupes[f] = map[string]bool{f: true}
	}
	for nom, fams := range groupes {
		if p, ok := poidsDeGroupe(ctx, jeu, nom, fams, masque, o); ok {
			fichier.Familles[nom] = p
		}
	}
	return ecrirePoids(ctx, cfg.PathResolver().SpawnModelPath(o.titleSlug), fichier)
}

// poidsDeGroupe ajuste un groupe de familles et mesure ce qu'il vaut en validation croisee.
func poidsDeGroupe(ctx context.Context, jeu *jeuReapparitions, nom string, fams map[string]bool,
	masque [tactical.NbVariables]bool, o spawnModelOptions) (domain.PoidsFamille, bool) {
	evts := jeu.evenementsDeFamilles(fams)
	reglages := tactical.ReglagesEchantillons{MatchsMinCarte: o.matchsMinCarte}
	if o.vue {
		reglages.Vue = fonctionVue(jeu.obstacles)
	}
	echs := tactical.ConstruireEchantillons(evts, tactical.CandidatsParCarte(evts), reglages)
	if len(echs) < effectifMinFamille {
		slog.WarnContext(ctx, "spawn-model fit : famille sous l'effectif minimal, pas de poids propres",
			"famille", nom, "reapparitions", len(echs), "minimum", effectifMinFamille)
		return domain.PoidsFamille{}, false
	}
	w := tactical.Ajuster(echs, func(tactical.Echantillon) bool { return true }, masque)
	m := tactical.Evaluer(echs, tactical.ScoresValides(echs, masque))
	f := tactical.Evaluer(echs, tactical.ScoresValides(echs, tactical.FrequenceSeule))
	cartes, matchs := map[string]bool{}, map[string]bool{}
	for _, e := range evts {
		cartes[e.CarteID], matchs[e.MatchID] = true, true
	}
	return domain.PoidsFamille{
		Poids: poidsArrondis(tactical.PoidsNommes(w)), Reapparitions: len(echs), Cartes: len(cartes), Matchs: len(matchs),
		Scores: domain.ScoresReapparition{
			Top1Hasard: arrondi1(m.Hasard1), Top1Frequence: arrondi1(f.Top1), Top1: arrondi1(m.Top1),
			Top3: arrondi1(m.Top3), Rayon3Hasard: arrondi1(m.HasardRayon), Rayon3: arrondi1(m.Rayon3),
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
