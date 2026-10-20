package main

// cmd_backfill_map_callouts.go — sous-commande `levelup backfill-map-callouts`.
//
// LE PREMIER PASSAGE SUR L'HISTORIQUE du rattrapage des zones nommées. Le fil de l'eau
// (`sync/replayartifacts/zones_rattrapage.go`) donne ses zones à chaque carte d'un film qu'il
// cuit ; les cartes déjà jouées passent par cette commande. C'est la MÊME chaîne, carte par
// carte (`replayartifacts.RattrapageZones.TraiterCarte`) : cascade des zones, variante au cache
// sinon téléchargée une fois et déposée au cache, extraction, lexique, ajout au catalogue
// GÉNÉRÉ (`reference/generated/map_callouts.json`). Le catalogue versionné n'est jamais écrit.
//
// # IDEMPOTENTE
//
// Une carte déjà couverte (catalogue versionné, catalogue généré, module d'une carte intégrée)
// est sautée sans appel ; une variante déjà au cache se relit hors ligne ; l'ajout est en ajout
// seul. Relancer la commande ne télécharge que ce qui manque encore et n'écrit que ce qui est
// neuf.
//
// # LE RÉSEAU N'EST OUVERT QUE S'IL LE FAUT
//
// Un premier tour traite toutes les cartes HORS LIGNE ; les jetons ne sont demandés (pool de la
// CLI, endpoint public) que s'il reste des variantes à télécharger, et jamais avec --dry-run ni
// --hors-ligne.
//
// # AUCUNE RECUISSON
//
// Les zones se résolvent au service, à la lecture : une carte ajoutée s'affiche au prochain
// rejeu ouvert, sur tous ses matchs.
//
// Usage (SERVEUR ARRÊTÉ : la liste des cartes se lit dans la base partagée, que DuckDB ne laisse
// ouvrir qu'à un processus, et le pool de jetons ne doit pas tourner en même temps que celui du
// serveur) :
//
//	levelup backfill-map-callouts --dry-run          # ce qui serait fait, aucun octet écrit ni téléchargé
//	levelup backfill-map-callouts --hors-ligne       # seulement les variantes déjà au cache
//	levelup backfill-map-callouts                    # tout l'historique JcJ
//	levelup backfill-map-callouts --carte ID[,ID...] # ces cartes seulement, sans ouvrir la base

import (
	"context"
	"flag"
	"fmt"
	"sort"
	"strings"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/mapcatalog"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/sync/replayartifacts"
)

// rpsVariantesParDefaut : débit des requêtes Halo de la passe (une requête d'asset et une de
// fichier par carte).
const rpsVariantesParDefaut = 4

// mapCalloutsOptions : les réglages de la passe.
type mapCalloutsOptions struct {
	titleSlug string
	dryRun    bool
	horsLigne bool
	cartes    string
	rps       int
	cacheDir  string
}

func runBackfillMapCallouts(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("backfill-map-callouts", flag.ExitOnError)
	o := mapCalloutsOptions{}
	fs.StringVar(&o.titleSlug, "title", titlePkg.DefaultSlug, "slug du titre")
	fs.BoolVar(&o.dryRun, "dry-run", false, "dire ce qui serait fait : aucun téléchargement, aucune écriture")
	fs.BoolVar(&o.horsLigne, "hors-ligne", false, "ne lire que les variantes déjà au cache (aucun jeton, aucun appel)")
	fs.StringVar(&o.cartes, "carte", "", "map_id à traiter, séparés par des virgules (la base n'est pas ouverte)")
	fs.IntVar(&o.rps, "rps", rpsVariantesParDefaut, "débit maximal des requêtes Halo")
	fs.StringVar(&o.cacheDir, "cache-dir", "", "racine du cache (défaut : data/cache du dépôt)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	ctx := context.Background()
	// LA PORTE EST UNE CAPABILITY, JAMAIS UN SLUG (ratchet no_slug_comparison_test.go).
	caps, err := games.LoadCapabilityMap(cfg.RepoRoot, o.titleSlug)
	if err != nil {
		return err
	}
	if !caps.Has(games.CapMapForgeCallouts) {
		fmt.Printf("titre %s : capability %s absente — aucune zone à rattraper\n",
			o.titleSlug, games.CapMapForgeCallouts)
		return nil
	}
	cartes, err := cartesDeLaPasse(ctx, cfg, o)
	if err != nil {
		return err
	}
	b, err := passeMapCallouts(ctx, cfg, o, cartes)
	if err != nil {
		return err
	}
	b.imprimer(o)
	if n := b.echecs(); n > 0 {
		return fmt.Errorf("%d carte(s) en échec — relancer la commande reprend là où elle s'est arrêtée", n)
	}
	return nil
}

// cartesDeLaPasse rend les cartes à examiner : celles de --carte (sans base), sinon les cartes
// JcJ du registre, lues EN LECTURE SEULE.
func cartesDeLaPasse(ctx context.Context, cfg *config.AppConfig, o mapCalloutsOptions) ([]replayartifacts.CarteJouee, error) {
	if strings.TrimSpace(o.cartes) != "" {
		var out []replayartifacts.CarteJouee
		for _, id := range strings.Split(o.cartes, ",") {
			if id = strings.TrimSpace(id); id != "" {
				out = append(out, replayartifacts.CarteJouee{MapID: id})
			}
		}
		return out, nil
	}
	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	sharedDB, releaseShared, err := duckdb.OpenReadForQuery(pr.SharedDBPath(o.titleSlug))
	if err != nil {
		return nil, fmt.Errorf("ouverture en lecture de la base partagée (serveur arrêté ?) : %w", err)
	}
	defer releaseShared()
	metaDB, releaseMeta, err := duckdb.OpenReadForQuery(pr.MetadataDBPath(o.titleSlug))
	if err != nil {
		return nil, fmt.Errorf("ouverture en lecture de la base metadata (serveur arrêté ?) : %w", err)
	}
	defer releaseMeta()
	return replayartifacts.CartesJoueesJcJ(ctx, sharedDB, metaDB)
}

// passeMapCallouts traite les cartes : un tour hors ligne, puis le réseau pour les seules
// variantes qui manquent au cache (cf. l'en-tête).
func passeMapCallouts(ctx context.Context, cfg *config.AppConfig, o mapCalloutsOptions,
	cartes []replayartifacts.CarteJouee) (*bilanMapCallouts, error) {
	opts := replayartifacts.OptionsRattrapageZones{
		RepoRoot: cfg.RepoRoot, TitleSlug: o.titleSlug, CacheRoot: racineDesVariantes(cfg, o), ABlanc: o.dryRun,
	}
	horsLigne, err := replayartifacts.PreparerRattrapageZones(ctx, opts)
	if err != nil {
		return nil, err
	}
	b := &bilanMapCallouts{parStatut: map[replayartifacts.StatutZones]int{}, sansLibelle: map[uint32]bool{}}
	var aTelecharger []replayartifacts.CarteJouee
	for _, c := range cartes {
		r := horsLigne.TraiterCarte(ctx, identitesDe(c))
		if r.Statut == replayartifacts.ZonesATelecharger && !o.dryRun && !o.horsLigne {
			aTelecharger = append(aTelecharger, c)
			continue
		}
		b.noter(c, r)
	}
	if len(aTelecharger) == 0 {
		return b, nil
	}
	fmt.Printf("%d variante(s) à télécharger (pool de jetons, %d requêtes/s)\n", len(aTelecharger), o.rps)
	client, fermer, err := newPooledClient(ctx, cfg, o.rps)
	if err != nil {
		return nil, fmt.Errorf("pool de jetons : %w", err)
	}
	defer fermer()
	opts.Fetcher = client
	enLigne, err := replayartifacts.PreparerRattrapageZones(ctx, opts)
	if err != nil {
		return nil, err
	}
	for _, c := range aTelecharger {
		b.noter(c, enLigne.TraiterCarte(ctx, identitesDe(c)))
	}
	return b, nil
}

// racineDesVariantes : la racine sous laquelle le serveur dépose les variantes (`<racine>/mvar/`).
func racineDesVariantes(cfg *config.AppConfig, o mapCalloutsOptions) string {
	if v := strings.TrimSpace(o.cacheDir); v != "" {
		return v
	}
	return titlePkg.NewPathResolver(cfg.RepoRoot).CacheRootDir()
}

func identitesDe(c replayartifacts.CarteJouee) mapcatalog.IdentitesDeCarte {
	return mapcatalog.IdentitesDeCarte{MapID: c.MapID, Noms: c.Noms}
}

// bilanMapCallouts : ce que la passe a fait, carte par carte et au total.
type bilanMapCallouts struct {
	parStatut   map[replayartifacts.StatutZones]int
	sansLibelle map[uint32]bool
	zones       int
	nommees     int
}

// noter imprime la ligne d'une carte et la verse au bilan.
func (b *bilanMapCallouts) noter(c replayartifacts.CarteJouee, r replayartifacts.ResultatZones) {
	b.parStatut[r.Statut]++
	for sid, resolu := range r.Couverture.StringIDs {
		if !resolu {
			b.sansLibelle[sid] = true
		}
	}
	if r.Statut == replayartifacts.ZonesAjoutees || r.Statut == replayartifacts.ZonesAjoutables {
		b.zones += r.Couverture.Zones
		b.nommees += r.Couverture.Nommees
	}
	fmt.Println(ligneDeCarte(c, r))
}

// ligneDeCarte : une ligne par carte — statut, identité, mesure des libellés, cause d'échec.
func ligneDeCarte(c replayartifacts.CarteJouee, r replayartifacts.ResultatZones) string {
	nom := "-"
	if len(c.Noms) > 0 {
		nom = c.Noms[0]
	}
	ligne := fmt.Sprintf("%-20s %s %q matchs=%d", r.Statut, c.MapID, nom, c.Matchs)
	if r.Origine != "" {
		ligne += " par=" + string(r.Origine)
	}
	if r.Couverture.Zones > 0 {
		ligne += fmt.Sprintf(" zones=%d nommees=%d string_id_sans_libelle=%d",
			r.Couverture.Zones, r.Couverture.Nommees, r.Couverture.SansLibelle())
	}
	if r.Telechargee {
		ligne += " (telechargee)"
	}
	if r.Err != nil {
		ligne += " erreur=" + r.Err.Error()
	}
	return ligne
}

// echecs compte les cartes en échec.
func (b *bilanMapCallouts) echecs() int {
	return b.parStatut[replayartifacts.ZonesEchecTelechargement] +
		b.parStatut[replayartifacts.ZonesVarianteIllisible] + b.parStatut[replayartifacts.ZonesEchecEcriture]
}

// imprimer rend le bilan chiffré de la passe, et les string_id qui manquent au lexique.
func (b *bilanMapCallouts) imprimer(o mapCalloutsOptions) {
	statuts := make([]string, 0, len(b.parStatut))
	total := 0
	for s, n := range b.parStatut {
		statuts = append(statuts, fmt.Sprintf("%s=%d", s, n))
		total += n
	}
	sort.Strings(statuts)
	mode := "ecriture"
	switch {
	case o.dryRun:
		mode = "a blanc"
	case o.horsLigne:
		mode = "hors ligne"
	}
	fmt.Printf("bilan (%s) : %d carte(s) — %s\n", mode, total, strings.Join(statuts, " "))
	fmt.Printf("zones des cartes ajoutees%s : %d dont %d nommees\n", siABlanc(o), b.zones, b.nommees)
	if len(b.sansLibelle) > 0 {
		ids := make([]string, 0, len(b.sansLibelle))
		for sid := range b.sansLibelle {
			ids = append(ids, fmt.Sprintf("0x%08X", sid))
		}
		sort.Strings(ids)
		fmt.Printf("%d string_id de lieu sans libelle au lexique (zones affichees sans nom ; "+
			"mapcallouts-build --lexique les ajoute s'ils sont au jeu installe) : %s\n",
			len(ids), strings.Join(ids, " "))
	}
}

// siABlanc précise, à blanc, que les cartes « ajoutées » ne sont qu'ajoutables.
func siABlanc(o mapCalloutsOptions) string {
	if o.dryRun {
		return " (ou ajoutables)"
	}
	return ""
}
