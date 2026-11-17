# Marqueurs `color-allow` inertes (item 0.3, pour D.3)

Mesure faite le 2026-11-17 sur le worktree `C:/t/bl1117` (branche `chore/830-backlog-1117`, base `3ae75c78f5`), en lecture seule.

## Critere

Source : `tools/lint-no-hardcoded-colors.lib.mjs` (fonction `scanLine`) et `tools/lint-no-hardcoded-colors.mjs` (balayage).

- Le balayage lit les `.ts`, `.tsx`, `.js`, `.jsx` de `apps/web/src/features`, `apps/web/src/components` et `apps/web/src/lib/replay`, hors fichiers de la liste `ALLOWED_FILES` (tests, palettes, `rarity.ts`, fichiers generes).
- Pour une ligne, `scanLine` rend la liste des violations (genres `hex`, `tailwind`, `fonction-couleur`) puis retire celles que le `color-allow(<genre>)` de la ligne nomme. Une ligne de commentaire (premier caractere non blanc `//` ou `*`) rend toujours une liste vide, avant tout motif.
- Un marqueur est INERTE quand, sans lui, `scanLine` ne signalerait rien sur sa ligne. Mise en oeuvre : pour chaque ligne qui contient `color-allow`, le mot est neutralise (`color-xxxxx`) puis `scanLine(chemin, ligne)` est rappelee ; liste vide = marqueur inerte. Le script de mesure est dans le dossier temporaire de la session, hors depot.
- Verifications croisees : 0 ligne a deux marqueurs ; 0 ligne qui rougit avec son marqueur (le lint passe : `lint-no-hardcoded-colors: clean (0 violation)`) ; 0 marqueur actif qui nomme un genre absent de sa ligne (pas de cas « partiellement inerte »).

## Comptes

- Lignes portant `color-allow` dans le perimetre balaye : 95 (47 fichiers). Attendu : 95. Confirme.
- Marqueurs inertes : 19 (15 fichiers) dont 5 sur une ligne de commentaire et 14 sur une ligne de code. Attendu : 19. Confirme.
- Marqueurs actifs : 76 (29 `tailwind`, 22 `hex`, 20 `fonction-couleur`, 5 `hex` + `fonction-couleur`).
- Hors perimetre du lint (donc ni comptes dans les 95, ni dans les 19) : 2 lignes de `apps/web/src/lib/accessibility/tokenTone.ts` (30 et 31, hors des trois dossiers balayes) et 13 lignes dans 2 fichiers de test (`.test.ts`, exclus par `ALLOWED_FILES`). Total brut `grep` dans `apps/web/src` : 110 lignes.

## Liste fermee des 19 marqueurs inertes

| # | Fichier:ligne | Commentaire du marqueur (texte de la ligne) | Pourquoi inerte |
|---|---|---|---|
| 1 | `features/explorer/ExplorerMatchesTable.tsx:553` | `// Les couleurs hex sont autorisées (color-allow) car identifiants UX` | ligne de commentaire : `scanLine` la tolere avant tout motif (`isCommentLine`) ; le marqueur n'excuse rien |
| 2 | `features/match-view/MatchHeader.card.tsx:468` | `{/* color-allow: currentColor uniquement — couleur hérité du contexte */}` | aucune couleur litterale sur la ligne (commentaire JSX seul : `currentColor` n'est pas un motif) ; le marqueur ne couvre rien |
| 3 | `features/match-view/MatchSummaryCharts.tsx:229` | `// color-allow: hex en commentaires de documentation token→couleur` | ligne de commentaire : `scanLine` la tolere avant tout motif (`isCommentLine`) ; le marqueur n'excuse rien |
| 4 | `features/media/MediaViewer.tsx:121` | `// color-allow: rose pour le bouton like (heart) — CLAUDE.md §20 tolère rose pour liked` | ligne de commentaire : `scanLine` la tolere avant tout motif (`isCommentLine`) ; le marqueur n'excuse rien |
| 5 | `features/session-detail/SessionFdaGapCumulative.tsx:146` | `...hoverRevealSymbol(tc.text), // color-allow: point neutre, l'aire porte le dégradé` | la valeur vient du theme (`tc.text`) : aucun litteral de couleur sur la ligne |
| 6 | `features/squad/charts/squadIntensityProfileChart.ts:226` | `* color-allow: 'color' reçoit 'tc.text', neutre de thème réservé aux séries` | ligne de commentaire : `scanLine` la tolere avant tout motif (`isCommentLine`) ; le marqueur n'excuse rien |
| 7 | `features/squad/emprise/PickupSheetsCard.tsx:46` | `const fg = 'var(--foreground)' // color-allow: encre neutre des marques de légende (maquette)` | la valeur est un jeton `var(--...)` : ni hex, ni classe Tailwind couleur, ni fonction de couleur sur la ligne |
| 8 | `features/squad/MedalDigest.tsx:64` | `background: 'var(--muted)', // color-allow: structurel — badge count neutre` | la valeur est un jeton `var(--...)` : ni hex, ni classe Tailwind couleur, ni fonction de couleur sur la ligne |
| 9 | `features/squad/MedalDigest.tsx:66` | `border: '1.5px solid var(--background)', // color-allow: structurel — séparation badge/icône` | la valeur est un jeton `var(--...)` : ni hex, ni classe Tailwind couleur, ni fonction de couleur sur la ligne |
| 10 | `features/squad/MedalDigest.tsx:107` | `background: 'var(--muted)', // color-allow: structurel — badge count neutre` | la valeur est un jeton `var(--...)` : ni hex, ni classe Tailwind couleur, ni fonction de couleur sur la ligne |
| 11 | `features/squad/MedalDigest.tsx:109` | `border: '1.5px solid var(--background)', // color-allow: structurel — séparation badge/icône` | la valeur est un jeton `var(--...)` : ni hex, ni classe Tailwind couleur, ni fonction de couleur sur la ligne |
| 12 | `features/squad/objectif/ObjectifFrame.tsx:123` | `'repeating-linear-gradient(to right, var(--muted-foreground) 0 3px, transparent 3px 5px)', // color-allow: repère neutre de la légende (maquette C3EW)` | la valeur est un jeton `var(--...)` : ni hex, ni classe Tailwind couleur, ni fonction de couleur sur la ligne |
| 13 | `features/squad/v2/SquadEngagementView.tsx:114` | `// color-allow: hex résolu depuis semantic tokens via colorHex ou resolveToken` | ligne de commentaire : `scanLine` la tolere avant tout motif (`isCommentLine`) ; le marqueur n'excuse rien |
| 14 | `features/timeseries/TimeseriesFdaGapTrend.tsx:138` | `...hoverRevealSymbol(tc.text), // color-allow: point neutre, l'aire porte le dégradé` | la valeur vient du theme (`tc.text`) : aucun litteral de couleur sur la ligne |
| 15 | `features/timeseries/TimeseriesNetLivesTrend.tsx:155` | `...hoverRevealSymbol(tc.text), // color-allow: point neutre, l'aire porte le dégradé` | la valeur vient du theme (`tc.text`) : aucun litteral de couleur sur la ligne |
| 16 | `features/timeseries/TimeseriesNetLivesTrend.tsx:163` | `color: tc.text, // color-allow: encre de texte du thème, pas une teinte de donnée` | la valeur vient du theme (`tc.text`) : aucun litteral de couleur sur la ligne |
| 17 | `features/timeseries/usages/EquipmentOutcomesCard.tsx:83` | `{ kind: 'thin', label: e.thinLegend, color: 'var(--muted-foreground)' }, // color-allow: repère neutre de la légende (maquette)` | la valeur est un jeton `var(--...)` : ni hex, ni classe Tailwind couleur, ni fonction de couleur sur la ligne |
| 18 | `features/timeseries/usages/LivesNearTeammateCard.tsx:50` | `{ kind: 'thin', label: l.thinLegend, color: 'var(--muted-foreground)' }, // color-allow: repère neutre de la légende (maquette)` | la valeur est un jeton `var(--...)` : ni hex, ni classe Tailwind couleur, ni fonction de couleur sur la ligne |
| 19 | `components/ui/match-card.tsx:639` | `{/* color-allow: currentColor uniquement — couleur hérité du contexte */}` | aucune couleur litterale sur la ligne (commentaire JSX seul : `currentColor` n'est pas un motif) ; le marqueur ne couvre rien |

Chemins relatifs a `apps/web/src/`. Pour D.3 : retirer le mot `color-allow` (et son eventuelle parenthese) de ces 19 lignes en gardant la raison en commentaire. `features/squad/MedalDigest.tsx` porte quatre de ces lignes (64, 66, 107, 109) ; `features/timeseries/TimeseriesNetLivesTrend.tsx` en porte deux (155, 163).

## Remarques utiles a D.3 et D.5

- 5 des 19 marqueurs sont sur des lignes de commentaire : `ExplorerMatchesTable.tsx:553` (la phrase dit que « les couleurs hex sont autorisees (color-allow) »), `MatchSummaryCharts.tsx:229`, `MediaViewer.tsx:121`, `squadIntensityProfileChart.ts:226` (ligne `*` d'un bloc JSDoc), `SquadEngagementView.tsx:114`. Le retrait ne doit pas laisser une phrase qui renvoie a un marqueur disparu.
- Pour D.5 (genre « marqueur inerte » au plafond 0) : le critere ci-dessus est calculable avec les seules fonctions exportees de la lib (`scanLine`, `allowedKinds`, `isAllowedFile`). Deux faits a connaitre : (1) une ligne de commentaire est toujours inerte ; (2) `ALLOWED_FILES` contient deux entrees mortes, `features/home/rarity.ts` (ligne 68) et `features/battlepass/rarity.ts` (ligne 69), dont les fichiers n'existent pas.
- `shadow`, `from`, `via`, `to` sont DEJA dans `ABSOLUTE_PREFIXES` (noir et blanc absolus : `from-black`, `shadow-white/40`...). Ils manquent seulement dans `TW_PREFIXES` (couleurs nommees : `from-red-500`). D.5 porte donc sur `TW_PREFIXES`.
