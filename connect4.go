package main

import (
	"fmt"
	"strings"
)

type color uint

const (
	empty color = iota
	red
	yellow
)

type c4result string

const (
	waiting    c4result = "Waiting for opponent"
	redWin     c4result = "Red wins"
	yellowWin  c4result = "Yellow wins"
	draw       c4result = "Draw"
	redTurn    c4result = "Red's turn"
	yellowTurn c4result = "Yellow's turn"
)

var connect4Games = make(map[string]*connect4)

type connect4 struct {
	ID              string
	redID, yellowID string // Player IDs
	board           [6][7]color
	Result          c4result
	turn            color
}

func (g *connect4) makeMove(playerID string, column int) {
	if g.Result == waiting ||
		(g.turn == red && playerID != g.redID) ||
		(g.turn == yellow && playerID != g.yellowID) {
		return
	}
	if column < 0 || column >= 7 || g.board[5][column] != empty {
		return
	}
	row := 5
	for row > 0 && g.board[row-1][column] == empty {
		row--
	}
	g.board[row][column] = g.turn

	g.turn = g.turn%2 + 1
	g.scanForWin()
}

func (g *connect4) isFull() bool {
	for i := range 7 {
		if g.board[5][i] == empty {
			return false
		}
	}
	return true
}

func (g *connect4) renderBoard() string {
	var sb strings.Builder
	for r := 5; r >= 0; r-- {
		for c := 0; c < 7; c++ {
			switch g.board[r][c] {
			case empty:
				sb.WriteString("⚫")
			case red:
				sb.WriteString("🔴")
			case yellow:
				sb.WriteString("🟡")
			}
		}
		sb.WriteString("\n")
	}
	sb.WriteString("1️⃣2️⃣3️⃣4️⃣5️⃣6️⃣7️⃣")
	return sb.String()
}

// scan for win by the player who just made a move
func (g *connect4) scanForWin() {
	type (
		coords struct {
			x, y int
		}
		qNode struct {
			directionStreak int
			streak          int
			coords          coords
		}
	)
	team := g.turn%2 + 1
	visited := make(map[coords]map[int]int)
	queue := make([]qNode, 0)
	for i := range g.board {
		if g.board[0][i] == team {
			queue = append(queue, qNode{-1, 1, coords{0, i}})
		}
	}
	for len(queue) > 0 {
		cur := queue[0]
		m, ok := visited[cur.coords]
		if !ok {
			visited[cur.coords] = make(map[int]int)
			m = visited[cur.coords]
		}
		m[cur.directionStreak] = cur.streak
		queue = queue[1:]
		fmt.Println(cur)
		if cur.streak >= 4 {
			switch team {
			case red:
				g.Result = redWin
			case yellow:
				g.Result = yellowWin
			}
			return
		}
		toCheck := []coords{
			{cur.coords.x, cur.coords.y + 1},
			{cur.coords.x + 1, cur.coords.y + 1},
			{cur.coords.x + 1, cur.coords.y},
			{cur.coords.x + 1, cur.coords.y - 1},
		}
		for i, coords := range toCheck {
			if coords.x >= 0 && coords.x < 7 && coords.y >= 0 &&
				coords.y < 7 && g.board[coords.x][coords.y] == team && visited[coords][i] < cur.streak+1 {
				qn := qNode{i, 2, coords}
				if cur.directionStreak == i {
					qn = qNode{cur.directionStreak, cur.streak + 1, coords}
				}
				queue = append(queue, qn)
			}
		}
	}
}
