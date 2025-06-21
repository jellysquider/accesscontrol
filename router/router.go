package router

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/doubleunion/accesscontrol/door"
	"github.com/doubleunion/accesscontrol/requests"
	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/crypto/acme/autocert"
)

const serviceFilePath = "/etc/systemd/system/accesscontrol.service"
const ipQueryURL = "https://wtfismyip.com/text"

var localInternetAddress = os.Getenv("LOCAL_INTERNET_ADDRESS")

func RunRouter() {
	signingKey := os.Getenv("ACCESS_CONTROL_SIGNING_KEY")
	if signingKey == "" {
		log.Fatal("signing key is missing")
	}

	if localInternetAddress == "" {
		log.Fatal("local internet address is missing")
	}

	door := door.New()

	e := echo.New()
	e.HideBanner = true
	e.AutoTLSManager.Cache = autocert.DirCache("/var/www/.cache")

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	e.GET("/", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	e.HEAD("/", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	e.GET("/api/v1/status", func(c echo.Context) error {
		return jsonResponse(c, http.StatusOK, "door control available")
	}, requireLocalNetworkMiddleware)

	e.POST("/api/v1/unlock", func(c echo.Context) error {
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return jsonResponse(c, http.StatusBadRequest, "error reading request body")
		}

		var request requests.UnlockRequest
		err = json.Unmarshal(body, &request)
		if err != nil {
			return jsonResponse(c, http.StatusBadRequest, "error unmarshaling request body")
		}

		token, ok := c.Get("user").(*jwt.Token)
		if !ok {
			return jsonResponse(c, http.StatusBadRequest, "error getting request token")
		}

		subject, err := token.Claims.GetSubject()
		if err != nil {
			return jsonResponse(c, http.StatusBadRequest, "error reading subject from token")
		}

		err = door.UnlockForDuration(time.Duration(request.Seconds)*time.Second, subject)
		if err != nil {
			return jsonResponse(c, http.StatusBadRequest, fmt.Sprintf("%+v", err))
		}

		return jsonResponse(c, http.StatusOK, "access granted")
	}, requireLocalNetworkMiddleware, echojwt.WithConfig(echojwt.Config{
		SigningKey: []byte(signingKey),
	}))

	e.Logger.Fatal(e.StartAutoTLS(":8443"))
}

func jsonResponse(c echo.Context, code int, message string) error {
	return c.JSON(code, map[string]string{"message": message})
}

func requireLocalNetworkMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Get the remote address from the request
		remoteAddr := c.Request().RemoteAddr

		// Parse the IP address
		ip, _, err := net.SplitHostPort(remoteAddr)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to parse IP address")
		}

		if ip != localInternetAddress {
			return jsonResponse(c, http.StatusForbidden, "requests not allowed from remote hosts")
		}

		// Continue to the next middleware or route handler
		return next(c)
	}
}

func UpdateIPAndRestart() error {
	// Step 1: Query current IP address
	resp, err := http.Get(ipQueryURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	ipBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	currentIP := strings.TrimSpace(string(ipBytes))

	// Step 2: Read the service file
	content, err := os.ReadFile(serviceFilePath)
	if err != nil {
		return err
	}

	// Step 3: Check if the IP matches
	lines := strings.Split(string(content), "\n")
	var updatedContent []string
	ipUpdated := false
	for _, line := range lines {
		if strings.HasPrefix(line, "Environment=LOCAL_INTERNET_ADDRESS=") {
			fileIP := strings.TrimPrefix(line, "Environment=LOCAL_INTERNET_ADDRESS=")
			if fileIP != currentIP {
				line = "Environment=LOCAL_INTERNET_ADDRESS=" + currentIP
				ipUpdated = true
			}
		}
		updatedContent = append(updatedContent, line)
	}

	// Step 4: Update the file if necessary
	if ipUpdated {
		// first we have to output the new contents to a temporary file
		// because we don't have access to the service file directly
		tempFilePath := "/tmp/accesscontrol.service"
		err = os.WriteFile(tempFilePath, []byte(strings.Join(updatedContent, "\n")), 0644)
		if err != nil {
			return err
		}

		// then we copy the temporary file to the service file path
		// the path is owned by the process user so this is allowed by the OS without sudo
		cmd := exec.Command("cp", tempFilePath, serviceFilePath)
		err = cmd.Run()
		if err != nil {
			return err
		}

		// Step 5: Restart the Raspberry Pi
		cmd = exec.Command("sudo", "shutdown", "-r", "now")
		//log.Printf("Error in updateIPAndRestart: %v", err)
		err = cmd.Run()
		if err != nil {
			return err
		}
	}

	return nil
}
