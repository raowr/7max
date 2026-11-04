package enter

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gogf/gf/v2/util/grand"

	v1 "game/api/enter/v1"
	"game/internal/consts"
	"game/internal/controller/log"
	"game/internal/controller/room"
	"game/internal/message"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g" // 修正：GFv2 正确导入路径
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/gorilla/websocket"
)

func (c *ControllerV1) Enter(ctx context.Context, req *v1.EnterReq) (res *v1.EnterRes, err error) {
	var (
		r          = g.RequestFromCtx(ctx)
		ws         *websocket.Conn
		wsUpGrader = websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// 从配置读取 debug 模式（推荐：配置文件中配置 app.debug = true/false）
				// MustGet: 键不存在时 panic；若想避免 panic，用 Get().Bool() 并处理默认值
				//isDebug := g.Cfg().MustGet(ctx, "app.debug").Bool()
				//if isDebug {
				//	return true // 调试模式允许所有跨域
				//}
				//// 2. Android 原生客户端（Origin 为空字符串）
				//origin := r.Header.Get("Origin")
				//allowedOrigins := map[string]bool{
				//	allowedOrigin: true, // 浏览器前端域名
				//	"": true,            // 允许空 Origin（Android 原生）
				//}
				//return allowedOrigins[origin]
				return true
			},
			Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
				g.Log().Errorf(ctx, "WebSocket升级失败: %v", reason)
				w.WriteHeader(status)
				w.Write([]byte(reason.Error()))
			},
		}
	)

	// 升级HTTP连接为WebSocket
	ws, err = wsUpGrader.Upgrade(r.Response.Writer, r.Request, nil)
	if err != nil {
		g.Log().Errorf(ctx, "WebSocket升级失败: %v", err)
		r.Response.WriteHeader(http.StatusInternalServerError)
		r.Response.Write([]byte("连接建立失败"))
		return
	}

	// 获取或生成用户ID
	userID := r.GetQuery("user_id").String()
	if userID == "" {
		userID = room.GenerateUserID()
		g.Log().Infof(ctx, "用户生成新ID: %s", userID)
	}

	client := NewClient(ws, userID, 0)
	g.Log().Infof(ctx, "用户 %s 连接成功", userID)

	// 启动协程（增加退出日志）
	go client.readLoop()
	go client.readServeLoop()
	go client.writeLoop()
	go client.heartbeatLoop()

	// 返回空响应
	return &v1.EnterRes{}, nil
}

// 读消息循环：处理客户端消息
func (c *Client) readLoop() {
	defer func() {
		if err := recover(); err != nil {
			g.Log().Errorf(context.Background(), "readLoop panic, userID:%s, err:%v", c.userID, err)
		}
		g.Log().Infof(context.Background(), "readLoop exit, userID:%s, ptr:%p", c.userID, c)
	}()

	for {
		if atomic.LoadInt32(&c.closed) == 1 {
			return
		}
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		// _ = c.conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		mt, data, err := c.conn.ReadMessage()

		if atomic.LoadInt32(&c.closed) == 1 {
			return
		}

		if err != nil {
			if websocket.IsCloseError(err) {
				return
			}
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			closeConnection(c, "read_error:"+err.Error())
			return
		}

		c.refreshHeartbeat()

		var msg message.ChatMsg
		if err := gconv.Struct(data, &msg); err != nil {
			errMsg := message.ChatMsg{
				Type: consts.ChatTypeError,
				Data: fmt.Sprintf("无效消息格式: %v", err),
				From: c.userID,
			}
			c.safeSendMessage(errMsg)
			continue
		}
		g.Log().Infof(c.ctx, "用户 %s 接收消息: %s（类型: %d）", c.userID, data, mt)
		// 根据消息类型处理业务（所有操作加锁保护全局rm）
		switch msg.Type {
		case consts.InitRoom:
			c.handleInitRoom()
		case consts.CreateRoom:
			c.handleCreateRoom()
		case consts.JoinRoom:
			c.handleJoinRoom(msg.Data)
		case consts.LeaveRoom:
			c.handleLeaveRoom(msg.Data)
		case consts.Play:
			c.handlePlay()
		case consts.PlayCard:
			c.handlePlayCard(msg.Data)
		case consts.GetInfo:
			c.handleGetInfo()
		case consts.Heartbeat:
			c.handleHeartbeat(msg.Data)
		default:
			errMsg := message.ChatMsg{
				Type: consts.ChatTypeError,
				Data: "未知消息类型",
				From: c.userID,
			}
			c.safeSendMessage(errMsg)
		}
	}
}

// 处理房间初始化
func (c *Client) handleInitRoom() {
	rmMu.Lock()
	defer rmMu.Unlock()
	c.clearRoom()
	roomInfo := rm.CreateRoom(1)
	playerName := "美女"
	humanPlayer := roomInfo.CreatePlayer(playerName, room.Human)
	humanPlayer.UserId = c.userID
	c.pid = humanPlayer.ID
	rm.PlayerList[humanPlayer.UserId] = humanPlayer

	log.SendLog(log.LogInfo{
		RoomID:     roomInfo.ID,
		Type:       roomInfo.Type,
		Status:     roomInfo.Status,
		UserID:     humanPlayer.UserId,
		Point:      humanPlayer.Point,
		Action:     consts.InitRoom,
		Remain:     "",
		OutCardIds: "",
		Text:       "",
	})

	aiCount := grand.N(0, 1)
	for i := 0; i < aiCount; i++ {
		aiName := fmt.Sprintf("帅锅%d号", i+1)
		roomInfo.CreatePlayer(aiName, room.AI)
	}

	c.safeSendToRoomChan(roomInfo)

	playerDTOs := getPlayers(roomInfo)

	players, err := json.Marshal(playerDTOs)
	if err != nil {
		g.Log().Errorf(c.ctx, "用户 %s 序列化玩家列表失败: %v", c.userID, err)
		return
	}
	msgData := message.ChatMsg{
		Type: consts.InitRoom,
		Data: gconv.String(players),
		From: c.userID,
	}
	c.safeSendMessage(msgData)
	if aiCount >= 1 {
		return
	}
	go func(roomID string, userID string, clientPid int) {
		localCtx := context.Background()
		g.Log().Infof(localCtx, "用户 %s 开始延迟添加AI到房间 %s", userID, roomID)

		<-time.After(2 * time.Second)
		rmMu.Lock()
		defer rmMu.Unlock()

		roomInfo, ok := rm.Rooms[roomID]
		if !ok {
			g.Log().Warningf(c.ctx, "房间 %s 已不存在，停止添加AI", roomID)
			return
		}
		aiNum := len(roomInfo.Players)
		for i := 0; i < 2-aiNum; i++ {
			aiName := fmt.Sprintf("帅锅%d号", aiNum+i+1)
			roomInfo.CreatePlayer(aiName, room.AI)
		}

		playerDTOs := getPlayers(roomInfo)

		players, err := json.Marshal(playerDTOs)
		if err != nil {
			g.Log().Errorf(c.ctx, "用户 %s 序列化玩家列表失败: %v", c.userID, err)
			return
		}
		msgData := message.ChatMsg{
			Type: consts.InitRoom,
			Data: gconv.String(players),
			From: c.userID,
		}
		c.safeSendMessage(msgData)
	}(roomInfo.ID, c.userID, c.pid)

}

// 处理创建房间
func (c *Client) handleCreateRoom() {
	rmMu.Lock()
	defer rmMu.Unlock()
	c.clearRoom()

	roomInfo := rm.CreateRoom(2)

	playerName := "美女"
	humanPlayer := roomInfo.CreatePlayer(playerName, room.Human)
	humanPlayer.UserId = c.userID
	c.pid = humanPlayer.ID
	rm.PlayerList[humanPlayer.UserId] = humanPlayer

	c.safeSendToRoomChan(roomInfo)

	playerDTOs := getPlayers(roomInfo)

	players, err := json.Marshal(playerDTOs)
	if err != nil {
		g.Log().Errorf(c.ctx, "用户 %s 序列化玩家列表失败: %v", c.userID, err)
		return
	}
	msgData := message.ChatMsg{
		Type: consts.InitRoom,
		Data: gconv.String(players),
		From: c.userID,
	}
	c.safeSendMessage(msgData)

	log.SendLog(log.LogInfo{
		RoomID:     roomInfo.ID,
		Type:       roomInfo.Type,
		Status:     roomInfo.Status,
		UserID:     humanPlayer.UserId,
		Point:      humanPlayer.Point,
		Action:     consts.InitRoom,
		Remain:     "",
		OutCardIds: "",
		Text:       "",
	})

}

// 处理加入房间
func (c *Client) handleJoinRoom(data string) {
	rmMu.Lock()
	defer rmMu.Unlock()
	// 解析加入房间数据（严格错误处理）
	var reqData struct {
		RoomID string `json:"roomID"`
	}
	if err := json.Unmarshal([]byte(data), &reqData); err != nil {
		g.Log().Errorf(c.ctx, "用户 %s 解析加入房间数据失败: %v", c.userID, err)
		return
	}
	roomID := reqData.RoomID
	if roomID == "" {
		g.Log().Errorf(c.ctx, "用户 %s 加入房间ID为空", c.userID)
		return
	}
	roomInfo, ok := rm.Rooms[roomID]
	if !ok {
		g.Log().Errorf(c.ctx, "用户 %s 房间 %s 不存在", c.userID, roomID)
		return
	}
	// 检查房间是否已满
	if len(roomInfo.Players) >= 2 {
		g.Log().Errorf(c.ctx, "用户 %s 房间 %s 已满", c.userID, roomID)
		return
	}
	// 检查房间是否正在游戏中
	if roomInfo.IsPlaying {
		g.Log().Errorf(c.ctx, "用户 %s 房间 %s 正在游戏中", c.userID, roomID)
		return
	}
	// 检查房间是否已开始游戏
	if roomInfo.Status == 1 {
		g.Log().Errorf(c.ctx, "用户 %s 房间 %s 已开始游戏", c.userID, roomID)
		return
	}
	//判断玩家已在房间
	inRoom := false
	for _, player := range roomInfo.Players {
		if player.UserId == c.userID {
			g.Log().Errorf(c.ctx, "用户 %s 已在房间 %s", c.userID, roomID)
			inRoom = true
			break
		}
	}
	//不在房间再加入房间，在房间直接返回房间数据
	var humanPlayer *room.Player
	if !inRoom {
		playerName := "好友"
		humanPlayer = roomInfo.CreatePlayer(playerName, room.Human)
		humanPlayer.UserId = c.userID
		c.pid = humanPlayer.ID
		humanPlayer.Name = playerName + gconv.String(humanPlayer.ID)
		rm.PlayerList[humanPlayer.UserId] = humanPlayer // 关联用户与玩家
		// rm.JoinRoom(humanPlayer, roomID)
	}

	c.safeSendToRoomChan(roomInfo)

	// 推送玩家列表（处理JSON错误）
	// 创建一个临时结构体，只包含可序列化的字段
	playerDTOs := getPlayers(roomInfo)

	//players, err := json.Marshal(playerDTOs)
	//if err != nil {
	//	g.Log().Errorf(ctx, "用户 %s 序列化玩家列表失败: %v", c.userID, err)
	//	return
	//}
	//msgData := message.ChatMsg{
	//	Type: consts.JoinRoom,
	//	Data: gconv.String(players),
	//	From: gconv.String(humanPlayer.ID),
	//}
	//应该通知到房间内每个人
	roomInfo.SendRoomMessage(consts.JoinRoom, playerDTOs)
	//c.safeSendMessage(ctx, msgData)

	//发送日志 加入房间
	log.SendLog(log.LogInfo{
		RoomID:     roomInfo.ID,
		Type:       roomInfo.Type,
		Status:     roomInfo.Status,
		UserID:     c.userID,
		Point:      humanPlayer.Point, //积分
		Action:     consts.JoinRoom,   //行为
		Remain:     "",                //剩余牌
		OutCardIds: "",                //玩家单次打出的牌id
		Text:       "",                //完整信息
	})
}

// 离开房间
func (c *Client) handleLeaveRoom(data string) {
	rmMu.Lock()
	defer rmMu.Unlock()
	//移除房间内该玩家
	player, ok := rm.PlayerList[c.userID]
	if !ok {
		g.Log().Errorf(c.ctx, "用户 %s 未找到玩家信息", c.userID)
		return
	}
	isHomeowner := false
	roomInfo, ok := rm.Rooms[player.RoomID]
	if !ok {
		g.Log().Errorf(c.ctx, "用户 %s 未找到房间 %s 信息", c.userID, player.RoomID)
		return
	}
	players := make([]*room.Player, 0) //临时玩家数组
	for _, player := range roomInfo.Players {
		if player.UserId != c.userID {
			players = append(players, player)
		}
		//如果是房主离开,重新牌ID
		if player.UserId == c.userID {
			if player.ID == 0 {
				isHomeowner = true
			}
		}

	}
	roomInfo.Players = players
	roomInfo.NextPlayerID = 0
	if isHomeowner {
		if len(roomInfo.Players) > 0 {
			for _, player := range roomInfo.Players {
				player.ID = roomInfo.NextPlayerID
				roomInfo.NextPlayerID++
			}
		}
	} else {
		if len(roomInfo.Players) > 0 {
			roomInfo.NextPlayerID = 1
			for _, player := range roomInfo.Players {
				if player.ID == 0 {
					continue
				} else {
					player.ID = roomInfo.NextPlayerID
					roomInfo.NextPlayerID++
				}
			}
		}
	}

	if player.MsgChan != nil {
		c.mutex.Lock()
		ch := player.MsgChan
		player.MsgChan = nil
		c.mutex.Unlock()
		close(ch)
		g.Log().Infof(context.Background(), "用户 %s 离开房间 %s 后 MsgChan 关闭", c.userID, roomInfo.ID)
	}

	//移除rm中用户列表
	delete(rm.PlayerList, c.userID)

	// 创建一个临时结构体，只包含可序列化的字段
	playerDTOs := getPlayers(roomInfo)

	//应该通知到房间内每个人
	roomInfo.SendRoomMessage(consts.JoinRoom, playerDTOs)

	//发送日志 离开房间
	log.SendLog(log.LogInfo{
		RoomID:     roomInfo.ID,
		Type:       roomInfo.Type,
		Status:     roomInfo.Status,
		UserID:     c.userID,
		Point:      player.Point,     //积分
		Action:     consts.LeaveRoom, //行为
		Remain:     "",               //剩余牌
		OutCardIds: "",               //玩家单次打出的牌id
		Text:       "",               //完整信息
	})

}

// 处理开始游戏
func (c *Client) handlePlay() {
	rmMu.RLock()
	player, ok := rm.PlayerList[c.userID]
	if !ok {
		rmMu.RUnlock()
		g.Log().Errorf(c.ctx, "用户 %s 未找到玩家信息", c.userID)
		return
	}
	roomInfo, ok := rm.Rooms[player.RoomID]
	rmMu.RUnlock()

	if !ok {
		g.Log().Errorf(c.ctx, "用户 %s 房间不存在", c.userID)
		return
	}

	// 启动游戏逻辑（传入上下文，支持取消）
	//判断人数是否足够
	if len(roomInfo.Players) < 2 {
		g.Log().Errorf(c.ctx, "用户 %s 房间人数不足", c.userID)
		return
	}

	//判断游戏进行中不能再次play,避免同一房间多次play
	if roomInfo.IsPlaying || roomInfo.Status == 1 {
		g.Log().Errorf(c.ctx, "用户 %s 房间 %s 进行中不能再次开始比赛", c.userID, roomInfo.ID)
		return
	}

	rmMu.Lock()
	roomInfo.IsPlaying = true
	roomInfo.Status = 1 //游戏中
	rmMu.Unlock()
	//通知房间内每个人开始游戏
	roomInfo.SendRoomMessage(consts.Play, "")

	room.PlayOneGame(roomInfo)

	//发送日志 开始游戏
	log.SendLog(log.LogInfo{
		RoomID:     roomInfo.ID,
		Type:       roomInfo.Type,
		Status:     roomInfo.Status,
		UserID:     c.userID,
		Point:      player.Point, //积分
		Action:     consts.Play,  //行为
		Remain:     "",           //剩余牌
		OutCardIds: "",           //玩家单次打出的牌id
		Text:       "",           //完整信息
	})
}

// 处理出牌
func (c *Client) handlePlayCard(data string) {
	rmMu.RLock()
	player, ok := rm.PlayerList[c.userID]
	rmMu.RUnlock()
	if !ok {
		g.Log().Errorf(c.ctx, "用户 %s 未找到玩家信息", c.userID)
		return
	}
	rmMu.RLock()
	roomInfo := rm.Rooms[player.RoomID]
	rmMu.RUnlock()
	// 解析出牌数据（严格错误处理）
	var reqData struct {
		Pid     int   `json:"pid"`
		CardIds []int `json:"cardIds"`
		Pass    int   `json:"pass"`
	}
	if err := json.Unmarshal([]byte(data), &reqData); err != nil {
		g.Log().Errorf(c.ctx, "用户 %s 解析出牌数据失败: %v", c.userID, err)
		return
	}
	if reqData.Pid != c.pid {
		g.Log().Errorf(c.ctx, "用户 %s PID不匹配（%d vs %d）", c.userID, reqData.Pid, c.pid)
		return
	}

	//如果不是人类出牌时间，返回
	if player.ID != roomInfo.Current {
		g.Log().Errorf(c.ctx, "用户 %s 不是该玩家出牌阶段,当前出牌玩家为:%d", c.userID, roomInfo.Current)
		return
	}

	// 更新玩家出牌信息
	rmMu.Lock()
	player.OutCardIds = reqData.CardIds
	player.Pass = reqData.Pass
	rmMu.Unlock()
}

// 处理获取房间信息
func (c *Client) handleGetInfo() {
	rmMu.Lock()
	defer rmMu.Unlock()
	// 1. 基础数据访问（假设是安全的）
	player, ok := rm.PlayerList[c.userID]
	if !ok {
		g.Log().Errorf(c.ctx, "用户 %s 未找到玩家信息", c.userID)
		return
	}

	roomInfo, ok := rm.Rooms[player.RoomID]
	if !ok {
		g.Log().Errorf(c.ctx, "用户 %s 房间不存在", c.userID)
		return
	}

	// 2. 如果涉及到可能被其他goroutine修改的数据，才加锁
	var current, outStarTime int
	var isPlaying bool
	var status int
	var roomType int

	// 快速获取易变数据
	current = roomInfo.Current
	outStarTime = roomInfo.OutStarTime
	isPlaying = roomInfo.IsPlaying
	status = roomInfo.Status
	roomType = roomInfo.Type

	// 3. 在锁外进行其他操作
	if isPlaying {
		c.safeSendToRoomChan(roomInfo)
	}

	// 4. 玩家卡片数据（通常比较稳定，可以不加锁）
	cards := make([]int, 0)
	cardsNum := make([]*message.PlayData, 0)
	var playerPoint int64
	var mustPid int

	for _, p := range roomInfo.Players {
		if p.ID == player.ID {
			playerPoint = p.Point
			for _, card := range p.Cards {
				cards = append(cards, card.Id)
			}
		} else {
			cardsNum = append(cardsNum, &message.PlayData{
				Id:      p.ID,
				CardNum: p.CardNum,
			})
		}
		if p.Must {
			mustPid = p.ID
		}
	}

	// 5. 计算剩余信息，在锁外进行
	outCardTimeout := room.GetOutCardTimeout()
	remainOutCardTimeout := outCardTimeout
	if outStarTime > 0 {
		now := int(time.Now().Unix())
		remainOutCardTimeout = outCardTimeout - (now - outStarTime)
		if remainOutCardTimeout < 0 {
			remainOutCardTimeout = 0
		}
	}

	// 本轮已出的牌堆
	outCards := make([]int, 0)
	for _, card := range roomInfo.TurnDeck {
		outCards = append(outCards, card.Id)
	}

	// 最后的一手牌（用于前端比较大小）
	lastOutCards := make([]int, 0)
	for _, card := range roomInfo.LastCards {
		lastOutCards = append(lastOutCards, card.Id)
	}

	// 上一位出牌玩家
	lastPid := (current - 1 + 2) % 2

	// 创建一个临时结构体，只包含可序列化的字段
	playerDTOs := getPlayers(roomInfo)

	// 6. 序列化并推送（处理JSON错误）
	resData, err := json.Marshal(struct {
		RoomId               string              `json:"roomId"`
		Players              []*PlayerDTO        `json:"players"`
		Cards                []int               `json:"cards"`
		Current              int                 `json:"current"`
		PlayerPoint          int64               `json:"playerPoint"`
		OutCardTimeout       int                 `json:"outCardTimeout"`
		RemainOutCardTimeout int                 `json:"remainOutCardTimeout"`
		CardsNum             []*message.PlayData `json:"cardsNum"`
		IsPlaying            bool                `json:"isPlaying"`
		OutCards             []int               `json:"outCards"`
		LastOutCards         []int               `json:"lastOutCards"` //最后的一手牌
		MustPid              int                 `json:"mustPid"`
		LastPid              int                 `json:"lastPid"`
		Status               int                 `json:"status"`
		Type                 int                 `json:"type"`        //房间类型 1比赛房，2好友房
		AllCardsNum          int                 `json:"allCardsNum"` //剩余未摸的牌数
	}{
		RoomId:               roomInfo.ID,
		Players:              playerDTOs,
		Cards:                cards,
		Current:              current,
		PlayerPoint:          playerPoint,
		OutCardTimeout:       outCardTimeout,
		RemainOutCardTimeout: remainOutCardTimeout,
		CardsNum:             cardsNum,
		IsPlaying:            isPlaying,
		OutCards:             outCards,
		LastOutCards:         lastOutCards,
		MustPid:              mustPid,
		LastPid:              lastPid,
		Status:               status,
		Type:                 roomType,
		AllCardsNum:          len(roomInfo.Deck),
	})
	if err != nil {
		g.Log().Errorf(c.ctx, "用户 %s 序列化房间信息失败: %v", c.userID, err)
		return
	}

	// 7. 最后在锁外发送消息
	msgData := message.ChatMsg{
		Type: consts.GetInfo,
		Data: gconv.String(resData),
		From: c.userID,
	}
	c.safeSendMessage(msgData)
}

// 处理心跳
func (c *Client) handleHeartbeat(data string) {
	if data == "ping" {
		// 加读锁保护对sendChan的访问
		c.mutex.RLock()
		sendChan := c.sendChan
		c.mutex.RUnlock()
		c.heartbeat = time.Now()
		// 非阻塞发送pong，避免通道满导致阻塞
		if sendChan != nil {
			select {
			case sendChan <- []byte(`{"type":"heartbeat","data":"pong"}`):
			default:
				g.Log().Warningf(c.ctx, "用户 %s 心跳响应发送阻塞（通道满）", c.userID)
			}
		}

	}
}

// 读取服务端消息并推送给客户端（修复空指针和资源泄漏）
func (c *Client) readServeLoop() {
	defer g.Log().Infof(context.Background(), "用户 %s readServeLoop 退出", c.userID)
	defer func() {
		if err := recover(); err != nil {
			g.Log().Errorf(context.Background(), "readServeLoop panic, userID:%s, err:%v", c.userID, err)
		}
		g.Log().Infof(context.Background(), "readServeLoop exit, userID:%s, ptr:%p", c.userID, c)
	}()
	// 循环读取玩家消息（带退出机制）
	var roomInfo *room.Room
	// 使用局部变量存储当前roomChan的引用
	var currentRoomChan chan *room.Room
	c.mutex.RLock()
	currentRoomChan = c.roomChan
	c.mutex.RUnlock()
	for {
		select {
		case <-c.ctx.Done():
			return

		case newRoom, ok := <-currentRoomChan:
			if !ok {
				return
			}
			if newRoom == nil {
				continue
			}
			c.mutex.Lock()
			roomInfo = newRoom
			c.mutex.Unlock()

			var currentPlayer *room.Player
			for _, p := range roomInfo.Players {
				if p.UserId == c.userID {
					currentPlayer = p
					break
				}
			}

			if currentPlayer != nil && currentPlayer.MsgChan != nil {
				go func(player *room.Player) {
					for {
						select {
						case playerMsg, ok := <-player.MsgChan:
							if !ok {
								return
							}
							msgData := message.ChatMsg{
								Type: playerMsg.Type,
								Data: playerMsg.Data,
								From: c.userID,
							}
							g.Log().Infof(c.ctx, "玩家 %d 向用户 %s 发送消息: %+v", player.ID, c.userID, msgData)
							c.safeSendMessage(msgData)
						case <-c.ctx.Done():
							return
						}
					}
				}(currentPlayer)
			}
		}
	}
}

// 写消息循环：发送消息到客户端
func (c *Client) writeLoop() {
	defer func() {
		if err := recover(); err != nil {
			g.Log().Errorf(context.Background(), "writeLoop panic, userID:%s, err:%v", c.userID, err)
		}
		g.Log().Infof(context.Background(), "writeLoop exit, userID:%s, ptr:%p", c.userID, c)
	}()
	for {
		// 检查连接是否已关闭
		if atomic.LoadInt32(&c.closed) == 1 {
			g.Log().Debugf(c.ctx, "用户 %s 连接已关闭，写循环退出", c.userID)
			return
		}
		var roomID string
		player, ok := rm.PlayerList[c.userID]
		if ok && player.RoomID != "" {
			roomID = player.RoomID
		}

		// 在 writeLoop 函数中修改为
		c.mutex.RLock()
		sendChan := c.sendChan
		c.mutex.RUnlock()
		if sendChan != nil {
			select {
			case <-c.ctx.Done():
				return
			case data, ok := <-sendChan:
				if !ok {
					g.Log().Infof(c.ctx, "用户 %s sendChan已关闭，writeLoop退出", c.userID)
					return
				}

				g.Log().Infof(c.ctx, " writeLoop 房间 %s 向用户 %s 发送消息: %s", roomID, c.userID, string(data))

				// 使用互斥锁保护并发访问
				c.mutex.Lock()
				conn := c.conn
				c.mutex.Unlock()

				if conn == nil {
					g.Log().Errorf(c.ctx, " writeLoop 房间 %s 用户 %s 连接已关闭，无法发送消息", roomID, c.userID)
					return
				}

				if err := conn.WriteMessage(ghttp.WsMsgText, data); err != nil {
					g.Log().Errorf(c.ctx, " writeLoop 房间 %s 用户 %s 消息发送失败: %v,消息内容: %s", roomID, c.userID, err, string(data))
					closeConnection(c, "write_error:"+err.Error())
					return
				}
			}
		}
	}
}

// 心跳检测：超时断开连接
// heartbeatLoop 心跳检测协程
func (c *Client) heartbeatLoop() {
	defer func() {
		if err := recover(); err != nil {
			g.Log().Errorf(context.Background(), "heartbeatLoop panic, userID:%s, err:%v", c.userID, err)
		}
		g.Log().Infof(context.Background(), "heartbeatLoop exit, userID:%s, ptr:%p", c.userID, c)
	}()

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if time.Since(c.heartbeat) > heartbeatTimeout {
				// 心跳超时，统一走关闭入口
				closeConnection(c, "heartbeat_timeout")
				return
			}
		}
	}
}

// 更新心跳，业务收到客户端心跳包调用这个
func (c *Client) refreshHeartbeat() {
	c.mutex.Lock()
	c.heartbeat = time.Now()
	c.mutex.Unlock()
}

// 安全发送消息的函数
func (c *Client) safeSendMessage(msg message.ChatMsg) {
	c.mutex.RLock()
	sendChan := c.sendChan
	c.mutex.RUnlock()

	if sendChan != nil {
		// 使用匿名函数包装发送操作并添加 recover 机制
		func() {
			defer func() {
				if r := recover(); r != nil {
					g.Log().Errorf(c.ctx, "用户 %s 消息发送发生panic: %v", c.userID, r)
				}
			}()

			select {
			case sendChan <- c.encodeMessage(msg):
				// 发送成功
			default:
				g.Log().Warningf(c.ctx, "用户 %s 消息发送阻塞（通道满）", c.userID)
			}
		}()
	}
}

// 2. 安全地向 roomChan 发送消息的方法
func (c *Client) safeSendToRoomChan(roomInfo *room.Room) bool {
	// 加读锁保护对 roomChan 的访问
	c.mutex.RLock()
	roomChan := c.roomChan
	c.mutex.RUnlock()

	// 检查通道是否存在
	if roomChan == nil {
		g.Log().Warningf(c.ctx, "用户 %s 尝试发送消息到未初始化的 roomChan", c.userID)
		return false
	}

	// 使用匿名函数包装发送操作并添加 recover 机制
	result := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				g.Log().Errorf(c.ctx, "用户 %s 向 roomChan 发送消息发生panic: %v", c.userID, r)
				result = false
			}
		}()

		// 非阻塞发送或阻塞发送取决于业务需求
		select {
		case roomChan <- roomInfo:
			result = true
		case <-c.ctx.Done():
			g.Log().Warningf(c.ctx, "用户 %s 向 roomChan 发送消息超时", c.userID)
			result = false
		default:
			g.Log().Warningf(c.ctx, "用户 %s 向 roomChan 发送消息阻塞", c.userID)
			result = false
		}
	}()

	return result
}

// 消息编码（处理错误）
func (c *Client) encodeMessage(msg message.ChatMsg) []byte {
	msgBytes, err := gjson.Encode(msg)
	if err != nil {
		g.Log().Errorf(c.ctx, "用户 %s 消息编码失败: %v", c.userID, err)
		return []byte(`{"type":"error","data":"消息编码失败"}`)
	}
	return msgBytes
}

// 清理旧房间
func (c *Client) clearRoom() {
	// 清理当前用户关联的旧房间（优化：定向清理，避免全量遍历）
	var oldRoomID string
	for _, player := range rm.PlayerList {
		if player.UserId == c.userID {
			oldRoomID = player.RoomID
			break
		}
	}
	if oldRoomID != "" {
		if roomInfo, ok := rm.Rooms[oldRoomID]; ok {
			if roomInfo.Rgtimer != nil {
				roomInfo.Rgtimer.Stop()
				roomInfo.Rgtimer.Close() // 停止定时器（确保资源释放）
			}
			delete(rm.Rooms, oldRoomID)
			g.Log().Infof(c.ctx, "用户 %s 清理旧房间: %s", c.userID, oldRoomID)
		}
		//关闭用户消息通道
		player, ok := rm.PlayerList[c.userID]
		if !ok {
			g.Log().Errorf(c.ctx, "用户 %s 未找到玩家信息", c.userID)
			return
		}
		if player.MsgChan != nil {
			c.mutex.Lock()
			ch := player.MsgChan
			player.MsgChan = nil
			c.mutex.Unlock()
			close(ch)
			g.Log().Infof(context.Background(), "用户 %s MsgChan 已关闭", c.userID)
		}
		// 从玩家列表移除旧玩家
		delete(rm.PlayerList, c.userID)
	}
}

/*
{"type":"initRoom","data":"","name":""}
*/
