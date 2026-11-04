package enter

import (
	"context"
	"game/internal/controller/room"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcache"
	"github.com/gorilla/websocket"
)

var (
	clientCache    *gcache.Cache
	cacheCtx       = context.Background()
	cacheKeyPrefix = "ws:client:"
	// 心跳配置，统一常量
	heartbeatInterval = 5 * time.Second  // 客户端上报心跳间隔
	heartbeatTimeout  = 60 * time.Second // 多久没心跳判定掉线
)

// Client 一个用户连接实例
type Client struct {
	conn      *websocket.Conn
	userID    string
	heartbeat time.Time // 最后收到心跳时间
	sendChan  chan []byte
	pid       int
	roomChan  chan *room.Room
	cancel    context.CancelFunc
	mutex     sync.RWMutex
	closed    int32 // 0=未关闭 1=已关闭，原子操作
	ctx       context.Context
}

var (
	rm            *room.RoomManager
	rmMu          sync.RWMutex
	allowedOrigin = room.GetAllowedOrigin()
)

type PlayerDTO struct {
	ID       int    `json:"ID"`
	Name     string `json:"Name"`
	RoomID   string `json:"RoomID"`
	Type     int    `json:"Type"`
	CardNum  int    `json:"CardNum"`
	Point    int64  `json:"Point"`
	UserId   string `json:"UserId"`
	RoomType int    `json:"RoomType"`
	InPoint  int64  `json:"InPoint"`
}

func init() {
	rmMu.Lock()
	if rm == nil {
		rm = room.NewRoomManager()
	}
	rmMu.Unlock()
	clientCache = gcache.New()
}

func getClientCacheKey(userID string) string {
	return cacheKeyPrefix + userID
}

func addClient(client *Client) error {
	key := getClientCacheKey(client.userID)
	return clientCache.Set(cacheCtx, key, client, time.Hour)
}

func getClient(userID string) (*Client, error) {
	key := getClientCacheKey(userID)
	val, err := clientCache.Get(cacheCtx, key)
	if err != nil || val == nil || val.IsNil() {
		return nil, nil
	}
	cli, ok := val.Val().(*Client)
	if !ok {
		_, _ = clientCache.Remove(cacheCtx, key)
		return nil, nil
	}
	return cli, nil
}

// removeClientIfMatch 只有缓存内指针和传入cli一致才删除，防止误删新连接
func removeClientIfMatch(userID string, cli *Client) bool {
	key := getClientCacheKey(userID)
	val, err := clientCache.Get(cacheCtx, key)
	if err != nil || val == nil || val.IsNil() {
		return false
	}
	existCli, ok := val.Val().(*Client)
	if !ok {
		_, _ = clientCache.Remove(cacheCtx, key)
		return false
	}
	if existCli == cli {
		_, _ = clientCache.Remove(cacheCtx, key)
		return true
	}
	return false
}

// ====================== 【唯一关闭入口】所有下线都走这里！重点 ======================
func closeConnection(c *Client, reason string) {
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return
	}
	logCtx := context.Background()
	g.Log().Infof(logCtx, "start close connection, userID:%s, ptr:%p, reason:%s", c.userID, c, reason)

	c.cancel()

	closeErr := c.conn.Close()
	g.Log().Infof(logCtx, "conn closed, userID:%s, ptr:%p, closeErr:%v", c.userID, c, closeErr)
	c.conn = nil

	c.mutex.Lock()
	if c.sendChan != nil {
		close(c.sendChan)
		c.sendChan = nil
	}
	if c.roomChan != nil {
		close(c.roomChan)
		c.roomChan = nil
	}
	c.mutex.Unlock()

	removed := removeClientIfMatch(c.userID, c)
	g.Log().Infof(logCtx, "finish close connection, userID:%s, ptr:%p, cacheRemoved:%t", c.userID, c, removed)
}

// ====================== 新建连接入口（ws升级成功后调用） ======================
func NewClient(conn *websocket.Conn, userID string, pid int) *Client {
	// 【重连关键】先查有没有旧连接，有就踢掉旧的
	oldCli, err := getClient(userID)
	if err == nil && oldCli != nil {
		closeConnection(oldCli, "reconnect_kick_old")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cli := &Client{
		conn:      conn,
		userID:    userID,
		pid:       pid,
		heartbeat: time.Now(),
		sendChan:  make(chan []byte, 32), // 缓冲通道
		roomChan:  make(chan *room.Room, 8),
		cancel:    cancel,
		ctx:       ctx,
		closed:    0,
	}

	// 注册到缓存
	if err := addClient(cli); err != nil {
		g.Log().Errorf(ctx, "add client cache fail, userID:%s, err:%v", userID, err)
		// 缓存落库失败，直接关闭，不拉起协程
		closeConnection(cli, "cache_add_fail")
		return nil
	}

	// 拉起三组协程
	// go cli.readLoop()
	// go cli.writeLoop()
	// go cli.heartbeatLoop()

	g.Log().Infof(ctx, "new client connected, userID:%s, ptr:%p", userID, cli)
	return cli
}

func getPlayers(roomInfo *room.Room) []*PlayerDTO {
	playerDTOs := make([]*PlayerDTO, len(roomInfo.Players))
	for i, p := range roomInfo.Players {
		playerDTOs[i] = &PlayerDTO{
			ID:       p.ID,
			Name:     p.Name,
			RoomID:   p.RoomID,
			Type:     int(p.Type),
			CardNum:  p.CardNum,
			Point:    p.Point,
			UserId:   p.UserId,
			RoomType: roomInfo.Type,
			InPoint:  p.InPoint,
		}
	}
	return playerDTOs
}
