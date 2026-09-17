// Package playerdirectory — purge.go : la sortie d'une identité (ADR 0035 D6).
//
// CE QU'ELLE NE TOUCHE JAMAIS : l'entrepôt partagé. Les matchs déjà persistés
// dans `shared_matches_v2.duckdb` portent aussi les données des adversaires et
// des coéquipiers du joueur purgé, et l'entrepôt est append-only par construction
// (ADR 0026). Un test compare le sha256 du fichier avant et après une purge.
// Ce paquet n'importe d'ailleurs aucun paquet DuckDB et n'ouvre aucune base —
// garde-rail `internal/archlint/no_duckdb_import_playerdirectory_test.go`.
//
// L'ORDRE compte : le suivi live part EN PREMIER. Retirer d'abord le profil
// laisserait un poller vivant sur un joueur dont les registres disparaissent
// sous lui — au mieux du bruit, au pire un sync soumis en pleine purge. Le compte
// part EN DERNIER : c'est lui qui donne son nom à l'identité dans les journaux.
//
// UNE ÉTAPE EN ÉCHEC N'ARRÊTE PAS LES SUIVANTES. Un token qu'on n'a pas pu
// retirer ne doit pas laisser en vie le compte, les profils et les dossiers :
// le rapport est TOUJOURS complet, et l'erreur rendue agrège les échecs.
package playerdirectory

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/domain"
)

// ProfilePurger retire les entrées de profil d'un gamertag et leurs dossiers
// joueur. Implémenté par *service.ProfileService (writer unique de
// `db_profiles.json`).
type ProfilePurger interface {
	PurgeIdentityData(gamertag string, titleSlugs []string) (map[string]bool, error)
}

// TokenPurger retire les credentials d'un xuid (ADR 0023). Implémenté par
// *auth.MultiUserTokenStore.
type TokenPurger interface {
	Remove(xuid string) error
}

// GroupsPurger retire un xuid de ses groupes. Implémenté par
// *groupstore.GroupStore.
type GroupsPurger interface {
	ListForXUID(xuid string) ([]domain.Group, error)
	RemoveMember(id, xuid string) error
}

// AccountPurger supprime le compte de connexion. Implémenté par
// *userstore.Store.
type AccountPurger interface {
	Delete(username string) error
}

// WatcherRemover retire un xuid du suivi live et rend les titres retirés.
// Implémenté par *watcher.Daemon. Absent en CLI (aucun daemon dans ce process).
type WatcherRemover interface {
	RemovePlayer(ctx context.Context, xuid string) []string
}

// PurgeDeps porte les écritures destructrices de l'annuaire. Une dépendance nil
// ne fait pas échouer la purge : elle rend en échec les étapes qui en dépendent,
// avec leur raison — mieux vaut une purge partielle DITE qu'une purge partielle
// silencieuse.
type PurgeDeps struct {
	Profiles ProfilePurger
	Tokens   TokenPurger
	Groups   GroupsPurger
	Accounts AccountPurger
	Watcher  WatcherRemover
}

// ErrPurgeAdminRefused : on ne purge pas l'administrateur de l'instance par
// cette porte. Retirer le dernier compte admin fermerait l'administration a clef
// de l'exterieur ; le faire volontairement releve d'une action manuelle assumee.
var ErrPurgeAdminRefused = errors.New("player_directory: purge d'un compte administrateur refusee")

// ErrPurgeInvalidXUID : une identite se purge par son xuid (ADR 0035 D1).
var ErrPurgeInvalidXUID = errors.New("player_directory: xuid vide")

// purgeRun porte l'état d'une purge en cours : le rapport en construction et les
// échecs accumulés.
type purgeRun struct {
	ctx    context.Context
	deps   PurgeDeps
	fs     FS
	dryRun bool
	xuid   string
	report domain.PurgeReport
	errs   []error
}

// Purge retire une identité de tous les registres, dans l'ordre de l'ADR 0035 D6.
// Voir l'en-tête du fichier pour l'ordre, la politique d'erreur et ce qui n'est
// jamais touché.
func (d *Directory) Purge(ctx context.Context, xuid string,
	opts domain.PurgeOptions) (domain.PurgeReport, error) {
	if xuid == "" {
		return domain.PurgeReport{}, ErrPurgeInvalidXUID
	}
	rec, err := d.Get(ctx, xuid)
	if err != nil {
		return domain.PurgeReport{XUID: xuid}, err
	}
	if admin := adminAccountOf(rec); admin != "" {
		slog.WarnContext(ctx, "player_directory: purge refusée — compte administrateur",
			"xuid", xuid, "username", admin)
		return domain.PurgeReport{XUID: rec.XUID, Gamertag: rec.Gamertag, DryRun: opts.DryRun},
			ErrPurgeAdminRefused
	}

	r := &purgeRun{
		// rec.XUID, pas la clé reçue : une identité sans xuid désignée par son
		// gamertag (dossier orphelin) n'a ni credentials, ni groupes, ni compte.
		ctx: ctx, deps: d.purge, fs: d.fs, dryRun: opts.DryRun, xuid: rec.XUID,
		report: domain.PurgeReport{XUID: rec.XUID, Gamertag: rec.Gamertag, DryRun: opts.DryRun},
	}
	slog.InfoContext(ctx, "player_directory: purge d'identité", "xuid", xuid,
		"gamertag", rec.Gamertag, "dry_run", opts.DryRun,
		"profiles", len(rec.Profiles), "orphan_dirs", len(rec.OrphanDirs))

	r.removeWatched(rec)
	r.removeProfiles(rec)
	r.removeOrphanDirs(rec)
	r.removeToken(rec)
	r.removeGroups()
	r.removeAccount(rec)

	return r.report, errors.Join(r.errs...)
}

// step inscrit une étape au rapport. En SIMULATION, `do` n'est jamais appelée et
// l'étape reste Done=false sans erreur : le rapport dit alors ce qui SERAIT fait.
func (r *purgeRun) step(kind, target string, do func() error) {
	if r.dryRun {
		r.report.Steps = append(r.report.Steps, domain.PurgeStep{Kind: kind, Target: target})
		return
	}
	st := domain.PurgeStep{Kind: kind, Target: target}
	if err := do(); err != nil {
		st.Err = err.Error()
		r.errs = append(r.errs, fmt.Errorf("%s %s: %w", kind, target, err))
		slog.ErrorContext(r.ctx, "player_directory: étape de purge en échec", "err", err,
			"xuid", r.xuid, "kind", kind, "target", target)
	} else {
		st.Done = true
		slog.InfoContext(r.ctx, "player_directory: étape de purge exécutée",
			"xuid", r.xuid, "kind", kind, "target", target)
	}
	r.report.Steps = append(r.report.Steps, st)
}

// removeWatched retire le suivi live. Un seul appel retire tous les titres du
// xuid ; le rapport en garde une ligne par titre, pour dire lequel a bougé.
func (r *purgeRun) removeWatched(rec domain.IdentityRecord) {
	if len(rec.Watched) == 0 {
		return
	}
	removed := map[string]bool{}
	if !r.dryRun && r.deps.Watcher != nil {
		for _, slug := range r.deps.Watcher.RemovePlayer(r.ctx, r.xuid) {
			removed[slug] = true
		}
	}
	for _, slug := range rec.Watched {
		r.step(domain.PurgeStepWatcher, slug, func() error {
			if r.deps.Watcher == nil {
				return errors.New("aucun retrait de suivi live cable dans ce process")
			}
			if !removed[slug] {
				return errors.New("le suivi live n'a pas ete retire")
			}
			return nil
		})
	}
}

// removeProfiles retire les entrées de `db_profiles.json` et les dossiers joueur
// qu'elles déclarent. Groupé par CLÉ de profil : une seule mutation atomique par
// clé, ce que l'invariant « au moins un titre actif » impose (cf.
// ProfileService.PurgeIdentityData).
func (r *purgeRun) removeProfiles(rec domain.IdentityRecord) {
	if len(rec.Profiles) == 0 {
		return
	}
	byKey := map[string][]string{}
	var order []string
	for _, p := range rec.Profiles {
		if _, seen := byKey[p.Key]; !seen {
			order = append(order, p.Key)
		}
		byKey[p.Key] = append(byKey[p.Key], p.TitleSlug)
	}
	for _, key := range order {
		slugs := byKey[key]
		removed, err := r.purgeProfileKey(key, slugs)
		for _, slug := range slugs {
			r.step(domain.PurgeStepProfile, slug+"/"+key, func() error {
				if err != nil {
					return err
				}
				if !removed[slug] {
					return errors.New("entree retiree mais dossier joueur non supprime")
				}
				return nil
			})
		}
	}
}

// purgeProfileKey exécute (hors simulation) le retrait d'une clé de profil pour
// ses titres. Rend, par titre, si le dossier a disparu.
func (r *purgeRun) purgeProfileKey(key string, slugs []string) (map[string]bool, error) {
	if r.dryRun {
		return nil, nil
	}
	if r.deps.Profiles == nil {
		return nil, errors.New("aucun retrait de profil cable dans ce process")
	}
	return r.deps.Profiles.PurgeIdentityData(key, slugs)
}

// removeOrphanDirs supprime les dossiers joueur qu'aucun profil ne déclarait —
// la trace exacte que laisse un sync ayant tourné pour un joueur inconnu de l'app.
func (r *purgeRun) removeOrphanDirs(rec domain.IdentityRecord) {
	for _, dir := range rec.OrphanDirs {
		slug, name := dir.TitleSlug, dir.Name
		r.step(domain.PurgeStepOrphanDir, slug+"/"+name, func() error {
			if r.fs == nil {
				return errors.New("aucun acces disque cable dans ce process")
			}
			return r.fs.RemovePlayerDir(slug, name)
		})
	}
}

func (r *purgeRun) removeToken(rec domain.IdentityRecord) {
	if rec.Token == nil {
		return
	}
	r.step(domain.PurgeStepToken, r.xuid, func() error {
		if r.deps.Tokens == nil {
			return errors.New("aucun retrait de credentials cable dans ce process")
		}
		return r.deps.Tokens.Remove(r.xuid)
	})
}

// removeGroups retire le xuid de chaque groupe dont il est membre. La liste est
// relue ici (et non prise de l'IdentityRecord) : l'annuaire ne porte pas les
// groupes, ils ne sont pas un des cinq registres qu'il compose.
//
// Un groupe dont l'identité purgée est PROPRIÉTAIRE refuse le retrait
// (`ErrCannotRemoveOwner`, groupstore) : l'étape est rendue en échec avec son
// identifiant de groupe, parce que supprimer le groupe emporterait les accès
// d'autres joueurs — c'est une décision d'administrateur, pas d'une purge.
func (r *purgeRun) removeGroups() {
	if r.deps.Groups == nil {
		return
	}
	if r.xuid == "" {
		return // identité sans xuid : aucune appartenance de groupe possible
	}
	groups, err := r.deps.Groups.ListForXUID(r.xuid)
	if err != nil {
		slog.ErrorContext(r.ctx, "player_directory: lecture des groupes impossible", "err", err,
			"xuid", r.xuid)
		r.errs = append(r.errs, fmt.Errorf("groups list: %w", err))
		return
	}
	for _, g := range groups {
		id := g.ID
		r.step(domain.PurgeStepGroup, id, func() error {
			return r.deps.Groups.RemoveMember(id, r.xuid)
		})
	}
}

func (r *purgeRun) removeAccount(rec domain.IdentityRecord) {
	// Tous les comptes du xuid, le principal ET les doublons (R1) : une purge
	// d'identité ne doit laisser aucun compte capable de se reconnecter.
	for _, acc := range allAccounts(rec) {
		username := acc.Username
		r.step(domain.PurgeStepAccount, username, func() error {
			if r.deps.Accounts == nil {
				return errors.New("aucune suppression de compte cablee dans ce process")
			}
			return r.deps.Accounts.Delete(username)
		})
	}
}

// allAccounts rend le compte principal suivi des doublons, dans l'ordre de lecture.
func allAccounts(rec domain.IdentityRecord) []domain.AccountRef {
	if rec.Account == nil {
		return rec.DuplicateAccounts
	}
	return append([]domain.AccountRef{*rec.Account}, rec.DuplicateAccounts...)
}

// adminAccountOf rend le nom du premier compte administrateur porté par
// l'identité (principal ou doublon), ou "" s'il n'y en a aucun.
func adminAccountOf(rec domain.IdentityRecord) string {
	for _, acc := range allAccounts(rec) {
		if acc.Role == domain.RoleAdmin {
			return acc.Username
		}
	}
	return ""
}
