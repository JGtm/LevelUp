// Package ops — seed_demo_publish.go : la démo se GÉNÈRE à part et se PUBLIE seulement si
// elle a passé tous ses contrôles.
//
// POURQUOI. Le seed réécrivait les bases de la démo EN PLACE, puis contrôlait l'anonymisation :
// un contrôle en échec laissait des bases fautives publiées, et le déploiement recréait le
// conteneur démo dessus (revue R1 du lot recos-d, P1-4). Désormais :
//  1. la génération s'écrit dans `<démo>.generation` (même parent : la publication est un
//     renommage, jamais une copie) ;
//  2. tous les titres, leurs contrôles et les configurations y sont produits ;
//  3. la publication REMPLACE, élément par élément, ce que la génération produit (bases,
//     joueurs, titres additionnels, rejeux, configurations) ; ce que le serveur écrit en
//     tournant (`runtime/`, `auth/`) n'est jamais touché ;
//  4. un échec AVANT la publication laisse la démo publiée intacte ; un échec PENDANT la
//     publication remet en place les éléments déjà remplacés.
package ops

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
)

// demoGeneratedItems : les éléments de la racine démo que le seed PRODUIT. Un élément de cette
// liste absent de la génération est RETIRÉ de la démo publiée (un titre additionnel qui ne se
// seede plus ne doit pas y survivre) ; tout autre élément de la racine démo est préservé.
var demoGeneratedItems = []string{"warehouse", "players", "titles", "replays", "db_profiles.json", "app_settings.json"}

// demoGenerationDir : le dossier de la génération en cours, à côté de la démo publiée.
func demoGenerationDir(out string) string { return filepath.Clean(out) + ".generation" }

// demoPreviousDir : où la démo publiée range ses éléments remplacés pendant la publication.
func demoPreviousDir(out string) string { return filepath.Clean(out) + ".previous" }

// prepareDemoGeneration vide puis crée le dossier de génération (un reste d'une génération
// interrompue n'est jamais publié).
func prepareDemoGeneration(out string) (string, error) {
	gen := demoGenerationDir(out)
	if err := os.RemoveAll(gen); err != nil {
		return "", fmt.Errorf("génération démo précédente non retirée: %w", err)
	}
	if err := os.MkdirAll(gen, 0o755); err != nil {
		return "", fmt.Errorf("dossier de génération démo: %w", err)
	}
	return gen, nil
}

// discardDemoGeneration retire une génération qui ne sera pas publiée.
func discardDemoGeneration(ctx context.Context, gen string) {
	if err := os.RemoveAll(gen); err != nil {
		slog.ErrorContext(ctx, "seed-demo: génération non publiée non retirée", "err", err, "path", gen)
	}
}

// publishDemoGeneration remplace, dans la démo publiée `out`, les éléments produits par la
// génération `gen`. Sur erreur, les éléments déjà remplacés sont remis en place.
func publishDemoGeneration(ctx context.Context, gen, out string) error {
	prev := demoPreviousDir(out)
	if err := os.RemoveAll(prev); err != nil {
		return fmt.Errorf("publication démo: %w", err)
	}
	if err := os.MkdirAll(prev, 0o755); err != nil {
		return fmt.Errorf("publication démo: %w", err)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return fmt.Errorf("publication démo: %w", err)
	}
	names, err := publishedNames(gen)
	if err != nil {
		return err
	}
	var done []string
	for _, name := range names {
		if err := swapDemoItem(gen, out, prev, name); err != nil {
			// L'élément en échec compte : son ancienne version a pu partir dans prev.
			rollbackDemoPublish(ctx, gen, out, prev, append(done, name))
			return fmt.Errorf("publication démo (%s): %w", name, err)
		}
		done = append(done, name)
	}
	if err := os.RemoveAll(prev); err != nil {
		slog.ErrorContext(ctx, "seed-demo: anciens éléments de la démo non retirés", "err", err, "path", prev)
	}
	discardDemoGeneration(ctx, gen)
	return nil
}

// publishedNames : les éléments produits par la génération, et ceux de demoGeneratedItems
// qu'elle ne produit plus (à retirer), triés.
func publishedNames(gen string) ([]string, error) {
	set := map[string]bool{}
	for _, n := range demoGeneratedItems {
		set[n] = true
	}
	entries, err := os.ReadDir(gen)
	if err != nil {
		return nil, fmt.Errorf("génération démo illisible: %w", err)
	}
	for _, e := range entries {
		set[e.Name()] = true
	}
	names := make([]string, 0, len(set))
	for n := range set {
		names = append(names, n)
	}
	sort.Strings(names)
	return names, nil
}

// swapDemoItem range l'ancienne version de `name` dans `prev`, puis installe la nouvelle.
func swapDemoItem(gen, out, prev, name string) error {
	if _, err := os.Stat(filepath.Join(out, name)); err == nil {
		if err := os.Rename(filepath.Join(out, name), filepath.Join(prev, name)); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if _, err := os.Stat(filepath.Join(gen, name)); errors.Is(err, os.ErrNotExist) {
		return nil // élément que la génération ne produit plus : retiré
	}
	return os.Rename(filepath.Join(gen, name), filepath.Join(out, name))
}

// rollbackDemoPublish remet les éléments déjà remplacés dans leur version publiée.
func rollbackDemoPublish(ctx context.Context, gen, out, prev string, done []string) {
	for _, name := range done {
		if _, err := os.Stat(filepath.Join(out, name)); err == nil {
			if err := os.Rename(filepath.Join(out, name), filepath.Join(gen, name)); err != nil {
				slog.ErrorContext(ctx, "seed-demo: retour arrière de la publication incomplet", "err", err, "item", name)
			}
		}
		if _, err := os.Stat(filepath.Join(prev, name)); err == nil {
			if err := os.Rename(filepath.Join(prev, name), filepath.Join(out, name)); err != nil {
				slog.ErrorContext(ctx, "seed-demo: retour arrière de la publication incomplet", "err", err, "item", name)
			}
		}
	}
}
