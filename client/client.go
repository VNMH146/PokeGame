package main

import (
	"fmt"
	"net"
)

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

	inBattle := false  // Tracks if the player is in a battle
	playerName := ""   // Stores the current player's name
	opponentName := "" // Stores the opponent's name

	for {
		fmt.Println("\nChoose an action:")
		fmt.Println("1. Capture Pokémon")
		if inBattle {
			fmt.Println("2. Continue battle actions")
		} else {
			fmt.Println("2. Start a battle")
		}
		fmt.Println("3. Destroy Pokémon")
		fmt.Println("4. Exit")
		fmt.Print("Enter choice: ")

		var choice int
		fmt.Scanln(&choice)

		switch choice {
		case 1:
			fmt.Print("Enter your name: ")
			fmt.Scanln(&playerName)

			fmt.Print("Enter Pokémon name to capture: ")
			var pokemonName string
			fmt.Scanln(&pokemonName)

			query := fmt.Sprintf("capturePokemon:%s|%s", playerName, pokemonName)
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
			if !inBattle {
				// Start a new battle
				fmt.Print("Enter your name: ")
				fmt.Scanln(&playerName)

				fmt.Print("Enter opponent's name: ")
				fmt.Scanln(&opponentName)

				query := fmt.Sprintf("startBattle:%s|%s", playerName, opponentName)
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

				if response[:6] == "Battle" {
					inBattle = true // Transition to battle mode
				}
			} else {
				// Continue battle actions
				fmt.Println("\nChoose your battle action:")
				fmt.Println("1. Attack")
				fmt.Println("2. Switch Pokémon")
				fmt.Println("3. Surrender")
				fmt.Print("Enter choice: ")

				var battleChoice int
				fmt.Scanln(&battleChoice)

				var query string
				switch battleChoice {
				case 1:
					query = fmt.Sprintf("processBattleTurn:%s|attack", playerName)
				case 2:
					fmt.Print("Enter Pokémon name to switch to: ")
					var newPokemon string
					fmt.Scanln(&newPokemon)
					query = fmt.Sprintf("processBattleTurn:%s|switch|%s", playerName, newPokemon)
				case 3:
					query = fmt.Sprintf("processBattleTurn:%s|surrender", playerName)
					inBattle = false // Exit battle mode after surrender
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

				if response == "Battle won by "+playerName || response == "Battle won by "+opponentName {
					inBattle = false // Reset battle mode when the battle ends
				}
			}

		case 3:
			fmt.Print("Enter your name: ")
			fmt.Scanln(&playerName)

			fmt.Print("Enter donor and recipient Pokémon names (space-separated): ")
			var donor, recipient string
			fmt.Scanln(&donor, &recipient)

			query := fmt.Sprintf("destroyPokemon:%s|%s|%s", playerName, donor, recipient)
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

		case 4:
			fmt.Println("Exiting...")
			return

		default:
			fmt.Println("Invalid choice")
		}
	}
}
