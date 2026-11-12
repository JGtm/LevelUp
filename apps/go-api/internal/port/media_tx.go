package port

// media_tx.go — la transaction des écritures médias durables (like, suppression, fusion d'une fiche
// introuvable). Le dépôt ouvre la transaction sur le writer partagé de la base sociale, la valide et force le
// CHECKPOINT (ADR 0021 / 0022) : une réponse de succès vaut « écrit et durable ». Aucun type SQL ici : le
// service ne manipule que des opérations métier.

import (
	"context"

	"levelup/go-api/internal/domain"
)

// MediaTx est une transaction ouverte sur le writer partagé. Elle se termine par Commit ou par Rollback,
// et un seul des deux : l'un comme l'autre libère le writer.
type MediaTx interface {
	// SetMediaLike écrit l'événement de like du liker et contrôle, dans la transaction, que le média est
	// visible. Rend false (sans rien écrire) si le média est inconnu ou supprimé.
	SetMediaLike(ctx context.Context, filePath, likerSlug, likerGamertag string, liked bool) (bool, error)
	// MarkMediaDeleted masque un média (status='deleted'), sans jamais supprimer la ligne ni toucher aux
	// tables append-only (domain/media_delete.go). Rend false si aucune ligne visible ne correspondait.
	MarkMediaDeleted(ctx context.Context, filePath string) (bool, error)
	// ApplyMediaMerge fond la fiche introuvable missing dans la fiche présente target : likes transférés,
	// rattachement manuel repris, fiche introuvable masquée. Aucune ligne n'est supprimée.
	ApplyMediaMerge(ctx context.Context, missing, target domain.MediaFileRef) error
	// Commit valide la transaction puis exécute un CHECKPOINT immédiat (jamais un Commit nu : un
	// redémarrage dans la fenêtre du WAL ferait disparaître l'écriture après coup), et libère le writer.
	Commit(ctx context.Context) error
	// Rollback annule la transaction et libère le writer.
	Rollback() error
}

// MediaTxOpener ouvre les transactions des écritures médias durables.
type MediaTxOpener interface {
	// BeginMediaTx acquiert le writer partagé et y ouvre une transaction. op nomme l'opération dans le
	// message d'erreur d'ouverture (« <op> begin tx »). L'échec de l'acquisition du writer est rendu tel
	// quel (dblease.ErrDBLocked reconnaissable).
	BeginMediaTx(ctx context.Context, op string) (MediaTx, error)
}
