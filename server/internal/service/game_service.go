package service

import (
	"errors"
	"fmt"
	"hangman/internal/domain"
	"hangman/internal/repository"
)

type GameServiceImpl struct {
	wordsRepo *repository.WordsRepository
}

func NewGameService(repo *repository.WordsRepository) domain.IGameService {
	return &GameServiceImpl{
		wordsRepo: repo,
	}
}

// StartGame запускает игру
func (gs *GameServiceImpl) StartGame(room *domain.Room) error {
	//room.Lock()
	//defer room.Unlock()

	if room.StateManager == nil {
		room.StateManager = domain.NewGameStateManager()
	}
	players := room.GetAllPlayers()

	fmt.Println("Получил игроков")
	for _, player := range players {
		word, err := gs.wordsRepo.GetRandomWord(room.Category)
		if err != nil {
			return err
		}
		attemptsCount := gs.wordsRepo.GetAttempts(word, room.Difficulty)
		room.StateManager.AddGame(word, domain.PlayerUsername(player), attemptsCount)
	}
	fmt.Println("получил игроков")
	return nil
}

// MakeGuess обрабатывает ход игрока
func (gs *GameServiceImpl) MakeGuess(room *domain.Room, player *domain.Player, letter rune) (bool, string, error) {
	room.RLock()
	defer room.RUnlock()
	//room.UpdateActivity()

	if room.StateManager == nil {
		return false, "", errors.New("no game in this room")
	}

	return room.StateManager.MakeGuess(domain.PlayerUsername(player.Username), letter)
}

// GetGameState возвращает текущее состояние игры для всех игроков в комнате
func (gs *GameServiceImpl) GetGameState(room *domain.Room) (map[string]*domain.GameState, error) {
	room.RLock()
	defer room.RUnlock()

	if room.StateManager == nil {
		return nil, errors.New("no game in this room")
	}

	// Создаём карту для хранения состояния игры каждого игрока
	playerGameStates := make(map[string]*domain.GameState)
	players := room.GetAllPlayers()
	fmt.Println("Получил игроков")
	// Получаем состояние для каждого игрока в комнате
	for _, player := range players {
		gameState, err := room.StateManager.GetState(domain.PlayerUsername(player))
		if err != nil {
			return nil, err
		}
		playerGameStates[player] = &gameState
	}

	return playerGameStates, nil
}
