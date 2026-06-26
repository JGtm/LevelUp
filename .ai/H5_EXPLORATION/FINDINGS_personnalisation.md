# Halo 5 — Personnalisation Spartan (emblème + bannière) : cryptum vs officiel

Sonde live JGtm 2026-06-26. Source cryptum = `haloplayer.svc.halowaypoint.com`
(SpartanToken v4). Source officielle = `www.haloapi.com/profile` (clé Ocp-Apim).

## TL;DR
- **Le projet câble DÉJÀ l'essentiel** : `appearance_persist.go` fetch appearance +
  emblème PNG + Spartan render PNG, et les cache (`emblem_image_url` / `banner_image_url`
  / `spartan_id`=service tag dans `career_progression`). Emblème + « bannière » (= render
  armure) sont donc opérationnels côté Home.
- **cryptum >> officiel pour la donnée STRUCTURÉE** (l'officiel ne renvoie quasi rien).
- **Le projet modélise seulement ServiceTag + Emblem + Company** → il DROPPE armure,
  stance, weapon skins, assassination, deathfx, voiceover, gender, + tout l'inventaire.
- **MMR : absent partout (cryptum inclus).** Aucun endpoint skill/CSR/MMR/rating dans
  l'authority SpartanStats de cryptum — mêmes endpoints qu'on utilise déjà.

## Comparaison endpoint par endpoint

| Donnée | Cryptum (haloplayer) | Officiel (www.haloapi.com) |
|---|---|---|
| Emblème (image) | `/h5/profiles/{gt}/emblem` → 302 CDN signé | `/profile/h5/profiles/{gt}/emblem?size=190` → PNG 200 |
| Spartan render (image) | `/h5/profiles/{gt}/spartan` → 302 CDN signé | `/profile/h5/profiles/{gt}/spartan?size=512` → PNG 200 |
| Appearance (JSON) | **729 o — RICHE** | 297 o — pauvre (ServiceTag+Company seulement) |
| Inventory (cosmétiques) | **67 Ko — tout le débloqué** | (aucun équivalent) |
| Preferences / Campaign / Controls | oui | (aucun équivalent) |
| MMR / skill | **non** | non |

## Appearance cryptum — contenu RÉEL (JGtm)
```json
{
  "Model": { "Gender": 0 },
  "Emblem": { "EmblemId":264, "ColorPrimary":46, "ColorSecondary":46,
              "ColorTertiary":24, "HarmonyGroupIndex":30, "HarmonyIndex":2 },
  "ModelCustomization": {
    "ArmorSuitId":1014, "HelmetId":2036001, "VisorId":3016,
    "ColorPrimary":48, "ColorSecondary":56,
    "Assassination":6016, "DeathFX":0, "VoiceOver":118000,
    "StanceRotation":20.0, "StanceZoom":1.0,
    "WeaponSkinIds": { "34195":4308, "34197":4011, "34198":4407, "44596":4207, "44600":4108 }
  },
  "StanceId":7024, "ServiceTag":"OKLM",
  "Company": { "Id":"117288b8-...", "Name":"HaloFrance" }
}
```
> Le projet ne garde que `ServiceTag`, `Emblem`, `Company`. Tout `ModelCustomization`
> (armure/visière/couleurs/skins/assassination) + `StanceId` + `Gender` est parsé-droppé.

## Appearance officielle — contenu RÉEL (JGtm)
```json
{ "Gamertag":"JGtm", "ServiceTag":"OKLM",
  "Company": { "Id":"117288b8-...", "Name":"HaloFrance" },
  "LastModifiedUtc": {...}, "FirstModifiedUtc": {...} }
```
> Ni emblème ni armure. → l'officiel ne sert QUE les images rendues + service tag/company.

## URLs CDN composées (cryptum 302)
- Emblème : `image.halocdn.com/h5/emblems/{EmblemId}_{cPrim}_{cSec}_{cTert}?width=256&hash=…`
  (ex. `264_46_46_24`). Le hash est signé → non reproductible client → on télécharge les octets.
- Spartan : `image.halocdn.com/h5/spartans/{ArmorSuitId}_{?}_{HelmetId}_{VisorId}_{cPrim}_{cSec}?width=256&crop=Full&…`
  (ex. `1014_0_2036001_3016_48_56`). Idem.

## Inventory cryptum (débloqué par JGtm)
Emblems 267 · Helmets 227 · ArmorSuits 227 · WeaponSkins 112 · Visors 63 ·
Assassinations 31 · Stances 27 · VoiceOvers 1 · DeathFXs 1. Chaque entrée porte une
`GrantDateUtc`. → base d'une page « collection / cosmétiques » si désiré (rien d'exploité).

## Verdict pour tes besoins
- **Emblème** : OK (déjà fetché + caché). Donnée structurée (EmblemId+couleurs) dispo via
  cryptum si on veut le recomposer/afficher des variantes.
- **Bannière** : H5 n'a pas de « banner » dédié ; le **Spartan render full-body** joue ce
  rôle (déjà mappé sur `banner_image_url`). Alternative « carte d'identité » = render +
  emblème + service tag + Company + SR rank, tous dispo.
- **À débloquer si on veut aller plus loin** : armure/visière/couleurs/skins (appearance,
  gratuit) + inventaire cosmétique (nouvel endpoint, déjà accessible).

## Bannière — la vraie réponse (sonde 2026-06-26)
Le « banner » actuel = le **render full-body du Spartan**. Ce n'est pas une bannière.
**Halo 5 n'a AUCUNE bannière joueur native** (pas de nameplate/backdrop comme Infinite —
absent de `appearance` et de toute la surface API). Le seul asset « bannière » de
l'écosystème H5 = la bannière de **Spartan Company** ; **mais elle n'est plus servie** :
l'endpoint company survivant (`/oban/companies/{id}`) ne renvoie que Id/Name/Creator/
Members/dates — **ni bannière, ni emblème, ni motto** (retirés par Waypoint vNext).
Vérifié sur HaloFrance (company de JGtm).

### Options réalistes (par préférence)
1. **Bannière synthétisée (recommandé)** — strip large composé côté app depuis la donnée
   qu'on fetch DÉJÀ : dégradé sur `ModelCustomization.ColorPrimary/Secondary` (couleurs
   d'armure du joueur), emblème en motif (gauche), `ServiceTag` (OKLM) + SR + tier CSR +
   `Company.Name` en texte. 100 % reconstructible, cohérent avec les tokens couleur de
   l'app, unique par joueur. C'est l'alternative honnête : H5 ne livre pas de bannière → on
   la fabrique depuis l'identité réelle.
2. **Backdrop = image de map** — image de la map la plus jouée (déjà dans `maps_catalog`)
   en fond large + emblème en overlay. Les images de maps sont larges → bonnes bannières.
3. **Emblème en grand** — layout emblème-forward (asset le plus iconique).
4. **Garder le render mais comme élément latéral** (portrait), pas comme bannière.

Note : l'endpoint officiel renvoie la **même image** pour `crop=full` et `crop=portrait`
(le crop ne différencie pas) → pas de buste exploitable tel quel.
