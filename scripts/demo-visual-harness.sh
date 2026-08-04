#!/usr/bin/env bash
# =============================================================================
# demo-visual-harness.sh — harnais de régression visuelle sur FIXTURE DÉMO
#
# POURQUOI. `apps/web/e2e/visual/app-pages.visual.spec.ts` vise `demo-player` par
# défaut, mais SKIPPE toute page sans graphe : sans fixture démo peuplée, les 7
# pages applicatives sautaient et les captures se prenaient en pratique sur le
# joueur RÉEL du poste (`E2E_VISUAL_PLAYER=JGtm`). Or ces données bougent à chaque
# sync : deux passes consécutives divergeaient sans qu'aucune régression de rendu
# n'existe, ce qui rend le harnais inexploitable comme référence AVANT/APRÈS.
#
# CE QUE FAIT CE SCRIPT. Il enchaîne les trois étapes qui rendent le harnais
# déterministe, sur une racine de données ISOLÉE (jamais `data/` du poste) :
#   1. génère la fixture démo synthétique (`levelup seed-demo --synthetic`) ;
#   2. démarre l'API en mode démo sur cette racine + le serveur Vite ;
#   3. lance le projet Playwright `visual` avec E2E_VISUAL_PLAYER=demo-player.
# Les serveurs démarrés ici sont arrêtés à la sortie (trap), y compris sur erreur.
#
# DÉTERMINISME. Le générateur est à graine fixe et date d'ancrage fixe
# (internal/ops/seed_demo_synthetic.go) : deux générations produisent les mêmes
# LIGNES (les fichiers .duckdb diffèrent au bit près — agencement de blocs interne
# et `written_at` DEFAULT now() — mais aucune donnée lue par l'app n'en dépend).
# D'où le critère de succès : régénérer puis rejouer les captures doit donner zéro
# diff pixel. Vérification : `--verify-determinism`.
#
# USAGE
#   bash scripts/demo-visual-harness.sh                      # génère + sert + compare
#   bash scripts/demo-visual-harness.sh --update-snapshots   # (re)génère les baselines
#   bash scripts/demo-visual-harness.sh --skip-seed          # réutilise la fixture existante
#   bash scripts/demo-visual-harness.sh --verify-determinism # régénère + rejoue, exige 0 diff
#   make demo-visual                                         # équivalent de la 1re forme
#
# VARIABLES
#   DEMO_ROOT   racine de la fixture (défaut : tests/fixtures/demo-root, gitignorée)
#   API_PORT    port de l'API démo (défaut 8000)
#   WEB_PORT    port Vite (défaut 5173 — l'en-tête Origin des specs y est adossé)
#
# BASELINES. Celles des pages applicatives ne sont PAS versionnées
# (`__screenshots__/app/` est gitignoré) : ce sont des références locales
# AVANT/APRÈS. Ce script ne change pas cette politique.
# =============================================================================
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

# Chemin natif de la plateforme : les binaires Go sont des exécutables Windows
# sous Git Bash et n'interprètent pas les chemins POSIX `/c/...`. Même bascule
# que le Makefile (`pwd -W 2>/dev/null || pwd`).
native_path() { (cd "$1" && { pwd -W 2>/dev/null || pwd; }); }

DEMO_ROOT="${DEMO_ROOT:-$REPO_ROOT/tests/fixtures/demo-root}"
API_PORT="${API_PORT:-8000}"
WEB_PORT="${WEB_PORT:-5173}"
GO_API_DIR="$REPO_ROOT/apps/go-api"
BIN_DIR="$GO_API_DIR/bin" # gitignoré
EXE_SUFFIX=""
case "$(uname -s 2>/dev/null || echo unknown)" in
MINGW* | MSYS* | CYGWIN*) EXE_SUFFIX=".exe" ;;
esac

SKIP_SEED=0
VERIFY_DETERMINISM=0
PW_ARGS=()
for arg in "$@"; do
	case "$arg" in
	--skip-seed) SKIP_SEED=1 ;;
	--verify-determinism) VERIFY_DETERMINISM=1 ;;
	*) PW_ARGS+=("$arg") ;;
	esac
done

# Logs des serveurs. PAS sous tests/e2e-results/ : c'est l'`outputDir` de
# Playwright, qui VIDE ce dossier au démarrage de chaque run — les logs
# disparaissaient au moment précis où on en avait besoin (diagnostic d'un échec).
LOG_DIR="$REPO_ROOT/tests/demo-harness-logs" # gitignoré
mkdir -p "$LOG_DIR"

say() { printf '\n  [demo-visual] %s\n' "$*"; }
fail() {
	printf '\n  [demo-visual] ERREUR : %s\n' "$*" >&2
	exit 1
}

# ── Garde-fous : ne jamais écrire dans les données du poste ───────────────────
case "$DEMO_ROOT" in
"$REPO_ROOT/data" | "$REPO_ROOT/data/"*)
	fail "DEMO_ROOT ($DEMO_ROOT) pointe sous data/ — la fixture doit vivre dans une racine ISOLÉE."
	;;
esac

# ── Ports : refuser de tuer un serveur dev déjà en place ──────────────────────
# Vite écoute sur `localhost` — qui résout en ::1 avant 127.0.0.1 sous Windows :
# sonder l'IPv4 littérale renvoie « connection refused » alors que le serveur est
# bien prêt. On sonde donc le WEB par nom d'hôte, l'API (bind 127.0.0.1) par IP.
web_up() { curl -sf "http://localhost:$WEB_PORT" >/dev/null 2>&1; }
api_up() { curl -sf "http://127.0.0.1:$API_PORT/health" >/dev/null 2>&1; }
if api_up; then
	fail "un serveur répond déjà sur :$API_PORT/health (make dev ?). L'arrêter (make stop) ou définir API_PORT."
fi
if web_up; then
	fail "un serveur répond déjà sur :$WEB_PORT (Vite ?). L'arrêter (make stop) ou définir WEB_PORT."
fi

API_PID=""
WEB_PID=""

# Kill par PORT — npm et le shell lancent des petits-enfants (vite, levelup-api) :
# tuer le PID du sous-shell ne suffit pas. Même approche que la cible `make stop`.
kill_port() {
	if command -v powershell >/dev/null 2>&1; then
		powershell -NoProfile -Command \
			"Get-NetTCPConnection -LocalPort $1 -State Listen -ErrorAction SilentlyContinue | ForEach-Object { Stop-Process -Id \$_.OwningProcess -Force -ErrorAction SilentlyContinue }; exit 0" >/dev/null 2>&1 || true
	else
		local pids
		pids="$(lsof -ti "tcp:$1" 2>/dev/null || true)"
		[ -n "$pids" ] && kill -9 $pids 2>/dev/null || true
	fi
}

cleanup() {
	local code=$?
	[ -n "$API_PID" ] && kill "$API_PID" 2>/dev/null || true
	[ -n "$WEB_PID" ] && kill "$WEB_PID" 2>/dev/null || true
	kill_port "$API_PORT"
	kill_port "$WEB_PORT"
	[ "$code" -eq 0 ] && say "serveurs arrêtés." || say "serveurs arrêtés (sortie $code)."
	return $code
}
trap cleanup EXIT INT TERM

# ── 1. Build des binaires Go (CGO obligatoire : DuckDB est un binding C) ──────
say "compilation des binaires Go (CGO)..."
mkdir -p "$BIN_DIR"
(cd "$GO_API_DIR" && CGO_ENABLED=1 go build -o "bin/levelup$EXE_SUFFIX" ./cmd/levelup/)
(cd "$GO_API_DIR" && CGO_ENABLED=1 go build -o "bin/levelup-api$EXE_SUFFIX" ./cmd/server/)

# ── 2. Génération de la fixture démo synthétique ──────────────────────────────
seed_fixture() {
	rm -rf "$DEMO_ROOT"
	mkdir -p "$DEMO_ROOT"
	LEVELUP_REPO_ROOT="$(native_path "$REPO_ROOT")" \
		"$BIN_DIR/levelup$EXE_SUFFIX" seed-demo --synthetic --out "$(native_path "$DEMO_ROOT")"
}

if [ "$SKIP_SEED" -eq 1 ]; then
	[ -f "$DEMO_ROOT/players/DEMO/stats.duckdb" ] ||
		fail "--skip-seed mais aucune fixture dans $DEMO_ROOT — relancer sans le drapeau."
	say "fixture réutilisée : $DEMO_ROOT"
else
	say "génération de la fixture démo synthétique dans $DEMO_ROOT ..."
	seed_fixture
fi

# ── 3. Démarrage de l'API démo sur la racine isolée ───────────────────────────
# LEVELUP_DEMO_MODE + DEMO_FIXTURES_DIR : resolveDemoPlayer (internal/config) lit
# {dir}/players/DEMO/stats.duckdb + {dir}/warehouse/*.duckdb. DB_PROFILES et
# APP_SETTINGS pointent les configs ÉMISES par le seed (parité avec le service
# levelup-demo de docker-compose.yml, qui les bind-mount par-dessus celles du
# repo). LEVELUP_DEMO_LOCALE=fr : la vitrine démo force l'anglais par défaut, les
# specs vérifient l'UI FR.
start_api() {
	(
		cd "$GO_API_DIR"
		LEVELUP_REPO_ROOT="$(native_path "$REPO_ROOT")" \
			LEVELUP_DEMO_MODE=true \
			LEVELUP_DEMO_FIXTURES_DIR="$(native_path "$DEMO_ROOT")" \
			LEVELUP_DEMO_LOCALE=fr \
			LEVELUP_DB_PROFILES="$(native_path "$DEMO_ROOT")/db_profiles.json" \
			LEVELUP_APP_SETTINGS="$(native_path "$DEMO_ROOT")/app_settings.json" \
			LEVELUP_API_PORT="$API_PORT" \
			"./bin/levelup-api$EXE_SUFFIX"
	) >>"$LOG_DIR/demo-api.log" 2>&1 &
	API_PID=$!
	for _ in $(seq 1 60); do
		api_up && break
		sleep 1
	done
	api_up || fail "l'API démo n'a pas démarré — voir $LOG_DIR/demo-api.log"
}

# L'API tient les DuckDB de la fixture ouvertes : elle doit être arrêtée avant
# toute régénération. Kill par PORT — le binaire est un petit-fils du shell
# (sous-shell -> exe), donc tuer $API_PID seul le laisserait tourner.
stop_api() {
	kill "$API_PID" 2>/dev/null || true
	kill_port "$API_PORT"
	API_PID=""
	for _ in $(seq 1 30); do
		api_up || return 0
		sleep 1
	done
	fail "l'API démo refuse de s'arrêter sur :$API_PORT"
}

say "démarrage de l'API démo sur :$API_PORT ..."
: >"$LOG_DIR/demo-api.log"
start_api

# Discriminant de peuplement : 404 = joueur démo non résolu (cf. e2e/_helpers/demoData.ts).
probe="$(curl -s -o /dev/null -w '%{http_code}' -H "Origin: http://localhost:$WEB_PORT" \
	"http://127.0.0.1:$API_PORT/api/v1/healthz/home?player=demo-player")"
[ "$probe" = "404" ] && fail "demo-player non résolu (HTTP 404) — fixture incomplète dans $DEMO_ROOT"
say "demo-player résolu (healthz/home HTTP $probe)."

# ── 4. Démarrage du serveur Vite ──────────────────────────────────────────────
say "démarrage de Vite sur :$WEB_PORT ..."
(
	cd "$REPO_ROOT/apps/web"
	VITE_API_PROXY_TARGET="http://127.0.0.1:$API_PORT" npm run dev -- --port "$WEB_PORT" --strictPort
) >"$LOG_DIR/demo-web.log" 2>&1 &
WEB_PID=$!

for _ in $(seq 1 60); do
	web_up && break
	sleep 1
done
web_up || fail "Vite n'a pas démarré — voir $LOG_DIR/demo-web.log"

# ── 5. Harnais visuel ─────────────────────────────────────────────────────────
# GARDE-FOU sur --update-snapshots : le projet `visual` contient AUSSI
# `lab-charts.visual.spec.ts`, dont les baselines sont VERSIONNÉES (données
# statiques, référence partagée). Une régénération globale les réécrivait en
# silence — 5 PNG committés modifiés sans rapport avec la démo. On restreint donc
# toute écriture de baselines au seul fichier de pages applicatives (baselines
# non versionnées). Régénérer la vitrine reste possible, explicitement :
#   cd apps/web && npx playwright test --project=visual lab-charts --update-snapshots
run_visual() {
	local scope=()
	for a in "$@"; do
		[ "$a" = "--update-snapshots" ] && scope=(app-pages.visual.spec.ts)
	done
	(
		cd "$REPO_ROOT/apps/web"
		E2E_VISUAL_PLAYER=demo-player \
			E2E_BASE_URL="http://localhost:$WEB_PORT" \
			E2E_API_URL="http://127.0.0.1:$API_PORT" \
			E2E_SYNTHETIC_DEMO=1 \
			npx playwright test --project=visual "${scope[@]+"${scope[@]}"}" "$@"
	)
}

if [ "$VERIFY_DETERMINISM" -eq 0 ]; then
	say "exécution du projet Playwright 'visual' (E2E_VISUAL_PLAYER=demo-player)..."
	run_visual "${PW_ARGS[@]+"${PW_ARGS[@]}"}"
	exit 0
fi

# ── 5bis. Preuve de déterminisme ──────────────────────────────────────────────
# Passe 1 : baselines fraîches sur la fixture courante. Puis on RÉGÉNÈRE la
# fixture de zéro (nouvelles DuckDB, nouveaux written_at) et on rejoue la
# comparaison : toute diff pixel signalerait une donnée non déterministe.
say "[déterminisme 1/3] génération des baselines sur la fixture courante..."
run_visual --update-snapshots

say "[déterminisme 2/3] régénération complète de la fixture (2e génération)..."
stop_api
seed_fixture
start_api

say "[déterminisme 3/3] comparaison aux baselines de la passe 1 (0 diff attendu)..."
run_visual
say "DÉTERMINISME VÉRIFIÉ : deux générations + deux passes, zéro diff pixel."
