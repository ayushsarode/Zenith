package server

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	pb "github.com/ayushsarode/termiXchat/proto"
)

type Message struct {
	ID        int32
	UserID    int32
	Username  string
	Content   string
	RoomID    int32
	Timestamp int64
	IsSystem  bool
	IsDirect  bool
}

func (s *Server) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	if req.Message == "" {
		return nil, status.Error(codes.InvalidArgument, "message cannot be empty")
	}

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

	if _, ok := room.Users[req.UserId]; !ok {
		return nil, status.Error(codes.PermissionDenied, "user is not in the room")
	}

	timestamp := time.Now().Unix()
	msg := &pb.ReceiveMessageResponse{
		MessageId:  s.NextMsgID,
		Username:   user.Username,
		Message:    req.Message,
		Timestamp:  timestamp,
		IsSystem:   false,
		IsDirect:   false,
	}
	s.NextMsgID++

	for _, client := range room.Clients {
		if err := client.Send(msg); err != nil {
			log.Printf("Failed to send message: %v", err)
		}
	}

	return &pb.SendMessageResponse{
		Status:    "sent",
		Timestamp: timestamp,
	}, nil
}

func (s *Server) SendDirectMessage(ctx context.Context, req *pb.SendDirectMessageRequest) (*pb.SendDirectMessageResponse, error) {
	if req.Message == "" {
		return nil, status.Error(codes.InvalidArgument, "message cannot be empty")
	}
	
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	
	sender, senderExists := s.Users[req.SenderId]
	if !senderExists {
		return nil, status.Error(codes.NotFound, "sender not found")
	}
	
	// find receiver by username
	var recipientID int32
	var recipientFound bool
	
	for id, user := range s.Users {
		if user.Username == req.RecipientUsername {
			recipientID = id
			recipientFound = true
			break
		}
	}
	
	if !recipientFound {
		return nil, status.Error(codes.NotFound, "recipient not found")
	}
	
	timestamp := time.Now().Unix()
	dmMsg := &pb.ReceiveMessageResponse{
		MessageId:  s.NextMsgID,
		Username:   sender.Username,
		Message:    req.Message,
		Timestamp:  timestamp,
		IsSystem:   false,
		IsDirect:   true,
	}
	s.NextMsgID++
	
	// find recipient in any room and send the message
	recipientFound = false
	for _, room := range s.Rooms {
		if client, ok := room.Clients[recipientID]; ok {
			recipientFound = true
			if err := client.Send(dmMsg); err != nil {
				log.Printf("Failed to send direct message: %v", err)
			}
		}
	}
	
	if !recipientFound {
		return nil, status.Error(codes.Unavailable, "recipient is not connected to any room")
	}
	
	return &pb.SendDirectMessageResponse{
		Status:    "sent",
		Timestamp: timestamp,
	}, nil
}
