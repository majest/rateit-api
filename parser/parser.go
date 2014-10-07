package parser

import (
	"code.google.com/p/go.net/html"
	"fmt"
	"net/http"
	"strings"
)

type Site struct {
	Url         string         `json:"url"`
	TokenCounts map[string]int `json:"tokens"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Keywords    string         `json:"keywords"`
}

func New(url string) *Site {
	site := &Site{Url: url}
	return site
}

// cheks the ration of div vs tables.
// the more tables vs divs the higher the number is going to be
// for 0 divs it will return 999 which is the highest number
func (s *Site) CheckTableDivRatio() float32 {

	div := s.TokenCounts["div"]
	tables := s.TokenCounts["table"]

	if div == 0 {
		return 999
	}

	ratio := float32(tables) / float32(div)
	return ratio
}

func (s *Site) AsCsv() string {
	result := []string{
		fmt.Sprintf("%v", s.TokenCounts["center"]),
		fmt.Sprintf("%v", s.TokenCounts["blink"]),
		fmt.Sprintf("%v", s.TokenCounts["hr"]),
		fmt.Sprintf("%v", s.TokenCounts["styleLines"]),
		fmt.Sprintf("%v", s.TokenCounts["scriptLines"]),
		fmt.Sprintf("%v", s.CheckTableDivRatio()),
	}
	return strings.Join(result, ",")
}

func (s *Site) Parse() error {

	resp, err := http.Get(s.Url)

	if err != nil {
		return err
	}

	s.TokenCounts = map[string]int{}

	d := html.NewTokenizer(resp.Body)
	previous := ""
	for {
		// token type
		tokenType := d.Next()

		if tokenType == html.ErrorToken {
			return nil
		}

		token := d.Token()

		switch tokenType {

		case html.StartTagToken:
			s.TokenCounts[token.Data]++
			//fmt.Printf("-%s\n", token.Data)
			previous = token.Data

		case html.TextToken: // text between start and end tag

			switch previous {
			case "style":
				s.TokenCounts["styleLines"] += len(strings.Split(token.Data, "\n"))
			case "script":
				s.TokenCounts["scriptLines"] += len(strings.Split(token.Data, "\n"))
			case "title":
				s.Title = token.Data
			}

		case html.EndTagToken: // </tag>
			previous = ""
		case html.SelfClosingTagToken: // <tag/>

		}
	}
	return nil
}
