package main

import (
	"log"
	"net"

	"google.golang.org/grpc"

	pb "github.com/ayushsarode/termiXchat/proto"
)


func main() {
	InitDB()

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

    chatService := &chatServer{roomUsers: make(map[int][]chan *pb.MessageResponse)}

	pb.RegisterChatServiceServer(grpcServer, chatService)

	log.Println("Server started at :50051")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to server: %v", err)
	}
}