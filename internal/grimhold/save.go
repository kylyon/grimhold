package grimhold

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

const DefaultSavePath = "save.json"

// SaveData represents the persisted game state written to save.json.
type SaveData struct {
	Location string      `json:"location"`
	Player   PlayerState `json:"player"`
}

// SaveGame writes the current game state atomically to path.
func SaveGame(path string) error {
	if activeEncounter != nil {
		return errors.New("you cannot save while in combat — finish the fight or flee first")
	}

	state := SaveData{
		Location: World.CurrentLocation,
		Player:   Player,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode save data: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write save file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to finalize save file: %w", err)
	}
	return nil
}

// LoadGame restores game state from path. Returns (false, nil) if no save file exists.
func LoadGame(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("failed to read save file: %w", err)
	}

	var state SaveData
	if err := json.Unmarshal(data, &state); err != nil {
		return false, fmt.Errorf("failed to parse save file: %w", err)
	}

	if state.Player.Slain == nil {
		state.Player.Slain = make(map[string]int)
	}
	if state.Player.HP <= 0 {
		state.Player.HP = 1
	}

	loc := state.Location
	if _, ok := World.Regions[loc]; !ok {
		loc = World.Start
	}

	Player = state.Player
	World.CurrentLocation = loc
	activeEncounter = nil
	return true, nil
}

func callbackSave(args []string) error {
	if err := SaveGame(DefaultSavePath); err != nil {
		return err
	}
	fmt.Printf("Game saved to %s.\n", DefaultSavePath)
	return nil
}

func callbackLoad(args []string) error {
	loaded, err := LoadGame(DefaultSavePath)
	if err != nil {
		return err
	}
	if !loaded {
		fmt.Printf("No save file found (%s).\n", DefaultSavePath)
		return nil
	}
	fmt.Printf("Game loaded from %s. Welcome back to %s!\n",
		DefaultSavePath, World.Regions[World.CurrentLocation].Name)
	return nil
}
