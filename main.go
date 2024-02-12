package main

import (
	"encoding/json"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// map cognito state with twitter state
var stateCache = map[string]string{}

// map generated state with code challenge
var stateCodeChallenge = map[string]string{}

// map code with code challenge
var codeCodeChallenge = map[string]string{}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func main() {

	godotenv.Load()

	cognitoUrl := os.Getenv("COGNITO_AUTH_URL")
	callbackURL := os.Getenv("CALLBACK_URL")

	engine := gin.Default()

	engine.GET("/authorize", func(c *gin.Context) {
		query := c.Request.URL.Query()
		twitterUrl := "https://twitter.com/i/oauth2/authorize"

		// replace redirect url with the callback url
		query.Del("redirect_uri")
		query.Set("redirect_uri", callbackURL)
		query.Del("scope")
		query.Set("scope", "users.read offline.access tweet.read")

		codeChallenge := RandStringRunes(30)
		query.Set("code_challenge", codeChallenge)
		query.Set("code_challenge_method", "plain")

		state := query.Get("state")
		// generate random state

		randomState := RandStringRunes(10)
		stateCache[randomState] = state
		stateCodeChallenge[randomState] = codeChallenge

		query.Del("state")
		query.Set("state", randomState)

		// redirect to twitter url with the query params
		c.Redirect(http.StatusFound, twitterUrl+"?"+query.Encode())
	})

	engine.GET("/callback", func(c *gin.Context) {
		query := c.Request.URL.Query()
		state := query.Get("state")
		query.Del("state")
		query.Del("client_secret")

		code := query.Get("code")
		codeChallenge := stateCodeChallenge[state]
		codeCodeChallenge[code] = codeChallenge

		oldState, ok := stateCache[state]
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "state not found"})
			return
		}
		query.Add("state", oldState)

		c.Redirect(http.StatusFound, cognitoUrl+"?"+query.Encode())
	})

	engine.POST("/token", func(c *gin.Context) {
		// parse form params
		if err := c.Request.ParseForm(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid form"})
			return
		}

		// get the form params
		form := c.Request.PostForm

		form.Del("redirect_uri")
		form.Set("redirect_uri", callbackURL)
		form.Del("client_secret")

		code := form.Get("code")
		codeVerifier := codeCodeChallenge[code]

		form.Set("code_verifier", codeVerifier)

		// make a request to twitter to get the access token
		body := form.Encode()
		log.Println(body)
		strReader := strings.NewReader(body)

		request, err := http.NewRequest("POST", "https://api.twitter.com/2/oauth2/token", strReader)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		client := &http.Client{}
		response, err := client.Do(request)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid response"})
			return
		}

		defer response.Body.Close()

		// return response as is
		c.Header("Content-Type", "application/json")
		c.Status(response.StatusCode)
		io.Copy(c.Writer, response.Body)
	})

	engine.GET("/userinfo", func(c *gin.Context) {
		// get token from the request
		token := c.Request.Header.Get("Authorization")
		if token == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token"})
			return
		}

		// make a request to twitter to get the user info
		request, err := http.NewRequest("GET", "https://api.twitter.com/2/users/me", nil)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		request.Header.Set("Authorization", token)

		client := &http.Client{}
		response, err := client.Do(request)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid response"})
			return
		}

		defer response.Body.Close()

		// read body as map[string]interface{}
		var body map[string]interface{}
		strOp, err := io.ReadAll(response.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid response"})
			return
		}

		if err := json.Unmarshal(strOp, &body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid response"})
			return
		}

		data := body["data"].(map[string]interface{})

		data["sub"] = data["id"]
		data["email"] = "nilkantha.dipesh+twitter-1@gmail.com"
		c.JSON(http.StatusOK, data)
	})

	engine.Run(":8080")
}
