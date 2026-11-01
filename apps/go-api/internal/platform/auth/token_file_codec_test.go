// Package auth — token_file_codec_test.go : forme sur disque des fichiers de jetons et relation au
// chiffreur du magasin (ADR 0045 D-11). Ces tests tournent partout : le chiffreur scellant y est un
// double qui porte le nom `dpapi` ; le coffre Windows réel est testé par token_file_vault_test.go.
package auth

import (
	"errors"
	"expvar"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

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
	nonSensibles := map[string]bool{
		"XUID": true, "Gamertag": true, "TokenClientFamily": true,
		"LastAuthErrorClass": true, "LastAuthError": true, // le message ne contient jamais de jeton
	}
	var u UserTokens
	base := reflect.ValueOf(&u).Pointer()
	sensibles := map[uintptr]bool{}
	for _, p := range u.sensitiveFields() {
		sensibles[reflect.ValueOf(p).Pointer()-base] = true
	}
	if len(sensibles) != len(u.sensitiveFields()) {
		t.Error("sensitiveFields rend un champ en double")
	}
	rt := reflect.TypeOf(u)
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if f.Type.Kind() != reflect.String {
			continue
		}
		if !sensibles[f.Offset] && !nonSensibles[f.Name] {
			t.Errorf("UserTokens.%s est une chaîne ni scellée ni déclarée non sensible", f.Name)
		}
		if sensibles[f.Offset] && nonSensibles[f.Name] {
			t.Errorf("UserTokens.%s est à la fois sensible et non sensible", f.Name)
		}
	}
}
