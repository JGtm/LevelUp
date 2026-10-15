// Package archlint — nginx_conf_test.go : garde-rail des configurations nginx
// versionnées (packaging/nginx/), plan sécurité du 2026-10-15, lot L1.
//
// POURQUOI. Le Go croit l'en-tête X-Real-IP derrière le proxy (LEVELUP_TRUST_PROXY_HEADERS) :
// c'est nginx qui le rend non falsifiable en l'écrasant à chaque requête. Le corps des
// requêtes est mis en tampon disque par nginx AVANT toute garde Go : sa taille se borne
// ici. Ces fichiers sont copiés à la main sur le VPS ; ce test tient la copie versionnée,
// qui sert de référence au contrôle de la configuration en place.
//
// CE QUI EST EXIGÉ de chaque `server` qui écoute en 443 :
//   - dans toute `location` qui relaie au Go (proxy_pass), en tenant compte de l'héritage
//     de proxy_set_header (une location qui en déclare un n'hérite plus de ceux du
//     server) : `proxy_set_header X-Real-IP $remote_addr` et
//     `proxy_set_header True-Client-IP ""` (en-tête vidé) ;
//   - `server_tokens off` au niveau server, jamais rallumé dans une location ;
//   - aucune `client_max_body_size` effective au-delà de 1m, sauf dans les locations
//     d'envoi listées dans nginxLargeBodyLocations, chacune sous son plafond ;
//   - pour la production : une `limit_req` effective et `limit_req_status 429` sur les
//     routes de l'API (le front lit le 429 ; le 503 par défaut se lirait panne) ;
//   - toute zone citée par `limit_req` est déclarée dans packaging/nginx/rate_limit.conf.
package archlint

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// nginxConfFiles classe chaque configuration de site versionnée. Un nouveau fichier
// .conf dans packaging/nginx/ doit être classé ici (le test échoue sinon).
var nginxConfFiles = map[string]bool{ // nom → site de production
	"levelup.conf": true,
	"demo.conf":    false,
}

// nginxRateLimitConf déclare les zones de limite de débit (bloc http, conf.d).
const nginxRateLimitConf = "rate_limit.conf"

// nginxDefaultBodyCap est le plafond de corps hors des locations d'envoi (1m, aussi la
// valeur par défaut de nginx quand la directive est absente).
const nginxDefaultBodyCap = 1 << 20

// nginxLargeBodyLocations — locations autorisées à recevoir un corps au-delà de 1m,
// par matcher exact, avec leur plafond. Toute entrée est datée et justifiée par le
// plafond Go de la route.
var nginxLargeBodyLocations = map[string]int64{
	// 2026-10-15 : envoi de clips vidéo, le Go plafonne à 500 Mo par requête.
	"~ ^/api/v1/players/[^/]+/media/upload$": 600 << 20,
	// 2026-10-15 : import multipart OpenSpartan, le Go plafonne à 1 Gio.
	"= /api/v1/import/openspartan": 1100 << 20,
	// 2026-10-15 : protocole ouvrier (jeton bearer), dépôt d'artefact plafonné à 16 Mio par le Go.
	"/api/v1/internal/": 20 << 20,
}

// nginxProdAPIPaths — chemins de l'API du site de production qui doivent passer par une
// limite de débit nginx. Les images (/api/v1/assets/) et le protocole ouvrier
// (/api/v1/internal/) en sont exclus : une page tire des dizaines d'images, et l'ouvrier
// est authentifié par jeton.
var nginxProdAPIPaths = []string{
	"/api/v1/session/context",
	"/api/v1/players/p/matches",
	"/api/v1/auth/login",
	"/api/v1/auth/password",
	"/api/v1/players/p/media/upload",
	"/api/v1/import/openspartan",
}

func TestNginxConfsAreClassified(t *testing.T) {
	dir := nginxPackagingDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("lecture de %s : %v", dir, err)
	}
	seen := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".conf") || e.Name() == nginxRateLimitConf {
			continue
		}
		if _, ok := nginxConfFiles[e.Name()]; !ok {
			t.Errorf("%s : configuration nginx non classée dans nginxConfFiles", e.Name())
		}
		seen[e.Name()] = true
	}
	for name := range nginxConfFiles {
		if !seen[name] {
			t.Errorf("%s : classée dans nginxConfFiles mais absente de packaging/nginx/", name)
		}
	}
}

func TestNginxConfsHardenTLSServers(t *testing.T) {
	dir := nginxPackagingDir(t)
	zones, err := nginxDeclaredZones(dir)
	if err != nil {
		t.Errorf("%s : %v", nginxRateLimitConf, err)
	}
	names := make([]string, 0, len(nginxConfFiles))
	for name := range nginxConfFiles {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		prod := nginxConfFiles[name]
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Errorf("lecture de %s : %v", name, err)
			continue
		}
		for _, v := range nginxConfViolations(string(data), prod, zones) {
			t.Errorf("%s : %s", name, v)
		}
	}
}

// TestNginxConfAnalysisDetectsDefects — l'analyse sait échouer : chaque défaut de la
// classe, injecté dans une configuration saine, est signalé.
func TestNginxConfAnalysisDetectsDefects(t *testing.T) {
	zones := map[string]bool{"lvl_api": true}
	healthy := `server {
    listen 443 ssl;
    server_tokens off;
    limit_req_status 429;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header True-Client-IP "";
    location = /api/v1/import/openspartan { client_max_body_size 1100m; limit_req zone=lvl_api burst=1; proxy_pass http://b; }
    location /api/ { client_max_body_size 1m; limit_req zone=lvl_api burst=1; proxy_pass http://b; }
    location / { proxy_pass http://b; }
}`
	if got := nginxConfViolations(healthy, true, zones); len(got) != 0 {
		t.Fatalf("configuration saine signalée : %v", got)
	}
	defects := map[string]string{
		"True-Client-IP transmis":       strings.Replace(healthy, `proxy_set_header True-Client-IP "";`, "", 1),
		"X-Real-IP absent":              strings.Replace(healthy, `proxy_set_header X-Real-IP $remote_addr;`, "", 1),
		"X-Real-IP falsifiable":         strings.Replace(healthy, `X-Real-IP $remote_addr`, `X-Real-IP $http_x_real_ip`, 1),
		"héritage cassé par location":   strings.Replace(healthy, `location / { proxy_pass`, `location / { proxy_set_header Host $host; proxy_pass`, 1),
		"version exposée":               strings.Replace(healthy, "server_tokens off;", "", 1),
		"version rallumée":              strings.Replace(healthy, `location / { proxy_pass`, `location / { server_tokens on; proxy_pass`, 1),
		"corps 2g sur /api/":            strings.Replace(healthy, "client_max_body_size 1m;", "client_max_body_size 2g;", 1),
		"corps illimité":                strings.Replace(healthy, "client_max_body_size 1m;", "client_max_body_size 0;", 1),
		"corps au niveau server":        strings.Replace(healthy, "server_tokens off;", "server_tokens off; client_max_body_size 50m;", 1),
		"envoi au-delà de son plafond":  strings.Replace(healthy, "1100m", "2g", 1),
		"pas de limite sur /api/":       strings.Replace(healthy, "client_max_body_size 1m; limit_req zone=lvl_api burst=1;", "client_max_body_size 1m;", 1),
		"limite rendue en 503":          strings.Replace(healthy, "limit_req_status 429;", "", 1),
		"zone non déclarée":             strings.Replace(healthy, "zone=lvl_api burst=1; proxy_pass http://b; }\n    location / {", "zone=inconnue burst=1; proxy_pass http://b; }\n    location / {", 1),
		"aucun server 443 (vacuité)":    strings.Replace(healthy, "listen 443 ssl;", "listen 80;", 1),
		"accolade non fermée (syntaxe)": strings.TrimSuffix(healthy, "}"),
	}
	for name, conf := range defects {
		if conf == healthy {
			t.Fatalf("%s : la mutation n'a rien changé (harnais)", name)
		}
		if got := nginxConfViolations(conf, true, zones); len(got) == 0 {
			t.Errorf("%s : défaut non signalé", name)
		}
	}
}

// ─── Analyse ─────────────────────────────────────────────────────────────────────

// nginxConfViolations rend les manquements d'une configuration de site.
func nginxConfViolations(src string, prod bool, zones map[string]bool) []string {
	root, err := parseNginxConf(src)
	if err != nil {
		return []string{err.Error()}
	}
	var out []string
	tls := 0
	for _, srv := range root.all("server") {
		if !srv.listensTLS() {
			continue
		}
		tls++
		out = append(out, nginxServerViolations(srv, prod, zones)...)
	}
	if tls == 0 {
		out = append(out, "aucun bloc server n'écoute en 443")
	}
	return out
}

func nginxServerViolations(srv *ngxNode, prod bool, zones map[string]bool) []string {
	var out []string
	if d := srv.last("server_tokens"); d == nil || d.arg(0) != "off" {
		out = append(out, fmt.Sprintf("server (ligne %d) : `server_tokens off` absent", srv.line))
	}
	if size, ok := srv.bodyCap(); ok && size > nginxDefaultBodyCap {
		out = append(out, fmt.Sprintf("server (ligne %d) : client_max_body_size au-delà de 1m au niveau server", srv.line))
	}
	srv.walkLocations(func(loc *ngxNode) {
		out = append(out, nginxLocationViolations(loc, zones)...)
	})
	if prod {
		for _, uri := range nginxProdAPIPaths {
			loc := srv.selectLocation(uri)
			if loc == nil {
				out = append(out, fmt.Sprintf("%s : aucune location ne sert ce chemin", uri))
				continue
			}
			if loc.effective("limit_req") == nil {
				out = append(out, fmt.Sprintf("%s (location %q) : aucune limit_req effective", uri, loc.matcher()))
			}
			if d := loc.effective("limit_req_status"); d == nil || d.arg(0) != "429" {
				out = append(out, fmt.Sprintf("%s (location %q) : `limit_req_status 429` absent (503 par défaut)", uri, loc.matcher()))
			}
		}
	}
	return out
}

func nginxLocationViolations(loc *ngxNode, zones map[string]bool) []string {
	var out []string
	where := fmt.Sprintf("location %q (ligne %d)", loc.matcher(), loc.line)
	if d := loc.last("server_tokens"); d != nil && d.arg(0) != "off" {
		out = append(out, where+" : server_tokens rallumé")
	}
	if loc.last("proxy_pass") != nil {
		headers := loc.effectiveProxyHeaders()
		if v, ok := headers["true-client-ip"]; !ok || v != "" {
			out = append(out, where+` : proxy_set_header True-Client-IP "" absent (en-tête du client transmis)`)
		}
		if v := headers["x-real-ip"]; v != "$remote_addr" {
			out = append(out, where+" : proxy_set_header X-Real-IP $remote_addr absent")
		}
	}
	size, explicit := loc.effectiveBodyCap()
	if limit, large := nginxLargeBodyLocations[loc.matcher()]; large && explicit {
		if size > limit {
			out = append(out, fmt.Sprintf("%s : client_max_body_size %d au-delà du plafond de cette location d'envoi (%d)", where, size, limit))
		}
	} else if size > nginxDefaultBodyCap {
		out = append(out, where+" : client_max_body_size au-delà de 1m hors d'une location d'envoi nommée")
	}
	for _, d := range loc.all("limit_req") {
		zone := strings.TrimPrefix(d.arg(0), "zone=")
		if !zones[zone] {
			out = append(out, fmt.Sprintf("%s : zone %q non déclarée dans %s", where, zone, nginxRateLimitConf))
		}
	}
	return out
}

// nginxDeclaredZones lit les zones `limit_req_zone ... zone=NOM:taille` de rate_limit.conf.
func nginxDeclaredZones(dir string) (map[string]bool, error) {
	zones := map[string]bool{}
	data, err := os.ReadFile(filepath.Join(dir, nginxRateLimitConf))
	if err != nil {
		return zones, err
	}
	root, err := parseNginxConf(string(data))
	if err != nil {
		return zones, err
	}
	for _, d := range root.all("limit_req_zone") {
		for _, a := range d.args {
			if name, ok := strings.CutPrefix(a.text, "zone="); ok {
				zones[strings.SplitN(name, ":", 2)[0]] = true
			}
		}
	}
	return zones, nil
}

// nginxPackagingDir remonte depuis ce fichier jusqu'au dossier packaging/nginx du dépôt.
func nginxPackagingDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	dir := filepath.Dir(thisFile)
	for range 12 {
		candidate := filepath.Join(dir, "packaging", "nginx")
		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("packaging/nginx introuvable en remontant depuis le paquet archlint")
	return ""
}
