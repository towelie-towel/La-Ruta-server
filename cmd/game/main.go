package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"chat/pkg/game"
	"chat/stats"
)

// game_stats
type GameStat struct {
	frameBuckets [8]int64
	activeGames  uint
}

func NewGameStat() *GameStat {
	return &GameStat{
		frameBuckets: [8]int64{0, 0, 0, 0, 0, 0, 0, 0},
	}
}

func (g *GameStat) AddDelta(delta int64) {
	if delta > 40_999 {
		g.frameBuckets[7] += 1
	} else if delta > 30_999 {
		g.frameBuckets[6] += 1
	} else if delta > 25_999 {
		g.frameBuckets[5] += 1
	} else if delta > 23_999 {
		g.frameBuckets[4] += 1
	} else if delta > 21_999 {
		g.frameBuckets[3] += 1
	} else if delta > 19_999 {
		g.frameBuckets[2] += 1
	} else if delta > 17_999 {
		g.frameBuckets[1] += 1
	} else {
		g.frameBuckets[0] += 1
	}
}

// main
type Game struct {
	Players [2]*game.Player
	sockets [2]game.Socket
	bullets []*game.Bullet
	queue   *game.GameQueue
	clock   game.IGameClock
	stats   *stats.GameStats
}

var addr = flag.String("addr", "0.0.0.0:42069", "http service address")

func main() {
	flag.Parse()
	log.SetFlags(0)

	server, err := game.NewServer()

	if err != nil {
		log.Fatalf("%+v\n", err)
	}

	go func() {
		for socketPair := range server.Out {
			// todo: how to listen to this?? more go funcs?
			NewGame(socketPair).Run()
		}
	}()

	http.HandleFunc("/", server.HandleNewConnection)
	fmt.Printf("listening on ws://%v", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func NewGame(sockets [2]game.Socket) *Game {
	return &Game{
		// TODO: finish this thing right
		[2]*game.Player{
			game.NewPlayer(game.Vector2D{2500, 0}, game.Vector2D{-1, 0}, 180),
			game.NewPlayer(game.Vector2D{-2500, 0}, game.Vector2D{1, 0}, 300), // THE LOSER
		},
		sockets,
		make([]*game.Bullet, 0),
		nil,
		&game.GameClock{},
		stats.NewGameStat(),
	}
}

func NewGameWithClock(sockets [2]game.Socket, clock game.IGameClock) *Game {
	return &Game{
		// TODO: finish this thing right
		[2]*game.Player{
			game.NewPlayerWithClock(game.Vector2D{2500, 0}, game.Vector2D{-1, 0}, 180, clock),
			game.NewPlayerWithClock(game.Vector2D{-2500, 0}, game.Vector2D{1, 0}, 300, clock), // THE LOSER
		},
		sockets,
		make([]*game.Bullet, 0),
		nil,
		clock,
		stats.NewGameStat(),
	}
}

func (g *Game) updateBulletPositions(delta int64) {
	deltaF := float64(delta) / 1000.0

	for _, bullet := range g.bullets {
		bullet.Geo.X += deltaF * bullet.Vel[0]
		bullet.Geo.Y += deltaF * bullet.Vel[1]
	}
}

func (g *Game) checkForBulletPlayerCollisions() *game.Player { // Note: Java for the bois
	// this is obvi not made fast.  Lets just get it done.
	var outPlayer *game.Player

loopMeDaddy:
	for _, player := range g.Players {

		for bIdx := 0; bIdx < len(g.bullets); bIdx += 1 {

			bullet := g.bullets[bIdx]
			if bullet.Geo.HasCollision(&player.Geo) {
				outPlayer = player
				break loopMeDaddy
			}
		}
	}

	return outPlayer
}

func (g *Game) checkBulletCollisions() {
loop_me_daddy:
	for idx1 := 0; idx1 < len(g.bullets); {
		bullet := g.bullets[idx1]
		for idx2 := idx1 + 1; idx2 < len(g.bullets); idx2 += 1 {
			bullet2 := g.bullets[idx2]
			if bullet.Geo.HasCollision(&bullet2.Geo) {
				// that is also very crappy code.  Why would I ever do this...
				g.bullets = append(g.bullets[:idx2], g.bullets[(idx2+1):]...)
				g.bullets = append(g.bullets[:idx1], g.bullets[(idx1+1):]...)
				break loop_me_daddy
			}
		}

		idx1 += 1
	}
}

func (g *Game) updateStateFromMessageQueue() {
	messages := g.queue.Flush()
	for _, message := range messages {
		if message.Message.Type == game.Fire {
			player := g.Players[message.From-1]
			fired := player.Fire()

			if fired {
				bullet := game.CreateBulletFromPlayer(player, 1.0)
				g.bullets = append(g.bullets, &bullet)
			}
		}
	}
}

func (g *Game) startGame() {
	g.queue = game.NewQueue()

	// unique..
	g.queue.Start(g.sockets[0], g.sockets[1])
}

func (g *Game) getSocket(player *game.Player) game.Socket {
	if player == g.Players[0] {
		return g.sockets[0]
	}
	return g.sockets[1]
}

func (g *Game) getOtherPlayer(player *game.Player) *game.Player {
	if player == g.Players[0] {
		return g.Players[1]
	}
	return g.Players[0]
}

func (g *Game) runGameLoop() {

	lastLoop := g.clock.Now().UnixMicro()
	var loser *game.Player

	stats.AddActiveGame()
	defer stats.RemoveActiveGame()

	// g.sockets[0].GetOutBound() <- game.CreateMessage(game.Play)
	// g.sockets[1].GetOutBound() <- game.CreateMessage(game.Play)

	for _, suckitBB := range g.sockets {
		suckitBB.GetOutBound() <- game.CreateMessage(game.Play)
	}

	for {
		start := g.clock.Now().UnixMicro()
		diff := start - lastLoop
		g.stats.AddDelta(diff)

		// 1.  check the message queue
		g.updateStateFromMessageQueue()

		// 2. update all the bullets
		g.updateBulletPositions(diff)

		// 3.  check for collisions
		g.checkBulletCollisions()

		// 3b.  check for player bullet collisions..
		loser = g.checkForBulletPlayerCollisions()
		if loser != nil {
			// 4.  Stop the loop if game is over
			break
		}

		// 5.   sleep for up to 16.66ms
		now := g.clock.Now().UnixMicro()
		time.Sleep(time.Duration(16_000-(now-start)) * time.Microsecond)

		lastLoop = start
	}

	// 4b.  Tell each player that they have won/lost.
	// 4b.  Close down the sockets and call it a day
	winnerMsg := game.CreateWinnerMessage(g.stats)
	loserMsg := game.CreateLoserMessage()
	winner := g.getOtherPlayer(loser)

	winnerSock := g.getSocket(winner)
	loserSock := g.getSocket(loser)

	winnerSock.GetOutBound() <- winnerMsg
	loserSock.GetOutBound() <- loserMsg

	winnerSock.Close()
	loserSock.Close()
}

// TODO: Bad naming here.  RENAME
// TODO: Make this into some sort of enum return.
func (g *Game) Run() WhenComplete {
	gameFinished := make(chan WaitForReadyResults)

	go func() {
		defer close(gameFinished)

		res := <-WaitForReady(g.sockets[0], g.sockets[1])

		// TODO: I don't like this.
		if res.timedout || res.readyError {
			gameFinished <- res
			return
		}

		g.startGame()
		g.runGameLoop()
	}()

	return gameFinished
}

// whait_for_ready

type WaitForReadyResults struct {
	readyError, timedout bool
}

type WhenComplete = <-chan WaitForReadyResults

func sendAndWait(s1 game.Socket, s2 game.Socket) chan bool {
	ready := make(chan bool)

	go func() {
		s1.GetOutBound() <- game.CreateMessage(game.ReadyUp)
		s2.GetOutBound() <- game.CreateMessage(game.ReadyUp)

		in1 := s1.GetInBound()
		in2 := s2.GetInBound()
		count := 0
		success := true

		for {
			select {
			case msg, ok := <-in1:
				if !ok {
					success = false
					break
				}

				if msg.Message.Type == game.ReadyUp {
					count += 1
					in1 = nil
				}

			case msg, ok := <-in2:
				if !ok {
					success = false
					break
				}

				if msg.Message.Type == game.ReadyUp {
					count += 1
					in2 = nil
				}
			}

			if count == 2 {
				break
			}
		}

		ready <- success
		close(ready)
	}()

	return ready
}

func WaitForReady(s0 game.Socket, s1 game.Socket) WhenComplete {
	ready := make(chan WaitForReadyResults)

	go func() {
		select {
		case success := <-sendAndWait(s0, s1):
			ready <- WaitForReadyResults{false, !success}
		case <-time.After(30 * time.Second):
			ready <- WaitForReadyResults{false, true}
		}
	}()

	return ready
}

func sendAndWaitBBy(sockets []game.Socket) chan bool {
	ready := make(chan bool)

	go func() {
		// readyCount := 0
		success := true

		for _, socket := range sockets {
			socket.GetOutBound() <- game.CreateMessage(game.ReadyUp)
		}

		/* for _, socket := range sockets {
			in := socket.GetInBound()
			msg, ok := <-in

			if !ok || msg.Message.Type != game.ReadyUp {
				success = false
			} else {
				readyCount++
			}

			if readyCount == len(sockets) {
				break
			}
		} */

		for _, socket := range sockets {
			in := socket.GetInBound()
			msg, ok := <-in

			if !ok || msg.Message.Type != game.ReadyUp {
				success = false
			}
		}

		ready <- success
		close(ready)
	}()

	return ready
}

func spreadSomeTaxisBBy(sockets []game.Socket) WhenComplete {
	ready := make(chan WaitForReadyResults)

	go func() {
		select {
		case success := <-sendAndWaitBBy(sockets):
			ready <- WaitForReadyResults{false, !success}
		case <-time.After(30 * time.Second):
			ready <- WaitForReadyResults{false, true}
		}
	}()

	return ready
}
