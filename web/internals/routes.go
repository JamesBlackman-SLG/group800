package internals

import (
	"database/sql"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"

	"github.com/gin-gonic/gin"
)

type Config struct {
	Router *gin.Engine
	DB     *sql.DB
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

	// Add logging middleware to debug static file serving
	app.Router.Use(gin.Logger())

	// Serve static files from the /views/static directory
	app.Router.Static("/static", "./views/static")
	app.Router.POST("/webhook", app.webHookHandler())

	// login
	app.Router.GET("/login", app.loginPageHandler())
	app.Router.POST("/signin", app.signInHandler())
	app.Router.GET("/signout", app.signOutHandler())

	// Apply authentication middleware
	app.Router.Use(app.authMiddleware())

	// views
	// app.Router.GET("/login", app.loginPageHandler())
	app.Router.GET("/", app.indexPageHandler())
	app.Router.GET("/:d", app.indexPageHandler())

	app.Router.GET("/timesheet/:d/:u", app.timeSheetPageHandler())
	app.Router.GET("/users", app.usersPageHandler())
}
