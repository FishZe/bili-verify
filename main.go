package main

import (
	"fmt"
	bili "github.com/FishZe/Go-BiliChat"
	"github.com/FishZe/Go-BiliChat/handler"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/patrickmn/go-cache"
	"golang.org/x/time/rate"
	"gopkg.in/yaml.v2"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	SuccessCode        = 0
	VerifyCodeNotUsed  = 1
	VerifyCodeNotFound = 2
	VerifyCodeEmpty    = 3
	ServerErrorCode    = 4
	AuthorizationError = 5
)

const (
	VerifyCodeNotUsedMsg  = "verify code not used"
	VerifyCodeNotFoundMsg = "verify code not found"
	VerifyCodeEmptyMsg    = "verify code is empty"
	AuthorizationErrorMsg = "authorization error"
	ServerErrorMsg        = "server error"
)

type Config struct {
	Port         int    `yaml:"port"`
	RoomId       int    `yaml:"room_id"`
	BaseUrl      string `yaml:"base_url"`
	NeedAuth     bool   `yaml:"need_auth"`
	ClientId     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
}

type User struct {
	Uid   int64  `json:"uid"`
	Name  string `json:"name"`
	Medal string `json:"medal"`
}

type Verify struct {
	Verified   bool   `json:"verified"`
	VerifyCode string `json:"verify_code"`
	User       User   `json:"user"`
}

type RespData struct {
	Error    string `json:"error"`
	QueryId  string `json:"queryId"`
	UserInfo User   `json:"userInfo"`
}

var verify = cache.New(5*time.Minute, 10*time.Minute)
var userLimit = cache.New(1*time.Minute, 1*time.Minute)
var conf Config
var limiter *rate.Limiter

func main() {
	gin.SetMode(gin.ReleaseMode)
	var err error
	conf, err = getConf()
	if err != nil {
		makeConfig()
		log.Fatal("请先配置config.yaml文件")
		return
	}
	if conf.NeedAuth {
		err = initDB()
		if err != nil {
			log.Printf("init db failed: %v", err)
		}
	}
	h := bili.GetNewHandler()
	h.AddOption(handler.CmdDanmuMsg, conf.RoomId, HandleDanmuMsg)
	// 连接到直播间
	h.AddRoom(conf.RoomId)
	// 启动处理器
	h.Run()
	h.Run()
	limiter = rate.NewLimiter(1000, 1000)
	router := gin.Default()
	verify := router.Group("/verify")
	{
		verify.POST("/new_verify", Auth(), MakeNewVerify)
		verify.POST("/query_verify", Auth(), QueryVerify)
	}
	if conf.NeedAuth {
		login := router.Group("/login")
		{
			login.GET("/", LimitRate(), LoginGithub)
			login.GET("/redirect", LimitRate(), RedirectGithub)
		}
	}
	err = router.Run(":" + strconv.Itoa(conf.Port))
	if err != nil {
		log.Printf("Run gin failed: %v", err)
		return
	}
}

func LimitRate() gin.HandlerFunc {
	return func(c *gin.Context) {
		ok := limiter.AllowN(time.Now(), 1)
		if ok {
			c.Next()
		} else {
			c.String(http.StatusTooManyRequests, "")
		}
	}
}

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !conf.NeedAuth {
			c.Next()
			return
		}
		authId := c.Request.Header.Get("Authorization")
		if authId == "" {
			c.JSON(http.StatusOK, gin.H{"code": AuthorizationError, "data": map[string]string{"error": AuthorizationErrorMsg}})
			c.Abort()
			return
		}
		user, err := getUserByUUID(authId)
		if err != nil {
			fmt.Println(err)
			c.JSON(http.StatusOK, gin.H{"code": ServerErrorCode, "data": map[string]string{"error": ServerErrorMsg}})
			c.Abort()
			return
		}
		if user.NodeId == "" {
			c.JSON(http.StatusOK, gin.H{"code": AuthorizationError, "data": map[string]string{"error": AuthorizationErrorMsg}})
			c.Abort()
			return
		}
		ok := limiter.AllowN(time.Now(), 1)
		if !ok {
			c.String(http.StatusTooManyRequests, "")
			c.Abort()
		}
		lim, found := userLimit.Get(authId)
		if found {
			userLim := lim.(*rate.Limiter)
			ok = userLim.AllowN(time.Now(), 1)
			if ok {
				userLimit.Set(authId, userLim, cache.DefaultExpiration)
				c.Next()
			} else {
				c.String(http.StatusTooManyRequests, "")
				c.Abort()
			}
		} else {
			userLim := rate.NewLimiter(50, 50)
			userLim.AllowN(time.Now(), 1)
			userLimit.Set(authId, userLim, cache.DefaultExpiration)
			c.Next()
		}
	}
}

func makeConfig() {
	f, err := os.Create("./config.yaml")
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Printf("close file failed: %v", err)
		}
	}(f)
	if err != nil {
		log.Printf("create file failed: %v", err)
	} else {
		s, err := yaml.Marshal(&Config{})
		if err != nil {
			log.Printf("marshal config failed: %v", err)
			return
		}
		_, err = f.WriteString(string(s))
	}
}

func getConf() (Config, error) {
	yamlFile, err := os.ReadFile("./config.yaml")
	if err != nil {
		return Config{}, err
	}
	var conf Config
	err = yaml.Unmarshal(yamlFile, &conf)
	if err != nil {
		return Config{}, err
	}
	return conf, nil
}

func randStr(length int) string {
	bytes := []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	var result []byte
	rand.Seed(time.Now().UnixNano() + int64(rand.Intn(100)))
	for i := 0; i < length; i++ {
		result = append(result, bytes[rand.Intn(len(bytes))])
	}
	return string(result)
}

func getUUID() string {
	id, err := uuid.NewV4()
	if err != nil {
		log.Printf("get uuid failed: %v", err)
		return getUUID()
	}
	return id.String()
}

func MakeNewVerify(c *gin.Context) {
	verifyCode := randStr(8)
	uid := getUUID()
	_, found := verify.Get(verifyCode)
	for found {
		verifyCode = randStr(8)
		_, found = verify.Get(verifyCode)
	}
	verify.Set(uid, Verify{Verified: false, VerifyCode: verifyCode, User: User{}}, cache.DefaultExpiration)
	verify.Set(verifyCode, false, cache.DefaultExpiration)
	err := insertVerify(c.Request.Header.Get("Authorization"), verifyCode, uid)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": ServerErrorCode, "data": map[string]string{"error": ServerErrorMsg}})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": SuccessCode,
		"data": gin.H{
			"verifyMsg": verifyCode,
			"queryId":   uid,
			"roomId":    conf.RoomId,
			"roomUrl":   "https://live.bilibili.com/" + strconv.Itoa(conf.RoomId),
			"text":      "请打开链接: https://live.bilibili.com/" + strconv.Itoa(conf.RoomId) + " , 在直播间内发送弹幕：" + verifyCode,
		},
	})

}

func QueryVerify(c *gin.Context) {
	verifyQueryId := c.PostForm("queryId")
	resp := gin.H{
		"code": SuccessCode,
		"data": RespData{},
	}
	if verifyQueryId == "" {
		resp["code"] = VerifyCodeEmpty
		resp["data"] = RespData{Error: VerifyCodeEmptyMsg}
		c.JSON(http.StatusOK, resp)
		return
	}
	msg, found := verify.Get(verifyQueryId)
	if !found {
		resp["code"] = VerifyCodeNotFound
		resp["data"] = RespData{Error: VerifyCodeNotFoundMsg, QueryId: verifyQueryId}

	} else {
		if msg.(Verify).Verified {
			resp["data"] = RespData{UserInfo: msg.(Verify).User, QueryId: verifyQueryId}
		} else {
			code, found := verify.Get(msg.(Verify).VerifyCode)
			if found {
				switch code.(type) {
				case bool:
					if code.(bool) {
						resp["data"] = RespData{UserInfo: msg.(Verify).User, QueryId: verifyQueryId}
					} else {
						resp["code"] = VerifyCodeNotUsed
						resp["data"] = RespData{Error: VerifyCodeNotUsedMsg, QueryId: verifyQueryId}
					}
				case User:
					resp["data"] = RespData{UserInfo: code.(User), QueryId: verifyQueryId}
					verify.Set(verifyQueryId, Verify{Verified: true, VerifyCode: msg.(Verify).VerifyCode, User: code.(User)}, cache.DefaultExpiration)
				}
			} else {
				resp["code"] = VerifyCodeNotUsed
				resp["data"] = RespData{Error: VerifyCodeNotUsedMsg, QueryId: verifyQueryId}
			}
		}
	}
	c.JSON(http.StatusOK, resp)
}

func HandleDanmuMsg(event handler.MsgEvent) {
	msg, found := verify.Get(event.DanMuMsg.Data.Content)
	if found {
		switch msg.(type) {
		case bool:
			if msg == false {
				UserInfo := User{Uid: event.DanMuMsg.Data.Sender.Uid, Name: event.DanMuMsg.Data.Sender.Name, Medal: event.DanMuMsg.Data.Medal.MedalName}
				verify.Set(event.DanMuMsg.Data.Content, UserInfo, cache.DefaultExpiration)
				_ = setBiliUid(event.DanMuMsg.Data.Content, UserInfo.Uid)
			}
		}
	}
}
