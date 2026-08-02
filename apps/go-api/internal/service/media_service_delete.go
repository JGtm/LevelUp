// Package service — media_service_delete.go : suppression définitive d'un média
// (v7.3 lot 2, item 3.1).
//
// La sémantique complète (soft-delete de media_files, zéro écriture sur les
// tables append-only, fichiers retirés du disque) est justifiée en tête de
// internal/domain/media_delete.go.
package service

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// WithMediaFileRemover injecte le port de retrait disque. Sans lui, DeleteMedia
// refuse d'agir : masquer un média en base sans effacer son fichier laisserait
// l'URL directe servable (ServeMediaFile ne consulte pas la base), c'est-à-dire
// une suppression qui n'en est pas une.
func WithMediaFileRemover(remover port.MediaFileRemover) MediaOption {
	return func(s *MediaService) { s.fileRemover = remover }
}

// DeleteMedia supprime définitivement un média : autorisation, masquage
// transactionnel durable, puis retrait des fichiers.
//
// ORDRE DES OPÉRATIONS — base d'abord, disque ensuite. C'est l'ordre qui a le
// pire cas le moins grave :
//   - base puis disque : un échec disque laisse un média invisible dont le
//     fichier subsiste. L'erreur est remontée (500) et tracée, l'utilisateur
//     peut réessayer, et l'opération est idempotente.
//   - disque puis base : un échec base laisse un média TOUJOURS LISTÉ dont le
//     fichier n'existe plus — vignettes cassées, lecture impossible, et aucun
//     moyen de revenir en arrière puisque le fichier est perdu.
func (s *MediaService) DeleteMedia(
	ctx context.Context,
	req domain.MediaDeleteRequest,
) (*domain.MediaDeleteResponse, error) {
	if req.FilePath == "" {
		return nil, domain.ErrBadRequest("file_path est requis")
	}
	deleter, ok := s.repo.(port.MediaDeleter)
	if !ok {
		return nil, domain.ErrInternal("suppression de média non supportée par ce dépôt")
	}
	if s.fileRemover == nil {
		return nil, domain.ErrInternal("suppression de média non configurée (aucun accès disque)")
	}
	if s.acquireWriter == nil {
		return nil, domain.ErrInternal("suppression de média non configurée (aucun writer)")
	}

	// 1. Charger la cible AVANT toute écriture : c'est elle qui porte le
	//    propriétaire réel du média, seule base légitime de l'autorisation.
	target, err := deleter.LoadMediaForDeletion(ctx, req.FilePath)
	if err != nil {
		return nil, err
	}

	// 2. Autorisation au niveau de la ressource (couche B, ADR 0029).
	if !domain.CanDeleteMedia(target.OwnerSlug, req) {
		slog.WarnContext(ctx, "media_delete refusé: ni propriétaire ni admin",
			"file_path", req.FilePath, "owner", target.OwnerSlug,
			"requester", req.RequesterSlug, "is_admin", req.RequesterIsAdmin)
		return nil, domain.ErrForbidden("Seul le propriétaire de ce média ou un administrateur peut le supprimer.")
	}

	// 3. Masquage durable.
	if err := s.markMediaDeleted(ctx, deleter, req.FilePath); err != nil {
		return nil, err
	}

	// 4. Retrait disque — c'est lui qui rend la suppression irréversible et qui
	//    garantit que l'URL directe ne sert plus rien.
	removed, err := s.fileRemover.RemoveMediaFiles(ctx, target.StoredPaths())
	if err != nil {
		slog.ErrorContext(ctx, "media_delete: média masqué mais fichiers non retirés",
			"err", err, "file_path", req.FilePath, "files_removed", removed)
		return nil, fmt.Errorf("media_delete remove files: %w", err)
	}

	slog.InfoContext(ctx, "media_delete: média supprimé",
		"file_path", req.FilePath, "owner", target.OwnerSlug,
		"requester", req.RequesterSlug, "is_admin", req.RequesterIsAdmin,
		"files_removed", removed)

	return &domain.MediaDeleteResponse{
		FilePath:     req.FilePath,
		Deleted:      true,
		FilesRemoved: removed,
	}, nil
}

// markMediaDeleted exécute le soft-delete sous LeasedWriter, puis commite avec
// CHECKPOINT.
//
// CommitWithCheckpoint et NON tx.Commit() : même garantie de durabilité que le
// like (item 1.5, media_service.go). Un commit nu laisse l'écriture dans le WAL
// shared_social jusqu'au tick du scheduler (5 min) — tout redémarrage dans cette
// fenêtre, et un push sur main déploie en prod, ferait RÉAPPARAÎTRE le média
// dans la galerie alors que ses fichiers ont été effacés du disque : une entrée
// fantôme, illisible et non réparable. C'est le pire état possible pour cette
// opération, d'où le CHECKPOINT immédiat.
//
// Ne pas revenir à tx.Commit() ici : verrouillé par
// TestMediaService_DeleteMedia_CommitsWithCheckpoint.
func (s *MediaService) markMediaDeleted(
	ctx context.Context,
	deleter port.MediaDeleter,
	filePath string,
) error {
	w, err := s.acquireWriter()
	if err != nil {
		return err
	}
	defer w.Release()

	tx, err := w.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("media_delete begin tx: %w", err)
	}
	updated, err := deleter.MarkMediaDeleted(ctx, tx, filePath)
	if err != nil {
		_ = tx.Rollback()
		slog.WarnContext(ctx, "media_delete rollback", "err", err, "file_path", filePath)
		return err
	}
	if !updated {
		// Course : le média a été supprimé entre le chargement et l'écriture.
		// Aucun fichier n'a encore été touché — on sort sans rien détruire.
		_ = tx.Rollback()
		return domain.ErrNotFound("media", filePath)
	}
	if err := w.CommitWithCheckpoint(ctx, tx); err != nil {
		return fmt.Errorf("media_delete commit: %w", err)
	}
	return nil
}
