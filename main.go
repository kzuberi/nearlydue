package main

import (
	"fmt"
	"strings"
	"strconv"
	"time"
	"github.com/headzoo/surf"
	"github.com/headzoo/surf/browser"
	"github.com/BurntSushi/toml"
	"github.com/headzoo/surf/jar"
	"github.com/PuerkitoBio/goquery"
	"net/smtp"
	"text/template"
	"bytes"
)

const mainpage = "http://catalogue.bramlib.on.ca/Mobile"

type Config struct {
	From    from
	Patrons []patron
}

type from struct {
	Smtp  string
	Email string
	Auth  string
}

type patron struct {
	Name  string
	Card  string
	Pin   string
	Email string
}

func main() {
	Getmainpage()
	fmt.Println("done")
}

func Getmainpage() {

	var conf Config
	_, err := toml.DecodeFile("config.toml", &conf)

	if err != nil {
		panic(err)
	}

	for _, p := range conf.Patrons {

		items := scrape(mainpage, p)
		if len(items) == 0 {
			panic("blah no items out")
		}

		message := compose(conf.From, items)
		fmt.Println("the message is:", message, "ok?")
		err = mail(message, conf.From, p)

		if err != nil {
			panic(err)
		}
	}

	fmt.Println("done")
}

func compose(f from, items []an_item) (message string) {

	t_header := template.New("header")
	t_body := t_header.New("body")
	t_header, err := t_header.Parse("To: {{.Email}}\r\nSubject: Library Alert\r\n")
	if err != nil {
		panic(err)
	}

	t_body, err = t_body.Parse("\r\n{{.Title}}")
	if err != nil {
		panic(err)
	}

	b := new(bytes.Buffer)

	err = t_header.Execute(b, f)
	if err != nil {
		panic(err)
	}

	message = b.String()

	b.Reset()

	err = t_body.Execute(b, items[0])
	if err != nil {
		panic(err)
	}

	return message + b.String()
}

func mail(message string, f from, p patron) (err error) {

	// parse host from Smtp = "host:port"
	host := strings.Split(f.Smtp, ":")[0]
	a := smtp.PlainAuth("", f.Email, f.Auth, host)
	err = smtp.SendMail(f.Smtp, a, f.Email,
		[]string{p.Email}, []byte(message))

	return err;
}

func scrape(page string, p patron) (items []an_item) {
	bow := surf.NewBrowser()
	historyJar := jar.NewMemoryHistory()
	bow.SetHistoryJar(historyJar)

	// grab main page and click to login page
	err := bow.Open(page)
	if err != nil {
		panic(err)
	}

	fmt.Println(bow.Title())

	err = bow.Click("a:contains('My Account')")
	if err != nil {
		panic(err)
	}

	fmt.Println(bow.Title())

	// populate login form and submit
	form, err := bow.Form("form[action='/Mobile/MyAccount/Logon']")
	if err != nil {
		panic(err)
	}

	fmt.Println("logging in", p.Name)
	form.Input("barcodeOrUsername", p.Card)
	form.Input("password", p.Pin)

	fmt.Println(form)

	err = form.Submit()
	if err != nil {
		panic(err)
	}

	items_out := bow.Find("a[href='/Mobile/MyAccount/ItemsOut']")
	num_items_out, err := strconv.Atoi(strings.Split(items_out.Text(), " ")[0])

	if err != nil {
		panic(err)
	}

	if num_items_out > 0 {
		items = getItemsOut(bow)
		return items
	}

	return nil
}

type an_item struct {
	Title string
	Due   string
}

func getItemsOut(bow *browser.Browser) (gather_items []an_item) {
	err := bow.Click("a[href='/Mobile/MyAccount/ItemsOut']")
	if err != nil {
		panic(err)
	}

	gather_items = []an_item{}

	bow.Find("tr").Each(func(i int, s *goquery.Selection) {
		b := s.Find("td > a")

		if b.Length() == 0 {
			return
		}

		title := b.Text()

		html, err := b.Parent().Html()
		if err != nil {
			panic(err)
		}

		parts := strings.Split(html, "<br/>")
		due := parts[2]
		due = strings.Split(due, "<img")[0]
		due = strings.Split(due, ":")[1]
		due = strings.TrimSpace(due)

		parts = strings.Split(due, "/")

		// how old?
		now := time.Now()
		due_time, err := time.Parse("2/1/2006", due)
		if err != nil {
			panic(err)
		}

		duration := time.Since(due_time)
		days := duration.Hours()/24.0

		if days > -2 && days < 0 {
			fmt.Println("due soon")
		} else if days >= 0 {
			fmt.Println("overdue")
		}

		fmt.Println("now:", now)
		fmt.Println("due_time:", due_time)

		a := an_item{title, due}
		gather_items= append(gather_items, a)

	})

	return
}