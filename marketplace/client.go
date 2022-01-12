package marketplace

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

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
	publishDraftModule(input: $input) {
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

type payload struct {
	Headers               map[string]string `json:"headers"`
	Path                  string            `json:"path"`
	HttpMethod            string            `json:"httpMethod"`
	QueryStringParameters map[string]string `json:"queryStringParameters"`
	Body                  string            `json:"body"`
}

type policy struct {
	Rules map[string]bool `json:"rules"`
}

func gqlQuery(query string, variables map[string]interface{}) []byte {
	type Body struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables"`
	}
	policy, _ := json.Marshal(&policy{
		Rules: map[string]bool{
			"publishContent": true,
		},
	})
	body, _ := json.Marshal(&Body{Query: query, Variables: variables})
	payload := &payload{
		Headers:               map[string]string{"LifeOmic-Account": "lifeomic", "LifeOmic-User": "marketplace-tf", "content-type": "application/json", "LifeOmic-Policy": string(policy)},
		HttpMethod:            "POST",
		QueryStringParameters: map[string]string{},
		Path:                  "/v1/marketplace/authenticated/graphql",
		Body:                  string(body),
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("Failed to marshall payload %v", err)
	}
	return bytes
}

type MarketplaceClient struct {
	c *lambda.Client
}

func (client *MarketplaceClient) gql(query string, variables map[string]interface{}) (*lambda.InvokeOutput, error) {
	MP_ARN := "marketplace-service:deployed"
	return client.c.Invoke(context.Background(), &lambda.InvokeInput{
		FunctionName: &MP_ARN,
		Payload:      gqlQuery(query, variables),
	})
}

type appTileModule struct {
	Name        string  `json:"title"`
	Description string  `json:"description"`
	Version     *string `json:"version"`
	Source      struct {
		Id string `json:"id"`
	} `json:"source"`
	Image *struct {
		Url           string `json:"url"`
		FileName      string `json:"fileName"`
		FileExtension string `json:"fileExtension"`
	} `json:"iconV2"`
}

func (client *MarketplaceClient) getAppTileModule(id string) (*appTileModule, error) {
	res, err := client.gql(GET_PUBLISHED_APP_TILE_MODULE, map[string]interface{}{"id": id})
	if err != nil {
		return nil, err
	}
	var payload responsePayload
	err = json.Unmarshal(res.Payload, &payload)
	if err != nil {
		return nil, err
	}
	var body struct {
		Data struct {
			MyModule appTileModule `json:"myModule"`
		} `json:"data"`
	}
	err = json.Unmarshal([]byte(payload.Body), &body)
	if err != nil {
		return nil, err
	}
	module := body.Data.MyModule
	return &module, nil
}

type appTileCreate struct {
	Name           string
	Description    string
	Image          string
	AppTileId      string
	Version        string
	ParentModuleId *string
}

type responsePayload struct {
	Body string `json:"body"`
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

func (client *MarketplaceClient) attachImageToDraftModule(moduleId string, image string) error {
	fileName := path.Base(image)
	startResponse, err := client.gql(START_IMAGE_UPLOAD, map[string]interface{}{
		"input": map[string]interface{}{
			"fileName": fileName,
		},
	})
	if err != nil {
		return err
	}
	var startPayload responsePayload
	err = json.Unmarshal(startResponse.Payload, &startPayload)
	if err != nil {
		return err
	}
	var startBody struct {
		Data struct {
			StartUpload struct {
				Fields map[string]string `json:"fields"`
				Url    string            `json:"url"`
				Id     string            `json:"id"`
			} `json:"startUpload"`
		} `json:"data"`
	}
	err = json.Unmarshal([]byte(startPayload.Body), &startBody)
	if err != nil {
		return err
	}

	err = postImageToUrl(startBody.Data.StartUpload.Url, image, fileName, startBody.Data.StartUpload.Fields)
	if err != nil {
		return err
	}

	finalizeResponse, err := client.gql(FINALIZE_IMAGE_UPLOAD, map[string]interface{}{
		"input": map[string]string{
			"id":       startBody.Data.StartUpload.Id,
			"moduleId": moduleId,
			"type":     "ICON",
		},
	})

	if err != nil {
		return nil
	}

	var finalizePayload responsePayload
	err = json.Unmarshal(finalizeResponse.Payload, &finalizePayload)
	if err != nil {
		return err
	}

	var finalizeBody struct {
		Data struct {
			FinalizeUpload struct {
				ModuleId string `json:"moduleId"`
			} `json:"finalizeUpload"`
		} `json:"data"`
	}

	err = json.Unmarshal([]byte(finalizePayload.Body), &finalizeBody)
	return err
}

func (client *MarketplaceClient) createAppTileDraftModule(params appTileCreate) (*string, error) {
	res, err := client.gql(CREATE_DRAFT_MODULE, map[string]interface{}{"input": map[string]interface{}{
		"title":       params.Name,
		"description": params.Description,
		// "iconV2":         params.Image, // Use upload
		"parentModuleId": params.ParentModuleId,
		"category":       "APP_TILE",
	}})

	if err != nil {
		return nil, err
	}

	var createDraftModulePayload responsePayload
	err = json.Unmarshal(res.Payload, &createDraftModulePayload)
	if err != nil {
		return nil, err
	}

	var createDraftModuleBody struct {
		Data struct {
			CreateDraftModule struct {
				Id string `json:"id"`
			} `json:"createDraftModule"`
		} `json:"data"`
	}

	err = json.Unmarshal([]byte(createDraftModulePayload.Body), &createDraftModuleBody)
	if err != nil {
		return nil, err
	}

	moduleId := createDraftModuleBody.Data.CreateDraftModule.Id

	res, err = client.gql(SET_APP_TILE, map[string]interface{}{"input": map[string]interface{}{
		"moduleId": moduleId,
		"sourceInfo": map[string]string{
			"id": params.AppTileId,
		},
	}})

	if err != nil {
		return nil, err
	}

	var setAppTilePayload responsePayload
	err = json.Unmarshal(res.Payload, &setAppTilePayload)
	if err != nil {
		return nil, err
	}
	var setAppTileBody struct {
		Data struct {
			SetPublicAppTileDraftModuleSource struct {
				ModuleId string `json:"moduleId"`
			} `json:"setPublicAppTileDraftModuleSource"`
		} `json:"data"`
	}
	err = json.Unmarshal([]byte(setAppTilePayload.Body), &setAppTileBody)
	if err != nil {
		return nil, err
	}

	err = client.attachImageToDraftModule(moduleId, params.Image)

	if err != nil {
		return nil, err
	}

	return &moduleId, nil
}

func (client *MarketplaceClient) publishNewAppTileModule(params appTileCreate) (*string, error) {
	draftModuleId, err := client.createAppTileDraftModule(params)
	if err != nil {
		return nil, err
	}
	publishRes, err := client.gql(PUBLISH_MODULE, map[string]interface{}{"input": map[string]interface{}{
		"moduleId": draftModuleId,
		"version": map[string]string{
			"version": params.Version,
		},
	}})
	if err != nil {
		return nil, err
	}
	var publishPayload responsePayload
	err = json.Unmarshal(publishRes.Payload, &publishPayload)
	if err != nil {
		return nil, err
	}
	var publishModuleBody struct {
		Data struct {
			PublishDraftModule struct {
				Id      string `json:"id"`
				Version struct {
					Version string `json:"version"`
				} `json:"version"`
			} `json:"publishDraftModule"`
		} `json:"data"`
	}
	err = json.Unmarshal([]byte(publishPayload.Body), &publishModuleBody)
	if err != nil {
		return nil, err
	}
	return &publishModuleBody.Data.PublishDraftModule.Id, nil
}

func BuildAppStoreClient() (*MarketplaceClient, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithSharedConfigProfile("lifeomic-dev"))
	if err != nil {
		return nil, err
	}
	client := MarketplaceClient{c: lambda.NewFromConfig(cfg)}
	return &client, nil
}
