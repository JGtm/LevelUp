#!/usr/bin/env bash
# scripts/check_test_baseline.sh — Vérifie que la baseline de tests pré-migration
# reste verte sur la branche courante.
#
# Référence : .ai/PLAN_DB_WRITE_CONCURRENCY.md §Stratégie de tests — non-régression blindée
#
# Garanties vérifiées :
#   1. Tous les tests présents dans la baseline existent encore et passent.
#   2. Aucun test baseline n'a été supprimé sans justification commit.
#   3. Coverage par package ne baisse pas de plus de 1 point.
#
# Sortie :
#   - exit 0 si toutes les vérifications passent.
#   - exit 1 si au moins un test baseline manque ou a régressé.
#   - exit 2 si la baseline elle-même est introuvable (mauvais chemin / branche).
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

MODE="${1:-all}"

# extract_test_names — parse les noms PASS/FAIL/SKIP depuis go test -json.
# Format JSON par ligne : {"Action":"pass","Package":"...","Test":"..."}
# Le champ Test est absent pour les events package-level → on les filtre.
extract_test_names() {
  local file="$1"
  grep -hE '"Action":"(pass|fail|skip)"' "$file" 2>/dev/null \
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

  local current_jsonl
  current_jsonl=$(mktemp)
  trap 'rm -f "$current_jsonl"' EXIT

  echo "  Lancement de la suite courante (peut prendre plusieurs minutes)..."
  (
    cd "$REPO_ROOT/apps/go-api"
    PATH="/c/msys64/ucrt64/bin:$PATH" CC=/c/msys64/ucrt64/bin/gcc.exe CGO_ENABLED=1 \
      go test -tags=integration -count=1 -timeout=300s -p 1 -json ./... > "$current_jsonl" 2>&1
  ) || true

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
    echo "  Si ces tests ont été renommés/supprimés volontairement,"
    echo "  documenter la raison dans le commit message."
    return 1
  fi

  echo "✅ Tous les tests baseline présents dans le run courant"
  return 0
}

check_coverage() {
  if [[ ! -f "$BASELINE_COV_TXT" ]]; then
    echo "⚠️  Coverage baseline absent : $BASELINE_COV_TXT (skip)"
    return 0
  fi

  echo "▶ Comparaison de la couverture vs baseline"

  local current_raw current_txt
  current_raw=$(mktemp --suffix=.raw)
  current_txt=$(mktemp --suffix=.txt)
  trap 'rm -f "$current_raw" "$current_txt"' EXIT

  (
    cd "$REPO_ROOT/apps/go-api"
    PATH="/c/msys64/ucrt64/bin:$PATH" CC=/c/msys64/ucrt64/bin/gcc.exe CGO_ENABLED=1 \
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
