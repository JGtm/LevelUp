// Package auth — token_file_codec.go : forme sur disque d'un fichier de jetons
// `watcher_tokens/{xuid}.json`, et sa relation au chiffreur du magasin (ADR 0045 D-11).
//
// Le fichier reste UN document JSON par joueur, source unique des jetons (ADR 0023). Quand le
// magasin porte un chiffreur autre que `plain`, le fichier gagne un champ `cipher` et ses champs
// sensibles ([UserTokens.sensitiveFields]) sont remplacés par leur scellé en base64 ; tout le reste
// (xuid, gamertag, dates, décision de mise en commun, état de ré-authentification) reste en clair.
//
// Règles de lecture :
//   - fichier `plain` ou sans champ `cipher` (tous les fichiers antérieurs) : lisible par tout magasin.
//     Sous un magasin scellant il est MIGRÉ à la prochaine écriture, jamais au simple chargement ;
//   - fichier scellé par un chiffreur autre que celui du magasin, ou dont l'ouverture échoue (autre
//     compte Windows, autre machine) : [ErrTokenFileSealed]. Le joueur est alors traité comme sans
//     jeton : un fichier chiffré copié ailleurs est illisible, c'est le but.
package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/platform/auth/tokencipher"
)

// ErrTokenFileSealed : le fichier de jetons est scellé et ce magasin ne sait pas l'ouvrir.
var ErrTokenFileSealed = errors.New("multi_user_token_store: fichier de jetons scellé, illisible par ce magasin")

// TokenCipher est le port de chiffrement des fichiers de jetons (ADR 0045 D-11).
type TokenCipher = tokencipher.Cipher

// tokenFile est la forme JSON d'un fichier de jetons : [UserTokens] à plat, précédé du chiffreur.
type tokenFile struct {
	// Cipher : nom du chiffreur des champs sensibles. Vide = `plain` (écrit ainsi par un magasin plain,
	// pour que les fichiers d'une instance hébergée restent identiques octet pour octet).
	Cipher string `json:"cipher,omitempty"`
	UserTokens
}

// sensitiveFields rend les champs de t que le chiffreur scelle. Toute chaîne ajoutée à UserTokens qui
// porte un secret DOIT entrer ici ; token_file_codec_test.go refuse une chaîne qui n'est ni ici ni
// déclarée non sensible.
func (u *UserTokens) sensitiveFields() []*string {
	return []*string{&u.OAuthRefreshToken, &u.AccessToken, &u.XSTSToken, &u.XSTSUserHash}
}

// encodeFile sérialise t pour le disque avec le chiffreur du magasin.
func (s *MultiUserTokenStore) encodeFile(t *UserTokens) ([]byte, error) {
	file := tokenFile{UserTokens: *t}
	if s.cipher.Name() != tokencipher.NamePlain {
		file.Cipher = s.cipher.Name()
		for _, field := range file.sensitiveFields() {
			if *field == "" {
				continue
			}
			sealed, err := s.cipher.Seal([]byte(*field))
			if err != nil {
				return nil, fmt.Errorf("multi_user_token_store: scellement (%s): %w", s.cipher.Name(), err)
			}
			*field = base64.StdEncoding.EncodeToString(sealed)
		}
	}
	return json.MarshalIndent(&file, "", "  ")
}

// decodeFile lit un fichier de jetons. path ne sert qu'aux messages.
func (s *MultiUserTokenStore) decodeFile(data []byte, path string) (*UserTokens, error) {
	var file tokenFile
	if err := json.Unmarshal(data, &file); err != nil {
		tokenFileUnreadable.Add(unreadableDecode, 1)
		return nil, fmt.Errorf("multi_user_token_store: décodage %s: %w", path, err)
	}
	t := file.UserTokens
	if file.Cipher == "" || file.Cipher == tokencipher.NamePlain {
		return &t, nil
	}
	if file.Cipher != s.cipher.Name() {
		return nil, s.sealedRefusal(path, fmt.Errorf("scellé par %q, ce magasin est %q", file.Cipher, s.cipher.Name()))
	}
	for _, field := range t.sensitiveFields() {
		if *field == "" {
			continue
		}
		plain, err := s.openField(*field)
		if err != nil {
			return nil, s.sealedRefusal(path, err)
		}
		*field = plain
	}
	return &t, nil
}

// openField ouvre un champ scellé (base64 d'un scellé du chiffreur du magasin).
func (s *MultiUserTokenStore) openField(encoded string) (string, error) {
	sealed, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("champ scellé non base64: %w", err)
	}
	plain, err := s.cipher.Open(sealed)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// sealedRefusal compte et journalise (une fois par fichier et par processus : le watcher relit ce
// fichier à chaque cycle) le refus d'un fichier scellé, et rend l'erreur typée.
func (s *MultiUserTokenStore) sealedRefusal(path string, cause error) error {
	tokenFileUnreadable.Add(unreadableSealed, 1)
	if _, seen := s.sealedLogged.LoadOrStore(path, struct{}{}); !seen {
		slog.Warn("multi_user_token_store: fichier de jetons scellé illisible, joueur traité comme sans jeton",
			"file", path, "cipher", s.cipher.Name(), "err", cause)
	}
	return fmt.Errorf("%w: %s: %v", ErrTokenFileSealed, path, cause)
}
