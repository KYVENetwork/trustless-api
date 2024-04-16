package types

type SaveDataItem interface {
	Save(dataitem TrustlessDataItem) (SavedFile, error)
	Load(link string) (TrustlessDataItem, error)
}
