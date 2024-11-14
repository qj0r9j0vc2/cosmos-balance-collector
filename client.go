package main

type Client interface {
	Query(path string, parameters map[string]string) ([]byte, error)
}
