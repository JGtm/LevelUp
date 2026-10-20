package main

// carte.go — LA CARTE DU MATCH, RESOLUE OU REFUSEE (2026-09-27).
//
// Regle utilisateur, fermee : « Le flux du film est la seule source fiable. Pas de repli. » Les
// largeurs d axe du chemin absolu de position sont installees au chargement de la CARTE et ne se
// lisent nulle part dans le film. Cette commande decodait tout film sans carte, donc aux largeurs
// de Cliffhanger : sur toute autre carte, la marche des morts s y desynchronise et le scan publie a
// sa place (enquete ENQUETE_MARCHE_KILLSOURCE_2026-09-27). Elle n a pas de base pour lire le nom de
// carte du match : l operateur le DONNE (`-carte`), et la commande le resout au catalogue de bornes
// par la meme voie que la production (`PathResolver.MapQuantBoundsPath`, `MapQuantCatalog.Lookup`).
// Sans nom, ou hors catalogue, elle REFUSE — jamais un decodage aux largeurs d une autre carte.

import (
	"errors"
	"fmt"
	"strings"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
)

// errCarteObligatoire : aucune carte donnee. La commande refuse AVANT de lire le film.
var errCarteObligatoire = errors.New("la carte du match est obligatoire (-carte <nom>, un nom par " +
	"film separes par des virgules) : sans elle le film serait decode aux largeurs d une autre carte")

// cartesDesFilms rend l entree de catalogue de chacun des `n` films, dans l ordre des films.
func cartesDesFilms(o options, n int) ([]decfilm.MapQuantEntry, error) {
	noms := nomsDeCarte(o.carte)
	if len(noms) == 0 {
		return nil, errCarteObligatoire
	}
	if len(noms) != n {
		return nil, fmt.Errorf("%d carte(s) pour %d film(s) : il faut une carte par film, dans l ordre "+
			"des films (-carte \"<film 1>,<film 2>\")", len(noms), n)
	}
	catalogue, err := chargerCatalogue(o.catalogue)
	if err != nil {
		return nil, err
	}
	cartes := make([]decfilm.MapQuantEntry, 0, n)
	for _, nom := range noms {
		e, err := catalogue.Lookup(nom)
		if err != nil {
			return nil, fmt.Errorf("carte %q : %w", nom, err)
		}
		cartes = append(cartes, e)
	}
	return cartes, nil
}

// nomsDeCarte : les noms passes a `-carte`, sans les vides.
func nomsDeCarte(brut string) []string {
	var noms []string
	for _, n := range strings.Split(brut, ",") {
		if n = strings.TrimSpace(n); n != "" {
			noms = append(noms, n)
		}
	}
	return noms
}

// chargerCatalogue : le catalogue de bornes COMMIS au depot (ou `-catalogue`, pour un autre fichier).
func chargerCatalogue(chemin string) (*decfilm.MapQuantCatalog, error) {
	if chemin == "" {
		racine, err := title.FindRepoRoot()
		if err != nil {
			return nil, fmt.Errorf("racine du depot : %w", err)
		}
		chemin = title.NewPathResolver(racine).MapQuantBoundsPath(title.DefaultSlug)
	}
	catalogue, err := decfilm.LoadMapQuantCatalog(chemin)
	if err != nil {
		return nil, fmt.Errorf("catalogue de bornes : %w", err)
	}
	return catalogue, nil
}
