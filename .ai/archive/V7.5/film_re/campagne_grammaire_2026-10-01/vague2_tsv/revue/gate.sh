#!/bin/bash
# gate.sh : replay-corpus-gate, base explicite 6fa631df0, parc COPIE au scratchpad (celui de l integration), travail au scratchpad.
source /c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/v2-integ/rev/env.sh
cd $API
$R/bin/tete/gate.exe --reference=base --base=6fa631df0 --parc-root $(cygpath -m -l $V/parc) --source-root $(cygpath -m -l $WT) \
  --work-root $(cygpath -m -l $R/gate_work) --json $(cygpath -m -l $R/revue_corpus_gate.json) > $R/gate.log 2>&1
echo "rc=$?" >> $R/gate.log
