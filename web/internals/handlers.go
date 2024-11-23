package internals

import (
	"context"
	"database/sql"
	"fmt"
	"group800_web/views"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

const appTimeout = time.Second * 10

func render(ctx *gin.Context, status int, template templ.Component) error {
	ctx.Status(status)
	return template.Render(ctx.Request.Context(), ctx.Writer)
}

func (app *Config) loginPageHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var err error
		_, cancel := context.WithTimeout(context.Background(), appTimeout)
		defer cancel()

		err = render(ctx, http.StatusOK, views.LoginPage())
		if err != nil {
			ctx.JSON(http.StatusBadRequest, err.Error())
			return
		}
	}
}

func (app *Config) indexPageHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var d time.Time
		var err error
		_, cancel := context.WithTimeout(context.Background(), appTimeout)
		defer cancel()
		dateParam := ctx.Param("d")

		if dateParam != "" {
			d, err = time.Parse("2006-01-02", dateParam)
			if err != nil {
				ctx.JSON(http.StatusBadRequest, err.Error())
				return
			}
		} else {
			d = time.Now()
		}
		// d = d.AddDate(0, 0, -3)

		locations, err := app.fetchDistinctLocations(app.DB, d)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, err.Error())
			return
		}

		var ll []*views.Location
		for _, t := range locations {

			data, err := dailyCheckInAnalysis(app.DB, t, d)
			if err != nil {
				ctx.JSON(http.StatusBadRequest, err.Error())
				log.Println(err)
			}

			l := &views.Location{
				Name: t,
				Data: data,
			}
			ll = append(ll, l)

		}

		err = render(ctx, http.StatusOK, views.Index(ll, d))
		if err != nil {
			ctx.JSON(http.StatusBadRequest, err.Error())
			return
		}
	}
}

func (app *Config) timeSheetPageHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var d time.Time
		var err error
		_, cancel := context.WithTimeout(context.Background(), appTimeout)
		defer cancel()
		dateParam := ctx.Param("d")

		if dateParam != "" {
			d, err = time.Parse("2006-01-02", dateParam)
			if err != nil {
				ctx.JSON(http.StatusBadRequest, err.Error())
				return
			}
		} else {
			d = time.Now()
		}

		userFullName := ctx.Param("u")

		// var data []views.CheckInData
		var weeklyData []*views.WeeklyTimeSheet

		// range over the week
		for i := 0; i < 7; i++ {
			today := d.Add(time.Duration(i) * time.Hour * 24)
			todaysData, err := weeklyTimeSheet(app.DB, userFullName, today)
			if err != nil {
				log.Println(err)
				ctx.JSON(http.StatusBadRequest, err.Error())
			}

			ww := views.WeeklyTimeSheet{
				Date: today,
				Data: todaysData,
			}
			weeklyData = append(weeklyData, &ww)
			// fmt.Println(d.Format("2006-01-02"))
			//
			// //	Create a new tab writer
			// var buf bytes.Buffer
			// writer := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', tabwriter.Debug)
			//
			// // Write the header
			// fmt.Fprintln(writer, "Date\tName\tLocation\tCheck In\tCheck Out\tDuration")
			//
			// // Write the data
			// for _, row := range todaysData {
			// 	name := row.Name
			// 	if len(name) > 24 {
			// 		name = name[:24] // Truncate if longer than 20 characters
			// 	} else {
			// 		name = fmt.Sprintf("%-24s", name) // Left-align and pad with spaces to 20 characters
			// 	}
			// 	dayOfWeek := today.Weekday().String()
			// 	fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\n", dayOfWeek, name, row.Location, row.CheckIn, row.CheckOut, row.Duration)
			// }
			//
			// // Flush the writer
			// writer.Flush()
			//
			// // Print the tabulated data
			// fmt.Println(buf.String())
			// data = append(data, todaysData...)
		}

		// ctx.String(200, "message")

		err = render(ctx, http.StatusOK, views.TimeSheet(weeklyData, d, userFullName))
		if err != nil {
			ctx.JSON(http.StatusBadRequest, err.Error())
			return
		}
	}
}

func (app *Config) logoHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		imagePath := filepath.Join("views", "images", "logo.png")
		c.File(imagePath)
	}
}

func (app *Config) handleStyles() gin.HandlerFunc {
	return func(c *gin.Context) {
		imagePath := filepath.Join("views", "images", "styles.css")
		c.File(imagePath)
	}
}

func (app *Config) faviconHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		faviconParam := c.Param("f")
		fmt.Println(faviconParam)
		if faviconParam == "" {
			faviconParam = "default" // Use "default" if no parameter is provided
		}
		faviconPath := filepath.Join("views", "images", "favicon", faviconParam)
		c.File(faviconPath)
	}
}

func (app *Config) manifestJson() gin.HandlerFunc {
	return func(c *gin.Context) {
		docPath := filepath.Join("views", "images", "manifest.json")
		c.File(docPath)
	}
}

func dailyCheckInAnalysis(db *sql.DB, locationName string, d time.Time) ([]views.CheckInData, error) {
	query := `
SELECT 
    ci.user_full_name,
    strftime('%H:%M', ci.time_logged) AS check_in,
    CASE 
        WHEN co.time_logged IS NOT NULL THEN strftime('%H:%M', co.time_logged)
        ELSE ''
    END AS check_out_time,
    CASE 
        WHEN co.time_logged IS NOT NULL THEN 
            strftime('%H:%M', (julianday(co.time_logged) - julianday(ci.time_logged)) * 86400, 'unixepoch')
        ELSE 
          ''
    END AS duration
FROM
    webhooks ci
LEFT JOIN 
    webhooks co 
ON 
    ci.user_id = co.user_id 
    AND ci.location_name = co.location_name 
    AND ci.clocking_type = 'In' 
    AND co.clocking_type = 'Out' 
    AND date(datetime(ci.time_logged)) = date(datetime(co.time_logged)) 
    AND datetime(co.time_logged) > datetime(ci.time_logged)
WHERE 
    ci.clocking_type = 'In'
    AND date(datetime(ci.time_logged)) = ?
    AND (co.time_logged IS NULL OR date(datetime(co.time_logged)) = ? )
    AND ci.location_name = ?
    -- AND (co.location_name IS NULL OR co.location_name = ci.location_name)
    -- AND (co.time_logged IS NULL OR (time(datetime(ci.time_logged)) >= '08:05:00' OR time(datetime(co.time_logged)) <= '16:55:00'))
GROUP BY 
    ci.user_id, ci.time_logged
HAVING check_out_time = '' OR duration > '00:00'
ORDER BY 
    ci.user_full_name, check_in;
`
	// Prepare the query
	rows, err := db.QueryContext(context.Background(), query, d.Format("2006-01-02"), d.Format("2006-01-02"), locationName)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	var data []views.CheckInData

	// Iterate over the rows
	for rows.Next() {
		var row views.CheckInData
		err := rows.Scan(&row.Name, &row.CheckIn, &row.CheckOut, &row.Duration)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		data = append(data, row)
	}

	// Check for errors during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	// if len(data) == 0 {
	// 	return "No data for today", nil
	// }

	// Create a new tab writer
	// var buf bytes.Buffer
	// writer := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', tabwriter.Debug)
	//
	// // Write the header
	// fmt.Fprintln(writer, "Name\tCheck In\tCheck Out\tDuration")
	//
	// // Write the data
	// for _, row := range data {
	// 	name := row.Name
	// 	if len(name) > 24 {
	// 		name = name[:24] // Truncate if longer than 20 characters
	// 	} else {
	// 		name = fmt.Sprintf("%-24s", name) // Left-align and pad with spaces to 20 characters
	// 	}
	// 	fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", name, row.CheckIn, row.CheckOut, row.Duration)
	// }
	//
	// // Flush the writer
	// writer.Flush()
	//
	// // Print the tabulated data
	// fmt.Println(buf.String())

	// Convert data to JSON for OpenAI
	// dataJSON, err := json.Marshal(data)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to marshal data to JSON: %w", err)
	// }

	// Convert JSON data to a string for easy submission
	return data, nil
}
