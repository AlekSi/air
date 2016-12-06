package air

import (
	"bytes"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/html"
)

// renderer is used to provide a `render()` method for an `Air` instance for renders a "text/html"
// response by using `template.Template`.
type renderer struct {
	goTemplate      *template.Template
	templateFuncMap template.FuncMap
	air             *Air
}

// defaultTemplateFuncMap is a default template func map of `renderer`.
var defaultTemplateFuncMap = template.FuncMap{
	"strlen":  strlen,
	"substr":  substr,
	"timefmt": timefmt,
	"eq":      eq,
	"ne":      ne,
	"lt":      lt,
	"le":      le,
	"gt":      gt,
	"ge":      ge,
}

// newRenderer returns a pointer of a new instance of `Renderer`.
func newRenderer(a *Air) *renderer {
	return &renderer{
		templateFuncMap: defaultTemplateFuncMap,
		air:             a,
	}
}

// parseTemplates parses files into templates.
//
// e.g. r.air.Config.TemplatesRoot == "templates"
//
// templates/
//   index.html
//   login.html
//   register.html
//
// templates/parts/
//   header.html
//   footer.html
//
// will be parsed into:
//
// "index.html", "login.html", "register.html", "parts/header.html", "parts/footer.html".
func (r *renderer) parseTemplates() {
	tr := filepath.Clean(r.air.Config.TemplatesRoot)

	var filenames []string
	err := filepath.Walk(tr, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			return err
		}
		fns, err := filepath.Glob(path + "/*.html")
		filenames = append(filenames, fns...)
		return err
	})
	if err != nil {
		panic(err)
	}

	m := minify.New()
	m.Add("text/html", &html.Minifier{
		KeepDefaultAttrVals: true,
		KeepDocumentTags:    true,
	})
	buf := &bytes.Buffer{}

	for _, filename := range filenames {
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			panic(err)
		}

		name := filepath.ToSlash(filename[len(tr):])
		if name[0] == '/' {
			name = name[1:]
		}

		if r.goTemplate == nil {
			r.goTemplate = template.New(name).Funcs(r.templateFuncMap)
		}

		var tmpl *template.Template
		if name == r.goTemplate.Name() {
			tmpl = r.goTemplate
		} else {
			tmpl = r.goTemplate.New(name)
		}

		if r.air.Config.MinifyTemplates {
			err = m.Minify("text/html", buf, bytes.NewReader(b))
			if err != nil {
				panic(err)
			}
			b = buf.Bytes()
			buf.Reset()
		}

		_, err = tmpl.Parse(string(b))
		if err != nil {
			panic(err)
		}
	}
}

// render renders a "text/html" response by using `template.Template`.
func (r *renderer) render(w io.Writer, templateName string, res *Response) error {
	return r.goTemplate.ExecuteTemplate(w, templateName, res.Data)
}

// strlen returns the number of char in the s.
func strlen(s string) int {
	return len([]rune(s))
}

// substr returns the substring consisting of the chars of the s starting at the index i and
// continuing up to, but not including, the char at the index j.
func substr(s string, i, j int) string {
	rs := []rune(s)
	return string(rs[i:j])
}

// timefmt returns a textual representation of the t formatted according to the layout.
func timefmt(t time.Time, layout string) string {
	return t.Format(layout)
}

// eq reports whether the v is equal to one of the ovs.
//
// It means v == v1 || v == v2 || ...
func eq(v interface{}, ovs ...interface{}) bool {
	for _, ov := range ovs {
		if ov == v {
			return true
		}
	}
	return false
}

// ne reports whether the v is not equal to any of the ovs.
//
// It means v != v1 && v != v2 && ...
func ne(v interface{}, ovs ...interface{}) bool {
	return !eq(v, ovs...)
}

// lt reports whether the a is less than the b.
//
// It means a < b.
func lt(a, b interface{}) bool {
	switch a.(type) {
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(a).Int() < reflect.ValueOf(b).Int()
	case uint, uint8, uint16, uint32, uint64, uintptr:
		return reflect.ValueOf(a).Uint() < reflect.ValueOf(b).Uint()
	case float32, float64:
		return reflect.ValueOf(a).Float() < reflect.ValueOf(b).Float()
	case string:
		return a.(string) < b.(string)
	default:
		panic("invalid kind")
	}
}

// le reports whether the a is less than or equal to the b.
//
// It means a <= b.
func le(a, b interface{}) bool {
	return lt(a, b) || eq(a, b)
}

// gt reports whether the a is greater than the b.
//
// It means a > b.
func gt(a, b interface{}) bool {
	return !le(a, b)
}

// ge reports whether the a is greater than or equal to the b.
//
// It means a >= b.
func ge(a, b interface{}) bool {
	return lt(a, b)
}
