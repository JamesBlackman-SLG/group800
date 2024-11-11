package internals

import (
	"database/sql"

	"github.com/gin-gonic/gin"
)

type Config struct {
	Router *gin.Engine
	DB     *sql.DB
}

func (app *Config) Routes() {
	// views
	app.Router.GET("/", app.indexPageHandler())
	app.Router.GET("/:d", app.indexPageHandler())
	app.Router.GET("/logo", app.imagesHandler())
}
