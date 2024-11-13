package internals

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"

	"github.com/gin-gonic/gin"
)

type Config struct {
	Router *gin.Engine
	DB     *sql.DB
}

func (app *Config) signInHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Dummy authentication logic
		username := c.PostForm("username")
		password := c.PostForm("password")

		if username == "admin" && password == "slg2024" {
			log.Print("Login successful")
			// Set a session to indicate successful login
			session := sessions.Default(c)
			session.Set("authenticated", true)
			_ = session.Save()

			c.Redirect(http.StatusFound, "/")
		} else {
			log.Print("Login failed")
			c.Redirect(http.StatusFound, "/login?login=failed")
			// c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid credentials"})
		}
	}
}

func (app *Config) signOutHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("authenticated", false)
		_ = session.Save()
		c.Redirect(http.StatusFound, "/login")
	}
}

func (app *Config) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if the user is authenticated
		session := sessions.Default(c)
		if authenticated := session.Get("authenticated"); authenticated == nil || !authenticated.(bool) {
			c.Redirect(http.StatusFound, "/login")
			return
		}
		c.Next()
	}
}

func (app *Config) Routes() {
	// Initialize session middleware
	store := cookie.NewStore([]byte("secret"))
	app.Router.Use(sessions.Sessions("mysession", store))

	// login
	app.Router.GET("/login", app.loginPageHandler())
	app.Router.POST("/signin", app.signInHandler())
	app.Router.GET("/signout", app.signOutHandler())
	app.Router.GET("/logo", app.logoHandler())
	app.Router.GET("/favicon/:f", app.faviconHandler())
	app.Router.GET("/manifest.json", app.manifestJson())
	app.Router.GET("/styles.css", app.handleStyles())

	// Apply authentication middleware
	app.Router.Use(app.authMiddleware())

	// views
	// app.Router.GET("/login", app.loginPageHandler())
	app.Router.GET("/", app.indexPageHandler())
	app.Router.GET("/:d", app.indexPageHandler())
}
