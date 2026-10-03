package main

// QUEL DOSSIER DE L'INSTALLATION EST QUELLE CARTE DU CATALOGUE.
//
// LE PROBLÈME, ET IL A COÛTÉ TROIS CARTES. `himap.ChercheModuleInstalle` apparie par JETONS DE
// NOM : ça marche pour `cliffhanger_ridgeline` -> `ridgeline`, et ça échoue en SILENCE pour
// `deadlock_map`, `oasis_map` et `scarr_map`, que le jeu range sous `btb_drydock`,
// `btb_exiled` et `btb_engine`. Elles se déclaraient « non installées » alors que leur dossier
// était là — un heuristique de nom qui échoue sans le dire fabrique une explication fausse.
//
// PISTE RÉFUTÉE, ET IL FAUT QU'ELLE RESTE ÉCRITE (2026-08-10) : apparier par les ANCRES, en
// cherchant le module dont un bsp les contient. Son témoin l'a tuée deux fois — d'abord parce
// que le bsp d'HORIZON de chaque carte couvre des kilomètres et contient les ancres de
// n'importe qui, puis, une fois départagé par le plus petit volume, parce que la mesure
// contredisait le nom sur ~20 cartes connues (Catalyst -> `ctf_breaker`, Oasis -> `ridgeline`).
// CAUSE STRUCTURELLE : chaque carte Halo est construite autour de SA propre origine, leurs
// boîtes monde se recouvrent, et une position monde ne désigne aucune carte. Ne pas rejouer.
//
// LA RÈGLE QUI TIENT est une donnée, pas un heuristique : le dépôt de variantes contient, pour
// chaque carte, un `.mvar` nommé `<carte>_<module installé>.mvar` à côté de son
// `<carte>_map.mvar`. `deadlock_btb_drydock.mvar` dit le lien noir sur blanc.
//
// CE QUI LA VALIDE : elle retrouve les 18 appariements déjà connus (les 17 résolus par le nom
// plus Vagabond), 0 manquant. Calibrer sur les cas connus puis appliquer aux inconnus : c'est
// la méthode du chantier.

import (
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/himap"
)

// suffixeVarianteParDefaut : le nom de la variante par défaut d'une carte. Il ne désigne aucun
// dossier — `deadlock_map` doit se lire « la carte Deadlock », pas « le module map ».
const suffixeVarianteParDefaut = "_map"

// liensDuDepot lit le dépôt de variantes et rend `nom de carte -> dossier installé`.
//
// Un fichier dont le suffixe ne désigne aucun dossier installé est ignoré : le dépôt porte des
// variantes de cartes absentes de cette installation, et c'est le cas nominal.
func liensDuDepot(repoRoot, levelsDir string) map[string]string {
	out := map[string]string{}
	fichiers, err := filepath.Glob(filepath.Join(repoRoot, himap.DepotVariantesCarte, "*.mvar"))
	if err != nil {
		return out
	}
	for _, f := range fichiers {
		nom := strings.TrimSuffix(filepath.Base(f), ".mvar")
		// On coupe à chaque `_` en partant de la GAUCHE, donc le suffixe le plus LONG gagne :
		// `oasis_sentry_defense_btb_exiled` doit rendre `btb_exiled`, pas un préfixe partiel.
		for i := 0; i < len(nom); i++ {
			if nom[i] != '_' {
				continue
			}
			carte, module := nom[:i], nom[i+1:]
			if dossierInstalle(levelsDir, module) {
				out[carte] = module
				break
			}
		}
	}
	return out
}

// moduleDuCatalogue rend le dossier installé d'une entrée du catalogue d'objectifs. Rend `""`
// si rien ne le désigne.
//
// Trois lectures, de la plus directe à la plus indirecte — et AUCUNE ne devine : chacune exige
// que le résultat soit un dossier réellement installé.
func moduleDuCatalogue(liens map[string]string, levelsDir, moduleCatalogue string) string {
	// 1. Le nom du catalogue porte deja le module : `cliffhanger_ridgeline`.
	if m := suffixeInstalle(levelsDir, moduleCatalogue); m != "" {
		return m
	}
	// 2. Le nom du catalogue est un nom de carte du depot : `oasis_sentry_defense`.
	if m, ok := liens[moduleCatalogue]; ok {
		return m
	}
	// 3. Le nom du catalogue est un nom de carte suffixe par la variante par defaut :
	//    `deadlock_map` -> carte `deadlock` -> `btb_drydock`.
	if carte := strings.TrimSuffix(moduleCatalogue, suffixeVarianteParDefaut); carte != moduleCatalogue {
		if m, ok := liens[carte]; ok {
			return m
		}
	}
	return ""
}

// suffixeInstalle rend le plus long suffixe de `nom` (apres un `_`) qui designe un dossier
// installe, ou `""`.
func suffixeInstalle(levelsDir, nom string) string {
	for i := 0; i < len(nom); i++ {
		if nom[i] != '_' {
			continue
		}
		if m := nom[i+1:]; dossierInstalle(levelsDir, m) {
			return m
		}
	}
	return ""
}

// dossierInstalle dit si un nom désigne un dossier de carte porteur d'un `.module`.
func dossierInstalle(levelsDir, nom string) bool {
	if levelsDir == "" || nom == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(levelsDir, nom, nom+"-rtx-new.module"))
	return err == nil
}
