package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"

	"github.com/nesbyte/loadr"
)

//go:embed "*"
var baseFS embed.FS

// The struct for the index page
type IndexData struct {
	Name    string
	Content BuisnessData
}

// Some business data I want to use in the template
// Might come from database or oher external service
type BuisnessData struct {
	NumRegisteredUsers int
	NumUsersToday      int
	TotalSales         int
	Profit             int
}

// fake data
var businessData = BuisnessData{NumRegisteredUsers: 5042, NumUsersToday: 48, TotalSales: 103452, Profit: 1932}

var base = loadr.NewTemplateContext(loadr.BaseConfig{baseFS}, loadr.NoData, "index.html", "global_components.html")
var index = loadr.NewTemplate(base, "index.html", IndexData{})
var data = loadr.NewSubTemplate(base, "business-data", BuisnessData{})
var profit = loadr.NewSubTemplate(base, "profit", BuisnessData{}.Profit)

func main() {
	err := loadr.LoadTemplates()
	if err != nil {
		log.Fatalln(err)
	}

	r := http.NewServeMux()

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		index.Render(w, IndexData{"Alice", businessData})
	})

	// The data endpoint to return only the business data
	// Might be for a seperate view - or for updating/swapping the component
	r.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		data.Render(w, businessData)
	})

	// Endpoint for just seeing the profit
	r.HandleFunc("/profit", func(w http.ResponseWriter, r *http.Request) {
		profit.Render(w, businessData.Profit)
	})

	fmt.Println("Listening on 8080, open http://localhost:8080/")
	err = http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatalln(err)
	}
}
