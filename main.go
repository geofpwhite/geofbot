package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
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

func onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
	}
}

func handleBlackjack(s *discordgo.Session, i *discordgo.InteractionCreate, om optionMap) {
	// zone := tracy.Zone("handleBlackjack")
	// defer zone.End()
	blackjackMessage(s, i, om)
}

func messageCreate(sh *stenchHandler) func(s *discordgo.Session, m *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return
		}
		// zone := tracy.Zone("messageCreate")
		// defer zone.End()
		if strings.HasPrefix(m.Content, "/") {
			// Ignore slash commands
			return
		}
		fields := strings.Fields(m.Content)
		fmt.Println("Fields: ", fields)
		if len(fields) > 0 {
			switch fields[0] {
			case "!eval":
				value := sh.eval(strings.Join(fields[1:], " "))
				fmt.Println(value)
				s.ChannelMessageSend(m.ChannelID, value)
			}
		}
	}
}

func blackjackMessage(s *discordgo.Session, i *discordgo.InteractionCreate, om optionMap) {
	// zone := tracy.Zone("blackjackMessage")
	// defer zone.End()
	dealerCards, playerCards := []string{}, []string{}
	deck := newDeck()
	deck.shuffle()
	dealerCard, deck := deck.deal()
	playerCard, deck := deck.deal()
	dealerCards = append(dealerCards, dealerCard)
	playerCards = append(playerCards, playerCard)
	dealerCard, deck = deck.deal()
	playerCard, deck = deck.deal()
	dealerCards = append(dealerCards, dealerCard)
	playerCards = append(playerCards, playerCard)
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
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
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
		},
	})
	if err != nil {
		fmt.Println("blackjackMessage respond error:", err)
	}
}

// buildContent formats the message content string based on the current game state.
func buildContent(g *game, playerScore, dealerScore int) string {
	switch g.Result {
	case "DealerWin":
		return fmt.Sprintf(
			"Dealer Cards: **%v**\r\nPlayer Cards: **%v** = **%d**\r\nDealer won with a score of %d",
			g.DealerCards, g.PlayerCards, playerScore, dealerScore,
		)
	case "PlayerWin":
		return fmt.Sprintf(
			"Dealer Cards: **%v**\r\nPlayer Cards: **%v** = **%d**\r\nPlayer Won with a score of %d",
			g.DealerCards, g.PlayerCards, playerScore, playerScore,
		)
	default: // "Playing"
		return fmt.Sprintf(
			"Dealer Cards: ? + **%v**\r\nPlayer Cards: **%v** = **%d**",
			g.DealerCards[1:], g.PlayerCards, playerScore,
		)
	}
}

// blackjackReset handles the reset-btn interaction. It tears down the old game
// and starts a fresh one for the user, updating the existing message in place.
func blackjackReset(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var userID string
	if i.User == nil {
		userID = i.Member.User.ID
	} else {
		userID = i.User.ID
	}

	// Remove old game (no-op if missing, e.g. after bot restart)
	delete(games, userID)

	// Deal a fresh game
	d := newDeck()
	d.shuffle()
	dealerCard1, d := d.deal()
	playerCard1, d := d.deal()
	dealerCard2, d := d.deal()
	playerCard2, d := d.deal()

	dealerCards := []string{dealerCard1, dealerCard2}
	playerCards := []string{playerCard1, playerCard2}

	games[userID] = &game{
		PlayerID:    userID,
		Deck:        d,
		DealerCards: dealerCards,
		PlayerCards: playerCards,
		Result:      "Playing",
	}

	content := fmt.Sprintf(
		"Dealer Cards: ? + **%v**\r\nPlayer Cards: **%v**",
		dealerCards[1:], playerCards,
	)

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Components: gameComponents("Playing"),
		},
	})
	if err != nil {
		fmt.Println("blackjackReset respond error:", err)
	}
}

func handleButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Ensure this is a component, not a slash command, modal, etc.
	if i.Type != discordgo.InteractionMessageComponent {
		return
	}
	data := i.MessageComponentData()

	// Reset is handled entirely by blackjackReset; no game lookup needed here.
	if data.CustomID == "reset-btn" {
		blackjackReset(s, i)
		return
	}

	var userID string
	if i.User == nil {
		userID = i.Member.User.ID
	} else {
		userID = i.User.ID
	}

	g := games[userID]
	var playerScore, dealerScore int
	switch data.CustomID {
	case "hit-btn":
		playerScore, dealerScore = g.hit()
	case "stay-btn":
		playerScore, dealerScore = g.stay()
	}

	fmt.Println(g.PlayerCards, g.DealerCards)

	content := buildContent(g, playerScore, dealerScore)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Components: gameComponents(g.Result),
		},
	})
}

func main() {
	flag.Parse()
	if *App == "" {
		content, err := os.ReadFile(".config")
		if err != nil {
			log.Fatal("application id is not set")
		}
		envvars := strings.Split(strings.ReplaceAll(string(content), "\r", ""), "\n")
		appid := envvars[0][strings.Index(envvars[0], ":")+1:]
		bottoken := envvars[1][strings.Index(envvars[1], ":")+1:]
		guildid := envvars[2][strings.Index(envvars[2], ":")+1:]
		*App = appid
		*Token = bottoken
		*Guild = guildid
	}
	cmd := exec.Command("escript", "stench", "-s")
	err := cmd.Start()
	if err != nil {
		fmt.Println(err)
		panic("can't start stench server")
	}
	defer fmt.Println(cmd.CombinedOutput())
	conn := starttcp()
	fmt.Println(conn)
	session, _ := discordgo.New("Bot " + *Token)

	s := newStenchHandler()
	session.AddHandler(messageCreate(s))

	session.AddHandler(handleButton)
	session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}
		data := i.ApplicationCommandData()
		fmt.Println(data.Name)
		switch data.Name {
		case "blackjack":
			fmt.Println("blackjack command received")
			handleBlackjack(s, i, parseOptions(data.Options))
		default:
			return
		}
	})

	session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as %s", r.User.String())
	})
	// session.AddHandler(onInteractionCreate)
	// session.AddHandler(handleMessage)

	err = session.Open()
	if err != nil {
		log.Fatalf("could not open session: %s", err)
	}

	_, err = session.ApplicationCommandBulkOverwrite(*App, "", commands)
	for _, c := range commands {
		session.ApplicationCommandCreate(*App, "", c)
	}
	if err != nil {
		log.Fatalf("could not register commands: %s", err)
	}

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)
	<-sigch

	err = session.Close()
	if err != nil {
		log.Printf("could not close session gracefully: %s", err)
	}
}
