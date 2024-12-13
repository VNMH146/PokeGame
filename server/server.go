package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"strings"
)

type Stat struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type Pokemon struct {
	Name           string   `json:"name"`
	Type           []string `json:"type"`
	BaseExp        int      `json:"base_exp"`
	Stats          []Stat   `json:"stats"`
	Level          int      `json:"level"`
	AccumulatedExp int      `json:"accumulated_exp"`
	HP             int      `json:"hp"`
}

type Player struct {
	Name     string
	Pokemons []Pokemon
}

var pokedex []Pokemon

type Battle struct {
	Player1 Player
	Player2 Player
	Turn    string
}

var battles = make(map[string]*Battle)
var playerBattleMap = make(map[string]string) // Maps player name to battle ID

// Load Pokedex
func loadPokedex(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &pokedex)
}

// Save Pokémon for a specific player
func savePlayerPokemonForPlayer(playerName string, pokemonList []Pokemon) error {
	filename := fmt.Sprintf("%s_pokemon.json", playerName)
	data, err := json.MarshalIndent(pokemonList, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, 0644)
}

// Load Pokémon for a specific player
func loadPlayerPokemonForPlayer(playerName string) ([]Pokemon, error) {
	filename := fmt.Sprintf("%s_pokemon.json", playerName)
	data, err := ioutil.ReadFile(filename)
	if os.IsNotExist(err) {
		return []Pokemon{}, nil
	}
	if err != nil {
		return nil, err
	}

	var pokemonList []Pokemon
	if err := json.Unmarshal(data, &pokemonList); err != nil {
		return nil, err
	}
	return pokemonList, nil
}

// Capture Pokémon
func capturePokemonForPlayer(conn *net.UDPConn, addr *net.UDPAddr, query string) {
	parts := strings.Split(query, "|")
	if len(parts) != 2 {
		conn.WriteToUDP([]byte("Invalid capture request format"), addr)
		return
	}

	playerName := parts[0]
	pokemonName := parts[1]

	for _, pokemon := range pokedex {
		if pokemon.Name == pokemonName {
			playerPokemon, err := loadPlayerPokemonForPlayer(playerName)
			if err != nil {
				conn.WriteToUDP([]byte("Error loading player data"), addr)
				return
			}

			pokemon.Level = 1
			pokemon.AccumulatedExp = 0

			playerPokemon = append(playerPokemon, pokemon)

			if err := savePlayerPokemonForPlayer(playerName, playerPokemon); err != nil {
				conn.WriteToUDP([]byte("Error saving player data"), addr)
				return
			}

			conn.WriteToUDP([]byte(fmt.Sprintf("%s captured successfully", pokemonName)), addr)
			return
		}
	}
	conn.WriteToUDP([]byte("Pokemon not found"), addr)
}

// Start a battle
func startBattle(conn *net.UDPConn, addr *net.UDPAddr, query string) {
	parts := strings.Split(query, "|")
	if len(parts) != 2 {
		conn.WriteToUDP([]byte("Invalid battle request format"), addr)
		return
	}

	player1Name := parts[0]
	player2Name := parts[1]

	player1Pokemons, err := loadPlayerPokemonForPlayer(player1Name)
	if err != nil {
		conn.WriteToUDP([]byte("Error loading player 1 Pokémon"), addr)
		return
	}

	player2Pokemons, err := loadPlayerPokemonForPlayer(player2Name)
	if err != nil {
		conn.WriteToUDP([]byte("Error loading player 2 Pokémon"), addr)
		return
	}

	if len(player1Pokemons) < 3 {
		conn.WriteToUDP([]byte(fmt.Sprintf("%s needs to capture more Pokémon", player1Name)), addr)
		return
	}
	if len(player2Pokemons) < 3 {
		conn.WriteToUDP([]byte(fmt.Sprintf("%s needs to capture more Pokémon", player2Name)), addr)
		return
	}

	battleID := fmt.Sprintf("%d", rand.Int())
	battles[battleID] = &Battle{
		Player1: Player{Name: player1Name, Pokemons: player1Pokemons[:3]},
		Player2: Player{Name: player2Name, Pokemons: player2Pokemons[:3]},
		Turn:    player1Name,
	}

	playerBattleMap[player1Name] = battleID
	playerBattleMap[player2Name] = battleID

	conn.WriteToUDP([]byte("Battle started with ID: "+battleID), addr)
}

func processBattleTurn(conn *net.UDPConn, addr *net.UDPAddr, query string) {
	parts := strings.Split(query, "|")
	if len(parts) < 2 {
		conn.WriteToUDP([]byte("Invalid turn request format"), addr)
		return
	}

	playerName := parts[0]
	action := parts[1]

	battleID, exists := playerBattleMap[playerName]
	if !exists {
		conn.WriteToUDP([]byte("No active battle found for player"), addr)
		return
	}

	battle, exists := battles[battleID]
	if !exists {
		conn.WriteToUDP([]byte("Battle not found"), addr)
		return
	}

	var attacker, defender *Player
	if battle.Turn == battle.Player1.Name {
		attacker = &battle.Player1
		defender = &battle.Player2
	} else {
		attacker = &battle.Player2
		defender = &battle.Player1
	}

	if playerName != attacker.Name {
		conn.WriteToUDP([]byte("It's not your turn!"), addr)
		return
	}

	switch action {
	case "attack":
		handleAttack(conn, attacker, defender, addr)
	case "switch":
		if len(parts) < 3 {
			conn.WriteToUDP([]byte("Invalid switch command"), addr)
			return
		}
		handleSwitch(conn, attacker, parts[2], addr)
	case "surrender":
		handleSurrender(conn, attacker, defender, addr)
	default:
		conn.WriteToUDP([]byte("Unknown action"), addr)
		return
	}

	if battle.Turn == battle.Player1.Name {
		battle.Turn = battle.Player2.Name
	} else {
		battle.Turn = battle.Player1.Name
	}

	conn.WriteToUDP([]byte("Turn processed"), addr)
}

// Handle request
func handleRequest(conn *net.UDPConn, addr *net.UDPAddr, query string) {
	parts := strings.Split(query, ":")
	action := parts[0]
	args := strings.Join(parts[1:], ":")

	switch action {
	case "capturePokemon":
		capturePokemonForPlayer(conn, addr, args)
	case "startBattle":
		startBattle(conn, addr, args)

	case "destroyPokemon":
		destroyPokemon(conn, addr, args)

	default:
		conn.WriteToUDP([]byte("Unknown action"), addr)
	}
}

func destroyPokemon(conn *net.UDPConn, addr *net.UDPAddr, query string) {
	// Parse the query: "playerName donorPokemon recipientPokemon"
	parts := strings.Split(query, "|")
	if len(parts) != 3 {
		conn.WriteToUDP([]byte("Invalid request format for destroy"), addr)
		return
	}

	playerName := parts[0]
	donor := parts[1]
	recipient := parts[2]

	fmt.Printf("DestroyPokemon called with Donor: %s, Recipient: %s for Player: %s\n", donor, recipient, playerName)

	// Load player's Pokémon
	playerPokemon, err := loadPlayerPokemonForPlayer(playerName)
	if err != nil {
		fmt.Println("Error loading player data:", err)
		conn.WriteToUDP([]byte("Error loading player data"), addr)
		return
	}

	var donorPokemon *Pokemon
	var recipientPokemon *Pokemon

	// Find the donor and recipient Pokémon
	for i, pokemon := range playerPokemon {
		if pokemon.Name == donor {
			donorPokemon = &playerPokemon[i]
		}
		if pokemon.Name == recipient {
			recipientPokemon = &playerPokemon[i]
		}
	}

	// Ensure both Pokémon exist and are of the same type
	if donorPokemon == nil || recipientPokemon == nil {
		fmt.Println("Either donor or recipient does not exist")
		conn.WriteToUDP([]byte("Both donor and recipient Pokémon must exist"), addr)
		return
	}

	sameType := false
	for _, t1 := range donorPokemon.Type {
		for _, t2 := range recipientPokemon.Type {
			if t1 == t2 {
				sameType = true
				break
			}
		}
	}
	if !sameType {
		fmt.Println("Donor and recipient are not of the same type")
		conn.WriteToUDP([]byte("Donor and recipient Pokémon must be of the same type"), addr)
		return
	}

	// Transfer accumulated experience
	recipientPokemon.AccumulatedExp += donorPokemon.AccumulatedExp
	fmt.Printf("Transferred %d experience from %s to %s\n", donorPokemon.AccumulatedExp, donor, recipient)

	// Check if recipient Pokémon can level up
	for {
		requiredExp := recipientPokemon.Level * 100 // Fixed experience threshold per level
		if recipientPokemon.AccumulatedExp >= requiredExp {
			recipientPokemon.Level++
			recipientPokemon.AccumulatedExp -= requiredExp // Deduct experience used for leveling up
			ev := 0.5 + rand.Float64()*0.5                 // Random EV between 0.5 and 1
			for i, stat := range recipientPokemon.Stats {
				if stat.Name != "speed" && stat.Name != "dmg_when_atked" {
					recipientPokemon.Stats[i].Value = int(float64(stat.Value) * (1 + ev))
				}
			}
			fmt.Printf("Leveled up %s to level %d\n", recipient, recipientPokemon.Level)
		} else {
			break
		}
	}

	// Remove the donor Pokémon from the player's list
	newPokemonList := []Pokemon{}
	for _, pokemon := range playerPokemon {
		if pokemon.Name != donorPokemon.Name {
			newPokemonList = append(newPokemonList, pokemon)
		}
	}

	// Save the updated Pokémon list
	if err := savePlayerPokemonForPlayer(playerName, newPokemonList); err != nil {
		fmt.Println("Error saving player data:", err)
		conn.WriteToUDP([]byte("Error saving player data"), addr)
		return
	}

	conn.WriteToUDP([]byte("Pokemon destroyed and experience transferred successfully"), addr)
}

// Handle battle turns
// Handle battle turns

func handleAttack(conn *net.UDPConn, attacker, defender *Player, addr *net.UDPAddr) {
	attackerPokemon := &attacker.Pokemons[0]
	defenderPokemon := &defender.Pokemons[0]

	// Randomly decide normal or special attack
	if rand.Intn(2) == 0 {
		// Normal attack
		attack := getStat(attackerPokemon, "attack")
		defense := getStat(defenderPokemon, "defense")
		damage := max(attack-defense, 1)
		defenderPokemon.HP -= damage
		fmt.Printf("%s attacked %s with normal attack. Damage: %d, Remaining HP: %d\n",
			attackerPokemon.Name, defenderPokemon.Name, damage, defenderPokemon.HP)
	} else {
		// Special attack
		spAttack := getStat(attackerPokemon, "special-attack")
		spDefense := getStat(defenderPokemon, "special-defense")
		damage := max(spAttack-spDefense, 1)
		defenderPokemon.HP -= damage
		fmt.Printf("%s attacked %s with special attack. Damage: %d, Remaining HP: %d\n",
			attackerPokemon.Name, defenderPokemon.Name, damage, defenderPokemon.HP)
	}

	if defenderPokemon.HP <= 0 {
		fmt.Printf("%s fainted!\n", defenderPokemon.Name)
		defender.Pokemons = defender.Pokemons[1:]
		if len(defender.Pokemons) == 0 {
			conn.WriteToUDP([]byte("Battle won by "+attacker.Name), addr)
			return
		}
		conn.WriteToUDP([]byte(defenderPokemon.Name+" fainted!"), addr)
	}
}

func handleSwitch(conn *net.UDPConn, player *Player, newPokemonName string, addr *net.UDPAddr) {
	for i, pokemon := range player.Pokemons {
		if pokemon.Name == newPokemonName {
			// Switch to the new Pokémon
			player.Pokemons[0], player.Pokemons[i] = player.Pokemons[i], player.Pokemons[0]
			conn.WriteToUDP([]byte("Switched to "+newPokemonName), addr)
			return
		}
	}
	conn.WriteToUDP([]byte("Pokemon not found in your list"), addr)
}

func handleSurrender(conn *net.UDPConn, loser, winner *Player, addr *net.UDPAddr) {
	totalExp := 0
	for _, pokemon := range loser.Pokemons {
		totalExp += pokemon.AccumulatedExp
	}

	expShare := totalExp / len(winner.Pokemons)
	for i := range winner.Pokemons {
		winner.Pokemons[i].AccumulatedExp += expShare
	}

	// Remove players from playerBattleMap
	delete(playerBattleMap, loser.Name)
	delete(playerBattleMap, winner.Name)

	conn.WriteToUDP([]byte("Battle won by "+winner.Name), addr)
}

func getStat(pokemon *Pokemon, statName string) int {
	for _, stat := range pokemon.Stats {
		if stat.Name == statName {
			return stat.Value
		}
	}
	return 0
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	if err := loadPokedex("../pokedex.json"); err != nil {
		fmt.Println("Error loading Pokedex:", err)
		return
	}

	addr, err := net.ResolveUDPAddr("udp", ":8080")
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("Error starting UDP server:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Server is listening on port 8080")

	buffer := make([]byte, 1024)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Error reading from client:", err)
			continue
		}

		query := string(buffer[:n])
		go handleRequest(conn, clientAddr, query)
	}
}
