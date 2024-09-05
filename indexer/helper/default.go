package helper

type DefaultIndexer struct{}

func (d *DefaultIndexer) GetErrorResponse(message string, data any) any {
	return nil
}
