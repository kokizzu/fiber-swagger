package swagger

import (
	"fmt"
	"html/template"
	"path"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/gofiber/utils/v2"
	swaggerFiles "github.com/swaggo/files/v2"
	"github.com/swaggo/swag"
)

const (
	defaultDocURL = "doc.json"
	defaultIndex  = "index.html"
)

var HandlerDefault = New()

// New returns custom handler
func New(config ...Config) fiber.Handler {
	cfg := configDefault(config...)

	index, err := template.New("swagger_index.html").Parse(indexTmpl)
	if err != nil {
		panic(fmt.Errorf("fiber: swagger middleware error -> %w", err))
	}

	var (
		basePrefix string
		once       sync.Once
		fs         = static.New("", static.Config{FS: swaggerFiles.FS})
	)

	return func(c fiber.Ctx) error {
		once.Do(func() {
			basePrefix = strings.ReplaceAll(c.Route().Path, "*", "")
		})

		prefix := basePrefix
		if forwardedPrefix := getForwardedPrefix(c); forwardedPrefix != "" {
			prefix = forwardedPrefix + prefix
		}

		cfgCopy := cfg
		if len(cfgCopy.URL) == 0 {
			cfgCopy.URL = path.Join(prefix, defaultDocURL)
		}

		p := c.Path(utils.CopyString(c.Params("*")))

		switch p {
		case defaultIndex:
			c.Type("html")
			return index.Execute(c, cfgCopy)
		case defaultDocURL:
			var doc string
			if doc, err = swag.ReadDoc(cfgCopy.InstanceName); err != nil {
				return err
			}
			return c.Type("json").SendString(doc)
		case "", "/":
			return c.Redirect().Status(fiber.StatusMovedPermanently).To(path.Join(prefix, defaultIndex))
		default:
			return fs(c)
		}
	}
}

func getForwardedPrefix(c fiber.Ctx) string {
	header := c.GetReqHeaders()["X-Forwarded-Prefix"]

	if len(header) == 0 {
		return ""
	}

	prefix := ""

	for _, rawPrefix := range header {
		endIndex := len(rawPrefix)
		for endIndex > 1 && rawPrefix[endIndex-1] == '/' {
			endIndex--
		}

		if endIndex != len(rawPrefix) {
			prefix += rawPrefix[:endIndex]
		} else {
			prefix += rawPrefix
		}
	}

	return prefix
}
