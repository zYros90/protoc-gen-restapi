package main

import (
	pgs "github.com/lyft/protoc-gen-star"
	pgsgo "github.com/lyft/protoc-gen-star/lang/go"
)

func main() {
	pgs.Init(
		pgs.DebugEnv("DEBUG"),
	).RegisterModule(
		APIAnnotations(),
	).RegisterPostProcessor(
		pgsgo.GoFmt(),
	).Render()
}
