// Package domain — media_delete.go : suppression définitive d'un média
// (v7.3 lot 2, item 3.1).
//
// SÉMANTIQUE DE SUPPRESSION (décision de design, ADR 0022/0026) — trois
// couches, une seule est une écriture DB :
//
//  1. FICHIERS DISQUE : supprimés définitivement (source + miniature + HLS).
//     C'est la seule garantie de « plus jamais servi » : ServeMediaFile ne
//     consulte PAS la base, il résout un chemin sur le disque. Tant que le
//     fichier existe, son URL directe reste servable.
//
//  2. media_files : SOFT-DELETE par `status = 'deleted'` (UPDATE d'une colonne
//     NON indexée), jamais un DELETE de la ligne. Motif : la table porte 4
//     index ART (PK + idx_mf_player_slug + idx_mf_created + idx_mf_player_stem)
//     et le repo a déjà subi une FATAL-invalidation de DB en prod sur un DELETE
//     simple d'une PK ART (incident catalog_fetch_queue 2026-06-19, cf.
//     no_art_patterns_test.go). Une invalidation de shared_social emporterait
//     médias + likes + followers + activité pour tout le process. L'UPDATE
//     d'une colonne non indexée a le même profil de risque que `liked`, chemin
//     déjà durci et prouvé par l'item 1.5.
//
//  3. media_likes_history / media_match_associations_history : AUCUNE écriture
//     — ORPHELINS INVISIBLES. Ces tables sont append-only strictes (id +
//     written_at + vue `_latest`, ADR 0026) : DELETE et UPDATE y sont interdits.
//     Un tombstone (INSERT d'un event is_liked=false) serait de surcroît
//     sémantiquement FAUX : il signifierait « ce liker a retiré son like »,
//     falsifiant l'historique social avec N events jamais produits par les
//     utilisateurs. Ces lignes deviennent inatteignables par construction : les
//     likers ne sont lus que par GetMediaLikers(mediaPaths) dont les chemins
//     proviennent de la galerie déjà filtrée, et les associations par un
//     LEFT JOIN partant de media_files. Média masqué ⇒ likes jamais lus.
//
// Cohérence avec `media_files.liked` GLOBAL (découverte item 1.5) : `liked` est
// un attribut du média, partagé par tous les viewers ; il disparaît AVEC le
// média puisque la ligne entière est masquée. `media_likes_history` est par
// liker : ses lignes survivent comme trace historique inerte. Les deux
// convergent vers « plus rien n'est servi » sans une seule écriture append-only.
package domain

import "strings"

// MediaStatusDeleted est la valeur de media_files.status marquant un média
// supprimé définitivement. Les autres valeurs observées sont NULL (majorité des
// lignes) et 'active' (lignes du rail home) — d'où le COALESCE obligatoire dans
// tout prédicat de visibilité (cf. duckdb.MediaVisiblePredicate).
const MediaStatusDeleted = "deleted"

// MediaDeleteRequest décrit une demande de suppression définitive de média.
//
// L'identité du demandeur est résolue par le handler (frontière HTTP) puis
// évaluée par la règle métier CanDeleteMedia — le handler ne décide pas.
type MediaDeleteRequest struct {
	// FilePath est le chemin STOCKÉ en base ({owner_slug}/{rel}, forward-slash),
	// déjà converti depuis l'URL servable par mediaServableURLToStoredPath.
	FilePath string `json:"file_path"`

	// RequesterSlug est le player_slug courant en session (vide si aucune session).
	RequesterSlug string `json:"-"`

	// RequesterIsAdmin reflète le rôle de session ("admin").
	RequesterIsAdmin bool `json:"-"`

	// AuthEnforced vaut true en multi-utilisateur authentifié. En mono-utilisateur
	// et en démo, les gardes d'identité du repo sont inertes par construction
	// (cf. authz.Enforced) : la suppression suit la même règle que le reste des
	// routes joueur, sinon elle deviendrait impossible sur une instance locale.
	AuthEnforced bool `json:"-"`
}

// MediaDeleteResponse confirme la suppression.
type MediaDeleteResponse struct {
	// FilePath écho du chemin reçu par le client (URL servable) : le cache
	// client indexe ses items par cette URL — même contrat que le like (1.5).
	FilePath string `json:"file_path"`
	// Deleted vaut toujours true en réponse 200 (média masqué ET fichiers retirés).
	Deleted bool `json:"deleted"`
	// FilesRemoved compte les fichiers effectivement retirés du disque
	// (source + miniature + playlist HLS) — 0 si tous étaient déjà absents.
	FilesRemoved int `json:"files_removed"`
}

// MediaDeletionTarget porte les chemins physiques et le propriétaire d'un média
// visé par une suppression. Chargé avant toute écriture : c'est lui qui fournit
// le player_slug servant à la règle d'autorisation.
type MediaDeletionTarget struct {
	ID            int64
	OwnerSlug     string
	FilePath      string
	ThumbnailPath string
	HLSPath       string
}

// StoredPaths retourne les chemins stockés à retirer du disque, sans doublon ni
// valeur vide. La source vient toujours en premier (c'est elle qui porte la
// garantie « plus servi »).
func (t MediaDeletionTarget) StoredPaths() []string {
	out := make([]string, 0, 3)
	seen := make(map[string]struct{}, 3)
	for _, p := range []string{t.FilePath, t.ThumbnailPath, t.HLSPath} {
		if p == "" {
			continue
		}
		if _, dup := seen[p]; dup {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

// CanDeleteMedia est la RÈGLE MÉTIER d'autorisation de suppression — fonction
// pure, testable seule (matrice complète : media_delete_test.go).
//
// Décision utilisateur verrouillée (v7.3 lot 2) : PROPRIÉTAIRE + ADMIN, et eux
// seuls. Le middleware RequirePlayerOwnership ne suffit PAS ici : via
// authz.CanAccessPlayer il laisse aussi passer les CO-MEMBRES DE GROUPE
// (famille), qui peuvent légitimement consulter une galerie mais ne doivent pas
// pouvoir en détruire le contenu. Cette règle est donc la couche B de l'ADR 0029
// (autorisation au niveau de la RESSOURCE), appliquée sur le propriétaire réel
// du média — media_files.player_slug — et non sur le player_slug de la route,
// puisqu'une galerie agrège les médias de plusieurs auteurs.
func CanDeleteMedia(ownerSlug string, req MediaDeleteRequest) bool {
	// Mono-utilisateur / démo : aucune identité à opposer, comportement aligné
	// sur les autres routes joueur (authz.Enforced == false ⇒ gardes inertes).
	if !req.AuthEnforced {
		return true
	}
	if req.RequesterIsAdmin {
		return true
	}
	if req.RequesterSlug == "" || ownerSlug == "" {
		return false
	}
	return strings.EqualFold(req.RequesterSlug, ownerSlug)
}
