package server

import (
"database/sql"
	"sync"

	pb "github.com/ayushsarode/termiXchat/proto"
	"github.com/ayushsarode/termiXchat/db"
)

type Server struct {
	pb.UnimplementedChatServiceServer
	Mutex      sync.RWMutex
	Users      map[int32]*User
	Rooms      map[int32]*Room
	NextUserID int32
	NextRoomID int32
	NextMsgID  int32
	DB 		   *sql.DB
}

func NewServer() (*Server, error) {
	database, err := db.InitDB()
	if err != nil {
		return nil, err
	}

	return &Server{
		Users:      make(map[int32]*User),
		Rooms:      make(map[int32]*Room),
		NextUserID: 1,
		NextRoomID: 1,
		NextMsgID:  1,
		DB:         database,
	}, nil
}
