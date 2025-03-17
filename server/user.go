package server

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	pb "github.com/ayushsarode/termiXchat/proto"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type User struct {
	ID       int32
	Username string
	Password string
}

// CreateUser creates a new user in the database
func (s *Server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	if req.Username == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "username and password are required")
	}

	// Hash the password before storing
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to hash password")
	}

	// Check if username already exists
	var exists bool
	err = s.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)", req.Username).Scan(&exists)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("database error: %v", err))
	}
	if exists {
		return nil, status.Error(codes.AlreadyExists, "username already exists")
	}

	// Insert user into database
	var userID int32
	err = s.DB.QueryRow(
		"INSERT INTO users (username, password) VALUES ($1, $2) RETURNING id",
		req.Username,
		string(hashedPassword),
	).Scan(&userID)

	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create user: %v", err))
	}

	// Update in-memory map for active users
	s.Mutex.Lock()
	s.Users[userID] = &User{
		ID:       userID,
		Username: req.Username,
		Password: string(hashedPassword),
	}
	s.Mutex.Unlock()

	return &pb.CreateUserResponse{
		UserId:   userID,
		Username: req.Username,
	}, nil
}

// LoginUser authenticates a user
func (s *Server) LoginUser(ctx context.Context, req *pb.LoginUserRequest) (*pb.CreateUserResponse, error) {
	var (
		userID   int32
		username string
		password string
	)

	// Get user from database
	err := s.DB.QueryRow(
		"SELECT id, username, password FROM users WHERE username = $1",
		req.Username,
	).Scan(&userID, &username, &password)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.Unauthenticated, "invalid username or password")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("database error: %v", err))
	}

	// Compare passwords
	err = bcrypt.CompareHashAndPassword([]byte(password), []byte(req.Password))
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid username or password")
	}

	// Add user to in-memory cache if not present
	s.Mutex.Lock()
	if _, exists := s.Users[userID]; !exists {
		s.Users[userID] = &User{
			ID:       userID,
			Username: username,
			Password: password,
		}
	}
	s.Mutex.Unlock()

	return &pb.CreateUserResponse{
		UserId:   userID,
		Username: username,
	}, nil
}

// ChangeUsername updates a user's username in the database
func (s *Server) ChangeUsername(ctx context.Context, req *pb.ChangeUsernameRequest) (*pb.ChangeUsernameResponse, error) {
	if req.NewUsername == "" {
		return nil, status.Error(codes.InvalidArgument, "new username cannot be empty")
	}

	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	// Check if the new username already exists
	var exists bool
	err := s.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = $1 AND id != $2)", 
		req.NewUsername, req.UserId).Scan(&exists)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("database error: %v", err))
	}
	if exists {
		return &pb.ChangeUsernameResponse{
			Success: false,
			Message: "username already taken",
		}, nil
	}

	// Get old username
	var oldUsername string
	err = s.DB.QueryRow("SELECT username FROM users WHERE id = $1", req.UserId).Scan(&oldUsername)
	if err != nil {
		if err == sql.ErrNoRows {
			return &pb.ChangeUsernameResponse{
				Success: false,
				Message: "user not found",
			}, nil
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("database error: %v", err))
	}

	// Update username in database
	_, err = s.DB.Exec("UPDATE users SET username = $1 WHERE id = $2", req.NewUsername, req.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to update username: %v", err))
	}

	// Update in-memory user
	user, exists := s.Users[req.UserId]
	if exists {
		user.Username = req.NewUsername
	}

	// Broadcast username change to all rooms the user is in
	for _, room := range s.Rooms {
		if _, ok := room.Users[req.UserId]; ok {
			// Broadcast username change to all users in the room
			systemMsg := &pb.ReceiveMessageResponse{
				MessageId:  s.NextMsgID,
				Username:   "SYSTEM",
				Message:    fmt.Sprintf("%s changed their username to %s", oldUsername, req.NewUsername),
				Timestamp:  time.Now().Unix(),
				IsSystem:   true,
				IsDirect:   false,
			}
			s.NextMsgID++
			for _, client := range room.Clients {
				if err := client.Send(systemMsg); err != nil {
					log.Printf("Failed to send system message: %v", err)
				}
			}
		}
	}

	return &pb.ChangeUsernameResponse{
		Success: true,
		Message: "username changed successfully",
	}, nil
}

// func returns a list of users in a room
func (s *Server) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()

	room, exists := s.Rooms[req.RoomId]
	if !exists {
		return nil, status.Error(codes.NotFound, "room not found")
	}

	users := make([]*pb.UserInfo, 0, len(room.Users))
	
	// for each user in the room, fetch their latest info from the database
	for id := range room.Users {
		var username string
		err := s.DB.QueryRow("SELECT username FROM users WHERE id = $1", id).Scan(&username)
		if err != nil {
			log.Printf("Error fetching username for user ID %d: %v", id, err)
			continue
		}
		
		users = append(users, &pb.UserInfo{
			UserId:     id,
			Username:   username,
			//rn we're using just current time
			LastActive: time.Now().Unix(),
		})
	}

	return &pb.ListUsersResponse{
		Users: users,
	}, nil
}