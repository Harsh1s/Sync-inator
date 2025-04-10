package SyncTest

import (
	"cse224/proj5/pkg/syncinator"
	"strconv"
	"strings"
)

func NewFileMetaData(InitMode int, filename string, version int, hashList []string, configStr string) *syncinator.FileMetaData {
	switch InitMode {
	case META_INIT_BY_PARAMS:
		return NewFileMetaDataFromParams(filename, version, hashList)
	case META_INIT_BY_CONFIG_STR:
		return NewFileMetaDataFromConfig(configStr)
	}

	// In default case, we return an empty file metadata object
	return &syncinator.FileMetaData{}
}

func NewFileMetaDataFromConfig(configString string) *syncinator.FileMetaData {
	configItems := strings.Split(configString, CONFIG_DELIMITER)

	filename := configItems[FILENAME_INDEX]
	version, _ := strconv.Atoi(configItems[VERSION_INDEX])

	blockHashList := strings.Split(strings.TrimSpace(configItems[HASH_LIST_INDEX]), HASH_DELIMITER)

	return &syncinator.FileMetaData{
		Filename:      filename,
		Version:       int32(version),
		BlockHashList: blockHashList,
	}
}

func NewFileMetaDataFromParams(filename string, version int, hashList []string) *syncinator.FileMetaData {
	return &syncinator.FileMetaData{
		Filename:      filename,
		Version:       int32(version),
		BlockHashList: hashList,
	}
}
