// Package service — media_service_missing.go : les actions de l'utilisateur sur les médias dont aucun
// fichier n'existe plus sur le disque (liste, fusion dans une autre fiche, oubli, confirmation d'une
// disparition massive) et l'ouverture de l'explorateur sur le fichier d'un média.
//
// L'autorisation est celle des autres écritures de médias (domain.CanModifyMedia : propriétaire ou
// admin), jugée sur le propriétaire réel de la fiche chargée par son chemin COMPLET, jamais sur un
// chemin libre du client. Les écritures suivent l'ordre de la suppression : base d'abord (transaction
// validée avec CHECKPOINT), disque ensuite, et seulement ce que l'application a créé (miniature,
// version convertie) : l'original de l'utilisateur n'est jamais touché ici.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/port"
)

// maxMergeCandidates borne la liste des fiches proposées à la fusion.
const maxMergeCandidates = 50

// WithMediaFileRevealer injecte l'ouverture de l'explorateur et le dossier des captures qui sert à
// résoudre le chemin stocké d'un média (relu à chaque appel : réglage modifiable à chaud). Sans lui,
// RevealMedia répond comme une route inexistante (capacité absente du déploiement).
func WithMediaFileRevealer(revealer port.MediaFileRevealer, capturesBase func() string) MediaOption {
	return func(s *MediaService) {
		s.fileRevealer = revealer
		s.capturesBase = capturesBase
	}
}

// missingStore rend le dépôt vu comme port.MediaMissingStore.
func (s *MediaService) missingStore() (port.MediaMissingStore, error) {
	store, ok := s.repo.(port.MediaMissingStore)
	if !ok {
		return nil, domain.ErrInternal("missing media not supported by this repository")
	}
	return store, nil
}

// requireMissingWrites vérifie que le service peut écrire en base ET retirer des fichiers : sans
// l'un des deux, une fusion ou un oubli laisserait un état à moitié fait.
func (s *MediaService) requireMissingWrites() error {
	if s.fileRemover == nil {
		return domain.ErrInternal("missing media not configured (no disk access)")
	}
	if s.txOpener == nil {
		return domain.ErrInternal("missing media not configured (no writer)")
	}
	return nil
}

// forbidMissing journalise et rend le refus d'agir sur la fiche d'un autre joueur.
func forbidMissing(ctx context.Context, action, filePath, owner string, who domain.MediaRequester) error {
	slog.WarnContext(ctx, "media_missing refusé: ni propriétaire ni admin",
		"action", action, "file_path", filePath, "owner", owner,
		"requester", who.RequesterSlug, "is_admin", who.RequesterIsAdmin)
	return domain.ErrForbidden("Only the owner of this media or an administrator can do this.")
}

// ListMissingMedia liste les médias introuvables du joueur de la route, avec l'état du dossier des
// captures constaté par sa dernière passe d'indexation (mémoire du process, ops.GetRootState).
func (s *MediaService) ListMissingMedia(
	ctx context.Context,
	req domain.MediaMissingListRequest,
) (*domain.MediaMissingResponse, error) {
	store, err := s.missingStore()
	if err != nil {
		return nil, err
	}
	player := s.repo.CurrentPlayerSlug()
	if !req.CanModify(player) {
		return nil, forbidMissing(ctx, "list", "", player, req.MediaRequester)
	}
	items, err := store.ListMissingMedia(ctx)
	if err != nil {
		return nil, err
	}
	state := ops.GetRootState(ctxkeys.TitleSlug(ctx), player)
	return &domain.MediaMissingResponse{
		Items:              items,
		RootUnavailable:    state.Unavailable,
		BulkMissingPending: state.BulkPending,
	}, nil
}

// ListMergeCandidates liste les fiches présentes dans lesquelles la fiche introuvable FilePath peut
// se fondre.
func (s *MediaService) ListMergeCandidates(
	ctx context.Context,
	req domain.MediaMergeCandidatesRequest,
) (*domain.MediaMergeCandidatesResponse, error) {
	if req.FilePath == "" {
		return nil, domain.ErrBadRequest("file_path is required")
	}
	store, err := s.missingStore()
	if err != nil {
		return nil, err
	}
	missing, err := store.LoadMissingTarget(ctx, req.FilePath)
	if err != nil {
		return nil, err
	}
	if !req.CanModify(missing.OwnerSlug) {
		return nil, forbidMissing(ctx, "candidates", req.FilePath, missing.OwnerSlug, req.MediaRequester)
	}
	candidates, err := store.ListMergeCandidates(ctx, missing, maxMergeCandidates)
	if err != nil {
		return nil, err
	}
	return &domain.MediaMergeCandidatesResponse{
		FilePath:   req.FilePath,
		CaptureUTC: missing.CaptureUTC,
		Candidates: candidates,
	}, nil
}

// MergeMissingMedia fond la fiche introuvable FilePath dans la fiche présente TargetFilePath : ses
// likes et son rattachement manuel passent à la cible, puis elle est masquée et ses fichiers créés
// par l'application sont retirés (hors ceux que la cible utilise elle-même).
func (s *MediaService) MergeMissingMedia(
	ctx context.Context,
	req domain.MediaMergeRequest,
) (*domain.MediaMergeResponse, error) {
	if req.FilePath == "" || req.TargetFilePath == "" {
		return nil, domain.ErrBadRequest("file_path and target_file_path are required")
	}
	if req.FilePath == req.TargetFilePath {
		return nil, domain.ErrBadRequest("target_file_path must differ from file_path")
	}
	store, err := s.missingStore()
	if err != nil {
		return nil, err
	}
	if err := s.requireMissingWrites(); err != nil {
		return nil, err
	}
	missing, target, err := s.loadMergePair(ctx, store, req)
	if err != nil {
		return nil, err
	}
	if err := s.commitMerge(ctx, missing, target); err != nil {
		return nil, err
	}
	removed, err := s.fileRemover.RemoveMediaFiles(ctx, withoutPaths(missing.ForgetPaths(), target.StoredPaths()))
	if err != nil {
		slog.ErrorContext(ctx, "media_merge: fiche fondue mais fichiers non retirés",
			"err", err, "file_path", req.FilePath, "target", req.TargetFilePath, "files_removed", removed)
		return nil, fmt.Errorf("media_merge remove files: %w", err)
	}
	slog.InfoContext(ctx, "media_merge: fiche introuvable fondue",
		"file_path", req.FilePath, "target", req.TargetFilePath,
		"requester", req.RequesterSlug, "files_removed", removed)
	return &domain.MediaMergeResponse{
		FilePath: req.FilePath, TargetFilePath: req.TargetFilePath, Merged: true, FilesRemoved: removed,
	}, nil
}

// loadMergePair charge les deux fiches d'une fusion et en juge l'autorisation et la cohérence, dans
// cet ordre : 404 si la fiche n'est pas introuvable ou la cible pas présente, 403 si le demandeur ne
// peut modifier l'une des deux, 400 si les propriétaires ou les natures diffèrent.
func (s *MediaService) loadMergePair(
	ctx context.Context,
	store port.MediaMissingStore,
	req domain.MediaMergeRequest,
) (missing, target domain.MediaFileRef, err error) {
	if missing, err = store.LoadMissingTarget(ctx, req.FilePath); err != nil {
		return missing, target, err
	}
	if !req.CanModify(missing.OwnerSlug) {
		return missing, target, forbidMissing(ctx, "merge", req.FilePath, missing.OwnerSlug, req.MediaRequester)
	}
	if target, err = store.LoadMergeTarget(ctx, req.TargetFilePath); err != nil {
		return missing, target, err
	}
	if !req.CanModify(target.OwnerSlug) {
		return missing, target, forbidMissing(ctx, "merge", req.TargetFilePath, target.OwnerSlug, req.MediaRequester)
	}
	if !strings.EqualFold(missing.OwnerSlug, target.OwnerSlug) {
		return missing, target, domain.ErrBadRequest("media have different owners")
	}
	if missing.Kind != target.Kind {
		return missing, target, domain.ErrBadRequest("media have different kinds")
	}
	return missing, target, nil
}

// commitMerge écrit la fusion dans une transaction du writer partagé, validée avec CHECKPOINT
// (même garantie de durabilité que la suppression : sans elle, un redémarrage dans la fenêtre du
// WAL ressusciterait la fiche introuvable alors que ses fichiers vont être effacés).
func (s *MediaService) commitMerge(
	ctx context.Context,
	missing, target domain.MediaFileRef,
) error {
	tx, err := s.txOpener.BeginMediaTx(ctx, "media_merge")
	if err != nil {
		return err
	}
	// Le writer est rendu même si une écriture panique : Rollback est sans effet après Commit ou Rollback.
	defer func() { _ = tx.Rollback() }()
	if err := tx.ApplyMediaMerge(ctx, missing, target); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			slog.WarnContext(ctx, "media_merge: rollback échoué", "err", rbErr, "file_path", missing.FilePath)
		}
		slog.WarnContext(ctx, "media_merge rollback", "err", err, "file_path", missing.FilePath)
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("media_merge commit: %w", err)
	}
	return nil
}

// ForgetMissingMedia oublie la fiche introuvable FilePath : elle est masquée et seuls la miniature
// et la version convertie créées par l'application sont retirées. L'original de l'utilisateur n'est
// jamais touché (domain.MediaDeletionTarget.ForgetPaths).
func (s *MediaService) ForgetMissingMedia(
	ctx context.Context,
	req domain.MediaForgetRequest,
) (*domain.MediaForgetResponse, error) {
	if req.FilePath == "" {
		return nil, domain.ErrBadRequest("file_path is required")
	}
	store, err := s.missingStore()
	if err != nil {
		return nil, err
	}
	if _, ok := s.repo.(port.MediaDeleter); !ok {
		return nil, domain.ErrInternal("forgetting media not supported by this repository")
	}
	if err := s.requireMissingWrites(); err != nil {
		return nil, err
	}
	missing, err := store.LoadMissingTarget(ctx, req.FilePath)
	if err != nil {
		return nil, err
	}
	if !req.CanModify(missing.OwnerSlug) {
		return nil, forbidMissing(ctx, "forget", req.FilePath, missing.OwnerSlug, req.MediaRequester)
	}
	if err := s.markMediaDeleted(ctx, req.FilePath); err != nil {
		return nil, err
	}
	removed, err := s.fileRemover.RemoveMediaFiles(ctx, missing.ForgetPaths())
	if err != nil {
		slog.ErrorContext(ctx, "media_forget: fiche oubliée mais fichiers non retirés",
			"err", err, "file_path", req.FilePath, "files_removed", removed)
		return nil, fmt.Errorf("media_forget remove files: %w", err)
	}
	slog.InfoContext(ctx, "media_forget: fiche introuvable oubliée",
		"file_path", req.FilePath, "requester", req.RequesterSlug, "files_removed", removed)
	return &domain.MediaForgetResponse{FilePath: req.FilePath, Forgotten: true, FilesRemoved: removed}, nil
}

// ConfirmBulkMissing confirme la disparition massive constatée pour le joueur de la route : la
// prochaine passe d'indexation applique le marquage suspendu. Sans disparition en attente, rien
// n'est enregistré (une confirmation sans objet serait consommée par une passe sans rapport).
func (s *MediaService) ConfirmBulkMissing(
	ctx context.Context,
	req domain.MediaConfirmBulkRequest,
) (*domain.MediaConfirmBulkResponse, error) {
	player := s.repo.CurrentPlayerSlug()
	if !req.CanModify(player) {
		return nil, forbidMissing(ctx, "confirm_bulk", "", player, req.MediaRequester)
	}
	title := ctxkeys.TitleSlug(ctx)
	pending := ops.GetRootState(title, player).BulkPending
	if pending == 0 {
		return &domain.MediaConfirmBulkResponse{}, nil
	}
	ops.ConfirmBulkMissing(title, player)
	slog.InfoContext(ctx, "media_missing: disparition massive confirmée",
		"titleSlug", title, "player", player, "pending", pending, "requester", req.RequesterSlug)
	return &domain.MediaConfirmBulkResponse{Confirmed: true, BulkMissingPending: pending}, nil
}

// RevealMedia ouvre l'explorateur sur le fichier du média : son original (source_path) quand il est
// renseigné, sinon file_path. Le chemin vient de la fiche en base, jamais du client, et la fiche peut
// être présente ou introuvable (l'explorateur s'ouvre alors sur le dossier existant le plus proche).
func (s *MediaService) RevealMedia(
	ctx context.Context,
	req domain.MediaRevealRequest,
) (*domain.MediaRevealResponse, error) {
	if s.fileRevealer == nil {
		return nil, domain.ErrNotFound("capability", "reveal_files_in_explorer")
	}
	if req.FilePath == "" {
		return nil, domain.ErrBadRequest("file_path is required")
	}
	deleter, ok := s.repo.(port.MediaDeleter)
	if !ok {
		return nil, domain.ErrInternal("revealing media not supported by this repository")
	}
	target, err := deleter.LoadMediaForDeletion(ctx, req.FilePath)
	if err != nil {
		return nil, err
	}
	if !req.CanModify(target.OwnerSlug) {
		return nil, forbidMissing(ctx, "reveal", req.FilePath, target.OwnerSlug, req.MediaRequester)
	}
	abs, err := s.revealPath(target)
	if err != nil {
		return nil, err
	}
	if err := s.fileRevealer.Reveal(ctx, abs); err != nil {
		slog.ErrorContext(ctx, "media_reveal: ouverture de l'explorateur échouée",
			"err", err, "file_path", req.FilePath, "abs", abs)
		return nil, fmt.Errorf("media_reveal: %w", err)
	}
	return &domain.MediaRevealResponse{Revealed: true}, nil
}

// revealPath rend le chemin absolu du fichier à montrer pour la fiche target.
func (s *MediaService) revealPath(target domain.MediaDeletionTarget) (string, error) {
	stored := target.SourcePath
	if stored == "" {
		stored = target.FilePath
	}
	base := ""
	if s.capturesBase != nil {
		base = s.capturesBase()
	}
	abs := ops.MediaPathStore{CapturesBase: base}.ToAbs(stored)
	if abs == "" || !filepath.IsAbs(abs) {
		return "", domain.ErrNotFound("media file", target.FilePath)
	}
	return abs, nil
}

// withoutPaths rend paths privé des chemins de excluded, ordre conservé.
func withoutPaths(paths, excluded []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if !slices.Contains(excluded, p) {
			out = append(out, p)
		}
	}
	return out
}
