# Urit
[![GoDoc](https://godoc.org/github.com/go-andiamo/urit?status.svg)](https://pkg.go.dev/github.com/go-andiamo/urit)
[![Latest Version](https://img.shields.io/github/v/tag/go-andiamo/urit.svg?sort=semver&style=flat&label=version&color=blue)](https://github.com/go-andiamo/urit/releases)
[![codecov](https://codecov.io/gh/go-andiamo/urit/branch/main/graph/badge.svg?token=igjnZdgh0e)](https://codecov.io/gh/go-andiamo/urit)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-andiamo/urit)](https://goreportcard.com/report/github.com/go-andiamo/urit)

## Overview

Go package for templated URIs - enables building URIs from templates and extracting path vars from templates

Define path vars by name...
```go
template := urit.MustCreateTemplate(`/foo/{foo-id:[a-z]*}/bar/{bar-id:[0-9]*}`)
pth, _ := template.PathFrom(urit.Named(
    "foo-id", "abc",
    "bar-id", "123"))
println(pth)
```
or positional...
```go
template := urit.MustCreateTemplate(`/foo/?/bar/?`)
pth, _ := template.PathFrom(urit.Positional("abc", "123"))
println(pth)
```

Extract vars from paths - using named...
```go
template := urit.MustCreateTemplate(`/credits/{year:[0-9]{4}}/{month:[0-9]{2}}`)
req, _ := http.NewRequest(`GET`, `http://www.example.com/credits/2022/11`, nil)
vars, ok := template.MatchesRequest(req)
println(ok)
println(vars.Get("year"))
println(vars.Get("month"))
```
Or extract using positional...
```go
template := urit.MustCreateTemplate(`/credits/?/?`)
req, _ := http.NewRequest(`GET`, `http://www.example.com/credits/2022/11`, nil)
vars, ok := template.MatchesRequest(req)
println(ok)
println(vars.Get(0))
println(vars.Get(1))
```

Generate path from a template...
```go
template := urit.MustCreateTemplate(`/credits/{year:[0-9]{4}}/{month:[0-9]{2}}`)

path, _ := template.PathFrom(urit.Named("year", "2022", "month", "11"))
println(path)
```

## Installation
To install Urit, use go get:

    go get github.com/go-andiamo/urit

To update Urit to the latest version, run:

    go get -u github.com/go-andiamo/urit

