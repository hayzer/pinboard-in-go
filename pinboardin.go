package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"gopkg.in/alecthomas/kingpin.v2"
)

var base_url string = "https://api.pinboard.in/v1"
var params string

var (
	app = kingpin.New("pinboardin",
		"A command line client for pinboard.in bookmarks service").
		Author("Thomas Maier").
		Version("0.001")
	username = app.Flag("username", "Pinboard username (default: $PINBOARD_USERNAME).").
			PlaceHolder("PINBOARD-USERNAME").
			Envar("PINBOARD_USERNAME").
			String()
	token = app.Flag("token", "Pinboard API Token (default: $PINBOARD_API_TOKEN).").
		PlaceHolder("PINBOARD-API-TOKEN").
		Envar("PINBOARD_API_TOKEN").
		String()
	json_resp = app.Flag("json", "JSON respons as it is.").
			Default("false").
			Bool()
	show_date = app.Flag("show-date", "Show date when bookmark was added.").
			Default("false").
			Bool()

	recent      = app.Command("recent", "Show URLs added lately.")
	recentCount = recent.Flag("count", "Number of showed URLs (default 15)").
			String()
	recentTags = recent.Flag("tag",
		"Filter recent URL by tags. (Up to 3, comma separated)").
		String()

	all      = app.Command("all", "Show all URLs.")
	allStart = all.Flag("start", "Offset value (default is 0).").
			String()
	allResults = all.Flag("results",
		"Number of results to be printed (default is all).").
		String()
	allTags = all.Flag("tag",
		"Filter recent URL by tags. (Up to 3, comma separated)").
		String()
	allFrom = all.Flag("from-date",
		"Only bookmarks since given date. UTC format (ie. 2010-12-11T19:48:02Z).").
		String()
	allTill = all.Flag("till-date",
		"Only bookmarks till given date. Same format as --from-date").
		String()

	add            = app.Command("add", "Add new URL.")
	addUrl         = add.Flag("url", "URL to add.").Required().URL()
	addTitle       = add.Flag("title", "Title of the URL.").Required().String()
	addDescription = add.Flag("description", "Add extended description of the URL").String()
	addTags        = add.Flag("tags", "Add up 100 tags (comma separated).").String()
	addNoReplace   = add.Flag("no-replace", "Don't replace existing URL.").
			Default("false").
			Bool()
	addPrivate = add.Flag("private", "Make URL private (default is public)").
			Default("false").
			Bool()
	addUnread = add.Flag("unread", "Make the URL unread").Default("false").Bool()

	delete    = app.Command("delete", "Delete URL.")
	deleteUrl = delete.Flag("url", "URL to delete.").Required().URL()

	get     = app.Command("get", "Show one/more URLs from single day.")
	getUrl  = get.Flag("url", "Return bookmark for given URL.").String()
	getTags = get.Flag("tag", "Return bookmark only for given tag(s).").String()
	getDate = get.Flag("date", "Return bookmark for given day (ie 2016-06-11).").String()

	suggest    = app.Command("suggest", "Show suggest tags for given URL.")
	suggestUrl = suggest.Flag("url", "URL to offer suggested tags.").Required().URL()
)

type SuggestContent []struct {
	Popular     []string `json:"popular,omitempty"`
	Recommended []string `json:"recommended,omitempty"`
}

type PostsContent struct {
	Date  time.Time `json:"date"`
	User  string    `json:"user"`
	Posts []struct {
		Href        string    `json:"href"`
		Description string    `json:"description"`
		Extended    string    `json:"extended"`
		Meta        string    `json:"meta"`
		Hash        string    `json:"hash"`
		Time        time.Time `json:"time"`
		Shared      string    `json:"shared"`
		Toread      string    `json:"toread"`
		Tags        string    `json:"tags"`
	} `json:"posts"`
}

type AllContent []struct {
	Href        string    `json:"href"`
	Description string    `json:"description"`
	Extended    string    `json:"extended"`
	Meta        string    `json:"meta"`
	Hash        string    `json:"hash"`
	Time        time.Time `json:"time"`
	Shared      string    `json:"shared"`
	Toread      string    `json:"toread"`
	Tags        string    `json:"tags"`
}

type ShortResponse struct {
	ResultCode string `json:"result_code"`
}

type UrlArgs struct {
	ResourceUri string
	Params      string
	Username    string
	Token       string
}

func (u UrlArgs) BuildUrl() string {
	return fmt.Sprintf("%s/%s?auth_token=%s:%s&format=json&%s",
		base_url,
		u.ResourceUri,
		*username,
		*token,
		u.Params,
	)
}

func HttpGet(url string) []uint8 {
	tr := &http.Transport{
		TLSHandshakeTimeout: time.Second * 60,
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get(url)
	if err != nil {
		panic("Can't access pinboard API server")
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		panic("Too many request to Pinboards API. Come back later")
	}

	if resp.StatusCode != http.StatusOK {
		log.Panic("Unexpected status code: %s. Expecting: %s", resp.StatusCode, http.StatusOK)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Panic(err)
	}

	if *json_resp {
		fmt.Println(string(body))
		os.Exit(0)
	}
	return body
}

func SuggestUrl() {
	p := url.Values{}
	p.Set("url", (*suggestUrl).String())
	u := UrlArgs{
		ResourceUri: "posts/suggest",
		Params:      p.Encode(),
	}
	suggest_url := u.BuildUrl()
	body := HttpGet(suggest_url)
	content := SuggestContent{}
	if err := json.Unmarshal(body, &content); err != nil {
		panic(err)
	}

	fmt.Println("Popular:")
	for _, tag := range content[0].Popular {
		fmt.Print(tag, ", ")
	}
	fmt.Print("\n")
	fmt.Println("Recommended:")
	for _, tag := range content[1].Recommended {
		fmt.Print(tag, ", ")
	}
	fmt.Print("\n")
}

func GetUrl() {
	p := url.Values{}
	if *getTags != "" {
		p.Set("tag", *getTags)
	}
	if *getDate != "" {
		p.Set("dt", *getDate)
	}
	if *getUrl != "" {
		p.Set("url", *getUrl)
	}
	u := UrlArgs{
		ResourceUri: "posts/get",
		Params:      p.Encode(),
	}

	get_url := u.BuildUrl()
	body := HttpGet(get_url)
	content := PostsContent{}
	err := json.Unmarshal(body, &content)
	if err != nil {
		panic(err)
	}

	for _, post := range content.Posts {
		if *show_date {
			fmt.Print(post.Time.Format(time.Stamp), ", ")
		}
		fmt.Print(post.Href, "\n")
	}
}

func DeleteUrl() {
	p := url.Values{}
	p.Set("url", (*deleteUrl).String())
	u := UrlArgs{
		ResourceUri: "posts/delete",
		Params:      p.Encode(),
	}
	delete_url := u.BuildUrl()
	body := HttpGet(delete_url)
	content := ShortResponse{}
	err := json.Unmarshal(body, &content)
	if err != nil {
		panic(nil)
	}
	fmt.Print(content.ResultCode, "\n")
}

func AddUrl() {
	p := url.Values{}
	p.Set("url", (*addUrl).String())
	p.Set("description", *addTitle)
	if *addDescription != "" {
		p.Set("extended", *addDescription)
	}
	if *addTags != "" {
		p.Set("tags", *addTags)
	}
	if *addNoReplace {
		p.Set("replace", "no")
	}
	if *addPrivate {
		p.Set("shared", "no")
	}
	if *addUnread {
		p.Set("toread", "yes")
	}
	u := UrlArgs{
		ResourceUri: "posts/add",
		Params:      p.Encode(),
	}
	add_url := u.BuildUrl()
	body := HttpGet(add_url)
	content := ShortResponse{}
	err := json.Unmarshal(body, &content)
	if err != nil {
		panic(nil)
	}
	fmt.Print(content.ResultCode, "\n")
}

func RecentUrls() {
	p := url.Values{}
	if *recentTags != "" {
		p.Set("tag", *recentTags)
	}
	if *recentCount != "" {
		p.Set("count", *recentCount)
	}
	u := UrlArgs{
		ResourceUri: "posts/recent",
		Params:      p.Encode(),
	}
	recent_url := u.BuildUrl()
	body := HttpGet(recent_url)
	content := PostsContent{}
	err := json.Unmarshal(body, &content)
	if err != nil {
		panic(err)
	}

	for _, post := range content.Posts {
		if *show_date {
			fmt.Print(post.Time.Format(time.Stamp), ", ")
		}
		fmt.Print(post.Href, "\n")
	}
}

func AllUrls() {
	p := url.Values{}
	if *allStart != "" {
		p.Set("start", *allStart)
	}
	if *allResults != "" {
		p.Set("results", *allResults)
	}
	if *allTags != "" {
		p.Set("tag", *allTags)
	}
	if *allFrom != "" {
		p.Set("fromdt", *allFrom)
	}
	if *allTill != "" {
		p.Set("todt", *allTill)
	}
	u := UrlArgs{
		ResourceUri: "posts/all",
		Params:      p.Encode(),
	}
	all_url := u.BuildUrl()
	body := HttpGet(all_url)
	content := AllContent{}
	err := json.Unmarshal(body, &content)
	if err != nil {
		log.Panic(err)
	}

	for _, post := range content {
		if *show_date {
			fmt.Print(post.Time.Format(time.Stamp), ", ")
		}
		fmt.Print(post.Href, "\n")
	}
}

func main() {
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case recent.FullCommand():
		RecentUrls()
	case all.FullCommand():
		AllUrls()
	case add.FullCommand():
		AddUrl()
	case delete.FullCommand():
		DeleteUrl()
	case get.FullCommand():
		GetUrl()
	case suggest.FullCommand():
		SuggestUrl()
	}
}
