package helper

import "github.com/KYVENetwork/trustless-api/files"

type DefaultIndexer struct{}

func (d *DefaultIndexer) GetErrorResponse(message string, data any) any {
	return map[string]any{
		"error":   message,
		"message": data,
	}
}

func (d *DefaultIndexer) InterceptRequest(get files.Get, indexId int, query []string) (*[]byte, error) {
	return nil, nil
}
