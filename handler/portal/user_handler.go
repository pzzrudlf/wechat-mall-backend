package portal

import (
	"encoding/json"
	"github.com/go-playground/validator"
	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strings"
	"wechat-mall-backend/dbops/rediscli"
	"wechat-mall-backend/defs"
	"wechat-mall-backend/errs"
	"wechat-mall-backend/utils"
)

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]
	if code == "" {
		panic(errs.NewParameterError("缺少code"))
	}
	token, userId := h.service.UserService.LoginCodeAuth(code)
	go h.recordVisitorRecod(userId, r)

	defs.SendNormalResponse(w, defs.WxappLoginVO{Token: token})
}

// 访客记录
func (h *Handler) recordVisitorRecod(userId int, r *http.Request) {
	defer func() {
		err := recover()
		if err != nil {
			log.Print(err)
		}
	}()
	userIP := utils.ReadUserIP(r)
	h.service.UserService.DoAddVisitorRecord(userId, userIP)
}

// 查询用户信息
func (h *Handler) UserInfo(w http.ResponseWriter, r *http.Request) {
	userId := r.Context().Value(defs.ContextKey).(int)

	userDO := h.service.UserService.QueryUserInfo(userId)
	userVO := defs.WxappUserInfoVO{}
	userVO.Nickname = userDO.Nickname
	userVO.Avatar = userDO.Avatar
	if userDO.Mobile != "" {
		userVO.Mobile = utils.PhoneMark(userDO.Mobile)
	}
	defs.SendNormalResponse(w, userVO)
}

func (h *Handler) AuthPhone(w http.ResponseWriter, r *http.Request) {
	req := defs.WxappAuthPhone{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		panic(err)
	}
	validate := validator.New()
	if err = validate.Struct(req); err != nil {
		panic(errs.NewParameterError(err.Error()))
	}
	userId := r.Context().Value(defs.ContextKey).(int)
	authorization := r.Header.Get("Authorization")
	accessToken := strings.Split(authorization, " ")[1]

	cacheData, err := rediscli.GetStr(defs.MiniappTokenPrefix + accessToken)
	if err == redis.Nil {
		panic(errs.ErrorTokenInvalid)
	}
	if err != nil {
		panic(err)
	}
	if cacheData == "" {
		panic(errs.ErrorTokenInvalid)
	}
	result := make(map[string]interface{})
	err = json.Unmarshal([]byte(cacheData), &result)
	if err != nil {
		panic(err)
	}
	h.service.UserService.DoWxUserPhoneSignature(userId, result["session_key"].(string), req.EncryptedData, req.Iv)
	defs.SendNormalResponse(w, "ok")
}

func (h *Handler) AuthUserInfo(w http.ResponseWriter, r *http.Request) {
	req := defs.WxappAuthUserInfoReq{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		panic(errs.ErrorParameterValidate)
	}
	validate := validator.New()
	err = validate.Struct(req)
	if err != nil {
		panic(errs.NewParameterError(err.Error()))
	}
	userId := r.Context().Value(defs.ContextKey).(int)

	h.service.UserService.DoUserAuthInfo(userId, req)
	defs.SendNormalResponse(w, "ok")
}
