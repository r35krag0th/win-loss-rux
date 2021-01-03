package main

import "github.com/gookit/rux"

func handleCounter(name string) *WinLossCounter {
	tmp := NewWinLossCounter(name)
	tmp.Load()
	return tmp
}

func main() {
	r := rux.New()

	// Use the /api/v1 path as the root
	r.Group("/api/v1", func() {

		// The Counter routes
		r.Group("/counters", func () {

			// List all Counters
			r.GET("", func (c *rux.Context) {
				counter := handleCounter("*")
				counterNames := counter.ListAll()
				c.JSON(200, counterNames)
			})

			// The "specific" counter routes
			r.Group("/{name}", func () {
				// Get the counter's W/L/D stats and Name
				r.GET("", func (c *rux.Context) {
					counter := handleCounter(c.Param("name"))
					c.JSON(200, counter)
				})

				// Allow deleting the counter
				r.DELETE("", func (c *rux.Context) {
					counter := handleCounter(c.Param("name"))
					counter.Destroy()
					c.JSON(200, counter)
				})

				// Allow resetting the counter to ZERO
				r.POST("/reset", func (c *rux.Context) {
					counter := handleCounter(c.Param("name"))
					counter.Reset()
					c.JSON(200, counter)
				})

				// Increment and Decrement Wins
				r.Group("/win", func (){
					r.PUT("", func (c *rux.Context) {
						counter := handleCounter(c.Param("name"))
						counter.AddWin()
						c.JSON(200, counter)
					})
					r.DELETE("", func (c *rux.Context) {
						counter := handleCounter(c.Param("name"))
						counter.RemoveWin()
						c.JSON(200, counter)
					})
				})

				// Increment and Decrement Losses
				r.Group("/loss", func (){
					r.PUT("", func (c *rux.Context) {
						counter := handleCounter(c.Param("name"))
						counter.AddLoss()
						c.JSON(200, counter)
					})
					r.DELETE("", func (c *rux.Context) {
						counter := handleCounter(c.Param("name"))
						counter.RemoveLoss()
						c.JSON(200, counter)
					})
				})

				// Increment and Decrement Draws
				r.Group("/draw", func (){
					r.PUT("", func (c *rux.Context) {
						counter := handleCounter(c.Param("name"))
						counter.AddDraw()
						c.JSON(200, counter)
					})
					r.DELETE("", func (c *rux.Context) {
						counter := handleCounter(c.Param("name"))
						counter.RemoveDraw()
						c.JSON(200, counter)
					})
				})
			})
		})
	})

	r.Listen(":3000")
}