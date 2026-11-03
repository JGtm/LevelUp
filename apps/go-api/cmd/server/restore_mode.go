// restore_mode.go — le mode -mode restore : restaure un jeu de sauvegarde dans une racine de données, puis sort.
//
// L'installeur de l'application ne livre pas levelup.exe : la restauration (« disque mort, nouveau PC ») se lance
// avec le serveur livré, sans configuration serveur ni base ouverte ni port lié. La commande est celle de
// parquetbackup.RunRestoreCommand, la même que l'alias `levelup backup restore` : elle prend le verrou
// d'instance de la racine cible et refuse tant qu'un serveur de cette racine tourne, quel que soit son profil.
package main

import (
	"context"
	"fmt"
	"os"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ops/parquetbackup"
)

// restoreModeFailure : code de sortie d'une restauration refusée ou en échec.
const restoreModeFailure = 1

// runRestoreMode exécute la restauration demandée par les drapeaux et rend le code de sortie du processus. Les
// racines de ressources et de journaux viennent de l environnement, comme pour tout processus du serveur. Le bilan va
// sur stderr : stdout est réservé aux événements du contrat de la coquille (stdout_guard_test.go).
func runRestoreMode(ctx context.Context, f serverFlags) int {
	if err := parquetbackup.RunRestoreCommand(ctx, f.Restore, config.ResolveRoots(), os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "restauration : %v\n", err)
		return restoreModeFailure
	}
	return 0
}
