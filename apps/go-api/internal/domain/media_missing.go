// Package domain — media_missing.go : les médias dont aucun fichier n'existe plus sur le disque
// (statut MediaStatusMissing), ce que l'utilisateur peut en faire (fusionner la fiche dans une autre,
// l'oublier, confirmer une disparition massive) et l'ouverture de l'explorateur sur un fichier.
//
// Aucune règle d'autorisation nouvelle : tout passe par CanModifyMedia (propriétaire ou admin), jugé
// sur le propriétaire réel de la fiche.
package domain

import "time"

// MediaRequester est l'identité du demandeur d'une écriture ou d'une lecture réservée au
// propriétaire, posée par le handler depuis la session (jamais depuis le corps de la requête).
type MediaRequester struct {
	// RequesterSlug : profil possédé par l'utilisateur de la session (vide sinon).
	RequesterSlug string `json:"-"`
	// RequesterIsAdmin reflète le rôle de session ("admin").
	RequesterIsAdmin bool `json:"-"`
	// AuthEnforced vaut true en multi-utilisateur authentifié.
	AuthEnforced bool `json:"-"`
}

// CanModify dit si le demandeur peut agir sur une fiche dont ownerSlug est le propriétaire réel.
func (r MediaRequester) CanModify(ownerSlug string) bool {
	return CanModifyMedia(ownerSlug, r.RequesterSlug, r.RequesterIsAdmin, r.AuthEnforced)
}

// MediaFileRef est une fiche de média telle que la lisent la fusion et l'oubli : de quoi la
// reconnaître, l'autoriser et retirer ce que l'application a créé pour elle.
type MediaFileRef struct {
	MediaDeletionTarget
	// Kind est la nature du média (valeur de media_files.kind) ; deux fiches ne se fusionnent que si
	// elle est égale.
	Kind string
	// CaptureUTC est l'instant de capture de référence de la fiche (début de capture, sinon fin),
	// nil quand aucun n'est connu.
	CaptureUTC *time.Time
}

// MediaMissingItem est un média introuvable, tel que la liste le montre.
type MediaMissingItem struct {
	FilePath      string     `json:"file_path"`
	Basename      string     `json:"basename"`
	Kind          string     `json:"kind"`
	ThumbnailPath *string    `json:"thumbnail_path,omitempty"`
	CaptureEndUTC *time.Time `json:"capture_end_utc,omitempty"`
	MatchID       *string    `json:"match_id,omitempty"`
}

// MediaMissingResponse liste les médias introuvables d'un joueur et l'état du dossier des captures
// constaté par la dernière passe.
type MediaMissingResponse struct {
	Items []MediaMissingItem `json:"items"`
	// RootUnavailable : le dossier des captures était inaccessible, la passe n'a rien marqué
	// introuvable.
	RootUnavailable bool `json:"root_unavailable"`
	// BulkMissingPending : nombre de médias dont le marquage est suspendu en attente de
	// confirmation (disparition massive).
	BulkMissingPending int `json:"bulk_missing_pending"`
}

// MediaMergeCandidate est une fiche présente dans laquelle une fiche introuvable peut se fondre.
type MediaMergeCandidate struct {
	FilePath      string     `json:"file_path"`
	Basename      string     `json:"basename"`
	Kind          string     `json:"kind"`
	ThumbnailPath *string    `json:"thumbnail_path,omitempty"`
	CaptureEndUTC *time.Time `json:"capture_end_utc,omitempty"`
	// DeltaSeconds est l'écart absolu, en secondes, entre les instants de capture des deux fiches ;
	// absent quand l'un des deux est inconnu.
	DeltaSeconds *int `json:"delta_seconds,omitempty"`
}

// MediaMergeCandidatesResponse regroupe les fiches candidates, de la plus proche en date à la plus
// éloignée.
type MediaMergeCandidatesResponse struct {
	FilePath   string                `json:"file_path"`
	CaptureUTC *time.Time            `json:"capture_utc,omitempty"`
	Candidates []MediaMergeCandidate `json:"candidates"`
}

// MediaMissingListRequest demande la liste des médias introuvables du joueur de la route.
type MediaMissingListRequest struct {
	MediaRequester `json:"-"`
}

// MediaMergeCandidatesRequest demande les fiches candidates pour une fiche introuvable.
type MediaMergeCandidatesRequest struct {
	MediaRequester `json:"-"`
	FilePath       string `json:"file_path"`
}

// MediaMergeRequest fond la fiche introuvable FilePath dans la fiche présente TargetFilePath.
type MediaMergeRequest struct {
	MediaRequester `json:"-"`
	FilePath       string `json:"file_path"`
	TargetFilePath string `json:"target_file_path"`
}

// MediaForgetRequest oublie une fiche introuvable.
type MediaForgetRequest struct {
	MediaRequester `json:"-"`
	FilePath       string `json:"file_path"`
}

// MediaConfirmBulkRequest confirme la disparition massive constatée pour le joueur de la route.
type MediaConfirmBulkRequest struct {
	MediaRequester `json:"-"`
}

// MediaRevealRequest demande d'ouvrir l'explorateur sur le fichier d'un média.
type MediaRevealRequest struct {
	MediaRequester `json:"-"`
	FilePath       string `json:"file_path"`
}

// MediaMergeResponse confirme la fusion.
type MediaMergeResponse struct {
	FilePath       string `json:"file_path"`
	TargetFilePath string `json:"target_file_path"`
	Merged         bool   `json:"merged"`
	// FilesRemoved compte les fichiers créés par l'application (miniature, version convertie) retirés
	// du disque pour la fiche fondue.
	FilesRemoved int `json:"files_removed"`
}

// MediaForgetResponse confirme l'oubli.
type MediaForgetResponse struct {
	FilePath  string `json:"file_path"`
	Forgotten bool   `json:"forgotten"`
	// FilesRemoved compte les fichiers créés par l'application (miniature, version convertie) retirés
	// du disque ; l'original de l'utilisateur n'est jamais touché.
	FilesRemoved int `json:"files_removed"`
}

// MediaConfirmBulkResponse confirme que le marquage suspendu sera appliqué à la passe suivante.
type MediaConfirmBulkResponse struct {
	// Confirmed vaut false quand aucune disparition massive n'attendait de confirmation : rien n'est
	// enregistré.
	Confirmed bool `json:"confirmed"`
	// BulkMissingPending est le nombre de médias dont le marquage a été confirmé.
	BulkMissingPending int `json:"bulk_missing_pending"`
}

// MediaRevealResponse confirme l'ouverture de l'explorateur.
type MediaRevealResponse struct {
	Revealed bool `json:"revealed"`
}
