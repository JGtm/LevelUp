# Rapport de calibration engagement — halo_5

> Genere le 2026-07-11 15:39 UTC par `cmd/engagement-calibrate` (chantier F7 E4a). DIAGNOSTIC — n'applique rien.

## Poids d'events actuels (constants.toml [engagement])

| titre | objective | assist | death | default |
|---|---|---|---|---|
| halo_5 | 1.5 | 0.5 | 0 | 1 |
| halo_infinite (ref) | 1.5 | 0.5 | 0 | 1 |

## Distribution par mode et bin d'intensite — halo_5 (4 joueurs)

| mode | n | rejets | coef global | calme | standard | chaotique | intensite p50 |
|---|---|---|---|---|---|---|---|
| PvP_ranked | 3473 | 17 | 0.968 | 1.043 (n=1150) | 0.953 (n=1153) | 0.916 (n=1153) | 3.029 |
| PvP_unranked | 1767 | 150 | 0.946 | 1.011 (n=539) | 0.951 (n=538) | 0.879 (n=540) | 2.731 |

## Reference halo_infinite (4 joueurs)

| mode | n | rejets | coef global | calme | standard | chaotique | intensite p50 |
|---|---|---|---|---|---|---|---|
| PvP_ranked | 29 | 0 | 0.871 | 0.841 (n=10) | 0.823 (n=9) | 0.914 (n=10) | 1.844 |
| PvP_unranked | 2094 | 20 | 0.975 | 1.031 (n=690) | 0.950 (n=691) | 0.958 (n=693) | 1.878 |

## Coefficients candidats

Methode : le score d'engagement est un percentile intra-personnel (invariant
d'echelle) ; le levier de calibration dependant du gameplay = les poids d'events.
Les coefficients candidats proposes = les poids actuels de `halo_5` (ci-dessus). Le
tableau ci-dessus permet de juger si la dispersion des ratios (coef par bin) et le
taux de rejet du titre sont comparables a la reference Infinite. Si oui, les poids
de reference conviennent (candidat = defaut) ; sinon, ajuster au gate humain E6.

```toml
[engagement]
objective = 1.5
assist    = 0.5
death     = 0
default   = 1
```

## Verdict automatique (indicatif — non liant)

- Donnees suffisantes : au moins un mode a un coef global exploitable.
- Candidat = poids de reference Infinite (le score etant percentile intra-personnel).
- A valider au gate humain (E6) : les scores H5 ont-ils du sens sur des matchs intenses vs calmes ?
