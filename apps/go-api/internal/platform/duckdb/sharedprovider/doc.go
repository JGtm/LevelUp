// Package sharedprovider — owner unique du handle shared_matches_v2.duckdb par
// chemin, avec swap dynamique RO ↔ RW pour résoudre la limite native du driver
// duckdb-go v2 : "Can't open a connection to same database file with a
// different configuration than existing connections".
//
// Pattern d'usage côté serveur HTTP :
//
//	mgr := sharedprovider.NewManager()
//	provider, err := mgr.For(sharedPath)
//	if err != nil { ... }
//	defer provider.Close()
//
//	// Lectures (handlers HTTP, pool joueur) :
//	db, err := provider.Get(ctx)
//
//	// Écritures (sync engine) — à venir au commit 3 :
//	w, err := provider.AcquireWriter(ctx)
//	defer w.Release()
//
// État : livré. AcquireWriter, WriterHandle et le mécanisme de swap RO↔RW sont
// implémentés (cf. l'interface Provider ci-dessous).
//
// Voir docs/adr/0016-shared-db-provider-b-swap.md pour la motivation
// architecturale complète.
package sharedprovider
