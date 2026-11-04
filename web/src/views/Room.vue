<template>
  <div id="room">
    <div class="background-container">
      <img src="@/assets/img/bg/indoor.png" alt="室内场景背景图">
    </div>

    <div class="room-container">
      <div class="header">
        <div class="back-btn" @click="leaveRoom">
          <img src="@/assets/img/ui/img_return1_bg.png" alt="返回按钮背景">
          <img src="@/assets/img/ui/img_return1.png" alt="返回" style="position: absolute; left: 0; top: 0;height: 100%;">
          <img :src="getRoomTitleImage()" alt="房间标题" class="room-title">
        </div>
      </div>

      <div class="main-content">
        <div class="controls-section">
          <div class="game-info">
            <div class="info-item">
              <div class="info-label">游戏模式</div>
              <div class="info-value">{{ state.gameType }}</div>
            </div>
            <div class="info-item">
              <div class="info-label">奖励</div>
              <div class="info-value">{{ state.gameReward }}</div>
            </div>
            <div class="info-item">
              <div class="info-label">房间号</div>
              <div class="info-value">{{ state.roomId }}</div>
            </div>
          </div>
        </div>

        <div class="players-section">
          <div class="player-card" v-for="(player, index) in state.players" :key="index">
            <img v-if="player.isOwner" src="@/assets/img/ui/roomowner.png" class="room-owner-badge" alt="房主标志">

            <div class="player-frame">
              <div class="player-bg-container">
                <img src="@/assets/img/ui/frame_gold.png" alt="玩家框" style="width: 100%; height: 100%;">
                <img src="@/assets/img/ui/character_bg.png" alt="玩家背景"
                  style="position: absolute; top: 0; left: 0; width: 100%; height: 100%;">
                <img v-if="player.lihui !== ''" :src="player.lihui" alt="玩家lihui"
                  style="position: absolute; top: 0; left: 0; width: 100%; height: 100%;">
              </div>

              <div class="player-content">
                <div class="player-inner">
                  <div class="player-name-container">
                    <img src="@/assets/img/ui/namebg.png" class="player-nameplate" alt="名称背景">
                    <div class="player-name">{{ player.name }}</div>
                  </div>

                  <div class="player-stars">
                    <img v-for="n in 3" :key="n"
                      :src="n <= player.stars ? starImg : starDarkImg" 
                      alt="星星">
                  </div>

                  <img src="@/assets/img/ui/ready.png" class="player-ready" :class="{ active: player.ready }"
                    alt="准备状态">
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="ready-btn-container" v-if="playerId == 0">
          <img v-if=!isReady() src="@/assets/img/ui/btn_start_no.png" class="ready-btn" alt="准备中" >
          <img v-else src="@/assets/img/ui/btn_start.png" class="ready-btn" alt="已准备" @click="toGame">
        </div>
      </div>

      <div class="connection-status" :class="state.connectionStatusClass">
        {{ state.connectionStatusText }}
      </div>
    </div>
  </div>
</template>
<script setup>
import { ref,reactive, onMounted, computed, onBeforeUnmount,onUnmounted } from 'vue'
import { websocket } from '@/utils/websocket'
import { audioManager } from '@/utils/audio'
import starImg from '@/assets/img/ui/star.png';
import starDarkImg from '@/assets/img/ui/star_dark.png';
import bighead15339 from '@/assets/img/touxiang/bighead15339.png';
import full16020 from '@/assets/img/lihui/full16020.png';
import bighead15419 from '@/assets/img/touxiang/bighead15419.png';
import full15418 from '@/assets/img/lihui/full15418.png';
import { storage } from '@/utils/storage'
import { getTouxiang } from '@/utils/touxiang'
import {useRoute,useRouter} from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus';

import { reconnectWebSocket } from '@/utils/websocketReconnect';

const route = useRoute()
const router = useRouter()
const getInitialPlayers = () => [
  {
    id: 1,
    name: '玩家一',
    avatar: bighead15339,
    stars: 3,
    ready: false,
    isOwner: true,
    lihui: full16020,
  },
  {
    id: 2,
    name: '等待中...',
    avatar: "",
    stars: 2,
    ready: false,
    isOwner: false,
    lihui: '',
  },
]
const state = reactive({
  players: getInitialPlayers(),
  gameType: '经典模式',
  gameReward: '100金币',
  roomId: '123456',

  ws: null,
  isConnected: false,
  isConnecting: false,
  reconnectAttempts: 0,
  maxReconnectAttempts: 5,

  connectionStatusClass: computed(() => ({
    connected: websocket.ws?.readyState === WebSocket.OPEN,
    disconnected: websocket.ws?.readyState === WebSocket.CLOSED
  })),
  connectionStatusText: computed(() => {
    switch (websocket.ws?.readyState) {
      case WebSocket.OPEN: return '已连接'
      case WebSocket.CONNECTING: return '连接中...'
      default: return '已断开'
    }
  })
})
const userId = ref('')
const playerId  = ref(0)
const roomType = ref(0)
onMounted(() => {
  init()
      if (!websocket.ws || 
        websocket.ws.readyState === WebSocket.CLOSED || 
        websocket.ws.readyState === WebSocket.CLOSING) {
        reconnectWebSocket(websocket);
    }
  
})
const onRoomOpen = () => {
  console.log('room on open');
  websocket.send({ "type": "getInfo", "data": "", "name": "" })
}
const onRoomError = () => {
  state.players.forEach(player => {
    player.ready = false
  })
  console.log('on error');
}
const onRoomClose = () => {
  console.log('on close');
}

const init = async () => {
  const bgmUrl = new URL('@/assets/music/game_bg.mp3', import.meta.url).href;
  audioManager.preload('bgm', bgmUrl);
  audioManager.playBGM('bgm');

  websocket.on('message', handleMessage);
  websocket.on('open', onRoomOpen);
  websocket.on('error', onRoomError);
  websocket.on('close', onRoomClose);

  if (websocket.ws && websocket.ws.readyState === WebSocket.OPEN) {
    console.log('getInfo room on open');
  }

  storage.local.set('player2avatar', getTouxiang())
}


const handleMessage = (data) => {
  console.log('room Received message:', data)
  const parsedData = JSON.parse(data)
  if (parsedData.from != undefined) {
    storage.local.set('user_id', parsedData.from)
    userId.value = parsedData.from
  }

  if (parsedData.type === "initRoom" || parsedData.type === "createRoom") {
    try {
        const serverPlayers = JSON.parse(parsedData.data)

        state.roomId=serverPlayers[0].RoomID
        roomType.value = serverPlayers[0].RoomType

        initPlayers(serverPlayers)

        state.players.splice(4)
      } catch (e) {
        console.error('解析玩家数据失败:', e)
      }
  }
  if(parsedData.type === "getInfo"){
      const data = JSON.parse(parsedData.data)
      state.roomId=data.roomId
      roomType.value = data.type
     if (data.isPlaying && route.name !== "Game") {
       storage.local.set('roomInfo', JSON.stringify(data))
        router.push('/game')
     }else {
         initPlayers(data.players)
     }
  }

    if (parsedData.type === "joinRoom" || parsedData.type === "leaveRoom") {
    try {
        const serverPlayers = JSON.parse(parsedData.data)

        state.roomId=serverPlayers[0].RoomID
        roomType.value = serverPlayers[0].RoomType
        initPlayers(serverPlayers)

        state.players.splice(4)
      } catch (e) {
        console.error('解析玩家数据失败:', e)
      }
  }
  if (parsedData.type === "play") {
    // 不在游戏页 操作跳到游戏页
    if (route.name !== "Game") {
      router.push('/game')  // 跳转到游戏页面
      // router.push({path:'/game'})
    }
  }
  if (parsedData.type == "healthTip") {
    data = JSON.parse(parsedData.data)
    let msg  = '游戏一定很精彩，但是今日你已在线'+data.hour+'小时,请注意休息！！！'
    ElMessage.warning(msg);
  }

}
const initPlayers = (serverPlayers) => {
  state.players = getInitialPlayers()

  serverPlayers.forEach(serverPlayer => {
    if (serverPlayer.UserId == userId.value) {
      playerId.value = serverPlayer.ID
      state.gameReward=serverPlayer.Point+'金币'
    }
    const convertedId = serverPlayer.ID + 1

    if (convertedId <= state.players.length) {
      const player = state.players[convertedId - 1]
      player.name = serverPlayer.Name
      player.isOwner = serverPlayer.ID == 0
      player.avatar = getAvatarByType(player.isOwner ? serverPlayer.Type : 1)
      player.lihui = getLihuiByType(player.isOwner ? serverPlayer.Type : 1)
      player.ready = true
      player.stars = 5
    }
  })
}
const getAvatarByType = (type) => {
  const avatars = [
    new URL('@/assets/img/touxiang/bighead15339.png', import.meta.url).href,
    new URL('@/assets/img/touxiang/bighead15419.png', import.meta.url).href
  ]
  return avatars[type] || new URL('@/assets/img/touxiang/bighead15729.png', import.meta.url).href
}

const getLihuiByType = (type) => {
  const lihuis = [
    new URL('@/assets/img/lihui/full16020.png', import.meta.url).href,
    new URL('@/assets/img/lihui/full15418.png', import.meta.url).href
  ]
  return lihuis[type] || new URL('@/assets/img/lihui/full15703.png', import.meta.url).href
}
onBeforeUnmount(() => {
  websocket.off('message', handleMessage)
  websocket.off('open', onRoomOpen)
  websocket.off('error', onRoomError)
  websocket.off('close', onRoomClose)
})

const isReady=() => {
  let ready = true
  state.players.forEach(serverPlayer => {
    if (!serverPlayer.ready) {
      ready = false
    }
  })
  return ready
}


const toGame = () => {
  if (isReady()) {
    websocket.send({ "type": "play", "data": "", "name": "" })
  }
}

const leaveRoom=()=>{
  websocket.send({ "type": "leaveRoom", "data": "", "name": "" })
  router.push({path:'/index'})
}

const getRoomTitleImage = () => {
  switch (roomType.value) {
    case 1:
      return new URL('@/assets/img/ui/duanweifang.png', import.meta.url).href;
    case 2:
      return new URL('@/assets/img/ui/txt_friendroom.png', import.meta.url).href;
    default:
      return new URL('@/assets/img/ui/duanweifang.png', import.meta.url).href;
  }
}
</script>

<style scoped>
* {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

#room {
  position: relative;
  height: 100vh;
  overflow: hidden;
  color: #ffffff;
  font-family: 'Microsoft YaHei', sans-serif;
}

.background-container {
  position: fixed;
  width: 100vw;
  height: 100vh;
  z-index: 0;
  left: 0;
  top: 0;
}

.background-container img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.room-container {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 12px;
  gap: 12px;
}

.header {
  display: flex;
  justify-content: flex-start;
  align-items: center;
  flex-shrink: 0;
}

.back-btn {
  display: flex;
  align-items: center;
  cursor: pointer;
  position: relative;
  transition: transform 0.2s;
}

.back-btn:active {
  transform: scale(0.95);
}

.back-btn img:first-child {
  height: 44px;
}

.room-title {
  position: absolute;
  left: 44px;
  top: 50%;
  transform: translateY(-50%);
  height: 28px;
}

.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
  overflow-y: auto;
}

.controls-section {
  flex-shrink: 0;
}

.game-info {
  background: rgba(0, 0, 0, 0.5);
  border-radius: 10px;
  padding: 10px 14px;
  display: flex;
  justify-content: space-around;
  gap: 10px;
  backdrop-filter: blur(5px);
}

.info-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  flex: 1;
}

.info-label {
  font-size: 12px;
  color: #ffae4c;
  margin-bottom: 2px;
}

.info-value {
  font-size: 14px;
  font-weight: bold;
}

.players-section {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  align-content: start;
  margin-top: 20px;
}

.player-card {
  position: relative;
  animation: slideIn 0.5s ease-out;
}

.player-frame {
  position: relative;
  width: 100%;
  padding-top: 130%;
}

.player-bg-container {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
}

.player-content {
  position: absolute;
  bottom: 0;
  left: 0;
  width: 100%;
  height: 50%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: flex-end;
  padding-bottom: 8%;
  z-index: 5;
}

.player-inner {
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.player-name-container {
  position: relative;
  width: 85%;
  margin-bottom: 2%;
  display: flex;
  justify-content: center;
  align-items: center;
}

.player-nameplate {
  width: 100%;
  z-index: 2;
}

.player-name {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 85%;
  text-align: center;
  font-size: 12px;
  z-index: 3;
  text-shadow: 1px 1px 2px rgba(0, 0, 0, 0.8);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: bold;
  color: #fff;
}

.player-stars {
  display: flex;
  justify-content: center;
  z-index: 3;
}

.player-stars img {
  height: 14px;
  margin: 0 2px;
}

.player-ready {
  position: absolute;
  top: 20%;
  left: 50%;
  transform: translate(-50%, -50%);
  z-index: 4;
  width: 45%;
  display: none;
}

.player-ready.active {
  display: block;
  animation: pulse 1.5s infinite;
}

.room-owner-badge {
  position: absolute;
  top: -8px;
  left: -8px;
  width: 36%;
  z-index: 10;
}

.ready-btn-container {
  flex-shrink: 0;
  display: flex;
  justify-content: center;
  padding: 4px 0;
}

.ready-btn {
  height: 56px;
  cursor: pointer;
  transition: transform 0.2s;
}

.ready-btn:active {
  transform: scale(0.95);
}

.connection-status {
  position: fixed;
  bottom: 8px;
  right: 8px;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 11px;
  z-index: 100;
}

.connected {
  background-color: rgba(76, 175, 80, 0.8);
}

.disconnected {
  background-color: rgba(244, 67, 54, 0.8);
}

.connecting {
  background-color: rgba(255, 193, 7, 0.8);
}

@keyframes pulse {
  0% {
    opacity: 0.6;
    transform: translate(-50%, -50%) scale(0.95);
  }
  50% {
    opacity: 1;
    transform: translate(-50%, -50%) scale(1.05);
  }
  100% {
    opacity: 0.6;
    transform: translate(-50%, -50%) scale(0.95);
  }
}

@keyframes slideIn {
  from {
    transform: translateY(20px);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}

.player-card:nth-child(2) { animation-delay: 0.1s; }
.player-card:nth-child(3) { animation-delay: 0.2s; }
.player-card:nth-child(4) { animation-delay: 0.3s; }
</style>