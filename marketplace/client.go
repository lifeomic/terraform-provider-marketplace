package marketplace

import (
	"context"
	"encoding/json"
	"log"

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
  mutation PublishModule($input: PublishDraftModuleInput!) {
	publishDraftModule(input: $input) {
	  id
	  version
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
	return &moduleId, nil
}

func (client *MarketplaceClient) publishNewAppTileModule(params appTileCreate) (*string, error) {
	draftModuleId, err := client.createAppTileDraftModule(params)
	if err != nil {
		return nil, err
	}
	publishRes, err := client.gql(PUBLISH_MODULE, map[string]interface{}{"input": map[string]interface{}{
		"moduleId": draftModuleId,
		"version":  params.Version,
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
				Version string `json:"version"`
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
