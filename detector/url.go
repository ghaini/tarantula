package detector

import (
	"github.com/ghaini/tarantula/constants"
	"net/url"
	"strconv"
	"strings"
)

func ConvertToUrlWithPort(url *url.URL) string {
	fullUrl := ""
	if url.Port() != "" {
		return url.String()
	}

	urlWithoutSlash := strings.TrimRight(url.String(), "/")
	if strings.HasPrefix(strings.TrimSpace(url.String()), constants.HTTPS) {
		fullUrl = urlWithoutSlash + ":" + strconv.Itoa(443)
	} else {
		fullUrl = urlWithoutSlash + ":" + strconv.Itoa(80)
	}
	return fullUrl
}
