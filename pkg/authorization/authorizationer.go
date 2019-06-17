package authorization

import (
	"context"
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/Optum/Redbox/pkg/common"
)

// This is a near copy of cmd/lambda/acctmgr/authorization/authorization.go but
// implements it through an interface. authorization.go should be removed once
// it's all been replaced under cmd/lambda/acctmgr/main.go

// OAuthGrantType specifies which grant type to use.
type OAuthGrantType int

const (
	// OAuthGrantTypeServicePrincipal for client credentials flow
	OAuthGrantTypeServicePrincipal OAuthGrantType = iota
	// OAuthGrantTypeDeviceFlow for device flow
	OAuthGrantTypeDeviceFlow
)

var (
	graphAuthorizer autorest.Authorizer
)

type graphCreds struct {
	graphID     *string
	graphSecret *string
}

func (g *graphCreds) getCreds() {
	// Create the Store service and get the ClientID and ClientSecret
	awsSession, err := session.NewSession()
	if err != nil {
		log.Fatalf("%s\n", err)
	}
	ssmClient := ssm.New(awsSession)
	store := common.SSM{
		Client: ssmClient,
	}
	clientFile := "/redbox/graph/client/id"
	secretFile := "/redbox/graph/client/secret"
	g.graphID, err = store.GetParameter(&clientFile)
	if err != nil {
		log.Fatalf("%s\n", err)
	}
	g.graphSecret, err = store.GetParameter(&secretFile)
	if err != nil {
		log.Fatalf("%s\n", err)
	}
}

// Authorizationer interface for providing helpfer methods for Authenticating to
// IDP and executing IDP actions
type Authorizationer interface {
	GetGraphAuthorizer(tenantID string) (autorest.Authorizer, error)
	GetAuthorizerForResource(grantType OAuthGrantType, resource string,
		tenantID string) (autorest.Authorizer, error)
	AddADGroupUser(ctx context.Context, memberID string, groupID string,
		tenantID string) (autorest.Response, error)
	RemoveADGroupUser(ctx context.Context, groupID string, memberID string,
		tenantID string) (result autorest.Response, err error)
	ADGroupMember(ctx context.Context, groupID *string, memberID *string,
		tenantID *string) (result bool, err error)
}

// AzureAuthorization implements Authorization for Azure
type AzureAuthorization struct {
}

// GrantType returns what grant type has been configured.
func (author *AzureAuthorization) grantType() OAuthGrantType {
	return OAuthGrantTypeServicePrincipal
}

// getADGroupsClient retrieves a GroupsClient to assist with creating and managing Active Directory groups
func (author *AzureAuthorization) getADGroupsClient(tenantID string) (graphrbac.GroupsClient, error) {
	groupsClient := graphrbac.NewGroupsClient(tenantID)
	a, error := author.GetGraphAuthorizer(tenantID)
	if error != nil {
		return graphrbac.GroupsClient{}, fmt.Errorf("AD Groups error: %s", error)
	}
	groupsClient.Authorizer = a
	groupsClient.AddToUserAgent("redbox-auth")
	return groupsClient, nil
}

// GetGraphAuthorizer gets an OAuthTokenAuthorizer for graphrbac API.
func (author *AzureAuthorization) GetGraphAuthorizer(tenantID string) (autorest.Authorizer, error) {
	if graphAuthorizer != nil {
		return graphAuthorizer, nil
	}

	var a autorest.Authorizer
	var err error

	a, err = author.GetAuthorizerForResource(author.grantType(),
		"https://graph.windows.net", tenantID)

	if err == nil {
		// cache
		graphAuthorizer = a
	} else {
		graphAuthorizer = nil
	}

	return graphAuthorizer, err
}

//GetAuthorizerForResource does
func (author *AzureAuthorization) GetAuthorizerForResource(grantType OAuthGrantType, resource string, tenantID string) (autorest.Authorizer, error) {
	var grCreds graphCreds
	var a autorest.Authorizer
	var err error
	grCreds.getCreds()

	switch grantType {

	case OAuthGrantTypeServicePrincipal:
		oauthConfig, err := adal.NewOAuthConfig(
			"https://login.microsoftonline.com/", tenantID)
		if err != nil {
			return nil, err
		}

		token, err := adal.NewServicePrincipalToken(
			*oauthConfig, *grCreds.graphID, *grCreds.graphSecret, resource)
		if err != nil {
			return nil, err
		}
		a = autorest.NewBearerAuthorizer(token)

	case OAuthGrantTypeDeviceFlow:
		deviceconfig := auth.NewDeviceFlowConfig(*grCreds.graphID, tenantID)
		deviceconfig.Resource = resource
		a, err = deviceconfig.Authorizer()
		if err != nil {
			return nil, err
		}

	default:
		return a, fmt.Errorf("invalid grant type specified")
	}

	return a, err
}

//AddADGroupUser adds user to AAD group
func (author *AzureAuthorization) AddADGroupUser(ctx context.Context, memberID string, groupID string, tenantID string) (autorest.Response, error) {
	groupcli, err := author.getADGroupsClient(tenantID)
	if err != nil {
		return autorest.Response{}, fmt.Errorf("AddGroupUser error: %s", err)
	}
	var m = make(map[string]interface{})
	var catStr = "https://graph.windows.net/" + tenantID + "/directoryObjects/" + memberID
	var urlStr = &catStr
	groupRet, err := groupcli.AddMember(ctx, groupID, graphrbac.GroupAddMemberParameters{
		AdditionalProperties: m,
		URL:                  urlStr,
	})
	if err != nil {
		return autorest.Response{}, fmt.Errorf("AddGroupUser error: %s", err)
	}
	return groupRet, err
}

//RemoveADGroupUser removes a member from AAD group
func (author *AzureAuthorization) RemoveADGroupUser(ctx context.Context, groupID string, memberID string, tenantID string) (autorest.Response, error) {
	groupcli, err := author.getADGroupsClient(tenantID)
	if err != nil {
		return autorest.Response{}, fmt.Errorf("RemoveGroupUser error: %s", err)
	}
	groupRet, err := groupcli.RemoveMember(ctx, groupID, memberID)
	return groupRet, err
}

//ADGroupMember checks if input user is a member of input group
func (author *AzureAuthorization) ADGroupMember(ctx context.Context, groupID *string, memberID *string, tenantID *string) (bool, error) {
	groupcli, err := author.getADGroupsClient(*tenantID)
	if err != nil {
		return false, fmt.Errorf("CheckGroupUser error: %s", err)
	}
	m := make(map[string]interface{})

	if groupID == nil || memberID == nil {
		return false, fmt.Errorf("ID is nil")
	}
	memRet, err := groupcli.IsMemberOf(ctx, graphrbac.CheckGroupMembershipParameters{
		AdditionalProperties: m,
		GroupID:              groupID,
		MemberID:             memberID,
	})
	if err != nil {
		return false, err
	}

	return *memRet.Value, nil
}
