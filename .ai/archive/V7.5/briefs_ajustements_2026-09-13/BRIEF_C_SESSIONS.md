# LOT C — Sessions : bloc « Usages d'équipement »

Worktree : `C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-ajust-sessions` · branche `feat/ajust-sessions` · port Vite **5183**.
Lis d'abord `BRIEF_COMMUN.md` (même dossier). Lot 100 % web.

## Repérage déjà fait (vérifie sur pièces)

- Page : `features/session-detail/SessionDetailPage.tsx` (colonne principale `:343-352`, drawer `:371-379`, colonne comparée `:434-444`) ; corps `SessionColumnBody.tsx:50-95`, bloc usage monté `:82`.
- Orchestrateur : `features/session-detail/SessionUsageSection.tsx:134-172` ; carte 1 `EquipmentCard :222-274` ; carte 2 `PadControlCard :276-352` (+ pied `:364-396`) ; carte 3 Objectifs `:398-462`. Les trois rendues à la suite.
- Sous-parties de la carte 1 : cadences `:244-249` (`buildCadenceGrid` `features/_shared/usage/usageGrids.ts:89-158` → `components/charts/ValueGrid.tsx`) ; parts `:250-255` (`metricGaugeRows :184-207`, `usageGaugeModel.ts:184-269`, rendu `UsageForms.tsx:216-292` `UsageGaugeGrid` + `UsageGauge :157-193`) ; régularité `:256-270` (`UsageForms.tsx:381-414`).
- « Compact » = mécaniquement `drawerOpen` (`SessionDetailPage.tsx:347`, `:438`) ; compact RETIRE : régularité (carte 1, `:256`), piste du lobby (carte 2, `:333`), régularité carte 2 (`:339`), deux grilles carte 3 (`:448`, `:454`).
- Repli des parts : `UsageForms.tsx:216-217` `useState(false)` → replié par défaut, seule la colonne `PRIMARY_GAUGE_INDEX = 1` (« Ma part dans mon équipe ») visible ; bouton `CollapsedItemsToggle`.
- Couleurs des jauges : `UsageForms.tsx` — `ALLY_INK = tokenCssVar('team-ally')` (`:58`, utilisé `:176`) ; avec `outcomes` les deux jauges portent la même pile `divergent-pos/neutral/neg` (`usageGaugeModel.ts:71-75`, rendu `:124-131`) ; parité `warning` ; repères `team-ally` plein / pointillé `muted-foreground`. « Ma part dans mon équipe » et « Ma part dans le lobby » sont donc INDISCERNABLES.
- Normalisation « par 10 minutes » : Go, `internal/analysis/sessionusage/usage.go:413-419` (`per10Min`, constante 600). Ne change pas le référentiel (décision utilisateur en suspens ; le superviseur l'explique) — tu ne touches pas au Go.

## Items

### C.1 — Trois blocs au lieu d'un
La carte « Usages d'équipement » devient trois `SectionCard` distincts, dans cet ordre : « Cadences » (titre exact : « Cadences par 10 minutes de jeu mesuré » — garde le libellé existant), « Parts » (titre : reprends le libellé existant de la vue parts, ex. « Ma part » — regarde `usageI18n.ts:197-206` `viewShares` et utilise-le), « Régularité match par match ». Les infobulles/notes de chaque vue suivent leur bloc. Le titre-adornment « Matchs mesurés N/M » reste sur le PREMIER bloc seulement (ne pas le tripler). Fais la même chose pour la carte 2 « Contrôle des armes spéciales » SEULEMENT si elle a la même structure en trois vues (cadences/parts/régularité) — regarde `:276-352` ; si oui, même découpage ; sinon laisse-la et dis pourquoi.

### C.2 — Parts toujours dépliées en vue étendue
En vue étendue (drawer fermé) : les trois jauges (« Mon équipe dans le lobby », « Ma part dans mon équipe », « Ma part dans le lobby ») sont TOUJOURS visibles, sans bouton de repli. Supprime le repli (`expanded`, `CollapsedItemsToggle`, clés `sharesShowMoreFmt/sharesHide/sharesHint` si plus utilisées, tests associés) — pas de « au cas où ».

### C.3 — La vue compacte montre ce que montre le drawer… et réciproquement
Ce que l'utilisateur veut : la vue compacte (celle du drawer, deux colonnes) doit contenir les MÊMES éléments que la vue étendue, en plus serré ; aujourd'hui « il manque des éléments du drawer ». Donc : en compact, ne retire plus la régularité (carte 1), la piste du lobby ni la régularité (carte 2), ni les deux grilles (carte 3) — tout reste, rendu compact (hauteurs/largeurs réduites, typographie plus petite si nécessaire, mais rien de masqué). Si un élément est réellement illisible à demi-largeur (ex. bande de régularité à 30 matchs), rends-le lisible (défilement horizontal interne, cases plus étroites) — ne le cache pas. Vérifie sur capture drawer ouvert.

### C.4 — Couleurs distinguables des deux jauges « Ma part »
« Ma part dans mon équipe » et « Ma part dans le lobby » doivent être distinguables au premier coup d'œil, y compris par un lecteur daltonien, en gardant la forme. Règle : les jetons existent tous dans `lib/accessibility/semantic-tokens.ts`, tu n'en crées pas ; invoque `color-tokens`. Pistes : la jauge « dans mon équipe » garde `team-ally` ; la jauge « dans le lobby » prend un jeton distinct de contraste suffisant (ex. `squad-player-1` ou `chart-series-*` — choisis celui qui n'entre pas en collision avec la pile d'issues divergent-pos/neutral/neg si `outcomes` est présent : dans ce cas la pile est la même par construction ; distingue alors par un liseré/bord de jauge ET par le libellé, et documente). Vérifie le contraste (WCAG ≥ 3:1 entre les deux teintes et vs fond) avec `resolveToken` dans un test ou par calcul ; montre les deux jauges côte à côte sur la capture APRÈS.

Captures APRÈS : page Sessions étendue (colonne entière), drawer ouvert (les deux colonnes).
