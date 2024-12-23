package tcp_server

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hangman/internal/errs"
	"hangman/pkg/utils"
	"net"

	"google.golang.org/protobuf/proto"
)

type ILogger interface {
	Info(msg string)
	Warning(msg string)
	Error(msg string)
	Debug(msg string)
}

type Server struct {
	address  string
	handlers map[string]HandleFunc

	logger ILogger
}

// New создает новый сервер
func New(address string, logger ILogger) *Server {
	return &Server{
		address:  address,
		handlers: make(map[string]HandleFunc),
		logger:   logger,
	}
}

// RegisterHandler registers a handler for a specific command.
func (s *Server) RegisterHandler(command string, handler HandleFunc) {
	s.handlers[command] = handler
}

// Start запускает сервер и обрабатывает подключения
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}
	defer listener.Close()

	s.logger.Info(fmt.Sprintf("Server is listening on: %s", s.address))
	for {
		conn, err := listener.Accept()
		if err != nil {
			s.logger.Error(fmt.Sprintf("Failed to accept connection:%v", err))
			continue
		}

		go s.handleConnection(conn)
	}
}

// handleConnection обрабатывает подключение клиента
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	clientAddr := conn.RemoteAddr().String()
	s.logger.Info(fmt.Sprintf("New connection from %s", clientAddr))

	for {
		var response []byte
		message, err := readMessage(conn)
		if err != nil {
			response = CreateErrorResponse(StatusInternalServerError, err.Error())
		} else {
			response = s.processMessage(message, conn)
		}
		writeMessage(conn, response)
	}
}

func readMessage(conn net.Conn) ([]byte, error) {
	// Чтение заголовка (4 байта)
	header := make([]byte, 4)
	if _, err := conn.Read(header); err != nil {
		return nil, err
	}

	// Преобразование заголовка в длину сообщения
	messageLength := int(binary.BigEndian.Uint32(header))

	// Чтение тела сообщения
	message := make([]byte, messageLength)
	if _, err := conn.Read(message); err != nil {
		return nil, err
	}
	return message, nil
}

// writeMessage отправляет сообщение с заголовком длины
func writeMessage(conn net.Conn, message []byte) error {
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(len(message)))

	if _, err := conn.Write(header); err != nil {
		return err
	}

	if _, err := conn.Write(message); err != nil {
		return err
	}

	return nil
}

func (s *Server) processMessage(message []byte, conn net.Conn) []byte {
	// Парсим сообщение клиента
	var clientMsg ClientMessage
	if err := proto.Unmarshal(message, &clientMsg); err != nil {
		s.logger.Error(fmt.Sprintf("Failed to parse Protobuf message: %v", err))
		return CreateErrorResponse(StatusBadRequest, "Invalid Protobuf format")
	}
	s.logger.Info(fmt.Sprintf("Request: Command: %s, PayloadВize: %d bytes", clientMsg.Command, len(clientMsg.Payload)))
	s.logger.Debug(fmt.Sprintf("Request: Command: %s, Payload: %s", clientMsg.Command, string(clientMsg.Payload)))

	// Ищем обработчик для команды
	handler, exists := s.handlers[clientMsg.Command]
	if !exists {
		s.logger.Error(fmt.Sprintf("Unknown command: %s", clientMsg.Command))
		return CreateErrorResponse(StatusNotFound, "Unknown command")
	}

	// Обрабатываем полезную нагрузку
	responsePayload, err := handler(conn, clientMsg.Payload)
	if err != nil {
		var customErr *errs.Error
		if errors.As(err, &customErr) {
			s.logger.Error(fmt.Sprintf("Error in handler: %v", customErr))
			return CreateErrorResponse(customErr.Code, customErr.Message)
		}

		// Если ошибка неизвестного типа, возвращаем стандартный код
		s.logger.Error(fmt.Sprintf("Unexpected error: %v", err))
		return CreateErrorResponse(StatusInternalServerError, "Internal server error")
	}
	// Создаем ответ
	serverResp := &ServerResponse{
		StatusCode: StatusSuccess,
		Message:    "Success",
		Payload:    responsePayload,
	}

	// Сериализация ответа
	respBytes, err := proto.Marshal(serverResp)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to serialize response: %v", err))
		return CreateErrorResponse(StatusInternalServerError, "Internal server error")
	}
	s.logger.Debug(fmt.Sprintf("Response: %s, Payload: %s", serverResp.Message, responsePayload))
	return respBytes
}

// CreateErrorResponse формирует Protobuf-ответ с ошибкой
func CreateErrorResponse(code int32, msg string) []byte {
	serverResp := &ServerResponse{
		StatusCode: code,
		Message:    msg,
	}

	respBytes, _ := proto.Marshal(serverResp)
	return respBytes
}

func Notify(event string, payload []byte, clients []net.Conn) {
	localLogger := utils.NewCustomLogger(utils.LevelDebug)
	for _, conn := range clients {
		// Формируем сообщение
		serverResp := &ServerResponse{
			StatusCode: 2000,
			Message:    event,
			Payload:    payload,
		}

		// Сериализация ответа
		respBytes, err := proto.Marshal(serverResp)
		if err != nil {
			localLogger.Error(fmt.Sprintf("Failed to serialize notification for %s: %v", conn.RemoteAddr().String(), err))
			continue
		}

		// Отправка сообщения
		if err := writeMessage(conn, respBytes); err != nil {
			localLogger.Error(fmt.Sprintf("Failed to send notification to %s: %v", conn.RemoteAddr().String(), err))
			continue
		}
		localLogger.Info(fmt.Sprintf("Notify: %s, Payload: %s", serverResp.Message, serverResp.Payload))
	}
}
