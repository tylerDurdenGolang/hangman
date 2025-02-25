using System.Net.Sockets;
using client.Domain.Events;
using Tcp;

namespace client.Domain.Interfaces
{
    public interface IGameDriver
    {
        string GetCurrentPlayerUsername();
        CheckUsernameResponse CheckUsername(string username);
        CreateRoomResponse CreateRoom(string roomId, string password, string category, string difficulty);
        UpdateRoomResponse UpdateRoom(
            string roomId,
            string password,
            string? category,
            string? difficulty,
            string? newPassword
        );
        StartGameResponse StartGame(string roomId, string password);
        JoinRoomResponse JoinRoom(string roomId, string password);
        LeaveRoomResponse LeaveFromRoom(string roomId, string password);
        DeleteRoomResponse DeleteRoom(string roomId, string password);
        GuessLetterResponse SendGuess(string roomId, string password, char letter);
        RoomGameStateResponse GetGameState(string roomId);
        GetRoomStateResponse GetRoomState(string roomId, string password);
        GetAllRoomsResponse GetAllRooms();
        GetLeaderBoardResponse GetLeaderBoard();
        Stream GetRoomStream();
        Task<GameEvent?> TryToGetServerEventAsync(CancellationToken cancellationToken);
        GameEvent? TryToGetServerEvent(CancellationToken cancellationToken);
    }
}
