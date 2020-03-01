package service

import (
	"encoding/json"
	"fmt"
	"wechat-mall-web/defs"
	"wechat-mall-web/env"
	"wechat-mall-web/store"
	"wechat-mall-web/utils"
)

type IUserService interface {
	LoginCodeAuth(code string) (defs.WxappLoginResp, error)
}

type UserService struct {
	DbStore    *store.MySQLStore
	RedisStore *store.RedisStore
	Conf       *env.Conf
}

func NewUserService(dbStore *store.MySQLStore, redisStore *store.RedisStore, conf *env.Conf) *UserService {
	return &UserService{DbStore: dbStore, RedisStore: redisStore, Conf: conf}
}

func (service *UserService) LoginCodeAuth(code string) (defs.WxappLoginResp, error) {
	baseUrl := "https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code"
	url := fmt.Sprintf(baseUrl, service.Conf.Wxapp.Appid, service.Conf.Wxapp.Appsecret, code)

	tmpVal, err := utils.HttpGet(url)
	if err != nil {
		panic(err)
	}

	result := make(map[string]interface{})
	err = json.Unmarshal([]byte(tmpVal), &result)
	if err != nil {
		panic("微信内部异常！")
	}
	if result["errcode"] != nil {
		panic(result["errmsg"])
	}

	// {"session_key":"TppZM2zEd6\/dGzkqbbrriQ==","expires_in":7200,"openid":"oQOru0EUuLdidBZH0r_F8fDURPjI"}
	token := utils.RandomStr(32)
	err = service.RedisStore.SetStr(store.MiniappTokenPrefix+token, tmpVal, store.MiniappTokenExpire)
	if err != nil {
		panic("redis异常")
	}
	registerUser(service, result["openid"].(string))

	resp := defs.WxappLoginResp{Token: token, ExpirationInMinutes: store.MiniappTokenExpire}
	return resp, nil
}

func registerUser(us *UserService, openid string) {
	user, err := us.DbStore.GetUserByOpenid(openid)
	if err != nil {
		panic(err.Error())
	}
	if user.Id == 0 {
		newUser := &WxappUser{Openid: openid, Nickname: "", Avatar: "", Mobile: "", City: ""}
		_, err := us.DbStore.AddMiniappUser((*store.WxappUser)(newUser))
		if err != nil {
			panic(err.Error())
		}
	}
}