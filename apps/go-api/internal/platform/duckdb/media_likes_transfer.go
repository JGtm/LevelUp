package duckdb

import (
	"context"
	"fmt"

	"levelup/go-api/internal/port"
)

// TransferMediaLikes fait suivre les likes actifs d'un média qui change de
// file_path : la clé des likes est le chemin stocké, un changement de chemin les
// rendrait sinon invisibles.
//
// Deux INSERT purs dans media_likes_history (append-only, jamais d'UPDATE ni de
// DELETE) : un like actif par liker sur newPath, puis un retrait par liker sur
// oldPath, pour que la lecture par media_likes_latest ne compte plus rien sur
// l'ancien chemin. L'ordre est celui-là : les deux événements d'une même
// transaction partagent leur written_at, la vue départage par id.
//
// Chemins identiques : aucune écriture. Un transfert sur soi-même écrirait un
// retrait sur la clé qui vient de recevoir ses likes et les effacerait.
//
// exec est la transaction de l'appelant ; le commit et son CHECKPOINT lui
// appartiennent. Unique copie de ces deux requêtes (réassociation, relocalisation
// par la passe de réconciliation, fusion de fiches).
func TransferMediaLikes(ctx context.Context, exec port.DBExecutor, oldPath, newPath string) error {
	if oldPath == newPath {
		return nil
	}
	if _, err := exec.ExecContext(ctx, `
		INSERT INTO media_likes_history (media_path, liker_slug, liker_gamertag, is_liked, liked_at)
		SELECT ?, liker_slug, liker_gamertag, TRUE, liked_at FROM media_likes_latest
		 WHERE media_path = ? AND is_liked = TRUE`, newPath, oldPath); err != nil {
		return fmt.Errorf("TransferMediaLikes: likes vers le nouveau chemin: %w", err)
	}
	if _, err := exec.ExecContext(ctx, `
		INSERT INTO media_likes_history (media_path, liker_slug, liker_gamertag, is_liked, liked_at)
		SELECT ?, liker_slug, NULL, FALSE, NULL FROM media_likes_latest
		 WHERE media_path = ? AND is_liked = TRUE`, oldPath, oldPath); err != nil {
		return fmt.Errorf("TransferMediaLikes: retrait sur l'ancien chemin: %w", err)
	}
	return nil
}
