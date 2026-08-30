package replay

// INDEX DES FONDS PUBLIÉS — de l'IDENTITÉ d'une carte jouée vers la CLÉ de son fond.
//
// LE DÉFAUT QUE CE FICHIER CORRIGE : LA DÉRIVE D'IDENTIFIANT D'ASSET. Un fond de carte Forge
// est publié sous le `map_id` (asset UGC) qui a servi à le cuire. Or le registre des matchs
// porte le `map_id` DU JOUR OÙ LE MATCH A ÉTÉ JOUÉ, et la même carte est republiée au fil des
// saisons sous un nouvel identifiant d'asset. Mesuré le 2026-08-27 sur le registre partagé :
// Salvation, Dynasty, Shogun, Houseki, Starboard et Shiro ont toutes été jouées sous un
// `map_id` DIFFÉRENT de celui de leur fond publié. L'image existe, la carte est sans fond à
// l'écran — le rejeu 2D cherchait une clé morte.
//
// POURQUOI PAS UNE TABLE D'ALIAS `map_id -> clé`. Une table écrite à la main ne rend pas la
// dérive impossible : elle la rend rattrapable, une entrée à la fois, APRÈS COUP. C'est le
// défaut « copy-paste config » appliqué à un catalogue qui va doubler (84 fonds publiés pour
// ~164 cartes en rotation), et c'est une SECONDE vérité à tenir à côté de celle que la cuisson
// écrit déjà.
//
// CE QU'ON FAIT À LA PLACE : ON LIT CE QUE LA CUISSON DÉCLARE DÉJÀ. Chaque sidecar publié porte
// `mapNames` — les noms affichés ET les noms de module du catalogue d'objectifs qui ont
// alimenté cette cuisson (cmd/mapfond-build, `noms` + `sources`). Ce champ n'était jusqu'ici
// que de la traçabilité, lu par personne. Il EST l'index inverse : une carte republiée sous un
// nouvel asset garde son nom, donc retrouve son fond sans qu'on écrive une ligne de donnée.
//
// CE QUI REND LA CHOSE SÛRE. Une identité portée par DEUX clés est AMBIGUË : elle est retirée
// de l'index (aucun fond) et publiée dans `Ambigues()` — jamais résolue au hasard. Servir le
// fond d'une AUTRE carte est pire que n'en servir aucun. Un garde-rail vérifie que le catalogue
// publié n'en porte aucune (map_background_index_catalogue_test.go).
//
// OFFLINE PUR ET DÉTERMINISTE : cette résolution ne lit que les sidecars versionnés du
// répertoire `map_backgrounds/`. Rien n'ouvre le jeu, rien ne va sur le réseau, aucune base.

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// identiteModuleGenerique est le nom de module GÉNÉRIQUE que le catalogue d'objectifs donne à
// certaines cartes Forge (`map`, cf. cartes_forge.go : « l'entrée de Vagabond y porte le module
// GENERIQUE `map`, comme Highpower »). Il ne désigne aucune carte : l'indexer ferait résoudre
// vers la première carte venue. Écarté à la construction.
const identiteModuleGenerique = "map"

// suffixeIdentiteModule est le suffixe que le catalogue d'objectifs accole aux noms de module
// (`salvation_map`, `oasis_sentry_defense_map`). Le retirer est ce qui met le nom AFFICHÉ
// (« Oasis Sentry Defense ») et le nom de MODULE (`oasis_sentry_defense_map`) sur la même clé.
const suffixeIdentiteModule = "_map"

// NormalizeMapIdentity met une identité de carte sous sa forme de clé d'index : minuscules,
// blancs resserrés puis remplacés par des soulignés, suffixe `_map` retiré.
//
// POURQUOI ELLE N'ENLÈVE NI « - Ranked » NI « Heavies », contrairement à
// filmdec.NormalizeMapName. Ce rabotage est JUSTE pour les bornes de déquantification (même
// niveau, mêmes bornes monde) et FAUX ici : sur les 84 fonds publiés, « Insolence » et
// « Insolence Heavies » sont deux assets Forge distincts avec deux fonds distincts — idem
// Fortitude, Thunderhead, Refuge, Obituary (Heavies) et Origin, Solitude (- Ranked). Raboter le
// suffixe les rendrait AMBIGUËS, donc sans fond toutes les deux. Ce que le rabotage apportait
// est déjà là autrement : les sidecars déclarent les variantes EXPLICITEMENT
// (`aquarius_-_ranked_map`, `oasis_heavies_map`), et la déclaration bat la règle devinée.
//
// Mesuré le 2026-08-27 sur les 84 fonds publiés : 184 identités distinctes, ZÉRO collision.
func NormalizeMapIdentity(s string) string {
	n := strings.ToLower(strings.Join(strings.Fields(s), "_"))
	return strings.TrimSuffix(n, suffixeIdentiteModule)
}

// MapBackgroundIndex est l'index inverse identité -> clé de fond publiée.
type MapBackgroundIndex struct {
	parIdentite map[string]string
	ambigues    map[string][]string
	cles        int
}

// Lookup rend la clé de fond d'une identité de carte. `false` quand l'identité est inconnue OU
// ambiguë — l'appelant dégrade sans fond, il ne choisit pas.
func (i *MapBackgroundIndex) Lookup(nom string) (string, bool) {
	if i == nil {
		return "", false
	}
	cle, ok := i.parIdentite[NormalizeMapIdentity(nom)]
	return cle, ok
}

// Ambigues rend les identités portées par plusieurs clés, avec leurs clés triées. Vide en
// régime normal ; non vide = deux cartes publiées se disputent un nom, et c'est un humain qui
// tranche (garde-rail de catalogue).
func (i *MapBackgroundIndex) Ambigues() map[string][]string {
	if i == nil {
		return nil
	}
	return i.ambigues
}

// Identites compte les identités résolvables.
func (i *MapBackgroundIndex) Identites() int {
	if i == nil {
		return 0
	}
	return len(i.parIdentite)
}

// Cles compte les fonds publiés effectivement lus.
func (i *MapBackgroundIndex) Cles() int {
	if i == nil {
		return 0
	}
	return i.cles
}

// BuildMapBackgroundIndex lit tous les sidecars d'un répertoire de fonds et construit l'index.
//
// Un sidecar illisible ou hors schéma est SIGNALÉ et sauté — jamais avalé, et jamais fatal : un
// fond abîmé ne doit pas priver de fond toutes les autres cartes.
func BuildMapBackgroundIndex(dir string) (*MapBackgroundIndex, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("répertoire de fonds illisible (%s) : %w", dir, err)
	}
	return indexDepuisEntrees(dir, entries), nil
}

// indexDepuisEntrees construit l'index à partir d'un listing déjà obtenu (le cache le réutilise
// pour ne pas lister deux fois).
func indexDepuisEntrees(dir string, entries []os.DirEntry) *MapBackgroundIndex {
	// porteurs : identité -> clés qui la revendiquent. On collecte TOUT avant de trancher —
	// décider au fil de l'eau ferait dépendre le résultat de l'ordre de lecture.
	porteurs := map[string]map[string]bool{}
	cles := 0
	for _, e := range entries {
		cle, ok := cleDeSidecar(e)
		if !ok {
			continue
		}
		chemin := filepath.Join(dir, e.Name())
		bg, errLect := LoadMapBackground(chemin)
		if errLect != nil {
			slog.Warn("index des fonds : sidecar illisible — carte non indexée",
				"err", errLect, "path", chemin)
			continue
		}
		cles++
		// La CLÉ elle-même est une identité : un registre qui nomme la carte par son module
		// installé (`btb_exiled`) doit retrouver son fond comme n'importe quel nom affiché.
		ajouteIdentite(porteurs, cle, cle)
		ajouteIdentite(porteurs, bg.Module, cle)
		for _, nom := range bg.MapNames {
			ajouteIdentite(porteurs, nom, cle)
		}
	}
	return trancheAmbiguites(porteurs, cles)
}

// trancheAmbiguites transforme les revendications en index : une identité revendiquée par une
// seule clé devient résolvable, les autres sont écartées et publiées.
func trancheAmbiguites(porteurs map[string]map[string]bool, cles int) *MapBackgroundIndex {
	idx := &MapBackgroundIndex{
		parIdentite: make(map[string]string, len(porteurs)),
		ambigues:    map[string][]string{},
		cles:        cles,
	}
	for identite, revendiquants := range porteurs {
		liste := make([]string, 0, len(revendiquants))
		for cle := range revendiquants {
			liste = append(liste, cle)
		}
		if len(liste) == 1 {
			idx.parIdentite[identite] = liste[0]
			continue
		}
		sort.Strings(liste)
		idx.ambigues[identite] = liste
		slog.Warn("index des fonds : identité de carte ambiguë — aucun fond servi sous ce nom",
			"identite", identite, "cles", strings.Join(liste, ", "))
	}
	return idx
}

// cleDeSidecar rend la clé de publication portée par un fichier `{clé}.json`.
func cleDeSidecar(e os.DirEntry) (string, bool) {
	if e.IsDir() {
		return "", false
	}
	nom := e.Name()
	if filepath.Ext(nom) != ".json" {
		return "", false
	}
	return strings.TrimSuffix(nom, ".json"), true
}

// ajouteIdentite enregistre une revendication, en écartant les identités qui ne désignent
// aucune carte (vide, ou le module générique).
func ajouteIdentite(porteurs map[string]map[string]bool, brut, cle string) {
	identite := NormalizeMapIdentity(brut)
	if identite == "" || identite == identiteModuleGenerique {
		return
	}
	if porteurs[identite] == nil {
		porteurs[identite] = map[string]bool{}
	}
	porteurs[identite][cle] = true
}

// ─────────────────────────────────────────────────────────────────────────────
// Cache par répertoire
// ─────────────────────────────────────────────────────────────────────────────

// POURQUOI UN CACHE. L'index est consulté à chaque ouverture de rejeu 2D et sa construction lit
// un fichier par carte publiée (84 aujourd'hui, ~164 visés). Relire le catalogue entier par
// requête serait payer la campagne de cuisson à chaque affichage.
//
// POURQUOI IL RESTE DÉTERMINISTE. Le cache est invalidé par la SIGNATURE du répertoire (nom,
// taille et date de chaque sidecar) : une carte cuite, recuite ou retirée change la signature,
// donc l'index. Aucun TTL — un cache qui périme au temps qui passe rendrait deux réponses
// différentes pour un même état du disque.
var (
	indexFondsMu    sync.Mutex
	indexFondsCache = map[string]indexFondsEntree{}
)

type indexFondsEntree struct {
	signature string
	index     *MapBackgroundIndex
}

// signatureIndatable compte les sidecars dont `Info()` a échoué : chaque appel rend alors une
// signature différente, donc une reconstruction. Un fichier qu'on ne sait pas dater ne doit
// jamais être servi depuis le cache comme s'il n'avait pas bougé.
var signatureIndatable atomic.Uint64

// MapBackgroundIndexFor rend l'index d'un répertoire de fonds, en réutilisant la construction
// précédente tant que le répertoire n'a pas bougé.
func MapBackgroundIndexFor(dir string) (*MapBackgroundIndex, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("répertoire de fonds illisible (%s) : %w", dir, err)
	}
	signature := signatureDossier(entries)

	indexFondsMu.Lock()
	defer indexFondsMu.Unlock()
	if cached, ok := indexFondsCache[dir]; ok && cached.signature == signature {
		return cached.index, nil
	}
	idx := indexDepuisEntrees(dir, entries)
	indexFondsCache[dir] = indexFondsEntree{signature: signature, index: idx}
	return idx, nil
}

// signatureDossier résume l'état des sidecars. `os.ReadDir` rend les entrées triées par nom :
// la signature est donc stable pour un même état du disque.
//
// Une entrée qu'on ne sait pas dater rend une signature VOLONTAIREMENT non reproductible
// (compteur d'appel) : sans cela, un sidecar remplacé passerait inaperçu derrière le cache.
func signatureDossier(entries []os.DirEntry) string {
	var sb strings.Builder
	for _, e := range entries {
		if _, ok := cleDeSidecar(e); !ok {
			continue
		}
		info, err := e.Info()
		if err != nil {
			fmt.Fprintf(&sb, "%s:indatable-%d;", e.Name(), signatureIndatable.Add(1))
			continue
		}
		fmt.Fprintf(&sb, "%s:%d:%d;", e.Name(), info.Size(), info.ModTime().UnixNano())
	}
	return sb.String()
}
