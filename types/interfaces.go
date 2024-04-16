package types

type SaveDataItem interface {
	Save(dataitem TrustlessDataItem) SavedFile
}
