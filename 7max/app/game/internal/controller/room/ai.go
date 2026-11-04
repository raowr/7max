package room

import "sort"

func DecideCards(player, landlord *Player, lastPH int, lastCards []Card) (pokerHandType int, playCards []Card) {
	// playCards := make([]Card, 0)
	//正常情况,正常出牌，但要比上一家大，而且牌型相同的最小牌组出
	if len(lastCards) > 0 {
		pokerHandCards := make([][]Card, 0)
		if lastPH == SINGLE || lastPH == PAIR { //单牌和对子这样比比较
			for pokerHand, v := range player.handPattern {
				if pokerHand == lastPH {
					for _, v1 := range v {
						if compareCards(lastPH, pokerHand, lastCards, v1) {
							pokerHandCards = append(pokerHandCards, v1)
						}
					}
				}
			}
			if len(pokerHandCards) > 0 {
				playCards = pokerHandCards[0] //最小牌组
				for _, v := range pokerHandCards {
					if !compareCards(lastPH, lastPH, playCards, v) {
						playCards = v
					}
				}
			}
			//先排序取最小牌
			sort.Slice(player.Cards, func(i, j int) bool {
				return player.Cards[i].Id < player.Cards[j].Id
			})
			//如果是单张打最小的单张
			if lastPH == SINGLE {
				//找比上一手牌大的最小牌
				for i := 0; i < len(player.Cards); i++ {
					if player.Cards[i].Id > lastCards[0].Id {
						playCards = player.Cards[i : i+1]
						break
					}
				}
			}
			//如果是对子打最小的对子
			if lastPH == PAIR {
				pairCardAll := [][]Card{}
				pairCard := []Card{}
				rankCount := player.countRanks()
				for rank, count := range rankCount {
					if count >= 2 {
						for _, v := range player.Cards {
							if v.Rank == rank && len(pairCard) < 2 {
								pairCard = append(pairCard, v)
							}
						}
						pairCardAll = append(pairCardAll, pairCard)
						pairCard = []Card{}
					}
				}
				//寻找大于上一手牌的最小对子，先找出对子里最大的id,和上一手牌的id比较
				lastCardsMaxCard := lastCards[0]
				for _, v1 := range lastCards {
					if v1.Id > lastCardsMaxCard.Id {
						lastCardsMaxCard = v1
					}
				}
				for _, v := range pairCardAll {
					maxCard := v[0]
					for _, v1 := range v {
						if v1.Id > maxCard.Id {
							maxCard = v1
						}
					}
					if maxCard.Id > lastCardsMaxCard.Id {
						playCards = v
						break
					}
				}
			}

		} else { //5张的这样比较,需要整理一下牌
			//重新整理牌
			player.ReCard = true
			player.handPattern = make(map[int][][]Card)
			player.Solve()
			pokerHandNew := make(map[int][][]Card, 0)
			for pokerHand, v := range player.handPattern {
				if pokerHand > lastPH { //牌型比上一手牌大
					pokerHandNew[pokerHand] = v
				} else if pokerHand == lastPH { //牌型形同
					for _, v1 := range v {
						if compareCards(lastPH, pokerHand, lastCards, v1) {
							pokerHandCards = append(pokerHandCards, v1)
						}
					}
				}
			}
			for pokerHand, v := range pokerHandNew {
				for _, v1 := range v {
					if compareCards(lastPH, pokerHand, lastCards, v1) {
						pokerHandCards = append(pokerHandCards, v1)
					}
				}
			}
			if len(pokerHandCards) > 0 {
				playCards = pokerHandCards[0] //最小牌组
				for _, v := range pokerHandCards {
					if !compareCards(lastPH, lastPH, playCards, v) {
						playCards = v
					}
				}
			}
		}
		pokerHandType = lastPH
	}
	//情况1：如果AI是大就是必出的，选择出最小牌所在的牌组
	if player.Must || len(lastCards) <= 0 {
		//重新整理牌
		player.ReCard = true
		player.handPattern = make(map[int][][]Card)
		player.Solve()
		//先取最小牌在牌组中，
		minCardId := 54 //最小牌id
		for _, v := range player.handPattern {
			for _, v1 := range v {
				for _, v2 := range v1 {
					if v2.Id < minCardId {
						minCardId = v2.Id
					}
				}
			}
		}
		findCard := false
		for pokerHand, v := range player.handPattern {
			for _, v1 := range v {
				for _, v2 := range v1 {
					if v2.Id == minCardId {
						findCard = true
						break
					}
				}
				if findCard {
					playCards = v1
					break
				}
			}
			if findCard {
				pokerHandType = pokerHand
				break
			}
		}
		//情况2,玩家牌数小于5张时,优先出比玩家剩余牌数多的牌组,从多的往少的找
		if 1 < len(landlord.Cards) && len(landlord.Cards) < 5 {
			pokerHandCards := make([]Card, 0)
			for pokerHand, v := range player.handPattern {
				if pokerHand > PAIR {
					for _, v1 := range v {
						pokerHandCards = append(pokerHandCards, v1...)
					}
				}
			}
			var minCardId int
			if len(pokerHandCards) > 0 {
				minCardId = pokerHandCards[0].Id //最小牌id
				for _, v := range pokerHandCards {
					if v.Id < minCardId {
						minCardId = v.Id
					}
				}
			}
			for pokerHand, v := range player.handPattern {
				if pokerHand <= PAIR {
					for _, v1 := range v {
						for _, v2 := range v1 {
							if v2.Id == minCardId {
								playCards = v1
								pokerHandType = pokerHand
								break
							}
						}
					}
				}
			}
		}
	}
	return
}
