// Code generation for the frontend OpenAPI DTOs.
//
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.7.1 -generate types,std-http-server -package api -o internal/api/api.go assets/openapi.json

package main
