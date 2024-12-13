package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Stat struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type Pokemon struct {
	Name    string   `json:"name"`
	Type    []string `json:"type"`
	BaseExp int      `json:"base_exp"`
	Stats   []Stat   `json:"stats"` // Update to a slice of Stat structs
}

func fetchPokemonDetails(url string) (Pokemon, error) {
	resp, err := http.Get(url)
	if err != nil {
		return Pokemon{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Pokemon{}, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return Pokemon{}, err
	}

	// Parse Pokémon details
	var pokemon Pokemon
	pokemon.Name = result["name"].(string)
	pokemon.BaseExp = int(result["base_experience"].(float64))

	// Parse Types
	types := result["types"].([]interface{})
	for _, t := range types {
		typeName := t.(map[string]interface{})["type"].(map[string]interface{})["name"].(string)
		pokemon.Type = append(pokemon.Type, typeName)
	}

	// Parse Stats
	stats := result["stats"].([]interface{})
	for _, s := range stats {
		statName := s.(map[string]interface{})["stat"].(map[string]interface{})["name"].(string)
		statValue := int(s.(map[string]interface{})["base_stat"].(float64))
		pokemon.Stats = append(pokemon.Stats, Stat{Name: statName, Value: statValue})
	}

	return pokemon, nil
}

func fetchAllPokemon() ([]Pokemon, error) {
	// Fetch list of Pokémon from the API
	resp, err := http.Get("https://pokeapi.co/api/v2/pokemon?limit=10") // Adjust limit as needed
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	pokemonList := []Pokemon{}
	results := result["results"].([]interface{})
	for _, item := range results {
		url := item.(map[string]interface{})["url"].(string)
		pokemon, err := fetchPokemonDetails(url)
		if err != nil {
			fmt.Println("Error fetching Pokémon details:", err)
			continue
		}
		pokemonList = append(pokemonList, pokemon)
	}

	return pokemonList, nil
}

func saveToFile(pokemonList []Pokemon, filename string) error {
	data, err := json.MarshalIndent(pokemonList, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, 0644)
}

func main() {
	fmt.Println("Fetching Pokémon data...")
	pokemonList, err := fetchAllPokemon()
	if err != nil {
		fmt.Println("Error fetching Pokémon data:", err)
		return
	}

	if err := saveToFile(pokemonList, "pokedex.json"); err != nil {
		fmt.Println("Error saving Pokedex:", err)
		return
	}

	fmt.Println("Pokedex data saved to pokedex.json")
}
