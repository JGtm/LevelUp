// Package port — media_missing.go : les interfaces des médias introuvables (statut missing) et de
// l'ouverture de l'explorateur sur un fichier.
package port

import (
	"context"

	"levelup/go-api/internal/domain"
)

// MediaMissingStore porte les lectures et l'écriture des fiches introuvables.
//
// Interface SÉPARÉE de MediaRepository, détectée par type-assertion sur le dépôt comme
// MediaDeleter : élargir MediaRepository casserait tous les mocks sans rien apporter. Toute
// méthode ne manipule que des fiches déjà en base ; aucun chemin libre du client n'est résolu sur le
// disque.
//
// Implémentée par platform/duckdb.MediaRepo (media_repo_missing.go).
type MediaMissingStore interface {
	// ListMissingMedia rend les médias introuvables du joueur du dépôt, le plus récemment capturé
	// d'abord.
	ListMissingMedia(ctx context.Context) ([]domain.MediaMissingItem, error)

	// LoadMissingTarget charge la fiche INTROUVABLE désignée par son chemin stocké. Rend
	// domain.ErrNotFound si le chemin est inconnu ou si la fiche n'est pas introuvable.
	LoadMissingTarget(ctx context.Context, filePath string) (domain.MediaFileRef, error)

	// LoadMergeTarget charge la fiche PRÉSENTE (visible) désignée par son chemin stocké. Rend
	// domain.ErrNotFound si le chemin est inconnu ou si la fiche est supprimée ou introuvable.
	LoadMergeTarget(ctx context.Context, filePath string) (domain.MediaFileRef, error)

	// ListMergeCandidates rend, pour la fiche introuvable ref, les autres fiches présentes du même
	// propriétaire et de même nature, de la plus proche en date de capture à la plus éloignée, limit
	// au plus.
	ListMergeCandidates(ctx context.Context, ref domain.MediaFileRef, limit int) ([]domain.MediaMergeCandidate, error)
}

// MediaFileRevealer ouvre l'explorateur du système sur un fichier.
//
// Implémenté par ops.OSFileRevealer, seulement quand le déploiement porte la capacité
// RevealFilesInExplorer.
type MediaFileRevealer interface {
	// Reveal ouvre l'explorateur en sélectionnant le fichier désigné par absPath ; si le fichier
	// n'existe plus, sur le dossier existant le plus proche. Une erreur dit qu'aucun dossier n'a pu
	// être ouvert.
	Reveal(ctx context.Context, absPath string) error
}

// MediaMissingService porte les actions de l'utilisateur sur les médias introuvables.
//
// Interface séparée de MediaService (même motif que MediaMissingStore) : le handler la détecte par
// type-assertion sur le service que lui rend sa fabrique. Implémentée par service.MediaService.
type MediaMissingService interface {
	// ListMissingMedia liste les médias introuvables du joueur de la route et l'état du dossier des
	// captures constaté par la dernière passe.
	ListMissingMedia(ctx context.Context, req domain.MediaMissingListRequest) (*domain.MediaMissingResponse, error)
	// ListMergeCandidates liste les fiches présentes dans lesquelles fondre une fiche introuvable.
	ListMergeCandidates(ctx context.Context, req domain.MediaMergeCandidatesRequest) (*domain.MediaMergeCandidatesResponse, error)
	// MergeMissingMedia fond une fiche introuvable dans une fiche présente.
	MergeMissingMedia(ctx context.Context, req domain.MediaMergeRequest) (*domain.MediaMergeResponse, error)
	// ForgetMissingMedia oublie une fiche introuvable sans toucher à l'original.
	ForgetMissingMedia(ctx context.Context, req domain.MediaForgetRequest) (*domain.MediaForgetResponse, error)
	// ConfirmBulkMissing confirme la disparition massive constatée pour le joueur de la route.
	ConfirmBulkMissing(ctx context.Context, req domain.MediaConfirmBulkRequest) (*domain.MediaConfirmBulkResponse, error)
	// RevealMedia ouvre l'explorateur sur le fichier d'un média, résolu depuis la fiche en base.
	RevealMedia(ctx context.Context, req domain.MediaRevealRequest) (*domain.MediaRevealResponse, error)
}
