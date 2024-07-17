// Package urit - Go package for building URIs from templates and extracting path vars from URIs (using template)
/*
Define path vars by name...
	template := urit.MustCreateTemplate(`/foo/{foo-id:[a-z]*}/bar/{bar-id:[0-9]*}`)
	pth, _ := template.PathFrom(urit.Named(
		"foo-id", "abc",
		"bar-id", "123"))
	println(pth)
or positional...
	template := urit.MustCreateTemplate(`/foo/?/bar/?`)
	pth, _ := template.PathFrom(urit.Positional("abc", "123"))
	println(pth)

Extract vars from paths - using named...
	template := urit.MustCreateTemplate(`/credits/{year:[0-9]{4}}/{month:[0-9]{2}}`)
	req, _ := http.NewRequest(`GET`, `http://www.example.com/credits/2022/11`, nil)
	vars, ok := template.MatchesRequest(req)
	println(ok)
	println(vars.Get("year"))
	println(vars.Get("month"))
Or extract using positional...
	template := urit.MustCreateTemplate(`/credits/?/?`)
	req, _ := http.NewRequest(`GET`, `http://www.example.com/credits/2022/11`, nil)
	vars, ok := template.MatchesRequest(req)
	println(ok)
	println(vars.Get(0))
	println(vars.Get(1))
*/
package urit
