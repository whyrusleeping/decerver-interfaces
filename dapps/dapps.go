package dapps

import (

)

const (
	PACKAGE_FILE_NAME = "package.json"
	INDEX_FILE_NAME = "index.html"
	MODELS_FOLDER_NAME = "models"
	
)

// Structs that are mapped to the package file.
type (
	PackageFile struct {
		Name               string              `json:"name"`
		Icon               string              `json:"app_icon"`
		Version            string              `json:"version"`
		Homepage           string              `json:"homepage"`
		Author             *Author             `json:"author"`
		Repository         *Repository         `json:"repository"`
		Bugs               *Bugs               `json:"bugs"`
		Licence            *Licence            `json:"licence"`
		ModuleDependencies *ModuleDependencies `json:"moduleDependencies"`
	}

	Author struct {
		Name string `json:"name"`
		Url  string `json:"url"`
	}

	Repository struct {
		Type string `json:"type"`
		Url  string `json:"url"`
	}

	Bugs struct {
		Url string `json:"url"`
	}

	Licence struct {
		Type string `json:"type"`
		Url  string `json:"url"`
	}

	ModuleDependencies struct {
		deps map[string]*Module
	}

	Module struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
)