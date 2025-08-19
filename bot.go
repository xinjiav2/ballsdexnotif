package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var (
	// Bot configuration loaded from .env
	triggerPhrase string
	usersToPing   []string
	botToken      string
)

func init() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	// Load configuration from environment variables
	botToken = os.Getenv("DISCORD_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("DISCORD_BOT_TOKEN environment variable is required")
	}

	triggerPhrase = os.Getenv("TRIGGER_PHRASE")
	if triggerPhrase == "" {
		triggerPhrase = "important announcement" // Default value
	}

	// Parse comma-separated user IDs
	userIdsStr := os.Getenv("USERS_TO_PING")
	if userIdsStr != "" {
		usersToPing = strings.Split(userIdsStr, ",")
		// Trim whitespace from each ID
		for i, id := range usersToPing {
			usersToPing[i] = strings.TrimSpace(id)
		}
	}
}

func main() {
	// Create a new Discord session
	dg, err := discordgo.New("Bot " + botToken)
	if err != nil {
		log.Fatal("Error creating Discord session: ", err)
	}

	// Register event handlers
	dg.AddHandler(onReady)
	dg.AddHandler(onMessageCreate)

	// Set intents to receive message content
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentsMessageContent

	// Open connection to Discord
	err = dg.Open()
	if err != nil {
		log.Fatal("Error opening Discord connection: ", err)
	}
	defer dg.Close()

	fmt.Println("Bot is running! Press Ctrl+C to stop.")

	// Wait for interrupt signal
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

// Event handler for when bot is ready
func onReady(s *discordgo.Session, event *discordgo.Ready) {
	fmt.Printf("%s has connected to Discord!\n", event.User.Username)
	fmt.Printf("Bot is in %d guilds\n", len(event.Guilds))
	fmt.Printf("Watching for phrase: '%s'\n", triggerPhrase)
	fmt.Printf("Users to ping: %v\n", usersToPing)
}

// Event handler for new messages
func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Don't respond to the bot's own messages
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Check if the trigger phrase is in the message (case-insensitive)
	if strings.Contains(strings.ToLower(m.Content), strings.ToLower(triggerPhrase)) {
		// Create mentions for specified users
		var mentions []string
		for _, userID := range usersToPing {
			if userID != "" {
				mentions = append(mentions, fmt.Sprintf("<@%s>", userID))
			}
		}

		if len(mentions) > 0 {
			// Send the ping message
			pingMessage := fmt.Sprintf(" %s - A countryball appeared!",
				strings.Join(mentions, " "))

			_, err := s.ChannelMessageSend(m.ChannelID, pingMessage)
			if err != nil {
				log.Printf("Error sending message: %v", err)
			}

			// Add reaction to the original message
			err = s.MessageReactionAdd(m.ChannelID, m.ID, "")
			if err != nil {
				log.Printf("Error adding reaction: %v", err)
			}
		}
	}

	// Handle commands
	handleCommands(s, m)
}

// Simple command handler
func handleCommands(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Check if message starts with command prefix
	if !strings.HasPrefix(m.Content, "!") {
		return
	}

	args := strings.Fields(m.Content)
	if len(args) == 0 {
		return
	}

	command := strings.ToLower(args[0])

	switch command {
	case "!add_ping":
		handleAddPing(s, m, args)
	case "!remove_ping":
		handleRemovePing(s, m, args)
	case "!ping_list":
		handlePingList(s, m)
	case "!set_phrase":
		handleSetPhrase(s, m, args)
	case "!help":
		handleHelp(s, m)
	}
}

func handleAddPing(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if !isAdmin(s, m) {
		s.ChannelMessageSend(m.ChannelID, "❌ You need administrator permissions to use this command!")
		return
	}

	if len(args) < 2 {
		s.ChannelMessageSend(m.ChannelID, "❌ Usage: `!add_ping <@user>` or `!add_ping <user_id>`")
		return
	}

	var userID string
	// Check if it's a mention
	if strings.HasPrefix(args[1], "<@") && strings.HasSuffix(args[1], ">") {
		userID = strings.Trim(args[1], "<@!>")
	} else {
		// Assume it's a user ID
		userID = args[1]
		// Validate it's a number
		if _, err := strconv.ParseUint(userID, 10, 64); err != nil {
			s.ChannelMessageSend(m.ChannelID, "❌ Invalid user ID format!")
			return
		}
	}

	// Check if user is already in the list
	for _, id := range usersToPing {
		if id == userID {
			s.ChannelMessageSend(m.ChannelID, "❌ User is already in the ping list!")
			return
		}
	}

	usersToPing = append(usersToPing, userID)
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("✅ Added <@%s> to the ping list!", userID))
}

func handleRemovePing(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if !isAdmin(s, m) {
		s.ChannelMessageSend(m.ChannelID, "❌ You need administrator permissions to use this command!")
		return
	}

	if len(args) < 2 {
		s.ChannelMessageSend(m.ChannelID, "❌ Usage: `!remove_ping <@user>` or `!remove_ping <user_id>`")
		return
	}

	var userID string
	// Check if it's a mention
	if strings.HasPrefix(args[1], "<@") && strings.HasSuffix(args[1], ">") {
		userID = strings.Trim(args[1], "<@!>")
	} else {
		userID = args[1]
	}

	// Find and remove user from the list
	for i, id := range usersToPing {
		if id == userID {
			usersToPing = append(usersToPing[:i], usersToPing[i+1:]...)
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("✅ Removed <@%s> from the ping list!", userID))
			return
		}
	}

	s.ChannelMessageSend(m.ChannelID, "❌ User is not in the ping list!")
}

func handlePingList(s *discordgo.Session, m *discordgo.MessageCreate) {
	if len(usersToPing) == 0 {
		s.ChannelMessageSend(m.ChannelID, "📝 No users in the ping list!")
		return
	}

	var mentions []string
	for _, userID := range usersToPing {
		if userID != "" {
			mentions = append(mentions, fmt.Sprintf("<@%s>", userID))
		}
	}

	if len(mentions) > 0 {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("📝 Current ping list: %s", strings.Join(mentions, ", ")))
	} else {
		s.ChannelMessageSend(m.ChannelID, "📝 No valid users found in the ping list!")
	}
}

func handleSetPhrase(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if !isAdmin(s, m) {
		s.ChannelMessageSend(m.ChannelID, "❌ You need administrator permissions to use this command!")
		return
	}

	if len(args) < 2 {
		s.ChannelMessageSend(m.ChannelID, "❌ Usage: `!set_phrase <new phrase>`")
		return
	}

	// Join all arguments after the command to form the phrase
	newPhrase := strings.Join(args[1:], " ")
	triggerPhrase = newPhrase
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("✅ Trigger phrase changed to: '%s'", newPhrase))
}

func handleHelp(s *discordgo.Session, m *discordgo.MessageCreate) {
	help := `
**Discord Phrase Bot Commands:**

**General Commands:**
• !help - Show this help message
• !ping_list - Show current users in the ping list

**Admin Commands:**
• !add_ping <@user> - Add a user to the ping list
• !remove_ping <@user> - Remove a user from the ping list
• !set_phrase <phrase> - Change the trigger phrase

**Current Settings:**
• Trigger phrase: '%s'
• Users in ping list: %d
`
	formattedHelp := fmt.Sprintf(help, triggerPhrase, len(usersToPing))
	s.ChannelMessageSend(m.ChannelID, formattedHelp)
}

// Check if user has administrator permissions
func isAdmin(s *discordgo.Session, m *discordgo.MessageCreate) bool {
	// Get guild member
	member, err := s.GuildMember(m.GuildID, m.Author.ID)
	if err != nil {
		return false
	}

	// Get guild
	guild, err := s.Guild(m.GuildID)
	if err != nil {
		return false
	}

	// Check if user is guild owner
	if m.Author.ID == guild.OwnerID {
		return true
	}

	// Check roles for administrator permission
	for _, roleID := range member.Roles {
		role, err := s.State.Role(m.GuildID, roleID)
		if err != nil {
			continue
		}
		if role.Permissions&discordgo.PermissionAdministrator != 0 {
			return true
		}
	}

	return false
}
