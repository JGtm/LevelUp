package service

// replay_map_background_registre_test.go — LE GARDE-RAIL DE COUVERTURE DES FONDS DE CARTE.
//
// CE QU'IL REMPLACE : LA DÉCOUVERTE À L'ŒIL. Jusqu'au 2026-08-27, qu'une carte jouée n'ait pas
// de fond ne se voyait qu'en ouvrant le rejeu du match concerné. C'est ainsi que la DÉRIVE
// D'IDENTIFIANT D'ASSET est restée invisible : Salvation, Dynasty, Shogun, Houseki, Starboard
// et Shiro avaient bien une image publiée, mais sous un map_id que plus aucun match ne portait.
// Onze matchs s'affichaient sans fond alors que le fond existait.
//
// CE QU'IL VÉRIFIE. Pour chaque map_id du registre partagé, la chaîne de résolution RÉELLE
// (`resolveBackgroundKey` : clé map_id puis index des fonds) doit rendre une clé. Les cartes
// dont le fond n'est PAS ENCORE cuit sont listées une par une, avec leur raison et leur
// condition de retrait — jamais un seuil global, qui laisserait rentrer une régression sous le
// couvert d'un compteur.
//
// POURQUOI UN INVENTAIRE GELÉ ET PAS UNE LECTURE DE BASE. `shared_matches_v2.duckdb` n'existe
// pas en CI, et en local le serveur la tient en écriture (un second process ne peut pas
// l'ouvrir). L'inventaire est donc figé en testdata, MÊME CONVENTION que l'oracle de la phase 0
// du rejeu (`objectifs_phase0_corpus_test.go`). La re-mesure sur la base vivante existe et
// tourne à la demande : `replay_map_background_registre_cgo_test.go`, garde `FOND_REGISTRE`.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/testutil"
)

// inventaireRegistreFichier est l'inventaire gelé des cartes jouées. Le nom porte sa DATE :
// un inventaire qu'on rafraîchit garde l'ancien à côté le temps de la revue, et on voit dans
// le diff ce que la nouvelle mesure a changé.
const inventaireRegistreFichier = "map_ids_joues_20260827.json"

// carteJouee est une carte du registre, telle que l'inventaire la gèle.
type carteJouee struct {
	MapID   string   `json:"map_id"`
	Noms    []string `json:"noms"`
	Matchs  int      `json:"matchs"`
	Dernier string   `json:"dernier"`
}

type inventaireRegistre struct {
	Source      string       `json:"source"`
	MesureLe    string       `json:"mesure_le"`
	MatchsTotal int          `json:"matchs_total"`
	Cartes      []carteJouee `json:"cartes"`
}

// fondManquantAdmis est une carte JOUÉE dont on ACCEPTE, à date, qu'elle n'ait pas de fond.
type fondManquantAdmis struct {
	// Raison dit pourquoi le fond n'existe pas — la cause, jamais « pas encore fait ».
	Raison string
	// Retrait est la condition MESURABLE qui referme l'exception.
	Retrait string
}

// fondsManquantsAdmis — ALLOWLIST DATÉE DU 2026-08-27, RAMENÉE À DEUX ENTRÉES LE 2026-09-03.
//
// Elle portait sept map_id et 120 matchs sur 1940 (6,2 %). Les cinq entrées des trois cartes
// jouées en mode SUPPORTÉ sont tombées avec la publication de leurs fonds ; il ne reste que
// deux matchs, tous deux hors périmètre du rejeu (Firefight, playlist personnalisée).
//
// Ce n'est PAS une liste de cartes « à faire un jour » : chaque entrée porte la cause établie
// et la condition qui la referme. Ne rien y ajouter sans avoir d'abord vérifié que la carte
// n'a pas simplement DÉRIVÉ D'IDENTIFIANT — c'est le défaut que ce fichier existe pour
// attraper, et il se corrige en cuisant ou en republiant, pas en allongeant cette liste.
var fondsManquantsAdmis = map[string]fondManquantAdmis{
	// LIVE FIRE (3 assets, 71 matchs), DETACHMENT (25) et ARGYLE (22) ONT ÉTÉ RETIRÉES LE
	// 2026-09-03 : leurs fonds sont publiés et gatés, ce test l'a imposé au moment même de la
	// publication — c'est exactement ce qu'un cliquet doit faire.
	//
	// Ce que leur retrait a coûté, pour que la prochaine carte bloquée sache où chercher :
	//
	//   - Live Fire tenait sur « ChoisitBSP retient le mauvais bsp ». C'était faux. Il retient
	//     le bon ; ce sont les BORNES d'un bsp qui ne bornent pas la pose de ses instances,
	//     si bien que `common-rtx-new` dessinait deux arènes dans la même image. Bornée à la
	//     boîte LUE du bsp retenu (`boiteUtile`), la carte sort en 1 302 x 1 192 px.
	//   - Detachment et Argyle tenaient sur « aucune ancre » — vrai, et c'était le premier des
	//     DEUX blocages. `cmd/mapobj-build` a ingéré leurs 14 objectifs chacune ; le canevas,
	//     lui, se lit dans le `level_id` sans le fichier-lien que leur asset ne publie pas.
	// COLE PROTOCOL — au catalogue d'objectifs et au dump .mvar depuis la campagne de
	// catalogage, fond pas encore cuit. Aucune dérive : aucun fond ne porte ce nom.
	"571afb7f-63c3-40a4-9c21-06ef921eb415": {
		Raison:  "Cole Protocol (1 match) : au catalogue d'objectifs, fond pas encore cuit",
		Retrait: "fond publié (la carte est cuisinable, canevas fo09_academy)",
	},
	// CARTE DE PARTIE PERSONNALISÉE — hors rotation, hors catalogue de cuisson.
	"ae4daed6-251a-4c2f-bc6f-eb25eac1bfd8": {
		Raison:  "« TFF | Night Of The Undead » (1 match) : carte de partie personnalisée, hors rotation",
		Retrait: "aucune — à retirer de la liste seulement si la carte entre en rotation",
	},
}

// chargeInventaireRegistre lit l'inventaire gelé.
func chargeInventaireRegistre(t *testing.T) inventaireRegistre {
	t.Helper()
	blob, err := os.ReadFile(filepath.Join("testdata", inventaireRegistreFichier))
	if err != nil {
		t.Fatalf("inventaire des cartes jouées illisible : %v", err)
	}
	var inv inventaireRegistre
	if err := json.Unmarshal(blob, &inv); err != nil {
		t.Fatalf("inventaire des cartes jouées invalide : %v", err)
	}
	if len(inv.Cartes) == 0 {
		t.Fatal("inventaire vide — le garde-rail ne garderait rien")
	}
	return inv
}

// resoutFondDeCarte rejoue la chaîne RÉELLE du service pour une carte de l'inventaire.
func resoutFondDeCarte(t *testing.T, root string, c carteJouee) (string, error) {
	t.Helper()
	svc := NewReplayService(title.DefaultSlug, root,
		&mapNamesStub{mapID: c.MapID, names: c.Noms})
	bg, err := svc.MapBackground(context.Background(), "m")
	if err != nil {
		return "", err
	}
	return bg.Module, nil
}

// TestRegistreChaqueCarteJoueeATrouveSonFond — LE GARDE-RAIL. Une carte jouée sans fond
// résolvable et hors allowlist fait ÉCHOUER le test, avec de quoi agir : son map_id, ses noms,
// son poids en matchs et sa dernière apparition.
func TestRegistreChaqueCarteJoueeATrouveSonFond(t *testing.T) {
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du dépôt introuvable : %v", err)
	}
	inv := chargeInventaireRegistre(t)

	var orphelines []string
	matchsOrphelins, matchsAdmis, resolues := 0, 0, 0
	for _, c := range inv.Cartes {
		cle, errRes := resoutFondDeCarte(t, root, c)
		if errRes == nil && cle != "" {
			resolues++
			continue
		}
		if !errors.Is(errRes, port.ErrMapBackgroundNotAvailable) {
			t.Errorf("%s (%v) : erreur inattendue — la dégradation attendue est la sentinelle : %v",
				c.MapID, c.Noms, errRes)
			continue
		}
		if _, admis := fondsManquantsAdmis[c.MapID]; admis {
			matchsAdmis += c.Matchs
			continue
		}
		matchsOrphelins += c.Matchs
		orphelines = append(orphelines, fmt.Sprintf("%s  %-28s %4d matchs  dernier %s",
			c.MapID, strings.Join(c.Noms, " / "), c.Matchs, c.Dernier))
	}
	if len(orphelines) > 0 {
		sort.Strings(orphelines)
		t.Errorf("cartes JOUÉES sans fond résolvable et hors allowlist (%d cartes, %d matchs sur %d).\n"+
			"VÉRIFIER D'ABORD LA DÉRIVE D'IDENTIFIANT : si un fond porte déjà ce nom de carte, il "+
			"manque son identité dans `mapNames` — c'est la cuisson qu'il faut rejouer, pas cette liste "+
			"qu'il faut allonger.\n  %s",
			len(orphelines), matchsOrphelins, inv.MatchsTotal, strings.Join(orphelines, "\n  "))
	}
	// Un oracle qui ne résout rien ne garde rien : on exige que l'écrasante majorité passe.
	if resolues < len(inv.Cartes)/2 {
		t.Errorf("%d cartes résolues sur %d — la chaîne de résolution est cassée, pas le catalogue",
			resolues, len(inv.Cartes))
	}
	t.Logf("couverture : %d/%d cartes jouées ont un fond ; %d matchs admis sans fond sur %d",
		resolues, len(inv.Cartes), matchsAdmis, inv.MatchsTotal)
}

// TestRegistreAllowlistSansEntreePerimee — LE CLIQUET. Une carte dont le fond a été cuit doit
// SORTIR de l'allowlist : une exception qui survit à sa cause est le défaut « compatibility
// guard forever ». Ce test l'impose au moment même où le fond est publié.
func TestRegistreAllowlistSansEntreePerimee(t *testing.T) {
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du dépôt introuvable : %v", err)
	}
	inv := chargeInventaireRegistre(t)
	parID := make(map[string]carteJouee, len(inv.Cartes))
	for _, c := range inv.Cartes {
		parID[c.MapID] = c
	}

	var perimees, inconnues []string
	for mapID, admis := range fondsManquantsAdmis {
		c, joue := parID[mapID]
		if !joue {
			inconnues = append(inconnues, mapID+" — "+admis.Raison)
			continue
		}
		if cle, errRes := resoutFondDeCarte(t, root, c); errRes == nil && cle != "" {
			perimees = append(perimees, fmt.Sprintf("%s (%s) résout désormais vers %q — condition de retrait : %s",
				mapID, strings.Join(c.Noms, " / "), cle, admis.Retrait))
		}
	}
	if len(perimees) > 0 {
		sort.Strings(perimees)
		t.Errorf("entrées d'allowlist PÉRIMÉES (%d) — le fond existe, retirer l'entrée :\n  %s",
			len(perimees), strings.Join(perimees, "\n  "))
	}
	if len(inconnues) > 0 {
		sort.Strings(inconnues)
		t.Errorf("entrées d'allowlist qui ne correspondent à aucune carte de l'inventaire (%d) — "+
			"map_id erroné, ou inventaire à rafraîchir :\n  %s",
			len(inconnues), strings.Join(inconnues, "\n  "))
	}
}

// TestRegistreDeriveIdentifiantAssetCorrigee — LE TÉMOIN DU DÉFAUT D'ORIGINE.
//
// Les six cartes dont la dérive a été mesurée le 2026-08-27 : jouées sous un map_id qui n'a
// AUCUN fond, elles doivent être servies sous la clé de leur fond publié. Le test nomme les
// deux identifiants, si bien qu'il documente la dérive en même temps qu'il la garde.
func TestRegistreDeriveIdentifiantAssetCorrigee(t *testing.T) {
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du dépôt introuvable : %v", err)
	}
	res := title.NewPathResolver(root)
	derives := []struct {
		nom, mapIDJoue, cleFond string
	}{
		{"Salvation", "f633db01-3989-41d1-b6d4-bf2d220fc619", "cd08bc7a-7ba5-4502-be87-c58b641fc94d"},
		{"Dynasty", "90cd321d-1439-49de-943d-06245a407909", "cfd90b63-62fd-441a-8015-8d7804b9c3c3"},
		{"Shogun", "8f51ccb9-7dc8-4bfb-8fca-6d84d0101ac0", "33075df7-01c8-40e1-8b3e-1baee0054c76"},
		{"Houseki", "6439625e-277b-4da9-9502-eefedb186ba8", "cf034ec8-ee47-43c2-b2e8-4751c22b3d4d"},
		{"Starboard", "50771a22-62a7-4f1f-8982-3403857ba225", "7a9265af-a880-487b-8829-68d88fcfb145"},
		{"Shiro", "2962c4e0-ab15-4979-8d28-c0792632c862", "2890782c-0a33-4f2c-a468-e3a7d6cd6db4"},
	}
	for _, d := range derives {
		t.Run(d.nom, func(t *testing.T) {
			// Le map_id JOUÉ n'a bien aucun fond : sans quoi le test ne dirait rien de la dérive.
			if _, statErr := os.Stat(res.MapBackgroundMetaPath(title.DefaultSlug, d.mapIDJoue)); statErr == nil {
				t.Fatalf("un fond existe désormais sous le map_id joué %s — la dérive est "+
					"refermée à la source, retirer %s de ce témoin", d.mapIDJoue, d.nom)
			}
			cle, errRes := resoutFondDeCarte(t, root,
				carteJouee{MapID: d.mapIDJoue, Noms: []string{d.nom}})
			if errRes != nil {
				t.Fatalf("%s joué sous %s : aucun fond servi (%v) — la dérive n'est pas corrigée",
					d.nom, d.mapIDJoue, errRes)
			}
			if cle != d.cleFond {
				t.Errorf("%s -> %q, attendu %q", d.nom, cle, d.cleFond)
			}
		})
	}
}
