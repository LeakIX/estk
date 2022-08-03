package lib

import (
	"fmt"
)

type LsCommand struct {
}

func (cmd *LsCommand) Run(dispatcher *EsQueryDispatcher) (err error) {
	var indexList []IndexInfo
	err, _ = dispatcher.ESRequest("get", "/_cat/indices?format=json&bytes=b", &indexList, nil)
	if err != nil {
		return err
	}
	for _, index := range indexList {
		fmt.Printf("%s (%s docs)\n", index.Index, index.DocCount)
	}
	return nil
}

type IndexInfo struct {
	Health    string `json:"health"`
	Index     string `json:"index"`
	DocCount  string `json:"docs.count"`
	StoreSize string `json:"pri.store.size"`
}
