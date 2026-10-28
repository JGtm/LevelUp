#!/usr/bin/env bash
# scripts/ci_go_shard.sh — arguments `go test` d'un lot de la suite Go de la CI.
#
# Usage :
#   bash scripts/ci_go_shard.sh <lot>        un argument par ligne : `-timeout` et sa durée, filtre
#                                            éventuel (-run/-skip et son motif), puis les paquets
#                                            du lot (./chemin)
#   bash scripts/ci_go_shard.sh --lots       noms des lots, un par ligne
#   bash scripts/ci_go_shard.sh --verifier   prouve la partition (cf. ci-dessous) et la résume
#
# La définition des lots vit dans .github/ci/go-shards.txt (syntaxe documentée en tête de ce
# fichier). Ce script la confronte à `go list -tags=integration ./...` — la MÊME énumération que
# `go test -tags=integration ./...` — et refuse (code 1, message sur stderr) toute définition qui
# perdrait ou dupliquerait un paquet ou des tests :
#   - un motif qui ne désigne aucun paquet (lot périmé) ;
#   - un paquet dans deux lots, hors paire complémentaire `run:L` / `skip:L` ;
#   - un paquet filtré sans son complément (des tests ne tourneraient nulle part) ;
#   - un complément (`@reste`) vide, ou qui n'est pas le dernier lot.
# Le complément ne peut donc pas laisser un paquet hors de la matrice : c'est `go list` qui le
# calcule, pas une liste écrite à la main.
#
# La sortie se consomme sans passer par le shell pour la découpe :
#   out=$(bash ../../scripts/ci_go_shard.sh "$LOT") || exit 1
#   mapfile -t args <<< "$out"
#   go test ... "${args[@]}"
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEF_FILE="${CI_GO_SHARDS_FILE:-$REPO_ROOT/.github/ci/go-shards.txt}"

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <lot> | --lots | --verifier" >&2
  exit 2
fi
ARG="$1"

if [[ ! -f "$DEF_FILE" ]]; then
  echo "ci_go_shard: définition introuvable : $DEF_FILE" >&2
  exit 1
fi

# Sur Windows local, le gcc de msys64 est requis pour que `go list` voie les fichiers cgo.
if [[ -f /c/msys64/ucrt64/bin/gcc.exe ]]; then
  export PATH="/c/msys64/ucrt64/bin:$PATH"
fi

cd "$REPO_ROOT/apps/go-api"
# `go list -m` rend TOUS les modules de l'espace de travail (go.work, ADR 0041) : le chemin du
# module go-api se lit dans son go.mod. `./...` ne franchit pas la frontière d'un module, donc
# la partition ne couvre que go-api ; les briques sous libs/ ont leurs pas de CI (job go-test).
MODULE="$(awk '$1 == "module" { print $2; exit }' go.mod | tr -d '\r')"
PKG_FILE="$(mktemp)"
trap 'rm -f "$PKG_FILE"' EXIT
CGO_ENABLED=1 go list -tags=integration ./... | tr -d '\r' | sed "s#^${MODULE}/##" > "$PKG_FILE"
if [[ ! -s "$PKG_FILE" ]]; then
  echo "ci_go_shard: go list n'a rendu aucun paquet" >&2
  exit 1
fi

awk -v mode="$ARG" '
  function die(msg) {
    print "ci_go_shard: " msg > "/dev/stderr"
    failed = 1
    exit 1
  }
  # Le paquet p (chemin relatif au module) est-il désigné par le motif pat (./a/b ou ./a/b/...) ?
  function matches(pat, p,    base) {
    sub(/^\.\//, "", pat)
    if (pat == "...") return 1
    if (pat ~ /\/\.\.\.$/) {
      base = substr(pat, 1, length(pat) - 4)
      return (p == base || index(p, base "/") == 1)
    }
    return (p == pat)
  }
  FNR == NR { if ($0 != "") pk[++np] = $0; next }
  {
    sub(/#.*/, ""); sub(/\r$/, "")
    if (NF == 0) next
    if ($1 == "heavy") {
      if (NF < 3) die("heavy " $2 " sans test")
      list = ""
      for (i = 3; i <= NF; i++) {
        if ($i !~ /^[A-Za-z0-9_]+$/) die("nom de test invalide dans heavy " $2 " : " $i)
        list = list (i > 3 ? "|" : "") $i
      }
      heavy[$2] = list
      next
    }
    if ($1 == "lot") {
      if (NF < 4) die("lot " $2 " incomplet")
      nl++
      lname[nl] = $2; lfilter[nl] = $3; lnp[nl] = 0
      for (i = 4; i <= NF; i++) lpat[nl, ++lnp[nl]] = $i
      if (!(index($3, "run:") == 1 || index($3, "skip:") == 1 || $3 == "-")) die("filtre inconnu pour " $2 " : " $3)
      next
    }
    if ($1 == "timeout-defaut") {
      if (NF != 2 || $2 !~ /^[0-9]+s$/) die("timeout-defaut attend une durée en secondes (ex. 600s)")
      tdefault = $2
      next
    }
    if ($1 == "timeout") {
      if (NF < 3 || $2 !~ /^[0-9]+s$/) die("timeout attend une durée en secondes puis des lots")
      for (i = 3; i <= NF; i++) tlot[$i] = $2
      next
    }
    die("directive inconnue : " $1)
  }
  END {
    if (failed) exit 1
    if (nl == 0) die("aucun lot défini")
    if (tdefault == "") die("timeout-defaut absent")
    for (name in tlot) {
      known = 0
      for (l = 1; l <= nl; l++) if (lname[l] == name) known = 1
      if (!known) die("timeout : lot inconnu " name)
    }

    # 1. Paquets de chaque lot (hors complément).
    for (l = 1; l <= nl; l++) {
      isrest = (lnp[l] == 1 && lpat[l, 1] == "@reste")
      for (k = 1; k <= lnp[l]; k++) if (lpat[l, k] == "@reste" && !isrest) die("@reste doit être seul dans son lot (" lname[l] ")")
      if (isrest) {
        if (l != nl) die("@reste doit être le dernier lot (" lname[l] ")")
        rest = l
        continue
      }
      for (k = 1; k <= lnp[l]; k++) {
        pat = lpat[l, k]
        if (substr(pat, 1, 1) == "!") continue
        hits = 0
        for (i = 1; i <= np; i++) if (matches(pat, pk[i])) { in_lot[l, i] = 1; hits++ }
        if (hits == 0) die("lot " lname[l] " : le motif " pat " ne désigne aucun paquet")
      }
      for (k = 1; k <= lnp[l]; k++) {
        pat = lpat[l, k]
        if (substr(pat, 1, 1) != "!") continue
        pat = substr(pat, 2); hits = 0
        for (i = 1; i <= np; i++) if (in_lot[l, i] && matches(pat, pk[i])) { delete in_lot[l, i]; hits++ }
        if (hits == 0) die("lot " lname[l] " : l exclusion !" pat " ne retire aucun paquet")
      }
    }

    if (!rest) die("le dernier lot doit être le complément (@reste) : sans lui un paquet ajouté au module ne tournerait nulle part")

    # 2. Propriétaires de chaque paquet.
    for (i = 1; i <= np; i++) {
      n = 0
      for (l = 1; l <= nl; l++) if (in_lot[l, i]) { n++; own[i, n] = l }
      nown[i] = n
      if (n == 0 && rest) { in_lot[rest, i] = 1; own[i, 1] = rest; nown[i] = 1; nrest++ }
    }
    if (rest && nrest == 0) die("le complément (@reste) est vide : tout est déjà listé explicitement")

    # 3. Un paquet : un lot sans filtre, ou la paire complémentaire run:L / skip:L.
    for (i = 1; i <= np; i++) {
      n = nown[i]
      if (n == 1) {
        l = own[i, 1]
        if (lfilter[l] != "-") die("paquet " pk[i] " : le lot " lname[l] " le filtre (" lfilter[l] ") sans lot complémentaire, des tests ne tourneraient nulle part")
        continue
      }
      if (n != 2) die("paquet " pk[i] " : présent dans " n " lots")
      a = own[i, 1]; b = own[i, 2]
      fa = lfilter[a]; fb = lfilter[b]
      ok = 0
      if (index(fa, "run:") == 1 && index(fb, "skip:") == 1 && substr(fa, 5) == substr(fb, 6)) ok = 1
      if (index(fb, "run:") == 1 && index(fa, "skip:") == 1 && substr(fb, 5) == substr(fa, 6)) ok = 1
      if (!ok) die("paquet " pk[i] " : présent dans les lots " lname[a] " et " lname[b] " sans être la paire complémentaire run:/skip:")
    }

    # 4. Filtres : la liste nommée existe.
    for (l = 1; l <= nl; l++) {
      f = lfilter[l]
      if (f == "-") continue
      name = substr(f, index(f, ":") + 1)
      if (!(name in heavy)) die("lot " lname[l] " : liste " name " non déclarée par heavy")
    }

    # 5. Sortie.
    if (mode == "--lots") {
      for (l = 1; l <= nl; l++) print lname[l]
      exit 0
    }
    if (mode == "--verifier") {
      for (l = 1; l <= nl; l++) {
        c = 0
        for (i = 1; i <= np; i++) if (in_lot[l, i]) c++
        printf "%-20s %4d paquets  filtre %-24s timeout %s\n", lname[l], c, lfilter[l], ((lname[l] in tlot) ? tlot[lname[l]] : tdefault)
      }
      printf "partition exacte : %d paquets de go list, %d lots, complément de %d paquets\n", np, nl, nrest
      exit 0
    }
    for (l = 1; l <= nl; l++) if (lname[l] == mode) target = l
    if (!target) die("lot inconnu : " mode " (lots : voir --lots)")
    print "-timeout"
    print ((lname[target] in tlot) ? tlot[lname[target]] : tdefault)
    f = lfilter[target]
    if (f != "-") {
      kind = substr(f, 1, index(f, ":") - 1)
      print "-" kind
      print "^(" heavy[substr(f, index(f, ":") + 1)] ")$"
    }
    for (i = 1; i <= np; i++) if (in_lot[target, i]) print "./" pk[i]
  }
' "$PKG_FILE" "$DEF_FILE"
