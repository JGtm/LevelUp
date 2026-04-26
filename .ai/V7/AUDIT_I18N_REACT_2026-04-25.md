# AUDIT_I18N_REACT_2026-04-25.md — Audit i18n React (apps/web/src/) en vue de Phase D du plan multi-titres

> Audit conduit le 2026-04-25 sur la branche `feat/accessibility-okabe-ito` — pré-requis recommandé du plan `.ai/PLAN_MULTI_TITLE_ADAPTERS_AND_MAPPINGS.md`.
>
> **Objectif** : inventorier l'état réel de l'i18n React pour calibrer Phase D (consommation de l'endpoint `/field-mappings` + nettoyage des strings métier hardcodées). Document descriptif uniquement, **aucune modification de code**.

---

## TL;DR

1. **Pas de react-i18next** dans le repo — pattern artisanal **dictionnaires typés FR/EN par feature** (`feature/i18n.ts`).
2. **9 fichiers i18n totalisant ~2 500 lignes**, répartis : 6 fichiers monolithiques (`features/{compare,help,lab,media,palmares,settings}/i18n.ts`) + 3 fichiers segmentés home (`features/home/{kpi,highlights,spartanIdentity}.i18n.ts`).
3. **Locale globale** : store Zustand `appShellStore.locale` (`'fr' | 'en'`, défaut `'fr'`) — bonne nouvelle, pas de plomberie i18n à inventer.
4. **Strings métier candidates au TOML** : ~150–200 clés identifiées (combat, match, career, outcomes), réparties dans ~7 fichiers (3 i18n.ts + 4 fichiers de page).
5. **Anti-pattern résiduel** : ternaires inline `locale === 'en' ? 'X' : 'Y'` partiellement éradiqués mais encore présents dans des composants UI (`timeseries-scatter.tsx`, `timeseries-kda-bars.tsx`, `MatchViewPage.tsx`, `SquadContributionsPage.tsx`).
6. **Effort Phase D estimé sur cette base réelle** : 4–5j (vs 3–4j initialement, marge +30 % comme prévu en §18 du plan multi-titres).
7. **Aucun risque bloquant détecté** — la migration vers `useFieldLabel(key)` est mécanique, pas conceptuelle.

---

## 1. Méthodologie

1. recherche `*.i18n.ts` et `*/i18n.ts` dans `apps/web/src/` ;
2. recherche litteral des FieldKey FR/EN candidats (`'Kills'`, `'Deaths'`, `'Eliminations'`, `'Précision'`, `'Accuracy'`, `'Morts'`, `'Assists'`) ;
3. inspection du store de locale (`appShellStore`) ;
4. comptage lignes + sondage de structure pour estimer la complexité de migration.

Pas d'exécution de code, pas de modification, pas de test. Tout est lecture statique.

---

## 2. Architecture i18n actuelle

### 2.1. Pattern : dictionnaires typés par feature

Aucun usage de `react-i18next`, `i18next`, `formatjs`, ni de fichiers `locales/*.json`. À la place, chaque feature qui a besoin de localiser exporte un fichier `i18n.ts` qui contient :

1. un type `XxxLocale = 'fr' | 'en'` ;
2. une interface `XxxText` décrivant la forme du dictionnaire ;
3. une constante `FR: XxxText` et `EN: XxxText` ;
4. un objet `DICTS: Record<XxxLocale, XxxText> = { fr: FR, en: EN }` ;
5. une fonction `getXxxText(locale)` qui retourne le bon dict.

Exemple type (extrait `features/home/kpi.i18n.ts`) :

```ts
const FR: KPITextDict = {
  labels: {
    matches: 'Parties',
    kda: 'FDA',
    winRate: 'Taux de victoire',
    accuracy: 'Précision',
    favoriteWeapon: 'Arme favorite',
    // ...
  },
  // ...
}
```

**Conséquence** : il n'y a **pas de fichier de locale plat à grepper**. Chaque feature a son propre objet TS imbriqué. Le lint anti-hardcode du plan multi-titres §9.5 doit donc parser AST plutôt que faire un grep naïf.

### 2.2. Source de la locale courante

```ts
// apps/web/src/stores/appShellStore.ts
locale: 'fr' | 'en'   // défaut 'fr'
setLocale: (locale: 'fr' | 'en') => void
```

Le store Zustand `appShellStore` détient la locale courante. Tout composant lit via `useAppShellStore(s => s.locale)`. Pas de `LocaleProvider`, pas de hook artificiel — c'est direct et propre.

### 2.3. Bonne pratique déjà en place

Les 3 fichiers `features/home/*.i18n.ts` (kpi, highlights, spartanIdentity) sont récents et bien structurés. Leur docstring mentionne explicitement :

> *« Centralise les chaînes hardcodées qui étaient disséminées sous forme de ternaires `locale === 'en' ? 'X' : 'Y'` dans HomePage.tsx. »*

C'est exactement la dynamique que Phase D pousse plus loin.

---

## 3. Inventaire détaillé des fichiers i18n

| Fichier | Lignes | Contenu | Catégorie pour Phase D |
|---|---:|---|---|
| `features/compare/i18n.ts` | 127 | Labels comparaison joueurs (kills, deaths, accuracy, kda, win_rate…) | **Métier — TOML** |
| `features/help/i18n.ts` | 442 | Glossaire pédagogique + release notes | **UI — i18n React** (descriptions de concepts, pas labels métier) |
| `features/lab/i18n.ts` | 606 | Labels admin/lab (titres pages, descriptions de jobs, status…) | **UI — i18n React** |
| `features/media/i18n.ts` | 135 | Labels galerie média (filtres, tris, modes) | **UI — i18n React** + qq labels mode → TOML |
| `features/palmares/i18n.ts` | 251 | Labels palmares/citations | **Mixte** (citations = à étudier ; UI = i18n React) |
| `features/settings/i18n.ts` | 598 | Labels page Settings (catégories, descriptions options) | **UI — i18n React** |
| `features/home/kpi.i18n.ts` | 90 | Labels KPI bar home (matches, kda, winRate, accuracy…) | **Métier — TOML** |
| `features/home/highlights.i18n.ts` | 159 | Labels highlights (kills, victoires, médailles…) | **Métier — TOML** (partiel) |
| `features/home/spartanIdentity.i18n.ts` | 69 | Labels identité spartan (rang, XP…) | **Mixte** (rang = career_rank asset, XP = FieldKey) |

**Total : 2 477 lignes**, dont ~30 % candidates au TOML, ~70 % restent en i18n React.

### 3.1. Frontière métier vs UI dans cet audit

| Reste en i18n React | Migré vers TOML backend |
|---|---|
| Titres de pages (« Comparaison joueurs ») | Labels FieldKey (`kills`, `accuracy`, `kda`…) |
| Boutons (« Sauvegarder », « Annuler ») | Labels Outcome (`Victoire`, `Défaite`) |
| Descriptions options Settings | Noms de modes (`Capture du drapeau`) |
| Tooltips et messages d'aide | Noms de rangs carrière |
| Messages d'erreur | Tier de médaille (`bronze`, `silver`, `gold`) |
| Glossaire pédagogique | Labels d'unité (`secondes`, `%`, `kills`) |

---

## 4. Strings métier hardcodées hors fichiers i18n

Recherche directe des littéraux candidats (`'Kills'`, `'Deaths'`, `'Assists'`, `'Eliminations'`, `'Morts'`, `'Précision'`, `'Accuracy'`) hors fichiers `*i18n*` :

| Fichier | Hits | Type d'usage |
|---|---:|---|
| `features/home/kpi.i18n.ts` | 3+ | Déjà dans i18n (à migrer dict → TOML) |
| `features/home/highlights.i18n.ts` | qq | Déjà dans i18n (à migrer dict → TOML) |
| `features/compare/i18n.ts` | qq | Déjà dans i18n (à migrer dict → TOML) |
| `components/ui/timeseries-scatter.tsx` | ≥1 | **Anti-pattern résiduel** — ternaire inline `locale === 'en' ? 'Kills' : 'Éliminations'` |
| `components/ui/timeseries-kda-bars.tsx` | ≥1 | **Anti-pattern résiduel** — ternaire inline |
| `features/squad/SquadContributionsPage.tsx` | ≥1 | **Anti-pattern résiduel** — ternaire inline |
| `features/match-view/MatchViewPage.tsx` | ≥1 | **Anti-pattern résiduel** — ternaire inline |

**Constat** : le plan multi-titres Phase D devra (1) migrer les dicts FR/EN existants vers consommation TOML, **et** (2) éradiquer les ternaires inline restants dans 4 fichiers de composants/pages.

---

## 5. Préparation Phase D

### 5.1. Liste de migration prévue (ordre suggéré)

| Priorité | Fichier / Composant | Action |
|---|---|---|
| 1 | `features/home/kpi.i18n.ts` | Remplacer `KPITextDict.labels.*` par `useFieldLabel(key)` ; conserver `units` et `pluralizers` côté React |
| 2 | `features/home/highlights.i18n.ts` | Idem (séparer libellés FieldKey vs phrases hardcodées) |
| 3 | `features/compare/i18n.ts` | Idem ; fichier petit (127L) donc rapide |
| 4 | `components/ui/timeseries-scatter.tsx` | Remplacer ternaires inline |
| 5 | `components/ui/timeseries-kda-bars.tsx` | Idem |
| 6 | `features/squad/SquadContributionsPage.tsx` | Idem |
| 7 | `features/match-view/MatchViewPage.tsx` | Idem |
| 8 | `features/home/spartanIdentity.i18n.ts` | Migrer XP/rang vers TOML, conserver le reste |
| 9 | `features/palmares/i18n.ts` | Tri citations métier vs UI |

### 5.2. Hook cible

```ts
// apps/web/src/lib/i18n/useFieldLabel.ts (à créer Phase D)
import { useQuery } from '@tanstack/react-query'
import { useAppShellStore } from '@/stores/appShellStore'

export function useFieldLabel(key: FieldKey): string {
  const locale = useAppShellStore(s => s.locale)
  const { data } = useQuery({
    queryKey: ['field-mappings', currentTitleSlug(), locale],
    queryFn: () => fetchFieldMappings(currentTitleSlug(), locale),
    staleTime: Infinity,
  })
  return data?.fields[key]?.label ?? key
}
```

### 5.3. Lint anti-hardcode

Le script `tools/lint-no-hardcoded-fields.mjs` (cf. plan multi-titres §9.5) doit :

1. parser AST TS via `@typescript-eslint/parser` (pas de grep naïf, à cause des dicts FR/EN imbriqués) ;
2. construire la liste des `FieldKey` depuis `config/titles/halo_infinite/mappings/fields.toml` au runtime ;
3. détecter les littéraux string qui correspondent **exactement** aux labels FR ou EN d'un FieldKey ;
4. **autoriser** les fichiers `*i18n*.ts` pendant la transition (whitelist), retirer la whitelist en fin de Phase D.

### 5.4. Effort réévalué pour Phase D

| Sous-tâche | Effort |
|---|:---:|
| Création hook `useFieldLabel` + tests Vitest | 0.5j |
| Création endpoint backend consommé (post-Phase A multi-titres) | inclus dans Phase A |
| Migration dicts existants (3 fichiers home + compare) | 1j |
| Éradication ternaires inline (4 fichiers) | 1j |
| Migration palmares/spartanIdentity | 0.5j |
| Création + ajustement lint custom | 0.5j |
| Tests E2E + golden frontend FR/EN | 1j |
| Buffer corrections | 0.5j |
| **Total Phase D** | **5j** (vs 3–4j initialement, marge +30 % confirmée) |

---

## 6. Risques et points d'attention

| Risque | Probabilité | Impact | Note |
|---|:---:|:---:|---|
| Dicts FR/EN imbriqués cassent le grep naïf | Certain | Faible | Lint AST obligatoire (cf. §5.3) |
| Pluralization (`matches: (count) => string`) non couverte par TOML | Certain | Moyen | Garder côté React, pas dans TOML |
| Help/Glossary contient des concepts métier (KDA, accuracy…) sans être un FieldKey | Possible | Faible | Reste en i18n React, le glossaire est éditorial |
| Composants Plotly réutilisent `tokenCssVar` mais avec libellés inline | Détecté | Faible | Migration ternaire → `useFieldLabel`, mécanique |
| Suppression dicts FR/EN casse une feature en prod | Faible | Haut | Migration 1 fichier à la fois + golden frontend FR/EN |

---

## 7. Décisions complémentaires identifiées (à acter en début Phase D)

1. **Pluralization** : le pattern actuel `(count: number) => string` reste **côté React**. TOML = labels, pas grammaire. → décision implicite (cf. plan §6.9).
2. **Help/Glossary** : reste 100 % en i18n React, c'est de l'éditorial. → confirmé.
3. **Lab admin** : reste 100 % en i18n React (aucune string métier joueur). → confirmé.
4. **Settings** : reste 100 % en i18n React (catégories de paramètres ≠ FieldKey). → confirmé.
5. **Citations / Palmares** : à étudier au début Phase D — chaque citation a un label FR/EN qui pourrait soit rester en i18n React soit basculer en TOML/DB selon si le produit la considère comme un asset versionné.

Décision recommandée pour **5** : citations = asset (comme médailles). Reste en DB (`match_citations` côté player, `medal_definitions` côté metadata), pas en TOML, pas en i18n React. À confirmer au démarrage Phase D.

---

## 8. Conclusion

L'i18n React du projet est **sain** : pas de plomberie héritée d'un framework lourd, dictionnaires typés clairs, locale globale propre via Zustand. Phase D ne réinvente rien : elle déplace seulement les libellés métier (~150–200 clés) vers le backend TOML, et éradique 4 ternaires inline résiduels.

**Aucun risque bloquant**. L'estimation 5j Phase D est réaliste avec marge +30 % comme prévu en §18 du plan multi-titres.

**Pré-requis confirmé** pour démarrer Phase A multi-titres : aucun. L'i18n React n'a aucune dépendance dure sur le backend actuel ; l'introduction de `useFieldLabel` se fait sans casser quoi que ce soit.

---

## 9. Documents liés

1. `.ai/PLAN_MULTI_TITLE_ADAPTERS_AND_MAPPINGS.md` — §9.5 (lint), §6.8/§6.9 (frontière TOML/i18n React), §18 (effort Phase D).
2. `apps/web/src/stores/appShellStore.ts` — store de locale.
3. `apps/web/src/features/home/kpi.i18n.ts` — pattern de référence à suivre pour Phase D.
