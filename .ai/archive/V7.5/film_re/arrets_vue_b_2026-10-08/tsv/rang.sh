#!/bin/bash
# usage: rang.sh <rev ancienne> <rev nouvelle> : monte grammar.Rev, regenere empreintes, golden de formes et fixtures de contrat, puis verifie
S=/c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/bfd187a4-ee4b-4656-9ab4-0654cbc1dfb8/scratchpad
W=/c/Users/Guillaume/Downloads/Scripts/LevelUp-wt-grammaire-arrets-vue-b-2
export GOCACHE=C:/Users/Guillaume/AppData/Local/go-build-vueb CGO_ENABLED=1 CC=gcc
cd $W/apps/go-api
F=internal/games/halo_infinite/film
sed -i "s/^const Rev = \"$1\"\$/const Rev = \"$2\"/" $F/internal/grammar/rev.go
grep -q "^const Rev = \"$2\"" $F/internal/grammar/rev.go || { echo "REV NON MONTEE"; exit 1; }
LEVELUP_UPDATE_GRAMMAR_REV=1 go test ./$F/internal/grammar/ -run TestGrammarRevSuitLaGrammaire -count=1 -update-grammar-rev >/dev/null 2>&1
LEVELUP_UPDATE_KILLSOURCE_REV=1 go test ./$F/internal/facts/killsource/ -run TestKillsourceRevSuitLaSortie -count=1 -update-killsource-rev >/dev/null 2>&1
LEVELUP_UPDATE_OBJECTIVES_REV=1 go test ./$F/internal/facts/objectives/ -run TestObjectivesRevSuitLaSortie -count=1 -update-objectives-rev >/dev/null 2>&1
LEVELUP_UPDATE_TYPES_SHAPES=1 go test ./$F/types/ -run TestFormesDesTypesEgalentLeGolden -count=1 -update-types-shapes >/dev/null 2>&1
REPLAY_CONTRACT_UPDATE=1 go test ./$F/replay/ -run ContractFixtures -count=1 -update >/dev/null 2>&1
go test ./$F/internal/grammar/ ./$F/internal/facts/killsource/ ./$F/internal/facts/objectives/ ./$F/types/ -run 'Rev|Chronique|Formes' -count=1 2>&1 | tail -5
$S/verif_fixtures.sh $1 $2
