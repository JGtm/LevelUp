package diagnostic

import (
	"bytes"
	"encoding/json"
	"io"
	"regexp"
	"strings"
)

// redacted remplace une valeur retirée.
const redacted = "[retiré]"

// camelBoundary : la frontière d'un nom en casse mixte (« OAuthRefreshToken » -> « OAuth_Refresh_Token »).
var camelBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])`)

// normalizeKey ramène un nom de clé à sa forme comparée : minuscules, mots séparés par « _ ».
func normalizeKey(key string) string {
	k := camelBoundary.ReplaceAllString(strings.TrimSpace(key), "${1}_${2}")
	k = strings.NewReplacer("-", "_", ".", "_", " ", "_").Replace(k)
	return strings.ToLower(k)
}

var (
	// secretKey : les NOMS de clé qui portent un secret, ancrés sur le nom entier (jamais une sous-chaîne
	// d'un mot : « token » dans « players_with_token », « refresh » dans « scheduled_refresh » ou
	// « spnkr_refresh_with_backfill », « xsts » dans « xsts_refreshed » sont des compteurs et des réglages, pas
	// des secrets). Une clé qui CONTIENT « secret », « password » ou « passwd » en est un en revanche.
	secretKey = regexp.MustCompile(`^(?:` +
		`token|bearer|authorization|cookie|signature|cipher|webhook|credentials?|session_key|api_?key|private_key|` +
		`(?:.*_)?(?:access|refresh|id|auth|bearer|session|oauth|device|api|csrf|xsts|spartan|halo|clearance)_token|` +
		`(?:.*_)?(?:webhook_url|authorization|set_cookie|cookie)|` +
		`(?:(?:auth|authorization|oauth|device|user)_)?code|` +
		`.*(?:secret|password|passwd).*)$`)
	// notSecretKey : une clé qui DIT quelque chose d'un secret sans le porter (drapeau, compteur) : « has_refresh_token »,
	// « discord_webhook_url_present », « client_secret_count ».
	notSecretKey = regexp.MustCompile(`^(?:has|is|can|needs)_|_(?:present|configured|set|enabled|count|total|age)$`)
)

// ForbiddenKey dit si un nom de clé désigne une valeur qui ne doit pas figurer dans le rapport. La comparaison
// porte sur le nom ENTIER (casse et séparateurs ramenés à « snake_case ») : voir [secretKey].
func ForbiddenKey(key string) bool {
	k := normalizeKey(key)
	return !notSecretKey.MatchString(k) && secretKey.MatchString(k)
}

// RedactValue retire d'un document décodé (map, tableau, scalaire) toute valeur portée par une clé interdite,
// récursivement, et passe les chaînes restantes par [Scrub]. secrets : valeurs exactes à effacer partout.
func RedactValue(v any, secrets []string) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			if ForbiddenKey(k) {
				out[k] = redacted
				continue
			}
			out[k] = RedactValue(val, secrets)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = RedactValue(val, secrets)
		}
		return out
	case string:
		return Scrub(t, secrets)
	}
	return v
}

var (
	// bearerPattern : un en-tête d'autorisation.
	bearerPattern = regexp.MustCompile(`(?i)\bbearer\s+[A-Za-z0-9._~+/=-]{8,}`)
	// jwtPattern : un jeton à trois segments (JWT : l'en-tête commence par « eyJ »).
	jwtPattern = regexp.MustCompile(`eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{4,}\.[A-Za-z0-9_-]{4,}`)
	// msTokenPattern : un jeton de rafraîchissement Microsoft (« M.C5… », « 0.A… ») ou un long blob opaque.
	msTokenPattern = regexp.MustCompile(`\b(?:M\.[A-Z]\d+_[A-Za-z0-9._!*$-]{20,}|0\.A[A-Za-z0-9._-]{40,})`)
	// pairPattern : une paire « clé=valeur », « clé: valeur » ou « "clé":"valeur" ». Elle ne dit pas si la clé est
	// sensible : [scrubPairs] le décide par [ForbiddenKey], le même critère que pour les documents structurés.
	// La valeur est bornée : entre guillemets, ou jusqu'au prochain blanc, virgule, point-virgule, accolade,
	// esperluette ou guillemet (jamais le guillemet fermant d'un document JSON qui enveloppe la paire).
	pairPattern = regexp.MustCompile(`("?)([A-Za-z0-9_.-]+)("?\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s,;}&"']+)`)
	// urlSecretPattern : un paramètre d'adresse sensible (?secret=…, &code=…, &state=…).
	urlSecretPattern = regexp.MustCompile(`(?i)([?&](?:secret|code|state|token|access_token|refresh_token|id_token)=)[^&\s"']+`)
)

// Scrub efface d'un texte de journal ce qui ressemble à un secret : les valeurs exactes de secrets, les
// en-têtes d'autorisation, les jetons de forme connue, les paires clé=valeur de clé sensible et les paramètres
// d'adresse sensibles. Une ligne n'est jamais supprimée : seule la valeur est remplacée, le journal reste lisible.
func Scrub(s string, secrets []string) string {
	for _, sec := range secrets {
		if len(sec) >= 6 {
			s = strings.ReplaceAll(s, sec, redacted)
		}
	}
	s = bearerPattern.ReplaceAllString(s, "Bearer "+redacted)
	s = jwtPattern.ReplaceAllString(s, redacted)
	s = msTokenPattern.ReplaceAllString(s, redacted)
	s = urlSecretPattern.ReplaceAllString(s, "${1}"+redacted)
	return scrubPairs(s)
}

// scrubPairs remplace la valeur de chaque paire dont la clé est interdite ; les guillemets de la valeur et ceux
// qui l'entourent restent en place. Après une paire dont la clé est permise, la lecture reprend AU DÉBUT de sa
// valeur : « auth: refresh_token=… » est la paire permise « auth: » dont la valeur contient la paire sensible.
func scrubPairs(s string) string {
	var out strings.Builder
	pos := 0
	for pos < len(s) {
		loc := pairPattern.FindStringSubmatchIndex(s[pos:])
		if loc == nil {
			break
		}
		for i := range loc {
			if loc[i] >= 0 {
				loc[i] += pos
			}
		}
		key := s[loc[4]:loc[5]]
		if !ForbiddenKey(key) {
			out.WriteString(s[pos:loc[8]])
			pos = loc[8]
			continue
		}
		val := s[loc[8]:loc[9]]
		repl := redacted
		if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') && val[len(val)-1] == val[0] {
			repl = val[:1] + redacted + val[len(val)-1:]
		}
		out.WriteString(s[pos:loc[8]])
		out.WriteString(repl)
		pos = loc[9]
	}
	out.WriteString(s[pos:])
	return out.String()
}

// ScrubLog expurge le texte d'un journal ligne à ligne. Une ligne qui est UN document JSON (le format des journaux
// par module) est lue comme tel : seules les valeurs des clés interdites et le texte des chaînes sont remplacés,
// l'ordre des clés est gardé et la ligne reste du JSON valide. Toute autre ligne passe par [Scrub].
func ScrubLog(text string, secrets []string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "{") {
			if out, ok := scrubJSONLine(trimmed, secrets); ok {
				lines[i] = out
				continue
			}
		}
		lines[i] = Scrub(line, secrets)
	}
	return strings.Join(lines, "\n")
}

// scrubJSONLine réécrit une ligne JSON en gardant l'ordre des clés. ok est faux quand la ligne n'est pas exactement
// un document JSON.
func scrubJSONLine(line string, secrets []string) (string, bool) {
	dec := json.NewDecoder(strings.NewReader(line))
	dec.UseNumber()
	var buf bytes.Buffer
	if err := copyValue(dec, &buf, secrets); err != nil {
		return "", false
	}
	if _, err := dec.Token(); err != io.EOF {
		return "", false // du texte suit le document
	}
	return buf.String(), true
}

// copyValue copie la valeur JSON suivante de dec vers buf, valeurs de clés interdites retirées, chaînes expurgées.
func copyValue(dec *json.Decoder, buf *bytes.Buffer, secrets []string) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	switch t := tok.(type) {
	case json.Delim:
		if t == '{' {
			return copyObject(dec, buf, secrets)
		}
		return copyArray(dec, buf, secrets)
	case string:
		return writeJSONString(buf, Scrub(t, secrets))
	case json.Number:
		buf.WriteString(t.String())
	case bool:
		if t {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case nil:
		buf.WriteString("null")
	}
	return nil
}

func copyObject(dec *json.Decoder, buf *bytes.Buffer, secrets []string) error {
	buf.WriteByte('{')
	first := true
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return err
		}
		key, _ := keyTok.(string)
		if !first {
			buf.WriteByte(',')
		}
		first = false
		if err := writeJSONString(buf, key); err != nil {
			return err
		}
		buf.WriteByte(':')
		if ForbiddenKey(key) {
			if err := skipValue(dec); err != nil {
				return err
			}
			if err := writeJSONString(buf, redacted); err != nil {
				return err
			}
			continue
		}
		if err := copyValue(dec, buf, secrets); err != nil {
			return err
		}
	}
	if _, err := dec.Token(); err != nil { // '}'
		return err
	}
	buf.WriteByte('}')
	return nil
}

func copyArray(dec *json.Decoder, buf *bytes.Buffer, secrets []string) error {
	buf.WriteByte('[')
	first := true
	for dec.More() {
		if !first {
			buf.WriteByte(',')
		}
		first = false
		if err := copyValue(dec, buf, secrets); err != nil {
			return err
		}
	}
	if _, err := dec.Token(); err != nil { // ']'
		return err
	}
	buf.WriteByte(']')
	return nil
}

// skipValue consomme la valeur JSON suivante sans la garder (objets et tableaux compris).
func skipValue(dec *json.Decoder) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	d, ok := tok.(json.Delim)
	if !ok || (d != '{' && d != '[') {
		return nil
	}
	for depth := 1; depth > 0; {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		if d, ok := tok.(json.Delim); ok {
			if d == '{' || d == '[' {
				depth++
			} else {
				depth--
			}
		}
	}
	return nil
}

// writeJSONString écrit s comme chaîne JSON, sans échapper <, > et & (un journal reste lisible).
func writeJSONString(buf *bytes.Buffer, s string) error {
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(s); err != nil {
		return err
	}
	buf.Truncate(buf.Len() - 1) // le retour à la ligne d'Encode
	return nil
}
