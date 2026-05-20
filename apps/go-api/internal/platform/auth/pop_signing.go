// Package auth — pop_signing.go : primitives PoP (Proof-of-Possession) ECDSA P-256.
//
// Portage de pop-crypto-provider.ts + xbox-authentication-client.ts#signRequest (Conch, MIT).
//
// Algorithme de signature :
//
//	payload = [4B policy_version=1 BE] + [0x00]
//	        + [8B Windows FILETIME BE]  + [0x00]
//	        + UTF-8("POST\0" + pathAndQuery + "\0" + authToken + "\0" + body + "\0")
//
//	sig_der   = ecdsa.Sign(rand, privateKey, sha256(payload))
//	sig_p1363 = DER→P1363(sig_der)   // r‖s, 32+32 octets pour P-256
//
//	header_bytes = [4B policy_version BE] + [8B timestamp BE] + sig_p1363
//	Signature    = base64.StdEncoding.EncodeToString(header_bytes)
package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"log/slog"
	"math/big"
	"net/url"
	"time"
)

const (
	// popPolicyVersion est la version de politique PoP (fixe à 1).
	popPolicyVersion = uint32(1)
	// windowsEpochDelta est l'offset en secondes entre l'epoch Unix et l'epoch Windows FILETIME.
	windowsEpochDelta = int64(11_644_473_600)
	// p256KeySize est la taille en octets de r et s pour P-256.
	p256KeySize = 32
)

// ProofKey est le JWK public à inclure dans le body des requêtes Device/SISU.
type ProofKey struct {
	Kty string `json:"kty"` // "EC"
	Alg string `json:"alg"` // "ES256"
	Crv string `json:"crv"` // "P-256"
	Use string `json:"use"` // "sig"
	X   string `json:"x"`   // base64url coordonnée X
	Y   string `json:"y"`   // base64url coordonnée Y
}

// PoPKeyPair encapsule la paire ECDSA P-256 éphémère et le JWK public.
type PoPKeyPair struct {
	privateKey *ecdsa.PrivateKey
	proofKey   ProofKey
}

// GeneratePoPKeyPair génère une paire de clés ECDSA P-256 éphémère.
func GeneratePoPKeyPair() (*PoPKeyPair, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("pop_signing: génération clé P-256: %w", err)
	}
	xBytes := priv.X.Bytes()
	yBytes := priv.Y.Bytes()
	// zero-pad à p256KeySize (32 octets)
	xPadded := make([]byte, p256KeySize)
	yPadded := make([]byte, p256KeySize)
	copy(xPadded[p256KeySize-len(xBytes):], xBytes)
	copy(yPadded[p256KeySize-len(yBytes):], yBytes)

	kp := &PoPKeyPair{
		privateKey: priv,
		proofKey: ProofKey{
			Kty: "EC",
			Alg: "ES256",
			Crv: "P-256",
			Use: "sig",
			X:   base64.RawURLEncoding.EncodeToString(xPadded),
			Y:   base64.RawURLEncoding.EncodeToString(yPadded),
		},
	}
	slog.Debug("pop_signing: clé PoP générée", "kty", kp.proofKey.Kty, "crv", kp.proofKey.Crv)
	return kp, nil
}

// GetProofKey retourne le JWK public (à inclure dans les requêtes Device/SISU).
func (kp *PoPKeyPair) GetProofKey() ProofKey {
	return kp.proofKey
}

// SignRequest construit et signe le header "Signature" PoP pour les endpoints Xbox.
// uri       : URL complète de la requête
// authToken : "" pour Device/SISU (non authentifié à ce stade)
// body      : corps JSON de la requête (string)
// Retourne  : header Signature encodé en Base64 standard.
func (kp *PoPKeyPair) SignRequest(uri, authToken, body string) (string, error) {
	ts := windowsFILETIME(time.Now())

	pathAndQuery := extractPathAndQuery(uri)
	payload := buildPayload(ts, pathAndQuery, authToken, body)

	hash := sha256.Sum256(payload)
	sigDER, err := ecdsa.SignASN1(rand.Reader, kp.privateKey, hash[:])
	if err != nil {
		return "", fmt.Errorf("pop_signing: signature ECDSA: %w", err)
	}

	sigP1363, err := derToP1363(sigDER)
	if err != nil {
		return "", fmt.Errorf("pop_signing: conversion DER→P1363: %w", err)
	}

	header := buildSignatureHeader(ts, sigP1363)
	encoded := base64.StdEncoding.EncodeToString(header)
	slog.Debug("pop_signing: signature construite", "uri", uri, "sig_len", len(header))
	return encoded, nil
}

// =============================================================================
// Fonctions internes
// =============================================================================

// windowsFILETIME convertit un time.Time en Windows FILETIME (100ns ticks depuis 1601-01-01).
func windowsFILETIME(t time.Time) int64 {
	return (t.Unix()+windowsEpochDelta)*10_000_000 + int64(t.Nanosecond())/100
}

// extractPathAndQuery extrait le chemin + query d'une URL.
// Retourne "/" en cas d'erreur de parsing.
func extractPathAndQuery(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Path == "" {
		return "/"
	}
	if u.RawQuery != "" {
		return u.Path + "?" + u.RawQuery
	}
	return u.Path
}

// buildPayload construit le payload à signer selon le protocole PoP Xbox.
func buildPayload(ts int64, pathAndQuery, authToken, body string) []byte {
	// [4B policy_version BE] + [0x00]
	// Pre-allocation : 4 (pv) + 1 + 8 (ft) + 1 + 4 (BE_uint32) + 1 +
	// path + 1 + token + 1 + body + 1.
	buf := make([]byte, 0, 22+len(pathAndQuery)+len(authToken)+len(body))
	pv := make([]byte, 4)
	binary.BigEndian.PutUint32(pv, popPolicyVersion)
	buf = append(buf, pv...)
	buf = append(buf, 0x00)

	// [8B FILETIME BE] + [0x00]
	ft := make([]byte, 8)
	binary.BigEndian.PutUint64(ft, uint64(ts))
	buf = append(buf, ft...)
	buf = append(buf, 0x00)

	// "POST\0" + pathAndQuery + "\0" + authToken + "\0" + body + "\0"
	buf = append(buf, []byte("POST")...)
	buf = append(buf, 0x00)
	buf = append(buf, []byte(pathAndQuery)...)
	buf = append(buf, 0x00)
	buf = append(buf, []byte(authToken)...)
	buf = append(buf, 0x00)
	buf = append(buf, []byte(body)...)
	buf = append(buf, 0x00)

	return buf
}

// buildSignatureHeader assemble le header final : [4B policy BE] + [8B ts BE] + sig_p1363.
func buildSignatureHeader(ts int64, sigP1363 []byte) []byte {
	header := make([]byte, 4+8+len(sigP1363))
	binary.BigEndian.PutUint32(header[0:4], popPolicyVersion)
	binary.BigEndian.PutUint64(header[4:12], uint64(ts))
	copy(header[12:], sigP1363)
	return header
}

// derToP1363 convertit une signature ECDSA ASN.1 DER en IEEE P1363 (r‖s).
// Pour P-256 : r et s sont zero-paddés à 32 octets chacun → 64 octets au total.
func derToP1363(der []byte) ([]byte, error) {
	var sig struct {
		R, S *big.Int
	}
	if rest, err := asn1.Unmarshal(der, &sig); err != nil {
		return nil, fmt.Errorf("asn1 unmarshal: %w", err)
	} else if len(rest) != 0 {
		return nil, fmt.Errorf("données residuelles après DER: %d octets", len(rest))
	}
	if sig.R == nil || sig.S == nil {
		return nil, fmt.Errorf("r ou s nil après unmarshal DER")
	}

	out := make([]byte, 2*p256KeySize)
	rBytes := sig.R.Bytes()
	sBytes := sig.S.Bytes()
	if len(rBytes) > p256KeySize || len(sBytes) > p256KeySize {
		return nil, fmt.Errorf("r ou s dépasse %d octets", p256KeySize)
	}
	// zero-pad à gauche
	copy(out[p256KeySize-len(rBytes):p256KeySize], rBytes)
	copy(out[2*p256KeySize-len(sBytes):], sBytes)
	return out, nil
}
