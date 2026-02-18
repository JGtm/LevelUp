"""Test de la résolution gamertag->xuid."""

import duckdb

conn = duckdb.connect("data/warehouse/shared_matches.duckdb", read_only=True)

# Test avec les 3 gamertags du trio
gamertags = ["Madina97294", "JGtm", "Chocoboflor"]

print("=== TEST RÉSOLUTION GAMERTAG -> XUID ===")
for gt in gamertags:
    print(f'\nTest pour "{gt}":')

    # Méthode 1: xuid_aliases
    result = conn.execute("SELECT xuid FROM xuid_aliases WHERE gamertag = ?", [gt]).fetchone()
    if result:
        print(f"  ✓ Trouvé dans xuid_aliases: {result[0]}")
    else:
        print("  ✗ Non trouvé dans xuid_aliases")

    # Méthode 2: match_participants (si gamertag non NULL)
    result = conn.execute(
        "SELECT DISTINCT xuid FROM match_participants WHERE gamertag = ? LIMIT 1", [gt]
    ).fetchone()
    if result:
        print(f"  ✓ Trouvé dans match_participants: {result[0]}")
    else:
        print("  ✗ Non trouvé dans match_participants")

    # Test sensibilité casse
    result = conn.execute(
        "SELECT xuid FROM xuid_aliases WHERE LOWER(gamertag) = LOWER(?)", [gt]
    ).fetchone()
    if result:
        print(f"  ✓ Trouvé (insensible casse) dans xuid_aliases: {result[0]}")

# Test: est-ce que load_teammate_stats pourrait fonctionner ?
print("\n=== TEST: Charger stats pour Madina97294 match b77dffd4 ===")
xuid = conn.execute("SELECT xuid FROM xuid_aliases WHERE gamertag = ?", ["Madina97294"]).fetchone()[
    0
]

print(f"XUID résolu: {xuid}")

result = conn.execute(
    """
    SELECT
        p.match_id,
        p.xuid,
        p.kills,
        p.deaths,
        p.assists
    FROM match_participants p
    WHERE p.xuid = ?
      AND p.match_id = 'b77dffd4-f97b-44e7-8a6a-1fc3ef3e0ca0'
""",
    [xuid],
).fetchone()

if result:
    print(f"Stats trouvées: K={result[2]} D={result[3]} A={result[4]}")
else:
    print("⚠️ Stats non trouvées !")

conn.close()
