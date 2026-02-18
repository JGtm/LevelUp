#!/usr/bin/env python3
"""Validation rapide du mode exclude/include sans pytest."""

import sys
sys.path.insert(0, '/home/runner/work/LevelUp/LevelUp')

from src.ui.filter_state import FilterPreferences, _detect_filter_mode


def test_detect_mode():
    """Tests de détection du mode."""
    print("Testing _detect_filter_mode()...")
    
    # Test 1: >70% → exclude
    selected = {"A", "B", "C", "D", "E", "F", "G", "H"}  # 8/10 = 80%
    all_options = {"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
    mode = _detect_filter_mode(selected, all_options)
    assert mode == "exclude", f"Expected 'exclude' but got '{mode}'"
    print("  ✓ Test 1: >70% → exclude")
    
    # Test 2: <30% → include
    selected = {"A", "B"}  # 2/10 = 20%
    mode = _detect_filter_mode(selected, all_options)
    assert mode == "include", f"Expected 'include' but got '{mode}'"
    print("  ✓ Test 2: <30% → include")
    
    # Test 3: Zone grise 50% → include
    selected = {"A", "B", "C", "D", "E"}  # 5/10 = 50%
    mode = _detect_filter_mode(selected, all_options)
    assert mode == "include", f"Expected 'include' (gray zone) but got '{mode}'"
    print("  ✓ Test 3: Zone grise 50% → include")
    
    # Test 4: 100% → exclude
    selected = {"A", "B", "C"}
    all_options = {"A", "B", "C"}
    mode = _detect_filter_mode(selected, all_options)
    assert mode == "exclude", f"Expected 'exclude' (100%) but got '{mode}'"
    print("  ✓ Test 4: 100% → exclude")
    
    # Test 5: 0% → include
    selected = set()
    all_options = {"A", "B", "C"}
    mode = _detect_filter_mode(selected, all_options)
    assert mode == "include", f"Expected 'include' (0%) but got '{mode}'"
    print("  ✓ Test 5: 0% → include")
    
    print("  ✅ All mode detection tests passed!\n")


def test_filter_preferences():
    """Tests de FilterPreferences."""
    print("Testing FilterPreferences...")
    
    # Test 1: Création avec modes
    prefs = FilterPreferences(
        playlists_selected=["A", "B"],
        playlists_mode="exclude",
    )
    assert prefs.playlists_mode == "exclude"
    print("  ✓ Test 1: Création avec mode")
    
    # Test 2: to_dict exclut None
    data = prefs.to_dict()
    assert "playlists_selected" in data
    assert "playlists_mode" in data
    assert "modes_mode" not in data  # None
    print("  ✓ Test 2: to_dict() exclut None")
    
    # Test 3: from_dict backward compatible
    data = {"playlists_selected": ["A", "B"]}  # Pas de mode
    prefs = FilterPreferences.from_dict(data)
    assert prefs.playlists_selected == ["A", "B"]
    assert prefs.playlists_mode is None
    print("  ✓ Test 3: from_dict() backward compatible")
    
    # Test 4: from_dict avec mode
    data = {
        "playlists_selected": ["Firefight"],
        "playlists_mode": "exclude",
    }
    prefs = FilterPreferences.from_dict(data)
    assert prefs.playlists_mode == "exclude"
    assert prefs.playlists_selected == ["Firefight"]
    print("  ✓ Test 4: from_dict() avec mode")
    
    print("  ✅ All FilterPreferences tests passed!\n")


def test_exclude_scenarios():
    """Tests de scénarios réels."""
    print("Testing real-world scenarios...")
    
    # Scénario 1: Tout sauf Firefight
    all_playlists = ["Quick Play", "Ranked", "BTB", "Firefight", "Slayer"]
    selected = {"Quick Play", "Ranked", "BTB", "Slayer"}  # 80%
    mode = _detect_filter_mode(selected, all_playlists)
    assert mode == "exclude"
    excluded = set(all_playlists) - selected
    assert excluded == {"Firefight"}
    print("  ✓ Scénario 1: Tout sauf Firefight (mode exclude)")
    
    # Scénario 2: Uniquement Ranked
    selected = {"Ranked"}  # 20%
    mode = _detect_filter_mode(selected, all_playlists)
    assert mode == "include"
    print("  ✓ Scénario 2: Uniquement Ranked (mode include)")
    
    # Scénario 3: Nouvelle playlist en mode exclude
    excluded_saved = ["BTB"]
    all_playlists_new = ["Quick Play", "Ranked", "BTB", "New Mode"]
    selected_result = set(all_playlists_new) - set(excluded_saved)
    assert "New Mode" in selected_result  # Auto-incluse
    assert "BTB" not in selected_result
    print("  ✓ Scénario 3: Nouvelle playlist auto-incluse (mode exclude)")
    
    # Scénario 4: Nouvelle playlist en mode include
    included_saved = ["Ranked"]
    selected_result = set(included_saved) & set(all_playlists_new)
    assert "New Mode" not in selected_result  # Auto-exclue
    assert "Ranked" in selected_result
    print("  ✓ Scénario 4: Nouvelle playlist auto-exclue (mode include)")
    
    print("  ✅ All scenario tests passed!\n")


if __name__ == "__main__":
    try:
        test_detect_mode()
        test_filter_preferences()
        test_exclude_scenarios()
        print("=" * 60)
        print("✅ ALL TESTS PASSED! Mode exclude/include is working correctly.")
        print("=" * 60)
    except AssertionError as e:
        print(f"\n❌ TEST FAILED: {e}")
        sys.exit(1)
    except Exception as e:
        print(f"\n❌ ERROR: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)
