// boot_migrations.go — migrations du démarrage : chemins des bases, dossier warehouse,
// migrations des bases partagées (porte de version et copie de sécurité comprises), titres
// additionnels, bases joueur, puis migrations de données au boot (amis, groupe par défaut).
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/games/weapons"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/friendstore"
	"levelup/go-api/internal/platform/groupstore"
)

// bootMigrate enchaîne les migrations de démarrage ; l'échec des bases partagées arrête le
// processus (migrateSharedOrRefuse).
func bootMigrate(st *bootState) {
	resolveBootPaths(st)
	prepareWarehouse(st)
	migrateSharedOrRefuse(st)
	migrateAdditionalAndPlayerDBs(st)
	// Les bases sont migrées et pas encore ouvertes par les fournisseurs : ce processus en est l'écrivain unique.
	compactPassesAtBootAllTitles(st.ctx, st.cfg, st.pr, title.DefaultRegistry())
}

// resolveBootPaths résout le PathResolver, le slug par défaut et les chemins des bases ; en
// démo, vérifie la fixture joueur.
func resolveBootPaths(st *bootState) {
	cfg := st.cfg
	// --- 3. Connexions DuckDB ---
	pr := cfg.PathResolver()
	titleSlug := title.DefaultSlug
	dbPaths := resolveBootDBPaths(cfg, pr)
	sharedPath := dbPaths.shared
	metaPath := dbPaths.metadata
	sharedSocialPath := dbPaths.sharedSocial
	pvePath := dbPaths.pve

	// En DEMO_MODE, les DB de la FIXTURE font autorité — INCONDITIONNELLEMENT.
	//
	// Cette bascule était auparavant conditionnée à l'ABSENCE de la DB de prod
	// (« utiliser la fixture si data/titles/… n'existe pas »). Sur un poste de
	// développement — qui a évidemment ses données réelles — la condition ne se
	// vérifiait jamais : le serveur « démo » ouvrait le shared_matches_v2 RÉEL,
	// où le joueur `demo-player` (xuid 0000000000000000) n'a aucun match. D'où
	// des pages vides (`teammates … top_rows_count=0`, healthz/home 503) et un
	// harnais visuel qui skippait 6 pages sur 7 en annonçant « données absentes ».
	// Pire : les migrations de boot s'appliquaient alors aux DB de PRODUCTION,
	// qu'un simple lancement de démo modifiait.
	//
	// Le mode démo doit être hermétique aux données comme il l'est au réseau
	// (cf. internal/platform/netguard) : la présence ou non de données réelles
	// sur la machine ne doit RIEN changer à ce que sert la démo.
	if cfg.DemoMode {
		slog.Info("demo_mode: utilisation fixture", "shared", sharedPath, "metadata", metaPath,
			"shared_social", sharedSocialPath, "shared_pve", pvePath)

		// Validation au boot — la fixture joueur doit exister, sinon les requêtes
		// /pages/* échouent avec un IO Error opaque et le frontend affiche
		// « DemoPlayer » en fantôme (incident 2026-04-29).
		//
		// On vérifie le chemin IMBRIQUÉ réellement produit par `seed-demo`
		// ({dir}/players/DEMO/stats.duckdb). L'ancienne vérification portait sur
		// le chemin PLAT legacy ({dir}/stats.duckdb), que le générateur n'écrit
		// jamais : elle criait « fixture absente » sur une fixture parfaitement
		// installée, à chaque démarrage — un WARN qui a masqué le vrai défaut
		// ci-dessus pendant tout le chantier.
		demoStats := demoFixturePlayerDBPath(cfg)
		if _, err := os.Stat(demoStats); os.IsNotExist(err) {
			slog.Warn(
				"demo_mode: fixture joueur absente — les requêtes /pages/* échoueront avec un IO Error",
				"path", demoStats,
				"hint", "générer la fixture : go run ./cmd/levelup seed-demo --synthetic --out "+cfg.DemoFixturesDir,
			)
		}
	}

	slog.Debug("ouverture DuckDB", "shared", sharedPath, "metadata", metaPath, "shared_social", sharedSocialPath)
	st.pr, st.titleSlug, st.dbPaths = pr, titleSlug, dbPaths
}

// prepareWarehouse crée le dossier warehouse (et extrait la metadata pré-construite hors démo).
func prepareWarehouse(st *bootState) {
	cfg, pr, titleSlug := st.cfg, st.pr, st.titleSlug
	sharedPath := st.dbPaths.shared

	// --- 3a. Migrations (read-write, avant l'ouverture des connexions runtime) ---
	// Sur un clone frais, le dossier warehouse n'existe pas encore : sans lui,
	// runMigrations puis l'ouverture du provider shared échouent et le serveur
	// fait os.Exit(1) avant même d'afficher /setup.
	// En démo, ces deux étapes visent l'arborescence de la FIXTURE : sans cela
	// elles créent/peuplent le warehouse de PRODUCTION que la démo n'ouvre plus.
	if cfg.DemoMode {
		if err := os.MkdirAll(filepath.Dir(sharedPath), 0o755); err != nil {
			slog.Error("création warehouse dir démo échouée", "err", err, "path", filepath.Dir(sharedPath))
			os.Exit(1)
		}
	} else {
		if err := ensureWarehouseDir(pr, titleSlug); err != nil {
			slog.Error("création warehouse dir échouée", "err", err)
			os.Exit(1)
		}
		// Clone frais : extraire la base metadata pré-construite (référentiels prêts)
		// si metadata.duckdb n'existe pas encore. Non-fatal — sinon les migrations
		// créeront une base vide à repeupler via fetch live. Inutile en démo :
		// `seed-demo` livre une metadata déjà migrée et peuplée.
		if err := extractPrebuiltMetadataIfAbsent(pr, titleSlug); err != nil {
			slog.Warn("prebuilt metadata: extraction échouée (non-fatal)", "err", err)
		}
	}
}

// migrateSharedOrRefuse applique les migrations des bases partagées ; un échec laisse le
// serveur sur la porte de démarrage (état d'échec) puis sort au signal d'arrêt.
func migrateSharedOrRefuse(st *bootState) {
	cfg, pr, titleSlug, dbPaths := st.cfg, st.pr, st.titleSlug, st.dbPaths
	metaPath, sharedPath, sharedSocialPath, pvePath := dbPaths.metadata, dbPaths.shared, dbPaths.sharedSocial, dbPaths.pve
	titleConfigDir := filepath.Join(cfg.RepoRoot, "config", "titles", titleSlug)
	// Garde de version, copie de sécurité, migrations (boot_shared_migrations.go). Un échec ne
	// se dégrade pas en démarrage sans migration : le serveur reste sur la porte de démarrage
	// avec son état d'échec, puis sort en erreur au signal d'arrêt (boot_failure.go).
	migrationBackupsDir := pr.MigrationBackupsDir()
	if cfg.DemoMode {
		migrationBackupsDir = "" // fixtures régénérables : pas de copie
	}
	if state, err := migrateSharedAtBoot(context.Background(), newBootMigrationDeps(
		titleSlug, migrationBackupsDir, titleConfigDir, sharedMigrationTargets(dbPaths),
		func() error { return runMigrations(metaPath, sharedPath, sharedSocialPath, pvePath, titleConfigDir) },
	)); err != nil {
		exitAfterRefusal(st, state, err)
	}
	slog.Debug("migrations appliquées")
}

// migrateAdditionalAndPlayerDBs provisionne les bases des titres additionnels actifs puis
// migre les bases joueur, hors démo.
func migrateAdditionalAndPlayerDBs(st *bootState) {
	cfg, pr, titleSlug := st.cfg, st.pr, st.titleSlug

	// --- 3a-bis. Provisionner les DB des titres ADDITIONNELS actifs (MT-16 /
	// day-one 2e titre). No-op en mono-titre (seul halo_infinite actif, déjà
	// provisionné ci-dessus). Un 2e titre déclaré en config (status="active") voit
	// ici ses warehouses créées + migrées (RunForTitleDB → jeu de migrations du
	// titre, fallback set Halo), isolées sous data/titles/<slug>/. Non-fatal : un
	// titre additionnel cassé ne doit jamais empêcher Halo de démarrer.
	if !cfg.DemoMode {
		provisionAdditionalActiveTitles(pr, title.DefaultRegistry())
	}

	// --- 3b. Migrations player DB (TargetPlayer), titre par titre ---
	// Cf. boot_player_migrations.go. Non-fatal : une player DB verrouillée/absente ne
	// doit pas bloquer le boot.
	if !cfg.DemoMode {
		if players, perr := cfg.LoadPlayers(); perr != nil {
			slog.Warn("migrations player: chargement profils échoué (non-fatal)", "err", perr)
		} else {
			n := migratePlayerDBs(context.Background(), players, pr, titleSlug, RunPlayerMigrations)
			slog.Debug("migrations player appliquées", "bases", n)
		}
	}
}

// ensureWarehouseDir crée le répertoire warehouse du titre s'il n'existe pas.
//
// metadata/shared/shared_pve/shared_social vivent tous dans ce dossier.
// runMigrations les ouvre via OpenReadWrite, qui crée le FICHIER .duckdb mais
// PAS le dossier parent (cf. initGlobalXuidAliasesSchema qui MkdirAll
// explicitement avant OpenReadWrite). Sur un clone frais le dossier est absent :
// les migrations échouent puis l'ouverture du provider shared fait os.Exit(1).
//
// Les autres dossiers runtime (players, sessions, cache, WAL, watcher_tokens)
// sont créés à la demande par leurs consommateurs respectifs (profile_service,
// session.NewStore, jobs.Store, persist.BatchQueue, MultiUserTokenStore).
func ensureWarehouseDir(pr *title.PathResolver, titleSlug string) error {
	dir := pr.WarehouseDir(titleSlug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("ensureWarehouseDir %s: %w", dir, err)
	}
	return nil
}

// runMigrations applique les migrations DuckDB dans l'ordre :
// metadata → shared → shared_pve → shared_social.
// Les migrations player sont gérées à l'ouverture de chaque player DB.
//
// prestige ConfigDir : chemin vers config/titles/{slug}/ pour le seed du catalogue
// Prestige (challenge_template + preset_arc). Si vide, le seed est ignoré.
func runMigrations(metaPath, sharedPath, sharedSocialPath, pvePath, prestigeConfigDir string) error {
	if err := prepareMigrationRegistry(prestigeConfigDir); err != nil {
		return err
	}
	if err := migrateMetadataDB(metaPath, prestigeConfigDir); err != nil {
		return err
	}
	if err := migrateSharedDB(sharedPath); err != nil {
		return err
	}
	if err := migrateSharedPvEDB(pvePath); err != nil {
		return err
	}
	return migrateSharedSocialDB(sharedSocialPath)
}

// migrateMetadataDB migre metadata.duckdb puis réconcilie ses seeds (traductions, registre
// d'armes, noms d'armes, citations).
func migrateMetadataDB(metaPath, prestigeConfigDir string) error {
	// 1. metadata.duckdb
	metaDB, err := duckdb.OpenReadWrite(metaPath)
	if err != nil {
		return fmt.Errorf("open metadata rw: %w", err)
	}
	if err := migration.RunForDB(metaDB.SQLDb(), migration.TargetMetadata); err != nil {
		metaDB.Close()
		return fmt.Errorf("metadata migrations: %w", err)
	}
	// Réconciliation idempotente des seeds de traduction (mode_name_tr + playlists
	// FR) : converge les bases dont la migration de seed est "done" mais incomplète
	// (sous-modes / "Quick Play" restés en anglais). Cf. ReconcileMetadataSeeds.
	if err := halomigrations.ReconcileMetadataSeeds(metaDB.SQLDb()); err != nil {
		metaDB.Close()
		return fmt.Errorf("metadata seed reconcile: %w", err)
	}
	// Réconciliation idempotente du registre d'armes canonique (weapons/weapon_ids/
	// weapon_families) : le runner saute add_weapon_registry une fois « done », donc les
	// lignes ajoutées au seed après coup (buckets non-combat, frag grenade…) n'atteignent
	// jamais une DB déjà migrée. Rejeu INSERT OR IGNORE à chaque boot → auto-guérison
	// (cf. ReconcileRegistry). Couvre le titre par défaut ; les titres additionnels
	// sont couverts dans provisionAdditionalTitle.
	if _, err := weapons.ReconcileRegistry(metaDB.SQLDb(), migration.DefaultSlug); err != nil {
		metaDB.Close()
		return fmt.Errorf("metadata weapon registry reconcile: %w", err)
	}
	// V72-06 : source unique des noms d'armes (weapon_name_labels keyée par weapon_key),
	// seedée depuis config/titles/{slug}/mappings/weapon_names.toml. Idempotent au boot
	// (comme le registre). prestigeConfigDir = config/titles/{DefaultSlug}.
	if _, err := weapons.ReconcileNameLabels(metaDB.SQLDb(), migration.DefaultSlug,
		mappings.WeaponNamesPath(filepath.Join(prestigeConfigDir, "mappings"))); err != nil {
		metaDB.Close()
		return fmt.Errorf("metadata weapon name labels reconcile: %w", err)
	}
	// Les citations ajoutées par une mise à jour atteignent une base déjà installée : le seed
	// n'est joué qu'à l'installation ou à la main (`levelup seed citation-mappings`), et
	// l'application de bureau n'a pas de ligne de commande. Une citation manquante ne casse
	// aucune lecture : l'échec est journalisé et le démarrage se poursuit.
	reconcileCitationsAtBoot(context.Background(), metaDB.SQLDb())
	metaDB.Close()
	return nil
}

// reconcileCitationMappings est la réconciliation des citations ; variable pour que les tests
// y substituent un échec.
var reconcileCitationMappings = ops.ReconcileCitationMappings

// reconcileCitationsAtBoot réconcilie les citations sur le handle de l'écrivain de metadata.duckdb.
// Jamais fatale : l'erreur est journalisée.
func reconcileCitationsAtBoot(ctx context.Context, db *sql.DB) {
	res, err := reconcileCitationMappings(ctx, db)
	if err != nil {
		slog.ErrorContext(ctx, "citations: réconciliation au démarrage échouée, démarrage poursuivi", "err", err)
		return
	}
	if res.Inserted > 0 || res.Updated > 0 {
		slog.InfoContext(ctx, "citations réconciliées au démarrage",
			"inserted", res.Inserted, "updated", res.Updated, "unchanged", res.Skipped-res.Updated)
	}
}

// migrateSharedDB migre shared_matches_v2.duckdb.
func migrateSharedDB(sharedPath string) error {
	// 2. shared_matches_v2.duckdb
	sharedDB, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		return fmt.Errorf("open shared rw: %w", err)
	}
	if err := migration.RunForDB(sharedDB.SQLDb(), migration.TargetShared); err != nil {
		sharedDB.Close()
		return fmt.Errorf("shared migrations: %w", err)
	}
	sharedDB.Close()
	return nil
}

// migrateSharedPvEDB migre shared_pve.duckdb s'il existe.
func migrateSharedPvEDB(pvePath string) error {
	// 3. shared_pve.duckdb (optionnel, fichier peut ne pas exister)
	if _, err := os.Stat(pvePath); err == nil {
		pveDB, err := duckdb.OpenReadWrite(pvePath)
		if err != nil {
			return fmt.Errorf("open shared_pve rw: %w", err)
		}
		if err := migration.RunForDB(pveDB.SQLDb(), migration.TargetSharedPvE); err != nil {
			pveDB.Close()
			return fmt.Errorf("shared_pve migrations: %w", err)
		}
		pveDB.Close()
	}
	return nil
}

// migrateSharedSocialDB migre shared_social.duckdb (créé s'il n'existe pas).
func migrateSharedSocialDB(sharedSocialPath string) error {
	// 4. shared_social.duckdb (créé automatiquement s'il n'existe pas)
	socialDB, err := duckdb.OpenReadWrite(sharedSocialPath)
	if err != nil {
		return fmt.Errorf("open shared_social rw: %w", err)
	}
	if err := migration.RunForDB(socialDB.SQLDb(), migration.TargetSharedSocial); err != nil {
		socialDB.Close()
		return fmt.Errorf("shared_social migrations: %w", err)
	}
	socialDB.Close()
	return nil
}

// migratePlayerFriendsAtBoot dote chaque profil configuré de sa propre liste
// d'amis, héritée de l'ancienne liste globale des amis d'app_settings
// (moins son propre gamertag). Best-effort + idempotent : no-op si le fichier
// data/global/player_friends.json existe déjà.
func migratePlayerFriendsAtBoot(ctx context.Context, cfg *config.AppConfig, fs *friendstore.FriendStore) {
	players, err := cfg.LoadPlayers()
	if err != nil {
		slog.WarnContext(ctx, "friends: migration ignorée — lecture des profils impossible", "err", err)
		return
	}
	created, err := friendstore.MigrateFromAppSettings(fs, cfg.AppSettingsPath, players)
	if err != nil {
		slog.WarnContext(ctx, "friends: migration des amis par joueur échouée (non bloquant)", "err", err)
		return
	}
	if created > 0 {
		slog.InfoContext(ctx, "friends: listes d'amis par joueur créées depuis l'ancienne liste globale",
			"created", created)
	}
}

// migrateDefaultGroupAtBoot crée un groupe par défaut "Mon foyer" depuis la liste
// d'amis de l'admin, pour préserver la continuité d'accès au passage au modèle
// multi-groupes. Best-effort + idempotent (no-op si un groupe existe déjà).
// Le propriétaire est l'admin de db_profiles.json ; ses amis résolus en xuid via
// les profils connus deviennent membres.
//
// Source des membres : le store d'amis PAR JOUEUR (2026-09-15), lui-même migré
// juste avant depuis l'ancienne liste globale des amis d'app_settings —
// d'où l'ordre imposé des deux migrations au boot.
func migrateDefaultGroupAtBoot(ctx context.Context, cfg *config.AppConfig, fs *friendstore.FriendStore, gs *groupstore.GroupStore) {
	adminGT := cfg.AdminPlayer()
	if adminGT == "" {
		return // aucun admin désigné → migration impossible
	}
	players, err := cfg.LoadPlayers()
	if err != nil {
		return
	}
	byGamertag := make(map[string]string, len(players))
	for _, p := range players {
		if p.XUID != "" {
			byGamertag[strings.ToLower(p.Gamertag)] = p.XUID
		}
	}
	ownerXUID := byGamertag[strings.ToLower(adminGT)]
	if ownerXUID == "" {
		slog.WarnContext(ctx, "groups: migration ignorée — xuid admin introuvable dans db_profiles", "admin", adminGT)
		return
	}

	adminFriends, err := fs.Get(ownerXUID)
	if err != nil {
		slog.WarnContext(ctx, "groups: migration ignorée — lecture des amis de l'admin impossible",
			"admin", adminGT, "err", err)
		return
	}
	var members []domain.GroupMember
	for _, gt := range adminFriends {
		if xuid := byGamertag[strings.ToLower(gt)]; xuid != "" {
			members = append(members, domain.GroupMember{XUID: xuid, Gamertag: gt})
		}
	}

	created, err := gs.MigrateDefault("Mon foyer", ownerXUID, adminGT, members)
	if err != nil {
		slog.WarnContext(ctx, "groups: migration groupe par défaut échouée (non bloquant)", "err", err)
		return
	}
	if created {
		slog.InfoContext(ctx, "groups: groupe par défaut créé depuis les amis de l'admin",
			"owner", adminGT, "members", len(members)+1)
	}
}
