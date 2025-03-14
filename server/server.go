package main 


import (
	"context"
	

	pb "github.com/ayushsarode/termiXchat/proto"
)

type chatServer struct {
	pb.UnimplementedChatServiceServer

}

func (s *chatServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	var id int 
	err := db.QueryRow("INSERT INTO users (username) VALUES ($1) RETURNING id", req.Username).Scan(&id)

	if err != nil {
		return nil, err
	}
	return &pb.CreateUserResponse{UserId: int32(id), Username: req.Username}, nil
}

func(s *chatServer) CreateRoom(ctx context.Context, req *pb.CreateRoomRequest) (*pb.CreateRoomResponse, error) {
	var id int 
	err := db.QueryRow("INSERT INTO rooms (name) VALUES ($1) RETURNING id", req.Name).Scan(&id)

	if err != nil {
		return nil, err
	}

	return &pb.CreateRoomResponse{RoomId: int32(id), Name: req.Name}, nil
}

func(s *chatServer) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	_, err := db.Exec("INSERT INTO messages (user_id, room_id, message) VALUES ($1, $2, $3)", req.UserId, req.RoomId, req.Message)

	if err != nil {
		return nil, err
	}

	return &pb.SendMessageResponse{Status: "Message Sent"}, nil
}

func(s *chatServer) JoinRoom(req *pb.JoinRoomRequest, stream pb.ChatService_JoinRoomServer) error {
	rows, err := db.Query("SELECT users.username, messages.message FROM messages JOIN users ON messages.user_id = users.id WHERE messages.room_id = $1", req.RoomId)

	if err != nil {
		return err
	}	
	defer rows.Close()

	for rows.Next() {
		var username, message string

		if err := rows.Scan(&username, &message); err != nil {
				return err
			}

			stream.Send(&pb.ReceiveMessageResponse{Username: username, Message: message})
		
	}
	return nil
}

