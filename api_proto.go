package main

import (
	"text/template"

	pgs "github.com/lyft/protoc-gen-star"
	pgsgo "github.com/lyft/protoc-gen-star/lang/go"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
)

type APIModule struct {
	*pgs.ModuleBase
	ctx pgsgo.Context
	tpl *template.Template

	MethodSets map[string][]*methodDesc
}

type methodDesc struct {
	// method
	Name         string
	OriginalName string // The parsed original name
	Num          int
	Request      string
	Reply        string
	// http_rule
	Path         string
	Method       string
	HasVars      bool
	HasBody      bool
	Body         string
	ResponseBody string
}

func APIAnnotations() *APIModule {
	return &APIModule{ModuleBase: &pgs.ModuleBase{}}
}

func (p *APIModule) InitContext(c pgs.BuildContext) {
	p.ModuleBase.InitContext(c)
	p.ctx = pgsgo.InitContext(c.Parameters())

	tpl := template.New("api").Funcs(map[string]interface{}{
		"package":    p.ctx.PackageName,
		"name":       p.ctx.Name,
		"methodsets": p.getMethodDesc,
	})

	p.tpl = template.Must(tpl.Parse(apiTpl))
}

func (p *APIModule) getMethodDesc(m pgs.Service) []*methodDesc {
	return p.MethodSets[m.Name().String()]

}

// Name satisfies the generator.Plugin interface.
func (p *APIModule) Name() string { return "api" }

func (p *APIModule) Execute(targets map[string]pgs.File, pkgs map[string]pgs.Package) []pgs.Artifact {
	p.MethodSets = make(map[string][]*methodDesc)

	for _, t := range targets {
		for _, svc := range t.Services() {
			p.setAPIAnnotations(svc.Name().String(), svc.Methods())
		}

		p.generate(t)
	}

	return p.Artifacts()
}

func (p *APIModule) setAPIAnnotations(svcName string, methods []pgs.Method) {
	methodSetsList := make([]*methodDesc, 0)
	for _, method := range methods {
		if method.ClientStreaming() || method.ServerStreaming() {
			continue
		}

		method.Descriptor()
		x := method.Descriptor().Options
		x.ProtoMessage()
		y := method.Descriptor().GetOptions()
		y.ProtoReflect()
		rule, ok := proto.GetExtension(y.ProtoReflect().Interface(), annotations.E_Http).(*annotations.HttpRule)
		if rule != nil && ok {
			md := buildHTTPRule(method, rule)
			methodSetsList = append(methodSetsList, md)
		}
	}
	p.MethodSets[svcName] = methodSetsList
}

func (p *APIModule) generate(f pgs.File) {
	if len(f.Messages()) == 0 {
		return
	}

	name := p.ctx.OutputPath(f).SetExt(".api.go")
	p.AddGeneratorTemplateFile(name.String(), p.tpl, f)
}

const apiTpl = `package {{ package . }}

import(
	_ "github.com/labstack/echo/v4"
)

{{ range $svc := .Services }}

{{ range $el :=  methodsets . }}
const {{$svc.Name}}_{{$el.Name}}_Method = "{{$el.Method}}"
const {{$svc.Name}}_{{$el.Name}}_Path = "{{$el.Path}}"
{{end}}
{{end}}
`
