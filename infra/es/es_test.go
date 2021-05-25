package es

import (
	"log"
	"testing"
)

func TestEsClient_AddDoc(t *testing.T) {
	es, err := NewEsClient("127.0.0.1:9200")
	defer es.Stop()
	if err != nil {
		log.Fatalln(err)
	}
	doc := `{"name":"hello","value":"world"}`

	es.AddDoc("test_267", "", doc)

}
