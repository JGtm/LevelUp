# Plan : médias alignés sur le disque + révision des KPI

Rédigé le 2026-11-08. Statut : **à valider par l'utilisateur, exécution non lancée.**
Exécutants visés : Sonnet (défaut) et Haiku (étapes marquées `[Haiku]`). Ce plan se suffit à lui-même.

---

## 1. Objectifs et critères de succès

**Partie M, médias.** L'explorateur Windows reste le maître, l'app s'y adapte.

| Geste de l'utilisateur | Résultat attendu |
|---|---|
| Renommer ou déplacer un média dans le dossier des captures | Même fiche : likes, rattachements et miniature conservés, aucun doublon |
| Supprimer un média dans l'explorateur | Disparaît de la galerie, listé dans « Fichiers introuvables » |
| Supprimer dans l'explorateur l'original d'une vidéo convertie | La vidéo reste dans la galerie et se lit depuis sa version convertie |
| Remettre le fichier | Réapparaît seul |
| Clip ré-encodé et renommé (contenu différent) | Deux fiches ; depuis « Fichiers introuvables », l'ancienne se fond dans la nouvelle avec ses likes et son match |
| Débrancher le disque / dossier inaccessible | Rien n'est marqué, un bandeau prévient |
| Retirer d'un coup au moins la moitié d'un dossier d'au moins 20 médias | Rien n'est masqué tout de suite ; un bandeau demande confirmation, un clic applique |
| Deux fichiers de même nom dans deux sous-dossiers | Deux médias distincts, chacun avec sa miniature et sa conversion |
| Synchroniser sans changement sur le disque | Aucun fichier indexé relu en entier |
| Supprimer un média depuis l'app sur le bureau | Message explicite avant confirmation ; l'original part à la Corbeille de Windows si le disque en a une ; le média ne revient pas au scan suivant, mais revient si l'original est restauré |

**Partie K, KPI.** Après une mise à jour qui change une formule de LUSR ou de note de performance, tout l'historique de chaque joueur est recalculé seul, au démarrage, sans outil manuel.

---

## 2. Contrat d'exécution

Le skill `plan-execution` s'applique intégralement. Avant de coder, le lire, puis lire la section 9 (journal) de ce plan.

- **Ordre strict** : M0, M1 … M8, K1, K2, K3, Z. Une étape est close quand : gate passé, cases statuées, plan mis à jour et commité, entrée `.ai/thought_log.md`, point d'étape à l'utilisateur.
- **Statuts** : `[x]` fait et vérifié ; `[~]` couvert par un autre item (dire lequel) ; `[!]` non traité (justification au journal). Aucune case vide à la clôture.
- **Rouvrir chaque fichier cité avant de le modifier** : les numéros de ligne datent du 2026-11-08.
- **Hors périmètre** : noter en section 8, ne pas corriger.
- **Reprise de session** : lire la section 9, reprendre à la première case non statuée.
- **Blocage ou comportement qui contredit le plan** : s'arrêter, noter au journal, prévenir. Ne pas adapter le périmètre en silence. Ne jamais retoucher un test pour le faire passer, sauf si le plan dit explicitement que son comportement change.

### Interdits, à relire à chaque étape

1. Jamais d'`UPDATE` de `media_files.file_stem`, `player_slug`, `created_at` : colonnes indexées, risque de corruption DuckDB (bug ART #23645). `file_stem` reste figé à sa valeur d'insertion.
2. Jamais de `DELETE FROM media_files`, ni d'`UPDATE` ou `DELETE` sur `media_likes_history` (INSERT seulement).
3. Jamais d'index neuf sur une colonne qui change.
4. Toute lecture de `media_files` porte un prédicat (`MediaVisiblePredicate` ou `MediaNotDeletedPredicate`) ; garde-rail : `TestAllMediaFilesReadsFilterDeleted`.
5. Tout processus enfant passe par `internal/platform/childproc` (garde-rail `childproc_single_spawn_test.go`).
6. Pas de `slug == "..."` : brancher sur les capacités. Pas de `.DemoMode`, `.AuthMode` ni `LEVELUP_ENV` hors du résolveur (ADR 0044).
7. Pas de texte français ou anglais en dur côté Go exposé à l'écran ; tout texte web existe en `fr` et `en`.
8. Pas de couleur hex ni de classe Tailwind couleur dans `features/` : tokens sémantiques seulement (`success`, `warning`, `destructive`, `muted`, `primary`…).
9. Commentaires : le contrat et le pourquoi, pas d'historique, **aucune citation de `.ai/`** (ratchet à plafond 0). Pas d'emojis.
10. Seuils : fonction ≤ 80 L, ≤ 5 paramètres, complexité ≤ 12, fichier ≤ 500 L. `internal/ops/media.go` est déjà à 563 L : il ne doit pas grandir, le neuf va dans de nouveaux fichiers.
11. Logs : `slog.*Context(ctx, ...)` structurés, jamais d'erreur avalée.
12. Git : jamais `git stash`, jamais `git add -A` (ajouter fichier par fichier), jamais sur `main`, jamais de push sans accord.
13. Essais : jamais de navigateur, de code d'appareil ni d'ouverture réelle de l'explorateur par un agent.

---

## 3. Décisions figées

Ne pas les rediscuter pendant l'exécution.

| # | Décision |
|---|---|
| D1 | L'identité d'un média est l'empreinte de son contenu (`file_hash`) ; le chemin est un attribut qui change. |
| D2 | Nouvelles colonnes de `media_files`, non indexées : `source_path VARCHAR`, `source_size BIGINT`, `source_mtime_ns BIGINT`. `source_path` = chemin stocké de l'original. NULL = aucun original attendu. |
| D3 | État « introuvable » : `status = 'missing'` (domaine : `domain.MediaStatusMissing`). Pas de nouvelle colonne, pour ne pas casser les 24 fichiers qui créent `media_files` à la main. Retour du fichier : `status` repasse à NULL. |
| D4 | Présence d'une ligne (décision utilisateur du 2026-11-08). Vidéo convertie (`hls_path` renseigné) : présente si la version convertie **ou** l'original existe. Autre média : présent si son fichier (`source_path`, sinon `file_path`) existe. Une vidéo convertie dont l'original disparaît reste visible et se lit depuis sa version convertie, car ne garder qu'un des deux formats est un nettoyage légitime ; `source_path` repasse à NULL. Si l'original est remis, il est retrouvé par empreinte. |
| D5 | Une relocalisation automatique ne se fait que sur empreinte identique, ou sur changement de format (même dossier, même nom sans extension, candidat unique). Jamais d'approximation. |
| D6 | Les likes sont indexés par `file_path` : tout changement de `file_path` transfère les likes par INSERT (voir M2). |
| D7 | Purge automatique : jamais. « Oublier » est un geste de l'utilisateur. |
| D8 | « Oublier » ne touche jamais l'original : il retire seulement les fichiers créés par l'app (miniature, HLS) et masque la ligne (`status='deleted'`). |
| D9 | Supprimer depuis l'app retire aussi l'original, pour que le média ne revienne pas au scan suivant (décision utilisateur du 2026-11-08). Sur le bureau, l'original part à la Corbeille de Windows, avec un message explicite avant confirmation. Support sans Corbeille (clé USB, disque réseau) : suppression définitive, annoncée dans ce message et dans le toast. Un échec de l'envoi à la Corbeille ne se transforme jamais en suppression définitive. Miniature et version convertie sont toujours effacées définitivement. Original restauré depuis la Corbeille : le média revient au scan suivant. En ligne : suppression définitive, comme aujourd'hui. |
| D10 | Garde-fou : marquage annulé si le dossier racine est indisponible. Marquage suspendu si au moins 20 lignes sont actives et que la moitié ou plus deviendrait introuvable en une passe (seules comptent les lignes qui ne sont pas déjà `missing`) : un bandeau demande confirmation, la confirmation applique le marquage à la passe suivante, lancée aussitôt. Les relocalisations restent appliquées dans tous les cas. |
| D11 | Pas de surveillance continue du dossier. Passe au démarrage (tâche de fond) et après chaque synchronisation. |
| D12 | Écran « Fichiers introuvables » : réutilise la coquille de `MediaMatchPicker`, extraite sans changement visuel. Aucun nouveau modèle d'interface. |
| D13 | Sur le bureau, la suppression de l'original après conversion est interdite (capacité `KeepMediaSources`, vraie sur le bureau), réglage masqué, variable d'environnement ignorée. |
| D14 | KPI : deux constantes de révision, une seule valeur stockée par joueur dans `sync_meta`, comparée sur « différent ». Tant que le recalcul n'a pas réussi, rien n'est noté. |
| D15 | KPI : aucun indicateur à l'écran, le recalcul dure quelques secondes. Une ligne dans les notes de version quand une formule change. Aucune compaction ajoutée. |
| D16 | La recherche manuelle est une **fusion** : un scan indexe déjà tout fichier inconnu, donc un clip ré-encodé a sa propre fiche. Les candidats sont les autres fiches présentes du joueur, de même nature. Confirmer transfère les likes et le rattachement courant de la fiche introuvable, manuel ou automatique, vers la fiche choisie où il devient manuel, sauf si celle-ci a déjà un rattachement manuel (le choix de l'utilisateur sur la cible prime), puis oublie la fiche introuvable (D8). |
| D17 | Vidéo convertie dont l'original est présent mais la copie de l'app absente : la ligne repasse en lecture directe (`file_path = source_path`, `hls_path` et `transcode_status` à NULL) et le balayage la reconvertit. |

---

## 4. Mise en place commune

- Décision de l'utilisateur du 2026-11-08 : travail sur la **branche courante** `feat/replay-vitesse-1-5`, dans le dépôt principal, sans worktree dédié. Ne pas toucher aux autres worktrees, ne pas changer de branche.
- Fichiers modifiés hors du plan (`.ai/BACKLOG.md`, `data/desktop/`, `data/levelup.lock`) : ne jamais les ajouter à un commit.
- La branche cible de la PR se décide à l'étape Z5, avec l'utilisateur.
- Cache dédié : `GOCACHE=%LOCALAPPDATA%\go-build-medias`, `GOLANGCI_LINT_CACHE=%LOCALAPPDATA%\golangci-medias`.
- Commits : `feat(medias Mx): ...` ou `feat(kpi Kx): ...`, un par étape au moins, sur demande d'accord préalable selon la règle 16 de `CLAUDE.md` (l'accord donné au lancement du plan vaut pour les commits d'étape dans le worktree).

---

## 5. Partie M : médias

### M0. Mise en place et témoins `[Haiku]` (rapide)

- [x] M0.1 Vérifier la branche courante (section 4) et commiter ce plan. Branche vérifiée : `feat/replay-vitesse-1-5`. Le plan est commité avec le témoin (commit unique de l'étape, décision de l'utilisateur).
- [x] M0.2 Témoin Go : relever la sortie de `cd apps/go-api && go test ./internal/ops/ ./internal/platform/duckdb/ ./internal/service/ ./internal/api/handlers/ -run 'Media|Reconcile' -count=1` dans `.ai/temoins_medias_avant.txt` (fichier non commité). Résultat : 4 paquets `ok`, `EXIT=0`, aucun test en échec.
- [x] M0.3 Témoin web : ajouter à `apps/web/src/features/media/MediaMatchPicker.test.tsx` un test `toMatchSnapshot()` du picker avec un candidat sélectionné (bouton de confirmation visible), `start_time: null` et `capture_utc: null` pour que l'instantané ne dépende pas du fuseau du poste. Lancer, commiter le snapshot **avant** toute modification du composant. Fait : test « témoin avant modification (M0) » ajouté, snapshot écrit dans `apps/web/src/features/media/__snapshots__/MediaMatchPicker.test.tsx.snap`, relancé deux fois, stable.

**Gate** : `cd apps/web && npx vitest run src/features/media/MediaMatchPicker.test.tsx` vert.

### M1. Colonnes, état « introuvable », prédicats (rapide)

- [x] M1.1 `internal/domain/media_delete.go` : ajouter `MediaStatusMissing = "missing"` ; champ `SourcePath string` dans `MediaDeletionTarget`.
- [x] M1.2 `MediaDeletionTarget.StoredPaths()` : inclure `SourcePath` juste après `FilePath`, sans doublon.
- [x] M1.3 Ajouter `ForgetPaths()` à côté : rend miniature + HLS, et `FilePath` seulement s'il est égal à `HLSPath`. Jamais `SourcePath` (D8).
- [x] M1.3b Ajouter `OriginalPaths()` : `SourcePath`, et `FilePath` s'il n'est pas égal à `HLSPath`, sans doublon ni valeur vide.
- [x] M1.4 `internal/ops/media_store.go`, `mediaFilesAddedColumns` : ajouter `source_path` (`colTypeVarchar`), `source_size` (`"BIGINT"`), `source_mtime_ns` (`"BIGINT"`).
- [x] M1.5 `internal/platform/duckdb/media_repo_delete.go` :
  - `MediaVisiblePredicate(alias)` exclut désormais `deleted` **et** `missing` : `COALESCE(status, '') NOT IN ('deleted', 'missing')` (valeurs via les constantes du domaine, COALESCE conservé).
  - Nouveau `MediaNotDeletedPredicate(alias)` : l'ancienne forme (`<> 'deleted'`).
- [x] M1.6 Remplacer `MediaVisiblePredicate` par `MediaNotDeletedPredicate` dans **exactement** : `loadKnownHashes` (`ops/media_store.go`), `LoadMediaForDeletion` et `MarkMediaDeleted` (`media_repo_delete.go`). Toutes les autres lectures gardent `MediaVisiblePredicate`.
- [x] M1.7 `LoadMediaForDeletion` : charger `source_path` par une requête séparée. Seule l'erreur « colonne inconnue » (`strings.Contains(err.Error(), "Binder Error")`, base jamais indexée) donne une chaîne vide ; toute autre erreur est rendue et fait échouer la suppression (sinon l'original reste sur le disque et le média revient). Test des deux cas.
- [x] M1.8 `media_repo_delete_guard_test.go` : ajouter `"MediaNotDeletedPredicate"` à `coveredByVisiblePredicate` ; `TestMediaVisiblePredicate_Forms` : vérifier la présence de `'missing'` ; nouveau test des formes de `MediaNotDeletedPredicate`.
- [x] M1.9 Tests de `StoredPaths()`, `ForgetPaths()` et `OriginalPaths()` dans `internal/domain/` : avec et sans source, avec et sans HLS, dédoublonnage.

**Gate** :
```
cd apps/go-api
go test ./internal/domain/ ./internal/platform/duckdb/ ./internal/ops/ ./internal/service/ -run 'Media|StoredPaths|ForgetPaths|LoadKnownHashes' -count=1
go test ./internal/archlint/ -count=1
```

### M2. Passe de réconciliation avec le disque (cœur du plan, Sonnet)

Réécrire `internal/ops/media_reconcile.go`. `ReconcileOrphanedMediaFiles` et `findFilesByStem` disparaissent. Découper en sous-fonctions (interdit n°10). Le transfert de likes va dans `internal/platform/duckdb/media_likes_transfer.go`, car `ops` importe `platform/duckdb` et il sert aussi au dépôt (M5).

```go
type MissingPolicy int
const (
    MissingIgnore MissingPolicy = iota  // défaut : envoi, CLI ; ne marque jamais, ne déclenche pas le bandeau
    MissingMark                         // racine fiable : marque, sous garde-fou D10
    MissingRootUnavailable              // racine indisponible : ne marque pas, déclenche le bandeau
)

type ReconcileInput struct {
    PlayerSlug string
    Files      []string          // chemins absolus, déjà filtrés des autres titres
    Store      MediaPathStore
    Policy     MissingPolicy
    SkipFresh  bool              // ignorer les fichiers modifiés depuis moins de mediaStableAfter
    AllowBulk  bool              // l'utilisateur a confirmé une disparition massive (D10)
    Now        time.Time         // zéro : time.Now()
    Hash       func(string) (string, error)  // nil : HashFile ; injecté par les tests
}

type ReconcileResult struct {
    Hashes                                                             map[string]string // clé : pathKey(abs)
    Fresh                                                              map[string]bool   // clé : pathKey(abs)
    Relinked, Converted, Backfilled, MarkedMissing, Restored, Reverted, SourcesCleared int
    BulkPending                                                        int // candidates suspendues par le garde-fou ; 0 sinon
}

func ReconcileMediaWithDisk(ctx context.Context, db *sql.DB, in ReconcileInput) (ReconcileResult, error)
```

Algorithme, à suivre dans cet ordre :

1. **Inventaire disque.** `os.Stat` de chaque fichier (`Size`, `ModTime().UnixNano()`). Fichier qui échoue : `slog.WarnContext`, ignoré. Si `SkipFresh` et âge < `mediaStableAfter` (`const mediaStableAfter = time.Minute`) : `Fresh`, pas de hachage. `pathKey` = `filepath.Clean`, et `strings.ToLower` sous Windows (`runtime.GOOS`).
2. **Lignes.** `SELECT id, file_path, source_path, source_size, source_mtime_ns, file_name, file_hash, hls_path, status FROM media_files WHERE player_slug = ? AND <MediaNotDeletedPredicate>`.
3. **Empreintes.** Pour chaque fichier non frais : si une ligne a `source_path` = ce chemin, `source_size` et `source_mtime_ns` égaux, et `file_hash` non vide, reprendre son empreinte sans lire le fichier. Sinon appeler `Hash`. Échec : log, fichier ignoré.
4. **Présence.** Un chemin existe si sa clé est dans l'inventaire, sinon si `os.Stat(store.ToAbs(chemin))` réussit. Chemin de référence = `source_path` si non vide, sinon `file_path`. Ligne sans `hls_path` : présente si le chemin de référence existe. Ligne avec `hls_path` : présente si `file_path` (la version convertie) existe ou si `source_path` existe. **Original perdu** : `source_path` renseigné et inexistant. Tout fichier existant référencé par `source_path` ou `file_path` d'une ligne est « réclamé ».
5. **Lignes absentes et lignes à l'original perdu**, dans cet ordre :
   - a. Fichier non réclamé de même empreinte : relocaliser (`Relinked`), le fichier devient réclamé.
   - b. Sinon, changement de format : dans le **dossier** du chemin de référence, fichiers non réclamés dont le nom sans extension égale celui de `file_name`. Exactement un : relocaliser avec la nouvelle empreinte (`Converted`). Plusieurs : `slog.WarnContext`, rien.
   - c. Sinon, ligne présente à l'original perdu (sa version convertie existe) : `source_path`, `source_size`, `source_mtime_ns` à NULL (`SourcesCleared`), la ligne reste visible (D4).
   - d. Sinon : si `status` n'est pas déjà `missing`, candidate au marquage. Déjà `missing` : rien.
6. **Complément.** Pour chaque ligne présente sans `source_path` :
   - sans `hls_path` : renseigner `source_path = file_path`, plus taille et date (`Backfilled`) ;
   - avec `hls_path` : si un fichier non réclamé porte son empreinte, renseigner `source_path` et le réclamer. Aucun fichier : ne rien faire.
   Pour une ligne présente dont la taille ou la date a changé, mettre à jour `source_size`, `source_mtime_ns` et `file_hash`.
   Pour une ligne présente avec `hls_path`, `source_path` renseigné et master absent du disque : retour en lecture directe (D17, `Reverted`).
7. **Garde-fou.** `actives` = lignes dont `status` n'est pas `missing`. Si `Policy == MissingMark`, `!AllowBulk`, `actives >= 20` et `candidates*2 >= actives` : ne rien marquer, `BulkPending = candidates`, `slog.WarnContext`. Si `Policy` n'est pas `MissingMark` : ne rien marquer, `BulkPending = 0`.
8. **Écritures**, dans une seule transaction. Une écriture qui n'affecte aucune ligne ne déclenche ni transfert de likes ni compteur.
   - Relocalisation : `UPDATE media_files SET source_path=?, source_size=?, source_mtime_ns=?, file_name=?, file_ext=?, file_hash=?, status = CASE WHEN status = 'missing' THEN NULL ELSE status END WHERE id=? AND COALESCE(status,'') <> 'deleted'`. Si la ligne n'a pas de `hls_path`, ajouter `file_path=?` et transférer les likes.
   - Retour en lecture directe : `UPDATE media_files SET file_path=source_path, hls_path=NULL, transcode_status=NULL WHERE id=? AND COALESCE(status,'') <> 'deleted'`, puis transfert des likes de l'ancien `file_path` vers `source_path`.
   - Original perdu : `UPDATE media_files SET source_path=NULL, source_size=NULL, source_mtime_ns=NULL WHERE id=? AND COALESCE(status,'') <> 'deleted'`.
   - Marquage : `UPDATE media_files SET status='missing' WHERE id=? AND COALESCE(status,'') NOT IN ('deleted','missing')`.
   - Retour : ligne présente de statut `missing` : `UPDATE media_files SET status=NULL WHERE id=? AND status='missing'`, `Restored`.
   - **Ne jamais écrire `file_stem`.**

Transfert de likes (`duckdb.TransferMediaLikes(ctx, exec port.DBExecutor, oldPath, newPath string) error`), dans cet ordre :
```sql
INSERT INTO media_likes_history (media_path, liker_slug, liker_gamertag, is_liked, liked_at)
SELECT ?, liker_slug, liker_gamertag, TRUE, liked_at FROM media_likes_latest
 WHERE media_path = ? AND is_liked = TRUE;   -- (newPath, oldPath)
INSERT INTO media_likes_history (media_path, liker_slug, liker_gamertag, is_liked, liked_at)
SELECT ?, liker_slug, NULL, FALSE, NULL FROM media_likes_latest
 WHERE media_path = ? AND is_liked = TRUE;   -- (oldPath, oldPath)
```

Items :

- [x] M2.1 Écrire `ReconcileMediaWithDisk` et ses sous-fonctions selon l'algorithme ci-dessus. Fait : `media_reconcile.go` (types, inventaire, lignes, empreintes, présence), `media_reconcile_plan.go` (étapes 5 à 7), `media_reconcile_apply.go` (étape 8). `ReconcileOrphanedMediaFiles` et `findFilesByStem` supprimées. Seul appelant, `IndexMedia`, branché au minimum sur `MissingIgnore` (voir journal).
- [x] M2.2 Écrire `duckdb.TransferMediaLikes` (exporté, unique copie de ces deux requêtes). Fait : `platform/duckdb/media_likes_transfer.go`, avec la garde « chemins identiques : aucune écriture ».
- [x] M2.3 `media_repo_delete_guard_test.go` : mettre à jour la raison d'allowlist de `internal/ops/media_reconcile.go` (nouveau nom de fonction).
- [x] M2.4 Réécrire `media_reconcile_cgo_test.go` sur la nouvelle fonction. Les six tests existants migrent ; seul `TestReconcile_OrphanNoMatch` change de sens (attendu : `missing` avec `MissingMark`). Couvrir le tableau suivant, un test par ligne, `Hash` injecté avec compteur : fait, R1 à R25 (avec R12b, R12c, R15b, R15c), répartis dans `media_reconcile_cgo_test.go` (socle, six cas migrés, R1 à R10) et `media_reconcile_cases_cgo_test.go` (R11 à R25).

| Test | Situation | Attendu |
|---|---|---|
| R1 | Fichier inchangé, deux passes | 0 écriture à la seconde passe ; compteur de hachage inchangé à la seconde passe |
| R2 | Renommé, même dossier | `file_path`, `file_name`, `file_ext`, `source_path` à jour ; `file_stem` et `id` inchangés |
| R3 | Déplacé dans un sous-dossier | Idem R2 |
| R4 | Renommé et déplacé | Idem R2 |
| R5 | Extension changée, contenu différent, candidat unique | Relocalisé, nouvelle empreinte |
| R6 | Deux candidats de même nom | Pas de relocalisation, marqué `missing` |
| R7 | Supprimé | `status='missing'`, `file_path` inchangé |
| R8 | Revient au même endroit | `status` NULL, `Restored=1` |
| R9 | Revient ailleurs | Relocalisé, `status` NULL |
| R10 | Renommé avec deux likes actifs et un retiré | Deux likes actifs sur le nouveau chemin, zéro sur l'ancien (lecture de `media_likes_latest`) |
| R11 | Vidéo convertie, original renommé, copie présente | `source_path` à jour, `file_path` inchangé, aucun INSERT de like |
| R12 | Vidéo convertie, original supprimé, copie présente, `source_path` renseigné | Reste visible, `status` inchangé, `source_path` NULL, `SourcesCleared=1` (D4) |
| R12b | Vidéo convertie, original et copie absents | `missing` |
| R12c | R12, puis original remis ailleurs | `source_path` renseigné par empreinte, `file_path` inchangé |
| R13 | Vidéo convertie, `source_path` NULL, copie présente, aucun original | Rien (cas serveur) |
| R14 | Vidéo convertie, `source_path` NULL, original présent | `source_path` renseigné |
| R15 | 20 lignes, 10 absentes, `MissingMark` | Aucun marquage, `BulkPending=10` |
| R15b | 40 lignes, 14 absentes : deux passes | 14 marquées à la première ; à la seconde, `BulkPending=0` et aucune écriture |
| R15c | R15 avec `AllowBulk` | 10 marquées, `BulkPending=0` |
| R16 | 20 lignes, 9 absentes, `MissingMark` | 9 marquées |
| R17 | `MissingIgnore` et `MissingRootUnavailable` | Aucun marquage ; `BulkPending=0` |
| R18 | Même empreinte chez un autre joueur | Ligne de l'autre joueur intacte |
| R19 | Copie d'un fichier dont la ligne est présente | Ligne inchangée, copie non réclamée |
| R20 | `status='deleted'` | Jamais touché, jamais relocalisé |
| R21 | Fichier frais, `SkipFresh` | Pas de hachage, dans `Fresh` ; la ligne qui le référence reste présente |
| R22 | Fichier modifié au même chemin | `source_size`, `source_mtime_ns`, `file_hash` à jour |
| R23 | Relancer après R2 | 0 écriture (idempotence) |
| R24 | Vidéo convertie, original présent, master absent | Lecture directe : `file_path = source_path`, `hls_path` et `transcode_status` NULL, likes transférés |
| R25 | Ligne passée `deleted` entre la lecture et l'écriture (`UPDATE` préalable injecté via le faux `Hash`) | Reste `deleted`, aucun transfert de likes |

- [x] M2.5 Test unitaire de `pathKey` (`filepath.Clean` ; casse sous Windows seulement). Fait : `media_reconcile_pathkey_test.go`, par `pathKeyOn(goos, ...)` pour tester les deux systèmes depuis n'importe quel poste.

**Gate** :
```
cd apps/go-api
CGO_ENABLED=1 go test ./internal/ops/ -run 'Reconcile|PathKey' -count=1
go test ./internal/platform/duckdb/ -run 'MediaFilesRead|AllMediaFiles|AllowlistMediaFiles' -count=1
```

### M3. Branchement dans l'indexation (Sonnet)

- [x] M3.1 `MediaIndexOptions` : ajouter `MissingPolicy MissingPolicy` et `SkipFreshFiles bool`. Fait : `MissingPolicy` et `SkipFreshFiles` dans `MediaIndexOptions`.
- [x] M3.2 Dans `IndexMedia`, nouvel ordre : `ensureMediaTables` → `recalcBeforeIndex` → `walkMediaDir` → filtre `matchesForeignPrefix` → `ReconcileMediaWithDisk` → `loadKnownHashes` → boucle d'insertion. Une erreur de `ReconcileMediaWithDisk` est fatale pour cette indexation : `slog.ErrorContext` et retour d'erreur. Si `opts.TitleSlug`/`opts.Gamertag` sont connus, enregistrer l'état (M3.7). Fait : ordre du plan ; la passe et l'enregistrement de l'état vivent dans `media_index_reconcile.go` (`reconcileIndexedFiles`), qui écarte aussi les fichiers des autres titres. Erreur : `slog.ErrorContext` et retour d'erreur (test).
- [x] M3.3 Boucle d'insertion : l'empreinte vient de `rec.Hashes` (plus d'appel à `HashFile` ici). Fichier absent de `Hashes` (frais ou en échec) : ignoré. `result.Scanned` conserve sa valeur. Fait : l'empreinte vient de `rec.Hashes`, `HashFile` n'est plus appelé par `IndexMedia`. Une erreur d'empreinte n'alimente plus `result.Errors` (elle est journalisée par la passe).
- [x] M3.4 `insertMediaFile` **garde sa signature** (neuf appels dans les tests) et fait lui-même `os.Stat(path)` pour la taille et la date. INSERT et UPDATE écrivent `source_path`, `source_size`, `source_mtime_ns`. Fait : signature inchangée, `os.Stat` interne (`sourceStat`), INSERT, UPDATE de conversion et réactivation écrivent `source_path`, `source_size`, `source_mtime_ns`.
- [x] M3.5 Déduplication par chemin (`media.go`, requête `WHERE file_path = ?`) : ajouter `OR source_path = ?`. Fait : `file_path = ? OR source_path = ?`.
- [x] M3.6 Recherche du « même nom » : remplacer la requête `WHERE player_slug = ? AND file_stem = ?` par une sélection de toutes les lignes de ce `file_stem`, filtrées en Go. Une ligne n'est retenue que si le nom sans extension de `file_name` égale le nom cherché, **et** si le dossier du chemin de référence (`source_path`, sinon `file_path`) égale le dossier du nouveau fichier. Exception : une ligne HLS sans `source_path` reste retenue (comportement actuel). Le reste du bloc (SKIP si l'ancien existe, UPDATE sinon) ne change pas. La fonction va dans un nouveau fichier `media_insert_lookup.go`. Elle doit voir les lignes `deleted` : ajouter à `allowlistMediaFilesRawRead` l'entrée `internal/ops/media_insert_lookup.go` avec sa raison et la date du jour. `TestInsertMediaFile_StemDedup_*` doivent rester verts sans modification. Fait : `media_insert_lookup.go` (`findSameNameRow`), allowlist datée. **Écart** : une fiche qui désigne déjà le fichier (même `file_path` ou même `source_path`) est écartée de cette recherche et laissée à la déduplication par chemin ; sans cela, un média supprimé dont le fichier revient au même endroit n'est jamais réactivé (la recherche le trouve, son fichier existe, SKIP), ce qui contredit le critère d'ensemble de la suppression depuis l'app (test, mutation vérifiée).
- [x] M3.6b `HLSPathsFor` (`ops/media_hls.go`), signature inchangée : une source directement dans `capturesDir` garde `hls/{nom}` (arbres existants intacts) ; une source en sous-dossier sort dans `hls/{nom}_{8 premiers hex du SHA-256 du dossier relatif en minuscules, séparateur /}`. Mettre à jour le commentaire de limite. Test : deux sous-dossiers, même nom, deux dossiers distincts ; racine inchangée. Fait : `hlsFolderSuffix` ; commentaire de limite mis à jour ; tests `media_hls_paths_test.go`.
- [x] M3.7 Nouveau fichier `internal/ops/media_root_state.go`, mémoire protégée par `sync.Mutex`, clé (titre, joueur) : `type RootState struct { Unavailable bool; BulkPending int }`, `RecordRootState(title, gamertag string, s RootState)`, `GetRootState(title, gamertag string) RootState`, `ConfirmBulkMissing(title, gamertag string)`, `ConsumeBulkConfirmation(title, gamertag string) bool` (lecture puis remise à zéro). `IndexMedia` : avec `MissingMark`, passe `AllowBulk = ConsumeBulkConfirmation(...)` et enregistre `{false, rec.BulkPending}` ; avec `MissingRootUnavailable`, enregistre `{true, 0}` ; avec `MissingIgnore`, n'enregistre rien. Fait : `media_root_state.go`. En cas d'échec de la passe, une confirmation consommée est remise en attente.
- [x] M3.8 `internal/service/media_index_service.go` : remplacer `effectiveMediaBase` par `resolveMediaBase(ctx, pr, configured) (base string, policy ops.MissingPolicy)`. Dossier configuré présent : `MissingMark`. Repli sur la base interne : `MissingRootUnavailable`. Aucun dossier configuré (`configured == ""`) : `MissingMark`. Dossier configuré absent sans repli : `MissingRootUnavailable`. Adapter les trois appelants (`ResetAndReindex`, `ScanAllMedia`, `BuildMediaScanHook`), qui posent `MissingPolicy` et `SkipFreshFiles: true`. Fait : `resolveMediaBase` ; les trois appelants posent `MissingPolicy` et `SkipFreshFiles: true` ; `media_index_base_test.go` adapté au nouveau nom et à la politique.
- [x] M3.9 Si le dossier du joueur n'existe pas et que la politique est `MissingRootUnavailable` : appeler `ops.RecordRootState(titleSlug, gamertag, ops.RootState{Unavailable: true})` avant de sortir ou de passer au joueur suivant. Fait : helper `noteRootUnavailable` (appelé aux trois endroits) ; tests du hook et de `ScanAllMedia`.
- [x] M3.10 `media_service_upload.go` et `cmd/levelup/cmd_data.go` : aucun changement (`MissingIgnore`, fichiers frais acceptés). Vérifier et cocher. Vérifié : `media_service_upload.go` et `cmd/levelup/cmd_data.go` ne posent ni politique ni `SkipFreshFiles` (défauts `MissingIgnore`, fichiers frais acceptés), aucun changement.
- [x] M3.11 `RunHLSTranscode` (`ops/media_hls.go`) : juste avant `os.Remove(p.SourceAbs)`, mettre `source_path`, `source_size`, `source_mtime_ns` à NULL (`WHERE file_path = hlsRel`, via `withSharedSocialDB`). Échec : log et **pas de suppression** de l'original. Fait : `clearMediaSource` avant `os.Remove` ; échec : log et original conservé. La fixture `media_hls_integration_test.go` (table à la main) reçoit les trois colonnes d'original, sans quoi l'original est désormais conservé.
- [x] M3.12 Miniatures. Nouveau fichier `internal/ops/media_thumbnails_ensure.go`, fonction `EnsureVideoThumbnails(ctx, db, opts, store) (int, []string)`, qui **remplace** dans `IndexMedia` les appels à `GenerateThumbnails` et `BackfillThumbnailPaths` (ces deux fonctions restent pour `cmd/regen-thumbnails`). Pour chaque ligne vidéo visible du joueur, source = `source_path`, sinon `file_path` s'il n'y a pas de `hls_path` : Fait : `media_thumbnails_ensure.go`. Le lien d'une miniature du dossier accepte aussi le nom exact de l'original (sans suffixe d'empreinte), en plus du nom privé du suffixe. Les références des fiches supprimées ne comptent pas parmi les miniatures prises (voir section 8).
  - `thumbnail_path` renseigné et fichier présent : rien ;
  - `thumbnail_path` renseigné et fichier absent, source présente : régénérer au même chemin ;
  - `thumbnail_path` NULL : si un fichier de `thumbs/`, suffixe d'empreinte retiré (`thumbHashSuffixRe`), porte le nom de la source, est de format `.webp` ou `.gif` et n'est référencé par aucune autre ligne, le lier ; sinon générer `{nom}.webp` si la source est à la racine et que ce nom est libre, sinon `{nom}_{12 premiers car. de file_hash}.webp`.
  Chaque liaison se fait par `UPDATE media_files SET thumbnail_path=? WHERE id=?`. Génération par `generateAnimatedThumbnail`. Source absente : rien.
- [x] M3.13 Tests de `EnsureVideoThumbnails` : renommer `clip.mp4` en `clip_old.mp4` puis déposer un autre `clip.mp4` donne deux miniatures distinctes ; miniature supprimée à la main, régénérée ; miniature historique liée sans nouvelle génération. Fait : `media_thumbnails_ensure_cgo_test.go` (7 tests).
- [x] M3.14 Tests CGO, dans un nouveau `media_index_reconcile_cgo_test.go` : Fait : `ops/media_index_reconcile_cgo_test.go` (renommage, `ForceRescan`, deux sous-dossiers, conversion avec `DeleteSource`, garde-fou massif, racine indisponible, échec fatal) et `service/media_index_reconcile_cgo_test.go` (cycle `DeleteMedia` puis `IndexMedia` d'une vidéo convertie : pas de retour, puis retour avec ses likes et sa miniature ; placé dans `service` car `ops` ne peut pas importer `service`).
  - renommer entre deux `IndexMedia` : une seule ligne, même `id`, miniature conservée ;
  - `ForceRescan: true` après renommage : pas de doublon ;
  - deux fichiers de même nom dans deux sous-dossiers : deux lignes, deux miniatures distinctes (`generateAnimatedThumbnail` sauté si ffmpeg absent, via le mécanisme des tests existants) ;
  - suppression via `DeleteMedia` puis `IndexMedia` d'une vidéo convertie : pas de retour (original retiré, D9) ; puis original remis au même endroit et deux `IndexMedia` : la fiche revient, avec ses likes et une nouvelle miniature ;
  - relance après `RunHLSTranscode` avec `DeleteSource: true` : aucun `missing`.

**Gate** :
```
cd apps/go-api
CGO_ENABLED=1 go test ./internal/ops/ ./internal/service/ -count=1
go test ./internal/platform/duckdb/ -run 'AllMediaFiles|AllowlistMediaFiles' -count=1
go test ./internal/archlint/ -count=1
wc -l internal/ops/media.go   # doit rester <= 563
```
Comparer à `.ai/temoins_medias_avant.txt` : tout écart doit correspondre à un comportement voulu par ce plan.

### M4. Tâche de démarrage et règle du bureau (Sonnet)

- [x] M4.1 `internal/config/deployment.go` : ajouter à `DeploymentCapabilities` `KeepMediaSources` (valeur `desktop`) et `RevealFilesInExplorer` (valeur `desktop`), avec leur commentaire. Fait : `KeepMediaSources`, `RevealFilesInExplorer` (et `TrashMediaSources`, M4.9) dans `DeploymentCapabilities`, vraies sur desktop seulement.
- [x] M4.2 `internal/domain/bootstrap.go` et `service/bootstrap_service.go` : `can_keep_media_sources`, `can_reveal_files_in_explorer` (mêmes formes que `CanUseCompactChrome`). Fait : `can_keep_media_sources`, `can_reveal_files_in_explorer` (et `can_trash_media_sources`) au bootstrap.
- [x] M4.3 `internal/config/config.go` : `func (c *AppConfig) MediaDeleteSource(store *bool) bool` : faux si `c.Capabilities.KeepMediaSources`, sinon la logique actuelle. Renommer `ResolveMediaDeleteSource` en `resolveMediaDeleteSource` (le compilateur devient le garde-rail). Changement de test autorisé : le nom dans `config/pure_funcs_test.go`. Fait. **Écart** : `MediaDeleteSource` vit dans le nouveau `config_media.go`, pas dans `config.go` (619 lignes, dette gelée à ne pas accroître) ; `ResolveMediaDeleteSource` est renommée `resolveMediaDeleteSource` sur place, le compilateur garde le reste.
- [x] M4.4 Dans `internal/service/media_index_service.go`, ajouter `NewLiveMediaScanHook(cfg, store, gamertag)` : il lit les réglages à chaque appel (dossier, fuseau, `cfg.MediaDeleteSource`) et rend `BuildMediaScanHook(...)`. Remplacer les trois blocs recopiés dans `cmd/server/sync_v2_wiring.go`, `scheduler/auto_sync_engine.go`, `handlers/sync_handler.go`. Migrer `handlers/settings.go` (trois appels) vers `cfg.MediaDeleteSource`. Fait : `NewLiveMediaScanHook` dans le nouveau `service/media_scan_hook.go` ; les trois blocs recopiés (`sync_v2_wiring.go`, `auto_sync_engine.go`, `sync_handler.go`) et les trois appels de `settings.go` migrés. Le ratchet `RepoRoot` a été abaissé de 2 à 1 pour ces trois fichiers (lecteur migré), et le hook lit `cfg.Roots().Resources` pour ne pas créer de lecteur neuf.
- [x] M4.4b `handlers/media_upload.go` : remplacer le champ `isProduction` et le setter `WithProduction` de `MediaHandler` par `deleteSource func(*bool) bool` et `WithMediaDeleteSource`, câblé à `api/server_apiv1.go:968` avec `cfg.MediaDeleteSource`. Champ nul : `(&config.AppConfig{}).MediaDeleteSource(store)`, soit le comportement actuel sans `WithProduction`. Fait : champ `deleteSource` et `WithMediaDeleteSource` ; `resolveDeleteSource` replie sur `(&config.AppConfig{}).MediaDeleteSource` ; câblé à `server_apiv1.go` sur `cfg.MediaDeleteSource`.
- [x] M4.5 `cmd/server/background_tasks.go` : `taskMediaReconcile = "alignement des médias sur le disque"`, politique `exceptDemo` ; test associé (`background_tasks_test.go` exige les quatre colonnes). Fait : `taskMediaReconcile`, politique `exceptDemo`, `background_tasks_test.go` complété (liste des noms et doublures).
- [x] M4.6 Câblage `boot_crons.go`, sur le modèle de `citationsRelanceTask` : passage unique, pour chaque titre de `title.DefaultRegistry().All()` et chaque joueur de `cfg.LoadPlayers(slug)`, appeler `service.NewLiveMediaScanHook(...)` avec le titre posé sur le contexte (`ctxkeys.WithTitleSlug`). Respecter l'annulation du contexte entre deux joueurs. Fait : `mediaReconcileTask` (`cmd/server/background_media_reconcile.go`), lancée depuis `launchMonitoring` ; titres portant la capacité média, joueurs de `LoadPlayers` hors `AuthOnly`, titre posé sur le contexte, arrêt entre deux joueurs ; tests dédiés.
- [x] M4.7 Mettre à jour `config/deployment_test.go`, `service/bootstrap_capabilities_profile_test.go`. Fait.
- [x] M4.8 Test de `MediaDeleteSource` : capacité vraie + variable d'environnement vraie + réglage vrai donne faux ; capacité à zéro : comportement actuel. Fait : `config/config_media_test.go`.
- [x] M4.9 Capacité `TrashMediaSources` (valeur `desktop`) et `can_trash_media_sources` au bootstrap, comme en M4.1 et M4.2. Fait (avec M4.1 et M4.2).
- [x] M4.10 `port/repository_data.go` : `MediaFileTrasher` avec `TrashMediaFile(ctx, storedPath string) (bool, error)` : `true` = envoyé à la Corbeille ; `false, nil` = fichier absent ou support sans Corbeille ; erreur = échec. `domain.MediaDeleteResponse` : champ `Trashed bool` (`json:"trashed"`). Fait : `port.MediaFileTrasher`, `MediaDeleteResponse.Trashed`.
- [x] M4.11 `internal/ops/media_trash.go`, `media_trash_win.go` (`//go:build windows`), `media_trash_other.go` (`//go:build !windows`, rend `false, nil`). `OSMediaFileTrasher` résout le chemin comme `OSMediaFileRemover` (`capturesBase func() string`, chemin non résolvable : log et `false, nil`). Sous Windows, le type du lecteur vient de `GetDriveTypeW` sur `filepath.VolumeName(abs) + "\\"` : autre que `DRIVE_FIXED` (3) donne `false, nil`, car `FOF_ALLOWUNDO` y supprimerait sans Corbeille. Sinon `SHFileOperationW` par `windows.NewLazySystemDLL("shell32.dll")`, `wFunc = FO_DELETE` (3), `pFrom` terminé par deux zéros, `fFlags = FOF_ALLOWUNDO | FOF_NOCONFIRMATION | FOF_SILENT | FOF_NOERRORUI` (0x40, 0x10, 0x4, 0x400). Réussite = retour 0 et `fAnyOperationsAborted == 0`. Type du lecteur et appel système passent par deux champs de fonction injectables. Aucun test ne doit envoyer un fichier réel à la Corbeille. Fait : `media_trash.go` (logique portable, deux champs injectables), `media_trash_win.go` (`x/sys/windows` confiné à ce fichier), `media_trash_other.go`. Garde ajoutée : fichier toujours présent après une réussite annoncée = erreur ; dossier refusé.
- [x] M4.12 `service/media_service_delete.go` : option `WithMediaFileTrasher`, câblée dans `wire/registry_media.go` seulement si `cfg.Capabilities.TrashMediaSources`. Dans `DeleteMedia`, avec un trasher : pour chaque `OriginalPaths()`, `TrashMediaFile`. Erreur : log, retour d'erreur, **rien d'autre n'est retiré de cet original**. `false, nil` : le chemin rejoint la liste de suppression définitive. Ensuite `RemoveMediaFiles` sur cette liste plus `ForgetPaths()`. `Trashed` vrai si au moins un original est parti à la Corbeille. Sans trasher : `StoredPaths()`, comportement actuel. Fait : `WithMediaFileTrasher`, `removeMediaFiles` et `trashOriginals` ; câblé dans `wire/registry_media.go` sous `TrashMediaSources`. L'ordre reste base d'abord, disque ensuite (voir section 8).
- [x] M4.13 Tests : `DeleteMedia` avec faux ports (Corbeille réussie : l'original n'est pas dans les suppressions définitives, miniature et HLS y sont ; support sans Corbeille : suppression définitive, `Trashed` faux ; échec : erreur et original intact ; sans trasher : inchangé). `OSMediaFileTrasher` avec fonctions injectées (lecteur fixe, amovible, réseau, fichier absent, code d'erreur, opération annulée). Fait : `service/media_service_trash_test.go` (faux ports), `media_service_trash_cgo_test.go` (`DeleteMedia` de bout en bout, Corbeille factice qui déplace dans un dossier de test), `ops/media_trash_test.go` (lecteurs fixe, amovible, réseau, autres ; fichier absent ; code d'erreur ; annulée ; appel en échec ; réussite mensongère ; dossier ; chemin non résolvable). Aucun fichier réel n'est envoyé à la Corbeille.

**Gate** :
```
cd apps/go-api
go build ./...
go test ./internal/config/ ./internal/service/ ./cmd/server/ ./internal/api/handlers/ ./internal/archlint/ -count=1
```

### M5. API des fichiers introuvables (Sonnet)

Tout est sous `/players/{player_slug}`. Nouveaux fichiers, car `handlers/media.go` est à 487 L : `handlers/media_missing.go`, `service/media_service_missing.go`, `platform/duckdb/media_repo_missing.go`, `port/media_missing.go`, `domain/media_missing.go`.

| Route | Rôle |
|---|---|
| `GET /media/missing` | Médias `status='missing'` du joueur de la route + `root_unavailable` + `bulk_missing_pending` |
| `GET /media/missing/candidates?file_path=` | Autres fiches présentes du joueur, de même nature, triées par proximité de date de capture, 50 au plus |
| `POST /media/missing/merge` `{file_path, target_file_path}` | Fusion (D16) |
| `POST /media/missing/forget` `{file_path}` | Oublie (D8) |
| `POST /media/missing/confirm-bulk` | Confirme une disparition massive (D10) |
| `POST /media/reveal` `{file_path}` | Ouvre l'explorateur ; 404 sans la capacité |

- [x] M5.1 `domain/media_missing.go` : `MediaMissingItem{FilePath, Basename, Kind, ThumbnailPath *string, CaptureEndUTC *time.Time, MatchID *string}`, `MediaMissingResponse{Items, RootUnavailable bool, BulkMissingPending int}`, `MediaMergeCandidate{FilePath, Basename, Kind, ThumbnailPath *string, CaptureEndUTC *time.Time, DeltaSeconds *int}`, `MediaMergeCandidatesResponse{FilePath, CaptureUTC, Candidates}`, requêtes avec `RequesterSlug`, `RequesterIsAdmin`, `AuthEnforced` en `json:"-"` (modèle : `MediaAssociateRequest`), réponses. Fait : types du plan ; `MediaRequester` (identité du demandeur, `CanModify`) embarqué dans les six requêtes plutôt que trois champs recopiés ; `MediaFileRef` (fiche chargée : propriétaire, nature, chemins, date de capture) ; réponses `MediaMergeResponse`, `MediaForgetResponse`, `MediaConfirmBulkResponse`, `MediaRevealResponse`.
- [x] M5.2 `port/media_missing.go` : `MediaMissingStore` (`ListMissingMedia`, `LoadMissingTarget`, `ListMergeCandidates`, `LoadMergeTarget`, `ApplyMediaMerge(ctx, exec, ...)`), `MediaFileRevealer` (`Reveal(ctx, absPath) error`). Fait : `MediaMissingStore`, `MediaFileRevealer`, et `MediaMissingService` (le handler la détecte par type-assertion sur le service, comme `MediaDeleter`, pour ne casser aucune doublure de `port.MediaService`).
- [x] M5.3 `media_repo_missing.go`, joueur = `pdb.Gamertag`. `ListMissingMedia` : `status = 'missing'`, tri par `capture_end_utc` décroissant. `ListMergeCandidates` : `MediaVisiblePredicate`, même `kind`, `id` différent, tri sur l'écart absolu de `COALESCE(capture_start_utc, capture_end_utc)` avec la fiche introuvable, 50 au plus. `ApplyMediaMerge`, dans la transaction reçue : `TransferMediaLikes(ancien file_path, file_path de la cible)` ; si la fiche introuvable a un rattachement manuel dans `media_match_associations_latest`, INSERT dans `media_match_associations_history` pour l'`id` de la cible avec `is_manual=TRUE` (mêmes colonnes que le repli de `SetMediaMatchAssociation`) ; `status='deleted'` sur la fiche introuvable (`AND status='missing'`, aucune ligne affectée = erreur et annulation). Fait : `LoadMissingTarget` / `LoadMergeTarget` partagent `loadMediaRef` ; l écart de date est calculé par la base (CTE sur l id de la fiche), aucun instant ne traverse la liaison de paramètres ; `ApplyMediaMerge` rend ErrNotFound si le masquage n affecte aucune ligne.
- [~] M5.4 `[~]` supprimé : la fusion ne reçoit que des fiches déjà en base, aucun chemin libre du client n'est résolu sur le disque. Statué par le plan : couvert par la fusion sur fiches en base (M5.3, M5.6), aucun inspecteur de disque.
- [x] M5.5 `ops/media_reveal.go`, `media_reveal_win.go` (`//go:build windows`), `media_reveal_other.go` (`//go:build !windows`) : `OSFileRevealer` via `childproc.Command(ctx, "explorer.exe")`. Sous Windows, poser `cmd.SysProcAttr.CmdLine = "explorer.exe /select,\"" + abs + "\""` après la création. Si le fichier n'existe plus, ouvrir le dossier existant le plus proche sans `/select`. Hors Windows : erreur. Ne pas nommer le fichier `*_windows.go` (`no_accidental_goos_suffix_test.go`). Ne pas tester le lancement réel. Fait : `OSFileRevealer` (logique portable, lanceur injectable), `media_reveal_win.go`, `media_reveal_other.go`. **Écart** : `HideWindow` est remis à faux après `childproc.Command` (le lanceur est fait pour des enfants console ; l explorateur est une fenêtre que l utilisateur doit voir), et le lanceur vit sous un contexte détaché de la requête, borné à 15 s.
- [x] M5.6 Service : `ListMissingMedia`, `ListMergeCandidates`, `MergeMissingMedia`, `ForgetMissingMedia`, `ConfirmBulkMissing`, `RevealMedia`. Autorisation par `domain.CanModifyMedia` sur `player_slug` réel, pour les deux fiches d'une fusion. Refus : 404 si le média n'est pas `missing` (merge, forget), 400 si cible identique, natures différentes ou propriétaire différent. Écriture sous `s.acquireWriter()` puis `w.CommitWithCheckpoint(ctx, tx)` (modèle `markMediaDeleted`). `ForgetMissingMedia` : `MarkMediaDeleted`, puis `RemoveMediaFiles(ForgetPaths())`. `MergeMissingMedia` : `ApplyMediaMerge`, puis `RemoveMediaFiles(ForgetPaths())` après avoir retiré tout chemin aussi utilisé par la cible. `RevealMedia` : ouvre `source_path` s'il est renseigné, sinon `file_path`, résolu depuis la **ligne en base** (jamais depuis un chemin du client). Fait dans `service/media_service_missing.go`. **Écarts** : (1) les deux routes GET sont aussi autorisées par `CanModifyMedia` sur le joueur de la route : le middleware d ownership laisse lire le profil à un co-membre de groupe, qui ne doit pas voir les fichiers introuvables d un ami ; (2) `root_unavailable` et `bulk_missing_pending` sont lus par le service (`ops.GetRootState`, titre du contexte, joueur du dépôt) et non par le handler ; (3) `ConfirmBulkMissing` n enregistre rien et rend `confirmed:false` quand aucune disparition massive n attend (sinon la confirmation serait consommée par une passe sans rapport) ; (4) `RevealMedia` charge la fiche par `LoadMediaForDeletion` (présente ou introuvable, jamais supprimée) et résout le chemin avec le dossier des captures relu à chaque appel (`WithMediaFileRevealer(revealer, capturesBase)`) ; (5) messages d erreur en anglais sans accent (ratchet des littéraux accentués).
- [x] M5.7 Handlers Huma (modèle `handlePostMediaAssociate`), `mediaServableURLToStoredPath` à l'entrée pour chaque chemin, `h.filePathToURL(slug, p, mediaCapturesBase(h.settingsStore))` à la sortie pour `FilePath` et `ThumbnailPath`, `BumpMediaFeedVersion()` après merge et forget. Identité : extraire `resolveMediaWriteRequester(ctx, opName) (slug string, isAdmin, enforced bool, err error)` dans `media_requester.go` à partir de `resolveAssociateRequester`, y raccorder `resolveAssociateRequester`, `resolveDeleteRequester` et les nouvelles routes (règle des ≤ 2 copies). Enregistrer les routes dans `MediaHandler.Mount`. Injecter les ports à côté de `WithMediaFileRemover` (`wire/registry_media.go`). `root_unavailable` et `bulk_missing_pending` : `ops.GetRootState(titre, joueur)`. `confirm-bulk` : `ops.ConfirmBulkMissing`, puis passe lancée en arrière-plan par un `WithMediaRescan(func(ctx context.Context, gamertag string))` câblé sur `service.NewLiveMediaScanHook`, sous `context.WithoutCancel` et titre posé, suivie de `BumpMediaFeedVersion()` à sa fin. Messages d'erreur : `mediaWriteError`, en anglais technique comme les messages actuels de ce service, sans texte affiché tel quel à l'écran. Fait dans `handlers/media_missing.go` (monté par `mountMissingRoutes`, une ligne dans `Mount`), `resolveMediaWriteRequester` extrait dans `media_requester.go` et raccordé à associer, supprimer et aux nouvelles routes ; `WithMediaRescan` câblé sur `service.NewLiveMediaScanHook` (`server_apiv1.go`), sans seconde passe tant qu une première tourne pour le même joueur ; revealer câblé sous `RevealFilesInExplorer` (`wire/registry_media.go`). **Écart** : le 401 de suppression sans session passe d un message français à l anglais commun (même code `delete_requires_session`).
- [x] M5.8 `make openapi-gen && make generate-types` (ne jamais éditer `openapi.yaml` ni `generated.ts` à la main). Si `lib/api/types.ts` réexporte les types à la main, y ajouter les nouveaux. Fait : `make openapi-gen` puis `make generate-types` ; les types de `lib/api/types.ts` sont réexportés. Les trois capacités de M4 (`can_keep_media_sources`, `can_reveal_files_in_explorer`, `can_trash_media_sources`) sont dans les fichiers générés ; elles ont été ajoutées à `DEPLOYMENT_CAPABILITY_DEFAULTS` (faux avant hydratation), seule table qui bloquait `tsc` : couvre M7.6 pour ce point.
- [x] M5.9 Tests : service (faux ports : autorisation sur chaque fiche, statut, cible identique, natures différentes, `ForgetPaths` sans original, fusion qui garde les fichiers de la cible, écriture via `CommitWithCheckpoint`, `RevealMedia` sur source puis `file_path`) ; repo (DuckDB mémoire : likes additionnés, rattachement manuel repris, aucune ligne supprimée) ; handlers `httptest` (404 de `/media/reveal` sans capacité, 403 pour un visiteur sur merge, forget, confirm-bulk et reveal). Fait : `service/media_service_missing_test.go` (faux ports : refus sans écriture ni retrait, statuts, cible identique, natures et propriétaires, état du dossier, confirmation, ouverture de la source puis de `file_path`), `media_service_missing_cgo_test.go` (base et fichiers réels : likes, WAL vide, original intact, fichiers de la cible conservés), `platform/duckdb/media_repo_missing_cgo_test.go`, `ops/media_reveal_test.go`, `handlers/media_missing_test.go`, `api/media_missing_visitors_test.go` (middleware d ownership réel : 403 du co-membre sur merge, forget, confirm-bulk et reveal avant le handler ; 403 du service sur les deux GET ; 404 de reveal sans capacité).

**Gate** :
```
cd apps/go-api
go test ./internal/service/ ./internal/api/... ./internal/platform/duckdb/ -run 'Media|OpenAPI' -count=1
go vet ./internal/ops/
go test ./internal/archlint/ -count=1
make check-types
```

### M6. Extraction de la coquille du picker, sans changement visuel (Sonnet)

- [x] M6.1 Fait (`MediaPickerDialog.tsx`, 179 L, quatre composants aux props du plan, classes recopiées à l'identique). Créer `apps/web/src/features/media/MediaPickerDialog.tsx`, avec exactement les classes actuelles de `MediaMatchPicker.tsx` :
  - `MediaPickerDialog` : props `title`, `subtitle?`, `closeLabel`, `toolbar?`, `onClose`, `footer?`, `children` (superposition, carte, en-tête, zone de défilement) ;
  - `MediaPickerListStatus` : props `isLoading`, `isError`, `isEmpty`, `loadingLabel`, `errorLabel`, `emptyLabel` ;
  - `MediaPickerRow` : props `selected`, `current`, `onSelect`, `leading`, `children` (le bouton de ligne avec ses trois états) ;
  - `MediaPickerConfirmBar` : props `title`, `detail`, `cancelLabel`, `confirmLabel`, `busyLabel`, `busy`, `onCancel`, `onConfirm`.
- [x] M6.2 Réécrire `MediaMatchPicker.tsx` sur ces composants, sans changer ses props ni sa logique. Fait : le retour du composant passe par les quatre briques ; états, gestionnaires, boucle des candidats, `CandidateHeading`, `LobbyTeams` et formats inchangés. Le snapshot de M0.3 passe sans régénération, `git diff --stat -- '*.snap'` vide. **Écart de détail** : `subtitle` et `detail` sont typés `ReactNode` (et non `string`), pour que les nœuds texte du DOM restent identiques (`{prefixe} {heure}`, `<CandidateHeading/> · heure`) ; la barre d'outils reçoit son contenu et la coquille porte les classes du conteneur ; `MediaPickerRow` rend le `<li>` (la `key` reste posée par l'appelant, le `<ul>` reste dans le picker).

**Gate** : `cd apps/web && npx vitest run src/features/media/ && npm run typecheck && npx eslint src/features/media/`. Le snapshot de M0.3 doit passer **sans être régénéré**. Vérifier avec `git diff --stat` qu'aucun fichier `.snap` n'a changé.

### M7. Écran « Fichiers introuvables » (Sonnet ; M7.1 et M7.6 `[Haiku]`)

Comportement : un bouton dans la barre d'outils ouvre la fenêtre. Vue liste (miniature, nom, date de capture) ; un clic sélectionne, avec en pied « Retrouver dans la galerie » et « Oublier ». « Retrouver » passe à la vue des fiches candidates dans la même fenêtre (bouton retour, miniature et date), sélection en deux temps avec `MediaPickerConfirmBar`, comme la réassociation. « Oublier » passe aussi par `MediaPickerConfirmBar`, avec un texte qui dit que le fichier d'origine n'est pas touché.

- [x] M7.1 `[Haiku]` `i18n-modals.ts` : `MissingMediaText` dans `MediaModalsText`, en `FR` et `EN`. Clés : `toolbarAriaLabel`, `title`, `loading`, `error`, `empty`, `capturePrefix`, `findMedia`, `forget`, `back`, `candidatesTitle`, `candidatesEmpty`, `confirmMergeTitle`, `confirmMergeDetail`, `confirmForgetTitle`, `confirmForgetDetail`, `cancel`, `confirm`, `applying`, `mergeSuccess`, `mergeError`, `forgetSuccess`, `forgetError`, `rootUnavailable`, `bulkPending` (fonction du nombre), `bulkConfirm`, `bulkConfirmError`, `revealButton`, `revealTitle`, `revealError`, `closeAriaLabel`, `deltaFormat`. Français sans anglicisme ; titres factuels, sans possessif. Fait : `MissingMediaText` (31 clés du plan) sous `missing` dans `MediaModalsText`, `FR` et `EN`. `deltaFormat` prend un nombre de minutes (0 = moins d'une minute), comme la réassociation.
- [x] M7.2 `lib/query/keys.ts` : `mediaMissing(playerSlug, titleSlug)` → `['media', playerSlug, titleSlug, 'missing']` ; `mediaMergeCandidates(playerSlug, titleSlug, filePath)` → `['media', playerSlug, titleSlug, 'merge-candidates', filePath]`. Fait dans `lib/query/keys.ts` ; `filePath` accepte `null` (requête désactivée tant qu'aucune fiche n'est choisie), comme `mediaMatchCandidates`. Les deux clés sont classées dans les garde-rails de clés (`keys.title-slug.guard.test.ts` : title-scopées ; `keys.locale.test.ts` : indépendantes de la langue, aucun libellé résolu).
- [x] M7.3 `queries.ts` : `useMissingMedia`, `useMergeCandidates`, `useMergeMissingMedia`, `useForgetMissingMedia`, `useConfirmBulkMissing`, `useRevealMedia`. Mutations : toast de succès et d'erreur, invalidation de `queryKeys.mediaBase(playerSlug)`. Après `useConfirmBulkMissing`, l'invalidation suit le prochain changement de `useFeedVersion`, qui rafraîchit déjà la page ; pour cela la passe lancée par `confirm-bulk` appelle `BumpMediaFeedVersion()` à sa fin (M5.7). Fait. **Écart** : les six hooks vivent dans `missingQueries.ts` et non dans `queries.ts` (424 L, les hooks l'auraient porté au-delà de 500 L). `useDeleteMedia` reste dans `queries.ts` (M7.9b). `useConfirmBulkMissing` : toast d'erreur seulement (le plan ne prévoit aucune clé de succès), invalidation immédiate seulement si `confirmed` est faux.
- [x] M7.4 `MediaMissingDialog.tsx` sur les composants de M6. Logique pure (formats, tri éventuel) dans `mediaMissing.logic.ts` avec tests. Fait : `MediaMissingDialog.tsx` (vue liste, vue des fiches candidates, pieds), `mediaMissing.logic.ts` (`formatCaptureTime`, `formatCaptureDelta`, `findMissingItem`, `folderNoticeKind`) et `mediaMissing.logic.test.ts`. Le pied de la vue liste (« Retrouver » et « Oublier ») n'est pas un pied de confirmation : il reprend les classes de `MediaPickerConfirmBar` dans un petit composant local. **Écart de détail** : `MediaPickerRow` reçoit une prop optionnelle `action`, rendue hors du bouton de ligne (un bouton ne peut pas en contenir un autre : bouton « Afficher dans l'explorateur » par ligne, M7.8) ; sans elle le rendu est identique (snapshot de M0.3 inchangé).
- [x] M7.5 `MediaToolbar.tsx` : props `missingCount`, `onOpenMissing`. Bouton au même format que le filtre « sans match », visible seulement si `missingCount > 0`, tokens `border-warning/60`, `bg-warning/10`, `text-warning`. Glyphe `!` + nombre. Fait dans `MediaToolbar.tsx` ; la langue est lue dans le store (libellé `toolbarAriaLabel`), les props restent celles du plan.
- [x] M7.6 `[Haiku]` `apps/web/src/lib/capabilities/deploymentCapabilities.ts` : `can_keep_media_sources: false`, `can_reveal_files_in_explorer: false`, `can_trash_media_sources: false` ; `src/test/capabilities.ts` à jour. Vérifié : les trois clés étaient déjà dans `DEPLOYMENT_CAPABILITY_DEFAULTS` (M5.8) et `testCapabilities` les reprend. Complété : `desktopCapabilities()` publie les trois à vrai ; test dédié dans `deploymentCapabilities.test.ts`.
- [x] M7.7 `MediaPage.tsx` : état d'ouverture, `useMissingMedia` si `canWrite`. Au-dessus de la grille, un seul bandeau (composant `Card` existant, nouveau composant `MediaFolderNotice.tsx` pour laisser `MediaPage.tsx` sous 500 L) : `rootUnavailable` si `root_unavailable` est vrai ; sinon, si `bulk_missing_pending > 0`, `bulkPending(n)` avec le bouton `bulkConfirm`. Si la fenêtre se vide après une fusion ou un oubli, elle reste ouverte sur l'état `empty`. Fait : `MediaFolderNotice.tsx` porte sa propre mutation (l'état du clic disparaît avec le bandeau) ; `MediaPage.tsx` 472 L. La liste n'est lue que si `canWrite`.
- [x] M7.8 Affichage dans l'explorateur : prop `onReveal?` dans `CoverFlowModal` à côté du bouton de réassociation, transmise par `MediaLightbox`, fournie par `MediaPage` si `canWrite` et `useDeploymentCapability('can_reveal_files_in_explorer')`. Bouton équivalent dans chaque ligne de `MediaMissingDialog`, sous la même condition. Fait : `onReveal` dans `CoverFlowModal` (même règle de visibilité que la réassociation : propriétaire ou admin), transmise par `MediaLightbox` (simple alias du composant), fournie par `MediaPage` ; bouton par ligne dans `MediaMissingDialog` sous `can_reveal_files_in_explorer`.
- [x] M7.9 `features/settings/_settingsCards.tsx` : masquer le réglage de suppression des originaux (lignes `ToggleRow` et texte d'aide) si `useDeploymentCapability('can_keep_media_sources')` est vrai. Fait dans `MediaCard`.
- [x] M7.9b Suppression sur le bureau. `i18n-modals.ts`, `coverFlow`, en `FR` et `EN` : `deleteConfirmBodyTrash` (« Le fichier d'origine sera envoyé à la Corbeille de Windows, d'où il pourra être restauré. Sa miniature et sa version convertie seront effacées. Sur un disque sans Corbeille (clé USB, disque réseau), le fichier sera supprimé définitivement. ») et `deleteSuccessTrash` (« Média envoyé à la Corbeille. »). `CoverFlowModal.tsx` : `deleteConfirmBodyTrash` si `useDeploymentCapability('can_trash_media_sources')`, sinon le texte actuel. `useDeleteMedia` : `deleteSuccessTrash` si `response.trashed`, sinon `deleteSuccess`. Test : les deux textes de confirmation et les deux toasts (`queries.delete.test.ts`, `CoverFlowModal.delete.test.tsx`). Fait. Les mocks de `queries.delete.test.ts` reçoivent `trashed: false` (champ désormais obligatoire du contrat).
- [x] M7.10 Tests : `MediaMissingDialog.test.tsx` (liste, vide, erreur, fusion en deux temps, oubli en deux temps, français et anglais) ; `MediaFolderNotice` (trois états, clic de confirmation) ; `MediaToolbar` (bouton masqué à 0) ; `CoverFlowModal` (bouton absent sans `onReveal`) ; réglage masqué. Fait : `MediaMissingDialog.test.tsx` (22 cas), `MediaFolderNotice.test.tsx`, `MediaToolbar.test.tsx`, ajouts à `CoverFlowModal.delete.test.tsx` (bouton absent sans `onReveal`, textes de confirmation) et `queries.delete.test.ts`, `settings/MediaCard.test.tsx`, `deploymentCapabilities.test.ts`.

**Gate** :
```
cd apps/web
npx vitest run src/features/media/ src/features/settings/ src/lib/capabilities/
npm run typecheck
npx eslint src/features/media/ src/features/settings/
```

- [!] M7.11 Gate visuel, **à l'utilisateur** (statut `[!]` valide jusqu'à sa réponse ; aucun navigateur ni explorateur ouvert par l'agent, interdit 13). Témoins à lui remettre : renommer un clip, le déplacer dans un sous-dossier, le supprimer, le remettre, ouvrir « Fichiers introuvables », retrouver un clip ré-encodé dans la galerie, oublier, « Afficher dans l'explorateur », dossier indisponible (bandeau), retrait de la moitié du dossier (bandeau de confirmation), suppression de l'original d'une vidéo convertie (elle reste lisible), suppression depuis l'app (message, fichier dans la Corbeille, restauration puis retour du média), suppression d'un fichier plus gros que la Corbeille (Windows demande confirmation ; refuser laisse le fichier en place).

### M8. Revue et documentation

- [x] M8.1 Revue `adversarial-review` du diff complet de la partie M, par un agent Sonnet, **une seule ronde**. Cibles à lui nommer : perte de likes, fausse disparition en masse, écriture de `file_stem`, confinement des chemins, suppression d'un original par « Oublier », suppression définitive déclenchée par un échec de la Corbeille. Traiter chaque constat ou le statuer. **Fait le 2026-11-08** : 2 relecteurs Sonnet, une ronde, 9 constats : 8 corrigés (C1 à C8 de la commande de correction), 1 consigné en section 8 (renommage avant toute passe suivi d'une suppression depuis l'app).
- [x] M8.2 `[Haiku]` ADR `docs/adr/0046-media-identity-and-disk-reconciliation.md` (anglais uniquement) : décisions D1 à D13, D16 et D17. Ligne `0046` dans la liste des ADR de `CLAUDE.md`. **Fait le 2026-11-08** (vérifié : fichier présent, ligne `0046` présente dans `CLAUDE.md`).
- [x] M8.3 `[Haiku]` `.claude/skills/db-schema/SKILL.md` : colonnes de M1.4, valeur `missing`, `file_stem` figé. **Fait le 2026-11-08** : section « media_files — identité et état (ADR 0046) » ajoutée sous la table de `stats.duckdb` ; colonnes vérifiées dans `ops/media_store.go`.

**Gate** : `make go-api-lint` et `make libs-test` ; `cd apps/go-api && go test ./... -count=1` et `go test -tags=integration ./internal/persist/... ./internal/sync/... -count=1`.

---

## 6. Partie K : révision des KPI

### K1. Révisions et garde-rail (Sonnet, rapide)

- [x] K1.1 Nouveau `internal/sync/skill/scoring_revision.go` (paquet `skill`, et non la racine de `sync/` : le ratchet `TestSyncRootPackageFrozen` interdit tout nouveau fichier racine, décision de l'utilisateur du 2026-11-08) :
```go
package skill

const (
    PerformanceScoreRev = "perf-2026-11-08"
    LUSRRev             = "lusr-2026-11-08"
    SyncMetaScoringRevision = "scoring_revision"
)
func ScoringRevision() string { return "perf=" + PerformanceScoreRev + ";lusr=" + LUSRRev }
```
  Commentaire : toute modification d'une formule ou d'un paramètre impose de changer la constante correspondante, ce qui relance le calcul complet au prochain démarrage. Fait ; la clé est exportée (`skill.SyncMetaScoringRevision`) pour K2, et les appelants écrivent `skill.ScoringRevision()`.
- [x] K1.2 `internal/sync/scoring_revision_test.go`, `TestScoringParametersMatchRevision` : construire une chaîne déterministe (clés triées) de **toutes** les constantes numériques et tables de poids de `sync/skill/skill_config.go` (dont `CompositeWeights`, `RelativeWeights`, `WeightsForChain` pour chaque chaîne connue), plus le `windowSize` de `batchComputePerformanceScores` (le remonter en constante de paquet `perfWindowSize`). Hacher en SHA-256 et comparer à une table `{révision → empreinte}`. En cas d'échec, le message dit : changer la révision concernée et ajouter la nouvelle empreinte. Pas de `filepath.Walk` dans ce test (`no_ad_hoc_source_fingerprint_test.go`). Fait : `internal/sync/scoring_revision_test.go` (reste dans `sync` pour lire `perfWindowSize`), deux empreintes, une par révision (`PerformanceScoreRev`, `LUSRRev`), chacune dans sa table `{révision -> empreinte}` ; une entrée par révision future s'ajoute, les anciennes restent. `perfWindowSize` et `perfFallbackDurationSeconds` remontés en constantes de paquet (`performance.go`, 498 L). Périmètre des paramètres : voir le journal.
- [x] K1.3 Vérifier que le garde-rail mord : modifier temporairement un poids, constater l'échec, annuler. Noter le résultat au journal. Fait : trois mutations temporaires, toutes détectées, toutes annulées (voir journal).

**Gate** : `cd apps/go-api && go test ./internal/sync/ -run 'ScoringParameters' -count=1 && go test ./internal/archlint/ -count=1`

### K2. Recalcul automatique au démarrage (Sonnet)

- [x] K2.0 `batchComputePerformanceScores` (`sync/performance.go`) : quand `BatchUpdateMulti` échoue, rendre une erreur après la passe de nettoyage et le résumé, au lieu de `updated, nil`. Vérifier que le post-sync, qui journalise cette erreur, reste non bloquant. Sans cela, un recalcul dont l'écriture a échoué serait noté comme réussi, et les anciennes notes resteraient. Fait : `flushErr` rendu en fin de fonction (`performance.go`, 499 L). Appelants vérifiés : post-sync (`engine_postsync_scoring.go`, warn + `trackFatalErr` qui ne retient que les bases invalidées), `RunBackfillPerf` (orchestrateur : warn au journal du job), `enrichment_backfill.go` (`track`, étape suivante), import OpenSpartan (erreur au rapport), outil `recompute_perfnote` (déjà fatal pour l'outil) : aucun ne devient bloquant, le post-sync reste non bloquant.
- [x] K2.1 `MatchRecomputer.ApplyScoringRevision(ctx, rev string) (bool, error)` écrit **dans le fichier existant** `internal/sync/match_recomputer.go`, sans nouveau fichier à la racine de `sync/` (ratchet `TestSyncRootPackageFrozen`, baseline 78 ; si le fichier dépasse 500 L, extraire vers un sous-paquet cohésif, jamais la racine) :
  1. lire `sync_meta[skill.SyncMetaScoringRevision]` (ouverture de lecture `OpenReadForQuery`) ; valeur égale à `rev` : rendre `false, nil` ;
  2. `exclusionRecomputes.Do(ctx, r.playerDBPath, func(ctx) error { return r.recompute(ctx, "scoring_revision") })`. Le contrat de `Do` garantit qu'un recalcul démarré après la demande s'est terminé, même s'il a été absorbé par un autre ;
  3. en cas de succès seulement : prendre l'écrivain (`dblease.AcquireWriterCtx`, `dblease.KindPlayer`), `OpenPlayerDB`, `SetSyncMeta`, relâcher ;
  4. `duckdb.InvalidatePlayerReadCaches(ctx, r.xuid, r.titleSlug)`.
  Fait dans `match_recomputer.go` (255 L) : `ApplyScoringRevision`, `storedScoringRevision`, `storeScoringRevision`.
- [x] K2.2 Nouveau paquet `internal/service/scoringrevision/service.go` (modèle `citationsrelance`) : port `port.ScoringRevisionApplier` avec `FollowedPlayers(titleSlug)` et `ApplyForPlayer(ctx, titleSlug, player) (bool, error)`. `Run(ctx)` : passage unique, joueur par joueur, pour chaque titre de `title.DefaultRegistry().All()` portant `title.CapLUSR`, en respectant `ctx`. Une erreur : log + compteur expvar, on passe au joueur suivant ; le prochain démarrage retentera. Compteurs `scoring_revision_appliquee_total`, `scoring_revision_a_jour_total`, `scoring_revision_erreurs_total`. Fait : port dans `internal/port/scoring_revision.go`, service dans `internal/service/scoringrevision/service.go` ; profils `AuthOnly` ignorés comme dans la tâche médias ; liste de joueurs illisible : titre sauté, compté en erreur.
- [x] K2.3 `internal/api/wire/registry_scoring_revision.go` (modèle `registry_citations_relance.go`) : construire `NewMatchRecomputer` avec `config.PlayerDBPath`, `sharedProviderForTitle`, `config.MetadataDBPath`, `.WithTitle(slug)`. Fait ; un profil sans fichier de base joueur est sauté avec un log Info, sans le créer (voir journal).
- [x] K2.4 `cmd/server/background_tasks.go` : `taskScoringRevision = "recalcul des KPI après changement de formule"`, politique `exceptDemo`. Lancement dans `boot_crons.go` ; test des colonnes à jour. Fait : `scoringRevisionTask` (suivie), lancée dans `launchMonitoring` après la tâche médias ; la tâche est ajoutée aux listes de `background_tasks_test.go` (doublures et `nomsDeclares`), qui couvrent les quatre colonnes.
- [x] K2.5 Tests : service (faux port : a jour, appliqué, erreur sans arrêt de la boucle, titre sans capacité sauté, annulation) ; intégration `ApplyScoringRevision` (base réelle de test, modèle `match_recomputer_lusr_v2_integration_test.go`) : premier appel recalcule et note, second rend `false` sans recalcul, recalcul en échec ne note rien, écriture des notes de performance en échec ne note rien, révision différente relance. Fait : `internal/service/scoringrevision/service_test.go` (6 tests) et `internal/sync/match_recomputer_scoring_revision_integration_test.go` (4 tests, dont un direct sur `batchComputePerformanceScores`).

**Gate** :
```
cd apps/go-api
go test ./internal/service/scoringrevision/ ./cmd/server/ ./internal/archlint/ -count=1
go test -tags=integration ./internal/sync/ -run 'ScoringRevision|MatchRecomputer' -count=1
```

### K3. Documentation et revue

- [x] K3.1 `[Haiku]` ADR `docs/adr/0047-scoring-revision-replay.md` (anglais) : D14, D15 ; le premier démarrage après livraison recalcule chaque joueur une fois. Ligne `0047` dans `CLAUDE.md`.
- [x] K3.2 `[Haiku]` `docs/RUNBOOK` de release concerné ou `.claude/skills/delivery-checklist/SKILL.md` : changer une formule implique de changer la révision et d'ajouter une ligne aux notes de version. Fait dans `delivery-checklist` (commit `c82e7ff6b3`).
- [x] K3.3 `cmd/recompute_perfnote/main.go` : conservé (outil de contrôle `report`, serveur arrêté). Vérifier et cocher, ne pas modifier. Aucun diff depuis le début de la partie K.
- [x] K3.4 Revue `adversarial-review` de la partie K par un agent Sonnet, une seule ronde. Cibles : révision notée sans recalcul réussi, course avec une exclusion en cours, deux écrivains. 2 constats recevables, corrigés au commit `dd0efaa5c5`.

**Gate** : `cd apps/go-api && go test ./... -count=1`

---

## 7. Clôture Z

- [x] Z1 Skill `delivery-checklist`. Parcourue : aucun TODO, `fmt.Println` ni couleur brute ajouté ; deux `t.Skip("ffmpeg absent")` justifiés ; `go vet` propre ; typecheck après purge du cache sans erreur ; eslint 0 erreur. État CI : à vérifier après le push (Z5).
- [x] Z2 `make go-api-lint`, `make libs-test`, `make check-types`, `make test-web`, `cd apps/go-api && go test ./... -count=1`, `go test -tags=integration ./...`. Tout vert : 0 issue, 202 paquets, 204 paquets en intégration (`-p 1`), 10 810 tests web.
- [!] Z3 `make gate-push` (environ 25 minutes). Rouge sur la seule étape `replay-equiv`, film `11de8353`, trois digests (`spawnPoints` attendu 0, obtenu 54 ; `spawnPointsState` ; `artifact`). **Sans rapport avec ce chantier** : aucun fichier de décodeur ou de rejeu n'est modifié, et le même écart à l'identique (mêmes empreintes `obtenu`) se reproduit sur le commit d'avant le chantier `9472512a11`, avec le harnais lancé pour ce seul film. Cause établie par la coordination de la release : la référence dépend d'un catalogue local non versionné de points d'apparition, qui porte bien 54 points pour cette carte. Non bloquant ; la session du rejeu re-figera la référence.
- [x] Z4 `.ai/thought_log.md` ; `.ai/project_map.md` si les chemins ont changé ; alimenter `.ai/REGISTRE_REPORTS.md` avec les découvertes non traitées. Aucun chemin de `project_map.md` n'a changé ; une ligne de renvoi vers la section 8 au registre.
- [ ] Z5 PR vers la branche de release retenue (section 4), après accord de l'utilisateur. Corps de PR : ce que l'utilisateur verra. En attente : accord de l'utilisateur et tour de fusion donné par la session de coordination de la release (ordre : ce chantier d'abord ; rebase sur `origin/release/v8.1.0`, CI du push, PR, squash ; entrées `## [8.1.0]` des deux CHANGELOG dans cette PR).
- [ ] Z6 Rappel à l'utilisateur : le premier démarrage après livraison hache tous les fichiers une fois et recalcule les KPI de chaque joueur une fois.
- [ ] Z7 `powershell -File scripts/disk-hygiene.ps1 -Lot medias -Apply`. À faire après la fusion, le cache servira au rebase.

---

## 8. Découvertes (consigner, ne pas traiter)

Connues à la rédaction :

- `loadKnownHashes` ne filtre pas par joueur : le même clip déposé par deux joueurs n'est indexé que pour le premier.
- `BackfillThumbnailPaths` met à jour par nom de fichier sans filtrer le joueur.
- Colonnes `media_files.mtime` (jamais écrite) et `file_stem` (conservée pour son index, plus lue que par la recherche de M3.6).
- `media_match_associations` (table d'origine vide) : vestige.
- La compaction nocturne n'existe que sur le serveur (`scripts/compact-passes-nuit.sh`) ; un recalcul de KPI ajoute une passe, négligeable sur le bureau.
- Copie d'un fichier déjà indexé : aucune ligne ne porte sa taille et sa date, elle est relue à chaque passe.
- Halo Infinite ne déclare aucun préfixe de nom de fichier : le titre Halo 5 indexe aussi ses clips quand les deux partagent un dossier. Un fichier supprimé apparaîtra alors introuvable sous les deux titres.
- Deux conteneurs jumeaux dans un même dossier (`clip.mkv` et `clip.mp4`) partagent encore leur dossier de conversion (limite déjà documentée dans `HLSPathsFor`).
- Les `UPDATE` de `BackfillThumbnailPaths` ne filtrent pas le joueur (outil `cmd/regen-thumbnails` seulement après ce plan).

Trouvées pendant l'exécution :

- M2 : un fichier référencé seulement par une ligne `deleted` n'est pas « réclamé » (la passe ne lit pas les lignes supprimées, étape 2) : il peut devenir la cible de relocalisation d'une autre ligne de même empreinte, et deux lignes partageraient alors un `file_path`. Cas rare (média supprimé puis fichier identique remis), non traité.
- M3 : la miniature d'une fiche supprimée reste inscrite dans sa colonne ; `EnsureVideoThumbnails` ne la compte pas comme prise (lecture sans les supprimées), donc un autre clip de même nom peut reprendre ce nom, et la fiche supprimée, si elle revient, pointerait vers la miniature de l'autre. Cas rare, non traité.
- M3 : `NewFiles` compte tout fichier présenté à l'insertion, y compris ceux qu'elle reconnaît et saute (`ForceRescan`). Comportement d'avant, non traité.
- M2 : `TransferMediaLikes` lit `media_likes_latest` : toute base sans les tables de likes (migrations non passées) fait échouer la relocalisation d'une ligne non convertie, donc la passe entière. Les tests de M3 qui renomment un média sur une base de test doivent créer ces tables.
- M4 : **risque de la Corbeille de Windows, non vérifiable sans envoyer un vrai fichier.** `SHFileOperationW` avec `FOF_ALLOWUNDO | FOF_NOCONFIRMATION` supprime définitivement, sans question, un fichier trop gros pour la Corbeille du lecteur (comportement de l'API, pas un échec : le retour est 0). Le contrat « un échec ne devient jamais une suppression définitive » ne couvre donc pas ce cas. Traité par le pilote : ajout de `FOF_WANTNUKEWARNING` (0x4000), Windows demande confirmation avant de détruire un fichier qu'il ne peut pas recycler, un refus remonte en erreur et l'original reste. Reste à valider à la main dans M7.11 sur un fichier plus gros que la capacité de la Corbeille.
- M4 : ordre de `DeleteMedia` inchangé, base d'abord puis disque. Si l'envoi à la Corbeille échoue, la fiche est déjà `deleted` et l'original reste en place : la requête rend une erreur, la fiche disparaît de la galerie, et l'original est réindexé (la fiche revient) au scan suivant. Rejouer la suppression donne 404 (une fiche supprimée n'est plus chargeable). **Remplacé par M8.1 (C4)** : avec une Corbeille, l'envoi des originaux précède le masquage ; un échec de la Corbeille ne laisse aucune écriture et rejouer fonctionne. Sans Corbeille, l'ordre ci-dessus est conservé.
- M4 : `SHFILEOPSTRUCTW` est déclarée avec l'alignement 64 bits ; une version 32 bits de Windows (structure compactée) n'est pas livrée mais ne serait pas gérée.
- M4 : `mediaCapturesBase` (`wire/registry_media.go`) avale l'erreur de lecture des réglages sans log (comportement d'avant, déplacé tel quel).
- M5 : `ForgetMissingMedia` masque par `MarkMediaDeleted(file_path)` : si deux fiches partageaient un même `file_path` (cas de la découverte M2), les deux seraient masquées. Cas rare, non traité.
- M5 : la miniature ou la version convertie d'une fiche oubliée ou fondue est retirée sans vérifier qu'aucune autre fiche ne la référence (seule la cible d'une fusion est protégée). Même cas rare que celui de la découverte M3, non traité.
- M5 : l'état du dossier (`root_unavailable`, `bulk_missing_pending`) vit en mémoire du process : après un redémarrage, le bandeau reste vide jusqu'à la passe de démarrage, qui le reconstitue. Voulu par M3.7, noté pour M7.
- M5 : la route de lecture d'une liste de fichiers introuvables est refusée au co-membre de groupe par le service, pas par le middleware (qui laisse lire le profil). Si une autre lecture réservée au propriétaire apparaît, la même décision serait à prendre : le test de famille ne couvre que les écritures.
- M5 : l'environnement de poste n'avait pas le paquet `opus-decoder` (déclaré dans `apps/web/package.json` et le verrou) : `tsc -b` échouait sur `src/features/match-replay/test/oggOpusFile.ts` avant toute modification. Installé sans toucher aux fichiers suivis (`npm install --no-save`), hors périmètre du code.
- M6 : `npx eslint src/features/media/` rend un avertissement (0 erreur) déjà présent : `react-hooks/set-state-in-effect` sur `MediaAudioConfigButton.tsx:75`. Hors périmètre, non traité.
- M7 : `useToggleMediaLike` parcourt toutes les requêtes du préfixe `['media', playerSlug]` ; les clés des fichiers introuvables et des fiches candidates en font partie. Sans effet (accès optionnels, garde de `updateMediaLikeInResponse`), mais tout nouveau parcours de ce préfixe doit tolérer des formes de réponse différentes.
- M7 : la barre d'outils de la galerie est une seule ligne sans retour à la ligne ; le bouton « Fichiers introuvables » s'ajoute aux contrôles existants (comme le filtre « sans match »), à vérifier à largeur étroite au gate visuel.
- M7 : le bouton « Afficher dans l'explorateur » de la galerie envoie l'URL servable de la fiche (le serveur résout le fichier depuis la ligne en base) ; sur une vidéo convertie dont l'original a disparu, l'explorateur s'ouvre sur le dossier existant le plus proche (M5.5), pas sur la copie convertie.
- M6 (relevé en M7, non traité) : `src/components/ui/detail-section.guard.test.ts` échoue depuis M6 (suite web complète : 2 échecs, aucun autre). Le gabarit de titre de section est passé de `MediaMatchPicker.tsx` à `MediaPickerDialog.tsx` : le garde-rail voit un nouveau fichier hors liste et une ligne périmée (`MediaMatchPicker.tsx` descend à 0). Correctif attendu : déplacer l'entrée de la liste d'exceptions de `MediaMatchPicker.tsx` vers `MediaPickerDialog.tsx`, ou migrer le titre vers `SectionTitle`. Le gate de M6 (`src/features/media/`) ne couvrait pas ce garde-rail.
- M5 : `data/film_identites/` (fichiers datés de 14:54) est apparu non suivi dans le dépôt pendant l étape ; origine non établie (aucune commande de cette étape ne l écrit), non ajouté au commit.
- M8 (revue) : suppression depuis l'app d'un fichier renommé avant toute passe : le chemin stocké est périmé, l'original n'est pas envoyé à la Corbeille et le fichier est réindexé à la passe suivante comme nouvelle fiche ; si un autre fichier a repris ce chemin, c'est lui qui part à la Corbeille (récupérable). `source_size` et `source_mtime_ns` ne sont pas comparés avant l'envoi.
- M8 (C1) : deux fichiers de même nom et de même dossier (`clip.jpg` et `clip.jpeg`) apparus après la disparition de l'original : la fiche introuvable n'est plus reprise, le premier fichier indexé devient une nouvelle fiche, mais le second reste écarté par la règle des fichiers de même nom qui coexistent (une fiche visible le désigne comme variante ; `TestInsertMediaFile_StemDedup_BothFilesPresent`). Comportement d'avant, non traité : le second fichier n'est indexé que si le premier disparaît.
- K1 : l'empreinte des paramètres (`TestScoringParametersMatchRevision`) ne couvre pas les littéraux en ligne illisibles hors du paquet `skill` : la table des scores de victoire (1 / 0,5 / 0 / 0,15) de `skill_rating_composite.go`, `teamWeightFloor` du solveur `ep`, les constantes non exportées de `skill_v2_shadow.go` et de `skill_v2_shared_access.go`. **Rappel : un changement de logique sans changement de nombre n'est pas détecté ; il impose de changer la révision (`PerformanceScoreRev` ou `LUSRRev`) à la main.** Non traité.
- K2 : le rejeu de démarrage ne vérifie pas que le profil porte un xuid : un profil avec base joueur mais sans xuid recalcule à vide (aucune ligne de note) puis range la révision. Cas non observé, non traité.
- K2 (revue) : un match dont le rejeu LUSR échoue est maintenant compté (`WithMatchFailureCounter`) ; `skill_v2_shadow.go` passe de 756 à 763 lignes (dette existante, +7 strictement nécessaires). `processOneShadowMatch` journalise toujours sans arrêter la boucle : seul l'appelant qui passe le compteur voit l'échec.

---

## 9. Journal d'exécution

| Date | Étape | Fait | Notes |
|---|---|---|---|
| 2026-11-08 | Avant exécution | Relecture indépendante du plan par deux relecteurs Sonnet, une ronde | 19 constats et 3 précisions vérifiés sur le code. Plan corrigé pour tous, sauf le partage de clips entre titres, consigné en section 8. Aucune nouvelle ronde prévue. |
| 2026-11-08 | Avant exécution | Décisions de l'utilisateur intégrées | D4 : une vidéo convertie dont l'original manque reste visible. D9 : suppression depuis l'app avec message explicite et Corbeille de Windows si possible. |
| 2026-11-08 | M0 | Témoins avant modification (M0.1 à M0.3) | Témoin Go : `.ai/temoins_medias_avant.txt`, 4 paquets `ok`, `EXIT=0`, aucun test en échec. Témoin web : `MediaMatchPicker.test.tsx` vert (6 tests), snapshot du picker avec candidat sélectionné écrit et stable sur deux exécutions. Gate M0 : vitest vert. |
| 2026-11-08 | M1 | Colonnes, état « introuvable », prédicats (M1.1 à M1.9) | `MediaStatusMissing`, `SourcePath` et `ForgetPaths()` / `OriginalPaths()` au domaine ; colonnes `source_path`, `source_size`, `source_mtime_ns` ; `MediaVisiblePredicate` exclut `deleted` et `missing`, `MediaNotDeletedPredicate` ajouté et branché sur `loadKnownHashes`, `LoadMediaForDeletion`, `MarkMediaDeleted` seulement ; `source_path` chargé à part (seul « Binder Error » toléré, test des deux cas). Garde-rail mis à jour. Gate passé (voir thought_log). Aucun écart au plan. |
| 2026-11-08 | M2 | Passe de réconciliation avec le disque (M2.1 à M2.5) | `ReconcileMediaWithDisk` en trois fichiers (lecture et empreintes, planification des étapes 5 à 7, écriture en une transaction) ; `duckdb.TransferMediaLikes` ; R1 à R25 (avec R12b, R12c, R15b, R15c), six cas migrés, `pathKey`. Mutation vérifiée : retirer le garde « pas supprimé » de la relocalisation fait échouer R25. **Branchement minimal dans `IndexMedia`** : l'appel à `ReconcileOrphanedMediaFiles` est remplacé par `ReconcileMediaWithDisk` (`MissingIgnore`, tous les fichiers du walk, erreur seulement journalisée), rien d'autre ; M3 fait le reste (filtre des préfixes étrangers avant la passe, erreur fatale, empreintes réutilisées, `SkipFresh`). En attendant, les fichiers sont hachés deux fois par `IndexMedia`. **Écarts de détail** : champ non exporté `writes` ajouté à `ReconcileResult` pour prouver « 0 écriture » ; un fichier dont l'empreinte échoue reste compté présent (pas marqué introuvable) mais ne peut cibler aucune relocalisation ; un fichier frais n'est pas relevé pour `source_*` (attend la passe suivante). |
| 2026-11-08 | M3 | Branchement dans l'indexation (M3.1 à M3.14, dont M3.6b) | `IndexMedia` suit l'ordre du plan : schéma, recalcul, parcours, filtre des autres titres, passe de réconciliation (erreur fatale), empreintes connues, insertions avec les empreintes déjà lues (plus de double hachage), association, miniatures par `EnsureVideoThumbnails`. Politique de marquage et fraîcheur posées par les trois appelants du service (`resolveMediaBase`), état de la racine en mémoire (`media_root_state.go`), original détaché de la fiche avant sa suppression par la conversion, dossier HLS distinct par sous-dossier. **Écarts** : (1) la recherche du « même nom » écarte une fiche qui désigne déjà le fichier (même `file_path` ou `source_path`) : sans cela un média supprimé dont le fichier revient au même endroit n'est jamais réactivé (mutation vérifiée) ; (2) la fixture à table écrite à la main de `media_hls_integration_test.go` reçoit les trois colonnes d'original, car l'original n'est désormais supprimé qu'après son détachement de la fiche ; (3) `media_index_base_test.go` suit le renommage `resolveMediaBase` ; (4) le test du cycle suppression puis indexation est dans `internal/service` (`ops` ne peut pas importer `service`) ; (5) le lien d'une miniature du dossier accepte aussi le nom exact de l'original. `media.go` : 547 lignes (563 au plus). |
| 2026-11-08 | M4 | Tâche de démarrage, règles du bureau, Corbeille (M4.1 à M4.13, dont M4.4b) | Capacités `KeepMediaSources`, `RevealFilesInExplorer`, `TrashMediaSources` (desktop) et leurs trois champs de `/bootstrap` ; `AppConfig.MediaDeleteSource` (faux sous `KeepMediaSources`), `NewLiveMediaScanHook` qui remplace les trois blocs recopiés ; `WithMediaDeleteSource` côté handler ; tâche de démarrage `alignement des médias sur le disque` (une passe par joueur de chaque titre média, hors démo) ; `port.MediaFileTrasher`, `OSMediaFileTrasher` (Windows par `SHFileOperationW`, lecteur fixe seulement, échec jamais converti en suppression), `DeleteMedia` avec `Trashed`. **Écarts** : (1) `MediaDeleteSource` dans `config_media.go` (config.go dépasse 500 lignes) ; (2) ratchet `RepoRoot` abaissé de 2 à 1 pour trois fichiers migrés, et le hook lit `cfg.Roots().Resources` ; (3) `OSMediaFileTrasher` refuse un dossier et traite une réussite annoncée avec fichier toujours présent comme une erreur. Risque Corbeille et ordre de suppression en section 8. `openapi.yaml` non régénéré (M5.8). |
| 2026-11-08 | M5 | API des fichiers introuvables (M5.1 à M5.9, M5.4 déjà `[~]`) | Six routes (liste, candidats, fusion, oubli, confirmation d'une disparition massive, ouverture de l'explorateur), service, dépôt (`media_repo_missing.go`), `OSFileRevealer`, `resolveMediaWriteRequester` unique, contrat régénéré et types web réexportés. Visiteurs et co-membres : le middleware refuse déjà le co-membre sur toute écriture, prouvé sur les quatre POST avec le service réel ; il le laisse lire, donc les deux GET sont refusés par le service. **Écarts** : voir les notes de M5.5 à M5.8 (`HideWindow` de l'explorateur, autorisation des GET, état du dossier lu par le service, confirmation sans objet ignorée, 401 de suppression en anglais) ; trois clés de capacité ajoutées à `DEPLOYMENT_CAPABILITY_DEFAULTS` ; plafond du ratchet des littéraux accentués abaissé pour `media_delete.go` ; `opus-decoder` installé localement hors suivi. |
| 2026-11-08 | M6 | Extraction de la coquille du picker (M6.1, M6.2) | `MediaPickerDialog.tsx` (`MediaPickerDialog`, `MediaPickerListStatus`, `MediaPickerRow`, `MediaPickerConfirmBar`), `MediaMatchPicker.tsx` réécrit dessus (506 L au total pour les deux fichiers, picker 327 L). Gate : vitest `src/features/media/` 13 fichiers et 124 tests verts, `npm run typecheck` sortie 0, eslint 0 erreur (un avertissement antérieur, section 8). Snapshot de M0.3 passé sans régénération (aucun `-u`), aucun `.snap` modifié. **Écart** : `subtitle` et `detail` en `ReactNode` pour garder les nœuds texte du DOM. |
| 2026-11-08 | M7 | Écran « Fichiers introuvables », Corbeille et explorateur (M7.1 à M7.10, dont M7.9b) | `MediaMissingDialog` (liste, fiches candidates, fusion et oubli en deux temps) sur les briques de `MediaPickerDialog`, `MediaFolderNotice` (dossier inaccessible, retrait massif à confirmer), bouton à `!` dans la barre d'outils, hooks et clés de requête, textes `FR` et `EN`, bouton « Afficher dans l'explorateur » (galerie et lignes de la fenêtre), réglage des originaux masqué sur le bureau, texte de confirmation et toast de suppression selon la Corbeille. Gate : vitest `src/features/media/ src/features/settings/ src/lib/capabilities/` 32 fichiers et 314 tests verts, `npm run typecheck` sans erreur, eslint 0 erreur (un avertissement antérieur) ; snapshot de M0.3 non modifié. Suite web complète : seul échec restant, antérieur à M7, en section 8 (`detail-section.guard.test.ts`). **Écarts** : (1) hooks dans `missingQueries.ts` (`queries.ts` aurait dépassé 500 L) ; (2) prop optionnelle `action` sur `MediaPickerRow` pour le bouton par ligne (un bouton ne peut pas en contenir un autre), rendu inchangé sans elle ; (3) `useConfirmBulkMissing` sans toast de succès (aucune clé prévue) ; (4) mocks de `queries.delete.test.ts` complétés par `trashed: false`. **M7.11 en `[!]`** : gate visuel de l'utilisateur, témoins listés à l'item. |
| 2026-11-08 | M8.1 | Revue adversariale du diff de la partie M, corrections C1 à C8 | 2 relecteurs Sonnet, une ronde, 9 constats : 8 corrigés, 1 consigné (section 8). C1 : une fiche `missing` n'est plus retenue par la recherche du même nom, la reprise d'une fiche par un autre format transfère les likes (une transaction). C2 : la résurrection d'une fiche supprimée dont la version convertie manque la remet en lecture directe (D17) et transfère les likes. C3 : test table des sept écritures de la passe sur une fiche supprimée, mutation vérifiée sur chaque garde. C4 : avec une Corbeille, envoi des originaux, puis masquage, puis retrait du reste ; un échec de la Corbeille n'écrit rien. C5 et C6 : libellés et toast du bureau (« Envoyer à la Corbeille », « Média supprimé définitivement »). C7 : la fusion reprend le rattachement courant, manuel ou automatique (D16 mis à jour). C8 : choix du trasher et du revealer en fonctions pures testées. **Écarts** : le second candidat de C1 reste écarté par la règle de coexistence (section 8) ; `media_backup_cgo_test.go` et `newEnsureSetup` reçoivent les tables de likes (trois tests de conversion) ; le test « rattachement automatique ne se reprend pas » est remplacé, son comportement changeant avec C7 ; les tests de `removeMediaFiles` passent par un helper des deux temps ; `media_insert_convert.go` entre dans l'allowlist de `TestAllMediaFilesReadsFilterDeleted` (lecture par id de la fiche supprimée qu'on ressuscite), justification datée. `media.go` : 533 lignes. |
| 2026-11-08 | M8.2, M8.3 | ADR 0046 (anglais) et section `media_files` du skill `db-schema` | ADR : D1 à D13, D16, D17 tels qu'écrits au plan ; ligne `0046` dans `CLAUDE.md`. Schéma : `source_path`, `source_size`, `source_mtime_ns`, `status='missing'`, `file_stem` figé ; colonnes vérifiées dans `ops/media_store.go`. Gate : `make go-api-lint` sortie 0 (0 issues, tous modules), `make libs-test` sortie 0, `go test ./... -count=1` sortie 0, `go test -tags=integration ./internal/persist/... ./internal/sync/... -count=1` sortie 0. Commit fait ; étape close ; point d'étape à l'utilisateur. |
| 2026-11-08 | K1 | Révisions et garde-rail (K1.1 à K1.3) | `PerformanceScoreRev`, `LUSRRev`, `ScoringRevision()` et `SyncMetaScoringRevision` dans `sync/skill/scoring_revision.go` ; `TestScoringParametersMatchRevision` (sous-tests `performance` et `lusr`) dans `sync/scoring_revision_test.go`, une empreinte SHA-256 par révision comparée à sa table `{révision -> empreinte}`, message d'échec : quelle constante changer, quelle entrée ajouter. **Paramètres couverts, note de performance** : `perfWindowSize` (50), `perfFallbackDurationSeconds` (600, littéral promu en constante), `MinMatchesForRelative`, `MinMatchesPerChainForRelative`, `RelativeWeights`, `WeightsForChain` pour les sept chaînes (profil objectif compris), `games.DefaultEffectiveHpToKill`. **LUSR, `skill_config.go`** : toutes les constantes numériques, `CompositeWeights`, `SkillTiers` (bornes et sous-paliers). **LUSR, hors `skill_config.go`** : `skill.QuitSecondaryFactor`, `games.DefaultEffectiveHpToKill`, et les valeurs par défaut exportées de `internal/analysis/skill_v2` : `DefaultPriors`, `DefaultCountHyperparamsMap`, `DefaultTierBoundaries`, `DefaultQuitDeltaRelated`, `DefaultQuitDeltaUnrelated`, `QuitPenaltyMuCap`, `SquadOffsetCap`, `DefaultModeCouplingWeight`, `Phase4ModeCouplingMaxWeight`, `PlacementMatches`, et `ep.DefaultMatch2TeamConfig`. **K1.3, mutations temporaires** : `MetricKeyKPM` 0.14 -> 0.15 : `performance` échoue seul ; `MetricKeyKillsVsExpected` (composite) 0.27 -> 0.28 : `lusr` échoue seul ; `perfWindowSize` 50 -> 51 : `performance` échoue ; les trois annulées (`git diff` de `skill_config.go` vide), test revenu vert. **Écart** : le gate a d'abord échoué sur `TestSyncRootPackageFrozen` (79 fichiers racine, baseline 78) parce que K1.1 prescrivait `sync/scoring_revision.go` ; décision de l'utilisateur : révisions dans `sync/skill/`, ratchet intact, K2.1 écrit dans `match_recomputer.go`. Littéraux non couverts : section 8. |
| 2026-11-08 | K2 | Recalcul automatique au démarrage (K2.0 à K2.5) | `batchComputePerformanceScores` rend l'erreur d'écriture des notes après nettoyage et résumé (aucun appelant ne devient bloquant) ; `MatchRecomputer.ApplyScoringRevision` (lecture de `sync_meta` par `OpenReadForQuery`, recalcul par `exclusionRecomputes.Do`, révision rangée sous lease d'écriture par `SetSyncMeta` seulement après succès, caches de lecture invalidés) ; service `scoringrevision` et son port, câblage `NewScoringRevision`, tâche de démarrage `recalcul des KPI après changement de formule` (`exceptDemo`, suivie). Titres filtrés par `HasCapability(title.CapLUSR)`. **Mutation vérifiée** : remplacer `return updated, flushErr` par `nil` fait échouer les deux tests d'écriture des notes (et eux seuls). **Écarts de détail** : (1) un profil sans fichier de base joueur est sauté (log Info, retour `false, nil`) dans le câblage : sans cela `OpenReadForQuery` échouerait à chaque démarrage et un profil jamais synchronisé compterait en erreur ; la base n'est jamais créée ; (2) l'échec d'écriture des notes est simulé en changeant le type de `performance_chain` en entier (une colonne renommée ou retirée est recréée par l'ouverture de la base joueur) ; (3) `performance.go` reste à 499 lignes (déclaration de `flushErr` dans le bloc de variables, message de log sur une ligne). |
| 2026-11-08 | K2 (revue) | Revue de la partie K, 2 constats recevables, tous corrigés | **P1** : un match en échec (`LoadState`, calcul, écriture de la note, persist de l'état) laissait `recompute` réussir et la révision être notée. Option `skill.WithMatchFailureCounter`, comptée aux quatre échecs ; `recompute` rend `ErrPartialRecompute` (nombre de matchs dans le message) si le compteur est non nul, l'erreur passe par `exclusionRecomputes.Do` à tous les demandeurs. `ApplyScoringRevision` ne note sur aucune erreur, partielle comprise. **Arbitrage de l'utilisateur** : l'exclusion d'un match ne doit pas échouer pour l'utilisateur (`SetExclusion` propage toute erreur du recalculateur) : `RecomputeAfterExclusion` journalise `ErrPartialRecompute` en WARN et rend `nil`, comme avant ; `RecomputePending` reste vrai (`FailedLast`), une demande répétée recalcule. **P2** : test du saut d'un profil sans base joueur (`registry_scoring_revision_test.go`) ; test de l'invalidation des caches de lecture (champ `invalidateReadCaches` du recalculateur, témoin dans les tests). **Mutations vérifiées** : retirer l'erreur sur compteur non nul fait échouer le test du match en échec ; retirer l'appel d'invalidation fait échouer le test des caches ; retirer le filtrage de `ErrPartialRecompute` fait échouer le test d'exclusion. Échec de note simulé par un changement de type de `match_skill_rank.rating_type` (vues retirées). |
