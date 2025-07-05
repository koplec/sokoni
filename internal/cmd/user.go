package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/term"

	"github.com/koplec/sokoni/internal/model"
	"github.com/koplec/sokoni/internal/service"
)

// UserCommand handles user management CLI commands
type UserCommand struct {
	userService *service.UserService
}

// NewUserCommand creates a new user command handler
func NewUserCommand(conn *pgx.Conn) *UserCommand {
	return &UserCommand{
		userService: service.NewUserService(conn),
	}
}

// Execute executes the user command based on the provided arguments
func (uc *UserCommand) Execute(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return uc.showUsage()
	}

	switch args[0] {
	case "create":
		return uc.createUser(ctx, args[1:])
	case "list":
		return uc.listUsers(ctx, args[1:])
	case "show":
		return uc.showUser(ctx, args[1:])
	case "update":
		return uc.updateUser(ctx, args[1:])
	case "delete":
		return uc.deleteUser(ctx, args[1:])
	default:
		return uc.showUsage()
	}
}

// createUser creates a new user
func (uc *UserCommand) createUser(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: sokoni user create --username <username> --email <email> [--password <password>]")
	}

	var username, email, password string
	
	// Parse command line arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--username":
			if i+1 >= len(args) {
				return fmt.Errorf("--username requires a value")
			}
			username = args[i+1]
			i++
		case "--email":
			if i+1 >= len(args) {
				return fmt.Errorf("--email requires a value")
			}
			email = args[i+1]
			i++
		case "--password":
			if i+1 >= len(args) {
				return fmt.Errorf("--password requires a value")
			}
			password = args[i+1]
			i++
		default:
			return fmt.Errorf("unknown option: %s", args[i])
		}
	}

	if username == "" {
		return fmt.Errorf("username is required")
	}
	if email == "" {
		return fmt.Errorf("email is required")
	}

	// If password is not provided, prompt for it
	if password == "" {
		var err error
		password, err = uc.promptPassword("Enter password: ")
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		
		confirmPassword, err := uc.promptPassword("Confirm password: ")
		if err != nil {
			return fmt.Errorf("failed to read password confirmation: %w", err)
		}
		
		if password != confirmPassword {
			return fmt.Errorf("passwords do not match")
		}
	}

	req := model.CreateUserRequest{
		Username: username,
		Email:    email,
		Password: password,
	}

	user, err := uc.userService.CreateUser(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	fmt.Printf("User created successfully:\n")
	fmt.Printf("  ID: %d\n", user.ID)
	fmt.Printf("  Username: %s\n", user.Username)
	fmt.Printf("  Email: %s\n", user.Email)
	fmt.Printf("  Created: %s\n", user.CreatedAt.Format(time.RFC3339))

	return nil
}

// listUsers lists all users
func (uc *UserCommand) listUsers(ctx context.Context, args []string) error {
	users, err := uc.userService.GetAllUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	if len(users) == 0 {
		fmt.Println("No users found")
		return nil
	}

	fmt.Printf("Found %d users:\n", len(users))
	fmt.Println()
	fmt.Printf("%-5s %-20s %-30s %-20s\n", "ID", "Username", "Email", "Created")
	fmt.Println(strings.Repeat("-", 80))

	for _, user := range users {
		fmt.Printf("%-5d %-20s %-30s %-20s\n", 
			user.ID, 
			user.Username, 
			user.Email, 
			user.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}

// showUser shows details of a specific user
func (uc *UserCommand) showUser(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: sokoni user show <user_id>")
	}

	userID, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	user, err := uc.userService.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	fmt.Printf("User Details:\n")
	fmt.Printf("  ID: %d\n", user.ID)
	fmt.Printf("  Username: %s\n", user.Username)
	fmt.Printf("  Email: %s\n", user.Email)
	fmt.Printf("  Created: %s\n", user.CreatedAt.Format(time.RFC3339))
	fmt.Printf("  Updated: %s\n", user.UpdatedAt.Format(time.RFC3339))

	return nil
}

// updateUser updates an existing user
func (uc *UserCommand) updateUser(ctx context.Context, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: sokoni user update <user_id> [--username <username>] [--email <email>] [--password <password>]")
	}

	userID, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	var username, email, password string
	
	// Parse command line arguments
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--username":
			if i+1 >= len(args) {
				return fmt.Errorf("--username requires a value")
			}
			username = args[i+1]
			i++
		case "--email":
			if i+1 >= len(args) {
				return fmt.Errorf("--email requires a value")
			}
			email = args[i+1]
			i++
		case "--password":
			if i+1 >= len(args) {
				return fmt.Errorf("--password requires a value")
			}
			password = args[i+1]
			i++
		default:
			return fmt.Errorf("unknown option: %s", args[i])
		}
	}

	if username == "" && email == "" && password == "" {
		return fmt.Errorf("at least one field must be provided for update")
	}

	req := model.UpdateUserRequest{
		Username: username,
		Email:    email,
		Password: password,
	}

	user, err := uc.userService.UpdateUser(ctx, userID, req)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	fmt.Printf("User updated successfully:\n")
	fmt.Printf("  ID: %d\n", user.ID)
	fmt.Printf("  Username: %s\n", user.Username)
	fmt.Printf("  Email: %s\n", user.Email)
	fmt.Printf("  Updated: %s\n", user.UpdatedAt.Format(time.RFC3339))

	return nil
}

// deleteUser deletes a user
func (uc *UserCommand) deleteUser(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: sokoni user delete <user_id>")
	}

	userID, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Get user details first for confirmation
	user, err := uc.userService.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Confirm deletion
	fmt.Printf("Are you sure you want to delete user '%s' (ID: %d)? [y/N]: ", user.Username, user.ID)
	var response string
	fmt.Scanln(&response)
	response = strings.ToLower(strings.TrimSpace(response))

	if response != "y" && response != "yes" {
		fmt.Println("Deletion cancelled")
		return nil
	}

	err = uc.userService.DeleteUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	fmt.Printf("User '%s' (ID: %d) deleted successfully\n", user.Username, user.ID)
	return nil
}

// promptPassword prompts for password input without echoing
func (uc *UserCommand) promptPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // Add newline after password input
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(password)), nil
}

// showUsage displays usage information for user commands
func (uc *UserCommand) showUsage() error {
	fmt.Println("Usage: sokoni user <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  create    Create a new user")
	fmt.Println("  list      List all users")
	fmt.Println("  show      Show user details")
	fmt.Println("  update    Update user information")
	fmt.Println("  delete    Delete a user")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  sokoni user create --username john --email john@example.com")
	fmt.Println("  sokoni user create --username john --email john@example.com --password secret123")
	fmt.Println("  sokoni user list")
	fmt.Println("  sokoni user show 1")
	fmt.Println("  sokoni user update 1 --email newemail@example.com")
	fmt.Println("  sokoni user delete 1")
	
	return nil
}