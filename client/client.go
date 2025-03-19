package main

import (
	"bufio"
	"context"
	"fmt"

	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"

	pb "github.com/ayushsarode/termiXchat/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	serverAddr = "localhost:50051"
	timeFormat = "15:04:05"
)

// cmd colors
const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorPurple  = "\033[35m"
	colorCyan    = "\033[36m"
	colorGray    = "\033[37m"
	colorWhite   = "\033[97m"
	colorBold    = "\033[1m"
	colorItalic  = "\033[3m"
)

type chatClient struct {
	client    pb.ChatServiceClient
	conn      *grpc.ClientConn
	userID    int32
	username  string
	roomID    int32
	roomName  string
	inputChan chan string
	msgChan   chan *pb.ReceiveMessageResponse
	errChan   chan error
}

func main() {

	displayColorZenithLogo()
	
	chat := &chatClient{
		inputChan: make(chan string),
		msgChan:   make(chan *pb.ReceiveMessageResponse),
		errChan:   make(chan error),
	}
	
	// connecting to server
	fmt.Printf("%s🔌 Connecting to chat server...%s\n", colorYellow, colorReset)
	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Printf("%s❌ Failed to connect to server: %v%s\n", colorRed, err, colorReset)
		return
	}
	chat.conn = conn
	defer conn.Close()
	
	chat.client = pb.NewChatServiceClient(conn)
	
	// shutdown gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Printf("\n%s👋 Disconnecting from chat server...%s\n", colorYellow, colorReset)
		conn.Close()
		// Move cursor to bottom of screen and restore terminal
		fmt.Print("\033[r\033[999;999H\n")
		os.Exit(0)
	}()
	

	if !chat.authenticate() {
		return
	}
	
	if !chat.setupRoom() {
		return
	}
	
	// clears the screen and show chat interface
	clearScreen()
	chat.displayChatHeader()
	
	// message receiver in a goroutine
	go chat.receiveMessages()
	
	// user input handler
	go chat.handleUserInput()
	
	// Message display loop
	chat.messageLoop()
}

func (c *chatClient) authenticate() bool {
	reader := bufio.NewReader(os.Stdin)
	
	fmt.Println("\n1. Create new account\n2. Login to existing account")
	fmt.Print("Choose an option (1/2): ")
	option, _ := reader.ReadString('\n')
	option = strings.TrimSpace(option)
	
	var username, password string
	var err error
	
	fmt.Print("Enter your username: ")
	username, _ = reader.ReadString('\n')
	username = strings.TrimSpace(username)
	
	if username == "" {
		fmt.Printf("%s❌ Username cannot be empty. Please try again.%s\n", colorRed, colorReset)
		return false
	}
	
	fmt.Print("Enter your password: ")
	// term package to hide password input
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Printf("%s❌ Error reading password: %v%s\n", colorRed, err, colorReset)
		return false
	}
	password = string(passwordBytes)
	fmt.Println() 
	
	if password == "" {
		fmt.Printf("%s❌ Password cannot be empty. Please try again.%s\n", colorRed, colorReset)
		return false
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	var userResp *pb.CreateUserResponse
	if option == "1" {
		// Create new user
		userResp, err = c.client.CreateUser(ctx, &pb.CreateUserRequest{Username: username, Password: password})
		if err != nil {
			fmt.Printf("%s❌ Failed to create user: %v%s\n", colorRed, err, colorReset)
			return false
		}
		fmt.Printf("%s✅ User created successfully!%s\n", colorGreen, colorReset)
	} else {
		
		userResp, err = c.client.LoginUser(ctx, &pb.LoginUserRequest{Username: username, Password: password})
		if err != nil {
			fmt.Printf("%s❌ Login failed: %v%s\n", colorRed, err, colorReset)
			return false
		}
		fmt.Printf("%s✅ Login successful!%s\n", colorGreen, colorReset)
	}
	
	c.userID = userResp.UserId
	c.username = username
	
	return true
}

func (c *chatClient) setupRoom() bool {
	reader := bufio.NewReader(os.Stdin)
	
	fmt.Println("\n1. Create a new room\n2. Join an existing room\n3. List available rooms")
	fmt.Print("Choose an option (1/2/3): ")
	option, _ := reader.ReadString('\n')
	option = strings.TrimSpace(option)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	switch option {
	case "1":
		
		fmt.Print("Enter new room name: ")
		roomName, _ := reader.ReadString('\n')
		roomName = strings.TrimSpace(roomName)
		
		if roomName == "" {
			fmt.Printf("%s❌ Room name cannot be empty.%s\n", colorRed, colorReset)
			return false
		}
		
		roomResp, err := c.client.CreateRoom(ctx, &pb.CreateRoomRequest{Name: roomName})
		if err != nil {
			fmt.Printf("%s❌ Could not create room: %v%s\n", colorRed, err, colorReset)
			return false
		}
		
		c.roomID = roomResp.RoomId
		c.roomName = roomName
		fmt.Printf("%s✅ Created and joined room '%s' with ID: %d%s\n", colorGreen, roomName, c.roomID, colorReset)
		
	case "2":
		// join existing room
		fmt.Print("Enter room ID: ")
		var roomID int32
		_, err := fmt.Scanf("%d", &roomID)
		if err != nil {
			fmt.Printf("%s❌ Invalid room ID: %v%s\n", colorRed, err, colorReset)
			return false
		}
		// clean up input buffer
		reader.ReadString('\n')
		
		// Get room info
		roomInfo, err := c.client.GetRoomInfo(ctx, &pb.GetRoomInfoRequest{RoomId: roomID})
		if err != nil {
			fmt.Printf("%s❌ Failed to get room info: %v%s\n", colorRed, err, colorReset)
			return false
		}
		
		c.roomID = roomID
		c.roomName = roomInfo.Name
		fmt.Printf("%s✅ Joined room '%s' with ID: %d%s\n", colorGreen, c.roomName, c.roomID, colorReset)
		
	case "3":
		
		rooms, err := c.client.ListRooms(ctx, &pb.ListRoomsRequest{})
		if err != nil {
			fmt.Printf("%s❌ Failed to list rooms: %v%s\n", colorRed, err, colorReset)
			return false
		}
		
		fmt.Println("\nAvailable Rooms:")
		fmt.Println("----------------------------------")
		for _, room := range rooms.Rooms {
			fmt.Printf("ID: %d | Name: %s | Users: %d\n", room.RoomId, room.Name, room.UserCount)
		}
		fmt.Println("----------------------------------")
		
		// prompt to join a room
		fmt.Print("Enter room ID to join: ")
		var roomID int32
		_, err = fmt.Scanf("%d", &roomID)
		if err != nil {
			fmt.Printf("%s❌ Invalid room ID: %v%s\n", colorRed, err, colorReset)
			return false
		}
		// Clean up input buffer
		reader.ReadString('\n')
		
		// Find room name in the list
		var roomName string
		for _, room := range rooms.Rooms {
			if room.RoomId == roomID {
				roomName = room.Name
				break
			}
		}
		
		c.roomID = roomID
		c.roomName = roomName
		fmt.Printf("%s✅ Joined room '%s' with ID: %d%s\n", colorGreen, c.roomName, c.roomID, colorReset)
		
	default:
		fmt.Printf("%s❌ Invalid option.%s\n", colorRed, colorReset)
		return false
	}
	
	return true
}

func (c *chatClient) displayChatHeader() {
	width := 60
	fmt.Printf("%s╔", colorBlue)
	for i := 0; i < width-2; i++ {
		fmt.Print("═")
	}
	fmt.Print("╗\n")
	
	roomInfo := fmt.Sprintf(" ROOM: %s (ID: %d) ", c.roomName, c.roomID)
	padding := (width - len(roomInfo)) / 2
	fmt.Print("║")
	for i := 0; i < padding; i++ {
		fmt.Print(" ")
	}
	fmt.Printf("%s%s%s", colorBold, roomInfo, colorReset+colorBlue)
	for i := 0; i < padding; i++ {
		fmt.Print(" ")
	}
	// Adjust for odd widths
	if (width-len(roomInfo))%2 != 0 {
		fmt.Print(" ")
	}
	fmt.Print("║\n")
	
	fmt.Print("╠")
	for i := 0; i < width-2; i++ {
		fmt.Print("═")
	}
	fmt.Print("╣\n")
	
	helpText := " Type /help for commands "
	leftPadding := (width - len(helpText)) / 2
	fmt.Print("║")
	for i := 0; i < leftPadding; i++ {
		fmt.Print(" ")
	}
	fmt.Printf("%s%s%s", colorYellow, helpText, colorReset+colorBlue)
	for i := 0; i < leftPadding; i++ {
		fmt.Print(" ")
	}
	// Adjust for odd widths
	if (width-len(helpText))%2 != 0 {
		fmt.Print(" ")
	}
	fmt.Print("║\n")
	
	fmt.Print("╚")
	for i := 0; i < width-2; i++ {
		fmt.Print("═")
	}
	fmt.Printf("╝%s\n\n", colorReset)
}

func (c *chatClient) receiveMessages() {
	ctx := context.Background()
	stream, err := c.client.JoinRoom(ctx, &pb.JoinRoomRequest{
		UserId: c.userID,
		RoomId: c.roomID,
	})
	
	if err != nil {
		c.errChan <- fmt.Errorf("error joining room: %v", err)
		return
	}
	
	// Send system message that user joined
	c.msgChan <- &pb.ReceiveMessageResponse{
		MessageId: -1,
		Username:  "SYSTEM",
		Message:   fmt.Sprintf("You joined room '%s'", c.roomName),
		Timestamp: time.Now().Unix(),
		IsSystem:  true,
	}
	
	for {
		resp, err := stream.Recv()
		if err != nil {
			c.errChan <- fmt.Errorf("error receiving message: %v", err)
			return
		}
		c.msgChan <- resp
	}
}

func (c *chatClient) handleUserInput() {
	reader := bufio.NewReader(os.Stdin)
	
	for {
		input, err := reader.ReadString('\n')
		if err != nil {
			c.errChan <- fmt.Errorf("error reading input: %v", err)
			return
		}
		
		input = strings.TrimSpace(input)
		c.inputChan <- input
	}
}

func (c *chatClient) messageLoop() {
	
	fmt.Print("> ")
	
	for {
		select {
		case input := <-c.inputChan:
			if input == "" {
				fmt.Print("> ")
				continue
			}
			
			// Handle commands
			if strings.HasPrefix(input, "/") {
				c.handleCommand(input)
				continue
			}
			
			// normal message
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, err := c.client.SendMessage(ctx, &pb.SendMessageRequest{
				UserId:  c.userID,
				RoomId:  c.roomID,
				Message: input,
			})
			cancel()
			
			if err != nil {
				fmt.Printf("\r\033[K%s❌ Error sending message: %v%s\n> ", colorRed, err, colorReset)
			} else {
				
				fmt.Print("\r\033[K> ")
			}
			
		case msg := <-c.msgChan:
			
			timestamp := time.Unix(msg.Timestamp, 0).Format(timeFormat)
			
			// clear current input line
			fmt.Print("\r\033[K")
			
			// Format based on message type
			if msg.IsSystem {
				fmt.Printf("%s[%s] %s%s\n> ", colorGray, timestamp, msg.Message, colorReset)
			} else if msg.Username == c.username {
				// Own messages (Green username, white message)
				fmt.Printf("%s[%s] %s%s[You]:%s %s\n> ", colorGray, timestamp, colorBlue, msg.Username, colorReset, msg.Message)
			} else {
				// Others' messages (Blue username, white message)
				fmt.Printf("%s[%s] %s%s:%s %s\n> ", colorGray, timestamp, colorPurple, msg.Username, colorReset, msg.Message)
			}
			
		case err := <-c.errChan:
			fmt.Printf("\r\033[K%s❌ Error: %v%s\n> ", colorRed, err, colorReset)
		}
	}
}

func (c *chatClient) handleCommand(cmd string) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		fmt.Print("> ")
		return
	}
	
	switch parts[0] {
	case "/help":
		c.displayHelp()
		
	case "/quit", "/exit":
		fmt.Printf("\n%s👋 Thanks for using Zenith!%s\n", colorYellow, colorReset)
		c.conn.Close()
		os.Exit(0)
		
	case "/clear":
		clearScreen()
		c.displayChatHeader()
		
	case "/users":
		c.listUsers()
		
	case "/dm":
		if len(parts) < 3 {
			fmt.Printf("\r\033[K%s❌ Usage: /dm <username> <message>%s\n> ", colorRed, colorReset)
			return
		}
		recipient := parts[1]
		message := strings.Join(parts[2:], " ")
		c.sendDirectMessage(recipient, message)
		
	case "/nick", "/rename":
		if len(parts) < 2 {
			fmt.Printf("\r\033[K%s❌ Usage: /nick <new_name>%s\n> ", colorRed, colorReset)
			return
		}
		newName := parts[1]
		c.changeUsername(newName)
		
	case "/rooms":
		c.listRooms()
		
	case "/join":
		if len(parts) < 2 {
			fmt.Printf("\r\033[K%s❌ Usage: /join <room_id>%s\n> ", colorRed, colorReset)
			return
		}
		var roomID int32
		_, err := fmt.Sscanf(parts[1], "%d", &roomID)
		if err != nil {
			fmt.Printf("\r\033[K%s❌ Invalid room ID%s\n> ", colorRed, colorReset)
			return
		}
		c.changeRoom(roomID)

	default:
		fmt.Printf("\r\033[K%s❌ Unknown command: %s. Type /help for available commands.%s\n> ", colorRed, parts[0], colorReset)
	}
}

func (c *chatClient) displayHelp() {
	fmt.Print("\r\033[K")
	fmt.Printf("%s\n╔════════════════════════════════════════╗\n", colorCyan)
	fmt.Printf("║%s             AVAILABLE COMMANDS           %s║\n", colorYellow+colorBold, colorReset+colorCyan)
	fmt.Printf("╠════════════════════════════════════════╣\n")
	fmt.Printf("║ /help    - Show this help message      ║\n")
	fmt.Printf("║ /quit    - Exit the chat application   ║\n")
	fmt.Printf("║ /clear   - Clear the screen            ║\n")
	fmt.Printf("║ /users   - List users in current room  ║\n")
	fmt.Printf("║ /dm <user> <msg> - Send private message║\n")
	fmt.Printf("║ /nick <name> - Change your username    ║\n")
	fmt.Printf("║ /rooms   - List available rooms        ║\n")
	fmt.Printf("║ /join <id> - Join a different room     ║\n")
	fmt.Printf("╚════════════════════════════════════════╝%s\n", colorReset)
}

func (c *chatClient) listUsers() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	resp, err := c.client.ListUsers(ctx, &pb.ListUsersRequest{RoomId: c.roomID})
	cancel()
	
	if err != nil {
		fmt.Printf("\r\033[K%s❌ Error listing users: %v%s\n> ", colorRed, err, colorReset)
		return
	}
	
	fmt.Print("\r\033[K")
	fmt.Printf("%s\n══════ Users in %s ══════\n", colorYellow, c.roomName)
	for _, user := range resp.Users {
		status := "🟢 Online"
		if user.Username == c.username {
			fmt.Printf("  %s%s (you)%s - %s\n", colorGreen, user.Username, colorReset, status)
		} else {
			fmt.Printf("  %s - %s\n", user.Username, status)
		}
	}
	fmt.Printf("══════ Total: %d users ══════%s\n", len(resp.Users), colorReset)
	fmt.Print("> ")
}

func (c *chatClient) sendDirectMessage(recipient, message string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, err := c.client.SendDirectMessage(ctx, &pb.SendDirectMessageRequest{
		SenderId:    c.userID,
		RecipientUsername: recipient,
		Message:     message,
	})
	cancel()
	
	if err != nil {
		fmt.Printf("\r\033[K%s❌ Error sending DM: %v%s\n> ", colorRed, err, colorReset)
		return
	}
	
	fmt.Printf("\r\033[K%s[%s] %s(DM to %s): %s%s\n> ", 
		colorGray, time.Now().Format(timeFormat), 
		colorPurple, recipient, colorReset, message)
}

func (c *chatClient) changeUsername(newName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, err := c.client.ChangeUsername(ctx, &pb.ChangeUsernameRequest{
		UserId:      c.userID,
		NewUsername: newName,
	})
	cancel()
	
	if err != nil {
		fmt.Printf("\r\033[K%s❌ Error changing username: %v%s\n> ", colorRed, err, colorReset)
		return
	}
	
	oldName := c.username
	c.username = newName
	fmt.Printf("\r\033[K%sSystem: Username changed from %s to %s%s\n> ", 
		colorYellow, oldName, newName, colorReset)
}

func (c *chatClient) listRooms() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	resp, err := c.client.ListRooms(ctx, &pb.ListRoomsRequest{})
	cancel()
	
	if err != nil {
		fmt.Printf("\r\033[K%s❌ Error listing rooms: %v%s\n> ", colorRed, err, colorReset)
		return
	}
	
	fmt.Print("\r\033[K")
	fmt.Printf("%s\n══════ Available Rooms ══════\n", colorCyan)
	for _, room := range resp.Rooms {
		if room.RoomId == c.roomID {
			fmt.Printf("  %s[%d] %s (current)%s - %d users\n", 
				colorGreen, room.RoomId, room.Name, colorReset+colorCyan, room.UserCount)
		} else {
			fmt.Printf("  [%d] %s - %d users\n", room.RoomId, room.Name, room.UserCount)
		}
	}
	fmt.Printf("══════ Total: %d rooms ══════%s\n", len(resp.Rooms), colorReset)
	fmt.Print("> ")
}

func (c *chatClient) changeRoom(roomID int32) {
	// First get the room info
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	roomInfo, err := c.client.GetRoomInfo(ctx, &pb.GetRoomInfoRequest{RoomId: roomID})
	cancel()
	
	if err != nil {
		fmt.Printf("\r\033[K%s❌ Error joining room: %v%s\n> ", colorRed, err, colorReset)
		return
	}
	
	// leave current room
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	_, err = c.client.LeaveRoom(ctx, &pb.LeaveRoomRequest{
		UserId: c.userID,
		RoomId: c.roomID,
	})
	cancel()
	
	if err != nil {
		fmt.Printf("\r\033[K%s❌ Error leaving current room: %v%s\n> ", colorRed, err, colorReset)
		return
	}
	
	// Update client state
	oldRoomName := c.roomName
	c.roomID = roomID
	c.roomName = roomInfo.Name
	
	// Clear screen and show new chat header
	clearScreen()
	c.displayChatHeader()
	
	fmt.Printf("%sSystem: You left room '%s' and joined '%s'%s\n> ", 
		colorYellow, oldRoomName, c.roomName, colorReset)
	
	// Restart message receiver
	go c.receiveMessages()
}



func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func displayColorZenithLogo() {
	// ANSI color codes
	gold := "\033[33m"
	purple := "\033[35m"
	cyan := "\033[36m"
	reset := "\033[0m"
	bold := "\033[1m"
	
	logo := purple + bold + `
    ███████╗███████╗███╗   ██╗██╗████████╗██╗  ██╗
    ╚══███╔╝██╔════╝████╗  ██║██║╚══██╔══╝██║  ██║
      ███╔╝ █████╗  ██╔██╗ ██║██║   ██║   ███████║
     ███╔╝  ██╔══╝  ██║╚██╗██║██║   ██║   ██╔══██║
    ███████╗███████╗██║ ╚████║██║   ██║   ██║  ██║
    ╚══════╝╚══════╝╚═╝  ╚═══╝╚═╝   ╚═╝   ╚═╝  ╚═╝` + reset + `
                                           
    ` + gold + `------------------------------------------------` + reset + `
    ` + cyan + bold + `      Terminal Chat at the Peak of Excellence` + reset + `
    ` + gold + `------------------------------------------------` + reset + `
	`
	fmt.Println(logo)
}