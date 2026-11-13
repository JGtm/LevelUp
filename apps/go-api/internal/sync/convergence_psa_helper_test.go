package sync

import (
	"context"
	"database/sql"
)

// convergePSA enchaîne les trois phases de la convergence PSA (convergePSACollect, flushAliasPairs,
// settlePSA) sur une base partagée donnée, sans burst d'écriture ni rapport : enrobage des tests des
// phases. Le post-sync les enchaîne par runPSAConvergence ; rend le nombre de matchs réglés.
func convergePSA(ctx context.Context, playerDB, sharedDB *sql.DB, client HaloClient, xuid string, ids []string) int {
	fetched := convergePSACollect(ctx, client, xuid, ids)
	return settlePSA(ctx, playerDB, xuid, flushAliasPairs(ctx, sharedDB, xuid, fetched))
}
