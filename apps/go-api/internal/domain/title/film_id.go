package title

import (
	"encoding/hex"
	"path/filepath"
	"strings"
)

// film_id.go — LA CLÉ DE TOUT CE QUI DÉRIVE D'UN FILM.
//
// Un fichier à part plutôt qu'une ligne de plus dans `registry.go` : ce n'est pas une
// méthode de `PathResolver`, et `registry.go` dépasse déjà largement le seuil du dépôt.

// MaxMatchIDLength borne la longueur d'un identifiant de match accepté d'une source
// externe (un UUID Halo en compte 36).
const MaxMatchIDLength = 64

// IsValidMatchID dit si id a la forme d'un identifiant de match Halo : 1 à
// MaxMatchIDLength caractères hexadécimaux ou tirets. Un identifiant venu d'une source
// externe (fichier importé, requête) qui ne la respecte pas ne doit être ni écrit, ni
// placé dans une requête SQL, ni transformé en chemin. Définition unique du module
// (garde-rail archlint/match_id_form_single_source_test.go).
func IsValidMatchID(id string) bool {
	if id == "" || len(id) > MaxMatchIDLength {
		return false
	}
	for _, c := range id {
		isHex := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
		if !isHex && c != '-' {
			return false
		}
	}
	return true
}

// invalidMatchKey rend la clé de FilmShortMatchID pour un identifiant mal formé : un octet
// NUL suivi de l'identifiant en hexadécimal. Les appels système refusent un chemin qui
// contient NUL (Linux et Windows : EINVAL) ; l'hexadécimal ne porte aucun séparateur. Un
// chemin construit sur cette clé reste donc lexicalement dans son dossier (un seul élément),
// ne désigne aucun fichier, et toute ouverture, création ou lecture échoue. Comme clé de
// comparaison, elle n'égale ni la forme courte d'un identifiant valide, ni la clé d'un autre
// identifiant mal formé, ni un nom lu sur disque.
func invalidMatchKey(matchID string) string {
	return "\x00" + hex.EncodeToString([]byte(matchID))
}

// FilmShortMatchID rend la forme COURTE d'un identifiant de match : les 8 caractères
// hexadécimaux qui précèdent le premier tiret.
//
// C'EST LA CLÉ DES CHUNKS, DES MANIFESTS ET DE L'ARTEFACT DE REJEU. Elle vient du cache
// historique et elle est TENUE par les répertoires déjà sur disque
// (`data/cache/film_chunks/000d5950/`, `film_manifests/000d5950.json`). Le reste de
// l'application, lui, manipule le match_id COMPLET.
//
// POURQUOI CETTE FONCTION EXISTE, ET UNE SEULE FOIS. Sans elle, les deux formes se
// croisaient sans jamais se rencontrer : l'outil hors ligne écrivait `000d5950.json`
// (forme courte, celle du film qu'il venait de décoder) et l'API cherchait
// `000d5950-….json` (forme complète, celle de la route). L'artefact existait, et le
// service répondait « aucun rejeu » — un 404 sur un fichier présent, donc un lien qui
// n'aurait JAMAIS pu apparaître. Mesuré le 2026-08-02 : les 3 artefacts du dépôt sont
// tous en forme courte.
//
// Un identifiant sans tiret et plus court que 8 caractères est rendu tel quel : on
// tronque une forme connue, on n'en fabrique pas une.
//
// FERMÉ PAR DÉFAUT : un identifiant qui n'a pas la forme d'un match (IsValidMatchID) rend
// invalidMatchKey(matchID). Tout chemin construit sur cette clé reste dans son dossier et ne
// désigne aucun fichier : lecture et écriture échouent (cf. invalidMatchKey).
func FilmShortMatchID(matchID string) string {
	if !IsValidMatchID(matchID) {
		return invalidMatchKey(matchID)
	}
	if i := strings.IndexByte(matchID, '-'); i > 0 {
		return matchID[:i]
	}
	if len(matchID) > 8 {
		return matchID[:8]
	}
	return matchID
}

// Extensions des fichiers de rejeu rangés dans le dossier des artefacts. Source unique : un
// littéral de ces extensions hors de ce fichier est refusé par
// `archlint/no_replay_json_literal_test.go`.
const (
	// ReplayArtifactExt : l'extension de l'artefact tel qu'il s'écrit (JSON compressé, gzip).
	ReplayArtifactExt = ".json.gz"
	// ReplayArtifactLegacyExt : l'extension d'un artefact brut, tel que les versions antérieures
	// l'écrivaient. Un parc peut en porter encore ; il se lit, il ne s'écrit plus.
	ReplayArtifactLegacyExt = ".json"
	// ReplayDerivationsMarkSuffix : le suffixe de la marque de dérivation, voisine de l'artefact
	// dans le même dossier. Elle finit par l'extension brute sans être un artefact.
	ReplayDerivationsMarkSuffix = ".derived.json"
	// ReplaySidecarExt : l'extension d'un sidecar de rejeu (raster tactique), JSON brut.
	ReplaySidecarExt = ".json"
)

// ReplayArtifactFileName rend le nom de fichier de l'artefact de rejeu d'un match :
// `{short8}.json.gz`, la forme COURTE du match (cf. FilmShortMatchID). Source unique du nom,
// partagée par PathResolver et DemoLayout.
func ReplayArtifactFileName(matchID string) string {
	return FilmShortMatchID(matchID) + ReplayArtifactExt
}

// IsReplayArtifactFile dit si ce NOM DE FICHIER est un artefact de rejeu : `{x}.json.gz`, ou
// `{x}.json` d'un parc ancien. Ni la marque de dérivation (`{x}.derived.json`), ni l'en-tête
// (`{x}.meta`), ni une sauvegarde manuelle ne le sont. Le nom, pas le chemin : les balayages de
// dossier n'ont que l'entrée. Prédicat unique de tous les listages d'artefacts.
func IsReplayArtifactFile(name string) bool {
	if strings.HasSuffix(name, ReplayDerivationsMarkSuffix) {
		return false
	}
	stem := ReplayArtifactStem(name)
	return stem != name && stem != "" && !strings.HasSuffix(stem, string(filepath.Separator))
}

// ReplayArtifactStem retire l'extension d'artefact (compressée ou brute) d'un nom ou d'un chemin :
// `{dir}/000d5950.json.gz` et `{dir}/000d5950.json` rendent `{dir}/000d5950`. Un nom sans
// extension d'artefact est rendu tel quel. C'est la base commune de l'en-tête (`.meta`) et de la
// marque de dérivation : elles gardent le même nom quelle que soit la forme de l'artefact.
func ReplayArtifactStem(nameOrPath string) string {
	if s, ok := strings.CutSuffix(nameOrPath, ReplayArtifactExt); ok {
		return s
	}
	if s, ok := strings.CutSuffix(nameOrPath, ReplayArtifactLegacyExt); ok {
		return s
	}
	return nameOrPath
}

// ReplayArtifactCandidates rend les deux fichiers possibles de l'artefact désigné par `path`
// (donné sous l'une ou l'autre forme) : le compressé d'abord, le brut ancien ensuite. Un chemin
// sans extension d'artefact ne désigne que lui-même : (path, "").
func ReplayArtifactCandidates(path string) (compressed, legacy string) {
	stem := ReplayArtifactStem(path)
	if stem == path {
		return path, ""
	}
	return stem + ReplayArtifactExt, stem + ReplayArtifactLegacyExt
}
