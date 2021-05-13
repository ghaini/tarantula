package detector

import (
	"net/http"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// ExtractTitle from a response
func ExtractTitle(bodyBytes []byte, header http.Header) (title string) {
	var re = regexp.MustCompile(`(?im)<\s*title.*>(.*?)<\s*/\s*title>`)
	bodyString := string(bodyBytes)
	for _, match := range re.FindAllString(bodyString, -1) {
		title = html.UnescapeString(trimTitleTags(match))
		break
	}

	// Non UTF-8
	contentType := header.Get("Content-Type")
	// special cases
	if strings.Contains(string(contentType), "charset=GB2312") {
		titleUtf8, err := Decodegbk([]byte(title))
		if err != nil {
			return
		}

		return string(titleUtf8)
	}

	return
}

func trimTitleTags(title string) string {
	// trim <title>*</title>
	titleBegin := strings.Index(title, ">")
	titleEnd := strings.Index(title, "</")
	return title[titleBegin+1 : titleEnd]
}