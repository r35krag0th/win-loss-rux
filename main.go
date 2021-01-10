package main

import (
	"bytes"
	"fmt"
	"github.com/gookit/rux"
	"github.com/hashicorp/consul/api"
	"html/template"
	"os"
)

//var consulClient *api.Client

func handleCounter(name string) *WinLossCounter {
	tmp := NewWinLossCounter(name)

	consulScheme := os.Getenv("CONSUL_SCHEME")
	consulAddress := os.Getenv("CONSUL_ADDR")

	targetConfig := api.DefaultConfig()

	if consulAddress != "" && consulScheme != "" {
		targetConfig = &api.Config{
			Address: consulAddress,
			Scheme:  consulScheme,
		}
	}

	consulClient, err := api.NewClient(targetConfig)
	if err != nil {
		fmt.Println("Failed to create Consul client", err)
	}
	tmp.SetConsulClient(consulClient)
	tmp.Load()
	return tmp
}

type CounterPage struct {
	Name   string
	Title  string
	Wins   int
	Losses int
	Draws  int
}

func NewCounterPageFromWinLossCounter(counter *WinLossCounter) *CounterPage {
	return &CounterPage{
		Title:  "",
		Wins:   counter.Wins,
		Losses: counter.Losses,
		Draws:  counter.Draws,
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
		counter := handleCounter("")
		//counter.SetConsulClient(consulClient)
		counterNames := counter.ListAll()

		tmpl := template.Must(template.ParseFiles("templates/index.gohtml"))
		data := IndexTemplateData{Title: "WLD - List of Counters"}

		for _, counterName := range counterNames {
			data.AddCounter(
				fmt.Sprintf("/counters/%s", counterName),
				counterName,
			)
		}
		out := bytes.Buffer{}
		err := tmpl.Execute(&out, data)
		if err != nil {
			fmt.Println("Something failed during execute", err)
			c.HTML(500, out.Bytes())
			return
		}
		c.HTML(200, out.Bytes())
	})

	r.GET("/counters/{name}", func(c *rux.Context) {
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

	// Use the /api/v1 path as the root
	r.Group("/api/v1", func() {

		// The Counter routes
		r.Group("/counters", func() {

			// List all Counters
			r.GET("", func(c *rux.Context) {
				counter := handleCounter("")
				//counter.SetConsulClient(consulClient)

				counterNames := counter.ListAll()
				c.JSON(200, counterNames)
			})

			// The "specific" counter routes
			r.Group("/{name}", func() {
				// Get the counter's W/L/D stats and Name
				r.GET("", func(c *rux.Context) {
					counter := handleCounter(c.Param("name"))
					//counter.SetConsulClient(consulClient)

					c.JSON(200, counter)
				})

				// Allow deleting the counter
				r.DELETE("", func(c *rux.Context) {
					counter := handleCounter(c.Param("name"))
					//counter.SetConsulClient(consulClient)

					counter.Destroy()
					c.JSON(200, counter)
				})

				// Allow resetting the counter to ZERO
				r.POST("/reset", func(c *rux.Context) {
					counter := handleCounter(c.Param("name"))
					//counter.SetConsulClient(consulClient)

					counter.Reset()
					c.JSON(200, counter)
				})

				// Increment and Decrement Wins
				r.Group("/win", func() {
					r.PUT("", func(c *rux.Context) {
						counter := handleCounter(c.Param("name"))
						//counter.SetConsulClient(consulClient)

						counter.AddWin()
						c.JSON(200, counter)
					})
					r.DELETE("", func(c *rux.Context) {
						counter := handleCounter(c.Param("name"))
						//counter.SetConsulClient(consulClient)

						counter.RemoveWin()
						c.JSON(200, counter)
					})
				})

				// Increment and Decrement Losses
				r.Group("/loss", func() {
					r.PUT("", func(c *rux.Context) {
						counter := handleCounter(c.Param("name"))
						//counter.SetConsulClient(consulClient)

						counter.AddLoss()
						c.JSON(200, counter)
					})
					r.DELETE("", func(c *rux.Context) {
						counter := handleCounter(c.Param("name"))
						//counter.SetConsulClient(consulClient)

						counter.RemoveLoss()
						c.JSON(200, counter)
					})
				})

				// Increment and Decrement Draws
				r.Group("/draw", func() {
					r.PUT("", func(c *rux.Context) {
						counter := handleCounter(c.Param("name"))
						//counter.SetConsulClient(consulClient)

						counter.AddDraw()
						c.JSON(200, counter)
					})
					r.DELETE("", func(c *rux.Context) {
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
