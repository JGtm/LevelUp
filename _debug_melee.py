"""Debug script to test melee_kills extraction pipeline."""

import sys

sys.path.insert(0, ".")

from src.data.sync.transformers import _find_core_stats_dict, extract_participants
from tests.test_transformers_coverage import _match_json, _player_obj

# Test 1: _find_core_stats_dict
p = _player_obj(melee_kills=7, grenade_kills=3, power_weapon_kills=2)
stats_dict = _find_core_stats_dict(p)
print("=== _find_core_stats_dict ===")
print(f"stats_dict is None: {stats_dict is None}")
if stats_dict:
    print(f"'MeleeKills' present: {'MeleeKills' in stats_dict}")
    print(f"MeleeKills value: {stats_dict.get('MeleeKills')}")
    print(f"GrenadeKills value: {stats_dict.get('GrenadeKills')}")
    print(f"PowerWeaponKills value: {stats_dict.get('PowerWeaponKills')}")

# Test 2: extract_participants
print("\n=== extract_participants ===")
mj = _match_json(players=[_player_obj(melee_kills=7, grenade_kills=3, power_weapon_kills=2)])
rows = extract_participants(mj)
print(f"Rows: {len(rows)}")
if rows:
    r = rows[0]
    print(f"melee_kills: {r.melee_kills}")
    print(f"grenade_kills: {r.grenade_kills}")
    print(f"power_weapon_kills: {r.power_weapon_kills}")
    print(f"headshot_kills: {r.headshot_kills}")

# Test 3: Check what happens with API real format
# The API returns JSON with PascalCase keys
# JsonResponse.json() returns raw dict — NOT parsed Pydantic
# So keys should be PascalCase: "MeleeKills", not "melee_kills"
print("\n=== API format verification ===")
print("SPNKr get_match_stats -> JsonResponse")
print("Our api_client calls resp.json() -> raw dict (PascalCase keys)")
print("NOT resp.parse() -> Pydantic model (snake_case)")
print("So _find_core_stats_dict searching 'MeleeKills' should work IF the API returns it")

# Test 4: Does the API actually return MeleeKills?
# Check SPNKr model
from spnkr.models.stats import CoreStats

print("\n=== SPNKr CoreStats model fields ===")
for name, field in CoreStats.model_fields.items():
    if "kill" in name.lower() or "melee" in name.lower() or "grenade" in name.lower():
        alias = field.alias
        print(f"  {name} (alias={alias})")
