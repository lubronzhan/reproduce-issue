package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/lubronzhan/reproduce-issue/pkg/localdirector"
)

func callBosh() error {
	log.Println("call bosh")
	c := &localdirector.Client{}
	boshHttp := localdirector.NewBoshHTTP(c)
	res, err := boshHttp.RawGet("/ok")
	fmt.Println(res)
	return err
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello odb %s\n", r.URL.Path[1:])
	err := callBosh()
	log.Println("Error: ")
	log.Println(err)
}

func main() {
	log.Println("ok")
	http.HandleFunc("/foo", handler)

	cert := "/Users/lzhan/Downloads/ca.crt"
	if len(os.Args) > 1 && os.Args[1] != "" {
		cert = os.Args[1]
	}

	key := "/Users/lzhan/Downloads/ca.key"
	if len(os.Args) > 2 && os.Args[2] != "" {
		key = os.Args[2]
	}

	log.Fatal(http.ListenAndServeTLS(":8080", cert, key, nil))
	log.Println("yes?")
}
