#!/bin/bash
# gate.sh : replay-corpus-gate du lot LN, base explicite 2393d7db7, parc COPIE au scratchpad, travail au scratchpad.
source /c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/v2-LN/env.sh
cd $API
$L/bin/final/gate.exe --reference=base --base=2393d7db7 --parc-root $(cygpath -m -l $L/parc) --source-root $(cygpath -m -l $WT) \
  --work-root $(cygpath -m -l $L/gate_work) --json $(cygpath -m -l $L/ln_corpus_gate.json) > $L/gate.log 2>&1
echo "rc=$?" >> $L/gate.log
