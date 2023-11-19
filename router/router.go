package router

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/doubleunion/accesscontrol/door"
	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func RunRouter() {
	signingKey := os.Getenv("ACCESS_CONTROL_SIGNING_KEY")
	if signingKey == "" {
		log.Fatal("signing key is missing")
	}

	door := door.New()

	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	e.GET("/api/v1/open", func(c echo.Context) error {
		token, ok := c.Get("user").(*jwt.Token)
		if !ok {
			return c.JSON(http.StatusBadRequest, map[string]string{"message": "error getting request token"})
		}

		subject, err := token.Claims.GetSubject()
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"message": "error reading subject from token"})
		}

		door.UnlockForDuration(30*time.Second, subject)
		return c.JSON(http.StatusOK, map[string]string{"message": "access granted"})
	}, echojwt.WithConfig(echojwt.Config{
		SigningKey: []byte(signingKey),
	}))

	e.Logger.Fatal(e.Start(":8080"))
}
