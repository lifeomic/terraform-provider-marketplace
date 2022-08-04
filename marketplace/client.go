package marketplace

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"

	"github.com/lifeomic/phc-sdk-go/client"

	"github.com/mitchellh/mapstructure"
)

//go:generate go run github.com/Khan/genqlient

const GET_PUBLISHED_APP_TILE_MODULE = `
  query GetPublishedModule($id: ID!, $version: String) {
    myModule(moduleId: $id, version: $version) {
      title
      description
	  version
	  source {
		... on AppTile {
		  id
		}
	  }
	  iconV2 {
		url
		fileName
		fileExtension
	  }
    }
  }
`

const CREATE_DRAFT_MODULE = `
 mutation CreateDraftModule($input: CreateDraftModuleInput!) {
   createDraftModule(input: $input) {
     id
   }
 }
`

const SET_APP_TILE = `
  mutation SetAppTile($input: SetPublicAppTileDraftModuleSourceInput!) {
	setPublicAppTileDraftModuleSource(input: $input) {
	  moduleId
	}
  }
`

const PUBLISH_MODULE = `
  mutation PublishModule($input: PublishDraftModuleInputV2!) {
	publishDraftModuleV2(input: $input) {
	  id
	  version {
		version
	  }
	}
  }
`

const START_IMAGE_UPLOAD = `
  mutation StartImageUpload($input: StartUploadInput!) {
	startUpload(input: $input) {
	  id
	  url
	  fields
	}
  }
`

const FINALIZE_IMAGE_UPLOAD = `
  mutation FinalizeImageUpload($input: FinalizeUploadInput!) {
	finalizeUpload(input: $input) {
	  moduleId
	}
  }
`

const GRAPHQL_URL = "marketplace-service:deployed/v1/marketplace/authenticated/graphql"

type MarketplaceClient struct {
	phcClient *client.Client
}

type appTileModule struct {
	Title       string
	Description string
	Version     string
	Source      struct {
		Id string
	}
	IconV2 *struct {
		Url           string
		FileName      string
		FileExtension string
	}
}

func (marketplace *MarketplaceClient) getAppTileModule(id string) (*appTileModule, error) {
	res, err := marketplace.phcClient.Gql(GRAPHQL_URL, GET_PUBLISHED_APP_TILE_MODULE, map[string]interface{}{"id": id})
	if err != nil {
		return nil, err
	}
	var data struct {
		MyModule *appTileModule
	}
	err = mapstructure.Decode(res, &data)
	if err != nil {
		return nil, err
	}
	return data.MyModule, nil
}

type appTileCreate struct {
	Name           string
	Description    string
	Image          string
	AppTileId      string
	Version        string
	ParentModuleId *string
}

func postImageToUrl(url string, image string, file_name string, fields map[string]string) error {
	file, err := os.Open(image)
	if err != nil {
		return err
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for key, val := range fields {
		err = writer.WriteField(key, val)
		if err != nil {
			return err
		}
	}
	part, err := writer.CreateFormFile("file", file_name)
	if err != nil {
		return err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return err
	}

	err = writer.Close()
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err != nil {
		return err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	responseBody := &bytes.Buffer{}
	responseBody.ReadFrom(resp.Body)
	resp.Body.Close()
	return nil
}

func (marketplace *MarketplaceClient) attachImageToDraftModule(moduleId string, image string) error {
	fileName := path.Base(image)
	startResponse, err := marketplace.phcClient.Gql(GRAPHQL_URL, START_IMAGE_UPLOAD, map[string]interface{}{
		"input": map[string]interface{}{
			"fileName": fileName,
		},
	})
	if err != nil {
		return err
	}
	var startData struct {
		StartUpload struct {
			Fields map[string]string
			Url    string
			Id     string
		}
	}
	err = mapstructure.Decode(startResponse, &startData)
	if err != nil {
		return err
	}

	err = postImageToUrl(startData.StartUpload.Url, image, fileName, startData.StartUpload.Fields)
	if err != nil {
		return err
	}

	finalizeResponse, err := marketplace.phcClient.Gql(GRAPHQL_URL, FINALIZE_IMAGE_UPLOAD, map[string]interface{}{
		"input": map[string]string{
			"id":       startData.StartUpload.Id,
			"moduleId": moduleId,
			"type":     "ICON",
		},
	})

	if err != nil {
		return nil
	}

	var finalizeData struct {
		FinalizeUpload struct {
			ModuleId string
		}
	}

	err = mapstructure.Decode(finalizeResponse, &finalizeData)
	return err
}

func (marketplace *MarketplaceClient) createAppTileDraftModule(params appTileCreate) (*string, error) {
	res, err := marketplace.phcClient.Gql(GRAPHQL_URL, CREATE_DRAFT_MODULE, map[string]interface{}{"input": map[string]interface{}{
		"title":       params.Name,
		"description": params.Description,
		// "iconV2":         params.Image, // Use upload
		"parentModuleId": params.ParentModuleId,
		"category":       "APP_TILE",
	}})

	if err != nil {
		return nil, err
	}

	var createDraftData struct {
		CreateDraftModule struct {
			Id string
		}
	}

	err = mapstructure.Decode(res, &createDraftData)
	if err != nil {
		return nil, err
	}

	moduleId := createDraftData.CreateDraftModule.Id

	res, err = marketplace.phcClient.Gql(GRAPHQL_URL, SET_APP_TILE, map[string]interface{}{"input": map[string]interface{}{
		"moduleId": moduleId,
		"sourceInfo": map[string]string{
			"id": params.AppTileId,
		},
	}})

	if err != nil {
		return nil, err
	}

	var setAppTileData struct {
		SetPublicAppTileDraftModuleSource struct {
			ModuleId string
		}
	}
	err = mapstructure.Decode(res, &setAppTileData)
	if err != nil {
		return nil, err
	}

	err = marketplace.attachImageToDraftModule(moduleId, params.Image)

	if err != nil {
		return nil, err
	}

	return &moduleId, nil
}

func (marketplace *MarketplaceClient) publishNewAppTileModule(params appTileCreate) (*string, error) {
	draftModuleId, err := marketplace.createAppTileDraftModule(params)
	if err != nil {
		return nil, err
	}
	publishRes, err := marketplace.phcClient.Gql(GRAPHQL_URL, PUBLISH_MODULE, map[string]interface{}{"input": map[string]interface{}{
		"moduleId": draftModuleId,
		"version": map[string]string{
			"version": params.Version,
		},
	}})
	if err != nil {
		return nil, err
	}
	var publishModuleData struct {
		PublishDraftModuleV2 struct {
			Id      string
			Version struct {
				Version string
			}
		}
	}
	err = mapstructure.Decode(publishRes, &publishModuleData)
	if err != nil {
		return nil, err
	}
	return &publishModuleData.PublishDraftModuleV2.Id, nil
}

func BuildAppStoreClient() (*MarketplaceClient, error) {
	phcClient, err := client.BuildClient("lifeomic", "marketplace-tf", map[string]bool{
		"publishContent": true,
	})
	if err != nil {
		return nil, err
	}
	client := MarketplaceClient{phcClient: phcClient}
	return &client, nil
}
