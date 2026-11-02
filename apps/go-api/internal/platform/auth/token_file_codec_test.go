// Package auth — token_file_codec_test.go : forme sur disque des fichiers de jetons et relation au
// chiffreur du magasin (ADR 0045 D-11). Ces tests tournent partout : le chiffreur scellant y est un
// double qui porte le nom `dpapi` ; le coffre Windows réel est testé par token_file_vault_test.go.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"expvar"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/platform/auth/tokencipher"
)

// reverseCipher scelle en retournant et masquant les octets : lisible par lui seul, jamais égal au clair.
type reverseCipher struct{ failOpen bool }

func (reverseCipher) Name() string { return tokencipher.NameDPAPI }

func reversed(b []byte) []byte {
	out := make([]byte, len(b))
	for i, c := range b {
		out[len(b)-1-i] = c ^ 0x5a
	}
	return out
}

func (reverseCipher) Seal(b []byte) ([]byte, error) { return reversed(b), nil }

func (c reverseCipher) Open(b []byte) ([]byte, error) {
	if c.failOpen {
		return nil, errors.New("autre compte")
	}
	return reversed(b), nil
}

// Valeurs de forme (pas des secrets) : des lettres répétées, pour que le balayage de secrets ne les
// prenne pas pour des clés ; seul leur scellé compte.
var (
	secretRT     = strings.Repeat("r", 20)
	secretAccess = strings.Repeat("a", 20)
	secretXSTS   = strings.Repeat("x", 20)
	secretHash   = strings.Repeat("h", 20)
)

func seedTokens(xuid string) *UserTokens {
	return &UserTokens{
		XUID: xuid, Gamertag: "Alice", OAuthRefreshToken: secretRT, AccessToken: secretAccess,
		XSTSToken: secretXSTS, XSTSUserHash: secretHash, TokenClientFamily: TokenFamilyAzure,
		PoolSharing: &PoolSharing{Granted: true, Basis: PoolSharingBasisUserChoice},
	}
}

func readFileString(t *testing.T, dir, xuid string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, xuid+".json"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func unreadableCount(reason string) int64 {
	v, _ := tokenFileUnreadable.Get(reason).(*expvar.Int)
	if v == nil {
		return 0
	}
	return v.Value()
}

// TestCodec_PlainAllerRetourSansChampCipher : un magasin plain écrit le format d'avant (aucun champ
// `cipher`, jetons en clair) et le relit.
func TestCodec_PlainAllerRetourSansChampCipher(t *testing.T) {
	dir := tempTokenDir(t)
	s := NewMultiUserTokenStoreWithCipher(dir, tokencipher.Plain())
	if err := s.Upsert(seedTokens("111")); err != nil {
		t.Fatal(err)
	}
	raw := readFileString(t, dir, "111")
	if strings.Contains(raw, `"cipher"`) {
		t.Errorf("un fichier plain ne porte pas de champ cipher :\n%s", raw)
	}
	if !strings.Contains(raw, secretRT) || !strings.Contains(raw, secretXSTS) {
		t.Errorf("un fichier plain porte les jetons en clair :\n%s", raw)
	}
	got, err := s.Load("111")
	if err != nil || got.OAuthRefreshToken != secretRT || got.XSTSUserHash != secretHash {
		t.Fatalf("aller-retour plain : %+v, %v", got, err)
	}
}

// TestCodec_ScelleSeulementLesChampsSensibles : sous un chiffreur scellant, les quatre jetons sont
// remplacés par un scellé base64, le reste reste lisible, et l'aller-retour restitue tout.
func TestCodec_ScelleSeulementLesChampsSensibles(t *testing.T) {
	dir := tempTokenDir(t)
	s := NewMultiUserTokenStoreWithCipher(dir, reverseCipher{})
	if err := s.Upsert(seedTokens("111")); err != nil {
		t.Fatal(err)
	}
	raw := readFileString(t, dir, "111")
	for _, secret := range []string{secretRT, secretAccess, secretXSTS, secretHash} {
		if strings.Contains(raw, secret) {
			t.Errorf("le secret %q est en clair dans le fichier :\n%s", secret, raw)
		}
	}
	for _, clair := range []string{`"cipher": "dpapi"`, `"xuid": "111"`, `"gamertag": "Alice"`, `"granted": true`} {
		if !strings.Contains(raw, clair) {
			t.Errorf("%s doit rester en clair :\n%s", clair, raw)
		}
	}
	got, err := s.Load("111")
	if err != nil {
		t.Fatal(err)
	}
	if got.OAuthRefreshToken != secretRT || got.AccessToken != secretAccess ||
		got.XSTSToken != secretXSTS || got.XSTSUserHash != secretHash || !got.SharesWithPool() {
		t.Fatalf("aller-retour scellé : %+v", got)
	}
	all, err := s.LoadAll()
	if err != nil || all["111"] == nil || all["111"].OAuthRefreshToken != secretRT {
		t.Fatalf("LoadAll : %+v, %v", all, err)
	}
	if byTag, err := s.LoadByGamertag("alice"); err != nil || byTag.XSTSToken != secretXSTS {
		t.Fatalf("LoadByGamertag : %+v, %v", byTag, err)
	}
}

// TestCodec_ChampsVidesRestentVides : un champ vide n'est pas scellé (le scellé d'une chaîne vide
// ferait croire à un jeton).
func TestCodec_ChampsVidesRestentVides(t *testing.T) {
	dir := tempTokenDir(t)
	s := NewMultiUserTokenStoreWithCipher(dir, reverseCipher{})
	if err := s.Upsert(&UserTokens{XUID: "111", Gamertag: "Alice", OAuthRefreshToken: secretRT}); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load("111")
	if err != nil || got.XSTSToken != "" || got.AccessToken != "" || got.OAuthRefreshToken != secretRT {
		t.Fatalf("champs vides : %+v, %v", got, err)
	}
}

// TestCodec_FichierPlainSousMagasinScellantMigreALEcriture : un fichier d'avant est accepté ; il
// n'est PAS réécrit au chargement ; il est réécrit scellé à la première écriture.
func TestCodec_FichierPlainSousMagasinScellantMigreALEcriture(t *testing.T) {
	dir := tempTokenDir(t)
	plain := NewMultiUserTokenStoreWithCipher(dir, tokencipher.Plain())
	if err := plain.Upsert(seedTokens("111")); err != nil {
		t.Fatal(err)
	}
	before := readFileString(t, dir, "111")

	sealing := NewMultiUserTokenStoreWithCipher(dir, reverseCipher{})
	got, err := sealing.Load("111")
	if err != nil || got.OAuthRefreshToken != secretRT {
		t.Fatalf("lecture d'un fichier plain par un magasin scellant : %+v, %v", got, err)
	}
	if _, err := sealing.LoadAll(); err != nil {
		t.Fatal(err)
	}
	if after := readFileString(t, dir, "111"); after != before {
		t.Fatalf("aucune écriture au simple chargement, le fichier a changé :\n%s", after)
	}

	if err := sealing.UpdateTokenClientFamily("111", TokenFamilyXboxNative); err != nil {
		t.Fatal(err)
	}
	migrated := readFileString(t, dir, "111")
	if strings.Contains(migrated, secretRT) || !strings.Contains(migrated, `"cipher": "dpapi"`) {
		t.Fatalf("première écriture : le fichier doit être scellé :\n%s", migrated)
	}
	if again, err := sealing.Load("111"); err != nil || again.OAuthRefreshToken != secretRT ||
		again.TokenClientFamily != TokenFamilyXboxNative {
		t.Fatalf("après migration : %+v, %v", again, err)
	}
}

// seedSealed écrit un fichier scellé et rend son dossier.
func seedSealed(t *testing.T, xuid string) string {
	t.Helper()
	dir := tempTokenDir(t)
	if err := NewMultiUserTokenStoreWithCipher(dir, reverseCipher{}).Upsert(seedTokens(xuid)); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestCodec_FichierScelleSousMagasinPlainRefuse : refus typé et compté, le joueur est traité comme
// sans jeton ; LoadAll ignore le fichier et sert les autres joueurs.
func TestCodec_FichierScelleSousMagasinPlainRefuse(t *testing.T) {
	dir := seedSealed(t, "111")
	plain := NewMultiUserTokenStoreWithCipher(dir, tokencipher.Plain())
	if err := plain.Upsert(&UserTokens{XUID: "222", Gamertag: "Bob", OAuthRefreshToken: "rt-bob"}); err != nil {
		t.Fatal(err)
	}

	before := unreadableCount(unreadableSealed)
	if _, err := plain.Load("111"); !errors.Is(err, ErrTokenFileSealed) {
		t.Fatalf("Load d'un fichier scellé par un magasin plain = %v, attendu ErrTokenFileSealed", err)
	}
	if unreadableCount(unreadableSealed) <= before {
		t.Error("le refus doit être compté")
	}
	all, err := plain.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll ne doit pas tomber : %v", err)
	}
	if all["111"] != nil || all["222"] == nil || all["222"].OAuthRefreshToken != "rt-bob" {
		t.Fatalf("LoadAll : le fichier scellé est ignoré, l'autre joueur servi : %+v", all)
	}
	if _, err := plain.LoadByGamertag("Alice"); !errors.Is(err, ErrTokenFileSealed) || errors.Is(err, ErrUserTokensNotFound) {
		t.Fatalf("LoadByGamertag d'un joueur scellé : %v (illisible, pas introuvable)", err)
	}
	if plain.IsReauthRequired("111") {
		t.Error("un fichier illisible n'est pas un état de ré-authentification")
	}
}

// TestCodec_OuvertureEnEchecEstUnRefusType : un scellé que le chiffreur n'ouvre pas (autre compte
// Windows) est le même refus.
func TestCodec_OuvertureEnEchecEstUnRefusType(t *testing.T) {
	dir := seedSealed(t, "111")
	s := NewMultiUserTokenStoreWithCipher(dir, reverseCipher{failOpen: true})
	if _, err := s.Load("111"); !errors.Is(err, ErrTokenFileSealed) {
		t.Fatalf("Open en échec = %v, attendu ErrTokenFileSealed", err)
	}
}

// TestCodec_EcritureSurFichierScelleIllisible : une écriture partielle ne détruit pas un credential
// peut-être récupérable ; une nouvelle connexion (refresh token neuf) le remplace.
func TestCodec_EcritureSurFichierScelleIllisible(t *testing.T) {
	dir := seedSealed(t, "111")
	sealedBytes := readFileString(t, dir, "111")
	plain := NewMultiUserTokenStoreWithCipher(dir, tokencipher.Plain())

	if err := plain.Upsert(&UserTokens{XUID: "111", XSTSToken: "x"}); !errors.Is(err, ErrTokenFileSealed) {
		t.Fatalf("Upsert partiel = %v, attendu ErrTokenFileSealed", err)
	}
	if err := plain.UpdateOAuthRefreshToken("111", "rt-rotate"); err == nil {
		t.Fatal("la rotation d'un RT ne peut pas partir d'un fichier illisible")
	}
	if got := readFileString(t, dir, "111"); got != sealedBytes {
		t.Fatalf("les refus n'écrivent rien :\n%s", got)
	}

	if err := plain.Upsert(&UserTokens{XUID: "111", Gamertag: "Alice", OAuthRefreshToken: "rt-neuf"}); err != nil {
		t.Fatalf("une nouvelle connexion remplace le fichier : %v", err)
	}
	if got, err := plain.Load("111"); err != nil || got.OAuthRefreshToken != "rt-neuf" {
		t.Fatalf("après remplacement : %+v, %v", got, err)
	}
}

// TestCodec_JSONInvalideCompte : un fichier corrompu reste une erreur ordinaire, comptée à part.
func TestCodec_JSONInvalideCompte(t *testing.T) {
	dir := tempTokenDir(t)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "111.json"), []byte("{pas du json"), 0o600); err != nil {
		t.Fatal(err)
	}
	before := unreadableCount(unreadableDecode)
	s := NewMultiUserTokenStoreWithCipher(dir, tokencipher.Plain())
	if _, err := s.Load("111"); err == nil || errors.Is(err, ErrTokenFileSealed) {
		t.Fatalf("JSON invalide = %v", err)
	}
	if unreadableCount(unreadableDecode) <= before {
		t.Error("le décodage en échec doit être compté")
	}
}

// TestCodec_LeConstructeurLitLeChiffreurDuProcessus : NewMultiUserTokenStore, utilisé par le serveur
// et par tous les outils, prend le chiffreur du processus.
func TestCodec_LeConstructeurLitLeChiffreurDuProcessus(t *testing.T) {
	t.Cleanup(func() { tokencipher.InstallProcess(tokencipher.Plain()) })
	tokencipher.InstallProcess(reverseCipher{})
	dir := tempTokenDir(t)
	if err := NewMultiUserTokenStore(dir).Upsert(seedTokens("111")); err != nil {
		t.Fatal(err)
	}
	if raw := readFileString(t, dir, "111"); strings.Contains(raw, secretRT) {
		t.Fatalf("le constructeur par défaut doit sceller sous le chiffreur du processus :\n%s", raw)
	}
	tokencipher.InstallProcess(tokencipher.Plain())
	if got := NewMultiUserTokenStore(dir).cipher.Name(); got != tokencipher.NamePlain {
		t.Fatalf("chiffreur par défaut = %q", got)
	}
}

// TestCodec_ToutChampTexteDeUserTokensEstClasse : une chaîne ajoutée à UserTokens est soit un secret
// (sensitiveFields, donc scellée), soit déclarée non sensible ICI. Sans ce test, un nouveau jeton
// resterait en clair dans un fichier « scellé ».
func TestCodec_ToutChampTexteDeUserTokensEstClasse(t *testing.T) {
	var u UserTokens
	exigerChampsClasses(t, &u, u.sensitiveFields(), map[string]bool{
		"XUID": true, "Gamertag": true, "TokenClientFamily": true,
		"LastAuthErrorClass": true, "LastAuthError": true, // le message ne contient jamais de jeton
	})
}

// TestCodec_ToutChampTexteDeStoredTokensEstClasse : même contrat pour le fichier d'état du watcher.
func TestCodec_ToutChampTexteDeStoredTokensEstClasse(t *testing.T) {
	var s StoredTokens
	exigerChampsClasses(t, &s, s.sensitiveFields(), map[string]bool{"XSTSGamertag": true, "XSTSXUID": true})
}

// exigerChampsClasses vérifie que chaque chaîne de la structure pointée par v est soit dans sensibles
// (donc scellée), soit déclarée non sensible, et jamais les deux.
func exigerChampsClasses(t *testing.T, v any, sensitivePtrs []*string, nonSensibles map[string]bool) {
	t.Helper()
	base := reflect.ValueOf(v).Pointer()
	sensibles := map[uintptr]bool{}
	for _, p := range sensitivePtrs {
		sensibles[reflect.ValueOf(p).Pointer()-base] = true
	}
	if len(sensibles) != len(sensitivePtrs) {
		t.Error("sensitiveFields rend un champ en double")
	}
	rt := reflect.TypeOf(v).Elem()
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if f.Type.Kind() != reflect.String {
			continue
		}
		if !sensibles[f.Offset] && !nonSensibles[f.Name] {
			t.Errorf("%s.%s est une chaîne ni scellée ni déclarée non sensible", rt.Name(), f.Name)
		}
		if sensibles[f.Offset] && nonSensibles[f.Name] {
			t.Errorf("%s.%s est à la fois sensible et non sensible", rt.Name(), f.Name)
		}
	}
}

// seedWatcherState : un état de watcher complet.
func seedWatcherState() *StoredTokens {
	return &StoredTokens{
		AccessToken: secretAccess, XSTSToken: secretXSTS, XSTSUserHash: secretHash,
		XSTSGamertag: "Alice", XSTSXUID: "111",
	}
}

func tempWatcherStatePath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "watcher_tokens.json")
}

func readPath(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestWatcherState_PlainIdentiqueALOctet : un magasin plain écrit exactement le format d'avant le
// chiffreur (aucun champ cipher) et le relit.
func TestWatcherState_PlainIdentiqueALOctet(t *testing.T) {
	p := tempWatcherStatePath(t)
	s := NewTokenStoreWithCipher(p, tokencipher.Plain())
	want := seedWatcherState()
	if err := s.Save(want); err != nil {
		t.Fatal(err)
	}
	legacy, err := json.MarshalIndent(want, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if got := readPath(t, p); got != string(legacy) {
		t.Fatalf("le fichier plain doit rester identique à l'octet :\n%s", got)
	}
	got, err := s.Load()
	if err != nil || got.XSTSToken != secretXSTS || got.AccessToken != secretAccess {
		t.Fatalf("aller-retour plain : %+v, %v", got, err)
	}
}

// TestWatcherState_ScelleSeulementLesSecrets : sous un chiffreur scellant, access_token, xsts_token et
// xsts_user_hash ne sont plus en clair ; l'identité reste lisible ; l'aller-retour restitue tout.
func TestWatcherState_ScelleSeulementLesSecrets(t *testing.T) {
	p := tempWatcherStatePath(t)
	s := NewTokenStoreWithCipher(p, reverseCipher{})
	if err := s.Save(seedWatcherState()); err != nil {
		t.Fatal(err)
	}
	raw := readPath(t, p)
	for _, secret := range []string{secretAccess, secretXSTS, secretHash} {
		if strings.Contains(raw, secret) {
			t.Errorf("le secret %q est en clair :\n%s", secret, raw)
		}
	}
	for _, clair := range []string{`"cipher": "dpapi"`, `"xsts_gamertag": "Alice"`, `"xsts_xuid": "111"`} {
		if !strings.Contains(raw, clair) {
			t.Errorf("%s doit rester en clair :\n%s", clair, raw)
		}
	}
	got, err := s.Load()
	if err != nil || got.AccessToken != secretAccess || got.XSTSToken != secretXSTS || got.XSTSUserHash != secretHash {
		t.Fatalf("aller-retour scellé : %+v, %v", got, err)
	}
}

// TestWatcherState_MigreALaPremiereEcriture : un fichier d'avant est lu tel quel, non réécrit au
// chargement, puis scellé par la première mise à jour (UpdateOAuth, UpdateXSTS) du cycle du watcher.
func TestWatcherState_MigreALaPremiereEcriture(t *testing.T) {
	p := tempWatcherStatePath(t)
	if err := NewTokenStoreWithCipher(p, tokencipher.Plain()).Save(seedWatcherState()); err != nil {
		t.Fatal(err)
	}
	before := readPath(t, p)

	s := NewTokenStoreWithCipher(p, reverseCipher{})
	if got, err := s.Load(); err != nil || got.XSTSToken != secretXSTS {
		t.Fatalf("lecture d'un fichier plain : %+v, %v", got, err)
	}
	if after := readPath(t, p); after != before {
		t.Fatalf("aucune écriture au simple chargement :\n%s", after)
	}

	if err := s.UpdateOAuth("at-neuf-0123456789", time.Hour); err != nil {
		t.Fatal(err)
	}
	migrated := readPath(t, p)
	if strings.Contains(migrated, secretXSTS) || strings.Contains(migrated, "at-neuf") ||
		!strings.Contains(migrated, `"cipher": "dpapi"`) {
		t.Fatalf("la première écriture scelle tout le fichier :\n%s", migrated)
	}
	if got, err := s.Load(); err != nil || got.AccessToken != "at-neuf-0123456789" || got.XSTSToken != secretXSTS {
		t.Fatalf("après migration : %+v, %v", got, err)
	}
}

// TestWatcherState_ScelleRefuseParUnMagasinPlain : refus typé, aucune mise à jour ne réécrit le fichier.
func TestWatcherState_ScelleRefuseParUnMagasinPlain(t *testing.T) {
	p := tempWatcherStatePath(t)
	if err := NewTokenStoreWithCipher(p, reverseCipher{}).Save(seedWatcherState()); err != nil {
		t.Fatal(err)
	}
	sealedBytes := readPath(t, p)
	plain := NewTokenStoreWithCipher(p, tokencipher.Plain())
	if _, err := plain.Load(); !errors.Is(err, ErrTokenFileSealed) {
		t.Fatalf("Load = %v, attendu ErrTokenFileSealed", err)
	}
	if err := plain.UpdateOAuth("x", time.Hour); !errors.Is(err, ErrTokenFileSealed) {
		t.Fatalf("UpdateOAuth = %v, attendu ErrTokenFileSealed", err)
	}
	if got := readPath(t, p); got != sealedBytes {
		t.Fatal("un refus n'écrit rien")
	}
	if _, err := NewTokenStoreWithCipher(p, reverseCipher{failOpen: true}).Load(); !errors.Is(err, ErrTokenFileSealed) {
		t.Fatalf("Open en échec = %v, attendu ErrTokenFileSealed", err)
	}
}

// TestWatcherState_LeConstructeurLitLeChiffreurDuProcessus : NewTokenStore (watcher du serveur, API,
// outils) prend le même chiffreur de processus que le magasin par joueur.
func TestWatcherState_LeConstructeurLitLeChiffreurDuProcessus(t *testing.T) {
	t.Cleanup(func() { tokencipher.InstallProcess(tokencipher.Plain()) })
	tokencipher.InstallProcess(reverseCipher{})
	p := tempWatcherStatePath(t)
	if err := NewTokenStore(p).Save(seedWatcherState()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(readPath(t, p), secretXSTS) {
		t.Fatal("NewTokenStore doit sceller sous le chiffreur du processus")
	}
}

// capturedWarns : un handler slog qui compte les avertissements.
type capturedWarns struct{ n *atomic.Int64 }

func (capturedWarns) Enabled(context.Context, slog.Level) bool { return true }
func (h capturedWarns) Handle(_ context.Context, r slog.Record) error {
	if r.Level == slog.LevelWarn {
		h.n.Add(1)
	}
	return nil
}
func (h capturedWarns) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h capturedWarns) WithGroup(string) slog.Handler      { return h }

// TestSealedRefusal_UnSeulAvertissementParFichierEtParProcessus : les magasins sont reconstruits à
// chaque passe ; la déduplication tient au processus (clé = chemin), le compteur compte chaque refus.
func TestSealedRefusal_UnSeulAvertissementParFichierEtParProcessus(t *testing.T) {
	resetSealedLogged()
	t.Cleanup(resetSealedLogged)
	var warns atomic.Int64
	prev := slog.Default()
	slog.SetDefault(slog.New(capturedWarns{n: &warns}))
	t.Cleanup(func() { slog.SetDefault(prev) })

	dir := seedSealed(t, "111")
	if err := NewMultiUserTokenStoreWithCipher(dir, reverseCipher{}).Upsert(seedTokens("222")); err != nil {
		t.Fatal(err)
	}
	statePath := tempWatcherStatePath(t)
	if err := NewTokenStoreWithCipher(statePath, reverseCipher{}).Save(seedWatcherState()); err != nil {
		t.Fatal(err)
	}

	before := unreadableCount(unreadableSealed)
	const passes = 5
	for i := 0; i < passes; i++ { // un magasin neuf par passe, comme la sync et les outils
		plain := NewMultiUserTokenStoreWithCipher(dir, tokencipher.Plain())
		_, _ = plain.LoadAll()
		_, _ = plain.Load("111")
		_, _ = NewTokenStoreWithCipher(statePath, tokencipher.Plain()).Load()
	}
	// Trois fichiers (111, 222, état du watcher) : un avertissement chacun, pas un par passe.
	if got := warns.Load(); got != 3 {
		t.Errorf("avertissements = %d, attendu 3 (un par fichier et par processus)", got)
	}
	// LoadAll lit 111 et 222, Load("111") lit 111, l'état est lu une fois : 4 refus par passe.
	if got := unreadableCount(unreadableSealed) - before; got != 4*passes {
		t.Errorf("refus comptés = %d, attendu %d (le compteur compte chaque refus)", got, 4*passes)
	}
}
