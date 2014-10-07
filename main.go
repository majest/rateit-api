package main

import (
	"fmt"
	"github.com/majest/htmlinf/db"
	"github.com/majest/htmlinf/parser"
)

var collection *db.Collection

func main() {
	collection = db.NewCollection("sites")

	site := parser.New("http://google.com")
	site.Parse()
	fmt.Println(site.TokenCounts)
	fmt.Printf("%v", site.AsCsv())
}

func dbConnect(dbName string) {
	mysqldb = m.New("tcp", "", "127.0.0.1:3306", "", "", "rater")
	err := mysqldb.Connect()
	if err != nil {
		panic(err)
	}
}
