# Etat des lieux i18n : preparer l'ajout d'une 3e ou 4e langue

Date : 2026-10-23. Statut : base de travail pour un plan, pas un plan.
Perimetre : mesures sur le code a la date ci-dessus (branche release/v8.0.4). Chaque chiffre
est reproductible avec la commande indiquee ; les rechercher a nouveau avant de planifier
(doctrine RE-VERIFIER).

## 1. Constat en une page

L'application a deja une bonne fondation : type `Locale` unique, manifests TOML avec ICU,
typage `Record<Locale, T>`, garde-rails. Ce qui bloque une 3e langue n'est pas la structure
generale, c'est qu'elle suppose **exactement deux langues a six endroits**, dont plusieurs
echouent en silence (aucune erreur de compilation, repli sur fr ou en).

Correction d'une affirmation faite en conversation : il EXISTE un catalogue de messages
(ADR 0003 : 23 manifests TOML, 3 648 cles `{ fr, en }`, formatage ICU via `intl-messageformat`,
generateur `apps/web/scripts/build_i18n_manifests.mjs`, 217 fichiers consommateurs). Le point
faible est ailleurs : la moitie du texte reste hors manifests (section 3) et le generateur
code `['fr', 'en']` en dur.

## 2. Les six points de couplage a deux langues

| # | Couplage | Echec si on ajoute `es` | Visible a la compilation ? |
|---|---|---|---|
| A | Ternaires `locale === 'en'` / `'fr'` cote web | repli silencieux sur fr (ou en) | non |
| B | Champs API suffixes `_fr` / `_en` (openapi, Go, web) | `es` impossible sans changer le contrat | partiel (types generes) |
| C | Mappings TOML des titres (`labels = { en, fr }`, `*_fr`) | pas de place pour `es` | non (chargeur Go) |
| D | Generateur de manifests (`REQUIRED_LOCALES = ['fr','en']`, refuse toute autre locale) | build en echec ou `es` ignore | oui (echec du build) |
| E | Backend : `resolveLocale` ne renvoie que `"en"` ou `"fr"` ; ~36 comparaisons dans 22 fichiers Go | `es` demande est servi en fr | non |
| F | Formatage : locales Intl en dur (`'fr-FR'`, `'en-GB'`, `'en-US'`), virgule decimale par `locale === 'fr'` | nombres et dates au format fr/en | non |

## 3. Inventaire chiffre

### A. Ternaires cote web (hors tests, hors fichiers generes)

- 88 fichiers, 156 occurrences de `locale|lang === 'fr'|'en'`, `!==`, `isFr`.
- Detail : `locale === 'en'` 96 occurrences / 64 fichiers ; `locale === 'fr'` 38 / 24 ; `isFr` 21 / 2.
- Plus gros fichiers : `components/ui/SessionMultiSelect.tsx` (12), `features/prestige/components/ArcPresetPicker.tsx` (10),
  `features/friends/AddFriendFlow.tsx` (9), `features/admin/format.ts` (6), `features/match-view/MatchViewPage.tsx` (5).
- Quatre sortes d'usage observees sur echantillon (a classifier exhaustivement) :
  1. choix d'un dictionnaire : `TABLE[locale === 'en' ? 'en' : 'fr']` (le plus courant ; au moins 8 helpers `localeKey`/`pickLocale` distincts : compare, media, palmares, friends, FilterOmnibar...) ;
  2. choix d'un champ de l'API : `locale === 'fr' ? m.title_fr : m.title_en` ;
  3. format : `locale === 'fr' ? abs.replace('.', ',') : abs`, `locale === 'en' ? 'en-GB' : 'fr-FR'` ;
  4. texte inline : `locale === 'en' ? 'Not available for this title' : 'Indisponible pour ce titre'`.
- Les usages 1 et 4 se corrigent mecaniquement ; 2 depend de B ; 3 depend de F.
- Reproduire : `rg -c --glob='!*.test.*' "(locale|lang|Locale|appLocale|shellLocale)\s*(===|!==)\s*'(fr|en)'|\bisEn\b|\bisFr\b" apps/web/src`
- Fait connexe : 43 fichiers portent `Record<Locale|ManifestLocale|AdminLocale|MatchViewLocale, ...>` (ceux-la sont surs : le compilateur signalera `es`) ; 65 fichiers ont des objets litteraux `fr:` (a verifier un par un : certains sont typés, d'autres non).

### B. Champs `_fr` / `_en` dans le contrat d'API

- `apps/go-api/api/openapi.yaml` : 47 proprietes, 25 noms distincts (`label_en` 11, `description_fr` 4, `label_fr` 3, `nom_fr/nom_en`, `name_fr/name_en`, `map_name_fr` 2, `title_fr/title_en`, `tier_name_fr`, `next_rank_name_fr/en`, `reason_key_fr/en`...).
- Usage web de ces champs hors `generated.ts` : `label_en` 34, `label_fr` 20, `description_fr` 17, `map_name_fr` 12, `name_fr` 11, puis ~10 autres noms (a lister).
- Go : 615 occurrences de `LabelFR|LabelEN|NameFR|NameEN|DescFR|DescEN|TitleFR|TitleEN`, reparties surtout dans `internal/domain/` (achievements, career, compare, filters, leaderboard, match_history, timeseries, tactical_page...), `internal/games/mappings/` (6 chargeurs), `internal/api/wire/registry*.go`.
- Deux modeles coexistent cote Go : (i) le serveur resout la locale de la requete (`ctxkeys.Locale`, header `X-LevelUp-Locale`) et renvoie UN libelle (rang de carriere, nom d'arme, CSR), (ii) le serveur renvoie les deux (`_fr` + `_en`) et le client choisit. Le modele (i) passe a N langues sans changer le contrat ; le modele (ii) non. **Question centrale du plan : unifier vers (i) ?**
- DuckDB : des colonnes `name_en` / `name_fr` existent (`internal/games/weapons/name_labels.go`, `labels.go`) ; une migration a deja retire des colonnes inertes (`steps_metadata_purge_*`). Inventorier ce qui reste (donnees reelles par langue = table `locale, key, label` plutot que colonnes).
- Reproduire : `grep -cE '^\s+"?[a-z_]+_(fr|en)"?:' apps/go-api/api/openapi.yaml` ; `rg -l "LabelFR|LabelEN|NameFR|NameEN|DescFR|DescEN|TitleFR|TitleEN" apps/go-api`.

### C. Configuration des titres (TOML)

- 14 fichiers sous `config/titles/` portent des champs de langue : `halo_infinite/mappings/{fields (76), assets (56), weapon_names (53), replay_labels (31), vehicle_weapons (30), outcomes (4)}`, `halo_5/mappings/{fields (65), weapon_names (55), assets (10), outcomes (4), asset_labels_fr (4)}`, `synthetic_title_b/...`.
- Aussi `halo_infinite/milestones/catalog.toml` et `challenges/templates.toml`.
- Forme dominante : `labels = { en = "Kills", fr = "Frags" }` (table par langue : extensible sans changer de schema). Forme a corriger : suffixes `*_fr` / `*_en` (non extensible) et `halo_5/mappings/asset_labels_fr.toml` (nom de fichier = langue).
- Les chargeurs Go (`internal/games/mappings/loader*.go`) mappent ces tables vers des structs `FR/EN` : c'est le point de contact avec B.

### D. Generateur de manifests

- `apps/web/scripts/build_i18n_manifests.mjs` : `REQUIRED_LOCALES = ['fr', 'en']` ; une feuille doit avoir fr ET en, toute autre locale est refusee (« unsupported locales »).
- Il faudra decider : locale obligatoire vs optionnelle (repli en/fr), et comment une langue partiellement traduite est livree.
- `lib/i18n/locale.ts` : `KNOWN_LOCALES = ['fr', 'en']` (point d'entree du type, du routage `/{-$lang}/`, du header API, de `pageTitle.ts`).

### E. Backend

- `internal/api/middleware/title.go` : `resolveLocale` -> `"en"` ou `"fr"` (defaut fr). Ajouter une langue = etendre ce `switch` ET tous les consommateurs de `ctxkeys.Locale`.
- ~36 comparaisons `lang|locale == "fr"|"en"` ou `case "fr"|"en"` dans 22 fichiers Go (hors tests). `ogmeta` (previews sociaux) gere sa propre locale (`ParseLocale`, defaut FR).
- Reproduire : `rg -l --glob='*.go' --glob='!*_test.go' "(lang|locale|Lang|Locale)\s*(==|!=)\s*\"(fr|en)\"|case \"fr\"|case \"en\"" apps/go-api`.
- Les appels sortants vers les API Xbox/Halo passent deja `Accept-Language` en parametre : bon point pour obtenir des noms localises a la source.

### F. Formatage

- ~65 occurrences de `'fr-FR'|'en-US'|'en-GB'` dans le web, dont le noyau `lib/formatters/{date,number,intlLocale}.ts`. `lib/formatters/intlLocale.ts` est l'endroit naturel pour centraliser (a verifier : combien de sites le contournent).
- Virgule decimale decidee par `locale === 'fr'` (ex. `features/admin/format.ts`, `features/_shared/usage/usageFormat.ts`) : a remplacer par `Intl.NumberFormat`.
- Pluriels : ICU deja disponible dans les manifests ; hors manifests, les pluriels sont ecrits a la main par langue.

### G. Texte hors manifests (inconnu a mesurer)

- ADR 0003 prevoyait de tout migrer vers les manifests ; en pratique ~91 fichiers `i18n.ts` / `*Strings.ts` / `*Text.ts` de features portent encore leurs dictionnaires `{fr, en}` (la regle ESLint `no-hardcoded-strings` ne vise que le texte JSX de 3 mots ou 15 caracteres et plus, pas les arguments de fonction ni les libelles courts).
- A mesurer : nombre de chaines dans ces fichiers, et combien sont deja en `Record<Locale, ...>` (sur) contre objets non types.
- Hors UI : e-mails, previews sociaux (`ogmeta`), messages d'erreur d'API, notifications, pages legales (`features/legal/*I18n.ts`, textes longs : traduction juridique a part).

## 4. Garde-rails existants (a etendre, pas a reinventer)

- `lib/i18n/locale.ratchet.test.ts` : interdit de redeclarer l'union `'fr' | 'en'` en position de type.
- `lib/i18n/no-hardcoded-locale-fallback.guard.test.ts`, `no-field-label-dictionary.test.ts`, `noUnknownMentions.guard.test.ts`.
- `lib/i18n/no-anglicisms.guard.test.ts` : propre au francais ; a cadrer (une liste par langue, ou francais seulement).
- ESLint `@levelup/no-hardcoded-strings` (error) et `tools/lint-no-hardcoded-fields.mjs`.
- Manque : un garde-rail contre `locale === 'fr'|'en'` en dehors du noyau (regle CLAUDE.md n°6 : a la 3e copie, centraliser ET garde-rail ; ici on en est a 156).

## 5. Decisions a trancher avant d'ecrire le plan

1. **Combien de langues visees, lesquelles, et traduction faite comment** (a la main, outil externe, brouillon automatique relu) ? Determine l'etape « sortir les chaines du code ».
2. **Contrat d'API** : tout passer en modele « le serveur renvoie le libelle de la locale demandee » (supprime B), ou garder les champs `_fr/_en` et en ajouter par langue (non recommande a 4 langues) ?
3. **Langue partielle** : une traduction incomplete est-elle livrable avec repli (en, puis fr) ? Si oui, le generateur n'exige plus toutes les locales.
4. **Stockage des libelles Halo dans DuckDB** : colonnes par langue ou table `(locale, cle, libelle)` ; impact migration et lecture `_latest` si concerne.
5. **Francais seulement** pour le garde-rail anti-anglicismes, ou un equivalent par langue ?
6. **Pages legales** : traduire, ou garder fr/en et renvoyer les autres langues vers l'anglais.

## 6. Ordre de travail possible (a affiner dans le plan)

1. Mesure exhaustive et classification des 156 ternaires (A) et des chaines hors manifests (G).
2. Centraliser : un seul module `pickLocalized` / `localeKey`, `intlLocale` et `formatNumber` (noyau F), puis garde-rail anti-`locale ===` hors noyau.
3. Convertir les ternaires de type 1 et 4 (mecaniques), puis 3.
4. Contrat d'API (B) : decision 2, puis migration du Go, de l'openapi, de `generated.ts` et des usages web.
5. TOML (C) : unifier les formes `*_fr/_en` vers des tables par langue ; chargeurs Go.
6. Backend (E) : `resolveLocale` pilote par la liste des langues connues, un seul point d'etat.
7. Generateur (D) : locales obligatoires/optionnelles, liste partagee avec `KNOWN_LOCALES`.
8. Preuve de bon fonctionnement : ajouter une langue fictive (`xx`) dans une branche jetable et verifier que `tsc`, les tests et la page ne laissent aucun repli silencieux.

## 7. Risques et pieges deja connus du projet

- Regle CLAUDE.md n°1 : UI en FR et EN, parite par typage `Record<Locale, T>` ; a garder vraie pendant la migration.
- Doctrine title-agnostic : les libelles Halo viennent de `config/titles/{slug}/mappings/` ; ne pas deplacer de libelle de jeu dans les manifests web.
- Pas de flag « OFF pour plus tard » : une langue ajoutee doit etre complete ou explicitement en repli documente.
- Tout changement de schema DuckDB de metadonnees passe par une migration (CTAS-swap, voir `steps_metadata_purge_*`) ; jamais de `ALTER ... DROP COLUMN` sous PK.
- Un push sur `main` deploie en prod : travailler sur une branche `release/vX.Y.Z`.
