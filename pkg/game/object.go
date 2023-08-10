package game

import (
	"math"
	"sync"
	"time"
)

// game_queue

type QueueMessage struct {
	From    uint
	Message GameMessage
}

type GameQueue struct {
	messages []*QueueMessage
	killChan chan struct{}
	mutex    sync.Mutex
}

func NewQueue() *GameQueue {
	return &GameQueue{
		messages: make([]*QueueMessage, 0),
		killChan: make(chan struct{}),
		mutex:    sync.Mutex{},
	}
}

// TODO: Learn about context
func (q *GameQueue) Start(s0, s1 Socket) {
	go func() {
	label_for_you:
		for {
			select {
			case msg := <-s0.GetInBound():
				q.mutex.Lock()
				q.messages = append(q.messages, &QueueMessage{
					1,
					msg.Message,
				})
				q.mutex.Unlock()
			case msg := <-s1.GetInBound():
				q.mutex.Lock()
				q.messages = append(q.messages, &QueueMessage{
					2,
					msg.Message,
				})
				q.mutex.Unlock()
			case <-q.killChan:
				break label_for_you
			}
		}
	}()
}

func (q *GameQueue) Stop() {
	q.killChan <- struct{}{}
}

func (q *GameQueue) emptyMessages() bool {
	out := true
	for _, msg := range q.messages {
		out = out && msg == nil
		if !out {
			break
		}
	}

	return out
}

func (q *GameQueue) Flush() []*QueueMessage {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.emptyMessages() {
		return nil
	}

	messages := q.messages
	q.messages = make([]*QueueMessage, 0)

	return messages
}

// game_clock
type IGameClock interface {
	Now() time.Time
}

type GameClock struct{}

func (g *GameClock) Now() time.Time {
	return time.Now()
}

type SyntheticGameClock struct {
	now time.Time
}

func (g *SyntheticGameClock) SetNow(now time.Time) {
	g.now = now
}

func (g *SyntheticGameClock) Now() time.Time {
	return g.now
}

// phisics
type Vector2D = [2]float64

type Updatable interface {
	Update(xDelta, yDelta float64)
	GetVelocity() *Vector2D
}

func UpdateItems[T Updatable](items []T, delta uint) {
	for _, item := range items {
		vel := item.GetVelocity()
		item.Update(vel[0]*float64(delta), vel[1]*float64(delta))
	}
}

// geometry
type AABB struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

func (a *AABB) HasCollision(b *AABB) bool {

	if a.X > b.X+b.Width || b.X > a.X+a.Width {
		return false
	}

	if a.Y > b.Y+b.Height || b.Y > a.Y+a.Height {
		return false
	}

	return true
}

func (a *AABB) HasCollisionFast(b *AABB, width float64) bool {
	return math.Abs(a.X-b.X) < width
}

func (a *AABB) SetPosition(x, y float64) {
	a.X = x
	a.Y = y
}

// objects
type Player struct {
	Geo          AABB
	Dir          Vector2D
	FireRate     int64
	lastFireTime int64
	clock        IGameClock
}

type Bullet struct {
	Geo AABB
	Vel Vector2D
}

const PLAYER_WIDTH = 100
const PLAYER_HEIGHT = 100
const BULLET_WIDTH = 35
const BULLET_HEIGHT = 3

func NewPlayer(pos, dir Vector2D, fireRate int64) *Player {
	return &Player{
		Geo: AABB{
			X:      pos[0],
			Y:      pos[1],
			Width:  PLAYER_WIDTH,
			Height: PLAYER_HEIGHT,
		},
		Dir:          dir,
		FireRate:     fireRate,
		lastFireTime: 0,
		clock:        &GameClock{},
	}
}

func NewPlayerWithClock(pos, dir Vector2D, fireRate int64, clock IGameClock) *Player {
	return &Player{
		Geo: AABB{
			X:      pos[0],
			Y:      pos[1],
			Width:  PLAYER_WIDTH,
			Height: PLAYER_HEIGHT,
		},
		Dir:          dir,
		FireRate:     fireRate,
		lastFireTime: 0,
		clock:        clock,
	}
}

func (p *Player) Fire() bool {
	now := time.Now().UnixMilli()

	if p.FireRate > now-p.lastFireTime {
		return false
	}

	p.lastFireTime = now
	return true
}

func newBullet() Bullet {
	return Bullet{
		AABB{0, 0, BULLET_WIDTH, BULLET_HEIGHT},
		Vector2D{0, 0},
	}
}

func CreateBulletFromPlayer(player *Player, speed float64) Bullet {
	bullet := newBullet()

	if player.Dir[0] == 1 {
		bullet.Geo.SetPosition(
			player.Geo.X+player.Geo.Width+1,
			0)
	} else {
		bullet.Geo.SetPosition(
			player.Geo.X-BULLET_WIDTH-1,
			0)
	}

	bullet.Vel[0] = player.Dir[0] * speed
	bullet.Vel[1] = player.Dir[1] * speed

	return bullet
}

func (b *Bullet) Update(xDelta, yDelta float64) {
	b.Geo.X += xDelta
	b.Geo.Y += yDelta
}
