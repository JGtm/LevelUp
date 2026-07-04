package livesync

// appearance_persist.go — hook de persistance de l'IDENTITÉ SPARTAN Halo 5
// (service tag + rendu Spartan + emblème), pour le bloc identitaire du Home.
//
// CONTEXTE : avant ce hook, la Home Halo 5 n'affichait AUCUNE image de Spartan,
// ni emblème, ni service tag — aucun fetch d'appearance H5 n'existait. Le bloc
// identitaire Home lit career_progression (spartan_id + *_image_url) via Q26c.
//
// SOURCES live confirmées (sonde 2026-06-25, JGtm) sur haloplayer :
//   - /h5/profiles/{gt}/appearance → 200 JSON : ServiceTag + Emblem{ids/couleurs}.
//   - /h5/profiles/{gt}/spartan     → 302 vers image.halocdn.com (rendu PNG armure).
//   - /h5/profiles/{gt}/emblem      → 302 vers image.halocdn.com (emblème PNG).
//
// STOCKAGE : les deux endpoints image rendent des URL CDN SIGNÉES (hash non
// reproductible côté client) → on TÉLÉCHARGE les octets PNG et on les écrit dans
// le cache d'assets local ({RepoRoot}/data/cache/{kind}/halo_5/{slug}.png), via le
// même LocalFSStore que le resolver. Le bloc Home pose alors la valeur RELATIVE
// {slug} dans career_progression et le handler /api/v1/assets/spartan/... la sert
// local-first (zéro fetch gamecms, zéro souci d'expiration d'URL signée).
//
// PERSISTANCE ART-safe : career_progression est append-only (INSERT-only via
// CareerLiveRepo.InsertCareerProgressionPartial). Le service tag est mappé sur la
// colonne spartan_id (convention existante — cf. domain.CareerProgressionPartial),
// le rendu Spartan sur banner_image_url (fond du bloc Home), l'emblème sur
// emblem_image_url (avatar circulaire). AUCUN changement de schéma ni de front.

import (
	"context"
	"fmt"
	"strings"

	"levelup/go-api/internal/assets"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/platform/duckdb"
	syncpkg "levelup/go-api/internal/sync"
	"levelup/go-api/internal/util/pointers"
)

// AppearanceSource est la surface live minimale consommée par le hook appearance.
// *halo5.Client la satisfait. Définie côté consommateur (livesync) pour rester
// mockable en test sans réseau, sans élargir CaptureSource/h5Source de l'adapter.
type AppearanceSource interface {
	GetAppearance(ctx context.Context, gamertag string) (*halo5.H5Appearance, error)
	GetSpartanRenderPNG(ctx context.Context, gamertag string) ([]byte, string, error)
	GetEmblemPNG(ctx context.Context, gamertag string) ([]byte, string, error)
}

var _ AppearanceSource = (*halo5.Client)(nil)

// AppearanceResult résume une passe de persistance d'appearance (logs / CLI).
type AppearanceResult struct {
	ServiceTag      string // service tag lu (vide si absent)
	SpartanRendered bool   // PNG rendu Spartan téléchargé + écrit
	EmblemRendered  bool   // PNG emblème téléchargé + écrit
	Persisted       bool   // une ligne career_progression a été insérée
}

// PersistAppearance fetch l'appearance Halo 5 d'un joueur (service tag + emblème),
// télécharge le rendu Spartan + l'emblème PNG dans le cache d'assets, et persiste
// l'identité dans la player DB h5 (career_progression, append-only). Best-effort par
// composant : un échec image n'avorte pas l'écriture du service tag, et inversement.
//
//   - src          : source live h5 authentifiée (cf. AppearanceSource).
//   - playerDBPath : data/titles/halo_5/players/{gamertag}/stats.duckdb.
//   - cacheRootDir  : racine du cache d'assets ({RepoRoot}/data/cache).
//   - gamertag     : le joueur consulté (l'API h5 indexe par gamertag, pas xuid).
//   - xuid         : xuid Xbox du joueur (clé career_progression). Requis (non vide).
func PersistAppearance(
	ctx context.Context,
	src AppearanceSource,
	playerDBPath, cacheRootDir, gamertag, xuid string,
) (AppearanceResult, error) {
	var res AppearanceResult
	if strings.TrimSpace(gamertag) == "" || strings.TrimSpace(xuid) == "" {
		return res, fmt.Errorf("h5 appearance: gamertag/xuid requis (gt=%q xuid=%q)", gamertag, xuid)
	}

	app, err := src.GetAppearance(ctx, gamertag)
	if err != nil {
		return res, fmt.Errorf("h5 appearance: GetAppearance(%s): %w", gamertag, err)
	}

	partial := &duckdb.CareerProgressionPartial{}
	slug := appearanceAssetSlug(gamertag)
	store := assets.NewLocalFSStore(cacheRootDir)

	// Service tag → spartan_id (convention existante).
	if st := strings.TrimSpace(app.ServiceTag); st != "" {
		res.ServiceTag = st
		partial.SpartanID = pointers.Ptr(st)
	}

	// Rendu Spartan → banner_image_url (fond du bloc Home).
	if persistAppearancePNG(ctx, store, assets.KindSpartanBanner, slug,
		func() ([]byte, string, error) { return src.GetSpartanRenderPNG(ctx, gamertag) }) {
		res.SpartanRendered = true
		partial.BannerImageURL = pointers.Ptr(slug)
	}

	// Emblème → emblem_image_url (avatar circulaire du bloc Home).
	if persistAppearancePNG(ctx, store, assets.KindSpartanEmblem, slug,
		func() ([]byte, string, error) { return src.GetEmblemPNG(ctx, gamertag) }) {
		res.EmblemRendered = true
		partial.EmblemImageURL = pointers.Ptr(slug)
	}

	if partial.IsEmpty() {
		return res, nil // rien d'exploitable rendu — pas d'écriture.
	}

	// OpenPlayerDB crée le dossier + EnsurePlayerSchema (career_progression). Idempotent.
	db, err := syncpkg.OpenPlayerDB(playerDBPath)
	if err != nil {
		return res, fmt.Errorf("h5 appearance: open player DB: %w", err)
	}
	defer db.Close()

	repo := duckdb.NewCareerLiveRepo(&duckdb.PlayerDB{Player: db, XUID: xuid, Gamertag: gamertag, TitleSlug: halo5.TitleSlug})
	inserted, err := repo.InsertCareerProgressionPartial(ctx, xuid, partial)
	if err != nil {
		return res, fmt.Errorf("h5 appearance: persist career_progression: %w", err)
	}
	res.Persisted = inserted
	return res, nil
}

// persistAppearancePNG télécharge un PNG via fetch() et l'écrit dans le store
// d'assets sous (kind, halo_5, slug). Best-effort : un échec fetch ou un corps non
// PNG renvoie false sans erreur (le caller dégrade ce composant). Le content-type
// renvoyé par l'endpoint n'est pas fiable (CDN) → on s'appuie sur les magic bytes
// PNG via le store (detectContentType à la lecture).
func persistAppearancePNG(
	ctx context.Context,
	store *assets.LocalFSStore,
	kind assets.Kind,
	slug string,
	fetch func() ([]byte, string, error),
) bool {
	data, _, err := fetch()
	if err != nil || len(data) == 0 {
		return false
	}
	if !isPNGBytes(data) {
		return false
	}
	ref := assets.Ref{Kind: kind, TitleID: halo5.TitleSlug, ID: slug}
	if err := store.PersistBinary(ctx, ref, assets.BinaryPayload{ContentType: assets.MimeImagePNG, Bytes: data}); err != nil {
		return false
	}
	return true
}

// isPNGBytes valide la signature magique PNG (défense : ne pas persister un corps
// d'erreur HTML/JSON renvoyé par le CDN sous un 200 inattendu).
func isPNGBytes(b []byte) bool {
	return len(b) >= 8 &&
		b[0] == 0x89 && b[1] == 0x50 && b[2] == 0x4e && b[3] == 0x47 &&
		b[4] == 0x0d && b[5] == 0x0a && b[6] == 0x1a && b[7] == 0x0a
}

// appearanceAssetSlug normalise un gamertag en identifiant d'asset sûr (minuscule,
// caractères non [a-z0-9-_] remplacés par '-'). Stable par joueur → le re-fetch
// écrase le même fichier (atomique tmp+rename côté store). Le slug est aussi la
// valeur relative posée dans career_progression (resolue local-first par le handler
// /api/v1/assets/spartan/{type}/halo_5/{slug}).
func appearanceAssetSlug(gamertag string) string {
	var sb strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(gamertag)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			sb.WriteRune(r)
		default:
			sb.WriteByte('-')
		}
	}
	s := strings.Trim(sb.String(), "-")
	if s == "" {
		return "player"
	}
	return s
}

// strPtr retourne un pointeur vers s (helper local — pas de dépendance externe).
