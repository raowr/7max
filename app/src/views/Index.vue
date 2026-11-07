<template>
  <div class="index">
    <div v-show="userIdShow" class="msg">{{ storage.local.get('user_id') }}</div>

    <header class="top-bar">
      <div class="character-info" @click="info()">
        <div class="avatar"></div>
        <div class="nickname">
          <p>{{ userName }}</p>
        </div>
      </div>
      <div class="currency-container">
        <div class="currency-item">
          <div class="gold-icon"></div>
          <span class="gold-amount">{{ point }}</span>
        </div>
        <div class="currency-item">
          <div class="diamond-icon"></div>
          <span class="diamond-amount">9999</span>
        </div>
      </div>
      <div class="right-settings">
        <div class="setting-btn btn-set" @click="toggleLogoutModal()"></div>
        <div class="setting-btn btn-guide" @click="toggleGuideModal()"></div>
        <div class="setting-btn btn-trophy" @click="toggleSettingImage()"></div>
      </div>
    </header>

    <div class="main-area">
      <div class="character-art"></div>

      <div class="menu-buttons" v-if="!showFriendRoom">
        <div class="menu-btn right1" @click="toRoom()"></div>
        <div class="menu-btn right2" @click="toggleFriendRoom()"></div>
        <div class="menu-btn right3"></div>
      </div>

      <div class="friend-room-panel" v-else>
        <div class="friend-room-header">
          <img class="title_bg" src="@/assets/img/ui/title_bg.png" alt="标题背景" />
          <img class="friendroom" src="@/assets/img/ui/txt_friendroom.png" />
          <div class="img_return2" @click="toggleFriendRoom()"><img src="@/assets/img/ui/img_return2.png" alt="返回" /></div>
        </div>
        <div class="friend-room-body">
          <div class="room-action" @click="createRoom()">
            <img src="@/assets/img/ui/btn_create_room.png" class="btn_create_room" />
            <img src="@/assets/img/ui/tips.png" class="tips" />
            <span class="free">限时免费</span>
          </div>
          <div class="room-action" @click="showJoinInput = true">
            <img src="@/assets/img/ui/w_joinroom.png" class="w_joinroom" />
            <img src="@/assets/img/ui/tips.png" class="tips2" />
            <span class="free2">限时免费</span>
          </div>
          <div class="join-input-row" v-if="showJoinInput">
            <input type="text" placeholder="房间号" name="roomid" class="roominput" id="inroom" />
            <input type="submit" value="确定" class="roomsubmit" id="btn_inroom" @click="joinroom()"/>
          </div>
        </div>
      </div>
    </div>

    <footer class="bottom-buttons">
      <!-- <div class="btn btn-1"></div> -->
      <div class="btn btn-2"></div>
      <div class="btn btn-3"></div>
      <div class="btn btn-4"></div>
      <div class="btn btn-5"></div>
      <div class="btn btn-6"></div>
      <div class="btn btn-7"></div>
    </footer>

    <div v-if="showSettingImage" class="setting-image-container">
      <img src="@/assets/img/creator.jpg" alt="设置图片" class="setting-image">
      <div class="setting-image-overlay" @click="toggleSettingImage()"></div>
    </div>

    <div v-if="showLogoutModal" class="logout-modal-container">
      <div class="logout-modal-content">
        <p>确定要退出登录吗？</p>
        <div class="logout-modal-buttons">
          <button class="logout-btn cancel-btn" @click="toggleLogoutModal()">取消</button>
          <button class="logout-btn exit-btn" @click="logout()">退出</button>
        </div>
      </div>
      <div class="logout-modal-overlay" @click="toggleLogoutModal()"></div>
    </div>

    <div v-if="showGuideModal" class="guide-modal-container">
      <img src="@/assets/img/ui/act_bg.png" alt="规则说明背景" class="guide-modal-bg">
      <div class="guide-modal-content">
        <h2>游戏规则说明</h2>
        <div class="guide-content">
          <p>1. 游戏基本玩法介绍: 核心规则是 4 人对战，，先出完所有牌者获胜。</p>
          <p>2. 卡牌规则说明: 使用一副 52 张扑克牌,每个玩家开始游戏时，会随机分配13张牌，<br/>基本牌型：单张、对子、顺子、5张同花、三带二、四带一、5张同花顺.<br/>大小规则：单张牌，点数从大到小为 2、A、K、Q、J、10…3，对子类似，5张牌:同花顺 > 四带一 > 三带二 > 同花 > 顺子 。其中牌数相同才能打
            <br/>出牌顺序：拥有最小牌方块3先出牌，之后按逆时针顺序，下家需出比上家大的牌型，或选择不出
          </p>
          <p>3. 胜利条件:任意一名玩家先出完所有牌则该获胜</p>
          <p>4. 获胜奖励：游戏结束时，结算所有玩家剩余牌总数就是获胜瓜子数</p>
        </div>
        <button class="guide-close-btn" @click="toggleGuideModal()">关闭</button>
      </div>
      <div class="guide-modal-overlay" @click="toggleGuideModal()"></div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { audioManager } from '@/utils/audio'
import { storage } from '@/utils/storage'
import { useRoute, useRouter } from 'vue-router'
import { websocket } from '@/utils/websocket'
import { ElMessage, ElMessageBox } from 'element-plus';

import { reconnectWebSocket } from '@/utils/websocketReconnect';

const userName = ref('')
const point = ref(0)

const router = useRouter()
const route = useRoute()
const showSettingImage = ref(false)
const showGuideModal = ref(false)
const showLogoutModal = ref(false)
const showFriendRoom = ref(false)
const showJoinInput = ref(false)

const handleVisibilityChange = () => {
  if (document.hidden) {
    // 页面不可见时暂停音乐
    audioManager.pauseBGM()
    console.log('页面不可见，暂停音乐')
  } else {
    // 页面重新可见时恢复音乐
    if (!audioManager.isBGMPlaying()) {
      audioManager.playBGM('bgm')
      console.log('页面可见，恢复音乐')
    }
  }
}
const toggleSettingImage = () => {
  showSettingImage.value = !showSettingImage.value
}
const toggleGuideModal = () => {
  showGuideModal.value = !showGuideModal.value
}
const toggleLogoutModal = () => {
  showLogoutModal.value = !showLogoutModal.value
}
const toggleFriendRoom = () => {
  showFriendRoom.value = !showFriendRoom.value
  showJoinInput.value = false
}
const logout = () => {
  showLogoutModal.value = false
  if (websocket && websocket.close) {
    websocket.close()
  }
  storage.local.set('user', null)
  storage.local.remove('user')
  storage.local.set('user_id', null)
  storage.local.remove('user_id')
  router.push('/')
}

const userIdShow = ref(false)
onMounted(() => {
  document.addEventListener('visibilitychange', handleVisibilityChange)

  var userInfo = storage.local.get('user')
  userName.value = userInfo.username
  point.value = userInfo.point

  if (websocket.ws && websocket.ws.readyState === WebSocket.OPEN) {
    init();
  } else {
    reconnectWebSocket(websocket);
    const handleOpen = () => {
      websocket.off('open', handleOpen);
      init();
    };
    websocket.on('open', handleOpen);
  }
})
onUnmounted(() => {
  document.removeEventListener('visibilitychange', handleVisibilityChange)
  websocket.off('message', handleMessage)
})

const init = async () => {
  try {
    const bgmUrl = new URL('@/assets/music/yuanshanshaonian.mp3', import.meta.url).href;
    audioManager.preload('bgm', bgmUrl);
    audioManager.playBGM('bgm')
  } catch (error) {
    console.error('初始化背景音乐失败:', error);
  }

  const handleClose = (event) => {
    websocket.off('close', handleClose);
  };
  websocket.on('close', handleClose);

  const handleError = (error) => {
    websocket.off('error', handleError);
  };
  websocket.on('error', handleError);

  const sendGetInfo = (retryCount = 0) => {
    if (!websocket.ws || websocket.ws.readyState !== WebSocket.OPEN) {
      if (retryCount < 5) {
        setTimeout(() => sendGetInfo(retryCount + 1), 100);
      }
      return;
    }
    websocket.send({ "type": "getInfo", "data": "", "name": "" });
    const timeout = setTimeout(() => {
      if (retryCount < 5) {
        sendGetInfo(retryCount + 1);
      }
    }, 500);
    const handleMessageOnce = (data) => {
      try {
        const parsedData = typeof data === 'string' ? JSON.parse(data) : data;
        if (parsedData.type === "getInfo") {
          clearTimeout(timeout);
          websocket.off('message', handleMessageOnce);
        }
      } catch (e) {}
    };
    websocket.on('message', handleMessageOnce);
  };

  websocket.on('open', sendGetInfo);
  if (websocket.ws) {
    if (websocket.ws.readyState === WebSocket.OPEN) {
      sendGetInfo();
    }
  }
  websocket.on('message', handleMessage)
}

const createRoom = () => {
  websocket.send({ "type": "createRoom", "data": "", "name": "" });
  router.push({ path: '/room' })
}
const joinroom = () => {
  const input = document.getElementById("inroom");
  const roomid = input.value
  const data = JSON.stringify({roomID:roomid})
  websocket.send({ "type": "joinRoom", "data": data, "name": "" });
  router.push({ path: '/room' })
}
const info = () => {
  userIdShow.value = !userIdShow.value
}

const handleMessage = (data) => {
  const parsedData = JSON.parse(data)
  if (parsedData.type == "getInfo") {
    data = JSON.parse(parsedData.data)
    if (data.isPlaying && route.name !== "Game") {
      storage.local.set('roomInfo', JSON.stringify(data))
      router.push('/game')
    }
  }
  if (parsedData.type == "healthTip") {
    data = JSON.parse(parsedData.data)
    let msg = '游戏一定很精彩，但是今日你已在线' + data.hour + '小时了,请注意休息！！！'
    ElMessage.warning(msg);
  }
}
const toRoom = () => {
  websocket.send({ "type": "initRoom", "data": "", "name": "" });
  router.push({ path: '/room' })
}
</script>

<style scoped>
.index {
  background: url("@/assets/img/bg/beijing_chunying.png") no-repeat;
  background-size: 100% 100%;
  height: 100vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  position: relative;
}

.msg {
  position: absolute;
  top: 0;
  left: 11%;
  color: gold;
  z-index: 10;
}

.top-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  flex-shrink: 0;
}

.character-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.avatar {
  background: url('@/assets/img/touxiang/bighead.png') no-repeat;
  background-size: 100% 100%;
  width: 48px;
  height: 48px;
  border-radius: 50px;
  border: 2px solid rgba(255, 255, 255, 0.5);
}

.nickname p {
  font-size: 16px;
  color: white;
  margin: 0;
  text-shadow: 1px 1px 2px rgba(0, 0, 0, 0.6);
}

.currency-container {
  display: flex;
  gap: 10px;
}

.currency-item {
  display: flex;
  align-items: center;
  gap: 4px;
  background: rgba(0, 0, 0, 0.3);
  padding: 4px 10px;
  border-radius: 20px;
}

.gold-icon {
  background: url('@/assets/img/ui/gold0.png') no-repeat;
  background-size: 100% 100%;
  width: 20px;
  height: 20px;
}

.diamond-icon {
  background: url('@/assets/img/ui/gold1.png') no-repeat;
  background-size: 100% 100%;
  width: 20px;
  height: 20px;
}

.gold-amount,
.diamond-amount {
  font-size: 14px;
  color: white;
  margin: 0;
}

.right-settings {
  display: flex;
  gap: 8px;
}

.setting-btn {
  width: 36px;
  height: 36px;
  cursor: pointer;
  background-size: 100% 100%;
  background-repeat: no-repeat;
}

.btn-set {
  background-image: url("@/assets/img/ui/btn_set.png");
}

.btn-guide {
  background-image: url("@/assets/img/ui/btn_xinshouyindao.png");
}

.btn-trophy {
  background-image: url("@/assets/img/ui/btn_trophy.png");
}

.main-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 0 20px;
  position: relative;
  gap: 16px;
}

.character-art {
  /* background: url('@/assets/img/lihui/full15920.png') no-repeat center; */
  background-size: contain;
  width: 100%;
  max-width: 320px;
  height: 40vh;
  max-height: 280px;
  position: absolute;
  top: 0;
  left: 50%;
  transform: translateX(-50%);
  z-index: 0;
  opacity: 0.6;
  pointer-events: none;
}

.menu-buttons {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 100%;
  max-width: 280px;
  z-index: 1;
}

.menu-btn {
  width: 100%;
  height: 64px;
  background-size: 100% 100%;
  background-repeat: no-repeat;
  cursor: pointer;
  transition: transform 0.2s;
}

.menu-btn:active {
  transform: scale(0.96);
}

.right1 {
  background-image: url("@/assets/img/ui/btn_yibanchang.png");
}

.right2 {
  background-image: url("@/assets/img/ui/btn_yourenchang.png");
}

.right3 {
  background-image: url("@/assets/img/ui/btn_dajiangsai.png");
}

.friend-room-panel {
  width: 100%;
  max-width: 320px;
  z-index: 1;
}

.friend-room-header {
  position: relative;
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.title_bg {
  position: absolute;
  width: 100%;
  height: 100%;
  top: 0;
  left: 0;
}

.friendroom {
  position: relative;
  z-index: 1;
  height: 30px;
}

.img_return2 {
  position: absolute;
  left: 8px;
  top: 50%;
  transform: translateY(-50%);
  height: 30px;
  z-index: 2;
  cursor: pointer;
}

.img_return2 img {
  height: 100%;
}

.friend-room-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 16px;
}

.room-action {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.btn_create_room {
  width: 100%;
  height: 56px;
  object-fit: contain;
}

.w_joinroom {
  width: 100%;
  height: 56px;
  object-fit: contain;
}

.tips,
.tips2 {
  position: absolute;
  right: 8px;
  top: 50%;
  transform: translateY(-50%);
  height: 20px;
}

.free,
.free2 {
  font-size: 14px;
  color: white;
  margin-top: 4px;
  text-shadow: 1px 1px 2px rgba(0, 0, 0, 0.6);
}

.join-input-row {
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: center;
  margin-top: 8px;
}

.roominput {
  flex: 1;
  max-width: 180px;
  height: 36px;
  padding: 0 10px;
  border-radius: 4px;
  border: 1px solid #ccc;
  font-size: 14px;
}

.roomsubmit {
  height: 36px;
  width: 70px;
  background: red;
  color: white;
  border: none;
  border-radius: 4px;
  font-size: 14px;
  cursor: pointer;
}

.bottom-buttons {
  display: flex;
  justify-content: center;
  gap: 8px;
  padding: 12px 16px;
  flex-shrink: 0;
}

.bottom-buttons .btn {
  width: 36px;
  height: 36px;
  background-size: 100% 100%;
  background-repeat: no-repeat;
  cursor: pointer;
}

.btn-1 { background-image: url("@/assets/img/ui/btn7.png"); }
.btn-2 { background-image: url("@/assets/img/ui/btn4.png"); }
.btn-3 { background-image: url("@/assets/img/ui/btn0.png"); }
.btn-4 { background-image: url("@/assets/img/ui/btn2.png"); }
.btn-5 { background-image: url("@/assets/img/ui/btn5.png"); }
.btn-6 { background-image: url("@/assets/img/ui/btn3.png"); }
.btn-7 { background-image: url("@/assets/img/ui/btn6.png"); }

.setting-image-container {
  position: fixed;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  z-index: 1000;
}

.setting-image {
  max-width: 280px;
  max-height: 280px;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.setting-image-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: transparent;
  z-index: -1;
}

.guide-modal-container {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 2000;
  display: flex;
  justify-content: center;
  align-items: center;
}

.guide-modal-bg {
  position: absolute;
  width: 90%;
  height: 90%;
  opacity: 0.9;
}

.guide-modal-content {
  position: relative;
  border-radius: 12px;
  padding: 20px;
  max-width: 85vw;
  max-height: 70vh;
  overflow-y: auto;
  z-index: 10;
  color: white;
}

.guide-modal-content::-webkit-scrollbar {
  width: 8px;
}
.guide-modal-content::-webkit-scrollbar-track {
  background: rgba(0, 0, 0, 0.3);
  border-radius: 6px;
}
.guide-modal-content::-webkit-scrollbar-thumb {
  background: linear-gradient(180deg, #999, #666);
  border-radius: 6px;
}
.guide-modal-content {
  scrollbar-width: thin;
  scrollbar-color: #888 rgba(0, 0, 0, 0.3);
}

.guide-modal-content h2 {
  text-align: center;
  color: gold;
  margin-bottom: 16px;
  font-size: 22px;
}

.guide-content {
  font-size: 14px;
  line-height: 1.8;
}

.guide-content p {
  margin-bottom: 12px;
}

.guide-close-btn {
  display: block;
  margin: 16px auto 0;
  padding: 8px 24px;
  background: gold;
  color: #333;
  border: none;
  border-radius: 20px;
  font-size: 16px;
  font-weight: bold;
  cursor: pointer;
}

.guide-modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: transparent;
  z-index: 9;
}

.logout-modal-container {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 3000;
  display: flex;
  justify-content: center;
  align-items: center;
}
.logout-modal-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  z-index: 1;
}
.logout-modal-content {
  position: relative;
  background: rgba(0, 0, 0, 0.7);
  border-radius: 12px;
  padding: 30px 40px;
  text-align: center;
  z-index: 2;
  border: 2px solid gold;
}
.logout-modal-content p {
  color: white;
  font-size: 18px;
  margin-bottom: 24px;
}
.logout-modal-buttons {
  display: flex;
  gap: 24px;
  justify-content: center;
}
.logout-btn {
  padding: 10px 28px;
  border: none;
  border-radius: 25px;
  font-size: 16px;
  font-weight: bold;
  cursor: pointer;
}
.cancel-btn {
  background: #666;
  color: white;
}
.exit-btn {
  background: gold;
  color: #333;
}
</style>
