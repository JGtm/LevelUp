package title

// registry_film_facts.go — LES CHEMINS DES FAITS PERSISTES PAR FILM (lot 4.1.1-c, 2026-09-17).
//
// Un fichier a part, pour la meme raison que `registry_tactical.go:3-12` : `registry.go` porte
// une dette de seuil GELEE par la baseline, et la regle du depot est de ne pas l'accroitre
// (CLAUDE.md n 5). Le paquet est le meme, `PathResolver` reste UN type avec UNE racine.
//
// # DEUX CHOSES S'APPELLENT « LES FAITS », ET ELLES SONT INVERSES
//
// `<short8>.facts.json` EXISTE DEJA et veut dire L'INVERSE de ce fichier-ci : ce sont les faits
// que LA BASE sait du match et que le film ne dit pas — lignes de match, scores des deux camps,
// nom de variante (`internal/replaybuild/facts_file.go`, ecrit par
// `levelup replay-facts-export`, relu par `cmd/replay-equiv`).
//
// `<short8>.filmfacts.bin` est ce que LE FILM dit : les entrees decodees de l'assemblage
// ([replay.FilmFacts]), ecrites par la cuisson et relues pour rejouer un artefact SANS
// redecoder le film. L'extension est donc `.filmfacts.bin` et PAS `.facts.bin` — deux fichiers
// nommes presque pareil pour des contenus inverses se confondraient au premier diagnostic.
//
// # POURQUOI FRERE DE `data/cache/replays/`, ET JAMAIS DEDANS
//
// Le sidecar de raster tactique vit SOUS le dossier des artefacts et ne survit que parce que les
// deux parcours de ce dossier sautent les repertoires (mise en garde ecrite
// `registry_tactical.go:35-40`). Les faits de film, eux, naissent FRERES : sous
// [PathResolver.CacheRootDir] (source unique du sous-chemin `data/cache`), donc hors de portee
// des deux parcours par CONSTRUCTION plutot que par tolerance.
//
// # PAR TITRE, ET A PLAT DANS LE TITRE
//
// Le cache de chunks range a plat (`data/cache/film_chunks/{short8}`), les artefacts rangent par
// titre. Les faits rangent PAR TITRE : la cle de cuisson porte deja `titleSlug`, et la purge
// comme la mesure admin (M4-D1) sont par titre. A plat DANS le titre, parce que la mesure est un
// `os.ReadDir` + `Stat` d'un seul niveau.
//
// # ON CONSERVE TOUT (M4-D1, decision V17)
//
// Aucun plafond, aucune purge par age : le cron de purge des artefacts ne touche ni les films ni
// les faits (`internal/scheduler/replay_purge_cron.go`). Ce qui est publie a la place, c'est la
// TAILLE OCCUPEE — section Ressources de `/admin/system`, alimentee par
// `GET /admin/monitoring/resources`.

import "path/filepath"

// SousDossierFilmFacts est le segment de chemin des faits persistes par film, sous la racine du
// cache applicatif.
//
// EXPORTE parce que le litteral a un second consommateur qui ne connait pas de `PathResolver` :
// le ratchet `internal/archlint/no_hardcoded_film_cache_dirs_test.go`, qui interdit toute autre
// occurrence du nom dans le module. La definition canonique est ICI — c'est pourquoi ce fichier
// figure dans l'allowlist de ce ratchet.
const SousDossierFilmFacts = "film_facts"

// ExtensionFilmFacts est le suffixe du fichier de faits d'un film.
//
// EXPORTE, et c'est la mesure M4-D1 qui l'exige : l'inventaire du dossier ne doit compter QUE
// des fichiers de faits (`ops.FilmFactsInventory`). Compter tout ce qui traine ferait d'un
// temporaire abandonne une ligne de mesure.
//
// `.filmfacts.bin` ET PAS `.facts.bin` : cf. l'en-tete de ce fichier.
const ExtensionFilmFacts = ".filmfacts.bin"

// FilmFactsDir retourne le dossier des faits persistes par film d'un titre.
// Ex: data/cache/film_facts/halo_infinite/
//
// FRERE de `data/cache/replays/{slug}/`, jamais dedans (cf. l'en-tete).
func (p *PathResolver) FilmFactsDir(titleSlug string) string {
	return filepath.Join(p.CacheRootDir(), SousDossierFilmFacts, titleSlug)
}

// FilmFactsPath retourne le chemin des faits persistes d'un match.
// Ex: data/cache/film_facts/halo_infinite/000d5950.filmfacts.bin
//
// MEME CLE QUE L'ARTEFACT ET QUE LES CHUNKS : la forme COURTE (cf. [FilmShortMatchID]), donc le
// match_id complet et sa forme courte donnent le MEME chemin.
func (p *PathResolver) FilmFactsPath(titleSlug, matchID string) string {
	return filepath.Join(p.FilmFactsDir(titleSlug), FilmShortMatchID(matchID)+ExtensionFilmFacts)
}
