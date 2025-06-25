package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type optionMap = map[string]*discordgo.ApplicationCommandInteractionDataOption

func parseOptions(options []*discordgo.ApplicationCommandInteractionDataOption) (om optionMap) {
	om = make(optionMap)
	for _, opt := range options {
		om[opt.Name] = opt
	}
	return
}

func interactionAuthor(i *discordgo.Interaction) *discordgo.User {
	if i.Member != nil {
		return i.Member.User
	}
	return i.User
}

func handleEcho(s *discordgo.Session, i *discordgo.InteractionCreate, opts optionMap) {
	builder := new(strings.Builder)
	if v, ok := opts["author"]; ok && v.BoolValue() {
		author := interactionAuthor(i.Interaction)
		builder.WriteString("**" + author.String() + "** says: ")
	}
	builder.WriteString(opts["message"].StringValue())

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: builder.String(),
		},
	})

	if err != nil {
		log.Panicf("could not respond to interaction: %s", err)
	}
}

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "echo",
		Description: "Say something through a bot",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "message",
				Description: "Contents of the message",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			},
			{
				Name:        "author",
				Description: "Whether to prepend message's author",
				Type:        discordgo.ApplicationCommandOptionBoolean,
			},
		},
	},
	{
		Name:        "blackjack",
		Description: "play blackjack",
		Options:     []*discordgo.ApplicationCommandOption{},
	},
}

var (
	Token = flag.String("token", "", "Bot authentication token")
	App   = flag.String("app", "", "Application ID")
	Guild = flag.String("guild", "", "Guild ID")
)

func main() {
	flag.Parse()
	if *App == "" {
		log.Fatal("application id is not set")
	}

	session, _ := discordgo.New("Bot " + *Token)
	session.AddHandler(handleButton)
	session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}

		data := i.ApplicationCommandData()
		switch data.Name {
		default:
			return
		case "echo":
			handleEcho(s, i, parseOptions(data.Options))
		case "blackjack":
			handleBlackjack(s, i, parseOptions(data.Options))
		}

	})

	session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as %s", r.User.String())
	})

	// session.AddHandler(onInteractionCreate)
	// session.AddHandler(handleMessage)

	_, err := session.ApplicationCommandBulkOverwrite(*App, *Guild, commands)
	if err != nil {
		log.Fatalf("could not register commands: %s", err)
	}

	err = session.Open()
	if err != nil {
		log.Fatalf("could not open session: %s", err)
	}

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)
	<-sigch

	err = session.Close()
	if err != nil {
		log.Printf("could not close session gracefully: %s", err)
	}

}

/* func onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
// Only care about component (e.g. button) interactions
s.ChannelMessageSendEmbedReply(i.ChannelID, &discordgo.MessageEmbed{
	Image: &discordgo.MessageEmbedImage{},
}, i.Message.Reference())

if i.Type != discordgo.InteractionMessageComponent {
	return
}
fmt.Println(s, i)

data := i.MessageComponentData()
switch data.CustomID {
case "press_me_button":
	// Acknowledge & reply
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "You pressed the button! 🎉",
		},
	})
} */
// }

func handleBlackjack(s *discordgo.Session, i *discordgo.InteractionCreate, om optionMap) {
	blackjackMessage(s, i, om)
}

type deck []string

func newDeck() deck {
	return []string{
		"2", "2", "2", "2",
		"3", "3", "3", "3",
		"4", "4", "4", "4",
		"5", "5", "5", "5",
		"6", "6", "6", "6",
		"7", "7", "7", "7",
		"8", "8", "8", "8",
		"9", "9", "9", "9",
		"10", "10", "10", "10",
		"J", "J", "J", "J",
		"Q", "Q", "Q", "Q",
		"K", "K", "K", "K",
		"A", "A", "A", "A",
	}
}

func (d deck) shuffle() {
	// Implement a simple shuffle algorithm, e.g., Fisher-Yates
	for i := len(d) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		d[i], d[j] = d[j], d[i]
	}
}
func (d deck) deal() string {
	if len(d) == 0 {
		return ""
	}
	card := d[0]
	d = d[1:]
	return card
}

func blackjackMessage(s *discordgo.Session, i *discordgo.InteractionCreate, om optionMap) {
	dealerCards, playerCards := []string{}, []string{}
	deck := newDeck()
	deck.shuffle()
	dealerCards = append(dealerCards, deck.deal(), deck.deal())
	playerCards = append(playerCards, deck.deal(), deck.deal())
	msg := &discordgo.MessageSend{
		Content: fmt.Sprintf("Dealer Cards: ? + **%v**\r\nPlayer Cards: **%v**", dealerCards[1:], playerCards),
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Style:    discordgo.DangerButton,
						Label:    "Hit",
						CustomID: "hit-btn",
					},
					discordgo.Button{
						Style:    discordgo.DangerButton,
						Label:    "Stay",
						CustomID: "stay-btn",
					},
				},
			},
		},
	}
	m, err := s.ChannelMessageSendComplex(i.ChannelID, msg)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(m)
}
func handleButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Ensure this is a component, not a slash command, modal, etc.
	if i.Type != discordgo.InteractionMessageComponent {
		return
	}
	data := i.MessageComponentData()

	switch data.CustomID {
	}

	// Parse the old count out of the message content.
	re := regexp.MustCompile(`Count:\s\*\*(\d+)\*\*`)
	matches := re.FindStringSubmatch(i.Message.Content)
	count := 0
	if len(matches) == 2 {
		count, _ = strconv.Atoi(matches[1])
	}
	count++
	// 1️⃣ ACK the click so the client stops spinning
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    fmt.Sprintf("🔢 Count: **%d**", count),
			Components: i.Message.Components, // keep the same button row
		},
	})
}
