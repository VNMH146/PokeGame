package main

import (
	"fmt"
	"math/rand"
	"net"
	"strings"
)

var players = make(map[string]*Player)
var battles = make(map[string]*Battle)

type Player struct {
	Name     string
	Pokemons []*Pokemon
}

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

type Battle struct {
	Player1 *Player
	Player2 *Player
	Turn    string
}

func main() {
	addr, err := net.ResolveUDPAddr("udp", ":8080")
	if err != nil {
		fmt.Println("Error resolving address:", err)
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Server is listening on", addr)

	for {
		buffer := make([]byte, 1024)
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Error reading from UDP:", err)
			continue
		}

		query := string(buffer[:n])
		handleQuery(conn, clientAddr, query)
	}
}

func handleQuery(conn *net.UDPConn, addr *net.UDPAddr, query string) {
	parts := strings.Split(query, ":")
	if len(parts) < 1 {
		conn.WriteToUDP([]byte("Invalid query format"), addr)
		return
	}

	command := parts[0]
	switch command {
	case "registerPlayer":
		registerPlayer(conn, addr, parts[1])
	case "startBattle":
		startBattle(conn, addr, parts[1:])
	case "processBattleTurn":
		processBattleTurn(conn, addr, parts[1:])
	default:
		conn.WriteToUDP([]byte("Unknown command"), addr)
	}
}

func registerPlayer(conn *net.UDPConn, addr *net.UDPAddr, playerName string) {
	if _, exists := players[playerName]; exists {
		conn.WriteToUDP([]byte("Player already registered"), addr)
		return
	}

	players[playerName] = &Player{Name: playerName, Pokemons: make([]*Pokemon, 0)}
	conn.WriteToUDP([]byte("Player registered successfully"), addr)
}

func startBattle(conn *net.UDPConn, addr *net.UDPAddr, parts []string) {
	if len(parts) != 2 {
		conn.WriteToUDP([]byte("Invalid startBattle request format"), addr)
		return
	}

	player1Name := parts[0]
	player2Name := parts[1]

	player1, ok1 := players[player1Name]
	player2, ok2 := players[player2Name]

	if !ok1 || !ok2 {
		conn.WriteToUDP([]byte("One or both players not registered"), addr)
		return
	}

	battleID := fmt.Sprintf("%s_vs_%s", player1Name, player2Name)
	battles[battleID] = &Battle{Player1: player1, Player2: player2, Turn: player1Name}
	conn.WriteToUDP([]byte(fmt.Sprintf("Battle started between %s and %s", player1Name, player2Name)), addr)
}

func processBattleTurn(conn *net.UDPConn, addr *net.UDPAddr, parts []string) {
	if len(parts) < 3 {
		conn.WriteToUDP([]byte("Invalid processBattleTurn request format"), addr)
		return
	}

	battleID := parts[0]
	action := parts[1]
	playerName := parts[2]

	battle, ok := battles[battleID]
	if !ok {
		conn.WriteToUDP([]byte("Battle not found"), addr)
		return
	}

	if battle.Turn != playerName {
		conn.WriteToUDP([]byte("Not your turn"), addr)
		return
	}

	switch action {
	case "attack":
		handleAttack(conn, battle, playerName, addr)
	case "switch":
		if len(parts) != 4 {
			conn.WriteToUDP([]byte("Invalid switch request format"), addr)
			return
		}
		newPokemonName := parts[3]
		handleSwitch(conn, battle, playerName, newPokemonName, addr)
	case "surrender":
		handleSurrender(conn, battle, playerName, addr)
	default:
		conn.WriteToUDP([]byte("Unknown action"), addr)
	}
}

func handleAttack(conn *net.UDPConn, battle *Battle, playerName string, addr *net.UDPAddr) {
	attacker, defender := getPlayers(battle, playerName)
	attackingPokemon := attacker.Pokemons[0]
	defendingPokemon := defender.Pokemons[0]

	damage := 0
	if rand.Intn(2) == 0 {
		// Normal attack
		damage = attackingPokemon.Attack - defendingPokemon.Defense
	} else {
		// Special attack
		elementalMultiplier := 1.0 // This should be calculated based on elemental types
		damage = int(float64(attackingPokemon.SpecialAttack)*elementalMultiplier) - defendingPokemon.SpecialDefense
	}

	if damage < 0 {
		damage = 0
	}

	defendingPokemon.HP -= damage
	if defendingPokemon.HP <= 0 {
		defendingPokemon.HP = 0
		conn.WriteToUDP([]byte(fmt.Sprintf("%s's %s fainted!", defender.Name, defendingPokemon.Name)), addr)
	} else {
		conn.WriteToUDP([]byte(fmt.Sprintf("%s's %s took %d damage!", defender.Name, defendingPokemon.Name, damage)), addr)
	}

	// Switch turn
	battle.Turn = defender.Name
}

func handleSwitch(conn *net.UDPConn, battle *Battle, playerName, newPokemonName string, addr *net.UDPAddr) {
	player := getPlayer(battle, playerName)
	for i, pokemon := range player.Pokemons {
		if pokemon.Name == newPokemonName {
			player.Pokemons[0], player.Pokemons[i] = player.Pokemons[i], player.Pokemons[0]
			conn.WriteToUDP([]byte(fmt.Sprintf("%s switched to %s!", playerName, newPokemonName)), addr)
			break
		}
	}

	// Switch turn
	battle.Turn = getOpponent(battle, playerName).Name
}

func handleSurrender(conn *net.UDPConn, battle *Battle, playerName string, addr *net.UDPAddr) {
	winner := getOpponent(battle, playerName)
	loser := getPlayer(battle, playerName)

	// Distribute experience points
	totalExp := 0
	for _, pokemon := range loser.Pokemons {
		totalExp += pokemon.AccumulatedExp
	}
	expPerPokemon := totalExp / len(winner.Pokemons)
	for _, pokemon := range winner.Pokemons {
		pokemon.AccumulatedExp += expPerPokemon
	}

	conn.WriteToUDP([]byte(fmt.Sprintf("%s surrendered! %s wins!", playerName, winner.Name)), addr)
	delete(battles, fmt.Sprintf("%s_vs_%s", battle.Player1.Name, battle.Player2.Name))
}

func getPlayers(battle *Battle, playerName string) (attacker, defender *Player) {
	if battle.Player1.Name == playerName {
		return battle.Player1, battle.Player2
	}
	return battle.Player2, battle.Player1
}

func getPlayer(battle *Battle, playerName string) *Player {
	if battle.Player1.Name == playerName {
		return battle.Player1
	}
	return battle.Player2
}

func getOpponent(battle *Battle, playerName string) *Player {
	if battle.Player1.Name == playerName {
		return battle.Player2
	}
	return battle.Player1
}
