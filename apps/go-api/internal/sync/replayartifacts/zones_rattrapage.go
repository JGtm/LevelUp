package replayartifacts

// zones_rattrapage.go — QUAND LA CHAINE RECUPERE UN FILM, ELLE DONNE SES ZONES NOMMEES A UNE
// CARTE QUI N'EN A PAS.
//
// # Le trou que ce fichier ferme
//
// Le catalogue versionne des zones (`reference/map_callouts.json`) ne couvre que les cartes
// integrees et les cartes Forge d'un inventaire DATE : une carte Forge jouee depuis, ou une
// autre version d'une carte connue (autre map_id), n'a pas de zones a l'ecran alors que le jeu
// les affiche. Ses zones sont dans sa variante (`.mvar`), que la chaine des films sait deja
// rapatrier pour les socles (mvar_rattrapage.go).
//
// # Ce qu'il fait, et rien de plus
//
// Carte DISTINCTE du lot (une fois par carte, jamais par match) :
//
//	zones deja resolues (mapcatalog.SourcesDeZones)   -> rien, pas meme un appel ;
//	sinon variante au cache (`<cache>/mvar/<map_id>/`) -> lecture hors ligne ;
//	sinon fournisseur UGC disponible                  -> UN appel, depot au cache ;
//	zones lues + au moins une nommee                  -> AJOUT au catalogue GENERE.
//
// Il tourne APRES le rattrapage des socles : une carte que celui-ci vient de rapatrier est deja
// au cache, et ne coute pas un second appel. Une carte sans zone (carte integree republiee en
// asset, canevas) reste au cache : le cycle suivant la relit HORS LIGNE et conclut pareil, sans
// appel reseau.
//
// # Il n'ecrit que le catalogue genere
//
// `reference/generated/map_callouts.json` (PathResolver.MapCalloutsOverlayPath), ignore par
// git, en ajout seul (mapcatalog.AddCalloutsOverlayEntry). Le catalogue versionne n'est jamais
// touche : garde-rail `archlint/no_runtime_versioned_catalog_write_test.go`.
//
// # Aucune recuisson
//
// Les zones ne sont pas dans l'artefact : le service les resout a la LECTURE (cascade de
// `mapcatalog.SourcesDeZones`). Une carte ajoutee ici s'affiche au prochain rejeu ouvert, sur
// tous ses matchs, anciens compris.
//
// # Best-effort STRICT
//
// Capabilities, lexique, reseau, parse, ecriture : TOUT echec est journalise et compte, et le
// cycle continue. Un film ne se perd jamais a cause d'une zone.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/mapcatalog"
	"levelup/go-api/internal/observability"
)

// entreeZonesFn est la couture qui rend le CHEMIN NOMINAL testable sans `.mvar` reel porteur
// de zones : le depot n'en versionne aucun (cf. entryFromMvarFn, meme raison). La chaine
// elle-meme est couverte par les tests de `mapcatalog` et de `mapvar`. Le code de production ne
// la reassigne jamais.
var entreeZonesFn = mapcatalog.EntreeCalloutsForge

// StatutZones dit ce que le rattrapage a fait d'une carte.
type StatutZones string

const (
	// ZonesDejaCouvertes : la cascade donne deja des zones a la carte — rien n'est fait.
	ZonesDejaCouvertes StatutZones = "deja_couverte"
	// ZonesAjoutees : la carte entre au catalogue genere.
	ZonesAjoutees StatutZones = "ajoutee"
	// ZonesAjoutables : passe A BLANC — la carte entrerait au catalogue genere.
	ZonesAjoutables StatutZones = "ajoutable"
	// ZonesAbsentes : la variante ne pose aucune zone nommee.
	ZonesAbsentes StatutZones = "sans_zone"
	// ZonesSansLibelle : des zones, mais aucun nom de lieu connu du lexique.
	ZonesSansLibelle StatutZones = "sans_libelle"
	// ZonesATelecharger : variante absente du cache, et la passe n'a pas le droit d'aller sur
	// le reseau (hors ligne ou a blanc).
	ZonesATelecharger StatutZones = "a_telecharger"
	// ZonesEchecTelechargement, ZonesVarianteIllisible, ZonesEchecEcriture : les trois echecs.
	ZonesEchecTelechargement StatutZones = "echec_telechargement"
	ZonesVarianteIllisible   StatutZones = "variante_illisible"
	ZonesEchecEcriture       StatutZones = "echec_ecriture"
)

// ResultatZones est le compte rendu d'UNE carte.
type ResultatZones struct {
	MapID  string
	Statut StatutZones
	// Origine : l'essai de la cascade qui couvrait deja la carte (statut deja_couverte).
	Origine mapcatalog.OrigineZones
	// Telechargee : la variante a ete rapatriee par le reseau pendant ce traitement.
	Telechargee bool
	// Couverture : la mesure des libelles, quand la variante a ete lue.
	Couverture mapcatalog.CouvertureLibelles
	// Err : la cause d'un echec.
	Err error
}

// RattrapageZones porte ce qu'il faut pour donner ses zones a une carte. Il se prepare une fois
// par cycle (ou par passe de CLI) : les catalogues sont lus une fois, pas par carte.
type RattrapageZones struct {
	titleSlug string
	cacheRoot string
	overlay   string
	// fetcher nil = HORS LIGNE : seules les variantes deja au cache sont lues.
	fetcher MvarFetcher
	// aBlanc : rien n'est ecrit (ni cache, ni catalogue) et rien n'est telecharge.
	aBlanc  bool
	sources mapcatalog.SourcesDeZones
	lexique mapcatalog.Lexique
}

// OptionsRattrapageZones : les reglages d'une preparation.
type OptionsRattrapageZones struct {
	RepoRoot, TitleSlug, CacheRoot string
	// Fetcher nil = hors ligne (cf. RattrapageZones.fetcher).
	Fetcher MvarFetcher
	// ABlanc : passe sans effet de bord, ni reseau.
	ABlanc bool
}

// PreparerRattrapageZones lit les sources de la cascade et le lexique. Un lexique illisible est
// une ERREUR : sans lui aucune zone ne porte de nom, donc aucune carte n'est publiable, et une
// passe qui ne le dirait pas laisserait croire que les cartes n'ont pas de zones.
func PreparerRattrapageZones(ctx context.Context, o OptionsRattrapageZones) (*RattrapageZones, error) {
	res := title.NewPathResolver(o.RepoRoot)
	lex, err := mapcatalog.ChargerLexique(res.MapCalloutsLexiquePath(o.TitleSlug))
	if err != nil {
		return nil, fmt.Errorf("lexique des noms de lieu illisible : %w", err)
	}
	return &RattrapageZones{
		titleSlug: o.TitleSlug,
		cacheRoot: o.CacheRoot,
		overlay:   res.MapCalloutsOverlayPath(o.TitleSlug),
		fetcher:   o.Fetcher,
		aBlanc:    o.ABlanc,
		sources:   mapcatalog.ChargerSourcesDeZones(ctx, res, o.TitleSlug),
		lexique:   lex,
	}, nil
}

// bilanZones compte ce qu'un cycle de rattrapage des zones a fait. Publie en jauges : sans
// denominateurs, « une carte ajoutee » ne se juge pas.
type bilanZones struct {
	ajoutees, dejaCouvertes, sansZone, sansLibelle, telechargees, echecs int
}

// compter verse le resultat d'une carte au bilan.
func (b *bilanZones) compter(r ResultatZones) {
	if r.Telechargee {
		b.telechargees++
	}
	switch r.Statut {
	case ZonesAjoutees:
		b.ajoutees++
	case ZonesDejaCouvertes:
		b.dejaCouvertes++
	case ZonesAbsentes:
		b.sansZone++
	case ZonesSansLibelle:
		b.sansLibelle++
	case ZonesEchecTelechargement, ZonesVarianteIllisible, ZonesEchecEcriture:
		b.echecs++
	}
}

// rattraperZonesNommees donne leurs zones aux cartes DISTINCTES du lot qui n'en ont pas.
//
// Appelee une fois par lot, APRES le rattrapage des socles (cf. l'en-tete). La porte est la
// capability `map.forge_callouts` : un titre qui ne la declare pas ne lit rien et n'ecrit rien.
// `fetcher` nil n'eteint pas l'etape : les variantes deja au cache sont lues hors ligne.
func rattraperZonesNommees(ctx context.Context, d Deps, work []buildWork, fetcher MvarFetcher) {
	var b bilanZones
	// Publiees sur TOUS les chemins de sortie, y compris les sorties precoces (meme patron que
	// publierBilanRattrapage).
	defer func() { publierBilanZones(ctx, b) }()
	armee, incident := porteCapability(ctx, d, games.CapMapForgeCallouts, "zones nommees Forge", nil)
	if !armee {
		if incident {
			b.echecs++
		}
		return
	}
	r, err := PreparerRattrapageZones(ctx, OptionsRattrapageZones{
		RepoRoot: d.RepoRoot, TitleSlug: d.TitleSlug, CacheRoot: d.CacheRoot, Fetcher: fetcher,
	})
	if err != nil {
		slog.WarnContext(ctx, "zones nommees: rattrapage saute — les films sont recuperes normalement",
			"err", err, "titleSlug", d.TitleSlug)
		b.echecs++
		return
	}
	vues := make(map[string]bool, len(work))
	for _, w := range work {
		mapID := w.facts.MapID
		if mapID == "" || vues[mapID] {
			continue
		}
		vues[mapID] = true
		res := r.TraiterCarte(ctx, mapcatalog.IdentitesDeCarte{MapID: mapID, Noms: w.mapNames})
		b.compter(res)
		journaliserCarte(ctx, res, w.mapNames, d.TitleSlug)
	}
}

// journaliserCarte dit ce qui est arrive a une carte : l'ajout et les echecs en clair, les
// cartes deja couvertes et sans zone en DEBUG (elles reviennent a chaque film de la carte).
func journaliserCarte(ctx context.Context, r ResultatZones, noms []string, titleSlug string) {
	attrs := []any{"map_id", r.MapID, "noms", noms, "statut", string(r.Statut), "titleSlug", titleSlug}
	switch r.Statut {
	case ZonesAjoutees:
		slog.InfoContext(ctx, "zones nommees: carte AJOUTEE au catalogue genere", append(attrs,
			"zones", r.Couverture.Zones, "nommees", r.Couverture.Nommees,
			"string_id_sans_libelle", r.Couverture.SansLibelle())...)
	case ZonesSansLibelle:
		slog.InfoContext(ctx, "zones nommees: zones sans aucun nom connu du lexique — carte non publiee",
			append(attrs, "zones", r.Couverture.Zones, "string_id_sans_libelle", r.Couverture.SansLibelle())...)
	case ZonesEchecTelechargement, ZonesVarianteIllisible, ZonesEchecEcriture:
		slog.WarnContext(ctx, "zones nommees: carte non traitee — le film est recupere normalement",
			append(attrs, "err", r.Err)...)
	default:
		slog.DebugContext(ctx, "zones nommees: carte examinee", attrs...)
	}
}

// publierBilanZones publie le bilan en JAUGES, meme a zero, et SUR TOUS LES CHEMINS.
func publierBilanZones(ctx context.Context, b bilanZones) {
	titre := ctxkeys.TitleSlug(ctx)
	observability.SetIntT(titre, JaugeZonesAjoutees, int64(b.ajoutees))
	observability.SetIntT(titre, JaugeZonesDejaCouvertes, int64(b.dejaCouvertes))
	observability.SetIntT(titre, JaugeZonesSansZone, int64(b.sansZone))
	observability.SetIntT(titre, JaugeZonesSansLibelle, int64(b.sansLibelle))
	observability.SetIntT(titre, JaugeZonesTelechargees, int64(b.telechargees))
	observability.SetIntT(titre, JaugeZonesEchecs, int64(b.echecs))
	if b.ajoutees > 0 || b.echecs > 0 {
		slog.InfoContext(ctx, "zones nommees: bilan du rattrapage",
			"ajoutees", b.ajoutees, "deja_couvertes", b.dejaCouvertes, "sans_zone", b.sansZone,
			"sans_libelle", b.sansLibelle, "telechargees", b.telechargees, "echecs", b.echecs)
	}
}

// TraiterCarte donne ses zones a UNE carte si elle n'en a pas (cf. l'en-tete du fichier).
func (r *RattrapageZones) TraiterCarte(ctx context.Context, id mapcatalog.IdentitesDeCarte) ResultatZones {
	out := ResultatZones{MapID: id.MapID}
	if _, origine, ok := r.sources.Resoudre(id); ok {
		out.Statut, out.Origine = ZonesDejaCouvertes, origine
		return out
	}
	blob, statut, err := r.variante(ctx, id.MapID, &out)
	if statut != "" {
		out.Statut, out.Err = statut, err
		return out
	}
	entry, couv, err := entreeZonesFn(blob, r.lexique)
	if err != nil {
		out.Statut, out.Err = ZonesVarianteIllisible, err
		return out
	}
	out.Couverture = couv
	switch couv.Verdict() {
	case mapcatalog.VerdictSansZone:
		out.Statut = ZonesAbsentes
	case mapcatalog.VerdictSansLibelle:
		out.Statut = ZonesSansLibelle
	case mapcatalog.VerdictPubliable:
		out.Statut, out.Err = r.publier(id.MapID, entry)
	}
	return out
}

// variante rend les octets de la variante d'une carte : cache d'abord, reseau ensuite. Un
// statut NON VIDE dit que la carte s'arrete la (et pourquoi).
func (r *RattrapageZones) variante(ctx context.Context, mapID string, out *ResultatZones,
) ([]byte, StatutZones, error) {
	blob, err := lireMvarEnCache(r.cacheRoot, mapID)
	switch {
	case err == nil:
		return blob, "", nil
	case !errors.Is(err, os.ErrNotExist):
		return nil, ZonesVarianteIllisible, err
	case r.fetcher == nil || r.aBlanc:
		return nil, ZonesATelecharger, nil
	}
	blob, base, err := r.fetcher.FetchMvarForMap(ctx, mapID, mapcatalog.NomDeLaVariante)
	if err != nil {
		return nil, ZonesEchecTelechargement, err
	}
	out.Telechargee = true
	if err := deposerMvar(r.cacheRoot, mapID, base, blob); err != nil {
		// Le depot est une TRACE (la relecture hors ligne au cycle suivant), pas une
		// dependance : la carte se traite quand meme.
		slog.WarnContext(ctx, "zones nommees: depot de la variante au cache echoue",
			"map_id", mapID, "err", err, "titleSlug", r.titleSlug)
	}
	return blob, "", nil
}

// publier range une carte au catalogue genere (ou dit qu'elle y entrerait, a blanc).
func (r *RattrapageZones) publier(mapID string, entry replay.MapCalloutsEntry) (StatutZones, error) {
	if r.aBlanc {
		return ZonesAjoutables, nil
	}
	err := mapcatalog.AddCalloutsOverlayEntry(r.overlay, r.titleSlug, mapID, entry)
	switch {
	case errors.Is(err, mapcatalog.ErrEntryExists):
		// Un autre ecrivain (la CLI, un autre cycle) l'a ajoutee entre-temps : l'ajout seul a
		// fait son travail.
		return ZonesDejaCouvertes, nil
	case err != nil:
		return ZonesEchecEcriture, err
	}
	if r.sources.Genere != nil {
		r.sources.Genere.MapsByID[mapID] = entry
	}
	return ZonesAjoutees, nil
}

// lireMvarEnCache lit la variante d'une carte deposee au cache. `os.ErrNotExist` quand aucune
// variante n'y est. Le fichier retenu suit la regle unique du choix de variante.
func lireMvarEnCache(cacheRoot, mapID string) ([]byte, error) {
	dir := dossierMvar(cacheRoot, mapID)
	entrees, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var noms []string
	for _, e := range entrees {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".mvar") {
			noms = append(noms, e.Name())
		}
	}
	if len(noms) == 0 {
		return nil, fmt.Errorf("aucune variante sous %s : %w", dir, os.ErrNotExist)
	}
	return os.ReadFile(filepath.Join(dir, mapcatalog.ChoisirFichierVariante(noms, mapcatalog.NomDeLaVariante)))
}
