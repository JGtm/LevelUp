#!/usr/bin/env bash
# scripts/check_test_baseline.sh — Vérifie que la baseline de tests pré-migration
# reste verte sur la branche courante.
#
# Référence : .ai/V7/PLAN_DB_WRITE_CONCURRENCY.md §Stratégie de tests — non-régression blindée
#
# Garanties vérifiées (mode tests) — dans cet ordre :
#   1. PRÉSENCE : tout test de la baseline existe encore dans le run courant.
#   2. VERDICT TEST : aucun test du run courant n'est en échec ("Action":"fail" + "Test").
#   3. VERDICT PACKAGE : aucun package en échec sans test en échec (compilation,
#      panic hors test, TestMain non-zéro).
#   4. Coverage par package ne baisse pas de plus de 1 point (mode coverage).
#
# Le contrôle 2 a été ajouté le 2026-07-26 : le `|| true` sur le `go test -json`
# (nécessaire pour pouvoir analyser le JSONL même quand la suite échoue) rendait
# le gate MENTEUR — un test FAIL était compté comme « présent » par le contrôle 1
# (le filtre accepte pass|fail|skip) et le script sortait 0. Les deux contrôles
# sont volontairement distincts : ils diagnostiquent des causes différentes
# (test renommé/supprimé, ou package qui ne compile pas VS régression réelle).
#
# Le contrôle 3 a été ajouté le 2026-07-26 avec le mode consommateur : quand ce
# script est le SEUL juge d'un run (CI, cf. ci-dessous), un package NOUVEAU qui ne
# compile pas passait entre les mailles — il n'a aucun test en baseline (contrôle 1
# aveugle) et un échec de compilation n'émet pas d'event fail test-level
# (contrôle 2 aveugle). Le seul signal restant est l'event fail PACKAGE-level.
#
# DEUX MODES (le code de vérification est le MÊME — verify_tests_jsonl) :
#   - AUTONOME (défaut) : le script lance lui-même la suite. C'est le mode du
#     filet local `make gate-push`.
#   - CONSOMMATEUR (`--from-jsonl <fichier>`) : la suite a DÉJÀ tourné en amont,
#     le script ne fait que vérifier son JSONL. C'est le mode de la CI : le job
#     `go-coverage` produit `-json` + `-coverprofile` en UNE exécution, au lieu
#     des deux runs complets (~22 min chacun) de l'ancien couple de jobs
#     go-baseline-tests / go-coverage (dédup 2026-07-26).
#
# Sortie :
#   - exit 0 si toutes les vérifications passent.
#   - exit 1 si un test baseline manque, si un test/package échoue, ou si la couverture régresse.
#   - exit 2 si la baseline elle-même est introuvable, si le JSONL fourni est
#     absent/vide, ou si les arguments sont invalides.
#
# Diagnostic : en mode autonome le JSONL du run courant est conservé (NON supprimé
# à la sortie) dans apps/go-api/baseline_current.jsonl — ignoré par git, uploadé en
# artefact par la CI (job go-coverage, step « Upload JSONL du run courant »,
# if: failure()). En cas d'échec, les lignes de sortie humaine des tests fautifs
# sont extraites du JSONL et affichées : sous `-json` le log de la CI ne contient
# plus aucune sortie lisible, il ne faut pas avoir à télécharger l'artefact pour
# un simple test rouge.
#
# Usage :
#   bash scripts/check_test_baseline.sh                              # tests + coverage (lance 2 suites)
#   bash scripts/check_test_baseline.sh tests                        # tests uniquement (lance la suite)
#   bash scripts/check_test_baseline.sh coverage                     # coverage uniquement (lance la suite)
#   bash scripts/check_test_baseline.sh tests --from-jsonl <fichier> # vérifie un JSONL déjà produit

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BASELINE_DIR="$REPO_ROOT/.ai/baselines"
BASELINE_TESTS="$BASELINE_DIR/tests_pre_migration.jsonl"
BASELINE_COV_TXT="$BASELINE_DIR/coverage_pre_migration.txt"
BASELINE_COV_RAW="$BASELINE_DIR/coverage_pre_migration.raw"

# JSONL du run courant (mode autonome) — chemin STABLE (pas un mktemp détruit à la
# sortie) : sans lui, un échec du gate en CI n'était pas diagnosticable (aucune
# trace des tests en cause). Ignoré par git (.gitignore), uploadé en artefact par
# la CI si échec.
CURRENT_TESTS_JSONL="$REPO_ROOT/apps/go-api/baseline_current.jsonl"

usage() {
  echo "Usage: $0 [tests|coverage|all] [--from-jsonl <fichier>]"
  echo "  tests                    présence + verdict de la suite (baseline de non-régression)"
  echo "  coverage                 couverture globale vs baseline"
  echo "  all (défaut)             les deux"
  echo "  --from-jsonl <fichier>   mode consommateur : ne relance PAS la suite, vérifie"
  echo "                           le JSONL 'go test -json' fourni (mode tests uniquement)"
}

MODE="all"
FROM_JSONL=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    tests | coverage | all)
      MODE="$1"
      shift
      ;;
    --from-jsonl)
      if [[ $# -lt 2 ]]; then
        echo "[ECHEC] --from-jsonl attend un chemin de fichier"
        usage
        exit 2
      fi
      FROM_JSONL="$2"
      shift 2
      ;;
    --from-jsonl=*)
      FROM_JSONL="${1#*=}"
      shift
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      echo "[ECHEC] Argument inconnu : $1"
      usage
      exit 2
      ;;
  esac
done

# Garde d'usage : le mode coverage relance NÉCESSAIREMENT une suite (il a besoin
# d'un profil de couverture, pas d'un JSONL). Accepter --from-jsonl avec coverage
# ferait croire à une vérification sans exécution alors que la plus longue des
# deux tournerait quand même.
if [[ -n "$FROM_JSONL" && "$MODE" != "tests" ]]; then
  echo "[ECHEC] --from-jsonl n'est valide qu'avec le mode 'tests' (mode reçu : '$MODE')."
  echo "        Le mode coverage doit produire son propre profil de couverture."
  echo "        Écrire : $0 tests --from-jsonl $FROM_JSONL"
  exit 2
fi

require_baseline_tests() {
  if [[ ! -f "$BASELINE_TESTS" ]]; then
    echo "[ECHEC] Baseline tests introuvable : $BASELINE_TESTS"
    echo "   La baseline doit être capturée au commit 0 de cette branche."
    return 2
  fi
  return 0
}

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
# panic hors test) → ils ne remontent pas ici : ce cas est couvert par
# extract_failed_packages (contrôle 3).
extract_failed_test_names() {
  local file="$1"
  grep -hE '"Action":"fail"' "$file" 2>/dev/null \
    | sed -n 's/.*"Package":"\([^"]*\)".*"Test":"\([^"]*\)".*/\1::\2/p' \
    | sort -u
}

# extract_failed_packages — packages en échec SANS aucun test en échec.
# go test émet un event fail PACKAGE-level (sans champ Test) pour tout package
# rouge ; quand ce package n'a par ailleurs AUCUN event fail test-level, la cause
# est ailleurs que dans un test : compilation impossible, panic hors test,
# TestMain qui sort non-zéro. C'est le seul signal disponible pour un package
# absent de la baseline (les contrôles 1 et 2 sont aveugles, cf. en-tête).
extract_failed_packages() {
  local file="$1"
  local pkg_failed pkg_with_failed_test
  pkg_failed=$(
    { grep -hE '"Action":"fail"' "$file" 2>/dev/null || true; } \
      | grep -v '"Test":"' \
      | sed -n 's/.*"Package":"\([^"]*\)".*/\1/p' \
      | sed '/^$/d' | sort -u
  ) || true
  pkg_with_failed_test=$(
    { grep -hE '"Action":"fail"' "$file" 2>/dev/null || true; } \
      | grep '"Test":"' \
      | sed -n 's/.*"Package":"\([^"]*\)".*/\1/p' \
      | sed '/^$/d' | sort -u
  ) || true
  comm -23 <(printf '%s\n' "$pkg_failed" | sed '/^$/d') \
    <(printf '%s\n' "$pkg_with_failed_test" | sed '/^$/d')
}

# print_failure_output — lignes de SORTIE HUMAINE (max $2) des tests/packages en
# échec, reconstituées depuis les events output du JSONL.
# POURQUOI : sous `-json` (le seul format exploitable par ce script), le log de la
# CI ne contient plus une seule ligne lisible — un test rouge obligeait à
# télécharger l'artefact JSONL de plusieurs Mo pour savoir CE qui avait cassé.
#
# TROIS familles d'events sont nécessaires (vérifié sur du go test -json réel,
# toolchain 1.26) — les deux dernières portent `ImportPath` et AUCUN champ
# `Package`, donc un extracteur qui ne regarde que `Package` rate exactement le cas
# le plus fréquent, l'erreur de compilation :
#   - {"Action":"output","Package":..,"Test":..}  sortie d'un test ;
#   - {"Action":"output","Package":..}            sortie package-level ;
#   - {"Action":"build-output","ImportPath":..}   sortie du compilateur, rattachée
#     au package via {"Action":"build-fail","ImportPath":..} et/ou le champ
#     "FailedBuild" de l'event fail package-level.
# Désescape best-effort (\n \r \t \" \\ uniquement) : c'est un extrait de
# diagnostic, la source de vérité reste le JSONL.
print_failure_output() {
  local file="$1"
  local max="${2:-20}"
  awk -v max="$max" '
    function jfield(line, name,    pat) {
      pat = "\"" name "\":\""
      if (match(line, pat "[^\"]*\"") == 0) return ""
      return substr(line, RSTART + length(pat), RLENGTH - length(pat) - 1)
    }
    function emit(line,    out) {
      out = line
      sub(/^.*"Output":"/, "", out)
      sub(/"[[:space:]]*}[[:space:]]*$/, "", out)
      gsub(/\\n/, "", out)
      gsub(/\\r/, "", out)
      gsub(/\\t/, "  ", out)
      gsub(/\\"/, "\"", out)
      gsub(/\\\\/, "\\", out)
      if (out ~ /^[[:space:]]*$/) return 0
      print "    " out
      return 1
    }
    # 1re passe : mémoriser tests, packages et builds en échec.
    NR == FNR {
      if (index($0, "\"Action\":\"build-fail\"") > 0) {
        ip = jfield($0, "ImportPath")
        if (ip != "") failed_build[ip] = 1
      } else if (index($0, "\"Action\":\"fail\"") > 0) {
        t = jfield($0, "Test")
        if (t != "") failed_test[jfield($0, "Package") "::" t] = 1
        else failed_pkg[jfield($0, "Package")] = 1
        fb = jfield($0, "FailedBuild")
        if (fb != "") failed_build[fb] = 1
      }
      next
    }
    # 2e passe : réémettre leurs lignes de sortie, dans l ordre du run.
    printed >= max { exit }
    {
      if (index($0, "\"Action\":\"build-output\"") > 0) {
        if (!(jfield($0, "ImportPath") in failed_build)) next
        printed += emit($0)
        next
      }
      if (index($0, "\"Action\":\"output\"") == 0) next
      p = jfield($0, "Package")
      t = jfield($0, "Test")
      if (t != "") {
        if (!((p "::" t) in failed_test)) next
      } else if (!(p in failed_pkg)) next
      printed += emit($0)
    }
    END {
      if (printed == 0) print "    (aucune ligne de sortie associée dans le JSONL)"
    }
  ' "$file" "$file"
}

# run_current_suite — mode AUTONOME : exécute la suite complète et écrit le JSONL.
# `-timeout=300s` : budget par package d'un run NON instrumenté. La CI, elle,
# instrumente la couverture dans la même exécution et utilise 600s (cf. ci.yml).
run_current_suite() {
  local out="$1"
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
      go test -tags=integration -count=1 -timeout=300s -p 1 -json ./... > "$out" 2>&1
  ) || true # verdict rendu par l'ANALYSE du JSONL (présence + échecs), pas par ce code retour
}

# verify_tests_jsonl — CŒUR DE VÉRIFICATION, partagé par les deux modes (autonome
# et consommateur). Toute évolution des contrôles se fait ICI et nulle part
# ailleurs : deux implémentations divergeraient (le mode local et le gate CI ne
# rendraient plus le même verdict, ce qui est précisément la panne qu'on évite).
verify_tests_jsonl() {
  local current_jsonl="$1"

  local baseline_tests current_tests
  baseline_tests=$(extract_test_names "$BASELINE_TESTS")
  current_tests=$(extract_test_names "$current_jsonl")

  local baseline_count current_count
  # `|| true` et NON `|| echo 0` : sur zéro test, `grep -c .` imprime DÉJÀ « 0 »
  # puis sort 1 ; le repli ajoutait une SECONDE ligne « 0 » et l'affichage devenait
  # « Baseline : 0\n0 tests ». Le cas se produit quand le run n'a rien émis — celui
  # où l'opérateur a le plus besoin d'un compte lisible.
  baseline_count=$(printf '%s\n' "$baseline_tests" | grep -c . || true)
  current_count=$(printf '%s\n' "$current_tests" | grep -c . || true)

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
    echo "  Sortie des tests en échec (extrait, 20 lignes max) :"
    print_failure_output "$current_jsonl" 20
    echo ""
    echo "  Ces tests EXISTENT (donc pas un problème de renommage) mais ne passent"
    echo "  plus : régression fonctionnelle à corriger avant livraison."
    return 1
  fi

  echo "[OK] Aucun test en échec dans le run courant"

  # Contrôle de VERDICT PACKAGE (cf. en-tête, contrôle 3).
  local failed_pkgs
  failed_pkgs=$(extract_failed_packages "$current_jsonl" || true)

  if [[ -n "$failed_pkgs" ]]; then
    echo ""
    echo "[ECHEC] Package(s) en échec SANS aucun test en échec (10 premiers) :"
    printf '%s\n' "$failed_pkgs" | head -10 | sed 's/^/    /'
    echo ""
    echo "  Sortie associée (extrait, 20 lignes max) :"
    print_failure_output "$current_jsonl" 20
    echo ""
    echo "  Signature d'une compilation impossible, d'un panic hors test ou d'un"
    echo "  TestMain qui sort non-zéro — PAS d'une assertion. Aucun test n'a pu"
    echo "  rendre de verdict pour ce(s) package(s)."
    return 1
  fi

  echo "[OK] Aucun package en échec hors test"
  return 0
}

# check_tests — mode AUTONOME : lance la suite puis la vérifie.
check_tests() {
  require_baseline_tests || return $?

  echo "[BASELINE] Comparaison de la suite courante vs baseline"
  echo "  Baseline : $BASELINE_TESTS"

  # Chemin STABLE conservé après exécution (cf. en-tête) : aucune trap de cleanup
  # ici. L'ancien mktemp + `trap rm` détruisait la seule pièce permettant de
  # diagnostiquer un échec (incident : gate rouge en CI sans trace exploitable).
  local current_jsonl="$CURRENT_TESTS_JSONL"
  mkdir -p "$(dirname "$current_jsonl")"
  echo "  JSONL du run courant : $current_jsonl"

  run_current_suite "$current_jsonl"
  verify_tests_jsonl "$current_jsonl"
}

# check_tests_from_jsonl — mode CONSOMMATEUR : aucune exécution, on vérifie le
# JSONL produit en amont (job CI go-coverage).
check_tests_from_jsonl() {
  local current_jsonl="$1"
  require_baseline_tests || return $?

  if [[ ! -f "$current_jsonl" ]]; then
    echo "[ECHEC] JSONL du run courant introuvable : $current_jsonl"
    echo "   Mode consommateur : le fichier doit avoir été produit en amont par un"
    echo "   « go test -json » (CI : job go-coverage, step « go test avec couverture »)."
    return 2
  fi
  if [[ ! -s "$current_jsonl" ]]; then
    echo "[ECHEC] JSONL du run courant VIDE : $current_jsonl"
    echo "   La suite n'a émis aucun event — le « go test -json » a-t-il démarré ?"
    echo "   (toolchain absente, flag invalide, redirection cassée)."
    return 2
  fi

  echo "[BASELINE] Comparaison de la suite courante vs baseline (JSONL fourni — aucune ré-exécution)"
  echo "  Baseline : $BASELINE_TESTS"
  echo "  JSONL du run courant : $current_jsonl"

  verify_tests_jsonl "$current_jsonl"
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
    if [[ -n "$FROM_JSONL" ]]; then
      check_tests_from_jsonl "$FROM_JSONL"
    else
      check_tests
    fi
    ;;
  coverage)
    check_coverage
    ;;
  all)
    check_tests || exit $?
    check_coverage || exit $?
    ;;
esac
