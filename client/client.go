package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

type Pokemon struct {
	Name           string
	Level          int
	HP             int
	Attack         int
	SpecialAttack  int
	Defense        int
	SpecialDefense int
	Speed          int
	ElementalType  string
	AccumulatedExp int
}

func main() {
	serverAddr, err := net.ResolveUDPAddr("udp", "localhost:8080")
	if err != nil {
		fmt.Println("Error resolving server address:", err)
		return
	}

	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	defer conn.Close()

	inBattle := false    // Tracks if the player is in a battle
	playerName := ""     // Stores the current player's name
	opponentName := ""   // Stores the opponent's name
	currentTurn := ""    // Tracks whose turn it is
	currentPokemon := "" // Tracks the current Pokémon playing

	for {
		if !inBattle {
			fmt.Println("\nChoose an action:")
			fmt.Println("1. Register Player")
			fmt.Println("2. Choose Pokémon")
			fmt.Println("3. Start a Battle")
			fmt.Println("4. Exit")
			fmt.Print("Enter choice: ")

			var choice int
			fmt.Scanln(&choice)

			switch choice {
			case 1:
				fmt.Print("Enter your name: ")
				fmt.Scanln(&playerName)
				query := fmt.Sprintf("registerPlayer:%s", playerName)
				if _, err := conn.Write([]byte(query)); err != nil {
					fmt.Println("Error sending query:", err)
					continue
				}
				buffer := make([]byte, 1024)
				n, _, err := conn.ReadFromUDP(buffer)
				if err != nil {
					fmt.Println("Error reading response:", err)
					continue
				}
				fmt.Println("Response from server:", string(buffer[:n]))

			case 2:
				if playerName == "" {
					fmt.Println("You need to register first.")
					continue
				}
				fmt.Println("Enter the names of 3 Pokémon to choose:")
				var pokemon1, pokemon2, pokemon3 string
				fmt.Print("Pokémon 1: ")
				fmt.Scanln(&pokemon1)
				fmt.Print("Pokémon 2: ")
				fmt.Scanln(&pokemon2)
				fmt.Print("Pokémon 3: ")
				fmt.Scanln(&pokemon3)
				query := fmt.Sprintf("choosePokemon:%s:%s:%s:%s", playerName, pokemon1, pokemon2, pokemon3)
				if _, err := conn.Write([]byte(query)); err != nil {
					fmt.Println("Error sending query:", err)
					continue
				}
				buffer := make([]byte, 1024)
				n, _, err := conn.ReadFromUDP(buffer)
				if err != nil {
					fmt.Println("Error reading response:", err)
					continue
				}
				fmt.Println("Response from server:", string(buffer[:n]))

				// Export chosen Pokémon to a JSON file in the Player folder
				chosenPokemons := []Pokemon{
					{Name: pokemon1},
					{Name: pokemon2},
					{Name: pokemon3},
				}
				playerFolder := "player"
				if err := os.MkdirAll(playerFolder, os.ModePerm); err != nil {
					fmt.Println("Error creating Player folder:", err)
					continue
				}
				filePath := filepath.Join(playerFolder, fmt.Sprintf("%s_pokemons.json", playerName))
				file, err := os.Create(filePath)
				if err != nil {
					fmt.Println("Error creating file:", err)
					continue
				}
				defer file.Close()
				encoder := json.NewEncoder(file)
				if err := encoder.Encode(chosenPokemons); err != nil {
					fmt.Println("Error encoding JSON:", err)
					continue
				}
				fmt.Println("Chosen Pokémon exported to JSON file.")

			case 3:
				fmt.Print("Enter your name: ")
				fmt.Scanln(&playerName)
				fmt.Print("Enter opponent's name: ")
				fmt.Scanln(&opponentName)
				query := fmt.Sprintf("startBattle:%s:%s", playerName, opponentName)
				if _, err := conn.Write([]byte(query)); err != nil {
					fmt.Println("Error sending query:", err)
					continue
				}
				buffer := make([]byte, 1024)
				n, _, err := conn.ReadFromUDP(buffer)
				if err != nil {
					fmt.Println("Error reading response:", err)
					continue
				}
				response := string(buffer[:n])
				fmt.Println("Response from server:", response)
				if strings.Contains(response, "Battle started") {
					inBattle = true
					currentTurn = playerName

					// Load the first Pokémon from the JSON file as the current Pokémon
					filePath := filepath.Join("Player", fmt.Sprintf("%s_pokemons.json", playerName))
					file, err := os.Open(filePath)
					if err != nil {
						fmt.Println("Error opening file:", err)
						continue
					}
					defer file.Close()
					var pokemons []Pokemon
					decoder := json.NewDecoder(file)
					if err := decoder.Decode(&pokemons); err != nil {
						fmt.Println("Error decoding JSON:", err)
						continue
					}
					if len(pokemons) > 0 {
						currentPokemon = pokemons[0].Name
					} else {
						currentPokemon = "unknown Pokémon"
					}

					fmt.Println("You go first!")
				} else if strings.Contains(response, "Opponent goes first") {
					inBattle = true
					currentTurn = opponentName
					currentPokemon = "opponent's first Pokémon" // Placeholder, update with actual Pokémon
					fmt.Println("Opponent goes first!")
				}

			case 4:
				fmt.Println("Exiting...")
				return

			default:
				fmt.Println("Invalid choice")
			}
		} else {
			fmt.Printf("\nChoose %s's battle action:\n", currentTurn)
			fmt.Printf("[Current Pokémon: %s]\n", currentPokemon)
			fmt.Println("1. Attack")
			fmt.Println("2. Switch Pokémon")
			fmt.Println("3. Surrender")
			fmt.Println("4. Check Pokémon List")
			fmt.Print("Enter choice: ")

			var battleChoice int
			fmt.Scanln(&battleChoice)

			var query string
			switch battleChoice {
			case 1:
				query = fmt.Sprintf("processBattleTurn:%s:attack", playerName)
			case 2:
				fmt.Print("Enter Pokémon name to switch to: ")
				var newPokemon string
				fmt.Scanln(&newPokemon)
				query = fmt.Sprintf("processBattleTurn:%s:switch:%s", playerName, newPokemon)
			case 3:
				query = fmt.Sprintf("processBattleTurn:%s:surrender", playerName)
				inBattle = false // Exit battle mode after surrender
			case 4:
				filePath := filepath.Join("Player", fmt.Sprintf("%s_pokemons.json", playerName))
				file, err := os.Open(filePath)
				if err != nil {
					fmt.Println("Error opening file:", err)
					continue
				}
				defer file.Close()
				var pokemons []Pokemon
				decoder := json.NewDecoder(file)
				if err := decoder.Decode(&pokemons); err != nil {
					fmt.Println("Error decoding JSON:", err)
					continue
				}
				fmt.Println("Your Pokémon List:")
				for _, pokemon := range pokemons {
					fmt.Printf("Name: %s, Level: %d, HP: %d, Attack: %d, Special Attack: %d, Defense: %d, Special Defense: %d, Speed: %d, Elemental Type: %s, Accumulated Exp: %d\n",
						pokemon.Name, pokemon.Level, pokemon.HP, pokemon.Attack, pokemon.SpecialAttack, pokemon.Defense, pokemon.SpecialDefense, pokemon.Speed, pokemon.ElementalType, pokemon.AccumulatedExp)
				}
				continue
			default:
				fmt.Println("Invalid choice")
				continue
			}

			if _, err := conn.Write([]byte(query)); err != nil {
				fmt.Println("Error sending query:", err)
				continue
			}

			buffer := make([]byte, 1024)
			n, _, err := conn.ReadFromUDP(buffer)
			if err != nil {
				fmt.Println("Error reading response:", err)
				continue
			}

			response := string(buffer[:n])
			fmt.Println("Response from server:", response)

			if strings.Contains(response, "Battle won") {
				inBattle = false // Reset battle mode when the battle ends
			} else if strings.Contains(response, "Turn:") {
				// Update current turn based on server response
				parts := strings.Split(response, ":")
				if len(parts) == 3 {
					currentTurn = parts[1]
					currentPokemon = parts[2]
				}
			}
		}
	}
}
