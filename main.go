package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/gookit/rux"
	"github.com/hashicorp/consul/api"
	"github.com/r35krag0th/win-loss-rux/version"
	"github.com/sirupsen/logrus"
)

//go:embed templates/counter.gohtml
var embedCounterTemplate string

//go:embed templates/index.gohtml
var embedIndexTemplate string

//go:embed templates/solo_counter.gohtml
var embedSoloCounterTemplate string

func init() {
	logrus.SetFormatter(&logrus.TextFormatter{})
	logrus.SetReportCaller(false)
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)
}

func handleCounter(ctx context.Context, name string) *WinLossCounter {
	logger := logrus.WithFields(logrus.Fields{
		"counter_name": name,
		"version":      version.Version,
	})

	span := sentry.StartSpan(ctx, "Initialize WinLossCounter")
	logger.Debug("Creating new WinLossCounter")
	tmp := NewWinLossCounter(name)
	span.Finish()

	consulScheme := os.Getenv("CONSUL_SCHEME")
	logger.Debugf("Fetched CONSUL_SCHEME environment variable: %s", consulScheme)

	consulAddress := os.Getenv("CONSUL_ADDR")
	logger.Debugf("Fetched CONSUL_ADDR environment variable: %s", consulAddress)

	targetConfig := api.DefaultConfig()

	if consulAddress != "" && consulScheme != "" {
		logger.Info("Using environment variables to configure Consul client")
		targetConfig = &api.Config{
			Address: consulAddress,
			Scheme:  consulScheme,
		}
	}

	span = sentry.StartSpan(ctx, "Initialize Consul Client")
	logger.Debug("Creating Consul client")
	consulClient, err := api.NewClient(targetConfig)
	if err != nil {
		logger.Error("Failed to create Consul client", err)
		sentry.CaptureException(err)
	}
	span.Finish()

	logger.Debug("Setting Consul Client in WinLossCounter")
	tmp.SetConsulClient(consulClient)

	span = sentry.StartSpan(ctx, "Load data from Consul")
	defer span.Finish()

	logger.Debug("Loading counter's data")
	tmp.Load()

	logger.Debug("Returning initialized counter")
	return tmp
}

func main() {
	rootLogger := logrus.WithFields(logrus.Fields{
		"version":  version.Version,
		"env_name": envName,
	})

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              os.Getenv("SENTRY_DSN"),
		EnableTracing:    true,
		TracesSampleRate: 0.01,
		Environment:      getenv("APP_ENV", "local"),
		Release:          version.Version,
	})
	if err != nil {
		rootLogger.Fatalf("sentry.Init: %s", err)
	}
	// Flush buffered events before the program terminates.
	defer sentry.Flush(2 * time.Second)

	sentryHandler := sentryhttp.New(sentryhttp.Options{})

	r := rux.New()
	// r.Use(func(c *rux.Context) {
	// 	sentryHandler.Handle(c.Handler())
	// 	c.Next()
	// })

	r.Use(rux.WrapHTTPHandlerFunc(sentryHandler.HandleFunc(func(rw http.ResponseWriter, r *http.Request) {
		hub := sentry.GetHubFromContext(r.Context())
		hub.Scope().SetTag("someRandomTag", "maybeYouNeedIt")
		hub.Scope().SetRequest(r)
	})))

	r.GET("", func(c *rux.Context) {
		logger := rootLogger.WithFields(logrus.Fields{
			"path": "/",
		})

		if hub := sentry.GetHubFromContext(c.Req.Context()); hub != nil {
			hub.Scope().SetTransaction("Frontend - Index")
		}

		logger.Debug("Creating blank Counter as helper")
		counter := handleCounter(c.Req.Context(), "")
		// counter.SetConsulClient(consulClient)

		logger.Debug("Listing all counters")
		counterNames := counter.ListAll()

		logger.WithFields(logrus.Fields{
			"template": "template/index.gohtml",
		}).Debug("Initializing template for index page")
		// tmpl := template.Must(template.ParseFiles("templates/index.gohtml"))
		tmpl, err := template.New("index").Parse(embedIndexTemplate)
		if err != nil {
			logger.WithError(err).Error("Failed to load template from embed")
			c.AbortWithStatus(500, "Something bad happened")
		}

		logger.Debug("Rendering template")
		data := IndexTemplateData{Title: "WLD - List of Counters"}

		logger.Debug("Adding discovered counters to template data")
		for _, counterName := range counterNames {
			logger.Debugf("Adding Counter: %s", counterName)
			data.AddCounter(
				fmt.Sprintf("/counters/%s", counterName),
				counterName,
			)
		}
		out := bytes.Buffer{}
		logger.Info("Rendering template with data")
		err = tmpl.Execute(&out, data)
		if err != nil {
			logger.WithError(err).Error("Something failed during execute")
			c.AbortWithStatus(500, "Something bad happened")
			return
		}

		logger.Info("Sending response for index.gohtml template")
		c.HTML(200, out.Bytes())
	})

	r.GET("/counters/{name}", func(c *rux.Context) {
		if hub := sentry.GetHubFromContext(c.Req.Context()); hub != nil {
			hub.Scope().SetTransaction("Frontend - Show Counter")
			hub.Scope().SetExtra("counter_name", c.Param("name"))
		}

		logger := rootLogger.WithFields(logrus.Fields{
			"path": "/counters/{name}",
			"name": c.Param("name"),
		})
		logger.Info("Creating counter")
		counter := handleCounter(c.Req.Context(), c.Param("name"))
		// counter.SetConsulClient(consulClient)

		// tmpl := template.Must(template.ParseFiles("templates/counter.gohtml"))
		tmpl, err := template.New("counter").Parse(embedCounterTemplate)
		if err != nil {
			logger.WithError(err).Error("Failed to load template from embed")
			c.AbortWithStatus(500, "Something bad happened")
			return
		}

		data := NewCounterPageFromWinLossCounter(counter)
		data.Name = counter.Name
		data.Title = fmt.Sprintf("WLD Counter - %s", counter.Name)

		out := bytes.Buffer{}
		err = tmpl.Execute(&out, data)
		if err != nil {
			logrus.WithError(err).Error("failed to render the template")
			c.AbortWithStatus(500, "Something bad happened")
			return
		}
		c.HTML(200, out.Bytes())
	})

	r.GET("/counters/{name}/solo", func(c *rux.Context) {
		if hub := sentry.GetHubFromContext(c.Req.Context()); hub != nil {
			hub.Scope().SetTransaction("Show Counter (Solo Mode)")
			hub.Scope().SetExtra("counter_name", c.Param("name"))
		}
		logger := rootLogger.WithFields(logrus.Fields{
			"path": "/counters/{name}/solo",
			"name": c.Param("name"),
		})
		counter := handleCounter(c.Req.Context(), c.Param("name"))
		// tmpl := template.Must(template.ParseFiles("templates/solo_counter.gohtml"))
		tmpl, err := template.New("soloCounter").Parse(embedSoloCounterTemplate)
		if err != nil {
			logger.WithError(err).Error("Failed to load template from embed")
			c.AbortWithStatus(500, "Something bad happened")
			return
		}

		data := NewCounterPageFromWinLossCounter(counter)
		data.Name = counter.Name
		data.Title = fmt.Sprintf("WLD Counter (Solo) - %s", counter.Name)

		out := bytes.Buffer{}
		err = tmpl.Execute(&out, data)
		if err != nil {
			logger.WithError(err).Error("Something failed during execute")
			c.AbortWithStatus(500, "Something bad happened")
			return
		}
		c.HTML(200, out.Bytes())
	})

	// Use the /api/v1 path as the root
	r.Group("/api/v1", func() {
		apiLogger := rootLogger.WithFields(logrus.Fields{
			"path": "/api/v1",
		})

		// The Counter routes
		r.Group("/counters", func() {
			counterLogger := apiLogger.WithFields(logrus.Fields{
				"path": "/aip/v1/counters",
			})
			// List all Counters
			r.GET("", func(c *rux.Context) {
				if hub := sentry.GetHubFromContext(c.Req.Context()); hub != nil {
					hub.Scope().SetTransaction("API - List Counters")
				}

				counterLogger.WithFields(logrus.Fields{
					"method": "GET",
				}).Info("Handling Index request")
				counter := handleCounter(c.Req.Context(), "")
				// counter.SetConsulClient(consulClient)

				counterNames := counter.ListAll()
				c.JSON(200, counterNames)
			})

			// The "specific" counter routes
			r.Group("/{name}", func() {
				// Get the counter's W/L/D stats and Name
				logger := counterLogger.WithFields(logrus.Fields{
					"path": "/api/v1/counters/{name}",
				})

				r.GET("", func(c *rux.Context) {
					if hub := sentry.GetHubFromContext(c.Req.Context()); hub != nil {
						hub.Scope().SetTransaction("API - Show Counter")
						hub.Scope().SetExtra("counter_name", c.Param("name"))
					}

					logger.WithFields(logrus.Fields{
						"name":   c.Param("name"),
						"method": "GET",
					}).Infof("Handling Show Counter -> %s", c.Param("name"))
					counter := handleCounter(c.Req.Context(), c.Param("name"))
					// counter.SetConsulClient(consulClient)

					c.JSON(200, counter)
				})

				// Allow deleting the counter
				r.DELETE("", func(c *rux.Context) {
					if hub := sentry.GetHubFromContext(c.Req.Context()); hub != nil {
						hub.Scope().SetTransaction("API - Delete Counter")
						hub.Scope().SetExtra("counter_name", c.Param("name"))
					}

					logger.WithFields(logrus.Fields{
						"name":   c.Param("name"),
						"method": "DELETE",
					}).Infof("Handling Delete Counter -> %s", c.Param("name"))
					counter := handleCounter(c.Req.Context(), c.Param("name"))
					// counter.SetConsulClient(consulClient)

					counter.Destroy()
					c.JSON(200, counter)
				})

				// Allow resetting the counter to ZERO
				r.POST("/reset", func(c *rux.Context) {
					if hub := sentry.GetHubFromContext(c.Req.Context()); hub != nil {
						hub.Scope().SetTransaction("API - Reset Counter")
						hub.Scope().SetExtra("counter_name", c.Param("name"))
					}

					logger.WithFields(logrus.Fields{
						"name":   c.Param("name"),
						"method": "POST",
					}).Infof("Handling Reset Counter -> %s", c.Param("name"))
					counter := handleCounter(c.Req.Context(), c.Param("name"))
					counter.Reset()
					c.JSON(200, counter)
				})

				// Increment and Decrement Wins
				r.Group("/win", func() {
					r.GET("", func(c *rux.Context) {
						if hub := sentry.GetHubFromContext(c.Req.Context()); hub != nil {
							hub.Scope().SetTransaction("API - Show Counter Wins")
							hub.Scope().SetExtra("counter_name", c.Param("name"))
						}

						counter := handleCounter(c.Req.Context(), c.Param("name"))
						color, ok := c.QueryParam("color")
						if !ok {
							color = "green"
						}
						_, ok = c.QueryParam("numerics")
						if ok {
							c.JSON(200, counter.WinsToNumericsCounter(color))
							return
						}
						c.JSON(200, counter)
					})
					r.PUT("", func(c *rux.Context) {
						if hub := sentry.GetHubFromContext(c.Req.Context()); hub != nil {
							hub.Scope().SetTransaction("API - Increment Counter Wins")
							hub.Scope().SetExtra("counter_name", c.Param("name"))
						}
						logger.WithFields(logrus.Fields{
							"name":   c.Param("name"),
							"method": "PUT",
						}).Infof("Handling Increment Wins -> %s", c.Param("name"))
						counter := handleCounter(c.Req.Context(), c.Param("name"))
						counter.AddWin()
						c.JSON(200, counter)
					})
					r.DELETE("", func(c *rux.Context) {
						if hub := sentry.GetHubFromContext(c.Req.Context()); hub != nil {
							hub.Scope().SetTransaction("API - Decrement Counter Wins")
							hub.Scope().SetExtra("counter_name", c.Param("name"))
						}
						logger.WithFields(logrus.Fields{
							"name":   c.Param("name"),
							"method": "DELETE",
						}).Infof("Handling Decrement Wins -> %s", c.Param("name"))
						counter := handleCounter(c.Req.Context(), c.Param("name"))
						counter.RemoveWin()
						c.JSON(200, counter)
					})
				})

				// Increment and Decrement Losses
				r.Group("/loss", func() {
					r.GET("", func(c *rux.Context) {
						if hub := sentry.GetHubFromContext(c.Req.Context()); hub != nil {
							hub.Scope().SetTransaction("API - Show Counter Losses")
							hub.Scope().SetExtra("counter_name", c.Param("name"))
						}

						counter := handleCounter(c.Req.Context(), c.Param("name"))
						color, ok := c.QueryParam("color")
						if !ok {
							color = "red"
						}
						_, ok = c.QueryParam("numerics")
						if ok {
							c.JSON(200, counter.LossesToNumericsCounter(color))
							return
						}
						c.JSON(200, counter)
					})
					r.PUT("", func(c *rux.Context) {
						if hub := sentry.GetHubFromContext(c.Req.Context()); hub != nil {
							hub.Scope().SetTransaction("API - Increment Counter Losses")
							hub.Scope().SetExtra("counter_name", c.Param("name"))
						}

						logger.WithFields(logrus.Fields{
							"name":   c.Param("name"),
							"method": "PUT",
						}).Infof("Handling Increment Losses -> %s", c.Param("name"))
						counter := handleCounter(c.Req.Context(), c.Param("name"))
						// counter.SetConsulClient(consulClient)

						counter.AddLoss()
						c.JSON(200, counter)
					})
					r.DELETE("", func(c *rux.Context) {
						if hub := sentry.GetHubFromContext(c.Req.Context()); hub != nil {
							hub.Scope().SetTransaction("API - Decrement Counter Losses")
							hub.Scope().SetExtra("counter_name", c.Param("name"))
						}

						logger.WithFields(logrus.Fields{
							"name":   c.Param("name"),
							"method": "DELETE",
						}).Infof("Handling Decrement Losses -> %s", c.Param("name"))
						counter := handleCounter(c.Req.Context(), c.Param("name"))
						// counter.SetConsulClient(consulClient)

						counter.RemoveLoss()
						c.JSON(200, counter)
					})
				})

				// Increment and Decrement Draws
				r.Group("/draw", func() {
					r.GET("", func(c *rux.Context) {
						if hub := sentry.GetHubFromContext(c.Req.Context()); hub != nil {
							hub.Scope().SetTransaction("API - Show Counter Draws")
							hub.Scope().SetExtra("counter_name", c.Param("name"))
						}

						counter := handleCounter(c.Req.Context(), c.Param("name"))
						color, ok := c.QueryParam("color")
						if !ok {
							color = "gray"
						}
						_, ok = c.QueryParam("numerics")
						if ok {
							c.JSON(200, counter.DrawsToNumericsCounter(color))
							return
						}
						c.JSON(200, counter)
					})
					r.PUT("", func(c *rux.Context) {
						if hub := sentry.GetHubFromContext(c.Req.Context()); hub != nil {
							hub.Scope().SetTransaction("API - Increment Counter Draws")
							hub.Scope().SetExtra("counter_name", c.Param("name"))
						}

						logger.WithFields(logrus.Fields{
							"name":   c.Param("name"),
							"method": "PUT",
						}).Infof("Handling Increment Draws -> %s", c.Param("name"))
						counter := handleCounter(c.Req.Context(), c.Param("name"))
						// counter.SetConsulClient(consulClient)

						counter.AddDraw()
						c.JSON(200, counter)
					})
					r.DELETE("", func(c *rux.Context) {
						if hub := sentry.GetHubFromContext(c.Req.Context()); hub != nil {
							hub.Scope().SetTransaction("API - Decrement Counter Draws")
							hub.Scope().SetExtra("counter_name", c.Param("name"))
						}

						logger.WithFields(logrus.Fields{
							"name":   c.Param("name"),
							"method": "DELETE",
						}).Infof("Handling Decrement Draws -> %s", c.Param("name"))
						counter := handleCounter(c.Req.Context(), c.Param("name"))
						// counter.SetConsulClient(consulClient)

						counter.RemoveDraw()
						c.JSON(200, counter)
					})
				})
			})
		})
	})

	r.Listen(":3000")
}
