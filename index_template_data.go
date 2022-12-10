package main

// IndexTemplateData is a data structure used to list all available counters.
// The data stored here are based on ClickableLinkData objects to form the list of links.
type IndexTemplateData struct {
	Title    string
	Counters []ClickableLinkData
}

// AddCounter will add a link (ClickableLinkData) to the counter's dashboard
func (i *IndexTemplateData) AddCounter(href, text string) {
	i.Counters = append(i.Counters, ClickableLinkData{
		Href: href,
		Text: text,
	})
}
