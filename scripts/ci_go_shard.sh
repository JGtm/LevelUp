#!/usr/bin/env bash
# scripts/ci_go_shard.sh — arguments `go test` d'un lot de la suite Go de la CI.
#
# Usage :
#   bash scripts/ci_go_shard.sh <lot>                un argument par ligne : `-timeout` et sa durée,
#                                                    filtre éventuel (-run/-skip et son motif), puis les
#                                                    paquets du lot (./chemin, relatifs à SON module)
#   bash scripts/ci_go_shard.sh --module <lot>       le module du lot (dossier relatif à la racine du
#                                                    dépôt) : le `go test` du lot se lance depuis lui
#   bash scripts/ci_go_shard.sh --lots               noms des lots, un par ligne
#   bash scripts/ci_go_shard.sh --lots-du-module <m> noms des lots du module <m>, un par ligne
#   bash scripts/ci_go_shard.sh --modules            modules de la définition, un par ligne
#   bash scripts/ci_go_shard.sh --verifier           prouve la partition (cf. ci-dessous) et la résume
#
# La définition des lots vit dans .github/ci/go-shards.txt (syntaxe documentée en tête de ce
# fichier). Chaque lot appartient à un MODULE Go de l'espace de travail (directive `module`, ADR
# 0041 ; `apps/go-api` par défaut) : `./...` ne franchit jamais la frontière d'un module, donc la
# partition se prouve MODULE PAR MODULE. Ce script confronte la définition à
# `go list -tags=integration ./...` de chaque module — la MÊME énumération que
# `go test -tags=integration ./...` depuis ce module — et refuse (code 1, message sur stderr) toute
# définition qui perdrait ou dupliquerait un paquet ou des tests :
#   - un motif qui ne désigne aucun paquet de son module (lot périmé) ;
#   - un paquet dans deux lots, hors paire complémentaire `run:L` / `skip:L` ;
#   - un paquet filtré sans son complément (des tests ne tourneraient nulle part) ;
#   - un module sans complément (`@reste`), un complément vide, ou qui n'est pas le dernier lot de
#     son module.
# Le complément ne peut donc pas laisser un paquet hors de la matrice : c'est `go list` qui le
# calcule, pas une liste écrite à la main.
#
# La sortie se consomme sans passer par le shell pour la découpe :
#   cd "$(bash scripts/ci_go_shard.sh --module "$LOT")"
#   out=$(bash ../../scripts/ci_go_shard.sh "$LOT") || exit 1
#   mapfile -t args <<< "$out"
#   go test ... "${args[@]}"
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEF_FILE="${CI_GO_SHARDS_FILE:-$REPO_ROOT/.github/ci/go-shards.txt}"
# MODULE_PAR_DEFAUT : le module des lots écrits avant toute directive `module`.
MODULE_PAR_DEFAUT="apps/go-api"

usage() {
  echo "usage: $0 <lot> | --module <lot> | --lots | --lots-du-module <module> | --modules | --verifier" >&2
  exit 2
}
case "${1:-}" in
  --module | --lots-du-module)
    [[ $# -eq 2 ]] || usage
    MODE="$1:$2"
    ;;
  "") usage ;;
  *)
    [[ $# -eq 1 ]] || usage
    MODE="$1"
    ;;
esac

if [[ ! -f "$DEF_FILE" ]]; then
  echo "ci_go_shard: définition introuvable : $DEF_FILE" >&2
  exit 1
fi

# Sur Windows local, le gcc de msys64 est requis pour que `go list` voie les fichiers cgo.
if [[ -f /c/msys64/ucrt64/bin/gcc.exe ]]; then
  export PATH="/c/msys64/ucrt64/bin:$PATH"
fi

# Modules de la définition : le module par défaut, puis ceux des directives `module`, dans
# l'ordre, sans doublon.
mapfile -t MODULES < <(
  { echo "$MODULE_PAR_DEFAUT"; sed 's/#.*//; s/\r$//' "$DEF_FILE" | awk '$1 == "module" && NF == 2 { print $2 }'; } |
    awk '!vu[$0]++'
)

# Paquets de chaque module, « <module> <chemin relatif au module> » par ligne. `go list -m` rend
# TOUS les modules de l'espace de travail (go.work) : le chemin d'un module se lit dans son go.mod.
PKG_FILE="$(mktemp)"
trap 'rm -f "$PKG_FILE"' EXIT
for mod in "${MODULES[@]}"; do
  if [[ ! -f "$REPO_ROOT/$mod/go.mod" ]]; then
    echo "ci_go_shard: module $mod sans go.mod ($REPO_ROOT/$mod/go.mod)" >&2
    exit 1
  fi
  chemin="$(awk '$1 == "module" { print $2; exit }' "$REPO_ROOT/$mod/go.mod" | tr -d '\r')"
  (cd "$REPO_ROOT/$mod" && CGO_ENABLED=1 go list -tags=integration ./...) | tr -d '\r' |
    awk -v mod="$mod" -v chemin="$chemin" '{ p = $0; if (p == chemin) p = "."; else if (index(p, chemin "/") == 1) p = substr(p, length(chemin) + 2); print mod, p }' >> "$PKG_FILE"
done
if [[ ! -s "$PKG_FILE" ]]; then
  echo "ci_go_shard: go list n'a rendu aucun paquet" >&2
  exit 1
fi

awk -v mode="$MODE" -v defmod="$MODULE_PAR_DEFAUT" '
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
  FNR == NR { if (NF == 2) { pk[++np] = $2; pkm[np] = $1 }; next }
  {
    sub(/#.*/, ""); sub(/\r$/, "")
    if (NF == 0) next
    if ($1 == "module") {
      if (NF != 2) die("module attend un dossier relatif à la racine du dépôt")
      cur = $2
      next
    }
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
      for (l = 1; l <= nl; l++) if (lname[l] == $2) die("lot " $2 " défini deux fois")
      nl++
      lname[nl] = $2; lfilter[nl] = $3; lnp[nl] = 0
      lmod[nl] = (cur == "" ? defmod : cur)
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

    # 1. Paquets de chaque lot (hors complément), pris dans SON module.
    for (l = 1; l <= nl; l++) {
      isrest = (lnp[l] == 1 && lpat[l, 1] == "@reste")
      for (k = 1; k <= lnp[l]; k++) if (lpat[l, k] == "@reste" && !isrest) die("@reste doit être seul dans son lot (" lname[l] ")")
      if (isrest) {
        for (m = l + 1; m <= nl; m++) if (lmod[m] == lmod[l]) die("@reste doit être le dernier lot de son module " lmod[l] " (" lname[l] ")")
        rest[lmod[l]] = l
        continue
      }
      for (k = 1; k <= lnp[l]; k++) {
        pat = lpat[l, k]
        if (substr(pat, 1, 1) == "!") continue
        hits = 0
        for (i = 1; i <= np; i++) if (pkm[i] == lmod[l] && matches(pat, pk[i])) { in_lot[l, i] = 1; hits++ }
        if (hits == 0) die("lot " lname[l] " : le motif " pat " ne désigne aucun paquet du module " lmod[l])
      }
      for (k = 1; k <= lnp[l]; k++) {
        pat = lpat[l, k]
        if (substr(pat, 1, 1) != "!") continue
        pat = substr(pat, 2); hits = 0
        for (i = 1; i <= np; i++) if (in_lot[l, i] && matches(pat, pk[i])) { delete in_lot[l, i]; hits++ }
        if (hits == 0) die("lot " lname[l] " : l exclusion !" pat " ne retire aucun paquet")
      }
    }

    # Chaque module qui porte des paquets a son complément : sans lui, un paquet ajouté au module ne
    # tournerait nulle part.
    for (i = 1; i <= np; i++) {
      if (!(pkm[i] in rest)) die("le module " pkm[i] " n a pas de complément (@reste en dernier lot) : sans lui un paquet ajouté au module ne tournerait nulle part")
    }

    # 2. Propriétaires de chaque paquet.
    for (i = 1; i <= np; i++) {
      n = 0
      for (l = 1; l <= nl; l++) if (in_lot[l, i]) { n++; own[i, n] = l }
      nown[i] = n
      if (n == 0) { r = rest[pkm[i]]; in_lot[r, i] = 1; own[i, 1] = r; nown[i] = 1; nrest[pkm[i]]++ }
    }
    for (m in rest) if (nrest[m] == 0) die("le complément (@reste) du module " m " est vide : tout est déjà listé explicitement")

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
    if (mode == "--modules") {
      for (l = 1; l <= nl; l++) if (!(lmod[l] in vu)) { vu[lmod[l]] = 1; print lmod[l] }
      exit 0
    }
    if (index(mode, "--lots-du-module:") == 1) {
      m = substr(mode, length("--lots-du-module:") + 1)
      for (l = 1; l <= nl; l++) if (lmod[l] == m) { print lname[l]; c++ }
      if (c == 0) die("module sans lot : " m)
      exit 0
    }
    if (mode == "--verifier") {
      for (l = 1; l <= nl; l++) {
        c = 0
        for (i = 1; i <= np; i++) if (in_lot[l, i]) c++
        printf "%-20s %-12s %4d paquets  filtre %-24s timeout %s\n", lname[l], lmod[l], c, lfilter[l], ((lname[l] in tlot) ? tlot[lname[l]] : tdefault)
      }
      nm = 0
      for (m in rest) nm++
      printf "partition exacte : %d paquets de go list (%d module(s)), %d lots\n", np, nm, nl
      for (m in rest) printf "  module %s : complément de %d paquets (lot %s)\n", m, nrest[m], lname[rest[m]]
      exit 0
    }
    if (index(mode, "--module:") == 1) {
      name = substr(mode, length("--module:") + 1)
      for (l = 1; l <= nl; l++) if (lname[l] == name) { print lmod[l]; exit 0 }
      die("lot inconnu : " name " (lots : voir --lots)")
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
    for (i = 1; i <= np; i++) if (in_lot[target, i]) print (pk[i] == "." ? "." : "./" pk[i])
  }
' "$PKG_FILE" "$DEF_FILE"
