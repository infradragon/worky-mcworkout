package main

import (
	"encoding/json"
	"github.com/go-andiamo/chioas"
	"net/http"
)

type User struct {
	Id       string `json:"_id" oas:"description: db oid"`
	Username string `json:"username" oas:"description: specific indexed username"`
	Name     string `json:"name" oas:"description: Persons name to use"`
}

var UserPath = chioas.Path{
	Methods: chioas.Methods{
		http.MethodGet: {
			Handler: getUsers,
			Responses: chioas.Responses{
				http.StatusOK: {
					Description: "List of Users",
					IsArray:     true,
					SchemaRef:   "User",
				},
			},
		},
	},
}

var UserSchemas = []chioas.Schema{
	(&chioas.Schema{
		Name:        "User",
		Description: "A User",
		Comment:     chioas.SourceComment(),
	}).Must(User{
		Id:       "66971add3abcef545e64400b",
		Name:     "Dug Somebody",
		Username: "dug",
	}),
}

func getUsers(writer http.ResponseWriter, request *http.Request) {
	result := [1]map[string]string{
		{
			"_id":      "66971add3abcef545e641111",
			"name":     "Jerry",
			"username": "jerry",
		},
	}
	enc := json.NewEncoder(writer)
	writer.Header().Set("Content-Type", "application/json")
	_ = enc.Encode(result)
}
