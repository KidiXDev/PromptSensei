package config

type DatasetPaths struct {
	TagCSV       string
	CharacterCSV string
	DBPath       string
	MetadataPath string
}

func (c Config) DatasetPaths() DatasetPaths {
	return DatasetPaths{
		TagCSV:       c.Dataset.TagCSVPath,
		CharacterCSV: c.Dataset.CharacterCSVPath,
		DBPath:       c.Dataset.CachePath,
		MetadataPath: c.Dataset.MetadataPath,
	}
}
