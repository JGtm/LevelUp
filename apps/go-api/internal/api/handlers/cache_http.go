// Package handlers — cache_http.go : ETag fort et 304 (Not Modified), centralisés.
//
// Avant ce fichier, le motif « poser un ETag et répondre 304 sur If-None-Match »
// existait déjà en 2 endroits indépendants : writeJSONCached (helpers.go) et l'ETag
// de assets.go:217 — ce dernier posé mais jamais honoré, faute de lire
// If-None-Match (l'en-tête était donc inerte). L'étape 1 du plan
// PLAN_FONDS_CARTE_WEBP_ETAG_2026-09-09.md ajoute une 3e et une 4e copie sur
// replay.go et tactical.go : la règle n°6 du dépôt (≤ 2 copies d'un même motif)
// impose donc le helper unique ci-dessous, plus le garde-rail grep
// (cache_http_test.go, TestNoRawETagHandlingOutsideCacheHTTP).
package handlers

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// servirBlobAvecETag sert un blob binaire opaque avec ETag fort et 304 (D7/D9 du
// plan) :
//   - ETag fort `"sha256-<12 hex du contenu>"` — le contenu fait foi, jamais mtime
//     ni taille (il est de toute façon déjà intégralement en mémoire) ;
//   - If-None-Match est analysé comme une liste séparée par virgules, avec
//     préfixe faible `W/` toléré et wildcard `*` accepté ; un en-tête absent ou
//     illisible se comporte comme une absence (200), jamais comme une erreur ;
//   - sur 304 : SEUL l'ETag est posé, aucun corps ni Content-Length ;
//   - sinon : Content-Type, Cache-Control (si non vide), Content-Length, corps.
func servirBlobAvecETag(w http.ResponseWriter, r *http.Request, blob []byte, contentType, cacheControl string) {
	etag := calculerETagFort(blob)
	w.Header().Set("ETag", etag)
	if ifNoneMatchCorrespond(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", contentType)
	if cacheControl != "" {
		w.Header().Set("Cache-Control", cacheControl)
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(blob)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(blob)
}

// calculerETagFort dérive un ETag fort du contenu servi (D7) : `sha256-` suivi de
// 12 caractères hexadécimaux (6 octets du condensé SHA-256), entre guillemets
// comme l'exige la grammaire HTTP de l'ETag.
func calculerETagFort(blob []byte) string {
	sum := sha256.Sum256(blob)
	return fmt.Sprintf(`"sha256-%x"`, sum[:6])
}

// ifNoneMatchCorrespond répond au contrat If-None-Match de la RFC 7232 §3.2, dans
// la mesure que ce dépôt utilise : liste séparée par virgules, préfixe faible
// `W/` toléré (comparaison faible, la RFC l'autorise pour un GET conditionnel),
// wildcard `*` accepté. Un en-tête absent ou dont aucun membre ne correspond
// laisse la requête suivre son cours normal (200) — jamais une erreur.
func ifNoneMatchCorrespond(ifNoneMatch, etag string) bool {
	if ifNoneMatch == "" {
		return false
	}
	for _, candidat := range strings.Split(ifNoneMatch, ",") {
		candidat = strings.TrimSpace(candidat)
		if candidat == "*" {
			return true
		}
		candidat = strings.TrimPrefix(candidat, "W/")
		if candidat == etag {
			return true
		}
	}
	return false
}
