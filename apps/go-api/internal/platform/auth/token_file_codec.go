// Package auth — token_file_codec.go : forme sur disque des fichiers de jetons et leur relation au
// chiffreur du magasin (ADR 0045 D-11). Deux fichiers portent des secrets et passent par ce codec,
// avec le MÊME chiffreur de processus, le même champ `cipher` et les mêmes règles :
//   - `watcher_tokens/{xuid}.json` ([MultiUserTokenStore], source unique des jetons, ADR 0023) ;
//   - `watcher_tokens.json` ([TokenStore], état mono-utilisateur du watcher RTA : access_token et XSTS).
//
// Chaque fichier reste UN document JSON. Quand le magasin porte un chiffreur autre que `plain`, le
// fichier gagne un champ `cipher` et ses champs sensibles ([UserTokens.sensitiveFields],
// [StoredTokens.sensitiveFields]) sont remplacés par leur scellé en base64 ; tout le reste (identité,
// dates, décision de mise en commun, état de ré-authentification) reste en clair.
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
	"sync"

	"levelup/go-api/internal/platform/auth/tokencipher"
)

// ErrTokenFileSealed : le fichier de jetons est scellé et ce magasin ne sait pas l'ouvrir.
var ErrTokenFileSealed = errors.New("auth: fichier de jetons scellé, illisible par ce magasin")

// TokenCipher est le port de chiffrement des fichiers de jetons (ADR 0045 D-11).
type TokenCipher = tokencipher.Cipher

// tokenFile est la forme JSON d'un fichier de jetons par joueur : [UserTokens] à plat, précédé du chiffreur.
type tokenFile struct {
	// Cipher : nom du chiffreur des champs sensibles. Vide = `plain` (écrit ainsi par un magasin plain,
	// pour que les fichiers d'une instance hébergée restent identiques octet pour octet).
	Cipher string `json:"cipher,omitempty"`
	UserTokens
}

// sensitiveFields rend les champs de u que le chiffreur scelle. Toute chaîne ajoutée à UserTokens qui
// porte un secret DOIT entrer ici ; token_file_codec_test.go refuse une chaîne qui n'est ni ici ni
// déclarée non sensible.
func (u *UserTokens) sensitiveFields() []*string {
	return []*string{&u.OAuthRefreshToken, &u.AccessToken, &u.XSTSToken, &u.XSTSUserHash}
}

// sealFields scelle sur place les champs non vides avec c et rend le nom à écrire dans `cipher`
// (vide pour `plain` : le champ est omis et le fichier reste identique à l'octet).
func sealFields(c TokenCipher, fields []*string) (string, error) {
	if c.Name() == tokencipher.NamePlain {
		return "", nil
	}
	for _, field := range fields {
		if *field == "" {
			continue // le scellé d'une chaîne vide ferait croire à un jeton
		}
		sealed, err := c.Seal([]byte(*field))
		if err != nil {
			return "", fmt.Errorf("auth: scellement (%s): %w", c.Name(), err)
		}
		*field = base64.StdEncoding.EncodeToString(sealed)
	}
	return c.Name(), nil
}

// openFields rend à fields leur clair d'après le champ `cipher` du fichier. Un fichier `plain` ou sans
// champ est laissé tel quel. Un chiffreur inconnu de c, ou un Open en échec, est ErrTokenFileSealed
// (compté, journalisé une fois par fichier et par processus). path ne sert qu'aux messages.
func openFields(c TokenCipher, fileCipher, path string, fields []*string) error {
	if fileCipher == "" || fileCipher == tokencipher.NamePlain {
		return nil
	}
	if fileCipher != c.Name() {
		return sealedRefusal(c, path, fmt.Errorf("scellé par %q, ce magasin est %q", fileCipher, c.Name()))
	}
	for _, field := range fields {
		if *field == "" {
			continue
		}
		plain, err := openField(c, *field)
		if err != nil {
			return sealedRefusal(c, path, err)
		}
		*field = plain
	}
	return nil
}

// openField ouvre un champ scellé (base64 d'un scellé de c).
func openField(c TokenCipher, encoded string) (string, error) {
	sealed, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("champ scellé non base64: %w", err)
	}
	plain, err := c.Open(sealed)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// sealedLogged : fichiers dont le refus ErrTokenFileSealed a déjà été journalisé, par PROCESSUS (clé =
// chemin). Les magasins sont reconstruits à chaque passe (sync, boot, outils) et relisent le même
// fichier à chaque cycle : une déduplication par instance rejouerait l'avertissement à chaque passe.
var sealedLogged sync.Map

// resetSealedLogged remet à zéro la déduplication (tests).
func resetSealedLogged() { sealedLogged.Clear() }

// sealedRefusal compte chaque refus, journalise le premier par fichier et par processus, et rend
// l'erreur typée.
func sealedRefusal(c TokenCipher, path string, cause error) error {
	tokenFileUnreadable.Add(unreadableSealed, 1)
	if _, seen := sealedLogged.LoadOrStore(path, struct{}{}); !seen {
		slog.Warn("token_file: fichier de jetons scellé illisible, traité comme sans jeton",
			"file", path, "cipher", c.Name(), "err", cause)
	}
	return fmt.Errorf("%w: %s: %v", ErrTokenFileSealed, path, cause)
}

// logUnreadableScan journalise un fichier ignoré par un scan (LoadAll, LoadByGamertag). Un fichier
// SCELLÉ a déjà été journalisé une fois par fichier et par processus (sealedRefusal) et compté : le
// rejouer à chaque scan donnerait un avertissement par passe et par joueur (les magasins sont
// reconstruits à chaque passe), il descend donc en Debug. Toute autre cause reste un Warn.
func logUnreadableScan(msg, key, id string, err error, extra ...any) {
	attrs := append([]any{key, id, "err", err}, extra...)
	if errors.Is(err, ErrTokenFileSealed) {
		slog.Debug(msg, attrs...)
		return
	}
	slog.Warn(msg, attrs...)
}

// encodeFile sérialise t pour le disque avec le chiffreur du magasin.
func (s *MultiUserTokenStore) encodeFile(t *UserTokens) ([]byte, error) {
	file := tokenFile{UserTokens: *t}
	name, err := sealFields(s.cipher, file.sensitiveFields())
	if err != nil {
		return nil, err
	}
	file.Cipher = name
	return json.MarshalIndent(&file, "", "  ")
}

// decodeFile lit un fichier de jetons par joueur. path ne sert qu'aux messages.
func (s *MultiUserTokenStore) decodeFile(data []byte, path string) (*UserTokens, error) {
	var file tokenFile
	if err := json.Unmarshal(data, &file); err != nil {
		tokenFileUnreadable.Add(unreadableDecode, 1)
		return nil, fmt.Errorf("multi_user_token_store: décodage %s: %w", path, err)
	}
	t := file.UserTokens
	if err := openFields(s.cipher, file.Cipher, path, t.sensitiveFields()); err != nil {
		return nil, err
	}
	return &t, nil
}
