package es

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/olivere/elastic/v7"
	"mgkj/util"
	"strings"
)

type EsClient struct {
	Client *elastic.Client
}

func NewEsClient(url string) (*EsClient, error) {
	client, err := elastic.NewClient(
		elastic.SetURL(util.ProcessUrlStringWithHttp(url)[:]...),
		elastic.SetSniff(false),
	)
	if err != nil {
		return nil, err
	}
	return &EsClient{
		client,
	}, nil
}
func (e *EsClient) Stop() {
	e.Client.Stop()
}
func (e *EsClient) Exist(index, id string) bool {
	exist, _ := e.Client.Exists().Index(index).Id(id).Do(context.Background())
	if !exist {
		return false
	}
	return true
}
func (e *EsClient) GetDoc(index, id string) (map[string]interface{}, error) {
	if !e.Exist(index, id) {
		return nil, fmt.Errorf("id not exist")
	}
	res, err := e.Client.Get().Index(index).Id(id).Do(context.Background())
	body := res.Source
	var r map[string]interface{}
	json.NewDecoder(strings.NewReader(string(body))).Decode(&r)
	return r, err
}
func (e *EsClient) AddDoc(index, id, doc string) error {
	_, err := e.Client.Index().Index(index).Id(id).BodyJson(doc).Do(context.Background())
	return err
}
func (e *EsClient) UpdataDoc(index, id string, updateField map[string]interface{}) error {
	if !e.Exist(index, id) {
		return fmt.Errorf("index not exist")
	}
	_, err := e.Client.Update().Index(index).Id(id).Doc(updateField).Do(context.Background())
	return err
}
func (e *EsClient) DeleteDoc(index, id string) error {
	_, err := e.Client.Delete().Index(index).Id(id).Do(context.Background())
	return err
}
