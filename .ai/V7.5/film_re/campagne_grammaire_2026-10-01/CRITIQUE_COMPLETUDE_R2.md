**Critique de complétude, contexte frais (2026-10-02)**

J'ai tout relu sur l'arbre de travail, où le traitement n'est pas commité : `git status` montre le plan, le rapport, les six `R_*.md` et `thought_log` modifiés, et `R_COMB_2.md`, `SURCOUCHE_UNIQUE.md`, `r_comb2_tsv/` et trois sondes non suivis. La tête poussée `df228c24c` date d'avant la critique (14:20 contre 14:24). Je n'ai lancé aucune commande `go` et je n'ai rien corrigé. Les recalculs sont faits par awk sur les TSV.

**Écart à déclarer :** une de mes commandes awk a écrit une fois `/tmp/x`, hors du worktree et du scratchpad. Le fichier est inutilisé et je l'ai effacé aussitôt (`ls` le confirme absent).

## 1. Statut des 25 points

PLAN = `.ai/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md` ; RAPPORT = `.ai/V7.5/film_re/RAPPORT_CAMPAGNE_GRAMMAIRE_PHASE1_2026-10-01.md`.

| Pt | Statut | Preuve, et ce qui reste |
|---|---|---|
| A1 | traité (reste à trancher : D18) | L0 en tête : §6.0 « Montée » l.1162-1170, §6.1 ligne L0, §6.2 L0 l.1678-1697, §6.5.4 point 2. Écart restant : le juge est dit « L0.6 à L0.8 » (l.826, l.1690) mais « L0.6 à L0.9 » au §6.5.2 l.2124, alors que L0.9 est un témoin, pas un invariant. |
| A2 | traité (D18) | §6.5.4 retitré « AMENDEMENT » ; D18 l.1973-1983. Mais `R_COMB_2.md` §10 dit encore « ne change aucune décision du §3 » : c'est la même erreur, et rien ne la corrige. |
| A3 | traité | §6.3 l.1737-1742 ; D1, D-RI, D-VEH et D2 sont marquées TRANCHÉES ; le prérequis 4 de L4 est barré (l.1635). |
| A4 | traité | Règle de lecture l.751-754 ; §6.1 en sains ; §6.5.4 ordonné par sains marginaux ; formats 24-25 en sains (§6.5.3). J'ai vérifié par awk : `1c4c63c2` 49,6 %, `60ae07c4` 65,9 %, `11de8353` 79,9 %, exacts. |
| B5 | traité | §6.2 L6b l.1542-1553 (« périmé », gate 2 en défaut) ; §6.5.2 « pas prêts ». |
| B6 | traité | §6.2 L6a l.1527-1535 ; D-110 ; §6.1 « Non retenus ». |
| B7 | traité, avec des trous déclarés | §6.0 gate 3 l.847-855 ; §6.2 LS l.1422-1426 ; LU l.1386-1389. Gate 3 joué sur 19 films seulement, celui de L9 reste ouvert, et « `killsource.Rev` n'a pas à monter pour un diagnostic » est supposé. |
| B8 | traité (D22) | D-67 l.511-516 ; D22 l.2001-2006. |
| B9 | traité pour L3a seulement | §6.2 L3 l.1604-1611 ; §6.5.1 R-L3. Les autres lots ne sont pas couverts (manque N4). |
| C10 | partiellement | Recalculé (§6.0 l.792-803, §6.5.3, D12). Mais la règle n'est toujours pas appliquée à la lettre (manque N2). |
| C11 | traité, vérifié | D-71 l.539-544. Mon recalcul sur `r_comb_par_film.tsv` (config `full`) et le fixe de R-COMB-2 donne `f75e7053` 98,3 %, `81c02726` 97,5 %, `c75f33b8` 87,7 %. |
| C12 | traité | D-100 l.662-669 ; §6.5.5 ; note sur le RAPPORT §8. L'entrée d'intégration de `thought_log` dit encore « rouge avec quatre » ; l'entrée suivante corrige (5 sur 6). |
| C13 | partiellement | Corrigé au §6.1 l.1231-1233, au §6.5.1 et dans `R_VEH.md` correction 7 (j'ai vérifié 13 films, 9 à pertes brutes, 7 en baisse nette sur le tableau §5.4). Mais §6.2 L6a l.1519-1521 dit encore « 9 films en baisse ». |
| C14 | traité | Section « Corrections du 2026-10-02 » en tête des six notes : `R_COMP` 1, `R_VEH` 1-3, `R_LOC` 1, `R_NAIS` 1-2, `R_FUSION` 1. |
| D15 | traité | `VERIFICATIONS_ADVERSES_R.md` existe ; l'exemple de D-93 s'y trouve (l.188, point iii). |
| D16 | partiellement | Rouge soldé par la surcouche unique (`SURCOUCHE_UNIQUE.md` §3, mesuré par sa note, « non rejoué ici » au §6.5.5) ; l'item `mesures_bis2_overlay/` est corrigé (l.1011-1015). Restent : l'item `[!]` « retirer les six anciennes surcouches » (l.1035-1041), reporté pour une raison de périmètre, qui n'est pas un report valide de `plan-execution` ; et toujours aucune entrée au journal §4 pour l'intégration, la surcouche unique, R-COMB-2 ni ce traitement (§6.5.6 le renvoie au superviseur). |
| D17 | traité | §6.0 l.997-1000 et item « Équivalence » l.1055-1075 (liste rejoué / supposé). |
| D18 | traité | §6.4 l.2041-2044 ; D-66 l.482-485. |
| D19 | traité pour les deux items cités | R-COMB-2 `[x]` (l.1096) et D-44 `[~]` (l.1104). Mais un nouvel item `[ ]` sans statut apparaît (manque N7). |
| D20 | traité | §6.0 gate 5 l.871-875 ; §6.2 L1 « Fichiers » l.1336 ; `R_NAIS.md` correction 9. L'inventaire se fait par nom (`*Marcher`) et reste « à re-vérifier à l'entrée du lot ». |
| E21 | partiellement | Règle « Gain » l.769-775 ; D-79 requalifiée ; L1, L9, D15 et D-RI annotées. Restent « borne » dans D3 (l.1851, l.1858) et dans la ligne R-P6 du §6.1 (l.1219). |
| E22 | traité (D6 ouverte) | LM × L6a = +22 194 (§6.2 L6a, D6 l.1877-1883 ; `R_COMB_2.md` §9). |
| E23 | partiellement | D-113 et la mesure préalable sont ajoutées (l.1108-1109, l.1695). Mais la généralisation reste écrite à trois endroits : §6.1 « Non retenus » l.1225 (« ces 4 598 fermetures sont factices »), §6.5.1 R-L1 (b) l.2077 (« établi … FACTICES ») et RAPPORT §8 (« (b) … sont factices », établi). S'y ajoute le chiffre « 1 008 faux sains connus » (manque N3). |
| E24 | traité (D21) | D-102 l.684-686 ; D21. Le troisième écart (`python3 --version` de R-COMB-2) est consigné. |
| E25 | traité pour la campagne | Les entrées de la campagne ont leur ligne vide. Trois entrées venues de `feat/v75` restent collées (`thought_log` l.114210, l.114221, l.114255). |

## 2. Nouveaux manques

**N1. La vague 1 n'est jamais mesurée comme combinaison.**
- Les configurations de R-COMB-2 (`r_comb2_configs*.tsv`) ne contiennent que `full`, `full-X`, `full+WO` et les leviers seuls. Aucune ne réunit les composants seuls, sans LS, L1a ni LP.
- Pourtant l'ordre de la vague 1 (§6.5.4) et l'indicateur D1 « publié à chaque vague » s'appuient sur les marginales de C11, qui contient des leviers de la vague 2, deux lots « pas prêts » (L2, L6b) et un oracle (L9).
- Mesuré : la marginale de LM inclut une interaction de +31 399, et L1a ne s'allume que sous LM (D-106).
- Conséquences :
  - « Gate 2 de C11 tenu 21/21 » ne prouve rien pour la vague 1 ;
  - l'indicateur après la vague 1 est inconnu ;
  - le −916 de L2 sur `1c4c63c2` est mesuré dans un monde qui contient LS et L1a.
- Le jour où le gate 2 juge la vague 1 lot par lot, la combinaison qui sera réellement livrée n'aura jamais été mesurée.

**N2. Le dénominateur fixe ne suit toujours pas sa règle (point C10 bis).**
- La règle (§6.0 l.797-799 ; `R_COMB_2.md` §3) exclut « les sites de position rejetés » et « les oracles de lots retirés ». Or `r_comb2_denominateurs_classes.tsv` classe `full+WO`, `WO`, `L6a+WO`, `L7`, `full-L7*` et F12 comme « inclus ».
- Mesuré : `full+WO` (D-110, rejeté) fixe le maximum de façon stricte sur six films :

  | Film | Fixe avec `full+WO` | C11 sans WO |
  |---|---|---|
  | `0797ce72` | 225 114 | 221 485 |
  | `60ae07c4` | 359 291 | 359 157 |
  | `084a804d`, `111fa685`, `4f77afc1`, `bcb6d393` | +1 à +22 | — |

- Effet au plus 3 788 records sur le corpus, soit au plus 0,05 % ; il est petit et tire vers le bas.
- « La forme large donne le même maximum » n'est donc vrai que parce que la forme stricte n'exclut pas ces variantes.

**N3. « 1 008 faux sains connus » contredit D-113.**
- Le chiffre apparaît §6.0 l.828, §6.2 L0 l.1694, §6.5.3 l.2160, RAPPORT §9 et `R_COMB_2.md` §0.2.
- D-113 dit que, parmi les 196 paquets inexpliqués, 32 sont sains au juge et seulement « probables » factices. Le compte établi est donc au plus de 976 ; ces 32 restent probables seulement.

**N4. Le correctif de B9 n'a été appliqué qu'à L3a.**
- La nouvelle règle du gate 2 (l.829-832) impose de publier les sains perdus en brut, et « 0 sain perdu » seulement si les deux comptes sont nuls.
- Mesuré dans `r_comb2_gate2_par_film_C11.tsv`, colonne `seul_perdus_sains` :

  | Lot | Sains perdus en brut (seul) | Ce que dit le plan |
  |---|---|---|
  | L1a | 473 (`4f77afc1` 307, `396cfc92` 120, `51ebbc0f` 46) | « tenu », rien sur les pertes |
  | L6a | 34 (`60ae07c4` 33) | rien |
  | L4a | 7 | §6.1 écrit « brut −0 » dans la même ligne que le +1 403 de R-COMB-2 |
  | L9 | 3 | rien |
  | LS | 1 | rien |
  | LP | 1 (`1c4c63c2`) | « 0 perdu » au §6.5.2 l.2113, au §6.5.1 (R-P3) et au RAPPORT §8 |

- D-114 ne liste pas ces lots.

**N5. L0 change la marche elle-même, pas seulement le juge.**
- Établi par lecture : `debutDeLaListe` (`debut_de_liste.go:42-44`) appelle `marchLocateStrict` puis `debutParFermeture`, qui lit `Fermee` (l.157). Ce chemin sert la carte de référence, le site de cuisson de LS (deuxième étape de son ordre retenu, « fermeture par NEW de tête ») et L1a.
- Conséquences :
  - « L0 : 0 fermeture gagnée » (§6.1) est supposé, pas mesuré : un `Fermee` plus strict peut faire choisir une autre tête ;
  - les chiffres de LS, de C11 et des sains de référence sont mesurés sous l'ancien `Fermee` ;
  - seul L1a est signalé à remesurer (D-107).
- La mesure préalable « R-COMB-2 sous le juge de L0 » (l.1114) ne joue L0 que comme juge, pas comme changement de `debutParFermeture`.

**N6. Gate adverse absent sur R-COMB-2 et la surcouche unique.**
- Les deux notes n'ont pas de vérificateur (§6.5.1 dernière ligne ; aucune mention dans `SURCOUCHE_UNIQUE.md`). Le §6.5.6 déclare pourtant « traités », sur leurs seuls chiffres, les points A4, B5 à B9, C10, C11, C13 et E21 à E23.
- Incohérences internes que j'ai relevées dans `R_COMB_2.md` :
  - §0.3 donne −913 / −16 078 pour L2 sur `1c4c63c2` : c'est la somme du build HI_1_10_0. Le §5.3 et le TSV donnent −916 / −16 120 pour le film ;
  - §5.3 donne +4 / +18 pour `084a804d` en marginal ; le TSV donne +2 / +18 ;
  - §13 annonce une sonde de signatures de 98 lignes ; `wc -l` en compte 91 ;
  - §10 contient la phrase du point A2.

**N7. Statuts et invariants du juge incohérents.**
- Item `[ ]` sans statut au §6.0 l.1106 (« Mesures préalables proposées… à lancer sur GO »). C'est la règle 3 de `plan-execution`, le même défaut que D19.
- L0.8 est « candidat, à mesurer d'abord » (l.1723), mais aucune mesure préalable ne le planifie. Pourtant le gate 2 se juge avec « L0.6 à L0.8 compris » (l.826).

**N8. Risque de décision cachée au regard de D-VEH (FERME : `ti=43` en vague 1).**
- L2 « pas prêt », à réparer selon D19 (a). Si la réparation échoue, L2 sort de la vague 1, ce qui contredirait D-VEH.
- Ni D19 ni D18 ne disent que ce cas exige une décision de l'utilisateur.

**N9. Interdits et traçabilité.**
- L'entrée de traitement de `thought_log` déclare une écriture dans `/tmp`. C'est hors worktree et hors scratchpad, et ce n'est pas consigné dans D-102 ni D21.
- Le journal §4 n'a toujours pas d'entrée pour l'intégration des R-*, la surcouche unique, R-COMB-2 ni ce traitement (voir D16).
- La tête poussée `df228c24c` ne porte aucun des correctifs de cette critique.