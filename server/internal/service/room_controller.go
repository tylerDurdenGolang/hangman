package service

import (
	"context"
	"errors"
	"hangman/internal/domain"
	"hangman/internal/errs"
	tcp "hangman/pkg/tcp-server"
	"net"
	"time"
)

type RoomController struct {
	roomRepo    domain.IRoomRepository
	playerRepo  domain.IPlayerRepository
	gameService domain.IGameService
}

func NewRoomController(
	roomRepo domain.IRoomRepository,
	playerRepo domain.IPlayerRepository,
	gameService domain.IGameService,
) *RoomController {
	return &RoomController{
		roomRepo:    roomRepo,
		playerRepo:  playerRepo,
		gameService: gameService,
	}
}

func (rc *RoomController) CreateRoom(player string, roomID, password, category, difficulty string) (*domain.Room, error) {
	room := &domain.Room{
		ID:      roomID,
		Owner:   &player,
		Players: make(map[string]*domain.Player),
		//StateManager: domain.NewGameStateManager(),
		Password:     password,
		Category:     category,
		Difficulty:   difficulty,
		LastActivity: time.Now(),
		MaxPlayers:   3,
		RoomState:    domain.Waiting,
	}

	if err := rc.roomRepo.AddRoom(room); err != nil {
		return nil, err
	}

	//err := rc.playerRepo.AddPlayer(player)
	//if err != nil {
	//	return nil, err
	//}

	return room, nil
}

func (rc *RoomController) JoinRoom(ctx context.Context, username, roomID, password string) (*domain.Room, error) {
	room, err := rc.roomRepo.GetRoomByID(roomID)
	if err != nil {
		return nil, err
	}

	// Проверяем пароль
	if room.Password != "" && room.Password != password {
		return nil, errs.NewError(tcp.StatusUnauthorized, "incorrect password")
	}
	clientKey := domain.NewClientKey(username, password)

	// Проверяем текущего игрока
	existingPlayer := room.HasPlayer(username)

	switch room.RoomState {
	case domain.Waiting, domain.GameOver:
		if !existingPlayer && room.GetPlayerCount() >= room.MaxPlayers {
			// Если комната заполнена и это новый игрок
			return nil, errs.NewError(tcp.StatusConflict, "room is full, cannot join")
		}

		conn := ctx.Value("conn").(net.Conn)
		// Новый или реконнект существующего игрока
		player := domain.NewPlayer(ctx, &conn, username, 0)

		// Добавляем/обновляем игрока в репозитории
		err = rc.playerRepo.AddPlayer(clientKey, player)
		if err != nil {
			return nil, err
		}

		// Если игрока еще нет в комнате, добавляем
		if !existingPlayer {
			room.AddPlayer(player)
			room.MonitorContext(ctx, username)
		}

		return room, nil

	case domain.InProgress:
		// Только реконнект существующего игрока
		if !existingPlayer {
			return nil, errs.NewError(tcp.StatusConflict, "game already in progress, new players cannot join")
		}

		// Получаем информацию о существующем игроке
		existingPlayerData, err := rc.playerRepo.GetPlayerByKey(clientKey)
		if err != nil {
			return nil, err
		}

		// Обновляем информацию о клиенте в репозитории
		err = rc.playerRepo.AddPlayer(clientKey, existingPlayerData)
		if err != nil {
			return nil, err
		}

		return room, nil
	}

	return nil, errs.NewError(tcp.StatusInternalServerError, "unknown room state")
}

func (rc *RoomController) LeaveRoom(clientKey domain.ClientKey, roomID string) error {
	room, err := rc.roomRepo.GetRoomByID(roomID)
	if err != nil {
		return err
	}
	player, err := rc.playerRepo.GetPlayerByKey(clientKey)
	if err != nil {
		return err
	}
	room.KickPlayer(player.Username) // Удаление из комнаты
	//err = rc.playerRepo.RemovePlayerByUsername(player.Username) // Удаление из глобального репозитория
	//if err != nil {
	//	return err
	//}
	return nil
}

func (rc *RoomController) DeleteRoom(clientKey domain.ClientKey, roomID string) error {
	room, err := rc.roomRepo.GetRoomByID(roomID)
	if err != nil {
		return err // Комната не найдена
	}

	player, err := rc.playerRepo.GetPlayerByKey(clientKey)
	if err != nil {
		return err
	}

	// Проверяем, является ли пользователь владельцем комнаты
	if *room.Owner != player.Username {
		return errs.NewError(tcp.StatusUnauthorized, "only the owner can delete the room")
	}
	// Удаляем всех игроков из комнаты
	for _, player := range room.GetAllPlayers() {
		err := rc.playerRepo.RemovePlayerByUsername(player)
		if err != nil {
			return err
		}
	}

	// Удаляем комнату из репозитория
	if err := rc.roomRepo.RemoveRoom(roomID); err != nil {
		return err
	}

	return nil
}

func (rc *RoomController) forceDeleteRoom(roomID string) error {
	room, err := rc.roomRepo.GetRoomByID(roomID)
	if err != nil {
		return err // Комната не найдена
	}

	room.RLock()
	players := room.GetAllPlayers()
	room.RUnlock()

	for _, player := range players {
		err := rc.playerRepo.RemovePlayerByUsername(player)
		if err != nil {
			return err
		}
	}

	if err := rc.roomRepo.RemoveRoom(roomID); err != nil {
		return err
	}

	return nil
}

func (rc *RoomController) CleanupRooms(timeoutSeconds int) {
	ticker := time.NewTicker(time.Second * time.Duration(timeoutSeconds))
	defer ticker.Stop()

	for range ticker.C {
		rooms := rc.roomRepo.GetAllRooms()
		for _, room := range rooms {
			if time.Since(room.LastActivity) > time.Duration(timeoutSeconds)*time.Second {
				rc.forceDeleteRoom(room.ID)
			}
		}
	}
}

func (rc *RoomController) StartGame(clientKey domain.ClientKey, roomID string) error {
	room, err := rc.roomRepo.GetRoomByID(roomID)
	if err != nil {
		return err
	}
	//room.Lock()
	//defer room.Unlock()
	//if room.RoomState == domain.InProgress {
	//	return errors.New("game already started")
	//}
	player, err := rc.playerRepo.GetPlayerByKey(clientKey)
	if err != nil {
		return err
	}
	if *room.Owner != player.Username {
		return errs.NewError(tcp.StatusUnauthorized, "only the owner can start the game")
	}

	return rc.gameService.StartGame(room)
}

func (rc *RoomController) MakeGuess(clientKey domain.ClientKey, roomID string, letter rune) (bool, string, error) {
	player, err := rc.playerRepo.GetPlayerByKey(clientKey)
	if err != nil {
		return false, "", err
	}
	room, err := rc.roomRepo.GetRoomByID(roomID)
	if err != nil {
		return false, "", err
	}
	err = rc.playerRepo.UpdatePlayerActivity(clientKey)
	if err != nil {
		return false, "", err
	}
	return rc.gameService.MakeGuess(room, player, letter)
}

func (rc *RoomController) GetRoomState(roomID, password string) (*domain.Room, error) {
	room, err := rc.roomRepo.GetRoomByID(roomID)
	if err != nil {
		return nil, err
	}
	// Проверяем пароль
	if room.Password != "" && room.Password != password {
		return nil, errs.NewError(tcp.StatusUnauthorized, "incorrect password")
	}

	return room, nil
}

func (rc *RoomController) GetGameState(roomID string) (map[string]*domain.GameState, error) {
	room, err := rc.roomRepo.GetRoomByID(roomID)
	if err != nil {
		return nil, err
	}
	// Проверяем, закончилась ли игра у всех игроков
	err = rc.CheckAndSetGameOver(room)
	if err != nil {
		return nil, err
	}
	err = rc.HandleOwnerChange(room)
	if err != nil {
		return nil, err
	}
	return rc.gameService.GetGameState(room)
}

func (rc *RoomController) HandleOwnerChange(room *domain.Room) error {
	// Если в комнате никого не осталось, удаляем её
	if room.GetPlayerCount() == 0 {
		if err := rc.roomRepo.RemoveRoom(room.ID); err != nil {
			return err
		}
		return nil
	}

	// Переназначаем владельца (первый оставшийся игрок становится владельцем)
	room.Owner = &room.GetAllPlayers()[0]

	// Обновляем комнату в репозитории
	if err := rc.roomRepo.UpdateRoom(room); err != nil {
		return err
	}

	return nil
}

func (rc *RoomController) CheckAndSetGameOver(room *domain.Room) error {
	// Проверяем, есть ли игроки в комнате
	if room.GetPlayerCount() == 0 {
		return errors.New("no players in the room")
	}

	// Проверяем, у всех ли игроков игра закончилась
	allGameOver := true
	gameStates, err := rc.gameService.GetGameState(room)
	if err != nil {
		return err
	}
	for _, playerGameState := range gameStates {
		if !playerGameState.IsGameOver { // Если хотя бы у одного игрока игра не закончилась
			allGameOver = false
			break
		}
	}

	// Если все игроки завершили игру, переводим комнату в статус GameOver
	if allGameOver {
		room.RoomState = domain.GameOver
		if err := rc.roomRepo.UpdateRoom(room); err != nil {
			return err
		}
	}

	return nil
}
func (rc *RoomController) GetAllRooms() ([]*domain.Room, error) {
	return rc.roomRepo.GetAllRooms(), nil
}

func (rc *RoomController) GetLeaderboard() (map[string]int, error) {
	leaderboard := rc.playerRepo.GetPlayerUsernamesAndScores()
	return leaderboard, nil
}
