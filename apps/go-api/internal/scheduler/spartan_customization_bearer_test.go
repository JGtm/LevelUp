package scheduler_test

// spartan_customization_bearer_test.go — le cron de personnalisation Spartan lit
// l'apparence d'un profil suivi dont les jetons sont morts avec le token du compte de
// l'utilisateur de l'instance (rôle admin), jamais avec celui d'un autre utilisateur,
// et l'enregistre dans la base de CE profil.
//
// Chaîne réelle de bout en bout, réseau excepté : pool de tokens réel (pool.NewPool,
// résolveur factice qui refuse le refresh token du joueur aux jetons morts), service
// d'identité Spartan réel (service.CareerLiveService, cache réel : c'est lui qui
// déclenche la lecture d'arrière-plan et l'écriture), API Halo et player DB
// remplacées par des enregistreurs.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/auth/pool"
	"levelup/go-api/internal/scheduler"
	"levelup/go-api/internal/service"
)

const (
	xuidChoco  = "2535469190789936"
	xuidJGtm   = "2533274823110022"
	xuidDanker = "2535405528935279"

	emblemChoco = "https://gamecms-hacs.svc.halowaypoint.com/hi/images/file/progression/Inventory/Emblems/nouvel_embleme.png"
	emblemJGtm  = "https://gamecms-hacs.svc.halowaypoint.com/hi/images/file/progression/Inventory/Emblems/343other_propaganda_emblem.png"

	// writeWait borne l'attente de l'écriture d'arrière-plan du service d'identité.
	writeWait = 10 * time.Second
)

// bearerResolver échange un refresh token factice ; ceux de dead sont refusés,
// comme un refresh token révoqué chez Microsoft.
type bearerResolver struct{ dead map[string]bool }

func (r bearerResolver) Resolve(_ context.Context, src pool.CredentialSource) (*pool.ResolvedTokens, error) {
	if r.dead[src.Gamertag] {
		return nil, errors.New("invalid_grant : refresh token refusé")
	}
	return &pool.ResolvedTokens{
		Gamertag:  src.Gamertag,
		XUID:      src.XUID,
		Tokens:    &domain.HaloTokens{SpartanToken: "spartan-" + src.Gamertag},
		ExpiresAt: time.Now().Add(4 * time.Hour),
		Source:    "test",
	}, nil
}

func (r bearerResolver) Refresh(_ context.Context, gamertag string) (*pool.ResolvedTokens, error) {
	return nil, fmt.Errorf("refresh indisponible en test (%s)", gamertag)
}

// newBearerPool construit un pool réel sur les comptes donnés (gamertag → xuid).
func newBearerPool(t *testing.T, accounts map[string]string, dead ...string) pool.Pool {
	t.Helper()
	deadSet := make(map[string]bool, len(dead))
	for _, gt := range dead {
		deadSet[gt] = true
	}
	sources := make([]pool.CredentialSource, 0, len(accounts))
	for gt, xuid := range accounts {
		sources = append(sources, pool.CredentialSource{
			Gamertag: gt, TitleSlug: "halo_infinite", XUID: xuid, RefreshToken: "rt-" + gt, Source: "test",
		})
	}
	p, err := pool.NewPool(context.Background(), bearerResolver{dead: deadSet}, sources, pool.PoolOptions{})
	if err != nil {
		t.Fatalf("pool.NewPool : %v", err)
	}
	t.Cleanup(p.Close)
	return p
}

// writeProfilesJSON écrit db_profiles.json et rend la config qui le lit.
func writeProfilesJSON(t *testing.T, body string) *config.AppConfig {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "db_profiles.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("écriture de db_profiles.json : %v", err)
	}
	return &config.AppConfig{RepoRoot: root, DBProfilesPath: path}
}

// liveCall est un appel à l'API Halo : token utilisé, porteur imputé, joueur ciblé.
type liveCall struct{ token, owner, subject string }

// liveRecorder enregistre les appels à l'API Halo faits par le service d'identité.
type liveRecorder struct {
	mu      sync.Mutex
	calls   []liveCall
	emblems map[string]string // xuid ciblé → emblème rendu par l'API
}

func (r *liveRecorder) add(c liveCall) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, c)
}

func (r *liveRecorder) callsFor(subject string) []liveCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []liveCall
	for _, c := range r.calls {
		if c.subject == subject {
			out = append(out, c)
		}
	}
	return out
}

func (r *liveRecorder) total() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

// factory remplace service.CareerFetcherFactoryFromTokens : même lecture du
// contexte (tokens, porteur), réponses de l'API simulées.
func (r *liveRecorder) factory() service.CareerFetcherFactory {
	return func(ctx context.Context) service.CareerFetcher {
		tokens := ctxkeys.HaloTokens(ctx)
		if tokens == nil || tokens.SpartanToken == "" {
			return nil
		}
		return &recordingFetcher{rec: r, token: tokens.SpartanToken, owner: ctxkeys.TokensOwnerXUID(ctx)}
	}
}

type recordingFetcher struct {
	rec          *liveRecorder
	token, owner string
}

func (f *recordingFetcher) GetCareerProgress(_ context.Context, xuid string) (*domain.CareerRankSnapshot, error) {
	f.rec.add(liveCall{token: f.token, owner: f.owner, subject: xuid})
	return &domain.CareerRankSnapshot{XUID: xuid, CurrentRank: 150}, nil
}

func (f *recordingFetcher) GetSpartanCustomization(_ context.Context, xuid string) (*domain.SpartanCustomizationData, error) {
	f.rec.add(liveCall{token: f.token, owner: f.owner, subject: xuid})
	return &domain.SpartanCustomizationData{EmblemImageURL: f.rec.emblems[xuid]}, nil
}

// writtenRow est une ligne career_progression écrite dans la base d'un joueur.
type writtenRow struct {
	xuid    string
	partial domain.CareerProgressionPartial
}

// playerCareerRepo tient lieu de la player DB d'UN joueur.
type playerCareerRepo struct{ written chan writtenRow }

func newPlayerCareerRepo() *playerCareerRepo {
	return &playerCareerRepo{written: make(chan writtenRow, 8)}
}

func (r *playerCareerRepo) LoadLastCareerRank(context.Context, string) (*domain.CareerRankRow, error) {
	return nil, nil
}

func (r *playerCareerRepo) EnrichFromMetadata(context.Context, *domain.CareerRankRow) error {
	return nil
}

func (r *playerCareerRepo) InsertCareerProgressionIfChanged(context.Context, string, *domain.CareerRankRow) (bool, error) {
	return false, errors.New("chemin d'écriture complet non attendu : le service écrit par partial")
}

func (r *playerCareerRepo) InsertCareerProgressionPartial(_ context.Context, xuid string, p *domain.CareerProgressionPartial) (bool, error) {
	r.written <- writtenRow{xuid: xuid, partial: *p}
	return true, nil
}

func (r *playerCareerRepo) wait(t *testing.T, who string) writtenRow {
	t.Helper()
	select {
	case row := <-r.written:
		return row
	case <-time.After(writeWait):
		t.Fatalf("aucune ligne écrite dans la base de %s après %v", who, writeWait)
		return writtenRow{}
	}
}

type nopIdentityBuilder struct{}

func (nopIdentityBuilder) BuildSpartanIdentityFromCareerRow(context.Context, *domain.CareerRankRow, bool) *domain.HomeSpartanIdentityRow {
	return &domain.HomeSpartanIdentityRow{}
}

// careerProvider rend, par profil, le service d'identité réel branché sur la base
// de CE profil (comme ServiceRegistry.CareerLiveCtx), avec un cache commun.
func careerProvider(repos map[string]*playerCareerRepo, rec *liveRecorder, calls *callCounter) scheduler.CareerLiveServiceProvider {
	cache := service.NewCareerLiveCache(service.CareerLiveCacheConfig{})
	return func(_ context.Context, slug string) (scheduler.SpartanIdentityFetcher, error) {
		calls.inc()
		repo, ok := repos[slug]
		if !ok {
			return nil, fmt.Errorf("profil inattendu %q", slug)
		}
		return service.NewCareerLiveService(repo, nopIdentityBuilder{}, rec.factory(), cache), nil
	}
}

type callCounter struct {
	mu sync.Mutex
	n  int
}

func (c *callCounter) inc() { c.mu.Lock(); c.n++; c.mu.Unlock() }
func (c *callCounter) get() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

// spartanCronError rend la dernière erreur rapportée par le cron ("" = succès).
func spartanCronError(t *testing.T) string {
	t.Helper()
	for _, r := range observability.CronStatusSnapshot() {
		if r.Name == "spartan_customization" {
			return r.LastError
		}
	}
	t.Fatal("le cron n'a rapporté aucune exécution")
	return ""
}

// instanceAccounts tient lieu du store des comptes (*userstore.Store) : JGtm est le
// compte de rôle admin, DankerGlue un autre utilisateur de l'instance.
type instanceAccounts struct {
	users []domain.AdminUserSummary
	err   error
}

func (a instanceAccounts) List() ([]domain.AdminUserSummary, error) { return a.users, a.err }

func jgtmAdminAccounts() instanceAccounts {
	return instanceAccounts{users: []domain.AdminUserSummary{
		{Username: "DankerGlue", Role: domain.RoleUser, Gamertag: "DankerGlue", XUID: xuidDanker},
		{Username: "JGtm", Role: domain.RoleAdmin, Gamertag: "JGtm", XUID: xuidJGtm},
	}}
}

// profilesChocoJGtmDanker déclare Chocoboflor et JGtm suivis, DankerGlue auth_only.
// Le champ « admin » de db_profiles.json nomme DankerGlue : le choix du porteur ne
// le lit pas, seul le rôle des comptes de l'instance compte.
const profilesChocoJGtmDanker = `{"version":"3.0","admin":"DankerGlue","profiles":{"halo_infinite":{` +
	`"Chocoboflor":{"db_path":"unused","xuid":"` + xuidChoco + `"},` +
	`"JGtm":{"db_path":"unused","xuid":"` + xuidJGtm + `"},` +
	`"DankerGlue":{"db_path":"unused","xuid":"` + xuidDanker + `","auth_only":true}}}}`

// TestSpartanCron_JetonsMorts_PorteurValide_LigneEnregistree : Chocoboflor (profil
// suivi, refresh token refusé : aucun créneau dans le pool) est lu avec le token de
// JGtm, lié au compte de rôle admin de l'instance, jamais avec celui de DankerGlue,
// autre utilisateur au token valide (que nomme pourtant le champ « admin » de
// db_profiles.json). La ligne lue (emblème, statut « ok ») va dans la base de
// Chocoboflor, sous son xuid. JGtm, aux jetons valides, reste lu avec son propre
// token et écrit dans SA base.
func TestSpartanCron_JetonsMorts_PorteurValide_LigneEnregistree(t *testing.T) {
	observability.ResetCronStatus()
	t.Cleanup(observability.ResetCronStatus)
	cfg := writeProfilesJSON(t, profilesChocoJGtmDanker)
	tokens := newBearerPool(t, map[string]string{
		"Chocoboflor": xuidChoco, "JGtm": xuidJGtm, "DankerGlue": xuidDanker,
	}, "Chocoboflor")
	rec := &liveRecorder{emblems: map[string]string{xuidChoco: emblemChoco, xuidJGtm: emblemJGtm}}
	repos := map[string]*playerCareerRepo{"Chocoboflor": newPlayerCareerRepo(), "JGtm": newPlayerCareerRepo()}
	var providerCalls callCounter
	bearerReadsBefore := observability.LoadCounter("spartan_cron_bearer_reads_total")

	scheduler.NewSpartanCustomizationCron(cfg, tokens, careerProvider(repos, rec, &providerCalls), "halo_infinite", 0).
		WithAccounts(jgtmAdminAccounts()).
		RunOnce(context.Background())

	choco := repos["Chocoboflor"].wait(t, "Chocoboflor")
	if choco.xuid != xuidChoco {
		t.Errorf("ligne de Chocoboflor écrite sous le xuid %q, attendu %q", choco.xuid, xuidChoco)
	}
	if got := deref(choco.partial.EmblemImageURL); got != emblemChoco {
		t.Errorf("emblème enregistré %q, attendu %q", got, emblemChoco)
	}
	if got := deref(choco.partial.LastFetchStatus); got != "ok" {
		t.Errorf("statut enregistré %q, attendu \"ok\"", got)
	}
	jgtm := repos["JGtm"].wait(t, "JGtm")
	if jgtm.xuid != xuidJGtm || deref(jgtm.partial.EmblemImageURL) != emblemJGtm {
		t.Errorf("ligne de JGtm : xuid=%q emblème=%q", jgtm.xuid, deref(jgtm.partial.EmblemImageURL))
	}

	assertCalls(t, rec.callsFor(xuidChoco), "Chocoboflor", liveCall{token: "spartan-JGtm", owner: xuidJGtm})
	assertCalls(t, rec.callsFor(xuidJGtm), "JGtm", liveCall{token: "spartan-JGtm", owner: xuidJGtm})
	if got := observability.LoadCounter("spartan_cron_bearer_reads_total") - bearerReadsBefore; got != 1 {
		t.Errorf("lectures portées comptées : %d, attendu 1 (Chocoboflor seul)", got)
	}
	if msg := spartanCronError(t); msg != "" {
		t.Errorf("cycle en échec alors que les deux joueurs ont été lus : %s", msg)
	}
}

// TestSpartanCron_CompteAdminInutilisable_AucunPorteur : le token du compte admin
// (JGtm) est malsain. DankerGlue, autre utilisateur au token valide, ne porte jamais
// la lecture : Chocoboflor et JGtm ne sont ni lus ni écrits, chaque absence de token
// est comptée et le cycle rapporte l'échec.
func TestSpartanCron_CompteAdminInutilisable_AucunPorteur(t *testing.T) {
	observability.ResetCronStatus()
	t.Cleanup(observability.ResetCronStatus)
	cfg := writeProfilesJSON(t, profilesChocoJGtmDanker)
	tokens := newBearerPool(t, map[string]string{
		"Chocoboflor": xuidChoco, "JGtm": xuidJGtm, "DankerGlue": xuidDanker,
	}, "Chocoboflor")
	tokens.MarkUnhealthy("JGtm", errors.New("401 en test"))
	rec := &liveRecorder{}
	repos := map[string]*playerCareerRepo{"Chocoboflor": newPlayerCareerRepo(), "JGtm": newPlayerCareerRepo()}
	var providerCalls callCounter
	noBearerBefore := observability.LoadCounter("spartan_cron_no_bearer_total")

	scheduler.NewSpartanCustomizationCron(cfg, tokens, careerProvider(repos, rec, &providerCalls), "halo_infinite", 0).
		WithAccounts(jgtmAdminAccounts()).
		RunOnce(context.Background())

	assertNothingRead(t, rec, &providerCalls, repos)
	if got := observability.LoadCounter("spartan_cron_no_bearer_total") - noBearerBefore; got != 2 {
		t.Errorf("absences de token comptées : %d, attendu 2 (Chocoboflor et JGtm)", got)
	}
	if msg := spartanCronError(t); !strings.Contains(msg, "aucun token utilisable") {
		t.Errorf("le cycle doit rapporter l'absence de token, erreur rapportée : %q", msg)
	}
}

// TestSpartanCron_ComptesIllisibles_AucunAutreCompte : sans lecture des comptes de
// l'instance, le compte de l'utilisateur n'est pas reconnu : aucun porteur n'est
// pris. Chocoboflor n'est ni lu ni écrit, l'absence est comptée et l'échec rapporté ;
// JGtm reste lu avec son propre token.
func TestSpartanCron_ComptesIllisibles_AucunAutreCompte(t *testing.T) {
	observability.ResetCronStatus()
	t.Cleanup(observability.ResetCronStatus)
	cfg := writeProfilesJSON(t, profilesChocoJGtmDanker)
	tokens := newBearerPool(t, map[string]string{
		"Chocoboflor": xuidChoco, "JGtm": xuidJGtm, "DankerGlue": xuidDanker,
	}, "Chocoboflor")
	rec := &liveRecorder{emblems: map[string]string{xuidJGtm: emblemJGtm}}
	repos := map[string]*playerCareerRepo{"Chocoboflor": newPlayerCareerRepo(), "JGtm": newPlayerCareerRepo()}
	var providerCalls callCounter
	noBearerBefore := observability.LoadCounter("spartan_cron_no_bearer_total")

	scheduler.NewSpartanCustomizationCron(cfg, tokens, careerProvider(repos, rec, &providerCalls), "halo_infinite", 0).
		WithAccounts(instanceAccounts{err: errors.New("users.json illisible en test")}).
		RunOnce(context.Background())

	repos["JGtm"].wait(t, "JGtm")
	assertCalls(t, rec.callsFor(xuidJGtm), "JGtm", liveCall{token: "spartan-JGtm", owner: xuidJGtm})
	if calls := rec.callsFor(xuidChoco); len(calls) != 0 || providerCalls.get() != 1 {
		t.Errorf("Chocoboflor lu sans compte admin reconnu : %d appels API, %d services", len(calls), providerCalls.get())
	}
	if got := observability.LoadCounter("spartan_cron_no_bearer_total") - noBearerBefore; got != 1 {
		t.Errorf("absences de token comptées : %d, attendu 1 (Chocoboflor)", got)
	}
	if msg := spartanCronError(t); !strings.Contains(msg, "aucun token utilisable") {
		t.Errorf("le cycle doit rapporter l'absence de token pour Chocoboflor, erreur rapportée : %q", msg)
	}
}

// TestSpartanCron_AucunPorteurValide_RienNEstEcrit : le compte admin (JGtm) n'a pas
// de profil déclaré, donc pas de créneau ; DankerGlue, au token valide, n'est pas le
// compte de l'utilisateur. Chocoboflor n'est ni lu ni écrit.
func TestSpartanCron_AucunPorteurValide_RienNEstEcrit(t *testing.T) {
	observability.ResetCronStatus()
	t.Cleanup(observability.ResetCronStatus)
	cfg := writeProfilesJSON(t, `{"version":"3.0","profiles":{"halo_infinite":{`+
		`"Chocoboflor":{"db_path":"unused","xuid":"`+xuidChoco+`"},`+
		`"DankerGlue":{"db_path":"unused","xuid":"`+xuidDanker+`","auth_only":true}}}}`)
	tokens := newBearerPool(t, map[string]string{"Chocoboflor": xuidChoco, "DankerGlue": xuidDanker}, "Chocoboflor")
	rec := &liveRecorder{}
	repos := map[string]*playerCareerRepo{"Chocoboflor": newPlayerCareerRepo()}
	var providerCalls callCounter
	noBearerBefore := observability.LoadCounter("spartan_cron_no_bearer_total")

	scheduler.NewSpartanCustomizationCron(cfg, tokens, careerProvider(repos, rec, &providerCalls), "halo_infinite", 0).
		WithAccounts(jgtmAdminAccounts()).
		RunOnce(context.Background())

	assertNothingRead(t, rec, &providerCalls, repos)
	if got := observability.LoadCounter("spartan_cron_no_bearer_total") - noBearerBefore; got != 1 {
		t.Errorf("absences de token comptées : %d, attendu 1", got)
	}
	if msg := spartanCronError(t); !strings.Contains(msg, "aucun token utilisable") {
		t.Errorf("le cycle doit rapporter l'absence de token, erreur rapportée : %q", msg)
	}
}

// TestSpartanCron_ProfilNonSuivi_RienNEstLu : un profil en pause et un compte
// auth_only ne sont pas des profils suivis (domain.SyncablePlayers, la définition de
// domain.ProfileGate) : même avec le token du compte admin valide dans le parc, aucun
// n'est lu ni écrit. Un xuid absent de db_profiles.json n'est jamais énuméré par le cron.
func TestSpartanCron_ProfilNonSuivi_RienNEstLu(t *testing.T) {
	observability.ResetCronStatus()
	t.Cleanup(observability.ResetCronStatus)
	cfg := writeProfilesJSON(t, `{"version":"3.0","profiles":{"halo_infinite":{`+
		`"Chocoboflor":{"db_path":"unused","xuid":"`+xuidChoco+`","sync_enabled":false},`+
		`"JGtm":{"db_path":"unused","xuid":"`+xuidJGtm+`","auth_only":true}}}}`)
	tokens := newBearerPool(t, map[string]string{"Chocoboflor": xuidChoco, "JGtm": xuidJGtm}, "Chocoboflor")
	rec := &liveRecorder{}
	repos := map[string]*playerCareerRepo{"Chocoboflor": newPlayerCareerRepo(), "JGtm": newPlayerCareerRepo()}
	var providerCalls callCounter

	scheduler.NewSpartanCustomizationCron(cfg, tokens, careerProvider(repos, rec, &providerCalls), "halo_infinite", 0).
		WithAccounts(jgtmAdminAccounts()).
		RunOnce(context.Background())

	assertNothingRead(t, rec, &providerCalls, repos)
	if msg := spartanCronError(t); msg != "" {
		t.Errorf("aucun profil suivi n'est pas un échec, erreur rapportée : %q", msg)
	}
}

// assertNothingRead vérifie qu'aucun service d'identité n'a été construit, qu'aucun
// appel API n'a eu lieu et qu'aucune ligne n'a été écrite. Aucune lecture n'ayant été
// lancée, aucune écriture d'arrière-plan ne peut arriver plus tard.
func assertNothingRead(t *testing.T, rec *liveRecorder, providerCalls *callCounter, repos map[string]*playerCareerRepo) {
	t.Helper()
	if providerCalls.get() != 0 || rec.total() != 0 {
		t.Errorf("lecture tentée : %d services, %d appels API", providerCalls.get(), rec.total())
	}
	for who, repo := range repos {
		select {
		case row := <-repo.written:
			t.Errorf("ligne écrite dans la base de %s : %+v", who, row)
		default:
		}
	}
}

// assertCalls vérifie que chaque appel API fait pour un joueur a utilisé le token
// et imputé le porteur attendus.
func assertCalls(t *testing.T, calls []liveCall, who string, want liveCall) {
	t.Helper()
	if len(calls) == 0 {
		t.Fatalf("aucun appel API pour %s", who)
	}
	for _, c := range calls {
		if c.token != want.token || c.owner != want.owner {
			t.Errorf("appel pour %s avec token %q imputé à %q, attendu token %q imputé à %q",
				who, c.token, c.owner, want.token, want.owner)
		}
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
