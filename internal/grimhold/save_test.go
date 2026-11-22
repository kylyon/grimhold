package grimhold

import (
	"path/filepath"
	"testing"
)

func TestSaveAndLoadGame(t *testing.T) {
	savePath := filepath.Join(t.TempDir(), "save.json")

	// Missing save file must not error or crash.
	loaded, err := LoadGame(savePath)
	if err != nil {
		t.Fatalf("unexpected error on missing save: %v", err)
	}
	if loaded {
		t.Fatal("expected loaded=false when save file does not exist")
	}

	Player = newDefaultPlayer()
	Player.Level = 3
	Player.XP = 250
	Player.HP = 22
	Player.MaxHP = 46
	Player.Gold = 42
	Player.RestedThisVisit = true
	Player.Slain["Goblin"] = 4
	World.CurrentLocation = "ashen-marsh"
	activeEncounter = nil

	if err := SaveGame(savePath); err != nil {
		t.Fatalf("SaveGame failed: %v", err)
	}

	// Mutate in-memory state before loading.
	Player = newDefaultPlayer()
	World.CurrentLocation = "grimhold-village"

	loaded, err = LoadGame(savePath)
	if err != nil {
		t.Fatalf("LoadGame failed: %v", err)
	}
	if !loaded {
		t.Fatal("expected loaded=true")
	}

	if World.CurrentLocation != "ashen-marsh" {
		t.Fatalf("expected location ashen-marsh, got %q", World.CurrentLocation)
	}
	if Player.Level != 3 || Player.XP != 250 || Player.HP != 22 || Player.MaxHP != 46 {
		t.Fatalf("unexpected player stats after load: %+v", Player)
	}
	if Player.Gold != 42 || !Player.RestedThisVisit || Player.Slain["Goblin"] != 4 {
		t.Fatalf("unexpected progression/rest/slain after load: %+v", Player)
	}
}
