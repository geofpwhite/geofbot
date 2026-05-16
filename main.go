package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

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
	{
		Name:        "connect4",
		Description: "play connect4",
		Options:     []*discordgo.ApplicationCommandOption{},
	},
}

var (
	Token = flag.String("token", "", "Bot authentication token")
	App   = flag.String("app", "", "Application ID")
	Guild = flag.String("guild", "", "Guild ID")
)

func handleBlackjack(s *discordgo.Session, i *discordgo.InteractionCreate, om optionMap) {
	// zone := tracy.Zone("handleBlackjack")
	// defer zone.End()
	blackjackMessage(s, i, om)
}

func connect4ColumnButtons(gameID string) []discordgo.MessageComponent {
	row1 := make([]discordgo.MessageComponent, 5)
	for i := 0; i < 5; i++ {
		row1[i] = discordgo.Button{
			Style:    discordgo.PrimaryButton,
			Label:    fmt.Sprintf("%d", i+1),
			CustomID: fmt.Sprintf("c4-drop-%s-%d", gameID, i),
		}
	}
	row2 := make([]discordgo.MessageComponent, 2)
	for i := 0; i < 2; i++ {
		row2[i] = discordgo.Button{
			Style:    discordgo.PrimaryButton,
			Label:    fmt.Sprintf("%d", i+6),
			CustomID: fmt.Sprintf("c4-drop-%s-%d", gameID, i+5),
		}
	}
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: row1},
		discordgo.ActionsRow{Components: row2},
	}
}

func handleConnect4(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var userID string
	if i.User == nil {
		userID = i.Member.User.ID
	} else {
		userID = i.User.ID
	}
	connect4Games[userID] = &connect4{
		ID:     userID,
		redID:  userID,
		Result: waiting,
		turn:   red,
	}
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("<@%s> wants to play Connect 4! 🔴 Click Join to play as 🟡.", userID),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Style:    discordgo.SuccessButton,
							Label:    "Join Game",
							CustomID: fmt.Sprintf("c4-join-%s", userID),
						},
					},
				},
			},
		},
	})
	if err != nil {
		fmt.Println("handleConnect4 respond error:", err)
	}
}

func handleConnect4Button(s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	var userID string
	if i.User == nil {
		userID = i.Member.User.ID
	} else {
		userID = i.User.ID
	}

	switch {
	case strings.HasPrefix(customID, "c4-join-"):
		gameID := customID[len("c4-join-"):]
		game, ok := connect4Games[gameID]
		if !ok || game.Result != waiting {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "This game is no longer available.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		if userID == game.redID {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "You can't join your own game!",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		game.yellowID = userID
		game.Result = redTurn
		content := fmt.Sprintf("🔴 <@%s> vs 🟡 <@%s>\n\n%s\n🔴 Red's turn!", game.redID, game.yellowID, game.renderBoard())
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    content,
				Components: connect4ColumnButtons(gameID),
			},
		})

	case strings.HasPrefix(customID, "c4-drop-"):
		rest := customID[len("c4-drop-"):]
		lastHyphen := strings.LastIndex(rest, "-")
		if lastHyphen < 0 {
			return
		}
		gameID := rest[:lastHyphen]
		col, err := strconv.Atoi(rest[lastHyphen+1:])
		if err != nil {
			return
		}
		game, ok := connect4Games[gameID]
		if !ok {
			return
		}
		if (game.turn == red && userID != game.redID) || (game.turn == yellow && userID != game.yellowID) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "It's not your turn!",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		game.makeMove(userID, col)
		if game.Result != redWin && game.Result != yellowWin {
			if game.isFull() {
				game.Result = draw
			} else if game.turn == red {
				game.Result = redTurn
			} else {
				game.Result = yellowTurn
			}
		}
		var content string
		var components []discordgo.MessageComponent
		switch game.Result {
		case redWin:
			content = fmt.Sprintf("🔴 <@%s> vs 🟡 <@%s>\n\n%s\n🔴 Red wins!", game.redID, game.yellowID, game.renderBoard())
		case yellowWin:
			content = fmt.Sprintf("🔴 <@%s> vs 🟡 <@%s>\n\n%s\n🟡 Yellow wins!", game.redID, game.yellowID, game.renderBoard())
		case draw:
			content = fmt.Sprintf("🔴 <@%s> vs 🟡 <@%s>\n\n%s\nIt's a draw!", game.redID, game.yellowID, game.renderBoard())
		case redTurn:
			content = fmt.Sprintf("🔴 <@%s> vs 🟡 <@%s>\n\n%s\n🔴 Red's turn!", game.redID, game.yellowID, game.renderBoard())
			components = connect4ColumnButtons(gameID)
		case yellowTurn:
			content = fmt.Sprintf("🔴 <@%s> vs 🟡 <@%s>\n\n%s\n🟡 Yellow's turn!", game.redID, game.yellowID, game.renderBoard())
			components = connect4ColumnButtons(gameID)
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    content,
				Components: components,
			},
		})
		if game.Result == redWin || game.Result == yellowWin || game.Result == draw {
			delete(connect4Games, gameID)
		}
	}
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
		blackjackGames[i.Member.User.ID] = &blackjack{
			PlayerID:    i.Member.User.ID,
			Deck:        deck,
			DealerCards: dealerCards,
			PlayerCards: playerCards,
			Result:      "Playing",
		}
	} else {
		blackjackGames[i.User.ID] = &blackjack{
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

// buildBlackJackContent formats the message content string based on the current game state.
func buildBlackJackContent(g *blackjack, playerScore, dealerScore int) string {
	switch g.Result {
	case "DealerWin":
		return fmt.Sprintf(
			"Dealer Cards: **%v**\r\nPlayer Cards: **%v** = **%d**\r\nDealer won with a score of %d",
			g.DealerCards, g.PlayerCards, playerScore, dealerScore,
		)
	case "PlayerWin":
		return fmt.Sprintf(
			"Dealer Cards: **%v**\r\nPlayer Cards: **%v** = **%d**\r\nPlayer won with a score of %d",
			g.DealerCards, g.PlayerCards, playerScore, playerScore,
		)
	case "Tie":
		return fmt.Sprintf(
			"Dealer Cards: **%v**\r\nPlayer Cards: **%v** = **%d**\r\nScores are tied at %d, so Player wins",
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
	delete(blackjackGames, userID)

	// Deal a fresh game
	d := newDeck()
	d.shuffle()
	dealerCard1, d := d.deal()
	playerCard1, d := d.deal()
	dealerCard2, d := d.deal()
	playerCard2, d := d.deal()

	dealerCards := []string{dealerCard1, dealerCard2}
	playerCards := []string{playerCard1, playerCard2}

	blackjackGames[userID] = &blackjack{
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

	if strings.HasPrefix(data.CustomID, "c4-") {
		handleConnect4Button(s, i, data.CustomID)
		return
	}

	var userID string
	if i.User == nil {
		userID = i.Member.User.ID
	} else {
		userID = i.User.ID
	}

	g := blackjackGames[userID]
	var playerScore, dealerScore int
	switch data.CustomID {
	case "hit-btn":
		playerScore, dealerScore = g.hit()
	case "stay-btn":
		playerScore, dealerScore = g.stay()
	}

	fmt.Println(g.PlayerCards, g.DealerCards)

	content := buildBlackJackContent(g, playerScore, dealerScore)
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
	conn := starttcp()
	fmt.Println(conn, "connection")
	session, _ := discordgo.New("Bot " + *Token)
	fmt.Println(session)
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
		case "connect4":
			fmt.Println("connect4 command received")
			handleConnect4(s, i)
		default:
			return
		}
	})

	session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as %s", r.User.String())
	})
	// session.AddHandler(onInteractionCreate)
	// session.AddHandler(handleMessage)

	fmt.Println(session.ApplicationCommands(*App, ""))
	err = session.Open()
	if err != nil {
		log.Fatalf("could not open session: %s", err)
	}

	acbo, err := session.ApplicationCommandBulkOverwrite(*App, "", commands)
	fmt.Println(acbo)
	// for _, c := range commands {
	// 	session.ApplicationCommandCreate(*App, "", c)
	// }
	if err != nil {
		log.Fatalf("could not register commands: %s", err)
	}

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt, syscall.SIGTERM)
	<-sigch

	if cmd.Process != nil {
		cmd.Process.Kill()
	}
	conn.Close()

	err = session.Close()
	if err != nil {
		log.Printf("could not close session gracefully: %s", err)
	}
}
