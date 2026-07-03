// Package sync — engine_introspect.go : accesseurs publics réservés aux
// tests de parité runtime. NE PAS UTILISER en production.
//
// Ces accesseurs existent pour permettre aux tests du package scheduler
// (et autres callers de BuildEngine) de vérifier que les options critiques
// d'un *SyncEngine sont bien câblées — c'est l'unique parade contre une
// régression silencieuse type "incident 2026-05-26" où un nouveau call
// site oubliait .WithSharedProvider et tombait en legacy.
//
// Si un nouveau champ optionnel est ajouté à SyncEngine, l'ajouter ici en
// même temps + ajouter une assertion dans
// scheduler/auto_sync_build_engine_test.go. Le test échouera si quelqu'un
// retire un With... de BuildEngine.
package sync

// HasSharedProvider retourne true si .WithSharedProvider a été appelé avec
// une instance non-nil. Test-only.
func (e *SyncEngine) HasSharedProvider() bool {
	return e.sharedProvider != nil
}

// HasFriendsLoader retourne true si .WithFriendsLoader a été appelé avec
// une closure non-nil. Test-only.
func (e *SyncEngine) HasFriendsLoader() bool {
	return e.friendsLoader != nil
}

// HasPostSyncRunner retourne true si .WithPostSyncRunner a été appelé.
// Test-only.
func (e *SyncEngine) HasPostSyncRunner() bool {
	return e.postSyncRunner != nil
}

// HasMediaScanHook retourne true si .WithMediaScanHook a été appelé.
// Test-only.
func (e *SyncEngine) HasMediaScanHook() bool {
	return e.mediaHook != nil
}

// HasCustomClient retourne true si .SetCustomClient a été appelé (typique
// pour pinned PooledHaloClient en production). Test-only.
func (e *SyncEngine) HasCustomClient() bool {
	return e.customClient != nil
}

// HasBatchQueue retourne true si .WithBatchQueue a été appelé. Test-only.
func (e *SyncEngine) HasBatchQueue() bool {
	return e.batchQueue != nil
}

// CSRSeasonIDForTest retourne le csrSeasonID câblé. Test-only.
func (e *SyncEngine) CSRSeasonIDForTest() string {
	return e.csrSeasonID
}

// GamertagForTest retourne le gamertag du moteur. Test-only.
func (e *SyncEngine) GamertagForTest() string {
	return e.gamertag
}

// XUIDForTest retourne le xuid du moteur. Test-only.
func (e *SyncEngine) XUIDForTest() string {
	return e.xuid
}

// TitleSlugForTest retourne le titleSlug du moteur (MT-11 / PMT-3). Test-only.
func (e *SyncEngine) TitleSlugForTest() string {
	return e.titleSlug
}

// PlayerDBPathForTest retourne le chemin de la DB player du moteur. Test-only.
func (e *SyncEngine) PlayerDBPathForTest() string {
	return e.playerDBPath
}
