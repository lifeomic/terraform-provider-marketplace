package marketplace

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"

	"github.com/Khan/genqlient/graphql"
	"github.com/lifeomic/phc-sdk-go/client"
)

//go:generate go run github.com/Khan/genqlient

const GRAPHQL_URL = "marketplace-service:deployed/v1/marketplace/authenticated/graphql"
const defaultUser = "tf-provider"

var defaultPolicy = map[string]bool{
	"publishContent": true,
}

type MarketplaceClient struct {
	phcClient *client.LambdaClient
	gqlClient graphql.Client
}

func (marketplace *MarketplaceClient) getAppTileModule(id string) (*AppTileModule, error) {
	resp, err := GetPublishedModule(context.Background(), marketplace.gqlClient, id, "")
	if err != nil {
		return nil, err
	}
	return &resp.MyModule.AppTileModule, nil
}

type appTileCreate struct {
	Name           string
	Description    string
	Image          string
	AppTileId      string
	Version        string
	ParentModuleId *string
	Scope          *string
	Account        *string
	Url            *string
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
	startResponse, err := StartImageUpload(context.Background(), marketplace.gqlClient, StartUploadInput{
		FileName: fileName,
	})
	if err != nil {
		return err
	}

	err = postImageToUrl(startResponse.StartUpload.Url, image, fileName, startResponse.StartUpload.Fields)
	if err != nil {
		return err
	}

	finalizeResponse, err := FinalizeImageUpload(context.Background(), marketplace.gqlClient, FinalizeUploadInput{
		Id:       startResponse.StartUpload.Id,
		ModuleId: moduleId,
		Type:     "ICON",
	})

	if err != nil {
		return err
	}

	if finalizeResponse == nil {
		return errors.New("unable to finalize image upload")
	}

	return nil

}

func (marketplace *MarketplaceClient) createAppTileDraftModule(params appTileCreate) (*string, error) {
	parentModuleId := ""
	if params.ParentModuleId != nil {
		parentModuleId = *params.ParentModuleId
	}

	var scope MarketplaceModuleScope
	if params.Scope != nil && *params.Scope != "" {
		scope = MarketplaceModuleScope(*params.Scope)
	}

	if scope != "" && scope != MarketplaceModuleScopeLicensed &&
		scope != MarketplaceModuleScopeOrganization &&
		scope != MarketplaceModuleScopePublic {
		return nil, fmt.Errorf("unexpected module scope given. Expected one of 'LICENSED', 'ORGANIZATION', 'PUBLIC'. instead got: %s", scope)
	}

	if scope == MarketplaceModuleScopeOrganization && (params.Url == nil || *params.Url == "") {
		return nil, fmt.Errorf("modules with 'ORGANIZATION' scope must have a url")
	}

	res, err := CreateDraftModule(context.Background(), marketplace.gqlClient, CreateDraftModuleInput{
		Title:          params.Name,
		Description:    params.Description,
		ParentModuleId: parentModuleId,
		Category:       "APP_TILE",
		Scope:          scope,
	})
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, errors.New("unable to create draft module")
	}

	if scope == MarketplaceModuleScopePublic || scope == "" {
		appTileRes, err := SetAppTile(context.Background(), marketplace.gqlClient, SetPublicAppTileDraftModuleSourceInput{
			ModuleId: res.CreateDraftModule.Id,
			SourceInfo: PublicAppTileModuleSourceInfo{
				Id: params.AppTileId,
			},
		})

		if err != nil {
			return nil, err
		}

		if appTileRes == nil {
			return nil, errors.New("unable to set app tile")
		}
	}

	if scope == MarketplaceModuleScopeOrganization {
		appTileRes, err := SetOrgAppTile(context.Background(), marketplace.gqlClient, SetOrgAppTileDraftModuleSourceInput{
			ModuleId: res.CreateDraftModule.Id,
			SourceInfo: OrgAppTileModuleSourceInfo{
				Url: *params.Url,
			},
		})

		if err != nil {
			return nil, err
		}

		if appTileRes == nil {
			return nil, errors.New("unable to set app tile")
		}
	}

	err = marketplace.attachImageToDraftModule(res.CreateDraftModule.Id, params.Image)

	if err != nil {
		return nil, err
	}

	return &res.CreateDraftModule.Id, nil
}

func (marketplace *MarketplaceClient) publishNewAppTileModule(params appTileCreate) (*string, error) {
	draftModuleId, err := marketplace.createAppTileDraftModule(params)
	if err != nil {
		return nil, err
	}
	publishRes, err := PublishModule(context.Background(), marketplace.gqlClient, PublishDraftModuleInputV2{
		ModuleId: *draftModuleId,
		Version: ModuleVersionInput{
			Version: params.Version,
		},
	})
	if err != nil {
		return nil, err
	}
	if publishRes == nil {
		return nil, errors.New("unable to publish module")
	}
	return &publishRes.PublishDraftModuleV2.Id, nil
}

func buildCustomClient(account, user string, policy map[string]bool) (*MarketplaceClient, error) {
	phcClient, err := client.BuildClient(account, user, policy)
	if err != nil {
		return nil, err
	}
	gqlClient := graphql.NewClient(GRAPHQL_URL, phcClient)
	client := MarketplaceClient{phcClient: phcClient, gqlClient: gqlClient}
	return &client, nil
}

func BuildAppStoreClient() (*MarketplaceClient, error) {
	return buildCustomClient("lifeomic", "marketplace-tf", defaultPolicy)
}
