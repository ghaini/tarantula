package detector

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ghaini/tarantula/constants"
	"github.com/valyala/fasthttp"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Technology struct {
	appDefs *appsDefinition
}

// appsDefinition type encapsulates the json encoding of the whole technologies.json file
type appsDefinition struct {
	Apps map[string]app      `json:"technologies"`
	Cats map[string]category `json:"categories"`
}

// app type encapsulates all the data about an app from technologies.json
type app struct {
	Cats     StringArray            `json:"cats"`
	CatNames []string               `json:"category_names"`
	Cookies  map[string]string      `json:"cookies"`
	Headers  map[string]string      `json:"headers"`
	Meta     map[string]StringArray `json:"meta"`
	HTML     StringArray            `json:"html"`
	Script   StringArray            `json:"script"`
	URL      StringArray            `json:"url"`
	Website  string                 `json:"website"`
	Implies  StringArray            `json:"implies"`

	HTMLRegex   []appRegexp `json:"-"`
	ScriptRegex []appRegexp `json:"-"`
	URLRegex    []appRegexp `json:"-"`
	HeaderRegex []appRegexp `json:"-"`
	MetaRegex   []appRegexp `json:"-"`
	CookieRegex []appRegexp `json:"-"`
}

// category names defined by wappalyzer
type category struct {
	Name string `json:"name"`
}

type appRegexp struct {
	Name    string
	Regexp  *regexp.Regexp
	Version string
}

// Match type encapsulates the app information from a match on a document
type Match struct {
	app     `json:"app"`
	AppName string     `json:"app_name"`
	Matches [][]string `json:"matches"`
	Version string     `json:"version"`
}

// StringArray type is a wrapper for []string for use in unmarshalling the technologies.json
type StringArray []string

func NewTechnology() *Technology {
	home, err := os.UserHomeDir()
	if err != nil {
		return &Technology{}
	}
	if _, err := os.Stat(home + "/.tarantula/technologies.json"); os.IsNotExist(err) {
		getTechnologyListFile()
	}
	home, err = os.UserHomeDir()
	if err != nil {
		return nil
	}
	t := &Technology{}
	appsFile, err := os.Open(home + "/.tarantula/technologies.json")
	if err != nil {
		return nil
	}

	defer appsFile.Close()
	t.loadApps(appsFile)
	return t
}

// UnmarshalJSON is a custom unmarshaler for handling bogus technologies.json types from wappalyzer
func (t *StringArray) UnmarshalJSON(data []byte) error {
	var s string
	var sa []string
	var na []int

	if err := json.Unmarshal(data, &s); err != nil {
		if err := json.Unmarshal(data, &na); err == nil {
			// not a string, so maybe []int?
			*t = make(StringArray, len(na))

			for i, number := range na {
				(*t)[i] = fmt.Sprintf("%d", number)
			}

			return nil
		} else if err := json.Unmarshal(data, &sa); err == nil {
			// not a string, so maybe []string?
			*t = sa
			return nil
		}
		fmt.Println(string(data))
		return err
	}
	*t = StringArray{s}
	return nil
}

func (t *Technology) Technology(url string, response []byte, headers http.Header, cookies []*http.Cookie) []Match {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(response))
	if err != nil {
		return []Match{}
	}

	var apps = make([]Match, 0)
	var cookiesMap = make(map[string]string)

	for _, c := range cookies {
		cookiesMap[c.Name] = c.Value
	}

	for appname, app := range t.appDefs.Apps {
		findings := Match{
			app:     app,
			AppName: appname,
			Matches: make([][]string, 0),
		}

		// check raw html
		if m, v := findMatches(string(response), app.HTMLRegex); len(m) > 0 {
			findings.Matches = append(findings.Matches, m...)
			findings.updateVersion(v)
		}

		// check response header
		headerFindings, version := app.FindInHeaders(headers)
		findings.Matches = append(findings.Matches, headerFindings...)
		findings.updateVersion(version)

		// check url
		if m, v := findMatches(url, app.URLRegex); len(m) > 0 {
			findings.Matches = append(findings.Matches, m...)
			findings.updateVersion(v)
		}

		// check script tags
		doc.Find("script").Each(func(i int, s *goquery.Selection) {
			if script, exists := s.Attr("src"); exists {
				if m, v := findMatches(script, app.ScriptRegex); len(m) > 0 {
					findings.Matches = append(findings.Matches, m...)
					findings.updateVersion(v)
				}
			}
		})

		// check meta tags
		for _, h := range app.MetaRegex {
			selector := fmt.Sprintf("meta[name='%s']", h.Name)
			doc.Find(selector).Each(func(i int, s *goquery.Selection) {
				content, _ := s.Attr("content")
				if m, v := findMatches(content, []appRegexp{h}); len(m) > 0 {
					findings.Matches = append(findings.Matches, m...)
					findings.updateVersion(v)
				}
			})
		}

		// check cookies
		for _, c := range app.CookieRegex {
			if _, ok := cookiesMap[c.Name]; ok {

				// if there is a regexp set, ensure it matches.
				// otherwise just add this as a match
				if c.Regexp != nil {

					// only match single appRegexp on this specific cookie
					if m, v := findMatches(cookiesMap[c.Name], []appRegexp{c}); len(m) > 0 {
						findings.Matches = append(findings.Matches, m...)
						findings.updateVersion(v)
					}

				} else {
					findings.Matches = append(findings.Matches, []string{c.Name})
				}
			}

		}

		if len(findings.Matches) > 0 {
			apps = append(apps, findings)

			// handle implies
			for _, implies := range app.Implies {
				for implyAppname, implyApp := range t.appDefs.Apps {
					if implies != implyAppname {
						continue
					}

					f2 := Match{
						app:     implyApp,
						AppName: implyAppname,
						Matches: make([][]string, 0),
					}
					apps = append(apps, f2)
				}

			}
		}
	}
	return apps
}

func (t *Technology) loadApps(r io.Reader) error {
	dec := json.NewDecoder(r)
	if err := dec.Decode(&t.appDefs); err != nil {
		return err
	}

	for key, value := range t.appDefs.Apps {

		app := t.appDefs.Apps[key]

		app.HTMLRegex = t.compileRegexes(value.HTML)
		app.ScriptRegex = t.compileRegexes(value.Script)
		app.URLRegex = t.compileRegexes(value.URL)

		app.HeaderRegex = t.compileNamedRegexes(app.Headers)
		app.CookieRegex = t.compileNamedRegexes(app.Cookies)

		// handle special meta field where value can be a list
		// of strings. we join them as a simple regex here
		metaRegex := make(map[string]string)
		for k, v := range app.Meta {
			metaRegex[k] = strings.Join(v, "|")
		}
		app.MetaRegex = t.compileNamedRegexes(metaRegex)

		app.CatNames = make([]string, 0)

		for _, cid := range app.Cats {
			if category, ok := t.appDefs.Cats[string(cid)]; ok && category.Name != "" {
				app.CatNames = append(app.CatNames, category.Name)
			}
		}

		t.appDefs.Apps[key] = app

	}

	return nil
}

func (t *Technology) compileRegexes(s StringArray) []appRegexp {
	var list []appRegexp

	for _, regexString := range s {

		// Split version detection
		splitted := strings.Split(regexString, "\\;")

		regex, err := regexp.Compile(splitted[0])
		if err != nil {
			// ignore failed compiling for now
			// log.Printf("warning: compiling regexp for failed: %v", regexString, err)
		} else {
			rv := appRegexp{
				Regexp: regex,
			}

			if len(splitted) > 1 && strings.HasPrefix(splitted[0], "version") {
				rv.Version = splitted[1][8:]
			}

			list = append(list, rv)
		}
	}

	return list
}

func (t *Technology) compileNamedRegexes(from map[string]string) []appRegexp {

	var list []appRegexp

	for key, value := range from {

		h := appRegexp{
			Name: key,
		}

		if value == "" {
			value = ".*"
		}

		// Filter out webapplyzer attributes from regular expression
		splitted := strings.Split(value, "\\;")

		r, err := regexp.Compile(splitted[0])
		if err != nil {
			continue
		}

		if len(splitted) > 1 && strings.HasPrefix(splitted[1], "version:") {
			h.Version = splitted[1][8:]
		}

		h.Regexp = r
		list = append(list, h)
	}

	return list
}

func (m *Match) updateVersion(version string) {
	if version != "" {
		m.Version = version
	}
}

func (app *app) FindInHeaders(headers http.Header) (matches [][]string, version string) {
	var v string

	for _, hre := range app.HeaderRegex {
		if headers.Get(hre.Name) == "" {
			continue
		}
		hk := http.CanonicalHeaderKey(hre.Name)
		for _, headerValue := range headers[hk] {
			if headerValue == "" {
				continue
			}
			if m, version := findMatches(headerValue, []appRegexp{hre}); len(m) > 0 {
				matches = append(matches, m...)
				v = version
			}
		}
	}
	return matches, v
}

func findMatches(content string, regexes []appRegexp) ([][]string, string) {
	var m [][]string
	var version string

	for _, r := range regexes {
		matches := r.Regexp.FindAllStringSubmatch(content, -1)
		if matches == nil {
			continue
		}

		m = append(m, matches...)

		if r.Version != "" {
			version = findVersion(m, r.Version)
		}

	}
	return m, version
}

func findVersion(matches [][]string, version string) string {
	var v string

	for _, matchPair := range matches {
		// replace backtraces (max: 3)
		for i := 1; i <= 3; i++ {
			bt := fmt.Sprintf("\\%v", i)
			if strings.Contains(version, bt) && len(matchPair) >= i {
				v = strings.Replace(version, bt, matchPair[i], 1)
			}
		}

		// return first found version
		if v != "" {
			return v
		}

	}

	return ""
}

func getTechnologyListFile() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	_ = os.MkdirAll(home+"/.tarantula", 0755)
	appsFile, err := os.Create(home + "/.tarantula/technologies.json")
	if err != nil {
		return err
	}

	defer appsFile.Close()
	_, resp, err := fasthttp.Get(nil, constants.TechnologiesFileAddress)
	if err != nil {
		return err
	}

	r := bytes.NewReader(resp)
	_, err = io.Copy(appsFile, r)
	if err != nil {
		return err
	}

	return nil
}
