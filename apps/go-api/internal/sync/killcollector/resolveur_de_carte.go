package killcollector

// resolveur_de_carte.go — LE RESOLVEUR DE CARTE DU POST-SYNC TRADUIT UN UUID BRUT (revue du
// correctif J7, 2026-09-27).
//
// # LA REGRESSION QU IL FERME
//
// Quand la synchro n a pas resolu le libelle de la carte (traduction absente au moment de
// l ingestion : pool GameCMS non cable, echec reseau, `LEVELUP_SYNC_RESOLVE_ASSETS=0`),
// `match_registry.map_name` porte l UUID de l asset (`ExtractRegistry`, repli nom -> id). Le
// resolveur du post-sync etait construit SANS metadonnees : son seul candidat etait cet UUID,
// absent du catalogue de bornes. Avant la carte obligatoire, le match etait decode (aux largeurs
// de Cliffhanger) ; depuis, il etait ecarte POUR TOUJOURS par le serveur alors que sa carte est
// connue d `asset_translations` — et que le backfill hors ligne, lui, la traduit.
//
// # LES METADONNEES SONT EMPRUNTEES, JAMAIS OUVERTES
//
// Le processus tient deja `metadata.duckdb` (serveur : handle RW partage ouvert au demarrage ;
// CLI : `SyncEngine.run`). En ouvrir un second handle est interdit (ADR 0013/0016, « different
// configuration »). Le resolveur EMPRUNTE donc le handle du cache du processus
// (`duckdb.LookupCachedDB`, emprunt non possedant, le meme que les citations du post-sync), a
// CHAQUE resolution : un handle ferme entre deux cycles n est jamais garde. Aucun handle tenu :
// le libelle brut seul, comme avant, et un WARN par resolveur (par cycle), jamais un silence.

import (
	"context"
	"log/slog"
	"sync"

	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
)

// ResolveurDeCartePostSync rend le resolveur de carte de l etape 1.57 : `match_registry` par le
// lecteur shared du cycle, et les traductions d assets par le handle metadata QUE LE PROCESSUS
// TIENT DEJA a `metadataDBPath` (jamais ouvert ici).
func ResolveurDeCartePostSync(shared duckdb.SharedReader, metadataDBPath string) port.ReplayMapNameRepo {
	return &resolveurEmprunteur{shared: shared, cheminMeta: metadataDBPath}
}

type resolveurEmprunteur struct {
	shared     duckdb.SharedReader
	cheminMeta string
	signale    sync.Once
}

// depot construit le resolveur de la base avec le handle metadata emprunte, ou sans.
func (r *resolveurEmprunteur) depot(ctx context.Context) *duckdb.ReplayMapRepo {
	meta, tenu := duckdb.LookupCachedDB(r.cheminMeta)
	if !tenu {
		r.signale.Do(func() {
			slog.WarnContext(ctx, "killsource: metadonnees non tenues par le processus — une carte "+
				"nommee par un UUID brut au registre ne sera pas traduite ce cycle",
				"metadata", r.cheminMeta)
		})
		meta = nil
	}
	return duckdb.NewReplayMapRepo(r.shared, meta)
}

func (r *resolveurEmprunteur) MapKeysForMatch(ctx context.Context, matchID string) (port.MatchMapKeys, error) {
	return r.depot(ctx).MapKeysForMatch(ctx, matchID)
}

func (r *resolveurEmprunteur) MapKeysForMap(ctx context.Context, mapID string) (port.MatchMapKeys, error) {
	return r.depot(ctx).MapKeysForMap(ctx, mapID)
}
