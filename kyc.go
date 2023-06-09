package kyc

import "go.riyazali.net/sqlite"

// ExtensionFunc returns a sqlite.ExtensionFunc that can be used to register kyc as a sqlite extension.
func ExtensionFunc() sqlite.ExtensionFunc {
	return func(ext *sqlite.ExtensionApi) (_ sqlite.ErrorCode, err error) {
		if err = ext.CreateModule("facts", &FactModule{}, sqlite.EponymousOnly(true)); err != nil {
			return sqlite.SQLITE_ERROR, err
		}

		if err = ext.CreateModule("commits", &CommitsModule{}, sqlite.EponymousOnly(true)); err != nil {
			return sqlite.SQLITE_ERROR, err
		}

		if err = ext.CreateFunction("head", &HeadFunc{}); err != nil {
			return sqlite.SQLITE_ERROR, err
		}

		if err = ext.CreateFunction("read_blob", &ReadBlob{}); err != nil {
			return sqlite.SQLITE_ERROR, err
		}

		if err = ext.CreateFunction("yaml_to_json", &YamlToJson{}); err != nil {
			return sqlite.SQLITE_ERROR, err
		}

		return sqlite.SQLITE_OK, nil
	}
}
