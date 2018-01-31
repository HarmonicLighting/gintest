package jwt

import (
	"local/gintest/services/db"
	"local/gintest/services/dbheap"
	"log"
	"sync"
	"time"

	"github.com/appleboy/gin-jwt"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

var jwtMiddleware *jwt.GinJWTMiddleware
var once sync.Once

func GetInstance() *jwt.GinJWTMiddleware {
	once.Do(func() {
		jwtMiddleware = &jwt.GinJWTMiddleware{
			Realm:         "test zone",
			Key:           []byte("secret key"),
			Timeout:       time.Hour,
			MaxRefresh:    time.Hour,
			Authenticator: authenticator,
			/*func(userId string, password string, c *gin.Context) (string, bool) {
				if (userId == "admin" && password == "admin") || (userId == "test" && password == "test") {
					return userId, true
				}

				return userId, false
			}*/

			Authorizator: func(userId string, c *gin.Context) bool {
				log.Println("In authorizator: ", userId)

				return true
			},
			Unauthorized: func(c *gin.Context, code int, message string) {
				log.Println("In unauthorized: ", code, " ", message)
				c.JSON(code, gin.H{
					"code":    code,
					"message": message,
				})
			},
			// TokenLookup is a string in the form of "<source>:<name>" that is used
			// to extract token from the request.
			// Optional. Default value "header:Authorization".
			// Possible values:
			// - "header:<name>"
			// - "query:<name>"
			// - "cookie:<name>"
			TokenLookup: "header:Authorization",
			// TokenLookup: "query:token",
			// TokenLookup: "cookie:token",

			// TokenHeadName is a string in the header. Default value is "Bearer"
			TokenHeadName: "Bearer",

			// TimeFunc provides the current time. You can override it to use another time value. This is useful for testing or if your server uses a different time zone than your tokens.
			TimeFunc: time.Now,
		}
	})
	return jwtMiddleware
}

func authenticator(userId string, password string, c *gin.Context) (string, bool) {
	log.Println("Inside authenticator: ", userId, " ", password)
	session, _ := dbheap.GetSession()
	defer session.Close()
	userStruct := db.DBUser{}
	err := session.ClientSession.GetUser(userId, &userStruct)
	if err != nil {
		log.Println("Error on authenticator: ", err)
		return userId, false
	}
	err = bcrypt.CompareHashAndPassword([]byte(userStruct.HashedPassword), []byte(password))
	if err != nil {
		log.Println("Error on authenticator comparing hash and password: ", err)
		return userId, false
	}
	log.Println("Authentication succeded")
	return userId, true
}

func HelloHandler(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	c.JSON(200, gin.H{
		"userID": claims["id"],
		"text":   "Hello World.",
	})
}
