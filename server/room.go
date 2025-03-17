package server

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/ayushsarode/termiXchat/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Room struct {
	ID        int32
	Name      string
	CreatedAt int64
	Users     map[int32]*User
	Clients   map[int32]pb.ChatService_JoinRoomServer
}

func (s *Server) CreateRoom(ctx context.Context, req *pb.CreateRoomRequest) (*pb.CreateRoomResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "room name cannot be empty")
	}

	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	for _, room := range s.Rooms {
		if room.Name == req.Name {
			return nil, status.Error(codes.AlreadyExists, "room name already exists")
		}
	}

	roomID := s.NextRoomID
	s.NextRoomID++

	s.Rooms[roomID] = &Room{
		ID:        roomID,
		Name:      req.Name,
		CreatedAt: time.Now().Unix(),
		Users:     make(map[int32]*User),
		Clients:   make(map[int32]pb.ChatService_JoinRoomServer),
	}

	return &pb.CreateRoomResponse{
		RoomId: roomID,
		Name:   req.Name,
	}, nil
}


func (s *Server) GetRoomInfo(ctx context.Context, req *pb.GetRoomInfoRequest) (*pb.GetRoomInfoResponse, error) {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()
	
	room, exists := s.Rooms[req.RoomId]
	if !exists {
		return nil, status.Error(codes.NotFound, "room not found")
	}
	
	return &pb.GetRoomInfoResponse{
		RoomId:    room.ID,
		Name:      room.Name,
		UserCount: int32(len(room.Users)),
		CreatedAt: room.CreatedAt,
	}, nil
}

func (s *Server) ListRooms(ctx context.Context, req *pb.ListRoomsRequest) (*pb.ListRoomsResponse, error) {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()
	
	rooms := make([]*pb.RoomInfo, 0, len(s.Rooms))
	for _, room := range s.Rooms {
		rooms = append(rooms, &pb.RoomInfo{
			RoomId:    room.ID,
			Name:      room.Name,
			UserCount: int32(len(room.Users)),
		})
	}
	
	return &pb.ListRoomsResponse{
		Rooms: rooms,
	}, nil
}


func (s *Server) JoinRoom(req *pb.JoinRoomRequest, stream pb.ChatService_JoinRoomServer) error {
	s.Mutex.Lock()
	
	user, userExists := s.Users[req.UserId]
	if !userExists {
		s.Mutex.Unlock()
		return status.Error(codes.NotFound, "user not found")
	}
	
	room, roomExists := s.Rooms[req.RoomId]
	if !roomExists {
		s.Mutex.Unlock()
		return status.Error(codes.NotFound, "room not found")
	}
	
	// add user to room
	room.Users[req.UserId] = user
	room.Clients[req.UserId] = stream
	
	// Create system message for user joining
	joinMsg := &pb.ReceiveMessageResponse{
		MessageId:  s.NextMsgID,
		Username:   "SYSTEM",
		Message:    fmt.Sprintf("%s has joined the room", user.Username),
		Timestamp:  time.Now().Unix(),
		IsSystem:   true,
		IsDirect:   false,
	}
	s.NextMsgID++
	
	// brodcast joining message to all users in that room
	for _, client := range room.Clients {
		if err := client.Send(joinMsg); err != nil {
			log.Printf("Failed to send join message: %v", err)
		}
	}
	
	s.Mutex.Unlock()
	
	// Keep the connection alive until the client disconnects
	<-stream.Context().Done()
	
	// Handle disconnection
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	
	// Check if room and user still exist
	if room, ok := s.Rooms[req.RoomId]; ok {
		delete(room.Users, req.UserId)
		delete(room.Clients, req.UserId)
		
		// Notify other users about the leave
		leaveMsg := &pb.ReceiveMessageResponse{
			MessageId:  s.NextMsgID,
			Username:   "SYSTEM",
			Message:    fmt.Sprintf("%s has left the room", user.Username),
			Timestamp:  time.Now().Unix(),
			IsSystem:   true,
			IsDirect:   false,
		}
		s.NextMsgID++
		
		for _, client := range room.Clients {
			if err := client.Send(leaveMsg); err != nil {
				log.Printf("Failed to send leave message: %v", err)
			}
		}
	}
	
	return nil
}

func (s *Server) LeaveRoom(ctx context.Context, req *pb.LeaveRoomRequest) (*pb.LeaveRoomResponse, error) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	
	user, userExists := s.Users[req.UserId]
	if !userExists {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	
	room, roomExists := s.Rooms[req.RoomId]
	if !roomExists {
		return nil, status.Error(codes.NotFound, "room not found")
	}
	
	// check if user is in the room
	if _, ok := room.Users[req.UserId]; !ok {
		return &pb.LeaveRoomResponse{
			Success: false,
			Message: "user is not in the room",
		}, nil
	}
	
	// remove the user from room
	delete(room.Users, req.UserId)
	delete(room.Clients, req.UserId)
	
	// notify other users bout user has left the room
	leaveMsg := &pb.ReceiveMessageResponse{
		MessageId:  s.NextMsgID,
		Username:   "SYSTEM",
		Message:    fmt.Sprintf("%s has left the room", user.Username),
		Timestamp:  time.Now().Unix(),
		IsSystem:   true,
		IsDirect:   false,
	}
	s.NextMsgID++
	
	for _, client := range room.Clients {
		if err := client.Send(leaveMsg); err != nil {
			log.Printf("Failed to send leave message: %v", err)
		}
	}
	
	return &pb.LeaveRoomResponse{
		Success: true,
		Message: "left room successfully",
	}, nil
}