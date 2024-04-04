package types

type SaveDataItem interface {
	Save(dataitem TrustlessDataItem) SavedFile
}

type UniqueDataItemKey interface {
	GetUniqueKey(dataitem TrustlessDataItem) string
}
