// Package auth — pop_signing_test.go : tests unitaires pour les primitives PoP.
package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/base64"
	"math/big"
	"strings"
	"testing"
	"time"
)

// TestDERtoP1363_WellFormed vérifie que la conversion produit exactement 64 octets
// et que r et s sont non nuls.
func TestDERtoP1363_WellFormed(t *testing.T) {
	kp, err := GeneratePoPKeyPair()
	if err != nil {
		t.Fatalf("GeneratePoPKeyPair: %v", err)
	}

	sig, err := kp.SignRequest("https://device.auth.xboxlive.com/device/authenticate", "", `{"test":1}`)
	if err != nil {
		t.Fatalf("SignRequest: %v", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		t.Fatalf("décodage Base64: %v", err)
	}
	// header = 4 (policy) + 8 (ts) + 64 (sig P1363) = 76 octets
	if len(decoded) != 76 {
		t.Fatalf("longueur header attendue 76, obtenu %d", len(decoded))
	}
	// Les 64 derniers octets ne doivent pas être tous zéros
	sigBytes := decoded[12:]
	allZero := true
	for _, b := range sigBytes {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Fatal("signature P1363 est tout à zéro — problème de conversion DER→P1363")
	}
}

// TestDERtoP1363_SmallRS vérifie le zero-padding quand r ou s < 32 octets.
func TestDERtoP1363_SmallRS(t *testing.T) {
	// Construire un DER avec r=1, s=1 (très petits)
	r := big.NewInt(1)
	s := big.NewInt(1)
	der, err := asn1.Marshal(struct{ R, S *big.Int }{r, s})
	if err != nil {
		t.Fatalf("asn1.Marshal: %v", err)
	}

	p1363, err := derToP1363(der)
	if err != nil {
		t.Fatalf("derToP1363: %v", err)
	}
	if len(p1363) != 64 {
		t.Fatalf("attendu 64 octets, obtenu %d", len(p1363))
	}
	// r doit être dans les 32 premiers octets, zero-padded
	if p1363[31] != 0x01 {
		t.Errorf("octet r[31] attendu 0x01, obtenu 0x%02x", p1363[31])
	}
	// Les 31 premiers octets de r doivent être 0
	for i := 0; i < 31; i++ {
		if p1363[i] != 0 {
			t.Errorf("p1363[%d] attendu 0x00, obtenu 0x%02x", i, p1363[i])
		}
	}
}

// TestDERtoP1363_InvalidDER vérifie qu'un DER invalide retourne une erreur.
func TestDERtoP1363_InvalidDER(t *testing.T) {
	_, err := derToP1363([]byte{0xDE, 0xAD, 0xBE, 0xEF})
	if err == nil {
		t.Fatal("attendu une erreur pour un DER invalide")
	}
}

// TestGeneratePoPKeyPair vérifie les champs du JWK public.
func TestGeneratePoPKeyPair(t *testing.T) {
	kp, err := GeneratePoPKeyPair()
	if err != nil {
		t.Fatalf("GeneratePoPKeyPair: %v", err)
	}
	pk := kp.GetProofKey()

	if pk.Kty != "EC" {
		t.Errorf("kty attendu EC, obtenu %q", pk.Kty)
	}
	if pk.Alg != "ES256" {
		t.Errorf("alg attendu ES256, obtenu %q", pk.Alg)
	}
	if pk.Crv != "P-256" {
		t.Errorf("crv attendu P-256, obtenu %q", pk.Crv)
	}
	if pk.Use != "sig" {
		t.Errorf("use attendu sig, obtenu %q", pk.Use)
	}

	// x et y doivent être des base64url valides de 32 octets
	for name, val := range map[string]string{"x": pk.X, "y": pk.Y} {
		b, err := base64.RawURLEncoding.DecodeString(val)
		if err != nil {
			t.Errorf("champ %s n'est pas base64url valide: %v", name, err)
		}
		if len(b) != 32 {
			t.Errorf("champ %s attendu 32 octets, obtenu %d", name, len(b))
		}
	}
}

// TestSignRequest_Base64Standard vérifie que la sortie est du Base64 standard (pas URL).
func TestSignRequest_Base64Standard(t *testing.T) {
	kp, err := GeneratePoPKeyPair()
	if err != nil {
		t.Fatalf("GeneratePoPKeyPair: %v", err)
	}
	sig, err := kp.SignRequest("https://device.auth.xboxlive.com/device/authenticate", "", `{}`)
	if err != nil {
		t.Fatalf("SignRequest: %v", err)
	}
	// Base64 standard utilise + et /, pas - et _
	if strings.ContainsAny(sig, "-_") {
		t.Error("signature contient des caractères base64url — doit être base64 standard")
	}
	// Doit se décoder sans erreur
	if _, err := base64.StdEncoding.DecodeString(sig); err != nil {
		t.Errorf("décodage base64 standard: %v", err)
	}
}

// TestSignRequest_TimestampEncoded vérifie que le FILETIME est encodé dans le header.
func TestSignRequest_TimestampEncoded(t *testing.T) {
	kp, err := GeneratePoPKeyPair()
	if err != nil {
		t.Fatalf("GeneratePoPKeyPair: %v", err)
	}

	before := windowsFILETIME(time.Now())
	sig, err := kp.SignRequest("https://device.auth.xboxlive.com/device/authenticate", "", `{}`)
	after := windowsFILETIME(time.Now())
	if err != nil {
		t.Fatalf("SignRequest: %v", err)
	}

	decoded, _ := base64.StdEncoding.DecodeString(sig)
	// octets [4:12] = FILETIME BE
	ts := int64(decoded[4])<<56 | int64(decoded[5])<<48 | int64(decoded[6])<<40 |
		int64(decoded[7])<<32 | int64(decoded[8])<<24 | int64(decoded[9])<<16 |
		int64(decoded[10])<<8 | int64(decoded[11])
	if ts < before || ts > after {
		t.Errorf("FILETIME %d hors de la fenêtre [%d, %d]", ts, before, after)
	}
}

// TestSignRequest_VerifiableSignature vérifie que la signature est vérifiable avec la clé publique.
func TestSignRequest_VerifiableSignature(t *testing.T) {
	kp, err := GeneratePoPKeyPair()
	if err != nil {
		t.Fatalf("GeneratePoPKeyPair: %v", err)
	}

	uri := "https://device.auth.xboxlive.com/device/authenticate"
	body := `{"test":"value"}`

	sigEncoded, err := kp.SignRequest(uri, "", body)
	if err != nil {
		t.Fatalf("SignRequest: %v", err)
	}

	decoded, _ := base64.StdEncoding.DecodeString(sigEncoded)
	// Reconstituer le payload avec le timestamp extrait
	ts := int64(decoded[4])<<56 | int64(decoded[5])<<48 | int64(decoded[6])<<40 |
		int64(decoded[7])<<32 | int64(decoded[8])<<24 | int64(decoded[9])<<16 |
		int64(decoded[10])<<8 | int64(decoded[11])

	pathAndQuery := extractPathAndQuery(uri)
	payload := buildPayload(ts, pathAndQuery, "", body)
	hash := sha256.Sum256(payload)

	// Reconstituer r, s depuis P1363
	sigP1363 := decoded[12:]
	r := new(big.Int).SetBytes(sigP1363[:32])
	s := new(big.Int).SetBytes(sigP1363[32:])

	pub := kp.privateKey.Public().(*ecdsa.PublicKey)
	if !ecdsa.Verify(pub, hash[:], r, s) {
		t.Error("signature P1363 non vérifiable avec la clé publique")
	}
}

// TestWindowsFILETIME vérifie la conversion Unix → FILETIME pour une date connue.
func TestWindowsFILETIME(t *testing.T) {
	// Unix epoch (1970-01-01T00:00:00Z) → FILETIME = 116444736000000000
	epoch := time.Unix(0, 0).UTC()
	ft := windowsFILETIME(epoch)
	const expected = int64(116_444_736_000_000_000)
	if ft != expected {
		t.Errorf("FILETIME epoch attendu %d, obtenu %d", expected, ft)
	}
}

// TestExtractPathAndQuery vérifie l'extraction du chemin depuis une URL.
func TestExtractPathAndQuery(t *testing.T) {
	cases := []struct {
		rawURL   string
		expected string
	}{
		{"https://device.auth.xboxlive.com/device/authenticate", "/device/authenticate"},
		{"https://sisu.xboxlive.com/authenticate?foo=bar", "/authenticate?foo=bar"},
		{"https://example.com/", "/"},
		// URL sans path : url.Parse retourne Path="" → fallback "/"
		{"https://example.com", "/"},
	}
	for _, tc := range cases {
		got := extractPathAndQuery(tc.rawURL)
		if got != tc.expected {
			t.Errorf("extractPathAndQuery(%q) = %q, attendu %q", tc.rawURL, got, tc.expected)
		}
	}
}

// TestProofKey_IndependentInstances vérifie que deux paires de clés ont des JWK distincts.
func TestProofKey_IndependentInstances(t *testing.T) {
	kp1, _ := GeneratePoPKeyPair()
	kp2, _ := GeneratePoPKeyPair()
	if kp1.GetProofKey().X == kp2.GetProofKey().X {
		t.Error("deux paires de clés distinctes ont le même X — génération non aléatoire")
	}
}

// Vérifie que P-256 est bien la courbe utilisée.
func TestGeneratePoPKeyPair_CurveIsP256(t *testing.T) {
	kp, _ := GeneratePoPKeyPair()
	if kp.privateKey.Curve != elliptic.P256() {
		t.Error("courbe attendue P-256")
	}
}
