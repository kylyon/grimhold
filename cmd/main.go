package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/kylyon/grimhold/internal/grimhold"
)

var scanner = bufio.NewScanner(os.Stdin)

func read() (string, error) {
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("reading input: %w", err)
		}
		return "", errors.New("EOF")
	}

	return scanner.Text(), nil
}

func main() {
	continueGame := flag.Bool("continue", false, "Load saved game from save.json")
	flag.Parse()

	if *continueGame {
		loaded, err := grimhold.LoadGame(grimhold.DefaultSavePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load save: %v\n", err)
		} else if !loaded {
			fmt.Println("No save file found — starting a new game.")
		} else {
			fmt.Printf("Resumed saved game at %s.\n", grimhold.World.Regions[grimhold.World.CurrentLocation].Name)
		}
	}

	for {
		fmt.Print("grimhold > ")
		input, err := read()
		if err != nil {
			if err.Error() != "EOF" {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
			return
		}

		if err := grimhold.HandleCommand(input); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}
