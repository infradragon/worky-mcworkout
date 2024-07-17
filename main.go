package main

import (
	"net/http"

	"github.com/go-andiamo/chioas"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	if err := workyApi.SetupRoutes(r, workyApi); err != nil {
		panic(err)
	}
	_ = http.ListenAndServe(":3009", r)
}

var allSchemas = UserSchemas // this will be changed to 'append' on second list

var workyApi = chioas.Definition{
	AutoHeadMethods: true,
	DocOptions: chioas.DocOptions{
		ServeDocs:       true,
		HideHeadMethods: true,
	},
	Paths: chioas.Paths{
		"/users": UserPath,
	},
	Components: &chioas.Components{
		Schemas: allSchemas,
	},
}
