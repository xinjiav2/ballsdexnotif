package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var (
	// Bot configuration loaded from .env
	triggerPhrase string
	roleToPing    string
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

	// Hardcoded role name
	roleName := "Ballsdex Spawn Notification"

	// Loop through guilds and roles to find the one we want
	for _, g := range event.Guilds {
		roles, err := s.GuildRoles(g.ID)
		if err != nil {
			log.Printf("Error fetching roles for guild %s: %v", g.ID, err)
			continue
		}
		for _, r := range roles {
			if r.Name == roleName {
				roleToPing = r.ID
				fmt.Printf("Found role '%s' with ID %s in guild %s\n", roleName, r.ID, g.ID)
			}
		}
	}

	if roleToPing == "" {
		log.Printf(" Role '%s' not found in any guilds!", roleName)
	}
}

// Event handler for new messages
func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Don't respond to the bot's own messages
	if strings.Contains(strings.ToLower(m.Content), strings.ToLower(triggerPhrase)) {
		if roleToPing != "" {
			mention := fmt.Sprintf("<@&%s>", roleToPing)
			pingMessage := fmt.Sprintf("%s - A countryball appeared!", mention)

			_, err := s.ChannelMessageSend(m.ChannelID, pingMessage)
			if err != nil {
				log.Printf("Error sending message: %v", err)
			}
		} else {
			log.Printf(" Role to ping is not set, message ignored.")
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
	case "!set_phrase":
		handleSetPhrase(s, m, args)
	case "!help":
		handleHelp(s, m)
	}
}

func handleSetPhrase(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if !isAdmin(s, m) {
		s.ChannelMessageSend(m.ChannelID, " You need administrator permissions to use this command!")
		return
	}

	if len(args) < 2 {
		s.ChannelMessageSend(m.ChannelID, " Usage: `!set_phrase <new phrase>`")
		return
	}

	// Join all arguments after the command to form the phrase
	newPhrase := strings.Join(args[1:], " ")
	triggerPhrase = newPhrase
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf(" Trigger phrase changed to: '%s'", newPhrase))
}

func handleHelp(s *discordgo.Session, m *discordgo.MessageCreate) {
	help := `
**Discord Phrase Bot Commands:**

**General Commands:**
• !help - Show this help message

**Admin Commands:**
• !set_phrase <phrase> - Change the trigger phrase

**Current Settings:**
• Trigger phrase: '%s'
`
	formattedHelp := fmt.Sprintf(help, triggerPhrase)
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
