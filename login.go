package main

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type GithubAuth struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
}

type GitHubUserInfo struct {
	Login             string    `json:"login"`
	ID                int       `json:"id"`
	NodeID            string    `json:"node_id"`
	AvatarURL         string    `json:"avatar_url"`
	GravatarID        string    `json:"gravatar_id"`
	URL               string    `json:"url"`
	HTMLURL           string    `json:"html_url"`
	FollowersURL      string    `json:"followers_url"`
	FollowingURL      string    `json:"following_url"`
	GistsURL          string    `json:"gists_url"`
	StarredURL        string    `json:"starred_url"`
	SubscriptionsURL  string    `json:"subscriptions_url"`
	OrganizationsURL  string    `json:"organizations_url"`
	ReposURL          string    `json:"repos_url"`
	EventsURL         string    `json:"events_url"`
	ReceivedEventsURL string    `json:"received_events_url"`
	Type              string    `json:"type"`
	SiteAdmin         bool      `json:"site_admin"`
	Name              string    `json:"name"`
	Company           string    `json:"company"`
	Blog              string    `json:"blog"`
	Location          string    `json:"location"`
	Email             string    `json:"email"`
	Hireable          bool      `json:"hireable"`
	Bio               string    `json:"bio"`
	TwitterUsername   string    `json:"twitter_username"`
	PublicRepos       int       `json:"public_repos"`
	PublicGists       int       `json:"public_gists"`
	Followers         int       `json:"followers"`
	Following         int       `json:"following"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func getReq(data url.Values, getUrl string, auth string) ([]byte, error) {
	u, err := url.ParseRequestURI(getUrl)
	if err != nil {
		log.Printf("Error orrured when parsing the url: %v", err)
		return nil, err
	}
	u.RawQuery = data.Encode()
	client := http.Client{}
	req, err := http.NewRequest("GET", u.String(), nil)
	req.Header = http.Header{
		"Accept":        {"application/json"},
		"Authorization": {"Bearer " + auth},
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error orrured when sending the request: %v", err)
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			log.Printf("Error orrured when closing the session: %v", err)
		}
	}(resp.Body)
	if resp.StatusCode != 200 {
		log.Printf("Status code is not 200: %v", resp.StatusCode)
		return nil, errors.New("status code error: " + strconv.Itoa(resp.StatusCode) + " " + resp.Status)
	}
	s, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error orrured when reading the response: %v", err)
		return nil, err
	}
	return s, nil
}

func LoginGithub(c *gin.Context) {
	clientID := conf.ClientId
	redirectURL := conf.BaseUrl + "/login/redirect"
	c.Redirect(http.StatusFound, "http://github.com/login/oauth/authorize?client_id="+clientID+"&redirect_uri="+redirectURL)
}

func getUserInfo(authToken string) (GitHubUserInfo, error) {
	data := url.Values{}
	getUrl := "https://api.github.com/user"
	s, err := getReq(data, getUrl, authToken)
	if err != nil {
		log.Printf("Error orrured when getting the response: %v", err)
		return GitHubUserInfo{}, err
	}
	var userInfo GitHubUserInfo
	err = json.Unmarshal(s, &userInfo)
	if err != nil {
		log.Printf("Error orrured when unmarshalling the response: %v", err)
		return GitHubUserInfo{}, err
	}
	return userInfo, nil
}

func regNewUser(auth GithubAuth, info GitHubUserInfo) (string, error) {
	var user SQLUser
	user.Name = info.Login
	user.UUID = getUUID()
	user.AccessToken = auth.AccessToken
	user.NodeId = info.NodeID
	user.Email = info.Email
	err := insertUser(user)
	if err != nil {
		log.Printf("Error orrured when inserting the user: %v", err)
		return "", err
	}
	return user.UUID, nil
}

func RedirectGithub(c *gin.Context) {
	code := c.Query("code")
	clientId := conf.ClientId
	clientSecret := conf.ClientSecret
	getUrl := "https://github.com/login/oauth/access_token"
	data := url.Values{}
	data.Set("client_id", clientId)
	data.Set("client_secret", clientSecret)
	data.Set("code", code)
	s, err := getReq(data, getUrl, "")
	if err != nil {
		log.Printf("Error orrured when getting the response: %v", err)
		c.JSONP(http.StatusInternalServerError, gin.H{"code": ServerErrorCode, "data": map[string]string{"error": ServerErrorMsg}})
		return
	}
	var auth GithubAuth
	err = json.Unmarshal(s, &auth)
	if err != nil {
		log.Printf("Error orrured when unmarshalling the response: %v", err)
		c.JSONP(http.StatusInternalServerError, gin.H{"code": ServerErrorCode, "data": map[string]string{"error": ServerErrorMsg}})
		return
	}
	info, err := getUserInfo(auth.AccessToken)
	if err != nil {
		log.Printf("Error orrured when getting the user info: %v", err)
		c.JSONP(http.StatusInternalServerError, gin.H{"code": ServerErrorCode, "data": map[string]string{"error": ServerErrorMsg}})
		return
	}
	rawUser, err := getUserByNodeID(info.NodeID)
	if err != nil {
		log.Printf("Error orrured when getting the user: %v", err)
		c.JSONP(http.StatusInternalServerError, gin.H{"code": ServerErrorCode, "data": map[string]string{"error": ServerErrorMsg}})
		return
	}
	if rawUser.UUID != "" {
		c.JSONP(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"uuid":  rawUser.UUID,
				"name":  rawUser.Name,
				"email": rawUser.Email,
			},
		})
		return
	}
	uuid, err := regNewUser(auth, info)
	if err != nil {
		log.Printf("Error orrured when registering the user: %v", err)
		c.JSONP(http.StatusInternalServerError, gin.H{"code": ServerErrorCode, "data": map[string]string{"error": ServerErrorMsg}})
		return
	}
	c.JSONP(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"uuid":  uuid,
			"name":  info.Login,
			"email": info.Email,
		},
	})

}
