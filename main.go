package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"slices"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type optionMap = map[string]*discordgo.ApplicationCommandInteractionDataOption

type game struct {
	ID          string
	PlayerID    string
	Deck        deck
	DealerCards []string
	PlayerCards []string
	Result      string
}

func (g *game) hit() (int, int) {
	g.PlayerCards = append(g.PlayerCards, g.Deck.deal())
	return g.react(true)
}

func (g *game) stay() (int, int) {
	return g.react(false)
}

func (g *game) react(playerHit bool) (playerScore int, dealerScore int) {
	for i := range g.DealerCards {
		numDealer := cardValues[g.DealerCards[i]]
		numPlayer := cardValues[g.PlayerCards[i]]
		dealerScore += numDealer
		playerScore += numPlayer
	}
	if playerScore > 21 && !slices.Contains(g.PlayerCards, "A") {
		g.Result = "DealerWin"
		return
	}
	if dealerScore > 21 && !slices.Contains(g.DealerCards, "A") {
		g.Result = "PlayerWin"
		return
	}
	dealerHit := false
	if dealerScore < 17 || (dealerScore == 17 && slices.Contains(g.DealerCards, "A")) {
		g.DealerCards = append(g.DealerCards, g.Deck.deal())
		dealerHit = true
	}
	dealerScore = dealerScore + cardValues[g.DealerCards[len(g.DealerCards)-1]]
	if dealerScore > 21 && !slices.Contains(g.DealerCards, "A") {
		g.Result = "PlayerWin"
		return
	}
	if !playerHit && !dealerHit {
		if playerScore > dealerScore {
			g.Result = "PlayerWin"
		} else if dealerScore > playerScore {
			g.Result = "DealerWin"
		}
		return
	}
	g.Result = "Playing"
	return
}

var games = make(map[string]*game)

var cardValues = map[string]int{
	"2":  2,
	"3":  3,
	"4":  4,
	"5":  5,
	"6":  6,
	"7":  7,
	"8":  8,
	"9":  9,
	"10": 10,
	"J":  10,
	"Q":  10,
	"K":  10,
	"A":  11,
}

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
	if i.User == nil {
		games[i.Member.User.ID] = &game{
			PlayerID:    i.Member.User.ID,
			Deck:        deck,
			DealerCards: dealerCards,
			PlayerCards: playerCards,
			Result:      "Playing",
		}
	} else {

		games[i.User.ID] = &game{
			PlayerID:    i.User.ID,
			Deck:        deck,
			DealerCards: dealerCards,
			PlayerCards: playerCards,
			Result:      "Playing",
		}
	}
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

	game := games[i.User.ID]
	var playerScore, dealerScore int
	switch data.CustomID {
	case "hit-btn":
		playerScore, dealerScore = game.hit()
	case "stay-btn":
		playerScore, dealerScore = game.stay()
	}
	var content string
	switch game.Result {
	case "Playing":
		content = fmt.Sprintf("Dealer Cards: ? + **%v**\r\nPlayer Cards: **%v** = **%d**", game.DealerCards[1:], game.PlayerCards, playerScore)
	case "DealerWin":
		content = fmt.Sprintf("Dealer Cards: **%v**\r\nPlayer Cards: **%v** = **%d**\r\nDealer won with a score of %d", game.DealerCards, game.PlayerCards, playerScore, dealerScore)
	case "PlayerWin":
		content = fmt.Sprintf("Dealer Cards: **%v**\r\nPlayer Cards: **%v** = **%d**\r\nPlayer Won with a score of %d", game.DealerCards, game.PlayerCards, playerScore, playerScore)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Components: i.Message.Components, // keep the same button row
		},
	})
}
