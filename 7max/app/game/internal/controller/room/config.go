package room

import (
	"context"
	"game/internal/controller/set_game"

	"github.com/gogf/gf/v2/frame/g"
)

// 将全局变量改为函数，每次需要时重新从配置中获取
// 最大出牌时间
func GetOutCardTimeout() int {
	return g.Cfg().MustGet(context.Background(), "game.outCardTimeout").Int() //出牌最大时间
}

// 每局抽水2(单位积分) /张
func GetGameCommission() int64 {

	commValue, _ := set_game.SetCache.Get(context.Background(), "comm")
	return commValue.Int64()
	//return g.Cfg().MustGet(context.Background(), "game.comm").Int64() //每局抽水2
}

// 初始化玩家积分
func GetGameInitPoint() int64 {
	pointValue, err := set_game.SetCache.Get(context.Background(), "point")
	if err != nil || pointValue == nil || pointValue.IsNil() {
		if pointValue != nil && pointValue.IsNil() {
			return 1000
		}
		if cfg, cfgErr := g.Cfg().Get(context.Background(), "game.point"); cfgErr == nil {
			return cfg.Int64()
		}
		return 1000
	}
	return pointValue.Int64()
}

// 初始化玩家积分
func GetAllowedOrigin() string {
	return g.Cfg().MustGet(context.Background(), "game.allowedOrigin").String() //初始化积分(玩家初始进来送的积分)
}
