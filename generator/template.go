package generator

import (
	"bytes"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"text/template"

	_ "embed"
	log "github.com/sirupsen/logrus"

	"github.com/Masterminds/sprig"
	"github.com/iancoleman/strcase"

	"github.com/nolotz/protoc-gen-grpc-gateway-ts/data"
	"github.com/nolotz/protoc-gen-grpc-gateway-ts/registry"
)

//go:embed tmpl.ts.tmpl
var tmpl string

//go:embed fetch.ts.tmpl
var fetchTmpl string

// GetTemplate gets the templates to for the typescript file
func GetTemplate(r *registry.Registry) *template.Template {
	t := template.New("file")
	t = t.Funcs(sprig.TxtFuncMap())

	t = t.Funcs(template.FuncMap{
		"include": include(t),
		"tsType": func(fieldType data.Type) string {
			return tsType(r, fieldType)
		},
		"renderURL":    renderURL(r),
		"buildInitReq": buildInitReq,
		"fieldName":    fieldName(r),
	})

	t = template.Must(t.Parse(tmpl))
	return t
}

func fieldName(r *registry.Registry) func(name string) string {
	return func(name string) string {
		if r.UseProtoNames {
			return name
		}

		return strcase.ToLowerCamel(name)
	}
}

func renderURL(r *registry.Registry) func(method data.Method) string {
	fieldNameFn := fieldName(r)
	return func(method data.Method) string {
		methodURL := method.URL
		reg := regexp.MustCompile("{([^}]+)}")
		matches := reg.FindAllStringSubmatch(methodURL, -1)
		fieldsInPath := make([]string, 0, len(matches))
		if len(matches) > 0 {
			log.Debugf("url matches %v", matches)
			for _, m := range matches {
				expToReplace := m[0]
				fieldName := fieldNameFn(m[1])
				part := fmt.Sprintf(`${req["%s"]}`, fieldName)
				methodURL = strings.ReplaceAll(methodURL, expToReplace, part)
				fieldsInPath = append(fieldsInPath, fmt.Sprintf(`"%s"`, fieldName))
			}
		}
		urlPathParams := fmt.Sprintf("[%s]", strings.Join(fieldsInPath, ", "))

		if !method.ClientStreaming && method.HTTPMethod == "GET" {
			// parse the url to check for query string
			parsedURL, err := url.Parse(methodURL)
			if err != nil {
				return methodURL
			}
			renderURLSearchParamsFn := fmt.Sprintf("${fm.renderURLSearchParams(req, %s)}", urlPathParams)
			// prepend "&" if query string is present otherwise prepend "?"
			// trim leading "&" if present before prepending it
			if parsedURL.RawQuery != "" {
				methodURL = strings.TrimRight(methodURL, "&") + "&" + renderURLSearchParamsFn
			} else {
				methodURL += "?" + renderURLSearchParamsFn
			}
		}

		return methodURL
	}
}

func buildInitReq(method data.Method) string {
	httpMethod := method.HTTPMethod
	m := `method: "` + httpMethod + `"`
	fields := []string{m}
	if method.HTTPRequestBody == nil || *method.HTTPRequestBody == "*" {
		fields = append(fields, "body: JSON.stringify(req)")
	} else if *method.HTTPRequestBody != "" {
		fields = append(fields, `body: JSON.stringify(req["`+*method.HTTPRequestBody+`"])`)
	}

	return strings.Join(fields, ", ")

}

// GetFetchModuleTemplate returns the go template for fetch module
func GetFetchModuleTemplate() *template.Template {
	t := template.New("fetch")
	return template.Must(t.Parse(fetchTmpl))
}

// include is the include template functions copied from
// copied from: https://github.com/helm/helm/blob/8648ccf5d35d682dcd5f7a9c2082f0aaf071e817/pkg/engine/engine.go#L147-L154
func include(t *template.Template) func(name string, data interface{}) (string, error) {
	return func(name string, data interface{}) (string, error) {
		buf := bytes.NewBufferString("")
		if err := t.ExecuteTemplate(buf, name, data); err != nil {
			return "", err
		}
		return buf.String(), nil
	}
}

func tsType(r *registry.Registry, fieldType data.Type) string {
	info := fieldType.GetType()
	typeInfo, ok := r.Types[info.Type]
	if ok && typeInfo.IsMapEntry {
		keyType := tsType(r, typeInfo.KeyType)
		valueType := tsType(r, typeInfo.ValueType)

		return fmt.Sprintf("{[key: %s]: %s}", keyType, valueType)
	}

	typeStr := ""
	if strings.Index(info.Type, ".") != 0 {
		typeStr = mapScalaType(info.Type)
	} else if !info.IsExternal {
		typeStr = typeInfo.PackageIdentifier
	} else {
		typeStr = data.GetModuleName(typeInfo.Package, typeInfo.File) + "." + typeInfo.PackageIdentifier
	}

	if info.IsRepeated {
		typeStr += "[]"
	}
	return typeStr
}

func mapScalaType(protoType string) string {
	switch protoType {
	case "uint64", "sint64", "int64", "fixed64", "sfixed64", "string":
		return "string"
	case "float", "double", "int32", "sint32", "uint32", "fixed32", "sfixed32":
		return "number"
	case "bool":
		return "boolean"
	case "bytes":
		return "Uint8Array"
	}

	return ""

}
