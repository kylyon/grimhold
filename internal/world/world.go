package world

import (
	"encoding/json/v2"
	"fmt"
	"os"
)

type Region struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Danger      int      `json:"danger"`
	Foes        []string `json:"foes"`
	Exits       []string `json:"exits"`
}

type World struct {
	Start           string
	Regions         map[string]Region
	CurrentLocation string
}

type worldJSON struct {
	Start   string   `json:"start"`
	Regions []Region `json:"regions"`
}

func Load(path string) (World, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return World{}, fmt.Errorf("reading world file %q: %w", path, err)
	}

	var raw worldJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return World{}, fmt.Errorf("decoding world JSON %q: %w", path, err)
	}

	regions := make(map[string]Region, len(raw.Regions))
	for _, r := range raw.Regions {
		regions[r.ID] = r
	}

	return World{
		Start:           raw.Start,
		Regions:         regions,
		CurrentLocation: raw.Start,
	}, nil
}
