// Package duckdb — social_persister_iface.go : interface SocialPersister
// utilisée par PlayerDB pour router les écritures shared_social vers le
// pattern Collect→Persist (ADR 0020).
//
// L'interface vit ici (pas dans internal/persist) pour éviter le cycle
// d'import : internal/persist importe déjà internal/platform/duckdb via
// combined_persister.go (pour le pool batch sync engine).
//
// L'implémentation concrète persist.SharedSocialPersister satisfait cette
// interface STRUCTURELLEMENT (Go duck typing) — pas de déclaration
// d'implémentation requise. Injection au boot dans main.go :
//
//	pdb.SocialPersister = persist.NewSharedSocialPersister(pdb.SharedSocial.SQLDb())
//
// Les repos qui écrivent sur shared_social DOIVENT passer par cette
// interface. Si nil (initialisation pas faite ou SharedSocial nil), le repo
// peut retomber sur l'ancien chemin db.Exec — mais cette dégradation sera
// supprimée en Phase 6 (sentinel parse-AST).

package duckdb

import (
	"context"
	"database/sql"
)

// SocialPersister est l'API d'écriture sur shared_social.duckdb.
//
// Le type *Batch est une interface vide (any) ici : on délègue au caller la
// construction du batch concret (persist.SharedSocialBatch) et la garantie
// que ce batch est compatible avec l'implémentation injectée. C'est moche
// mais c'est le seul moyen sans cycle d'import.
//
// Alternative envisagée : déplacer SharedSocialBatch dans un sous-package
// shared (sans dépendance duckdb). Non fait dans cette session pour limiter
// le scope du refactor.
type SocialPersister interface {
	// PersistBatch persiste un *persist.SharedSocialBatch (typé en any pour
	// éviter le cycle d'import). L'implémentation concrète fait le cast en
	// interne.
	PersistBatch(ctx context.Context, batch any) error
}

// SocialPersisterFactory est un hook configuré par main.go au boot pour
// permettre à openPlayerDB d'instancier un SocialPersister sans importer
// internal/persist (qui causerait un cycle).
//
// Wiring attendu dans main.go :
//
//	duckdb.SocialPersisterFactory = func(db *sql.DB) duckdb.SocialPersister {
//	    return persist.NewSharedSocialPersister(db)
//	}
//
// Si nil (cas tests, bootstrap CLI), pdb.SocialPersister reste nil et les
// repos retombent sur leur chemin legacy db.Exec.
var SocialPersisterFactory func(db *sql.DB) SocialPersister
