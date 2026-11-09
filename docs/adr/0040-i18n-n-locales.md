# ADR 0040 — i18n for N languages: one list of languages, no code change to add one

Status: accepted (2026-10-26); implementation tracked in `.ai/V8/PLAN_I18N_MULTILANGUE.md` (steps 1 to 11).
Amends: ADR 0003 (i18n via TOML manifests): the manifest generator no longer hard-codes `fr` and
`en`; the languages come from `config/locales.toml`. The manifests stay the home of UI strings.
Relates to: ADR 0011 (canonical / semantic / asset-URL adapter boundary), ADR 0026 (append-only
tables: `match_skill_rank.tier_fr` stays in place, see D4).

## Vocabulary

- **Locale**: the primary language subtag in lower case (`fr`, `en`, `es`). It is the key of every
  per-language table, URL segment, header value and cache key.
- **Intl tag**: the BCP-47 tag handed to `Intl.*` (`fr-FR`, `en-GB`) for number and date formatting.
- **Halo tags**: the ordered BCP-47 tags the Halo API and `asset_translations` use for a locale
  (`["es-ES", "es-MX"]`); a locale may need several.
- **Required locale**: a locale every UI string must cover. A listed, non-required locale is still
  complete (D3), but its absence never blocks the build.
- **Game name**: a name that belongs to the game (weapon, medal, map, mode, playlist), as opposed
  to a string written by the product.

## Context

The application supports two languages, and the assumption is spread over six couplings: the web
locale list (`KNOWN_LOCALES`), the manifest generator (`REQUIRED_LOCALES = ['fr', 'en']`), 157
ternaries `locale === 'en' ? ... : ...` in 88 web files, 36 comparisons in 22 Go files, 25 `*_fr` /
`*_en` field names in the OpenAPI contract (580 Go occurrences of the matching identifiers), and
the hard-coded `fr-FR` / `en-US` Intl tags. Two defects recur: a third locale compiles and is then
served in French through the `locale === 'en' ? 'en' : 'fr'` normalisers (27 copies), and the
defaults disagree (`resolveLocale`, `ctxkeys.Locale`, `domain.SessionData` and the web boot fall
back to `fr`; `medalLangCode` falls back to `en`).

Two facts already in the code shape the decision: the Halo names exist in 14 languages at the
source (`domain.TargetLanguages`, table `asset_translations (…, lang, name)`, command
`populate-assets`), and the feature dictionaries are already typed `Record<Locale, T>`, so the
compiler reports a missing language there. The silent leak is in the normalisers, not in the
dictionaries.

## Decision

Adding a language is one entry in `config/locales.toml`, translation files, and no code change. A
language that is incomplete is refused by the build or the tests; it is never served through a
silent fallback.

**D-1. One source, two generated artefacts.** `config/locales.toml` holds `default` (`fr`),
`fallback` (`en`, the game-name fallback), and one table per language: `code`, `intl`, `halo`
(ordered), `required`. The generator that already builds the manifests
(`apps/web/scripts/build_i18n_manifests.mjs`, through `locales.mjs`) reads it and emits
`apps/web/src/lib/i18n/generated/locales.ts` (`KNOWN_LOCALES`, `DEFAULT_LOCALE`,
`FALLBACK_LOCALE`, `INTL_TAG`) and `apps/go-api/internal/i18n/locales_gen.go` (constants, ordered
list, intl and halo tables). Go does not read the TOML at runtime. Both files are committed; a
freshness check (`tools/check-generated-types-fresh.mjs`, run by the web test suite) and a Go test
that re-reads the TOML keep them equal to it, so the two lists cannot diverge.

**D-2. One normaliser per side.** Web: `lib/i18n/locale.ts` owns `normalizeLocale()` (unknown or
empty goes to `DEFAULT_LOCALE`, never a binary `en`-or-`fr`) and `intlLocale()` reads `INTL_TAG`.
Go: package `internal/i18n`, domain layer, no internal dependency: `Locale`, `Locales()`,
`Default`, `Fallback`, `Parse` (trim, lower case, primary subtag: `en-US`, `en_us`, `EN` give `en`;
unknown or empty gives `Default`), `FromContext`, `IntlTag`, `HaloTags`. The context key lives in
`internal/i18n` and `ctxkeys.WithLocale` / `ctxkeys.Locale` delegate to it: `ctxkeys` imports
`domain`, which imports `i18n`, so the key cannot live in `ctxkeys` without a cycle. Every default
(`resolveLocale`, session, OG previews, boot) is `i18n.Default` / `DEFAULT_LOCALE`.

**D-3. The server returns the label in the request language (contract).** The 25 `*_fr` / `*_en`
field names leave the OpenAPI contract; each localised object exposes `label` (and `description`,
`name`) already resolved through `i18n.FromContext`. Web and API ship together (atomic break).
Every server cache of a localised response and every web query key of one carries the locale.
Go types carry a `Text map[Locale]string` with `Pick(locale)` for data, and a plain `string` in
responses.

**D-4. A language is complete or refused; game names may fall back.** Interface strings: every
listed language covers 100 % of the manifests and dictionaries, checked at build time (error naming
the key and the language); `fr` and `en` are required. Game names (weapons, medals, maps): a missing
translation falls back to `en` (`i18n.Text.Pick`), the fallback is counted by an atomic expvar counter
(`levelup.i18n.text_fallback_total`, ADR 0009; `Text.Resolve` also reports it, so a caller that holds a
context and the label key logs it with `slog.DebugContext`), and no translation is ever invented.
Product-authored text that belongs to neither (legal pages, help, release notes) is
the exception of D-6.

**D-5. Halo names live in one table per language.** Names the Halo API provides go to
`asset_translations` / `medal_translations` (existing tables; the Halo 5 `_fr` / `_en` columns migrate
there) and are read through the Halo tags of `locales.toml`. Names curated by hand (weapons, fields,
outcomes, milestones, challenges, arc presets) are TOML tables `{ code = "..." }`, no suffix.
`match_skill_rank.tier_fr` is no longer read or written (the label is derived from the rank code and
the TOML); the column stays, inert, because rebuilding an append-only table (ADR 0026) for an unused
column is not worth the risk.

**D-6. Product texts that are not translated.** Legal pages, help and release notes are translated
in `fr` and `en` only; any other locale gets the English version with a visible "available in
English" notice. This is the single written exception to D-4.

**D-7. The anti-anglicism guard stays French-only.** `no-anglicisms.guard.test.ts` encodes a French
product rule ("série", not "streak"); no equivalent is created per language.

**D-8. Ratchets before migration.** Three counters are frozen at the measured value and may only
fall: web comparisons `=== 'fr'|'en'` outside `lib/i18n/` and `lib/formatters/` (target 0 once the
web step is done), Go comparisons `== "fr"|"en"` outside `internal/i18n` (target 0 once the Go
text step is done), and web Intl tag literals outside `lib/formatters/` and `lib/i18n/`. A pseudo
language `xx` (text derived from `en` between brackets) proves the claim on a throw-away branch:
any visible text outside the brackets is a leak.

**D-9. The replay document keeps one label per language inside the document (exception to D-3).**
The replay document is large, cached by the client, and its cooked artefacts are not re-cooked, so
resolving a label to one language on the server would re-download it at every language change and
force a re-cook for a new language. Its labels (`Label`, `WeaponLabel`, `VehicleLabel`, `CalloutZone`,
`ObjectiveObjectLife`, `VehicleWeapon`, in the stored form `film/replay` and the served form
`domain/replaydoc`) therefore carry an `i18n.Text`, and their JSON is flat: one member per language of
`config/locales.toml`, at the same level as the other fields and at the place where `en` / `fr` used
to be written (`i18n.MarshalFlat` / `UnmarshalFlat`; no sub-object). For `fr` and `en` the bytes are
those of the two former fields, so cooked artefacts read back unchanged. The contract generator
(`openapigen`) turns the Go marker member `i18n_text` into one `string` property per language. The web
reads such a label only through `pickText(text, locale)` (language, else `FALLBACK_LOCALE`, else
empty; guard `replay-label-access.guard.test.ts`). An artefact cooked before a language was added
lacks its member and reads through the fallback (D-4, game names). The reflected shape of these
types is part of `testdata/document_shape.golden`: it is independent of the language list, so adding a
language never changes it.

**D-10. Messages written by the server live in one embedded catalogue.** Text the server composes
(coach arcs and synthesised challenges, impact badges, tier and chain names, season and currency
names, filter labels, link previews, release-note headings) is a leaf of
`internal/i18n/messages/*.toml`: one table per message, one entry per language of
`config/locales.toml`, variables written `{name}`. `i18n.Msg(locale, key, params)` renders it,
`i18n.MsgText` renders every language into an `i18n.Text` for data fields resolved later, `i18n.MsgAll`
lists the forms of a message in every language. The completeness test fails on a language missing from
any message or on a message whose variables differ between languages; an `archlint` guard requires
every call to name a catalogue key literally (or a literal prefix) and every catalogue key to be
called. At run time a missing language falls back to the game-name fallback (`levelup.i18n.message_fallback_total`)
and an absent key renders the key itself and is counted and logged once. Plural forms are not
implemented: no message of the first inventory needs one, and unused code is dead code; the first
message that does adds a `one` / `other` leaf shape together with its language rule.

**D-11. Text stored once per language is resolved on read.** A notification or a coach proposal keeps
its parameters as JSON in a player database. A label it carries is written once per language under
`<base>_<code>` (`title_fr`, `title_en`, `title_es`) by `Text.AddParams`; the server replaces each
family with one `<base>` key in the request language when it serves the row (`i18n.ResolveParams`,
applied by `notifications.LocalizeParams` and `coach_advisor.ResolveReasonParams`; the families are
declared in one list per owner). The web never sees a pair of per-language keys. The stored shape is
the current shape, so rows written before a language existed are read through the fallback chain
(language, fallback, default, first language filled) and need neither a migration nor a dated read
fallback. A language added later appears in new rows only; old rows are served in the fallback
language until they expire.

**D-12. Documents are one file per language, falling back to the fallback language.** Release notes
read `docs/RELEASE_NOTES.md` for the fallback language and `docs/<CODE>/RELEASE_NOTES.md` for another
language when the file exists; otherwise the fallback file is served after a notice written in the
requested language (D-6), counted by `levelup.i18n.document_fallback_total`. Embedded demo fixtures
follow the same shape (`challenges.<code>.json`).

**D-13. Definition texts live in per-language tables, read through one rule.** The names and
descriptions of medals, commendations, team colours, achievements (and their locked text) and
milestones are rows of `medal_translations` / `asset_translations` (`asset_type` `commendation`,
`team_color`, `achievement`, `achievement_locked`, `milestone`; `lang` is the first Halo tag of the
language), no longer `_fr` / `_en` column pairs. Weapon names are rows `(title_slug, weapon_key, lang,
name)` of `weapon_name_labels` and `(weapon_id, lang, name)` of `weapon_label_names` (`lang` is the
locale code). Migrations `metadata_definition_texts_to_translations_v1` and
`metadata_weapon_names_to_translations_v1` copy every non-empty text without loss, then drop the
columns (a translation row already present for an id and a language wins). Readers return an
`i18n.Text` per definition and choose the language with `Text.Pick`: the requested language, then the
fallback language, never a translation made up. `i18n.UsesFallbackColumn` and the cross-language SQL
`COALESCE` chains are gone: a new language is a set of rows, not a column.

**D-14. A stored placement label carries no word of a language.** `tier_label` of a ranked row in its
placement phase is `Placement (N)` (`domain.PlacementLabel`, read back by
`domain.ParsePlacementRemaining`, which also reads the older `Placement (N restants)` rows); the web
shows its own text. `match_skill_rank.tier_fr` is neither read nor written (D-5); the label of a rank
in a language is derived from the rank code through the message catalogue (`skill.tier.*`).

**D-15. Curated texts and display names are read per language, for every known language.** The
names and descriptions of citations (`citation_mappings`), challenge templates (`challenge_template`)
and preset arcs (`preset_arc`) are rows of `asset_translations` (`asset_type` `citation`,
`challenge_template`, `preset_arc`; `lang` is the first Halo tag of the language), no longer
`_fr` / `_en` column pairs: migration `metadata_curated_texts_to_translations_v1` (same recipe as
D-13), writers `migration.UpsertAssetTexts` / `DB.UpsertAssetTexts` / `ops.UpsertAssetTexts`, readers
`LoadAssetTexts`. `career_ranks.title_en` / `subtitle_en` stay (single English source of the stored
rank name, the displayed rank names are the per-language `career_rank_translations`), and so do
`mode_name_tr.mode_en` and `mode_prefix_names.prefix_en` (technical keys of tables that already carry a
`lang` column). Readers of map, playlist, pair and game-variant names gather them for every language of
`config/locales.toml` (`assetNameTexts`, `modeTextsByLocale`, `fillUntranslated`, `pickName`), then the
request language is chosen by `Text.Pick` / `pickAcrossChain` (`i18n.Chain`): no loader names a language.
The `*_fr` columns of the shared database are not migrated; they are read as the registry value of the
default language and complete, never replace, a per-language name. A key that must not depend on the
request language (the canonical map label behind `?maps=`, the pattern to solo link) is the name in
`i18n.Default`. `archlint/i18n_halo_literal_test.go` forbids a Halo tag literal and a literal
`PreferredLangsForLocale("fr"|"en")` in `platform/duckdb` and `service` (empty allowlist).

The same rule covers the last readers that completed `fr` and `en` only. The match tiles of the home
page (`homeAssetNames`, `canonicalAssetTranslations`) gather the name of every language like the
history does, and pick the language downstream; the squad table reads `SquadRepository.LoadAssetTexts`
(one `i18n.Text` per asset) and serves its map, playlist and mode names in the request language
(`SquadMatchRow.UILocale`, no cache holds those rows). The texts of the citations are a data table per
language and per entry, `internal/ops/citation_texts.toml` (embedded like the ranked playlist labels,
because the synthetic demo seed runs without a configuration tree), validated at load time with
`Text.Validate` (every `required` language, no language outside the list): the seed writes one
`asset_translations` row per language. Outside `internal/i18n`, `i18n.LocaleFR` / `i18n.LocaleEN` name
a language only to read or write a source that exists in two `fr` / `en` columns (legacy column pairs
of a migration, `*_fr` columns of a table, a game file with two language slots, demo fixtures):
`archlint/i18n_locale_constant_test.go` is a ratchet on their number (39 on 2026-10-27, it only goes
down); any other choice of language is `i18n.Default`, `i18n.Fallback`, the request language or a loop
over `i18n.Locales()`.

**D-16. A game name is served in the request language; its key stays in the default language.** The
map, mode, playlist, game-variant and season names that the Explorer, the match history (rows, CSV
export, briefing), the global filter, the career highlights, the time series, the session page, the
synthesis, the media gallery, the engagement page, the rivalry duels and the Explorer target served in
the default language (`Pick(i18n.Default)`) follow the request language like the squad table does
(`Text.Pick(i18n.FromContext(ctx))`, the display name of the language falling back on the fallback
language, D-4). Whatever is reused as a filter value, an URL parameter, a grouping key, a stable sort or a
cache key does not move with the language: it is the name in `i18n.Default`
(`?maps=`, `?modes=`, `?playlists=`, `mode_uis`, `playlist_names`, the `map_id` of a series built without
an identifier). A field that played both roles is split: `available_playlists` / `available_maps` /
`available_modes` (Explorer summary) become `LabelValue` lists, `available_modes` / `available_playlists`
of the career highlights gain a `label`, and the options of the global filter keep `value` and relabel
`label`, the same shape the experience types already had (GH5-2, GH6-1) and `map_label` /
`map_filter_key` had on the tactical page (step 6). Ties are broken on the key (breakdowns of the
synthesis and of the briefing, map breakdown of the series), so the selection and the order of the entries do not
depend on the language; the options of a dropdown are ordered by their displayed label. The removal of the
category prefix and the overrides of a playlist name (`PlaylistLabelConfig.DisplayIn`) target the
default-language name, as everywhere else: another language gets its asset name as is. Server caches hold the names
of every language and the choice is made after the cache (the history rows, the filter rows and the canonical rows
carry an `i18n.Text`); the web query keys of these responses carry the locale (`filtersResolve`,
`filtersPreview`, `matchHistory`, `explorer`, `careerHighlightMatches`, `engagementMatch`,
`palmaresRelations`). A name that was neither in the default language nor in the request language but the
raw registry name (English for every client) is served in the request language too (pilot decision,
2026-10-27: D2 and the rule of a French UI without anglicisms outweigh the "French stays identical" constraint
of the previous item, so a French client now sees the French name): the mode of the rivalry duels and the
playlist of the LUSR gaps (admin) go through `duckdb.PairNamesIn` (`mode_name_tr`, then the asset name of
the pair, then the registry name), the map of the duels through `nomsDeCarte` in every language, the squad
v2 history labels, the `map_name` / `playlist_name` of a time-series row and the `pair_name` of a session row
through `Text.Pick`. A language without a name falls back on the fallback language (D-4). The two routes
that served raw names to nobody (`GET .../pages/career/top-matches`, the list of excluded matches) were
dead code and are deleted with their chain. The return descriptor of the match view splits label and
key too: `ContextDescriptor` (`playlist`, `mode`) and `MatchFilterSpec` gain a `label` (resp.
`playlist_labels` / `mode_labels`) shown in the page language, while `name` / `category` /
`playlist_names` / `mode_categories` stay the keys that filter and serialize into the URL (a shared link
carries keys only, so it shows the key). Index accesses by a language literal
(`m["fr"]`, `t[i18n.Locale("en")]`) outside `internal/i18n` are counted by
`archlint/i18n_locale_constant_test.go` (`localeLiteralIndexPlafond`, 0 on 2026-10-27). A title whose
registry names nothing (Halo 5) is served from `asset_translations` in every language, the fallback
language included: `fillUntranslated` sets the fallback-language name when the registry has none (it never
overwrites a registry name), so `en` is not served empty while the data exists (D-3). The `map_name` of the
tactical map cards follows the request language too (it serves as a sort key and a search term on the page,
not as a filter value: `map_filter_key` is the key). The return descriptor also carries the labels of the
global filter (Explorer table and squad page read them from the resolved filter context of their store).

## Consequences

- A new language costs a `[[locale]]` entry, the translations (manifests, feature dictionaries,
  Go message catalogue, TOML tables) and the `populate-assets` run for its Halo tags. The build
  refuses it until it is complete.
- The embedded manifests grow with each language (about half again per language at the time of
  writing). The weight is measured with the proof language before any per-language split; no
  preventive split is made.
- A web build and an API of different revisions disagree on the label fields (D-3). The deployment
  is atomic (one image), so this is a development-time concern only.
- The Go catalogue of product messages (`internal/i18n/messages/*.toml`, D-10) replaces the strings
  written inline in coach, impact, tier, season-pass, link-preview and release-note code. The Go
  language comparison ratchet is at zero and also covers the forms `ctxkeys.Locale(ctx) == "en"`,
  `strings.EqualFold` and `strings.HasPrefix` on a locale.
- Existing French and English behaviour does not change: unknown or empty inputs still resolve to
  `fr`; only inputs that were silently mapped (`en-GB`, `EN`) now resolve by their primary subtag.

## References

- Plan and steps: `.ai/V8/PLAN_I18N_MULTILANGUE.md`; survey: `.ai/V8/ETAT_DES_LIEUX_I18N_MULTILANGUE_2026-10-23.md`.
- Source list: `config/locales.toml`; generator: `apps/web/scripts/build_i18n_manifests.mjs`,
  `apps/web/scripts/locales.mjs`.
- Go core: `apps/go-api/internal/i18n/`; web core: `apps/web/src/lib/i18n/locale.ts`.
