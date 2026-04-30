package main

import (
	"math/rand"
	"slices"
)

type game struct {
	ID          string
	PlayerID    string
	Deck        deck
	DealerCards []string
	PlayerCards []string
	Result      string
}

func (g *game) hit() (int, int) {
	// zone := tracy.Zone("game.hit")
	// defer zone.End()
	playerCard, newDeck := g.Deck.deal()
	g.PlayerCards = append(g.PlayerCards, playerCard)
	g.Deck = newDeck
	return g.react(true)
}

func (g *game) stay() (int, int) {
	// zone := tracy.Zone("game.stay")
	// defer zone.End()
	return g.react(false)
}

func (g *game) react(playerHit bool) (playerScore int, dealerScore int) {
	// zone := tracy.Zone("game.react")
	// defer zone.End()
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
		dealerCard, newDeck := g.Deck.deal()
		g.DealerCards = append(g.DealerCards, dealerCard)
		g.Deck = newDeck
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

type deck []string

func (d deck) deal() (string, deck) {
	if len(d) == 0 {
		return "", d
	}
	card := d[0]
	d = d[1:]
	return card, d
}

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
