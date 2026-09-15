// Package playerdirectory — collect.go : réunion des cinq sources par identité.
//
// L'agrégat est construit en une passe par registre, dans un ordre qui n'est pas
// arbitraire : profils, comptes, credentials, suivi live, disque. C'est l'ordre
// de PRIORITÉ du gamertag affiché (profil > compte > token), et il garantit que
// les entrées porteuses d'un xuid existent avant celles qui n'en ont pas (le
// suivi live et les dossiers disque, qui se rattachent alors à la bonne ligne).
package playerdirectory

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
)

// identityBuilder agrège les registres par identité. byKey contient DEUX clés
// possibles pour une même ligne (par xuid et par gamertag) quand le xuid est
// appris après coup ; order ne porte que la clé de création, donc records() ne
// rend jamais de doublon.
type identityBuilder struct {
	byKey      map[string]*domain.IdentityRecord
	order      []string
	byGamertag map[string]string
}

func newIdentityBuilder() *identityBuilder {
	return &identityBuilder{
		byKey:      make(map[string]*domain.IdentityRecord),
		byGamertag: make(map[string]string),
	}
}

// resolve rend la ligne d'une identité, en la créant au besoin.
//
// fallbackKey sert aux comptes sans identité Xbox (ni xuid ni gamertag) : ils
// restent visibles sous leur nom d'utilisateur plutôt que d'être perdus. Rend
// nil seulement si les trois entrées sont vides.
func (b *identityBuilder) resolve(xuid, gamertag, fallbackKey string) *domain.IdentityRecord {
	if xuid != "" {
		key := "xuid:" + xuid
		if rec, ok := b.byKey[key]; ok {
			b.indexGamertag(gamertag, key)
			return rec
		}
		// Une ligne sans xuid a pu être créée plus tôt (profil sans xuid) : lui
		// POSER le xuid plutôt qu'en ouvrir une seconde pour le même joueur.
		if rec := b.byGamertagRecord(gamertag); rec != nil && rec.XUID == "" {
			rec.XUID = xuid
			b.byKey[key] = rec
			return rec
		}
		rec := b.create(key, xuid, gamertag)
		b.indexGamertag(gamertag, key)
		return rec
	}
	if rec := b.byGamertagRecord(gamertag); rec != nil {
		return rec
	}
	if gamertag != "" {
		key := "gt:" + strings.ToLower(gamertag)
		rec := b.create(key, "", gamertag)
		b.indexGamertag(gamertag, key)
		return rec
	}
	if fallbackKey == "" {
		return nil
	}
	if rec, ok := b.byKey[fallbackKey]; ok {
		return rec
	}
	return b.create(fallbackKey, "", "")
}

func (b *identityBuilder) create(key, xuid, gamertag string) *domain.IdentityRecord {
	rec := &domain.IdentityRecord{XUID: xuid, Gamertag: gamertag}
	b.byKey[key] = rec
	b.order = append(b.order, key)
	return rec
}

func (b *identityBuilder) indexGamertag(gamertag, key string) {
	if gamertag == "" {
		return
	}
	if _, ok := b.byGamertag[strings.ToLower(gamertag)]; !ok {
		b.byGamertag[strings.ToLower(gamertag)] = key
	}
}

// byGamertagRecord résout un gamertag sans tenir compte de la casse, comme le
// fait dbprofiles.File.FindKey : un dossier `Spartan` et un profil `spartan`
// sont le même joueur.
func (b *identityBuilder) byGamertagRecord(gamertag string) *domain.IdentityRecord {
	if gamertag == "" {
		return nil
	}
	key, ok := b.byGamertag[strings.ToLower(gamertag)]
	if !ok {
		return nil
	}
	return b.byKey[key]
}

// records rend les lignes dans leur ordre de création, listes vides normalisées
// (le contrat HTTP promet des tableaux, jamais `null`).
func (b *identityBuilder) records() []domain.IdentityRecord {
	out := make([]domain.IdentityRecord, 0, len(b.order))
	for _, key := range b.order {
		rec := b.byKey[key]
		if rec.Profiles == nil {
			rec.Profiles = []domain.ProfileRef{}
		}
		if rec.Watched == nil {
			rec.Watched = []string{}
		}
		sort.Strings(rec.Watched)
		out = append(out, *rec)
	}
	return out
}

// collect lit les cinq sources et rend une ligne par identité, sans anomalies
// (elles se calculent ensuite, sur la ligne complète).
//
// Politique d'erreur : une lecture de REGISTRE en échec (profils, comptes,
// credentials) interrompt — un annuaire amputé d'un registre inventerait des
// anomalies fausses, ce qui est pire que pas d'annuaire du tout. Le balayage
// disque, lui, dégrade par titre après journalisation.
func (d *Directory) collect(ctx context.Context) ([]domain.IdentityRecord, error) {
	b := newIdentityBuilder()
	profileKeys, err := d.addProfiles(ctx, b)
	if err != nil {
		return nil, err
	}
	if err := d.addAccounts(ctx, b); err != nil {
		return nil, err
	}
	if err := d.addTokens(ctx, b); err != nil {
		return nil, err
	}
	d.addWatched(b)
	d.addOrphanDirs(ctx, b, profileKeys)
	return b.records(), nil
}

// addProfiles ingère `db_profiles.json` et rend les clés de profil par titre
// (minuscules), qui serviront à reconnaître les dossiers orphelins.
func (d *Directory) addProfiles(ctx context.Context, b *identityBuilder) (map[string]map[string]bool, error) {
	keysByTitle := make(map[string]map[string]bool)
	if d.profiles == nil {
		slog.ErrorContext(ctx, "player_directory: lecteur de profils absent — annuaire lu sans les profils")
		return keysByTitle, nil
	}
	players, err := d.profiles.LoadPlayers()
	if err != nil {
		slog.ErrorContext(ctx, "player_directory: lecture des profils impossible", "err", err)
		return nil, err
	}
	for _, p := range players {
		slug := p.TitleSlug
		if slug == "" {
			slug = title.DefaultSlug
		}
		rec := b.resolve(p.XUID, p.Gamertag, "")
		if rec == nil {
			slog.WarnContext(ctx, "player_directory: profil sans xuid ni gamertag, ignoré", "title_slug", slug)
			continue
		}
		rec.Profiles = append(rec.Profiles, domain.ProfileRef{
			TitleSlug:   slug,
			Key:         p.PlayerSlug,
			SyncEnabled: p.SyncEnabled,
			AuthOnly:    p.AuthOnly,
			DirExists:   d.dirExists(slug, p.PlayerSlug),
			DBExists:    d.dbExists(slug, p.PlayerSlug),
		})
		if keysByTitle[slug] == nil {
			keysByTitle[slug] = make(map[string]bool)
		}
		keysByTitle[slug][strings.ToLower(p.PlayerSlug)] = true
	}
	return keysByTitle, nil
}

func (d *Directory) addAccounts(ctx context.Context, b *identityBuilder) error {
	if d.accounts == nil {
		return nil
	}
	accounts, err := d.accounts.List()
	if err != nil {
		slog.ErrorContext(ctx, "player_directory: lecture des comptes impossible", "err", err)
		return err
	}
	for _, u := range accounts {
		rec := b.resolve(u.XUID, u.Gamertag, "user:"+strings.ToLower(u.Username))
		if rec == nil {
			continue
		}
		rec.Account = &domain.AccountRef{
			Username:    u.Username,
			Role:        u.Role,
			CreatedAt:   u.CreatedAt,
			LastLoginAt: u.LastLoginAt,
		}
		if rec.Gamertag == "" {
			rec.Gamertag = u.Gamertag
		}
	}
	return nil
}

func (d *Directory) addTokens(ctx context.Context, b *identityBuilder) error {
	if d.tokens == nil {
		return nil
	}
	tokens, err := d.tokens.LoadAll()
	if err != nil {
		slog.ErrorContext(ctx, "player_directory: lecture des credentials impossible", "err", err)
		return err
	}
	for xuid, t := range tokens {
		if t == nil {
			continue
		}
		rec := b.resolve(xuid, t.Gamertag, "")
		if rec == nil {
			continue
		}
		rec.Token = &domain.TokenRef{
			HasRefreshToken: t.OAuthRefreshToken != "",
			ReauthRequired:  t.ReauthRequired,
			LastAuthError:   t.LastAuthError,
			UpdatedAt:       formatTime(t.UpdatedAt),
		}
		if rec.Gamertag == "" {
			rec.Gamertag = t.Gamertag
		}
	}
	return nil
}

func (d *Directory) addWatched(b *identityBuilder) {
	if d.watched == nil {
		return
	}
	for _, w := range d.watched.WatchedPlayers() {
		rec := b.resolve(w.XUID, w.Gamertag, "")
		if rec == nil {
			continue
		}
		if !containsString(rec.Watched, w.TitleSlug) {
			rec.Watched = append(rec.Watched, w.TitleSlug)
		}
	}
}

// addOrphanDirs confronte les dossiers joueur du disque aux clés de profil, par
// titre. Un dossier qu'aucun profil ne déclare est la trace d'un sync qui a
// tourné pour un joueur inconnu de l'app — exactement ce que l'incident du
// 2026-07-23 a laissé derrière lui.
func (d *Directory) addOrphanDirs(ctx context.Context, b *identityBuilder, profileKeys map[string]map[string]bool) {
	if d.fs == nil {
		return
	}
	for _, slug := range d.titles {
		names, err := d.fs.ListPlayerDirs(slug)
		if err != nil {
			slog.WarnContext(ctx, "player_directory: balayage des dossiers joueur impossible pour ce titre",
				"title_slug", slug, "err", err)
			continue
		}
		for _, name := range names {
			if profileKeys[slug][strings.ToLower(name)] {
				continue
			}
			rec := b.resolve("", name, "")
			if rec == nil {
				continue
			}
			rec.OrphanDirs = append(rec.OrphanDirs, domain.OrphanDirRef{TitleSlug: slug, Name: name})
		}
	}
}

func (d *Directory) dirExists(titleSlug, key string) bool {
	return d.fs != nil && d.fs.PlayerDirExists(titleSlug, key)
}

func (d *Directory) dbExists(titleSlug, key string) bool {
	return d.fs != nil && d.fs.PlayerDBExists(titleSlug, key)
}

func containsString(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
