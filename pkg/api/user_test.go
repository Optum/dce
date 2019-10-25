package api_test

import (
	"fmt"
	"testing"

	"github.com/Optum/Redbox/pkg/api"
	"github.com/Optum/Redbox/pkg/awsiface/mocks"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/stretchr/testify/require"
)

func TestUser(t *testing.T) {

	t.Run("NonCognitoAuthIsAdmin, Output", func(t *testing.T) {

		mockCognitoIdp := &mocks.CognitoIdentityProviderAPI{}
		userGetter := api.UserDetails{
			CognitoUserPoolID:        "us_east_1-test",
			RolesAttributesAdminName: "admin",
			CognitoClient:            mockCognitoIdp,
		}

		user := userGetter.GetUser(&events.APIGatewayProxyRequest{
			RequestContext: events.APIGatewayProxyRequestContext{
				Identity: events.APIGatewayRequestIdentity{
					CognitoIdentityPoolID: "",
				},
			},
		})
		require.Equal(t, user.Username, "")
		require.Equal(t, user.Role, api.AdminGroupName)
	})

	t.Run("CognitoAuthInAdminsGroup, Output", func(t *testing.T) {

		mockCognitoIdp := &mocks.CognitoIdentityProviderAPI{}
		userGetter := api.UserDetails{
			CognitoUserPoolID:        "us_east_1-test",
			RolesAttributesAdminName: "admins",
			CognitoClient:            mockCognitoIdp,
		}

		mockCognitoIdp.On("ListUsers", &cognitoidentityprovider.ListUsersInput{
			Filter:     aws.String("sub = \"abcdef-123456\""),
			UserPoolId: aws.String("us_east_1-test"),
		}).Return(&cognitoidentityprovider.ListUsersOutput{
			Users: []*cognitoidentityprovider.UserType{
				{
					Username: aws.String("testuser"),
				},
			},
		}, nil)
		mockCognitoIdp.On("AdminListGroupsForUser", &cognitoidentityprovider.AdminListGroupsForUserInput{
			Username:   aws.String("testuser"),
			UserPoolId: aws.String("us_east_1-test"),
		}).Return(&cognitoidentityprovider.AdminListGroupsForUserOutput{
			Groups: []*cognitoidentityprovider.GroupType{
				{
					GroupName: aws.String("Admins"),
				},
			},
		}, nil)

		user := userGetter.GetUser(&events.APIGatewayProxyRequest{
			RequestContext: events.APIGatewayProxyRequestContext{
				Identity: events.APIGatewayRequestIdentity{
					CognitoIdentityPoolID:         "us_east_1-test",
					CognitoAuthenticationProvider: "UserPoolID:CognitoSignIn:abcdef-123456",
				},
			},
		})
		require.Equal(t, user.Username, "testuser")
		require.Equal(t, user.Role, api.AdminGroupName)
	})

	t.Run("CognitoAuthInAdminsRoleAttributes, Output", func(t *testing.T) {

		mockCognitoIdp := &mocks.CognitoIdentityProviderAPI{}
		userGetter := api.UserDetails{
			CognitoUserPoolID:        "us_east_1-test",
			RolesAttributesAdminName: "admins",
			CognitoClient:            mockCognitoIdp,
		}

		mockCognitoIdp.On("ListUsers", &cognitoidentityprovider.ListUsersInput{
			Filter:     aws.String("sub = \"abcdef-123456\""),
			UserPoolId: aws.String("us_east_1-test"),
		}).Return(&cognitoidentityprovider.ListUsersOutput{
			Users: []*cognitoidentityprovider.UserType{
				{
					Username: aws.String("testuser"),
					Attributes: []*cognitoidentityprovider.AttributeType{
						{
							Name:  aws.String("custom:roles"),
							Value: aws.String("group1, group2,group3, admins"),
						},
					},
				},
			},
		}, nil)
		mockCognitoIdp.On("AdminListGroupsForUser", &cognitoidentityprovider.AdminListGroupsForUserInput{
			Username:   aws.String("testuser"),
			UserPoolId: aws.String("us_east_1-test"),
		}).Return(&cognitoidentityprovider.AdminListGroupsForUserOutput{
			Groups: []*cognitoidentityprovider.GroupType{},
		}, nil)

		user := userGetter.GetUser(&events.APIGatewayProxyRequest{
			RequestContext: events.APIGatewayProxyRequestContext{
				Identity: events.APIGatewayRequestIdentity{
					CognitoIdentityPoolID:         "us_east_1-test",
					CognitoAuthenticationProvider: "UserPoolID:CognitoSignIn:abcdef-123456",
				},
			},
		})
		require.Equal(t, user.Username, "testuser")
		require.Equal(t, user.Role, api.AdminGroupName)
	})

	t.Run("LookForStringInCommaList, Output", func(t *testing.T) {

		mockCognitoIdp := &mocks.CognitoIdentityProviderAPI{}
		userGetter := api.UserDetails{
			CognitoUserPoolID:        "us_east_1-test",
			RolesAttributesAdminName: "admins",
			CognitoClient:            mockCognitoIdp,
		}

		mockCognitoIdp.On("ListUsers", &cognitoidentityprovider.ListUsersInput{
			Filter:     aws.String("sub = \"abcdef-123456\""),
			UserPoolId: aws.String("us_east_1-test"),
		}).Return(&cognitoidentityprovider.ListUsersOutput{
			Users: []*cognitoidentityprovider.UserType{
				{
					Username: aws.String("testuser"),
					Attributes: []*cognitoidentityprovider.AttributeType{
						{
							Name:  aws.String("custom:roles"),
							Value: aws.String("Group1,Group2,admins"),
						},
					},
				},
			},
		}, nil)

		user := userGetter.GetUser(&events.APIGatewayProxyRequest{
			RequestContext: events.APIGatewayProxyRequestContext{
				Identity: events.APIGatewayRequestIdentity{
					CognitoIdentityPoolID:         "us_east_1-test",
					CognitoAuthenticationProvider: "UserPoolID:CognitoSignIn:abcdef-123456",
				},
			},
		})
		require.Equal(t, user.Username, "testuser")
		require.Equal(t, user.Role, api.AdminGroupName)
	})

	t.Run("LookForStringInCommaListEmptyComma, Output", func(t *testing.T) {

		mockCognitoIdp := &mocks.CognitoIdentityProviderAPI{}
		userGetter := api.UserDetails{
			CognitoUserPoolID:        "us_east_1-test",
			RolesAttributesAdminName: "admins",
			CognitoClient:            mockCognitoIdp,
		}

		mockCognitoIdp.On("ListUsers", &cognitoidentityprovider.ListUsersInput{
			Filter:     aws.String("sub = \"abcdef-123456\""),
			UserPoolId: aws.String("us_east_1-test"),
		}).Return(&cognitoidentityprovider.ListUsersOutput{
			Users: []*cognitoidentityprovider.UserType{
				{
					Username: aws.String("testuser"),
					Attributes: []*cognitoidentityprovider.AttributeType{
						{
							Name:  aws.String("custom:roles"),
							Value: aws.String(","),
						},
					},
				},
			},
		}, nil)
		mockCognitoIdp.On("AdminListGroupsForUser", &cognitoidentityprovider.AdminListGroupsForUserInput{
			Username:   aws.String("testuser"),
			UserPoolId: aws.String("us_east_1-test"),
		}).Return(&cognitoidentityprovider.AdminListGroupsForUserOutput{
			Groups: []*cognitoidentityprovider.GroupType{},
		}, nil)

		user := userGetter.GetUser(&events.APIGatewayProxyRequest{
			RequestContext: events.APIGatewayProxyRequestContext{
				Identity: events.APIGatewayRequestIdentity{
					CognitoIdentityPoolID:         "us_east_1-test",
					CognitoAuthenticationProvider: "UserPoolID:CognitoSignIn:abcdef-123456",
				},
			},
		})
		require.Equal(t, user.Username, "testuser")
		require.Equal(t, user.Role, api.UserGroupName)
	})

	t.Run("CognitoAuthInAdminsGroup, Output", func(t *testing.T) {

		mockCognitoIdp := &mocks.CognitoIdentityProviderAPI{}
		userGetter := api.UserDetails{
			CognitoUserPoolID:        "us_east_1-test",
			RolesAttributesAdminName: "admins",
			CognitoClient:            mockCognitoIdp,
		}

		mockCognitoIdp.On("ListUsers", &cognitoidentityprovider.ListUsersInput{
			Filter:     aws.String("sub = \"abcdef-123456\""),
			UserPoolId: aws.String("us_east_1-test"),
		}).Return(&cognitoidentityprovider.ListUsersOutput{
			Users: []*cognitoidentityprovider.UserType{
				{
					Username: aws.String("testuser"),
					Attributes: []*cognitoidentityprovider.AttributeType{
						{
							Name:  aws.String("email"),
							Value: aws.String("invalid"),
						},
					},
				},
			},
		}, nil)
		mockCognitoIdp.On("AdminListGroupsForUser", &cognitoidentityprovider.AdminListGroupsForUserInput{
			Username:   aws.String("testuser"),
			UserPoolId: aws.String("us_east_1-test"),
		}).Return(&cognitoidentityprovider.AdminListGroupsForUserOutput{
			Groups: []*cognitoidentityprovider.GroupType{
				{
					GroupName: aws.String("Admins"),
				},
			},
		}, nil)

		user := userGetter.GetUser(&events.APIGatewayProxyRequest{
			RequestContext: events.APIGatewayProxyRequestContext{
				Identity: events.APIGatewayRequestIdentity{
					CognitoIdentityPoolID:         "us_east_1-test",
					CognitoAuthenticationProvider: "UserPoolID:CognitoSignIn:abcdef-123456",
				},
			},
		})
		require.Equal(t, user.Username, "testuser")
		require.Equal(t, user.Role, api.AdminGroupName)
	})
	t.Run("CognitoAuthNotInAdminsGroup, Output", func(t *testing.T) {

		mockCognitoIdp := &mocks.CognitoIdentityProviderAPI{}
		userGetter := api.UserDetails{
			CognitoUserPoolID:        "us_east_1-test",
			RolesAttributesAdminName: "admins",
			CognitoClient:            mockCognitoIdp,
		}

		mockCognitoIdp.On("ListUsers", &cognitoidentityprovider.ListUsersInput{
			Filter:     aws.String("sub = \"abcdef-123456\""),
			UserPoolId: aws.String("us_east_1-test"),
		}).Return(&cognitoidentityprovider.ListUsersOutput{
			Users: []*cognitoidentityprovider.UserType{
				{
					Username: aws.String("testuser"),
					Attributes: []*cognitoidentityprovider.AttributeType{
						{
							Name:  aws.String("email"),
							Value: aws.String("invalid"),
						},
					},
				},
			},
		}, nil)
		mockCognitoIdp.On("AdminListGroupsForUser", &cognitoidentityprovider.AdminListGroupsForUserInput{
			Username:   aws.String("testuser"),
			UserPoolId: aws.String("us_east_1-test"),
		}).Return(&cognitoidentityprovider.AdminListGroupsForUserOutput{
			Groups: []*cognitoidentityprovider.GroupType{
				{
					GroupName: aws.String("Users"),
				},
			},
		}, nil)

		user := userGetter.GetUser(&events.APIGatewayProxyRequest{
			RequestContext: events.APIGatewayProxyRequestContext{
				Identity: events.APIGatewayRequestIdentity{
					CognitoIdentityPoolID:         "us_east_1-test",
					CognitoAuthenticationProvider: "UserPoolID:CognitoSignIn:abcdef-123456",
				},
			},
		})
		require.Equal(t, user.Username, "testuser")
		require.Equal(t, user.Role, api.UserGroupName)
	})
	t.Run("CognitoAuthInAdminsGroupMultiple, Output", func(t *testing.T) {

		mockCognitoIdp := &mocks.CognitoIdentityProviderAPI{}
		userGetter := api.UserDetails{
			CognitoUserPoolID:        "us_east_1-test",
			RolesAttributesAdminName: "admins",
			CognitoClient:            mockCognitoIdp,
		}

		mockCognitoIdp.On("ListUsers", &cognitoidentityprovider.ListUsersInput{
			Filter:     aws.String("sub = \"abcdef-123456\""),
			UserPoolId: aws.String("us_east_1-test"),
		}).Return(&cognitoidentityprovider.ListUsersOutput{
			Users: []*cognitoidentityprovider.UserType{
				{
					Username: aws.String("testuser"),
					Attributes: []*cognitoidentityprovider.AttributeType{
						{
							Name:  aws.String("email"),
							Value: aws.String("invalid"),
						},
					},
				},
			},
		}, nil)
		mockCognitoIdp.On("AdminListGroupsForUser", &cognitoidentityprovider.AdminListGroupsForUserInput{
			Username:   aws.String("testuser"),
			UserPoolId: aws.String("us_east_1-test"),
		}).Return(&cognitoidentityprovider.AdminListGroupsForUserOutput{
			Groups: []*cognitoidentityprovider.GroupType{
				{
					GroupName: aws.String("Users"),
				},
				{
					GroupName: aws.String("Admins"),
				},
			},
		}, nil)

		user := userGetter.GetUser(&events.APIGatewayProxyRequest{
			RequestContext: events.APIGatewayProxyRequestContext{
				Identity: events.APIGatewayRequestIdentity{
					CognitoIdentityPoolID:         "us_east_1-test",
					CognitoAuthenticationProvider: "UserPoolID:CognitoSignIn:abcdef-123456",
				},
			},
		})
		require.Equal(t, user.Username, "testuser")
		require.Equal(t, user.Role, api.AdminGroupName)
	})
	t.Run("CognitoAuthInAdminsError, Output", func(t *testing.T) {

		mockCognitoIdp := &mocks.CognitoIdentityProviderAPI{}
		userGetter := api.UserDetails{
			CognitoUserPoolID:        "us_east_1-test",
			RolesAttributesAdminName: "admins",
			CognitoClient:            mockCognitoIdp,
		}

		mockCognitoIdp.On("ListUsers", &cognitoidentityprovider.ListUsersInput{
			Filter:     aws.String("sub = \"abcdef-123456\""),
			UserPoolId: aws.String("us_east_1-test"),
		}).Return(&cognitoidentityprovider.ListUsersOutput{
			Users: []*cognitoidentityprovider.UserType{
				{
					Username: aws.String("testuser"),
					Attributes: []*cognitoidentityprovider.AttributeType{
						{
							Name:  aws.String("email"),
							Value: aws.String("invalid"),
						},
					},
				},
			},
		}, nil)
		mockCognitoIdp.On("AdminListGroupsForUser", &cognitoidentityprovider.AdminListGroupsForUserInput{
			Username:   aws.String("testuser"),
			UserPoolId: aws.String("us_east_1-test"),
		}).Return(&cognitoidentityprovider.AdminListGroupsForUserOutput{}, fmt.Errorf("Fail"))

		user := userGetter.GetUser(&events.APIGatewayProxyRequest{
			RequestContext: events.APIGatewayProxyRequestContext{
				Identity: events.APIGatewayRequestIdentity{
					CognitoIdentityPoolID:         "us_east_1-test",
					CognitoAuthenticationProvider: "UserPoolID:CognitoSignIn:abcdef-123456",
				},
			},
		})
		require.Equal(t, user.Username, "testuser")
		require.Equal(t, user.Role, api.UserGroupName)
	})
}
