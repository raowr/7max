<template>
  <div class="game">
    <section class="top-bar">
      <div class="score-panel">
        <div class="score-row">本局得分：{{ state.player2InPoint }}</div>
      </div>
      <div class="opponent-card">
        <div class="avatar-wrap">
          <img :src="player2touxiang" class="avatar big">
          <div v-if="state.countdownPlayer == 2" class="opponent-countdown">
            <svg width="50" height="50">
              <circle cx="25" cy="25" r="22" stroke="#eee" stroke-width="4" fill="transparent"/>
              <circle cx="25" cy="25" r="22"
                :stroke="countdownPlayer2 > (state.outCardTimeout / 3) ? '#4CAF50' : '#ff5722'"
                stroke-width="4" fill="transparent"
                :style="{
                  strokeDasharray: 138.23,
                  strokeDashoffset: 138.23 * (1 - countdownPlayer2 / state.outCardTimeout),
                  transition: 'stroke-dashoffset 1s linear'
                }"/>
            </svg>
            <span class="cd-num">{{ countdownPlayer2 }}</span>
          </div>
        </div>
        <div class="name-plate opponent-name">
          <img src="@/assets/img/ui/namebg.png" class="name-bg">
          <span class="name-text">{{ state.player2Name }}</span>
        </div>
        <div v-if="player2pass" class="pass-tag opponent-pass">过</div>
      </div>
    </section>

    <section class="opponent-hand">
      <img v-for="n in state.player2CardsNum" :key="n"
        src="@/assets/img/54.png" class="card-back">
    </section>

    <section class="play-area">
      <div class="discard-pile">
        <img v-for="cardId in state.outCards" :key="cardId" :src="getCardImage(cardId)" class="discard-card">
      </div>
    </section>


        <section class="info-panel">
      <div class="info-row">剩余牌数：{{ state.allCardsNum }}</div>
      <div class="info-row">本局得分：{{ state.player1InPoint }}</div>
    </section>

    <section  class="actions-area">
      <img  src='@/assets/img/btn_buchu.png' class="action-btn buchu"
        :style="{ transform: isPassBtnPressed ? 'scale(0.9)' : 'scale(1)' }"
        @click='pass()' @mousedown="isPassBtnPressed = true" @mouseup="isPassBtnPressed = false"
        @mouseleave="isPassBtnPressed = false" @touchstart="isPassBtnPressed = true"
        @touchend="isPassBtnPressed = false">
        
      <div class="cd-wrap">
        <svg width="56" height="56">
          <circle cx="28" cy="28" r="24" stroke="#eee" stroke-width="4" fill="transparent"/>
          <circle cx="28" cy="28" r="24"
            :stroke="countdownPlayer1 > (state.outCardTimeout / 3) ? '#4CAF50' : '#ff5722'"
            stroke-width="4" fill="transparent"
            :style="{
              strokeDasharray: 150.8,
              strokeDashoffset: 150.8 * (1 - countdownPlayer1 / state.outCardTimeout),
              transition: 'stroke-dashoffset 1s linear'
            }"/>
        </svg>
        <span class="cd-num big">{{ countdownPlayer1 }}</span>
      </div>

      <img v-if="checkOut()" src='@/assets/img/btn_chupai.png' class="action-btn chupai"
        :style="{ transform: isChupaiBtnPressed ? 'scale(0.9)' : 'scale(1)' }"
        @click='chupai()' @mousedown="isChupaiBtnPressed = true" @mouseup="isChupaiBtnPressed = false"
        @mouseleave="isChupaiBtnPressed = false" @touchstart="isChupaiBtnPressed = true"
        @touchend="isChupaiBtnPressed = false">
      <img v-else src='@/assets/img/btn_chupai_hui.png' class="action-btn chupai disabled">



    </section>



    <section class="bottom-area">
      <div class="self-card">
        <div class="avatar-wrap">
          <img src="@/assets/img/touxiang/smallhead.png" class="avatar me">
          <div v-if="state.countdownPlayer == 1" class="self-countdown">
            <svg width="40" height="40">
              <circle cx="20" cy="20" r="17" stroke="#eee" stroke-width="3" fill="transparent"/>
              <circle cx="20" cy="20" r="17"
                :stroke="countdownPlayer1 > (state.outCardTimeout / 3) ? '#4CAF50' : '#ff5722'"
                stroke-width="3" fill="transparent"
                :style="{
                  strokeDasharray: 106.81,
                  strokeDashoffset: 106.81 * (1 - countdownPlayer1 / state.outCardTimeout),
                  transition: 'stroke-dashoffset 1s linear'
                }"/>
            </svg>
            <span class="cd-num small">{{ countdownPlayer1 }}</span>
          </div>
        </div>
        <div class="name-plate self-name">
          <img src="@/assets/img/ui/namebg.png" class="name-bg">
          <span class="name-text">{{ state.player1Name }}</span>
        </div>
        <div v-if="player1pass" class="pass-tag self-pass">过</div>
      </div>

      <div class="hand-cards">
        <img v-for="(cardId, index) in state.cards" :key="cardId"
          :src="getCardImage(cardId)" class="hand-card"
          :style="{
            transform: selectedCards.includes(cardId) ? 'translateY(-18px)' : '',
            zIndex: selectedCards.includes(cardId) ? 10 : index
          }"
          @click="toggleCard(cardId)">
      </div>
    </section>

    <div v-if="showGameOverModal" class="game-over-modal">
      <div class="modal-content">
        <h2 class="result-title">{{ isWinner ? '恭喜胜利!' : '游戏失败' }}</h2>
        <div class="result-bg" :style="{
          backgroundImage: isWinner ? `url(${winBg})` : `url(${loseBg})`
        }"></div>
        <p class="score-info">{{ gameOverMessage }}</p>
        <div class="buttons-container">
          <button class="room-btn" @click="toRoom()">返回房间</button>
          <button class="restart-btn" v-if="currentPlayerId === 0" @click="state.player1Point === 0 ? goToHome() : restartGame()">
            {{ state.player1Point === 0 ? '返回首页' : '再来一局' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted, onBeforeUnmount, computed } from 'vue'
import { useRouter } from 'vue-router'
import { audioManager } from '@/utils/audio'
import { websocket } from '@/utils/websocket'
import { cardUtil, CARD_TYPE } from '@/utils/card';
import { storage } from '@/utils/storage'
import { getTouxiang } from '@/utils/touxiang'
import { ElMessage, ElMessageBox } from 'element-plus';
import winBg from '@/assets/img/ui/win_bg.png'
import loseBg from '@/assets/img/ui/lose_bg.png'
import { reconnectWebSocket } from '@/utils/websocketReconnect';

const state = reactive({
  deck: [],
  countdownPlayer: 0,
  cards: [1,2,3,4,5],
  allCardsNum: 44,
  player2CardsNum: 5,
  outCards: [],
  lastOutCards: [],
  lastmsg: "",
  mustPid: 0,
  player1Point: 0,
  outCardTimeout: 30,
  containerAtEndPosition: false,
  movingCards: [],
  player1Name: "",
  player2Name: "",

  player1InPoint: 0,
  player2InPoint: 0,
})
const isPassBtnPressed = ref(false)
const isChupaiBtnPressed = ref(false)

const showGameOverModal = ref(false)
const isWinner = ref(false)
const gameOverMessage = ref('')
const router = useRouter()

const selectedCards = ref([])

const countdownPlayer1 = ref(state.outCardTimeout)
const countdownPlayer2 = ref(state.outCardTimeout)
let timer1 = null
let timer2 = null

const isPlayingCard = ref(false)
const pendingCards = ref([])
const playCardTimer = ref(null)

const screenWidth = ref(window.innerWidth);
const scaleFactor = ref(screenWidth.value < 998 ? 0.6 : 1);
const bgmInitialized = ref(false)

const player2touxiang = ref("")

const player1pass = ref(false)
const player2pass = ref(false)

const currentPlayerUid = ref(storage.local.get('user_id'))
const currentPlayerId = ref(0)
const proactivePass = ref(false)
const roomType = ref(0)



const handleResize = () => {
  screenWidth.value = window.innerWidth;
  scaleFactor.value = screenWidth.value < 998 ? 0.6 : 1;
};

const fetchGameDataFromWebSocket = () => {
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
    } else if (websocket.ws.readyState === WebSocket.CONNECTING) {
      console.log('game WebSocket 正在连接中，等待 open 事件');
    } else {
      console.log('game WebSocket 未连接，等待重连');
    }
  }
}

onMounted(() => {
  state.deck = initDeck()
  reconnectWebSocket(websocket);

  const bgmUrl = new URL('@/assets/music/game_bg1.mp3', import.meta.url).href;
  audioManager.preload('bgm', bgmUrl);
  const initBGMOnInteraction = () => {
    if (!bgmInitialized.value) {
      audioManager.playBGM('bgm')
      bgmInitialized.value = true
      document.removeEventListener('click', initBGMOnInteraction)
      document.removeEventListener('touchstart', initBGMOnInteraction)
    }
  }
  document.addEventListener('click', initBGMOnInteraction)
  document.addEventListener('touchstart', initBGMOnInteraction)

  const cachedData = storage.local.get('roomInfo');
  if (cachedData) {
    try {
      const gameData = JSON.parse(cachedData);
      initGameWithData(gameData);
      storage.local.remove('roomInfo');
    } catch (e) {
      console.error('解析缓存数据失败', e);
      fetchGameDataFromWebSocket();
    }
  } else {
    fetchGameDataFromWebSocket();
  }

  websocket.on('message', handleMessage)
  window.addEventListener('resize', handleResize)

  player2touxiang.value = storage.local.get('player2avatar')
})

onUnmounted(() => {
  websocket.off('open', () => {});
  websocket.off('message', () => {});
  websocket.off('error', () => {});
  websocket.off('close', () => {});
  document.removeEventListener('click', () => {});
  document.removeEventListener('touchstart', () => {});
  window.removeEventListener('resize', handleResize);
});

const initDeck = () => {
    const values = ["4", "6", "8", "9", "10", "J", "Q", "K", "A", "3", "2", "5", "鬼", "7"];
    const suits = ["♦", "♣", "♥", "♠"];
    const joker = ["小", "大"];
    const deck = [];
    let cardId = 0;

    for (const [i, valueName] of values.entries()) {
        const rank = 1 + i;
        const isJoker = valueName === "鬼";
        if (isJoker) {
            for (const [s, suitName] of joker.entries()) {
                cardId++;
                deck.push({
                    id: cardId,
                    value: valueName,
                    suit: s,
                    name: suitName + valueName,
                    rank: rank,
                    suitName: suitName,
                });
            }
        } else {
            for (const [s, suitName] of suits.entries()) {
                cardId++;
                deck.push({
                    id: cardId,
                    value: valueName,
                    suit: s,
                    name: suitName + valueName,
                    rank: rank,
                    suitName: suitName,
                });
            }
        }
    }

    return deck;
}

const initGameWithData =(data) =>{
    roomType.value = data.type
    for (let i = 0; i < data.players.length; i++) {
      if (data.players[i].UserId == currentPlayerUid.value){
        currentPlayerId.value = data.players[i].ID
      }
    }
    for (let i = 0; i < data.players.length; i++) {
      switch (getPlayerPosition(data.players[i].ID) + 1) {
        case 1: state.player1Name = data.players[i].Name; break;
        case 2: state.player2Name = data.players[i].Name; break;
      }
    }
    if (data.isPlaying) {
      state.outCards = (data.outCards)
      state.lastOutCards = (data.outCards)
      state.mustPid = data.mustPid
      state.cards = data.cards.sort((a, b) => b - a);
      for (let i = 0; i < data.cardsNum.length; i++) {
        switch (getPlayerPosition(data.cardsNum[i].id) + 1) {
          case 2: state.player2CardsNum = data.cardsNum[i].cardNum; break;
        }
      }
      if (data.outCardTimeout !== undefined) {
        state.outCardTimeout = data.outCardTimeout
      }
      startCountdown(getPlayerPosition(data.current) + 1, data.remainOutCardTimeout)
      state.player1Point = data.playerPoint
      let cardsMsg = ""
      for (let i = 0; i < data.outCards.length; i++) {
        for (let j = 0; j < state.deck.length; j++) {
          if (state.deck[j].id == data.outCards[i]) {
            cardsMsg += state.deck[j].name + " "
          }
        }
      }
      switch (getPlayerPosition(data.lastPid)) {
        case 0: state.lastmsg = state.player1Name + "出了：" + cardsMsg; break;
        case 1: state.lastmsg = state.player2Name + "出了：" + cardsMsg; break;
      }
    } else {
      switch (data.status) {
        case 2:
          console.log("结算中");
          router.push('/index')
          break;
      }
    }
}

const handleMessage = (data) => {
  console.log('Received message:', data)
  const parsedData = JSON.parse(data)
  if (parsedData.type == "getInfo") {
    data = JSON.parse(parsedData.data)
    initGameWithData(data)
  }
  if (parsedData.type == "showCard") {
    showGameOverModal.value = false
    resetAvatar()
    data = JSON.parse(parsedData.data)
    state.cards = data.cards.sort((a, b) => b - a);
    state.player2CardsNum = 5
    if (data.outCardTimeout !== undefined) {
      state.outCardTimeout = data.outCardTimeout
    }
    startCountdown(getPlayerPosition(data.current) + 1, state.outCardTimeout)
    state.mustPid = data.current
    state.player1Point = data.playerPoint
    countdownPlayer1.value = state.outCardTimeout
    selectedCards.value = []
    player1pass.value = false
    player2pass.value = false
  }
  if (parsedData.type == "outCard") {
    data = JSON.parse(parsedData.data)
    let cardsMsg = ""
    switch (getPlayerPosition(data.pid)) {
      case 0:
        if (isPlayingCard.value) {
          isPlayingCard.value = false
          clearTimeout(playCardTimer.value)
          state.movingCards = []
        }
        if (data.code == 0) {
          for (let i = 0; i < data.cardIds.length; i++) {
            for (let j = 0; j < state.deck.length; j++) {
              if (state.deck[j].id == data.cardIds[i]) {
                cardsMsg += state.deck[j].name + " "
              }
            }
          }
          state.lastmsg = state.player1Name + "出了：" + cardsMsg
          state.cards = state.cards.filter(item =>
            !data.cardIds.some(cardId => cardId === item)
          );
          selectedCards.value = []
          pendingCards.value = []
          player1pass.value = false
        } else {
          state.cards = [...state.cards, ...pendingCards.value]
          state.cards.sort((a, b) => b - a)
          state.outCards = state.lastOutCards
        }
        break
      case 1:
        state.player2CardsNum = data.cards_num
        for (let i = 0; i < data.cardIds.length; i++) {
          for (let j = 0; j < state.deck.length; j++) {
            if (state.deck[j].id == data.cardIds[i]) {
              cardsMsg += state.deck[j].name + " "
            }
          }
        }
        state.lastmsg = state.player2Name + "出了：" + cardsMsg
        player2pass.value = false
        break
    }
    if (data.code == 0) {
      state.outCards = (data.cardIds)
      state.lastOutCards = (data.lastOutCards)
      if (data.outCardTimeout !== undefined) {
        state.outCardTimeout = data.outCardTimeout
      }
      startCountdown(getPlayerPosition(data.current) + 1, state.outCardTimeout)
      state.mustPid = data.mustPid

      const cardType = cardUtil.getCardType(state.outCards.map(id => state.deck.find(c => c.id === id)).filter(Boolean))
      const playedCards = state.outCards.map(id => state.deck.find(c => c.id === id)).filter(Boolean)
      let musicPath = ""
      switch (cardType) {
        case CARD_TYPE.SINGLE: musicPath = "single/" + playedCards[0].id; playSound(musicPath); break
        case CARD_TYPE.PAIR: musicPath = "pair/" + playedCards[0].rank; playSound(musicPath); break
        case CARD_TYPE.STRAIGHT: musicPath = "straight/straight"; playSound(musicPath); break
        case CARD_TYPE.SUIT: musicPath = "suit/suit"; playSound(musicPath); break
        case CARD_TYPE.FULL_HOUSE:
          let countFull = cardUtil.countRanks(playedCards)
          let rankFull = cardUtil.getRankByCount(countFull, 3)
          musicPath = "full_house/" + rankFull; playSound(musicPath); break
        case CARD_TYPE.FOUR_OF_A_KIND:
          let countFour = cardUtil.countRanks(playedCards)
          let rankFour= cardUtil.getRankByCount(countFour, 4)
          musicPath = "four/" + rankFour; playSound(musicPath); break
        case CARD_TYPE.STRAIGHT_FLUSH: musicPath = "straight_flush/straight_flush"; playSound(musicPath); break
        default: playSound('default'); break
      }
     if (getPlayerPosition(data.current) == 0) player1pass.value = false
     else if (getPlayerPosition(data.current) == 1) player2pass.value = false
    }
  }
  if (parsedData.type == "pass") {
    data = JSON.parse(parsedData.data)
    state.mustPid = data.mustPid
    if (!proactivePass.value) {
      playSound("guo")
    }
    startCountdown(getPlayerPosition(data.current) + 1, state.outCardTimeout)
    if (data.pid == currentPlayerId.value ){
      proactivePass.value = false
    }
     if (getPlayerPosition(data.pid) == 0) player1pass.value = true
    else if (getPlayerPosition(data.pid) == 1) player2pass.value = true

    if (getPlayerPosition(data.current) == 0) player1pass.value = false
    else if (getPlayerPosition(data.current) == 1) player2pass.value = false

    state.allCardsNum = data.allCardsNum

    state.player1InPoint = data.inPoint?.["0"] ?? 0
    state.player2InPoint = data.inPoint?.["1"] ?? 0

    state.cards = data.cards.sort((a, b) => b - a);

    state.outCards = []

    state.player2CardsNum = data.playerCardNum?.["1"] ?? 0

  }
  if (parsedData.type == "over") {
    clearInterval(timer1)
    clearInterval(timer2)
    state.countdownPlayer = 0
    data = JSON.parse(parsedData.data)
    data.forEach(winer=>{
        if (winer.win > 0 ){
          state.lastmsg = "游戏结束，玩家: " + winer.winName + "胜利,赢得:" + winer.win + "颗瓜子"
        }
      if (winer.winner == currentPlayerId.value ){
        var win = Math.abs(winer.win)
        isWinner.value = winer.win > 0
        gameOverMessage.value = isWinner.value
                ? `哇！恭喜你赢得了${win}颗瓜子!`
                : `咦！你很菜,这局输了${win}颗瓜子`
        state.player1Point = winer.point
      }
    })
    showGameOverModal.value = true
    state.outCards = []
    if (isPlayingCard.value) {
      isPlayingCard.value = false
      pendingCards.value = []
      clearTimeout(playCardTimer.value)
    }
    player1pass.value = false
    player2pass.value = false
    if (state.player1Point <= 0) {
      router.push({ path: '/index' })
    }
    var userInfo = storage.local.get('user')
    userInfo = userInfo || {"name":"","point":0}
    userInfo.point = state.player1Point
    if (roomType.value == 1) {
      storage.local.set('user', userInfo)
    }
  }
  if (parsedData.type === "play") {
    showGameOverModal.value = false
  }
  if (parsedData.type == "healthTip") {
    data = JSON.parse(parsedData.data)
    let msg  = '游戏一定很精彩，但是今日你已在线'+data.hour+'小时,请注意休息！！！'
    ElMessage.warning(msg);
  }
}

const audioFiles = import.meta.glob('@/assets/music/**/*.mp3', { eager: true, import: 'default' });
const getAudioUrl = (musicPath) => {
  try {
    const audioKey = `/src/assets/music/${musicPath}.mp3`;
    return audioFiles[audioKey] || '';
  } catch (error) {
    console.error('获取音频URL失败:', error);
    return '';
  }
};

const playSound = (musicPath) => {
  try {
    audioManager.playOnce(musicPath);
    if (!audioManager.audioElements.has(musicPath)) {
      const audioUrl = getAudioUrl(musicPath);
      if (audioUrl) {
        const audio = new Audio(audioUrl);
        audio.play().catch(err => {
          console.warn('播放音频失败:', err);
        });
      }
    }
  } catch (error) {
    console.error('播放声音时发生错误:', error);
  }
};

const restartGame = () => {
  websocket.send({ "type": "play", "data": "", "name": "" })
}

const toggleCard = (n) => {
  const index = selectedCards.value.indexOf(n)
  if (index > -1) {
    selectedCards.value.splice(index, 1)
  } else {
    selectedCards.value.push(n)
  }
}

const checkOut = () => {
  // console.log(state.lastOutCards)
  if (selectedCards.value.length === 0) return false;
  const currentCards = selectedCards.value.map(cardId => {
    return state.deck.find(card => card.id === cardId);
  }).filter(Boolean);
  const lastHand = {
    cards: state.lastOutCards.map(cardId => {
      return state.deck.find(card => card.id === cardId);
    }).filter(Boolean),
    isSelf: state.mustPid === currentPlayerId.value,
    type: state.lastOutCards.length > 0
      ? cardUtil.getCardType(state.lastOutCards.map(id => state.deck.find(c => c.id === id)).filter(Boolean))
      : null
  };
  let res = cardUtil.canPlayCards(currentCards, lastHand);
  return res
};

const startCountdown = (pid, remainOutCardTimeout) => {
  clearInterval(timer1)
  clearInterval(timer2)
  state.countdownPlayer = pid
  switch (pid) {
    case 1:
      clearInterval(timer2)
      countdownPlayer1.value = remainOutCardTimeout
      let isPlayKuaidian = false
      timer1 = setInterval(() => {
        if (countdownPlayer1.value > 0) {
          countdownPlayer1.value--
          if (countdownPlayer1.value < remainOutCardTimeout/2) {
            if (!isPlayKuaidian) {
              isPlayKuaidian = true
              playSound('kuaidian')
            }
          }
        } else {
          clearInterval(timer1)
          state.countdownPlayer = 0
        }
      }, 1000)
      break
    case 2:
      clearInterval(timer1)
      countdownPlayer2.value = remainOutCardTimeout
      timer2 = setInterval(() => {
        if (countdownPlayer2.value > 0) {
          countdownPlayer2.value--
        } else {
          clearInterval(timer2)
          state.countdownPlayer = 0
        }
      }, 1000)
      break
  }
}

onBeforeUnmount(() => {
  clearInterval(timer1); clearInterval(timer2)
})

const chupai = () => {
  if (isPlayingCard.value) return false
  if (selectedCards.value.length === 0) return false
  const cardsToPlay = [...selectedCards.value]
  pendingCards.value = cardsToPlay
  isPlayingCard.value = true
  const originalCards = [...state.cards]
  state.cards = state.cards.filter(card => !cardsToPlay.includes(card))
  websocket.send({
    type: "playCard",
    data: JSON.stringify({
      pid: currentPlayerId.value,
      cardIds: cardsToPlay,
    }),
  })
  state.movingCards = [...cardsToPlay]
  selectedCards.value = []
  setTimeout(() => {
    state.movingCards = []
    // state.outCards = [...cardsToPlay]
    if (state.lastOutCards[0] != cardsToPlay[0]) {
      state.outCards.push(...cardsToPlay)
    }
    console.log("state.outCards:",state.outCards)
  }, 250)
  playCardTimer.value = setTimeout(() => {
    if (isPlayingCard.value) {
      state.cards = [...originalCards]
      state.cards.sort((a, b) => b - a)
    }
    isPlayingCard.value = false
    pendingCards.value = []
    state.movingCards = []
  }, 30000)
}

const getMovingCardStyle = (index) => {
  const totalCards = state.movingCards.length;
  const cardWidth = 120 * scaleFactor.value;
  const cardHeight = 140 * scaleFactor.value;
  const overlap = 10;
  const effectiveWidth = cardWidth - overlap;
  const totalWidth = cardWidth + (totalCards - 1) * effectiveWidth;
  const startOffset = -totalWidth / 2;
  const horizontalOffset = startOffset + index * effectiveWidth;
  return {
    width: `${cardWidth}px`,
    height: `${cardHeight}px`,
    transform: `translateX(${horizontalOffset}px) rotate(0deg)`
  };
};

const pass = () => {
  websocket.send({
    type: "playCard",
    data: JSON.stringify({
      pid: currentPlayerId.value,
      pass: 1,
    }),
  })
  state.countdownPlayer = 0
  player1pass.value = true
  playSound("guo")
  startCountdown(2, state.outCardTimeout)
  player2pass.value = false
  proactivePass.value = true
}

const goToHome = () => {
  showGameOverModal.value = false
  router.push('/index')
}

const cardImages = import.meta.glob('@/assets/img/cards/*.png', { eager: true, import: 'default' });
const getCardImage = (cardId) => {
  const cardIdStr = String(cardId).trim();
  const imageKey = `/src/assets/img/cards/${cardIdStr}.png`;
  return cardImages[imageKey] || '';
};

const chatlogBgUrl = computed(() => {
  return new URL('@/assets/img/ui/chatlog.png', import.meta.url).href;
});

const toRoom = () => {
  showGameOverModal.value = false
  if (roomType.value !== 2){
    websocket.send({"type":"initRoom","data":"","name":""});
  }
  router.push({path:'/room'})
}

const resetAvatar = () => {
  if (Math.random() > 0.5) {
    let player2avatar = getTouxiang()
    player2touxiang.value = player2avatar
    storage.local.set('player2avatar', player2avatar)
  }
}

const getPlayerPosition = (playerId) => {
  const position = (playerId - currentPlayerId.value + 2) % 2
  return position
}
</script>

<style scoped>
* { box-sizing: border-box; margin: 0; padding: 0; }

.game {
  background: url('@/assets/img/bg/beijing_chunying.png') no-repeat center/cover;
  height: 100vh;
  width: 100vw;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  color: white;
  padding: 8px;
}

.top-bar {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  flex-shrink: 0;
}

.score-panel {
  background: rgba(0, 0, 0, 0.35);
  padding: 6px 12px;
  border-radius: 10px;
}

.score-row {
  font-size: 14px;
  color: #FFE082;
  text-shadow: 1px 1px 2px rgba(0, 0, 0, 0.8);
}

.opponent-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  position: relative;
}

.avatar-wrap {
  position: relative;
  width: 54px;
  height: 54px;
  flex-shrink: 0;
}

.avatar {
  width: 54px;
  height: 54px;
  border-radius: 50%;
  border: 2px solid rgba(255, 255, 255, 0.7);
  background: #333;
}

.avatar.big {
  width: 60px;
  height: 60px;
}

.avatar.me {
  width: 44px;
  height: 44px;
}

.opponent-countdown {
  position: absolute;
  top: -4px;
  left: -4px;
  z-index: 2;
}

.cd-wrap {
  position: relative;
  width: 7vh;
  height: 7vh;
  min-width: 44px;
  min-height: 44px;
  max-width: 64px;
  max-height: 64px;
  flex-shrink: 1;
}

.cd-num {
  position: absolute;
  top: 67%;
  left: 61%;
  transform: translate(-50%, -50%);
  font-size: 14px;
  font-weight: bold;
  color: white;
  text-shadow: 0 0 4px rgba(0, 0, 0, 0.8);
}

.cd-num.big {
  font-size: 20px;
}

.cd-num.small {
  font-size: 12px;
}

.name-plate {
  position: relative;
  height: 24px;
  display: flex;
  align-items: center;
  width: 100%;
}

.opponent-name {
  width: auto;
  min-width: 70px;
}

.self-name {
  width: auto;
  min-width: 60px;
}

.name-bg {
  position: absolute;
  width: 100%;
  height: 100%;
  left: 0;
  top: 0;
}

.name-text {
  position: relative;
  z-index: 2;
  padding: 0 8px;
  font-size: 12px;
  color: white;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  text-shadow: 1px 1px 2px rgba(0, 0, 0, 0.8);
  text-align: center;
}

.pass-tag {
  position: absolute;
  background: linear-gradient(45deg, #FF6600, #FF3300);
  color: white;
  font-size: 14px;
  font-weight: bold;
  padding: 2px 8px;
  border-radius: 10px;
  box-shadow: 0 2px 8px rgba(255, 102, 0, 0.6);
  z-index: 5;
  animation: pulsePass 1s infinite;
}

.opponent-pass {
  top: -4px;
  right: -4px;
}

.self-pass {
  top: -4px;
  right: -4px;
}

@keyframes pulsePass {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.1); }
}

.opponent-hand {
  display: flex;
  justify-content: center;
  gap: 4px;
  padding: 6px 0;
  flex-shrink: 0;
}

.card-back {
  width: 30px;
  height: 42px;
  border-radius: 4px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.5);
}

.play-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 0;
}

.discard-pile {
  display: flex;
  justify-content: flex-start;
  align-items: center;
  flex-wrap: wrap;
  gap: 2px;
  padding: 8px;
}

.discard-card {
  width: 50px;
  height: 72px;
  border-radius: 5px;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.5);
  flex-shrink: 0;
}

.last-msg {
  font-size: 12px;
  color: #FFE082;
  text-shadow: 0 1px 3px rgba(0, 0, 0, 0.8);
  text-align: center;
  padding: 0 10px;
}

.actions-area {
  flex-shrink: 0;
  height: 12vh;
  min-height: 54px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 2vw;
}

.action-btn {
  height: 9vh;
  min-height: 44px;
  max-height: 64px;
  flex-shrink: 0;
  width: auto;
  max-width: 35vw;
  object-fit: contain;
  cursor: pointer;
  transition: transform 0.1s ease;
  display: block;
}

.action-btn.chupai { height: 10vh; min-height: 46px; max-height: 66px; }
.action-btn.chupai.disabled { opacity: 0.6; cursor: not-allowed; }
.action-btn.buchu { height: 9vh; min-height: 44px; max-height: 64px; }

.info-panel {
  flex-shrink: 0;
  padding: 4px 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  background: rgba(0, 0, 0, 0.25);
  border-radius: 8px;
  padding: 6px 12px;
  align-self: flex-start;
  margin-bottom: 6px;
}

.info-row {
  font-size: 13px;
  color: #FFE082;
  text-shadow: 1px 1px 2px rgba(0, 0, 0, 0.8);
}

.bottom-area {
  flex-shrink: 0;
  display: flex;
  align-items: flex-end;
  gap: 8px;
}

.self-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 3px;
  position: relative;
  flex-shrink: 0;
}

.self-countdown {
  position: absolute;
  top: -4px;
  left: -4px;
  z-index: 2;
}

.hand-cards {
  display: flex;
  justify-content: center;
  align-items: flex-end;
  flex: 1;
  min-height: 128px;
  overflow: visible;
}

.hand-card {
  width: 14vw;
  height: 110px;
  margin-left: -5px;
  flex-shrink: 0;
  border-radius: 5px;
  box-shadow: 0 2px 5px rgba(0, 0, 0, 0.5);
  transition: transform 0.2s ease;
  cursor: pointer;
}

.hand-card:first-child {
  /* margin-left: 0; */
}

.game-over-modal {
  position: fixed;
  top: 0; left: 0;
  width: 100vw; height: 100vh;
  background-color: rgba(0, 0, 0, 0.7);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.modal-content {
  padding: 20px;
  border-radius: 15px;
  text-align: center;
  width: 300px;
  background: rgba(0, 0, 0, 0.3);
  display: flex;
  flex-direction: column;
  align-items: center;
}

.result-title {
  color: white;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.8);
  font-size: 26px;
  margin-bottom: 12px;
}

.result-bg {
  width: 200px;
  height: 140px;
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
  margin-bottom: 12px;
  border-radius: 10px;
}

.score-info {
  font-size: 18px;
  margin-bottom: 20px;
  color: gold;
  font-weight: bold;
}

.buttons-container {
  display: flex;
  gap: 12px;
  justify-content: center;
}

.restart-btn,
.room-btn {
  padding: 10px 20px;
  font-size: 15px;
  font-weight: bold;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
}

.restart-btn {
  background-color: #4CAF50;
  color: white;
}

.room-btn {
  background: linear-gradient(45deg, #4169E1, #1E90FF);
  color: white;
}

.restart-btn:active,
.room-btn:active {
  transform: scale(0.95);
}
</style>