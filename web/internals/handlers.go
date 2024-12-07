package internals

import (
	"context"
	"database/sql"
	"fmt"
	"group800_web/views"
	"group800_web/webhook"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const appTimeout = time.Second * 10

func renderTemplate(ctx *gin.Context, status int, template templ.Component) error {
	ctx.Status(status)
	return template.Render(ctx.Request.Context(), ctx.Writer)
}

func (app *Config) editUserHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := ctx.PostForm("userID")
		newTrade := ctx.PostForm("trade")

		// Update the user's trade in the database
		err := app.updateUserTrade(app.DB, userID, newTrade)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, err.Error())
			return
		}

		// Redirect back to the user form page
		ctx.Redirect(http.StatusFound, "/")
	}
}

func (app *Config) updateUserTrade(db *sql.DB, userID, trade string) error {
	query := `
	UPDATE workers
	SET trade = ?
	WHERE time_moto_user_id = ?;
	`
	_, err := db.ExecContext(context.Background(), query, trade, userID)
	if err != nil {
		return fmt.Errorf("failed to update user trade: %w", err)
	}
	return nil
}

func (app *Config) userFormHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := ctx.Param("userID")

		// Fetch user details from the database
		user, err := app.getUserDetails(app.DB, userID)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, err.Error())
			return
		}

		// Render the user form template
		err = renderTemplate(ctx, http.StatusOK, views.UserForm(user))
		if err != nil {
			ctx.JSON(http.StatusBadRequest, err.Error())
			return
		}
	}
}

func (app *Config) getUserDetails(db *sql.DB, userID string) (*views.User, error) {
	query := `
	SELECT time_moto_user_id, first_name, last_name, IFNULL(trade, '?') AS trade
	FROM workers
	WHERE time_moto_user_id = ?;
	`
	row := db.QueryRowContext(context.Background(), query, userID)

	var user views.User
	err := row.Scan(&user.UserID, &user.FirstName, &user.LastName, &user.Trade)
	if err == sql.ErrNoRows {
		log.Println("Worker not found - creating worker record")
		// If no rows are found, insert a new record with NULL for trade
		insertQuery := `
		INSERT INTO workers (time_moto_user_id, first_name, last_name)
    SELECT user_id, user_first_name, user_last_name
    FROM webhooks
    WHERE user_id = ?
    LIMIT 1;
		`
		_, insertErr := db.ExecContext(context.Background(), insertQuery, userID, "")
		if insertErr != nil {
			return nil, fmt.Errorf("failed to insert new user: %w", insertErr)
		}

		query := `
	SELECT time_moto_user_id, first_name, last_name, IFNULL(trade, '?') AS trade
	FROM workers
	WHERE time_moto_user_id = ?;
	`
		rowNew := db.QueryRowContext(context.Background(), query, userID)

		err := rowNew.Scan(&user.UserID, &user.FirstName, &user.LastName, &user.Trade)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch user details: %w", err)
		}
		return &user, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to fetch user details: %w", err)
	}
	return &user, nil
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

func (app *Config) webHookHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		signature := strings.TrimSpace(ctx.GetHeader("Timemoto-Signature"))
		if signature == "" {
			ctx.JSON(http.StatusBadRequest, "Missing signature")
			return
		}
		bodyBytes, err := io.ReadAll(ctx.Request.Body)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, "Failed to read request body")
			return
		}
		body := string(bodyBytes)
		err = webhook.HandlePost(app.DB, body)
		if err != nil {
			log.Println(err)
		}
	}
}

func (app *Config) usersPageHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		users, err := app.listUsers(app.DB)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, err.Error())
			return
		}

		err = renderTemplate(ctx, http.StatusOK, views.UserList(users))
		if err != nil {
			ctx.JSON(http.StatusBadRequest, err.Error())
			return
		}
	}
}

func (app *Config) loginPageHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var err error
		_, cancel := context.WithTimeout(context.Background(), appTimeout)
		defer cancel()

		err = renderTemplate(ctx, http.StatusOK, views.LoginPage())
		if err != nil {
			ctx.JSON(http.StatusBadRequest, err.Error())
			return
		}
	}
}

func (app *Config) indexPageHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var (
			d         time.Time
			err       error
			locations []string
			ll        []*views.Location
		)
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

		locations, err = app.fetchDistinctLocations(app.DB, d)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, err.Error())
			return
		}

		// if len(locations) == 0 {
		// 	ctx.JSON(http.StatusBadRequest, "No locations found")
		// 	return
		// }

		for _, t := range locations {
			data, err := dailyCheckInAnalysis(app.DB, t, d)
			if err != nil {
				ctx.JSON(http.StatusBadRequest, err.Error())
				log.Println(err)
				continue
			}

			l := &views.Location{
				Name: t,
				Data: data,
			}
			ll = append(ll, l)
		}

		// if len(ll) == 0 {
		// 	ctx.JSON(http.StatusBadRequest, "No locations found")
		// 	return
		// }

		err = renderTemplate(ctx, http.StatusOK, views.Index(ll, d))
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

		userID := ctx.Param("userID")

		// var data []views.CheckInData
		var weeklyData []*views.WeeklyTimeSheet

		// range over the week
		for i := 0; i < 7; i++ {
			today := d.Add(time.Duration(i) * time.Hour * 24)
			todaysData, err := weeklyTimeSheet(app.DB, userID, today)
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

		users, err := app.listUsers(app.DB)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, err.Error())
			return
		}

		err = renderTemplate(ctx, http.StatusOK, views.TimeSheet(weeklyData, d, userID, users))
		if err != nil {
			ctx.JSON(http.StatusBadRequest, err.Error())
			return
		}
	}
}

func dailyCheckInAnalysis(db *sql.DB, locationName string, d time.Time) ([]views.CheckInData, error) {
	query := `
SELECT 
    ci.user_id,
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
    END AS duration,
    IFNULL(w.trade, '?') AS trade
FROM
    webhooks ci
LEFT JOIN 
    webhooks co
ON 
    ci.user_id = co.user_id 
    AND ci.location_name = co.location_name 
    AND co.clocking_type = 'Out' 
    AND date(datetime(ci.time_logged)) = date(datetime(co.time_logged)) 
    AND datetime(co.time_logged) > datetime(ci.time_logged)
LEFT JOIN workers w ON ci.user_id = w.time_moto_user_id
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
		err := rows.Scan(&row.UserID, &row.Name, &row.CheckIn, &row.CheckOut, &row.Duration, &row.Trade)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		data = append(data, row)
	}

	// Check for errors during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	if len(data) == 0 {
		return nil, err
	}

	// var buf bytes.Buffer
	// writer := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', tabwriter.Debug)

	// // Write the header
	// fmt.Fprintln(writer, "Name\tCheck In\tCheck Out\tDuration")

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

	// // Flush the writer
	// writer.Flush()

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
