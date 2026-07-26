#!/usr/bin/env bash
# scripts/check_test_baseline.sh — Vérifie que la baseline de tests pré-migration
# reste verte sur la branche courante.
#
# Référence : .ai/V7/PLAN_DB_WRITE_CONCURRENCY.md §Stratégie de tests — non-régression blindée
#
# Garanties vérifiées (mode tests) — dans cet ordre :
#   1. PRÉSENCE : tout test de la baseline existe encore dans le run courant.
#   2. VERDICT : aucun test du run courant n'est en échec ("Action":"fail").
#   3. Coverage par package ne baisse pas de plus de 1 point (mode coverage).
#
# Le contrôle 2 a été ajouté le 2026-07-26 : le `|| true` sur le `go test -json`
# (nécessaire pour pouvoir analyser le JSONL même quand la suite échoue) rendait
# le gate MENTEUR — un test FAIL était compté comme « présent » par le contrôle 1
# (le filtre accepte pass|fail|skip) et le script sortait 0. Les deux contrôles
# sont volontairement distincts : ils diagnostiquent des causes différentes
# (test renommé/supprimé, ou package qui ne compile pas VS régression réelle).
#
# Sortie :
#   - exit 0 si toutes les vérifications passent.
#   - exit 1 si un test baseline manque, si un test échoue, ou si la couverture régresse.
#   - exit 2 si la baseline elle-même est introuvable (mauvais chemin / branche).
#
# Diagnostic : le JSONL du run courant est conservé (NON supprimé à la sortie) dans
# apps/go-api/baseline_current.jsonl — ignoré par git, uploadé en artefact par la CI
# (job go-baseline-tests, step « Upload JSONL du run courant », if: failure()).
#
# Usage :
#   bash scripts/check_test_baseline.sh           # check complet (tests + coverage)
#   bash scripts/check_test_baseline.sh tests     # tests uniquement
#   bash scripts/check_test_baseline.sh coverage  # coverage uniquement

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BASELINE_DIR="$REPO_ROOT/.ai/baselines"
BASELINE_TESTS="$BASELINE_DIR/tests_pre_migration.jsonl"
BASELINE_COV_TXT="$BASELINE_DIR/coverage_pre_migration.txt"
BASELINE_COV_RAW="$BASELINE_DIR/coverage_pre_migration.raw"

# JSONL du run courant — chemin STABLE (pas un mktemp détruit à la sortie) : sans
# lui, un échec du gate en CI n'était pas diagnosticable (aucune trace des tests
# en cause). Ignoré par git (.gitignore), uploadé en artefact par la CI si échec.
CURRENT_TESTS_JSONL="$REPO_ROOT/apps/go-api/baseline_current.jsonl"

MODE="${1:-all}"

# extract_test_names — parse les noms PASS/FAIL/SKIP depuis go test -json.
# Format JSON par ligne : {"Action":"pass","Package":"...","Test":"..."}
# Le champ Test est absent pour les events package-level → on les filtre.
# NB : ce filtre répond à « le test existe-t-il ? », PAS à « passe-t-il ? » —
# le verdict pass/fail est le rôle de extract_failed_test_names ci-dessous.
extract_test_names() {
  local file="$1"
  grep -hE '"Action":"(pass|fail|skip)"' "$file" 2>/dev/null \
    | sed -n 's/.*"Package":"\([^"]*\)".*"Test":"\([^"]*\)".*/\1::\2/p' \
    | sort -u
}

# extract_failed_test_names — tests en ÉCHEC du run courant.
# Le champ Test est absent des events "fail" package-level (échec de compilation,
# panic hors test) → ils ne remontent pas ici : ce cas se manifeste par des tests
# baseline « absents », d'où le message d'aide du contrôle de présence.
extract_failed_test_names() {
  local file="$1"
  grep -hE '"Action":"fail"' "$file" 2>/dev/null \
    | sed -n 's/.*"Package":"\([^"]*\)".*"Test":"\([^"]*\)".*/\1::\2/p' \
    | sort -u
}

check_tests() {
  if [[ ! -f "$BASELINE_TESTS" ]]; then
    echo "❌ Baseline tests introuvable : $BASELINE_TESTS"
    echo "   La baseline doit être capturée au commit 0 de cette branche."
    return 2
  fi

  echo "▶ Comparaison de la suite courante vs baseline"
  echo "  Baseline : $BASELINE_TESTS"

  # Chemin STABLE conservé après exécution (cf. en-tête) : aucune trap de cleanup
  # ici. L'ancien mktemp + `trap rm` détruisait la seule pièce permettant de
  # diagnostiquer un échec (incident : gate rouge en CI sans trace exploitable).
  local current_jsonl="$CURRENT_TESTS_JSONL"
  mkdir -p "$(dirname "$current_jsonl")"
  echo "  JSONL du run courant : $current_jsonl"

  echo "  Lancement de la suite courante (peut prendre plusieurs minutes)..."
  (
    cd "$REPO_ROOT/apps/go-api"
    # Sur Windows local, utiliser le gcc fourni par msys64 (chemin POSIX via
    # git-bash). Sur Linux CI, utiliser le gcc système (déjà dans le PATH).
    if [[ -f /c/msys64/ucrt64/bin/gcc.exe ]]; then
      export PATH="/c/msys64/ucrt64/bin:$PATH"
      export CC="/c/msys64/ucrt64/bin/gcc.exe"
    fi
    CGO_ENABLED=1 \
      go test -tags=integration -count=1 -timeout=300s -p 1 -json ./... > "$current_jsonl" 2>&1
  ) || true   # verdict rendu par l'ANALYSE du JSONL (présence + échecs), pas par ce code retour

  local baseline_tests current_tests
  baseline_tests=$(extract_test_names "$BASELINE_TESTS")
  current_tests=$(extract_test_names "$current_jsonl")

  local baseline_count current_count
  baseline_count=$(printf '%s\n' "$baseline_tests" | grep -c . || echo 0)
  current_count=$(printf '%s\n' "$current_tests" | grep -c . || echo 0)

  echo "  Baseline : $baseline_count tests"
  echo "  Courant  : $current_count tests"

  local missing
  missing=$(comm -23 <(printf '%s\n' "$baseline_tests") <(printf '%s\n' "$current_tests"))

  if [[ -n "$missing" ]]; then
    echo ""
    echo "❌ Tests baseline absents du run courant :"
    printf '%s\n' "$missing" | sed 's/^/    /'
    echo ""
    echo "  CAUSE LA PLUS FRÉQUENTE : un package qui ne COMPILE pas rend TOUS ses"
    echo "  tests absents (aucun event pass/fail/skip n'est émis) — vérifier d'abord"
    echo "  la sortie de compilation dans $current_jsonl (lignes \"Action\":\"output\")."
    echo "  Sinon : si ces tests ont été renommés/supprimés volontairement,"
    echo "  documenter la raison dans le commit message."
    return 1
  fi

  echo "✅ Tous les tests baseline présents dans le run courant"

  # Contrôle de VERDICT (distinct de la présence) : un test peut exister et échouer.
  # NB : on teste la chaîne AVANT de compter (comme pour "$missing") — `grep -c`
  # sort 1 sur zéro match, et le `|| echo 0` de repli produirait alors deux lignes
  # ("0\n0"), inexploitable dans une comparaison arithmétique.
  local failed
  # `|| true` OBLIGATOIRE : sans aucun test en échec (le cas NOMINAL), le `grep`
  # de la fonction sort 1 ; sous `set -e` + `pipefail` l'affectation ferait sortir
  # le script en erreur — un run 100 % vert aurait rendu le gate rouge.
  failed=$(extract_failed_test_names "$current_jsonl" || true)

  if [[ -n "$failed" ]]; then
    local failed_count
    failed_count=$(printf '%s\n' "$failed" | wc -l | tr -d '[:space:]')
    echo ""
    echo "[ECHEC] $failed_count test(s) en échec dans le run courant (20 premiers) :"
    printf '%s\n' "$failed" | head -20 | sed 's/^/    /'
    if [[ "$failed_count" -gt 20 ]]; then
      echo "    ... ($((failed_count - 20)) autres — liste complète dans $current_jsonl)"
    fi
    echo ""
    echo "  Ces tests EXISTENT (donc pas un problème de renommage) mais ne passent"
    echo "  plus : régression fonctionnelle à corriger avant livraison."
    return 1
  fi

  echo "[OK] Aucun test en échec dans le run courant"
  return 0
}

check_coverage() {
  if [[ ! -f "$BASELINE_COV_TXT" ]]; then
    echo "⚠️  Coverage baseline absent : $BASELINE_COV_TXT (skip)"
    return 0
  fi

  echo "▶ Comparaison de la couverture vs baseline"

  # Note: globaux volontairement (pas `local`) — la trap EXIT s'exécute APRÈS
  # le retour de la fonction (cf. check_tests pour le même pattern).
  current_raw=$(mktemp --suffix=.raw)
  current_txt=$(mktemp --suffix=.txt)
  trap 'rm -f "$current_raw" "$current_txt"' EXIT

  (
    cd "$REPO_ROOT/apps/go-api"
    # Sur Windows local, utiliser le gcc fourni par msys64. Sur Linux CI,
    # utiliser le gcc système (déjà dans le PATH).
    if [[ -f /c/msys64/ucrt64/bin/gcc.exe ]]; then
      export PATH="/c/msys64/ucrt64/bin:$PATH"
      export CC="/c/msys64/ucrt64/bin/gcc.exe"
    fi
    CGO_ENABLED=1 \
      go test -tags=integration -count=1 -timeout=300s -p 1 \
        -coverprofile="$current_raw" -covermode=atomic -coverpkg=./... ./... > /dev/null 2>&1
    go tool cover -func="$current_raw" > "$current_txt"
  ) || {
    echo "❌ Échec capture coverage courant"
    return 1
  }

  local baseline_pct current_pct
  baseline_pct=$(awk '/^total:/ { gsub("%",""); print $3 }' "$BASELINE_COV_TXT")
  current_pct=$(awk '/^total:/ { gsub("%",""); print $3 }' "$current_txt")

  echo "  Baseline : ${baseline_pct}%"
  echo "  Courant  : ${current_pct}%"

  awk -v c="$current_pct" -v b="$baseline_pct" 'BEGIN {
    if (c + 1.0 < b) {
      printf "❌ Coverage %.1f%% < baseline %.1f%% - 1.0 (régression > 1 point)\n", c, b
      exit 1
    }
    printf "✅ Coverage %.1f%% >= baseline %.1f%% - 1.0\n", c, b
    exit 0
  }'
}

case "$MODE" in
  tests)
    check_tests
    ;;
  coverage)
    check_coverage
    ;;
  all)
    check_tests || exit $?
    check_coverage || exit $?
    ;;
  *)
    echo "Usage: $0 [tests|coverage|all]"
    exit 2
    ;;
esac
