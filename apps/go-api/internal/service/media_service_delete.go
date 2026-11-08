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

// WithMediaFileTrasher injecte le port de la Corbeille du système. Avec lui, DeleteMedia envoie
// l'original déposé par l'utilisateur à la Corbeille (restaurable) au lieu de l'effacer ; la
// miniature et la version convertie, créées par l'application, restent effacées définitivement.
// Sans lui, tout est effacé définitivement (instance hébergée).
func WithMediaFileTrasher(trasher port.MediaFileTrasher) MediaOption {
	return func(s *MediaService) { s.fileTrasher = trasher }
}

// DeleteMedia supprime un média : autorisation, envoi des originaux à la Corbeille (bureau),
// masquage transactionnel durable, puis retrait des fichiers.
//
// ORDRE DES OPÉRATIONS. Sans Corbeille (instance hébergée), base d'abord, disque ensuite, l'ordre
// qui a le pire cas le moins grave :
//   - base puis disque : un échec disque laisse un média invisible dont le
//     fichier subsiste. L'erreur est remontée (500) et tracée, l'utilisateur
//     peut réessayer, et l'opération est idempotente.
//   - disque puis base : un échec base laisse un média TOUJOURS LISTÉ dont le
//     fichier n'existe plus — vignettes cassées, lecture impossible, et aucun
//     moyen de revenir en arrière puisque le fichier est perdu.
//
// Avec une Corbeille, les originaux y partent AVANT le masquage : un échec de la Corbeille rend
// une erreur sans aucune écriture en base ni autre retrait, donc rejouer la suppression fonctionne.
// Cet envoi est récupérable, le pire cas est donc bénin : si le masquage échoue ensuite, l'original
// est dans la Corbeille et la fiche reste visible, et la passe d'alignement suivante la répare
// (fiche introuvable, ou retrouvée par empreinte si l'original est restauré). Seuls les fichiers de
// l'application et les originaux sans Corbeille sont effacés définitivement, après le masquage.
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

	// 3. Corbeille : sans effet sans trasher. Un échec arrête tout, rien n'est écrit.
	disposal, err := s.trashFirst(ctx, target)
	if err != nil {
		slog.ErrorContext(ctx, "media_delete: Corbeille échouée, ni masquage ni retrait",
			"err", err, "file_path", req.FilePath, "trashed", disposal.trashedCount)
		return nil, err
	}

	// 4. Masquage durable.
	if err := s.markMediaDeleted(ctx, deleter, req.FilePath); err != nil {
		if disposal.trashedCount > 0 {
			slog.WarnContext(ctx, "media_delete: originaux dans la Corbeille, média non masqué",
				"err", err, "file_path", req.FilePath, "trashed", disposal.trashedCount)
		}
		return nil, err
	}

	// 5. Retrait disque : il garantit que l'URL directe ne sert plus rien.
	removed, trashed, err := s.removeRemaining(ctx, target, disposal)
	if err != nil {
		slog.ErrorContext(ctx, "media_delete: média masqué mais fichiers non retirés",
			"err", err, "file_path", req.FilePath, "files_removed", removed)
		return nil, fmt.Errorf("media_delete remove files: %w", err)
	}

	slog.InfoContext(ctx, "media_delete: média supprimé",
		"file_path", req.FilePath, "owner", target.OwnerSlug,
		"requester", req.RequesterSlug, "is_admin", req.RequesterIsAdmin,
		"files_removed", removed, "trashed", trashed)

	return &domain.MediaDeleteResponse{
		FilePath:     req.FilePath,
		Deleted:      true,
		FilesRemoved: removed,
		Trashed:      trashed,
	}, nil
}

// mediaDisposal : ce que l'envoi à la Corbeille a décidé des originaux.
type mediaDisposal struct {
	// permanent : originaux sans Corbeille (support amovible ou réseau, chemin non résolvable),
	// à effacer définitivement après le masquage.
	permanent    []string
	trashedCount int
	// viaTrash : un trasher était configuré ; sinon removeRemaining retire tous les chemins stockés.
	viaTrash bool
}

// trashFirst envoie chaque original à la Corbeille, avant toute écriture en base. Sans trasher, rend
// un état vide. La première erreur arrête la boucle : les originaux suivants ne sont pas touchés, et
// l'état rendu avec elle porte ceux déjà partis.
func (s *MediaService) trashFirst(ctx context.Context, target domain.MediaDeletionTarget) (mediaDisposal, error) {
	if s.fileTrasher == nil {
		return mediaDisposal{}, nil
	}
	d := mediaDisposal{viaTrash: true}
	for _, p := range target.OriginalPaths() {
		ok, err := s.fileTrasher.TrashMediaFile(ctx, p)
		if err != nil {
			slog.ErrorContext(ctx, "media_delete: envoi à la Corbeille échoué, original laissé en place",
				"err", err, "stored", p)
			return d, fmt.Errorf("media_delete trash original: %w", err)
		}
		if ok {
			d.trashedCount++
			continue
		}
		d.permanent = append(d.permanent, p)
	}
	return d, nil
}

// removeRemaining retire de leur dossier les fichiers que la Corbeille n'a pas pris. Sans trasher :
// tous les chemins stockés, définitivement. Avec trasher : les originaux sans Corbeille, la miniature
// et la version convertie. Rend le nombre de fichiers retirés (Corbeille comprise) et si au moins un
// original est dans la Corbeille.
func (s *MediaService) removeRemaining(
	ctx context.Context,
	target domain.MediaDeletionTarget,
	d mediaDisposal,
) (removed int, trashed bool, err error) {
	if !d.viaTrash {
		removed, err = s.fileRemover.RemoveMediaFiles(ctx, target.StoredPaths())
		return removed, false, err
	}
	removed, err = s.fileRemover.RemoveMediaFiles(ctx, append(d.permanent, target.ForgetPaths()...))
	return removed + d.trashedCount, d.trashedCount > 0, err
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

// authorizeMediaAssociation applique à l'association média -> match la règle de la
// suppression (domain.CanModifyMedia : propriétaire ou admin), jugée sur le
// propriétaire réel du média (media_files.player_slug) chargé par son chemin COMPLET.
// Sans cette vérification, n'importe quel joueur déplaçait le clip d'autrui vers un
// autre match, pour tout le monde (l'association manuelle prime dans la vue _latest).
func (s *MediaService) authorizeMediaAssociation(ctx context.Context, req domain.MediaAssociateRequest) error {
	if !req.AuthEnforced {
		return nil
	}
	deleter, ok := s.repo.(port.MediaDeleter)
	if !ok {
		return domain.ErrInternal("media association not supported by this repository")
	}
	target, err := deleter.LoadMediaForDeletion(ctx, req.FilePath)
	if err != nil {
		return err
	}
	if !domain.CanModifyMedia(target.OwnerSlug, req.RequesterSlug, req.RequesterIsAdmin, req.AuthEnforced) {
		slog.WarnContext(ctx, "media_associate refusé: ni propriétaire ni admin",
			"file_path", req.FilePath, "owner", target.OwnerSlug,
			"requester", req.RequesterSlug, "is_admin", req.RequesterIsAdmin)
		return domain.ErrForbidden("Only the owner of this media or an administrator can associate it with a match.")
	}
	return nil
}
