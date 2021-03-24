package main

import (
	"bytes"
	"fmt"
	"github.com/gookit/rux"
	"github.com/hashicorp/consul/api"
	"github.com/sirupsen/logrus"
	"html/template"
	"os"
)

//var consulClient *api.Client
func init() {
	logrus.SetFormatter(&logrus.TextFormatter{})
	logrus.SetReportCaller(false)
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)
}

func handleCounter(name string) *WinLossCounter {
	logger := logrus.WithFields(logrus.Fields{
		"counter_name": name,
	})

	logger.Debug("Creating new WinLossCounter")
	tmp := NewWinLossCounter(name)

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

	logger.Debug("Creating Consul client")
	consulClient, err := api.NewClient(targetConfig)
	if err != nil {
		logger.Error("Failed to create Consul client", err)
	}

	logger.Debug("Setting Consul Client in WinLossCounter")
	tmp.SetConsulClient(consulClient)

	logger.Debug("Loading counter's data")
	tmp.Load()

	logger.Debug("Returning initialized counter")
	return tmp
}

type CounterPage struct {
	Name       string
	Title      string
	Wins       int
	Losses     int
	Draws      int
	PrettyName string
}

func NewCounterPageFromWinLossCounter(counter *WinLossCounter) *CounterPage {
	logger := logrus.WithFields(logrus.Fields{
		"wins":        counter.Wins,
		"losses":      counter.Losses,
		"draws":       counter.Draws,
		"pretty_name": counter.PrettyName,
		"name":        counter.Name,
	})
	logger.Debug("Creating CounterPage")
	return &CounterPage{
		Title:      "",
		Wins:       counter.Wins,
		Losses:     counter.Losses,
		Draws:      counter.Draws,
		PrettyName: counter.PrettyName,
		Name:       counter.Name,
	}
}

type ClickableLinkData struct {
	Href string
	Text string
}

type IndexTemplateData struct {
	Title    string
	Counters []ClickableLinkData
}

func (i *IndexTemplateData) AddCounter(href, text string) {
	i.Counters = append(i.Counters, ClickableLinkData{
		Href: href,
		Text: text,
	})
}

func main() {
	r := rux.New()

	r.GET("", func(c *rux.Context) {
		logger := logrus.WithFields(logrus.Fields{
			"path": "/",
		})

		logger.Debug("Creating blank Counter as helper")
		counter := handleCounter("")
		//counter.SetConsulClient(consulClient)

		logger.Debug("Listing all counters")
		counterNames := counter.ListAll()

		logger.WithFields(logrus.Fields{
			"template": "template/index.gohtml",
		}).Debug("Initializing template for index page")
		tmpl := template.Must(template.ParseFiles("templates/index.gohtml"))

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
		err := tmpl.Execute(&out, data)
		if err != nil {
			logger.Error("Something failed during execute", err)
			c.HTML(500, out.Bytes())
			return
		}

		logger.Info("Sending response for index.gohtml template")
		c.HTML(200, out.Bytes())
	})

	r.GET("/counters/{name}", func(c *rux.Context) {
		logger := logrus.WithFields(logrus.Fields{
			"path": "/counters/{name}",
			"name": c.Param("name"),
		})
		logger.Info("Creating counter")
		counter := handleCounter(c.Param("name"))
		//counter.SetConsulClient(consulClient)

		tmpl := template.Must(template.ParseFiles("templates/counter.gohtml"))
		data := NewCounterPageFromWinLossCounter(counter)
		data.Name = counter.Name
		data.Title = fmt.Sprintf("WLD Counter - %s", counter.Name)

		out := bytes.Buffer{}
		err := tmpl.Execute(&out, data)
		if err != nil {
			fmt.Println("Something failed during execute", err)
		}
		c.HTML(200, out.Bytes())
	})

	r.GET("/counters/{name}/solo", func(c *rux.Context) {
		logger := logrus.WithFields(logrus.Fields{
			"path": "/counters/{name}/solo",
			"name": c.Param("name"),
		})
		counter := handleCounter(c.Param("name"))
		tmpl := template.Must(template.ParseFiles("templates/solo_counter.gohtml"))
		data := NewCounterPageFromWinLossCounter(counter)
		data.Name = counter.Name
		data.Title = fmt.Sprintf("WLD Counter (Solo) - %s", counter.Name)

		out := bytes.Buffer{}
		err := tmpl.Execute(&out, data)
		if err != nil {
			logger.Error("Something failed during execute", err)
		}
		c.HTML(200, out.Bytes())
	})

	// Use the /api/v1 path as the root
	r.Group("/api/v1", func() {
		apiLogger := logrus.WithFields(logrus.Fields{
			"path": "/api/v1",
		})

		// The Counter routes
		r.Group("/counters", func() {
			counterLogger := apiLogger.WithFields(logrus.Fields{
				"path": "/aip/v1/counters",
			})
			// List all Counters
			r.GET("", func(c *rux.Context) {
				counterLogger.WithFields(logrus.Fields{
					"method": "GET",
				}).Info("Handling Index request")
				counter := handleCounter("")
				//counter.SetConsulClient(consulClient)

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
					logger.WithFields(logrus.Fields{
						"name":   c.Param("name"),
						"method": "GET",
					}).Infof("Handling Show Counter -> %s", c.Param("name"))
					counter := handleCounter(c.Param("name"))
					//counter.SetConsulClient(consulClient)

					c.JSON(200, counter)
				})

				// Allow deleting the counter
				r.DELETE("", func(c *rux.Context) {
					logger.WithFields(logrus.Fields{
						"name":   c.Param("name"),
						"method": "DELETE",
					}).Infof("Handling Delete Counter -> %s", c.Param("name"))
					counter := handleCounter(c.Param("name"))
					//counter.SetConsulClient(consulClient)

					counter.Destroy()
					c.JSON(200, counter)
				})

				// Allow resetting the counter to ZERO
				r.POST("/reset", func(c *rux.Context) {
					logger.WithFields(logrus.Fields{
						"name":   c.Param("name"),
						"method": "POST",
					}).Infof("Handling Reset Counter -> %s", c.Param("name"))
					counter := handleCounter(c.Param("name"))
					//counter.SetConsulClient(consulClient)

					counter.Reset()
					c.JSON(200, counter)
				})

				// Increment and Decrement Wins
				r.Group("/win", func() {
					r.PUT("", func(c *rux.Context) {
						logger.WithFields(logrus.Fields{
							"name":   c.Param("name"),
							"method": "PUT",
						}).Infof("Handling Increment Wins -> %s", c.Param("name"))
						counter := handleCounter(c.Param("name"))
						//counter.SetConsulClient(consulClient)

						counter.AddWin()
						c.JSON(200, counter)
					})
					r.DELETE("", func(c *rux.Context) {
						logger.WithFields(logrus.Fields{
							"name":   c.Param("name"),
							"method": "DELETE",
						}).Infof("Handling Decrement Wins -> %s", c.Param("name"))
						counter := handleCounter(c.Param("name"))
						//counter.SetConsulClient(consulClient)

						counter.RemoveWin()
						c.JSON(200, counter)
					})
				})

				// Increment and Decrement Losses
				r.Group("/loss", func() {
					r.PUT("", func(c *rux.Context) {
						logger.WithFields(logrus.Fields{
							"name":   c.Param("name"),
							"method": "PUT",
						}).Infof("Handling Increment Losses -> %s", c.Param("name"))
						counter := handleCounter(c.Param("name"))
						//counter.SetConsulClient(consulClient)

						counter.AddLoss()
						c.JSON(200, counter)
					})
					r.DELETE("", func(c *rux.Context) {
						logger.WithFields(logrus.Fields{
							"name":   c.Param("name"),
							"method": "DELETE",
						}).Infof("Handling Decrement Losses -> %s", c.Param("name"))
						counter := handleCounter(c.Param("name"))
						//counter.SetConsulClient(consulClient)

						counter.RemoveLoss()
						c.JSON(200, counter)
					})
				})

				// Increment and Decrement Draws
				r.Group("/draw", func() {
					r.PUT("", func(c *rux.Context) {
						logger.WithFields(logrus.Fields{
							"name":   c.Param("name"),
							"method": "PUT",
						}).Infof("Handling Increment Draws -> %s", c.Param("name"))
						counter := handleCounter(c.Param("name"))
						//counter.SetConsulClient(consulClient)

						counter.AddDraw()
						c.JSON(200, counter)
					})
					r.DELETE("", func(c *rux.Context) {
						logger.WithFields(logrus.Fields{
							"name":   c.Param("name"),
							"method": "DELETE",
						}).Infof("Handling Decrement Draws -> %s", c.Param("name"))
						counter := handleCounter(c.Param("name"))
						//counter.SetConsulClient(consulClient)

						counter.RemoveDraw()
						c.JSON(200, counter)
					})
				})
			})
		})
	})

	r.Listen(":3000")
}
